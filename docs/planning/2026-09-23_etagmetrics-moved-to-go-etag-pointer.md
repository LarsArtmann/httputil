# Pointer — `etagmetrics` moved to go-etag as `metrics`

_2026-09-23. This is a routing note, not a plan: the full design + execution
record lives in the go-etag repo._

The `etagmetrics` sub-module (added in this repo's v1.3.0, 2026-09-22) was
moved to go-etag as [`github.com/larsartmann/go-etag/metrics`](https://github.com/larsartmann/go-etag/tree/main/metrics)
on 2026-09-22/23. The adapter counts go-etag's own observability hooks
(`OnETagGenerated`, `On304`, `OnBufferOverflow`), so it belongs next to them.

- **Design + execution plan (authoritative):**
  [go-etag `docs/planning/2026-09-22_23-25_move-etagmetrics-into-go-etag-metrics.md`](https://github.com/LarsArtmann/go-etag/blob/main/docs/planning/2026-09-22_23-25_move-etagmetrics-into-go-etag-metrics.md)
- **This repo's side of the move** (removal, correction-of-record for the
  v1.3.0 `HitRatio()` double-count, migration path): CHANGELOG `[Unreleased]`.
- **Incident appendix** (the auto-commit daemon restore-war during the
  concurrent migration): `docs/status/2026-09-23_00-04_superb-examples-session-status.md`
  and `docs/status/2026-09-23_00-04_etagmetrics-move-to-go-etag-status.md`;
  the durable lesson lives in AGENTS.md "Auto-Git-Commit Daemon" →
  Parallel-writer protocol.

Consumers migrate by swapping the import:
`github.com/larsartmann/httputil/etagmetrics` →
`github.com/larsartmann/go-etag/metrics` (`Attach`/`Counters`/`Snapshot`
unchanged; `HitRatio` is corrected there).
