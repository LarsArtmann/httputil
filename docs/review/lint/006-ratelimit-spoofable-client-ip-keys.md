# LINT-006: Rate-limit keys derived from spoofable client-IP headers (and the `RemoteAddr` opposite)

- **Date:** 2026-10-08
- **Severity:** High (vote integrity, auth brute-force), Medium elsewhere
- **Status:** Open
- **Consumers:** SwettySwipperWeb, cqrs-htmx/ci-siblings usermgmt, browser-history, DiscordSync, storbi, games/SEC, Standup-Killer, nsfw-classifier (solved), webphone (solved), CV, artmann (solved)
- **Ground truth:** AGENTS.md — "`ClientIP` trusts proxy headers blindly — it does not validate X-Forwarded-For or X-Real-IP. Only safe behind a reverse proxy that strips/overwrites these headers." Also the v1.4.2 changelog: `KeyExtractorFromRemoteAddr` is per-TCP-connection, so connection churn multiplies the effective budget.

## The two failure directions

**FromClientIP (spoofable):** `httputil.ClientIP` takes the first
X-Forwarded-For entry, then X-Real-IP, then RemoteAddr — no trust list.
Any client that can reach the backend directly (or via a proxy that
appends instead of strips) chooses its own rate-limit key.

**FromRemoteAddr (collective):** behind any TLS terminator all requests
share the proxy's address, so every client shares one bucket — one
abuser exhausts the global budget for everyone (ci-siblings
`usermgmt/http.go:118` still has this v1.2.0 shape; nsfw-classifier's
non-`BehindProxy` mode accepts this deliberately with a flag, which is
the honest version).

## Evidence of real damage in consumers

- **SwettySwipperWeb** `services/api/handler/vote_guard.go:160`: vote
  deduplication is keyed on `httputil.KeyExtractorFromClientIP()(r)`.
  Vote manipulation requires only a rotating XFF header unless the
  deployment's proxy scrubs inbound XFF; the same extractor feeds the
  login/vote/import limiters (middleware.go:112-126, 229, 245).
- **cqrs-htmx usermgmt** `usermgmt/http.go:105-123`: registration,
  TOTP-verify, webauthn, and OAuth budgets keyed on FromClientIP with a
  comment that correctly states the spoofing caveat — the knowledge is
  documented in the code and the spoofable extractor ships anyway, with
  no `TrustedProxies`-style gate.
- **browser-history** `api/middleware.go:166`: comment claims the
  fallback "is proxy-aware (X-Forwarded-For / X-Real-IP) so it works
  correctly behind nginx/Docker" — misleading: it is proxy-*trusting*,
  not proxy-aware; the auth-endpoint limiter (:199) inherits the same
  key, so brute-force throttling on WebAuthn ceremonies is bypassable
  the same way.
- **DiscordSync, storbi, games/SEC, Standup-Killer**: same extractor,
  no documented trust boundary, no flag.

## The in-fleet correct patterns (copy these)

| Consumer | Pattern |
|---|---|
| CV (`platform/middleware/ratelimit.go:41`) | `trustedProxyKeyExtractor`: forwarded headers honored only for `httpx.SetTrustedProxies` CIDRs, falls back to RemoteAddr host so the key is never empty |
| artmann-technologies-website (`middleware.go:172`) | `trustedProxyEnabled()` gate choosing ClientIP vs stripped-host RemoteAddr, with the port-strip rationale documented and rehearsal-tested |
| nsfw-classifier (`command/server/server.go:511`) | explicit `--behind-proxy` flag choosing the extractor |
| webphone (`internal/server/server.go:109`) | RemoteAddr-host key with a written flip rule to switch to FromClientIP once the stack proves XFF sanitization |

## Fix

Adopt one of the four patterns above wherever FromClientIP is used; the
CV variant is the most complete (trust list from config, non-empty key
invariant). Until a trust list exists, a spoofable key on a
vote/security endpoint should be treated as no rate limit for an
attacker worth stopping.
