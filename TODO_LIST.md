# TODO List

_Last verified against code: 2026-07-02_

---

## Completed

- [x] Core middleware suite (10 middlewares): CORS, ClientIP, RequestID, SecurityHeaders, Recovery, Timeout, Logging, ResponseRecorder, Compression, ETag
- [x] `Chain()` middleware composition with reverse-order application
- [x] Error classification system with `go-error-family` (5 error codes)
- [x] `RegisterErrorClassifications()` for stdlib HTTP error mapping
- [x] `ResponseRecorder` with `WriteHeader`, `Write`, `Flush`, `Hijack` support
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
- [x] `Compression` `Accept-Encoding` negotiation with RFC 7231 q-value parsing
- [x] `Compression` deflate support via `compress/flate`
- [x] `Compression` `WriterFactory` plugin interface for brotli/zstd/lz4
- [x] `Compression` per-factory writer pools and buffer pre-allocation
- [x] `RequestID` time-ordered ID generator with amortized `crypto/rand` buffer
- [x] `ETag` with RFC 7232 compliance, 1MB memory limit, zero-allocation hex encoding
- [x] `wrapper.go` shared ResponseWriter wrapper extracting duplication from compress/etag writers
- [x] `example_test.go` with 11 example functions
- [x] Benchmarks for all middlewares (CORS, ClientIP, Compression, ETag, RequestID, SecurityHeaders, Recovery, Timeout, Logging, ResponseRecorder, Chain)
- [x] Fuzz tests for ClientIP, Compression, ETag, CORS, RequestID
- [x] Integration tests for `Chain(Compression, ETag)` and `Chain(Recovery, Logging, CORS)` ordering
- [x] Integration test for WebSocket upgrade (Hijack) through Compression + ETag
- [x] Content-Length preservation test for small responses through Compression + ETag
- [x] Example functions for all public API (11 examples)
- [x] `httpspec` behavioral spec subpackage (13 standard specs + 4 helper builders)
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
- [x] `go test ./...` and `go vet ./...` clean, >90% coverage
- [x] FEATURES.md — honest feature inventory
- [x] TODO_LIST.md — centralized task list

## Not Started (v0.2.0+)

### Near-term

- [x] Improve test coverage to 90%+ (achieved)
- [x] Make content-type filtering configurable via `CompressionConfig`
- [x] Add `MiddlewareStack` type with ordering validation
- [x] Add `ResponseWriter` capability interface for Hijack/Flush

### Medium-term

- [x] Implement deflate support using `compress/flate`
- [x] Add `Accept-Encoding` quality value parsing per RFC 7231
- [x] Evaluate streaming ETag option using rolling hash — **Rejected**: HTTP requires headers before body, buffering is mandatory

### Worth considering

- [x] Consider request/response metrics middleware — implemented as `Metrics()` with pluggable `MetricsRecorder` interface
- [x] Consider rate-limiting middleware — implemented as `RateLimit()` with pluggable `RateLimiter` interface and `TokenBucketLimiter` built-in
- [x] Consider request body size limit middleware — implemented as `MaxBodySize()`
