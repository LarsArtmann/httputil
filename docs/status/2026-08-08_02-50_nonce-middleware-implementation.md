# Status Report: 2026-08-08 02:50 — Nonce Middleware Implementation

## Session Summary

Implemented CSP nonce support (`nonce.go` + `nonce_test.go`) for per-request
inline script/style allow-listing. The implementation compiles, lints clean
(0 issues across ~70 linters), and passes `go test -race -count=10`.

**But the integration is incomplete.** Several files and docs that every
other middleware touches were missed entirely.

---

## a) FULLY DONE

| Item                              | Status | Evidence                                                                                                                                                                                                          |
| --------------------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `nonce.go` — core implementation  | DONE   | `NonceConfig`, `Nonce()`, `WithNonce()`, `NonceFromContext()`, `NonceFromRequest()`, `RecommendedCSPWithNonce()`, `Validate()`                                                                                    |
| `nonce_test.go` — test suite      | DONE   | 16 test functions + 1 benchmark. Covers generation, context storage, CSP header, nil CSPBuilder, uniqueness, custom size, base64 encoding, from-request/from-context, Validate (3 cases), RecommendedCSPWithNonce |
| Lint gate                         | DONE   | `golangci-lint run` = 0 issues                                                                                                                                                                                    |
| Race detection                    | DONE   | `go test -race -count=10 ./...` = pass                                                                                                                                                                            |
| `AGENTS.md` file table entry      | DONE   | Row added with exports + purpose                                                                                                                                                                                  |
| `AGENTS.md` non-obvious behaviors | DONE   | Bullet added documenting per-request generation + CSPBuilder nil mode                                                                                                                                             |
| `AGENTS.md` file count            | DONE   | Updated 33 → 34                                                                                                                                                                                                   |
| `AGENTS.md` test files list       | DONE   | Added `nonce_test.go` to middleware list                                                                                                                                                                          |
| `AGENTS.md` makezero note         | DONE   | Added `nonce.go` to makezero false-positive list                                                                                                                                                                  |
| Formatting                        | DONE   | `golangci-lint fmt` applied                                                                                                                                                                                       |

---

## b) PARTIALLY DONE

Nothing — the implementation is either done or not started, no half-measures.

---

## c) NOT STARTED (Critical Omissions)

> **Annotation (2026-08-08 04:00):** All 11 items below (1-7 integration gaps +
> 8-10 testing gaps) were resolved in the subsequent session. See the
> comprehensive audit at `2026-08-08_03-20_nonce-middleware-comprehensive-audit.md`
> for details. Inline markers below.

### Integration Gaps (Every Other Middleware Has These)

| #   | Missing Item                                        | Impact                                                                                                               | Existing Pattern                                                                                                                           |
| --- | --------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **`MiddlewareNonce` constant in `stack.go`**        | ~~Resolved at `stack.go:25` — `MiddlewareNonce = "nonce"` added.~~                                                   | Every middleware has a `Middleware*` constant (lines 11-25). ETag and Decompression both got theirs on the same session they were created. |
| 2   | **`buildFullStack` in `stack_integration_test.go`** | ~~Resolved — Nonce added to `buildFullStack`, count bumped to 18, `verifyGETHeaders` checks CSP header presence.~~   | Every middleware is added here. Count was already bumped 16→17 for ETag.                                                                   |
| 3   | **CHANGELOG.md `[Unreleased]` entry**               | ~~Resolved — detailed entry added with all features listed.~~                                                        | Decompression, ETag, TLSConfig, etc. all have detailed entries.                                                                            |
| 4   | **`ExampleNonce` in `example_test.go`**             | ~~Resolved — `ExampleNonce` added with `// Output:` directive.~~                                                     | Every middleware has an `Example*` function with `// Output:` directive (25 examples total).                                               |
| 5   | **FEATURES.md update**                              | ~~Resolved — nonce row added, suite count updated to 18, header refreshed.~~                                         | Decompression and ETag both have entries.                                                                                                  |
| 6   | **README.md update**                                | ~~Resolved — CSP Nonce section with code examples, ProductionCSPWithNonce, NonceAttr, ordering + caching guidance.~~ | Every middleware is documented here.                                                                                                       |
| 7   | **DOMAIN_LANGUAGE.md update**                       | ~~Resolved — CSP Nonce bounded context, entity, value objects, 8 commands added.~~                                   | Decompression got a full bounded context (entity, value objects, commands, events, rules).                                                 |

