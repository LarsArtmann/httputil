# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- Classified errors via `go-error-family` integration for `ResponseRecorder` (`Write`, `Hijack`, `Push`)
- Error code constants: `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodePushUnsupported`, `ErrCodePushFailed`
- `CORSConfig.Validate()` method for validating CORS configuration
- `WithClientIP` and `ClientIPFromContext` request context helpers
- `SecurityHeaders` middleware for common security response headers
- `RequestID` middleware for propagating or generating request IDs
- `Recovery` middleware for catching panics and returning 500
- `Timeout` middleware for request deadline enforcement
- `Logging` middleware for structured request logging
- CORS wildcard origin matching (e.g., `*.example.com`)
- `ResponseRecorder.HeaderSnapshot()` for capturing response headers
- Stdlib HTTP error sentinel registration (`http.ErrNotSupported`, `http.ErrAbortHandler`)
- Error message templates for all error codes
- Benchmarks for `itoa`, `join`, `ClientIP`, and CORS middleware
- Fuzz tests for `ClientIP`
- Example functions for all public API
- `CONTRIBUTING.md` contribution guidelines
- `doc.go` package-level documentation
- GitHub Actions CI workflow
- Comprehensive CORS edge case tests

### Changed

- `util.go`: Fixed `itoa` MinInt overflow bug with per-digit absolute value
- `ResponseRecorder`: `Write`, `Hijack`, `Push` return classified errors instead of bare `fmt.Errorf`
- `ResponseRecorder`: Fixed nil-wrapping bug where successful operations returned non-nil errors

### Fixed

- `itoa` overflow for `math.MinInt` (-9223372036854775808)
- `ResponseRecorder.Write` wrapping nil errors on success
- `gosec G115` integer conversion warning in `itoa`

## [0.1.0] - 2026-01-01

### Added

- Initial release with CORS middleware, Client IP extraction, ResponseRecorder, and middleware chaining
