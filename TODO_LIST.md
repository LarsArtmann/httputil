# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code. Completed work lives in [CHANGELOG.md](CHANGELOG.md); rejected ideas live in [ROADMAP.md](ROADMAP.md) Non-goals; process decisions live in [docs/DECISION_LOG.md](docs/DECISION_LOG.md).

_Updated: 2026-08-30 (docs-health pass after the full-code-review execution session)._

---

## High Priority

- [ ] **v1.0 release decision → cut v1.0** — one stabilization cycle, then per [docs/migrating-to-keyed-rate-limiter.md](docs/migrating-to-keyed-rate-limiter.md) remove deprecated `TokenBucketLimiter`/`RateLimit()` (plan T18) and confirm the rate-limiter admission contract per [docs/planning/2026-08-29_21-30_rate-limiter-ctx-cancellation-design-note.md](docs/planning/2026-08-29_21-30_rate-limiter-ctx-cancellation-design-note.md). Sources: `23-08:f7`, ROADMAP "v1.0", `11-30:f4–f7`.
- [ ] **Verify the CSRF `Sec-Fetch-Site` trust model against nosurf before v1.0** — surfaced by full-code-review 2026-08-30: `SetPlaintextHTTPOrigin` forges `Sec-Fetch-Site: same-origin` for trusted origins, and the broader question is how much nosurf trusts client-supplied `Sec-Fetch-Site` generally. Read justinas/nosurf's same-origin implementation (verify at the source; do not encode assumptions) and document the trust boundary in `csrf.go` + README. If a client-supplied header alone can bypass origin checks on plain HTTP, consider rejecting client-sent `Sec-Fetch-Site` at the middleware boundary — and whether an httpspec "no injection header reflection" spec is warranted (`11-30:f2`, `f3`, `f37`).
- [ ] **Decide `CompressionConfig.Level = 0` semantics** — the constructor remaps `0` to `DefaultCompression` while `Validate()` accepts `0` as a valid flate level; align Validate + constructor + docs so the zero value has one documented meaning (surfaced by full-code-review 2026-08-30; `11-30:f6`).

## Medium Priority

- [ ] **Extract response compression into `go-compression`** — full Pareto plan: [docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md](docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md). Trigger: go-datastar needs SSE-safe compression. Decompression stays in httputil.
- [ ] **CI release workflow** — automated tag → build → GitHub Release pipeline (including `go vet` for both modules if CI lands). Sources: `23-33:f24`, `00-51:f24`, `11-30:f33`, `f41`.
- [ ] **go-error-family upstream: conditional-request classification** — propose classification guidance upstream. Sources: `23-33:f40`, `22-43:f48`, `05-45:f47`. (Run the `verify-before-filing` skill before opening the upstream issue.)
- [ ] **`architecture-review` re-run** — last full pass predates ETag extraction + adapter + keyed limiter. Sources: `05-45:f38`, `06-50:f38`, `11-30:f31`.
- [ ] **Convert the remaining legacy table-driven tests** (surfaced by full-code-review 2026-08-30) — `clientip_test.go` TestClientIP, `queryparam_test.go` TestParseUintQuery, `httpspec/cors_ratelimit_specs_test.go` TestVaryContainsToken + TestValidateNonNegativeInt, `httpspec/httpspec_test.go` TestHasVersionLeakDetectsVersionPattern, `server_timing/server_timing_test.go` TestFormatMillis + TestServerTiming_NameSanitization. Either split into standalone `TestX_Case` funcs or amend the AGENTS.md convention to allow property-style subtests; decide once, apply consistently (`11-30:f16`).
- [ ] **Refresh `docs/benchmarks.md` rows for benches changed 2026-08-30** — full-code-review moved recorders inside the loop (health/metrics/recorder/httpspec-check benches), added `b.ReportAllocs` (compression/CORS/ClientIP), and pointed the keyed-limiter eviction bench at the true slow path. Re-measure with the documented 3s×5 protocol and update the affected rows. While there: audit remaining pre-`b.Loop` benchmarks for stale `b.ResetTimer` usage (`11-30:f10`, `f39`).
- [ ] **Finish the T13 line-by-line test review** — `csrf_test.go`, `nonce_test.go`, `security_test.go`, `requestid_test.go`, `id_generator_test.go` got structural checks only (parallel/alloc/sleep audit) in the 2026-08-30 review; read them line by line (`11-30:b1`, `f8`).
- [ ] **Migrate `exhaustruct` → `exhaustruct_v5`** in `.golangci.yml` before the v2 deprecation becomes removal; re-check the golangci-lint upgrade path (config on v2.12.2 semantics) in the same pass (`11-30:f45`, `f46`).
- [ ] **`CSRFConfig.Validate` side-effect cleanup (post-v1.0 candidate)** — Validate mutates `TrustedProxiesCIDR` on a pointer receiver and logs inside Validate; split into a pure `Validate()` + a separate parse step at construction when the API can afford it (surfaced by full-code-review 2026-08-30).

## Low Priority