### Testing Gaps

| #   | Missing Item                                | Impact                                                                                                                   |
| --- | ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| 8   | **Fuzz test for `RecommendedCSPWithNonce`** | ~~Resolved — `FuzzNonce` in `nonce_fuzz_test.go`, 610K execs, 0 failures, covers both CSP builders for CRLF injection.~~ |
| 9   | **`generateNonce` isolated benchmark**      | ~~Resolved — `BenchmarkGenerateNonce` added to `nonce_test.go`.~~                                                        |
| 10  | **CSP header injection test**               | ~~Resolved — covered by `FuzzNonce` which verifies base64 validity + CRLF resistance across all sizes 1-1024.~~          |

---

## d) TOTALLY FUCKED UP

Nothing catastrophically broken — the code works and is clean. But the
omissions above mean the feature is **not wired into the ecosystem**. A
user who adds `Nonce()` to their stack won't find it in the stack name
constants, won't see it in examples, and won't find it in the README.

---

## e) WHAT WE SHOULD IMPROVE

### Design Issues to Address

1. **CSP header conflict between `SecurityHeaders` and `Nonce`** — ~~Resolved
   via documentation: README + nonce.go doc comment specify placing `Nonce`
   after `SecurityHeaders`. Tests `TestNonce_OverwritesStaticCSP` (correct
   ordering) and `TestNonce_BeforeSecurityHeaders_LosesCSP` (wrong ordering)
   document the behavior.~~

2. **`Nonce()` doesn't call `Validate()`** — ~~Resolved: `Nonce()` now calls
   `cfg.Validate()` at construction time (matching CSRF pattern). `Validate()`
   was also updated to accept `Size == 0` as valid (use default). Invalid
   configs are logged via `slog.Error`.~~

3. **`RecommendedCSPWithNonce` is minimal** — ~~Resolved: `ProductionCSPWithNonce`
   added with `object-src 'none'`, `base-uri 'self'`, `frame-ancestors 'none'`.~~

4. **No `Nonce` in stack ordering validation** — ~~Resolved via documentation
   and tests. Structural enforcement in `stack.Validate()` was not added (not
   needed — the conflict is last-writer-wins and the inner middleware wins,
   which is the desired behavior when ordering is correct).~~

---

## f) Next 50 Things to Get Done

### Immediate (Required for Feature Completeness)

1. Add `MiddlewareNonce = "nonce"` to `stack.go` name constants.
2. Add `Nonce()` to `buildFullStack` in `stack_integration_test.go`, bump count 17 → 18.
3. Add CHANGELOG.md `[Unreleased]` entry for nonce feature.
4. Add `ExampleNonce` to `example_test.go` with `// Output:` directive.
5. Add nonce to FEATURES.md feature inventory.
6. Add nonce to README.md: feature section, API table, middleware ordering.
7. Add CSP nonce bounded context to DOMAIN_LANGUAGE.md.
8. Run `go test -race ./...` + `golangci-lint run` after all above.

### Hardening & Testing

9. Add `FuzzNonce` fuzz test — verify CSP header is always valid for random byte inputs.
10. Add isolated `BenchmarkGenerateNonce` (crypto/rand + base64 only, no middleware overhead).
11. Add test: base64 boundary characters don't corrupt CSP header value.
12. Add test: `Nonce()` chained after `SecurityHeaders` — verify CSP header is correct.
13. Add test: `Nonce()` chained before `SecurityHeaders` — verify the conflict is detectable or documented.
14. Add test: `NonceConfig.Validate()` rejects `Size: 0` vs `Nonce()` accepting it (document the split behavior).
15. Add test: nonce survives through `Chain()` middleware composition.
16. Consider a `Nonce` + `Recovery()` interaction test (nonce must be available even if downstream panics).
17. Add test: multiple `Nonce()` instances in one stack produce different nonces per request.

### Design & API

