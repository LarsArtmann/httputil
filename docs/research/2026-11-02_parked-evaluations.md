# Parked evaluations — httpspec strict set + website-launch provenance

Date: 2026-11-02. Both items were parked in the CUE-research report's work
pool (items 34 + 35) as Low/S; this note resolves the evaluation honestly
instead of letting them rot as open checkboxes.

## 1. httpspec "strict set" (closed-struct-style full-spec opt-in) — parked, stay parked

**Idea:** a mode where `httpspec.Run` requires the caller to acknowledge the
FULL standard spec set (mirroring CUE's closed-struct semantics: nothing
silent, everything explicit), instead of the current default set + opt-in
extras (`WithExtraSpecs`/`SkipSpec`).

**Evaluation:** the demand signal is absent, and the current design already
covers the real use cases:

- `Run` ships the default convention set; `WithExtraSpecs` adds bundles
  (CORS, RateLimit, PrivateNetwork, Tracing, Language) explicitly — that IS
  the explicit opt-in the idea wants, one level up.
- A "closed" mode would break the frozen v1.0 `Run`/`Option` signatures
  (`docs/v1-stability.md`: both Frozen) if implemented as a signature
  change, or add an Option variant that changes nothing observable (spec
  membership is already caller-controlled).
- Zero importer pull: the discovery push (TODO_LIST row) shows no public
  importers yet; nobody has asked for stricter membership semantics.

**Verdict: keep parked — DEMAND_GATED.** Re-open only when a real consumer
asks for all-specs-or-error semantics, and then implement it additively
(new Option, zero signature change).

## 2. Website-launch issue-link provenance (page + commit) — recorded, owner-gated

**Idea:** issue/file links rendered by the website-launch pattern should
carry their provenance — which page revision and which commit the snapshot
came from — so rendered docs can be traced back to the exact source state.

**Evaluation:** the pattern is real (the website-launch skill renders
project sites and files provenance-carrying artifacts), but the change is
cross-repo (the skill lives in the fleet skill fan-out; consumers are other
projects' sites) and the value is unproven — no rendered site today has
been unable to answer "which commit produced this?" without it. Adding
provenance to every link also grows the rendered surface that then needs
its own freshness discipline.

**Verdict: recorded for the next website-launch session, not actionable
from this repo.** The owner gate stands (item 35's original framing); when
a launch happens, the skill session should decide provenance-per-link vs
provenance-per-build-footer before implementation.
