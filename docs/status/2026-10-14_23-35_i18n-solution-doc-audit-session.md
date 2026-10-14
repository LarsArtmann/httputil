# Status Report — 2026-10-14 23:35 CEST — i18n Solution Q&A + Documentation Audit (Session-Scoped)

**Session type:** Read-only Q&A. **Turns:** (1) "What is our i18n solution?" (2) "Is that well documented?" (3) this report.
**Scope discipline:** Per operator instruction, this report covers ONLY this session's work and observations — not a whole-project sweep.
**Format note:** Status-report skill default is a styled HTML dashboard; the operator's explicit `.md` path demand wins (recorded per skill spec as a one-off override, not a new default).

---

## Session Facts

- **No code changed.** Zero edits, zero test obligations triggered, zero lint runs needed. Everything below is reads and one report write.
- **Git tail at report time:** tree clean; `git log origin/master..master` empty (everything pushed); recent commits are daemon-heuristic only (`aecdf49`, `332abed`, `8e5364c`). Session-tail discipline satisfied.
- **Verification ledger (claims made in-session, closure status):**

| Claim (in-session) | Status at the time | Status now |
|---|---|---|
| "all 4 codes in both classification tables" | Verified (arch-ref lines 101–104, README:482 read directly) | Stands |
| "the error-template/docs sweep **appears to** have been done" | Hedged, NOT verified | **Now line-verified**: all four `codeLanguage*` entries in `errors.go:387–405` (errorTemplates map) and listed in `errors_templates_test.go:81–84` |
| "One honest gap: locale formatting" | Asserted from partial doc reads | **Confirmed by full read**: neither integration guide mentions formatting (dates/numbers/currency) |
| "no drift I could find" | Spot-check basis | Understated — see (b)2: TODO_LIST tracks five open i18n doc residuals my verdict didn't surface |

---

## a) FULLY DONE

1. **i18n solution answer (turn 1)** — complete and correct: negotiation in the core (`Language()` middleware, extractor chain, exact-then-primary-subtag matcher, `TagMatcher` plugin hatch, declared-spelling serving, `Content-Language` + `Vary` stamping, context accessors, CSRF/CORS/ETag ordering, four `language.*` Rejection codes, `httpspec.LanguageSpecs()`), translation delegated to go-i18n / x-text via dependency-free integration guides. Verified against `language.go`'s actual symbol table, not memory.
2. **Documentation audit verdict structure (turn 2)** — the six-layer table (code docs / README / FEATURES / arch-ref / integration guides / tests+fuzz+specs) was accurate as far as it went: every layer checked really does carry Language coverage, with real line references.
3. **Claim ledger closed (during this report)** — the hedged error-template claim is now line-verified (see facts table). Substance was right; the hedge was the only defect.
4. **Session-tail check (during this report)** — tree clean, zero unpushed commits, nothing lost to the daemon window.

## b) PARTIALLY DONE

1. **The documentation audit was a spot-check, not an audit.** Method was `rg` mention-counts plus section heads across ~10 files. Counts prove mentions exist; they don't prove the mentions are complete or current. Partially redeemed during this report: both integration guides are now fully read (go-i18n.md 57 lines, x-text.md 76 lines — I had read 70 of 133 lines in-session). Both are materially stronger than their heads suggested: bridge exactness + cache correctness + Localizer lifetime (go-i18n); confidence gating + script-class motivating case (`zh-TW` → `zh-Hant`) + concurrency contract + Parse-vs-Make (x-text).
2. **TODO_LIST cross-check was skipped in-session, done during this report.** Result: `TODO_LIST.md:15` tracks **five bounded i18n residuals** ("Language train leftovers") my "well documented" answer did not surface: (1) middleware-survey re-verify vs upstream HEADs — blocked on owner's evidence-bar answer; (2) sperrmuell-shape cookbook page (D5) — config exists in `language.go`'s doc comment, the docs page does not; (3) nightly fuzz step — DONE 2026-10-10; (4) sperrmuell guard-test pattern section for go-i18n.md; (5) consumer-audit checklist additions post-v1.6.0. Plus `TODO_LIST.md:16` (blocked cross-repo follow-ups) and `TODO_LIST.md:17` (BenchmarkLanguage harness realism). Not drift — tracked-but-pending work is not drift — but "one honest gap" understated the true open-docs inventory by five items.

