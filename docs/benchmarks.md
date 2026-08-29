# Benchmark Baseline

**Measured:** 2026-08-29 · **Method:** `go test -run='^$' -bench . -benchtime=3s -count=5`, Go 1.26.7 linux/amd64 (32 threads) · **Statistical note:** `-count=5` gives a distribution per benchmark; compare against it with `benchstat`, not single runs. Numbers below show one representative run per benchmark (first of five) — full raw data in CI artifacts.

**KeyedRateLimiterMiddleware** (re-measured after a config fix in its harness; burst economics made the original baseline run trip the limiter): ~191 ns/op, 208 B/op, 4 allocs/op.

| BenchmarkGenerateTimeOrderedID | 87.28 ns/op | 32 B/op | 1 allocs/op |

| Benchmark                                   | ns/op    | B/op      | allocs/op |
| ------------------------------------------- | -------- | --------- | --------- |
| BenchmarkGenerateTimeOrderedID              | 87.28    | 32        | 1         |
| BenchmarkGenerateTimeOrderedIDParallel      | 91.94    | 32        | 1         |
| BenchmarkCSRFMiddleware_PlainHTTPNosurf     | 1569     | 2336      | 23        |
| BenchmarkKeyedRateLimiterConfigValidate     | 2.039    | 0         | 0         |
| BenchmarkServerConfigValidateWithTLS        | 2.827    | 0         | 0         |
| BenchmarkChain                              | 3084     |           |           |
| BenchmarkClientIP                           | 53.84    |           |           |
| BenchmarkCodeRejectionConstruction          | 100.4    |           |           |
| BenchmarkSentinelCloneWithContext           | 165.1    |           |           |
| BenchmarkWrapTransientWithCause             | 126.0    |           |           |
| BenchmarkDomainOf                           | 10.50    |           |           |
| BenchmarkInDomain                           | 10.71    |           |           |
| BenchmarkCompression                        | 7738     |           |           |
| BenchmarkCompressionNegotiator/singleToken  | 6.216    | 0         | 0         |
| BenchmarkCompressionNegotiator/browserMulti | 66.91    | 0         | 0         |
| BenchmarkCompressionNegotiator/qvalues      | 44.75    | 0         | 0         |
| BenchmarkCompressionNegotiator/emptyHeader  | 1.584    | 0         | 0         |
| BenchmarkCORS                               | 409.4    |           |           |
| BenchmarkCSRFMiddleware_GET                 | 1510     | 2320      | 22        |
| BenchmarkCSRFMiddleware_POSTWithToken       | 1492     | 2328      | 22        |
| BenchmarkCSRFMiddleware_POSTRejection       | 3607     | 2495      | 29        |
| BenchmarkCSRFMiddleware_PostForm            | 5276     | 9519      | 57        |
| BenchmarkCSRFTokenFromContext               | 21:35:32 | httputil: | Secure    |
| BenchmarkDecompression_Gzip                 | 137.6    | 119055.41 | 160       |
| BenchmarkDecompression_Deflate              | 110.8    | 147845.51 | 160       |
| BenchmarkDecompression_Passthrough          | 92.53    | 270.19    | 160       |
| BenchmarkHealthHandler                      | 199.6    |           |           |
| BenchmarkLiveHandler                        | 207.6    |           |           |
| BenchmarkReadyHandler                       | 214.5    |           |           |
| BenchmarkMetricsMiddleware                  | 74.90    |           |           |
| BenchmarkMetricsMiddlewareWithBody          | 84.46    |           |           |
| BenchmarkMetricsMiddlewareWithCustomPath    | 95.17    |           |           |
| BenchmarkLogging                            | 1322     |           |           |
| BenchmarkNonce                              | 563.8    |           |           |
| BenchmarkGenerateNonce                      | 104.7    |           |           |
| BenchmarkNonceAttr                          | 44.93    |           |           |
| BenchmarkParseUintQuery                     | 207.7    | 432       | 4         |
| BenchmarkKeyedRateLimiter_Allow             | 443.3    | 746       | 7         |
| BenchmarkKeyedRateLimiter_Reject            | 524.8    | 1032      | 10        |
| BenchmarkKeyedRateLimiter_HighCardinality   | 1947     | 5331      | 15        |
| BenchmarkKeyedRateLimiter_EmptyKey          | 105.2    | 208       | 4         |
| BenchmarkKeyedRateLimiter_EvictionOverhead  | 199.1    | 208       | 4         |
| BenchmarkKeyedRateLimiter_ClientIPExtractor | 776.9    | 1066      | 10        |
| BenchmarkTokenBucketLimiter                 | 83.67    | 0         | 0         |
| BenchmarkTokenBucketLimiterWithEviction     | 176.2    | 13        | 1         |
| BenchmarkResponseRecorder                   | 38.56    |           |           |
| BenchmarkRecovery                           | 99.54    |           |           |
| BenchmarkRequestID                          | 605.5    |           |           |
| BenchmarkSecurityHeaders                    | 336.3    |           |           |
| BenchmarkTimeout                            | 568.6    |           |           |

## Reading this baseline

- **Overhead-sensitive paths** (middleware wrappers): single-digit microseconds at most; anything regressing by >2x on these is a bug, not noise.
- **Allocation counts** (`allocs/op`) are the stablest signal across machines — track those first when touching hot paths.
- ID generation amortizes `crypto/rand` via a process-wide buffer (~256 IDs per refill): `GenerateTimeOrderedID` shows ~1 alloc/op; the refill path appears as periodic latency spikes, which the parallel variant smooths out.
- Regenerate after material changes to a hot path; keep this file's method line in sync with the actual invocation.
