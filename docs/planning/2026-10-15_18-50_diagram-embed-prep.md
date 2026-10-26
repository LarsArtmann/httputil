# Diagram Embedding Prep — Draft Diff for the Owner's Eyeball

_Created: 2026-10-15 18:50 CEST (pareto T24 / TODO row 38) · Gate: the owner's eyeball review of the SVGs — nothing below is applied to README/AGENTS yet · Sources: `docs/status/2026-09-10_06-09` b1/b5/f5 (routed)_

## Candidates (verified renderable 2026-10-15 — d2-generated SVGs with real canvases)

| SVG                                             | Canvas               | Content age                                                     | Fit                                                               |
| ----------------------------------------------- | -------------------- | --------------------------------------------------------------- | ----------------------------------------------------------------- |
| `2026-09-10_04_49-request-flow.svg`             | 1770×1963 (portrait) | 2026-09-10 — predates Language middleware                       | **README, Architecture section** — the request pipeline picture   |
| `2026-09-10_04_49-go-modularize-boundaries.svg` | 2455×1710            | 2026-09-10 — module layout still HOLD-current (T16 disposition) | **README, Architecture section** (second embed) or AGENTS pointer |
| `2026-08-29_internal-coupling.svg`              | 1448×2358            | pre-Language, pre-composition                                   | Skip — superseded shape                                           |
| `2026-08-05/2026-07-05_httputil-current*.svg`   | 5424×2280            | stale (pre-extractions)                                         | Skip                                                              |

Known imperfection to eyeball: both candidates predate the `Language` middleware (2026-10-10), so the request-flow diagram shows one fewer middleware than the shipped chain. Options: embed as-is with a caption note, or re-render the two d2 sources with Language added (a ~15-minute d2 edit — the `.d2` sources sit next to the SVGs).

## Draft README diff (NOT applied)

```diff
 ### Architecture

 Two Go modules in a workspace: the root `httputil` module (flat package
 with middleware + server lifecycle, and the `httputil/httpspec` subpackage
 for reusable HTTP behavior specs) and the `httputil/server_timing`
 sub-module.

+![Request flow through the middleware chain](docs/architecture-understanding/2026-09-10_04_49-request-flow.svg)
+
+*Request flow (2026-09-10): middleware chain → handler → response; see
+`docs/architecture-understanding/` for the d2 sources. Pre-dates the
+`Language` middleware — read the chain as "one middleware per concern".*
+
+![Module boundaries (go-modularize HOLD verdict)](docs/architecture-understanding/2026-09-10_04_49-go-modularize-boundaries.svg)
+
+*Module boundaries: root + `httpspec` + `server_timing`, consumers → root →
+server_timing, no cycles.*
```

## Draft AGENTS touch (NOT applied)

No AGENTS embed — the single-file property argues against binary-ish blobs in the session-context file; one pointer sentence in the Architecture section would do: "Diagrams: docs/architecture-understanding/ (request-flow, modularize-boundaries; d2 sources alongside)."

## Lychee/CI considerations

Relative image links in README resolve in the GitHub renderer (SVGs render inline); lychee will check them as links (they resolve — verified files exist). GitHub renders repo SVGs with a strict sanitizer; d2 output is static markup (no scripts), which passes.

## What the eyeball decides

1. Embed both, one, or neither (caption honesty about the missing Language node).
2. Embed as-is vs 15-minute re-render first (recommended: re-render — stale diagrams in a sales page contradict the docs-freshness discipline).
3. README width handling (the portrait request-flow at 1770px needs `width` capping or the markdown image renders full-width).
