package httputil

import (
	"context"
	"net/http"
)

type clientIPKey struct{}

// WithClientIP stores the extracted client IP in the request context.
// Retrieve it later with ClientIPFromContext.
func WithClientIP(parent context.Context, ip string) context.Context {
	return context.WithValue(parent, clientIPKey{}, ip)
}

// ClientIPFromContext retrieves a previously stored client IP from the context.
// Returns empty string if no IP was stored.
func ClientIPFromContext(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)

	return ip
}

// ClientIPMiddleware extracts the client IP and stores it in the request
// context for downstream handlers.
func ClientIPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := ClientIP(r)
		ctx := WithClientIP(r.Context(), ip)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
