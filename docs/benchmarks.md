# Benchmark Baseline

**Measured:** 2026-11-14 (v1.9.0 cut; all three modules) · **Method:** `go test -run='^$' -bench . -benchtime=3s -count=5` (or `nix run .#bench`; GOWORK=off, flake-pinned Go 1.27.3 linux/amd64, 32 threads) · **Statistical note:** `-count=5` gives a distribution per benchmark; compare against it with `benchstat` (pinned in the flake: `nix run .#benchstat old.txt new.txt`), not single runs.

**Load note (read before comparing):** the machine shares load with the fleet, and window state swings ns/op 1.5–3x on CPU-bound rows while `B/op`/`allocs/op` stay load-stable — compare allocations first, wall-clock second. This re-measure ran the three modules sequentially (~01:20–02:04 CET): the root suite drew a loaded window (most CPU rows 1.5–2.5x the 2026-10-25 numbers), while `httpspec` and `server_timing` ran after the load dropped and came in at or below the previous table — the suite is one protocol, not one window, so rows are not mutually comparable in ns/op. The previous † markers (Timeout, SecurityHeaders never catching a quiet window) are retired this round: both landed near their old quiet-window values (855.0 / 351.3 ns) despite the window, so per-row marks would lie about which rows were load-affected. ‡ keeps marking the flate rows, whose byte output is Go-version- and codec-state-dependent. A raw benchstat-format baseline (`docs/benchmarks.bench.txt`, full root-module 3s×5 plus the sniff micro-bench) is committed; CI posts a `benchstat` comparison against it into the job summary (informational — CI runners are too noisy to gate on).

