# TODO List

_Last verified against code: 2026-06-08_

---

## Completed

- [x] Core middleware suite (10 middlewares): CORS, ClientIP, RequestID, SecurityHeaders, Recovery, Timeout, Logging, ResponseRecorder, Compression, ETag
- [x] `Chain()` middleware composition with reverse-order application
- [x] Error classification system with `go-error-family` (7 error codes)
- [x] `RegisterErrorClassifications()` for stdlib HTTP error mapping
- [x] `ResponseRecorder` with `WriteHeader`, `Write`, `Flush`, `Hijack`, `Push` support
- [x] `HeaderSnapshot()` for capturing response headers
- [x] `CORSConfig.Validate()` for startup configuration validation
- [x] `CompressionConfig.Validate()` for startup configuration validation
- [x] `RequestIDConfig.Validate()` for startup configuration validation (nil GenerateID, empty headers)
- [x] `ETagConfig.Validate()` for startup configuration validation (MaxBufferSize guard)
- [x] `SecurityHeadersConfig.Validate()` for startup configuration validation (all fields optional)
- [x] CORS wildcard origin matching (`*.example.com`)
- [x] Client IP extraction with `X-Forwarded-For` → `X-Real-IP` → `RemoteAddr` precedence
- [x] `WithClientIP()` / `ClientIPFromContext()` / `ClientIPMiddleware()` context helpers
- [x] `RequestIDFromContext()` context helper
- [x] `Compression` with `sync.Pool`, content-type filtering, bounded buffering
- [x] `ETag` with RFC 7232 compliance, 1MB memory limit, zero-allocation hex encoding
- [x] `wrapper.go` shared ResponseWriter wrapper extracting duplication from compress/etag writers
- [x] `example_test.go` with 11 example functions
- [x] Benchmarks for all middlewares (CORS, ClientIP, Compression, ETag, Itoa, Join, RequestID, SecurityHeaders, Recovery, Timeout, Logging, ResponseRecorder, Chain)
- [x] Fuzz tests for ClientIP, Compression, ETag, CORS, RequestID
- [x] Integration tests for `Chain(Compression, ETag)` and `Chain(Recovery, Logging, CORS)` ordering
- [x] Integration test for WebSocket upgrade (Hijack) through Compression + ETag
- [x] Content-Length preservation test for small responses through Compression + ETag
- [x] Example functions for all public API (11 examples)
- [x] Document brotli policy decision in README
- [x] Fix data race in `getGzipPool()` (added `sync.RWMutex`)
- [x] Improve flake.nix (source filtering, writeShellApplication, format check)
- [x] Strengthen CI workflow (build, vet, benchmark, govulncheck steps)
- [x] Pin golangci-lint version in CI (v2.12)
- [x] Improve .golangci.yml (gocognit test exclusion, goexperiment build tags, varnamelen ignores)
- [x] GitHub Actions CI workflow (test + lint)
- [x] Release workflow with `govulncheck`
- [x] Nix flake for reproducible dev environment
- [x] `CHANGELOG.md` with version history
- [x] `AGENTS.md` with architecture reference and lint rules
- [x] `docs/DOMAIN_LANGUAGE.md` with domain glossary
- [x] `doc.go` package-level godoc
- [x] `golangci-lint` ~70 linters, 0 issues
- [x] 110 tests passing, `go vet` clean, 89.1% coverage
- [x] FEATURES.md — honest feature inventory
- [x] TODO_LIST.md — centralized task list

## Not Started

- [ ] Implement deflate support using `compress/flate`
- [ ] Add `Accept-Encoding` quality value parsing per RFC 7231
- [ ] Make content-type filtering configurable via `CompressionConfig`
- [ ] Add `MiddlewareStack` type with ordering validation
- [ ] Add `ResponseWriter` capability interface for Hijack/Push/Flush
- [ ] Improve test coverage to 90%+ (currently 89.1%)
- [ ] Evaluate streaming ETag option using rolling hash
- [ ] Consider request/response metrics middleware
- [ ] Consider rate-limiting middleware
- [ ] Consider request body size limit middleware
