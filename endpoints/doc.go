// Package endpoints registers HTTP routes on a [http.ServeMux] and serves a
// self-describing listing of them at GET /endpoints.
//
// *http.ServeMux cannot enumerate the patterns registered on it, so the
// registry IS the source of truth: routes registered through
// [Registry.Handle] / [Registry.HandleFunc] are mounted on the mux AND
// recorded in the listing in one declaration. [Register] mounts the listing
// handler itself, which then lists every registered route including its own.
//
//	routes := endpoints.NewRegistry()
//	mux := http.NewServeMux()
//
//	err := routes.HandleFunc(mux, "GET /users/{id}", getUser, "Fetch one user")
//	err = endpoints.Register(mux, routes) // mounts "GET /endpoints"
//
// The listing negotiates its representation from the Accept request header:
// browsers (text/html) get a plain HTML table, API clients (application/json,
// */*, or no Accept header at all) get JSON. The two representations share one
// URL, so the response carries Vary: Accept.
//
// This module is stdlib-only: it has zero external dependencies, keeping it
// embeddable in contexts that cannot take on the root httputil module's
// dependency set. It composes with httputil's middleware (Compression,
// SecurityHeaders, ...) like any handler, but never requires it.
package endpoints
