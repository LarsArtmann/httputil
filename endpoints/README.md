# endpoints

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

Self-describing HTTP route registration for Go's `http.ServeMux`: register your routes through a `Registry` and get a `GET /endpoints` listing of them all — JSON for API clients, an HTML table for humans, picked by content negotiation on the `Accept` header.

`*http.ServeMux` cannot enumerate the patterns registered on it, so a listing needs its own source of truth. The registry is that source: every route registered through it is mounted on the mux AND recorded in one declaration, so the listing can never drift from what is actually served.

Zero dependencies. This module is stdlib-only, so you can add it to a service without pulling in the full [`httputil`](https://github.com/larsartmann/httputil) dependency set.

## Install

```bash
go get github.com/larsartmann/httputil/endpoints
```

## Quick Start

```go
package main

import (
	"net/http"

	"github.com/larsartmann/httputil/endpoints"
)

func main() {
	routes := endpoints.NewRegistry()
	mux := http.NewServeMux()

	// One declaration mounts the route AND records it for the listing.
	err := routes.HandleFunc(mux, "GET /users/{id}", getUser, "Fetch one user")
	if err != nil {
		// Wiring errors are values, never panics: nil mux, empty pattern,
		// or a pattern ServeMux rejects (malformed or duplicate).
		panic(err)
	}

	// Mounts "GET /endpoints" — the listing includes itself.
	if err := endpoints.Register(mux, routes); err != nil {
		panic(err)
	}

	http.ListenAndServe(":8080", mux)
}
```

`curl :8080/endpoints` (no `Accept` header) serves JSON:

```json
{
  "endpoints": [
    {
      "method": "GET",
      "pattern": "GET /endpoints",
      "summary": "Endpoint listing (this page)"
    },
    {
      "method": "GET",
      "pattern": "GET /users/{id}",
      "summary": "Fetch one user"
    }
  ]
}
```

The same URL in a browser (`Accept: text/html,...`) serves a plain HTML table.

## Content Negotiation

Proactive negotiation (RFC 9110) between two representations:

- `text/html` (and `text/*`) → HTML
- `application/json` (and `application/*`, `*/*`, or no `Accept` header) → JSON

Highest q-value wins; ties break toward JSON, the machine default — so browsers get the table and `curl` gets JSON from one URL. The response carries `Vary: Accept` (caches must not cross the two representations) and `Cache-Control: no-store` (a stale route listing after a redeploy is worse than a refetch). Unparseable Accept entries are skipped rather than rejected — the listing never answers 406, mirroring httputil's `Language` middleware posture.

## Concurrency

A `Registry` is safe for concurrent use. Reads (`Routes`, the listing handler) run off immutable atomic snapshots, so you can add routes while a listing is serving; every request sees a consistent pre- or post-add state.

## Security Notes

- **The listing documents your attack surface.** Mount it deliberately; on public services, consider registering it only behind your admin/auth middleware.
- The HTML table carries no inline styles or scripts, so it renders under a strict CSP. All route fields are HTML-escaped (fuzz-tested).
- Patterns and summaries are validated as UTF-8 at registration; encoding/json v2 refuses invalid UTF-8, and rejecting at the seam keeps the listing renderable by construction.

## Composability

The listing is a plain `http.Handler`; wrap it in any middleware chain (compression, auth, logging). With [`httputil`](https://github.com/larsartmann/httputil), it composes like any other handler — but it never requires the root module.

## License

[MIT](LICENSE)