**Sniff provenance (issue #46, pinned):** v1.9.0 fills a missing `Content-Type` on compressed responses by sniffing the buffered plaintext (at most its first 512 bytes) before the header commits. `BenchmarkCompressionContentTypeSniff` isolates it: the `contentTypeAbsent` sub-benchmark (sniff fires) differs from `contentTypeSet` by exactly ONE header-map insert — 13 allocs/op both ways, because in the CT-set variant the handler performs that same insert; `B/op` differs by ~70–130 (the sniffed value's map slot + string), and the ns delta is inside window noise (≤2%). `BenchmarkCompression` itself moved 12 → 13 allocs / 2,149 → 2,278 B vs the 2026-10-25 table: that is this insert (its harness handler sets no Content-Type), not middleware overhead. The toolchain is exonerated by A/B: v1.8.0 code benched under the same Go 1.27.3 in a scratch worktree still reports 12 allocs/op.

**Runtime-patch deltas (environmental, not code):** the flake Go moved 1.27.1 → 1.27.3 between the last two tables. Two allocation-stable rows shifted bytes without allocation-count changes on code untouched since v1.7.0: the health handler trio gained ~+1 B/op (1,042/1,043; `LiveHandler` flat) and the flate decompression rows moved ±~50 B — stdlib patch class, treat like the ‡ codec state, not as library movement.

**Superseded baselines:** the 2026-08-29, 2026-09-10, 2026-09-11, 2026-10-09/10, and 2026-10-25 tables are historical. Key notes on the 2026-10-09 numbers vs 2026-09-11 (Go 1.26.7): Go 1.27 changed `compress/flate`'s encoded output, so `BenchmarkCompression` produces different bytes now — compression re-measured 2.2x faster (7,874 → 3,527 ns), while `Decompression/deflate` decodes the new output ~24% slower and `Decompression/gzip` ~19% faster; these three rows measure a different codec output than the 1.26-era table did. Small-allocation-heavy paths got broadly faster under the 1.27 runtime (handler trio 668–853 → 579–786 ns; `Recovery` 97 → 66 ns; `Timeout` 728 → 456 ns). `BenchmarkServerTiming_MiddlewareDisabledPassthrough` dropped 273 → 105 ns — that is the `flushHeader()` merge (implicit-WriteHeader prologue dedup), not the runtime. `BenchmarkCSRFMiddleware_UnsafeMethodAttestationCheck` is new since the 2026-09-11 table (the origin-attestation defense); it originally logged one WARN per op, making runs intrinsically noisy — the benchmark now swaps `slog` to a discard handler for its run (restored via `b.Cleanup`), so current numbers are clean.

| Benchmark                                                                | ns/op  | B/op   | allocs/op |
| ------------------------------------------------------------------------ | ------ | ------ | --------- |
| BenchmarkGenerateTimeOrderedID                                           | 240.7  | 41     | 1         |
| BenchmarkGenerateTimeOrderedIDParallel                                   | 178.3  | 41     | 1         |
| BenchmarkIDGeneratorRefillSwap (≈29.0 ns/ID amortized, load window)      | 7,432  | 2,304  | 1         |
| BenchmarkIDGeneratorRefillRawRandRead (baseline; ≈21.2 ns/ID)            | 5,436  | 0      | 0         |
| BenchmarkCSRFMiddleware_PlainHTTPNosurf                                  | 3,071  | 2,336  | 23        |
| BenchmarkCSRFMiddleware_UnsafeMethodAttestationCheck                     | 3,623  | 2,644  | 33        |
| BenchmarkKeyedRateLimiterConfigValidate                                  | 1.861  | 0      | 0         |
| BenchmarkServerConfigValidateWithTLS                                     | 2.680  | 0      | 0         |
| BenchmarkKeyedRateLimiterMiddleware                                      | 261.2  | 208    | 4         |
| BenchmarkKeyedRateLimiter_MaxKeysChurn (fresh key per op)                | 533.4  |        |           |
| BenchmarkChain                                                           | 5,479  |        |           |
| BenchmarkClientIP                                                        | 94.44  | 32     | 1         |
| BenchmarkCodeRejectionConstruction                                       | 240.0  |        |           |
| BenchmarkSentinelCloneWithContext                                        | 487.8  |        |           |
| BenchmarkWrapTransientWithCause                                          | 259.5  |        |           |
| BenchmarkDomainOf                                                        | 36.37  |        |           |
| BenchmarkInDomain                                                        | 42.27  |        |           |
| BenchmarkCompression ‡                                                   | 14,235 | 2,278  | 13        |
| BenchmarkCompressionContentTypeSniff/contentTypeSet (no sniff)           | 4,549  | 2,159  | 13        |
| BenchmarkCompressionContentTypeSniff/contentTypeAbsent (sniff fires)     | 4,591  | 2,228  | 13        |
| BenchmarkCompressionNegotiator/singleToken                               | 15.46  | 0      | 0         |
| BenchmarkCompressionNegotiator/browserMulti                              | 224.0  | 0      | 0         |
| BenchmarkCompressionNegotiator/qvalues                                   | 96.67  | 0      | 0         |
| BenchmarkCompressionNegotiator/emptyHeader                               | 2.430  | 0      | 0         |
| BenchmarkCORS                                                            | 602.4  | 592    | 9         |
| BenchmarkMaxBodySize (4 KiB body, full read)                             | 3,836  | 5,427  | 15        |
| BenchmarkDecompression/gzip ‡                                            | 14,499 | 45,853 | 14        |
| BenchmarkDecompression/deflate ‡                                         | 12,372 | 45,150 | 13        |
| BenchmarkDecompression/passthrough                                       | 197.4  | 224    | 5         |
| BenchmarkHealthHandler                                                   | 2,496  | 1,043  | 12        |
| BenchmarkLiveHandler                                                     | 3,107  | 1,042  | 12        |
| BenchmarkReadyHandler                                                    | 2,263  | 1,043  | 12        |
| BenchmarkMetricsMiddleware                                               | 512.5  | 240    | 5         |
| BenchmarkMetricsMiddlewareWithBody                                       | 881.7  | 304    | 6         |
| BenchmarkMetricsMiddlewareWithCustomPath                                 | 1,203  | 240    | 5         |
| BenchmarkLogging                                                         | 1,476  |        |           |
| BenchmarkNonce                                                           | 732.0  |        |           |
| BenchmarkGenerateNonce                                                   | 132.1  |        |           |
| BenchmarkNonceAttr                                                       | 90.17  |        |           |
| BenchmarkParseUintQuery                                                  | 263.3  | 432    | 4         |
| BenchmarkResponseRecorder                                                | 733.0  | 1,008  | 9         |
| BenchmarkHTTPRequestConstruction/httptestNewRequest                      | 2,037  | 5,104  | 9         |
| BenchmarkHTTPRequestConstruction/httptestNewRequestWithContext           | 1,704  | 5,105  | 9         |
| BenchmarkHTTPRequestConstruction/httpNewRequestWithContext               | 220.8  | 512    | 3         |
| BenchmarkRecovery                                                        | 80.88  |        |           |
| BenchmarkRequestID                                                       | 650.6  |        |           |
| BenchmarkSecurityHeaders                                                 | 351.3  |        |           |
| BenchmarkTimeout                                                         | 855.0  |        |           |
| BenchmarkCompose/Construct (3 no-op middlewares, one-time)               | 12.06  | 0      | 0         |
| BenchmarkCompose/Serve (same stack, steady state)                        | 689.5  | 208    | 4         |
| BenchmarkMiddlewareStack_Middleware (3 entries, per apply)               | 24.98  | 0      | 0         |
| BenchmarkLanguage (path-miss + header hit, de/en) *                      | 572.6  | 584    | 8         |
| BenchmarkLanguageHeaderOnly *                                            | 693.9  | 584    | 8         |
| BenchmarkLanguageStaticRequest (bare-header writer, no recorder) *       | 824.5  | 584    | 8         |
| BenchmarkParseAcceptLanguageCandidates *                                 | 194.2  | 176    | 3         |
| \* Language rows: the 584 B/op is dominated by the query/URL parse on    |        |        |           |
| the reused harness request — allocation counts are the trustworthy       |        |        |           |
| signal, ns/op the noisy one. `BenchmarkLanguageStaticRequest` (added     |        |        |           |
| 2026-10-15, T18) replaces the recorder with a bare-header writer over    |        |        |           |
| `io.Discard`: identical 584 B/op and 8 allocs/op, slightly lower ns/op — |        |        |           |
| proving the httptest baseline was already allocation-clean in steady     |        |        |           |
| state and the recorder delta is wall-clock bookkeeping only.             |        |        |           |

### `httpspec` (3s×5, own module; re-measured 2026-11-14 after the load dropped — at or below every previous table; `B/op`/allocs are the trustworthy signal)

| Benchmark                            | ns/op  | B/op  | allocs/op |
| ------------------------------------ | ------ | ----- | --------- |
| BenchmarkCheckServesRequest          | 574.2  | 1,048 | 11        |
| BenchmarkCheck/index_not_404         | 1,234  |       |           |
| BenchmarkCheck/body_has_content_type | 888.0  |       |           |
| BenchmarkCheck/expect_status         | 858.8  |       |           |
| BenchmarkCheck/unknown_path_404      | 1,574  |       |           |
| BenchmarkCheck/no_leaked_internals   | 1,582  |       |           |
| BenchmarkCheck/long_url_handled      | 46,675 |       |           |

### `server_timing` (3s×5, own module; re-measured 2026-11-14 after the load dropped)

| Benchmark                                           | ns/op | B/op | allocs/op |
| --------------------------------------------------- | ----- | ---- | --------- |
| BenchmarkServerTiming_DisabledOverhead              | 4.439 |      |           |
| BenchmarkServerTiming_EnabledMeasure                | 153.8 |      |           |
| BenchmarkServerTiming_EnabledMeasureViaContext      | 163.6 |      |           |
| BenchmarkServerTiming_Record                        | 95.90 |      |           |
| BenchmarkServerTiming_HeaderValue                   | 419.6 |      |           |
| BenchmarkServerTiming_MiddlewareDisabledPassthrough | 113.6 |      |           |

## Reading this baseline

- **Overhead-sensitive paths** (middleware wrappers): single-digit microseconds at most; anything regressing by >2x on these is a bug, not noise.
- **Allocation counts** (`allocs/op`) are the stablest signal across machines and load — track those first when touching hot paths; this table's ns/op values are best-pass estimates on a shared machine (see the load note above).
- ID generation amortizes `crypto/rand` through generation buffers (256 IDs per refill): `GenerateTimeOrderedID` shows ~1 alloc/op; the refill cost is quantified directly by the two `BenchmarkIDGeneratorRefill*` rows. Even their amortization DELTA is window-sensitive (swap − syscall ≈ 1.6 ns/ID in the 2026-10-25 quiet window, ≈ 7.8 ns/ID in this loaded one) — the load-stable facts are the allocation rows (2,304 B / 1 alloc for the swap vs 0 / 0 for the raw syscall).
- Regenerate after material changes to a hot path; keep this file's method line in sync with the actual invocation. `nix run .#bench` runs exactly the documented protocol (root package; `httpspec` and `server_timing` are benched from their own module directories).
