# Status Report: Middleware Type Alias & ETag Deprecation

**Date:** 2026-08-14 12:38
**Session scope:** Changing `Middleware` from named type to type alias across httputil, server_timing, and go-etag; deprecating `httputil.ETag()` adapter.

---

## a) FULLY DONE

1. **`Middleware` type alias in httputil** (`recorder.go:15`): changed from `type Middleware func(http.Handler) http.Handler` to `type Middleware = func(http.Handler) http.Handler`. All internal code compiles and composes without conversions.

2. **`Middleware` type alias in server_timing** (`server_timing/middleware.go:6`): same change. `ServerTimingMiddleware()` and `ServerTimingMiddlewareWhen()` return `Middleware` (now an alias), which composes directly with `Chain`.

3. **`Middleware` type alias in go-etag** (`go-etag/middleware.go:6`): same change. `etag.New()` return value composes directly with httputil's `Chain` and `MiddlewareStack`.

4. **Removed `Middleware()` conversion in `etag.go`**: `return Middleware(etag.New(cfg))` → `return etag.New(cfg)`. The conversion is unnecessary now that both types are aliases of the same underlying type.

5. **Fixed stale `httputil.` → `servertiming.` doc comments** in `server_timing/server_timing.go`: 4 code-comment examples referenced `httputil.MeasureServerTiming`, `httputil.ServerTimingMiddleware()`, and `httputil.ServerTimingMiddlewareWhen()` — all updated to `servertiming.` prefix. These were pre-existing stale references from the server_timing module extraction, not introduced this session.

6. **Deprecated `httputil.ETag()`** (`etag.go`): added `// Deprecated:` directive pointing to `etag.New()`. Doc comment rewritten to explain why (type alias makes the adapter a pure passthrough).

7. **Migrated all internal call sites** from `ETag(etag.DefaultETagConfig())` to `etag.New(etag.DefaultETagConfig())`:
   - `etag_test.go` (4 call sites)
   - `stack_integration_test.go` (1 call site)
   - `example_test.go` (1 call site)

8. **Moved error classification guidance** from `ETag()` doc comment to `RegisterErrorClassifications()` doc comment (`errors.go:45-53`): now documents that it registers a strict superset of go-etag's error templates and that consumers should not also call `etag.RegisterErrorClassifications()`.

9. **Updated living docs**:
   - `README.md`: API table marks `ETag` as deprecated; usage examples use `etag.New()`; composition explanation updated.
   - `AGENTS.md`: `etag.go` row marks `ETag()` as deprecated.
   - `docs/DOMAIN_LANGUAGE.md`: `ETag(cfg)` → `etag.New(cfg)` in verb table; narrative updated.
   - `docs/v1-stability.md`: `ETag` row marked deprecated.
   - `ROADMAP.md`: `[Unreleased]` narrative updated to mention deprecation and type alias.
   - `CHANGELOG.md`: `[Unreleased]` section added with Changed (type alias) and Deprecated (`ETag()`) entries.

10. **All tests pass with race detection** across all three modules:
    - `httputil`: `go test -race ./...` — ok
    - `httputil/server_timing`: `go test -race ./...` — ok
    - `go-etag`: `go test -race ./...` — ok

11. **All lint passes with 0 issues** across all three modules:
    - `httputil`: `golangci-lint run` — 0 issues
    - `httputil/server_timing`: `golangci-lint run` — 0 issues
    - `go-etag`: `golangci-lint run` — 0 issues

12. **Formatting clean** across all modules (`golangci-lint fmt`).

---

## b) PARTIALLY DONE

1. **Historical status reports not annotated.** Several docs in `docs/status/` reference `httputil.ETag()` and the old named `Middleware` type. Per the AGENTS.md doc-freshness cadence, these should be annotated with inline `~~item~~ done at <hash>` markers when read. I did not annotate them. They are frozen history but contain stale claims that a reader might take as current truth.

2. **`docs/v1-stability.md` `Middleware` type references.** I updated the `ETag` row but did not audit the entire file for other references to the named `Middleware` type that might now be misleading (e.g., if any stability claims depend on it being a named type rather than an alias). The file should be reviewed.

