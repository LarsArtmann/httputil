# ADR 0001 — Adapter Pattern for External Middleware

**Status:** Accepted
**Date:** 2026-08-29 (retroactive for go-etag 2026-08-07 and nosurf 2026-08-05)
**Context:** how httputil consumes third-party middleware functionality.

## Decision

When third-party functionality must compose with httputil's middleware chain, wrap it behind a thin adapter with the `Middleware` signature (`func(http.Handler) http.Handler` — a true type alias, so foreign middlewares compose directly). Domain types stay in the foreign module; httputil does **not** re-export them.

## Precedents

1. **go-etag** (`etag.go`): ETag generation and conditional-request handling live in the independent `go-etag` module. `httputil.ETag()` was a thin adapter until the `Middleware` type alias made `etag.New()` directly composable, at which point the adapter was deprecated. Consumers import go-etag for `ETagConfig` and domain types.
2. **justinas/nosurf** (`csrf.go`): double-submit-cookie CSRF is security-critical and complex; hand-rolling was rejected. `CSRFMiddleware` wraps nosurf behind httputil's config model (`CSRFConfig` + `Validate`), and HTMX-aware token helpers (`CSRFTokenHXHeaders`, …) expose the token in templ/HTMX-consumable shapes without leaking nosurf's API.

## Rationale

- The adapter boundary keeps dependency surface honest: third-party packages enter at exactly one file, and their types do not leak into error codes, context keys, or config structs.
- The type-alias `Middleware` means adapters are eventually deletable — as happened with `ETag()` — when direct composition is sufficient.
- Re-exporting foreign config types (e.g. `type ETagConfig = etag.ETagConfig`) was evaluated and rejected: it duplicates the API surface, forces this module to track every upstream addition, and splits the brain about where a type lives.

## Consequences

- Consumers import external modules directly for configuration of those features.
- Adapter files (etag.go, csrf.go) own ALL knowledge of the foreign API inside httputil.
- Breaking changes in a foreign module surface at the adapter, not across the codebase.
