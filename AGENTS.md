# httputil — AGENTS.md

## Hard Constraints (Will Break Your Code)

The non-obvious rules that cause immediate lint failures. Read before writing any code.

- **Allowed dependencies (`depguard`)**: `$gostd`, this module root + subpackages (explicit `github.com/larsartmann/httputil` + `/**` entries — `$module` does not expand in depguard v2.12.2), `…/httputil/server_timing` (explicit — `/**` does not match separate go.mod modules), `github.com/larsartmann/go-error-family`, `github.com/larsartmann/go-etag` (adapter removed v1.1.0; dep stays for its error-code registration), `golang.org/x/time`, `github.com/justinas/nosurf`. Nothing else.
- **`exhaustruct_v5`**: populate EVERY field of every struct literal (relaxed in test files; stdlib `os/exec.Cmd` ignored via config). Settings keys `enforce-patterns`/`ignore-patterns`; struct tags NOT honored — use `//nolint:exhaustruct_v5`.
- **`err113`**: no inline `errors.New()`/`fmt.Errorf()` sentinels. Pattern: `const codeFoo = Code("foo.failure")` + package-level `var errFoo = codeFoo.Rejection("message")`, return `errFoo.WithContext(...)`/`WithCause(...)` clones (With* copies — safe).
- **`wsl_v5`**: blank lines before `return`, after declarations, around control flow — run `golangci-lint fmt` after editing; manual whitespace will likely be wrong.
- **`nonamedreturns`**: no named returns in signatures.
- **`noctx`**: `http.NewRequest` banned — use `http.NewRequestWithContext`; `httptest.NewRequest` in tests excluded via `.golangci.yml`.
- **`godot`**: all comments end with a period.
- **`mnd`**: no magic numbers — named constants (`defaultMaxAge`, `defaultCompressionMinSize`).
- **`gosec` G705 excluded globally** (response writes are this library's purpose — structural false positive). Do NOT re-add per-site `//nolint:gosec` on response writes: fragile under `nolintlint` (gosec taint analysis is non-deterministic across cache states).
- **`paralleltest`**: every test calls `t.Parallel()` as its first line.
- **`noinlineerr`**: no `if err := foo(); err != nil` — separate assignment, then check.
- **`canonicalheader`**: canonical MIME header keys (`X-Api-Key`, not `X-API-Key`). `Get/Set/Add/Del` canonicalize silently; direct map access (`w.Header()["X-API-Key"] = …`) bypasses canonicalization and can split one header into two entries — never mix map-style with method-style access. All repo literals verified canonical (2026-08-29); zero nolint directives.
- **`testableexamples`**: every `Example*` has a `// Output:` directive.
- **`thelper`**: test helpers calling `t.Fatal`/`t.Error` start with `t.Helper()`.
- **No `Must*` APIs, never-panic API surface (owner directives 2026-09-10)**: panics are rejected as an API design tool; wiring/construction errors return errors (`MiddlewareStack.Add` is the pattern; ROADMAP Non-goal). `MiddlewareFunc.Then(nil)` wires a 500-stub; the dead `crypto/rand.Read` guards were deleted (documented never to error). Remaining panic sites are deliberate: `recovery.go` re-panics `http.ErrAbortHandler` (net/http contract), `compress_pool.go` factory-contract violations (inside `Recovery`'s envelope), `httpspec.go`'s unexported `mustRequest`.

## Commands

```bash
# /mnt/buildcache is unwritable in some environments — export these or every Go
# toolchain call fails with "failed to initialize build cache":
export GOCACHE=$HOME/.cache/go-build-httputil GOLANGCI_LINT_CACHE=$HOME/.cache/golangci-lint-httputil
# go.work requires >= 1.27.1 (go-etag v0.5.0); the local default toolchain is
# older, so without this the first go command of every fresh session fails:
export GOTOOLCHAIN=auto

go test ./...              # Run tests
go test -race ./...        # Race detection (REQUIRED for tests with t.Parallel() or shared state)
go test -race -count=N ./... # Surface timing-dependent races — repeat N times
go vet ./...               # Vet
go test -bench=. ./...     # Benchmarks
golangci-lint run          # Lint (~70 linters, 0 issues) — full-package runs only: linting a file subset typechecks incompletely and reports phantom issues
golangci-lint run --fix    # Auto-fix what's possible
golangci-lint fmt          # Format (gofumpt + golines@120 + gci)

nix fmt                    # treefmt (Go via golangci-lint formatters; run before the auto-commit daemon sees the tree)
nix flake check            # Full flake gates (includes treefmt verification)

# erraudit (aligned with go-error-family policy; NEVER --enforce-samber-oops).
# CANONICAL HOME for erraudit policy and residual verdicts — the BuildFlow
# residuals bullet and the honest-silence paragraph point here; update this
# block, not copies.
# Real gates (exit 0 required): legacy_as + stdlib_constructor below.
# --type-aware is ADVISORY-ONLY: ~45 `sentinel_concrete_type`
# (measured 2026-10-10, erraudit 1c6809a: 45 in root, server_timing clean; the
# load-bearing `*errorfamily.Error` sentinels — do NOT migrate;
# declaring `var errX error` breaks WithContext/WithCause call sites) and
# ~44 test-side `errors.Is` matches (all correct) — verdict history:
# docs/status/2026-09-11_10-03 and _13-49 (also the `_ =` honest-silence
# discards and the scripts/ exemptions).
# Upstream tool bugs filed 2026-10-10: LarsArtmann/erraudit#10 (the
# rewrite hint does not compile for the 29 cloned sentinels), #11 (the
# legacy_is guard misses unexported err* sentinels — cause of the ~44).
# GOEXPERIMENT=jsonv2 prefix dropped 2026-10-09: json/v2 is standard in Go 1.27.
erraudit lint ./... --type-aware --enforce-go-error-family
erraudit lint ./... --type legacy_as
erraudit lint ./... --type stdlib_constructor --enforce-go-error-family

# server_timing sub-module (run from server_timing/)
cd server_timing && go test -race ./... && golangci-lint run

# Documented 3s×5 benchmark protocol (see docs/benchmarks.md; the flake app
# covers ONLY the root module — run httpspec/ and server_timing/ in-module)
nix run .#bench

# Clone-detection baseline (see Accepted Code Duplication below): -t 2 shows
# 0 groups; -t 1 shows the two accepted pairs — anything NEW is a finding.
art-dupl --sort total-tokens -t 1 --type-aware .
art-dupl --sort total-tokens -t 2 --type-aware .

# Release: scripts/prerelease-check.sh automates the gates; runbook is
# docs/RELEASE.md. CHANGELOG [version] sections freeze at the tag.
```

**Go 1.27 notes (changelog review 2026-10-09):** json/v2 is standard (the erraudit flag prefix was dropped); vet's new printf `%w`-pointer check is clean here; the gofmt alignment change caused no drift; compress/flate output changed (round-trip fuzz invariants are robust to it); net/http gained `Server.MaxHeaderValueCount` (NOT reachable through the `Server` wrapper — `ServerConfig` exposes no raw-server override; documented as a pass-through boundary in README).

**`go test -count=1` does NOT detect data races.** Only `go test -race` catches shared-state access between goroutines. After writing or modifying ANY test with `t.Parallel()`, shared fixtures, or closures over mutable state, run `go test -race -count=10 ./...` (a 2026-08-05 race passed `count=1` clean but failed 60% of `-race` runs — `cors_ratelimit_specs_test.go:138`).

`golangci-lint run` is the authoritative quality gate — it's configured with ~70 linters (see `.golangci.yml`). `go vet` alone is insufficient.

### Auto-Git-Commit Daemon

An auto-git-commit daemon commits continuously; unexpected commits are expected, and inferred messages may be generic. For deliberate commits, run `git commit` explicitly with `--no-verify` when the pre-commit hook is unavailable (e.g., `dprint` missing). Consequence: git-log co-change analysis is unreliable here — the daemon batches unrelated files into single commits, so use dependency graphs, not "changed together" evidence.

**Parallel-writer protocol (2026-09-23 restore-war lesson):** parallel agent sessions run in this fleet; "vanished files" can be another writer's deliberate removal. Before restoring anything, `git log --format='%h %s' -10` and scan for NON-heuristic messages touching that path (reasoned message = deliberate act; `chore: auto-commit N file(s)` = the daemon). Deletions get an immediate deliberate commit (the daemon resurrects deleted paths); version-directive downgrades contradicting a dependency's `go` requirement are daemon casualties — verify against the tag and restore. Never cite daemon heuristic commits in external artifacts — reference CHANGELOG sections, tags, or deliberate commits.

### Session-Tail Discipline (recurring lesson: 2026-10-09 ×2)

Before declaring a session done: confirm the last commit is pushed (`git log origin/master..master` empty), CI green on that exact head, and the tree clean. The tail is where daemon/parallel-writer races bite (the 2026-10-09 unpushed-annotation and unwatched-tail-commit incidents). Never pipe release-critical commands through filters that eat errors or exit codes — redirect to a file and read `$?` from the command, not the pipe.

### Doc-Freshness Cadence

Living docs (`TODO_LIST.md`, `FEATURES.md`, `ROADMAP.md`, `CHANGELOG.md`) are verified via the `docs-health` skill before each version tag and at least monthly. Historical `docs/status/` reports get inline `~~item~~ done at <hash>` annotations when read; fully-resolved reports move to `<dir>/archived/` via `git mv` (`a)` FULLY DONE tables, `d)`/`e)` sections, and session timelines are historical records and are never struck). The archived-completeness gate (`grep -rLn '~~'` must print nothing) applies to `.md` files only — two archived `.html` reports are rendered twins of annotated `.md` siblings in the same directory and need no separate annotation. Files referenced by living docs stay in place. Struck-done TODO_LIST items are deleted at each docs-health rebuild (completed work lives in CHANGELOG), not accumulated.

### BuildFlow Pipeline

`buildflow` (source: `~/projects/BuildFlow`) orchestrates the quality pipeline. `dev` is a build MODE, not a command: full run is `buildflow --build-mode dev`; `buildflow --dry-run` previews (including `skip_steps`); `buildflow -s <step>` runs ONE step — multiple `-s` flags do NOT accumulate (last one wins). Detect-only tools (markdown-lint, lychee, branching-flow, erraudit, go-structure-linter, gomod-check) never fail the run unless `--fail-on-findings`. A full `--build-mode dev` run is expected to exit non-zero on the findings gate (the documented policy-rejected residuals below); treat only NEW finding classes or failed fixable steps as breakage.

- **`.buildflow.yml`** skips `go-auto-upgrade`: its suggestions require `samber/lo`, which the depguard allowlist bans (see Allowed Dependencies).
- **`.markdownlint.json`** disables MD001/009/010/012/013/022/024/026/028/029/031/034/036/037/040/051 — each disabled rule's findings were located and judged individually. Tool quirks: markdownlint-cli JSON output only appears with stderr merged (`-j '**/*.md' 2>&1`); lychee needs `--exclude-path vendor`.
- **Stale-binary trap:** the global `~/.local/bin/buildflow` is a plain copy, not a symlink — after landing BuildFlow changes, `cp ~/projects/BuildFlow/result/bin/buildflow ~/.local/bin/buildflow` and confirm `buildflow --version` matches HEAD (`buildflow doctor` flags staleness as `env/binary-freshness`; `nix run .#reinstall` only refreshes `./result`).
- **No `vendor/` directory (removed 2026-09-11, owner decision):** the module cache is canonical for workspace AND `GOWORK=off` builds (both verified); `.gitignore` still lists `vendor/`; full runs auto-skip the vendor steps, and `buildflow -s go-mod-vendor` exits 69 ("no tools matched") — expected, don't. Vendoring-reintroduction pitfalls (modules.txt `## explicit; go <ver>` markers vs the gomod-check parser, `go work vendor` omitting replace-markers) are recorded in docs/status/2026-09-11_13-49.
- **Result-cache gotcha:** deleting an untracked directory does NOT invalidate buildflow's result cache (keys cover tracked files only; findings replay for the 7-day TTL). Purge surgically: `nix shell nixpkgs#sqlite -c sqlite3 ~/.cache/buildflow/buildflow.db "DELETE FROM result_cache WHERE value LIKE '%<finding text>%';"`.
- **Ghost-module enumeration (2026-09-23):** after deleting a module directory, the golangci step can still enumerate and LINT it (stale workspace snapshot; `BUILDFLOW_NO_RESULT_CACHE=1` doesn't help) — confirm the directory is really gone (`ls`), then re-run; a lint failure naming a nonexistent path means the tree, not the cache, is the problem.
- **Sandboxed treefmt vs toolchain download (2026-09-23):** `nix flake check`'s treefmt gate fails with `go: download go1.27.1: ... connection refused` after a go-directive bump — nixpkgs goimports shells out to `go` and the sandbox has no network. Fixed in `flake.nix` by wrapping goimports with `GOTOOLCHAIN=local` + `go_1_27` and a TMPDIR GOCACHE fallback; the same pattern applies to ANY go toolchain call in a sandboxed check after a directive bump.
- **Snapshot before `--fix`:** `git status --short > /tmp/pre-buildflow.txt` before any `buildflow … --fix` step — with a foreign writer in the same window, attribution becomes guesswork (2026-09-23 misattribution incident).
- **Residual detect-only findings are policy-rejected, not debt:** branching-flow (bool-bitflags, single-implementer interfaces, httpspec panic — contradict documented decisions), erraudit `_ =` "honest silence" discards (canonical verdict: the Commands erraudit block above), go-structure-linter (flat root package deliberate; its "compiled binary in git" claim is false), flake-meta-checker (project flake ships without a nixpkgs-style `meta` on purpose), jscpd (the root/`server_timing` `.golangci.yml` pair is two per-module configs that must each be complete — YAML anchors cannot cross files). These are the expected composition of the dev-mode findings gate's error count; anything outside these classes is NEW and needs triage. The tool's remaining analysis sections are verified noise (2026-10-09 direct `branching-flow all` run): PHANTOM_TYPE transpose rows mirror stdlib signatures (`httptest.NewRequest(method, target)`, `ListenAndServeTLS(certFile, keyFile)`, Server-Timing name/description vocabulary — accepted design, already part of the 2026-09-14 findings-gate baseline); every index-out-of-range row is provably bounded (insertion-sort parallel-slice invariant in `compression_negotiator.go`, `trim` loop guard, `container/heap` `Less`/`Swap` interface contract — do NOT add bounds checks to heap methods); and the flag-param "not last" claim miscounts the variadic (`runSpecs.parallel` is the last named param, already split into public `Run`/`RunSerial`). Do not bulk-fix without re-reading the decision docs.

## Architecture

Two Go modules in a workspace (`go.work`): the root `httputil` module (flat package with middleware + server lifecycle, and the `httputil/httpspec` subpackage for reusable HTTP behavior specs) and the `httputil/server_timing` sub-module (W3C Server-Timing instrumentation, stdlib-only, zero external deps). The root module has four external dependencies: `github.com/larsartmann/go-error-family`, `golang.org/x/time`, `github.com/justinas/nosurf`, and `github.com/larsartmann/go-etag` (the `ETag()` passthrough adapter was removed in v1.1.0; the dependency remains for its error-code registration). Go 1.27+.

### CHANGELOG Freeze Policy

Once a version tag is created, the corresponding `[version]` section in `CHANGELOG.md` is **frozen** — immutable history; corrections for released work go in `[Unreleased]`. **Release mechanics live in RELEASE.md:** every version heading needs a trailing `[X.Y.Z]:` compare-link definition with `[Unreleased]:` retargeted at tag time (CI's changelog-link check enforces it — the v1.1.0 tag shipped red on it); master + ALL tags push in ONE `git push` (RELEASE.md step 12 — sequential pushes are how v1.1.0 AND v1.4.2 shipped red); and the Release workflow's `GO_VERSION`/`GOLANGCI_LINT_VERSION` pins are frozen at the tag (RELEASE.md gate 6.6 — golangci-lint forces `GOTOOLCHAIN=local`, so only exact setup-go pins reach it; bump it together with the Go patch).

### Tag Immutability (owner directive 2026-09-11: "We never retag!")

Tags are never deleted or re-cut — not even when the tag predates code its own changelog documents. Drift between a pushed tag and its frozen changelog section ships as a patch release with a correction-of-record note (the `v1.0.1` pattern). Re-cutting would break every consumer that pinned or cached the tag.

### Why the Root Package Is Flat (Deliberate, Not Debt)

One flat root package by decision (user-confirmed 2026-08-05, re-affirmed 2026-08-30): for a middleware library where everything shares one signature, a single import path (`httputil.CORS()`) beats fragmented namespaces; compression cannot be a public sub-package anyway (root-symbol cycle). Deferred: `internal/` extraction until post-v1.0 or ~50 non-test files. Analysis: `docs/modularization/2026-08-05_DECISION.html`.

**Code map:** the file-by-file export tables for the root package, the `httpspec` subpackage, and the `server_timing` sub-module live in [docs/architecture-reference.md](docs/architecture-reference.md) — refresh that page whenever files or exports change; do not re-grow tables here.

### `httpspec` subpackage

Reusable BDD-style HTTP behavior specifications. Point `httpspec.Run(t, handler)` at any `http.Handler` to validate standard HTTP conventions via parallel subtests with human-readable names.

**Middleware pattern:** All middleware is `func(http.Handler) http.Handler` (aliased as `Middleware` in `recorder.go`). `Chain()` and `Compose()` apply middlewares in declaration order (first = outermost); `Compose()` bundles them into one reusable `Middleware`. `MiddlewareStack` adds names, duplicate prevention, and opt-in ordering validation (`Validate()`: Recovery outermost when present; `Build()` does NOT auto-validate); `MiddlewareStack.Middleware()` nests the whole stack as one middleware.

The `server_timing` sub-module is a separate Go module (`github.com/larsartmann/httputil/server_timing`, package `servertiming`), stdlib-only with zero external deps, wired via a `replace` directive (`=> ./server_timing`) and `go.work`. `ServerTimingMiddleware` injects `*servertiming.ServerTiming` via context; `servertiming.WrapServerTiming(w, r)` wraps manually; header values are CRLF-sanitized.

## Error Model

Every error the package produces is classified via `go-error-family` and typed through the `Code`/`Domain` model in `code.go`:

- **`Code`** (`type Code string`, e.g. `cors.max_age_negative`) — machine-readable identity. Constructor methods (`code.Rejection(msg)`, `code.WrapTransient(cause, msg)`, ...) return `*errorfamily.Error`.
- **`Domain`** — the code prefix before the first dot (`cors`, `server`, `compression`, `decompression`, `ratelimit`, `stack`, `maxbodysize`, `requestid`, `security`, `metrics`, `nonce`, `http`, `csrf`). By component, not lifecycle — `Family` encodes lifecycle.
- **`DomainOf(err)` / `InDomain(err, domain)`** — hierarchy queries via `errors.AsType[errorfamily.Coded]` (Go 1.26 generic).
- **Sentinels**: package-level `err*` vars built from `Code` constructors. `errors.Is` matches by code+family (errorfamily `Is` semantics), so `WithContext`/`WithCause` clones still match their sentinel.
- **Exported `ErrCode*` string constants stay untyped** for backward compatibility; internal construction uses typed mirrors (`codeWriteFailed = Code(ErrCodeWriteFailed)`). Do NOT create parallel exported `Code` aliases.
- **Config errors are `Rejection`** (fix the config, never retry); runtime write/hijack failures are `Transient`; shutdown and pool-contract violations are `Infrastructure`; corrupt compressed bodies are `Corruption`. Legacy CSRF config codes stay `Infrastructure` with `WithCause(ErrCSRFConfig)` chaining for backward compatibility, and use the historical underscore spelling (`csrf_samesite_insecure`, no dot).
- **Message templates**: `errors.go` holds an `errorTemplates` map (what/why/fix/wayOut per code, `{key}` placeholders from context). `errors_templates_test.go` asserts completeness — when adding a code, add it to the map, the list, and the domain test, then sweep the doc side in the same change: FEATURES.md inventory, docs/v1-stability.md, and the README/architecture-reference classification tables.

Per-source error codes, families, retryability, and triggers: [docs/architecture-reference.md](docs/architecture-reference.md) Error Classification section. Classified errors implement `Coded`, `Classified`, `Contextual`, `Retryable`; use `errorfamily.Classify(err)` for retry decisions and `httputil.InDomain(err, domain)` to route by component; sentinels live next to their validators.

## Non-Obvious Behaviors

- **`Start`/`StartTLS` clear listener state before delivering a Serve error** — `clearListener()` and `started.Store(false)` run before the error is sent to the returned channel, so receiving an error implies (happens-before) `ListenerAddr()` reports not-listening and a retry `Start` is possible. Never move cleanup back into a `defer` after the send: the buffered channel send completes before the defer runs, which re-opens the race caught by `TestServer_StartTLS_ListenerClearedOnCertFailure` (2026-09-11).
- **`Server.Start`/`StartTLS` are single-shot** — a second call returns `server.already_started` (Rejection, `errServerAlreadyStarted`) instead of binding an untracked orphan listener; the guard is an `atomic.Bool` CAS, and both the started flag and the listener are cleared when Serve/ServeTLS returns or fails, so a retried `Start` is legal. `ListenerAddr()` reports `(nil, false)` whenever the server is not currently listening (never started, failed to bind, or shut down).
- **`ResponseRecorder.Status()` returns `0`** (not `200`) when `WriteHeader` hasn't been called. Check `WroteHeader()` to distinguish "no status set" from "status was actually 0".
- **`ClientIP` trusts proxy headers blindly** — it does not validate X-Forwarded-For or X-Real-IP. Only safe behind a reverse proxy that strips/overwrites these headers.
- **`Compression` negotiates encodings** per request from `Accept-Encoding` using RFC 7231 q-values and a server priority order (brotli > zstd > gzip > deflate > identity). A missing header (or an empty one) is served uncompressed by default — `CompressionConfig.AbsentEncoding` (zero value `AbsentEncodingIdentity`, issue #4); `AbsentEncodingFirstConfigured` restores the pre-v1.2 behavior of picking the highest-priority configured encoding. Validate rejects unknown policy values (`compression.absent_encoding_invalid`).
- **`Compression` pools writers per encoding**, so gzip and deflate each have their own `sync.Pool` owned by the negotiator (one pool per encoding per `Compression` instance). Custom factories can opt into pooling by implementing `Reset(io.Writer)`.
- **`Compression` short-circuits identity encoding** — when the client requests `identity` (or no encoding is negotiated), the middleware passes the raw `ResponseWriter` through without wrapping. This means `nopCloserWriter`, `nopFlushCloser`, and `passthroughFactory` are only reachable via direct `compressWriter` construction (unit tests), not through the `Compression()` middleware. They are defensive code for the `WriterFactory` contract.
- **`RequestID` default generator** produces a 16-byte time-ordered ID (Unix seconds + atomic counter + random tail) and amortizes `crypto/rand` syscalls across ~256 IDs via a generation-swapped immutable ring: published generation buffers are never written again (reader copies are race-free), and slot claims are monotonic and generation-stamped, so no slot is drawn twice across refills. Hot path costs one extra atomic pointer load; don't "simplify" it back to a single shared buffer — that reintroduces the copy-vs-refill torn-read race.
- **`CORS` denies unmatched origins by default** — `DefaultCORSConfig()` sets `DenyUnmatched: true`. When `AllowAllOrigins` is false and the origin matches no `AllowedOrigins` entry, no `Access-Control-Allow-Origin` header is sent. Bare `CORSConfig{...}` literals get the zero value (`false`), which falls back to `"*"` — set `DenyUnmatched: true` explicitly or start from `DefaultCORSConfig()`.
- **`CORS` `AllowPrivateNetwork` is preflight-only and opt-in** — when true, only the middleware-generated preflight 204 carries `Access-Control-Allow-Private-Network: true` (Chrome's Private Network Access / Local Network Access check), never on actual requests and never with `OptionsPassthrough`. It is sent unconditionally on the preflight rather than echoed (matching the Chromium-verified dnsblockd consumer implementation — do not "simplify" it into an echo; Chrome-side signal changes would silently break every LNA-gated fetch). Default `false`. **Spec-churn note (2026-09-23):** the header is the PNA draft (on hold); shipping LNA (Chrome 142+) gates via a user permission and ignores this header — the middleware stays as-is (harmless, self-documenting, honored by PNA-era builds), but current-Chrome LNA compliance comes from the permission flow.
- **`Compression` uses `Level` when `WriterFactories` is empty** — `Compression()` builds factories from `cfg.Level` (defaulting to `gzip.DefaultCompression` when Level is 0 — "0 means unset" is the field's one documented meaning; a genuinely uncompressed stream is not expressible via Level). When `WriterFactories` is supplied, it takes precedence and `Level` is ignored — accordingly the Level range check applies only when `WriterFactories` is empty, and the constructor checks the level at its use site (before filling), so configs that set both never get a spurious warning. `DefaultWriterFactoriesForLevel` takes a raw stdlib level where `0` means `gzip.NoCompression`.
- **`CompressionConfig.IncompressibleTypes`** — nil uses `DefaultIncompressibleTypes()` (backward compatible); an empty slice compresses everything (including images/video). Use `DefaultIncompressibleTypes()` to extend rather than replace the list.
- **`TokenBucketLimiter` / `RateLimit()` are removed** (v1.1.0, per the pre-declared deprecation window) — use `KeyedRateLimiterMiddleware` (`ratelimit_keyed.go`) instead; migration guide: [docs/migrating-to-keyed-rate-limiter.md](docs/migrating-to-keyed-rate-limiter.md).
- **`KeyedRateLimiter` uses O(log n) min-heap eviction** — when `MaxKeys` is set, the oldest accessed key is evicted when capacity is reached. Without `MaxKeys`, growth is unbounded. `EvictionTTL` provides lazy time-based eviction. Both can be combined.
- **`CSRFMiddleware` wraps `justinas/nosurf`** — the CSRF middleware requires the `github.com/justinas/nosurf` dependency (the third external dep). The middleware uses a double-submit cookie pattern. `CSRFConfig.Validate()` enforces secure defaults. HTMX-aware helpers (`CSRFTokenHXHeaders`, `CSRFTokenHTMLMeta`, `CSRFTokenFormField`) expose the token in formats templ/HTMX can consume.
- **CSRF origin-attestation trust boundary (verified at nosurf v1.2.0)** — nosurf's `ensureSameOrigin` short-circuits ALL Origin/Referer validation on a literal `Sec-Fetch-Site: same-origin` header. Browsers set `Sec-*` truthfully (forbidden header names), so this is safe against the classic cross-site attacker, and it is what lets non-browser API clients with valid tokens work; but `CSRFMiddleware` additionally rejects (403, `ErrCSRFAttestationConflict`) unsafe-method requests whose client-supplied attestation is contradicted by an `Origin` that nosurf itself would reject (cross-origin, not in `TrustedOrigins`, or unparseable). `Origin: null` is exempt (nosurf treats it as absent). The check runs before `SetPlaintextHTTPOrigin` and mirrors nosurf's own scheme+host comparison, with two trust-model refinements: the host comparison is case-insensitive (`strings.EqualFold` — DNS hosts are), and the client-facing scheme comes from `requestScheme(r, cfg)`, which honors `X-Forwarded-Proto` **only when the request arrives from a configured `TrustedProxies`/`TrustedProxiesCIDR` address** — behind TLS termination the browser's `Origin: https://site` otherwise reads as a forged attestation against the plaintext local view. With no trusted proxies configured the header is never honored (it is attacker-controlled from anywhere else).
- **CSRF `SameSite=None` without `Secure` falls back to `Secure=true` at construction (2026-09-15)** — rfc6265bis §5.7 makes such cookies unstorable in current browsers, so the misconfiguration already broke every state-changing browser request. `CSRFMiddleware` applies `withSecureFallback()` (keeps the `None` intent, forces `Secure=true`, logs `csrf_samesite_insecure`); `InvalidateCSRFCookie` applies the same fallback — do not bypass the helper in new cookie-writing paths. `AllowInsecureSameSiteNone` opts out verbatim for legacy clients; `Validate()` still rejects the combo; `ConfigureNosurfHandler` stays the verbatim low-level configurator. Design: docs/planning/2026-09-15_csrf-security-degrading-config-design-note.md.
- **CSRF `TrustedOrigins` parsing is all-or-nothing on both sides (2026-09-14)** — `parseTrustedOrigin` (`csrf.go`) is the single parse point (mirroring `nosurf.StaticOrigins`): both Validate's shape gate and the attestation check call it, and `parseTrustedOriginURLs` returns nil when ANY entry fails, so the attestation check can never trust a different set than the nosurf handler enforces. `Validate` rejects non-`scheme://host` entries with `csrf.trusted_origin_invalid` (Rejection, cause-chained to `ErrCSRFConfig`) — the old skip-invalid-keep-rest parsing silently split the trust semantics. Construction stays log-only: the fallback is same-origin-only validation, loudly logged.
- **`ServerTimingMiddleware` lives in the `server_timing` sub-module** — import `github.com/larsartmann/httputil/server_timing` (package `servertiming`). It injects `*servertiming.ServerTiming` via context — use `servertiming.ServerTimingFromContext(ctx)` or `servertiming.MeasureServerTiming(ctx, name)` to record metrics inside handlers. `servertiming.WrapServerTiming(w, r)` provides manual wrapping without the middleware. Header values are sanitized against CRLF injection.
- **`MiddlewareStack.Validate()` is opt-in** — `Build()` does NOT call `Validate()`. The caller decides whether to check ordering. `Validate()` enforces that Recovery is outermost when present.
- **`MiddlewareStack` is safe for concurrent use** — entries publish as atomic immutable snapshots (`atomic.Pointer[[]middlewareEntry]`, CAS publication in `Add`), so `Add` during `Build`/`Middleware`/`Names`/`Validate` is race-free and readers observe either the pre- or post-add state, never a torn mixture. `Add("")` is rejected (`stack.name_empty`, Rejection) instead of silently defeating duplicate detection. Stress-tested with 8 writers × 200 adds vs 8 readers under `-race`.
- **All middleware constructors call `Validate()` at construction** — every constructor (`CORS`, `SecurityHeaders`, `Compression`, `Decompression`, `MaxBodySizeMiddleware`, `Metrics`, `RequestID`, `KeyedRateLimiterMiddleware`, `CSRFMiddleware`, `Nonce`) calls `cfg.Validate()` via a shared `validateConfig(name, err)` helper in `recorder.go`. Invalid configs are logged via `slog` (JSON handler record with structured `code`, `family`, and `domain` fields when the error is classified) but do NOT abort construction — the middleware falls back to default values where applicable. This is the validate-and-log pattern, not validate-and-abort. For constructors with default-filling logic (Compression fills `WriterFactories`, KeyedRateLimiter fills `Limit`/`Window`), `Validate()` runs **after** defaults are applied to avoid false-positive warnings on zero-value fields. Tests that swap `slog.SetDefault` must NOT use `t.Parallel()` (global mutation races with constructor tests that log).
- **`Nonce` generates a per-request CSP nonce** — `Nonce()` produces a cryptographically random base64-encoded value (default 20 bytes / 160 bits), stores it in context via `WithNonce()`, and optionally sets the `Content-Security-Policy` header via `CSPBuilder`. Set `CSPBuilder` to nil for context-only mode (no CSP header). The nonce value in context is the raw base64 string (no `'nonce-'` prefix) — use it directly in HTML `nonce="..."` attributes or via `NonceAttr(r)` which returns `nonce="<value>"`. `Nonce()` calls `Validate()` at construction via the shared `validateConfig` helper; `Size == 0` is valid and uses the default. When `Size` is non-zero but below `minNonceSize` (16), the constructor logs a warning and falls back to `defaultNonceSize` (20) — it does NOT use the insecure small size. Two CSP builders: `RecommendedCSPWithNonce` (script-src + style-src) and `ProductionCSPWithNonce` (adds object-src/base-uri/frame-ancestors). When chaining with `SecurityHeaders`, place `Nonce` **after** (inner to) `SecurityHeaders` so the nonce-bearing CSP overwrites any static CSP — the default `SecurityHeadersConfig` sets no CSP, so there is no conflict unless you explicitly set `ContentSecurityPolicy` in both. Responses with per-request nonces must not be cached (set `Cache-Control: no-store`).
- **`Server` mutates the `*tls.Config` you hand it (via stdlib `ServeTLS`)** — `http.Server.ServeTLS`/`ListenAndServeTLS` append h2 ALPN protocols to `TLSConfig.NextProtos`. Never share one `*tls.Config` between a `Server` and a TLS client without cloning; the test suite caught exactly this race (`TestServerStartTLSServesHTTPSWithSelfSignedCert`).
- **Request-body nil-vs-`http.NoBody` convention** — `httptest.NewRequest(GET, ...)` yields `http.NoBody`, while a raw constructed request may carry a nil `Body`. Middleware must treat BOTH as "no body" and must never replace a nil `Body` with a typed nil reader (and must not panic on either). `MaxBodySize` and `Decompression` guard `r.Body == nil` explicitly before wrapping; `MaxBodySize` leaves the body untouched for methods it does not wrap. Fuzz tests (`recorder_fuzz_test.go`, `decompression_fuzz_invariants_test.go`) pin these invariants.
- **Post-header-commit writes discard errors by design ("honest silence")** — after `WriteHeader` is called, the HTTP response is in-flight and write failures cannot be surfaced to the client or caller. Error-response body writes go through the `writeCommittedBody(w, body)` helper in `recorder.go` (used by `Recovery()`, `CSRFMiddleware()`, `KeyedRateLimiterMiddleware()`). Other intentional discards with inline comments: `compressWriter.Flush()` (post-handler body flush); `Compression()` defer Close (cleanup); `limitedReader.Read()` bomb-protection Close. The erraudit-policy side of the `_ =` discards lives in the Commands erraudit block (canonical).
- **pkg.go.dev license gating — both modules are MIT** (owner decisions 2026-10-08 for `server_timing`, 2026-10-09 for the root; recorded in CHANGELOG [1.5.0]) — pkg.go.dev hides READMEs AND all docs for non-redistributable licenses (empirically verified 2026-10-08: the root page at `v1.4.2`, 18 importers, rendered neither the 782-line root `README.md` nor any documentation), and it re-indexes per tagged release, so visibility arrives only from the first tag cut after a LICENSE change (`v1.5.0` root, `server_timing/v1.0.2` sub-module). The old "root stays Proprietary — do NOT mirror the MIT switch without an explicit owner decision" guard is resolved; do not reintroduce a Proprietary LICENSE in this repo without an explicit owner decision.

## Testing Conventions

### Test Naming and Assertion Conventions

Codified 2026-08-30 (source: `06-12:f.6` — "test names must match what they assert"):

- **Name = claim.** A test name must describe exactly what is asserted, no more. `TestChain_CompressionETag_HijackPassthrough` implied byte passthrough but only asserted interface delegation — the name was a lie. If the name promises more than the body checks, add the assertions or rename.
- **Shape:** `Test<Component>_<Behavior>` with an optional `_Condition` suffix (e.g. `TestResponseRecorder_Hijack_Unsupported`, `TestNegotiator_Property_SelectsHighestQAvailable`). Benchmarks: `Benchmark<Subject>`, with `b.Run` groups when one benchmark family varies by parameter (see `BenchmarkETagAdapterOverhead`, `BenchmarkDecompression`). Fuzz targets: `Fuzz<Subject>`.
- **Helpers:** constructors `newXxx()`, assertions `assertXxx(t, ...)` starting with `t.Helper()`, doubles as unexported structs in `testutil_test.go`.
- **Assertion messages:** `got = %v, want %v` style with the expression under test first; `t.Fatalf` when continuing is meaningless, `t.Errorf` otherwise.
- **Mutation-check preservation tests:** any test whose name says "preserves"/"passthrough" must fail when the preserved behavior is mutated away, not merely when the symbol disappears. Interface-assertion-only tests get a companion delegation call (see `TestServerTimingMiddleware_PreservesHijacker`, mutation-verified 2026-08-30).
- **Never assert sync.Pool instance reuse across Put→Get** — a same-goroutine Put-then-Get is not GC-stable under parallel test load (a single GC between the two statements clears the slot), so identity assertions flake. Test pool-path selection with a fake `pool.New` returning a sentinel, and factory-call-count bounds, instead.
- **Benchmark harnesses must reset mutated request state every iteration** (2026-08-30 rule, born from the Decompression incident): middleware that consumes the request body or deletes headers (`Decompression` removes `Content-Encoding`/`Content-Length`) leaves a reused `*http.Request` in a degraded state, so a benchmark that constructs the request once decompresses on iteration 1 and then times no-ops. Restore `req.Body` and the deleted headers inside `b.Loop`. Eyeball the implied bytes/second (`SetBytes` ÷ ns/op): anything above memory bandwidth (~50 GB/s) is a broken harness, not a fast benchmark.
- **Every "0 means X" config claim gets an execution-probe test** (2026-08-30 rule): two consecutive sessions found documented zero-value semantics that were false (`MaxDecompressionSize: 0` "disables" — it selects the 16 MiB default; `MaxBytes: 0` "unlimited" — it rejects any non-empty body). When documenting a zero-value special case, pin it with a test that executes the behavior (`TestMaxBodySize_ZeroLimitRejectsNonEmptyBody` is the pattern).
- **Fuzz invariants over line coverage for response-transforming code** (2026-08-30 rule): 96.9% line coverage missed the compression exact-fill duplication bug; a decode-and-compare round-trip invariant found it in seconds. Every component that transforms response bytes gets a gunzip-and-compare (or equivalent) invariant in its fuzz target. Reference decoders inside fuzz targets must be bounded (`io.LimitReader`) so corrupt input cannot OOM the runner.
- **Fuzz oracles model Go's normalization, not raw ABNF** (2026-09-15/10-28 rule, born from the `HealthResponse` encoding oracle): an oracle stricter than the stdlib flags valid inputs. Where Go trims/normalizes (OWS `" \t"` around header values, encoding quirks), the oracle applies the same normalization before comparing — and sweep ALL fuzz oracles for this class when one is found, not just the one that crashed.

### Example Conventions

Codified 2026-09-22 (plan: `docs/planning/2026-09-22_23-04_superb-examples.md`):

- **Examples are self-contained.** An `Example*` function must not reference `testutil_test.go` helpers (`newNoOpHandler`, `newWriteStatusHandler`, `newPanicHandler`) — unexported symbols from other test files are invisible on pkg.go.dev, making the example uncopy-pasteable. Inline handler doubles as local closures instead.
- **Deterministic outputs over raw prints.** Random values (tokens, request IDs, durations) are asserted via boolean probes or deterministic prefixes (`strings.HasPrefix`), never printed raw; a fully deterministic `// Output:` (like `ExampleNewServerTiming`'s wire-format walkthrough) beats a non-empty assertion when achievable.
- **Each Go module carries its own examples** in `example_test.go` — per-module examples attach to that module's pkg.go.dev page (the `server_timing` examples live in `server_timing/`, not root).
- **Headline features get end-to-end examples against real entry points** — e.g. CSRF token rendering through `CSRFMiddleware` output, error routing through `CORSConfig.Validate`, not synthetic error values.

### Coverage Methodology

Coverage is measured per module with `go test -race -coverprofile` (race detector on — the race build exercises different paths than the plain build); the two module percentages are reported separately (httputil vs httpspec), never averaged. Sub-100% functions are not chased to 100%: each remaining gap is individually documented in FEATURES.md with the reason it stays uncovered (kernel-level fault injection, unit-only-reachable internal paths, nosurf-internal branches). A gap without a documented reason is a bug in the docs, not a pass.

- **Same package** (`package httputil`, not `package httputil_test`) — tests can access unexported symbols
- **Plain `testing`** — no assertion libraries
- **No table-driven tests** — each case is a standalone `func Test*(t *testing.T)`
- **`t.Errorf`** for non-fatal, `t.Fatalf` for fatal assertions
- **`httptest.NewRecorder()`** + `httptest.NewRequest()` for HTTP doubles
- **Shared test helpers** in `testutil_test.go`: `newNoOpHandler()`, `newCountingHandler()`, `newTestRequest()`, `newRecorder()`
- **Test files split by middleware** — each middleware has its own `*_test.go`; chain integration in `chain_test.go`; the full file map lives in [docs/architecture-reference.md](docs/architecture-reference.md). Server-Timing tests/benchmarks/fuzz live in the `server_timing` sub-module.

## Pre-Existing Lint Warnings

There are **0 active warnings** across ~70 linters. Site-specific suppressions: `//nolint:makezero` on pre-allocated direct-index writes (`recorder.go`, `stack.go`, `nonce.go`, `id_generator*`); `varnamelen` ignores `w`, `r`, `n`, `rw`; `noctx` in test files excluded via `.golangci.yml`. In `_test.go` files generally: `exhaustruct_v5`, `testpackage`, `gochecknoglobals`, `funlen`, `cyclop`, `goconst`, `unused` are suppressed. Other active linters worth knowing: `wrapcheck`, `godox`, `forbidigo`, `gosec`, `cyclop` (max 12), `gocritic`, `ireturn`, `varnamelen`, `makezero`, `modernize`, `nolintlint` — per-linter detail in [docs/architecture-reference.md](docs/architecture-reference.md) (lint profile).

## Accepted Code Duplication

Clone baseline (re-measured 2026-09-23, art-dupl 0.7.0-81ce00b `--type-aware`): **0 groups at `-t 2`**; `-t 1` shows exactly the `Middleware` alias pair and the `scripts/doc-snippet-refs` if-err pair — anything NEW at these thresholds is a finding. The four accepted clones and their rationale (Middleware alias across the module boundary; `mw1`/`mw2`; `newTypedBodyHandler`; the scripts if-err pair): [docs/architecture-reference.md](docs/architecture-reference.md) Accepted Code Duplication section.
