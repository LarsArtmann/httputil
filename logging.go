package httputil

import (
	"log/slog"
	"net/http"
	"time"
)

// Logging returns middleware that logs each request with method, path, status,
// and duration using the provided slog.Logger.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
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

			logger.Info(
				"request",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", status),
				slog.Duration("duration", duration),
				slog.String("client_ip", ClientIP(req)),
			)
		})
	}
}
