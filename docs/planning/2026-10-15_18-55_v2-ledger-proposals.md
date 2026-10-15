# v2.0 Ledger Notes — the Four Post-v1.1 Proposals

_Created: 2026-10-15 18:55 CEST (pareto T25 / TODO Post-v1.1 rows) · Sources: the 2026-09-11 full-code review findings 1/7/8/9 (harvested into TODO_LIST Post-v1.1) · Status: proposals only — every item is frozen-API or semantics material; nothing here executes without the v2.0 ruling_

## 1. `ValidateCSRF` result type (review finding 1)

**Problem.** A production API returns `*httptest.ResponseRecorder` — a test-support type with hidden behaviors (buffered body, code field) that leaks the test doubles' semantics into consumer code paths.

**Proposal.** Introduce an exported result type (e.g. `type CSRFValidation struct{ Status int; Body []byte; ... }`) or narrow the signature to `(status int, body []byte)`; keep `ValidateCSRF` as a deprecated wrapper through v2.x.

**Tradeoff.** Signature change = frozen-API break; every consumer of `ValidateCSRF` migrates. Hidden benefit: the current return also makes the function unattractive for exactly the production uses it exists for — the type IS the misuse vector.

**Sizing.** Small (one type + one wrapper), but consumer migration is fleet-wide (cqrs-htmx admin/usermgmt are the known callers).

## 2. `MiddlewareFunc` vs `Middleware` alias split (review finding 7)

**Problem.** Two names for `func(http.Handler) http.Handler`: the defined `MiddlewareFunc` (with the `Then` method) and the type alias `Middleware`. Inline literals typed as `Middleware` silently lack `.Then(nil)`'s 500-stub contract — the two names behave differently at call sites that look identical.

**Proposal.** Make `MiddlewareFunc` canonical in v2.0; `Middleware` stays as a deprecated alias (type aliases deprecate poorly — it would remain forever for source compat) or becomes a plain `type Middleware = MiddlewareFunc` kept undocumented.

**Tradeoff.** The alias is the accepted cross-module duplication pair (`server_timing` mirrors it — CHANGELOG-pinned via the chain_test assertion); renaming ripples the docs, the `MiddlewareStack` API, and every consumer's signatures. Gain: one honest name with one behavior.

**Sizing.** Medium; mechanical but broad.

## 3. Typed `MiddlewareStack` names (review finding 8)

**Problem.** `MiddlewareStack.Add` takes untyped `string` names while the package ships a `Middleware*` constant set — nothing binds the two; a typo'd name defeats duplicate detection.

**Proposal.** Additive path (no break): a `StackName` string type + `AddNamed(StackName, MiddlewareFunc)` alongside the existing `Add`; the constants gain a typed form. `Build`-without-validate stays as documented (relied upon).

**Tradeoff.** Two Add methods forever; the typed path is opt-in so legacy call sites are untouched. Alternative (typechange `Add`) is the v3.0 conversation.

**Sizing.** Small and genuinely additive — could ship in a minor release if the owner wants it before v2.0.

## 4. `Chain`/`Compose` nil-entry semantics (review finding 9)

**Problem.** A `nil` in the variadic list defers the panic to serve time (nil deref mid-request), while `MiddlewareFunc.Then(nil)` wires a diagnosable 500-stub. Same input shape, two opposite failure philosophies.

**Proposal.** Symmetric treatment, two directions: (a) skip-nil in both (silent tolerance — hides wiring bugs), or (b) construction-time rejection in both (panic at Chain/Compose — violates the never-panic API surface rule) or (c) 500-stub in both (consistent with Then, per the 2026-09-10 owner directive that wiring errors return/neutralize rather than panic).

**Tradeoff.** (c) is the ruling-consistent option but changes `Chain`/`Compose` observable behavior for nil inputs from panic-to-stub; (a) is additive-safe but hides bugs. The owner directive context (never-panic API surface, `MiddlewareFunc.Then(nil)` stub) points at (c).

**Sizing.** Small code, philosophy-heavy — this is a decision, not a task.

## Ruling shape requested

One decision per item: adopt as-written / adopt-with-changes / decline. Items 3 (additive) and 1 (wrapper) can land pre-v2.0 if ruled; items 2 and 4 are v2.0-proper.
