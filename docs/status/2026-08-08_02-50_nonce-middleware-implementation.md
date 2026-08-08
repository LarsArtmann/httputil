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

### Integration Gaps (Every Other Middleware Has These)

| #   | Missing Item                                        | Impact                                                                                                                 | Existing Pattern                                                                                                                           |
| --- | --------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **`MiddlewareNonce` constant in `stack.go`**        | Nonce can't participate in `MiddlewareStack` ordering validation with a well-known name. **This is the biggest miss.** | Every middleware has a `Middleware*` constant (lines 11-25). ETag and Decompression both got theirs on the same session they were created. |
| 2   | **`buildFullStack` in `stack_integration_test.go`** | Test comment says "all 17 middlewares" — now stale. Nonce not in the full-stack composition test.                      | Every middleware is added here. Count was already bumped 16→17 for ETag.                                                                   |
| 3   | **CHANGELOG.md `[Unreleased]` entry**               | Release history gap. Every feature gets an entry here.                                                                 | Decompression, ETag, TLSConfig, etc. all have detailed entries.                                                                            |
| 4   | **`ExampleNonce` in `example_test.go`**             | No runnable example. `testableexamples` linter will flag this if the file is touched.                                  | Every middleware has an `Example*` function with `// Output:` directive (25 examples total).                                               |
| 5   | **FEATURES.md update**                              | Nonce not listed in feature inventory.                                                                                 | Decompression and ETag both have entries.                                                                                                  |
| 6   | **README.md update**                                | No nonce documentation, no API table entry, no middleware ordering guidance for CSP nonce.                             | Every middleware is documented here.                                                                                                       |
| 7   | **DOMAIN_LANGUAGE.md update**                       | "nonce" only appears in the CSRF row. No CSP nonce bounded context.                                                    | Decompression got a full bounded context (entity, value objects, commands, events, rules).                                                 |

### Testing Gaps

| #   | Missing Item                                | Impact                                                                                                                                                                                 |
| --- | ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 8   | **Fuzz test for `RecommendedCSPWithNonce`** | No CRLF injection fuzzing. `server_timing` has CRLF fuzz tests; nonce CSP output should be similarly hardened.                                                                         |
| 9   | **`generateNonce` isolated benchmark**      | Current benchmark includes full middleware overhead. No isolated crypto/rand + base64 benchmark.                                                                                       |
| 10  | **CSP header injection test**               | No test verifying that nonces containing special characters (e.g., base64 `+`, `/`, `=`) don't break the CSP header. (Though `RawURLEncoding` avoids `+`/`/`/`=`, this is not tested.) |

---

## d) TOTALLY FUCKED UP

Nothing catastrophically broken — the code works and is clean. But the
omissions above mean the feature is **not wired into the ecosystem**. A
user who adds `Nonce()` to their stack won't find it in the stack name
constants, won't see it in examples, and won't find it in the README.

---

## e) WHAT WE SHOULD IMPROVE

### Design Issues to Address

1. **CSP header conflict between `SecurityHeaders` and `Nonce`** — If a
   user chains `SecurityHeaders` (which may set a static CSP) with `Nonce`
   (which sets a dynamic CSP), `Header.Set` is last-writer-wins. This is a
   **footgun**. Options:
   - Document that `Nonce` must be placed after `SecurityHeaders` (and
     `SecurityHeaders` CSP should be empty/suppressed when using nonces)
   - Add a `NonceFromContext`-aware CSP builder to `SecurityHeaders`
   - Add a `SecurityHeadersConfig.NonceAware bool` flag

2. **`Nonce()` doesn't call `Validate()`** — Consistent with other
   middleware, but `Size == 0` silently uses the default instead of being
   validated. A user who sets `Size: 8` gets 8 bytes and doesn't know it's
   below the CSP Level 3 minimum. `Validate()` exists but is never called
   by `Nonce()`.

3. **`RecommendedCSPWithNonce` is minimal** — Only covers `default-src`,
   `script-src`, `style-src`. Missing `img-src`, `connect-src`, `font-src`,
   `object-src 'none'`, `base-uri 'self'`. Production CSP typically needs
   more. Consider a `ProductionCSPWithNonce` variant or a `CSPBuilder`
   that merges into an existing policy.

4. **No `Nonce` in stack ordering validation** — `Validate()` in `stack.go`
   enforces Recovery is outermost. Should it also enforce that Nonce is
   before SecurityHeaders (or vice versa)? At minimum document the
   interaction.

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
