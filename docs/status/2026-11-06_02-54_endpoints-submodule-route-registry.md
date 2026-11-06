# Status Report — 2026-11-06 02:54 — `endpoints` sub-module (RouteRegistry + `/endpoints` listing)

Session scope: designed, implemented, tested, and wired the new `endpoints`
Go sub-module (`github.com/larsartmann/httputil/endpoints`) — the
`/endpoints` content-negotiated route listing requested by the owner — plus
its repo machinery, mirroring `server_timing`.

## Scope line — what this report VERIFIED vs NOT

**Verified with live commands this session (all green at 2026-11-06 02:5x):**

- `go test -race -count=1 ./...` in `endpoints/` — ok, **100.0% statement
  coverage with the race detector** (`go test -race -coverprofile`, 0
  sub-100% functions).
- `golangci-lint config verify` + `golangci-lint run ./...` in `endpoints/`
  — **0 issues** across the full ~70-linter per-module config (after one
  fix round: makezero nolint, errors.AsType, wg.Go, SplitSeq, noinlineerr,
  usestdlibvars, varnamelen).
- `erraudit lint --type legacy_as` and `--type stdlib_constructor
  --enforce-go-error-family` in `endpoints/` — exit 0.
- Fuzz smoke runs (the mandatory pre-rotation discipline): 30s
  `FuzzRenderHTML` (3.7M execs), 14s `FuzzNegotiateRepresentation` (468k
  execs), 10s `FuzzParseQValue` (1M execs) — PASS after one oracle fix.
- `./scripts/check-module-boundaries.sh` — all three modules build+vet
  standalone (GOWORK=off).
- Root workspace `go build ./...` + `go vet ./...` + `go test -race -count=1
  ./...` — ok (root, httpspec, scripts; sub-modules are not covered by root
  `./...`, see e)/3).
