# Domain Language — httputil

A **Unified Language** for `httputil` — shared across Contributor, Consumer (developer using this library), Reviewer, and AI.
Inspired by Domain-Driven Design (DDD) Ubiquitous Language.

Every term below should mean the **same thing** to everyone who reads it.
If a word means something different to a contributor than to a consumer, define it here.

---

## Glossary

| Term       | Definition                                                               | Context                                                      |
| ---------- | ------------------------------------------------------------------------ | ------------------------------------------------------------ |
| httputil   | A Go library providing composable HTTP middleware and utility primitives | The project itself; module `github.com/larsartmann/httputil` |
| Consumer   | A developer who imports and uses `httputil` in their Go HTTP service     | Library user                                                 |
| Middleware | A function that wraps an `http.Handler` to intercept/modify request flow | Signature: `func(http.Handler) http.Handler`                 |
| Request    | An incoming `*http.Request` being processed by the HTTP server           | Go `net/http` standard library                               |
| Response   | An `http.ResponseWriter` used to construct the HTTP response             | Go `net/http` standard library                               |

---

## Bounded Contexts

The library has three bounded contexts, each with a distinct vocabulary and responsibility.

| Context          | Description                                                     | Key Type(s)                 |
| ---------------- | --------------------------------------------------------------- | --------------------------- |
| Client IP        | Extracting the true client IP from proxied requests             | `ClientIP`                  |
| CORS             | Configuring and enforcing Cross-Origin Resource Sharing policy  | `CORSConfig`, `CORS`        |
| Response Capture | Recording response state for inspection (status, headers, body) | `ResponseRecorder`, `Chain` |

---

## Entities

Objects with identity and lifecycle within the library.

| Term             | Definition                                                                          | Context          |
| ---------------- | ----------------------------------------------------------------------------------- | ---------------- |
| ResponseRecorder | A wrapping `http.ResponseWriter` that captures the status code and write state      | Response Capture |
| CORSConfig       | A configuration value object defining CORS policy (origins, methods, headers, etc.) | CORS             |

---

## Value Objects

Immutable objects defined by their attributes.

| Term              | Definition                                                                                  | Context          |
| ----------------- | ------------------------------------------------------------------------------------------- | ---------------- |
| Client IP         | The extracted IP address string identifying the originating client                          | Client IP        |
| Origin            | The value of the `Origin` request header; identifies the requesting site's scheme+host+port | CORS             |
| Allowed Origin    | An origin string permitted by the CORS policy; `*` means any origin is allowed              | CORS             |
| Preflight Request | An `OPTIONS` request sent by the browser before the actual cross-origin request             | CORS             |
| Actual Request    | The real request (GET, POST, etc.) following a successful preflight                         | CORS             |
| Status Code       | The HTTP status code captured by the ResponseRecorder (e.g., 200, 404)                      | Response Capture |
| Write State       | Whether `WriteHeader` has been called on the ResponseRecorder                               | Response Capture |

---

## Commands

Actions the library performs.

| Term                         | Definition                                                                                             | Context          |
| ---------------------------- | ------------------------------------------------------------------------------------------------------ | ---------------- |
| `ClientIP(r)`                | Extract the client IP from a request using header precedence: X-Forwarded-For → X-Real-IP → RemoteAddr | Client IP        |
| `CORS(cfg)`                  | Create middleware that sets CORS response headers and handles preflight requests                       | CORS             |
| `NewResponseRecorder(w)`     | Create a ResponseRecorder wrapping the given ResponseWriter, defaulting to unwritten state             | Response Capture |
| `Chain(handler, mw...)`      | Compose multiple middleware around a handler; first middleware in list is outermost                    | Response Capture |
| `DefaultCORSConfig()`        | Return a permissive CORS config suitable for local development (allows all origins)                    | CORS             |
| `resolveOrigin(origin, cfg)` | Determine the `Access-Control-Allow-Origin` value based on the config's allowed list                   | CORS             |

---

## Events

State transitions within the library.

| Term              | Definition                                                                        | Context          |
| ----------------- | --------------------------------------------------------------------------------- | ---------------- |
| Header Written    | `WriteHeader` called on ResponseRecorder; status is now captured and immutable    | Response Capture |
| Body Written      | `Write` called on ResponseRecorder; implicitly sets status 200 if not yet written | Response Capture |
| Preflight Handled | CORS middleware intercepts an OPTIONS request and returns 204 No Content          | CORS             |
| Request Passed    | CORS middleware delegates to the next handler (non-OPTIONS or passthrough mode)   | CORS             |

---

## Rules

Invariants and policies that the library enforces.

### Client IP Extraction Order

1. `X-Forwarded-For` header — use the **first** entry in the comma-separated list
2. `X-Real-IP` header — use the trimmed value directly
3. `RemoteAddr` — strip the port via `net.SplitHostPort`; fall back to raw value on error

> **Security Warning:** `ClientIP` trusts proxy headers without validation. Only use behind a reverse proxy that strips/overwrites these headers.

### CORS Policy

- If `AllowAllOrigins` is true → always respond with `Access-Control-Allow-Origin: *`
- If the request `Origin` matches an entry in `AllowedOrigins` → echo that origin back
- If no match → fall back to `*`
- Preflight `OPTIONS` requests receive `204 No Content` (unless `OptionsPassthrough` is set)
- `MaxAge` is sent as `Access-Control-Max-Age` in seconds (default: 86400 = 24 hours)

### ResponseRecorder Invariants

- `WriteHeader` only captures on **first call**; subsequent calls are ignored for capture but still delegated
- `Write` implicitly sets status 200 if no `WriteHeader` was called yet
- `Flush`, `Hijack`, and `Push` are optional — they delegate only if the underlying ResponseWriter supports them
- `Hijack` returns `http.ErrNotSupported` if the underlying writer is not an `http.Hijacker`
- `Push` wraps the underlying error with context including the push target

### Middleware Chaining

- `Chain` applies middleware in **reverse order** so the first middleware in the variadic list is the outermost handler
- Execution order: first middleware → ... → last middleware → final handler

---

## Conventions

Patterns consumers and contributors should follow.

| Convention               | Description                                                                    |
| ------------------------ | ------------------------------------------------------------------------------ |
| Middleware signature     | Always `func(http.Handler) http.Handler` — the Go standard library convention  |
| No external dependencies | The library uses only the Go standard library and `slices` (stdlib since 1.21) |
| Zero-allocation hot path | Internal helpers (`join`, `itoa`) avoid `fmt` or `strconv` allocations         |
| `httputil` import name   | Consumers import as `httputil`; no aliases needed                              |

---

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first
