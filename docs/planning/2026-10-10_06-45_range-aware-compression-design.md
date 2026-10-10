# Range-aware compression guard — design note (M14)

Status: EXECUTED under the standing full-execution mandate (2026-10-10).
Evidence base: consumer-audit filing evidence of hand-rolled CORS/Range
interplay bugs (docs/status/2026-10-09_02-29 f.39), the TODO item, and the
frozen-API constraint (behavior addition only, no signature change).

## Problem

`Compression` wrapped every compressible response, including answers to
requests carrying a `Range` header. Compressing the full representation
breaks range semantics: the client's byte offsets refer to the decoded
representation, so a gzipped `206` body (or a gzip stream that ignores the
requested window) cannot be spliced, verified, or resumed — the exact
hand-rolled interplay bug the consumer audit caught in the fleet.

## Rulings

- **D1 behavior** — a request with a `Range` header passes through
  uncompressed (nginx `gzip` behaves the same). The middleware still stamps
  `Vary: Accept-Encoding` (the negotiation input is unchanged), then serves
  the handler's response untouched — whether the handler answers `206` or
  ignores the header and answers `200`.
- **D2 no Vary: Range** — the served representation for a `200` does not
  depend on the `Range` header (the header is only a request hint; ignoring
  it produces the same full-body bytes). `206` responses are already
  cache-keyed by their range, so an extra `Vary` token would be noise.
- **D3 additive/frozen-safe** — no config field, no signature change: the
  guard is unconditional. Consumers who want compressed full responses for
  range-bearing clients (a contradiction in terms) have no knob to preserve;
  nobody in the surveyed fleet compresses `206`s on purpose.
- **D4 tests** — passthrough test with a real byte-exact comparison, a
  control test proving compression still applies without `Range`, a fuzz
  seed with a `Range` header (oracle: passthrough identity), and a 206
  handler-status variant.
