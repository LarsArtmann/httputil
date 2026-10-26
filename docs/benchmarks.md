# Benchmark Baseline

**Measured:** 2026-10-25 (v1.7.0 cut; all three modules) · **Method:** `go test -run='^$' -bench . -benchtime=3s -count=5` (or `nix run .#bench`; GOWORK=off, flake-pinned Go 1.27.1 linux/amd64, 32 threads) · **Statistical note:** `-count=5` gives a distribution per benchmark; compare against it with `benchstat` (pinned in the flake: `nix run .#benchstat old.txt new.txt`), not single runs.

**Load note (read before comparing):** the machine shares load with the fleet, and window state swings ns/op 1.5–3x on CPU-bound rows while `B/op`/`allocs/op` stay load-stable — compare allocations first, wall-clock second. The 2026-10-25 re-measure (v1.7.0 cut) ran one full-suite 3s×5 pass plus one targeted re-run of load-suspect rows; the table reports the best pass per row. Since v1.6.0 the production diff is `server.go` (listener lifecycle) and `errors.go` (two templates) only — no middleware path changed — so cross-table movement vs the 2026-10-09/10 numbers is environmental, not code. The two † rows (`Timeout`, `SecurityHeaders`) never caught their old quiet window this day (~1.5x vs 2026-10-10 on unchanged code, allocations flat); ‡ keeps marking the flate rows, whose byte output is Go-version-dependent and whose gzip/deflate split inverted this day (now both ≈15 µs) — treat their ns/op as codec+machine state, not code. A raw benchstat-format baseline (`docs/benchmarks.bench.txt`, full root-module 3s×5) is committed; CI posts a `benchstat` comparison against it into the job summary (informational — CI runners are too noisy to gate on).

**Superseded baselines:** the 2026-08-29, 2026-09-10, 2026-09-11, and 2026-10-09/10 tables are historical. Key notes on the 2026-10-09 numbers vs 2026-09-11 (Go 1.26.7): Go 1.27 changed `compress/flate`'s encoded output, so `BenchmarkCompression` produces different bytes now — compression re-measured 2.2x faster (7,874 → 3,527 ns), while `Decompression/deflate` decodes the new output ~24% slower and `Decompression/gzip` ~19% faster; these three rows measure a different codec output than the 1.26-era table did. Small-allocation-heavy paths got broadly faster under the 1.27 runtime (handler trio 668–853 → 579–786 ns; `Recovery` 97 → 66 ns; `Timeout` 728 → 456 ns). `BenchmarkServerTiming_MiddlewareDisabledPassthrough` dropped 273 → 105 ns — that is the `flushHeader()` merge (implicit-WriteHeader prologue dedup), not the runtime. `BenchmarkCSRFMiddleware_UnsafeMethodAttestationCheck` is new since the last table (the origin-attestation defense); it originally logged one WARN per op, making runs intrinsically noisy — the benchmark now swaps `slog` to a discard handler for its run (restored via `b.Cleanup`), so current numbers are clean.

