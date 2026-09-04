# Research Note — `httptest.NewRequest` Allocation Cost (the noctx warnings)

**Date:** 2026-08-30
**Source items:** `05-45:f27`, `00-51:f36` ("measure allocation cost behind the noctx warnings")
**Benchmark:** `BenchmarkHTTPRequestConstruction` (`newrequest_bench_test.go`, b.Run groups, `-benchtime=1s -count=3`)

## Why the linter cares

`noctx` flags `http.NewRequest` (no context) in production code. In tests it also flags `httptest.NewRequest`-adjacent patterns historically; the repo suppressed the test-file warnings (`.golangci.yml` exclusions) because the guidance was intuition, not measurement. This note is the measurement.

## Measured cost (Go 1.26.7, linux/amd64, AMD Ryzen AI Max+ 395, 2026-08-30)

| Construction path                | ns/op (3 runs)  | B/op | allocs/op |
| -------------------------------- | --------------- | ---- | --------- |
| `httptest.NewRequest`            | 750 / 768 / 755 | 5104 | 9         |
| `httptest.NewRequestWithContext` | 818 / 762 / 764 | 5104 | 9         |
| `http.NewRequestWithContext`     | 114 / 107 / 109 | 512  | 3         |

`httptest.NewRequest` is ~7x slower and allocates ~10x the bytes of `http.NewRequestWithContext`. The context variant is indistinguishable from the plain one — the cost is entirely in httptest's synthetic wire request: it builds a full `http.ReadRequest` pipeline (TCP-style request text + `bufio` reader + `http.Request` with parsed URL/headers), so per-call it also leaves more garbage for GC regardless of which entry point you use.

## Conclusions

1. **Correctness guidance unchanged:** always `WithContext` in non-test code; in tests any variant is fine.
2. **Perf guidance for benchmarks, not tests:** constructing requests _inside_ a `b.Loop` adds ~750 ns and 9 allocs on top of the thing being measured. Every existing benchmark in this repo hoists request construction out of the loop — keep it that way. A regression here would be visible as inflated `allocs/op` (5 KB per iteration is the loudest signal).
3. **Per-test cost is irrelevant:** even a suite of 20,000 tests constructing one request each pays ~15 ms total. The lint relaxation for test files costs nothing.
4. **The noctx warnings in tests were noise by measurement, not by assumption** — this note is the falsifiable evidence for keeping the suppression.