## c) NOT STARTED

1. **Locale-formatting coverage gap** — dates/numbers/currency formatting is named out-of-scope in `language.go`'s doc comment ("Message catalogs, plurals, and formatting are out of scope") but neither integration guide routes the reader to the tool that DOES formatting (`golang.org/x/text/message`; go-i18n is catalogs-only). Confirmed absent after full read. Not acted on in-session; not in TODO_LIST either.
2. **The five tracked i18n residuals** (pre-existing; pointers only this session — I did not know they existed until the cross-check).
3. **HARVEST of this report into TODO_LIST.md** — deliberately not run: operator said WAIT FOR INSTRUCTIONS. Section (f) is staged as HARVEST input.

## d) TOTALLY FUCKED UP

**Nothing.** Read-only session: no breakage, no lost work, no lint/test violations, nothing unrecoverable. One credibility near-miss, documented honestly because it is exactly the failure class this repo has codified skills against:

- **The hedged claim.** "The error-template/docs sweep … appears to have been done" — published with a hedge where a one-command verification (`rg codeLanguage errors.go errors_templates_test.go`) existed. The substance turned out correct, but verify-before-claiming is a hard rule here (see the `verify-external-claims` skill), and "appears to" is a hedge doing the work of a check. Severity: LOW (answer was right), class: WRONG (method was luck).

## e) WHAT WE SHOULD IMPROVE

1. **Verify-or-flag, always.** Never publish "appears to X" when a one-command verification exists. Either run the check or say plainly "unverified: <exact check that would settle it>".
2. **Audit order: TODO_LIST first.** For any "is X well documented?" verdict, read the area's tracked-pending items BEFORE answering. Tracked residuals are the exact thing a doc audit must surface; I declared "no drift I could find" without looking at the one file that lists known doc debt for the area.
3. **Read tiny docs fully the first time.** The two integration guides total 133 lines; I read 70. Full reads are cheap and would have caught the formatting gap with certainty instead of inference.
4. **No shrug endings.** "Minor, arguably." is not a recommendation. The correct ending was the concrete fix: "add one routing line to each integration guide pointing at x/text/message".

## f) WHAT WE SHOULD GET DONE NEXT (23 items, impact-first, all session-derived)

**High impact — ship-blocking for the i18n story:**

1. **HARVEST this section into `TODO_LIST.md`** (docs-health → HARVEST), folding new rows into the existing "Language train leftovers" group — do NOT create parallel i18n rows (split-brain prevention).
2. **Add locale-formatting routing to `docs/integrations/go-i18n.md`** — one section: go-i18n owns catalogs/plurals, `golang.org/x/text/message` owns formatting; show the per-request `message.Printer` fed from `LanguageFromRequest(r)`.
3. **Mirror the same routing in `docs/integrations/x-text.md`** — x/text ships `message`; the guide currently stops at matching.
4. **Write the sperrmuell-shape cookbook page (D5)** — German-canonical + `/en` worked example with hreflang caveat pointer (TODO_LIST:15.2; the config already exists as the doc-comment example in `language.go:9-38`).
5. **Add the sperrmuell guard-test pattern section to `go-i18n.md`** — visible-missing-translation CI gate advice (TODO_LIST:15.4).
6. **Add Language() misuse classes to the consumer-audit checklist** — Vary-suppressed-while-body-varies, ETag-before-Language, extractor-ordering mistakes (TODO_LIST:15.5, gate on v1.6.0 shipping).
7. **Re-verify the middleware-survey claim vs upstream HEADs** — BLOCKED on the owner's evidence-bar answer (question g1 below; TODO_LIST:15.1).
8. **Land v1.6.0** — CHANGELOG [Unreleased] holds the language train; run `scripts/prerelease-check.sh`, RELEASE.md mechanics, single push (TODO_LIST:5 notes the tracking).

