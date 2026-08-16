package httputil

import (
	"encoding/json/v2"
	"net/http"
)

// HealthStatus represents the status of a health check.
type HealthStatus string

const (
	// HealthStatusUp indicates the service is healthy.
	HealthStatusUp HealthStatus = "up"
	// HealthStatusDown indicates the service is unhealthy.
	HealthStatusDown HealthStatus = "down"
)

// HealthResponse is the JSON response from a health check endpoint.
type HealthResponse struct {
	Status HealthStatus `json:"status"`
}

// HealthHandler returns an http.HandlerFunc that responds with a simple
// {"status": "up"} JSON payload. Use this for basic liveness probes.
//
// json.Encoder.Encode appends a trailing newline after the JSON body,
// producing `{"status":"up"}\n` (16 bytes). This is intentional: the newline
// improves terminal output and matches the convention of json.Marshal followed
// by fmt.Println. The exact-byte test in health_test.go guards this behavior.
func HealthHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Type", "application/json")
		resp.WriteHeader(http.StatusOK)

		_ = json.MarshalWrite(resp, HealthResponse{Status: HealthStatusUp})
	}
}

// LiveHandler returns an http.HandlerFunc for Kubernetes liveness probes.
// It is functionally equivalent to HealthHandler but semantically distinct.
func LiveHandler() http.HandlerFunc {
	return HealthHandler()
}

// ReadyHandler returns an http.HandlerFunc for Kubernetes readiness probes.
// The default implementation always reports up. To supply a readiness check
// that verifies dependencies (database, cache, etc.), use ReadyHandlerWithProbe.
func ReadyHandler() http.HandlerFunc {
	return HealthHandler()
}

// ReadyHandlerWithProbe returns an http.HandlerFunc that calls the provided
// readiness function on each request. When ready returns true, the handler
// responds with 200 {"status":"up"}. When it returns false, the handler
// responds with 503 {"status":"down"}. This lets Kubernetes route traffic
// away from instances that are alive but not yet ready to serve (e.g., still
// warming caches or waiting on dependencies).
func ReadyHandlerWithProbe(ready func() bool) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Type", "application/json")

		if ready() {
			resp.WriteHeader(http.StatusOK)

			_ = json.MarshalWrite(resp, HealthResponse{Status: HealthStatusUp})

			return
		}

		resp.WriteHeader(http.StatusServiceUnavailable)

		_ = json.MarshalWrite(resp, HealthResponse{Status: HealthStatusDown})
	}
}

// RegisterHealth registers /health, /health/live, and /health/ready on the
// given *http.ServeMux using the default handlers.
func RegisterHealth(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", HealthHandler())
	mux.HandleFunc("GET /health/live", LiveHandler())
	mux.HandleFunc("GET /health/ready", ReadyHandler())
}