18. Decide CSP conflict resolution between `SecurityHeaders` and `Nonce` (document or merge).
19. Consider `SecurityHeadersConfig.NonceAware bool` to auto-inject `'nonce-...'` from context.
20. Consider `ProductionCSPWithNonce(nonce)` with more directives (`img-src`, `connect-src`, `object-src 'none'`, `base-uri 'self'`).
21. Consider `NonceConfig.Generator func() string` override for testability (like `RequestIDConfig.GenerateID`).
22. Consider exposing `GenerateNonce(size int) string` as a public function for non-middleware use.
23. Consider `NonceMiddlewareWhen(condition)` variant like `servertiming.ServerTimingMiddlewareWhen`.
24. Decide: should `Nonce()` call `cfg.Validate()`? (Currently doesn't, like other middleware.)

### Documentation

25. Update `docs/v1-stability.md` with `NonceConfig` and `DefaultNonceConfig` stability classification.
26. Add nonce middleware ordering guidance to README.md (before or after SecurityHeaders?).
27. Add CSP nonce usage example with templ in README.md or a dedicated guide.
28. Update AGENTS.md "Allowed Dependencies" — nonce uses only stdlib (`crypto/rand`, `encoding/base64`), no new deps. Verify this is clear.
29. Update AGENTS.md middleware count in any place that says "13 middlewares" or "17 middlewares" (multiple locations).
30. Add nonce to `docs/v1-stability.md` error classification section (if nonce errors are classified).

### Code Quality

31. Consider whether `generateNonce` should use the amortized random buffer from `id_generator.go` (`drawRandomBytes`) instead of a direct `crypto/rand.Read` call. The ID generator amortizes syscalls across 256 IDs; nonce does one syscall per request.
32. Verify `nonce.go` doesn't need `go-error-family` classification (it currently panics on `rand.Read` failure — is panic the right call? Other code also panics here, so this is consistent).
33. Consider adding `Nonce` to the `httpspec` standard specs (verify CSP header is present when nonce middleware is used).
34. Run `art-dupl --type-aware` to verify no new code duplication introduced.
35. Run `govulncheck` to verify no new vulnerabilities from the implementation.

### Release Prep

36. Update coverage badge (`update-coverage-badge.sh`) after nonce tests are finalized.
37. Bump any version references if preparing for a tag.
38. Verify `go test -bench=. ./...` includes nonce benchmarks.
39. Run `nix flake check` to verify Nix build still passes.
40. Run `nix run .#test` and `nix run .#lint` (if those apps exist).

### Ecosystem Integration

41. Verify nonce works correctly with `Compression` middleware (CSP header must not be compressed away).
42. Verify nonce works with `ETag` middleware (nonce changes per request, so ETag must hash body only, not headers).
43. Verify nonce works with `CORS` middleware (CSP and CORS headers coexist).
44. Verify nonce works with `ServerTiming` (both set headers, no conflict).
45. Consider a nonce + HTMX integration test (HTMX requires inline scripts).
46. Consider how nonce interacts with `CSRFTokenHTMLMeta()` (both inject into HTML `<head>`).

### Polish

47. Add `// nonce.go` to any file-level documentation index or cross-reference.
48. Consider `NonceConfig` doc comment mentioning interaction with `SecurityHeaders`.
49. Add `# CSP Nonce` section to README.md with before/after comparison.
50. Consider a blog post or guide on "CSP nonce-based inline script policy with httputil."

---

## Quality Gate Summary

| Gate                                       | Status                                                                                                     |
| ------------------------------------------ | ---------------------------------------------------------------------------------------------------------- |
| `golangci-lint run`                        | PASS (0 issues)                                                                                            |
| `go test -race -count=10 ./...`            | PASS                                                                                                       |
| `golangci-lint fmt`                        | PASS                                                                                                       |
| Feature completeness (vs other middleware) | **FAIL** — missing stack constant, integration test, CHANGELOG, example, README, FEATURES, DOMAIN_LANGUAGE |
| Ecosystem integration                      | **FAIL** — not in `buildFullStack`, no CSP conflict resolution, no interaction tests                       |

---

## Verdict

The core implementation is solid and well-tested in isolation. But the
feature is **not integrated into the library ecosystem**. It's a standalone
file that passes its own tests but isn't wired into the stack system,
documentation, or examples. Every previous middleware (ETag, Decompression,
CSRF) was shipped with full integration on day one. Nonce should match that
standard.

**Estimated remaining work:** ~30 minutes for items 1-8 (critical path),
~1-2 hours for items 9-35 (hardening + docs).
