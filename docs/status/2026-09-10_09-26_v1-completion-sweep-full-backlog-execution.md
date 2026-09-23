# Status: v1.0 Completion Sweep — Full Backlog Execution

**Date:** 2026-09-10, 09:26 CEST
**Session scope:** executed the entire TODO_LIST backlog (27 tracked tasks), cut v1.0.0 locally.
**Numbers:** 63 files changed, +4,329 / −892 across 27 daemon+deliberate commits since `d70c593`; annotated tag `v1.0.0` on HEAD (NOT pushed).

---

## a) FULLY DONE

Each item below was implemented AND verified (race detector, lint, erraudit, or execution as appropriate).

1. **Baseline established** — build, vet, `go test -race`, golangci-lint (0 issues), both erraudit real gates green before any change.
2. **`writerPool` resettability probe** (`compress_pool.go`): construction-time probe of the factory; non-resettable factories skip the pool entirely (kills the wasted per-request pooled allocation). Deterministic tests: skip-path factory-call-count, pool-path sentinel, destination Rebind proof (gzip round-trip through the recycled writer), probe decision pin, release-drop. Panic-on-factory-error test updated to construction time. Pool-type-mismatch errors keep classification + context.
3. **`Server.ListenerAddr() (net.Addr, bool)`** (`server.go`): `Start`/`StartTLS` bind via `net.ListenConfig.Listen` first (bind errors immediate on the channel), listener tracked under a mutex, cleared on successful `Shutdown`. Three new tests (not-listening false, ephemeral-port resolution + request + post-shutdown false, TLS test migrated to `:0`). `Addr()` documented as never rewritten.
4. **TLS test modernization**: RSA-2048 → Ed25519 self-signed cert (PKCS#8), `waitForTLS` moved to `testutil_test.go` and now fail-fasts on startup errors and dials the RESOLVED address, `reserveFreePort` deleted (obsolete — it was a port-reuse race).
5. **`TestChain_DecompressionThenMaxBodySizeLimitsDecompressed`** now asserts the classified `errDecompressionSizeExceeded` sentinel directly instead of treating the handler's 417 as the signal.
6. **KeyExtractor `""` semantics** — docs verified unambiguous (exempt/always-allowed vs nil = shared global key, both stated on type + config field). No change needed; item closed as verified.
7. **`Code.WrapConflict` / `Code.WrapOrchestration`** added (`code.go`) — constructor + Wrap pairs now cover all six families; cause+family tests for both.
8. **KeyedRateLimiter property tests** (`ratelimit_keyed_test.go`): seeded 2,000-op churn above `MaxKeys` asserting map/heap size equality, heapRef index/identity, per-key tracking, and min-heap ordering every 97 ops; deterministic sliding-window test (unique keys in time order → survivors are exactly the newest `MaxKeys`); `BenchmarkKeyedRateLimiter_MaxKeysChurn` (fresh key per op = true eviction path).
9. **`FuzzCORSOriginEcho`** (`cors_fuzz_test.go`): never reflect a third-party origin; exact-allowlist+DenyUnmatched oracle (echo ⟺ allowlisted, silence otherwise); ~1.4M execs clean.
10. **`FuzzNegotiatorWireFormat`**: raw Accept-Encoding fuzzing — always succeeds with identity registered, results are registered canonical encodings, q ∈ (0,1] (verified against the q-value parser before writing the invariant); ~970K execs clean. Gzip multistream reliance documented in `FuzzCompression`'s round-trip comment.
11. **`FuzzMaxBodySize` + `BenchmarkMaxBodySize`**: pins the probed stdlib contract (read error ⟺ `len(body) > max(limit,0)`) with `errors.AsType[*http.MaxBytesError]`; `maxbodysize.go` doc updated (zero AND negative reject every non-empty body, empty body passes).
12. **Four runnable examples**: `ExampleMetrics` (with recorder double), `ExampleHealthHandler` (exact bytes), `ExampleServer` (real bind + `ListenerAddr` + context-carrying request + graceful shutdown), `ExampleMiddlewareStack`. `ExampleRateLimit` formally declined (deprecated API must not gain discoverable examples).
13. **ID-generator refill benchmarks**: `BenchmarkIDGeneratorRefillSwap` (alloc + crypto/rand + atomic publication, `ns/ID-amortized` metric) vs `BenchmarkIDGeneratorRefillRawRandRead` baseline → swap overhead ≈5 ns/ID over ~18.6 ns/ID syscall amortization.
14. **Nonce design decisions**: declined `NonceConfig.Generator` override and public `GenerateNonce` (rationale in `nonce.go` doc comment + DECISION_LOG entry; `WithNonce` is the injection escape hatch). Four composition tests: nonce × Compression (CSP nonce equals body nonce after gzip round-trip), nonce × CORS, nonce × ServerTiming (`WrapServerTiming` wrapper keeps context nonce + both headers), nonce × CSRF token helpers.
15. **ID-generator generation-swapped ring** (`id_generator.go`): published generation buffers immutable; monotonic generation-stamped claims (slot s → gen s/256, offset s%256) so no slot is ever drawn twice; superseded generations GC-reclaimed; the formal copy-vs-refill data race is eliminated. Stress test: 8 goroutines × 20k IDs under `-race`, asserting BOTH full-ID and random-tail uniqueness. DECISION_LOG entry records the design and the rejected alternatives.
16. **`bench_batch_test.go` dissolved** into per-middleware bench files (`id_generator_bench_test.go`, `csrf_bench_test.go`, `server_bench_test.go`, additions to `ratelimit_keyed_bench_test.go`); two stale `b.ResetTimer` calls removed in the move.
17. **AGENTS.md slimmed 58.5 KB → 29.3 KiB**: file-export tables, Error Classification table, and lint profile moved to new `docs/architecture-reference.md` (three rows refreshed for today's API changes); Hard Constraints / Non-Obvious Behaviors / Testing Conventions kept intact per the item's constraint.
18. **CI workflows** (all YAML-validated):
    - `ci.yml`: coverage gate now `scripts/coverage-threshold` (Go, library packages only, fails loudly on malformed reports) replacing the awk; govulncheck for `server_timing`; new `commit-lint` job (`scripts/check-commit-lint.sh` over `base..head`); `GOTOOLCHAIN` 1.26.6 → 1.26.7 (the old pin was OLDER than go.mod's requirement — CI was broken).
    - `release.yml`: build + vet + race tests for BOTH modules, lint, govulncheck both modules, then GitHub Release.
    - `nightly-fuzz.yml`: 3 new fuzz targets + crash-issue step (run-log link, corpus pointer, `issues: write`, label fallback).
19. **Release automation**: `scripts/prerelease-check.sh` (nine gates, both modules), `docs/RELEASE.md` updated to reference the automation + library-only coverage command, `scripts/check-commit-lint.sh`, `scripts/coverage-threshold`.
20. **flake.nix**: `nix run .#bench` = the documented 3s×5 protocol (smoke-run verified); `dprint` in the devShell (verified 0.56.1 on PATH). `nix flake check` green.
21. **`CSRFConfig.Validate` cleanup**: pure value receiver (no mutation, no logging), entry validation kept as pure checks; `withParsedTrustedProxies()` parses at construction (any parse failure → empty trust list, safer than the old partial-parse-on-error state); Secure=false warning moved to the constructor; all `CSRFConfig` methods normalized to value receivers (recvcheck). Fuzz target upgraded to cross-check Validate-accept ⟺ parse-exactly-one; two new unit tests (populate + invalid-yields-empty). Landed BEFORE the tag — the last cheap moment for an API-semantics cleanup.
22. **Integration docs**: samber-do / huma / prometheus verified against current APIs; brotli-zstd gained the pool-skip semantics; redis deprecation notice corrected ("removed at v1.0" → post-v1.0 stabilization release); new `docs/integrations/compose-bundles.md` (bundle pattern, stack nesting, ordering rules, `MiddlewareFunc.Then` contract) linked from README.
23. **`docs/benchmarks.md` re-measured end-to-end**: full 3s×5 on post-sweep code, ~51 rows refreshed (health handlers now show the newline write; ID gen 108.8 ns/op with the ring; keyed-limiter true eviction path 417 ns/op; deprecated rows labeled). Missing/failed benchmark rounds inventoried and re-run to 5-round completeness.
24. **go-error-family upstream draft** (`docs/planning/2026-09-10_go-error-family-conditional-request-classification-issue-draft.md`): all verify-before-filing gates executed — source grep (no 304/412/conditional guidance anywhere), family table review (400/409/503/500 only), TODO/ROADMAP/FEATURES grep (untracked), `HTTPStatuser` escape hatch identified in `http.go`; documentation-only proposal with code example and filing checklist.
25. **go-compression extraction**: formally deferred post-v1.0 with preconditions recorded in TODO_LIST (v1.0 precedence; today's writerPool refactor invalidated the plan's line-by-line inventory — the plan's own anti-Verschlimmbesserung guardrail requires a fresh pass; new-repo/remote setup is owner work).
26. **v1.0.0 cut locally**: CHANGELOG `[Unreleased]` → `[1.0.0] - 2026-09-10` (sections normalized to Added 17 / Changed 51 / Fixed 13; duplicate-heading misfire fixed; link checker green with `[1.0.0]` compare link + fresh `[Unreleased]`), fresh empty `[Unreleased]`, FEATURES.md refreshed (rows, fuzz/bench/example counts, server lifecycle, error-model section), TODO_LIST rewritten with per-item evidence, release-prep commit + annotated `v1.0.0` tag on HEAD, tree clean.
27. **Final verification sweep**: both modules — build, vet, `-race -count=1` green; golangci-lint 0 issues; both erraudit gates green; `nix fmt` clean; `nix flake check` green; 97.2% coverage through the new threshold checker; `GOWORK=off` consumer-style build/vet/test green; `go mod tidy` produces no diff (root go.mod safe to publish, same-repo `replace => ./server_timing` ships inside the module zip).

## b) PARTIALLY DONE

1. ~~**v1.0.0 push** — tag + release-prep commit are local; `git push origin master && git push origin v1.0.0`, proxy/pkg.go.dev propagation, `go get` verification (docs/RELEASE.md §15) remain. Deliberate: push requires explicit owner action.~~ done (pushed — v1.0.0 out 2026-09-10/11; pkg.go.dev verified later (10-28 pass a12))
2. **Full-code-review re-run** — recorded as DUE in TODO_LIST (this sweep landed substantial code; the item's own rule says re-run before v1.0), but NOT executed. The tag exists locally without it; the honest recommendation recorded there is review-then-push.
3. ~~**Upstream issue filing** — draft verified and saved, but the issue is not filed (owner repo, owner action; gh auth untested for it).~~ done (filed 2026-09-15 as go-error-family#5)
4. **Test-helper consolidation** — `reserveFreePort` deleted, `waitForTLS` relocated + upgraded, `waitForListenerAddr` added; but `waitForServerStart` (timeout-heuristic) is now arguably redundant with `waitForListenerAddr` and its existing callers were NOT migrated. Consolidation is ~80% done.
5. **`docs/architecture-reference.md` freshness** — the three rows I knew changed were refreshed (code.go, compress_pool.go, server.go), but the table was verified 2026-08-30 and this session's other new files (fuzz/bench files, nonce additions, compose API surface details) were not exhaustively re-inventoried.
6. ~~**AGENTS.md slimming** — under budget, but compression cost some nuance (a few sections are now summaries pointing at the reference doc), and the docs-health skill was not re-run to validate the new structure against its rubric.~~ done (v1.0.0 sweep: AGENTS.md 58.5 KB to 29.3 KiB via the docs/architecture-reference.md split; 30.1 KiB on 2026-09-11 (a hair over the 30 KB flag line))
7. ~~**`docs/status/` annotation obligation** — I read `docs/status/2026-08-06_23-33_etag-weak-comparison-fix-and-gap-analysis.md` during upstream research (and grepped others) but did not annotate the stale claims I noticed in the one I read. The doc-freshness cadence calls reading-without-annotating a missed obligation — this was missed this session.~~ done (docs-health pass 2026-09-11: the etag gap-analysis report fully annotated (last open item f38 marked with the sweep evidence))

## c) NOT STARTED

1. ~~**full-code-review execution** (see b2) — the single biggest open quality gate before push.~~ done (cut and pushed 2026-09-10/11 — the v1.0.0 tag shipped; v1.0.1 followed)
2. **Filing the go-error-family issue** (draft ready).
3. **go-compression extraction execution** (deferred by decision, preconditions recorded).
4. **Local govulncheck run** — CI does it and the flake has the app, but I never ran it in-session against the release commit.
5. **httpspec / server_timing benchmark baselines** — not re-measured this session (benchmarks.md explicitly scopes them out, but they have not been refreshed under the 3s×5 protocol at all).
6. **erraudit `--type-aware` advisory review** — only the two real gates were re-run; the ~30-advisory full run was not re-checked post-changes.
7. **Release workflow execution test** — release.yml changes are YAML-validated only; first real execution happens at push time.
8. **Nightly fuzz crash-issue path** — the `gh issue create` step has never executed; the `bug` label's existence in the repo is unverified (fallback warning exists).
9. **README v1.0 wording audit** — I fixed the redis doc's "removed at v1.0" claim but did not grep README / `docs/migrating-to-keyed-rate-limiter.md` / `docs/v1-stability.md` for the same stale phrasing.
10. **docs/v1-stability.md reconciliation** — referenced in FEATURES.md; not checked against what v1.0.0 actually froze (ListenerAddr etc. are IN, deprecated APIs are still IN pending post-v1.0 removal).

## d) TOTALLY FUCKED UP (own failures this session, all recovered)

1. **Flaky sync.Pool test** (`TestWriterPool_ResettableFactory_ReusesPooledWriter`): I asserted Put→Get instance reuse — not GC-stable under parallel load. Failed under `-race -count` in the full suite, passed standalone (worst kind). Cost three debug rounds before the isolated probe proved my test wrong, not the code. Rewritten as deterministic pool-path/factory-count tests. Lesson now in AGENTS.md Testing Conventions.
2. **FuzzMaxBodySize shipped two wrong invariants** before the right one: (1) "negative limits error unconditionally" — falsified in seconds by the fuzzer; (2) "error ⟺ len > maxBytes" — still wrong for negatives (empty body). Only then did I probe stdlib properly: error ⟺ `len(body) > max(limit, 0)`. Good outcome (the fuzz target pins real semantics and found a genuinely undocumented boundary), bad process — a 30-second probe up front would have saved two failures.
3. **`Recovery(nil)` in compose-bundles.md** — the doc example I wrote nil-panics on the first recovered panic (`Recovery` requires a logger). Caught on later self-review; docs/integrations snippets are not compile- or run-tested, so nothing else would have caught it.
4. **Duplicate RELEASE.md** — created a root RELEASE.md without checking that `docs/RELEASE.md` already existed (the CHANGELOG literally referenced it). Discovered late; trashed the duplicate, merged automation into the canonical doc, fixed references in three files. Textbook read-before-write failure on file CREATION.
5. **Long lint tail on my own new code** — across the session my new/edited files produced: forcetypeassert, nilnil, wrapcheck ×2, noctx ×2, varnamelen ×3, recvcheck, godot, gci, forbidigo, err113, makezero ×2, exhaustruct_v5 (missing struct field), modernize (WaitGroup.Go, errors.AsType), nolintlint (directive displaced by nix fmt re-wrapping). Every one fixed and final state is 0 issues, but the pattern is clear: I batched lint runs instead of running golangci-lint immediately after each new file.
6. **Stray `}` syntax error in server.go** — a careless edit left a leftover closing brace; caught by build, but it should not have survived past the edit.
7. **`listenAddr.String()` on a string** — wrote the TLS test against the wrong type, then the first sed fix missed one occurrence; three attempts to converge.
8. **CSRF parse test used an impossible input** — `TestCSRFConfig_WithParsedTrustedProxies_InvalidCIDRYieldsEmpty` used `"not-a-cidr"` (no slash → correctly treated as a bare IP, so "1 parsed CIDR" was RIGHT and my test expectation was wrong). Fixed with `"10.0.0.0/99"`.
9. **CHANGELOG restructure misfire** — first heading-merge pass produced a WORSE layout (duplicate Changed/Fixed headings) on the file that becomes immutable at the tag; fixed with a robust block-collection pass and boundary spot-checks (Added 17 / Changed 51 / Fixed 13). Should have been a single scripted transformation reviewed by diff.
10. **Background bench run died silently** — the chained 3-module 3s×5 run crashed mid-way ("exit status 2", truncated line, no diagnostic in the log — environmental, BenchmarkNonce runs fine standalone). Had to inventory missing rounds programmatically and re-run them in a second pass. Chunked execution from the start would have avoided this.

## e) WHAT WE SHOULD IMPROVE

1. **Probe external behavior before writing invariant tests.** Both multi-iteration failures (sync.Pool, MaxBytesReader) would have been one-round with an upfront `go run` probe. Make it a personal pre-test ritual for any stdlib-behavior claim.
2. **Lint as-you-go.** Run `golangci-lint fmt && golangci-lint run` immediately after every new file; the batch-at-the-end pattern produced a ~15-finding tail this session.
3. **Compile the docs.** Markdown code snippets in docs/integrations are never compiled; `Recovery(nil)` proves the risk. A build-tested snippet harness (parse fences → generated test file, or duplicated minimal examples in `example_test.go`) would make doc claims falsifiable.
4. **Read-before-write applies to file creation**: `ls docs/` / grep for the topic before creating any new doc (RELEASE.md dupe).
5. **Chunk long background runs** and make each chunk independently resumable; a single crashed chained command cost the whole root-suite log.
6. **Use benchstat** for baseline comparisons instead of "best of five" eyeballing; consider adding it to the flake so the documented comparison workflow is executable.
7. **Verify CI-referenced labels/resources exist** (the `bug` label for the nightly-fuzz issue step) at authoring time, not at first failure.
8. **Annotation discipline**: reading any docs/status report creates an obligation to annotate stale claims found; I skipped it once this session.
9. **The v1.0 sequencing critique**: the tag exists locally before the scheduled full-code-review ran. Local tags are reversible (`git tag -d`), but the cleaner sequence would have been review → fix → tag. Keep "review before immutable boundary" literally.
10. **The `[Unreleased]` → `[X.Y.Z]` cut should be a single reviewed transformation** (scripted, diffed) rather than incremental text surgery — the first merge pass degraded the file.

## f) NEXT 50 (ordered, concrete)

**Immediate — before pushing v1.0.0**

1. Run the scheduled full-code-review over everything in the `[1.0.0]` section (the recorded gate).
2. ~~Fix whatever that review finds; if anything lands, `git tag -d v1.0.0`, re-cut, re-verify.~~ obsolete — v1.0.0 reached origin before the fixes landed, and tags are immutable; the delta ships in v1.0.1 (CHANGELOG `[1.0.1]` records the contract as already-described-in-`[1.0.0]`)
3. Run govulncheck locally on the release commit (both modules).
4. Run the erraudit `--type-aware` advisory pass once over the post-sweep code (expect ~30 known test advisories; confirm no NEW ones).
5. ~~Grep README + `docs/migrating-to-keyed-rate-limiter.md` + `docs/v1-stability.md` for "removed at v1.0"-class stale claims (only the redis doc was fixed).~~ done (docs-health pass 2026-09-11: README + migrating-doc + v1-stability grepped for stale v1.0 claims; v1-stability removal targets reworded post-v1.0)
6. ~~Reconcile `docs/v1-stability.md` with what v1.0.0 actually froze (ListenerAddr/WrapConflict in; deprecated APIs still present pending removal).~~ done (docs-health pass 2026-09-11: ListenerAddr, Compose, MiddlewareFunc.Then, MiddlewareStack.Middleware, WrapConflict/Orchestration + attestation code added to v1-stability.md)
7. ~~Verify the `bug` label exists for the nightly-fuzz issue step (or change the step's label).~~ done (docs-health pass 2026-09-11: gh label list confirms the bug label exists)
8. Push: `git push origin master && git push origin v1.0.0` (owner action or explicit instruction).
9. Watch the tag-triggered release.yml run end-to-end; fix anything the real environment surfaces.
10. Post-push: `go get github.com/larsartmann/httputil@v1.0.0` + `go mod verify` from a scratch module; check pkg.go.dev.

**Follow-ups from this session**
11. File the go-error-family conditional-request issue from the verified draft.
12. Finish helper consolidation: migrate `waitForServerStart` callers to `waitForListenerAddr` (or delete the heuristic helper).
13. Re-inventory `docs/architecture-reference.md` tables file-by-file against HEAD (this session added 7+ files not in the tables).
14. Add a docs-snippet compilation harness so integration-doc examples (compose-bundles etc.) are build-tested.
15. Re-measure httpspec + server_timing benchmark baselines under the 3s×5 protocol.
16. ~~Annotate `docs/status/2026-08-06_23-33_etag-weak-comparison-fix-and-gap-analysis.md` (read, not annotated — missed obligation).~~ done (docs-health pass 2026-09-11: item f38 of that report marked done with the sweep evidence; file now fully resolved)
17. ~~Run the docs-health skill over the new AGENTS.md / architecture-reference split to validate against the quality rubric.~~ done (docs-health pass 2026-09-11 (this pass executed the skill over the corpus; scores in the session report))
18. Add tests for `scripts/coverage-threshold` (currently the only Go in the repo with 0% coverage, excluded from the gate for that reason).
19. Investigate the gopls `stdversion` warnings (`json.MarshalWrite requires go1.27`) that shadowed the whole session — decide whether go.mod moves to 1.27 when json/v2 stabilizes.
20. Add `benchstat` to the flake so the documented comparison workflow is executable.
21. Consider uploading bench artifacts in CI (benchmarks.md claims "full raw data in CI artifacts"; the bench job currently uploads nothing).
22. `KeyedRateLimiterMiddleware` measured 220 ns/op vs the doc's historical ~191 — sanity-check whether that is harness noise or a real regression from the sweep (property-test additions did not touch the hot path, so probably protocol difference; confirm with benchstat).
23. ~~Sweep TODO_LIST/ROADMAP for "v1.0" milestone references that now need marking done/re-baselined.~~ done (docs-health pass 2026-09-11: TODO_LIST rebuilt (completed items deleted to CHANGELOG))
24. ~~Update ROADMAP: the v1.0 vision entry and the `Wait(ctx)` post-v1.0 path need their status text refreshed now that v1.0.0 exists.~~ done (docs-health pass 2026-09-11: ROADMAP Current Position + v1.0 section rewritten for the shipped v1.0.0)
25. Re-run `nix flake check --all-systems` on a darwin host or accept the documented omission explicitly.

**Post-v1.0 stabilization release (v1.1.0 candidates)**
26. Execute plan T18: remove deprecated `TokenBucketLimiter`/`RateLimit()` per the migration guide (one stabilization cycle after v1.0.0).
27. Delete the deprecated `httputil.ETag()` adapter if its removal window also matured (check docs/v1-stability.md first).
28. Remove `BenchmarkTokenBucketLimiter*` rows + redis integration-doc deprecation scaffolding when 26 lands.
29. Refresh the go-error-family `require` to the latest patch before the first post-v1.0 release.

**Bigger items (planned/deferred)**
30. go-compression extraction: precondition pass — refresh the plan's line-by-line inventory against post-sweep code (writerPool changed the picture).
31. go-compression: owner setup (repo, remote, LICENSE/CI per the go-etag debt list), then execute the plan's phased commits.
32. go-compression: go-datastar consumer wiring (README row + datastartest optional dep).
33. Rate-limiter `Wait(ctx, key)` (ROADMAP post-v1.0 additive path) — design note already exists.
34. CSRFConfig: revisit whether `withParsedTrustedProxies` should become exported post-v1.0 if external consumers need to construct the CIDR form themselves.

**Hygiene & tooling**
35. Re-run the nightly-fuzz workflow manually (`workflow_dispatch`) to validate the crash-issue plumbing before the first scheduled run.
36. Wire a coverage-badge refresh into the release runbook (the badge step exists in ci.yml only; release.yml does not run it).
37. Make `prerelease-check.sh` accept `--skip-flake` for environments without nix (CI parity).
38. Consider a `justfile`-free task table in README pointing at the flake apps (bench/test/lint/vet/coverage) so consumers discover them.
39. ~~Update the coverage badge value if it still states the 2026-08-30 number (97.0% → 97.2%).~~ done (docs-health pass 2026-09-11: README badge + gates table updated to fresh race-measured 97.4%/98.6%)
40. ~~Normalize `docs/status/` — run the docs-health pass: annotate struck items found during this sweep, archive fully-resolved reports via `git mv`.~~ done (docs-health pass 2026-09-11 (this pass: Sep/Aug reports annotated, resolved files archiving))
41. ~~Fix the pre-existing CHANGELOG markdown quirk in the frozen `[1.0.0]` health bullet (embedded literal newline inside a code span) ONLY IF a correction policy for frozen sections is agreed — otherwise leave per the freeze rule and note it in `[Unreleased]`.~~ done (docs-health pass 2026-09-11: freeze policy honored; duplication + broken code span recorded in [Unreleased])
42. Decide a tag-annotation convention for multi-module releases (single-tag-with-replace is the current, documented pattern — re-verify it against the multi-module reference after the first real consumer appears).
43. Add `GOWORK=off` server_timing build to the flake checks (module-boundary script exists in CI; the flake check does not run it).
44. Review `.golangci.yml` exclusions for test-file `noctx` now that `httptest.NewRequest`-with-context patterns exist (`newTestRequest`) — possibly narrowable.

**Ideas surfaced, not yet decided**
45. Property-test the middleware stack ordering rules (Recovery-outermost) the way the limiter heap invariants are now tested.
46. Add an execution-probe test for `ServeTLS` ALPN mutation (the h2-append behavior is documented in AGENTS; a test would pin it).
47. ~~Consider exporting `CSRFTestToken`-style deterministic helpers for Nonce if composition-test authors keep needing `WithNonce` scaffolding (currently declined — revisit only with a concrete second consumer).~~ **Won't implement — declined 2026-09-10 in the v1.0 sweep; WithNonce is the injection escape hatch, revisit only with a concrete second consumer.**
48. Document the three-layer Go version policy (go.mod minimum / CI pin / local patch) in docs/RELEASE.md so the pin alignment rule survives sessions.
49. Evaluate `go test -fuzz` corpus seeds committed for the three new targets (corpus only lands on failure; consider committing representative seeds for the CORS echo oracle).
50. After the first post-sweep full-code-review: schedule the NEXT one (the cadence this item establishes is the real deliverable).

## g) QUESTIONS FOR THE OWNER (cannot decide myself)

1. **Push sequencing:** should I push `v1.0.0` as-is, or do you want the full-code-review to run first? If the review finds issues, do I delete the local tag and re-cut after fixes, or ship and patch in v1.0.1? — resolved by events: the tag pushed before the review fixes landed; the delta ships in v1.0.1.
2. **Deprecated-API removal cadence:** does `TokenBucketLimiter`/`RateLimit()` removal (plan T18) target v1.1.0 as the first post-v1.0 stabilization release, or do you want a longer deprecation window now that v1.0 declared stability?
3. **go-compression trigger:** do you want me to start the plan-refresh pass for the extraction next session regardless of go-datastar's timeline, or hold until go-datastar actually pulls?

---

_Verification summary at time of writing: tree clean, `v1.0.0` tag on HEAD (local), both modules race-green, lint 0 issues, erraudit gates green, flake check green, coverage 97.2% (library packages), consumer-mode build green._
