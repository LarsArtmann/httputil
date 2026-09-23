# Status Report — 2026-09-15 17:04 CEST — Issue #4 fixed: absent `Accept-Encoding` served uncompressed by default

**Session scope:** owner instruction to READ/UNDERSTAND/RESEARCH/REFLECT/EXECUTE/VERIFY
[issue #4](https://github.com/LarsArtmann/httputil/issues/4) ("Compression
negotiates gzip for an absent Accept-Encoding header (RFC 7231 says
identity)"). Single-issue session; everything below is from this run.

**Headline:** issue #4 premise verified against current code, fixed end-to-end
(new `CompressionConfig.AbsentEncoding AbsentEncodingPolicy`, zero value
`AbsentEncodingIdentity` = new safe default, `AbsentEncodingFirstConfigured`
restores v1.1.x), full gate battery green, issue closed with root-cause
comment. Bonus: the mandated fuzz run exposed a latent fuzz-oracle bug
(`strings.TrimSpace` vs HTTP OWS) in `FuzzNegotiatorWireFormat` — oracle
fixed, minimized seed committed as a regression pin. Tree clean at report
time; master is 4 daemon commits ahead of origin (unpushed, owner-gated).

---

## a) FULLY DONE

1. **Issue #4 premise verified before coding** — read the issue, then
   confirmed `negotiator.negotiateEmptyHeader` still returned
   `order[0]` for a missing header (the issue cites v0.12.0 line numbers;
   behavior unchanged since filing). Evidence: compression_negotiator.go
   read at session start; no drift.
2. **`AbsentEncodingPolicy` designed and implemented** — typed int enum
   (precedent: `http.SameSite`), field `CompressionConfig.AbsentEncoding`,
   zero value = identity. Negotiator carries the policy
   (`buildNegotiator(factories, absentEncoding)`); `negotiateEmptyHeader`
   returns `("", 0, false)` under identity, `order[0]` under
   `FirstConfigured`. Empty header value is now also RFC-correct
   (§5.3.4: empty value = no content-coding). Files: compression.go,
   compression_negotiator.go, 4 call sites updated.
3. **Validation + full error-model sweep** — unknown policy values rejected
   with new code `compression.absent_encoding_invalid` (Rejection);
   template added to errors.go, code registered in
   errors_templates_test.go; template + domain-prefix tests green.
   Evidence: `go test -race -run 'TestEvery'` passes.
4. **8 new test functions + 2 helpers** — BDD behavior specs (absent →
   uncompressed with exact-body assertion; empty value → uncompressed;
   `FirstConfigured` → gzip with gunzip round-trip), negotiator-level tests
   for both policies (incl. empty-order under both), Validate
   rejects-unknown/accepts-both, and the mandatory zero-value execution
   probe (`TestCompression_ZeroValueAbsentEncodingServesUncompressed`,
   bare config literal, per the "0 means X" rule). Evidence: full
   `go test -race -count=1 ./...` green; targeted `-race -count=10` green.
5. **Latent fuzz-oracle bug found and fixed** — `FuzzNegotiatorWireFormat`'s
   single-token micro-oracle normalized with `strings.TrimSpace`, which
   trims `\v`, but HTTP OWS is SP/HTAB only — the malformed header
   `"\vGZIP"` was wrongly expected to negotiate `gzip` (the parser
   correctly treats it as an unsupported token → identity fallback). Oracle
   now trims `" \t"`; the failing input is committed as seed
   `testdata/fuzz/FuzzNegotiatorWireFormat/9340f3bee00680ac` (verified
   tracked in git). Evidence: seed corpus passes; 30s fuzz run PASS.
6. **Parsing oracles deliberately unperturbed** — `newTestNegotiator()` pins
   `AbsentEncodingFirstConfigured` with a rationale comment, so the
   property tests and fuzz oracle (which model q-value parsing, not
   absent-header policy) keep their invariants; the new policy has its own
   focused tests. Evidence: property tests + fuzz green unchanged.
7. **Doc sweep complete (AGENTS-mandated), counts measured not guessed** —
   CHANGELOG `[Unreleased]` Changed entry (with `fixes #4`),
   docs/migrating-to-v1.2.md (intro now "four mechanisms" + new What-to-do
   section incl. guard-retirement note), AGENTS.md compression bullet
   rewritten, FEATURES.md behavior line + error-code inventory,
   docs/v1-stability.md (CompressionConfig note, `AbsentEncodingPolicy`
   row with table re-pad, v1.2.0 outlook bullet now four changes),
   docs/architecture-reference.md exports row, README compression
   paragraph + classification-table row, TODO_LIST issue-#4 ruling struck
   with done-evidence. erraudit advisory counts re-measured on this tree:
   45 sentinels / 41 test-side advisories (AGENTS.md updated).
8. **Gate battery green** — golangci-lint fmt + run (**0 issues**; one
   cyclop 13>12 hit fixed by extracting `validateAbsentEncoding()`),
   erraudit `legacy_as` and `stdlib_constructor` gates exit 0,
   `check-changelog-links.sh` OK, `doc-snippet-refs` all-references-resolve,
   markdownlint on all touched .md clean (see b2 for the pre-existing
   MD060 class), `nix fmt`, `nix flake check` all checks passed,
   server_timing sub-module tests + lint green (untouched, sanity).
9. **Issue #4 closed** with a root-cause + fix + verification comment and a
   pointer to the v1.2.0 migration notes.
10. **No stale doc claims left behind** — grep confirms no surviving
    "highest-priority configured encoding is chosen" phrasing outside
    historical status/archived reports, and ROADMAP.md has no issue-#4
    mentions needing updates.

## b) PARTIALLY DONE

1. ~~**Gate battery is not the full release battery** — everything above ran;~~ done (superseded by the v1.2.0 release (2026-09-16) — the release battery incl. prerelease-check.sh ran per RELEASE.md; CI green on the tag)
   ~~`scripts/prerelease-check.sh` (which includes the 95% coverage gate)~~
   ~~and the documented bench protocol did **not** run this session.~~
   ~~Remaining: coverage gate over the new code+tests, benchstat~~
   ~~before/after for the negotiator. Blocker: none, time-scoped.~~
   ~~Effort: S for the script, M for benches.~~
2. **MD060 table style** — net improvement 4 → 3 findings vs HEAD (my
   compression-table re-pad in v1-stability.md eliminated one; my new rows
   are aligned), but 3 pre-existing findings remain
   (docs/architecture-reference.md:25 ×2, docs/v1-stability.md:29 ×1),
   exactly the class awaiting the owner table-style ruling (audit report
   g2). Not fixable without that ruling; single-space compact style would
   remove the class entirely.
3. **Consumer follow-through (website guard retirement)** — the issue's
   consumer-impact note says artmann-technologies-website's guard wrapper
   can be retired; the migration doc and issue comment say so, but no
   cross-repo issue/PR exists yet (different repo; needs owner
   authorization; ideally after the v1.2.0 tag exists to point at).
   Effort: S to file, M to retire+test there.
4. ~~**Nightly-fuzz validation** — fix + seed are committed but the first~~ done (observed green — subsequent nightly runs over the fixed oracle + seed reported no negotiator crashes (the #9-#13 false-positive class stayed closed))
   ~~nightly run over them (03:05 UTC) hasn't happened; "green" is tonight's~~
   ~~expected result, not yet an observed one. Effort: S to watch.~~
5. ~~**v1.2.0** — content is staged (now four behavior changes + two~~ done at `9b9e032`
   ~~additive knobs), CHANGELOG and migration doc updated, but the tag is~~
   ~~not cut and master is 4 commits ahead of origin, unpushed. Blocker:~~
   ~~owner push/release ruling (standing since the audit session).~~

## c) NOT STARTED

All pre-existing, deliberately untouched per the single-issue scope:

1. **Website guard-retirement issue** (see b3) — waiting on owner
   authorization + v1.2.0 tag. Priority: Medium.
2. **Bench before/after for the negotiator change** — waiting for the
   v1.2.0 release protocol where benchstat comparison happens anyway
   (hot-path impact is expected negligible: the new branch only executes
   for empty headers). Priority: Medium.
3. ~~**Per-module coverage re-measure** — standing ticket from the audit~~ done (docs-health pass 2026-09-23, re-measured — 97.5% httputil / 99.1% httpspec, FEATURES+README refreshed)
   ~~session (post-CSRF-fallback); this session adds tested code that will~~
   ~~shift the httputil number again. Priority: Medium.~~
4. **Audit-session owner rulings** (B1 CSRF default interpretation;
   MD060 style + buildflow cache purge; push/v1.2.0 timing) — still open,
   unchanged this session. Priority: High (they gate the release).
5. ~~**TODO_LIST batches** (release-engineering hardening, CSRF docs polish~~ done (all five batches re-verified and carried into the rebuilt TODO_LIST (2026-09-23))
   ~~batch, CSRF test-depth batch, `csrf.trusted_origin_invalid` export~~
   ~~question, go-error-family#5 follow-through, archived-corpus sampled~~
   ~~audit) — untouched; all remain sourced in TODO_LIST.md.~~

## d) TOTALLY FUCKED UP

Radical honesty; ranked by severity. Nothing currently red — but two real
self-inflicted defects this session, one shipped:

1. **The `FuzzNegotiatorWireFormat` oracle was wrong since its 2026-09-11
   hardening** (shipped bug, fixed this session). Severity: was silently
   wrong — it would have flagged correct parser behavior as a crash the
   first time the nightly fuzzer hit a control-character header, i.e. a
   future false "crash" of exactly the #9–#13 class this repo just spent a
   week cleaning up. Root cause: Go string helpers (`strings.TrimSpace`)
   encode Unicode whitespace intuition, not HTTP ABNF (OWS = SP/HTAB).
   Workaround: none needed now — oracle fixed, seed pinned, 30s run green.
   Residual risk: sibling oracles may carry the same normalization bug
   (see e1/f8).
2. **The issue-close comment cites a daemon commit hash** (`1ab88cd`,
   message "chore: auto-commit 5 changed file(s) (heuristic)") — a
   meaningless reference for any future reader, and it will stay dangling
   because the daemon batched the fix with unrelated files. Severity: low
   (the comment also points at v1.2.0 + migration doc, which degrade
   gracefully). Mitigation: follow-up comment citing the deliberate
   commit/tag once pushed (f3). Root cause: I optimized for closing within
   the session instead of for durable references.
3. **Nothing is pushed.** Four commits containing the only copy of this
   fix sit local-only (standing owner-gated state, not new — but a
   machine-die before push loses the session). Severity: medium until
   pushed. Workaround: none available to me; push is owner-gated.
4. **One AGENTS-mandated sweep item was initially missed** — the README
   error-classification table lacked a row for the new code. Caught during
   this report's self-review (not by a gate), fixed inline
   (`compression.absent_encoding_invalid` row added, lint clean).
   Severity: was a docs-drift bug for minutes, now closed. Root cause: I
   treated the sweep as a checklist from memory instead of re-reading the
   AGENTS Error Model paragraph before declaring docs done.

## e) WHAT WE SHOULD IMPROVE

1. **Oracle-normalization hygiene** — any fuzz oracle that normalizes wire
   input with Go helpers (`TrimSpace`, `ToLower`, `EqualFold`) can diverge
   from HTTP ABNF. Concrete fix: a one-pass sweep over all fuzz targets'
   oracles for Go-normalization on header-like inputs, replacing with
   HTTP-precise cutsets (`" \t"`). This is the second wrong-oracle
   incident (health encoding oracle 2026-09-11, negotiator OWS oracle
   today) — the pattern, not the instance, is the finding.
2. **Never cite daemon commits in external artifacts** — issue comments,
   docs, and reports should reference CHANGELOG sections, releases, or
   deliberate commits only. Concrete: add the rule to AGENTS.md
   (one line) and to the issue-closing habit.
3. **Re-read the AGENTS sweep paragraph before declaring a change done** —
   the missed README row (d4) was listable from the Error Model section.
   Concrete: behavior-change sessions end with a literal re-check of the
   Error Model + Testing Conventions sweep lists, not a mental summary.
4. **Run the coverage gate in behavior-change sessions** — the 95% gate is
   cheap and catches untested new branches immediately; skipping it was an
   unforced gap (b1).
5. **Settle MD060 once** — every table edit risks a new aligned-style
   finding; the pending ruling (compact style) would delete the finding
   class and the per-edit padding ceremony.
6. **Issue sessions should grep TODO_LIST for the issue number first** — I
   found the standing "Issue #4 ruling" item only when striking it at the
   end; reading it first would have surfaced the owner context
   ("standing since 2026-09-11") and the two suggested directions before
   design instead of after.
7. **Test-fixture dedup in compression tests** —
   `strings.Repeat("a", defaultCompressionMinSize+1)` now appears in 4+
   places (3 new); a `newLargePlainTextBody()` helper would keep the
   size-coupling to `defaultCompressionMinSize` in one place.

## f) Up to 50 things we should get done next

Ranked by impact. Impact/Effort/Category per item (Effort: S <30min,
M 30min-2hr, L >2hr). HARVEST note: items 1-16 are TODO_LIST-grade;
17+ are mostly ROADMAP fuel or conditional.

**Session follow-ups (this work, before it ages):**

1. ~~Push master (4 commits: the issue-#4 fix, oracle fix + seed, doc sweep).~~ done (pushed — origin/master current through v1.3.0 and beyond (verified 2026-09-23))
   ~~— Critical / S / Release (owner-gated).~~
2. ~~Watch tonight's 03:05 UTC nightly fuzz — first run over the fixed~~ done (observed green — no negotiator crashes in nightly runs since the fix)
   ~~negotiator oracle + new seed; re-dispatch or investigate if red.~~
   ~~— High / S / Quality.~~
3. After push/tag: follow-up comment on issue #4 citing the deliberate
   commit or v1.2.0 tag, replacing the dangling daemon-hash reference.
   — Low / S / Documentation.
4. ~~Run `scripts/prerelease-check.sh` on this tree — the 95% coverage gate~~ done (ran with the v1.2.0 release battery per RELEASE.md (tag CI green))
   ~~has not seen the new code+tests yet. — High / S / Quality.~~
5. ~~Benchstat before/after for `Compression` (documented 3s×5 `nix run~~ **Won't implement — window closed — v1.2.0 shipped 2026-09-16; the pre-tag comparison window is gone.**
   ~~.#bench` protocol) pre-v1.2.0. — Medium / M / Quality.~~
6. Second fuzz instance (or policy parameter) so `FuzzNegotiatorWireFormat`
   also exercises the new identity-policy branch (today's fuzz corpus only
   covers the legacy policy via `newTestNegotiator`). — Medium / S /
   Quality.
7. Extract `newLargePlainTextBody()` test fixture (dedupe the repeated
   `strings.Repeat` body). — Low / S / Cleanup.
8. Sweep all fuzz oracles for Go-normalization vs HTTP ABNF mismatches
   (`TrimSpace`/`ToLower`/`EqualFold` on header-like inputs) — the
   generalization of today's find. — High / M / Quality.
9. File the artmann-technologies-website issue to retire the compression
   guard wrapper (after v1.2.0 tag exists to point at). — Medium / S /
   Cleanup (cross-repo, owner approval).
10. ~~Re-measure per-module coverage (post-CSRF-fallback + this change) and~~ done (docs-health pass 2026-09-23, 97.5% httputil / 99.1% httpspec; FEATURES + README badge refreshed)
    ~~refresh FEATURES/README numbers. — Medium / M / Quality.~~
11. ~~Add a testable `ExampleCompression` variant showing~~ done (routed — folded into the TODO_LIST Example gap batch item)
    ~~`AbsentEncodingFirstConfigured` (README currently documents it in~~
    ~~prose only). — Low / S / Documentation.~~
12. ~~Document in README's compression section that `Vary: Accept-Encoding`~~ done (README compression section now documents Vary-on-every-response (2026-09-23); code paths verified at compression.go:295,302)
    ~~is added even on uncompressed/identity responses (cache-correctness~~
    ~~for CDNs; I confirmed the middleware adds it in both paths).~~
    ~~— Low / S / Documentation.~~

**v1.2.0 release (staged content + standing rulings):**

13. Owner ruling: B1 interpretation of the CSRF `SameSite=None` fallback
    default (audit report g1) — flip `withSecureFallback` + tests +
    migration note in one change if the B1 default was wrong.
    — High / S / Owner decision.
14. Owner ruling: MD060 table style (compact vs aligned) + buildflow
    result-cache purge for the replaying MD060 rows (audit g2).
    — Medium / S / Owner decision.
15. ~~Owner ruling: push now vs at release; cut v1.2.0 per RELEASE.md~~ done at `9b9e032`
    ~~including step 12.5 (`gh run watch` on the exact tag commit) once~~
    ~~13-14 are resolved. — Critical / M / Release (owner-gated).~~
16. ~~Final CHANGELOG `[Unreleased]` proofread — it now carries four~~ done (v1.2.0 shipped with the four-change section; changelog link gate green (re-verified 2026-09-23))
    ~~behavior changes; confirm ordering and that each bullet's~~
    ~~migration-doc link resolves. — High / S / Documentation.~~
17. ~~Update ROADMAP "Current Position" wording if needed once v1.2.0 is~~ done (docs-health pass 2026-09-23, ROADMAP Current Position rewritten for v1.2.0 + v1.3.0)
    ~~cut (it currently says v1.2.0 staged). — Low / S / Documentation.~~
18. Verify pkg.go.dev rendering for v1.2.0 after tagging (the v1.1.0
    pattern). — Low / S / Release.
19. Decide `AbsentEncodingFirstConfigured` lifecycle: permanent knob or
    pre-declared v2 removal? Record the answer in ROADMAP Non-goals +
    v1-stability notes. — High / S / Owner decision (see g1).

**Standing TODO_LIST batches (pre-existing, still open):**

20. Release-engineering: gate `release.yml` on CI success (workflow_run or
    single-publisher) so a red tag cannot publish silently.
    — High / M / Release.
21. Add a lychee link gate to CI/pre-commit (links at 0 since 2026-09-11).
    — Medium / M / Quality.
22. Fix the dev-mode lychee no-op (`exec: lychee: not found` via the nix
    fallback counts as success). — Medium / S / Bug.
23. CSRF docs polish batch (README SameSite=None prose, testable Example
    for the fallback, `ConfigureNosurfHandler` verbatim warning,
    `CSRFResponseHeaderMiddleware` cookie-path audit).
    — Medium / M / Documentation.
24. CSRF test-depth batch (opaque-URL TrustedOrigins pins, broken-list
    fallback-log assertion, `ValidateCSRF` idempotency, `needsTranslation`
    branch, `forwardedProtoFromTrustedProxy` branches, `requestScheme` TLS
    fixture, HX fuzz seed, `ErrCSRFConfig` chain asserts, CSRF benchmark
    row). — Medium / L / Quality.
25. Owner question: export `csrf.trusted_origin_invalid` as an exported
    `Code` constant? — Medium / S / Owner decision.
26. go-error-family#5 follow-through (upstream "Conditional requests" docs
    section if accepted; then sweep README + classification tables).
    — Medium / M / Feature.
27. Archived-corpus sampled audit (~2,300-marker debt, owner question from
    04-30 g3). — Low / L / Documentation.

**Ideas surfaced this session (ROADMAP fuel unless promoted):**

28. ~~httpspec: an optional compression spec (absent-header → uncompressed,~~ done (routed — added to ROADMAP Ecosystem extensions (httpspec compression spec idea))
    ~~`Vary` correctness) so consumers get the issue-#4 contract checked for~~
    ~~free via `Run()`. — Low / M / Feature.~~
29. ~~AGENTS.md one-liner: "never reference daemon commits in external~~ done (rule added to AGENTS.md Auto-Git-Commit Daemon section 2026-09-23)
    ~~artifacts" (from e2). — Low / S / Documentation.~~
30. Negotiator: consider precomputing the empty-order guard —
    `buildNegotiator` with an empty factory map is a Validate-rejected
    state; the len(order)==0 branch is defensive-only (document or
    contract it). — Low / S / Quality.
31. `nix flake check` on `--all-systems` (aarch64 warnings suppressed
    today). — Low / M / Quality.
32. buildflow full `--build-mode dev` run to confirm no NEW finding classes
    from this change (expected non-zero on documented residuals only).
    — Medium / M / Quality.
33. ~~README coverage badge re-verify after item 10's re-measure.~~ done (badge re-verified and updated to 97.5% (2026-09-23))
    ~~— Low / S / Documentation (conditional on 10).~~
34. Consider committing the remaining fuzz-workflow expectation: nightly
    run count/health alerting so five consecutive reds (the #9-#13 class)
    auto-escalates instead of auto-filing only. — Medium / M / Quality.
35. CHANGELOG: add the `AbsentEncodingPolicy` constants to the Changed
    bullet's code back-references if the owner wants API-level callouts
    (currently prose only). — Low / S / Documentation.
36. Fuzz seed inventory: confirm every fuzz target has at least one
    committed regression seed (the negotiator one landed only today by
    accident of a failure; make it policy). — Low / M / Quality.
37. Docs: single glossary entry for "OWS" with the SP/HTAB definition, so
    oracle comments can cite it instead of re-explaining.
    — Low / S / Documentation.
38. Property test `TestNegotiator_Property_*` generators: add an explicit
    zero-entries case so the empty-order branch is property-covered too.
    — Low / S / Quality.
39. CI: run `doc-snippet-refs` over docs/*.md (not just README +
    integrations) if the owner wants migration docs drift-checked too.
    — Low / S / Quality (owner preference).
40. Post-release: re-run the 33-project adoption sweep note for the new
    default (the CORS AllowPrivateNetwork entry came from dnsblockd; a
    header-less-client consumer survey would validate the identity
    default). — Low / L / Feature/ROADMAP.

## g) Three questions I cannot answer myself

1. **`AbsentEncodingFirstConfigured` lifecycle:** is the legacy knob a
   _permanent_ configuration axis, or should it be pre-declared for
   removal in v2.0 (like `RateLimit()`/`ETag()` were)? This decides
   ROADMAP Non-goals wording, whether v1-stability should mark the
   constant "Evolving", and how loudly the migration doc should nudge
   consumers off it. I tried to infer the policy from the two
   pre-declared removals, but both were interfaces deleted for
   duplication reasons, not behavior knobs — genuinely underdetermined.
2. **Cross-repo follow-up:** may I file the artmann-technologies-website
   issue to retire the compression guard wrapper now (pointing at
   master), or do you want it held until v1.2.0 is tagged so the issue
   can reference a released version? I cannot see that repo's guard tests
   from here, so I also cannot verify the retirement is safe — that
   verification belongs to that repo's session.
3. ~~**Push timing:** the only copy of this fix is 4 unpushed local commits.~~ done (resolved — master pushed continuously since; origin current through v1.3.0+)
   ~~Is pushing master (not tagging) authorized now, or is everything held~~
   ~~for the single v1.2.0 push? Everything else in section f is unblocked~~
   ~~except the things downstream of this one decision.~~

---

_Report format override note: written as `.md` per the owner's explicit
instruction (status-report skill's canonical output is a styled HTML
dashboard; the explicit `.md` path in the instruction wins). Point-in-time
snapshot — annotate, don't rewrite, per docs-health ANNOTATE mode._
