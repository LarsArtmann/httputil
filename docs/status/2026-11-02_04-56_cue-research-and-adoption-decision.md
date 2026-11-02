# Status Report: CUE Research & Adoption Decision Session

- **Timestamp:** 2026-11-02 04:56 CET
- **Scope:** This session ONLY (3 turns: CUE research → adoption verdict → this report). No code, no features, no other projects investigated.
- **Files changed this session:** none (this report is the first artifact).
- **Tree state at write time:** `M flake.lock` (pre-existing, not session-authored; recent history is daemon heuristic commits).
- **Commits by session:** none. Manual commit skipped per harness rule; the auto-commit daemon will pick this file up.

---

## Self-Review (brutal, session-scoped)

| # | Question | Honest answer |
|---|----------|---------------|
| 1 | What did I forget? | (1) Persisting the research at the moment of discovery — turn-1 lessons and the turn-2 verdict lived only in chat until this report. (2) The ONE gating question ("is fleet-central policy a goal?") was stated as a caveat but never actually asked via the question tool — two turns ended with dangling "say the word" offers instead. (3) The nixpkgs-packaging claim was never verified. (4) The BuildFlow external-tool step model was never checked before recommending a BuildFlow step. (5) apidiff was recommended twice but never scoped (multi-module baselines, tool source, CI placement). |
| 2 | What is stupid that we do anyway? | Research sessions that end in chat prose. The fleet's whole memory doctrine says "write it down immediately" — this session did the opposite twice, and only the explicit status demand forced a durable artifact. |
| 3 | What could I have done better? | Primary-source breadth: I never opened CUE's repo, compatibility policy, or engineering practices — the "what can we learn" question had an unmined vein (their compat promises and triage are exactly the class of process we care about). Fleet claims were made from a 40-line `ls` glimpse instead of a real census. |
| 4 | What could I still improve? | See (e). |
| 5 | Did I lie? | No intentional falsehoods. One memory-based assertion delivered as fact: "cue is in nixpkgs" (unverified). Two glimpse-based generalizations ("fleet has dozens of repos", OpenAPI lane fit) were labeled needs-check at delivery time — labeled, but still shipped without evidence. |
| 6 | How can we be less stupid? | Every recommendation that names an integration point (flake, BuildFlow, CI) gets a 60-second existence check before it ships. Gating decisions get asked the same turn they're identified. |
| 7 | Ghost systems? | None created (no code). But two "ghost proposals" exist only in chat: the apidiff gate and the CUE pilot — capabilities proposed, wired into nothing. This report integrates them into (f) so HARVEST can pull them forward. |
| 8 | Scope creep trap? | Brushed twice: the OpenAPI lane and the website-launch provenance pattern both flirt with unrequested expansion. Both are parked as needs-check items, not commitments. |
| 9 | Removed something useful? | No. Nothing was touched. |
| 10 | Split brains? | None created. One latent risk recorded: if the CUE pilot lands, its documentation would span AGENTS.md Commands, a new policy repo, and BuildFlow — that needs a single canonical home on day one. |
| 11 | Tests? | N/A honestly: a research session has no test surface. Quality control was chat-review only; nothing mechanical guards any claim made this session. The apidiff track (f) is the fix for the one promise (v1-stability) that deserves a gate. |
| 12 | Re-verified inherited claims? | Re-verified against the repo: `docs/v1-stability.md` exists, zero `apidiff` usage, `cue` binary absent, workflows enumerated (ci.yml, nightly-fuzz.yml, release.yml). Leaned on AGENTS.md's depguard list as-is (acceptable: mechanically enforced by the linter itself). |

---

## a) FULLY DONE

