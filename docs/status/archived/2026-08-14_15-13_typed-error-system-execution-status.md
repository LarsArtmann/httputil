# Status Report — 2026-08-14 15:13 — Typed Hierarchical Error System: Executed

**Scope:** This session only — the review, amendment, and full execution of `docs/planning/2026-08-14_13-53_pareto-typed-hierarchical-error-system.md`. Point-in-time snapshot; route new findings to `TODO_LIST.md` via docs-health HARVEST.

**Session verdict:** Plan amended (8 amendments), then executed end-to-end. Final gates all green: `golangci-lint run` 0 issues, `go vet` clean, `go test -race -count=10 ./...` green (both modules), erraudit gates (`legacy_as`, `stdlib_constructor`) exit 0. Nothing is broken. But the execution was process-messy in places, and one design gap I criticized in the plan review survived into the implementation.

---

## a) FULLY DONE

| Area                                | Detail                                                                                                                                                                                                                                                                                                                          |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Plan amendment                      | 8 amendments written into the plan .md (scope fix, validateConfig logging, M15 rework, domain taxonomy, erraudit flags, predicate cut, test-style fix, CSRF re-estimate). HTML twin bannered as superseded snapshot.                                                                                                            |
| Typed model (`code.go`, new)        | `Code`, `Domain`, `Code.Domain()`, 6 constructor methods + 4 wrap methods returning `*errorfamily.Error`, `DomainOf`/`InDomain` via `errors.AsType[errorfamily.Coded]`.                                                                                                                                                         |
| All 13 config validators converted  | CORS, Server, Compression, KeyedRateLimiter, RateLimit (deprecated), MaxBodySize, RequestID, SecurityHeaders, Decompression, Metrics, Nonce, CSRF (context + typed codes), MiddlewareStack. All Rejection family, offending field values in context, `errors.Is` sentinel matching preserved (errorfamily matches code+family). |
| Runtime errors converted            | `server.shutdown_failed` (Infrastructure, cause via `WithCause`), `decompression.size_exceeded` (Rejection — bomb protection), `decompression.read_failed` (Corruption, cause), `decompression.close_failed` (Transient, cause), `compression.pool_type_unexpected` (Infrastructure), 4 q-value codes (Rejection).              |
| Internal call sites typed           | `recorder.go`, `wrapper.go`, `compress_writer.go`, `compress_writer_compress.go` use `codeWriteFailed`-style typed constants; exported `ErrCode*` remain untyped strings (zero breakage, no dual aliases).                                                                                                                      |
| Templates                           | 45+ message templates in `errors.go` as an `errorTemplates` data map (what/why/fix/wayOut, `{key}` placeholders). Completeness + domain-prefix + CSRF/ETag tests in `errors_templates_test.go`.                                                                                                                                 |
| `validateConfig` structured logging | Logs `code`, `family`, `domain` fields when classified; plain fallback otherwise. 3 tests incl. log-capture (sequential, `nolint:paralleltest` with reason — swaps the global default logger).                                                                                                                                  |
| `writeCommittedBody`                | Central helper in `recorder.go`; replaced 4 scattered honest-silence sites (Recovery, CSRF, keyed + deprecated rate limit).                                                                                                                                                                                                     |
| Docs                                | README "Handling errors from httputil" section; AGENTS.md: new Error Model section, expanded classification table, commands incl. erraudit invocation, file-table rows for `code.go`/`errors.go`/`recorder.go`; CHANGELOG `[Unreleased]` Added/Changed.                                                                         |
| erraudit alignment                  | Verified `--enforce-go-error-family` exists on the installed binary. Gates pass: `--type legacy_as` exit 0, `--type stdlib_constructor --enforce-go-error-family` exit 0. Remaining ~30 findings are `errors.Is` sentinel advisories in tests — all correct as-is (decision tree), documented not to be migrated.               |
| Benchmarks                          | `code_bench_test.go`: construction ~166ns, clone+context ~301ns, wrap+cause ~292ns, `DomainOf` ~53ns, `InDomain` ~49ns.                                                                                                                                                                                                         |
| Incidental fixes                    | Latent `csrf_test.go` SA5011 (else-branch restructure); `.golangci.yml` canonicalheader exclusion for httpspec rate-limit headers (surfaced by cache re-analysis; file untouched this session).                                                                                                                                 |

## b) PARTIALLY DONE

