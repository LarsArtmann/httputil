# LINT-011: Stale httputil pins — consumers missing shipped security and correctness fixes

- **Date:** 2026-10-08
- **Severity:** Medium
- **Status:** Open
- **Consumers (by pin, latest is v1.4.2):**
  - **v1.2.0:** CV (direct, + CV/platform module), games/SEC, index (indirect), ci-siblings (entire cqrs-htmx mirror, ~20 go.mod files)
  - **v1.4.0:** SwettySwipperWeb, artmann-technologies-website, auto-deduplicate, dynamic-markdown-site, template-arch-lint, accountability-system (indirect)
  - **v1.4.1:** the long tail (28 projects, incl. all games/* except SEC)
- **Ground truth:** CHANGELOG v1.3.0 / v1.4.0 / v1.4.2 sections; `go.mod` pins collected via `rg 'larsartmann/httputil v' ~/projects/*/go.mod`

## What the stale pins actually miss

### v1.2.0 → v1.4.2 (CV, games/SEC, index, ci-siblings)

- **v1.4.2 `Metrics` nil-Recorder fix:** before this, `Metrics()` was
  the only constructor that skipped validate-and-log; a nil-Recorder
  config nil-panicked on the first request. Any v1.2.0 consumer adding
  metrics today hits the old panic behavior.
- **v1.4.2 `KeyExtractorFromRemoteAddr` per-connection caveat:** the
  documented semantics (connection churn = fresh buckets) postdate the
  pin; ci-siblings still ships the RemoteAddr extractor (LINT-006).
- **v1.4.0 route-pattern propagation regression fixes and httpspec
  LNA spec** (the pattern-propagation fix itself is in v1.2.0, so the
  v1.2.0 pins are covered for that one; the _middleware-showcase_
  examples and otel-based consumers pinned below v1.2.0 would not be,
  and the v0.12.0 testdata pin inside ci-siblings predates it).
- **v1.3.0 `etagmetrics`** (moved to go-etag v0.5.0) and the **Go
  1.27.1 floor** — relevant to ci-siblings' frozen workspace, which
  cannot build with the current toolchain matrix.

### v1.4.0/1.4.1 → v1.4.2

The v1.4.2 `Metrics` fix and the `KeyExtractorFromRemoteAddr`
documentation affect anyone constructing `Metrics` or using the
RemoteAddr extractor: game servers and template seeds are exactly the
consumers that copy config shapes without re-reading semantics.

## Why it is wrong (fleet-wide)

httputil's release cadence carries security-posture changes (CORS
`DenyUnmatched` default, CSRF SameSite/Secure fallbacks, attestation
conflict checks) and silent-failure fixes. A consumer pinned two-to-six
minor releases back silently misses both, and the fleet's "what
version is out there" question currently has four answers. The
who-uses output (2026-10-08) shows 45/49 consumers below v1.4.2.

Special cases:

- **CV** is the fleet's reference implementation for rate limiting
  (LINT-006) yet runs the oldest direct pin — its exemplary patterns
  are frozen against v1.2.0 semantics.
- **ci-siblings** is a full mirror of cqrs-htmx at v1.2.0 (~20 go.mod
  entries) used for CI matrices; every cqrs-htmx filing in this
  directory applies twice (once per copy) with different versions,
  which is also a drift hazard: fixes landing in cqrs-htmx
  (e.g. `KeyExtractorFromRemoteAddr` → `KeyExtractorFromClientIP` in
  usermgmt) do not exist in the mirror.

## Fix

- Sweep direct consumers to v1.4.2 (`go get github.com/larsartmann/httputil@v1.4.2 && go mod tidy`),
  starting with CV and games/SEC (v1.2.0).
- Decide the ci-siblings mirror's lifecycle: either automate its
  refresh from cqrs-htmx main or demote it to archived — a CI matrix
  against two-version-old middleware semantics provides false
  confidence.
- template-arch-lint (the template future projects are seeded from)
  should always track the latest tag.

## Coordination note (2026-10-10, Pareto-execution session — appended only; body above untouched per the file-boundary rule)

This report predates v1.5.0 (2026-10-09). Two facts change its framing for whoever picks it up:

1. **"Latest is v1.4.2" is stale** — latest is v1.5.0, and both modules are MIT as of that tag, so pkg.go.dev now renders the README and docs from `v1.5.0` / `server_timing/v1.0.2` onward. The consumer-upgrade pitch ("you are pinning an invisible package") gets materially stronger from v1.5.0.
2. The 2026-10-09/10 pipeline work (nightly-fuzz rolling issue, CI toolchain-skew preflight) is fleet context, not consumer-facing.

Owned by the lint-audit session; the sweep commands in the body need re-validation against v1.5.0 pins before execution.
