# consumer-audit

Re-runs the httputil consumer lint audit mechanically. The 2026-10-08 audit
(docs/review/lint/000-index.md) was a half-day of manual work; this pipeline
produces its evidence corpus in minutes so each re-audit starts from fresh,
cited data instead of a stale /tmp dump.

## The six steps

| Step | What                                                | Automated |
| ---- | --------------------------------------------------- | --------- |
| 1    | who-uses inventory (direct + indirect consumers)    | yes       |
| 2    | per-consumer httputil go.mod pins                   | yes       |
| 3    | corpus of every `httputil.*` call site              | yes       |
| 4    | pattern-pack greps per filing class                 | yes       |
| 5    | report skeleton with the evidence paths             | yes       |
| 6    | human per-pattern review against documented semantics | manual  |

## Usage

```bash
bash scripts/consumer-audit/run-audit.sh [output-dir]
```

Defaults to `/tmp/httputil-consumer-audit-<date>`. Steps 1–4 write evidence
files there; step 5 copies `report-template.md` next to them. Step 6 is yours:
read each pattern hit against the semantics documented in AGENTS.md
("Non-Obvious Behaviors") and the godoc, then file or clear per consumer.

## Requirements

- `~/projects/project-dependency-graph` (step 1's who-uses command)
- `rg` (ripgrep)
- consumers checked out under `~/projects`
