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
