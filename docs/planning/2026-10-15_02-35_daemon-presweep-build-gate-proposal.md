# Daemon Pre-Sweep Build Gate — Proposal for the Fleet Auto-Committer

_Created: 2026-10-15 02:35 CEST · Author context: the 2026-10-14 master-red incident (commit `aecdf49` committed a non-compiling intermediate state produced by a known-broken linter rewrite hint; it stayed red for hours and broke a docs-health session) and the 2026-10-15 vendorHash incident (same class, caught locally this time) · Target: `projects-management-automation` (pma), the fleet auto-commit daemon · Upstream evidence: pma's own status report `docs/status/2026-10-01_09-58` §e.1 ("Build-gate the auto-committer") and this repo's AGENTS.md daemon paragraph._

## Problem

The daemon commits continuously by design; nothing in its commit path compiles the tree first. Every session that leaves a broken intermediate state (mid-edit, linter auto-fix fights, dependency bumps) gets that state snapshotted into master history — and pushed. Consequences observed in this repo alone: red CI for hours, broken `go.work` for parallel sessions, heuristic commits that must be reverted rather than amended.

## Current state (source-verified 2026-10-15)

- Commit chain: `pma-daemon/batch.go:131` `batchWorker` → `batch.go:227` `Committer.Commit` → `pma-daemon/committer/committer.go:233` `commitHandler.Execute`.
- Pre-commit checks today: conflict-state only (`batch.go:190` → `conflict/resolver_operations.go:60`), min-commit interval, failure cooldown. **No build/vet/test check.**
- The config key already exists and is dead: `require_clean_build` (`pma-daemon/config/config.go:21`, default `false` at `:107`) is read in `Merge` but never consulted by `Service` or `Committer` — `ToCommitOptions()` never passes it.
- Known-contention hazard: J2 in pma's TODO records 387 `git add --all` exit-128 failures from `.git/index.lock` contention with concurrent BuildFlow pre-commit runs — any new gate must not add a second lock holder.

## Proposed design

**What runs:** when a repo contains a `go.mod` (and the `go` binary is on PATH), before staging: `go build ./...`, then `go vet ./...`. Nothing heavier — no tests, no lint (those are BuildFlow/CI's job; the gate's only job is "don't snapshot a tree that doesn't compile"). For non-Go repos the gate no-ops; if `go` is missing, the gate no-ops with one WARN (never blocks).

**When:** in `processBatch` (`batch.go`, after `IsSafeToCommit`, before `ShouldCommit`/`Commit`) rather than inside `Committer.Commit`. Rationale: a failed gate is not a failed commit — reusing the commit-failure cooldown would conflate two classes and hide real commit bugs. A separate gate result ("dirty-build, skipped") gets its own cooldown/backoff so a permanently broken tree does not rebuild on every sweep.

**Failure mode:** gate fails → skip the commit, log WARN with the build/vet stderr tail (truncated, e.g. last 20 lines) and the gate duration; leave files uncommitted for the next sweep. Escalating backoff (reuse the 5m→4h cooldown shape) with a reset on first green. The watcher stays fully active; nothing is lost — work simply stays uncommitted until it compiles.

**Cost budget:** one warm `go build ./...` + `go vet ./...` per batch attempt per Go repo — seconds with a warm GOCACHE (the daemon user needs a writable `GOCACHE`; env already flows through). Bounded by the existing batch/min-commit intervals. An in-memory "last gate result" per repo short-circuits re-checks inside the same batch window.

**Config:** activate `require_clean_build` (it already parses; flip its default to `true` fleet-wide or per-repo once per-repo config exists). The vet step should be a separate key (`require_clean_vet`, default `false`) so the minimal gate ships first.

**Interaction guard (do not skip):** BuildFlow's pre-commit hook and this gate both exec git/go concurrently on the same repo — the index.lock contention class J2 already recorded. Mitigation: run the gate BEFORE any `git add` (build reads the worktree, not the index; it takes no git locks), and serialize per-repo with the same lock the committer uses for `git add`, not a new global one.

## Verification plan (for the pma repo)

1. Unit: gate skips non-Go repos; WARN-no-op when `go` is absent; failed build → no commit call, backoff set; green build → commit proceeds.
2. Integration: break a watched fixture repo, confirm no commit lands and the WARN carries the compiler error; fix it, confirm the next sweep commits.
3. Journal: after a week, `journalctl -t pma | grep -c 'build-gate'` vs the heuristic-fallback count — the gate should show near-zero false blocks (gate passes on every tree CI would pass).

## Requested ruling

The gate changes fleet-wide daemon behavior (owner-owned tooling). Options: (a) ship as opt-in (`require_clean_build: true` in `service.yaml`), (b) ship default-on for Go repos with a per-repo opt-out, (c) reject. This repo's vote: (b) — the red-master incidents cost more than the gate's seconds.

---

## Update 2026-10-23 — validated by a live incident; the sweep won three times

