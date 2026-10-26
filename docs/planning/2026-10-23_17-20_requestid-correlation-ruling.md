# RequestID ↔ Correlation ID ↔ Trace ID — Ruling Note (decision pending)

- **Date:** 2026-10-23 17:20 CEST
- **Status:** PROPOSED — owner decision required before any code
- **Drives:** `RequestID` dual-header correlation (plan item B02), the parked dnsblockd item in [ROADMAP.md](../../ROADMAP.md), and the OTEL guidance in [docs/integrations/otel.md](../integrations/otel.md)
- **Non-goal:** a third correlation mechanism. One ruling, one convention.

## Problem

Three identifiers describe the same request at different layers:

| Identifier         | Where it comes from                                                          | What it answers                                                              |
| ------------------ | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `RequestID`        | httputil generates it (time-ordered 16-byte ID) or echoes an incoming header | Which request is this in _our_ logs?                                         |
| `trace_id`         | W3C `traceparent` (OpenTelemetry)                                            | Which distributed trace is this part of?                                     |
| `X-Correlation-ID` | dnsblockd's consumer convention                                              | Which business operation is this, across services that do not share a trace? |

Today they are disconnected: `RequestID` emits/propagates `X-Request-ID` only, `trace_id` lives in the span context, and the fleet's baggage convention is `correlation_id` (go-appkit/otel `CorrelationIDKey`). A consumer (dnsblockd) needs `X-Correlation-ID` echoed with a fallback to the generated request ID, and currently plans a hand-rolled 15-line wrapper.

## Options

### A. Echo + fallback (recommended)

Extend `RequestIDConfig` with an optional correlation header:

- If the request carries `X-Correlation-ID`, echo it on the response and reuse it (verbatim) as the generated request ID when no `X-Request-ID` is present.
- Otherwise emit the generated ID in both `X-Request-ID` and `X-Correlation-ID`.
- Never invent a second value; the two headers name the same identity.

**Pros:** matches the dnsblockd requirement exactly; no new dependency; one config knob; backward compatible when unused (zero value keeps today's behavior). **Cons:** two headers on the wire; consumers must not treat them as independent.

### B. Seed RequestID from a valid traceparent

When a valid `traceparent` arrives, derive/fill `RequestID` from `trace_id`.

**Pros:** one identifier to log. **Cons:** trace IDs are 32 hex chars with no time ordering (the current generator's amortized-random design is deliberate); a request may belong to a trace but still need its own log correlation; and a public endpoint receiving attacker-controlled `traceparent` would let callers choose our request IDs. Reject unless scoped to trusted infrastructure and ordered-generator semantics are abandoned.

### C. Baggage-only (`correlation_id`)

Carry correlation exclusively in W3C baggage; log it via `appkitotel.CorrelationID(ctx)`.

**Pros:** standards-compliant, already implemented in the SDK lane. **Cons:** baggage requires a tracing/propagation setup; plain-`net/http` consumers and non-OTEL callers get nothing; dnsblockd explicitly needs a response header.

## Recommendation

**Option A**, with C as the tracing-aware complement (not an alternative):

1. `RequestIDConfig.CorrelationHeader string` (empty = disabled). When set, echo and fall back as described.
2. Document that `X-Correlation-ID` and `X-Request-ID` carry the same value, and that `trace_id` is read from span context, never used to seed RequestID.
3. Keep `correlation_id` baggage as the cross-service propagation mechanism; the header is the HTTP-edge representation of the same value.
4. Implementation lands as plan item B02 only after this ruling.

## Owner question

> Adopt Option A (add `RequestIDConfig.CorrelationHeader` echo + fallback, default off) and record "traceparent never seeds RequestID" as policy?

Answering "yes" unblocks B02. Answering "no" archives the item and documents the consumer-wrapper path instead.

## Evidence

- `~/projects/dnsblockd/docs/research/2026-09-15_httputil-adoption-analysis.md` — dual-header requirement and the 15-line local-wrapper fallback.
- `~/projects/go-appkit/otel/baggage.go` — `CorrelationIDKey = "correlation_id"`, `CorrelationID(ctx)`, `WithCorrelationID(ctx, id)`.
- `requestid.go` — current generation/propagation behavior and the `X-Request-ID` contract.
- `docs/integrations/otel.md` — logging guidance that pairs `RequestIDFromRequest` with `trace.SpanContextFromContext`.
