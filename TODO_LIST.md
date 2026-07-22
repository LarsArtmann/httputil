# TODO List — httputil

Short- and mid-term improvement tasks, each verified against the actual code.

_Updated: 2026-07-22._

---

## High Priority

- [ ] **Resolve `GOEXPERIMENT=jsonv2` build requirement** — `health.go` imports `encoding/json/v2` and uses `json.MarshalWrite` (Go 1.27 API), but the module declares `go 1.26.4` and the flake installs Go 1.26. The build fails without `GOEXPERIMENT=jsonv2`. Either pin Go 1.27+ in `flake.nix` or downgrade to `encoding/json` v1. _(rate-limiter status report, item d.1)_
- [ ] **Document `GOEXPERIMENT=jsonv2` in AGENTS.md commands** — the Commands section lists plain `go test ./...` and `golangci-lint run` without the env var. A contributor following the docs hits an immediate build failure. _(rate-limiter status report, item b.1)_
- [ ] **Populate CHANGELOG `[Unreleased]`** — empty since v0.5.0 despite: rate-limiter library switch (`4ce4fdf`, breaking: `burst` param `float64`→`int`), `ParseUintQuery` (`94030f4`), WebSocket upgrade test (`f6c4860`), `GOEXPERIMENT=jsonv2` (`f616f9f`), `go-error-family` upgrade to v0.7.0, new dependency `golang.org/x/time` v0.15.0.

## Medium Priority

- [ ] **Add remaining config field tables to README** — only `CORSConfig`, `ResponseRecorder`, and `CompressionConfig` have field detail tables. Missing: `ETagConfig`, `RateLimitConfig`, `MetricsConfig`, `SecurityHeadersConfig`, `RequestIDConfig`, `ServerConfig`. _(v0.5.0 status report, item b)_
- [ ] **Push v0.5.0 tag to origin** — `git ls-remote --tags origin` shows v0.4.0 as latest remote tag. The v0.5.0 tag exists locally only (commit `204feb9` is on `origin/master`). Decide whether to push v0.5.0 or skip to the next version. _(v0.5.0 status report)_

## Low Priority

- [ ] **Add `TokenBucketLimiter` benchmark** — the rate limiter was switched to `golang.org/x/time/rate`; a benchmark would prove the switch was a net win and guard against regressions. _(rate-limiter status report, item f.6)_
- [ ] **Add body-before-hijack test variant** — the WebSocket upgrade test hijacks immediately (buffer always empty). A handler that writes partial body before upgrading would exercise the `beginPlainResponse()` buffer-drain path. _(websocket status report, item b)_
- [ ] **Mutation-test the ETag path in WebSocket upgrade test** — only the compression path was mutation-tested. The ETag assertions are present but unverified for teeth. _(websocket status report, item b)_

---

## Deferred to v1.0 (breaking naming changes)

- [ ] **`RequestIDConfig.ForwardHeader` → `IncomingHeader`** — names the wrong direction; it reads an incoming header, not forwards one. The most dishonest name in the codebase. _(naming review, data-model review)_
- [ ] **`RequestIDConfig.HeaderName` → `ResponseHeader`** — vague name for the outgoing response header. _(naming review)_
