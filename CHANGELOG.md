# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

## [0.1.1] - 2026-06-08

### Fixed

- `CORS()`: Eliminated data race where `allowOrigin` was a shared mutable closure variable across concurrent requests
- `Compression()`: Added nil guard in gzip writer pool — `gzip.NewWriterLevel` errors now panic at construction time instead of producing nil writers
- Removed unreachable `errPoolTypeMismatch` dead code, replaced with `panic()` for impossible states

### Changed

- Updated CHANGELOG, AGENTS.md, and status docs to reflect accurate metrics (112 tests, 91.2% coverage)

## [0.1.0] - 2026-06-08

### Added

- CORS middleware with configurable origins, methods, headers, credentials, and preflight handling
- CORS wildcard origin matching (e.g., `*.example.com`)
- `CORSConfig.Validate()`, `CompressionConfig.Validate()`, `RequestIDConfig.Validate()`, `ETagConfig.Validate()`, `SecurityHeadersConfig.Validate()` — all config types have startup validation
- `DefaultCORSConfig()` with permissive development defaults (allows all origins)
- Client IP extraction (`ClientIP`) with `X-Forwarded-For` → `X-Real-IP` → `RemoteAddr` precedence
- `WithClientIP()`, `ClientIPFromContext()`, `ClientIPMiddleware()` context helpers
- `ResponseRecorder` with `WriteHeader`, `Write`, `Flush`, `Hijack`, `Push` support
- `HeaderSnapshot()` for capturing response headers
- `Chain()` for composing middleware in declaration order (first = outermost)
- Security headers middleware (`SecurityHeaders`) with sensible defaults
- Request ID middleware (`RequestID`) with propagation and generation
- `RequestIDFromContext()` context helper
- Panic recovery middleware (`Recovery`) with stack trace logging
- Request timeout middleware (`Timeout`) with context deadline enforcement
- Structured request logging middleware (`Logging`)
- Response compression middleware (`Compression`) with gzip, `sync.Pool`, content-type filtering, and bounded buffering
- `CompressionConfig.Validate()` for startup configuration validation (gzip levels, min size)
- `RequestIDConfig.Validate()` for startup validation (nil GenerateID, empty headers)
- `ETagConfig.Validate()` for startup validation (non-positive MaxBufferSize)
- `SecurityHeadersConfig.Validate()` for startup validation (all fields optional, consistent API)
- Classified errors via `go-error-family` integration for `ResponseRecorder` (`Write`, `Hijack`, `Push`), `compressWriter`, and `etagWriter`
- 7 error code constants: `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodePushUnsupported`, `ErrCodePushFailed`, `ErrCodeCompressWriteFailed`, `ErrCodeETagWriteFailed`
- `RegisterErrorClassifications()` for stdlib HTTP error mapping
- Error message templates (`what/why/fix/wayOut`) for all classified errors
- `wrapper.go` shared `ResponseWriter` wrapper eliminating ~80 lines of duplication from compress/etag writers
- 112 tests, 11 example functions, 15 benchmarks, 5 fuzz tests
- 91.2% test coverage
- `golangci-lint` with ~70 linters, 0 issues
- GitHub Actions CI workflow (test, lint, build, vet)
- Release workflow with `govulncheck`
- Nix flake for reproducible development environment
- `CHANGELOG.md`, `FEATURES.md`, `TODO_LIST.md`, `AGENTS.md`
- `docs/DOMAIN_LANGUAGE.md` with domain glossary
- `doc.go` package-level godoc
- `CONTRIBUTING.md` and `CODE_OF_CONDUCT.md`

### Changed

- `util.go`: Fixed `itoa` MinInt overflow bug with per-digit absolute value
- `ResponseRecorder`: `Write`, `Hijack`, `Push` return classified errors instead of bare `fmt.Errorf`
- `ResponseRecorder`: Fixed nil-wrapping bug where successful operations returned non-nil errors
- `CORS()`: Fixed data race where `allowOrigin` was a shared mutable closure variable across concurrent requests
