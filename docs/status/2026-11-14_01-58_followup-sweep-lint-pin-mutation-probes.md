# Session Status — 2026-11-14 01:58: Post-Fix Follow-Up Sweep (§f List Execution, Pre-Release)

Scope: continue executing the 24-item §f list from
[docs/status/2026-11-14_00-37_issues-45-49-five-open-issue-fixes.md](2026-11-14_00-37_issues-45-49-five-open-issue-fixes.md)
(the five issue fixes themselves landed last session). This session: items 1, 4, 5, 6, 14, 15,
16, 17, 23 done; item 11 (benchmarks) launched and ~80% complete at report time; items
3/12/19/22/24/7–10 queued behind it. All edits swept by the daemon (4 heuristic commits,
unpushed — see d/e).

## a) FULLY DONE (this session)

1. **§f.2 CI tail re-check** — the final CODE head `1587b51` is CI-green (run 43518839667,
   11m58s, 2026-11-13T22:23Z). The two commits after it (`f5349b9`, `ece47fd`, 00:39/00:48)
   are docs-only (the prior status report). Daemon `auto_push` is currently OFF — tree is
   4 heuristic commits ahead of origin; resolves in the single release push (RELEASE.md step 12).
2. **§f.1 docs/v1-stability.md sweep** (the missed ritual step): `KeyedRateLimiterConfig.Now`
   row ("added in v1.9.0, issue #45"), `ResponseRecorder.Push` note, `Capabilities`
   reclassified Frozen → Additive with the `Pusher` note, two new error-code rows
   (`ErrCodePushUnsupported`/`ErrCodePushFailed`), `ExpectNotModifiedDynamic` row, and a
   **v1.9.0 Versioning Policy bullet** (two behavior changes within unchanged signatures:
   CSRF QUERY safe-list, compression Content-Type sniffing restored + the additive surface).
3. **§f.4 `Capabilities.Pusher` probe** — field + `http.Pusher` probe in `DetectCapabilities`
   (`capabilities.go`), audit-decision comment updated (now cites Additive), Pusher assertions
   added to all three existing tests + new `TestDetectCapabilitiesReportsPusherForPushCapableWriter`
   (reuses `newPushRecorder`). Green under `-race`. Swept to README/arch-ref/FEATURES rows
   ("Hijacker/Flusher/Pusher") and a sentence on the CHANGELOG #47 entry.
4. **§f.5 `ConfigureNosurfHandler` QUERY asymmetry** — doc paragraph (only `CSRFMiddleware`
   safe-lists QUERY; the low-level configurator keeps nosurf's own safe list; register your own
   `ExemptFunc` if you build the handler that way) + pinning test
   `TestConfigureNosurfHandler_QueryStillChallenged` (403 through the low-level path while the
   middleware passes QUERY). nosurf import added to csrf_test.go. 4/4 QUERY tests green.
