# Design Note — `RunSerial` and Spec State Sharing

**Date:** 2026-08-30
**Status:** DECIDED — keep per-spec isolation; no shared spec-execution state. Documented so the decision does not get re-litigated every audit.

## Problem

`httpspec.RunSerial` runs each spec sequentially, but specs do not share any request/response state: if two specs both need to inspect the same response (e.g., the CORS specs), each makes its own request (`07-45:f47`). Is this a design flaw worth optimizing away?

## Options considered

| Option                                                   | Shape                                                                        | Verdict                                                                                                                                                                                                                     |
| -------------------------------------------------------- | ---------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A. Shared response cache                                 | specs declare "reads response #1"; runner caches requests per (method, path) | Reject: changes `Check` from a pure function of the handler into a function of runner state; specs become order- and cache-dependent; breaks `Check(handler) Result` composability that lets users run single specs ad hoc. |
| B. Context object threaded through checks                | `Check func(ctx, handler) Result` with typed getters                         | Reject: signature break for every existing check and consumer; buys a micro-optimization (one extra request per spec) that no user has ever measured as a cost.                                                             |
| C. Keep isolation: each spec is a self-contained request | status quo                                                                   | **Selected.**                                                                                                                                                                                                               |

## Why isolation wins

- **Correctness model:** each `Check` is a pure function of the handler. No spec can pass because another spec's side effect warmed state, and no spec can fail because it ran after a state-mutating sibling. This is what makes `Run`, `RunSerial`, and ad-hoc `spec.Check(handler)` calls equivalent.
- **The cost being optimized is negligible.** A "request" is one in-memory `ServeHTTP` against the caller's handler plus an `httptest.ResponseRecorder` — hundreds of nanoseconds. The CORS suite costs one extra request per check; five checks, five requests. Handler-side mutable state (the thing `RunSerial` exists for) is served strictly sequentially, so per-spec requests cannot interfere.
- **`RunSerial`'s job is ordering, not sharing.** Its contract is "run the same checks without `t.Parallel()`" — for handlers whose state cannot take concurrent requests. Adding shared state would conflate execution order with data flow.
- **Spec independence is the extension surface.** `WithExtraSpecs` composes user specs with the standard suite precisely because specs are hermetic. A shared-state runner would need transactional semantics (which spec resets which state?) that no opt-in API should impose.

## Consequences

- Specs that need multi-request flows (e.g., rate-limit probes) implement the loop inside their own `Check` (see `firstTooManyRequests` in `cors_ratelimit_specs.go`).
- Handlers with per-request side effects (counters, token buckets) see one request per spec, not one per run. Document this in handler test setups.
- If a future spec class genuinely needs a shared response (none exists today), the right shape is a new check builder that makes N requests itself and returns one `Result` — not runner-level state sharing.

## Evidence

- Spec contract: `Check func(handler http.Handler) Result` (`httpspec/httpspec.go`).
- Runner: `runSpecs` builds specs once, executes each against the handler independently (`httpspec/httpspec.go`).
- Origin of the question: `docs/status/2026-08-05_07-45_todo-list-execution-sweep.md` finding 47.
