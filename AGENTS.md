# httputil — AGENTS.md

## Hard Constraints (Will Break Your Code)

These are the non-obvious rules that cause immediate lint failures. Read these before writing any code.

### Allowed Dependencies

`depguard` allows `$gostd`, `$module` root and subpackages (via explicit `github.com/larsartmann/httputil` + `/**` entries because `$module` does not expand correctly in depguard v2.12.2), `github.com/larsartmann/httputil/server_timing` (the sub-module, listed explicitly because `/**` does not match separate go.mod modules), `github.com/larsartmann/go-error-family` (same author, zero transitive deps), `github.com/larsartmann/go-etag` (same author; the `ETag()` adapter was removed in v1.1.0 — the dependency stays for the go-etag error-code registration superset), `golang.org/x/time` (canonical Go extension for rate limiting), and `github.com/justinas/nosurf` (CSRF protection, used by `csrf.go`). No other third-party libraries.

### `exhaustruct_v5` — Every Struct Field Must Be Set

When creating any struct literal, you must populate **every field**. This applies to `CORSConfig`, `ResponseRecorder`, and all stdlib structs except `os/exec.Cmd`; relaxed in test files. Settings keys are `enforce-patterns`/`ignore-patterns`; struct tags are not honored — use `//nolint:exhaustruct_v5` directives.

### `err113` — No Inline `errors.New()`

Package-level sentinel errors only. Do not call `errors.New()` or `fmt.Errorf()` inside functions to create error values that could be package-level sentinels. The approved construction pattern is the `Code` type in `code.go`: define a `const codeFoo = Code("foo.failure")` and a package-level `var errFoo = codeFoo.Rejection("message")` sentinel, then return `errFoo.WithContext(...)` / `WithCause(...)` clones (they are safe: With* methods copy).

### `wsl_v5` — Strict Whitespace Rules

Blank lines before `return`, after declarations, around control flow. Run `golangci-lint fmt` after editing — manual whitespace will likely be wrong.

### `nonamedreturns` — No Named Return Values

Do not use named returns in function signatures.

### `noctx` — Always Use Context

`http.NewRequest` is banned. Use `http.NewRequestWithContext` (or `net.ListenConfig.Listen`); `httptest.NewRequest` in test files is excluded via `.golangci.yml`.

### `godot` — Comments End With Periods

All doc comments and regular comments must end with a period.

### `mnd` — No Magic Numbers

Extract numeric literals into named constants (`defaultMaxAge`, `defaultCompressionMinSize`).

### `gosec` — G705 Excluded Globally

G705 ("XSS via taint analysis") is excluded in `.golangci.yml` gosec settings. This library's purpose is writing HTTP response bodies, so every `ResponseWriter.Write` is intentional output — G705 is structurally a false positive here. Do **not** re-add per-site `//nolint:gosec` directives for response writes; they are fragile under `nolintlint` (flagged as "unused" because gosec taint analysis is non-deterministic across cache states).

### `paralleltest` — Every Test Must Call `t.Parallel()`

If you write a test function, it must call `t.Parallel()` as its first line.

### `noinlineerr` — No Inline Error Checks

Forbidden: `if err := foo(); err != nil`. Use a separate assignment followed by the check.

### `canonicalheader` — Canonical Header Keys

Header keys must match Go's canonical MIME header form (`textproto.CanonicalMIMEHeaderKey`). Use `X-Api-Key`, not `X-API-Key` — all letters after the first hyphen segment are lowercased.

**Get-vs-Set asymmetry (the footgun behind this linter):** `Header.Get/Set/Add/Del` canonicalize their argument silently, so `Get("x-api-key")` finds what `Set("X-Api-Key")` wrote — a non-canonical _literal_ is functionally harmless through the methods and only trips the linter. The real bug appears with direct map access: `w.Header()["X-API-Key"]` bypasses canonicalization entirely, so mixing map-style access with method-style access can create two distinct entries for what looks like one header. Rules of thumb:

```go
w.Header().Set("X-Api-Key", k)        // linter-checked, canonicalized: fine
v := w.Header().Get("x-api-key")      // canonicalizes the lookup: works, but keep literals canonical
w.Header()["X-API-Key"] = []string{k} // WRONG: bypasses canonicalization; splits the entry
```

