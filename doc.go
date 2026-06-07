// Package httputil provides composable HTTP middleware and utility primitives for Go.
//
// The library offers CORS configuration, client IP extraction, response recording,
// middleware chaining, security headers, request ID propagation, panic recovery,
// request timeout enforcement, structured request logging, response compression,
// and ETag generation with conditional request handling.
//
// All middleware follows the standard func(http.Handler) http.Handler signature,
// making it compatible with any Go HTTP framework.
//
// Errors from ResponseRecorder operations are classified using go-error-family
// with behavioral families (Transient, Infrastructure) for retry decisions and
// structured observability.
package httputil
