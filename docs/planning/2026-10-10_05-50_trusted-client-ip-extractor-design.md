# `KeyExtractorFromTrustedClientIP` / `TrustedProxySet` — design note (M07)

Status: DESIGN EXECUTED under the standing full-execution mandate (2026-10-10,
"Keep going until everything works"). Evidence base: consumer-audit filings
005/006/007 (six consumers hand-roll trusted-proxy-gated client-IP keying;
the un-gated variant mints attacker-chosen rate-limit buckets — SwettySwipperWeb
vote guard "High"), the CV reference implementation
(`~/projects/CV/platform/httpx/clientip.go`), TODO_LIST "Additive API:
`KeyExtractorFromTrustedClientIP(cidrs)`", and report §f.8
(docs/status/2026-10-10_05-16).

## Problem

`KeyExtractorFromClientIP()` trusts `X-Forwarded-For`/`X-Real-IP` blindly
(documented), which is only safe when an outer proxy strips/overwrites those
headers. Every consumer that runs WITHOUT such a proxy but WITH a local
(appending) proxy hand-rolls the same fix: honor forwarded headers only when
the socket peer is inside a trusted CIDR set, else key on `RemoteAddr`. Six
fleet consumers carry copies of this gate; at least two shipped the un-gated
variant first and got attacker-controlled buckets (filing 006 evidence).

## Rulings (D1–D7, as adopted)

- **D1 home** — `ratelimit_keyed_trusted.go`, domain `ratelimit`, error code `ratelimit.trusted_proxy_cidr_invalid`. The API is named for its primary consumer (rate-limit keying, per the TODO), but the parsed value's `ClientIP`/`Contains` methods serve the non-limiter consumers (vote guards, logging) so they stop re-implementing the gate too.
- **D2 shape** — Parse-validated value type `TrustedProxySet` (immutable `[]*net.IPNet`, exported struct, unexported field): `ParseTrustedProxies(cidrs []string) (TrustedProxySet, error)` returns a classified `Rejection` on an invalid CIDR — fail-LOUD, deliberately NOT the validate-and-log pattern. Rationale: a typo'd CIDR degrades to loopback-only trust, which collapses every client onto the proxy's IP (one shared bucket) — exactly the accidental-single-bucket failure the `KeyExtractor` doc comment warns about. Silent fallback would hide it; a constructor error surfaces it at wiring time. `TrustedProxySet.KeyExtractor() KeyExtractor` slots into `KeyedRateLimiterConfig.KeyExtractor`; `TrustedProxySet.ClientIP(r)` and `TrustedProxySet.Contains(ip)` serve the guard/logging consumers. Convenience wrapper `KeyExtractorFromTrustedClientIP(cidrs []string) (KeyExtractor, error)` for one-liner configs.
- **D3 algorithm** — Rightmost-untrusted walk, NOT CV's leftmost XFF: gate on the socket peer first (headers ignored entirely for untrusted peers); then walk XFF entries right-to-left, skipping entries inside the trust set; the first untrusted entry is the client; all entries trusted → leftmost entry. Kills the forged-leftmost class: a client that sends its own `XFF: 9.9.9.9` through an appending trusted proxy produces `9.9.9.9, <real IP>`, and the walk answers `<real IP>` where leftmost answers the forgery. Deviation from the CV reference recorded here deliberately — CV predates the audit findings that motivated the libraryization.
- **D4 validation** — Every header candidate must `net.ParseIP`. Unparseable entry means the chain is not the documented honest-append shape: distrust the whole header and fall back to the `RemoteAddr` host (abort the walk; do not skip past garbage). The extractor never returns `""` (the `KeyExtractor` contract's shared-bucket failure mode) — the `RemoteAddr` tail mirrors `ClientIP`'s. (Fuzz-verified refinement: a host-extraction miss like `RemoteAddr=":"` falls back to the raw `RemoteAddr`, so the never-empty contract holds even for malformed socket addresses.)
- **D5 empty cidrs** — `nil`/empty `cidrs` substitute the loopback-only default (`127.0.0.0/8`, `::1/128`) — CV parity: local nginx/caddy and `httptest` clients work with zero config, remote trusts require explicit configuration. Empty never means "trust everything" and never means "trust nothing" (the latter would silently equal `KeyExtractorFromRemoteAddr`). The "empty means loopback" zero-value claim is pinned by an execution-probe test (2026-08-30 rule). The zero value of `TrustedProxySet` itself (hand-declared, never parsed) trusts NOTHING — headers ignored — which is the safe direction for an unvalidated value.
- **D6 concurrency** — Immutable value, no atomics. CV needs an atomic global because it is process-wide config; a library value is owned by its config object and captured by the extractor closure. Race-freedom by construction.
- **D7 known limits** — A trusted proxy that APPENDS client-supplied XFF entries (instead of stripping) still lets the attacker choose the keyspace (any routable IP); the walk only removes forgeries LEFT of the honest entries. Documented on the API; `MaxKeys` remains the churn mitigation. No opt-out knob for leftmost semantics — no surveyed consumer wants it; additively evolvable if one appears.

## API surface (v1)

```go
type TrustedProxySet struct{ /* unexported []*net.IPNet */ }

func ParseTrustedProxies(cidrs []string) (TrustedProxySet, error)
func (s TrustedProxySet) Contains(ip net.IP) bool
func (s TrustedProxySet) ClientIP(r *http.Request) string
func (s TrustedProxySet) KeyExtractor() KeyExtractor

func KeyExtractorFromTrustedClientIP(cidrs []string) (KeyExtractor, error)
```

Error surface: `ratelimit.trusted_proxy_cidr_invalid` (Rejection, sentinel
`errTrustedProxyCIDRInvalid`, context key `cidr` carrying the offending
entry). `ParseTrustedProxies(nil)` is the loopback default and returns no
error.

## Tests

Standalone tests per repo convention (no tables): peer-trust gate (trusted/
untrusted), rightmost-untrusted walk (honest chain, forged-leftmost,
multi-hop trusted chain, all-trusted fallback), XRI fallback, unparseable
header abort, `""` never returned, invalid CIDR error (code + context +
`errors.Is`), empty-cidrs loopback execution probe, IPv6 loopback peer, zero
value trusts nothing. Fuzz `FuzzTrustedProxySetClientIP`: never panics, never
returns `""`, result is either the `RemoteAddr`-derived host or a parseable
IP. Race run under `-race -count=10` (the extractor is closure-captured and
shared across parallel requests in the keyed-limiter tests).

## Docs sweep

FEATURES.md inventory, README rate-limiter section, architecture-reference
(export table + error classification), docs/v1-stability.md (additive API),
CHANGELOG [Unreleased], TODO_LIST strike, AGENTS.md non-obvious behaviors
(the "empty means loopback" + rightmost-walk semantics).
