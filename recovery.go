package httputil

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery returns middleware that catches panics in downstream handlers,
// logs the panic value and stack trace, and returns 500 Internal Server Error.
func Recovery(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error(
						"panic recovered",
						slog.Any("error", rec),
						slog.String("method", req.Method),
						slog.String("path", req.URL.Path),
						slog.String("stack", string(debug.Stack())),
					)

					resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
					resp.WriteHeader(http.StatusInternalServerError)

					_, _ = resp.Write([]byte("Internal Server Error"))
				}
			}()

			next.ServeHTTP(resp, req)
		})
	}
}
