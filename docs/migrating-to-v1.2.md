# Migrating to httputil v1.2 (behavior changes)

The next minor release (v1.2.0) changes the behavior of two existing
mechanisms. No exported signature changes; both deltas are visible only
through error classification results and CSRF config validation. Read this
if you use `CSRFConfig.TrustedOrigins` or route retry decisions through
`errorfamily.Classify`/`IsRetryable` on the stdlib sentinels
`http.ErrNoCookie`/`http.ErrNoLocation`.

## CSRF `TrustedOrigins` parsing is all-or-nothing and validated

**Before (v1.1.x):** a `TrustedOrigins` list containing one entry that
`url.Parse` rejects silently split the trust semantics — the nosurf handler
trusted no custom origins at all (wholesale failure), while the
attestation-consistency check kept the parseable siblings. Entries without a
scheme or host (e.g. `"example.com"`) were accepted at construction and
died silently: they can never match an `Origin` header.

**After (v1.2.0):** both sides parse the list all-or-nothing (mirroring
`nosurf.StaticOrigins`), so the attestation check can never trust a
different set than the nosurf handler enforces — any unparseable entry
leaves no trusted origins at all, and construction logs the fallback to
same-origin-only validation loudly. `CSRFConfig.Validate()` additionally
rejects entries that are not usable `scheme://host` origins (unparseable,
missing scheme, or missing host) with the new
`csrf.trusted_origin_invalid` code (Rejection, cause-chained to
`ErrCSRFConfig`).

**What to do:**

- Write full origins: `"https://app.example.com"`, not `"app.example.com"`.
- Audit configs that relied on lenient parsing — entries that never worked
  now fail construction-time validation instead of being ignored.
- Configs whose entries were already valid `scheme://host` URLs behave
  identically.

## `http.ErrNoCookie` / `http.ErrNoLocation` are classified Rejection

**Before (v1.1.x):** both stdlib sentinels were registered `Transient`
(`errorfamily.Classify(err).ErrorFamily()` = Transient, `IsRetryable` true).

**After (v1.2.0):** both are registered `Rejection` — the named cookie or
header location is deterministically absent, so an unchanged retry can
never succeed, which matches the package taxonomy for deterministic
failures.

**What to do:** no code change is required unless your retry middleware or
SLO routing treats `IsRetryable(err)` on these sentinels as retryable —
such requests now land in the non-retryable path, which is the intended
correction.

## Also in this release (additive, no action required)

- **`CORSConfig.AllowPrivateNetwork`** (default `false`): opt-in support
  for Chrome's Private Network Access / Local Network Access preflight
  check — the middleware-generated preflight 204 carries
  `Access-Control-Allow-Private-Network: true` when enabled. See the
  CHANGELOG entry for the exact grant semantics.
