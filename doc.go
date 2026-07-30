// Package httputil provides composable HTTP middleware, utility primitives, and
// server lifecycle helpers for Go.
//
// The library offers CORS configuration, client IP extraction, response recording,
// middleware chaining, security headers, request ID propagation, panic recovery,
// request timeout enforcement, structured request logging, response compression,
// ETag generation with conditional request handling, an HTTP server wrapper with
// graceful shutdown, standard health check handlers (/health, /health/live,
// /health/ready), W3C Server-Timing instrumentation, CSRF protection (double-submit
// cookie via justinas/nosurf, with HTMX-aware helpers), and keyed token-bucket
// rate limiting with min-heap eviction and a MaxKeys cap.
//
// All middleware follows the standard func(http.Handler) http.Handler signature,
// making it compatible with any Go HTTP framework. Use Chain() to compose multiple
// middleware — first argument is outermost.
//
// All config types implement Validate() error for startup configuration checks.
//
// Errors from ResponseRecorder operations are classified using go-error-family
// with behavioral families (Transient, Infrastructure) for retry decisions and
// structured observability.
package httputil
