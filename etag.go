package httputil

import (
	etag "github.com/larsartmann/go-etag/server"
)

// ETag returns middleware that generates ETag headers based on response body
// content and handles If-None-Match conditional requests with 304 Not Modified.
//
// Deprecated: Use [etag.New] directly. Now that httputil.Middleware is a type
// alias for func(http.Handler) http.Handler, go-etag's constructor composes
// frictionlessly with Chain and MiddlewareStack — this function is a pure
// passthrough with no added behavior.
//
// # Error classification
//
// [RegisterErrorClassifications] registers a strict superset of go-etag's error
// templates and stdlib classifications, so consumers need only call the
// httputil registration once. Do not also call etag.RegisterErrorClassifications.
func ETag(cfg etag.ETagConfig) Middleware {
	return etag.New(cfg)
}
