# Status Report — i18n / Language-Negotiation Research Session

- **Date:** 2026-10-09 22:32 CEST
- **Session scope:** exploratory research only ("How do we help with i18n and co?") — read httputil's i18n surface (none exists), study sperrmuell-direct as the real i18n consumer, survey the Go i18n ecosystem, synthesize a design direction, seed it into ROADMAP.md
- **Trigger:** owner question referencing `~/projects/sperrmuell-direct` and web research; explicit "Composability + Superb defaults for integrations" lens
- **Code impact:** zero production/test code touched. Documentation-only: `ROADMAP.md` (one new Post-v1.0-ideas bullet + `_Updated_` line). No tests were broken because none exist for this area — there was nothing to run.

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                                                                                                                                                                                                              | Evidence                                                                                   |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| a1 | Confirmed httputil has ZERO i18n/locale surface: grep for `Accept-Language`, `Content-Language`, `i18n`, `Locale` across all `.go` and all `*.md` → no hits outside doc-health noise. Also confirmed `golang.org/x/text` is NOT in go.mod (dep surface: nosurf, go-error-family, go-etag, x/time). Idea is greenfield; not in ROADMAP Non-goals either (checked).      | grep sweeps, `go.mod` read, ROADMAP §Dependency policy + §Non-goals read                    |
| a2 | Full read of sperrmuell-direct's i18n stack: `internal/domain/locale.go` (typed `Locale uint8` enum, zero-invalid, `ParseLocale` primary-subtag, `Prefix()`, `Other()`, JSON marshal), `internal/i18n/i18n.go` (embedded JSON bundles via go-i18n, `T/S/TData/SData/TPlural` helpers, German-fallback-then-key-visible policy, hand-rolled `ParseAcceptLanguage` q-parser), `cmd/server/locale.go` (`withLocale` middleware: URL prefix beats header, slug-aware 308 redirects, public-path context for switcher) | file reads in `~/projects/sperrmuell-direct`                                                |
| a3 | Verified the generic-vs-app split with evidence: HTTP-edge parts (Accept-Language parsing, prefix resolution, context injection) are library-shaped; message bundles, Locale business enum, route/slug registry, hreflang sitemap, locale-on-Quote persistence are app-specific and must stay in the app                                                                 | sperrmuell reads + its CHANGELOG 2026-10-08 i18n entries (T49 P1–P4, T67)                   |
| a4 | Ecosystem survey: `golang.org/x/text/language` API verified from pkg.go.dev (`ParseAcceptLanguage` → tags sorted by weight, q=0 dropped; `NewMatcher` first-supported-tag-is-default; `Match` returns tag/index/confidence; `PreferSameScript`). Middleware survey: gin-contrib/i18n (header-first, no headers set), itpey/echoi18n (no headers set), kaptinlin/go-i18n (query→cookie→header priority, no headers set), go-i18n itself (no HTTP layer) | pkg.go.dev fetch + agentic_fetch survey with primary-source links                           |
| a5 | Identified the market gap with a consumer bug to prove it: NO surveyed library/middleware sets `Content-Language` or `Vary: Accept-Language`, and sperrmuell-direct itself sets neither (grep across its non-test Go code: zero hits) — its header-negotiated `/api/quote` responses are cache-incorrect under any shared cache/CDN                                                                 | grep (empty result) in sperrmuell `cmd/server` + `internal`; httputil contrast: `compression.go:295` sets `Vary: Accept-Encoding` |
| a6 | Verified composability anchors in httputil the design reuses: allocation-free RFC 7231 q-value parser (`compression_qvalue.go:9-128`, reusable grammar for Accept-Language), `KeyExtractor` func-type plugin pattern (`ratelimit_keyed.go:45-75`), precompiled-negotiator pattern (`compression_negotiator.go:9-28`), validate-and-log constructor convention, `Vary` Add precedent, Nonce context pattern                                                                    | file reads in httputil                                                                      |
| a7 | Confirmed the design respects the dependency policy: full BCP 47 matching (x/text `Matcher`) stays a user-supplied adapter/plugin surface, not a core dep — matches the documented "plugin interfaces + docs/integrations examples, not core dependencies" policy and sperrmuell's own "region subtags ignored" semantics (primary-subtag default matcher is sufficient and zero-dep) | ROADMAP §Dependency policy quote; sperrmuell `ParseLocale` comment                          |
| a8 | Synthesis delivered + seeded as a raw idea in ROADMAP.md "Post-v1.0 ideas" (full proposed shape: `LanguageExtractor` plugin mirroring `KeyExtractor`, first-wins `LanguageExtractorChain`, precompiled primary-subtag matcher, Content-Language + Vary by default with opt-outs, `LanguageFromContext` mirroring Nonce, `language.*` error domain, explicit out-of-scope list, sources) + `_Updated_` line annotated                    | `git diff --stat` shows ROADMAP.md +2/−1; entry greppable ("Language-negotiation" ×2)       |
| a9 | Self-check of every research claim against code before writing it down (the "verify-external-claims" gate): no-i18n-in-httputil, no-x-text-dep, no-Vary-in-sperrmuell, KeyExtractor/negotiator/q-parser precedents, x/text signatures — each was re-verified in-session, not assumed                                                                                              | session grep/bash transcript                                                                |

