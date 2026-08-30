# Package Structure Analysis — Refresh (2026-08-30)

**Supersedes:** the numbers in [2026-08-05_06-56_package-structure-analysis.html](2026-08-05_06-56_package-structure-analysis.html); the decision itself is unchanged. This refresh exists because the 08-05 analysis predates the go-etag extraction and the `server_timing` sub-module split (both landed 2026-08-07), so its counts and dependency picture were stale.

## Current shape (verified 2026-08-30)

| Unit                                        | Non-test files | External deps                                                        | Notes                                                              |
| ------------------------------------------- | -------------- | -------------------------------------------------------------------- | ------------------------------------------------------------------ |
| Root `httputil` package (flat)              | **36**         | go-error-family, golang.org/x/time, justinas/nosurf, go-etag         | All middleware shares the `Middleware` type; one import path       |
| `httpspec` subpackage                       | 4              | none (stdlib only)                                                   | Behavior-spec runner; leaf package                                 |
| `server_timing` sub-module                  | 3              | none (stdlib only)                                                   | Separate go.mod, wired via go.work + replace                       |
| go-etag (external repo)                     | —              | —                                                                    | Extracted 2026-08-07; consumed via the thin `etag.go` adapter      |

## Why the flat root still holds

The 08-05 decision (user-confirmed) stands on the same evidence, now with stronger support:

- **Extraction pressure went DOWN, not up.** The two biggest extraction candidates since the analysis — ETag (hash/conditional logic) and Server-Timing (instrumentation) — were both extracted into their own modules. The root shed complexity while only gaining `etag.go` (one adapter file) back.
- **36 of the ~50-file trigger.** Growth since 08-05 is +2 files (`headers.go` shared-consts extraction, `etag.go` adapter) minus none. At this rate the ~50-file revisit trigger is several months of feature work away.
- **No new cycles appeared.** `httpspec` and `server_timing` remain leaves; compression still depends on root symbols (`Middleware`, `responseWrapper`, error codes), so a `compression` sub-package is still structurally impossible without an `internal/` step.

## Decision

Keep the flat root. Revisit triggers (unchanged from 08-05, re-affirmed):

1. Root exceeds ~50 non-test files, **or**
2. v1.0 ships and a second consumer asks for `internal/` hygiene, **or**
3. go-compression extraction lands (per [docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md](../planning/2026-08-16_08-03_extract-compression-into-go-compression.md)) — re-run this analysis afterward, since that removes the dependency that blocks compression sub-packaging.