Literals in this repo are all canonical (verified 2026-08-29); no `//nolint:canonicalheader` directives remain.

### `testableexamples` — Examples Need Output

Every `Example*` function must include a `// Output:` comment directive. Untestable examples fail the lint.

### `thelper` — Test Helpers Must Call `t.Helper()`

Any function taking `*testing.T` that calls `t.Fatal`/`t.Error` must start with `t.Helper()`.

### No `Must*` Functions — Owner API Constraint (not lint-enforced)

The owner rejects ALL `Must*`-prefixed APIs (2026-09-10): they panic, and panics are rejected as an API design tool. Never propose, add, or document `MustAdd`/`MustNew*`/etc. — wiring and construction errors return errors (`MiddlewareStack.Add` is the pattern) and the caller decides how to fail. Recorded as a Non-goal in ROADMAP.md. API functions never panic by design (owner directive 2026-09-10): `MiddlewareFunc.Then(nil)` wires a 500-stub handler, and the dead `crypto/rand.Read` failure guards in `id_generator.go`/`nonce.go` were deleted (`rand.Read` is documented never to return an error). The remaining panic sites are deliberate, not API panics: `recovery.go` re-panics the stdlib `http.ErrAbortHandler` sentinel (net/http contract), `compress_pool.go` panics on writer-factory contract violations (Infrastructure family, inside `Recovery`'s catch envelope), and `httpspec.go`'s unexported `mustRequest` test helper.

## Commands

```bash
# /mnt/buildcache is unwritable in some environments — export these or every Go
# toolchain call fails with "failed to initialize build cache":
export GOCACHE=$HOME/.cache/go-build-httputil GOLANGCI_LINT_CACHE=$HOME/.cache/golangci-lint-httputil

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
# Real gates (exit 0 required): no legacy errors.As, no inline stdlib constructors.
# The full --type-aware run reports two advisory classes (measured 2026-09-14,
# do NOT migrate either): 44 `sentinel_concrete_type` — the load-bearing
# `*errorfamily.Error` sentinels, +1 per added sentinel; and 40 test-side
# `errors.Is` advisories — all correct sentinel matches.
GOEXPERIMENT=jsonv2 erraudit lint ./... --type-aware --enforce-go-error-family
GOEXPERIMENT=jsonv2 erraudit lint ./... --type legacy_as
GOEXPERIMENT=jsonv2 erraudit lint ./... --type stdlib_constructor --enforce-go-error-family
# Review verdict 2026-09-11 for a full `erraudit . --enforce-samber-oops
# --enforce-generic-return --no-suppress` run (61 findings; newer erraudit
# builds): 43 sentinel_concrete_type — REJECTED, the concrete
# `*errorfamily.Error` sentinel type is load-bearing (`errX.WithContext/
# WithCause` clone call sites need it; declaring `var errX error` breaks the
# build); 11 `ignored` `_ =` discards — the documented honest-silence set; 4
# stdlib_constructor + 2 generic_return in scripts/ — standalone stdlib tools
# the documented gate deliberately exempts (verified: gate exits 0 scoped to
# ./scripts/...); 1 real context_loss in doc-snippet-refs — fixed (import %s).

# server_timing sub-module (run from server_timing/)
cd server_timing && go test -race ./... && golangci-lint run

# Documented 3s×5 benchmark protocol (see docs/benchmarks.md)
nix run .#bench

# Release: scripts/prerelease-check.sh automates the gates; runbook is
# docs/RELEASE.md. CHANGELOG [version] sections freeze at the tag.
```

**`go test -count=1` does NOT detect data races.** Only `go test -race` catches shared-state access between goroutines. After writing or modifying ANY test that uses `t.Parallel()`, shared fixtures, or closures over mutable state, run `go test -race -count=10 ./...` to surface timing-dependent races before declaring done. (See 2026-08-05 fix in `cors_ratelimit_specs_test.go:138` for an example of a race that passed `go test -count=1 ./...` clean but failed 60% of `-race` runs.)

`golangci-lint run` is the authoritative quality gate — it's configured with ~70 linters (see `.golangci.yml`). `go vet` alone is insufficient.

### Auto-Git-Commit Daemon

An auto-git-commit daemon commits continuously; unexpected commits are expected, and inferred messages may be generic. For deliberate commits, run `git commit` explicitly with `--no-verify` when the pre-commit hook is unavailable (e.g., `dprint` missing). Consequence: git-log co-change analysis is unreliable here — the daemon batches unrelated files into single commits, so use dependency graphs, not "changed together" evidence.

### Doc-Freshness Cadence

Living docs (`TODO_LIST.md`, `FEATURES.md`, `ROADMAP.md`, `CHANGELOG.md`) are verified via the `docs-health` skill before each version tag and at least monthly. Historical `docs/status/` reports get inline `~~item~~ done at <hash>` annotations when read; fully-resolved reports move to `<dir>/archived/` via `git mv` (`a)` FULLY DONE tables, `d)`/`e)` sections, and session timelines are historical records and are never struck). Files referenced by living docs stay in place. Struck-done TODO_LIST items are deleted at each docs-health rebuild (completed work lives in CHANGELOG), not accumulated.

### BuildFlow Pipeline

`buildflow` (source: `~/projects/BuildFlow`) orchestrates the quality pipeline. `dev` is a build MODE, not a command: full run is `buildflow --build-mode dev`; `buildflow --dry-run` previews (including `skip_steps`); `buildflow -s <step>` runs ONE step — multiple `-s` flags do NOT accumulate (last one wins). Detect-only tools (markdown-lint, lychee, branching-flow, erraudit, go-structure-linter, gomod-check) never fail the run unless `--fail-on-findings`. A full `--build-mode dev` run is expected to exit non-zero on the findings gate (the documented policy-rejected residuals below); treat only NEW finding classes or failed fixable steps as breakage.

- **`.buildflow.yml`** skips `go-auto-upgrade`: its suggestions require `samber/lo`, which the depguard allowlist bans (see Allowed Dependencies).
- **`.markdownlint.json`** disables MD001/009/010/012/013/022/024/026/028/029/031/034/036/037/040/051 — each disabled rule's findings were located and judged individually (frozen archived logs, CHANGELOG section repetition, Makefile tabs, long-line style); `.markdownlintignore` excludes `vendor/`. Tool quirks: markdownlint-cli JSON output only appears with stderr merged (`-j '**/*.md' 2>&1`); lychee needs `--exclude-path vendor`.
- **Stale-binary trap:** the global `~/.local/bin/buildflow` is a plain copy, not a symlink. `nix run .#reinstall` (in `~/projects/BuildFlow`) only refreshes `./result` — after landing BuildFlow changes, `cp ~/projects/BuildFlow/result/bin/buildflow ~/.local/bin/buildflow` and confirm `buildflow --version` matches BuildFlow HEAD (`buildflow doctor` flags staleness as `env/binary-freshness`).
- **No `vendor/` directory (removed 2026-09-11, owner decision):** every dependency is public, so workspace and `GOWORK=off` builds resolve from the module cache (both verified). `.gitignore` still lists `vendor/` so a stray local `go mod vendor` is never committed. Full runs auto-skip `go-mod-vendor`/`go-work-vendor` ("no files matching: vendor/modules.txt") and the `vendor/vendor-freshness` preflight stays silent; forcing `buildflow -s go-mod-vendor` in single-step mode exits 69 ("no tools matched the project state") — expected, don't. The old "directory vendor exists but is not ignored" info finding died with the directory; an `ignore ./vendor` directive stays deliberately unadded (top-level `vendor/` is special-cased by the go toolchain, and the ecosystem convention ignores only non-Go directories like `node_modules`).
- **Result-cache gotcha:** deleting an untracked directory does NOT invalidate buildflow's result cache (keys cover tracked files only), so findings about it keep replaying for the 7-day TTL. Purge surgically: `nix shell nixpkgs#sqlite -c sqlite3 ~/.cache/buildflow/buildflow.db "DELETE FROM result_cache WHERE value LIKE '%<finding text>%';"` (52 stale rows were purged this way on 2026-09-11).
- **Historical vendor knowledge (2026-09-11 incident, relevant only if vendoring is ever reintroduced):** go1.26.7 writes `## explicit; go <ver>` markers and a `## workspace` header into modules.txt; BuildFlow's gomod-check parser matched only bare `## explicit` and mass false-positived — fixed upstream in `parseVendorModulesTxt` with regression tests (if they reappear, the installed binary is stale). `go work vendor` output also omits the replace-marking header, so `GOWORK=off go build -mod=vendor ./...` fails "not marked as replaced" — a second reason the module cache is canonical here.
- **Residual detect-only findings are policy-rejected, not debt:** branching-flow (bool-bitflags, single-implementer interfaces, httpspec panic — contradict documented decisions), erraudit `_ =` "honest silence" discards (see Post-header-commit writes below), go-structure-linter (flat root package is deliberate; its "compiled binary in git" claim is false — `scripts/coverage-threshold/` is Go source). Do not bulk-fix these without re-reading the decision docs.