3. **`FEATURES.md` not updated.** The FEATURES.md middleware table likely still lists `ETag()` without deprecation marker. I did not check or update it this session.

4. **`TODO_LIST.md` not updated.** If there are open items about the `Middleware` type or ETag adapter, they should be marked done or updated.

---

## c) NOT STARTED

1. **`server_timing/middleware.go` is a new file — no test coverage.** It's a 1-line type alias, so testing it directly is pointless, but the file has no test file and no `// Output:` example. The `testableexamples` linter doesn't flag it (it's not an `Example*` function), but the file is new and unstated in FEATURES.md.

2. **go-etag CHANGELOG not updated.** The `go-etag` module's `CHANGELOG.md` should have an `[Unreleased]` entry for the `Middleware` type alias change. I did not update it.

3. **go-etag AGENTS.md not updated.** If go-etag has an AGENTS.md documenting the `Middleware` type, it should reflect the alias change.

4. **No `Deprecated` lint verification.** I did not run a linter or compiler flag to verify that the `// Deprecated:` directive is correctly formatted and will produce deprecation warnings in IDEs. `staticcheck` (if available) would flag `ETag()` usage as deprecated.

5. **No consumer migration guide.** The CHANGELOG entry mentions the deprecation but there's no migration snippet or section showing the before/after for consumers.

6. **`docs/v1-stability.md` stability tier for `Middleware` type itself.** The `Middleware` type alias is a semantic change (named → alias). If `Middleware` was listed as "Frozen" as a named type, the stability tier should be updated to reflect that it's now an alias. I did not check this.

7. **`stack.go` `middlewareEntry.middleware` field type.** The field is `Middleware` (now alias). This is correct but the `Build()` method still allocates an intermediate `[]Middleware` slice to pass to `Chain`. This allocation was unnecessary before (could iterate directly) and is still unnecessary now. Not a regression, just pre-existing debt.

---

## d) TOTALLY FUCKED UP

1. **The `Middleware` conversion in `stack_integration_test.go`.** During the session, I went through three iterations of the test file:
   - First: added `Middleware(servertiming.ServerTimingMiddleware())` conversions (when server_timing had a named type).
   - Then: removed them (when `Chain` params changed to unnamed type).
   - Then: the auto-git-commit daemon reformatted the file, causing a stale-read edit failure.
   - Then: re-read and re-edited.
   This churn was caused by me changing the design mid-session (named type → unnamed params → alias) instead of thinking through the design first. The final state is correct, but the path was wasteful.

2. **The `nosurf.NewPure` fabrication.** In an earlier response, I wrote `nosurf.NewPure CSRFProtection(csrfFailureHandler)` as an example of middleware that would compose frictionlessly. This was completely fabricated — `nosurf.NewPure` returns `http.Handler`, not `func(http.Handler) http.Handler`. It's not middleware at all. I should have verified the API before using it in an example.

3. **The `Chain` parameter type churn.** I changed `Chain`'s parameter from `...Middleware` to `...func(http.Handler) http.Handler` and back to `...Middleware` within the same session. The intermediate change (unnamed type) was correct but unnecessary once we settled on the alias approach. The final state (`...Middleware` where `Middleware` is an alias) is the right design, but I wasted a round trip by not arriving there directly.

4. **Bogus `nosurf.NewPure` in composition example could have misled a user.** If the user had copied that code, it would not have compiled. I corrected it when called out, but the error should not have happened.

---

## e) WHAT WE SHOULD IMPROVE

1. **Think through the design before editing.** The session went: add named type → discover friction → change `Chain` params → discover alias solution → revert `Chain` params. Three design iterations on the same problem in one session. The alias solution was obvious from the start once the user asked about `typealias` — I should have recognized that immediately instead of going through the unnamed-parameter detour.

2. **Verify external API signatures before using them in examples.** The `nosurf.NewPure` fabrication was preventable by a 10-second search. I did verify it when asked, but I should have done so before writing the example.

3. **The `Middleware` type alias pattern should be documented as a design decision.** The AGENTS.md "Non-Obvious Behaviors" section should explain why `Middleware` is an alias (not a named type) so future contributors don't "fix" it back to a named type and reintroduce the conversion friction.

4. **`Build()` in `stack.go` allocates an intermediate slice unnecessarily.** `Build()` copies `[]middlewareEntry` → `[]Middleware` → passes to `Chain`. Since `Chain` now takes `...Middleware`, `Build` could iterate the entries directly and call `Chain` with a variadic spread, or `Chain` could accept an interface. Minor allocation, but it's pure debt.

5. **`ETag()` deprecation should have a removal timeline.** The `// Deprecated:` directive doesn't say when it will be removed. Go convention is to specify the target version (e.g., "Removed in v1.0"). Without a timeline, the deprecation is open-ended.

6. **go-etag and httputil CHANGELOGs should be coordinated.** The type alias change spans two modules. Both CHANGELOGs should have entries, and they should reference each other. Only httputil's was updated.

7. **The `server_timing` doc comments had stale `httputil.` prefixes from a prior extraction.** These were pre-existing bugs that I fixed opportunistically. This suggests the extraction review (status report 2026-08-07) missed them. A grep-based verification step should be part of any module extraction.

---

## f) Up to 50 Things We Should Get Done Next

### High Priority

1. Update `FEATURES.md` to mark `ETag()` as deprecated and reflect the `Middleware` type alias change.
2. Update `TODO_LIST.md` if there are open items about `Middleware` type or ETag adapter.
3. Update go-etag `CHANGELOG.md` with `[Unreleased]` entry for the `Middleware` type alias change.
4. Update go-etag `AGENTS.md` if it documents the `Middleware` type.
5. Add a design decision note to httputil `AGENTS.md` "Non-Obvious Behaviors" explaining why `Middleware` is an alias, not a named type.
6. Add a removal timeline to the `ETag()` `// Deprecated:` directive (e.g., "Removed in v1.0").
7. Audit `docs/v1-stability.md` for any stability claims that depend on `Middleware` being a named type.
8. Verify `// Deprecated:` directive produces IDE warnings (run `staticcheck` or `gopls` check).
9. Annotate historical status reports that reference `httputil.ETag()` or the named `Middleware` type with inline `~~item~~` markers.
10. Run `go test -race -count=10 ./...` to surface any timing-dependent races introduced by the test churn.

### Medium Priority

11. Refactor `MiddlewareStack.Build()` to avoid the intermediate `[]Middleware` slice allocation — iterate `entries` directly.
12. Add a consumer migration snippet to the CHANGELOG `[Unreleased]` entry showing before/after for `ETag()` → `etag.New()`.
13. Review all `docs/status/` reports for stale `Middleware` type references and annotate.
14. Check if `docs/planning/` documents reference the old named `Middleware` type and need updates.
15. Verify the `server_timing` README (if any) references the correct `Middleware` type.
16. Run `go mod tidy` in all three modules to ensure no stale dependencies from the changes.
17. Check if any `// Output:` examples in `example_test.go` need updating beyond the `ETag` → `etag.New` migration (output values should be unchanged, but verify).
18. Review whether `CSRFResponseHeaderMiddleware` (which returns `func(http.Handler) http.Handler` directly, not `Middleware`) should be updated for consistency — it's already compatible via the alias, but the return type is inconsistent with other middleware constructors.
19. Review whether `ClientIPMiddleware` (also returns bare `func(http.Handler) http.Handler`) should return `Middleware` for consistency.
20. Check if the `httpspec` subpackage has any references to `Middleware` that need updating.
21. Verify the `art-dupl` duplication report is still 0 clonegroups after the changes.
22. Run `govulncheck` if available in the devShell to ensure no new vulnerabilities from dependency changes.
23. Check if the go-etag README needs updating to mention the type alias change.
24. Review whether `server_timing/middleware.go` should be mentioned in the httputil AGENTS.md file table (it's a new file in the sub-module).

### Low Priority

25. Consider adding a `go-etag/middleware_test.go` with a compile-time assertion that `Middleware` is assignable to `func(http.Handler) http.Handler` (though the alias makes this tautological, it documents intent).
26. Consider whether the `Middleware` type alias should be mentioned in the `doc.go` package documentation.
27. Review whether any blog posts or external documentation reference the named `Middleware` type and need updating.
28. Consider adding a `Chain` example to `example_test.go` that shows external middleware (e.g., a bare closure) composing without conversion — demonstrating the alias benefit.
29. Review the `recorder.go` doc comment on `Middleware` — it says "wraps an http.Handler" which is accurate for the alias, but could mention the alias nature.
30. Check if the `stack.go` `middlewareEntry` struct doc comment needs updating.
31. Consider whether `Chain` should have a `ChainFromHandler` variant that accepts `http.Handler` middleware (for libraries that return `http.Handler` instead of `func(http.Handler) http.Handler`, like nosurf's `NewPure`).
32. Review whether the `Metrics` middleware's `MetricsRecorder` interface should follow a similar alias pattern if it's used across modules.
33. Audit all `// nolint` directives in the three modified files to ensure they're still needed.
34. Consider adding a fuzz test for `etag.New()` composition with `Chain` to verify no panics from the alias change.
35. Review the go-etag `example_test.go` to ensure examples use `etag.New()` directly (they likely already do, but verify).
36. Check if the httputil `doc.go` package comment mentions `Middleware` as a named type and needs updating.
37. Consider whether the type alias change warrants a version bump (v0.12.0) since it's a semantic change to the public API (though backward-compatible).
38. Review whether any `go.work` workspace changes are needed (likely not, but verify).
39. Check if the `flake.nix` needs updating to reflect the new `server_timing/middleware.go` file.
40. Consider documenting the `type Alias = Underlying` vs `type Name Underlying` distinction in AGENTS.md as a Go design pattern reference.
41. Review whether `WrapServerTiming` should return `Middleware` instead of `(http.ResponseWriter, *http.Request)` for consistency (it's a building block, not middleware, so probably not — but review).
42. Check if any test in `server_timing` tests the `Middleware` return type specifically (the return type changed from unnamed to `Middleware` alias, which is the same type, but verify tests don't assert on type identity).
43. Consider whether the deprecation of `ETag()` should also deprecate the `MiddlewareETag` constant in `stack.go` (probably not — the constant is for the stack name, not the constructor).
44. Review whether the `ETag()` deprecation should be mentioned in the `doc.go` package comment.
45. Consider adding a `// Deprecated:` lint rule to `.golangci.yml` to enforce that deprecated functions are flagged (the `staticcheck` linter does this, verify it's enabled).
46. Check if the `httpspec` tests reference `ETag()` and need migration.
47. Review whether the go-etag `wrapper.go` needs updating for the alias change (it uses `Middleware` internally).
48. Consider whether the `compress_writer.go` or `wrapper.go` files reference `Middleware` in a way that needs updating.
49. Review whether the `recorder.go` `Chain` function should document that it accepts any `func(http.Handler) http.Handler` (via the alias) including external middleware.
50. Run a full `golangci-lint run --fix` across all three modules to catch any auto-fixable issues from the changes.

---

## g) Questions I Cannot Figure Out Myself

1. **Should `ETag()` be removed in v1.0 or v0.12.0?** The deprecation directive needs a removal version. Given that v1.0 is the stability freeze, removing it there is natural, but if you want a faster cycle, v0.12.0 could do it. I cannot determine your release cadence preference.

2. **Should the `Middleware` type alias change warrant a version bump (v0.12.0) now, or wait for the next scheduled release?** The alias is backward-compatible (all existing code compiles), but it's a semantic change to the public API. I cannot determine your versioning policy for backward-compatible semantic changes.

3. **Should `CSRFResponseHeaderMiddleware` and `ClientIPMiddleware` be updated to return `Middleware` (the alias) instead of bare `func(http.Handler) http.Handler` for consistency?** They're already compatible via the alias, but the return types are inconsistent with other constructors. This is a style decision — I cannot determine your preference for return-type uniformity vs leaving working code alone.
