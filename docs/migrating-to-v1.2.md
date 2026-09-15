# Migrating to httputil v1.2 (behavior changes)

The next minor release (v1.2.0) changes the behavior of four existing
mechanisms. No exported signature changes; the deltas are visible only
through error classification results, CSRF config validation, the CSRF
cookie's attributes, and response compression for header-less requests. Read
this if you use `CSRFConfig.TrustedOrigins`, configure `SameSite=None`,
route retry decisions through `errorfamily.Classify`/`IsRetryable` on the
stdlib sentinels `http.ErrNoCookie`/`http.ErrNoLocation`, or serve
compression to clients that send no `Accept-Encoding` header.

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

## CSRF `SameSite=None` without `Secure` falls back to `Secure=true`

**Before (v1.1.x):** `CSRFMiddleware` honored `SameSite: SameSiteNoneMode`
combined with `Secure: false` verbatim (after logging the
`csrf_samesite_insecure` validation error). Current browsers ignore such
cookies entirely (rfc6265bis §5.7), so the CSRF cookie never persisted and
every state-changing browser request failed validation — the
misconfiguration was already self-announcing, not a silent downgrade.

**After (v1.2.0):** the constructor keeps the `SameSite=None` intent but
forces `Secure=true`, logging the fallback (code `csrf_samesite_insecure`).
`InvalidateCSRFCookie` applies the same fallback so the deletion cookie
matches. `Validate()` is unchanged and still reports the combination as
invalid.

**What to do:**

- HTTPS deployments (direct or TLS-terminated behind a proxy): no action —
  the fallback makes the configured cross-site cookie actually work.
- Plain-HTTP deployments: runtime behavior is unchanged (the cookie was
  already unstorable in enforcing browsers), but set `Secure: true` or drop
  `SameSite=None` to clear the construction log.
- Deployments that must keep the insecure cookie verbatim (legacy clients
  that ignore the Secure requirement): set
  `AllowInsecureSameSiteNone: true`.

## Requests without `Accept-Encoding` are served uncompressed by default

**Before (v1.1.x):** a request carrying no `Accept-Encoding` header was
compressed with the highest-priority configured encoding. RFC 7231 §5.3.4
makes an absent header "no explicit preference" — compressing is legal —
but minimal HTTP clients, health probes, and intermediaries that do not
declare support may not decompress, so the compressed bytes could be
misinterpreted or stored corrupted (issue #4). Consumers worked around this
with explicit guard wrappers upstream.

**After (v1.2.0):** header-less requests receive the uncompressed
representation. The new `CompressionConfig.AbsentEncoding` field
(`AbsentEncodingPolicy`) controls the behavior: the zero value
`AbsentEncodingIdentity` is the new default (and makes an empty header
value — which RFC 7231 defines as "no content-coding" — correct as well);
`AbsentEncodingFirstConfigured` restores the exact v1.1.x behavior.
Explicit `Accept-Encoding` negotiation (q-values, `identity`, q=0
exclusions, server priority tiebreaks) is unchanged.

**What to do:**

- Most deployments: no action — header-less clients now get bytes they can
  always read, and browsers (which always send `Accept-Encoding: gzip`) are
  unaffected.
- Deployments whose clients send no header but always decompress (custom
  SDKs, trusted intermediaries): set
  `AbsentEncoding: httputil.AbsentEncodingFirstConfigured` to keep the old
  compression behavior.
- Remove any upstream guard wrapper that forced identity for header-less
  requests — the middleware now does it.

## Also in this release (additive, no action required)

- **`CORSConfig.AllowPrivateNetwork`** (default `false`): opt-in support
  for Chrome's Private Network Access / Local Network Access preflight
  check — the middleware-generated preflight 204 carries
  `Access-Control-Allow-Private-Network: true` when enabled. See the
  CHANGELOG entry for the exact grant semantics.
- **`CSRFConfig.AllowInsecureSameSiteNone`** (default `false`): opts out of
  the CSRF `SameSite=None` fallback above, keeping the insecure cookie
  verbatim. No action required unless you serve legacy clients that ignore
  the Secure requirement.