## Architecture

Two Go modules in a workspace (`go.work`): the root `httputil` module (flat package with middleware + server lifecycle, and the `httputil/httpspec` subpackage for reusable HTTP behavior specs) and the `httputil/server_timing` sub-module (W3C Server-Timing instrumentation, stdlib-only, zero external deps). The root module has four external dependencies: `github.com/larsartmann/go-error-family`, `golang.org/x/time`, `github.com/justinas/nosurf`, and `github.com/larsartmann/go-etag` (the `ETag()` passthrough adapter was removed in v1.1.0; the dependency remains for its error-code registration). Go 1.26+.

### CHANGELOG Freeze Policy

Once a version tag (e.g., `v0.8.0`) is created, the corresponding `[version]` section in `CHANGELOG.md` is **frozen** — it is immutable history. Corrections, additions, or clarifications for already-released work go in `[Unreleased]`. This prevents retroactive edits that make release history unreliable.

**Link-definition convention (hard release constraint):** every version heading needs a trailing `[X.Y.Z]:` compare-link definition in the block at the bottom of `CHANGELOG.md`, and `[Unreleased]:` must be retargeted to `vX.Y.Z...HEAD` at tag time. CI's changelog-link check enforces this; it is discoverable only by a red run otherwise (the v1.1.0 tag shipped red on exactly this, 2026-09-11).

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
- **Message templates**: `errors.go` holds an `errorTemplates` map (what/why/fix/wayOut per code, `{key}` placeholders from context). `errors_templates_test.go` asserts completeness via `allHTTputilErrorCodes` — when adding a code, add it to the map, the list, and the domain test, then sweep the doc side in the same change: the erraudit advisory note above (sentinel count), FEATURES.md inventory, docs/v1-stability.md (exported identifiers only), and the README/architecture-reference classification tables.

