# CUE Engineering Lessons — durable notes

Distilled from the primary-source research (6 cuelang.org pages, 2026-11-02;
lesson names and mappings as recorded in
[the adoption status report](../status/2026-11-02_04-56_cue-research-and-adoption-decision.md)),
updated with what the same-day pilot actually demonstrated. Each lesson ends
with its fleet application — the part that outlives the CUE research.

1. **The GCL one-way-door lesson.** CUE's lineage (GCL, the config language
   inside Google) shows what happens when configuration languages grow
   complexity without a model: they accrete features until they become a
   programming language nobody can safely remove anything from. CUE's design
   is the countermeasure — a small core (unification + types-are-values)
   chosen deliberately, pre-1.0, with the freedom to still say no.
   *Fleet application:* the fleet's "config tools own their formats; we don't
   embed config engines in Go" stance is this lesson operationalized.

2. **No silent overrides.** CUE disallows override entirely — two values
   either unify or the program fails; last-write-wins does not exist.
   *Fleet application:* the same instinct behind `DenyUnmatched: true`
   defaults and fail-closed CSRF origin parsing: ambiguity is a bug class,
   not a convenience. Prefer designs where conflicting inputs cannot merge
   silently.

3. **Validation as a type system.** In CUE, constraints ARE types
   (`#Schema` definitions); validating is unifying, and "is A a subtype of
   B" is decidability over the value lattice. Validation is not a gate at
   the edge — it is the type system of the data.
   *Fleet application:* httputil's `Validate()`-at-construction pattern and
   the fleet's make-impossible-states-unrepresentable rule are the Go-side
   expression of the same idea. For untyped data (YAML workflows), the
   `#ReleaseWorkflow`/`#CIWorkflow` schemas give that data a type system it
   never had.

4. **Error provenance.** CUE's evaluation errors name the conflicting
   fields and the constraint that failed — the diagnostic carries its own
   provenance. *Fleet application:* the go-error-family `WithContext`
   contract (every classified error names its config field/middleware) is
   the same principle on the runtime side.

5. **Gradual adoption.** CUE is designed to enter a codebase as a tool
   before it becomes a dependency (validate existing files first; adopt the
   API later, if ever). *Fleet application:* exactly the adopted lane —
   external `cue vet` on workflows, zero Go dependency, depguard untouched.
   The pilot proved the ramp: schema → findings → wiring, in one session.

6. **Shift-left + spec containment.** CUE's compatibility story is
   machine-checkable ("is every instance of A also an instance of B" —
   containment over value sets) instead of prose. *Fleet application:* this
   idea directly produced the apidiff gate: the frozen-API promise is now
   containment over Go's exported-API sets, machine-checked against
   committed baselines. Written promises get a mechanical twin.

