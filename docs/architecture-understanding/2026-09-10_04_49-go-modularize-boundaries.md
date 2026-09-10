# Go Module Boundary Assessment (go-modularize run, 2026-09-10 04:49)

**Verdict: HOLD all boundaries.** No go.mod changes, no package moves. All future moves are trigger-gated. Companion diagrams: [2026-09-10_04_49-go-modularize-boundaries.d2](2026-09-10_04_49-go-modularize-boundaries.d2) (current) and [-improved](2026-09-10_04_49-go-modularize-boundaries-improved.d2) (target).

## Phase 1-2 evidence (all verified this run)

| Check | Result |
| --- | --- |
| go.mod inventory | 2 modules (`.` + `./server_timing`) + `go.work` — workspace mode, already split |
| `GOWORK=off go build` per module | both pass (FM#4/FM#12 clean) |
| `scripts/check-module-boundaries.sh` | OK both modules |
| `go work sync` + `go work edit -fmt` | clean (no FM#9 drift) |
| Module DAG | consumers → root → server_timing; no cycles (compiler + module graph) |
| server_timing external deps | **zero** (no `require` block at all) |
| root external deps | 5, each behind a plugin seam or adapter |
| Test-dep leaks in production go.mod | none (FM#3 clean) |
| Co-change via git log | **unreliable here** — the auto-commit daemon batches all changed files per commit; dependency graph + release cadence used instead |

## Phase 1.5 boundary scoring

| Boundary | Cohesion | Coupling | Independent build/version | Depth | Composability payoff | Action |
| --- | --- | --- | --- | --- | --- | --- |
| root module (packages `httputil` + `httpspec`) | 4 — flat by documented decision, file-per-concern | low — 5 ext deps, all isolated | yes | right-for-now (38/50 non-test files) | yes — one import path is the product | **Keep** |
| `server_timing` module | 5 — single concern | zero deps | yes — own go.mod, own v0.12.x tags, own lint config | right | yes — importable standalone | **Keep** |
| `httpspec` as separate module | 5 — single concern | zero deps | would be yes | **too fine today** — no known consumer needs it versioned apart from root | unproven | **Keep as subpackage**; revisit only on consumer demand |

## God-package check (root `httputil`)

Fires the rules of thumb (38 files, 28 exported types, 75 exported funcs, 14 middleware concerns) — but concern clusters are file-separated with near-zero cross-references, sub-packaging compression is structurally impossible (root-symbol dependency), and the flat layout is a confirmed user decision (2026-08-05, re-affirmed 2026-08-30). Revisit triggers unchanged: >50 non-test files, v1.0 + a second consumer asking for `internal/` hygiene, or the go-compression extraction landing.

## Self-review (Phase 4) — challenges considered and declined

- **Error model as its own module:** declined — no consumer imports `Code`/`Domain` without the middleware that throws them; no composability payoff.
- **Server/health as its own module:** declined — same; 2 files.
- **httpspec module split:** declined for now — a module boundary with no consumer demand is overhead, not architecture (direction-neutral rule: a seam must earn its keep).
- **Merging server_timing back into root:** declined — it has independent versioning, zero deps, and its own release cadence; merging would destroy a working seam.

## Trigger-gated future moves (the "improved" diagram's dashed edges)

1. **go-compression extraction** — when go-datastar's SSE-compression need lands (plan: docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md). Re-run this assessment afterward.
2. **`internal/` hygiene step** — only if a second consumer demands it post-v1.0.
3. **httpspec → own module** — only if a consumer needs it versioned independently of root.

## Known-evidence gap

Whether any external consumer imports `httpspec` (or `server_timing`) without the root middleware is unknown from inside the repo — that fact would upgrade or kill the httpspec module-split question definitively.