- [ ] **Chain-level regression for the exact-fill duplication fix** — the unit test covers `compressWriter` directly; add the same 512-byte-exact-fill case through the full `Compression()` middleware (`11-30:f18`).
- [ ] **KeyedRateLimiter property test** — heap/map consistency under churn above `MaxKeys` (pattern: the compression negotiator property test) plus a benchmark with real `MaxKeys`-pressure churn (`11-30:f20`, `f21`).
- [ ] **CORS fuzz invariant for exact-origin allowlists** — echo property beyond `DenyUnmatched` (`11-30:f22`).
- [ ] **Document `compressWriter.Hijack` buffered-bytes-dropped semantics** in the code (`11-30:f23`).
- [ ] **Decide `WrapConflict`/`WrapOrchestration` symmetry** — either add the missing family constructors or document the asymmetry as intentional (`11-30:f24`).
- [ ] **Skip pool `Get` for non-resettable factories** — currently one wasted allocation per request for custom factories without `Reset` (`11-30:f25`).
- [ ] **`Server.Addr()` resolved-port variant** — `":0"` pain bitten twice in tests; decide an API (`11-30:f26`).
- [ ] **`TestChain_RecoveryErrAbortHandler_ThroughStack`** — sentinel through `Chain`, not just `Recovery` alone (`11-30:f44`).
- [ ] **Negotiator wire-format fuzz target** — name/q parsing is property-tested; a fuzz pass over raw `Accept-Encoding` strings would complement it (plus documenting gzip multistream behavior in the round-trip fuzz comment) (`11-30:f40`, `f43`).
- [ ] **Test-helper hygiene trio** — dissolve `bench_batch_test.go` into per-middleware bench files, consolidate `waitForTLS`/`reserveFreePort` with `waitForServerStart`, and switch the TLS test cert from RSA-2048 to Ed25519 (`23-05:f12–f14`).
- [ ] **flake.nix benchmark protocol app** — one-command entry for the documented 3s×5 baseline so `docs/benchmarks.md` refreshes stop being manual (`11-30:f48`).
- [ ] **Slim AGENTS.md below the 30 KB docs-health budget** (currently ~53 KB) — move the architecture file-export table to a `docs/` reference page (AGENTS keeps a pointer) or compress it to one line per file; keep the Hard Constraints, Non-Obvious Behaviors, and Testing Conventions sections intact. Content is current and non-redundant today — this is size-budget hygiene, not staleness (`docs-health` AGENTS quality rubric).
- [ ] **Config-validation hardening batch** — `CORSConfig.AllowedMethods` non-empty, `DecompressionConfig.Encodings` recognized values + duplicate rejection, `CompressionConfig.IncompressibleTypes` prefix validity, `CSRFConfig.MaxAge` negative rejection, `Level` checked when `WriterFactories` set (`08-52:f7–f10`, `f37`, `f38`).
- [ ] **Test-coverage gaps: MaxBodySize bench/fuzz + five missing examples** — `MaxBodySize` is the only middleware with neither benchmark nor fuzz target; `ExampleMetrics`, `ExampleRateLimit` (deprecated, optional), `ExampleHealthHandler`, `ExampleServer`, `ExampleMiddlewareStack` do not exist (`00-23:f8–f9`, `f33–f37`).
- [ ] **Nonce design decisions + composition-test cluster** — decide `NonceConfig.Generator` override and a public `GenerateNonce` (implement or formally decline); add the never-run composition tests (nonce × Compression/CORS/ServerTiming/CSRF-token helpers) that 02-50 f41–f46 and the 08-08 audits keep listing (`20-09:f49`, `02-50:f21/f22`).
- [ ] **ID-generator refill-path benchmark** — quantify the amortized-random buffer refill cost (`20-09:f28`).
- [ ] **dprint availability in the dev environment** — markdown formatter verification has been skipped three sessions running (dprint not on PATH); either add it to the flake devShell or formalize the `--no-verify` pre-commit story (`11-30:f29`, `23-05:f15/f16`).
- [ ] **Nightly fuzz: add a crash issue-template step** — failures currently only show in the run log (`23-05:f45`).
- [ ] **govulncheck for `server_timing` in CI** — currently root-module only (`23-05:f46`).
- [ ] **Commit-lint CI step** — reject commits lacking a conventional prefix (the auto-commit daemon's inferred messages are sometimes generic) (`23-05:f47`).
- [ ] **CI/release tooling extras** — Go-based coverage threshold check (replace the fragile awk in ci.yml) and a pre-release checklist script automating the RELEASE.md gates (`20-09:f46–f47`).
- [ ] **Resolve `go.work` `go 1.26.5` vs CI `1.26.x` vs local `1.26.7` version pinning** (`23-05:f44`).
- [ ] **`TestChain_DecompressionThenMaxBodySize` 417-as-signal assertion** — consider reading the limiter error directly instead of treating 417 as the signal (`23-05:f49`).
- [ ] **Post-v1.0: revisit `KeyExtractor` returning `""`** — exempt vs shared-bucket semantics read differently; confirm docs are unambiguous (`11-30:f49`).
- [ ] **Integration-docs content refresh** — import paths verified current 2026-08-30; the samber/do, HTMX-ideas, Redis, and Prometheus docs could each take a content pass against the current API (`23-05:f37–f40`).
- [ ] **Schedule the next `full-code-review`** — the 2026-08-30 report is a snapshot; re-run before v1.0 if substantial code lands, otherwise post-v1.0 (`11-30:f50`).

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md). Historical open-item evidence lives in the `## Resolution` appendices of `docs/status/2026-*.md`; this list is the live backlog._
