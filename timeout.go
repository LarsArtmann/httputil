package httputil

import (
	"context"
	"net/http"
	"time"
)

// Timeout returns middleware that enforces a deadline on the request context.
// If the handler does not complete within the given duration, the context is
// cancelled. The handler must respect context cancellation for this to work.
func Timeout(duration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			ctx, cancel := context.WithTimeout(req.Context(), duration)
			defer cancel()

			next.ServeHTTP(resp, req.WithContext(ctx))
		})
	}
}