- `cue vet` — `release.yml` (#ReleaseWorkflow), `ci.yml` +
  `nightly-fuzz.yml` (#CIWorkflow) against the pinned cue-policy checkout —
  all pass after adding the endpoints steps.
- `nix fmt` — 0 changed; `nix flake check` — **all checks passed**
  (including the new `endpoints-standalone` GOWORK=off+GOPROXY=off build).
- `art-dupl` combined-mode clone re-measurement — recorded (see a)/11).
- `git status -sb` at 02:54 — clean, `origin/master..master` empty (daemon
  swept and pushed everything).

**NOT verified (out of scope or not attempted this session):**

- **GitHub CI on the new head** — the daemon pushed minutes before this
  report; no `gh run watch` was done, so the new/edited workflow jobs
  (govulncheck endpoints, lint endpoints, nightly targets, release trigger)
  have never executed on a runner.
- **markdownlint + lychee on the new/edited `.md` files** — detect-only
  steps, not run locally this session.
- **`buildflow` full pass** — not run; expected composition: only
  policy-rejected residuals, but unproven for the new module.
- **pkg.go.dev rendering** — impossible until the first tag exists.
- Root `doc-snippet-refs` was not re-run after doc edits (it checks root
  README + docs/integrations only; neither was edited yet).

---

## a) FULLY DONE

1. **Module implementation** — `endpoints/` (package `endpoints`), stdlib-only,
   zero external deps:
   - `endpoints.go`: `Registry` (atomic-snapshot CAS publication, mirroring
     `MiddlewareStack`; zero-value usable; concurrent-safe), `Route`,
     `Handle`/`HandleFunc` (register on mux AND record in one declaration),
     `Routes()` (sorted defensive copy), `Handler()` (GET/HEAD only, 405 +
     `Allow` otherwise), `Register()` (mounts `GET /endpoints`, listing
     includes itself), `DefaultPattern`.
   - ServeMux panics translated to errors (`muxRejectedError` via recover
     closure — never-panic API surface, no named returns).
   - UTF-8 validation of pattern + summary at registration (closes the
     json/v2 reject-on-invalid-UTF-8 hole at the seam).
   - Method extraction from patterns (space/tab separators, host patterns,
     pure `methodOf` with its defensive guard unit-tested directly).
2. **Negotiation** (`negotiate.go`): RFC 9110 proactive negotiation between
   `application/json` and `text/html`; q-values via the cloned root parser;
   `text/*`→HTML, `application/*`+`*/*`→JSON; absent/garbage/broken-q
   lenient postures; q=0 excludes; ties→JSON; **never 406** (mirrors the
   `Language` middleware posture). `Vary: Accept` + `Cache-Control:
   no-store` on both representations.
3. **Renderers** (`render.go`): JSON via json/v2 with trailing newline
   (health-body convention, exact-byte pinned); HTML5 table via
   `html.EscapeString` per field, zero inline styles/scripts (CSP-clean,
   composes with `SecurityHeaders`/`Nonce`).
4. **Tests** — 70 test functions across 6 files, each a standalone
   name=claim test, `t.Parallel()` everywhere, no assertion libs, no table
   tests (repo conventions); concurrency stress (8 writers × 25 adds vs
   readers under `-race`); real-server HEAD test
   (`http.NewRequestWithContext` + `httptest.NewServer`); exact-byte render
   assertions.
5. **Fuzz** — 3 targets with lenient-posture oracles (the 2026-10-10 lesson
   applied from the start): negotiation totality+determinism, q-value bound
   invariant `[-1,1]`, HTML escaped-containment. All wired into the nightly
   rotation.
6. **Benchmarks** — 4 (`BenchmarkRegistry_Routes`, `Handler_JSON`,
   `Handler_HTML`, `BenchmarkNegotiateRepresentation`), `b.Loop` style.
7. **Examples** — 3 with verified `// Output:` blocks, self-contained.
8. **Sentinel guard** — `sentinels_type_test.go` mirrored; module carries
   plain stdlib sentinels (concrete types; guard + erraudit both clean).
9. **Module infra** — `go.mod` (go 1.27), MIT `LICENSE` (holder matches the
   other modules), complete standalone `.golangci.yml` (the doctrine: each
   module's config must be complete), `README.md` (no pkg.go.dev badge
   until the first tag — lychee-safe).
10. **Repo wiring, all verified**:
    - `go.work` `use ./endpoints`.
    - `flake.nix`: `endpoints-standalone` check (GOWORK=off, GOPROXY=off —
      proves the zero-dep claim hermetically).
    - `scripts/check-module-boundaries.sh`, `scripts/pre-sweep-check.sh`,
      `scripts/prerelease-check.sh` (vet, race tests, sentinel guard, lint;
      apidiff step deliberately absent with a pending-first-tag comment).
    - `ci.yml`: govulncheck + golangci config-verify + lint steps for
      endpoints.
    - `release.yml`: `endpoints/v*` tag trigger + sub-module gates +
      govulncheck.
    - `nightly-fuzz.yml`: the 3 new targets at 300s each.
    - `apidiff/README.md`: endpoints row (baseline pending first release).
11. **Clone baseline re-measured and recorded** — art-dupl combined mode:
    `-t 2` shows exactly the q-value clone fragments; `-t 1` adds the two
    long-standing pairs; no generics candidates. The deliberate
    `compression_qvalue.go` ↔ `endpoints/qvalue.go` clone is documented in
    architecture-reference.md Accepted Code Duplication with the
    module-isolation rationale (same doctrine as the `Middleware` alias and
    the `.golangci.yml` configs, now a triple).
12. **architecture-reference.md** — new `## endpoints sub-module` section
    (file-by-file export table); **stale-claim fix on sight**:
    server_timing's intro still said "wired via a `replace` directive" —
    false since v1.7.0/issue #43; corrected to the proxy-requirement truth.

## b) PARTIALLY DONE

1. ~~**Living-docs sweep** — only `docs/architecture-reference.md` is updated.~~
   ~~The same change must sweep (all NOT yet edited):~~
   ~~- `FEATURES.md` (Updated header, endpoints section, counts: benchmarks~~
   ~~53→57, examples 60→63, fuzz targets 30→33, coverage add 100.0%~~
   ~~endpoints; Distribution section "Both modules"→three).~~
   ~~- `CHANGELOG.md` `[Unreleased]` Added entry.~~
   ~~- Root `README.md` (module section near Server-Timing, export-table~~
   ~~rows, license line).~~
   ~~- `AGENTS.md` (Architecture "Two Go modules"→three + module description;~~
   ~~Commands: per-module endpoints block + erraudit note; Accepted Code~~
   ~~Duplication: mirror the new baseline + parser-sync duty; jscpd~~
   ~~residual "pair"→"triple").~~ done at `70d998a`,`94b9626`
2. ~~**Session-tail** — tree clean and pushed, but **CI green on the exact~~
   ~~head is unobserved** (the tail discipline requires watching it).~~ done —
   CI green observed on 94b9626 + 7989eb8 (2026-11-06); tail closed by the
   docs-sweep session.

## c) NOT STARTED

1. ~~**First release of the module** — tag (`endpoints/v0.1.0` vs `v1.0.0` is~~
   ~~owner-gated, see g), `apidiff/endpoints.export.zip` baseline generation~~
   ~~- wiring into the ci apidiff job and prerelease gate 3b, pkg.go.dev~~
   ~~badge in the module README, CHANGELOG compare-link for the new tag.~~
   ~~**→ routed to the TODO_LIST first-endpoints-release row (owner-gated tag ruling; f1–f6 ride it)**~~
   done at `ce3471a` (tag `endpoints/v0.1.0`, pre-1.0 iteration tier — owner
   ruling 2026-11-06)
2. ~~**TODO_LIST.md harvest** from this report's (f) section (docs-health~~
   ~~HARVEST).~~ done at `94b9626`
