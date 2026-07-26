# httputil Status Report — Rate Limiter Library Switch

**Session date:** 2026-07-16 07:30 CEST  
**Generated from:** current session run and observed state  
**Report location:** `docs/status/2026-07-16_07-30_rate-limiter-library-switch.md`

---

## Executive Summary

Replaced the hand-rolled token bucket in `ratelimit.go` with `golang.org/x/time/rate`. Tests and lint pass. Two structural problems surfaced during the run: the repo is currently unbuildable without `GOEXPERIMENT=jsonv2`, and `flake.lock` was modified outside this session.

> **Update 2026-07-22 (commit `4ce4fdf`):** the rate-limiter switch was committed and `flake.lock` was committed (`32528ff`). The two build issues (`GOEXPERIMENT=jsonv2` requirement, Go 1.27 API on Go 1.26) are **still unresolved**. Full item-by-item status in [Resolution](#resolution-2026-07-22) below.

| Metric            | Value |
| ----------------- | ----- |
| Files changed     | 5     |
| Lint issues       | 0     |
| Tests / race      | PASS  |
| Partially done    | 1     |
| Totally fucked up | 2     |
| Open questions    | 2     |

---

## a) Fully Done

1. **Switched `TokenBucketLimiter` to `golang.org/x/time/rate`**  
   Each key now owns a `*rate.Limiter`. Idle-bucket eviction and the `now` clock injection for deterministic tests are preserved. The internal bucket type is gone; token-refill math is now the library's responsibility.

2. **Added dependency and updated policy**  
   `go.mod` and `go.sum` now include `golang.org/x/time v0.15.0`. `.golangci.yml` depguard was updated to allow it. `AGENTS.md` documents the new allowed dependency.

3. **API alignment**  
   `NewTokenBucketLimiter(rate float64, burst int)` now matches `rate.NewLimiter(r Limit, b int)`. Tokens are discrete, so `burst` being an `int` is the correct type.

4. **Tests and lint pass**  
   `GOEXPERIMENT=jsonv2 go test ./... -race -count=1` passes.  
   `GOEXPERIMENT=jsonv2 golangci-lint run ./...` reports 0 issues.

---

## b) Partially Done

1. **Test environment is not self-documenting**  
   `AGENTS.md` Commands still lists plain `go test ./...` and `golangci-lint run`, but both now require `GOEXPERIMENT=jsonv2` because `health.go` imports `encoding/json/v2`. A new contributor following the documented commands will hit a build failure immediately.  
   **Fix:** update the Commands block in `AGENTS.md` to prefix every Go command with `GOEXPERIMENT=jsonv2`.

2. **Status report not committed**  
   The status-report skill says to commit the report, but the user explicitly said to wait for instructions after writing it. This report is written but uncommitted.

---

## c) Not Started

- ~~Committing the rate-limiter changes.~~ Done — `4ce4fdf`.
- Updating `CHANGELOG.md` with the breaking API change. _(Still open as of 2026-07-22.)_
- Updating `TODO_LIST.md` if there are open rate-limiter items. _(Still open.)_
- Adding a benchmark comparing the old and new `TokenBucketLimiter`.
- Adding a usage example or updating `README.md` for the new signature. _(README still shows old `float64` burst param.)_
- Adding a distributed / Redis-backed `RateLimiter` example.

---

## d) Totally Fucked Up!

### 1. Build is broken without `GOEXPERIMENT=jsonv2`

**Where:** `health.go` — `encoding/json/v2`  
**Severity:** Critical

Running `go test ./...` without `GOEXPERIMENT=jsonv2` fails with:

```text
imports encoding/json/v2: build constraints exclude all Go files
```

The recent commit `f616f9f` added `GOEXPERIMENT=jsonv2`, but the flake provides Go 1.26.4, which appears to lack the experiment files. This is a hard blocker for any contributor not in the exact nix environment.

**Fix:** either pin Go 1.27+ in `flake.nix` (if jsonv2 is available there) or downgrade `health.go` to `encoding/json` v1.

### 2. `health.go` uses Go 1.27-only API on Go 1.26

**Where:** `health.go:30,61,69` — gopls `stdversion` warnings  
**Severity:** Critical

`json.MarshalWrite` requires Go 1.27 or later, but the module declares `go 1.26.4` and the flake installs `go_1_26`. This is a version mismatch independent of the experiment flag.

### 3. `flake.lock` was modified outside this session

**Where:** `flake.lock`  
**Severity:** Important

`git status` shows `flake.lock` as modified with a nixpkgs bump from `0bb7ec54c8483066ec9d7720e780a5caa71f8612` to `18b9261cb3294b6d2a06d03f96872827b8fe2698`. I did not run any nix command. It may have been pre-existing or updated by the environment, but it is currently uncommitted and unexplained.

---

## e) What We Should Improve!

1. **Make build commands reproducible without magic env vars.**  
   If `GOEXPERIMENT=jsonv2` is required, it should be encoded in the flake devShell or in a `GOEXPERIMENT` wrapper, not rely on users remembering it.

2. **Add deterministic benchmark for the rate limiter.**  
   We now rely on an external library. A benchmark would prove the switch was a net win and guard against regressions in future dependency updates.

3. **Expose richer rate-limit primitives.**  
   `rate.Limiter` supports `AllowN`, `Reserve`, and `Wait`. The current `RateLimiter` interface only allows one token at a time. Consider extending the interface or adding a separate advanced limiter for consumers who need burst > 1 per request.

4. **Document the breaking API change.**  
   `NewTokenBucketLimiter`'s second parameter changed from `float64` to `int`. This needs a CHANGELOG entry and possibly a migration note in the README.

5. **Audit other custom parsers for stdlib-equivalent replacements.**  
   The project is now more open to stdlib-adjacent dependencies. Other middlewares may have custom parsing or state machines that could be replaced by well-tested libraries (e.g., q-value parsing, compression negotiation).

---

## f) Up to 50 Things We Should Get Done Next

Sorted roughly by impact: unblock contributors first, then harden, then extend.

| #   | Task                                                                          | Impact   | Status |
| --- | ----------------------------------------------------------------------------- | -------- | ------ |
| 1   | Resolve Go 1.26 vs 1.27 mismatch in `flake.nix` / `health.go`                 | Critical | Open   |
| 2   | Document or automate `GOEXPERIMENT=jsonv2` requirement                        | Critical | Open   |
| 3   | Investigate and explain the uncommitted `flake.lock` diff                     | High     | Open   |
| 4   | Commit or revert the rate-limiter changes                                     | High     | Open   |
| 5   | Update `CHANGELOG.md` with the breaking signature change                      | High     | Open   |
| 6   | Add `TokenBucketLimiter` benchmark                                            | Medium   | Open   |
| 7   | Add stress/concurrency test for per-key limiter map                           | Medium   | Open   |
| 8   | Add usage example to `README.md` or package examples                          | Medium   | Open   |
| 9   | Evaluate exposing `AllowN` on the `RateLimiter` interface                     | Medium   | Open   |
| 10  | Consider adding `Retry-After` header support to `RateLimit`                   | Medium   | Open   |
| 11  | Add a distributed rate-limiter example (Redis-backed)                         | Medium   | Open   |
| 12  | Review and update `TODO_LIST.md` for rate-limiter items                       | Medium   | Open   |
| 13  | Run the full code-quality scan skill on the current state                     | Medium   | Open   |
| 14  | Run architecture-review skill on middleware boundaries                        | Medium   | Open   |
| 15  | Audit other custom parsers for stdlib-equivalent replacements                 | Low      | Open   |
| 16  | Add test for `Allow` behavior across many keys (memory pressure)              | Low      | Open   |
| 17  | Add negative/float input validation tests                                     | Low      | Open   |
| 18  | Evaluate whether `rate.Inf` should be exposed for unlimited paths             | Low      | Open   |
| 19  | Ensure CI uses the same Go/nixpkgs versions as the flake                      | Medium   | Open   |
| 20  | Re-enable or document any suppressed linters                                  | Low      | Open   |
| 21  | Add test for zero `EvictionTTL` (no eviction) with deterministic time         | Low      | Open   |
| 22  | Verify the `golang.org/x/time` transitive dependency tree                     | Low      | Open   |
| 23  | Update nix apps to set `GOEXPERIMENT=jsonv2` if it remains required           | High     | Open   |
| 24  | Consider vendoring or pinning the exact Go toolchain in the flake             | Low      | Open   |
| 25  | Schedule a naming-review pass for the rate-limiter symbols                    | Low      | Open   |
| 26  | Add property-based tests for token bucket behavior                            | Low      | Open   |
| 27  | Review timeout middleware for clock injectability                             | Low      | Open   |
| 28  | Add example for `RateLimit` middleware with custom `OnDenied`                 | Low      | Open   |
| 29  | Evaluate `golang.org/x/time/rate` alternatives if any concerns arise          | Low      | Open   |
| 30  | Add integration test for rate limiter through full middleware stack           | Low      | Open   |
| 31  | Document per-key memory behavior in `AGENTS.md`                               | Low      | Open   |
| 32  | Add `RateLimit` to `httpspec` standard specs if applicable                    | Low      | Open   |
| 33  | Review `RateLimitConfig` for missing fields (e.g., `ExcludeFunc`)             | Low      | Open   |
| 34  | Consider rate-limit metrics emission                                          | Low      | Open   |
| 35  | Add optional logging when rate limit is exceeded                              | Low      | Open   |
| 36  | Ensure `RateLimit` middleware works correctly with `ResponseRecorder`         | Low      | Open   |
| 37  | Test rate limiter with IPv6 `RemoteAddr` strings                              | Low      | Open   |
| 38  | Add flake check that catches Go version mismatch                              | Low      | Open   |
| 39  | Review recent `flake.lock` diff for unintended nixpkgs changes                | Low      | Open   |
| 40  | Verify `go-error-family` dependency is still up to date                       | Low      | Open   |
| 41  | Add `go vet` to nix test app alongside `go test`                              | Low      | Open   |
| 42  | Document why `GOEXPERIMENT=jsonv2` was introduced                             | Low      | Open   |
| 43  | Add test coverage for `RateLimitConfig.Validate`                              | Low      | Open   |
| 44  | Add test coverage for `DefaultRateLimitConfig`                                | Low      | Open   |
| 45  | Consider adding `SetLimit`/`SetBurst` dynamic methods to `TokenBucketLimiter` | Low      | Open   |
| 46  | Add expiration/cleanup stress test for `EvictionTTL`                          | Low      | Open   |
| 47  | Review `compression` middleware for library replacement opportunities         | Low      | Open   |
| 48  | Review `etag` middleware for library replacement opportunities                | Low      | Open   |
| 49  | Add a `doc.go` example for rate limiting                                      | Low      | Open   |
| 50  | Schedule a full-code-review skill pass                                        | Low      | Open   |

---

## g) Top 2 Questions I Cannot Figure Out Myself

### Question 1: Why is `flake.lock` modified?

`git diff` shows nixpkgs was bumped from `0bb7ec54c8483066ec9d7720e780a5caa71f8612` to `18b9261cb3294b6d2a06d03f96872827b8fe2698`. I did not run `nix flake update` or any nix command. Was this done by another agent, by the environment, or was it pre-existing from before the session snapshot? Should it be committed, reverted, or left alone?

### Question 2: Should we upgrade Go or downgrade `health.go`?

`health.go` uses `json.MarshalWrite` (Go 1.27+), but the flake pins Go 1.26.4. The intended fix is unclear: is the project committed to jsonv2 and should therefore move to Go 1.27+, or is the flake the source of truth and `health.go` should be reverted to `encoding/json` v1?

---

_Report generated from session on 2026-07-16 07:30 CEST. Based on the current working tree and the run of tests/lint performed during this session. Does not include external research._

---

## Resolution (2026-07-22)

| Item in report          | Status   | Detail                                                                                                          |
| ----------------------- | -------- | --------------------------------------------------------------------------------------------------------------- |
| Rate-limiter committed  | **Done** | Committed as `4ce4fdf` ("feat(ratelimit): switch token bucket to golang.org/x/time/rate")                       |
| `flake.lock` diff       | **Done** | Committed as `32528ff` ("chore: update flake.lock with latest nixpkgs and treefmt-nix revisions")               |
| Build needs `jsonv2`    | **Open** | `health.go` still imports `encoding/json/v2`; `go build ./...` fails without `GOEXPERIMENT=jsonv2` on Go 1.26.4 |
| `health.go` Go 1.27 API | **Open** | `json.MarshalWrite` still in use; module still declares `go 1.26.4`                                             |
| CHANGELOG updated       | **Done** | Documented in v0.6.0 CHANGELOG (breaking `burst int` signature change)                                          |
| TODO_LIST updated       | **Open** | Rate-limiter items not yet tracked in `TODO_LIST.md`                                                            |
| `GOEXPERIMENT` in flake | **Open** | `flake.nix` does not set `GOEXPERIMENT=jsonv2` in the devShell; contributors must remember the env var          |

The rate-limiter switch shipped, but the two critical build issues (jsonv2 requirement, Go 1.27 API on Go 1.26) remain unresolved.
