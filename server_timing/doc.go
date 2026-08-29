// Package servertiming implements W3C Server-Timing header instrumentation
// for HTTP handlers.
//
// The collector is injected through the request context. Use
// [ServerTimingMiddleware] to install a [ServerTiming] per request, then
// record metrics from anywhere downstream of the middleware:
//
//	mw := servertiming.ServerTimingMiddleware()
//	h := mw(myHandler)
//
// Inside a handler:
//
//	t := servertiming.MeasureServerTiming(r.Context(), "db")
//	defer t.Done()
//
// The middleware serializes the collected metrics into the
// [HeaderServerTiming] response header on completion. Values are sanitized
// against CRLF injection ([HeaderValue] rejects control characters).
//
// The middleware composes with httputil.Middleware: both have the
// func(http.Handler) http.Handler shape. WrapServerTiming provides manual
// wrapping without the middleware.
//
// This module is stdlib-only: it has zero external dependencies, keeping it
// embeddable in contexts that cannot take on the root httputil module's
// dependency set.
package servertiming