3. ~~**docs/v1-stability.md** — endpoints mention (separately versioned, no~~
   ~~baseline yet).~~ done at `94b9626`
4. ~~**docs/RELEASE.md** — verify the runbook covers a third module tag~~
   ~~(mirror of the server_timing steps).~~ done at `94b9626`
5. Optional: integration example wiring `endpoints` under httputil
   middleware chains. **→ routed to the ROADMAP Post-v1.0 ideas batch (f43)**

## d) TOTALLY FUCKED UP!

Nothing shipped broken — final state is fully green (see scope line). But
honest in-session mistakes, all self-caught:

1. **Two placeholder-garbage drafts landed briefly**: `endpoints.go` was
   first written with a nonsense `errorsNewMuxNil()` sentinel block, and
   `qvalue.go` carried a `var _ = strings.Contains` import-keeper hack plus
   a redundant duplicate of `trimOWS`. Both rewritten before commit — but
   the daemon commits intermediates, so heuristic commits may hold them.
   Lesson: write mentally-complete files, not sketches.
2. **`FuzzRenderHTML` oracle false positive** — the raw-containment check
   fired on the document's own `charset="utf-8"` markup (a lone `"` field
   value "appears raw" inside the page's own attributes). Caught by the
   seed corpus in 0.58s during the mandatory smoke run; replaced with the
   positive escaped-containment oracle; 3.7M execs clean after. This is the
   smoke-before-rotate discipline paying for itself.
3. **Wrong assumption about workspace test coverage** — I assumed root
   `go test ./...` covers `go.work` members; it does not (verified
   empirically). Consequence: the CI test job does not run endpoints tests
   — same pre-existing shape as server_timing (release-time only). Kept
   parity rather than unilaterally expanding CI policy; flagged as
   question g)/2.
4. **`ExampleRegistry_Handler` initially forgot to `Register`** the listing
   on its own mux (example printed `false`) — caught by the example runner,
   fixed immediately.

## e) WHAT WE SHOULD IMPROVE!

1. **CI sub-module test gap** (pre-existing, now doubled): full
   `go test -race` for `server_timing` and `endpoints` runs only in the
   Release workflow; CI gives them lint + govulncheck + nightly fuzz
   corpus runs only. One cheap step in the CI test job would close it
   (owner-gated, g)/2).
2. **No automated root↔endpoints parser-sync check** — the q-value clone is
   doctrine-accepted, but nothing fails if the root parser changes and the
   endpoints copy drifts. Minimum: a sync-duty note in AGENTS.md (pending
   edit); better: mirrored test-vector files both suites import... not
   possible cross-module without a dep — so the AGENTS note + re-measuring
   art-dupl on root parser changes is the realistic guard.
3. **Draft discipline** (see d)/1): complete-then-write, especially in a
   daemon-commit environment.
4. **Local doc gates**: markdownlint/lychee findings for new `.md` files
   surface only in CI/buildflow — running `buildflow -s markdown-lint` +
   lychee locally before finishing a docs sweep would catch them at the
   desk.
5. **`negotiateRepresentation` reads only the first Accept header**
   (`Header.Get`); multiple Accept header lines (rare, legal) would be
   partially ignored. Either document or join `Header.Values` — decide at
   first API review.
6. **Coverage badge blind spot** — `update-coverage-badge.sh` reads root
   coverage only; endpoints' 100% is invisible there (cosmetic; decide
   whether the badge should be workspace-wide).

## f) Next things (up to 50; brainstorm — ROADMAP/TODO_LIST fuel, not commitments)

Release lane:

