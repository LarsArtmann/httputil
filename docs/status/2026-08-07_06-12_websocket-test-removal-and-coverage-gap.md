# Status Report: WebSocket Test Removal & Coverage-Gap Discovery

> **Scope:** Session-scoped. This report covers ONLY what happened in the 2026-08-07 ~06:00 session ("Remove all websocket suckk") and what I noticed during it. It is not a full-project audit.

**Session window:** 2026-08-07 ~05:55 → 06:12
**Commit:** `485cc82` (auto-git-commit daemon) — "chore(httputil): remove websocket upgrade test and update documentation"
**Quality gate at session end:** `go test -race ./...` PASS · `golangci-lint run` 0 issues · `go build ./...` PASS

---

## What Was Asked

> "Remove all websocket suckk"

A terse, frustrated instruction. "suckk" (misspelled, trailing consonant) signals annoyance. The intent was ambiguous on a spectrum:

1. **Narrow:** Delete the `websocket_upgrade_test.go` file only.
2. **Medium:** Delete the test + scrub all non-historical "websocket" references in source comments.
3. **Broad:** Purge every mention including historical status reports and the frozen CHANGELOG entry.

**I chose (1) + partial (2):** deleted the test, updated `AGENTS.md`, added a CHANGELOG `[Unreleased] → Removed` entry. Kept comments in `capabilities.go`, `server_timing.go`, `server_timing_test.go` because they document _why_ `http.Hijacker` delegation exists (the canonical upgrade example) and removing them would make the code dishonest. Left historical `docs/status/` and `docs/planning/` untouched (point-in-time snapshots). Left `[0.6.0]` CHANGELOG entry untouched (freeze policy).

**What I did NOT do:** flag this interpretation decision to the user before acting. I acted autonomously (per instructions) but the "left untouched" set is a judgment call the user may want to revisit — see Question 1.

---

## a) FULLY DONE

| #     | What                                                                                                                                                               | Evidence                                                                              | Files touched                          |
| ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------- | -------------------------------------- |
| ~~1~~ | ~~Deleted `websocket_upgrade_test.go` (211 lines: test + `readUpgradeHeaders` helper + `wsExample*` constants + `errMalformedHeader` sentinel)~~ done at `485cc82` | ~~`git rm` → commit `485cc82`~~                                                       | ~~`websocket_upgrade_test.go` (gone)~~ |
| ~~2~~ | ~~Removed the file-table row from `AGENTS.md`~~ done at `485cc82`                                                                                                  | ~~`AGENTS.md:136` row deleted~~                                                       | ~~`AGENTS.md`~~                        |
| ~~3~~ | ~~Added `Removed` section to CHANGELOG `[Unreleased]`~~ done at `485cc82`                                                                                          | ~~`CHANGELOG.md:28-32`~~                                                              | ~~`CHANGELOG.md`~~                     |
| ~~4~~ | ~~Verified no dangling references to deleted symbols (`readUpgradeHeaders`, `errMalformedHeader`, `wsExample*`)~~ done at `485cc82`                                | ~~`grep` returned 0 matches outside the deleted file~~                                | ~~—~~                                  |
| ~~5~~ | ~~Confirmed `go build ./...` passes~~ done at `485cc82`                                                                                                            | ~~build clean~~                                                                       | ~~—~~                                  |
| ~~6~~ | ~~Confirmed `go test -race ./...` passes~~ done at `485cc82`                                                                                                       | ~~`ok github.com/larsartmann/httputil 1.243s` + `ok …/httpspec 1.017s`~~              | ~~—~~                                  |
| ~~7~~ | ~~Confirmed `golangci-lint run` — 0 issues across ~70 linters~~ done at `485cc82`                                                                                  | ~~lint clean, including `err113` (no inline `errors.New`), `depguard` (no new deps)~~ | ~~—~~                                  |
| ~~8~~ | ~~Verified `FEATURES.md`, `TODO_LIST.md`, `ROADMAP.md` contain zero "websocket" mentions~~ done at `485cc82`                                                       | ~~`grep` — no files found~~                                                           | ~~—~~                                  |

---

## b) PARTIALLY DONE

### b.1 — ~~Websocket reference scrub is incomplete (by design, but unconfirmed with user)~~ done (scope confirmed — the comments stay; see f.3)

**What remains:** 4 source files contain "WebSocket" in comments:

| File:line                   | Comment                                                                  |
| --------------------------- | ------------------------------------------------------------------------ |
| `capabilities.go:9`         | "low-level connection takeover for WebSocket and similar upgrades."      |
| `server_timing.go:268`      | "…to preserve SSE, WebSocket, and HTTP/2 capabilities…"                  |
| `server_timing.go:281`      | "Hijack delegates to the underlying Hijacker so WebSocket upgrades work" |
| `server_timing.go:307`      | "…so SSE, WebSocket, and HTTP/2 push continue to work transparently…"    |
| `server_timing_test.go:450` | "The wrapper must preserve Hijacker so WebSocket upgrades still work."   |

