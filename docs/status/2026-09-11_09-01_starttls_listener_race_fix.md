# Status Report — StartTLS/Start Listener-State Race Fix

- **Generated:** 2026-09-11 09:01 CEST
- **Scope:** This session only — diagnosis and fix of the flaky `TestServer_StartTLS_ListenerClearedOnCertFailure` failure surfaced by BuildFlow step `test-race`. No unrelated research was done, per instruction.
- **Session verdict:** Root cause found, fixed at the contract level, all gates green. Two honest gaps remain (public godoc, CHANGELOG) and one adjacent pre-existing issue was noticed but deliberately not touched (go1.26/go1.27 stdversion drift).

---

## Direct answers: What did I forget? What could I have done better? What could I still improve?

**What I forgot:**

1. The public godoc on `Start`/`StartTLS` still does not state the new guarantee ("receiving an error implies the server is no longer listening"). I recorded it in AGENTS.md (agent memory) but _not_ where library users read it. The guarantee is now an API contract; contracts belong in godoc.
2. A `CHANGELOG.md [Unreleased]` entry for the fix. The repo freezes `[version]` sections at tag time, so release-time backfill means relying on memory — I should have written the entry at fix time.
3. I ran `nix fmt` but never `nix flake check` (the full flake gate) after the change.

**What I could have done better:**

1. I fixed the two known instances (`Start`, `StartTLS`) but did not sweep the codebase for the _class_ of bug ("state cleanup ordered after a buffered-channel send"). The fix was instance-driven, not pattern-driven.
2. I claimed "the failure state is settled" but did not pin one subtle consequence with a test: after a _failed_ `StartTLS`, is a retry `Start`/`StartTLS` guaranteed to succeed? My fix resets `started` before the error send, so yes — but the existing test only asserts `ListenerAddr`. A restart assertion would pin the stronger claim.
3. My first AGENTS.md edit bounced ("read before edit") — I had the content in project context but the tool requires an explicit View. Small, avoidable round trip.

**What I could still improve:**

1. The shutdown-drain semantics: `started.Store(false)` runs when `Serve` returns (immediately on listener close), while in-flight handlers may still be draining inside `httpServer.Shutdown`. A `Start` issued in that window can succeed while the old server is still draining. This is pre-existing (the old `defer` reset at the same point), unchanged by my fix, and undocumented.
2. Turn the AGENTS.md lesson into enforcement (regression tests already pin it; a lint rule for "send-after-cleanup" ordering is realistically not expressible with current linters — documented invariant + tests is the honest best available).
3. Race soaks are reactive (CI caught it). A scheduled `-race -count=25 -shuffle=on` soak would catch this class proactively — owner decision (see g3).

---

## a) FULLY DONE