~~1. Owner rules on the first tag version (g)/1) → cut via docs/RELEASE.md ritual.~~ **→ rides the TODO_LIST first-endpoints-release row** (owner-gated; ROADMAP Open questions)
~~2. Generate `apidiff/endpoints.export.zip` at the tag; wire `check-apidiff.sh` into ci.yml's apidiff job.~~ **→ rides the TODO_LIST first-endpoints-release row**
~~3. Wire the same into `scripts/prerelease-check.sh` gate 3b; delete the pending-first-tag comment.~~ **→ rides the TODO_LIST first-endpoints-release row**
~~4. Add the pkg.go.dev badge + verify install section of `endpoints/README.md` at first tag.~~ **→ rides the TODO_LIST first-endpoints-release row**
~~5. CHANGELOG `[Unreleased]` → frozen section + compare-link + `[Unreleased]` retarget (one push with tag + master).~~ **→ rides the TODO_LIST first-endpoints-release row**
~~6. Verify pkg.go.dev renders the endpoints module page (per-tag indexing).~~ **→ rides the TODO_LIST first-endpoints-release row**
~~7. Audit `docs/RELEASE.md` for third-module coverage (tag push ordering, gates).~~ done at `94b9626`
~~8. Decide + document: root go.mod deliberately does NOT require endpoints (keep it that way until proven otherwise).~~ done at `94b9626`

Docs lane (the b)/1 remainder):

~~9. FEATURES.md — Updated header with re-derived counts (57 benchmarks, 63 examples, 33 fuzz targets).~~ done at `70d998a`,`94b9626`
~~10. FEATURES.md — endpoints module section (registry, negotiation, renderers, security posture).~~ done at `70d998a`,`94b9626`
~~11. FEATURES.md — coverage bullets: add 100.0% (endpoints); gaps list: none (100%).~~ done at `70d998a`,`94b9626`
~~12. FEATURES.md — Distribution: three modules MIT; endpoints README line.~~ done at `70d998a`,`94b9626`
~~13. README.md root — endpoints section (import, quick start, negotiation summary) near Server-Timing.~~ done at `70d998a`,`94b9626`
~~14. README.md root — export table rows for the 9 endpoints exports.~~ done at `70d998a`,`94b9626`
~~15. README.md root — license paragraph mentions endpoints.~~ done at `70d998a`,`94b9626`
~~16. AGENTS.md — Architecture: three modules, endpoints description, go.work membership.~~ done at `94b9626`
~~17. AGENTS.md — Commands: `cd endpoints && go test -race ./... && golangci-lint run` block.~~ done at `94b9626`
~~18. AGENTS.md — erraudit block: endpoints clean note (re-measure `--type-aware` advisory once).~~ done at `94b9626`
~~19. AGENTS.md — Accepted Code Duplication: new baseline + q-value clone verdict + parser-sync duty.~~ done at `94b9626`
~~20. AGENTS.md — jscpd residual: `.golangci.yml` pair → triple.~~ done at `94b9626`
~~21. docs/v1-stability.md — endpoints line (separately versioned; baseline pending first tag).~~ done at `94b9626`
~~22. TODO_LIST.md — HARVEST this report's (f) into the backlog (docs-health).~~ done at `94b9626`
~~23. Re-run `doc-snippet-refs` after README edits (CI parity).~~ done at `70d998a`,`94b9626`
~~24. Annotate this report as items complete (docs-health discipline).~~ done (docs-health pass 2026-11-06)

Quality/CI lane:

~~25. `gh run watch` the pushed head; confirm all jobs green (first-ever run of the new steps).~~ done — CI green on 94b9626 and 7989eb8 (endpoints lint + config-verify steps first-run green)
~~26. Full `buildflow --build-mode dev` pass; triage anything outside the policy-rejected residual classes (expect jscpd to now see the config triple — verdict already documented).~~ **→ routed to TODO_LIST** (buildflow dev-pass row; f30/f31 folded in)
~~27. Local `buildflow -s markdown-lint` + lychee over the new/edited docs.~~ done — markdown-lint + lychee green locally this session
~~28. `go test -race -count=10 ./...` in endpoints (the count=10 lesson; CI won't do it).~~ done — `go test -race -count=10` green in endpoints/ this session
~~29. Longer local fuzz (5 min/target) before the first nightly rotation hits.~~ done — 3×5min local fuzz PASS (39.5M/206k/40M execs) this session
~~30. Confirm buildflow's golangci step fans out to endpoints (no ghost-module staleness; `ls endpoints` + re-run).~~ **→ folded into the TODO_LIST buildflow row**
~~31. Confirm go-mod-normalize leaves the new `go.mod` minor-form directive alone (fleet policy).~~ **→ folded into the TODO_LIST buildflow row**
~~32. `buildflow -s dependabot-auto-configure --fix` — add the new module's dependabot group.~~ done — dependabot endpoints group already present, verified
~~33. Check the nightly fuzz run after first rotation (rolling issue stays closed).~~ **→ routed to TODO_LIST** (nightly-fuzz row)
~~34. Decide the multiple-Accept-headers posture (e)/5) and pin with a test.~~ **→ routed to TODO_LIST** (Accept-headers posture row)

