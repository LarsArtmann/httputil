package httputil

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Logging returns middleware that logs each request with method, path, status,
// and duration using the provided slog.Logger.
func Logging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			start := time.Now()
			rec := NewResponseRecorder(resp)

			next.ServeHTTP(rec, req)

			duration := time.Since(start)

			status := rec.Status()
			if status == 0 {
				status = http.StatusOK
			}

			attrs := []slog.Attr{
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", status),
				slog.Duration("duration", duration),
				slog.String("client_ip", ClientIP(req)),
			}

			if requestID := RequestIDFromContext(req.Context()); requestID != "" {
				attrs = append(attrs, slog.String("request_id", requestID))
			}

			// WithoutCancel keeps context values (trace/span IDs) while detaching
			// from request cancellation: Timeout() and client disconnects cancel
			// the request context, and exactly those aborted-request logs must
			// never be dropped by cancellation-aware slog handlers.
			logger.LogAttrs(context.WithoutCancel(req.Context()), slog.LevelInfo, "request", attrs...)
		})
	}
}