5. **§f.16 server_timing Push delegation — NO ACTION NEEDED (prior report's worry unfounded)**:
   `delegatingWriter.Push` already exists (`server_timing/server_timing.go:293`) with
   `TestDelegatingWriter_PushDelegates`, `TestDelegatingWriter_PushNotSupported`, and
   `TestServerTimingMiddleware_PreservesPusher`. The sub-module's delegating design was ahead
   of the root wrapper.
6. **§f.23 OTEL interop guide** — both writer-wrapper contract mentions (intro bullet + gotcha
   bullet) now name `http.Pusher` alongside Flusher/Hijacker; the matching AGENTS.md
   Observability-Lane bullet updated too.
7. **§f.14 httpspec opt-in decision — MADE and documented in the doc comment**:
   `ExpectNotModifiedDynamic` (like the static variant) stays opt-in via `WithExtraSpecs` —
   no default spec can assume the handler serves ETag-validated resources or which path.
   No `SpecName*` constant (those name standard-suite entries; these are caller-named extras).
8. **§f.6 + §f.17 lint pin v2.14.0 → v2.15.0 with the wrapcheck wave triaged**:
   - Reproduced the 47 v2.15.0 wrapcheck findings; tallied signatures: `.WithContextAny(` 24,
     `.WithContext(` 16, `.WithContextf(` 5, `.WithCause(` 2 — ALL errorfamily clone methods
     returning `*errorfamily.Error` (the classification wrapper itself): signature-based
     false positives, same class as the pre-existing `.Wrap(` exemptions.
   - Added the four `ignore-sigs` entries to `.golangci.yml` with an explanatory comment.
   - Verified **0 issues on BOTH v2.15.0 (local nix) and v2.14.0 (`go run @v2.14.0`)** with the
     one config; nested modules clean under v2.15.0 (stdlib-only, no errorfamily).
   - Bumped both pins together per RELEASE.md gate 6.6: ci.yml action `version: v2.15.0`
     (+ go1.27.4 comment) and release.yml `GOLANGCI_LINT_VERSION: v2.15.0` (+ comment).
   - RELEASE.md gate 6.6 prose updated (was "v2.14.0 since 2026-10-09").
   - `cue vet` (#CIWorkflow, #ReleaseWorkflow) passes on both edited workflows.
   - AGENTS.md Pre-Existing Lint Warnings section records the bump + triage + "fleet ripple is
     a TODO_LIST row, not this repo's gate"; CHANGELOG [Unreleased] Changed entry added.
9. **§f.15 mutation probes — both #49 guards proven mutation-sensitive** (backup → sed-mutate →
   targeted test → restore, each inside one command to stay under the daemon's sweep window):
   neutering `bodyAllowedForStatus` fails `TestCompression_NoContentTypeSniffed_WhenStatusForbidsBody`;
   neutering the Content-Encoding guard fails `TestCompression_NoContentTypeSniffed_WhenHandlerSetContentEncoding`.
   File byte-restored; sniff tests green after restore.
10. **CHANGELOG [Unreleased]**: added the "Post-fix follow-up sweep (session 2)" bullet and the
    lint-pin-bump bullet; appended the Pusher-probe sentence to the #47 entry.
11. **Release prep reading + skills loaded** for the queued phases: RELEASE.md (all 16 steps),
    apidiff/README.md regeneration ritual, prerelease-check.sh shape (clean-tree gate!),
    CHANGELOG compare-link structure, and the go-release (+multi-module reference), github-voice,
    docs-health, buildflow skills.
12. **Sub-module drift check for the v1.9.0 cut**: server_timing has ZERO commits since
    `server_timing/v1.0.3`; endpoints has ONE test-only commit since `endpoints/v0.1.0`
    (db6d8a8: qvalue corpus test + mirror — no exported-API movement). → **No sub-module tags
    with v1.9.0** (the v1.7.0 precedent); record that in the release section prose.

## b) PARTIALLY DONE

1. **§f.11 benchmark protocol — RUNNING (background job, started ~01:20)**: root-module 3s×5
   pass ~90% complete (240/265 result lines in /tmp/bench-root.txt) under the flake-pinned Go
   (prints go1.27.3 — the doc's method line says 1.27.1 and needs the refresh anyway); the
   httpspec and server_timing passes run sequentially after. NOT yet consumed: table refresh,
   `docs/benchmarks.bench.txt` regeneration, benchstat comparison vs 2026-10-25
   (allocations-first per the load note), and the §f.13 sniff micro-bench decision. Kept the
   machine quiet during the run (doc-only work in parallel).
2. **Session-tail state**: tree clean but 4 daemon commits unpushed (auto_push off) — planned
   to resolve inside the single release push; not yet verifiable-green on CI for today's heads.

## c) NOT STARTED (queued, deliberate order)

1. §f.11/13 benchmark consumption (blocked on the running job).
2. Full verification battery: `go test -race -shuffle=on -count=1` ×3 modules, `-race -count=10`
   parity, lint v2.15.0 + `config verify` ×3, all 6 erraudit gates, `scripts/pre-sweep-check.sh`,
   corpus-sync script, 3× apidiff gates.
3. `scripts/prerelease-check.sh` (needs a clean tree — daemon timing).
4. §f.24 `buildflow --build-mode dev` on the final tree (freshness check first per AGENTS).
5. §f.12/22 docs-health: annotate the 00-37 report's §f items done today; TODO_LIST rebuild
   (delete the buildflow row if green, add the fleet lint-ripple row); docs-gates battery.
6. §f.19 Release v1.9.0: CHANGELOG cut + `[1.9.0]:` link + `[Unreleased]` retarget, RELEASE.md
   step-9 annotation sweep, deliberate release commit (`--no-verify`), regen all three apidiff
   zips from the tag worktree, SSH-signed annotated tag, ONE push (master + v1.9.0), CI green
   watch on the tag, Release object verify, consumer probe build.
7. §f.3 Close #45–#49 with terse github-voice comments pointing at v1.9.0.
8. §f.7–10 consumer repos (after the tag): quotaboard CT-workaround simplification, CV refill
   pin on the `Now` seam, company-master-data `ExpectNotModifiedDynamic` adoption, cqrs-htmx
   native QUERY route (ADR-0064).
9. Final session-tail check + AGENTS lesson updates.

## d) TOTALLY FUCKED UP

Nothing shipped broken; all failures were caught at the tool boundary and fixed in-place:

1. **multiedit parameter inversion** — my first csrf_test.go edit passed the new text as
   `old_string` and vice versa; the tool rejected it (old_string not found); redone correctly.
2. **Two stale-read edit rejections** (README.md, architecture-reference.md) — the previous
   session's daemon commits had touched them after my context snapshot; re-read the exact
   lines, then edited. Cost: one round-trip each.
3. **`.golangci.yml` edit rejected** — hadn't Viewed it in this session; viewed, then edited.
4. **Introduced a typo into otel.md** (comma inside backticks: `` `http.Hijacker,` ``) in the
   same edit that added the Pusher mention — caught by re-reading my own diff one tool call
   later and fixed immediately.
5. **Over-120-column test line** — my first QUERY-asymmetry `t.Fatalf` was one line too long;
   the v2.15.0 lint run flagged it (golines); wrapped. (Nice side effect: the lint run doubled
   as proof the new config catches real findings.)
6. **First signature-tally grep returned nothing** (`\(\*` regex escaping); re-ran with a
   corrected pattern. Minor.

## e) WHAT WE SHOULD IMPROVE

1. **Edit-tool discipline** (3 avoidable rejections): View the exact target file immediately
   before editing; keep multiedit's old/new parameter order straight; wrap long lines at
   authoring time, not after golines complains.
2. **Verify-before-writing next-up lists**: §f.16's server_timing worry ("a Push there would
   also be silently dropped today") was disproven by one grep — the delegation + three tests
   already existed. A 30-second check at list-writing time beats a wasted todo item.
3. **`release.yml` `GOTOOLCHAIN: go1.27.1` may be stale** vs the current patch (bench ran
   1.27.3; local golangci built with 1.27.4): gate 6.6's concern is golangci export data
   (covered by the v2.15.0 bump), but the GOTOOLCHAIN pin exists so govulncheck scans a
   current-stdlib — needs a deliberate look at the v1.9.0 cut (queued; question g2).
4. **apidiff/README.md table** still stamps `httputil.export.zip` as "tag v1.7.0" while the
   regeneration example uses v1.8.0 — the table lags the actual regen; refresh it in the
   v1.9.0 regeneration commit.
5. **Daemon auto_push is off** — heuristic commits accumulate unpushed for hours (4 now).
   Harmless while a release push is planned, but any long-lived branch would silently lose
   CI coverage; worth an owner glance at the daemon config.

## f) NEXT UP (prioritized, ≤50)

1. Consume the benchmark job: refresh docs/benchmarks.md (root + httpspec + server_timing
   tables, method line: flake Go 1.27.3, measured date), regenerate `docs/benchmarks.bench.txt`,
   benchstat-compare vs 2026-10-25 (allocations first; ns/op load-noise caveats).
2. §f.13: if `BenchmarkCompression` allocations/ns moved measurably from the sniff, add
   `BenchmarkCompressionContentTypeSniff` micro-bench; else record "no movement" in the doc.
3. Verification battery (c2) — full race/shuffle ×3 modules, count=10 parity, lint + config
   verify ×3, erraudit ×6, pre-sweep, corpus, apidiff ×3.
4. `scripts/prerelease-check.sh` (catch a clean-tree window; --skip-flake if needed then
   `nix flake check` separately).
5. §f.24 `buildflow --build-mode dev`: `buildflow --version` vs `git -C ~/projects/BuildFlow
   rev-parse --short HEAD` first; expect ONLY policy-rejected residual classes.
6. §f.12/22 docs-health: strike the 00-37 report's §f items resolved today; TODO_LIST: delete
   the buildflow row (if green), add the fleet lint-ripple row, refresh the "Updated:" header;
   run the docs-gates battery.
7. §f.19 v1.9.0 release (full RELEASE.md order): CHANGELOG cut/links, FEATURES verify,
   step-9 annotation sweep (`rg -l 'GOEXPERIMENT|json/v2|...' docs/`), release commit
   (`--no-verify`), 3× apidiff zip regen from the tag worktree + README table refresh,
   SSH-signed annotated tag `v1.9.0`, ONE push `git push origin master v1.9.0`, `gh run watch`
   the tag CI, verify the Release object, consumer probe (`go get @v1.9.0` + main.go build).
8. §f.3 close #45–#49: one terse comment each ("Fixed in v1.9.0 — <one-line mechanism>,
   CHANGELOG [1.9.0]") in the github-voice comment register (no headers, no footer).
9. §f.7 quotaboard: bump httputil → v1.9.0, simplify `TestHTMLPagesCarryExplicitContentType`,
   run its suite.
10. §f.8 CV: bump → v1.9.0, rewrite `TestRateLimitRefillRatePinned` on `KeyedRateLimiterConfig.Now`
    (fake clock, no 6.2s sleep, drop `-short` skip), run its suite.
11. §f.9 company-master-data: bump → v1.9.0, adopt `ExpectNotModifiedDynamic`, delete the
    hand-rolled ETag describe block.
12. §f.10 cqrs-htmx: native QUERY route behind `CSRFMiddleware` (load the cqrs-htmx skill,
    read ADR-0064 first).
13. GOTOOLCHAIN pin decision (e3/g2) at the release cut.
14. Check tonight's nightly-fuzz run green (existing TODO row).
15. Session tail: `git log origin/master..master` empty, CI green on the exact head, tree
    clean, AGENTS lessons (e.g. the §f.16 lesson) recorded.

## g) QUESTIONS FOR THE OWNER

1. **Daemon auto_push is off — intended?** Four heuristic commits sit unpushed since 00:39
   (CI last ran on the 22:23 head). I will fold them into the single v1.9.0 release push
   regardless, but if auto_push-off is NOT intended, the daemon config needs a look before
   the next long session.
2. **`release.yml` `GOTOOLCHAIN: go1.27.1`** pins govulncheck's stdlib while the current patch
   is 1.27.3+ (golangci v2.15.0 is built with 1.27.4). Bump the pin inside the v1.9.0 cut
   (gate 6.6 "bump together with the Go patch"), or is 1.27.1 deliberately frozen until a
   security release demands it?
3. **Issue closing style for #45–#49**: terse per-issue comments + close right after the tag
   lands (my plan), or a single batch note + close, or hold until the GitHub Release object
   is verified? (External-visible bookkeeping; taste call.)

## Session facts

- Background job running: 3s×5 benchmark protocol (root ~90%, httpspec + server_timing
  queued) — killable via job id if the owner wants the machine freed.
- Tree clean; master 4 daemon commits ahead of origin (auto_push off); latest CI green on
  1587b51 (the final code head of the issue-fix session).
- No deliberate commits this session (daemon heuristic only, per the no-commit rule);
  the release commit will be the first deliberate one.