| Item                                                     | What's missing                                                                                                                                                                                                                                                                                   |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| F16.2 "integration test through middleware constructors" | Tests call `Validate()` directly and `validateConfig` directly. No test drives a real constructor (`CORS(badCfg)` → log line with classified fields end-to-end).                                                                                                                                 |
| Open question #2 (erraudit config file)                  | Resolved as "config file if the binary supports one". I verified CLI flags but never checked whether erraudit supports a config file at all (root `--help` not inspected for it).                                                                                                                |
| Legacy CSRF domain membership                            | `csrf_samesite_insecure` etc. use the historical underscore spelling → `DomainOf` returns the whole code as its own domain; `InDomain(err, Domain("csrf"))` is **false** for CSRF config errors. Handled by exempting them in the domain test, not by fixing. Documented wart, consumer-visible. |
| Plan-doc internal consistency                            | Step 3 header still says "60 fine tasks" (not recounted after amendments); minor residual drift between mermaid graph and amended task list.                                                                                                                                                     |
| AGENTS.md file table                                     | New test files (`code_test.go`, `validation_classified_test.go`, `errors_templates_test.go`, `validate_config_log_test.go`, `code_bench_test.go`) not added to the table, although `testutil_test.go` is listed there.                                                                           |

## c) NOT STARTED

