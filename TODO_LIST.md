# TODO List — httputil

Short- and mid-term improvement tasks, each verified against the actual code.

_Updated: 2026-07-22._

---

## High Priority

- [x] **Populate CHANGELOG `[Unreleased]`** — post-v0.6.0 changes logged: flake GOEXPERIMENT fix, DOMAIN_LANGUAGE corrections, CONTRIBUTING rewrite, HTML inline corrections, D2/SVG updates. _(11-01 status report, item d.2)_
- [ ] **Resolve `encoding/json/v2` permanently** — `health.go` imports `encoding/json/v2` and uses `json.MarshalWrite` (Go 1.27 API), but the module declares `go 1.26.4` and the flake pins Go 1.26. `GOEXPERIMENT=jsonv2` is now set in `flake.nix` shellHook and all app scripts as a workaround. For a permanent fix: either upgrade the flake to Go 1.27+ or downgrade `health.go` to `encoding/json` v1. _(rate-limiter status report, item d.1)_

## Medium Priority

- [ ] **Add remaining config field tables to README** — only `CORSConfig`, `ResponseRecorder`, and `CompressionConfig` have field detail tables. Missing: `ETagConfig`, `RateLimitConfig`, `MetricsConfig`, `SecurityHeadersConfig`, `RequestIDConfig`, `ServerConfig`. _(v0.5.0 status report, item b)_
- [ ] **Decide release strategy post-v0.6.0** — v0.6.0 tagged and pushed. Next release should resolve the `encoding/json/v2` question (Go version bump or revert). Determine whether the jsonv2 experiment stays or goes before tagging v0.7.0.

## Low Priority

- [ ] **Add `TokenBucketLimiter` benchmark** — the rate limiter was switched to `golang.org/x/time/rate`; a benchmark would prove the switch was a net win and guard against regressions. _(rate-limiter status report, item f.6)_
- [ ] **Add body-before-hijack test variant** — the WebSocket upgrade test hijacks immediately (buffer always empty). A handler that writes partial body before upgrading would exercise the `beginPlainResponse()` buffer-drain path. _(websocket status report, item b)_
- [ ] **Mutation-test the ETag path in WebSocket upgrade test** — only the compression path was mutation-tested. The ETag assertions are present but unverified for teeth. _(websocket status report, item b)_
- [ ] **Fix `modularity.html` body prose** — line 636 still says "at 28 files it is approaching the point" despite the inline correction banner above. Actual file count is ~69. _(11-01 status report, item b.1)_

---

## Deferred to v1.0 (breaking naming changes)

- [ ] **`RequestIDConfig.ForwardHeader` → `IncomingHeader`** — names the wrong direction; it reads an incoming header, not forwards one. The most dishonest name in the codebase. _(naming review, data-model review)_
- [ ] **`RequestIDConfig.HeaderName` → `ResponseHeader`** — vague name for the outgoing response header. _(naming review)_
