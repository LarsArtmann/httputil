package httputil

import (
	"net/http"
	"time"
)

// codeMetricsNilRecorder classifies a missing metrics recorder as Rejection.
const codeMetricsNilRecorder = Code("metrics.nil_recorder")

var errNilMetricsRecorder = codeMetricsNilRecorder.Rejection("metrics config: Recorder must not be nil")

// MetricsRecorder receives one observation per request. Implementations must
// be safe for concurrent use.
type MetricsRecorder interface {
	// Record is called after each request completes with the HTTP method,
	// path, response status code, and request duration.
	Record(method, path string, status int, duration time.Duration)
}

// MetricsConfig holds configuration for the metrics recording middleware.
type MetricsConfig struct {
	// Recorder receives one observation per request. Required.
	Recorder MetricsRecorder

	// PathFunc extracts the path to record from the request. If nil,
	// r.URL.Path is used.
	PathFunc func(r *http.Request) string
}

// DefaultMetricsConfig returns a config with sensible defaults. The caller
// must set Recorder before use.
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Recorder: nil,
		PathFunc: nil,
	}
}

// Validate checks the MetricsConfig for invalid values.
func (c MetricsConfig) Validate() error {
	if c.Recorder == nil {
		return errNilMetricsRecorder
	}

	return nil
}

// Metrics returns middleware that records request metrics via the configured
// [MetricsRecorder]. The middleware wraps the handler with a
// [ResponseRecorder] to capture the status code.
func Metrics(cfg MetricsConfig) Middleware {
	pathFunc := cfg.PathFunc
	if pathFunc == nil {
		pathFunc = func(r *http.Request) string {
			return r.URL.Path
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := NewResponseRecorder(w)

			next.ServeHTTP(rec, r)

			status := rec.Status()
			if status == 0 {
				status = http.StatusOK
			}

			cfg.Recorder.Record(r.Method, pathFunc(r), status, time.Since(start))
		})
	}
}
