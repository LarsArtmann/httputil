# Status Report — Fleet-Stack Utilization Linter Decision (Advisory Session)

**Date:** 2026-10-25 15:13 CET
**Session scope:** One advisory question — _should the company-master-data fleet-stack audit findings (docs/research/2026-10-25_httputil-fleet-stack-deep-dive.html) become a linter, as with samber-linter and go-humanize-linter?_ — plus two follow-up rounds ("any relevant skills?", "view and tell me if it changes anything") and this report.
**Repos touched by me:** none. Zero code changes anywhere this session. This report is the only file written.
**Working dir:** /home/lars/projects/httputil (report placed here per instruction; most content is fleet/skills-repo scope — see HARVEST note in §f).

---

## Session Verdict (the deliverable)

**No to a general "fleet utilization linter." One narrow rule family is linterable, gated on a corpus sweep.**

| Audit finding class                                                                                                  | Verdict                                                        | Basis                                                                                                                                                                                                                                                                                                                          |
| -------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 10 "missed opportunity" rows (ETag, dashboard options, `checks.Disk`, errorpages, CORS/rate-limit, tracing, systemd) | Keep as audit, not a linter                                    | Applicability is human judgment — the audit itself marks 13/44 rows N/A with hand-written reasons. No stateable false-positive budget. library-deep-dive's refresh-trigger loop owns this class.                                                                                                                               |
| Version gap (httputil v1.5.0 → v1.6.0)                                                                               | Delegate — **but see §d.1: the delegate target is unverified** | Claimed `gomod-check` coverage from the BuildFlow detect-only tool list; never verified its check set.                                                                                                                                                                                                                         |
| Hand-rolled duplicate of a fleet library API (`newLogger` ≈ `appkit.InitLogger`)                                     | The only linterable class — gated                              | Exactly the go-humanize-linter shape. Base rate today: **one** proven instance (this audit) vs go-humanize-linter's "dozens per codebase" premise. Gate: ≥3 instances across ≥2 repos with a mechanically stateable multi-signal pattern. If the gate clears: README-as-contract spec first (samber-linter method), then code. |

---

## a) FULLY DONE

1. **linter-building skill loaded and applied** — full SKILL.md read; mechanism decision tree + when-NOT-to-build walked against all three finding classes. Evidence: skill read this session; verdict table above.
2. **Audit document parsed and classified** — 75 KB HTML extracted to text; all 10 findings and the 44-row Appendix A inventory mapped to mechanism classes (audit-owned judgment / delegate / linterable). Evidence: /home/lars/projects/company-master-data/docs/research/2026-10-25_httputil-fleet-stack-deep-dive.html §§01–03, Appendix A.
3. **Both reference linters characterized** — samber-linter: health-washing class for samber/do, rules HW-1…HW-8, confidence-gated exit codes 0/1/2, threshold policy dated 2026-09-20. go-humanize-linter: 11 multi-signal rules (H001–H012), built on go-linter-sdk + go-finding, per-rule `pattern_*.go` files on disk. Evidence: both READMEs + file listings.
4. **Verdict delivered with the refined gated path** (sweep → spec → code) — the session's core output, in chat.
5. **Skill re-verification round** ("view and tell me if it changes anything") — library-deep-dive SKILL.md, linter-building/references/ecosystem.md, and deduplicate-code SKILL.md read in full; verdict unchanged, four refinements recorded (corpus precedent, spec-first shape, deduplicate-code gap, stale ecosystem.md row). Evidence: the three files, this session.
6. **One discrepancy surfaced** — ecosystem.md's samber-linter row says "spec-only (not started)"; the repo has `cmd/`, `pkg/`, `internal/`, `plugin/`, `go.mod`, and a dated threshold policy. Row is stale (the table's own provenance note warns exactly this). Evidence: `ls` + README head vs ecosystem.md row.

---

## b) PARTIALLY DONE

