# LINT-013: Twin repositories — byte-identical httputil integrations drifting independently

- **Date:** 2026-10-08
- **Severity:** Low (process), rising with each divergence
- **Status:** Open
- **Consumers:** e-invoicing ↔ ksef-sandbox, crm ↔ crm-exec-stage, cqrs-htmx ↔ ci-siblings
- **Ground truth:** verified 2026-10-08 — `diff` of `e-invoicing/internal/http/middleware.go` vs `ksef-sandbox/internal/http/middleware.go` is empty; `crm` and `crm-exec-stage` carry the same `identity.go` setup block; ci-siblings is a full cqrs-htmx mirror pinned to httputil v1.2.0 (see LINT-011)

## The twins

| Pair | Shared surface | Divergence risk already realized |
|---|---|---|
| e-invoicing / ksef-sandbox | `internal/http` middleware + server wiring (CORS wildcard, hand-rolled Recovery — LINT-002/012) | any fix must land twice; the CORS finding applies to both today |
| crm / crm-exec-stage | `internal/identity/identity.go` setup (CSRF `Secure: true` wiring — the *correct* pattern) | the good pattern equally must be maintained twice |
| cqrs-htmx / ci-siblings | the whole framework + usermgmt limiters | ci-siblings frozen at httputil v1.2.0 with `KeyExtractorFromRemoteAddr`; cqrs-htmx moved on |

## Why it is wrong

The fleet's security posture is duplicated rather than shared: a CORS
fix, a limiter cap, or a CSRF default must be applied N times, and the
review burden multiplies (this audit found identical findings in both
copies repeatedly). The ci-siblings case shows the end state: a mirror
that has already diverged in version and in extractor choice, so CI
"green" against it validates middleware semantics from September,
not current ones.

## Fix

- Pick a canonical home per shared surface and make the twin consume
  it (module dependency, template, or vendored sync script). For
  ksef/e-invoicing and crm/crm-exec-stage the natural canonical home
  is a small shared internal module or the cqrs-htmx setup layer they
  already sit on.
- For ci-siblings: either auto-refresh from cqrs-htmx main on each
  tag or archive it (LINT-011 recommendation).
- Until consolidation happens: any filing in this directory that cites
  one twin must be checked and fixed in both — the index (000) lists
  the pairs.