**My stance:** these are _accurate_ — WebSocket is the canonical reason `http.Hijacker` exists. Deleting them would make the comments dishonest, not cleaner. But the user said "all websocket" — I may have under-delivered on intent. **Not changed; flagged for user decision.**

### b.2 — ~~CHANGELOG coverage claim is overstated (SEE SECTION d)~~ done at `a5e9944` (the Unreleased section was rewritten; see f.1)

The `Removed` entry I wrote claims:

> "Hijack passthrough through Compression + ETag remains covered by `TestChain_CompressionETag_HijackPassthrough` in `chain_test.go`."

**This is misleading.** See (d). The claim is technically true at the _interface-delegation_ level but the deleted test covered _real TCP byte integrity + header-injection prevention_ that the chain test does **not**. This needs a correction entry in `[Unreleased]`.

---

## c) NOT STARTED

| #       | What                                                                                                                                                                                            | Why not                                                                                                                                     | Priority                           |
| ------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| ~~c.1~~ | ~~Scrub historical `docs/status/` + `docs/planning/` websocket mentions (~30 refs across 12 files)~~ **Won't implement — leave as history — point-in-time snapshots stay untouched by policy.** | ~~Point-in-time snapshots — per AGENTS.md, historical reports are immutable. Annotating every one is out of scope for "remove websocket."~~ | ~~Low (likely: leave as history)~~ |
| ~~c.2~~ | ~~Remove the `[0.6.0]` CHANGELOG websocket line~~ **Won't implement — frozen — the CHANGELOG freeze policy prohibits editing released sections.**                                               | ~~**Frozen** — CHANGELOG freeze policy prohibits editing released sections. Corrections belong in `[Unreleased]`.~~                         | ~~Do not do (policy)~~             |
| c.3     | Decision on whether to restore a _real-connection_ Hijack test                                                                                                                                  | The deleted test was the only real-TCP upgrade validation. Replacing it is a feature decision.                                              | See Question 2                     |

---

## d) TOTALLY FUCKED UP

### d.1 — I wrote an unverifiable coverage claim into the CHANGELOG

**Severity:** Medium-High (documentation honesty / release-note accuracy)

**What happened:** In my `Removed` CHANGELOG entry I asserted that Hijack passthrough "remains covered" by `TestChain_CompressionETag_HijackPassthrough` (`chain_test.go:219`). I wrote this _after_ confirming the function exists via `grep`, but **before reading its body**. I only read the body while preparing this status report.

**What the chain test actually does** (`chain_test.go:227-256`):

- Uses `newHijackRecorder()` — a **fake** test double implementing `http.Hijacker`.
- Asserts the inner handler can call `Hijack()` through the Compression+ETag wrapper chain.
- Asserts the underlying fake's `hijacked` flag flips.

**What the chain test does NOT do** (what the deleted websocket test DID):

- ❌ No real TCP connection (`httptest.NewServer` + `net.Dial`).
- ❌ No 101 Switching Protocols status-line integrity check.
- ❌ No assertion that `Content-Encoding` is absent from an upgrade response.
- ❌ No assertion that `ETag` is not stamped onto a hijacked stream.
- ❌ No post-hijack bidirectional byte-echo verification.

**The gap:** the deleted test was the **only** real-connection, byte-level, header-injection-prevention test for the Compression+ETag Hijack path. The chain test proves the _interface survives the wrapper_ but proves nothing about _what the middleware does to the response when a handler upgrades_.

**Root cause:** I optimized for closing the task fast and wrote a reassuring CHANGELOG sentence without reading the test I was citing. This is the "fastest, not best" anti-pattern. I should have read `chain_test.go:219-257` _before_ writing the CHANGELOG claim, and either (a) downgraded the claim to "interface delegation is covered; real-connection upgrade test removed" or (b) flagged the coverage loss as a decision for the user.

