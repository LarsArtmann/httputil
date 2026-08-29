# Status Report — 2026-08-07 08:39

**Scope:** Review of `~/projects/go-retry` and `~/projects/go-idempotency` for applicability to `httputil`. This report covers **only this session's work and what was noticed in passing** — no fresh codebase-wide scan was performed (per instruction).

**Format note:** Written as Markdown (`.md`) per explicit user request, overriding the status-report skill's canonical HTML dashboard format. Flagged here so the divergence is visible.

---

## Session Summary

The user asked: _"Review go-retry and go-idempotency — any use for them in this repo?"_

**Verdict delivered:** Neither library should be added as a dependency to `httputil` at this time.

- **go-retry** → No fit. Application-layer concern (retrying outbound calls with backoff). No natural integration point in a server-side `func(http.Handler) http.Handler` middleware chain; a "retry middleware" would semantically mean replaying inbound requests through the handler, which is unsafe for non-idempotent methods.
- **go-idempotency** → Tempting pattern (Stripe-style `Idempotency-Key` middleware is a legitimate httputil-shaped concern), wrong time, and not as a hard dependency. Its `Store` only dedupes keys (seen/not-seen + TTL); it does **not** cache the response body needed to replay a prior result, so httputil would have to build the response-caching half regardless. The Store interface is trivially small (~100 LOC) and in-process only; httputil's established plugin pattern (`WriterFactory`, `MetricsRecorder`, `KeyedRateLimiter`) argues for a native interface over an import. And it's scope creep against the v1.0 API freeze.
- **Recommendation filed (verbally):** Track idempotency as a post-v1.0 idea in `ROADMAP.md`; if pursued, define a native `IdempotencyStore` interface rather than importing `go-idempotency`.

---

## a) FULLY DONE

| #   | Item                                                                                        | Notes                                                                                                                                                                                                                                |
| --- | ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| ~~1~~   | ~~Read both repos' README + AGENTS + public exports~~ done at `510f06d` | ~~`go-retry`: `Do`/`Config`/`Backoff`/`ComputeDelay` + `ErrExhausted`/`ErrCanceled`, `go-error-family`-based. `go-idempotency`: `Store` iface, `MemoryStore` (`Seen`/`Record`/`CheckAndRecord`/`Close`), `ErrDuplicate` as `Conflict`.~~ |
| ~~2~~   | ~~Cross-checked httputil's ROADMAP / FEATURES / TODO for existing idempotency/retry mentions~~ done at `510f06d` | ~~Only hits were `Retry-After` headers in rate limiting and `httpspec` rate-limit specs — unrelated to these libs. No prior idempotency-key work exists.~~ |
| ~~3~~   | ~~Produced a categorized recommendation (no-fit / pattern-fit-but-defer / plugin-over-import)~~ done at `510f06d` | ~~Delivered in chat with reasoning grounded in httputil's dependency policy and plugin conventions.~~ |

## b) PARTIALLY DONE

| #   | Item                                    | What's missing                                                                                                                                                 |
| --- | --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~1~~   | ~~**ROADMAP note for idempotency**~~ done at `a5e9944` | ~~I **recommended** "file idempotency as a post-v1.0 idea in `ROADMAP.md`" but **did not actually write it**. This is the session's primary gap — see (d) below.~~ |
| ~~2~~   | ~~Applicability assessment for `go-retry`~~ done at `a5e9944` | ~~Analysis is complete but I did not record the "non-goal" rationale anywhere durable (e.g. a ROADMAP non-goal entry), so the reasoning is session-only.~~ |

## c) NOT STARTED

- No code changes anywhere (this was a review task — correct to not edit code).
- No `ROADMAP.md` / `FEATURES.md` / `TODO_LIST.md` edits made.
- No verification of whether a sibling `go-etag`-style extraction pattern would apply to an idempotency middleware (mentioned `go-etag` in passing but did not confirm its shape).
- No `httpspec` spec design sketched for an idempotency-key middleware (would be the natural test-home if built).

## d) TOTALLY FUCKED UP

**Nothing is broken.** But one self-inflicted gap worth naming explicitly:

> **I recommended an action ("file idempotency as a post-v1.0 idea in `ROADMAP.md`") and then stopped instead of doing it.** This violates the "never describe what you'll do next — just do it" principle. The recommendation was framed as work for the user to do, when it was a one-line, low-risk, reversible doc edit I should have performed myself and then reported. The only reason to pause would have been if ROADMAP edits were out of scope — but the user asked for a review **and** a recommendation, so landing the durable artifact was in scope.

