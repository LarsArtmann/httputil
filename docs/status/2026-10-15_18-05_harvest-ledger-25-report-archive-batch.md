# Consolidated Harvest Ledger — the 25-Report 2026-10-14 Archive Batch

_Created: 2026-10-15 18:05 CEST (pareto T15 / P100.1–P100.2) · Inputs: the [archive manifest](archived/README.md) deciding reasons, the routed-arrow verdicts inside the 25 archived reports, and the living destinations verified 2026-10-15 · Purpose: one table proving every routed arrow from the archived corpus lands in a living, state-tracked destination — the "nothing routed is lost" property in auditable form._

**Method note.** The 2026-10-14 docs-health AUDIT already resolved each of the 25 reports item-by-item (struck / routed / NOT-DO) before `git mv` — re-reading the files would duplicate, not consolidate. This ledger therefore consolidates at the DESTINATION level: each arrow class the manifest records, the living doc that owns it, and its state as of 2026-10-15. Per the archive rule the archived files themselves are not re-opened.

## Arrow → destination ledger

| Routed arrow class (source reports) | Destination | State 2026-10-15 |
| --- | --- | --- |
| AGENTS byte-budget watch (`2026-09-11_04-30`, `2026-09-11_09-35`) | TODO_LIST row 26 → slim outline | Outline shipped (`docs/planning/2026-10-15_02-25_agents-slim-outline.md`, 48,175 B inventory + 5 candidates); execution = decision-brief row T7 (owner) |
| go-etag residue: bench artifact, DOMAIN_LANGUAGE entries, no-fuzz decision, stale require (`2026-09-23_00-04_etagmetrics`, `2026-09-27_15-55`) | TODO_LIST row 39 → pareto T20 | Closed 2026-10-15: glossary rows + no-fuzz decision + drift banner + doc-snippet-refs port landed (go-etag `7f39f78`, CI gate `ff2cf28`); `-count=6` metrics artifact saved locally per bench-artifact convention; the KeyHolderAI stale-require finding was already superseded — the v0.6 sweep dropped the root require, go.mod now carries entitytag/server v0.6.1 indirect (lines 93–94), matching latest |
| BuildFlow cross-repo batch: 11 failed format steps, downgrade issue, v2.14.0 pin ripple (`2026-09-11_06-41`, `2026-09-23_16-44`, `2026-09-22_21-46`) | TODO_LIST rows 22/40 → pareto T21; BuildFlow's own backlog (P8/ZCC9/G19) | Closed 2026-10-15 with two verdicts instead of an issue: (1) the 11-step enumeration is unrecoverable (state DB `step_timings` starts 2026-10-06; the run was 2026-09-23) and the mutating step class is identified at source — format mode = fast build-mode + AutoFix, and the directive surface is owned by go-version-auto-configure's floor-safe patch strips; (2) the downgrade issue is OBSOLETE — the strip became fleet policy on 2026-10-03 (go-etag v0.6.1 de-poisoning cascade; httputil directives now minor-form `go 1.27`) and the opt-out shipped 2026-10-14 (`honor-the-pin` knob, go-version-auto-configure `ed93855`). Pin ripple: BuildFlow itself pins no lint version (CI installs @latest); the real stale fleet pin was go-etag's ci.yml v2.13.2, bumped to v2.14.0 (`ff2cf28`) |
| t.Run-cluster ruling (`2026-09-10_04-03`) | TODO_LIST row 37 | Closed: 0 `t.Run` remain in `security_test.go` (verified in-tree 2026-10-15; conversion shipped 2026-10-10) |
| CSRF test-depth batch (`2026-09-14_17-14`, `_18-14`) | TODO_LIST (done) | Closed 2026-10-10: every named test/fuzz pin exists; CHANGELOG [Unreleased] carries the batch entry |
| Post-v1.1 review findings: result type, alias, typed stack names, nil-entry (`2026-09-11_09-05`, `2026-09-11_09-35`, `2026-09-14_*`) | TODO_LIST Post-v1.1 rows 46–49 → pareto T25 | Open; proposal notes written 2026-10-15 (T25, this session) |
| Consumer-audit program + remediation backlog (`2026-09-11_04-30` class superseded, `2026-09-27_15-55`) | TODO_LIST rows 33–35 + `docs/review/lint/` | Pipeline as code + claims closed (2026-10-10, CHANGELOG); backlog read-only pending R10; GmbH pre-staged 2026-10-15 |
| Architecture-review re-run (`2026-08-30_11-30`, `2026-09-10_04-03`) | TODO_LIST row 7 | Open; next quarterly cycle (consciously unscheduled) |
| go-compression extraction (`2026-08-30_11-30`, `2026-09-10_04-03`) | TODO_LIST row 20 | Precondition 1 DONE 2026-10-15 (§10 inventory refresh + parseQValue constraint); R15 scope question filed; setup = owner |
| Shutdown drain window + race-soak (`2026-09-11_09-01`) | ROADMAP "Server shutdown-drain semantics" (sources cited) → pareto T23 | Row present; decision note written 2026-10-15 (T23, this session) |
| Nightly-fuzz scaling ideas (`2026-09-11_09-05/_09-35/_13-49`, `2026-09-15_17-04`, `2026-09-23_15-07`) | ROADMAP "Nightly-fuzz workflow scaling ideas" | Row present; 4 of the ideas shipped 2026-10-15 (the +4 rotation, uniform budgets); the rest await the owner cost-appetite call |
| TLS-config clone-on-write (`2026-09-11_09-35`) | ROADMAP | Row present (idea-stage) |
| README flake-app discovery table (`2026-09-10_09-26`) | ROADMAP | Row present (sales-page fit = owner judgment) |
| art-dupl `-t 1` gate + filter trust (`2026-09-22_21-46`, `2026-09-23_16-22`) | ROADMAP Open questions ×2 | Rows present; baseline re-verified 2026-10-15 (-t 2: 0 groups, -t 1: the 2 accepted pairs) |
| CHANGELOG granularity + commit-flow (`2026-09-22_21-46`) | ROADMAP Open questions ×2 + decision brief | Rows present (owner convention calls) |
| go.work membership policy (`2026-09-23_00-04_superb`) | ROADMAP Open question | Row present |
| Trash-mover identity (`2026-09-23_00-04_superb`, `_etagmetrics`) | ROADMAP Open question | Row present (fleet-level) |
| Vestigial vendor refs + MD060 + cache purge (`2026-09-11_06-41` class) | Decision brief | Rows present (owner) |
| Denied-origin LNA posture, B1 SameSite, `csrf.trusted_origin_invalid` export (`2026-09-15_06-14`, `2026-09-15_07-00`) | Decision brief R-rows | Rows present (owner) |
| Diagram embedding (`2026-09-10_06-09`) | TODO_LIST row 38 → pareto T24 | Prep executed 2026-10-15 (T24, this session); eyeball = owner |
| v2.0 items (`2026-09-14_18-14`, `2026-09-22_21-46`) | TODO_LIST Post-v1.1 → pareto T25 | Proposal notes written 2026-10-15 (T25) |
| KEEP-file bulk strike (`2026-09-23_13-26` class) | Decision brief | Row present (owner) |
| Lychee 404, evidence-debt doc residues (`2026-09-23_13-26`, `2026-09-23_16-44`) | Closed | Shipped 2026-10-10 / resolved at archive time |
| Low-priority backlog (`2026-09-15_06-14`) | Closed | Executed at archive time (LNA pin, spec, fuzz, RELEASE.md 6.5) |
| Nightly oracle class → rolling issue (`2026-09-11_13-49`) | Closed | Shipped (owner decision 2026-10-09, CHANGELOG) |
| erraudit consolidation (`2026-09-11_10-03`) | Closed | Canonical Commands block + ledger (51/49 re-derived 2026-10-15) |

## Completeness statement

Every arrow class the 25-report manifest records maps to exactly one living destination (TODO_LIST row, ROADMAP row, decision-brief row, BuildFlow backlog, or a Closed disposition with its CHANGELOG/verification evidence). No arrow terminates in prose only. The five pareto tasks that consumed open arrows today (T20, T21, T23, T24, T25) are marked in-line above.
