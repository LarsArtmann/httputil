# AGENTS.md Slim Outline — Byte Inventory + Pointer-ification Candidates

_Created: 2026-10-15 02:25 CEST · Inputs: TODO_LIST row 26 (AGENTS size budget: ~48 KB vs the docs-health >30 KB rubric), owner brief row T7 ([2026-10-15_02-12_owner-decision-brief.md](2026-10-15_02-12_owner-decision-brief.md)) · Status: OUTLINE ONLY — execution waits on the owner's pointer-ify vs re-baseline ruling._

## Byte inventory (measured 2026-10-15, `awk` per `##` section; file total 48,175 B)

| Section                    | Bytes  | Share | Every-session value | Verdict                               |
| -------------------------- | ------ | ----- | ------------------- | ------------------------------------- |
| Non-Obvious Behaviors      | 17,031 | 35%   | Partial             | **Candidate 1 (biggest win)**         |
| Commands                   | 12,600 | 26%   | High (canonical)    | **Candidate 2 (slim narrative only)** |
| Testing Conventions        | 6,884  | 14%   | High                | **Candidate 3 (strip provenance)**    |
| Hard Constraints           | 3,895  | 8%    | High (breaks code)  | Keep as-is                            |
| Architecture               | 3,672  | 8%    | Medium              | **Candidate 4**                       |
| Error Model                | 2,354  | 5%    | Medium              | **Candidate 5**                       |
| Pre-Existing Lint Warnings | 757    | 2%    | High                | Keep                                  |
| Accepted Code Duplication  | 503    | 1%    | Medium              | Keep (already a pointer)              |

## The five candidates

1. **Non-Obvious Behaviors → one-line invariants + pointers (~10 KB saved).** The long per-middleware paragraphs (CSRF trust-boundary deep dive, RequestID ring design, CORS PNA spec-churn note, Language negotiation rules, Server listener state machine) each restate material that `docs/architecture-reference.md` or the integration docs already own in full. Keep: the invariant sentence + the "do not simplify this" guard. Move: the design narrative, with a `→ docs/...` pointer per bullet. The do-not-touch guards stay verbatim — they exist to stop future sessions from "fixing" deliberate behavior.
2. **Commands → strip incident narratives, keep every command and the canonical-home erraudit block (~4 KB saved).** The block declares itself the canonical home for erraudit policy — it stays, but its delta-ledger prose (per-commit count history, upstream-bug numbers, verdict-file paths) compresses to: current counts (51/49), the two enforced gates, and pointers to the verdict docs. Same for the BuildFlow section: keep the operational rules, point the incident stories at their status reports (already referenced).
3. **Testing Conventions → drop "born from" incident retellings (~2.5 KB saved).** Each rule keeps its claim + the source pointer it already carries; the multi-sentence incident retellings (what broke, which test line, which session) move behind the existing `docs/status/archived/...` links.
4. **Architecture → collapse the middleware-pattern paragraph into the code-map pointer (~1 KB saved).** The `httpspec` and `server_timing` paragraphs keep one sentence each + pointer; the "why flat" decision keeps its one-sentence form (analysis already linked).
5. **Error Model → deduplicate against the arch-ref classification tables (~1 KB saved).** Keep the Code/Domain definitions and the sentinel rules; the per-family lifecycle mapping and template-sweep checklist collapse to the existing arch-ref pointer with the sweep list kept as a single sentence.

**Projected result:** ~48 KB → ~29-30 KB (under the 30 KB rubric line) without losing a single command, guard, or constraint — every moved paragraph keeps a resolving pointer, so fact-loss risk stays low.

## The single-file-property tradeoff (stated, not decided)

AGENTS.md loads in full into every session; pointer-ification trades guaranteed context for tokens. A pointer the session never follows is silent fact loss — that is exactly what TODO_LIST's fact-loss-audit row (T14, pareto plan) checks for. The outline therefore only pointer-ifies material that is (a) duplicated in arch-ref/integration docs today, or (b) historical narrative whose claim is already stated in the kept sentence.

## Execution preconditions

1. Owner ruling (brief row T7): pointer-ify vs re-baseline vs both.
2. After execution: run the fact-loss audit pass (T14) against the pre-slim version (`git log` holds it) and the full gate battery.
3. Cadence note: the slimmed file re-enters the docs-health VERIFY cycle; any section that regrows past its byte budget gets flagged at the next pass.
