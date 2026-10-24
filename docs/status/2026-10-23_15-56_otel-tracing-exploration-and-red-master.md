# Status Report — 2026-10-23 15:56 CEST — OTEL/Tracing Exploration Session + Red-Master Discovery

**Session scope:** One exploration session ("How can we help projects using us better support OTEL, especially tracing?") followed by this status report. **I changed zero product code in this session.** Everything below is based strictly on what this session ran and noticed — no new research areas were opened.

**Tree state at report time:** `## master...origin/master` — clean, synced, **and RED** (see section d).

---

## Headline findings

1. **Master is red and has been for ~8 days.** `example_test.go:859:30: ErrCSRFInvalid.WithContext undefined (type error has no field or method WithContext)` — confirmed by `go vet` this session. The breaking declaration rode a heuristic auto-commit daemon commit (`0ae65b9`, 2026-10-15 19:31) and is **pushed to origin**.
2. **This is the third recurrence of a rogue "error-interface sentinel sweep"** — twice deliberately reverted on 2026-10-15, then re-landed by a daemon commit 14 minutes after the second revert. It contradicts a documented AGENTS.md directive.
3. **The OTEL exploration produced a recommendation but zero durable artifacts** — the analysis lives only in the chat transcript (entombed), including a split-brain ruling candidate (go-appkit/otel vs httputil lanes) that is recorded nowhere.

---

## a) FULLY DONE (this session)

| # | Item | Evidence |
|---|------|----------|
| a1 | **OTEL current-state survey** — established what already exists: `Middleware` is a type alias composing with otelhttp directly; every request-forking middleware propagates `r.Pattern` back so outer otelhttp gets correct span names (pinned in `pattern_propagation_test.go:11`); CHANGELOG [1.4.x] documents both | grep sweep + CHANGELOG read |
| a2 | **Fleet OTEL lane identified** — `go-appkit/otel` is the existing OTEL integration: no-op-until-`Setup` `Middleware()` (otelhttp bridge), span-aware `Recovery` (records `exception.*` events + error status), instrumented `Transport`, baggage `correlation_id`, OTLP/SigNoz setup, dashboard JSON. Plain net/http, already imports `github.com/larsartmann/httputil` | `~/projects/go-appkit/otel/` inspected (doc.go, middleware.go, baggage.go) |
| a3 | **Policy grounding** — how-to-golang skill: reusable libraries get "nil-default hooks, never a telemetry dependency"; depguard allowlist bans OTEL in this repo; ROADMAP.md:31 parks OTEL integration with "no consumer demand" | skill + AGENTS.md + ROADMAP |
| a4 | **Options analysis delivered with recommendation** — A: zero-dep tracing-support layer in httputil (integration guide + `RecoveryConfig.OnPanic` nil-default hook + `httpspec.TracingSpecs()` + RequestID↔trace correlation evaluation) — RECOMMENDED. B: `httputil/otel` sub-module — dismissed (split-brain with go-appkit/otel, violates no-telemetry-dep, go.work membership question ROADMAP.md:54 still open). C: docs-only — dismissed (Recovery exception gap unfixable by docs) | previous assistant turn |
| a5 | **Demand-signal identification** — the owner's OTEL question is itself the consumer-demand signal the parked ROADMAP item said it was waiting for | ROADMAP.md:31 |
| a5b | **Red-master discovery + full forensics** — compile error confirmed via `go vet`, offending declaration located (`csrf.go:60`), attribution traced through git (section d), blast-radius checked (`rg "^var [Ee]rr\w+ error ="` → 4 interface-typed sentinels, 1 breaking) | this session, after the user's report request |

## b) PARTIALLY DONE

| # | Item | Done | Missing |
|---|------|------|---------|
| b1 | **OTEL support initiative overall** | Analysis, options, recommendation | Design note NOT written; zero code; owner ruling (A/B/C) pending; ROADMAP still carries the now-false "no consumer demand" wording |
| b2 | **Recovery→span exception bridge** | Gap identified precisely (`Recovery(logger)` at `recovery.go:14` has no config type, swallows panics; outer otelhttp sees 500 without exception events; go-appkit solved it lane-locally) | `RecoveryConfig` design + `OnPanic` hook + tests |
| b3 | **Split-brain ruling (appkit/otel vs httputil)** | Analysis in chat: appkit = SDK lane, httputil = zero-dep hooks/contracts lane (mirrors the i18n precedent where one lane wins) | Ruling recorded nowhere durable — will be re-derived (or contradicted) by the next session |
| b4 | **Sentinel-type integrity** | Breaking site found + 3 more interface-typed sentinels identified (`csrf.go:66` ErrCSRFConfig, `decompression.go:41` errDecompressionSizeExceeded, `metrics.go:11` errNilMetricsRecorder — policy violations, not compile-breaking today) | Fix not applied; the sweep SOURCE not found; no type-level guard exists to pin sentinels against a fourth recurrence |

