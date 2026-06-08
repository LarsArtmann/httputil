// Package httputil provides 10 composable HTTP middleware and utility primitives for Go.
//
// The library offers CORS configuration, client IP extraction, response recording,
// middleware chaining, security headers, request ID propagation, panic recovery,
// request timeout enforcement, structured request logging, response compression,
// and ETag generation with conditional request handling.
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
