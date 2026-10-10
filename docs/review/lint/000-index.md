# httputil Consumer Lint — Filing Index

- **Date:** 2026-10-08
- **Scope:** every project in `~/projects` that imports `github.com/larsartmann/httputil` (45 consumers in the `who-uses` output plus the code-level extras: `ci-siblings`, `crm-exec-stage`, `e-invoicing`, `games/*`, `go-plugin-mvp`, `plugmarket`, `reports`, `sales-landing-page/cloud-run`)
- **Method:** `who-uses` inventory → corpus of all 1,233 `httputil.*` call sites (rg over non-vendor, non-testdata Go files) → per-pattern review against httputil's documented semantics (AGENTS.md "Non-Obvious Behaviors", godoc, CHANGELOG v1.2.0–v1.4.2) → manual verification of every claim at the cited file:line
- **Status convention:** every filing is point-in-time evidence (paths and line numbers as of 2026-10-08). Fix a filing by linking the fixing commit; do not edit the evidence retroactively.

## Filings

| #   | Filing                                                                                                       | Severity    | Consumers                                                                                                                                         |
| --- | ------------------------------------------------------------------------------------------------------------ | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| 001 | [CORS wildcard origin echo + `AllowCredentials`](001-cors-credentials-wildcard-echo.md)                      | High        | GmbH, blog                                                                                                                                        |
| 002 | [CORS allow-all configs on production APIs](002-cors-allow-all-configs.md)                                   | Medium      | ksef-sandbox, e-invoicing, storbi, ChastityAPI, sales-landing-page, middleware-showcase example                                                   |
| 003 | [Zero-value `CSRFConfig` (non-Secure cookies) in production paths](003-csrf-zero-value-nonsecure-cookies.md) | High/Medium | CV (logic bug), cqrs-htmx setup default, games/SEC, timesheets, Zlota44, GmbH (absent)                                                            |
| 004 | [CSRF behind TLS termination without `TrustedProxies`](004-csrf-trusted-proxy-tls-termination-gap.md)        | Medium      | cqrs-htmx setup, timesheets, games/SEC, Zlota44 (partial), CV (partial)                                                                           |
| 005 | [Rate limiters without `MaxKeys`/`TTL` (unbounded key growth)](005-ratelimit-unbounded-key-growth.md)        | Medium      | cqrs-htmx (+ci-siblings), DiscordSync, storbi, SwettySwipperWeb, games/SEC, Standup-Killer, artmann                                               |
| 006 | [Rate-limit keys from spoofable client-IP headers](006-ratelimit-spoofable-client-ip-keys.md)                | High        | SwettySwipperWeb (vote guard), cqrs-htmx usermgmt, browser-history, DiscordSync, storbi, games/SEC, Standup-Killer, ci-siblings (RemoteAddr side) |
| 007 | [`ClientIP` for security decisions / trusted log fields](007-clientip-untrusted-header-consumers.md)         | Medium/Low  | games/KeyCountdown, Rolls-Royce, DiscordSync, cqrs-htmx requestmeta, dynamic-markdown-site                                                        |
| 008 | [Shutdown paths that cancel themselves or never drain](008-server-shutdown-cancelled-context.md)             | Medium      | testing, reports, AI-Speed-Test, overview, github-local-sync                                                                                      |
| 009 | [Custom `Recovery` divergence + panic-driven control flow](009-recovery-divergence-panic-control-flow.md)    | Medium      | Rolls-Royce-mtuGoHelpCenter-golang                                                                                                                |
| 010 | [`Nonce` composed outside `SecurityHeaders`](010-nonce-securityheaders-ordering.md)                          | Low         | crush-daily, bank-sync                                                                                                                            |
| 011 | [Stale httputil pins (v1.2.0–v1.4.1 of v1.4.2)](011-stale-httputil-versions.md)                              | Medium      | CV, games/SEC, index, ci-siblings, 6× v1.4.0, 28× v1.4.1                                                                                          |
| 012 | [Hand-rolled middleware twins (incl. broken PapDashboard CORS)](012-hand-rolled-middleware-parallels.md)     | Low         | PapDashboard, ksef-sandbox/e-invoicing, german-business-contract-automation, dynamic-markdown-site                                                |
| 013 | [Twin repositories drifting independently](013-twin-repos-drift.md)                                          | Low         | e-invoicing↔ksef-sandbox, crm↔crm-exec-stage, cqrs-htmx↔ci-siblings                                                                               |
| 014 | [Nonce'd HTML without `Cache-Control: no-store`](014-nonce-response-caching.md)                              | Low         | cqrs-htmx adminui/dashboardui, ChastityAPI dashboard, crush-daily, timesheets                                                                     |

## Cross-filing rules

- **Twin check (013):** any fix for 002/003/005/006/011/012 must be applied to both members of a twin pair.
- **Reference implementations inside the fleet** (copy these before inventing new patterns): CV `platform/middleware/ratelimit.go` (trusted-proxy key extractor + per-profile `MaxKeys`/`TTL`), webphone `internal/server/server.go` (server lifecycle with `WithoutCancel` shutdown and a documented extractor flip rule), InboxClean `internal/web/server.go` (CSRF `TrustedProxies` + secure cookie naming), nsfw-classifier (explicit `--behind-proxy` extractor selection and `DefaultCORSConfig` narrowing), file-and-image-renamer `healthd` (SecurityHeaders→Nonce order), DiscordSync (default Cache-Control middleware), blog (httputil.Metrics + `MetricsRecorder` + `PathFunc`).
- **Upstream feedback surfaced by this audit** (library gaps, not consumer errors, for the httputil backlog): `httputil.Logging` does not propagate OTel context (DiscordSync maintains `traceAwareLogging` as a workaround, documented in their repo); `Compression` is Range-request-unaware (artmann-technologies-website gates compression on `Range` themselves); `CSRFMiddleware` cannot propagate the ServeMux pattern through nosurf (documented limitation, v1.4.0 changelog).

## Consumers reviewed without a filing

Reviewed at call-site level and found to use httputil within documented semantics: webphone, InboxClean, bank-sync (except 010), crm, crm-exec-stage, library-policy, auto-deduplicate, ChastityAPI (except 002), dynamic-markdown-site (except 012), go-appkit (+ otel/flightrecorder/security subpackages), mr-sync, file-and-image-renamer, german-business-contract-automation (except 012), RedditParse, overview (except 008), blog (except 001), browser-history (except 006), nsfw-classifier, storbi (except 002/005), plugmarket, go-plugin-mvp, go-policy-dsl, github-local-sync (except 008), cv-indirect consumers (KeyHolderAI, dnsblockd, accountability-system, legal-graph-ai-system — httputil reaches them only via cqrs-htmx/go-appkit, which carry the findings above).

## Regenerating the inventory

```bash
cd ~/projects/project-dependency-graph && GOTOOLCHAIN=auto go run . who-uses github.com/larsartmann/httputil --dir ~/projects --direct-only=false
cd ~/projects && rg -l -g '*.go' -g '!**/vendor/**' '"github.com/larsartmann/httputil"' .
```

The pipeline is now code: `scripts/consumer-audit/run-audit.sh` automates the
inventory → pins → corpus → pattern-grep steps and scaffolds a report.

## Verification log

- **2026-10-10 — the three open claims from the audit session are closed:**
  1. *overview's `s.rateLimit` definition* — located: `overview/internal/server/middleware.go:60`, a hand-rolled `golang.org/x/time/rate` limiter (one global bucket, `rate.Every(10s)`/burst 1, 429 error page). It is not an httputil limiter; no httputil rate-limit misuse.
  2. *storbi's `MaxRequestBodySize` constant value* — checked: `storbi/internal/middleware/middleware.go:35` = `1 << 20` (1 MiB), wired via `httputil.MaxBodySize` at :75. Documented semantics, correct usage; no filing change.
  3. *GmbH's cookie-setting site* — resolved negative: GmbH's Go server sets NO cookies anywhere (`http.Cookie{`/`SetCookie`/`SameSite`: zero non-vendor hits); auth is header-based JWT (`server/app/server.go:302`, `server/infrastructure/auth/middleware.go:14`). The SameSite/zero-value-cookie class is vacuous for GmbH; filing 003's "GmbH (absent)" entry is confirmed and strengthened.

