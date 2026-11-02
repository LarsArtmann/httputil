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
