# Session Status — 2026-11-14 00:37: All Five Open Issues (#45–#49) Fixed in One Pass

Scope: `gh issue list` showed 5 open issues (#49, #48, #47, #46, #45). The instruction was
"fix it all." Everything below happened in this single session (2026-11-13 evening →
2026-11-14 00:37). All work was swept into daemon heuristic commits and pushed; master ==
origin/master (0 unpushed).

## What was done

| Issue                                                 | Fix                                                                                                                                                                                                                                                                                                                                                                                                               | Key sites                                                                                                                                                                                 |
| ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| #49 Compression strips net/http Content-Type sniffing | `maybeSniffContentType()` before every header commit (compressed start, plain flush, small-response Close), mirroring net/http's exact guards (`bodyAllowedForStatus` 1xx/204/304, never override explicit CT, handler-set `Content-Encoding` suppresses sniff per net/http issue 31753; the compressed path sniffs BEFORE installing its own CE so the type describes the decoded representation). 6 tests.      | `compress_writer.go` (`maybeSniffContentType`, `bodyAllowedForStatus`, `beginPlainResponse`, `Close`), `compress_writer_compress.go` (`startCompression`), `compression_behavior_test.go` |
| #48 CSRF vs HTTP QUERY (RFC 10008)                    | TWO halves, because the issue's suggested one-liner was insufficient (verified in nosurf v1.2.0 source: its own `safeMethods` list challenges QUERY independently): (1) `isUnsafeCSRFMethod` adds QUERY; (2) method-scoped nosurf `ExemptFunc` in `CSRFMiddleware`. QUERY passes token-less, still seeds the CSRF cookie (nosurf regenerates before its exemption check). 3 tests. Behavior change release-noted. | `csrf.go` (`methodQuery` const, `isUnsafeCSRFMethod`, middleware wiring, doc comment), `csrf_test.go`                                                                                     |
| #47 ResponseRecorder drops http.Pusher                | `pushDelegate` in `wrapper.go` + `Push` methods on BOTH `ResponseRecorder` and `responseWrapper` (so push survives `Compression`, not just the recorder). New codes `http.push_unsupported` (Infrastructure) / `http.push_failed` (Transient, cause preserved) with the full error ritual (constants, typed mirrors, `errorTemplates`, `allHTTputilErrorCodes`, FEATURES/README/arch-ref tables). 9 tests.        | `wrapper.go`, `recorder.go`, `errors.go`, `errors_templates_test.go`, `testutil_test.go` (`pushRecorder`, `failingPusher`), `errors_test.go`, `chain_hijack_test.go`                      |
| #45 KeyedRateLimiter clock seam                       | `KeyedRateLimiterConfig.Now func() time.Time` (nil → `time.Now`), threaded through last-use stamps, TTL eviction, AND token refill (`AllowN(now, 1)` — refill determinism requires the clock, not just eviction; the issue text under-specified this). `fakeClock` + 3 deterministic tests (refill without sleeping, TTL eviction, nil-default).                                                                  | `ratelimit_keyed.go`, `ratelimit_keyed_test.go`, 3 `newPerKeyLimiter` call sites updated (tests+bench)                                                                                    |
| #46 httpspec harvest-then-revalidate                  | `ExpectNotModifiedDynamic(method, path)`: request → harvest `ETag` → re-request with `If-None-Match` → assert 304 + empty body + tag echoed; fails loudly when no ETag served. 5 tests (incl. changed-representation, 304-with-body, unechoed-tag) + runnable `ExampleExpectNotModifiedDynamic`.                                                                                                                  | `httpspec/httpspec.go`, `httpspec/httpspec_test.go`, `httpspec/example_test.go`                                                                                                           |

Docs swept: CHANGELOG `[Unreleased]` (3 Added, 1 Fixed, 1 Changed), FEATURES.md (code list,
wrapper/compression/ratelimit/CSRF/httpspec sections), README.md (recorder prose, error
table, recorder-methods table, `KeyedRateLimiterConfig.Now` row), docs/architecture-reference.md
(recorder/wrapper/errors/httpspec rows + Push error rows), AGENTS.md (4 new Non-Obvious
Behaviors bullets).

## Verification (all green)

- `go build`, `go vet`, `go test -race -shuffle=on -count=1 ./...` — root + httpspec + scripts.
- Nested modules: `go test -race -shuffle=on` in `server_timing/` and `endpoints/` — pass.
- golangci-lint **v2.14.0 (the CI pin)** via `go run …@v2.14.0 run` — **0 issues**.
- All 6 erraudit gates (legacy_as + stdlib_constructor × 3 modules) — exit 0.
- Fuzz smoke: `FuzzCompression` 20s (66 new interesting inputs, PASS — the CT-sniffing
  change sits on the fuzzed paths), `FuzzCSRFMiddleware_OriginHeaders` 15s PASS.
- apidiff vs frozen baselines — compatible (additive only; baselines correctly NOT
  regenerated — release-commit-only ritual).
- `scripts/check-qvalue-corpus-sync.sh`, `scripts/pre-sweep-check.sh` (build+vet+sentinel
  guards ×3 modules + corpus oracle) — OK.
- `golangci-lint fmt` + `nix fmt` — clean.
- Quick compression benchmark sanity (BenchmarkCompression 4.6µs/op) — no pathological cost
  from the sniff (one bounded 512-byte `DetectContentType` per CT-less response).

## a) FULLY DONE

1. All five issues (#45–#49): code + claim-named tests + error ritual + docs, verified green
   under the CI-pinned lint, race detector, fuzz smoke, apidiff, erraudit, pre-sweep gates.
2. Root-cause depth beyond the issue text: #48's nosurf-internal challenge (the issue's
   one-liner would NOT have fixed the repro), #45's refill-math `AllowN` requirement, #49's
   third commit path (small-response `Close`) and the 31753/`bodyAllowedForStatus` guards.
3. Docs sweep across all five living docs + AGENTS.md behaviors.

## b) PARTIALLY DONE

1. **docs/v1-stability.md NOT updated** (caught during this self-review): the error-model
   ritual in AGENTS.md explicitly lists it; the config-types table's "changes" column should
   note `KeyedRateLimiterConfig.Now` (the `CompressionConfig.AbsentEncoding added in v1.2.0`
   precedent), and the additive surface (`Push` methods, `ExpectNotModifiedDynamic`) belongs
   in the stability tables. The API is compatible; only the stability DOC lags.
2. **Session-tail discipline executed late**: push state (master == origin, 0 unpushed) and
   CI were only checked during THIS self-review, not before declaring the work done
   (2026-10-09 lesson ×2 says check first). Latest COMPLETED CI run green at 22:23 on a
   mid-session head; a run for the final head (docs + makezero-fix commits pushed ~00:3x)
   was not yet visible in `gh run list` — needs one re-check.
3. **Cross-repo verification items from the issues' own "To verify" sections** are repo-side
   done but consumer-side untouched (see c).

## c) NOT STARTED (deliberate: other repos / owner calls)

1. quotaboard (#49): drop/simplify the consumer-side `TestHTMLPagesCarryExplicitContentType`
   workaround now that the library sniffs.
2. CV (#45): rewrite `TestRateLimitRefillRatePinned` (6.2s real sleep, `-short` skip) on the
   new `Now` seam.
3. company-master-data (#46): adopt `ExpectNotModifiedDynamic`, delete the hand-rolled
   "conditional requests (ETag)" describe block.
4. cqrs-htmx (#48): ship native `QUERY` routes (ADR-0064) instead of the permanent
   POST+`X-HTTP-Method-Override` fallback.
5. GitHub issue bookkeeping: none of #45–#49 have resolution comments or "Fixes #NN"
   references (daemon heuristic commits carry none) — owner decides close-on-push vs. manual.
6. `docs/benchmarks.md` / the documented 3s×5 protocol was not re-run for the sniffing
   change (a 200ms sanity bench was run instead).
7. TODO_LIST.md rows for the follow-ups below.

## d) TOTALLY FUCKED UP

Nothing shipped broken. Honest near-misses:

1. **Wrote a nonsense `if r := recover() { panic(r) }` block** into a draft of the
   changed-representation httpspec test — caught and removed before ever running it. Zero
   impact; process slip.
2. **First `newPerKeyLimiter` edit mismatched parameter order** (put `cfg.Now` last while
   the new signature takes it first) — compiler caught it immediately; fixed in the next edit.
3. **The v1-stability.md ritual step was skipped** (see b1) — the only documented sweep item
   missed, discovered during self-review rather than during the sweep.
4. **makezero fight**: my first binary-body test used `make([]byte, N)` + index writes,
   which `makezero: always: true` rejects; fixed to the append-blessed shape. The LSP then
   showed a STALE makezero diagnostic on the fixed code for one round — trusted the real
   binary over the LSP, correctly.

## e) WHAT WE SHOULD IMPROVE (observed this session)

1. **Local vs CI golangci-lint drift is unrecorded**: local nix binary is v2.15.0 and its
   wrapcheck flags 47 PRE-EXISTING findings (untouched files: server.go, stack.go, csrf.go
   Validate()...); CI pin v2.14.0 reports 0. The authoritative gate is the pin — but the
   drift is written nowhere durable (AGENTS/TODO_LIST). Next pin bump will surface 47
   findings at once and someone will burn time re-triaging what I already established.
2. **`DetectCapabilities()` (capabilities.go) still probes only Hijacker/Flusher** — after
   #47, Pusher is delegated everywhere but not reported by the capability probe. Either add
   a `Pusher` field (additive, v1-safe) or document why push is excluded.
3. **`ConfigureNosurfHandler` asymmetry is underdocumented**: the low-level configurator
   deliberately does NOT exempt QUERY (only `CSRFMiddleware` does). One sentence on the
   function + a pinning test would prevent a "why does QUERY pass here but not there" report.
4. **The CSRFMiddleware ExemptFunc slot is now occupied** by the QUERY exemption — any
   future consumer-exemption config field must compose with it, not overwrite it. Worth an
   AGENTS line (partially covered by the "two halves must stay in sync" note).
5. **Daemon-committed work has no issue references**: five issue fixes exist only as
   heuristic commits + CHANGELOG entries. Closing the issues cleanly needs either manual
   closes with a CHANGELOG-pointer comment or a deliberate `Fixes #NN` commit at next release.
6. **Session-tail check ordering** (see b2): do the push/CI check BEFORE writing the final
   summary, not during the self-review after it.

## f) NEXT UP (prioritized)

1. Sweep `docs/v1-stability.md`: `Now` row in the config-types table + additive entries for
   `Push`/`ExpectNotModifiedDynamic` (closes the missed ritual step).
2. Re-check CI on the final head (run for the ~00:3x push was not yet visible).
3. Comment-and-close (or annotate) issues #45–#49 pointing at the CHANGELOG `[Unreleased]`
   entries — owner call.
4. `DetectCapabilities()`: add `Pusher bool` probe + test (or document exclusion).
5. `ConfigureNosurfHandler` doc sentence + asymmetry-pinning test for QUERY.
6. Record the golangci-lint v2.15.0-vs-v2.14.0 wrapcheck drift in AGENTS.md (lint section)
   - a TODO_LIST row for the next pin bump.
7. quotaboard: simplify `TestHTMLPagesCarryExplicitContentType` (library now sniffs).
8. CV: rewrite the 6.2s refill pin on `KeyedRateLimiterConfig.Now`; drop the `-short` skip.
9. company-master-data: adopt `ExpectNotModifiedDynamic`, delete the hand-rolled block.
10. cqrs-htmx: native QUERY route behind `CSRFMiddleware` (ADR-0064 follow-through).
11. Re-run the documented 3s×5 benchmark protocol; refresh `docs/benchmarks.md` (the sniff
    added a one-time 512-byte scan per CT-less compressed response — pin the number).
12. TODO_LIST rebuild row for items 1–10 (docs-health).
13. Add a `BenchmarkCompressionContentTypeSniff` micro-bench if the protocol run (11) shows
    measurable movement.
14. Consider an `httpspec` spec-name constant + default-spec discussion for
    `ExpectNotModifiedDynamic` (currently opt-in via `WithExtraSpecs` only, like the static
    variant — probably correct, but the question is unasked).
15. Mutation-probe the new guards once (delete the `bodyAllowedForStatus` call → expect the
    204 test to fail; delete the CE guard → expect the 31753 test to fail) — written from
    first principles as mutation-sensitive, never actually mutated this session.
16. `server_timing` middleware doc sentence mentions Flusher/Hijacker passthrough — check
    whether its wrapper should also grow `Push` delegation for symmetry (its writer wraps
    for header injection; a Push there would also be silently dropped today).
17. Fleet lint-pin bump decision (v2.14.0 → v2.15.x) with the 47 wrapcheck findings triaged
    in the same change (likely all config-ignore-sigs additions for
    `WithContext*`/`WithCause`, matching the existing `.Wrap(` entries).
18. Consider `methodQuery` upstream: Go has no `http.MethodQuery` — if a future Go adds it,
    swap the literal (leave a TODO nowhere; the AGENTS bullet already names the situation).
19. Next release (`v1.9.0`-shaped): regen apidiff baselines (three zips), freeze CHANGELOG
    section, retarget `[Unreleased]` link, coordinated sub-module check per RELEASE.md.
20. Check whether `endpoints` listing should surface QUERY routes distinctly (it lists
    whatever is registered; no action unless a consumer asks).
21. The `fakeClock` could graduate to a shared test util if a second consumer of the seam
    appears (CV will likely copy it — consider exporting nothing until then; YAGNI held).
22. Docs-health pass over the four living docs touched this session (VERIFY mode) before
    the next tag.
23. Add `Push` delegation note to the OTEL interop guide's writer-wrapper contract list
    (`docs/integrations/otel.md` names Flusher/Hijacker survival; Pusher now belongs).
24. Re-run `buildflow --build-mode dev` once on the final tree to confirm the expected
    findings-gate composition (policy-rejected residuals only, nothing NEW) — not run this
    session (the real gates were run individually instead).

## g) QUESTIONS FOR THE OWNER (cannot be answered from the repo)

1. **QUERY default-on confirmation**: #48 shipped the safe-list as default behavior with a
   release note (per the issue's ask). Consumers who deliberately wanted QUERY challenged
   must re-add a gate. Keep default-on, or do you want an opt-out knob
   (e.g. `CSRFConfig.ChallengeQuery`) before this ships in a tagged release?
2. **Issue bookkeeping**: close #45–#49 now with comments pointing at the CHANGELOG
   `[Unreleased]` entries (the daemon's heuristic commits carry no "Fixes #NN"), or leave
   them open until the next tag cuts a release that names them?
3. **Consumer follow-ups in their repos** (quotaboard guard simplification, CV clock-seam
   rewrite, company-master-data httpspec adoption, cqrs-htmx native QUERY): want me to
   execute those next, and in what order relative to cutting the next httputil release
   (the #45/#46/#49 consumer cleanups only make sense after a tag they can pin)?

## Session facts

- Working tree clean after daemon sweep; `master == origin/master` (0 unpushed).
- Latest completed CI: success (11m58s) on the 22:23 push; final-head run pending visible
  completion at report time.
- No deliberate commits were made this session (daemon heuristic commits only) — per the
  no-commit-without-instruction rule.
