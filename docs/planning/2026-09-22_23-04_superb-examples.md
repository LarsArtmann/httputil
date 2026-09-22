# 2026-09-22 23:04 — Superb Examples

## Goal

Make the package examples superb: every example on pkg.go.dev is
self-contained (copy-pasteable with no invisible dependencies), every
sub-module carries its own examples, and the library's headline features
(CSRF token flow, error-domain routing) are demonstrated.

Audited 2026-09-22: 33 examples exist, all runnable with `// Output:`,
near-total root API coverage — but 6 root examples call unexported test
helpers from `testutil_test.go` that are invisible to pkg.go.dev readers,
`server_timing` and `etagmetrics` have zero examples on their own pages,
and the CSRF token helpers plus the `DomainOf`/`InDomain` error-routing
API have no examples anywhere.

## Pareto Breakdown

| Slice | Delivers | Content |
| ----- | -------- | ------- |
| 1%    | 51%      | Inline the invisible test helpers (`newNoOpHandler`, `newWriteStatusHandler`, `newPanicHandler`) so all 25 root examples are self-contained |
| 4%    | 64%      | + `server_timing` examples on its own pkg.go.dev page |
| 20%   | 80%      | + `etagmetrics` example, + CSRF token-flow examples |
| 100%  | 100%     | + error-routing examples (`DomainOf`/`InDomain`), CHANGELOG + AGENTS.md sync, full verification gates, commit + push |

## Task Plan

### Comprehensive tasks (10-30 min each)

| # | Task | Impact | Effort | Customer value |
| - | ---- | ------ | ------ | -------------- |
| 1 | Root examples self-contained | High | Low | Every example copy-pasteable from pkg.go.dev |
| 2 | `server_timing` example suite | High | Low | Sub-module page teaches its 3 usage paths |
| 3 | `etagmetrics` example | Medium | Low | New module gets a zero-to-counter walkthrough |
| 4 | CSRF token-flow examples | High | Low | The #1 CSRF consumer question answered in godoc |
| 5 | Error-routing examples | Medium | Low | Headline error model demonstrated |
| 6 | Docs sync (CHANGELOG, AGENTS.md) | Low | Low | Convention codified, change recorded |
| 7 | Verification gates (fmt, lint, race) | High | Medium | Zero regressions across 3 modules |
| 8 | Commit + push | Medium | Low | Work lands with a detailed message |

### Micro tasks (≤12 min each)

| # | Task | Verify |
| - | ---- | ------ |
| 1.1 | Inline 6 helper call sites in `example_test.go` | `go test -run Example .` green; no helper refs remain |
| 2.1 | `ExampleNewServerTiming` (deterministic wire format) | module example test |
| 2.2 | `ExampleServerTimingMiddleware` + `ExampleWrapServerTiming` | module example test |
| 3.1 | `ExampleAttach` in `etagmetrics` | module example test |
| 4.1 | `ExampleCSRFTokenFormField` | root example test |
| 4.2 | `ExampleCSRFTokenHXHeaders` | root example test |
| 5.1 | `ExampleDomainOf` + `ExampleInDomain` via `CORSConfig.Validate` | root example test |
| 6.1 | CHANGELOG `[Unreleased]` entry; AGENTS.md example-convention note | link check n/a (no new links) |
| 7.1 | `golangci-lint fmt` in all 3 modules | clean diff after fmt |
| 7.2 | `golangci-lint run` in all 3 modules | 0 issues |
| 7.3 | `go test -race ./...` in all 3 modules | all green |
| 8.1 | Detailed commit, push | `git status` clean |

## Execution Graph

```mermaid
graph TD
    P[Plan doc] --> A[1.1 Inline helpers in root examples]
    A --> AV{go test -run Example}
    AV --> B[2.1-2.2 server_timing examples]
    B --> BV{server_timing tests}
    BV --> C[3.1 etagmetrics ExampleAttach]
    C --> CV{etagmetrics tests}
    CV --> D[4.1-4.2 CSRF token examples]
    D --> E[5.1 DomainOf/InDomain examples]
    E --> F[7.1 golangci-lint fmt x3]
    F --> G{7.2 lint: 0 issues}
    G --> H{7.3 go test -race x3}
    H --> I[6.1 CHANGELOG + AGENTS.md]
    I --> J[8.1 Commit + push]
```

## Design Notes

- **Self-containment over brevity.** The 6 affected root examples inline
  their handler doubles as local closures. The example gets slightly
  longer; the reader gets complete, runnable knowledge. This is the
  documented godoc best practice.
- **Determinism.** Random outputs (tokens, durations, IDs) are asserted
  via deterministic prefixes / boolean probes, never printed raw — the
  existing convention (`ExampleRequestID`, `ExampleNonce`).
- **server_timing wire-format example is fully deterministic**
  (`Record` with an explicit duration), so its `// Output:` shows the
  exact header format — the module's best teaching moment.
- **Error-routing examples drive a real validator** (`CORSConfig.Validate`
  with a negative MaxAge) instead of fabricating errors, showing the
  intended end-to-end path: validate → classify → route by domain.
- **Package placement.** `server_timing/example_test.go` uses
  `package servertiming` (matches existing module tests);
  `etagmetrics/example_test.go` uses `package etagmetrics_test` (matches
  existing module tests).
- **No API changes, no new dependencies.** Test-file-only additions;
  depguard allowlist untouched; sub-modules keep their zero-dep
  (server_timing) / single-dep (etagmetrics) profiles.
