# Benchmark Baseline

**Measured:** 2026-10-09 (all three modules, first re-measure since the Go 1.27 move) · **Method:** `go test -run='^$' -bench . -benchtime=3s -count=5` (or `nix run .#bench`; GOWORK=off, flake-pinned Go 1.27.1 linux/amd64, 32 threads) · **Statistical note:** `-count=5` gives a distribution per benchmark; compare against it with `benchstat` (pinned in the flake: `nix run .#benchstat old.txt new.txt`), not single runs.

**Load note (read before comparing):** the machine shares load with a parallel session; ns/op on allocation-heavy rows inflated 1.5–3x whenever a load spike coincided (verified with in-run detector rows: `ReadyHandler` measured 689 ns in a quiet window and 1,547 ns in a loaded one, same binary). Each benchmark therefore ran up to three 3s×5 passes this day (two full-suite runs plus one targeted re-run of load-suspect rows), and the table reports the best pass value — the load-resistant estimator. Allocation counts (`B/op`, `allocs/op`) are load-stable and can be trusted as-is. Rows marked † never caught a fully quiet window; ‡ marks the flate rows, which are not comparable to the Go 1.26 baseline at all (below).

**Superseded baselines:** the 2026-08-29, 2026-09-10, and 2026-09-11 tables are historical. Key notes on the 2026-10-09 numbers vs 2026-09-11 (Go 1.26.7): Go 1.27 changed `compress/flate`'s encoded output, so `BenchmarkCompression` produces different bytes now — compression re-measured 2.2x faster (7,874 → 3,527 ns), while `Decompression/deflate` decodes the new output ~24% slower and `Decompression/gzip` ~19% faster; these three rows measure a different codec output than the 1.26-era table did. Small-allocation-heavy paths got broadly faster under the 1.27 runtime (handler trio 668–853 → 579–786 ns; `Recovery` 97 → 66 ns; `Timeout` 728 → 456 ns). `BenchmarkServerTiming_MiddlewareDisabledPassthrough` dropped 273 → 105 ns — that is the `flushHeader()` merge (implicit-WriteHeader prologue dedup), not the runtime. `BenchmarkCSRFMiddleware_UnsafeMethodAttestationCheck` is new since the last table (the origin-attestation defense); it logs one WARN per op, which makes its runs intrinsically noisy — silencing that log is a tracked TODO.

| Benchmark                                                      | ns/op  | B/op   | allocs/op |
| -------------------------------------------------------------- | ------ | ------ | --------- |
| BenchmarkGenerateTimeOrderedID                                 | 91.98  | 41     | 1         |
| BenchmarkGenerateTimeOrderedIDParallel                         | 86.01  | 41     | 1         |
| BenchmarkIDGeneratorRefillSwap (≈17.3 ns/ID amortized)         | 4,431  | 2304   | 1         |
| BenchmarkIDGeneratorRefillRawRandRead (baseline; ≈16.1 ns/ID)  | 4,117  | 0      | 0         |
| BenchmarkCSRFMiddleware_PlainHTTPNosurf                        | 1,570  | 2,336  | 23        |
| BenchmarkCSRFMiddleware_UnsafeMethodAttestationCheck           | 4,220  | 2,655  | 33        |
| BenchmarkKeyedRateLimiterConfigValidate                        | 2.089  | 0      | 0         |
| BenchmarkServerConfigValidateWithTLS                           | 2.07   | 0      | 0         |
| BenchmarkKeyedRateLimiterMiddleware                            | 174.8  | 208    | 4         |
| BenchmarkKeyedRateLimiter_MaxKeysChurn (fresh key per op) †    | 620    |        |           |
| BenchmarkChain                                                 | 3,091  |        |           |
| BenchmarkClientIP                                              | 43.57  | 32     | 1         |
| BenchmarkCodeRejectionConstruction                             | 77.55  |        |           |
| BenchmarkSentinelCloneWithContext                              | 112.7  |        |           |
| BenchmarkWrapTransientWithCause                                | 95.32  |        |           |
| BenchmarkDomainOf                                              | 10.76  |        |           |
| BenchmarkInDomain                                              | 10.71  |        |           |
| BenchmarkCompression ‡                                         | 3,527  | 2,167  | 12        |
| BenchmarkCompressionNegotiator/singleToken                     | 7.581  | 0      | 0         |
| BenchmarkCompressionNegotiator/browserMulti †                  | 127.9  | 0      | 0         |
| BenchmarkCompressionNegotiator/qvalues                         | 57.47  | 0      | 0         |
| BenchmarkCompressionNegotiator/emptyHeader                     | 2.002  | 0      | 0         |
| BenchmarkCORS                                                  | 409.2  | 592    | 9         |
| BenchmarkMaxBodySize (4 KiB body, full read)                   | 1,517  | 5,429  | 15        |
| BenchmarkDecompression/gzip ‡                                  | 8,017  | 45,837 | 14        |
| BenchmarkDecompression/deflate ‡                               | 11,867 | 45,134 | 13        |
| BenchmarkDecompression/passthrough                             | 125.5  | 224    | 5         |
| BenchmarkHealthHandler                                         | 579.2  | 1,042  | 12        |
| BenchmarkLiveHandler                                           | 786.4  | 1,042  | 12        |
| BenchmarkReadyHandler                                          | 688.7  | 1,042  | 12        |
| BenchmarkMetricsMiddleware                                     | 171.4  | 240    | 5         |
| BenchmarkMetricsMiddlewareWithBody                             | 164.5  | 304    | 6         |
| BenchmarkMetricsMiddlewareWithCustomPath                       | 147.4  | 240    | 5         |
| BenchmarkLogging                                               | 911.9  |        |           |
| BenchmarkNonce                                                 | 577.4  |        |           |
| BenchmarkGenerateNonce                                         | 114.9  |        |           |
| BenchmarkNonceAttr                                             | 49.15  |        |           |
| BenchmarkParseUintQuery                                        | 188.5  | 432    | 4         |
| BenchmarkResponseRecorder                                      | 519.5  | 1,008  | 9         |
| BenchmarkHTTPRequestConstruction/httptestNewRequest            | 1,649  | 5,104  | 9         |
| BenchmarkHTTPRequestConstruction/httptestNewRequestWithContext | 1,729  | 5,105  | 9         |
| BenchmarkHTTPRequestConstruction/httpNewRequestWithContext     | 197.1  | 512    | 3         |
| BenchmarkRecovery                                              | 65.6   |        |           |
| BenchmarkRequestID                                             | 461.8  |        |           |
| BenchmarkSecurityHeaders                                       | 230.8  |        |           |
| BenchmarkTimeout                                               | 456.1  |        |           |
| BenchmarkCompose/Construct (3 no-op middlewares, one-time)     | 4.936  | 0      | 0         |
| BenchmarkCompose/Serve (same stack, steady state)              | 99.54  | 208    | 4         |
| BenchmarkMiddlewareStack_Middleware (3 entries, per apply)     | 6.415  | 0      | 0         |

