# Issue draft — go-error-family: classification guidance for conditional-request outcomes (304/412)

Status: verified 2026-09-10 against go-error-family `family.go` / `http.go` (v0.10.0 consumer: httputil). Not yet filed — the repo is the owner's; review, adjust, and file when ready. Verified per the verify-before-filing discipline (no existing guidance in source, README, TODO_LIST, ROADMAP, or FEATURES).

## Title

Document classification guidance for conditional-request outcomes (304 Not Modified, 412 Precondition Failed)

## Body

### Problem

`Family.HTTPStatus()` provides a canonical family → HTTP status mapping (Rejection → 400, Conflict → 409, Transient → 503, Corruption → 500, Infrastructure → 503, Orchestration → 500, `family.go` `familyData`), and `HTTPStatuser` lets an error override the status. What the classification table does not answer is how a **library author** should treat conditional-request outcomes under RFC 9110 §13:

1. **304 Not Modified** is a _success_ outcome of a successful conditional GET, not an error. Nothing in the docs says "do not route this through error classification," so the temptation is to model it as some family and get a nonsense 4xx/5xx out of `HTTPStatus(err)`.
2. **412 Precondition Failed** has no canonical pairing. The table's closest rows are Rejection (400) and Conflict (409), and neither is obviously right: the request _syntax_ is fine (so 400/Rejection reads wrong), and 409/Conflict is about request-state conflicts with the current state of the target resource — close, but the doc gives no ruling.
3. **304/412 in handler-returning-error designs.** With `HandlerFunc`/`HTTPHandler`, a handler that wants to end with 304/412 must either return `nil` (losing the reason) or an error whose family maps to the wrong status. The `HTTPStatuser` escape hatch exists but the docs don't present it as the answer for these cases.

### Evidence this is a real gap

- `grep -rn "conditional|304|412|Precondition|Not Modified"` over `*.go` + `docs/` (excluding node_modules): zero guidance hits in the library.
- The family table in `README.md` (lines 116-117) and `familyData` in `family.go` (lines 79-132) cover no 3xx outcome and no precondition outcome.
- Consumer evidence: the `httputil`/`go-etag` ETag middleware emits 304 on `If-None-Match` matches and tracks 412-producing `If-Match` preconditions on its ROADMAP; at every step the author has to make an undocumented family decision.

### Proposal (documentation-only, no API change)

Add a short "Conditional requests" subsection to the classification guidance (README family table area or `docs/`):

1. **304 Not Modified is not an error.** A successful conditional request should return `nil` from a `HandlerFunc` (or write the 304 directly); do not model it as a classified error — every family maps to 4xx/5xx.
2. **412 Precondition Failed: classify as `Conflict`, override the status with `HTTPStatuser`.** `Conflict` is the semantically correct family (the client's precondition contradicts the resource's current state), while the status override preserves the RFC-mandated 412 because `Conflict.HTTPStatus()` is 409.
3. **428 Precondition Required** (if servers choose to demand preconditions) fits the same `Conflict` + override pattern.

Example snippet to include:

```go
type preconditionFailedError struct{ *errorfamily.Error }

func (e *preconditionFailedError) HTTPStatus() int { return http.StatusPreconditionFailed }

err := &preconditionFailedError{errorfamily.NewConflict(
    "etag.precondition_failed",
    "If-Match precondition failed",
)}
// errorfamily.HTTPStatus(err) == 412; Classify(err).ErrorFamily() == Conflict
```

### Scope

- Who benefits: any consumer building ETag/Last-Modified/Range handlers on top of `HTTPHandler`/`HTTPStatus` — i.e., every REST API doing caching or concurrency control.
- Who is unaffected: anyone not modeling conditional requests; docs-only change, no behavior change.
- Alternatives considered and rejected: adding a seventh family (unnecessary — Conflict carries the semantics; families are behavioral, statuses are wire-level, and `HTTPStatuser` already decouples them); mapping Conflict → 412 globally (breaks the 409 semantics for existing consumers).

## Filing checklist (before posting)

- [ ] Re-verify against the go-error-family HEAD at filing time (guidance may have landed).
- [ ] Confirm no open/closed issue proposes the same (`gh issue list --repo larsartmann/go-error-family --state all`).
- [ ] File from an account with write access decisions made by the owner (own repo).
