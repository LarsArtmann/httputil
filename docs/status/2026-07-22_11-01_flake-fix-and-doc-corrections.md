# Status Report — Flake Fix, Doc Corrections, Diagram Updates

**Date:** 2026-07-22 11:01 CEST
**Session Scope:** Resolve remaining failures from the previous docs-health session (07-46): fix GOEXPERIMENT contributor blocker, inline-correct stale HTML metric cards, update D2/SVG diagrams, audit DOMAIN_LANGUAGE.md + CONTRIBUTING.md
**Starting point:** Previous status report (`2026-07-22_07-46_docs-health-and-update-old-docs-session.md`) identified 3 critical failures + 4 untouched D2/SVG files
**Commits:** `a933df1`, `5ac9571`, `46351f1`, `71e0fd4` (auto-committed by hook)

---

## a) FULLY DONE

### 1. GOEXPERIMENT=jsonv2 encoded in flake.nix (7 insertion points)

**File:** `flake.nix`

The #1 contributor blocker is resolved. `GOEXPERIMENT=jsonv2` is now set in:

| Location | Line | Purpose |
| -------- | ---- | ------- |
| `shellHook` | 73 | Auto-sets for anyone in `nix develop` |
| `apps.test` | 93 | `nix run .#test` |
| `apps.test-race` | 111 | `nix run .#test-race` |
| `apps.build` | 129 | `nix run .#build` |
| `apps.vet` | 147 | `nix run .#vet` |
| `apps.lint` | 168 | `nix run .#lint` |
| `apps.coverage` | 186 | `nix run .#coverage` |

Before: every contributor hit an immediate build failure (`encoding/json/v2: build constraints exclude all Go files`) unless they knew the secret env var. After: the devShell and all nix apps just work.

### 2. DOMAIN_LANGUAGE.md audited and corrected (8 edits)

**File:** `docs/DOMAIN_LANGUAGE.md`

| Section | Before | After |
| ------- | ------ | ----- |
| Bounded Contexts | Missing "Query Parameters" | Added row for `ParseUintQuery` |
| Compression context | "gzip/deflate + pluggable encodings" | "gzip/deflate/brotli/zstd + pluggable encodings" |
| CompressionConfig entity | "gzip compression parameters" | "compression parameters (encodings, level, min size)" |
| Compression command | "gzip-compresses responses" | "compresses responses based on Accept-Encoding negotiation" |
| DefaultCompressionConfig | "gzip default level" | "default level" |
| Compression Applied event | "gzip-encodes" | "encodes using the negotiated encoding" |
| compress_write_failed error | "Gzip writer Write fails" | "Compression writer Write fails" |
| Conventions table | "Single dependency: Only go-error-family" | "Allowed dependencies: go-error-family + golang.org/x/time" |
| Commands table | Missing ParseUintQuery | Added `ParseUintQuery(r, key)` row |

### 3. CONTRIBUTING.md fully rewritten

**File:** `CONTRIBUTING.md`

- All 4 example commands prefixed with `GOEXPERIMENT=jsonv2`
- PR checklist updated: `GOEXPERIMENT=jsonv2 go test ./...` instead of bare `go test ./...`
- Dependency claim: "No third-party dependencies (only go-error-family)" → "Allowed dependencies: `$gostd`, `go-error-family`, and `golang.org/x/time` only (enforced by `depguard`)"

### 4. Three HTML files inline-corrected (fresh-open test now passes)

| File | Inline correction | What it says |
| ---- | ----------------- | ------------ |
| `modularity.html` | Banner after stat-grid (line ~573) | "External Deps is now 2, Files (root pkg) is now ~69" |
| `full-code-review.html` | Banner after stat-grid (line ~496) + strength card fix (line ~817) | "Coverage and file count are point-in-time. Dependencies 1→2." Also fixed "1 same-author dep" → "2 deps" |
| `code-quality-scan.html` | Hero subtitle fix + banner after stat-grid (line ~518) | "93.4% coverage at time of scan" + dependency count correction |

### 5. D2 diagrams updated + SVGs regenerated

**Files:** 2 `.d2` + 2 `.svg`

