# Status Report — httputil Consumer Lint Audit

- **Date:** 2026-10-09 02:29 CEST
- **Session scope:** consumer audit of `github.com/larsartmann/httputil` across `~/projects`, filings written to `docs/review/lint/` (000-index + 14 filings)
- **Trigger:** `who-uses` inventory (45 direct+indirect httputil consumers, 49 go-etag consumers) + directive to "review and write down EVERYTHING consumers of httputil do wrong"
- **Format note:** skill default is styled HTML; the operator explicitly demanded `.md` at a `docs/status/` path — `.md` shipped per that instruction (one-off override, not propagated into the skill).

---

## a) FULLY DONE

| Item | Evidence |
|---|---|
| Consumer inventory: mapped all who-uses entries to on-disk repos (incl. the non-obvious `games/{KeyCountdown,SEC,SwettySwipperWeb}`, `sales-landing-page/cloud-run`, `sec`=games/SEC) plus 8 code-level consumers absent from who-uses (`ci-siblings`, `crm-exec-stage`, `e-invoicing`, `reports`, `plugmarket`, `go-plugin-mvp`, `go-policy-dsl`, `games/*`) | corpus build 2026-10-08 ~23:30, 1,233 `httputil.*` call-site lines |
| Ground-truth reads of httputil semantics used as the judging standard: `cors.go` (`resolveOrigin` origin echo, `matchWildcardOrigin` `*.`-prefix-only, `DefaultCORSConfig` allow-all, `Validate` credentials warning), `csrf.go` (`CSRFConfig` field defaults, `Secure: false` default), `server.go` (Shutdown applies `ShutdownTimeout` only when ctx has no deadline), `maxbodysize.go` (`MaxBodySize(int64)`), CHANGELOG v1.2.0–v1.4.2 | file reads this session |
| 14 filings + index written to `docs/review/lint/`, every finding verified at `file:line` before filing (no unverified claims filed) | `docs/review/lint/000-index.md` … `014-nonce-response-caching.md` |
| Citation verification loop: re-checked uncertain line numbers and corrected 8 of them (`bundle.go:140`, `testing:54`/`reports:76`, AI-Speed-Test `:1027`, browser-history `:168`, games/SEC `:44`, artmann `:187`, error.go `:63-64`, `matchWildcardOrigin` `:198`) | grep sweeps + edits this session |
| Version-staleness map of every consumer pin (v1.2.0: CV+CV/platform, games/SEC, index, ci-siblings mirror; v1.4.0: 6; v1.4.1: 28; v1.4.2: 5) with changelog-based impact analysis | filing 011 |
| False positives eliminated before filing: local packages named `httputil` (dnsblockd, standard-bug-tracking-schema, picoclaw, ai-task-prioritizer, go-health-dashboard, templ) excluded; `httputil.Compress` hit identified as comment-only; KeyCountdown `new(expr)` confirmed as Go 1.26 `new(expr)`, not a bug | rg checks this session |
| In-fleet positive patterns identified and documented as copy-sources (CV trusted-proxy extractor, webphone lifecycle + flip rule, InboxClean TrustedProxies, nsfw-classifier BehindProxy flag, file-and-image-renamer chain order, DiscordSync default Cache-Control, blog MetricsRecorder+PathFunc) | filings 005/006/008/010/012, index |

## b) PARTIALLY DONE

