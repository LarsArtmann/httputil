package httputil

import (
	"net/http"
)

// MaxBodySize returns middleware that limits request body size to maxBytes.
// When the limit is exceeded, the underlying read returns an error and the
// connection is closed to prevent the client from sending more data.
//
// Handlers should check for errors from r.Body.Read and respond with
// http.StatusRequestEntityTooLarge (413) as appropriate.
func MaxBodySize(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}

			next.ServeHTTP(w, r)
		})
	}
}
