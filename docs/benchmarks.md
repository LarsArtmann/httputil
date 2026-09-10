# Benchmark Baseline

**Measured:** 2026-09-10 (main table, post-sweep code) · **Method:** `go test -run='^$' -bench . -benchtime=3s -count=5` (or `nix run .#bench`), Go 1.26.7 linux/amd64 (32 threads) · **Statistical note:** `-count=5` gives a distribution per benchmark; compare against it with `benchstat`, not single runs. Numbers below show one representative run per benchmark (best of five) — full raw data in CI artifacts.

**Superseded baselines:** the 2026-08-29 table (and the 2026-08-30 additions run at 1s) are historical. Key deltas since then, all measured under the full 3s×5 protocol: health handlers now include the trailing-newline body write and the recorder-in-loop harness (~600-730 ns/op, was ~200); the ID generator's generation-swapped ring adds ~20 ns/op over the old shared-buffer design (immutability for the data race, see DECISION_LOG); the keyed-limiter churn bench now drives the true eviction path (~417 ns/op fresh-key insert at capacity); `BenchmarkDecompression/*` rows are the header-restored harness.

| Benchmark                                                          | ns/op  | B/op | allocs/op |
| ------------------------------------------------------------------ | ------ | ---- | --------- |
| BenchmarkGenerateTimeOrderedID                                     | 108.80 | 41   | 1         |
| BenchmarkGenerateTimeOrderedIDParallel                             | 114.10 | 41   | 1         |
| BenchmarkIDGeneratorRefillSwap (per refill; ≈23.6 ns/ID amortized) | 6036   | 2304 | 1         |
| BenchmarkIDGeneratorRefillRawRandRead (baseline; ≈18.6 ns/ID)      | 4762   | 0    | 0         |
| BenchmarkCSRFMiddleware_PlainHTTPNosurf                            | 1659   | 2336 | 23        |
| BenchmarkKeyedRateLimiterConfigValidate                            | 2.22   | 0    | 0         |
| BenchmarkServerConfigValidateWithTLS                               | 2.13   | 0    | 0         |
| BenchmarkKeyedRateLimiterMiddleware                                | 220.10 | 208  | 4         |
| BenchmarkKeyedRateLimiter_MaxKeysChurn (fresh key per op)          | 417.20 |      |           |
| BenchmarkChain                                                     | 4078   |      |           |
| BenchmarkClientIP                                                  | 58.09  | 32   | 1         |
| BenchmarkCodeRejectionConstruction                                 | 100.80 |      |           |
| BenchmarkSentinelCloneWithContext                                  | 150.70 |      |           |
| BenchmarkWrapTransientWithCause                                    | 120.80 |      |           |
| BenchmarkDomainOf                                                  | 19.05  |      |           |
| BenchmarkInDomain                                                  | 12.59  |      |           |
| BenchmarkCompression                                               | 8486   | 2046 | 12        |
| BenchmarkCompressionNegotiator/singleToken                         | 9.53   | 0    | 0         |
| BenchmarkCompressionNegotiator/browserMulti                        | 97.17  | 0    | 0         |
| BenchmarkCompressionNegotiator/qvalues                             | 51.44  | 0    | 0         |
| BenchmarkCompressionNegotiator/emptyHeader                         | 1.67   | 0    | 0         |
| BenchmarkCORS                                                      | 455.40 | 592  | 9         |
| BenchmarkMaxBodySize (4 KiB body, full read)                       | 4132   | 5422 | 15        |
| BenchmarkDecompression/gzip                                        | 14602  |      |           |
| BenchmarkDecompression/deflate                                     | 13562  |      |           |
| BenchmarkDecompression/passthrough                                 | 258.70 |      |           |
| BenchmarkHealthHandler                                             | 731.30 | 1042 | 12        |
| BenchmarkLiveHandler                                               | 586.50 | 1042 | 12        |
| BenchmarkReadyHandler                                              | 610.20 | 1042 | 12        |
| BenchmarkMetricsMiddleware                                         | 190.20 | 240  | 5         |
| BenchmarkMetricsMiddlewareWithBody                                 | 519.40 | 304  | 6         |
| BenchmarkMetricsMiddlewareWithCustomPath                           | 356.60 | 240  | 5         |
| BenchmarkLogging                                                   | 1612   |      |           |
| BenchmarkNonce                                                     | 964.90 |      |           |
| BenchmarkGenerateNonce                                             | 112.80 |      |           |
| BenchmarkNonceAttr                                                 | 48.38  |      |           |
| BenchmarkParseUintQuery                                            | 219.20 | 432  | 4         |
| BenchmarkTokenBucketLimiter _(deprecated)_                         | 82.90  | 0    | 0         |
| BenchmarkTokenBucketLimiterWithEviction _(deprecated)_             | 135.70 | 13   | 1         |
| BenchmarkResponseRecorder                                          | 537.00 | 1008 | 9         |
| BenchmarkETagAdapterOverhead/baselineNoMiddleware                  | 159.80 | 304  | 6         |
| BenchmarkETagAdapterOverhead/directEtagNew                         | 617.10 | 1240 | 14        |
| BenchmarkETagAdapterOverhead/httputilAdapter                       | 719.40 | 1240 | 14        |
| BenchmarkHTTPRequestConstruction/httptestNewRequest                | 1502   | 5101 | 9         |
| BenchmarkHTTPRequestConstruction/httptestNewRequestWithContext     | 1531   | 5105 | 9         |
| BenchmarkHTTPRequestConstruction/httpNewRequestWithContext         | 210.00 | 512  | 3         |
| BenchmarkRecovery                                                  | 62.43  |      |           |
| BenchmarkRequestID                                                 | 451.70 |      |           |
| BenchmarkSecurityHeaders                                           | 295.30 |      |           |
| BenchmarkTimeout                                                   | 478.60 |      |           |

`server_timing` and `httpspec` benchmark baselines are measured separately by their own modules; re-measure them with the same protocol from their directories when their code changes.

## Reading this baseline

- **Overhead-sensitive paths** (middleware wrappers): single-digit microseconds at most; anything regressing by >2x on these is a bug, not noise.
- **Allocation counts** (`allocs/op`) are the stablest signal across machines — track those first when touching hot paths.
- ID generation amortizes `crypto/rand` through generation buffers (256 IDs per refill): `GenerateTimeOrderedID` shows ~1 alloc/op; the refill cost is quantified directly by the two `BenchmarkIDGeneratorRefill*` rows (the swap publication overhead is ~5 ns/ID over the raw syscall, amortized to ~23.6 vs ~18.6 ns/ID).
- Regenerate after material changes to a hot path; keep this file's method line in sync with the actual invocation. `nix run .#bench` runs exactly the documented protocol.