| Item | What works | What remains | Effort |
|---|---|---|---|
| Coverage depth | Every project's httputil call sites reviewed with context; deep full-chain review done for ~15 consumers (cqrs-htmx, ci-siblings, go-appkit, CV, games×3, webphone, InboxClean, DiscordSync, storbi, blog, ChastityAPI, GmbH, Rolls, ksef/e-invoicing) | The "reviewed without a filing" list in the index overstates depth for low-usage consumers (mr-sync, Zlota44 handlers, RedditParse beyond its CORS block, GermanBA beyond its chain, overview beyond server config) — call-site-level only | M |
| Unresolved claims flagged in thinking but not closed | 3 claims remain unverified: overview's `s.rateLimit` definition never located; storbi's `MaxRequestBodySize` constant value never checked; GmbH's actual cookie-setting site never located (SameSite claim rests on a negative grep of `GmbH/server`) | Locate each, update or confirm the filings | S |
| Filing 011 index wording | Versions and missing fixes are correct | The `index` project entry implies the deployment resolves v1.2.0; its go.mod line is only the recorded minimum — with go-appkit v1.4.1 in the graph, MVS likely resolves higher. Needs a resolved-version check (`go list -m`) and a wording correction | S |
| ci-siblings drift | Version pin (v1.2.0) and the extractor divergence (`KeyExtractorFromRemoteAddr`) documented | Extent of overall drift vs cqrs-htmx main never quantified (full diff not run) | M |
| go-etag side of the data | The user's second `who-uses` run (49 consumers) is in hand | Not audited at all — out of the stated httputil scope, but the command output was provided and unused | L |
| Evidence preservation | All citations are inline in the filings | The working corpus (`/tmp/httputil-lint/`, per-project call-site dumps) is ephemeral; a future re-audit must rebuild it | S |

## c) NOT STARTED

- **Any remediation in consumer repos** — the audit is read-only evidence; no fix PRs/commits opened anywhere (deliberate: waiting on operator instruction).
- **Harvest of section (f) into `TODO_LIST.md`/`ROADMAP.md`** (docs-health HARVEST) — not run; the 50 items below would die in this timestamped file without it.
- **go-etag consumer audit** (49 consumers), **httpspec consumer audit**, **server_timing sub-module consumer audit** — all unstarted.
- **Recurring-audit tooling** — no committed script for who-uses + corpus + pattern pack; this session was ad-hoc bash.
- **CI integration** — no job detects "new consumer appears" or "new httputil misuse class" between audits.

## d) TOTALLY FUCKED UP

Nothing I wrote breaks the repo (docs-only change, no code touched). The honest fuck-ups are in the audit's own quality:

1. **The index's clean-list overstates review depth.** "Reviewed at call-site level" is true but reads as stronger than it is; for ~10 consumers I read only the lines matching my pattern pack plus a few lines of context. A misuse that doesn't mention any tracked symbol (e.g., misusing `WriteJSON` status codes, wrong `Timeout` nesting) in those repos would have been missed. Severity: medium — the index is the document people will trust.
2. **Severity ratings are deployment-blind.** 001/003/004/006 severities assume deployment topologies (TLS-terminating proxy present? XFF stripped? internet-facing?) that I cannot see from the repos and did not verify from deploy manifests either. GmbH "High" and SwettySwipperWeb vote-guard "High" are conditional on topologies I inferred, not observed. Root cause: no deployment-context data source consulted (k8s/cloud-run YAMLs exist in some repos and were not read).
3. **Filing 011's `index` entry is probably wrong in the direction that matters** (recorded-minimum pin presented as resolved version). Known, not yet corrected.
4. **Two range citations in 009 (`error.go:55-66`) are inferred, not re-verified** after the final edit round (the load-bearing `:63-64` echo citation is verified). Low.
5. **The audit's pattern pack had blind spots I knew about and did not cover:** `WriteJSON` call-site correctness (55 uses), `Timeout` nesting, `Decompression` config misuse (zero-value `MaxDecompressionSize` semantics), `ResponseRecorder.Status()==0` handling in consumer metrics (DiscordSync, PapDashboard record status `"0"` labels), `Metrics` nil-Recorder construction on pre-1.4.2 pins. None filed; none swept.

## e) WHAT WE SHOULD IMPROVE

1. **Make the consumer audit a committed script, not session bash.** The 6-step pipeline (who-uses → go.mod pins → rg corpus per project → pattern pack → report skeleton) is fully deterministic and would turn a half-day manual audit into a 5-minute re-run. Suggested home: `scripts/consumer-audit/` in httputil.
2. **Citation verification should be mechanical.** I hand-verified line numbers and still shipped 8 stale ones. A `file:line` extractor that re-opens each citation and diffs the cited line against the quoted snippet would catch all of them.
3. **Severity needs a deployment-context input.** A short per-repo fact sheet (internet-facing? proxy? TLS termination? cookie auth?) — even operator-maintained YAML — would let filings carry real severities instead of conditional ones.
4. **Fix-in-the-same-session.** All flagged repos are first-party (Lars-owned). The audit → fix → annotate loop could close inside one session per repo instead of leaving 14 open filings.
5. **Machine-readable verdicts.** A per-consumer YAML verdict (uses: [...]; findings: [001, 005]; clean: false) would make future audits diffable and CI-able.
6. **Scope boundary explicitness.** The task said "httputil consumers"; the go-etag output rode along unused. Either commit to auditing both supply-side libs in one pass or say so in the report header.

