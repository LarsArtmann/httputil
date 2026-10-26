# httpspec Docs-Site Page — Scoping Outline (pareto T27)

_Created: 2026-10-15 19:00 CEST · Gate: the website-launch program (owner; TODO row 27/28) · Verdict behind the page: discovery push, not retirement (owner decision recorded in the TODO row — zero public importers despite private fleet consumers)_

## Page goal

Convert a Go developer scanning the README into someone who runs `httpspec.Run(t, handler)` against their own handler within five minutes. The differentiator to sell: a router-agnostic behavioral spec suite does not exist in chi/gorilla/echo/gin/stdlib land — this is the page's one claim, everything else supports it.

## Outline (Astro/Starlight page under the httputil site's guides section)

1. **Hero line** — "Does your handler behave like HTTP? 19 specs say so in one call."
2. **The 30-second demo** — the `httpspec.Run(t, handler)` snippet verbatim from the README section (single source: reuse the README block via the doc-snippet-refs mechanism the site build already supports).
3. **What the specs check** — grouped table (routing, methods, headers, security), generated from the `SpecName` list, not hand-copied (count = the doc.go 19 standard + the opt-in families: LanguageSpecs, PrivateNetworkSpecs, CORS/rate-limit families).
4. **Opt-in families** — one paragraph each: `LanguageSpecs()` (Content-Language/Vary contract), the private-network family, CORS + rate-limit specs; link the integration guides that pair with them (x-text, go-i18n).
5. **Parallel-subtest semantics** — why specs run as parallel subtests with human-readable names (the failure-output experience, one real example of a failure line).
6. **`RunSerial` escape hatch** — shared-state handlers; the one caveat, stated once.
7. **Provenance strip** — "part of github.com/larsartmann/httputil, MIT, Go 1.27+; import path `github.com/larsartmann/httputil/httpspec`".

## Assets needed

| Asset                  | Source                                                                          | Work                                                                         |
| ---------------------- | ------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| Demo snippet           | README "Behavioral Spec Suite" section (existing, gate-checked)                 | Reuse verbatim                                                               |
| Spec table             | `httpspec` `SpecName` list via a small generator or the existing arch-ref table | Generate, don't hand-copy (drift rule)                                       |
| Failure-output example | Run one spec deliberately against a broken handler, capture output              | 10-minute capture, deterministic                                             |
| Hero/diagram           | `2026-09-10_04_49-request-flow.svg` is README-scoped, NOT httpspec-shaped       | Optional new d2 (spec-runner flow) — only if the eyeball pass wants a visual |
| Cross-links            | README section, FEATURES row, arch-ref httpspec table                           | Bidirectional links at build time                                            |

## Out of scope for this page

Benchmark claims (the httpspec bench table lives in docs/benchmarks.md — informational only), the consumer list (zero public importers; private repos aren't nameable), and any comparison table vs. other libraries (the huma analysis stays a research note, not sales copy).

## Build-order dependencies

1. The website-launch effort itself (Firebase/Astro skeleton) — program gate.
2. This page is a natural FIRST content page: it needs no new rendering work beyond the snippet mechanism, so it can double as the site's content-pipeline smoke test.
3. If the spec table generator gets built, feed the same output back into the arch-ref table to kill that hand-copied surface too.