**Medium — quality and completeness:**

9. **Add declared-spelling preservation to `httpspec.LanguageSpecs()`** as a third opt-in spec (`zh-Hans` stays `zh-Hans` on the wire) — currently unit-tested only, not contract-pinned.
10. **BenchmarkLanguage static-request harness** — isolate middleware cost from httptest construction (TODO_LIST:17; eyeball rule applies).
11. **Decide close-or-accept for the two documented coverage gaps** — `language.go:167 supports` (75.0%), `language.go:180 validLanguageTag` (83.3%); reasons exist in FEATURES.md, so this is a decision, not a fix.
12. **Verify README's Language ordering prose ships a copy-paste `Chain()` snippet** (CSRF/CORS/ETag placement) — prose verified this session; snippet convenience unverified.
13. **Check per-leg extractor example coverage** — query/cookie/path-prefix extractors vs the 7 Language mentions in `example_test.go`.
14. **Check `docs/DOMAIN_LANGUAGE.md` for negotiation vocabulary** — "declared spelling", "extractor chain", first-wins semantics; FEATURES.md references the file, language-domain entries unverified.
15. **One-line confirm `FuzzParseAcceptLanguage` has a nightly workflow step** — FEATURES says 26/30 run nightly and the four named-missing don't include it; confirm rather than infer (this session's lesson applied).
16. **Confirm language benchmarks are documented in `docs/benchmarks.md`** per the documented protocol (unverified this session).
17. **Check FEATURES WORTH CONSIDERING for a locale-formatting row**; add if absent (pairs with items 2–3).

**Lower — hygiene and process:**

18. **Post-v1.6.0 cross-repo follow-ups when the tag exists** — sperrmuell-direct T24 (`cmd/server/locale.go` migration), T70 (x/text Matcher upgrade, trigger-gated), dnsblockd F96 needs-check (TODO_LIST:16; blocked, pointers only).
19. **Add the hreflang/sitemap guidance pointer into the cookbook page** when item 4 is written (same change, not separate work).
20. **Run docs-health VERIFY over all language docs at the next cadence checkpoint** — pre-tag or monthly, per the Doc-Freshness Cadence in AGENTS.md.
21. **Consider a one-line AGENTS.md note** that locale formatting is out of scope by design (the code doc comment says it; AGENTS.md's Language bullet does not) so future sessions don't re-discover the gap as news.
22. **Behavioral (no repo change): make verify-or-flag the default Q&A posture** — see (e)1.
23. **Behavioral (no repo change): TODO_LIST cross-check becomes a standard step in doc-audit answers** — see (e)2.

*Honest note: "up to 50" allows fewer — 23 grounded items beat 50 padded ones. Items 7, 8, 18 are pre-existing tracked work surfaced by the cross-check, not new discoveries.*

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF (3)

1. **Evidence bar for the middleware-survey re-verify** (TODO_LIST:15.1 explicitly waits on your §g3 answer from the 2026-10-10_04-07 report): what evidence satisfies you before an external artifact may cite the survey claim — CI-run upstream test suites, vendored example apps, or manual HEAD scans with pinned SHAs?
2. **Is the i18n area feature-complete for v1.6.0** — negotiation-only middleware with docs-only follow-ups — or is more middleware work planned (formatting helpers, additional extractor legs, header legs beyond the four)? This single answer reorders section (f) entirely.
3. **Harvest policy for Q&A findings:** should gaps discovered in answer-only sessions (like the formatting gap) be written into TODO_LIST immediately at discovery, or only at docs-health cadence? My instinct: immediately, into the existing "Language train leftovers" group — but the fleet runs parallel writers, and I don't know your preference for answer-session write scope.

---

**Next action:** WAIT FOR INSTRUCTIONS (per operator demand). HARVEST deliberately deferred.
