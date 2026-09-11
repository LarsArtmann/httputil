# Session Status — 2026-09-11, 09:05 CET — TODO-List Execution + v1.0.x Full Code Review

Session scope: execute the live TODO_LIST (all priorities), run the scheduled full-code-review over the v1.0.x scope, and land everything verified. This report is a point-in-time snapshot of that run.

Working-tree note: all session work is committed by the auto-daemon (heuristic messages, expected). One untracked file from a parallel session sits in the tree (`docs/status/2026-09-11_09-01_starttls_listener_race_fix.md`, 09:01 — created after my review report at 08:59); left untouched, not part of this session.

---

## a) FULLY DONE (implemented + verified)

1. **v1.0.0 tag drift resolved** — discovered the drift was already shipped: `v1.0.1` (cb53786, contains the panic-free `Then(nil)`) is cut, pushed, and live on the module proxy; `origin/master..HEAD` was empty. No retag (owner directive recorded). The frozen `[1.0.0]` section carries the correction-of-record note.
2. **"We never retag" policy recorded** — new rows in `docs/DECISION_LOG.md` and a Tag Immutability section in `AGENTS.md` (never delete/re-cut a tag, even unpushed; drift ships as a patch release).
3. **Test-helper consolidation finished** — `waitForServerStart` (timeout heuristic) deleted; its 3 callers in `server_test.go` migrated to the deterministic `waitForListenerAddr`; race tests green.
4. **Three 0% `Code` constructors covered** — direct tests for `Code.Conflict`, `Code.Orchestration`, `Code.WrapRejection`; `go tool cover` shows all five constructor/Wrap pairs at 100%.
5. **CSRF attestation fuzz upgraded** — method parameter added to `FuzzCSRFMiddleware_OriginHeaders`, contradictory POST/PUT/PATCH/DELETE seeds, boundary seeds (absent/null/self origins), and a dual-channel oracle (403 + `csrf.origin_attestation_conflict` body). Mutation check: disabling the attestation check FAILS the seeds. ~2.7M execs clean. Found and fixed my own seed-order bug along the way (seeds must match the fuzz callback's parameter order — the first version silently skipped everything).
6. **Composition examples + benchmarks** — `ExampleCompose` (empty-list identity), `ExampleCompose_ordered`, `ExampleMiddlewareFunc_Then`, `BenchmarkCompose/Construct|Serve`, `BenchmarkMiddlewareStack_Middleware`. All testable examples pass.
7. **v1.1.0 stabilization batch** — removed `RateLimit()`, `RateLimitConfig`, `DefaultRateLimitConfig`, `RateLimiter`, `TokenBucketLimiter`, `NewTokenBucketLimiter`, their 4 error codes + templates, `BenchmarkTokenBucketLimiter*` rows, `FuzzEvictionTTL`, the deprecated-API redis integration doc, AND the `httputil.ETag()` adapter (removal window matured per ROADMAP; `etag.New` is the replacement; `BenchmarkETagAdapterOverhead` deleted; go-etag stays for the error-template superset). Migration doc, `v1-stability.md`, README, FEATURES, ROADMAP, SECURITY, DOMAIN_LANGUAGE, architecture-reference all reconciled. `go-error-family` v0.10.0 verified as the latest proxy version (no bump needed).
8. **architecture-reference.md re-inventoried** — deleted stale `etag.go` row; corrected `recorder.go` (Middleware alias, Status/WroteHeader), `server.go` (ListenerAddr), `compose.go` (Then(nil) semantics) rows; added missing `headers.go`, `httpspec/cors_ratelimit_specs.go` + its test, `server_timing/middleware.go`, `server_timing/doc.go` rows.
9. **Benchmark baselines re-measured (3s×5, all three modules)** — `docs/benchmarks.md` rewritten with the 2026-09-11 data (48 root benches, 7 httpspec, 6 server_timing), ETag rows gone, composition rows added, ID-amortization notes recomputed, header/method line synced.
10. **benchstat pinned in the flake** — `buildGoModule` from golang.org/x/perf (pseudo-version pinned, src+vendor hashes discovered), exposed as `packages.benchstat` (works via `nix run .#benchstat`) and in the devShell.
11. **KeyedRateLimiter regression sanity-check closed** — 195.6 ns/op isolated (5×3s) vs 220 ns/op in-suite, allocation profile identical (208 B/op, 4 allocs): the historical ~191-vs-220 delta is machine-state noise, not a regression. Recorded in benchmarks.md.
12. **CI hardening batch (5/5)** — (1) golangci `config verify` on both modules + a real server_timing lint job with the pinned binary (the 13-finding regression class is now structurally caught); (2) the Benchmark step uploads `bench.txt` (with `pipefail`) so "full raw data in CI artifacts" is finally true; (3) nightly-fuzz dispatched via `workflow_dispatch` and the `bug` label verified to exist; (4) `server-timing-standalone` flake check (`GOWORK=off GOPROXY=off CGO_ENABLED=0 go build+vet`); (5) `--skip-flake` flag on `prerelease-check.sh` (shellcheck clean).
13. **Nightly-fuzz workflow root-caused and fixed** — the scheduled runs have been failing since the sweep: unanchored `-fuzz` patterns match multiple targets now (`FuzzDecompression` vs `FuzzDecompressionInvariants`), and `FuzzEvictionTTL` stopped existing with the v1.1.0 removal. All 25 step patterns anchored (`^Name$`), stale step dropped, every pattern smoke-verified (`-fuzztime 1x`, 25/25 OK). False-positive auto-filed issues #7/#8 closed with root-cause comments.
14. **`scripts/doc-snippet-refs`** — new checker: parses ```go fences from README + docs/integrations, resolves every `httputil.*`/`servertiming.*`/`httpspec.*` selector against the real packages (source importer), exits 1 on drift. Mutation-verified against the v1.1.0 removals; wired into CI.
15. **gopls stdversion decision recorded** — stay on go 1.26 + `GOEXPERIMENT=jsonv2`; the 5 warnings are accepted noise until json/v2 stabilizes in 1.27 (DECISION_LOG row).
16. **CSRF docs follow-ups** — attestation 403 migration note in README (behavior change + client-side remediations), no-injection-header-reflection spec added to the httpspec block, `csrf.origin_attestation_conflict` surfaced in a consumer routing example, fuzz-target count corrected 26→25.
17. **Full-code review executed (scope: v1.0.x + today)** — 2 of 4 parallel architect passes completed (production core; fuzz/bench/pool), 2 lost to LLM rate limits (see b). Report: `docs/reviews/2026-09-11_08-59_full-code-review.html` — 12 findings fixed on the spot, 9 recorded with reasons, 4 patterns-to-preserve.
18. **Review fixes landed (all tested)** — stack.go data race fixed via atomic immutable snapshots + concurrent stress test (`-race -count=5` clean); CSRF attestation now honors `X-Forwarded-Proto` only from trusted proxies (fixes the TLS-termination false positive, 4 new boundary tests) and compares hosts case-insensitively (EqualFold); `CSRFTokenHXHeaders` HTML-escapes its JSON (XSS hardening); `Server.Start/StartTLS` double-start guard (`server.already_started`) + listener cleared on Serve/ServeTLS return (2 new tests); `Then(nil receiver)` applies as identity (nil next still 500-stubs); TokenValidation fuzz oracle now rejects every bogus pair incl. mismatched double-submit; `FuzzCompressWriterState` got a real bounded round-trip oracle + raw wire fuzzing; negotiator single-token micro-oracle; MaxBodySize fuzz reads bounded; `CodeOf` added (DomainOf reimplemented on top); `Add("")` rejected (`stack.name_empty`); id_generator doc lies fixed + `hexCharsPerByte` rename.
19. **All gates green at session end** — `go test -race` clean (9+ consecutive full-suite runs), `golangci-lint run` 0 issues, erraudit gates exit 0, govulncheck clean on both modules, `nix flake check` all checks passed, doc-snippet-refs clean.
20. **CHANGELOG [Unreleased] fully updated** — Added/Changed/Fixed sections covering the CSRF trust-model change, stack concurrency, server guard, CodeOf, doc-snippet-refs, benchstat, flake check, CI changes, and the nightly-fuzz root cause.

## b) PARTIALLY DONE

1. **Full-code review coverage** — 2 of 4 review passes completed; the "sweep test targets" pass (chain_test.go, server_test.go, ratelimit_keyed_test.go, id_generator tests) and the "scripts/examples/testutil" pass were lost to LLM rate limits and never re-run. Those scopes are covered only by my direct hands-on work this session (I edited or rewrote most of those files), not by an independent second pass.
2. **AGENTS.md Non-Obvious Behaviors update — interrupted mid-task.** The three NEW contracts (Server double-start guard, MiddlewareStack concurrency safety, the X-Forwarded-Proto trust addition to the attestation bullet) are in CHANGELOG but the AGENTS.md bullet updates were the exact task in flight when this report was requested. The old attestation bullet at AGENTS.md:195 still describes the pre-XFP scheme comparison.
3. **httpspec discovery push** — README block enriched (injection-header-reflection spec, routing examples), but no dedicated standalone runnable example and no docs-site page.
4. **TODO_LIST.md reconciliation** — not yet rewritten; today's completions and the 9 recorded review findings are documented in CHANGELOG + the review report but not folded back into the live backlog.
5. **Push** — the TODO explicitly lists `git push origin master`; not executed yet (was sequenced after review + docs; docs are done, AGENTS.md bullet update is not).
6. **Nightly-fuzz verification on the fixed workflow** — the dispatch during the session ran the OLD committed workflow (failed fast at step 1, as predicted, no new false-positive issue); the fixed workflow needs one post-push dispatch + first-step confirmation.

## c) NOT STARTED

1. AGENTS.md Non-Obvious Behaviors bullet updates (the interrupted item).
2. TODO_LIST.md rewrite (close ~14 done items, add the 9 recorded review findings + remaining work).
3. `nix fmt` + `nix flake check` re-run after the last wave of Go edits (treefmt vs golangci-fmt can disagree on the new files).
4. `go mod tidy` re-run + the 95% coverage-gate measurement after today's additions.
5. End-to-end `prerelease-check.sh` run (individual gates all pass; the script itself hasn't run clean start-to-finish on this tree).
6. Push master + post-push nightly-fuzz re-dispatch.
7. v1.1.0 release cut (tag/CHANGELOG freeze) — deliberately not done without an owner decision, since tags are forever.
8. The 9 recorded review findings as backlog entries (they live in the HTML report only so far).
9. go-compression extraction (preconditions unchanged: fresh plan inventory + owner repo setup).
10. go-error-family issue filing (owner action, draft exists).

## d) TOTALLY FUCKED UP (all caught before landing, listed for honesty)

1. **CSRF fuzz seeds silently did nothing for one full fuzz cycle** — I appended the method parameter at the END of `f.Add(...)` while the callback takes `method` FIRST, so every seed's "method" was an Origin URL and got skipped by the token filter. The mutation check exposed it. Lesson: after touching a fuzz target, always verify a seed actually reaches the interesting branch.
2. **My first Then(nil-receiver) fix introduced a nil-handler panic** (wrong check order: nil receiver + nil next returned nil). The fuzz seed pass caught it before any commit. My own review fix was briefly the worst code in the repo.
3. **Applied an agent suggestion without verifying the stdlib** — "reset the recorder" → `httptest.ResponseRecorder` has no `Reset`. Build caught it; reverted to the BenchmarkChain harness style.
4. **Created a duplicate `codeServerShutdownFailed` entry** in the template list while registering the new code; caught on sight, fixed by proper placement.
5. **One unexplained single FAIL in a full race suite run** mid-session (file edits + formatter daemon running concurrently); 9 consecutive clean runs (including `-count=5`) after. Almost certainly a mid-edit compile state, but it was never positively identified — if it reappears, treat as real.
6. **The dispatched nightly-fuzz run failed** (ran the old committed workflow) — expected, but it briefly auto-filed a false-positive issue (#8) before the fix landed; closed with explanation. The scheduled workflow has been silently broken since the sweep — the 03:04 failure today was pre-existing, not caused by this session.

## e) WHAT WE SHOULD IMPROVE

1. **Verify stdlib APIs before applying reviewer suggestions** (the Reset incident). Suggestions from sub-agents get the same skepticism as tool output.
2. **Fuzz discipline: mutation-check + seed-reach check after every oracle or behavior change** — both incidents (silent skips, stale oracle) were caught exactly by doing this; make it unconditional, not retrospective.
3. **Rate-limit-resilient agent orchestration** — spawn review agents sequentially or with retry; losing 2 of 4 passes to 429s left the review half-covered and unrecoverable until re-run.
4. **Behavior change ⇒ same-commit doc + oracle + corpus update.** The EqualFold change, oracle update, and corpus pin landed in sequence with the fuzzer flagging the gap in between; they should move as one unit.
5. **The auto-daemon's heuristic commits bury meaningful history** — every real change in this session is only describable via the CHANGELOG, not via git log (AGENTS.md already warns co-change analysis is unreliable here; this session is more evidence).
6. **Record-vs-fix judgment worked well** — 9 Frozen-API/behavior-change findings were documented instead of rushed; keep that bar for the v1.1.0 window.
7. **Finish-what-you-started ordering** — AGENTS.md was chosen as the next edit precisely when the report was requested; high-value-but-unstarted items should be either done first or explicitly deferred in a visible list (this report's section b/c).

## f) NEXT 50 (ordered: immediate housekeeping → release → backlog)

1. Finish the interrupted AGENTS.md Non-Obvious Behaviors update (double-start guard, stack concurrency, XFP trust note at the :195 bullet).
2. Rewrite TODO_LIST.md: close the ~14 completed items, add the 9 recorded review findings as post-v1.1 candidates.
3. `nix fmt` then `nix flake check` (treefmt pass over all new files).
4. `go mod tidy` + full build.
5. 95% coverage-gate measurement (`go test -coverprofile` + threshold checker).
6. End-to-end `prerelease-check.sh` run.
7. Push `master`.
8. Re-dispatch `nightly-fuzz`, confirm step 1 passes, close any new false-positive issue.
9. Re-run the 2 missing review passes (sweep test targets; scripts/examples scope).
10. Cut v1.1.0 (CHANGELOG freeze + tag + push) — owner-timed.
11. docs-health VERIFY pass over living docs after the release.
12. Harvest the 9 recorded findings into TODO_LIST (ValidateCSRF result type v2.0, security-config remediation fallback, TrustedOrigins parser unification, error reclassification, ValidateCSRF mutation docs, pool provenance, typed middleware names, BuildValidated, MiddlewareFunc-canonical discussion).
13. Add `server.already_started` + `stack.name_empty` to the architecture-reference error-classification table.
14. SECURITY.md: document the XFP-from-trusted-proxies trust addition.
15. CSRF attestation fuzz: add a TrustedProxies variant so the XFP paths get fuzz coverage (the unit tests pin them; the fuzzer doesn't reach them).
16. Decide/document X-Forwarded-Proto list semantics (first-hop choice) in the CSRF doc comment + DOMAIN_LANGUAGE.
17. Verify nosurf's method-case handling (is `post` unsafe for nosurf?) and pin with a test if divergent from `isUnsafeCSRFMethod`.
18. Consider `sync.Once` idempotency for `RegisterErrorClassifications`.
19. Key `errorTemplates` by `Code` (or add a round-trip test) to kill typo-orphaned templates.
20. Pool hardening: reject `(nil, nil)` factory returns in the probe; store the factory in `writerPool` so `acquire` can't diverge; owner-tag pooled writers in `release`.
21. Add `doc-snippet-refs` + a fuzz seed-run to `prerelease-check.sh`.
22. Document `Chain`/`Compose` nil-element behavior (currently panics at apply time for `Compose(nil)`).
23. Cache the last snapshot in `MiddlewareStack.Middleware()` if per-apply allocation ever matters (measure first).
24. httpspec: dedicated runnable example (executable on pkg.go.dev) + docs-site page.
25. Recurring httpspec/server_timing baseline re-measure cadence (they're now IN benchmarks.md; keep them fresh).
26. go-compression extraction: refresh the plan inventory (precondition #1), then execute per phased-commit discipline when the owner fires the trigger.
27. File the go-error-family issue (owner action; draft at docs/planning/2026-09-10_…issue-draft.md).
28. Owner decision: `CSRFConfig.withParsedTrustedProxies` export.
29. architecture-review re-run post-v1.0 (composability + full structural pass).
30. gopls stdversion: when Go 1.27 ships, drop GOEXPERIMENT + bump go.mod in one change.
31. Triage open issue #4 (CORS gzip for absent Accept-Encoding — RFC 7231 identity question).
32. CI: add a concurrency group to cancel superseded runs; consider Go build caching for job speed.
33. Commit representative fuzz corpus entries (testdata/fuzz) for the strengthened oracles as permanent regression seeds.
34. benchmarks.md: add a benchstat A/B workflow example (old.txt vs new.txt).
35. benchmarks.md: note single-machine drift risk (the 191-vs-220 incident).
36. Nightly-fuzz: consider `-fuzzminimizetime` and per-target time budgets review (75 min total today).
37. Consider splitting the nightly workflow into matrix jobs so one target crash doesn't hide later targets.
38. README: re-read the attestation migration note after v1.1.0 ships (verify claims match the tagged behavior).
39. DOMAIN_LANGUAGE.md: add "effective request scheme" / "attestation" entries.
40. ResponseRecorder-like helper for `ValidateCSRF` returns (internal candidate for v1.x additive API: `ValidationResult`).
41. Evaluate `errors.AsType` usage audit (go-error-modernization policy) on the new test code.
42. Add fuzz seed-run (`go test -run Fuzz`) to CI's test job so corpus regressions surface outside the nightly.
43. Check whether `X-Forwarded-Proto` should also feed `SetPlaintextHTTPOrigin`/`shouldBypassPlaintextOrigin` for consistency, or document why not.
44. Update `docs/architecture-reference.md` lint-profile section if any linter config changes ship in v1.1.0.
45. Tag the review report's deferred items with docs-health HARVEST conventions (inline markers, not appendix-only).
46. Consider a CHANGELOG "Behavior changes" summary block for v1.1.0 (removals + XFP + EqualFold + Add("") in one scannable list).
47. Verify the httpspec sub-module is untouched by the root removals (it is — but assert in the release checklist).
48. Run `go vet` on the server_timing module after the root changes (it's independent; confirm zero blast radius — expected).
49. Housekeeping: delete `/tmp` bench artifacts after recording (they're the raw data behind benchmarks.md; consider committing them under docs/benchmarks/data/ instead of losing them).
50. Owner question follow-ups (below) folded into TODO_LIST once answered.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Release timing:** should I push master now and let the v1.1.0 removals sit as `[Unreleased]`, or do you want the v1.1.0 tag cut and pushed in the same motion (the TODO's push step lists only master + v1.0.0, which is already pushed)? Tags are forever, so I won't cut v1.1.0 without your explicit go.
2. **Review coverage:** the two lost review passes (sweep test targets, scripts/examples) — re-run them now as agents, or do you accept my direct-authorship coverage of those files for the v1.0.x review cycle?
3. **CSRF validate-and-log vs remediate:** for security-degrading configs (`SameSite=None` + `Secure=false`), is log-and-continue the final contract, or should v1.1.x remediate to secure defaults (a behavior change for anyone currently running that misconfiguration)?