| Item                                                          | Note                                                                                                                                                                                                                                         |
| ------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Exported domain constants (`DomainCORS`, `DomainServer`, ...) | Consumers must hand-write `Domain("cors")` — exactly the stringly mixing I criticized in the plan review. The type exists; the vocabulary doesn't.                                                                                           |
| Placeholder-syntax unification                                | Old writer templates use `{{.status}}` (Go-template style); new templates use `{key}` (errorfamily's documented syntax). Did not verify whether `{{.status}}` renders or leaks literally, and did not unify.                                 |
| Context on runtime constructor errors                         | `NewTokenBucketLimiter` returns `errInvalidRate`/`errInvalidBurst` bare — classified but no context (the offending rate/burst value). Same for `errDecompressionSizeExceeded` (no `max_decompression_size` context at the trip site).        |
| TODO_LIST.md harvest                                          | Plan footer says route follow-ups via docs-health HARVEST — not done (this report is the interim ledger).                                                                                                                                    |
| Error-message-format migration note                           | Sentinel message strings changed format (`[rejection:cors.max_age_negative] msg` vs old `"msg: got -5"`). CHANGELOG covers the classification change but not the literal message-format change downstream string-matcher consumers will see. |

## d) TOTALLY FUCKED UP

Nothing in the end state — all gates green, nothing reverted. But three **process** failures worth recording:

1. **`errors.go` funlen whack-a-mole (3 rounds).** I converted templates into functions, hit funlen (60-line cap), split, hit it again, split again — churning the file structure three times before wising up and switching to a declarative `errorTemplates` map, which was obviously right from the start (data, not code). ~20 wasted minutes and a messy intermediate history the auto-commit daemon may have snapshotted.
2. **`slog` API fumbles (2 attempts).** First `slog.Attrs` (doesn't exist), then hand-rolled `slog.NewRecord` + `Handler().Handle(...)` — when `slog.Default().LogAttrs(ctx, level, msg, attrs...)` is the intended one-liner for a `[]slog.Attr`. Current code works and is tested, but it's needlessly low-level. Should have read the `log/slog` API surface before typing.
3. **Benchmark written from stale memory.** `*testing.B` param named `t` + `t.N()` + `for range t.N` — three compile failures before landing on `b.Loop()`. Sloppy; Go 1.24+ `b.Loop` should be muscle memory.

**Honesty check (skill question 5):** I did not lie in the final summary, but it was **incomplete**: I reported "plan amended and fully executed" without flagging the F16.2 constructor-integration gap or the legacy-CSRF domain wart. This report corrects that.

## e) WHAT WE SHOULD IMPROVE (ranked)

1. **Define exported domain constants** (`DomainCORS`, `DomainCSRF`, ...) — closes the stringly-typed gap at the heart of the model. ~15 min + tests.
2. **Migrate or shim legacy CSRF codes** (`csrf_samesite_insecure` → `csrf.samesite_insecure`) so `InDomain(err, DomainCSRF)` works uniformly. Either break it now (v0.x, one-line CHANGELOG note) or special-case underscore codes in `DomainOf`. Decision needed — see questions.
3. **Replace hand-rolled slog Record with `LogAttrs`** — 5-line simplification, same behavior, re-run log tests.
4. **Verify + unify template placeholder syntax** — read errorfamily's rendering; if `{key}` is the only interpolation, fix the two `{{.status}}` templates (write_failed, etag_write_failed).
5. **Add constructor-integration test** — `CORS(invalid)` through the real constructor, assert log fields end-to-end.
6. **Attach context at remaining bare return sites** (`errInvalidRate`, `errInvalidBurst`, size-exceeded trip).
7. **Add `docs/status` + new test files to AGENTS.md file table**; recount plan Step 3.
8. **Rename `allHTTputilErrorCodes`** → `allHTTPUtilErrorCodes` (typo in the authoritative test list, referenced from the errors.go comment).
9. **Narrow the httpspec canonicalheader exclusion** from path-based (disables the linter for whole files) to text-based, matching the `X-CSRF-Token` precedent in the same file.
10. **Delete dead code in `captureValidateConfigLog`** (unused `Level`/`Msg` struct fields, pointless string→Code `codesOf` indirection) and the magic `cfg.SameSite = 4` in the CSRF test (use `http.SameSiteNoneMode`).
11. **CHANGELOG note on message-format change** for string-matching consumers.

## f) NEXT UP TO 50 (session-derived backlog)

**Error model polish (1–10):** domain constants; CSRF legacy-code migration; `LogAttrs` simplification; placeholder unification; constructor-integration test; context at bare sites; `allHTTputilErrorCodes` rename; `Domain()` fast path benchmark vs allocations; doc example (`ExampleDomainOf`) with `// Output:`; consider `Domain` constants for `http` (shared with go-etag codes).

**Test hardening (11–18):** middleware-constructor log assertions for every constructor (table of 13); fuzz `Code.Domain()` on arbitrary strings; test `InDomain` through `fmt.Errorf("%w")` double-wrap; assert `WithCause` chains preserve `DomainOf`; verify parallel `RegisterErrorClassifications` under `-race -count=50`; benchmark `writeCommittedBody` vs direct write (should be identical).

**Docs (19–25):** AGENTS.md file-table rows for new test files; Testing Conventions section update (classification test patterns, non-parallel logger-swap pattern); plan Step 3 recount; docs-health HARVEST into TODO_LIST.md; README API table row for `DomainOf`/`InDomain`/`Code`; CHANGELOG message-format note; FEATURES.md error-model feature entry.

**Debt and alignment (26–33):** narrow canonicalheader exclusion; erraudit config-file support check (root `--help`); wire erraudit gates into flake.nix/CI if a hook point exists; `nix flake check` (never run this session); consider deprecating `errInvalidRate`/`errInvalidBurst` messages to include values; ratelimit.go deprecated-path audit for remaining fmt verbs; unify "wayOut" tone across templates (some end with periods, some don't — check).

**Future-facing (34–40):** v1 decision — `Validate() *errorfamily.Error` vs `error` (resolved: keep `error`; revisit at v1); consider `Domain.Registry()` validation that all codes declare known domains at init; explore `errorfamily.Handle` integration for middleware error responses; server_timing sub-module error audit (currently error-free by design?); httpspec specs asserting classified errors on 4xx; etag adapter deprecation removal timeline.

**Housekeeping (41–45):** commit strategy for the 29 modified + 6 new files (see questions); verify the auto-commit daemon's inferred messages don't mangle the docs; tag-check against CHANGELOG freeze policy before next release; re-run `art-dupl` to confirm the new test helper didn't clone; check `go test -bench=. ./...` full suite for regressions in other benchmarks.

**Verification backlog (46–50):** pkg.go.dev rendering spot-check after next tag; run `erraudit lint` on `server_timing/` too (separate module, wasn't included in this session's runs — I only ran the root module); confirm `--enforce-go-error-family` behavior matches expectation on a deliberately-introduced violation; grep for any remaining `errors.New`/`fmt.Errorf` in non-test root files as a final sweep; lint the two "stale" LSP-diagnostic files via full-package run to prove the panel is cache-stale (already done once — 0 issues).

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. ~~**Commit granularity:** 29 modified + 6 new files are uncommitted (I don't commit without instruction). One feature commit ("typed hierarchical error system"), or split — model / validators+runtime / templates+logging / docs? The auto-commit daemon may also snapshot the messy intermediate states; do you want a deliberate commit to supersede those?~~ done (superseded by history: shipped as v0.12.0)
2. ~~**Legacy CSRF code migration:** keep `csrf_samesite_insecure` (underscore, breaks `InDomain(DomainCSRF)`) for backward compatibility, or migrate to `csrf.samesite_insecure` now while pre-v1? Both are defensible; the tradeoff (compat vs. hierarchy consistency) is a product call.~~ done (decided: keep underscore spelling + Infrastructure family for backward compatibility (AGENTS.md Error Model))
3. ~~**Exported domain constants now or at v1?** Adding `DomainCORS`, `DomainServer`, ... is new public API surface in v0.x. Add now (consumers get compile-safe domains immediately) or hold until the v1 API freeze?~~ done (decided: not exported pre-v1; consumers use Domain("cors") literals (README pattern))

---

**Gates at time of writing:** `golangci-lint run` **0 issues** · `go vet ./...` clean · `go test -race -count=10 ./...` green (root + httpspec) · `server_timing` green + 0 issues · erraudit `legacy_as` **0**, `stdlib_constructor` **0** · uncommitted working tree (see question 1).
