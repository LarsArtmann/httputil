package httputil

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// RecoveryConfig configures [RecoveryWithConfig]. Every field has a defined
// zero-value behavior, so there is no invalid configuration to validate.
type RecoveryConfig struct {
	// Logger receives the panic log line. A nil Logger falls back to
	// [slog.Default].
	Logger *slog.Logger
	// OnPanic, when non-nil, runs after the panic is logged and before the
	// 500 response is written. It is the bridge to observability SDKs: record
	// the panic on the active span (for example with otel.RecordError)
	// without adding a tracing dependency to this package.
	//
	// The hook runs synchronously on the request goroutine. A panic from the
	// hook is logged and swallowed so it cannot replace the original panic
	// handling; the 500 response is still written.
	OnPanic func(r *http.Request, recovered any)
}

// Recovery returns middleware that catches panics in downstream handlers,
// logs the panic value and stack trace, and returns 500 Internal Server Error.
// Panics with the net/http sentinel [http.ErrAbortHandler] are re-panicked
// unchanged so the server's own silent connection-abort handling applies.
//
// It is equivalent to [RecoveryWithConfig] with a nil OnPanic hook.
func Recovery(logger *slog.Logger) Middleware {
	return RecoveryWithConfig(RecoveryConfig{Logger: logger, OnPanic: nil})
}

// RecoveryWithConfig returns panic-recovery middleware configured by cfg.
// Behavior matches [Recovery]; cfg.OnPanic additionally exposes the recovered
// value to observability integrations.
func RecoveryWithConfig(cfg RecoveryConfig) Middleware {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

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

					callOnPanic(cfg.OnPanic, logger, req, rec)

					resp.Header().Set(headerContentType, "text/plain; charset=utf-8")
					resp.WriteHeader(http.StatusInternalServerError)

					writeCommittedBody(resp, []byte("Internal Server Error"))
				}
			}()

			next.ServeHTTP(resp, req)
		})
	}
}

// callOnPanic invokes hook, isolating a panicking hook from the recovery path:
// the original panic is already handled, so a hook failure must not escape and
// abort the response.
func callOnPanic(hook func(*http.Request, any), logger *slog.Logger, req *http.Request, recovered any) {
	if hook == nil {
		return
	}

	defer func() {
		if hookPanic := recover(); hookPanic != nil {
			logger.Error(
				"panic recovery hook panicked",
				slog.Any("hook_panic", hookPanic),
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
			)
		}
	}()

	hook(req, recovered)
}

// isErrAbortHandler reports whether v is (or wraps) the net/http
// ErrAbortHandler sentinel. The type assertion is guarded so a panic value of
// non-comparable type cannot crash the comparison itself.
func isErrAbortHandler(v any) bool {
	recErr, ok := v.(error)

	return ok && errors.Is(recErr, http.ErrAbortHandler)
}