Secondary, lesser misses:

- I asserted "you already have `ResponseRecorder` for the response-caching half" of idempotency without checking that `ResponseRecorder` can actually **replay** a captured response back to a client (it records status/headers/body; it is not designed as a replay/cache primitive). The claim was directionally reasonable but overstated — I did not verify the replay path.
- I did not confirm whether `go-error-family` (shared dependency across all three projects) would make a hypothetical go-idempotency dep more palatable on classification-alignment grounds. Minor, but it's a real consideration I skipped.

## e) WHAT WE SHOULD IMPROVE

1. **Close the recommendation→action loop.** When a review produces a concrete, low-risk, reversible next action (add a ROADMAP line, open a doc note), perform it in the same turn rather than handing it back as homework. Reserve "ask first" for genuinely ambiguous/irreversible decisions.
2. **Verify load-bearing claims before asserting them.** The `ResponseRecorder`-as-response-cache line is the example: I should have either read `recorder.go` to confirm replay semantics or hedged the claim explicitly.
3. ~~**Record "non-goal" decisions durably.** The ROADMAP already has a strong Non-goals section (HTTP/2 push, internal `compress/`, brotli, etc.). "No retry middleware" and "idempotency deferred to post-v1.0" both belong there so future sessions don't re-litigate them. Right now the rationale lives only in chat.~~ done at `a5e9944`
4. **Cross-project dependency map.** All three projects (`httputil`, `go-retry`, `go-idempotency`) sit on `go-error-family`, and `go-retry`/`go-idempotency` are CQRS-family siblings. A one-line note in AGENTS.md linking the sibling ecosystem would help future sessions answer "is this a dep candidate?" faster.

## f) Top things to get done next

> Per instruction, scoped to what surfaced this session + what I noticed in the already-read `ROADMAP.md`. Items marked **[ROADMAP fuel]** are raw ideas, not commitments — they belong in `ROADMAP.md`, not `TODO_LIST.md`. A deeper codebase-wide backlog was **not** generated (user scoped this to the session).

### Direct session follow-ups (high confidence)

1. ~~**Add the ROADMAP note for idempotency** (the action I skipped). Post-v1.0 idea, native `IdempotencyStore` interface, not an import.~~ done at `a5e9944`
2. ~~**Add "no retry middleware" and "idempotency deferred to post-v1.0" to ROADMAP Non-goals** so the rationale is durable.~~ done at `a5e9944`
3. ~~**Verify `ResponseRecorder` replay semantics** (`recorder.go`) — can it serve as the response-cache half of an idempotency middleware, or is a separate cache type needed? Record the answer in the ROADMAP/ADR.~~ done (answered in ROADMAP — ResponseRecorder is not a replay primitive; a separate cache type is needed)
4. **Skim `~/projects/go-etag`** to confirm whether the extracted-module pattern (how ETag was spun out) is the right template for a future idempotency extraction if it doesn't live in core.

### v1.0 freeze track (noticed in ROADMAP, session-relevant because dep decisions hinge on it)

5. **Remove deprecated `TokenBucketLimiter` / `RateLimiter` interface** (pre-v1.0 cleanup, ROADMAP-flagged).
6. ~~**Evaluate `AllowN` + `context.Context` cancellation on `KeyedRateLimiter`** (ROADMAP-flagged, may shape the final v1.0 interface).~~ done (evaluated — AllowN rejected, context.Context cancellation deferred to v1.0 (ROADMAP))
7. **Run one stabilization cycle** then cut v1.0 with the API-stability doc (`docs/v1-stability.md`).
8. ~~**Write the idempotency decision as an ADR** (defer-to-post-v1.0) so v1.0's scope boundary is explicit and reviewable.~~ done (decision recorded in ROADMAP post-v1.0 ideas, satisfying the scope-boundary intent)

### Smaller doc/maintenance items I noticed in passing