## c) NOT STARTED

1. The fix for the red master (revert `var ErrCSRFInvalid error` → `*errorfamily.Error` on `csrf.go:60`, plus the 3 sibling sentinels per policy).
2. A type-level guard (test or lint) asserting error-family sentinels keep their concrete `*errorfamily.Error` type — the thing that would have blocked recurrences 2 and 3.
3. `docs/integrations/otel.md` (the canonical OTEL guide: two lanes, ordering rules — span outside Recovery, inside it; Server-Timing placement; Compression writer-wrapping gotchas).
4. `httpspec.TracingSpecs()` — opt-in contract specs (traceparent round-trip, `Flusher`/`Hijacker` preservation through tracing wrappers, status-code→span-status mapping).
5. RequestID ↔ trace correlation (traceparent-seeded request ID, or documenting the go-appkit baggage `correlation_id` convention for the fleet lane).
6. ROADMAP annotation of the OTEL parked item with this session's demand signal.
7. TODO_LIST/ROADMAP harvest of this report's section f (docs-health HARVEST — the loop-not-closed warning applies to my own session).
8. Identifying WHICH session/tool keeps running the error-interface sweep (unidentified actor, three occurrences).
9. Everything in section f beyond items 1–14 (carried/parked items — listed for completeness, not commitments).

## d) TOTALLY FUCKED UP

### d1. RED MASTER — committed AND pushed, red for ~8 days

**Error:** `vet: ./example_test.go:859:30: ErrCSRFInvalid.WithContext undefined (type error has no field or method WithContext)`

**Root cause:** `csrf.go:60` declares `var ErrCSRFInvalid error = errorfamily.NewRejection(...)` — explicit `error` interface type. The concrete `*errorfamily.Error` type is load-bearing: `WithContext`/`WithCause` live only on it. AGENTS.md states this verbatim in the erraudit block: *"The load-bearing `*errorfamily.Error` sentinels — do NOT migrate; declaring `var errX error` breaks WithContext/WithCause call sites."*

**Forensic timeline (2026-10-15, all same day):**

| Time | Commit | What | Nature |
|------|--------|------|--------|
| 18:47 | `28681f2` | "fix: revert error-interface annotations on error-family sentinels" | reasoned — revert #1 |
| 19:08 | `e6431ce` | daemon, 9 files | heuristic |
| 19:17 | `a6e0a12` | "fix: re-revert the error-interface sentinel sweep (second occurrence)" | reasoned — revert #2 |
| 19:31 | `0ae65b9` | daemon, 9 files | heuristic — **re-introduced `var ErrCSRFInvalid error`** (confirmed via `git log -S`) |

Since 19:31 on 10-15: **master red for ~8 days**, and `master...origin/master` shows the red state is pushed. This is the exact documented red-master class ("the daemon commits non-compiling intermediates") that the still-pending build-gate proposal (`docs/planning/2026-10-15_02-35_daemon-presweep-build-gate-proposal.md`) targets. The proposal has now been validated by a live incident — it is not hypothetical anymore.

**Blast radius:** `go vet` stops at the first error. `rg "^var [Ee]rr\w+ error ="` finds 4 interface-typed sentinels: `csrf.go:60` (breaks compilation), `csrf.go:66` ErrCSRFConfig, `decompression.go:41` errDecompressionSizeExceeded, `metrics.go:11` errNilMetricsRecorder (the latter three are policy violations that compile today but violate the typed-sentinel pattern — `var errFoo = code.Rejection(...)` without explicit type, per AGENTS Hard Constraints).

### d2. The recurring sweep itself — unidentified actor, three occurrences

Something in the fleet (most likely a parallel session cargo-culting an erraudit/go-error-modernization drive-to-zero) has now flipped error-family sentinels to `error` interface type **three times in one day**, twice reverted deliberately. The reverts are losing: the daemon re-swept the change back within 14 minutes. Until the SOURCE is found or a type-level guard exists, recurrence #4 is a when, not an if. Per the parallel-writer protocol: reasoned commits (`28681f2`, `a6e0a12`) = deliberate reverts by a writer who understood the problem; the heuristic commits that re-land it = the sweep riding daemon sweeps. **Do not "fix forward" by hand-editing blindly** — the fix is the documented pattern (drop the explicit `error` annotation), but the source hunt matters more, or we are bailing a boat forever.

