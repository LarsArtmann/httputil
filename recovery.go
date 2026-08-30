package httputil

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery returns middleware that catches panics in downstream handlers,
// logs the panic value and stack trace, and returns 500 Internal Server Error.
// Panics with the net/http sentinel [http.ErrAbortHandler] are re-panicked
// unchanged so the server's own silent connection-abort handling applies.
func Recovery(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if isErrAbortHandler(rec) {
						panic(rec)
					}

					logger.Error(
						"panic recovered",
						slog.Any("error", rec),
						slog.String("method", req.Method),
						slog.String("path", req.URL.Path),
						slog.String("stack", string(debug.Stack())),
					)

					resp.Header().Set(headerContentType, "text/plain; charset=utf-8")
					resp.WriteHeader(http.StatusInternalServerError)

					writeCommittedBody(resp, []byte("Internal Server Error"))
				}
			}()

			next.ServeHTTP(resp, req)
		})
	}
}

// isErrAbortHandler reports whether v is (or wraps) the net/http
// ErrAbortHandler sentinel. The type assertion is guarded so a panic value of
// non-comparable type cannot crash the comparison itself.
func isErrAbortHandler(v any) bool {
	recErr, ok := v.(error)

	return ok && errors.Is(recErr, http.ErrAbortHandler)
}
