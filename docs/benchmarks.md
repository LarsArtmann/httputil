# Benchmark Baseline

**Measured:** 2026-09-11 (all three modules, post-v1.1.0 deprecation removal) · **Method:** `go test -run='^$' -bench . -benchtime=3s -count=5` (or `nix run .#bench`), Go 1.26.7 linux/amd64 (32 threads) · **Statistical note:** `-count=5` gives a distribution per benchmark; compare against it with `benchstat` (now pinned in the flake: `nix run .#benchstat old.txt new.txt`), not single runs. Numbers below show one representative run per benchmark (best of five) — full raw data in CI artifacts (the CI bench step uploads `bench.txt`).

**Superseded baselines:** the 2026-08-29 and 2026-09-10 tables are historical. Key notes on the 2026-09-11 numbers: the `BenchmarkETagAdapterOverhead` rows died with the `httputil.ETag()` adapter (v1.1.0 removal); `BenchmarkKeyedRateLimiterMiddleware` re-measured at 195.6-220 ns/op across isolated and full-suite runs with an unchanged allocation profile (208 B/op, 4 allocs) — the historical ~191 vs 220 delta is machine-state noise, not a regression (verified with benchstat); the ID-generator ring's publication overhead over the raw syscall is ~0.5 ns/ID at this machine's granularity (≈17.8 vs ≈17.4 ns/ID amortized); `BenchmarkCompose/*` and `BenchmarkMiddlewareStack_Middleware` are the new composition benches.

| BenchmarkGenerateTimeOrderedID | 93.07 | 41 | 1 |
| BenchmarkGenerateTimeOrderedIDParallel | 94.19 | 41 | 1 |
| BenchmarkIDGeneratorRefillSwap (≈17.8 ns/ID amortized) | 4,568 |  |  |
| BenchmarkIDGeneratorRefillRawRandRead (baseline; ≈17.4 ns/ID) | 4,448 | 0 | 0 |
| BenchmarkCSRFMiddleware_PlainHTTPNosurf | 1,685 | 2,336 | 23 |
| BenchmarkKeyedRateLimiterConfigValidate | 2.16 | 0 | 0 |
| BenchmarkServerConfigValidateWithTLS | 2.43 | 0 | 0 |
| BenchmarkKeyedRateLimiterMiddleware | 215.80 | 208 | 4 |
| BenchmarkKeyedRateLimiter_MaxKeysChurn (fresh key per op) | 479.20 |  |  |
| BenchmarkChain | 3,353 |  |  |
| BenchmarkClientIP | 48.46 | 32 | 1 |
| BenchmarkCodeRejectionConstruction | 80.85 |  |  |
| BenchmarkSentinelCloneWithContext | 137.20 |  |  |
| BenchmarkWrapTransientWithCause | 110.00 |  |  |
| BenchmarkDomainOf | 16.75 |  |  |
| BenchmarkInDomain | 18.12 |  |  |
| BenchmarkCompression | 7,874 | 1,935 | 12 |
| BenchmarkCompressionNegotiator/singleToken | 6.46 | 0 | 0 |
| BenchmarkCompressionNegotiator/browserMulti | 75.00 | 0 | 0 |
| BenchmarkCompressionNegotiator/qvalues | 51.49 | 0 | 0 |
| BenchmarkCompressionNegotiator/emptyHeader | 1.67 | 0 | 0 |
| BenchmarkCORS | 428.30 | 592 | 9 |
| BenchmarkMaxBodySize (4 KiB body, full read) | 1,668 | 5,427 | 15 |
| BenchmarkDecompression/gzip | 9,903 |  |  |
| BenchmarkDecompression/deflate | 9,556 |  |  |
| BenchmarkDecompression/passthrough | 140.10 |  |  |
| BenchmarkHealthHandler | 667.80 | 1,042 | 12 |
| BenchmarkLiveHandler | 821.70 | 1,042 | 12 |
| BenchmarkReadyHandler | 852.50 | 1,042 | 12 |
| BenchmarkMetricsMiddleware | 177.10 | 240 | 5 |
| BenchmarkMetricsMiddlewareWithBody | 211.10 | 304 | 6 |
| BenchmarkMetricsMiddlewareWithCustomPath | 173.20 | 240 | 5 |
| BenchmarkLogging | 1,030 |  |  |
| BenchmarkNonce | 589.40 |  |  |
| BenchmarkGenerateNonce | 107.90 |  |  |
| BenchmarkNonceAttr | 48.02 |  |  |
| BenchmarkParseUintQuery | 227.00 | 432 | 4 |
| BenchmarkResponseRecorder | 628.60 | 1,008 | 9 |
| BenchmarkHTTPRequestConstruction/httptestNewRequest | 1,649 | 5,105 | 9 |
| BenchmarkHTTPRequestConstruction/httptestNewRequestWithContext | 2,109 | 5,104 | 9 |
| BenchmarkHTTPRequestConstruction/httpNewRequestWithContext | 170.40 | 512 | 3 |
| BenchmarkRecovery | 97.42 |  |  |
| BenchmarkRequestID | 613.50 |  |  |
| BenchmarkSecurityHeaders | 305.80 |  |  |
| BenchmarkTimeout | 727.80 |  |  |
| BenchmarkCompose/Construct (3 no-op middlewares, one-time) | 7.27 | 0 | 0 |
| BenchmarkCompose/Serve (same stack, steady state) | 144.90 | 208 | 4 |
| BenchmarkMiddlewareStack_Middleware (3 entries, per apply) | 6.09 | 0 | 0 |

### `httpspec` (3s×5, own module)

| Benchmark                          | ns/op   | B/op | allocs/op |
| ---------------------------------- | ------- | ---- | --------- |
| BenchmarkCheckServesRequest        | 491.60  | 1048 | 11        |
| BenchmarkCheck/index_not_404       | 769.00  |      |           |
| BenchmarkCheck/body_has_content_type | 618.40 |     |           |
| BenchmarkCheck/expect_status       | 612.70  |      |           |
| BenchmarkCheck/unknown_path_404    | 1,062   |      |           |
| BenchmarkCheck/no_leaked_internals | 1,516   |      |           |
| BenchmarkCheck/long_url_handled    | 34,397  |      |           |

### `server_timing` (3s×5, own module)

| Benchmark                                        | ns/op  | B/op | allocs/op |
| ------------------------------------------------ | ------ | ---- | --------- |
| BenchmarkServerTiming_DisabledOverhead           | 4.32   |      |           |
| BenchmarkServerTiming_EnabledMeasure             | 150.90 |      |           |
| BenchmarkServerTiming_EnabledMeasureViaContext   | 155.60 |      |           |
| BenchmarkServerTiming_Record                     | 107.40 |      |           |
| BenchmarkServerTiming_HeaderValue                | 382.00 |      |           |
| BenchmarkServerTiming_MiddlewareDisabledPassthrough | 272.50 |    |           |

## Reading this baseline

- **Overhead-sensitive paths** (middleware wrappers): single-digit microseconds at most; anything regressing by >2x on these is a bug, not noise.
- **Allocation counts** (`allocs/op`) are the stablest signal across machines — track those first when touching hot paths.
- ID generation amortizes `crypto/rand` through generation buffers (256 IDs per refill): `GenerateTimeOrderedID` shows ~1 alloc/op; the refill cost is quantified directly by the two `BenchmarkIDGeneratorRefill*` rows (the swap publication overhead is ~5 ns/ID over the raw syscall, amortized to ~23.6 vs ~18.6 ns/ID).
- Regenerate after material changes to a hot path; keep this file's method line in sync with the actual invocation. `nix run .#bench` runs exactly the documented protocol.
