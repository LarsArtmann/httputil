# Go Module Boundary Assessment (go-modularize run, 2026-09-10 04:49)

**Verdict: HOLD all boundaries.** No go.mod changes, no package moves. All future moves are trigger-gated. Companion diagrams: [2026-09-10_04_49-go-modularize-boundaries.d2](2026-09-10_04_49-go-modularize-boundaries.d2) (current) and [-improved](2026-09-10_04_49-go-modularize-boundaries-improved.d2) (target).

## Phase 1-2 evidence (all verified this run)

| Check | Result |
| --- | --- |
| go.mod inventory | 2 modules (`.` + `./server_timing`) + `go.work` — workspace mode, already split |
| Non-test file count | **37** in package `httputil`, 41 module-wide incl. `httpspec` (corrects the 36/38 figures circulating in older docs) |
| Tag audit | dual namespace works: root plain `v*` (latest v0.12.0), submodule `server_timing/v*` (latest v0.12.0); `go.work.sum` absent — harmless, local `replace` pins resolve without it |
| `GOWORK=off go build` per module | both pass (FM#4/FM#12 clean) |
| `scripts/check-module-boundaries.sh` | OK both modules |
| `go work sync` + `go work edit -fmt` | clean (no FM#9 drift) |
| Module DAG | consumers → root → server_timing; no cycles (compiler + module graph) |
| server_timing external deps | **zero** (no `require` block at all) |
| root external deps | 5, each behind a plugin seam or adapter |
| Test-dep leaks in production go.mod | none (FM#3 clean) |
| Co-change via git log | **unreliable here** — the auto-commit daemon batches all changed files per commit; dependency graph + release cadence used instead |
| **External consumer scan** (pkg.go.dev imported-by + Sourcegraph, 2026-09-10) | root: 4 repos, all owner-owned (`emeet-pixyd`, `template-arch-lint`, `go-appkit`, `cqrs-htmx` v2/v3/v4). `server_timing`: imported standalone by `cqrs-htmx` v4 + admin-demo — composability payoff **proven**. `httpspec`: **zero importers anywhere** |
| CI wiring | verified real, not ghost gates: `ci.yml:44` runs `check-module-boundaries.sh`, golangci-lint action + `GOEXPERIMENT=jsonv2` set, `nightly-fuzz.yml` + `release.yml` present |

## Phase 1.5 boundary scoring

| Boundary | Cohesion | Coupling | Independent build/version | Depth | Composability payoff | Action |
| --- | --- | --- | --- | --- | --- | --- |
| root module (packages `httputil` + `httpspec`) | 4 — flat by documented decision, file-per-concern | low — 5 ext deps, all isolated | yes | right-for-now (37/50 non-test files; 4 consumer repos depend on the single import path) | yes — proven by 4 importing repos | **Keep** |
| `server_timing` module | 5 — single concern | zero deps | yes — own go.mod, own v0.12.x tags, own lint config | right | yes — importable standalone | **Keep** |
| `httpspec` as separate module | 5 — single concern | zero deps | would be yes | **dead question for now — zero importers found** | none proven (0 importers) | **Keep as subpackage**; splitting a module nobody imports is overhead, not architecture |

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

## Resolved: the consumer-evidence gap

The 08-30 analysis and earlier versions of this doc listed "no known external consumers" as an open question. Resolved 2026-09-10 via pkg.go.dev imported-by + Sourcegraph:

- **4 repos import the root module** — all owned by LarsArtmann (`emeet-pixyd`, `template-arch-lint`, `go-appkit`, `cqrs-htmx`). No third-party importers are public. The flat root's one-import-path ergonomics now serve 4 codebases — raising the cost of any future split and strengthening the Hold verdict.
- **`server_timing` has a proven standalone consumer** (`cqrs-htmx` v4 + admin-demo) — the module boundary earns its keep by the skill's own litmus test.
- **`httpspec` has zero importers** — built, tested, documented, unused externally. The module-split question is closed (nobody needs it versioned apart); the real gap is **discovery/marketing**, which belongs to the README/docs-site workstream, not to go.mod surgery. The flat-root "second consumer asks for internal/ hygiene" trigger is now *partially armed*: consumer count already exceeds two, so any one of them asking fires it.

## Known-evidence gap (remaining)

pkg.go.dev/Sourcegraph only see public code. Private in-family consumers beyond the 4 found would not appear — the owner knows the full set.