| #   | Item                                                                                                                                                                                                                                                                          | Evidence                                                                                      |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| a1  | Root cause identified: in the `StartTLS` goroutine, the error was sent into the buffered `errChan` _before_ the deferred `clearListener()`/`started.Store(false)` ran; the test received the error and read a stale live listener inside that window (no happens-before edge) | Pre-fix `server.go:269-284`; failure log `server_test.go:819` (got `(127.0.0.1:41079, true)`) |
| a2  | Fix applied to **both** `Start` and `StartTLS`: `clearListener()` + `started.Store(false)` now run **before** the channel send, so the send creates the happens-before edge — receiving an error implies `ListenerAddr()` reports not-listening                               | `server.go:222-234` (`Start`), `server.go:271-287` (`StartTLS`)                               |
| a3  | Targeted race verification: server lifecycle tests ×10 with race detector + shuffle                                                                                                                                                                                           | `go test -race -count=10 -shuffle=on -run 'TestServer_StartTLS_…                              |
| a4  | Full suite race-verified, twice: `-race -shuffle=on ./...` and `-race -count=5 -shuffle=on ./...`                                                                                                                                                                             | all modules ok                                                                                |
| a5  | `golangci-lint run` (≈70 linters): **0 issues**                                                                                                                                                                                                                               | tool exit clean                                                                               |
| a6  | erraudit real gates clean: `--type legacy_as` and `--type stdlib_constructor --enforce-go-error-family`                                                                                                                                                                       | both exit 0                                                                                   |
| a7  | `nix fmt` (treefmt): 0 changes — edit was format-clean                                                                                                                                                                                                                        | tool output                                                                                   |
| a8  | The originally failing BuildFlow step re-run: `buildflow -s test-race` → ✓ passed (1/1)                                                                                                                                                                                       | tool output                                                                                   |
| a9  | Invariant + anti-pattern warning recorded in AGENTS.md (Non-Obvious Behaviors), including "never move cleanup back into a post-send defer"                                                                                                                                    | AGENTS.md, first bullet of section                                                            |
| a10 | The existing test now serves as a deterministic regression test (its claim was correct; the implementation raced)                                                                                                                                                             | `TestServer_StartTLS_ListenerClearedOnCertFailure` passes 10/10 under race+shuffle            |

## b) PARTIALLY DONE

| #  | Item                                       | Done                                                | Open                                                                                                                                                                                                                                                                                                                                                  | Effort |
| -- | ------------------------------------------ | --------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| b1 | Documentation of the new guarantee         | AGENTS.md entry written                             | Public godoc on `Start`/`StartTLS` still silent about the contract; `docs/architecture-reference.md` lifecycle wording not cross-checked                                                                                                                                                                                                              | S      |
| b2 | Regression coverage for the fixed contract | Cert-failure path pinned (`ListenerAddr` assertion) | No restart-after-failure assertion (stronger claim: retry `StartTLS` succeeds because `started` reset precedes the error send); no equivalent forced-failure test for `Start`'s `Serve` error path (forcing a non-`ErrServerClosed` `Serve` failure needs listener injection — genuinely hard, not an excuse, a scoping decision I made unilaterally) | M      |
| b3 | Post-fix hygiene                           | Lint, erraudit, fmt, test-race step all green       | Full `nix flake check` and a full `buildflow --build-mode dev` run not executed this session                                                                                                                                                                                                                                                          | S      |

Blockers: none — all open items are unblocked.

## c) NOT STARTED

| #  | Item                                                                                                                                              | Why not started                                                                       | Priority |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- | -------- |
| c1 | `CHANGELOG.md [Unreleased]` entry for the race fix                                                                                                | Forgotten at fix time; user instruction was to fix, verify, and wait                  | Medium   |
| c2 | Codebase sweep for the anti-pattern class (cleanup ordered after buffered-channel send; grep `errChan <-` / lifecycle channels + adjacent defers) | Fix was instance-driven; sweep is a follow-up quality pass                            | High     |
| c3 | Probe test pinning or fixing the shutdown-drain window (`Start` succeeding while old server drains)                                               | Pre-existing semantics, unchanged by this fix; needs a design decision first (see g1) | High     |
| c4 | Race soak in CI/nightly (`-race -count=25 -shuffle=on`)                                                                                           | Resource/policy decision — see g3                                                     | Medium   |

Nothing in (c) is blocked by missing information except c3, which needs g1 answered.

## d) TOTALLY FUCKED UP

| #  | What                                                                                                                                                                                                                                                                                                                                                                                                                                                         | Severity                                                                                                                                                                                             | Root cause                                                                                                                          | Status / Mitigation                                                                                                                                                                       |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| d1 | **(This session's bug — now fixed)** The shipped `Start`/`StartTLS` API lied about state: a caller that received a `StartTLS` cert-load error could still observe `ListenerAddr() == (addr, true)` — a closed listener reported as live. Every library consumer was exposed, not just the test.                                                                                                                                                              | Medium at the time (state-query lie only: the socket _was_ closed by the stdlib, so no leak/security/data impact — but the query contract was broken and flaky red CI undermined trust in the suite) | Cleanup ordered after a buffered (non-blocking) channel send — the send completed before the `defer` ran, so no happens-before edge | **Fixed deterministically** (a2); regression test passes 10/10 under race+shuffle                                                                                                         |
| d2 | **(Pre-existing, noticed, NOT touched — out of session scope)** gopls flags 5 stdversion warnings: production files `health.go:83`, `csrf.go:790` and test files `health_test.go:136,146`, `validate_config_log_test.go:33` use go1.27 stdlib APIs (`json.MarshalWrite`, `json.Unmarshal[1.27]`) while the module declares go1.26. Builds/tests pass (toolchain accepts it), but the module's declared language version and the APIs actually used disagree. | Low-Medium (doc/toolchain drift; will bite on a real go1.26 toolchain)                                                                                                                               | go.mod `go` directive vs actual API usage                                                                                           | Needs owner decision: bump go.mod to 1.27 (AGENTS.md says "Go 1.26+", so bumping is plausible) or downgrade the API calls. Deliberately untouched: files I didn't change, per repo rules. |

Nothing is currently red: the suite is green under race, lint is at 0 issues. (d) is honest history + one adjacent landmine.

## e) WHAT WE SHOULD IMPROVE

| #  | Pattern / practice                                                                     | Impact                                                                  | Concrete fix                                                                                                             |
| -- | -------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| e1 | Concurrency/state guarantees documented only in AGENTS.md (agent memory), not in godoc | Every external consumer is blind to the contract                        | Add the guarantee sentence to `Start`/`StartTLS` godoc; audit other channel-returning APIs for unstated state guarantees |
| e2 | Bug fixes without a same-session CHANGELOG `[Unreleased]` entry                        | Release-time backfill relies on memory; drift against the Freeze Policy | Write the entry at fix time (c1)                                                                                         |
| e3 | Instance-driven concurrency fixes                                                      | Same bug class survives elsewhere                                       | Post-fix sweep checklist: grep every lifecycle channel send and verify ordering of adjacent state cleanup (c2)           |
| e4 | Flake detection is reactive (CI caught it once)                                        | Low-probability races ship                                              | Scheduled race soak with recorded shuffle seeds (c4)                                                                     |
| e5 | go.mod / stdlib API version drift (d2)                                                 | 5 standing warnings; real go1.26 toolchain breakage risk                | One decision (g2), then either `go mod` bump or API downgrade — 5 warnings disappear                                     |
| e6 | Test asserts only the immediate claim, not the implied stronger claim                  | "Listener cleared" tested; "retry works" untested                       | Add restart-after-failure assertion (b2)                                                                                 |
| e7 | My own loop: editing from project context without an explicit View                     | One wasted round trip                                                   | Always View before Edit, even for "already known" files                                                                  |

## f) NEXT TASKS (up to 50 — brainstorm ranked by impact; most extras are ROADMAP fuel for `docs-health` HARVEST)

Impact: Critical / High / Medium / Low. Effort: S <30min / M 30min–2h / L >2h.

| #  | Task                                                                                                                                                                          | Impact   | Effort | Category      |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | Decide shutdown-drain semantics (g1): make `Shutdown` wait for the Serve goroutine (e.g. `sync.WaitGroup`) so `started`/listener are guaranteed reset when `Shutdown` returns | Critical | M      | Quality       |
| 2  | Add restart-after-failure assertion to `TestServer_StartTLS_ListenerClearedOnCertFailure` (second `StartTLS` with valid certs succeeds)                                       | High     | S      | Quality       |
| 3  | Add the guarantee sentence to `Start`/`StartTLS` godoc ("an error delivered on the channel implies the server is no longer listening and `Start`/`StartTLS` may be retried")  | High     | S      | Documentation |
| 4  | Sweep codebase for cleanup-after-send pattern (all lifecycle channels: server, timeout, health)                                                                               | High     | S      | Quality       |
| 5  | Probe/pin the drain window: `Start` during `httpServer.Shutdown` drain (depends on #1)                                                                                        | High     | M      | Quality       |
| 6  | `CHANGELOG.md [Unreleased]` entry for the listener-state race fix                                                                                                             | Medium   | S      | Documentation |
| 7  | Run `nix flake check` (full gate) on the fixed tree                                                                                                                           | Medium   | S      | Quality       |
| 8  | Run full `buildflow --build-mode dev` to confirm zero regressions across all steps                                                                                            | Medium   | S      | Quality       |
| 9  | Resolve go1.26 vs go1.27 stdlib API drift (d2, depends on g2): bump module or downgrade `json.MarshalWrite`/`json.Unmarshal` usages in `health.go:83`, `csrf.go:790`          | Medium   | S      | Cleanup       |
| 10 | Cross-check `docs/architecture-reference.md` lifecycle/error sections reflect the cleanup-before-send ordering                                                                | Medium   | S      | Documentation |
| 11 | HARVEST this report's section (f) into `TODO_LIST.md` / `ROADMAP.md` via docs-health                                                                                          | Medium   | S      | Documentation |
| 12 | Forced-failure regression test for `Start()`'s `Serve` error path (listener injection or direct close)                                                                        | Medium   | M      | Quality       |
| 13 | Add `StartTLS` in-memory-cert failure test (`GetCertificate` returns error) to cover the non-file failure branch                                                              | Medium   | S      | Quality       |
| 14 | Race-stress test: Start/StartTLS/Shutdown loop with high `-count` and short server lifecycles                                                                                 | Medium   | M      | Quality       |
| 15 | Port-rebind probe: after failed `StartTLS`, the ephemeral port is immediately reusable (OS-level close confirmed)                                                             | Medium   | M      | Quality       |
| 16 | Audit `server_test.go` waits (`waitForListenerAddr` polling, `time.After` selects) for other flake-prone timing                                                               | Medium   | M      | Quality       |
| 17 | Nightly/CI race soak: `-race -count=25 -shuffle=on`, echo the shuffle seed into the log (depends on g3)                                                                       | Medium   | M      | Quality       |
| 18 | Document `ListenerAddr` nuance: during graceful drain the flag can flip before `Shutdown` returns (or fix via #1)                                                             | Low      | S      | Documentation |
| 19 | Verify `ErrServerClosed` filtering cannot swallow a real error during a shutdown race                                                                                         | Low      | M      | Quality       |
| 20 | `Shutdown`-then-`StartTLS` restart test (fresh TLS serve after clean shutdown)                                                                                                | Medium   | S      | Quality       |
| 21 | Review `errors.go` template for `server.already_started`: message should mention retryability after a delivered error                                                         | Low      | S      | Documentation |
| 22 | Annotate any older `docs/status/` reports that mention this flake (docs-health ANNOTATE: `~~item~~ done at <hash>`)                                                           | Low      | S      | Documentation |
| 23 | Check `FEATURES.md` for lifecycle-guarantee inventory and update if present                                                                                                   | Low      | S      | Documentation |
| 24 | README lifecycle section: mention the error-implies-not-listening guarantee (sales page honesty)                                                                              | Medium   | S      | Documentation |
| 25 | Ensure next release checklist (docs/RELEASE.md flow) includes this fix in its CHANGELOG section before tagging                                                                | Medium   | S      | Documentation |
| 26 | Record shuffle seeds in BuildFlow `test-race` output for forensic reproduction                                                                                                | Low      | S      | Quality       |
| 27 | `go vet ./...` explicit run (golangci covers it; AGENTS.md lists it as a gate)                                                                                                | Low      | S      | Quality       |
| 28 | Confirm the two rewritten goroutine comments are accurate post-edit (ServeTLS clone comment placement)                                                                        | Low      | S      | Cleanup       |
| 29 | Consider `t.Attr` seed echo in test binaries for flake forensics                                                                                                              | Low      | S      | Quality       |
| 30 | Evaluate whether `errServerAlreadyStarted` (Rejection) is the right classification for a Start-after-failed-Start retry race                                                  | Low      | S      | Documentation |
| 31 | Benchmark sanity: confirm removing the `defer` has no measurable Server lifecycle cost (or note lifecycle isn't benchmarked and leave it)                                     | Low      | S      | Quality       |
| 32 | Add an `Example` for Start/StartTLS lifecycle with error handling (with `// Output:` per testableexamples)                                                                    | Low      | S      | Documentation |
| 33 | Document in architecture-reference that `started`/listener reset happens-before error delivery (cross-ref AGENTS.md bullet)                                                   | Low      | S      | Documentation |
| 34 | Review whether any other public API returns a channel whose send should imply settled state (systematic contract audit)                                                       | Medium   | M      | Documentation |
| 35 | Add `docs/status` archived-movement check: this report to be moved/annotated when its items resolve                                                                           | Low      | S      | Documentation |
| 36 | Evaluate golangci `predeclared`/custom rule feasibility for send-after-defer ordering (likely infeasible — record the negative decision so it isn't re-tried)                 | Low      | M      | Quality       |
| 37 | Confirm `server_timing` sub-module untouched and green in the next full pipeline run                                                                                          | Low      | S      | Quality       |
| 38 | Re-run `erraudit` full `--type-aware` sweep on the changed file set (advisory-only) to confirm no new advisories                                                              | Low      | S      | Quality       |

Items 39–50 deliberately left empty rather than padded with vague filler; the guide is explicit that vague items are not actionable and would pollute HARVEST.

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Shutdown-drain contract:** Should `Shutdown()` block until the Serve goroutine has fully finished (so `started`/listener are guaranteed reset when `Shutdown` returns), even though that changes when `Shutdown` returns? Or is the current semantics (state resets when `Serve` returns, mid-drain) intended? I cannot decide this from code or docs — it is an API contract choice with real tradeoffs (blocks #1, #5, #18).
2. **Go version policy (d2):** Production files already use go1.27 stdlib APIs while the module declares go1.26. Do you want `go.mod` bumped to 1.27 (consistent with "Go 1.26+" and the actual API usage), or should the API usages be downgraded for a true go1.26 floor? This sets the minimum-supported toolchain for every consumer.
3. **Race-soak policy (c4):** Do you want a scheduled race soak (e.g. nightly `go test -race -count=25 -shuffle=on ./...` in CI/BuildFlow) accepting the extra compute minutes, to catch remaining timing flakes of this class proactively? CI caught this one only by luck of scheduling.

---

_Point-in-time snapshot — 2026-09-11 09:01 CEST. Section (f) is the input for `docs-health` HARVEST into `TODO_LIST.md`/`ROADMAP.md`; do not let it die in this timestamped file._