Feature lane (v0.2 candidates, additive only):

~~35. `?format=json|html` override for debugging/demos.~~ **→ routed to ROADMAP Post-v1.0 ideas batch**
~~36. Deterministic ETag + If-None-Match handling on the listing; revisit Cache-Control.~~ **→ routed to ROADMAP Post-v1.0 ideas batch**
~~37. Per-route hiding (`Hidden` flag or `HideFunc`) for admin-only routes.~~ **→ routed to ROADMAP Post-v1.0 ideas batch**
~~38. Richer route metadata (query params, auth class) — only on demand.~~ **→ routed to ROADMAP Post-v1.0 ideas batch**
~~39. `Add(pattern, summary)` recorder for non-ServeMux routers (kept out of v0 deliberately).~~ **→ routed to ROADMAP Post-v1.0 ideas batch**
~~40. OpenAPI-lite export shape (the JSON listing is deliberately minimal today).~~ **→ routed to ROADMAP Post-v1.0 ideas batch**
~~41. Snapshot versioning/callback on Routes() change (for cache invalidation in consumers).~~ **→ routed to ROADMAP Post-v1.0 ideas batch**
~~42. httpspec-flavored contract specs for the listing (Vary/Cache-Control/405) — possibly `EndpointsSpecs()` upstream of the module.~~ **→ routed to ROADMAP Post-v1.0 ideas batch**
~~43. Docs/integrations guide: endpoints under a full httputil middleware chain (Recovery→Logging→...→mux) with SecurityHeaders CSP proof.~~ **→ routed to ROADMAP Post-v1.0 ideas batch**

Housekeeping lane:

~~44. Run `scripts/pre-sweep-check.sh` before leaving the tree unattended (now covers all three modules).~~ done — scripts/pre-sweep-check.sh green (all three modules) this session
~~45. Session-tail: confirm CI green on head + tree clean (per AGENTS Session-Tail Discipline).~~ done — session tail observed 2026-11-06: CI green on both content heads, tree clean
~~46. coverage badge decision (e)/6).~~ **→ routed to TODO_LIST Low** (badge scope)
~~47. If any CI job fails on first run: triage via `gh run view --log-failed` before touching code.~~ moot — no CI job failed on first run
~~48. Consider recording the endpoints module in the fleet `meta` inventory (stack-selection discovery) once released.~~ **→ routed to TODO_LIST Low** (fleet meta inventory)
~~49. Re-run art-dupl after any root parser change (sync duty, e)/2).~~ done — codified as the standing parser-sync duty in AGENTS Accepted Code Duplication
~~50. Mark this report's items done via docs-health ANNOTATE as they land.~~ done (docs-health pass 2026-11-06)

## g) Questions I cannot answer myself

1. **First tag: `endpoints/v0.1.0` or `endpoints/v1.0.0`?** v0.1.0 keeps
   pre-1.0 iteration freedom for the API shape; v1.0.0 matches
   server_timing's day-one v1 stability posture. Tag immutability makes
   this a one-way door. And: cut it soon, or let it soak until the pending
   docs sweep + CI observation land?
2. **CI parity for sub-module tests**: add explicit
   `cd endpoints && go test -race ./...` (and the same for server_timing?)
   steps to the CI test job, closing the "sub-modules only fully tested at
   release" gap — or keep release-time-only as the deliberate shape?
3. **Consumer proof before freeze**: do you want `/endpoints` wired into a
   concrete service (e.g. dnsblockd or a cqrs-htmx app) first, so real
   usage validates the `Handle`/`Register` API shape before the first tag
   freezes it?

---

_Report written at 2026-11-06 02:54 from session-local evidence only (no
out-of-scope research), per the status-report scope-line contract. The
auto-commit daemon sweeps this file; no manual commit made (harness
contract)._
