package httputil

import (
	etag "github.com/larsartmann/go-etag"
)

// ETag returns middleware that generates ETag headers based on response body
// content and handles If-None-Match conditional requests with 304 Not Modified.
//
// This is a thin adapter over [github.com/larsartmann/go-etag], which remains
// an independent, self-contained module. Import go-etag directly for domain
// types ([etag.ETag], [etag.ParseETag], [etag.MatchesIfNoneMatch], etc.) and
// configuration ([etag.ETagConfig], [etag.DefaultETagConfig]); httputil does
// not re-export them.
//
// See [etag.New] for full behavioral documentation.
//
// # Error classification
//
// [RegisterErrorClassifications] registers a strict superset of go-etag's error
// templates and stdlib classifications, so consumers need only call the
// httputil registration once. Do not also call etag.RegisterErrorClassifications.
func ETag(cfg etag.ETagConfig) Middleware {
	return Middleware(etag.New(cfg))
}