1. **CUE primary-source research** — 6 official pages fetched and read in full: landing, docs index, introduction (philosophy/history), data-validation guide, Go integration guide, configuration guide. Evidence: fetched content quoted in session; claims cited to specific pages.
2. **Lessons synthesis (8 lessons)** — GCL one-way-door lesson, no-silent-overrides, validation-as-type-system, error provenance, gradual adoption, shift-left + spec-containment, machine-first design, config≠computation — each mapped to httputil/fleet practice with page evidence.
3. **Fleet-policy grounding** — how-to-golang skill loaded and applied: validation lanes confirmed owned (govalid, Huma, go-faster/yaml), so the CUE-as-Go-library lane was correctly closed by policy, not just by the depguard ban.
4. **Repo fact-checks** — verified: `docs/v1-stability.md` exists; `apidiff` appears nowhere in the repo; `cue` binary not installed locally; httputil has exactly 3 workflows (`ci.yml`, `nightly-fuzz.yml`, `release.yml`).
5. **Adoption verdict delivered** — Yes as external `cue vet` pilot / No as Go dependency / No for tool-owned configs, with an explicit adoption gate (catch a real defect actionlint can't, else retire).
6. **Constraint compliance** — recommendation never violated the depguard allowlist or fleet library policy.

## b) PARTIALLY DONE

1. **CUE pilot** — designed (4 steps: install pinned CUE → policy module → vet 3 workflows → adopt/retire gate), not executed. Remaining: everything concrete — no cue binary, no policy module, no vet runs, no findings. Blocker: none technical; gated on owner intent (Q1). Effort to finish: M per step, L total.
2. **apidiff gate** — the gap is verified real (v1-stability is prose; zero apidiff usage). Remaining: scoping (multi-module baseline story for root/httpspec/server_timing, tool source, CI placement), then implementation. Blocker: placement is an owner decision (Q2). Effort: M.
3. **Secondary-claim verification** — nixpkgs cue packaging, fleet repo census, a-team-openapi-spec fit, BuildFlow step model: all identified, none verified. Blocker: none; pure discipline debt. Effort: S each.
4. **Knowledge persistence** — done retroactively by this report; the durable distillation (a stable notes doc vs. this timestamped snapshot) is still open. Effort: S.
5. **Turn-1 deliverable completeness** — lessons were delivered but the "learn from their engineering practices" half of the question (repo, compat policy, triage) was never researched. Effort: M.

## c) NOT STARTED

1. **Pilot execution** (blocked on Q1, priority conditional).
2. **apidiff baseline + CI gate** (blocked on Q2, priority high — it mechanizes an existing written promise).
3. **CUE-as-OpenAPI-interlingua needs-check** (parked until a-team-openapi-spec is inspected; priority unknown until then).
4. **Decision records** for "no CUE as Go dependency" and "CUE as external tool, conditional" (cheap, waiting on the pilot decision to write once, correctly).
5. **Standalone docs/reviews HTML self-review entry** — deliberately skipped this session per the single-file instruction; convention conflict recorded as Q3.
6. **HARVEST of this report's (f)** into TODO_LIST.md/ROADMAP.md — pending; user instruction is to wait, so it runs on next instruction.

## d) TOTALLY FUCKED UP

Nothing was destroyed — no code or files were touched, so there is no broken state, data loss, or red-tree risk from this session. What IS fucked up is process, and it is named precisely:

1. **One unverified claim shipped as fact.** "cue is in nixpkgs (pin it)" — stated with confidence, never checked. Severity: low today (no one acted on it yet), but it is exactly the verify-before-encoding failure class the fleet has standing rules against. Mitigation: item (f)#2 clears it in one command.
2. **A gating decision was identified and then not asked.** Turn 2 hinged entirely on "is fleet-central policy a goal?" — I wrote it as a caveat and ended with "say the word" instead of forcing the question. Severity: the entire pilot track is blocked on nothing but this omission. Mitigation: asked now as Q1.
3. **Research output had zero durability until the user demanded it.** Two turns of verified findings existed nowhere but scrollback — one context loss away from gone. Severity: medium (wasted work risk), and it violates the memory doctrine ("update at the moment of discovery"). Mitigation: this report + item (f)#20.

## e) WHAT WE SHOULD IMPROVE

1. **"Say the word" endings are decision avoidance.** When a recommendation has exactly one gating unknown, ask it immediately (question tool), don't defer it to a future turn. Impact: this session's main deliverable sat unactionable for its entire life. Fix: rule of thumb — every verdict names its gate in the same message, as a question.
2. **Integration-point claims need existence checks.** Any plan that says "add a step to X / install via Y" gets a 60-second verification of X and Y first. Impact: prevents the nixpkgs-class error and the BuildFlow assumption. Fix: make it part of the recommendation template.
3. **Research sessions must end with a durable artifact.** Even a 10-line notes file beats chat-only. Impact: findings survive context loss; HARVEST has something to pull from. Fix: close every research turn by writing the distillation.
4. **Primary-source breadth before synthesis.** For "what can we learn from X", X's repo/engineering practices are half the answer. Impact: turn 1's answer was deliberately partial. Fix: budget one fetch round for the project's own engineering docs.
5. **Fleet claims need a real census, not glimpses.** `head -40` of ~/projects is not a fleet inventory. Fix: count once, store the list, reference it.

## f) Next tasks (40, session-scoped, HARVEST input)

Impact: Critical/High/Medium/Low · Effort: S <30min, M 30min–2h, L >2h

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Owner ruling: is fleet-central workflow/config policy a goal? (Q1 — gates all of Track A) | Critical | S | Decision |
| 2 | Verify cue packaging: `nix eval nixpkgs#cue.version` (clears session's one unverified claim) | High | S | Quality |
| 3 | Install pinned cue in the dev environment (home-manager profile or flake input) | High | S | Feature |
| 4 | Scaffold central CUE policy module (new repo, `cue mod init`, README stating the one canonical doc home) | Medium | S | Feature |
| 5 | Author `#ReleaseWorkflow` schema: changelog compare-link rule, GO_VERSION/pin format, runner allowlist, concurrency invariants | High | M | Feature |
| 6 | Author `#CIWorkflow` schema covering ci.yml + nightly-fuzz.yml invariants | Medium | M | Feature |
| 7 | Run `cue vet` against httputil's 3 workflows; record findings verbatim | High | S | Quality |
| 8 | Install + run actionlint on the same 3 workflows; build the CUE-vs-actionlint delta matrix | High | S | Quality |
| 9 | Adoption memo: adopt or retire the pilot based strictly on the delta evidence | High | S | Documentation |
| 10 | If adopted: verify BuildFlow supports external-tool steps, then wire the vet in | Medium | M | Feature |
| 11 | If adopted: add a per-repo CI job running the policy vet | Medium | M | Feature |
| 12 | If adopted: roll out to the next 3 repos; measure signal/noise for 30 days | Medium | M | Feature |
| 13 | If adopted: watch CUE releases (v0.18+) for breaking churn | Low | S | Cleanup |
| 14 | If retired: close-out appendix in this report's successor with the evidence | Low | S | Documentation |
| 15 | Verify apidiff works across go.work multi-module (root/httpspec/server_timing baselines) | High | M | Quality |
| 16 | Check whether Go 1.27 ships a first-party API-compat tool before committing to x/exp apidiff | Medium | S | Quality |
| 17 | Generate exported-API baselines at the current released tags of all 3 modules | High | M | Quality |
| 18 | Add an apidiff step to scripts/prerelease-check.sh | High | M | Quality |
| 19 | Add apidiff to CI (placement per Q2) | High | M | Quality |
| 20 | Reference the mechanical gate from docs/v1-stability.md (prose becomes enforced) | Medium | S | Documentation |
| 21 | Distill the 8 CUE lessons into a durable notes doc (not just this timestamped file) | Medium | S | Documentation |
| 22 | Write decision record: "CUE is not a Go dependency here" (depguard + fleet validation-lane policy) | Medium | S | Documentation |
| 23 | Write decision record: "CUE as external tool — conditional pilot" (the adoption gate) | Medium | S | Documentation |
| 24 | HARVEST this report's (f) into TODO_LIST.md / ROADMAP.md (docs-health, on next instruction) | High | M | Documentation |
| 25 | Annotate this report as items complete (docs-health ANNOTATE mode) | Low | S | Documentation |
| 26 | If pilot lands: add the cue commands to AGENTS.md Commands block (single canonical home) | Medium | S | Documentation |
| 27 | If policy repo lands: add it to the meta project inventory | Low | S | Documentation |
| 28 | Census ~/projects properly (full count + list, stored once) | Low | S | Cleanup |
| 29 | Inspect a-team-openapi-spec (size, CI, validation state) to judge the CUE-OpenAPI lane | Medium | S | Quality |
| 30 | Verify BuildFlow's step model for arbitrary external binaries before any integration claim | Medium | S | Quality |
| 31 | Pin the apidiff tool source (nixpkgs vs `go install`) once #16 settles the choice | Medium | S | Quality |
| 32 | Prototype a CSP-conflict detector (Nonce + SecurityHeaders double-CSP made explicit, not docs-only) | Medium | M | Feature |
| 33 | Audit WithContext keys: every classified error names its config field/middleware provenance | Medium | M | Quality |
| 34 | Evaluate an httpspec "strict set" (closed-struct-style full-spec opt-in) — park unless demand appears | Low | S | Feature |
| 35 | Add page+commit provenance to issue links in the website-launch pattern (owner-gated, cross-repo) | Low | S | Documentation |
| 36 | Research CUE's own engineering practices (compat policy, repo hygiene, triage) — the unmined half of turn 1 | Low | M | Documentation |
| 37 | Read the-logic-of-cue concept guide (skipped this session) | Low | S | Documentation |
| 38 | Resolve the self-review format convention (Q3) and record the ruling | Medium | S | Decision |
| 39 | After Q1/Q2 answers: re-baseline this report's (a)–(d) sections | Low | S | Documentation |
| 40 | If CUE adopted anywhere: periodic re-check that the pin still exists in nixpkgs at that version | Low | S | Cleanup |

## g) Questions I cannot answer myself

1. **Is fleet-central workflow/config policy a goal you actually hold?** The red-tag incidents (v1.1.0, v1.4.2) were process failures no YAML schema catches; I verified no cue, no actionlint, no policy infra exists anywhere — so there is no evidence trail telling me whether you *want* central policy or were fine with per-repo process gates. This single answer gates items 1–14.
2. **If the v1-stability promise gets mechanized with apidiff, where does the gate live: per-repo CI or a central BuildFlow step?** I verified the gap is real and unowned, but placement is fleet-architecture preference I should not guess.
3. **Going forward: should brutal self-reviews be separate `docs/reviews/*.html` series entries (skill default) or folded into the status `.md` (today's explicit instruction)?** The two conventions conflict by design; a ruling now prevents the next session from re-hitting this.

---

*Point-in-time snapshot. HARVEST pending (item 24). Annotate, never rewrite (docs-health ANNOTATE convention).*

## Appendix: Rulings recorded (2026-11-02, same session)

1. **Q1 → Yes, run the CUE pilot.** Track A unblocked: items 1–14 proceed (starting with #2 nixpkgs verification, #3 install, #4 policy module scaffold).
2. **Q2 → Both layers.** The apidiff gate lives in per-repo CI AND a central BuildFlow step; items 18–19 are no longer placement-blocked.
3. **Q3 → Separate HTML series.** Brutal self-reviews remain `docs/reviews/<ts>_brutal-self-review.html` per skill default; today's folded-into-status-.md version was a one-off under the single-file instruction. A standalone HTML self-review for this session was NOT backfilled (noted as intentional gap; next session's review opens the series).

---

## Execution appendix (same day, post-ruling Pareto plan run)

Executed per `docs/planning/2026-11-02_05-05_pareto-master-plan_cue-pilot-apidiff.md`.
Item numbers refer to this report's (f) table; hashes are httputil commits
unless the cue-policy or BuildFlow repo is named.

- **DONE — Tier 1 (apidiff gate, both layers):** 15, 16, 31 (feasibility +
  tool decision: apidiff pinned `v0.0.0-20261014143641-f0251eb4aa3e`,
  committed export-data baselines, output-not-exit-code gate — appendix in
  AGENTS Commands block, `6db3763`); 17 (baselines at v1.7.0/v1.0.3,
  `4cbb0c9`); 18, 19a (prerelease gate 3b + ci.yml job + negative test,
  `1a869e0`); 19b, 30 (BuildFlow central step — external: BuildFlow
  `c8eff161f`, edge ledger +2, verified green on this repo); 20
  (v1-stability enforcement section, `f9ae3b5`).
- **DONE — Tier 2 (pilot, ADOPTED):** 2 (nixpkgs cue VERIFIED at 0.17.1 —
  the one-time unverified claim of this report is now evidence); 3
  (installed via nix profile); 4 (policy repo: `LarsArtmann/cue-policy`,
  public, canonical-doc-home decided, `081e3dc` there); 5 (`#ReleaseWorkflow`
  + the REAL finding it caught: missing release concurrency guard, fixed
  `48b664e`); 7, 8 (vet + actionlint delta matrix — second REAL finding:
  `github.head_ref` shell-injection class in ci.yml, fixed same commit);
  9, 14 (adoption memo: **ADOPT** as the complementary external-tool lane;
  cue-policy `b00ef1f`).
- **DONE — Tier 3 (wiring + records):** 10, 11 (BuildFlow `cue-vet` step
  `f263829b7`/`2d6bbee17` + per-repo `policy-vet` CI job `290c0a2`);
  6 (`#CIWorkflow`, cue-policy `90c4183`); 24 (HARVEST into TODO_LIST +
  CHANGELOG, `52ece58`); 21, 22, 23 (decision records ×2 in DECISION_LOG +
  durable lessons doc, `5ac7bc8`).
- **DONE — Tier 4 (rest):** 32 (CSP-conflict detector: request-time
  `security.csp_conflict` warning, behavior unchanged, `3534246`); 33
  (WithContext provenance audit: 60 templates audited, one real gap fixed —
  trusted-proxy CIDR now names its config field, `bc04305`); 28, 29
  (census 348 dirs stored + a-team-openapi-spec verdict: CUE-OpenAPI lane
  NOT NEEDED, `5c50829`); 36, 37 (round-2 research: logic guide + CUE
  project practices, 8 more lessons, `b24f693`); 13, 40, 26, 27 (pin-watch
  protocol + AGENTS cue commands + meta inventory entry, `6db3763` and
  cue-policy `1ea1c3d`); 34, 35 (parked evals resolved in writing:
  strict-set stays DEMAND_GATED, provenance owner-gated, `a97d871`).
- **OPEN (harvested to TODO_LIST, not forgotten):** 12 (fleet rollout wave
  1 — three more repos + 30-day review); the buildflow binary redeploy so
  the store binary gains apidiff + cue-vet; cue-policy SHA-pin bumping as
  an ongoing duty.
- **Closed by owner ruling at read time:** 1, 38.
- **Section re-baseline:** (a) research/verdict unchanged; (b) items 1–3
  are now DONE (pilot executed and adopted; apidiff gate shipped both
  layers; BuildFlow step model verified for real — the report's
  recommendation was validated, including its one wrong assumption about
  config-defined steps: BuildFlow has none, and the step is Go code);
  (c) items 1–6 all executed or closed above; (d) the unverified-claim and
  never-asked-question debts were cleared this session (claim verified
  0.17.1; the gating questions WERE asked and answered via the question
  tool before execution).