## f) TOP 50 NEXT TASKS (ranked; Impact / Effort / Category)

| # | Task | Impact | Effort | Category |
|---|---|---|---|---|
| 1 | GmbH: replace `AllowedOrigins:["*"]`+`AllowCredentials` with explicit origins (001) | Critical | S | Bug |
| 2 | GmbH: adopt `httputil.CSRFMiddleware` + SameSite/Secure session cookie (003) | Critical | M | Bug |
| 3 | CV: fix `csrfConfig` — `Secure = cfg.IsProduction()`, delete dead branch; build invalidation config from same source (003) | High | S | Bug |
| 4 | cqrs-htmx `setup`: secure nil-CSRF default (env-derived `Secure`) (003/004) | High | M | Bug |
| 5 | cqrs-htmx usermgmt: fill `MaxKeys`/`TTL` defaults in `applyConfigDefaults` (005) | High | S | Bug |
| 6 | SwettySwipperWeb: trusted-proxy-gate the vote guard + all three limiters (006) | High | M | Bug |
| 7 | testing+reports: `context.WithoutCancel` shutdown fix (008) | High | S | Bug |
| 8 | CV: sweep httputil+platform module to v1.4.2 (011) | High | S | Cleanup |
| 9 | blog: derive `AllowCredentials` from config; fail fast on wildcard at config load (001) | High | S | Bug |
| 10 | cqrs-htmx `RecommendedSecurityMiddleware`: add `Cache-Control: no-store` for nonce'd HTML — fixes every consumer at once (014) | High | M | Bug |
| 11 | DiscordSync: MaxKeys/TTL + delete the false "zero-value production defaults" comment (005) | Medium | S | Bug |
| 12 | storbi: named MaxKeys/TTL constants; delete dead `https://*` entries (002/005) | Medium | S | Bug |
| 13 | games/SEC: CSRF `Secure`+`TrustedProxies`; stop logging CSRF header values; limiter caps (003/004/005) | Medium | S | Bug |
| 14 | timesheets: CSRF `Secure` + TrustedProxies wiring (003/004) | Medium | S | Bug |
| 15 | Zlota44: CSRF `Secure` flag; confirm proxy topology covers loopback-only trust (003/004) | Medium | S | Bug |
| 16 | Standup-Killer: limiter caps + trusted-proxy keying (005/006) | Medium | S | Bug |
| 17 | artmann: MaxKeys/TTL on contact limiter (005) | Low | S | Bug |
| 18 | browser-history: correct the "proxy-aware" comment; add trust gate for XFF keying (006) | Medium | S | Bug |
| 19 | crush-daily: swap Nonce inside SecurityHeaders + add order-regression test (010) | Low | S | Bug |
| 20 | bank-sync: same Nonce/SecurityHeaders swap (010) | Low | S | Bug |
| 21 | Rolls: replace custom Recovery with `httputil.Recovery`; drop panic-driven `HTTPError` control flow (009) | Medium | M | Bug |
| 22 | Rolls: rename `internal/pkg/httputil` wrapper (009) | Low | S | Cleanup |
| 23 | PapDashboard: replace hand-rolled CORS with `httputil.CORS` (fixes broken preflights) (012) | Medium | S | Bug |
| 24 | PapDashboard: migrate metrics to `httputil.Metrics`+`MetricsRecorder` (012) | Low | M | Cleanup |
| 25 | ksef-sandbox+e-invoicing: adopt `httputil.Recovery`/`SecurityHeaders`; drop hand-rolled twins (012) | Low | S | Cleanup |
| 26 | dynamic-markdown-site: `httputil.SecurityHeaders` swap; ClientIP trust gate (012/007) | Low | S | Cleanup |
| 27 | KeyCountdown: replace IP+UA "session IDs" with server-issued sessions (007) | Medium | M | Bug |
| 28 | cqrs-htmx requestmeta: mark `client_ip` as client-claimed until proxy-attested (007) | Low | S | Quality |
| 29 | GermanBA: swap hand-rolled security headers for `httputil.SecurityHeaders` (012) | Low | S | Cleanup |
| 30 | ChastityAPI: environment-driven CORS origins instead of `DefaultCORSConfig` (002) | Medium | S | Bug |
| 31 | sales-landing-page/cloud-run: document intentional allow-all (002) | Low | S | Documentation |
| 32 | ci-siblings: decide auto-refresh vs archive; execute (011/013) | Medium | M | Cleanup |
| 33 | ksef↔e-invoicing and crm↔crm-exec-stage: consolidate shared integration surfaces (013) | Low | L | Cleanup |
| 34 | Version sweep: all v1.4.0/v1.4.1 consumers → v1.4.2 (011) | Medium | M | Cleanup |
| 35 | template-arch-lint: track latest tag; write template-freshness policy (011) | Low | S | Documentation |
| 36 | games/* v1.4.0 consumers → v1.4.2 (011) | Low | S | Cleanup |
| 37 | `index`: check resolved httputil version (`go list -m`); correct filing 011 wording | Medium | S | Bug |
| 38 | httputil upstream: context-aware `Logging` variant (DiscordSync's workaround is the spec) | Medium | M | Feature |
| 39 | httputil upstream: Range-aware compression (skip 206 / `Vary` handling) | Medium | M | Feature |
| 40 | httputil upstream: boot-time log when CSRF enabled with nil `TrustedProxies` (opt-in) | Low | S | Feature |
| 41 | httputil upstream: libraryize the trusted-proxy key extractor (`KeyExtractorFromTrustedClientIP(cidrs)`) — the CV pattern | High | M | Feature |
| 42 | Harvest section (f) into `TODO_LIST.md`/`ROADMAP.md` via docs-health | High | S | Documentation |
| 43 | Commit `scripts/consumer-audit/` (the 6-step pipeline as code) | Medium | M | Quality |
| 44 | Re-run audit post-fixes; annotate each filing with its fix commit (docs-health ANNOTATE convention) | Medium | M | Documentation |
| 45 | Close the 3 unverified claims (overview `s.rateLimit`, storbi body-limit constant, GmbH cookie site) and update filings | Medium | S | Quality |
| 46 | Depth pass: full-file review of the ten shallow-reviewed "clean" consumers | Medium | L | Quality |
| 47 | Audit go-etag's 49 consumers (second who-uses output) | Medium | L | Quality |
| 48 | Audit httpspec + server_timing consumer adoption/correctness | Low | M | Quality |
| 49 | Machine-readable verdict file (per-consumer YAML) for audit diffing | Low | S | Quality |
| 50 | CI job: run who-uses monthly; flag new consumers + new httputil-introducing PRs for audit | Low | M | Quality |

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Which of these deployments are actually internet-facing, and which sit behind TLS-terminating proxies that strip vs append `X-Forwarded-For`?** I looked for deploy manifests and proxy configs in the repos (found none authoritative; Cloud Run/nginx reality lives outside the code). The answer re-ranks 001, 003, 004, 006 — especially SwettySwipperWeb's vote guard and GmbH's auth API.
2. **What is `ci-siblings` for, going forward — should it auto-refresh from cqrs-htmx tags, or is it a frozen regression mirror to archive?** Its name and the v1.2.0 freeze suggest intent I cannot infer from the repo.
3. **Is this audit read-only evidence, or do you want fix branches/PRs opened per consumer repo now** (and if so: severity order across repos, or repo-by-repo until clean)?

---

*Point-in-time snapshot. Section (f) is the docs-health HARVEST input — do not let it die here. Filings live in `docs/review/lint/` (000-index + 014); corrections to evidence go through the filings, not this report.*