1. **samber-linter implementation-depth verification** — directory structure + README verified; whether the HW rules are complete working analyzers vs partial scaffolding is unreviewed. Remaining: read rule sources / run the tool once. Effort: M.
2. **gomod-check delegation claim** — named from httputil AGENTS.md's detect-only tool list without reading what it actually checks. The audit's own "How to Verify" step does version currency _manually_ (`git describe` vs go.mod), which is evidence AGAINST a tool covering "behind latest release tag". Remaining: verify the check set, confirm or retract the delegation advice. Effort: S.
3. **Corpus-sweep design** — gate criteria stated (≥3 instances, ≥2 repos, mechanically stateable pattern); sweep not designed or executed. Precedent exists: go-humanize-linter's 158-project validation corpus (verified 2026-09-09 per linter-building skill ledger). Remaining: repo list, grep patterns, evidence format. Effort: M once authorized.
4. **Ecosystem.md staleness** — found, flagged, NOT fixed: the fan-out copy is read-only by design and the SKILLS repo source path was never located this session. Effort: S.

---

## c) NOT STARTED

1. **Corpus sweep across fleet consumers** for hand-rolled fleet-API duplicates — blocked on authorization (multi-repo) + priority call (see §g Q1/Q2).
2. **README-as-contract spec** for the gated linter (samber-linter method: verification ledger pinned to upstream file:line, discrimination proof, reason-required suppressions) — gated on the sweep.
3. **Linter implementation** — gated on the spec.
4. **Durable decision record** — the "audit classes vs linterable classes" taxonomy lives only in this session's chat; canonical placement undecided (see §g Q3).
5. **company-master-data audit items 1–6** (ETag, dashboard WithShutdownDrain/WithTrend/WithMetrics, InitLogger swap, Server-Timing segments, checks.Disk, errorpages) — explicitly out of this session's scope; untouched.
6. **httputil v1.6.0 + server_timing v1.0.3 bump** in company-master-data (audit finding 7) — untouched.
7. **httputil-repo work of any kind** — no tests run, no lint, no code read or changed in this repo. The session was pure advisory/research. Nothing here is broken by the session; nothing here was advanced by it either.

---

## d) TOTALLY FUCKED UP

Nothing is broken in any repo (no code was touched). What is fucked up is **decision quality** — radical honesty about the advice itself:

1. **The delegation leg shipped unverified.** "Version currency is already owned by gomod-check — delegate, don't build" was asserted from a tool _name_ on a list. The audit doing this check manually with `git describe` suggests NO tool covers "behind latest release tag". If that reading is right, my delegation advice is wrong and finding 7 has no tooling owner at all. Severity: medium (decision-quality, not runtime). Root cause: claimed a capability without reading the tool. Mitigation: §f item 2 before anyone acts on that leg.
2. **The decisive economic argument was never measured.** "One proven instance vs dozens" killed the linter proposal — but the base rate is a single audit of a single repo. The no-verdict is a hypothesis with a stated gate, not a measured fact. Severity: medium. Mitigation: that is precisely what the corpus sweep is for; do not cite this session as evidence the gate fails.
3. **ecosystem.md actively misleads future sessions.** "samber-linter — spec-only (not started)" while the repo is implemented. A linter-building session consulting the canonical stack map will mis-plan around a nonexistent gap. Severity: low-medium (doc rot in the canonical skill, self-flagged as stale by its provenance note, still unfixed).
4. **Minor: publish-status overclaim.** I passed along samber-linter's `go run github.com/larsartmann/samber-linter/cmd/samber-linter@latest` quick-start as if shipping/publishing were confirmed. Tag/module-proxy status never checked.

---

## e) WHAT WE SHOULD IMPROVE