- `httputil-current.d2`: Added `golang.org/x/time/rate` node, wired `ratelimit -> xtime: token bucket via`
- `httputil-current-improved.d2`: Same node, wired `extension.ratelimiter -> xtime: token bucket via`
- Both SVGs regenerated with `d2 --layout=elk`
- Verified `golang.org/x/time` text appears in both rendered SVGs

### 6. TODO_LIST.md cleaned up

**File:** `TODO_LIST.md`

Removed 3 already-done items (CHANGELOG populated, AGENTS.md GOEXPERIMENT commands, v0.5.0 tag push). Updated remaining items for post-v0.6.0 reality. The High Priority item now reflects that the flake workaround is in place but the root cause (Go version mismatch) remains.

### 7. AGENTS.md commands section updated

**File:** `AGENTS.md`

Updated the GOEXPERIMENT explanation paragraph to note that `flake.nix` now sets it automatically in the devShell and all nix apps. Manual `go` invocations outside the devShell still need it.

### 8. Full quality gate passed

| Gate | Result |
| ---- | ------ |
| `GOEXPERIMENT=jsonv2 go build ./...` | PASS |
| `GOEXPERIMENT=jsonv2 go test ./... -count=1` | PASS (285 tests) |
| `GOEXPERIMENT=jsonv2 go vet ./...` | PASS |
| `golangci-lint run` | **0 issues** |
| `nix flake check --no-build` | PASS (devShells, checks evaluated) |
| `nix fmt` | 0 files changed |

---

## b) PARTIALLY DONE

### 1. modularity.html body narrative still says "28 files"

The metric card has an inline correction banner, but line 636 in the body text still reads:

> "flat ergonomics, not a deep package tree. But at 28 files it is approaching the point"

This narrative claim is now stale (actual: ~69 .go files). I caught the metric card but missed the prose. The inline correction banner above it partially mitigates this, but a reader scanning the body text without scrolling up still sees the wrong number.

### 2. Previous session's status report NOT annotated

**File:** `docs/status/2026-07-22_07-46_docs-health-and-update-old-docs-session.md`

This file (from earlier today) claims:

- "Lint: **3 FAILURES** (paralleltest in queryparam_test.go)" — now false (fixed in `2c0cf36`)
- "Tags: v0.5.0 local only (origin latest: v0.4.0)" — now false (v0.5.0 + v0.6.0 pushed)
- "D2/SVG architecture diagrams untouched (4 files)" — now false (fixed this session)

This is a `2026-07-*` historical file that I should have annotated per the `update-old-docs` skill. I was working from it as a reference and forgot it is itself a historical snapshot that went stale due to subsequent commits.

---

## c) NOT STARTED

| Item | Why it matters |
| ---- | -------------- |
| **CHANGELOG `[Unreleased]` is empty** | The flake.nix GOEXPERIMENT fix and doc corrections are meaningful changes post-v0.6.0. The `[Unreleased]` section exists but has no entries. |
| **`nix flake check` full run (with build)** | I ran `--no-build` which evaluates derivations but doesn't build them. The apps might have issues not caught by evaluation. |
| **README.md full audit** | Previous session claims it was fixed. Spot-checked the dependency claim ("two dependencies" — correct). Did not verify every API signature, code example, or config table. |
| **ROADMAP.md creation** | Optional per docs-health model. Still doesn't exist. |
| **flake.lock modified state** | `git status` at conversation start showed `M flake.lock`. It appears to be committed now (clean working tree). Did not investigate what changed or whether it's relevant. |

---

## d) TOTALLY FUCKED UP

### 1. Did not annotate the previous session's own status report

The file `docs/status/2026-07-22_07-46_docs-health-and-update-old-docs-session.md` is a `2026-07-*` historical file. I read it at the start of this session to understand the work. It contains false claims (3 lint failures, v0.5.0 unpushed, D2 untouched) that were resolved by commits `2c0cf36`, `d8cf648`, and my work this session. I should have annotated it with a resolution section. This is the exact failure mode the `update-old-docs` skill exists to catch — a historical snapshot with stale claims that a reader might trust.

