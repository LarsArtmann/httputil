# Design Note — `context.Context` Cancellation for the Keyed Rate Limiter

**Date:** 2026-08-29
**Status:** PROPOSAL — implementation is gated on the v1.0 API-freeze decision (ROADMAP "v1.0 — API stability commitment"). Not implemented pre-v1.0 because it shapes the frozen interface.

## Problem

`KeyedRateLimiterMiddleware` decides allow/reject synchronously and never observes request cancellation. Two consequences:

1. A client that disconnects mid-flight still consumes a token from its bucket (the request already "happened" from the limiter's perspective).
2. Long `Retry-After` waits cannot be cancelled by the caller; the limiter offers no wait primitive at all — it rejects immediately and delegates backpressure to the caller.

Consequence 2 is by design (immediate rejection is the contract; `OnRejected`/`RejectionHandler` own the response). Consequence 1 is a real semantics question: token consumption on aborted requests.

## Options considered

| Option                                                               | Shape                                                                                            | Verdict                                                                                                                                            |
| -------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| A. Cancel-aware `Allow(ctx)` on a new interface                      | `type ContextKeyedRateLimiter interface { Allow(ctx, key) (ok bool, retryAfter time.Duration) }` | Reject: two parallel interfaces for one concept; the split brain the docs-health model warns about.                                                |
| B. Refund tokens on request cancellation                             | wrap `OnAllowed` with a `context.AfterFunc` that refunds                                         | Reject: refunds break the security property (a flood of aborted requests still gets its tokens back immediately — free retry budget for scanners). |
| C. Keep `Allow(key)`; document that tokens are consumed at admission | status quo + doc                                                                                 | Viable fallback.                                                                                                                                   |
| D. Add cancellation only where waiting exists (`Wait(ctx, key)`)     | additive method, never wired into the middleware by default                                      | **Recommended**, post-v1.0 additive evolution path.                                                                                                |

## Recommendation

Ship v1.0 with the current admission-only contract (Option C's documentation), because:

- The middleware's `Allow` decision is instantaneous; by the time cancellation could matter, the downstream handler has already been entered, and unwinding it would require handler-level ctx propagation the limiter cannot own.
- Token-on-admission is the predictable semantics: one request in flight = one token. Aborted-request refunds create timing-dependent budget drift.
- `Wait(ctx, key)` (Option D) is additive — it can be added post-v1.0 without a breaking change, so deciding now buys nothing.

## What must happen before v1.0 instead

- Document "tokens are consumed at admission; request cancellation does not refund" on `KeyedRateLimiter` and in `docs/migrating-to-keyed-rate-limiter.md`.
- Record the `Wait(ctx)` post-v1.0 path in ROADMAP.

## Evidence

- Token consumption at admission: `ratelimit_keyed.go` `Allow`/middleware decision point.
- Rejection contract: `RejectionHandler` default writes 429 + `Retry-After` (`ratelimit_keyed.go`).
- The abort-refund hazard is why go-jsonnet-style refunds were rejected in prior sessions (see `docs/status/2026-07-16_07-30` item 16-era discussion).