9. **Update AGENTS.md sibling-ecosystem note** — link `go-retry`/`go-idempotency`/`go-etag`/`go-error-family` as the cross-project family.
10. **ROADMAP "Updated" date** is 2026-08-07 (current) — good; keep it current as decisions land.
11. ~~**[ROADMAP fuel]** Design sketch: `IdempotencyKeyMiddleware` config struct mirroring the established `*Config` + `Validate()` + `Default*Config()` pattern.~~ done (filed as ROADMAP post-v1.0 idempotency idea)
12. ~~**[ROADMAP fuel]** Define `IdempotencyStore` interface (native) — likely `Get(ctx, key)`, `Save(ctx, key, resp, ttl)`, with `MemoryStore` + Redis plugin docs in `docs/integrations/`.~~ done (filed as ROADMAP post-v1.0 idempotency idea)
13. ~~**[ROADMAP fuel]** Add `httpspec` specs for an idempotency middleware (409 on duplicate, replayed status/headers match original, `Idempotency-Key` header handling).~~ done (filed as ROADMAP post-v1.0 idempotency idea)
14. ~~**[ROADMAP fuel]** Decide store-eviction semantics: reuse `KeyedRateLimiter`'s O(log n) min-heap + TTL pattern for consistency.~~ done (filed as ROADMAP post-v1.0 idempotency idea)
15. ~~**[ROADMAP fuel]** Response-cache keying: hash of method + path + body, or raw client-supplied key only?~~ done (filed as ROADMAP post-v1.0 idempotency idea)
16. ~~**[ROADMAP fuel]** Interaction with `Compression` — replayed cached responses must re-negotiate `Accept-Encoding`.~~ done (filed as ROADMAP post-v1.0 idempotency idea)
17. ~~**[ROADMAP fuel]** Interaction with `MaxBodySize` — body must be captured for hashing before the size limit rejects it.~~ done (filed as ROADMAP post-v1.0 idempotency idea)
18. ~~**[ROADMAP fuel]** TTL vs. idempotency-window semantics — Stripe uses client-supplied; define whether httputil allows server override.~~ done (filed as ROADMAP post-v1.0 idempotency idea)

### Hard cap honesty

Beyond ~18 items I'd be padding. The user allowed up to 50 but also said "report based on this current sessions run and what you noticed" and "DO NOT RESEARCH OTHER STUFF UNRELATED." A legitimate 50-item list would require a full codebase scan, which is out of scope. The items above are what the session genuinely surfaced.

## g) Questions I can NOT figure out myself

1. ~~**Do you want me to land the ROADMAP edits now** (the idempotency post-v1.0 note + the two Non-goals entries), or were you treating this purely as analysis? I assumed "review + recommend" meant "also durably record," but ROADMAP edits are yours to approve — I won't mutate project docs without confirmation given the v1.0 freeze sensitivity.~~ done (answered — ROADMAP edits landed (a5e9944))
2. ~~**Is the v1.0 stabilization window still accepting new ADRs/scope-boundary docs**, or is it strictly lock-down from here? The ROADMAP says "one stabilization cycle before the commitment" — I can't tell from the text whether that cycle has started or whether design docs (like the idempotency-defer ADR) are still welcome.~~ done (answered — stabilization window still accepting scope-boundary docs)
3. ~~**For idempotency, if/when pursued: native interface vs. exception import?** I recommended native (consistent with the plugin pattern), but the `justinas/nosurf` precedent shows this project _does_ make exceptions for correctness/security-critical libraries it doesn't want to hand-roll. Idempotency is correctness-critical — so is there a threshold (security? concurrency-hardness?) where you'd accept `go-idempotency` as a fourth exception dep rather than reinvent the store?~~ done (answered — native interface preferred; nosurf stays the exception)

---

_Report scope: this session only. No codebase-wide scan performed._

---

## Resolution (2026-08-07 docs-health pass; upgraded to per-item markers 2026-08-29)

Every numbered item is resolved inline; unmarked items are still open by convention. The header banner was removed — its verdicts live on the items themselves (ROADMAP landings: `a5e9944`).

Open as of 2026-08-29: f4 (go-etag pattern skim — deferred design research), f5 (TokenBucketLimiter removal — ROADMAP v1.0), f7 (stabilization cycle then v1.0 cut — not yet tagged), f9 (AGENTS.md sibling-ecosystem note), f10 (keep ROADMAP "Updated" date current — ongoing practice, not closable). Items e1/e2/e4 and the d) post-mortem are narrative session facts and intentionally unmarked. The f11–f18 [ROADMAP fuel] items were filed under ROADMAP's post-v1.0 idempotency idea, which links back to this report.
