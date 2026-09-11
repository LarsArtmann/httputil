# Status Report — go-modularize + architecture-visualization session (2026-09-10 06:09 CEST)

**Scope:** this session only — the `go-modularize` + `architecture-visualization` run, plus the forced improvement pass after the "Is this the best you can produce?!" challenge. Earlier sessions are covered by `docs/status/2026-09-10_03-41_composability-session-status.md` (its nonce follow-up is scheduled, not started).

**Deliverable commits (daemon, hash-mapped):** `5f0cd0c`, `91b80a7`, `70eb897` carry the diagrams, assessment, and review report. Working tree: only `.config/metadata.yaml` dirty (not mine).

---

## a) FULLY DONE

| # | Item                                                                                                                                                                                                                                                                                                                                                          | Evidence                                                                                    |
| - | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| 1 | **go-modularize 7-phase run with a HOLD verdict** — all boundaries kept; 4 counter-moves formally declined (error-model module, server/health module, httpspec module split, server_timing merge-back), each with the skill's own litmus test                                                                                                                 | `docs/architecture-understanding/2026-09-10_04_49-go-modularize-boundaries.md`              |
| 2 | **Boundary verification battery, all green** — `GOWORK=off go build` per module, `scripts/check-module-boundaries.sh` OK, `go work sync` + `go work edit -fmt` clean; failure modes FM#3/4/9/12 confirmed absent                                                                                                                                              | session run, re-runnable                                                                    |
| 3 | **External consumer research (the gap I had hand-waved, now closed)** — root: 4 owner repos (`emeet-pixyd`, `template-arch-lint`, `go-appkit`, `cqrs-htmx` v2/v3/v4); `server_timing`: standalone importer proven (`cqrs-htmx` v4 + admin-demo); `httpspec`: **zero importers anywhere**                                                                      | pkg.go.dev imported-by + Sourcegraph, recorded in boundaries.md "Resolved" section          |
| 4 | **Release + infra audit** — dual tag namespace verified (root `v*` → v0.12.0; submodule `server_timing/v*`); `go.work.sum` absence confirmed harmless (local replace pins); **CI gates verified real, not ghosts**: `ci.yml:44` runs the boundary script, golangci-lint action + `GOEXPERIMENT=jsonv2` wired, `nightly-fuzz.yml` + `release.yml` present      | git ls-remote, workflow files                                                               |
| 5 | **Three presentation-grade D2 diagrams, rendered and permission-normalized** — current boundaries (named consumers, zero-dep leaves in blue), improved target (solid = verified, dashed amber = trigger-gated, legend), and a NEW request-flow diagram (the events-and-commands analog: one GET through `Compose(Recovery, Logging, RequestID, Compression)`) | `2026-09-10_04_49-*.d2/.svg` + `2026-09-10_04_49-request-flow.d2/.svg`, all exit-code gated |
| 6 | **The lying diagram edge fixed** — first attempt drew `compose → go-compression "WriterFactory plugin"`, a dependency that does not exist; rewritten to the true direction (go-compression _provides_ factories to `Compression()`'s seam, trigger-gated)                                                                                                     | diff of `-improved.d2` across the session                                                   |
| 7 | **File-count split brain killed** — 36/37/38 circulated across session docs; precise count taken: **37** non-test files in package `httputil`, 41 module-wide; corrected in boundaries.md and both diagrams                                                                                                                                                   | `ls *.go \| grep -v _test \| wc -l` = 37                                                    |
| 8 | **Brutal self-review delivered as its skill requires** — HTML report with lying-claim inventory, ghost-system check (httpspec flagged; CI gates cleared), and 4-step improvement plan                                                                                                                                                                         | `docs/reviews/2026-09-10_05-18_brutal-self-review.html`                                     |
| 9 | **Skipped mandated step honored** — how-to-golang loaded during the improvement pass; its "no panics in library code" principle is now on record feeding the open `Then(nil)` decision                                                                                                                                                                        | review report §open                                                                         |

## b) PARTIALLY DONE

| # | Item                             | Done                                                                                                                        | Open                                                                                                                     | Blocker            |
| - | -------------------------------- | --------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ | ------------------ |
| 1 | Diagram quality                  | Correct facts, semantic color/shape vocabulary, legend, 3 diagrams                                                          | I verified exit codes and structure, not visual layout — no human has reviewed the SVGs yet                              | your eyeballs      |
| 2 | Evidence-based boundary analysis | Public-code evidence complete and recorded                                                                                  | Private repos beyond the 4 found are invisible to pkg.go.dev/Sourcegraph — only you know the full consumer set           | owner knowledge    |
| 3 | `Then(nil)` contract             | Flagged twice with full option analysis (panic / 500-stub / drop-guard) and the how-to-golang no-panics principle on record | No decision; API unchanged                                                                                               | owner decision     |
| 4 | Docs consistency                 | New deliverables internally consistent (37 everywhere)                                                                      | 2026-08-30 refresh doc still says "36 non-test files" — historical doc, needs a docs-health ANNOTATE pass, not a rewrite | scheduled, not run |
| 5 | Diagram discoverability          | Diagrams exist in docs/architecture-understanding/                                                                          | Not referenced from README/AGENTS — a diagram nobody opens has the same problem as httpspec: zero importers              | next docs pass     |

## c) NOT STARTED

| # | Item                                                                                                                         | Why                                                                                                                       | Wanted?              |
| - | ---------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- | -------------------- |
| ~~1~~ | ~~`NonceConfig.Generator` + `GenerateNonce` + composition-test cluster~~ **Won't implement — formally declined in the v1.0 sweep (rationale in nonce.go doc + DECISION_LOG; WithNonce is the injection escape hatch).** | ~~You said "implement" at 04:0x — scheduled as next session's first item, not started yet (this session was modularize+viz)~~ | ~~Yes — committed plan~~ |
| 2 | httpspec discovery push (README section, runnable example)                                                                   | New finding from the importer scan; needs a decision on how hard to push                                                  | Yes, likely          |
| 3 | Embedding the request-flow diagram into README/living docs                                                                   | Diagram written for humans; wiring not done                                                                               | Yes, cheap           |
| ~~4~~ | ~~ANNOTATE pass on older architecture docs (stale 36-file count)~~ done — docs-health pass 2026-09-11: 36-count inline-corrected in 2026-08-30_package-structure-analysis-refresh.md | ~~docs-health ANNOTATE mode, inline markers~~ | ~~Yes~~ |
| 5 | Fix `d2-syntax.md` reference (its `style.border-dashed` claim fails on the pinned d2 v0.8.1 — verified key is `stroke-dash`) | Skill-asset bug found by hitting it; the reference lives in your skills dir                                               | Yes, one line        |
| ~~6~~ | ~~CI check that .d2 files still render (`d2` diff/exit-code gate)~~ done — docs-health pass 2026-09-11: harvested to ROADMAP ideas (d2 render CI gate) | ~~Idea from this run; diagrams can rot silently otherwise~~ | ~~Nice-to-have~~ |
| ~~7~~ | ~~Quarterly re-run of the consumer/importer scan~~ done — docs-health pass 2026-09-11: harvested to ROADMAP ideas (quarterly importer scan) | ~~pkg.go.dev lags; the boundary verdict should be re-fed with fresh evidence~~ | ~~Nice-to-have~~ |

## d) TOTALLY FUCKED UP

Radical honesty about this session:

1. **I shipped a fabricated dependency edge in a diagram.** The first `-improved.d2` had `compose → go-compression "WriterFactory plugin"` — I drew it from memory instead of re-reading the seam. It took a user challenge ("Is this the best you can produce?!?!") to catch it. A diagram that lies about architecture is worse than no diagram. Root cause: I verified render exit codes but not diagram _content_. Fixed, but the failure class is: **verification stopped at "compiles"**.
2. **I skipped two mandated procedure steps in the first pass** — how-to-golang (go-modularize Phase 2.4) and the consumer research that its own output template asks for. Both were caught and corrected only because you pushed back. The skill checklist exists precisely so diligence doesn't depend on mood.
3. **My own docs disagreed on a count I measured three times** (36/37/38 across three docs written within 24 hours). The 08-30 historical doc still carries the stale figure — annotated-not-rewritten is the policy, so the residue is deliberate but must actually get annotated (§c-4).
4. **git-log co-change analysis is structurally unreliable in this repo** — the auto-commit daemon batches unrelated changes into single commits, so "these files changed together" is meaningless as coupling evidence. I caught it mid-analysis, but any future session could easily trust it. Not yet written down in AGENTS.md (see §e-6).

Nothing in the Go tree was broken this session: no code changes were needed for the verdict, and all gates (boundary script, GOWORK=off, sync) pass.

## e) WHAT WE SHOULD IMPROVE

1. **Diagram-edge provenance rule** — every edge in an architecture diagram cites its evidence (file:line or verified doc) before render, same discipline as factual claims in reports. The lying edge would have died at authoring time.
2. **Run the whole skill checklist before declaring done** — "compiles" is not "done"; the phases.md steps I skipped were the ones that produced this session's real findings.
3. **Count, never estimate** — 36/37/38 happened because I carried a number forward instead of re-running `ls \| wc -l` (5 seconds).
4. **Trigger brutal-self-review myself at the end of analysis tasks** — the challenge dynamic worked; it shouldn't need a prompt.
5. **Fix the d2-syntax skill reference** (`border-dashed` → `stroke-dash`) so the next session doesn't repeat the render failure.
6. **One line in AGENTS.md**: git-log co-change is unreliable here (daemon batching) — use dependency graphs and release cadence instead.
7. **Make the importer scan a standing input** to any future boundary decision — cheap (two queries), and it converted this session's weakest section into its strongest.

## f) 50 things to get done next

_Ranked by impact. "(tracked)" = already in TODO_LIST/ROADMAP — HARVEST must dedupe. Effort: S <30min, M 30min-2h, L >2h._

| #  | Task                                                                                                                                                                                                                    | Impact | Effort | Category      |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| ~~1~~  | ~~Implement `NonceConfig.Generator` + public `GenerateNonce` (owner-approved; scheduled first)~~ **Won't implement — formally declined in the v1.0 sweep (rationale in nonce.go doc + DECISION_LOG; WithNonce is the injection escape hatch).** | ~~High~~ | ~~M~~ | ~~Feature~~ |
| ~~2~~  | ~~Nonce × Compression/CORS/ServerTiming/CSRF composition tests (owner-approved)~~ done — v1.0.0 sweep: four composition tests landed (nonce x Compression/CORS/ServerTiming/CSRF); CHANGELOG [1.0.0] | ~~High~~ | ~~M~~ | ~~Quality~~ |
| ~~3~~  | ~~v1.0 cut decision — 4 consumer repos now exist, the flat-root trigger is armed (tracked)~~ done — v1.0.0 cut locally 2026-09-10 (tag 7a6d11b); push pending, tracked in TODO_LIST | ~~High~~ | ~~M~~ | ~~Feature~~ |
| 4  | Fix `d2-syntax.md` skill reference: `border-dashed` → `stroke-dash` (verified broken on pinned d2)                                                                                                                      | Medium | S      | Cleanup       |
| 5  | Embed request-flow + boundaries diagrams into README/AGENTS so they are discoverable                                                                                                                                    | Medium | S      | Documentation |
| 6  | httpspec discovery push: README section + runnable example (0 importers today)                                                                                                                                          | Medium | S      | Documentation |
| ~~7~~  | ~~Decide `Then(nil)` contract: panic vs 500-stub vs drop-guard; implement + document~~ done at `45026c1` | ~~Medium~~ | ~~S~~ | ~~Feature~~ |
| ~~8~~  | ~~ANNOTATE stale figures in `2026-08-30_package-structure-analysis-refresh.md` (36 → 37)~~ done — docs-health pass 2026-09-11: 36-count inline-corrected in the 08-30 refresh doc | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~9~~  | ~~AGENTS.md line: git-log co-change unreliable (daemon batching)~~ done — docs-health pass 2026-09-11: one-line consequence added to the AGENTS.md Auto-Git-Commit Daemon section | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| 10 | CI gate: `.golangci.yml`-touching commits must pass `golangci-lint run` (tracked from 03-41 report; still open, still the root-cause class that produced 13 findings)                                                   | High   | S      | Quality       |
| 11 | README composition section: secure-stack bundle idiom (tracked)                                                                                                                                                         | High   | S      | Documentation |
| ~~12~~ | ~~Verify CSRF `Sec-Fetch-Site` trust model vs nosurf (tracked)~~ done — 2026-09-10 (04-03 session): verified at nosurf v1.2.0 source; ErrCSRFAttestationConflict defense landed; see CHANGELOG [1.0.0] | ~~High~~ | ~~M~~ | ~~Quality~~ |
| 13 | ExampleCompose / ExampleMiddlewareFunc_Then / ExampleMiddlewareStack (tracked)                                                                                                                                          | Medium | S      | Documentation |
| 14 | BenchmarkCompose + BenchmarkMiddlewareStack_Middleware (tracked)                                                                                                                                                        | Medium | S      | Quality       |
| ~~15~~ | ~~`docs/integrations/compose-patterns.md` (tracked)~~ done — v1.0.0 sweep: docs/integrations/compose-bundles.md, linked from README | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~16~~ | ~~Tag + release next version carrying the composition API (tracked)~~ done — v1.0.0 cut locally 2026-09-10; push pending | ~~Medium~~ | ~~M~~ | ~~Feature~~ |
| 17 | Re-run consumer/importer scan quarterly; re-feed boundary verdict (new)                                                                                                                                                 | Low    | S      | Quality       |
| 18 | CI check that committed .d2 files still render (new)                                                                                                                                                                    | Low    | S      | Quality       |
| 19 | Fix `d2-syntax.md` install line parity check: add `stroke-dash` snippet + quoting examples for `;`/`,` labels (extends #4)                                                                                              | Low    | S      | Cleanup       |
| ~~20~~ | ~~Resolve go 1.26.7 vs gopls stdversion warnings (tracked)~~ done — v1.0.0 sweep: go.mod/go.work/CI converged on 1.26.7 (the older CI pin was broken) | ~~Medium~~ | ~~S~~ | ~~Cleanup~~ |
| ~~21~~ | ~~Migrate `exhaustruct` → `exhaustruct_v5` (tracked; warning fires on every lint run)~~ done — 2026-09-10 (04-03 a8): both .golangci.yml files migrated on golangci-lint 2.13.2 | ~~Medium~~ | ~~S~~ | ~~Cleanup~~ |
| ~~22~~ | ~~Decide `CompressionConfig.Level = 0` semantics (tracked)~~ done — 2026-09-10 (04-03 a4): 0 means unset; TestCompression_ZeroLevelMeansDefaultCompression pins it | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| ~~23~~ | ~~CHANGELOG coverage for the concurrent session's csrf/errors/decompression work, if unwritten (carry-over check)~~ done — 2026-09-10 (04-03 a17): CHANGELOG entries written; now in the frozen [1.0.0] section | ~~Medium~~ | ~~S~~ | ~~Documentation~~ |
| ~~24~~ | ~~Spot-check new error codes against `errorTemplates` + `allHTTputilErrorCodes` (carry-over check)~~ done — 2026-09-10 (04-03 a15): five hardened validators each carry code + template + allHTTputilErrorCodes entry + tests | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| ~~25~~ | ~~Confirm remaining table-driven conversions landed; close or trim the TODO item (the other session marked one item `[x]` but left it in the list — docs-health says completed items get deleted, they live in CHANGELOG)~~ done — 2026-09-10 (04-03 a6/a17): 7 legacy tables converted; TODO item closed | ~~Medium~~ | ~~S~~ | ~~Cleanup~~ |
| ~~26~~ | ~~Sweep for other Must*/panic-site doc references after the no-Must* decision (README, integrations docs)~~ done — docs-health pass 2026-09-11: grep of README + docs/integrations finds zero Must* references | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~27~~ | ~~CSRF docs: canonical-spelling note where `X-CSRF-Token` appears in README/integrations (tracked)~~ done — docs-health pass 2026-09-11: README CSRFConfig table now shows the canonical X-Csrf-Token value with a case-insensitivity note | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~28~~ | ~~Slim AGENTS.md below 30KB (tracked; grew again this session)~~ done — v1.0.0 sweep: 58.5 KB to 29.3 KiB via the docs/architecture-reference.md split | ~~Low~~ | ~~M~~ | ~~Documentation~~ |
| ~~29~~ | ~~dprint into flake devShell (tracked; markdown still unverified)~~ done — v1.0.0 sweep: pkgs.dprint in the flake devShell (verified 0.56.1) | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
| ~~30~~ | ~~Conventional-commit lint in CI (tracked)~~ done — v1.0.0 sweep: commit-lint job + scripts/check-commit-lint.sh | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
| 31 | TokenBucketLimiter removal at v1.0 (tracked, plan T18)                                                                                                                                                                  | Medium | M      | Cleanup       |
| ~~32~~ | ~~Config-validation hardening batch (tracked)~~ done — 2026-09-10 (04-03 a15): cors.methods_empty, decompression.encoding_unrecognized/_duplicate, compression.incompressible_prefix_invalid, csrf.max_age_negative | ~~Medium~~ | ~~M~~ | ~~Quality~~ |
| ~~33~~ | ~~KeyedRateLimiter property test + MaxKeys churn bench (tracked)~~ done — v1.0.0 sweep: property tests + BenchmarkKeyedRateLimiter_MaxKeysChurn; CHANGELOG [1.0.0] | ~~Low~~ | ~~M~~ | ~~Quality~~ |
| ~~34~~ | ~~MaxBodySize bench + fuzz target (tracked: only middleware with neither)~~ done — v1.0.0 sweep: FuzzMaxBodySize + BenchmarkMaxBodySize; CHANGELOG [1.0.0] | ~~Low~~ | ~~M~~ | ~~Quality~~ |
| ~~35~~ | ~~Negotiator wire-format fuzz target (tracked)~~ done — v1.0.0 sweep: FuzzNegotiatorWireFormat; multistream note in the FuzzCompression comment | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~36~~ | ~~Nightly fuzz crash issue-template (tracked)~~ done — v1.0.0 sweep: nightly-fuzz.yml crash-issue step (run-log link + corpus pointer + label fallback) | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~37~~ | ~~govulncheck for `server_timing` in CI (tracked)~~ done — v1.0.0 sweep: govulncheck scans both modules in ci.yml and release.yml | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~38~~ | ~~CI release workflow (tracked)~~ done — v1.0.0 sweep: release.yml builds, vets, race-tests, lints, and govulnchecks both modules before publishing | ~~Low~~ | ~~M~~ | ~~Feature~~ |
| ~~39~~ | ~~go-error-family upstream conditional-request classification (tracked; verify-before-filing first)~~ done — v1.0.0 sweep: verified draft at docs/planning/2026-09-10_go-error-family-conditional-request-classification-issue-draft.md; filing remains an owner action | ~~Low~~ | ~~M~~ | ~~Feature~~ |
| 40 | go-compression extraction when the go-datastar trigger fires (tracked; now with a named consumer path in the improved diagram)                                                                                          | Low    | L      | Feature       |
| ~~41~~ | ~~Finish T13 line-by-line test review (tracked)~~ done — 2026-09-10 (04-03 a9): 5 test files read end-to-end; 2 real test bugs fixed | ~~Low~~ | ~~M~~ | ~~Quality~~ |
| ~~42~~ | ~~Test-helper hygiene trio (tracked)~~ done — v1.0.0 sweep: bench_batch_test.go dissolved, waitForTLS consolidated + fail-fast, reserveFreePort deleted, Ed25519 cert | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
| ~~43~~ | ~~flake.nix benchmark protocol app (tracked)~~ done — v1.0.0 sweep: nix run .#bench implements the 3s x 5 protocol | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
| ~~44~~ | ~~Integration docs content refresh (tracked)~~ done — v1.0.0 sweep: samber-do/huma/prometheus verified, brotli-zstd pool-skip semantics, redis deprecation corrected | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~45~~ | ~~Server.Addr resolved-port variant decision (tracked)~~ done — v1.0.0 sweep: shipped as Server.ListenerAddr() (net.Addr, bool) | ~~Low~~ | ~~S~~ | ~~Feature~~ |
| ~~46~~ | ~~ID-generator refill-path benchmark (tracked)~~ done — v1.0.0 sweep: BenchmarkIDGeneratorRefillSwap vs raw-syscall baseline | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~47~~ | ~~Chain-level exact-fill regression — verify the `[x]` item's test actually landed, then DELETE the item from TODO_LIST per docs-health (completed TODOs live in CHANGELOG)~~ done — 2026-09-10 (04-03 a12) + TODO_LIST rebuilt in the sweep (completed items live in CHANGELOG) | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
| 48 | Schedule next full-code-review before v1.0 (tracked)                                                                                                                                                                    | Medium | L      | Quality       |
| 49 | pkg.go.dev rendering check of compose.go + new docs after next tag (tracked)                                                                                                                                            | Low    | S      | Documentation |
| 50 | Re-run go-modularize assessment when any trigger fires (50 files / internal/ request / go-compression landing) — the diagrams + md make the re-run cheap (Low)                                                          | Low    | S      | Documentation |

## g) Questions I cannot answer myself

1. **Private consumer set:** pkg.go.dev/Sourcegraph see only public code and found 4 owner-owned repos. Are there private repos importing httputil (or httpspec) that I cannot see? This gates the flat-root "second consumer" trigger and the httpspec story.
2. **`Then(nil)` contract:** fail-fast panic (current, my recommendation), 500-stub handler, or drop the guard? Flagged twice, still open.
3. **httpspec intent:** zero importers anywhere — is it a deliberate in-house tool you want left alone, or should I push discovery (README + example + docs-site)? This decides whether the "discovery gap" is a backlog item or a non-goal.

---

_Point-in-time snapshot. Section (f) items marked "(tracked)" already live in TODO_LIST.md/ROADMAP.md; unmarked session-born items (#4-10, #17-19, #25-27, #47, #50) are HARVEST candidates for the next docs-health pass._

---

## Resolution addendum (2026-09-10 ~06:30 CEST) — owner answers executed

- **§g-1 answered:** PRIVATE consumer repos DO exist beyond the 4 public ones (count unknown to me). The flat-root "second consumer" trigger is armed beyond what public scans show; recorded in TODO_LIST httpspec item.
- **§g-2 answered + executed ("just don't panic"):** `MiddlewareFunc.Then(nil)` now wires a 500-stub handler (compose.go:27); the dead `crypto/rand.Read` panic guards in `id_generator.go`/`nonce.go` deleted (`rand.Read` documented never to fail — verified via go doc); test rewritten (`TestMiddlewareFunc_Then_NilHandlerServesInternalServerError`); CHANGELOG + AGENTS.md updated. Remaining 4 panic sites documented as deliberate: `recovery.go` stdlib ErrAbortHandler sentinel, `compress_pool.go` factory-contract violations (inside Recovery's envelope), `httpspec.go` unexported test helper. Gates re-run: build, lint 0 issues, `test -race -count=10`, both erraudit gates — all green.
- **§g-3 answered:** push httpspec discovery → TODO_LIST.md item added (README section + runnable example + docs-site page).
- **Context note:** a parallel session executed a v1.0.0 completion sweep during this window (compose-bundles docs, KRL property test, CORS fuzz invariant, `Server.ListenerAddr`, pool-Get skip, WrapConflict/WrapOrchestration — all landed DONE) and swept some of my in-flight doc edits into its commits. Coordination note stands: check `git log` before editing shared living docs.