### d3. My own session's failures (brutal self-review — the user asked directly)

**What did I forget?**
- **The loop.** I delivered an OTEL recommendation as chat text and stopped. No design note, no ROADMAP annotation, no TODO_LIST row, no recorded split-brain ruling. The status-report skill's own warning ("entombed in a timestamped file no later session reads") applies one level up: my recommendation is entombed in a transcript no later session reads. Next session rediscovers go-appkit/otel from scratch — or worse, proposes the `httputil/otel` sub-module I already dismissed.
- **Session-tail compile check.** I read AGENTS.md this session — including the exact sentence describing this failure mode ("declaring `var errX error` breaks WithContext/WithCause call sites") — and never ran a compile. The tree has been red for 8 days across N sessions; none of us caught it. Cheap gate, never applied.
- **Un-parking the trigger.** ROADMAP.md:31 says OTEL has "no consumer demand; revisit individually only if one appears." The owner's question WAS the appearing demand. I noticed it, said so out loud, and didn't write it down. The ROADMAP line is now false and I left it false.

**What could I have done better?**
- Produced artifacts in-session. One ROADMAP edit + one TODO_LIST row = minutes. The analysis was finished; only the writing-down was skipped.
- Measured cleanly: my `go vet` verification piped through `head` and I reported `vet-exit: 0` — which was **head's** exit code, not vet's. The diagnostic text made the failure dispositive anyway, but I repeated the exact pipe-trap AGENTS.md's session-tail discipline names ("never pipe through filters that eat errors or exit codes") while quoting that discipline back to myself. Sloppy. Corrected here: **the failure is real**; vet exits non-zero.
- Checked the tree state BEFORE the exploration answer, not after: had `go vet` run in my first tool batch, the red master would have been this session's headline instead of a by-product of the report request.

**What could I still improve?**
- Make the compile check a session-START gate (not just session-tail): `go vet ./...` takes seconds and would have surfaced d1 immediately.
- Turn every "dismissed alternative" into a recorded one-liner at dismissal time (B was dismissed with three reasons — none persisted).
- Treat "owner asked about X" as a doc-freshness event for X's ROADMAP/TODO_LIST entries, not just a conversation.

**Did I lie?** No material lies found. Two precision notes: (1) "depguard bans OTEL" is true of the documented allowlist — I did not re-run depguard to verify the config hasn't drifted this session; (2) the `vet-exit: 0` I displayed mid-session was misleading as an exit-code claim (see above) — the underlying diagnostic was accurate.

**Ghost systems / split brains found:** No new ghost systems. One ACTIVE split-brain candidate needing a durable ruling: OTEL middleware could live in go-appkit/otel (exists today) or httputil (proposed as option B) — undecided-in-writing. Plus one doc/code split: ROADMAP says "no consumer demand" for OTEL while the owner is asking for it.

**Scope-creep check:** The OTEL recommendation deliberately stays out of core deps and out of a new sub-module — no creep. The biggest creep risk is section f items 15–50: they are carried ROADMAP/parking-lot items, NOT new commitments; most are ROADMAP fuel per the status-report skill.

## e) WHAT WE SHOULD IMPROVE

1. **Pre-sweep build gate** — the pending daemon proposal (`2026-10-15_02-35`) should be promoted from proposal to ruling: d1 is its second live validation (2026-10-14 was the first). A tree that cannot compile must not be swept.
2. **Type-level sentinel guard** — a test asserting `reflect.TypeOf(errX) == reflect.TypeOf(&errorfamily.Error{})` for every sentinel (or a tiny linter rule) would have blocked recurrences 2 and 3 mechanically. One test file, permanent payoff.
3. **Session-start compile gate** — `go vet ./...` before any work; this session proved 8 days of red can hide behind "clean tree, in sync with origin."
4. **Close-the-loop discipline for exploration sessions** — an exploration that ends in a recommendation ends with either an artifact (design note) or a ROADMAP/TODO_LIST mutation. Chat-only = not done.
5. **Demand-signal routing** — when an owner question touches a parked ROADMAP item, the item gets annotated in the same session ("demand signal appeared 2026-10-23, session …").
6. **Daemon attribution ergonomics** — d1 took 4 git commands to attribute. Heuristic commits that land KNOWN-reverted changes (the -S evidence) could carry a stronger signal (e.g., the daemon could refuse to commit files whose change was reverted in the last N commits — that is essentially the build-gate proposal again).
7. **The `.md` override convention** — this report is `.md` by explicit user instruction (status-report skill is HTML-canonical); flagged per the skill's rule, not propagated as a default.