| Benchmark                                                                | ns/op  | B/op   | allocs/op |
| ------------------------------------------------------------------------ | ------ | ------ | --------- |
| BenchmarkGenerateTimeOrderedID                                           | 95.29  | 41     | 1         |
| BenchmarkGenerateTimeOrderedIDParallel                                   | 107.1  | 41     | 1         |
| BenchmarkIDGeneratorRefillSwap (≈18.45 ns/ID amortized)                  | 4,723  | 2,304  | 1         |
| BenchmarkIDGeneratorRefillRawRandRead (baseline; ≈16.86 ns/ID)           | 4,316  | 0      | 0         |
| BenchmarkCSRFMiddleware_PlainHTTPNosurf                                  | 1,388  | 2,336  | 23        |
| BenchmarkCSRFMiddleware_UnsafeMethodAttestationCheck                     | 3,756  | 2,645  | 33        |
| BenchmarkKeyedRateLimiterConfigValidate                                  | 2.43   | 0      | 0         |
| BenchmarkServerConfigValidateWithTLS                                     | 3.24   | 0      | 0         |
| BenchmarkKeyedRateLimiterMiddleware                                      | 184.6  | 208    | 4         |
| BenchmarkKeyedRateLimiter_MaxKeysChurn (fresh key per op)                | 477.8  |        |           |
| BenchmarkChain                                                           | 3,288  |        |           |
| BenchmarkClientIP                                                        | 47.98  | 32     | 1         |
| BenchmarkCodeRejectionConstruction                                       | 76.82  |        |           |
| BenchmarkSentinelCloneWithContext                                        | 107.6  |        |           |
| BenchmarkWrapTransientWithCause                                          | 103.3  |        |           |
| BenchmarkDomainOf                                                        | 10.41  |        |           |
| BenchmarkInDomain                                                        | 10.6   |        |           |
| BenchmarkCompression ‡                                                   | 3,228  | 2,149  | 12        |
| BenchmarkCompressionNegotiator/singleToken                               | 5.74   | 0      | 0         |
| BenchmarkCompressionNegotiator/browserMulti                              | 74.62  | 0      | 0         |
| BenchmarkCompressionNegotiator/qvalues                                   | 50.57  | 0      | 0         |
| BenchmarkCompressionNegotiator/emptyHeader                               | 1.56   | 0      | 0         |
| BenchmarkCORS                                                            | 330.5  | 592    | 9         |
| BenchmarkMaxBodySize (4 KiB body, full read)                             | 1,817  | 5,430  | 15        |
| BenchmarkDecompression/gzip ‡                                            | 15,148 | 45,799 | 14        |
| BenchmarkDecompression/deflate ‡                                         | 15,062 | 45,102 | 13        |
| BenchmarkDecompression/passthrough                                       | 119.7  | 224    | 5         |
| BenchmarkHealthHandler                                                   | 643.3  | 1,042  | 12        |
| BenchmarkLiveHandler                                                     | 817.1  | 1,042  | 12        |
| BenchmarkReadyHandler                                                    | 733.8  | 1,042  | 12        |
| BenchmarkMetricsMiddleware                                               | 162.5  | 240    | 5         |
| BenchmarkMetricsMiddlewareWithBody                                       | 202.1  | 304    | 6         |
| BenchmarkMetricsMiddlewareWithCustomPath                                 | 169.7  | 240    | 5         |
| BenchmarkLogging                                                         | 1,030  |        |           |
| BenchmarkNonce                                                           | 583    |        |           |
| BenchmarkGenerateNonce                                                   | 105.2  |        |           |
| BenchmarkNonceAttr                                                       | 50.38  |        |           |
| BenchmarkParseUintQuery                                                  | 218.7  | 432    | 4         |
| BenchmarkResponseRecorder                                                | 583.8  | 1,008  | 9         |
| BenchmarkHTTPRequestConstruction/httptestNewRequest                      | 1,498  | 5,105  | 9         |
| BenchmarkHTTPRequestConstruction/httptestNewRequestWithContext           | 1,502  | 5,105  | 9         |
| BenchmarkHTTPRequestConstruction/httpNewRequestWithContext               | 167.9  | 512    | 3         |
| BenchmarkRecovery                                                        | 58.4   |        |           |
| BenchmarkRequestID                                                       | 512.9  |        |           |
| BenchmarkSecurityHeaders †                                               | 404.4  |        |           |
| BenchmarkTimeout †                                                       | 735.8  |        |           |
| BenchmarkCompose/Construct (3 no-op middlewares, one-time)               | 4.62   | 0      | 0         |
| BenchmarkCompose/Serve (same stack, steady state)                        | 77.93  | 208    | 4         |
| BenchmarkMiddlewareStack_Middleware (3 entries, per apply)               | 5.94   | 0      | 0         |
| BenchmarkLanguage (path-miss + header hit, de/en) *                      | 515.1  | 584    | 8         |
| BenchmarkLanguageHeaderOnly *                                            | 440.3  | 584    | 8         |
| BenchmarkLanguageStaticRequest (bare-header writer, no recorder) *       | 572.1  | 584    | 8         |
| BenchmarkParseAcceptLanguageCandidates *                                 | 151.9  | 176    | 3         |
| \* Added 2026-10-10 with the `Language()` middleware (one 3s×5 pass;     |        |        |           |
| best-of-5 reported, same estimator as the load note above). The 584 B/op |        |        |           |
| is dominated by the query/URL parse on the reused harness request —      |        |        |           |
| allocation counts are the trustworthy signal, ns/op the noisy one.       |        |        |           |
| `BenchmarkLanguageStaticRequest` (added 2026-10-15, T18) replaces the    |        |        |           |
| recorder with a bare-header writer over `io.Discard`: identical 584      |        |        |           |
| B/op and 8 allocs/op, slightly lower ns/op — proving the httptest        |        |        |           |
| baseline was already allocation-clean in steady state and the recorder   |        |        |           |
| delta is wall-clock bookkeeping only.                                    |        |        |           |

### `httpspec` (3s×5, own module; re-measured 2026-10-25 in a loaded window — see the load note; `B/op`/allocs are the trustworthy signal)

| Benchmark                            | ns/op  | B/op  | allocs/op |
| ------------------------------------ | ------ | ----- | --------- |
| BenchmarkCheckServesRequest          | 1,265  | 1,048 | 11        |
| BenchmarkCheck/index_not_404         | 1,940  |       |           |
| BenchmarkCheck/body_has_content_type | 1,502  |       |           |
| BenchmarkCheck/expect_status         | 1,247  |       |           |
| BenchmarkCheck/unknown_path_404      | 2,835  |       |           |
| BenchmarkCheck/no_leaked_internals   | 2,887  |       |           |
| BenchmarkCheck/long_url_handled      | 69,762 |       |           |

### `server_timing` (3s×5, own module; re-measured 2026-10-25)

| Benchmark                                           | ns/op | B/op | allocs/op |
| --------------------------------------------------- | ----- | ---- | --------- |
| BenchmarkServerTiming_DisabledOverhead              | 6.24  |      |           |
| BenchmarkServerTiming_EnabledMeasure                | 217.7 |      |           |
| BenchmarkServerTiming_EnabledMeasureViaContext      | 179.2 |      |           |
| BenchmarkServerTiming_Record                        | 114.3 |      |           |
| BenchmarkServerTiming_HeaderValue                   | 523   |      |           |
| BenchmarkServerTiming_MiddlewareDisabledPassthrough | 171.4 |      |           |

## Reading this baseline

- **Overhead-sensitive paths** (middleware wrappers): single-digit microseconds at most; anything regressing by >2x on these is a bug, not noise.
- **Allocation counts** (`allocs/op`) are the stablest signal across machines and load — track those first when touching hot paths; this table's ns/op values are best-pass estimates on a shared machine (see the load note above).
- ID generation amortizes `crypto/rand` through generation buffers (256 IDs per refill): `GenerateTimeOrderedID` shows ~1 alloc/op; the refill cost is quantified directly by the two `BenchmarkIDGeneratorRefill*` rows (the swap publication overhead is ~1.2 ns/ID over the raw syscall at this machine's granularity, amortized to ≈17.3 vs ≈16.1 ns/ID).
- Regenerate after material changes to a hot path; keep this file's method line in sync with the actual invocation. `nix run .#bench` runs exactly the documented protocol (root package; `httpspec` and `server_timing` are benched from their own module directories).
