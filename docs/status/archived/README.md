# Archived Status Reports

Closed history: every report here was annotated to full resolution (inline `~~item~~ done at <hash>` / routed / NOT-DO verdicts on every open item) before being moved via `git mv`. Per the docs-health model, archived files are not re-opened during HARVEST/VERIFY/AUDIT — they hold zero open items. The one `.html` file here is a rendered twin of an annotated `.md` sibling and carries no separate annotation.

**Snapshot count: 104 reports + this index = 105 files (verified 2026-10-14 against the directory listing).**

## 2026-10-14 bulk-archive manifest (docs-health AUDIT)

All 25 remaining 2026-08/09 reports were annotated to full resolution in this pass (per-item verdicts: `done at <hash>` / routed arrows to the living TODO_LIST/ROADMAP rows / Won't-implement) and moved via `git mv`. One documented gate exception: `2026-09-11_09-35` §a) carries a single legacy strike from a prior pass (a-sections are otherwise never struck) — its check-rows "incomplete" is that historical row, not an open item.

| File(s) | Classification | Deciding reason |
| ------- | -------------- | --------------- |
| `2026-08-30_14-58` + `2026-08-30_19-17` | ARCHIVE | The corpus conventions they established were codified (corpus-hygiene 2026-10-10); every residue shipped, routed, or frozen-by-rule |
| `2026-09-10_04-03` | ARCHIVE | CSRF trust-model hardening shipped (attestation conflict, fallback, TrustedOrigins all-or-nothing); t.Run-cluster ruling + go-compression/arch-review routed |
| `2026-09-10_06-09` | ARCHIVE | Diagrams shipped; embedding + d2 render-check routed; the d2 skill-asset class swept 2026-10-10 |
| `2026-09-10_09-26` | ARCHIVE | The v1.0 completion sweep executed (v1.0 shipped 2026-09-10, removals in v1.1.0); every f-item closed or routed |
| `2026-09-11_04-30` | ARCHIVE | go-etag v0.3.x arc superseded by v0.5.0 (verified 15-07 §a13); consumer-sweep class superseded by the 2026-10-09 audit program; AGENTS byte-budget routed to TODO_LIST |
| `2026-09-11_06-41` | ARCHIVE | Vendor removal done; buildflow superseded by the 2026-10-10 delivery rewrite; vestigial-config + BuildFlow S64 routed |
| `2026-09-11_09-01` | ARCHIVE | The StartTLS/Start settlement guarantee shipped (godoc + tests + docs aligned 2026-10-10); drain-window + race-soak routed |
| `2026-09-11_09-05` | ARCHIVE | v1.1.0 shipped; the 9 review findings harvested (Post-v1.1 rows); AGENTS contracts documented |
| `2026-09-11_09-35` | ARCHIVE | The resume batch executed (prerelease first full green run); the review findings routed; AGENTS size watch routed |
| `2026-09-11_10-03` | ARCHIVE | The erraudit consolidation shipped (canonical Commands block, `5361480`); the advisory ledger re-measured 2026-10-11 |
| `2026-09-11_13-49` | ARCHIVE | v1.1.0 shipped; the nightly oracle class fixed → rolling-issue workflow; TrustedOrigins unification shipped |
| `2026-09-14_17-14` + `2026-09-14_18-14` | ARCHIVE | The CSRF test-depth batches shipped 2026-10-10 (every named test/fuzz pin exists); Post-v1.1 + export rows routed |
| `2026-09-15_06-14` | ARCHIVE | The low-priority backlog executed (LNA pin, spec, fuzz, RELEASE.md 6.5); owner rulings routed |
| `2026-09-15_10-28` | ARCHIVE | v1.2.0 shipped; the oracle fix held (nightly green); the docs-health cadence continued through this pass |
| `2026-09-15_17-04` | ARCHIVE | Issue #4 closed (AbsentEncoding policy, v1.2.0); the ABNF oracle sweep verdict recorded 2026-10-10 |
| `2026-09-22_21-46` | ARCHIVE | The alias-identity assertion exists (`229f3ba`); the baseline is pinned + re-verified; the -t 1/commit-flow/CHANGELOG rulings routed to ROADMAP |
| `2026-09-23_00-04_etagmetrics` | ARCHIVE | The move shipped (go-etag v0.5.0, verified 15-07 §a13); go-etag-side residue routed to TODO_LIST/ROADMAP |
| `2026-09-23_00-04_superb` | ARCHIVE | The examples shipped (a6 batch + the 2026-10-10 sweep); trash/daemon/go.work questions routed to ROADMAP |
| `2026-09-23_13-26` | ARCHIVE | The evidence-debt items verified (tags CI-green, nightly green); the doc-debt residues shipped 2026-10-10 |
| `2026-09-23_15-07` | ARCHIVE | Every b/c/f item closed or routed (the TODO sweep executed by the 2026-09-26/10-10 sessions) |
| `2026-09-23_16-22` | ARCHIVE | The baseline is flake-pinned (`c80ab65`) and re-verified 0 groups at -t 2; filter-trust + -t 1 policy routed to ROADMAP |
| `2026-09-23_16-44` | ARCHIVE | The incident class documented (AGENTS); lychee 404 resolved; the release shipped; BuildFlow batch routed |
| `2026-09-27_15-55` | ARCHIVE | The post-v1.4.0 residue batch closed 2026-10-10 (f10/f29–f45 verified in CHANGELOG); owner rulings routed |

## 2026-10-09 bulk-archive manifest (docs-health AUDIT)

| File                                                                             | Classification | Deciding reason                                                                                                                 |
| -------------------------------------------------------------------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `2026-08-30_11-30_full-code-review-execution-and-compression-duplication-fix.md` | ARCHIVE        | Open items resolved (v1.0/reviews/`ErrCSRFAttestationConflict`) or routed (go-compression, architecture-review → TODO_LIST)     |
| `2026-09-10_03-41_composability-session-status.md`                               | ARCHIVE        | Compose API fully shipped (examples/benchmarks/README); g-questions answered in addendum; residuals routed                      |
| `2026-09-11_05-37_buildflow-recovery-link-debt-and-vendor-parser-fix.md`         | ARCHIVE        | Residue is BuildFlow-repo work tracked in BuildFlow's own backlog (P8/ZCC9/G19); httputil-side items all done                   |
| `2026-09-11_06-16_regression-test-binary-rebuild-and-vendor-decisions.md`        | ARCHIVE        | Same BuildFlow-tracking disposition; lychee/sign-off/standing-warnings resolved 2026-09-15                                      |
| `2026-09-15_05-47_cors-lna-blocker-implementation.md`                            | ARCHIVE        | LNA batch shipped (spec/example/seed/docs); denied-origin pinned 2026-09-23; cross-repo dnsblockd rows deliberately NOT-DO here |
| `2026-09-15_07-00_csrf-samesite-fallback-design-and-implementation.md`           | ARCHIVE        | Fallback hardening + docs/verification all executed (a3–a10); B1 interpretation + export questions tracked in TODO_LIST         |
| `2026-10-08_23-49_server-timing-mit-readme.md`                                   | ARCHIVE        | Entire license/README/release arc completed at `19f6a91`/`2a3294f` and render-verified 2026-10-09                               |
