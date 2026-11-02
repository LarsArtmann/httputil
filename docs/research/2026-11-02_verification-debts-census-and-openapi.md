# Verification debts cleared — fleet census + a-team-openapi-spec verdict

Date: 2026-11-02 (clears items 28 + 29 of the CUE-research report's work
pool; see `docs/status/2026-11-02_04-56_cue-research-and-adoption-decision.md`).

## Fleet census (r1)

`~/projects` holds **348 directories** as of this date; the full sorted list
is stored next to this note
([2026-11-02_fleet-projects-census.txt](2026-11-02_fleet-projects-census.txt))
so future "does the fleet already have X" questions have a point-in-time
baseline instead of a memory claim. The census is a snapshot — re-run
`ls ~/projects | sort` when precision matters.

## a-team-openapi-spec inspection (r2 + r3)

What the repo is: a Bun/TypeScript utility pipeline (`har-to-openapi`,
`js-yaml`, `clean-openapi-spec.ts`, `convert-har-to-openapi.ts`) that
captures real A.Team Platform traffic (HAR files) and converts it into an
OpenAPI 3.0 document (`a-team-api-clean.yaml`), plus a consumer script
(`a-team-profile-sync.ts`).

**Verdict: the CUE-as-OpenAPI-interlingua lane is NOT NEEDED — close the
needs-check with evidence, no further research owed.** The reasoning:

1. The spec is captured OUTPUT, not authored source. CUE's OpenAPI value
   proposition (schemas as the source of truth, OpenAPI as an export; or
   OpenAPI validated as CUE data) applies when humans maintain the schema.
   Here the schema is reverse-engineered from traffic — the "truth" is the
   traffic, and no maintenance drift exists for CUE to prevent.
2. Single consumer, single producer, no multi-source unification problem —
   the lane where CUE's unification semantics pay off does not exist here.
3. The repo is TypeScript/Bun; introducing CUE would add a foreign tool to
   a pipeline whose language ecosystem (TS tooling) already covers it.

Re-open only on a demand signal: if the A.Team spec becomes hand-maintained
(evolution instead of capture), or a second consumer needs to validate
payloads against it, the `cue import`/`cue vet` path over OpenAPI becomes
worth a pilot.