## Error Classification

Per-source error codes, families, retryability, and triggers: [docs/architecture-reference.md](docs/architecture-reference.md), Error Classification section. Classified errors implement `Coded`, `Classified`, `Contextual`, `Retryable`; use `errorfamily.Classify(err)` for retry decisions and `httputil.InDomain(err, domain)` to route by component. Config validators return classified `Rejection` errors with field values in context; sentinels live next to their validators.

## Non-Obvious Behaviors

- **`Start`/`StartTLS` clear listener state before delivering a Serve error** — `clearListener()` and `started.Store(false)` run before the error is sent to the returned channel, so receiving an error implies (happens-before) `ListenerAddr()` reports not-listening and a retry `Start` is possible. Never move cleanup back into a `defer` after the send: the buffered channel send completes before the defer runs, which re-opens the race caught by `TestServer_StartTLS_ListenerClearedOnCertFailure` (2026-09-11).
- **`Server.Start`/`StartTLS` are single-shot** — a second call returns `server.already_started` (Rejection, `errServerAlreadyStarted`) instead of binding an untracked orphan listener; the guard is an `atomic.Bool` CAS, and both the started flag and the listener are cleared when Serve/ServeTLS returns or fails, so a retried `Start` is legal. `ListenerAddr()` reports `(nil, false)` whenever the server is not currently listening (never started, failed to bind, or shut down).
- **`ResponseRecorder.Status()` returns `0`** (not `200`) when `WriteHeader` hasn't been called. Check `WroteHeader()` to distinguish "no status set" from "status was actually 0".
- **`ClientIP` trusts proxy headers blindly** — it does not validate X-Forwarded-For or X-Real-IP. Only safe behind a reverse proxy that strips/overwrites these headers.
- **`Compression` negotiates encodings** per request from `Accept-Encoding` using RFC 7231 q-values and a server priority order (brotli > zstd > gzip > deflate > identity). If no header is present, the highest-priority configured encoding is chosen.
- **`Compression` pools writers per encoding**, so gzip and deflate each have their own `sync.Pool` owned by the negotiator (one pool per encoding per `Compression` instance). Custom factories can opt into pooling by implementing `Reset(io.Writer)`.
- **`Compression` short-circuits identity encoding** — when the client requests `identity` (or no encoding is negotiated), the middleware passes the raw `ResponseWriter` through without wrapping. This means `nopCloserWriter`, `nopFlushCloser`, and `passthroughFactory` are only reachable via direct `compressWriter` construction (unit tests), not through the `Compression()` middleware. They are defensive code for the `WriterFactory` contract.
- **`RequestID` default generator** produces a 16-byte time-ordered ID (Unix seconds + atomic counter + random tail) and amortizes `crypto/rand` syscalls across ~256 IDs via a generation-swapped immutable ring: published generation buffers are never written again (reader copies are race-free), and slot claims are monotonic and generation-stamped, so no slot is drawn twice across refills. Hot path costs one extra atomic pointer load; don't "simplify" it back to a single shared buffer — that reintroduces the copy-vs-refill torn-read race.
- **`CORS` denies unmatched origins by default** — `DefaultCORSConfig()` sets `DenyUnmatched: true`. When `AllowAllOrigins` is false and the origin matches no `AllowedOrigins` entry, no `Access-Control-Allow-Origin` header is sent. Bare `CORSConfig{...}` literals get the zero value (`false`), which falls back to `"*"` — set `DenyUnmatched: true` explicitly or start from `DefaultCORSConfig()`.
- **`CORS` `AllowPrivateNetwork` is preflight-only and opt-in** — when true, only the middleware-generated preflight 204 carries `Access-Control-Allow-Private-Network: true` (Chrome's Private Network Access / Local Network Access check). Never set on actual requests and never with `OptionsPassthrough` (the handler owns the preflight there). It is sent unconditionally on the preflight rather than echoed from the request header, matching the Chromium-verified consumer implementation (dnsblockd `server/cors.go`) — do not \"simplify\" it into an echo, a Chrome-side signal change would silently break every LNA-gated fetch. Default `false`.
- **`Compression` uses `Level` when `WriterFactories` is empty** — `Compression()` builds factories from `cfg.Level` (defaulting to `gzip.DefaultCompression` when Level is 0 — "0 means unset" is the field's one documented meaning; a genuinely uncompressed stream is not expressible via Level). When `WriterFactories` is supplied, it takes precedence and `Level` is ignored — accordingly the Level range check applies only when `WriterFactories` is empty, and the constructor checks the level at its use site (before filling), so configs that set both never get a spurious warning. `DefaultWriterFactoriesForLevel` takes a raw stdlib level where `0` means `gzip.NoCompression`.
- **`CompressionConfig.IncompressibleTypes`** — nil uses `DefaultIncompressibleTypes()` (backward compatible); an empty slice compresses everything (including images/video). Use `DefaultIncompressibleTypes()` to extend rather than replace the list.
- **`TokenBucketLimiter` / `RateLimit()` are removed** (v1.1.0, per the pre-declared deprecation window) — use `KeyedRateLimiterMiddleware` (`ratelimit_keyed.go`) instead; migration guide: [docs/migrating-to-keyed-rate-limiter.md](docs/migrating-to-keyed-rate-limiter.md).
- **`KeyedRateLimiter` uses O(log n) min-heap eviction** — when `MaxKeys` is set, the oldest accessed key is evicted when capacity is reached. Without `MaxKeys`, growth is unbounded. `EvictionTTL` provides lazy time-based eviction. Both can be combined.
- **`CSRFMiddleware` wraps `justinas/nosurf`** — the CSRF middleware requires the `github.com/justinas/nosurf` dependency (the third external dep). The middleware uses a double-submit cookie pattern. `CSRFConfig.Validate()` enforces secure defaults. HTMX-aware helpers (`CSRFTokenHXHeaders`, `CSRFTokenHTMLMeta`, `CSRFTokenFormField`) expose the token in formats templ/HTMX can consume.
- **CSRF origin-attestation trust boundary (verified at nosurf v1.2.0)** — nosurf's `ensureSameOrigin` short-circuits ALL Origin/Referer validation on a literal `Sec-Fetch-Site: same-origin` header. Browsers set `Sec-*` truthfully (forbidden header names), so this is safe against the classic cross-site attacker, and it is what lets non-browser API clients with valid tokens work; but `CSRFMiddleware` additionally rejects (403, `ErrCSRFAttestationConflict`) unsafe-method requests whose client-supplied attestation is contradicted by an `Origin` that nosurf itself would reject (cross-origin, not in `TrustedOrigins`, or unparseable). `Origin: null` is exempt (nosurf treats it as absent). The check runs before `SetPlaintextHTTPOrigin` and mirrors nosurf's own scheme+host comparison, with two trust-model refinements: the host comparison is case-insensitive (`strings.EqualFold` — DNS hosts are), and the client-facing scheme comes from `requestScheme(r, cfg)`, which honors `X-Forwarded-Proto` **only when the request arrives from a configured `TrustedProxies`/`TrustedProxiesCIDR` address** — behind TLS termination the browser's `Origin: https://site` otherwise reads as a forged attestation against the plaintext local view. With no trusted proxies configured the header is never honored (it is attacker-controlled from anywhere else).
- **CSRF `SameSite=None` without `Secure` falls back to `Secure=true` at construction (2026-09-15)** — rfc6265bis §5.7 makes such cookies unstorable in current browsers, so the previously-honored misconfiguration meant every state-changing browser request failed CSRF validation (self-announcing breakage, verified in the design pass). `CSRFMiddleware` now applies `withSecureFallback()` (keeps the `None` intent, forces `Secure=true`, logs with `csrf_samesite_insecure`); `InvalidateCSRFCookie` applies the same fallback so the deletion cookie matches — do not bypass the helper in new cookie-writing paths. `AllowInsecureSameSiteNone` opts out verbatim for legacy-client deployments. `Validate()` is unchanged and still rejects the combo. This completes the validate-and-log contract's "falls back" step — the combo was the one `Validate()` rejection honored verbatim. `ConfigureNosurfHandler` stays the verbatim low-level configurator by contract. Design: docs/planning/2026-09-15_csrf-security-degrading-config-design-note.md.
- **CSRF `TrustedOrigins` parsing is all-or-nothing on both sides (2026-09-14)** — `parseTrustedOrigin` (`csrf.go`) is the single parse point: a TrustedOrigins string becomes a URL in exactly one place (mirroring `nosurf.StaticOrigins`), and both `validateTrustedOriginEntry` (Validate's stricter shape gate) and `parseTrustedOriginURLs` (attestation check) call it. `parseTrustedOriginURLs` returns nil when ANY entry fails `url.Parse`, mirroring `nosurf.StaticOrigins`' wholesale failure, so the attestation check can never trust a different set than the nosurf handler enforces. `CSRFConfig.Validate` rejects entries that are not usable `scheme://host` origins (unparseable, missing scheme, missing host) with `csrf.trusted_origin_invalid` (Rejection, cause-chained to `ErrCSRFConfig`) — the previous skip-invalid-keep-rest parsing silently split the trust semantics (nosurf trusted nothing, the attestation check kept parseable siblings). Construction stays log-only per the validate-and-log contract: the fallback is same-origin-only validation, loudly logged.
- **`ServerTimingMiddleware` lives in the `server_timing` sub-module** — import `github.com/larsartmann/httputil/server_timing` (package `servertiming`). It injects `*servertiming.ServerTiming` via context — use `servertiming.ServerTimingFromContext(ctx)` or `servertiming.MeasureServerTiming(ctx, name)` to record metrics inside handlers. `servertiming.WrapServerTiming(w, r)` provides manual wrapping without the middleware. Header values are sanitized against CRLF injection.
- **`MiddlewareStack.Validate()` is opt-in** — `Build()` does NOT call `Validate()`. The caller decides whether to check ordering. `Validate()` enforces that Recovery is outermost when present.
- **`MiddlewareStack` is safe for concurrent use** — entries publish as atomic immutable snapshots (`atomic.Pointer[[]middlewareEntry]`, CAS publication in `Add`), so `Add` during `Build`/`Middleware`/`Names`/`Validate` is race-free and readers observe either the pre- or post-add state, never a torn mixture. `Add("")` is rejected (`stack.name_empty`, Rejection) instead of silently defeating duplicate detection. Stress-tested with 8 writers × 200 adds vs 8 readers under `-race`.
- **All middleware constructors call `Validate()` at construction** — every constructor (`CORS`, `SecurityHeaders`, `Compression`, `Decompression`, `MaxBodySizeMiddleware`, `RequestID`, `KeyedRateLimiterMiddleware`, `CSRFMiddleware`, `Nonce`) calls `cfg.Validate()` via a shared `validateConfig(name, err)` helper in `recorder.go`. Invalid configs are logged via `slog` (JSON handler record with structured `code`, `family`, and `domain` fields when the error is classified) but do NOT abort construction — the middleware falls back to default values where applicable. This is the validate-and-log pattern, not validate-and-abort. For constructors with default-filling logic (Compression fills `WriterFactories`, KeyedRateLimiter fills `Limit`/`Window`), `Validate()` runs **after** defaults are applied to avoid false-positive warnings on zero-value fields. Tests that swap `slog.SetDefault` must NOT use `t.Parallel()` (global mutation races with constructor tests that log).
- **`Nonce` generates a per-request CSP nonce** — `Nonce()` produces a cryptographically random base64-encoded value (default 20 bytes / 160 bits), stores it in context via `WithNonce()`, and optionally sets the `Content-Security-Policy` header via `CSPBuilder`. Set `CSPBuilder` to nil for context-only mode (no CSP header). The nonce value in context is the raw base64 string (no `'nonce-'` prefix) — use it directly in HTML `nonce="..."` attributes or via `NonceAttr(r)` which returns `nonce="<value>"`. `Nonce()` calls `Validate()` at construction via the shared `validateConfig` helper; `Size == 0` is valid and uses the default. When `Size` is non-zero but below `minNonceSize` (16), the constructor logs a warning and falls back to `defaultNonceSize` (20) — it does NOT use the insecure small size. Two CSP builders: `RecommendedCSPWithNonce` (script-src + style-src) and `ProductionCSPWithNonce` (adds object-src/base-uri/frame-ancestors). When chaining with `SecurityHeaders`, place `Nonce` **after** (inner to) `SecurityHeaders` so the nonce-bearing CSP overwrites any static CSP — the default `SecurityHeadersConfig` sets no CSP, so there is no conflict unless you explicitly set `ContentSecurityPolicy` in both. Responses with per-request nonces must not be cached (set `Cache-Control: no-store`).
- **`Server` mutates the `*tls.Config` you hand it (via stdlib `ServeTLS`)** — `http.Server.ServeTLS`/`ListenAndServeTLS` append h2 ALPN protocols to `TLSConfig.NextProtos`. Never share one `*tls.Config` between a `Server` and a TLS client without cloning; the test suite caught exactly this race (`TestServerStartTLSServesHTTPSWithSelfSignedCert`).
- **Request-body nil-vs-`http.NoBody` convention** — `httptest.NewRequest(GET, ...)` yields `http.NoBody`, while a raw constructed request may carry a nil `Body`. Middleware must treat BOTH as "no body" and must never replace a nil `Body` with a typed nil reader (and must not panic on either). `MaxBodySize` and `Decompression` guard `r.Body == nil` explicitly before wrapping; `MaxBodySize` leaves the body untouched for methods it does not wrap. Fuzz tests (`recorder_fuzz_test.go`, `decompression_fuzz_invariants_test.go`) pin these invariants.
- **Post-header-commit writes discard errors by design ("honest silence")** — after `WriteHeader` is called, the HTTP response is in-flight and write failures cannot be surfaced to the client or caller. Error-response body writes go through the `writeCommittedBody(w, body)` helper in `recorder.go` (used by `Recovery()`, `CSRFMiddleware()`, `KeyedRateLimiterMiddleware()`). Other intentional discards with inline comments: `compressWriter.Flush()` (post-handler body flush); `Compression()` defer Close (cleanup); `limitedReader.Read()` bomb-protection Close.

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

### Coverage Methodology

Coverage is measured per module with `go test -race -coverprofile` (race detector on — the race build exercises different paths than the plain build); the two module percentages are reported separately (httputil vs httpspec), never averaged. Sub-100% functions are not chased to 100%: each remaining gap is individually documented in FEATURES.md with the reason it stays uncovered (kernel-level fault injection, unit-only-reachable internal paths, nosurf-internal branches). A gap without a documented reason is a bug in the docs, not a pass.

- **Same package** (`package httputil`, not `package httputil_test`) — tests can access unexported symbols
- **Plain `testing`** — no assertion libraries
- **No table-driven tests** — each case is a standalone `func Test*(t *testing.T)`
- **`t.Errorf`** for non-fatal, `t.Fatalf` for fatal assertions
- **`httptest.NewRecorder()`** + `httptest.NewRequest()` for HTTP doubles
- **Shared test helpers** in `testutil_test.go`: `newNoOpHandler()`, `newCountingHandler()`, `newTestRequest()`, `newRecorder()`
- **Test files split by middleware** — each middleware has its own `*_test.go` (e.g., `security_test.go`, `requestid_test.go`, `nonce_test.go`, `nonce_fuzz_test.go`, `recovery_test.go`, `timeout_test.go`, `logging_test.go`, `csrf_test.go`, `ratelimit_keyed_test.go`). Compression middleware tests are in `compression_test.go` and `compression_negotiator_test.go`; q-value parsing tests are in `compression_qvalue_test.go`; factory tests are in `compression_factory_test.go`; the ID generator has `id_generator_test.go`. Chain integration tests in `chain_test.go`. Server-Timing tests, benchmarks, and fuzz tests are in the `server_timing` sub-module (`server_timing/server_timing_test.go`, `server_timing/server_timing_bench_test.go`, `server_timing/server_timing_fuzz_test.go`).

### Test File Lint Relaxations

In `_test.go` files: `exhaustruct_v5`, `testpackage`, `gochecknoglobals`, `funlen`, `cyclop`, `goconst`, `unused` are suppressed.

## Pre-Existing Lint Warnings

There are **0 active warnings** across ~70 linters. Site-specific suppressions in place: `//nolint:makezero` on pre-allocated direct-index writes (`recorder.go`, `stack.go`, `nonce.go`, `id_generator*`), `varnamelen` ignores `w`, `r`, `n`, `rw`, and `noctx` warnings in test files are excluded via `.golangci.yml`.

## Accepted Code Duplication

`art-dupl --type-aware` reports **0 clone groups at thresholds `-t 2` to `-t 25`** (test files auto-excluded). Intentional clones that remain:

- **`mw1`/`mw2` in `stack_test.go`** — the integer label is intrinsic to the order-assertion test.
- **`newTypedBodyHandler` in both `testutil_test.go` and `httpspec/handlers_test.go`** — the root package cannot import `httpspec` (dependency direction).

Non-test duplication was extracted (compress-write error wrapping → `compressWriteError`; default-OK header write → `responseWrapper.writeDefaultOK()`; Hijack plain-mode switch → `beginPlainResponse()`).

## Additional Active Linters Worth Knowing

`wrapcheck`, `godox`, `forbidigo`, `gosec`, `cyclop` (max 12), `gocritic`, `ireturn`, `varnamelen`, `makezero`, `modernize`, `nolintlint` (explanations required, unused directives fail). Per-linter detail: [docs/architecture-reference.md](docs/architecture-reference.md) lint-profile section.