1. **Verify-before-claiming applies to INTERNAL tools too.** The gomod-check miss is the verify-external-claims failure mode pointed at our own fleet tooling. Rule going forward: name a tool as the owner of a concern only after reading its rule/check set.
2. **Buy cheap data before shipping verdicts.** A 10-minute `rg` preview across 2–3 fleet repos (hand-rolled `log.NewWithOptions`/charmbracelet init) would have grounded the base-rate claim. Recommendation quality > recommendation latency.
3. **The skill-source edit path is undiscoverable in-session.** Fan-out is read-only (by design), and I never located the SKILLS repo canonical path, so the stale ecosystem.md row survives. Improvement: record the SKILLS repo path + fix-routing convention in linter-building's fan-out footer or crush-config README.
4. **ecosystem.md needs a batch freshness pass.** Its header admits non-ledger rows are research-pass-derived; the samber-linter miss proves it. Re-verify the State column against each repo's `git log` in one sitting, fix rows in batch.
5. **Capture the doctrine so it is not re-litigated.** One bullet in linter-building's when-NOT-to-build: "per-project utilization judgment → audit skill (library-deep-dive), not detection #N; only mechanically decidable duplicate pairs graduate to rules, gated on a cross-repo corpus."
6. **Citation discipline for derived artifacts.** I quoted audit numbers (13/44 N/A, 61/100) without re-deriving them — acceptable for advisory chat, but any committed artifact must re-run the audit's verify appendix first.

---

## f) Next tasks (up to 50 — 35 specific items; padding to 50 would be junk)

Impact: Critical/High/Medium/Low. Effort: S (<30min) / M (30min–2h) / L (>2h).

### Track 1 — Linter decision (this session's thread)

