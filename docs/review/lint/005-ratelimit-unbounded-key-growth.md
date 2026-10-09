# LINT-005: `KeyedRateLimiter` instances without `MaxKeys`/`TTL` grow without bound

- **Date:** 2026-10-08
- **Severity:** Medium (memory exhaustion vector), and two comments document the wrong semantics
- **Status:** Open
- **Consumers:** cqrs-htmx (+ its ci-siblings mirror), DiscordSync, storbi, SwettySwipperWeb, games/SEC, Standup-Killer, artmann-technologies-website, browser-history (partial)
- **Ground truth:** AGENTS.md "Non-Obvious Behaviors": "`KeyedRateLimiter` uses O(log n) min-heap eviction — when `MaxKeys` is set ... Without `MaxKeys`, growth is unbounded. `EvictionTTL` provides lazy time-based eviction."

## What the consumers do

Every limiter below keys on client IP (see LINT-006 for why those keys
are attacker-chosen in the relevant deployments) and sets neither
`MaxKeys` nor `TTL`:

| Consumer | Site | Note |
|---|---|---|
| cqrs-htmx | `usermgmt/http.go:118` `newLimiterFromConfig` | six limiters: register, import, TOTP, verification, webauthn, oauth |
| ci-siblings | mirror of the above at `.../usermgmt/http.go:118` | plus uses `KeyExtractorFromRemoteAddr` (v1.2.0 shape) |
| DiscordSync | `internal/api/server.go:244` | comment claims "TTL/MaxKeys/callbacks default to zero-value production defaults" — false: zero `MaxKeys` means unbounded, zero `TTL` means no lazy eviction |
| storbi | `internal/middleware/middleware.go:143` | full struct literal with explicit `TTL: 0, MaxKeys: 0` — the zeros are load-bearing and wrong |
| SwettySwipperWeb | `services/api/middleware.go:112-126` | login, vote, import limiters |
| games/SEC | `server/middleware.go:44` | player/user/IP extractor, no caps |
| Standup-Killer | `api/server.go:142` | `WithRateLimit` option |
| artmann-technologies-website | `cmd/.../middleware.go:187` | contact-form limiter (`exhaustruct` nolint implies the fields are unset) |

Positive in-fleet contrasts: browser-history sets `MaxKeys: 10000/1000`
(api/middleware.go:171, 199); CV sets per-profile `MaxKeys` constants
plus `TTL: 10m` (platform/middleware/ratelimit.go:27); go-appkit's
security package sets `TTL: limiterTTL` and exposes `MaxKeys`
(security/ratelimit.go:105).

## Why it is wrong

Auth endpoints (`/auth/register`, TOTP verify, login, vote) are exactly
where an attacker can mint unlimited distinct keys cheaply: one
spoofable `X-Forwarded-For` value per request (LINT-006) creates one
map entry + heap node each, and without `TTL` nothing is ever freed.
The result is an unbounded memory growth vector whose cost is one
header per request, on the endpoints least able to afford downtime.
cqrs-htmx amplifies this because usermgmt is vendored into every
consumer of its `usermgmt` module (dnsblockd, GmbH, KeyHolderAI via
their user management stacks).

The DiscordSync comment is separately harmful: it encodes the false
claim that zero-value equals "production defaults", teaching the next
maintainer that the config is already hardened.

## Fix

- Set `MaxKeys` (orders of magnitude above legitimate concurrent
  clients: CV uses 5k-10k per profile) and `TTL` (10m is the in-fleet
  precedent) on every limiter above.
- Fix the DiscordSync comment; storbi's explicit zeros should become
  named constants like CV's.
- cqrs-htmx `newLimiterFromConfig` should fill `MaxKeys`/`TTL` defaults
  in `applyConfigDefaults` so every usermgmt consumer inherits the cap.