**Why I missed it:** I was reading the file as a *task list* (what to fix), not as a *historical artifact* (what needs annotation). Context-dependent blindness — the file was simultaneously my work order and a stale document.

### 2. Did not update CHANGELOG `[Unreleased]`

I made a meaningful infrastructure change (GOEXPERIMENT in flake.nix) and multiple doc corrections, but the CHANGELOG `[Unreleased]` section is empty. The previous session's status report explicitly called out the empty `[Unreleased]` as a "major release-discipline gap" — and I repeated the same gap one session later.

### 3. Partially fixed the HTML files — metric cards but not body prose

In `modularity.html`, I corrected the stat-grid metric cards (the "1 External Dep" / "28 files" numbers) but left the narrative body text at line 636 still claiming "at 28 files it is approaching the point." The inline correction banner is visible above this text, but a reader scanning the body may still take away the wrong number. This is a half-measure — I saw the metric card, fixed it, and didn't grep for the same claim in the prose.

---

## e) WHAT WE SHOULD IMPROVE

### Process failures this session

1. **Annotate the file you're reading from.** If a historical file is your reference material AND it goes stale due to your work, it needs annotation. Double-duty files are the most dangerous because you're looking *through* them, not *at* them.

2. **Grep for stale claims in body prose, not just metric cards.** Inline corrections in HTML should cover ALL instances of a stale number, not just the most visible card. A simple `grep -n "28 files"` would have caught the body text.

3. **Update CHANGELOG as you go, not at release time.** The empty `[Unreleased]` is a recurring failure. Every meaningful change should get a CHANGELOG entry immediately.

4. **Run `nix flake check` with build, not just `--no-build`.** Evaluation passing doesn't mean the apps work. I should have run `nix run .#build` to verify the GOEXPERIMENT fix actually works end-to-end through the flake.

### Architectural observations

5. **The `encoding/json/v2` situation is now mitigated but not resolved.** The flake workaround (GOEXPERIMENT in shellHook + apps) means contributors no longer hit a brick wall. But the root cause — `health.go` uses a Go 1.27 API while the module declares `go 1.26.4` — is still there. Every doc now says "requires GOEXPERIMENT=jsonv2" instead of fixing the version mismatch. This is a ticking bomb: if someone uses this library without the flake (e.g., `go get` in a non-Nix project), they get a build failure with no explanation.

6. **The auto-commit hook is a mixed blessing.** It committed my work in 4 separate commits (`a933df1`, `5ac9571`, `46351f1`, `71e0fd4`) — sometimes mid-edit. This makes the commit history noisy and means I can never batch related changes into a single logical commit. The hook fired between my D2 edits and SVG regeneration, splitting them across commits.

---

## f) Up to 50 Things We Should Get Done Next

### Critical — fix what I left broken

| # | Task | Impact | Effort |
| - | ---- | ------ | ------ |
| 1 | **Annotate `2026-07-22_07-46_docs-health-and-update-old-docs-session.md`** — add resolution section noting all 3 claims (lint failures, tags, D2) are now resolved | High — stale claims in a status report | 5 min |
| 2 | **Populate CHANGELOG `[Unreleased]`** — add flake.nix GOEXPERIMENT fix, DOMAIN_LANGUAGE.md corrections, CONTRIBUTING.md rewrite, D2/SVG updates, HTML inline corrections | High — release discipline | 10 min |
| 3 | **Fix `modularity.html` line 636** — change "at 28 files" to "~69 files" in the body prose | Medium — stale claim in narrative text | 2 min |

### High — verify the flake actually works end-to-end

| # | Task | Impact | Effort |
| - | ---- | ------ | ------ |
| 4 | **Run `nix run .#build`** — verify the GOEXPERIMENT fix works through the flake app, not just direct `go build` | High — untested path | 2 min |
| 5 | **Run `nix run .#test`** — same for tests | High — untested path | 2 min |
| 6 | **Run `nix run .#lint`** — same for lint | High — untested path | 2 min |
| 7 | **Run full `nix flake check` (with build)** — not just `--no-build` | Medium — only evaluation verified | 5 min |

### High — resolve the jsonv2 root cause

