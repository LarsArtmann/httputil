package httputil

import (
	"net/http"
	"time"
)

// codeMetricsNilRecorder classifies a missing metrics recorder as Rejection.
const codeMetricsNilRecorder = Code("metrics.nil_recorder")

var errNilMetricsRecorder = codeMetricsNilRecorder.Rejection(
	"metrics config: Recorder must not be nil",
)

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
//
// A nil Recorder is invalid; per the validate-and-log contract the
// constructor logs the misconfiguration and still builds a working
// middleware that serves requests without recording, instead of panicking
// on the first request.
func Metrics(cfg MetricsConfig) Middleware {
	validateConfig("MetricsConfig", cfg.Validate())

	pathFunc := cfg.PathFunc
	if pathFunc == nil {
		pathFunc = func(r *http.Request) string {
			return r.URL.Path
		}
	}

	recorder := cfg.Recorder

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if recorder == nil {
				// Nil recorder was already logged at construction; serving
				// without recording is the documented fallback.
				next.ServeHTTP(w, r)

				return
			}

			start := time.Now()
			rec := NewResponseRecorder(w)

			next.ServeHTTP(rec, r)

			status := rec.Status()
			if status == 0 {
				status = http.StatusOK
			}

			recorder.Record(r.Method, pathFunc(r), status, time.Since(start))
		})
	}
}