## f) Up to 50 things we should get done next

**Group 1 — from THIS session (impact-sorted; 1–4 are urgent):**

| # | Task | Impact |
|---|------|--------|
| 1 | Fix red master: drop the explicit `error` annotation on `ErrCSRFInvalid` (`csrf.go:60`) restoring `*errorfamily.Error` | BLOCKER — everything downstream of a compiling tree |
| 2 | Sweep the 3 sibling interface-typed sentinels (`csrf.go:66`, `decompression.go:41`, `metrics.go:11`) back to the documented pattern in the same fix | BLOCKER-adjacent — same sweep residue, recurrence fuel |
| 3 | Add the type-level sentinel guard test (locks all error-family sentinels to `*errorfamily.Error`) | prevents recurrence #4 mechanically |
| 4 | Verify green: `go vet ./...` + `go test -race ./...`, confirm `origin/master..master` empty, CI green on head (session-tail discipline) | BLOCKER-adjacent |
| 5 | Hunt the sweep source: which session/tool keeps running the error-interface migration (check erraudit invocation logs/parallel sessions; the go-error-modernization skill documents this exact cargo-cult trap) | stops the boat-bailing |
| 6 | Owner ruling: OTEL Option A (recommended) vs B vs C — the open question from the exploration turn | unblocks the whole OTEL lane |
| 7 | Annotate ROADMAP.md:31 OTEL item: demand signal appeared 2026-10-23; wording "no consumer demand" now false | doc honesty, minutes |
| 8 | Write the OTEL design note (`docs/planning/`, Language-middleware pattern: decision gates D1–D4, split-brain ruling recorded) | durable foundation for A |
| 9 | `docs/integrations/otel.md` — two-lane guide (bare otelhttp wiring + go-appkit/otel), ordering rules, gotchas | highest-reach deliverable of A |
| 10 | `RecoveryConfig` with nil-default `OnPanic(r *http.Request, recovered any)` hook + tests (exception-event bridge, zero OTEL dep) | closes the gap docs cannot |
| 11 | `httpspec.TracingSpecs()` — opt-in specs: traceparent round-trip, Writer-interface preservation, status mapping | unique httputil contribution |
| 12 | Evaluate RequestID ↔ trace correlation (traceparent-seeded request ID vs documented baggage convention) | decision before code |
| 13 | Record the lane ruling (go-appkit/otel = SDK lane; httputil = zero-dep hooks/contracts) in AGENTS.md + architecture-reference | prevents future split-brain relitigation |
| 14 | HARVEST this report's section f into TODO_LIST.md/ROADMAP.md (docs-health) | the loop-not-closed rule |

**Group 2 — carried, confirmed still-open during THIS session's ROADMAP read (ROADMAP fuel, not commitments):**