| # | Task | Impact | Effort |
| - | ---- | ------ | ------ |
| 8 | **Decide: upgrade flake to Go 1.27+ or downgrade `health.go` to `encoding/json` v1** | Critical — permanent fix | decision |
| 9 | **If upgrading: check nixpkgs has `go_1_27`** — may not be in unstable yet | High — feasibility | 5 min |
| 10 | **If downgrading: rewrite `health.go` to use `encoding/json` v1 `json.NewEncoder` or `json.Marshal`** — removes GOEXPERIMENT requirement entirely | High — simplest permanent fix | 15 min |
| 11 | **Remove GOEXPERIMENT from flake.nix if jsonv2 is dropped** — revert the workaround once root cause is fixed | Medium — cleanup | 5 min |

### Medium — docs completeness

| # | Task | Impact | Effort |
| - | ---- | ------ | ------ |
| 12 | **Full README.md audit** — verify every API signature, config table, code example against actual source | Medium — unverified this session | 30 min |
| 13 | **Add remaining config field tables to README** — ETagConfig, RateLimitConfig, MetricsConfig, SecurityHeadersConfig, RequestIDConfig, ServerConfig | Medium — API completeness | 60 min |
| 14 | **Create ROADMAP.md** — capture v1.0 vision (breaking renames, DenyUnmatched default, stability commitment) | Low — optional for libraries | 30 min |
| 15 | **Add CHANGELOG comparison links** — `[Unreleased]`, `[0.6.0]` link references at bottom of file | Low — Keep a Changelog compliance | 10 min |
| 16 | **DOMAIN_LANGUAGE.md: Compression Level value object** — line 86 says "gzip compression tradeoff" but compression level applies to all encodings, not just gzip | Low — minor inaccuracy | 2 min |

### Medium — test coverage and quality

| # | Task | Impact | Effort |
| - | ---- | ------ | ------ |
| 17 | **Add `TokenBucketLimiter` benchmark** — prove the `x/time/rate` switch was a net win | Medium | 30 min |
| 18 | **Add body-before-hijack WebSocket test variant** — exercise `beginPlainResponse()` buffer-drain path | Medium | 45 min |
| 19 | **Mutation-test the ETag path in WebSocket upgrade test** — verify ETag assertions have teeth | Low | 15 min |
| 20 | **Add fuzz test for `ParseUintQuery`** — edge cases, overflow, empty, negative | Low | 15 min |
| 21 | **Add `Example*` function for `ParseUintQuery`** — `testableexamples` linter requires `// Output:` | Low | 10 min |
| 22 | **Close compression error-branch coverage gap** — `startCompression` type mismatch, `Close` errors | Low | 30 min |
| 23 | **Close CORS wildcard edge-case coverage gap** — unusual patterns with ports, lookalike domains | Low | 30 min |
| 24 | **Add `RateLimitConfig` test for `Validate()` success path** | Low | 5 min |
| 25 | **Add `MetricsConfig` test for `Validate()` success path** | Low | 5 min |
| 26 | **Test rate limiter with IPv6 `RemoteAddr` strings** | Low | 10 min |
| 27 | **Add property-based tests for token bucket behavior** | Low | 1 hr |

### Lower — polish and future