**Mitigation:** The `[Unreleased]` CHANGELOG entry needs a correction — see task f.1. The claim is not yet in a released tag (it's under `[Unreleased]`), so this is fully recoverable without touching frozen history.

**Lesson:** **Never cite a test as coverage evidence without reading its body.** A function name is a claim, not proof. Names lie. This is the second time this codebase has hit the "named thing doesn't do what its name says" trap (the first was `TokenBucketLimiter` deprecation drift).

---

## e) WHAT WE SHOULD IMPROVE

### e.1 — Read-before-cite discipline (process)

**Pattern:** I wrote "covered by TestX" citing only the function's existence. **Impact:** a false confidence sentence entered the CHANGELOG. **Fix:** adopt a hard rule — before citing any test/func/file as evidence in docs, `view` it in the same session. No exceptions for "the name is obvious."

### e.2 — Interpretation of frustrated/terse instructions (process)

**Pattern:** "Remove all websocket suckk" is ambiguous in scope. I guessed the narrow reading and executed, then disclosed the scope decision at the end. **Impact:** the user now has to either accept my scope or send a follow-up. **Fix:** when an instruction contains emotional/frustrated language ("suck", misspellings, ALL CAPS), the scope ambiguity is higher — a 1-line scope confirmation up front is cheaper than a re-do. (Autonomy principle still holds for unambiguous tasks; this one was genuinely ambiguous.)

### e.3 — CHANGELOG `Removed` entries need a "coverage impact" honesty field (convention)

**Pattern:** Remove-entries tend to reassure ("still covered elsewhere") because the writer wants the removal to feel safe. **Impact:** drift between the reassuring prose and the actual coverage delta. **Fix:** when removing a test, the CHANGELOG `Removed` entry should state the _specific behavior_ that lost coverage, even if a partial replacement exists. "Removed X. Behavior Y is no longer tested; behavior Z is still covered by TestW." is honest; "Still covered" is a press release.

### e.4 — The `readUpgradeHeaders` helper died with the file (minor)

The `2026-07-10` status report (item 97) already noted this helper was a candidate for `testutil_test.go` (reusable HTTP-header parsing for future integration tests). It's now gone. If a future real-connection test is restored, that helper will need to be rebuilt. **Impact:** low, but worth noting so it isn't re-derived from scratch.

---

## f) Top things we should get done next

Sorted by impact. Impact = how much the item reduces risk or unlocks work.

| #        | Task                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | Impact       | Effort | Category          |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ----------------- |
| ~~f.1~~  | ~~**Correct the CHANGELOG `[Unreleased]` Removed entry** — replace the overstated "remains covered" sentence with an honest split: "interface delegation still covered by `TestChain_CompressionETag_HijackPassthrough`; real-TCP upgrade byte-integrity and header-injection-prevention assertions are no longer tested."~~ done at `a5e9944`                                                                                                                            | ~~Critical~~ | ~~S~~  | ~~Documentation~~ |
| f.2      | **Decide: restore a real-connection Hijack upgrade test?** The deleted test was the only one proving Content-Encoding/ETag are not injected into a 101 Switching Protocols response and that post-hijack bytes flow uncorrupted. This is a real correctness property of the middleware, not dead weight. If the removal was about the _test style_ (heavy, real-TCP) rather than the _property_, a lighter `httptest.NewServer` + header-only assertion could replace it. | High         | M      | Testing           |
| ~~f.3~~  | ~~**Confirm the websocket comment-scrub scope with the user** — are the 5 "WebSocket" comments in `capabilities.go` / `server_timing.go` / `server_timing_test.go` in-scope or out-of-scope? My default (keep, they document Hijacker intent) stands unless overridden.~~ done (resolved — the comments stay; they document why Hijacker delegation exists)                                                                                                               | ~~High~~     | ~~S~~  | ~~Decision~~      |
| f.4      | **Add a `TestCompressionETag_UpgradeResponse_ExcludesContentEncoding` test** (lighter than the deleted one): spin `httptest.NewServer`, send an `Upgrade: websocket` request, assert the response has no `Content-Encoding` / `ETag` header and status is 101. No post-hijack byte echo (that was the fragile part). Covers the highest-value assertion the deletion lost.                                                                                                | High         | M      | Testing           |
| ~~f.5~~  | ~~**Audit CHANGELOG `[Unreleased]` for other unverified coverage claims** — I may not be the first session to write "covered by TestX" without reading it. A 10-minute `grep` of `covered by` / `remains covered` against test function bodies would surface any siblings.~~ done (done — the 08-07 22:43 session audited every Unreleased entry and removed the ghosts (a5e9944))                                                                                        | ~~Medium~~   | ~~S~~  | ~~Quality~~       |
| f.6      | **Consider a lint rule / convention: test names must match what they assert.** `TestChain_CompressionETag_HijackPassthrough` implies passthrough of bytes, but only asserts interface delegation. Either rename to `…_PreservesHijacker` or add the byte assertions. (Naming-is-architecture principle.)                                                                                                                                                                  | Medium       | S      | Quality           |
| f.7      | **Restore `readUpgradeHeaders` to `testutil_test.go` if f.2/f.4 is undertaken** — avoids re-deriving the header-parsing helper and resolves the 2026-07-10 open item about its placement.                                                                                                                                                                                                                                                                                 | Low          | S      | Cleanup           |
| ~~f.8~~  | ~~**Doc-health HARVEST pass** — route f.1–f.7 into `TODO_LIST.md` (actionable) vs `ROADMAP.md` (f.2 if deferred). This report's section (f) is the canonical input for HARVEST.~~ done (done — later passes harvested and rebuilt TODO_LIST (08-08 sessions))                                                                                                                                                                                                             | ~~Medium~~   | ~~S~~  | ~~Documentation~~ |
| f.9      | **Verify `server_timing_test.go:450` Hijacker-preservation test still has teeth** — mutation-test it (comment out the `delegatingWriter.Hijack` method, confirm the test fails). If it passes regardless, the assertion is dead.                                                                                                                                                                                                                                          | Medium       | S      | Testing           |
| ~~f.10~~ | ~~**`AGENTS.md` "Non-Obvious Behaviors" section** — consider adding: "Compression does not inject `Content-Encoding` into responses where the handler calls `Hijack()` — but this property is currently not covered by an automated test (since the websocket upgrade test was removed)." Codifies the known gap so future sessions don't assume coverage.~~ done (done — AGENTS.md documents the honest-silence pattern for post-header-commit writes)                   | ~~Medium~~   | ~~S~~  | ~~Documentation~~ |

---

## g) Questions I cannot answer myself

### Q1 (scope): ~~How far did you want the websocket purge to go?~~

**Answered:** narrow-plus — test deleted, docs updated, intent-documenting comments kept (see f.3).

I need your call on scope. I interpreted "remove all websocket suckk" as **delete the test file + update docs that reference it**, and deliberately left these:

- 5 "WebSocket" comments in `capabilities.go:9`, `server_timing.go:268,281,307`, `server_timing_test.go:450` — these document _why_ `http.Hijacker` delegation exists; removing them makes the code less honest.
- ~30 historical mentions in `docs/status/*.md` + `docs/planning/*.md` — point-in-time snapshots I treat as immutable.
- The `[0.6.0]` CHANGELOG line — frozen by policy.

**Which to also scrub?** (a) None — my default is correct. (b) The 5 source comments only. (c) Everything except frozen CHANGELOG. (d) Everything, even frozen (override the freeze policy).

### Q2 (coverage): ~~Was the deletion about the _property_ or the _test style_?~~

**Answered:** deferred as a design decision — a lighter real-connection test remains open (see f.2).

The deleted `TestCompressionETag_WebSocketUpgrade_Passthrough` verified a real correctness property (compression/ETag middleware must not corrupt a 101 Switching Protocols handshake or inject headers into an upgrade response) via a heavy real-TCP test (186 lines, `net.Dial`, manual HTTP/1.1 byte parsing). If the annoyance was the **test's weight/maintenance** (real TCP, fragile, the body-before-hijack variant that deadlocked), I recommend restoring a **lighter** version (`httptest.NewServer` + header assertions only — task f.4). If the annoyance is that the **property itself isn't worth testing**, then leave it deleted and I'll make the CHANGELOG honest per f.1. **Which?**

### Q3 (convention): ~~Should `Removed` CHANGELOG entries always state the coverage delta?~~

**Answered:** adopted as convention — `Removed` entries state the coverage delta.

I propose a convention (e.3): every `Removed` entry that touches a test must explicitly state which behavior lost automated coverage, even if a partial replacement exists. This would have prevented the d.1 overstatement. **Adopt as a standing rule, or leave it to per-session judgment?**

---

## Session metrics

| Metric                                   | Value                                          |
| ---------------------------------------- | ---------------------------------------------- |
| Files deleted                            | 1 (`websocket_upgrade_test.go`, 211 lines)     |
| Files edited                             | 2 (`AGENTS.md`, `CHANGELOG.md`)                |
| Commits                                  | 1 (`485cc82`, auto-daemon)                     |
| Tests run                                | `go test -race ./...` — PASS                   |
| Lint                                     | `golangci-lint run` — 0 issues                 |
| Build                                    | `go build ./...` — clean                       |
| Claims made in docs without verification | **1** (the CHANGELOG coverage claim — see d.1) |
| Honest self-corrections in this report   | 1 (d.1)                                        |

---

_Generated 2026-08-07 06:12. Scope: this session only. The d.1 finding is the headline — a coverage claim entered the CHANGELOG without the cited test being read. Recoverable under `[Unreleased]`; no frozen history was touched._

---

## Resolution (2026-08-07 docs-health pass; upgraded to per-item markers 2026-08-29)

Every actionable item is resolved inline; unmarked items are still open by convention. The header banner was removed — its verdicts live on the items.

Open as of 2026-08-29: c.3/f.2/f.7 (restore a real-connection Hijack test — deferred design decision, plus the helper restore contingent on it), f.4 (lighter upgrade-response test — not written), f.6 (test-name/assertion lint convention), f.9 (mutation-test the Hijacker-preservation test). Sections d)/e) are narrative session facts and process lessons, intentionally unmarked.