| # | Item | # | Item |
|---|------|---|------|
| 15 | Daemon pre-sweep build-gate proposal → owner ruling (d1 validates it again) | 33 | Trash-process mystery 2026-09-22/23 (unidentified MOVED_FROM) |
| 16 | RequestID correlation-ID dual-header echo (dnsblockd need) | 34 | Preferred commit flow in daemon-active windows (owner decision) |
| 17 | go.work membership policy for new sub-modules (gated option B's dismissal) | 35 | go-etag `go` directive reconciliation |
| 18 | HSTS middleware (validated-config pattern) | 36 | CHANGELOG granularity for doc-only changes (convention) |
| 19 | HTTPS-redirect helper (trusted-proxy trust model) | 37 | art-dupl `-t 1` baseline policy ruling |
| 20 | `MaxHeaderBytes` on ServerConfig | 38 | Tracked root binaries (scripts/*.exe artifacts) keep-or-untrack |
| 21 | Idempotency-key middleware (native Store interface) | 39 | Vestigial vendor references (`markdownlintignore`, lychee flag) |
| 22 | Server shutdown-drain semantics (`Shutdown` vs `Start` race window) | 40 | CSP nonce extensions batch (StrictCSP, Report-Only, hashes) |
| 23 | Nightly-fuzz workflow scaling (concurrency/matrix/inputs) | 41 | CSRF hardening follow-ups (Referer attestation, Sec-Fetch normalization) |
| 24 | TLS-config clone-on-write for `Server` | 42 | Ecosystem extensions: brotli/zstd factories, Prometheus recorder, Redis store, HTMX helpers, compression httpspec spec |
| 25 | Consumer-audit program as standing tooling | 43 | StartupHandler warmup primitive (designed, unscheduled) |
| 26 | Consumer-audit operating model (3 sub-rulings) | 44 | Property-test middleware stack ordering rules |
| 27 | Autonomous nightly-crasher authority ruling | 45 | Execution-probe test: ServeTLS ALPN h2-append mutation |
| 28 | `AbsentEncodingFirstConfigured` lifecycle ruling | 46 | Fuzz-corpus seed inventory policy |
| 29 | Branching-flow dispositions (3 calls incl. upstream filings) | 47 | CI check that committed `.d2` diagrams render |
| 30 | `MiddlewareFunc.Append(...)` decide-or-decline | 48 | Runnable `example/` directory (parked) |
| 31 | Quarterly consumer/importer scan re-feed | 49 | HTTP/2 integration tests (parked) |
| 32 | Coverage-badge refresh step in release runbook | 50 | README flake-app discovery table |

## g) Questions I cannot figure out myself

1. **What is the real OTEL demand source — a specific consumer project that hit a tracing wall, or a fleet-posture decision?** This sets scope and urgency: one consumer = design note + targeted fix (Option A lane); fleet posture = consider whether go-appkit/otel should be promoted as THE documented answer for all consumers and httputil only ships contracts (a stronger version of my C-dismissal).
2. **Is an additive nil-default `RecoveryConfig.OnPanic` hook acceptable API surface for the frozen v1 (minor release), or must panic→span-event bridging stay exclusively in the go-appkit/otel lane?** This is an owner ruling on the frozen surface; it decides whether item 10 exists in httputil at all.
3. **For bare-httputil consumers (not appkit services), is importing `go-appkit/otel` an acceptable dependency, or must the documented OTEL path be pure `otelhttp`?** This decides how `docs/integrations/otel.md` is framed — and whether an httputil-owned shim is ever justified despite the split-brain risk.

---

## Verification appendix (commands run this session)

- `date` → 2026-10-23 15:56 CEST
- `git status -sb` + `git log -5` → clean tree, synced with origin, daemon-heuristic commits at HEAD
- `go vet ./...` → **FAILS**: `example_test.go:859:30: ErrCSRFInvalid.WithContext undefined (type error has no field or method WithContext)` (NOTE: the `vet-exit: 0` shown mid-session was `head`'s exit code through the pipe — vet's failure is established by its diagnostic output; see d3)
- `rg ErrCSRFInvalid` → definition `csrf.go:60` (`var ErrCSRFInvalid error = errorfamily.NewRejection(`), example call site `example_test.go:859`
- `git log` on `csrf.go` / `example_test.go` + `git log -S 'var ErrCSRFInvalid error'` → sweep timeline (section d1 table)
- `rg "^var [Ee]rr\w+ error ="` → 4 interface-typed sentinels (csrf.go ×2, decompression.go ×1, metrics.go ×1)

**Commit status:** No manual commit made — the harness forbids commits without an explicit user request; the auto-commit daemon will pick this file up.

**THEN WAIT FOR INSTRUCTIONS.**

## Resolution (2026-10-23, v1.6.0 release)

Both halves of this report are closed:

- **Red master**: the 15 sentinel declarations were restored, and the class is now mechanically blocked (AST guards in both modules, each with a self-test proving the detector works). The sweep's actor was identified during the release hardening: it is an agent applying `BuildFlow/docs/GOTCHAS.md` #125's blanket rule to a repo whose sentinels are load-bearing. See the AGENTS erraudit block and the daemon gate proposal's 2026-10-23 update.
- **OTEL exploration**: the demand question was answered as far as this repo can answer it (the two-lane ruling), and the deliverable is released — `RecoveryConfig`/`RecoveryWithConfig`, `TracingSpecs` (4 specs), `RunTracing` (5 checks), `docs/integrations/otel.md`, and the executable interop fixture.

The report's closing instruction ("THEN WAIT FOR INSTRUCTIONS") was superseded by the owner's direction to execute the resulting plan; this appendix records the outcome rather than removing the instruction.