7. **Machine-first design.** CUE ships as a tool with stable, scriptable
   semantics (vet, fmt, export) before polished UIs — designed for pipelines
   and automation consumers first. *Fleet application:* every fleet gate is
   a CLI with stable exit/output contracts first (apidiff's report-as-gate,
   actionlint's exit codes); human-friendly rendering is a layer on top,
   never the interface.

8. **Configuration ≠ computation.** CUE keeps configuration declarative —
   logic lives in the unification model, not in imperative scripts; there is
   no general-purpose escape hatch into programming inside a config file.
   *Fleet application:* workflow YAML stays data; the policy that constrains
   it lives in schemas (data about data), not in more scripting around the
   files.

**Pilot addendum (evidence added same day):** the two real findings of the
delta matrix (missing release concurrency guard — CUE's catch;
`github.head_ref` shell injection — actionlint's catch) are lessons 3 and 4
made concrete: the missing *type* on `release.yml` and the unprovenanced
trust of an attacker-controlled field.

## Round 2 (same day): the logic + the project's own practices

Sources read in full: the Logic of CUE concept guide (cuelang.org,
2026-05-23 revision) and the cue-lang/cue CONTRIBUTING.md + triage labels
(main @ 2026-11-02 shallow clone).

9. **Backwards compatibility IS subsumption.** The logic guide states the
   API-compat metaphor as a theorem, not an analogy: a new version is
   backwards compatible iff the OLD version is an instance (subsumed by) the
   new one — "a new version may not forbid what was allowed". Optional
   fields may become required but never vanish. This is the theoretical
   twin of the apidiff gate: Go's exported-API sets play the role of the
   lattice, and our gate fails exactly when the containment breaks.
   *Fleet application:* when documenting breaking-change rules, state them
   as a set-containment property that a machine can check — it worked for
   the frozen-API promise.

10. **Defaults need order-independence machinery, or they become solver
    bait.** CUE keeps two parallel values (all-possible vs default) per
    field and REFUSES to pick a default when picks would interact —
    choosing either would make the result depend on implementation mood
    and edge toward NP-complete constraint solving. Unification stays
    pseudo-linear because of this refusal.
    *Fleet application:* when a default interacts with another default
    through a constraint, the correct behavior is to fail loudly rather
    than pick; httputil's validate-and-log constructor posture already
    embodies "no silent picking".

11. **Optional-field conflicts drop the field — by logic, not convenience.**
    Conflicting values for an optional field infer "not that field" (from
    P→⊥), while required-field conflicts fault the struct. Fitting
    closedness into the lattice FORCED this distinction, and the logic
    justifies it.
    *Fleet application:* distinguish "reject" from "treat as absent" only
    where the type system justifies it; don't hand-wave the difference.

12. **Cycles are equations, not errors.** `a: b+1; b: a-1; b: 1` resolves
    to a=2 by plain logic (the cycle forces a-1 & 1, which is bottom unless
    a-1 is 1) — no algebraic solver. Most non-infinite cycles resolve in
    bounded time by design.
    *Fleet application:* before banning recursion in config or template
    systems, check whether the equations actually determine values; a ban
    that exists only because the evaluator is eager is an implementation
    limit dressed as a rule.

13. **Constraints simplify; assertions accrete.** CUE simplifies
    `>=3 & <=10` unified with `>=5 & <=20` to `>=5 & <=10`; GCL/Jsonnet
    assertion piles grow instead, decoupled from their fields — hard to
    discover, hard to reason about.
    *Fleet application:* prefer intersectable constraint representations
    (ranges, typed enums) over piled-up boolean checks; in Go, that argues
    for typed values whose constructors compose rather than ad-hoc
    `if` chains.

14. **The project gates design at issues, not PRs.** CONTRIBUTING: "open an
    issue before writing code"; an approach is agreed when a maintainer
    says so or labels it NeedsFix; PRs without an agreed approach "may not
    be reviewed, particularly a large or AI-generated one".
    *Fleet application:* the fleet's owner-ruling-before-execution gate is
    the same shape; for upstream contributions, expect design agreement —
    not code volume — to be the review currency.

15. **Unity is their compat canary fleet.** CUE invites real projects into
    a program (cue.dev/products/unity) whose purpose is testing CUE changes
    against member projects — the same pattern as our BuildFlow dogfooding
    and the wave-1 rollout: a tool's compatibility is proven by running it
    on real consumers, continuously.
    *Fleet application:* when adopting a tool fleet-wide, join (or build)
    its early-feedback channel; being a canary converts upstream breakage
    from surprise to signal.

16. **Triage taxonomy as workflow contract.** Labels Triage →
    NeedsInvestigation → NeedsDecision → NeedsFix make each issue's next
    ACTION explicit and owner-able. Workflow labels that name the blocking
    step (not the topic) let contributors pick work matching what they can
    unblock.
    *Fleet application:* the TODO_LIST " — BLOCKED: <reason>" convention
    already does this; label-shaped next-actions beat topic-shaped ones.
