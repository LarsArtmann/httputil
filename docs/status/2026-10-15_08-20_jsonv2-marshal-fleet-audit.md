# json/v2 Marshal Fleet Audit — Every Call Site Verdicted

_Created: 2026-10-15 08:20 CEST · Trigger: the `FuzzCSRFTokenHTMLFormatters` smoke crasher (`b2bedba`) · Scope: root module, `httpspec`, `server_timing`, `scripts/` · Method: `grep -rn "json\.Marshal|MarshalWrite|MarshalEncode" --include="*.go" .` over the whole repo, per-site read of the surrounding function, verdict on whether the marshaled strings are externally influenced and whether json/v2's reject-invalid-UTF-8 semantics (Go 1.27) can surface._

## The semantic diff being audited

Go 1.27's `encoding/json` (json/v2 standard) **rejects** strings containing invalid UTF-8 in `Marshal`/`MarshalWrite`; json/v1 coerced them to U+FFFD. Behavior depends on the consuming module's `go` directive — a call site that "worked" under v1 can start returning an error the moment the module declares `go 1.27`. `CSRFTokenHXHeaders` was the live example: for a context token with invalid UTF-8 it now fails closed to `""` (pinned by `TestCSRFTokenHXHeaders_InvalidUTF8TokenRendersEmpty` and the committed corpus seed).

## Verdicts

| # | Site                                                                                                 | Strings marshaled                                                                                                               | Externally influenced?                                                                    | Verdict                                                                                                                                                                                                                                                                          |
| - | ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | `health.go:83` `writeHealthBody` — `json.MarshalWrite(resp, HealthResponse{Status: status})`         | `status` is one of the closed constants `HealthStatusUp`/`HealthStatusDown` ("up"/"down")                                       | No — handlers receive typed constants only; the wire-facing surface cannot inject strings | **SAFE** — no externally influenced strings; the `FuzzHealthResponse_Encoding` oracle already models json/v2's reject-on-invalid-UTF-8 both ways (v1.5.0-era fix)                                                                                                                |
| 2 | `csrf.go:956` `CSRFTokenHXHeaders` — `json.Marshal(map[string]string{DefaultCSRFHeaderName: token})` | `token` from `CSRFTokenFromRequest` — the middleware's hex token, OR any context value a consumer sets via the context override | Yes (the consumer-controlled token class)                                                 | **SAFE (fail-closed contract)** — invalid-UTF-8 token ⇒ `Marshal` errors ⇒ empty attribute returned; the safer direction (coercion would render MORE attacker-controlled attribute content); pinned by regression test + corpus seed; documented in CHANGELOG [Unreleased] Fixed |

Negative results (grep, whole repo incl. `httpspec/`, `server_timing/`, `scripts/`): no other `json.Marshal`/`MarshalIndent`/`MarshalWrite`/`MarshalEncode` call sites in production code. `server_timing` marshals nothing (stdlib-only header building). `scripts/*.go` contain no `json.Marshal`.

## Fleet rule (repeat this audit per repo after any Go 1.27 bump)

1. `grep -rn 'json\.Marshal\|MarshalWrite\|MarshalEncode' --include='*.go' .` (production first, tests separately).
2. For each site: can the marshaled string carry bytes from outside the binary (user input, header values, context overrides, generated IDs)?
3. If yes: does the json/v2 error path fail closed (render/write nothing) or fail open (partial write, silent drop)? Pin the intended direction with an execution-probe test.
4. If the site must keep rendering something for every token, sanitize BEFORE marshal — never re-enable v1 coercion semantics.

Verdict classes: SAFE (closed constants), SAFE-fail-closed (error path renders nothing, pinned), ACTION (error path silently drops a value consumers expect to see — needs a sanitize step or a documented contract).

## Residuals

None in this repo. The audit method is the deliverable for the rest of the fleet (AGENTS Go-1.27 notes bullet points here).