**Status: proposal VALIDATED by a red-master incident that lasted 8 days. Ruling still pending.**

The 2026-10-15 02:35 draft above was written after two red-master incidents. On that same day, a third and far more expensive instance proved the failure class end-to-end — and this one was caused by the daemon re-landing a lint auto-fix that humans had deliberately reverted twice.

### Timeline (source-verified with `git log -S`)

| Time (2026-10-15) | Commit    | Event                                                                                           |
| ----------------- | --------- | ----------------------------------------------------------------------------------------------- |
| 18:47             | `28681f2` | deliberate `fix: revert error-interface annotations on error-family sentinels` (first reversal) |
| 19:08             | `e6431ce` | daemon heuristic commit — **re-lands the sweep across 9 files**, 21 min later                   |
| 19:17             | `a6e0a12` | deliberate `fix: re-revert the error-interface sentinel sweep (second occurrence)`              |
| 19:31             | `0ae65b9` | daemon heuristic commit — **re-lands the sweep again**, 14 min later                            |

Master stayed red and pushed from 19:31 on 2026-10-15 until 2026-10-23. `go vet ./...` failed with:

```text
example_test.go:859:30: ErrCSRFInvalid.WithContext undefined (type error has no field or method WithContext)
```

### Why this specific incident strengthens the proposal

1. **It is exactly the gate's target class.** A mechanically-applied rewrite produced a tree that does not compile, and the daemon snapshotted it without asking. `go build ./...` would have refused the commit in under a second.
2. **Human reverts are not a control.** The sweep had already been reverted, documented, and reverted again. Neither revert survived the next daemon cycle. Policy text alone (AGENTS.md already said _"declaring `var errX error` breaks WithContext/WithCause call sites"_) did not stop it.
3. **The residue was almost entirely latent.** Only one of the 15 annotations (`csrf.go:60`, the only sentinel with a direct `.WithContext` call in compiled code) broke the build; the other 14 were pattern violations that would have surfaced on the next consumer. A build gate catches the first; only a test catches the rest — which is why this repo added the AST guard `sentinels_type_test.go` alongside the fix (commit `9aa4178`).
4. **Recovery cost was ~8 days of unusable post-10-15 master** for every consumer pinned to it, plus a full forensics session to reconstruct what happened from `git show 0ae65b9` (the failing `go vet` names only one of the 15 sites).

### Consequence for the requested ruling

The vote for option (b) — default-on for Go repos with a per-repo opt-out — is unchanged, now with an incident that the gate would have blocked three separate times. Detection-only tooling cannot cover this class: the first two sweeps compiled cleanly at the point of commit and only failed when a consumer symbol was touched.

Evidence trail: `docs/status/2026-10-23_15-56_otel-tracing-exploration-and-red-master.md` §a–c and the execution plan `docs/planning/2026-10-23_16-12_green-master-and-otel-tracing-1-4-20-plan.md` (§P0 executed, §P1.1).

**Ruling: pending — owner-owned tooling (pma). Do not wire anything until the ruling lands.**

### Update 2026-10-23 — the actor is identified, and it is a fleet rule (P05)

The sweep is not a mechanical tool applying a fix: `erraudit lint --help` has no `--fix` flag. The actor is an agent applying a **different repo's policy** to this one. `BuildFlow/docs/GOTCHAS.md` #125 states, as a hard rule, _"Sentinel variables MUST be declared as the `error` interface type"_, because declaring the concrete type makes erraudit's `legacy_is` rule flag every `errors.Is` call site.

That rule is correct for BuildFlow and wrong here. The two repos hold sentinels with opposite lifecycles:

- BuildFlow's sentinels are compared with `errors.Is` and never call `.WithContext`/`.WithCause`, so the `error` interface costs nothing and silences the advisory.
- httputil's are the opposite: they are _load-bearing_. `ErrCSRFInvalid.WithContext("path", "/login")` (`example_test.go:859`) is the exact call site the 2026-10-15 sweep broke.

The tool has since moved toward our position but has not arrived. erraudit `119791b` splits the 51 `sentinel_concrete_type` findings into 36 labeled _"load-bearing — keep the declaration"_ and 15 that still carry the `var errX error` rewrite hint. `ErrCSRFInvalid` is one of the 15 — the tool's load-bearing analysis reads non-test files only, so a method call site that lives in `example_test.go` is invisible to it. Acting on that hint re-creates the outage.

Consequence for the ruling: a pre-sweep build gate would have blocked all three re-lands, but it would not have prevented the _rewrite_. The durable controls are the ones now in place — AST guards in both modules, a self-test proving the guard is not vacuous, `scripts/pre-sweep-check.sh`, and the explicit prerelease gate. The fleet-level corrections (erraudit's hint vs test-file call sites; GOTCHAS #125's blanket wording) are cross-repo and owner-gated; they are recorded here and in the AGENTS erraudit block rather than edited silently.

**Ruling: still pending — and now independent of the actor question.** The gate is wanted for the daemon class regardless of who writes the tree.
