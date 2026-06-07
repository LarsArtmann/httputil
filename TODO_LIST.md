# TODO List

_Last verified against code: 2026-06-07_

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
- [x] CORS wildcard origin matching (`*.example.com`)
- [x] Client IP extraction with `X-Forwarded-For` → `X-Real-IP` → `RemoteAddr` precedence
- [x] `WithClientIP()` / `ClientIPFromContext()` / `ClientIPMiddleware()` context helpers
- [x] `RequestIDFromContext()` context helper
- [x] `Compression` with `sync.Pool`, content-type filtering, bounded buffering
- [x] `ETag` with RFC 7232 compliance, 1MB memory limit, zero-allocation hex encoding
- [x] `wrapper.go` shared ResponseWriter wrapper extracting duplication from compress/etag writers
- [x] `example_test.go` with 6 example functions
- [x] Benchmarks for CORS, ClientIP, Compression, ETag, Itoa, Join
- [x] Fuzz tests for ClientIP, Compression, ETag
- [x] Integration tests for `Chain(Compression, ETag)` ordering
- [x] GitHub Actions CI workflow (test + lint)
- [x] Release workflow with `govulncheck`
- [x] Nix flake for reproducible dev environment
- [x] `CHANGELOG.md` with version history
- [x] `AGENTS.md` with architecture reference and lint rules
- [x] `docs/DOMAIN_LANGUAGE.md` with domain glossary
- [x] `doc.go` package-level godoc
- [x] `golangci-lint` ~70 linters, 0 issues
- [x] 114+ tests passing, `go vet` clean, 86.9% coverage
- [x] FEATURES.md — honest feature inventory
- [x] TODO_LIST.md — centralized task list

## Not Started

- [x] Add `BenchmarkRequestID` — benchmark for request ID middleware
- [x] Add `BenchmarkSecurityHeaders` — benchmark for security headers middleware
- [x] Add `BenchmarkRecovery` — benchmark for panic recovery middleware
- [x] Add `BenchmarkTimeout` — benchmark for timeout middleware
- [x] Add `BenchmarkLogging` — benchmark for logging middleware
- [x] Add `BenchmarkResponseRecorder` — benchmark for response recorder
- [x] Add `BenchmarkChain` — benchmark for middleware chaining
- [x] Add `ExampleRequestID` — godoc example for request ID middleware
- [x] Add `ExampleSecurityHeaders` — godoc example for security headers middleware
- [x] Add `ExampleRecovery` — godoc example for panic recovery middleware
- [x] Add `ExampleTimeout` — godoc example for timeout middleware
- [x] Add `ExampleLogging` — godoc example for logging middleware
- [x] Add `FuzzCORS` — fuzz test for CORS origin matching
- [x] Add `FuzzRequestID` — fuzz test for request ID generation
- [x] Add integration tests for `Chain(Recovery, Logging, CORS)` and other common combinations
- [x] Document brotli policy decision in README
- [x] Fix data race in `getGzipPool()` (added `sync.RWMutex`)
- [x] Add FEATURES.md with honest feature inventory
- [x] Add TODO_LIST.md with completed and pending tasks
- [x] Improve flake.nix (source filtering, writeShellApplication, format check)
- [x] Strengthen CI workflow (build, vet, benchmark steps)
- [x] Improve .golangci.yml (gocognit test exclusion, goexperiment build tags, varnamelen ignores)
- [ ] Add WebSocket upgrade test through Compression + ETag
- [ ] Add `Content-Length` preservation test for small responses
- [ ] Implement deflate support using `compress/flate`
- [ ] Add `Accept-Encoding` quality value parsing per RFC 7231
- [ ] Make content-type filtering configurable via `CompressionConfig`
- [ ] Add `MiddlewareStack` type with ordering validation
- [ ] Add `ResponseWriter` capability interface for Hijack/Push/Flush
- [ ] Improve test coverage to 90%+
- [ ] Evaluate streaming ETag option using rolling hash
- [ ] Consider request/response metrics middleware
- [ ] Consider rate-limiting middleware
- [ ] Consider request body size limit middleware