| # | Task | Impact | Effort |
| - | ---- | ------ | ------ |
| 28 | **Add `MustNewTokenBucketLimiter` convenience variant** — panic on error for known-valid inputs | Low | 15 min |
| 29 | **Add `Retry-After` header support to `RateLimit`** — standard 429 companion header | Low | 20 min |
| 30 | **Document middleware ordering recommendations in README** — Recovery → RateLimit → MaxBodySize → CORS → ... | Low | 10 min |
| 31 | **Add brotli/zstd WriterFactory example** — via plugin pattern, no core dependency | Low | 30 min |
| 32 | **Add distributed rate-limiter example (Redis-backed)** — as documented example, not dependency | Low | 1 hr |
| 33 | **Evaluate exposing `AllowN` on the `RateLimiter` interface** — burst > 1 per request | Low | decision |
| 34 | **Consider `context.Context` support in rate limiter interface** — cancellation | Low | 30 min |
| 35 | **Add `MetricsRecorder` Prometheus-compatible example** — documented, not a dependency | Low | 30 min |
| 36 | **Add request body decompression middleware** — counterpart to Compression | Low | 2 hr |
| 37 | **Consider `httpspec` spec for CORS headers** — standard specs don't validate CORS behavior | Low | 30 min |
| 38 | **Consider `httpspec.ExpectJSON` / `ExpectHTML` builders** — verify Content-Type | Low | 15 min |
| 39 | **Add `Content-Length` preservation test for small responses** | Low | 30 min |
| 40 | **Test `Compression` with `Accept-Encoding: br` when only gzip is configured** | Low | 10 min |
| 41 | **Test compression writer pool reuse under concurrent load** | Low | 30 min |
| 42 | **Test ETag with weak indicator (`W/`) on conditional requests** | Low | 15 min |
| 43 | **Test ETag buffer overflow streaming path** (body > `MaxBufferSize`) | Low | 15 min |
| 44 | **Run `govulncheck` locally before next release** — preempt CI failure | Low | 2 min |
| 45 | **Schedule a full `nix flake check` run (with build)** — verify reproducibility end-to-end | Low | 5 min |
| 46 | **Audit all `Validate()` methods for completeness** | Low | 1 hr |
| 47 | **Add `ServerConfig.TLSConfig` validation** — accepted but not validated | Low | 30 min |
| 48 | **Consider removing the auto-commit hook** — it splits logical changes across multiple commits and fires mid-edit | Medium — commit hygiene | decision |
| 49 | **Pin D2 layout engine version** — SVGs depend on `d2 --layout=elk`, different versions may produce different output | Low | 5 min |
| 50 | **Add `CONTRIBUTING.md` section on GOEXPERIMENT permanent vs temporary status** — set expectations for contributors | Low | 5 min |

---

## g) Top 3 Questions I Cannot Figure Out Myself

### Q1: Should `encoding/json/v2` be kept or reverted to v1?

The flake workaround (GOEXPERIMENT in shellHook + all apps) means contributors inside `nix develop` no longer hit the build failure. But anyone using this library via `go get` in a non-Nix project still gets a broken build with a cryptic error. The real question is: is `json.MarshalWrite` in `health.go` important enough to justify a Go 1.27 dependency, or should it be `json.NewEncoder(w).Encode(...)` from v1? The health handler writes a tiny 2-field struct — there's no performance or API benefit from jsonv2 here. This seems like an accidental upgrade, not an intentional choice. But I can't tell if there's a plan to adopt jsonv2 more broadly.

### Q2: Should the auto-commit hook be kept, removed, or reconfigured?

The hook fired 4 times this session, splitting my work into commits that don't map to logical units of change (e.g., D2 source edits in one commit, SVG regeneration in the next). It also means I can never intentionally batch related changes. On the other hand, it guarantees nothing is lost. Is this intentional? If so, I'll work with it. If not, it's making the commit history noisier than it needs to be.

### Q3: Is the `go.mod` version (`go 1.26.4`) intentionally pinned, or should it track the latest Go?

The module declares `go 1.26.4` and the flake pins `go_1_26`. If we commit to jsonv2, both need to bump to 1.27+. If nixpkgs unstable doesn't have `go_1_27` yet, that's a blocker. I can't determine whether the 1.26 pin is a deliberate compatibility floor (consumers on older Go) or just "what was current when the project started."

---

## Metrics Snapshot

| Metric | Value |
| ------ | ----- |
| Files changed this session | 12 |
| Flake.nix GOEXPERIMENT fixes | 7 insertion points |
| DOMAIN_LANGUAGE.md corrections | 8 edits |
| HTML inline corrections | 3 files |
| D2 diagrams updated | 2 files |
| SVGs regenerated | 2 files |
| Build | PASS |
| Tests | 285 PASS |
| Vet | PASS |
| Lint | 0 issues |
| `nix flake check --no-build` | PASS |
| `nix fmt` | 0 changed |
| Git state | Clean (auto-committed across `a933df1`..`71e0fd4`) |
| CHANGELOG `[Unreleased]` | **Empty** (should have entries) |
| Previous status report annotated | **No** (should be) |
