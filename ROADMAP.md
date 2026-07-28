# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to TODO_LIST.md.

## Themes

### 1. v1.0 — API honesty and stability commitment

The library is at v0.6.0 with a complete, tested middleware suite. The path to
v1.0 is about resolving the few dishonest names and the build-environment
question, then committing to stability for the public surface.

Raw ideas:

- ~~Rename `RequestIDConfig.ForwardHeader` to `IncomingHeader`~~ — done in v0.7.0.
- ~~Rename `RequestIDConfig.HeaderName` to `ResponseHeader`~~ — done in v0.7.0.
- Evaluate flipping `DenyUnmatched` default to `true` so the CORS allowlist is
  secure by default rather than by opt-in.
- Define which APIs are frozen at v1.0 and document the stability guarantee.

### 2. Extensibility without new dependencies

The dependency policy (stdlib + `go-error-family` + `golang.org/x/time` only) is
a deliberate feature. Extensibility should come through documented examples and
plugin seams, not core dependencies.

Raw ideas:

- Brotli / zstd / lz4 encoder examples via the `WriterFactory` plugin pattern.
- A distributed (Redis-backed) `RateLimiter` implementation as a documented
  example.
- A Prometheus-compatible `MetricsRecorder` implementation as a documented
  example.
- Request body decompression middleware as a counterpart to `Compression`.

### 3. Depth and confidence

The suite is broad; v1.0 should make it deep enough to trust without audit.

Raw ideas:

- Fuzz tests and `Example*` functions for the newer surface (`ParseUintQuery`,
  `ReadyHandlerWithProbe`, `DenyUnmatched`, `EvictionTTL`).
- Close the documented coverage gaps in compression error branches, CORS
  wildcard edge cases, and `ResponseRecorder` hijack failure paths.
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