| #  | Task                                                                                                                                            | Impact | Effort | Category      |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Owner ruling: authorize the fleet corpus sweep now, or park until company-master-data items 1–6 ship (§g Q1/Q2)                                 | High   | S      | Decision      |
| 2  | Read gomod-check's actual check set; confirm or RETRACT the "version currency delegated" claim                                                  | High   | S      | Verification  |
| 3  | Cheap rg preview: hand-rolled logger init (charmbracelet/`log.NewWithOptions`) across 3–5 fleet consumer repos                                  | High   | S      | Research      |
| 4  | If preview is positive: full corpus sweep using go-humanize-linter's 158-project methodology                                                    | High   | M      | Research      |
| 5  | Write the corpus report (per-instance evidence, file:line) — the go/no-go gate artifact                                                         | High   | M      | Documentation |
| 6  | If gate clears: draft the linter README-as-contract spec (samber-linter method: verification ledger, discrimination proof, suppression syntax)  | High   | M      | Spec          |
| 7  | Define rule R001 `manual-logger-init`: multi-signal pattern (backend call + level-parse shape + stdout/timestamp deltas vs `appkit.InitLogger`) | Medium | M      | Feature       |
| 8  | Decide rule-family scope: appkit-only vs fleet-wide target libraries (go-health `checks` battery, `errorpages`, …)                              | Medium | S      | Decision      |
| 9  | Scaffold repo on go-linter-sdk + go-finding + toolsdk provider (blank-import wiring)                                                            | Medium | M      | Feature       |
| 10 | Port go-humanize-linter's confidence / suppression / baseline machinery                                                                         | Medium | M      | Feature       |
| 11 | Golden corpus: positive AND negative fixtures per rule                                                                                          | High   | M      | Quality       |
| 12 | Fleet validation sweep with measured false-positive rate before any default-on flip                                                             | High   | L      | Quality       |
| 13 | toolsdk self-registration so BuildFlow hosts run it with zero per-repo wiring                                                                   | Medium | S      | Feature       |
| 14 | golangci-lint v2 module-plugin wiring (go-humanize-linter's plugin pattern)                                                                     | Low    | M      | Feature       |
| 15 | Exit-code contract 0/1/2 via `linter.ExitCodeByConfidence`                                                                                      | Low    | S      | Feature       |
| 16 | If gate FAILS: record the negative verdict durably (see item 21) and close the thread                                                           | Medium | S      | Documentation |

### Track 2 — Skill/doc maintenance (found this session)

| #  | Task                                                                                                                 | Impact | Effort | Category      |
| -- | -------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 17 | Locate the SKILLS repo source behind the linter-building fan-out (`readlink` the symlink)                            | Medium | S      | Cleanup       |
| 18 | Fix ecosystem.md's samber-linter row ("spec-only" → implemented); fix in the source repo, verify through the fan-out | Medium | S      | Bug           |
| 19 | Batch freshness pass over ecosystem.md's State column vs each repo's `git log`                                       | Medium | M      | Cleanup       |
| 20 | Add the "utilization judgment → audit, not detection" bullet to linter-building when-NOT-to-build                    | Medium | S      | Documentation |
| 21 | Record this session's verdict durably at the location decided in §g Q3                                               | Medium | S      | Documentation |
| 22 | Check samber-linter's tag/publish status backing the `go run …@latest` quick-start                                   | Low    | S      | Verification  |
| 23 | One line in deduplicate-code: art-dupl is single-tree; consumer-vs-library duplicates need the audit or a linter     | Medium | S      | Documentation |

### Track 3 — company-master-data (adjacent scope, from the audit under review; NOT this session's work)

| #  | Task                                                                                                                                                      | Impact | Effort | Category      |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 24 | ETag/If-None-Match via go-etag/server on the read-only API (audit item 1, priority 25)                                                                    | High   | S      | Feature       |
| 25 | Dashboard WithShutdownDrain + WithTrend/WithMetrics (audit item 2)                                                                                        | High   | S      | Feature       |
| 26 | Replace `newLogger` with `appkit.InitLogger`; decide/document the stdout-vs-stderr + timestamp deltas (audit item 3 — and the linter's founding instance) | High   | S      | Feature       |
| 27 | Server-Timing lookup/list segments beside `render` (audit item 4)                                                                                         | Medium | S      | Feature       |
| 28 | `checks.Disk(cfg.DataDir)` as non-critical probe check (audit item 5)                                                                                     | Medium | S      | Feature       |
| 29 | httputil v1.6.0 + server_timing v1.0.3 bump, flake vendorHash re-paste (audit item 6; go-ecosystem-upgrade skill)                                         | Medium | M      | Feature       |
| 30 | Pre-write LAN-vs-internet CMD_ config variants for rate-limit/CORS/SSE-cap (audit item 7; exposure-gated)                                                 | Medium | M      | Feature       |
| 31 | errorpages for the datasheet route — decide together with the combined-group datasheet redesign (audit item 8)                                            | Low    | M      | Feature       |
| 32 | Drain-window e2e via testkit.DrainWindowProbe, after item 25 (audit item 9)                                                                               | Low    | M      | Quality       |
| 33 | Tracing: OuterMiddlewares + go-appkit/otel + RecoveryWithConfig OnPanic (audit item 10; deploy-gated)                                                     | Medium | L      | Feature       |
| 34 | StartHooks sd_notify + systemd module (audit item 11; deploy-gated)                                                                                       | Low    | L      | Feature       |
| 35 | Re-run the adoption audit after items 24–29 + the exposure/deploy answers land (the report's own refresh triggers)                                        | Medium | M      | Documentation |

**Track 4 — httputil repo: empty, honestly.** This session produced no httputil work and surfaced no httputil defects. No invented filler.

**HARVEST routing (docs-health):** Track 1–2 items belong to the SKILLS/fleet repos, NOT httputil's TODO_LIST. Track 3 items belong to company-master-data's TODO_LIST. httputil receives nothing from this report — do not entomb §f here and do not misroute it.

---

## g) Top 3 questions I cannot figure out myself

1. **Authorization/scope:** May I run the corpus sweep across fleet consumer repos now (read-only, multi-repo), or is the whole linter thread parked until the company-master-data audit items 1–6 ship? _Tried:_ derived the gate criteria myself; multi-repo access and cross-repo priority are owner calls.
2. **Shape, if the gate clears:** new dedicated repo per the one-repo-per-target-library pattern (e.g. `go-appkit-linter`), or fold the rule family into an existing linter repo? Fleet precedent says new repo; a 1–3-rule family might argue for folding. _Tried:_ checked ecosystem.md conventions — the pattern says dedicated repo, but the economics are yours to weigh.
3. **Doctrine placement:** where does the durable verdict live — a when-NOT-to-build bullet + ecosystem.md fix in the SKILLS repo (linter-building), an ADR next to the audit in company-master-data, or both? I can draft either; the SKILLS repo is commit-based (not in-session-writable), so routing is your call.

---

_Point-in-time snapshot. Sections (b)–(d) are honest about the advice's weak links, not just the work items. Written 2026-10-25 15:13 CET; no code changed anywhere in this session._
