# TODO List — httputil

Short- and mid-term improvement tasks, each verified against the actual code.

_Updated: 2026-07-26._

---

## Medium Priority

- [ ] **Add remaining config field tables to README** — only `CORSConfig`, `ResponseRecorder`, and `CompressionConfig` have field detail tables. Missing: `ETagConfig`, `RateLimitConfig`, `MetricsConfig`, `SecurityHeadersConfig`, `RequestIDConfig`, `ServerConfig`. _(v0.5.0 status report, item b)_

## Low Priority

- [ ] **Add `TokenBucketLimiter` benchmark** — the rate limiter was switched to `golang.org/x/time/rate`; a benchmark would prove the switch was a net win and guard against regressions. _(rate-limiter status report, item f.6)_
- [ ] **Add body-before-hijack test variant** — the WebSocket upgrade test hijacks immediately (buffer always empty). A handler that writes partial body before upgrading would exercise the `beginPlainResponse()` buffer-drain path. _(websocket status report, item b)_
- [ ] **Mutation-test the ETag path in WebSocket upgrade test** — only the compression path was mutation-tested. The ETag assertions are present but unverified for teeth. _(websocket status report, item b)_

---

_The breaking naming changes (`ForwardHeader`→`IncomingHeader`,
`HeaderName`→`ResponseHeader`) were completed in v0.7.0. The v1.0 stability plan
lives in [ROADMAP.md](ROADMAP.md)._
