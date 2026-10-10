# Execution status — TODO-LIST continuation (2026-10-10, 08:19–09:30 CEST)

**Date:** 2026-10-10 09:30 CEST · **Scope:** continuation of the whole-TODO-LIST run after the owner's "Execute and Verify them one step at a time ... until everything works" mandate. Prior state: docs/status/2026-10-10_08-05 (§h addendum records the tail-debt verification this session opened with).

## a) FULLY DONE

1. **Tail debt + first push** — lint 0 issues (`--timeout 5m`), fmt-idempotent, `-race -count=10` clean, both real erraudit gates exit 0; consolidated the mixed daemon commits into explicit-path task commits and pushed; CI green (run 38030629769, sha f65aec6). Every later push also verified via `--json conclusion`.
2. **M10 — post-v1.4.0 residue batch (12 items)**: `AbsentEncodingPolicy.String()`; `KeyedRateLimiterConfig.Validate` Burst>`math.MaxInt32` guard (new `ratelimit.keyed_burst_too_large` Rejection, full error ceremony: template, completeness list, docs sweep); keyed-limiter default Retry-After documented; `MiddlewareStack` zero-value documented + execution-probe test; `TestTimeout_NegativeDurationExpiresContextImmediately`; httpspec `ExpectVaryContains`/`ExpectNotModifiedWithETag` examples (+2 pre-existing examples made self-contained); `-shuffle=on` in both CI test steps; off-cycle `prerelease-check.sh` **all gates passed**; three items verified already-done (invalid-gzip test, M08 listener-occupation fixture, coverage-threshold CI gate).
3. **M20 — buildflow/erraudit hygiene**: `buildflow --dry-run` re-run caught a NEW tool class exactly as designed (fleet `securitymd` linter wanted a "Security Practices" section — satisfied with real content, step green; composition recorded in AGENTS); noctx blanket test exclusion narrowed to a text-scoped rule AND the genuinely context-less sites fixed in both modules (`http.Get`/`client.Get` → `Do(newTestGetRequest(t, url))`, `net.Listen` → `new(net.ListenConfig).Listen(ctx, …)`, `tls.DialWithDialer` → `tls.Dialer.DialContext`); `nix flake check --all-systems` all-checks-pass (darwin accepted); formatter×nolint watch retired (3 directives live, 0-changed fmt pass); erraudit tooling hygiene closed (binary located, zero nolint needs, pre-commit parity verified).
4. **M22 — consumer-audit**: `scripts/consumer-audit/` pipeline as code (who-uses → pins → corpus → 11 pattern greps → report scaffold; smoke-tested: 46 consumers, 1,423 call sites); the 3 unverified claims closed with a dated verification log in 000-index (overview's `s.rateLimit` located — global x/time/rate bucket; storbi's body limit = 1 MiB via `httputil.MaxBodySize`; GmbH sets NO cookies — header JWT, SameSite class vacuous).
5. **M23 — t.Run clusters + corpus hygiene**: both `security_test.go` clusters converted to 6 standalone tests (0 `t.Run` remain); corpus-hygiene sweep verdict recorded (all 5 dated batch markers live in frozen archived history; 3 sampled claims verified true; no snapshot cited as current evidence).
6. **M21 — benchmarks**: quiet-window re-measure retired both † flags (MaxKeysChurn 620→460.5, browserMulti 127.9→73.71) and the httpspec table (all rows improved/within noise); full root-module 3s×5 baseline committed as `docs/benchmarks.bench.txt`; CI posts a `benchstat` comparison into the job summary (informational — CI runners too noisy to gate).
7. **§f items**: server_timing + limiter examples (`ExampleServerTiming_MeasureWithDesc`, nil-safe `ExampleServerTimingFromContext`, `ExampleNewKeyedRateLimiter` MaxKeys-eviction walk); `httptest.NewTestServer` evaluated → NOT adopted (no real-time waits in the Timeout tests); go-error-family#5 CLOSED upstream as implemented → httputil README + arch-ref now link the 428=Rejection/412=Conflict/304=success guidance; `docs/external-claim-extraction.md` written; architecture-reference re-inventory (script-assisted per-file export diff; 2 method rows completed); AGENTS stale-binary paragraph rewritten to the nix-profile delivery; session follow-ups 3/5 closed (a5 struck, archive-count gate resolved claim-free, check-rows exemption recorded).

## b) PARTIALLY DONE / REMAINING (all non-owner-gated items now scoped)

1. **Skill-defect report** (annotate-status-items `| N |` mangling) — external filing; requires verify-before-filing reproduction; deferred deliberately.
2. **BuildFlow cross-repo batches** (golangci v2.14.0 fleet-pin ripple; upstream issue batch) — next-session items; they edit ~/projects/BuildFlow and file external issues.
3. **go-etag hygiene batch** — cross-repo docs work, untouched.
4. **AGENTS fact-loss audit** — needs an independent reader; this session edited AGENTS itself, so a fresh-eyes pass remains honest-open.
5. **architecture-review re-run** — a full skill run; heavy, scheduled separately.

## c) OWNER-GATED (untouched, per standing rule)

All §5 items: LNA denied-origin posture, B1 SameSite-default ruling, go-compression extraction, httpspec docs-site, `csrf.trusted_origin_invalid` export, MD060 ruling, writeHealthBody, consumer fix-PRs, diagram embedding, v2.0 items, KEEP-file bulk strike.

## d) DISCLOSED MISTAKES

1. Two `edit` misfires dropped a trailing newline and merged lines (compression_test.go, server_timing/example_test.go) — both caught by immediate re-read and repaired before any gate ran.
2. A truncated `grep | head -3` hid 6 `newStatusOnlyHandler` usages; I deleted a still-used helper and broke the httpspec build — restored within one tool call, then re-grepped untruncated.
3. The design-note commit message says "add" but the file was already on master (the change was a trailing-newline removal) — daemon fragmentation made the unpushed file list ambiguous; mislabel disclosed here, no functional impact.
4. sed first attempt on the `net.ListenConfig` pointer-method fix used the wrong escape and matched nothing — caught by grep-before-continue, fixed with the correct pattern.

## e) VERIFICATION TRAIL

Every macro-task closed with: lint 0 issues → fmt/nix-fmt 0-changed → `go test -race -shuffle=on` green → markdownlint on touched docs → CHANGELOG entry → TODO strike → push → `gh run view --json conclusion` = success. Final state: master green through this session's last push; working tree committed; parallel session's files published under an attributed commit after 50+ minutes of quiescence.

_Arte in Aeternum_