### `httpspec` (3s×5, own module; ran in one short window — treat ±25% deltas vs 2026-09-11 as load noise)

| Benchmark                            | ns/op  | B/op | allocs/op |
| ------------------------------------ | ------ | ---- | --------- |
| BenchmarkCheckServesRequest          | 440.1  | 1048 | 11        |
| BenchmarkCheck/index_not_404         | 892.8  |      |           |
| BenchmarkCheck/body_has_content_type | 728.3  |      |           |
| BenchmarkCheck/expect_status         | 648.5  |      |           |
| BenchmarkCheck/unknown_path_404      | 1,304  |      |           |
| BenchmarkCheck/no_leaked_internals   | 1,085  |      |           |
| BenchmarkCheck/long_url_handled      | 35,242 |      |           |

### `server_timing` (3s×5, own module)

| Benchmark                                           | ns/op | B/op | allocs/op |
| --------------------------------------------------- | ----- | ---- | --------- |
| BenchmarkServerTiming_DisabledOverhead              | 3.976 |      |           |
| BenchmarkServerTiming_EnabledMeasure                | 140.2 |      |           |
| BenchmarkServerTiming_EnabledMeasureViaContext      | 140.8 |      |           |
| BenchmarkServerTiming_Record                        | 80.11 |      |           |
| BenchmarkServerTiming_HeaderValue                   | 321   |      |           |
| BenchmarkServerTiming_MiddlewareDisabledPassthrough | 104.9 |      |           |

## Reading this baseline

- **Overhead-sensitive paths** (middleware wrappers): single-digit microseconds at most; anything regressing by >2x on these is a bug, not noise.
- **Allocation counts** (`allocs/op`) are the stablest signal across machines and load — track those first when touching hot paths; this table's ns/op values are best-pass estimates on a shared machine (see the load note above).
- ID generation amortizes `crypto/rand` through generation buffers (256 IDs per refill): `GenerateTimeOrderedID` shows ~1 alloc/op; the refill cost is quantified directly by the two `BenchmarkIDGeneratorRefill*` rows (the swap publication overhead is ~1.2 ns/ID over the raw syscall at this machine's granularity, amortized to ≈17.3 vs ≈16.1 ns/ID).
- Regenerate after material changes to a hot path; keep this file's method line in sync with the actual invocation. `nix run .#bench` runs exactly the documented protocol (root package; `httpspec` and `server_timing` are benched from their own module directories).