## b) PARTIALLY DONE

| #  | Item                                                                                     | What works                                                                                                                                              | What remains                                                                                                                                                                                                                     | Effort |
| -- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| b1 | sperrmuell-direct i18n depth                                                             | Core files read end-to-end (`i18n.go`, `locale.go` middleware, `domain/locale.go`); detection-order rationale and redirect semantics understood                          | `guard_test.go` (210 lines, CI-makes-missing-translation-impossible pattern) read only in outline; its guard-test pattern should inform Language()'s test strategy but was not extracted into the design                                          | S      |
| b2 | "and co" scope of the question                                                           | Content-Language, Vary, Accept-Language negotiation, x/text-vs-handrolled matching, hreflang placement covered                                                          | Adjacent HTTP-edge i18n concerns left unexamined: `Accept-Language` × ETag/conditional-request interaction (cache-key explosion), Vary dedup semantics when Compression and Language both Add to `Vary` (two `Add` calls → two entries vs one comma-joined field), `Accept-Charset` (dead, deliberately skipped) | M      |
| b3 | MiddlewareStack integration story                                                        | Design notes where Language() context must sit (before handlers, after Recovery per Validate()'s only rule)                                                              | No ordering guidance for Language() vs CORS (preflight OPTIONS carries no locale — skip negotiation?) vs CSRF (error pages are localized in sperrmuell via `i18n.S` on CSRF errors — so Language must be OUTER to CSRF there, contradicting the innermost placement sperrmuell chose for its own `withLocale`) | S      |
| b4 | Evidence base breadth                                                                    | One deep consumer (sperrmuell) + one light reference (dnsblockd exists in ROADMAP as another consumer but was not checked for i18n needs)                               | Second-consumer evidence (dnsblockd, others from the 45-consumer inventory) and an upstream gh-issues scan for existing i18n feature requests both missing                                                                                       | S      |
| b5 | Design documentation                                                                     | Full shape lives in the ROADMAP bullet (deliberate: raw-idea stage per docs-health conventions)                                                                          | No `docs/planning/` design note yet; a future implementation session will need the refined config/Validate/extractor-contract spec — the ROADMAP entry is a summary, not a spec                                                                  | M      |

## c) NOT STARTED

| #  | Item                                                                                                                     | Why it did not start                                                        |
| -- | ------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------- |
| c1 | Implementation of `language.go` (`Language()` middleware, extractor chain, matcher, context accessors, `language.*` error domain) | Exploration-mode session; post-v1.0 idea requires owner refinement into TODO_LIST first (ROADMAP governance) |
| c2 | TODO_LIST refinement / FEATURES.md inventory rows / CHANGELOG entry                                                        | Gated on c1 decision                                                          |
| c3 | Fuzz target (`FuzzParseAcceptLanguage`), benchmarks, execution-probe tests, httpspec `Localizes` spec                       | Gated on c1                                                                   |
| c4 | `docs/integrations/x-text.md` (bring-your-own BCP 47 matcher) and `docs/integrations/go-i18n.md` (sperrmuell-shaped wiring)  | Gated on c1                                                                   |
| c5 | sperrmuell-direct standalone fix: add `Vary: Accept-Language` + `Content-Language` to its API responses (independent of library adoption) | Different repo; no cross-repo write mandate this session                      |
| c6 | `art-dupl` re-baseline after any future `language.go` lands (current baseline claims 0 groups at `-t 2`)                     | Nothing added yet                                                             |

## d) TOTALLY FUCKED UP

**Nothing.** No production code, no tests, no git state harmed; the single file edit (ROADMAP.md) succeeded on the second attempt only because the edit tool correctly refused an un-viewed file — that was tool-guard compliance, not a mistake, and it was recovered immediately. Honest near-misses worth recording rather than hiding:

| #  | Near-miss                                                                                      | Reality check                                                                                     |
| -- | ----------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| d1 | First ROADMAP edit attempted without a prior View of the file                                    | Tool refused; re-read then clean landing. Zero damage; lesson is procedural only                     |
| d2 | Date inconsistency risk: env clock said 2026-10-08, repo evidence (ROADMAP `_Updated_`, sperrmuell status reports) says 2026-10-09; the new bullet cites "2026-10-09" | Resolved in favor of repo convention; the `date` run for this report confirms 2026-10-09 22:32 — citation stands |

## e) WHAT WE SHOULD IMPROVE

| #  | Improvement                                                                                             |
| -- | --------------------------------------------------------------------------------------------------------- |
| e1 | **What I forgot in the design:** middleware-ordering guidance (b3) — the CSRF-localization conflict above is a real API-design decision (Language likely needs an explicit "outer to CSRF" recommendation or per-handler negotiation) and it is absent from the ROADMAP entry |
| e2 | **What I forgot in the design:** Vary header composition semantics when both Compression and Language are chained (b2) — must be specified or the superb-default becomes a footgun (duplicate/incorrect Vary) |
| e3 | **What I could have done better:** evidence breadth (b4) — one consumer is a data point, two is a pattern; the consumer-audit program (ROADMAP) already lists the machinery to check dnsblockd et al. cheaply |
| e4 | **What I could have done better:** extract sperrmuell's guard-test lesson (missing translation must be visible, CI-enforced) into Language()'s test plan instead of noting it in passing (b1) |
| e5 | **What I could still improve:** run the survey claims through a second independent source pass before anything upstream-shaped cites them (the "no library sets Content-Language/Vary" claim rests on one agentic_fetch pass over five libraries); verify-before-filing applies if this ever becomes an upstream issue or blog post |
| e6 | **What I could still improve:** the "just googling" leg was thin on prior art outside Go (Django LocaleMiddleware precedence, Rails I18n locale conventions) — the gin precedence-bug claim is cited via a Django reference inside the survey; a first-hand Django doc check would harden it |
| e7 | Process: the research produced exactly one durable artifact (ROADMAP). A `docs/planning/` design note would have made the session's thinking load-bearing for the implementation session instead of compressed into one bullet (b5) — deliberate raw-idea-stage tradeoff, but the next session should not refine the design from the bullet alone |

## f) NEXT — up to 50 things to get done

Sequenced; 1 gates 2–25, which gate 26–50. (IDs f1…f50 for cross-referencing.)

**Decision gates**

| #   | Task                                                                                                                          |
| --- | ------------------------------------------------------------------------------------------------------------------------------- |
| f1  | Owner decides: promote the ROADMAP bullet into a TODO_LIST work item (implement `Language()` in httputil core) vs keep as integrations-doc pattern only — everything below gates on this |
| f2  | Owner rules the Vary default: always emit `Vary: Accept-Language` when the middleware is chained (RFC-strict) vs only when the header leg can influence output (minimalist) — same lifecycle shape as the open `AbsentEncodingFirstConfigured` question |
| f3  | Owner rules Content-Language shape: single canonical tag vs multi-tag lists (RFC allows both)                                     |
| f4  | Owner rules the 406 question: q=0-only / no-match requests default silently, or a `RejectOnNoMatch` config axis exists (and what the error domain is) |
| f5  | Owner rules whether the negotiated confidence (x/text-style) is exposed in context or only the canonical tag                      |

**Refinement (design note → TODO_LIST)**

| #   | Task                                                                                                                          |
| --- | ------------------------------------------------------------------------------------------------------------------------------- |
| f6  | Write `docs/planning/2026-10-*_language-negotiation-design-note.md` from the ROADMAP bullet (config shape, Validate rules, extractor contract, matcher contract) |
| f7  | Fix the b3 gap in the design: ordering rules vs CSRF (localized error pages) and CORS preflight (skip OPTIONS?), possibly a documented recommended chain |
| f8  | Fix the b2 gap in the design: Vary composition semantics when chained with Compression; dedupe or document multi-Add behavior              |
| f9  | Decide middleware/identifier naming: `Language()` vs `Locale()`; `language.go`; exported surface list for docs/v1-stability.md            |
| f10 | Define `LanguageExtractor` + built-ins (`FromAcceptHeader`, `FromQuery`, `FromCookie`, `FromPathPrefix`) exact signatures and zero-value behavior |
| f11 | Define `LanguageExtractorChain` semantics: first-wins, nil-entry rejection (`stack.name_empty` precedent), empty-chain behavior (header-only default?) |
| f12 | Define the matcher contract: primary-subtag canonicalization rules, `*` handling, q=0 drop, empty header, tie-break order — pin each as an execution-probe test claim (AGENTS.md zero-value rule) |
| f13 | Define the `TagMatcher` plugin escape hatch signature + fallback behavior when it returns not-ok                                     |
| f14 | Draft the `language.*` error domain: `Domain` constant, sentinels (`language.supported_empty`, `language.default_unsupported`, `language.extractor_nil`…), families (all Rejection?) |
| f15 | Plan the documented error-code sweep in one change: errorTemplates map, `allHTTputilErrorCodes`, domain test, FEATURES.md, docs/v1-stability.md, README/architecture-reference classification tables, erraudit sentinel-count note in AGENTS.md |

**Implementation (post-f1)**

| #   | Task                                                                                                                          |
| --- | ------------------------------------------------------------------------------------------------------------------------------- |
| f16 | Implement `language.go`: config + Validate + validate-and-log constructor via `validateConfig`                                     |
| f17 | Implement precompiled matcher over `compression_qvalue.go` grammar (config-time priority compile, per-request match — negotiator pattern) |
| f18 | Implement context accessors (`LanguageFromContext`, `WithLanguage` for tests) mirroring Nonce                                          |
| f19 | Implement response-header writes: `Content-Language` + `Vary` with opt-outs; honest-silence rules for post-commit writes                |
| f20 | Tests: standalone per-case (no table-driven — repo convention), name = claim, `t.Parallel()` everywhere                                |
| f21 | Tests: Accept-Language grammar edges (whitespace, casing, `*`, q=0, >3 decimals, empty, garbage) reusing compression_qvalue_test patterns       |
| f22 | Tests: mutation-check preservation tests (extraction chain order actually flips results when reordered — the name says "beats", the body must prove it) |
| f23 | Fuzz target `FuzzParseAcceptLanguage` with round-trip/bounded invariants (28th fuzz target; consider corpus seed)                       |
| f24 | Benchmark `BenchmarkLanguage` vs Compression negotiator cost; docs/benchmarks.md protocol entry if it joins `nix run .#bench`              |
| f25 | `go test -race -count=10 ./...` + golangci-lint (~70 linters, 0 issues) + `nix fmt` + art-dupl re-measure after landing                   |

**Integration + docs (post-f1)**

| #   | Task                                                                                                                          |
| --- | ------------------------------------------------------------------------------------------------------------------------------- |
| f26 | `docs/integrations/x-text.md`: 5-line `TagMatcher` adapter example for `language.NewMatcher` users                                     |
| f27 | `docs/integrations/go-i18n.md`: wiring `Language()` context into `i18n.NewLocalizer(bundle, tag)`                                       |
| f28 | httpspec opt-in `Localizes` spec (Content-Language present, Vary correctness) — the "consumers get the contract checked for free" pattern     |
| f29 | Example in `example_test.go`: self-contained, deterministic `// Output:` (testableexamples gate)                                        |
| f30 | MiddlewareStack docs: recommended position in `Chain()`/stack ordering guidance                                                          |
| f31 | ETag-interaction caveat documented (language-varies responses + conditional requests share validators — cache-key guidance)                 |

**Consumer-side**

| #   | Task                                                                                                                          |
| --- | ------------------------------------------------------------------------------------------------------------------------------- |
| f32 | sperrmuell-direct: add `Vary: Accept-Language` + `Content-Language` to `/api/*` responses NOW (independent of f1 — the bug exists today)    |
| f33 | sperrmuell-direct migration guide: replace hand-rolled `ParseAcceptLanguage` + `withLocale` header leg with extractor chain; keep `domain.Locale` mapped at one edge |
| f34 | sperrmuell-direct: extract its guard-test pattern (visible-missing-translation CI gate) as reusable advice in the integrations doc              |
| f35 | dnsblockd i18n-needs check (second consumer evidence; cheap via the consumer-audit program)                                            |
| f36 | gh-issues scan: any upstream i18n feature requests to fold into the design                                                             |
| f37 | Consumer-audit program: add Language() misuse classes to the audit checklist once shipped                                              |

**Hardening / hardening-adjacent**

| #   | Task                                                                                                                          |
| --- | ------------------------------------------------------------------------------------------------------------------------------- |
| f38 | Harden survey claim e5 with a second source pass before any external citation                                                           |
| f39 | First-hand Django LocaleMiddleware precedence check to firm up the "documented precedence bug class" claim (e6)                            |
| f40 | Decide whether `Accept-Charset` deserves an explicit Non-goals entry (dead header; cheap to record why it is ignored)                      |
| f41 | Decide whether an hreflang/`Link` alternates helper is WORTH CONSIDERING or gets a Non-goals entry (app route-registry lane — my recommendation: Non-goal) |
| f42 | Decide `LanguageWhen()` route-conditional variant now or later (NonceMiddlewareWhen precedent — later)                                   |
| f43 | Post-implementation: ROADMAP bullet updated/collapsed into the shipped CHANGELOG entry (docs-health harvest rule)                          |
| f44 | Post-implementation: AGENTS.md "Non-Obvious Behaviors" entry (Vary semantics, ordering rule, matcher defaults)                            |
| f45 | Post-implementation: coverage measurement with documented gaps (FEATURES.md per-methodology rule)                                        |
| f46 | Post-implementation: v1-stability surface classification decision (Evolving vs frozen) — owner call                                       |
| f47 | Cross-check ROADMAP entry wording against the final design so the raw idea and the spec never diverge silently                             |
| f48 | If sperrmuell fix (f32) lands: verify against its smoke gates (`verify-gates.sh`), not just its tests                                      |
| f49 | Keep ROADMAP `_Updated_` line accurate on every subsequent edit of this idea                                                             |
| f50 | Owner review of this report; strike/annotate resolved items per docs-health inline-annotation convention                                   |

## g) QUESTIONS FOR THE OWNER (not answerable from the repos)

| #  | Question                                                                                                                                                                                                                                  |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| g1 | **Scope call:** implement `Language()` as a core httputil middleware (post-v1.0 additive, new public surface), or ship it only as a plugin-shape pattern doc (`docs/integrations/`)? This single decision gates f6–f31 and the v1-stability implications. |
| g2 | **Vary semantics ruling:** must `Vary: Accept-Language` be emitted whenever the middleware runs (RFC-strict, safe under CDN even with URL/cookie extraction), or only when the Accept-Language leg can influence the response (minimalist, fewer surprise cache-misses in caching setups)? My lean: RFC-strict default with an opt-out, but this is the same class of lifecycle call as `AbsentEncodingFirstConfigured`, which is an open owner question. |
| g3 | **Cross-repo authority:** may a follow-up session fix sperrmuell-direct's missing `Vary`/`Content-Language` on `/api/*` independently (f32), before/independent of any httputil work — and if yes, with its own verify-gates run in that repo?                                                        |

---

_Arte in Aeternum_
