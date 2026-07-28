# DenyUnmatched Default-Flip Evaluation

**Date:** 2026-07-29
**Status:** Recommended for v0.7.0 — **YES, flip the default**

## Problem

`CORSConfig.DenyUnmatched` currently defaults to `false`. When `AllowAllOrigins` is `false` and an origin matches no entry in `AllowedOrigins`, the middleware falls back to responding with `Access-Control-Allow-Origin: *`.

This is a security footgun: a developer who configures an allowlist to restrict cross-origin access gets the **opposite** — every unmatched origin is silently allowed via the wildcard fallback. The allowlist is advisory, not enforced.

## Current Behavior

```go
// resolveOrigin() in cors.go:
if cfg.DenyUnmatched {
    return ""  // no ACAO header → browser blocks
}
return "*"     // wildcard → browser allows any origin
```

`DefaultCORSConfig()` sets `AllowAllOrigins: true`, so the `DenyUnmatched` path is only reached when a developer explicitly opts out of `AllowAllOrigins` and sets `AllowedOrigins`. These are precisely the users who expect their allowlist to be enforced.

## Security Analysis

| Scenario | Current (default false) | After flip (default true) |
| -------- | ----------------------- | ------------------------- |
| `AllowAllOrigins: true` | `*` for all (correct) | `*` for all (unchanged) |
| Origin in `AllowedOrigins` | Echo origin (correct) | Echo origin (unchanged) |
| Origin NOT in list, `DenyUnmatched: false` | Falls back to `*` (insecure) | Falls back to `*` (opt-in) |
| Origin NOT in list, default behavior | Falls back to `*` (**insecure**) | No header (**secure**) |

The only behavioral change affects: `AllowAllOrigins: false` + unmatched origin + default config. This is the exact scenario where the current behavior is wrong.

## Breaking Change Assessment

**Who is affected:** Consumers who set `AllowAllOrigins: false` with an `AllowedOrigins` list, and who rely on unmatched origins receiving `*` instead of being blocked.

**How likely:** Unlikely. Developers who set `AllowAllOrigins: false` typically do so because they want to restrict access, not because they want a decorative allowlist with a wildcard backdoor. Any deployment relying on the wildcard fallback for unmatched origins has a misconfigured security posture.

**Migration:** Set `DenyUnmatched: false` explicitly to preserve old behavior. One line of code.

**Semver:** v0.7.0 is pre-1.0 with an existing batch of breaking changes. This fits naturally.

## Recommendation

**Flip the default to `true` in v0.7.0.** Rationale:

1. **Secure by default** is the correct posture for a security-adjacent library.
2. The current default actively undermines the allowlist feature it claims to provide.
3. The blast radius is small — only affects explicit allowlist users with unmatched origins.
4. Migration is trivial (one explicit field).
5. v0.7.0 already ships breaking changes.

## Implementation

- `DefaultCORSConfig()`: change `DenyUnmatched: false` → `DenyUnmatched: true`
- README `CORSConfig` table: update default column
- AGENTS.md: update the DenyUnmatched note
- CHANGELOG: document as **Breaking**
