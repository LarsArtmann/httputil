# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to TODO_LIST.md.

## Themes

### 1. v1.0 — API honesty and stability commitment

The library is at v0.7.1 with a complete, tested middleware suite and a
frozen v1.0 API surface documented in `docs/v1-stability.md`. One more
stabilization cycle (v0.8.0) will close remaining depth gaps before the v1.0
commitment.

Resolved in v0.7.0:

- ~~Rename `RequestIDConfig.ForwardHeader` to `IncomingHeader`~~ — done in v0.7.0.
- ~~Rename `RequestIDConfig.HeaderName` to `ResponseHeader`~~ — done in v0.7.0.
- ~~Evaluate flipping `DenyUnmatched` default to `true`~~ — done in v0.7.0
  (see `docs/research/deny-unmatched-default-evaluation.md`).
- ~~Define which APIs are frozen at v1.0~~ — done: `docs/v1-stability.md`.

Remaining raw ideas:

### 2. Extensibility without new dependencies

The dependency policy (stdlib + `go-error-family` + `golang.org/x/time` only) is
a deliberate feature. Extensibility should come through documented examples and
plugin seams, not core dependencies.

Raw ideas:

- ~~Brotli / zstd / lz4 encoder examples via the `WriterFactory` plugin
  pattern~~ — documented example exists: `docs/integrations/brotli-zstd.md`.
- ~~A distributed (Redis-backed) `RateLimiter` implementation~~ — documented
  example exists: `docs/integrations/redis-ratelimiter.md`.
- ~~A Prometheus-compatible `MetricsRecorder` implementation~~ — documented
  example exists: `docs/integrations/prometheus-metrics.md`.
- Request body decompression middleware as a counterpart to `Compression`.

### 3. Depth and confidence

The suite is broad; v1.0 should make it deep enough to trust without audit.

Raw ideas:

- Fuzz tests and `Example*` functions for the newer surface (`ParseUintQuery`,
  `ReadyHandlerWithProbe`, `DenyUnmatched`, `EvictionTTL`) — seeds added in
  v0.7.0; `-fuzztime` runs pending.
- Close the remaining coverage gaps in compression error branches (Flush,
  Close, streaming write errors). Several gaps closed in v0.7.0; a handful of
  error branches remain below 100%.
- An `httpspec` spec covering common CORS header behavior.

## Non-goals

Things we are deliberately NOT pursuing and why:

- **HTTP/2 Server Push:** removed in Chrome 2023, absent from HTTP/3. All
  `http.Pusher` code was deleted in v0.3.0.
- **Streaming ETag with a rolling hash:** HTTP requires headers before the body,
  so buffering is mandatory. The FNV-64a + 1MB buffer is correct and optimal.
- **Internal `compress/` subpackage:** evaluated and rejected — compression
  files depend on root symbols (`Middleware`, `responseWrapper`, `ErrCode*`),
  so extracting them creates a circular import. The flat layout is structural.
- **Built-in brotli/zstd encoders:** kept as `WriterFactory` plugin examples to
  preserve the zero-extra-dependency policy. Adding a compression codec as a
  core dependency would break depguard.
- **Functional options (`With*`) pattern:** the struct-config + `Validate()`
  pattern is established and consistent. Introducing functional options would
  create two parallel configuration styles.
