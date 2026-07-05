package httputil

import (
	"encoding/json"
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
func HealthHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Type", "application/json")
		resp.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(resp).Encode(HealthResponse{Status: HealthStatusUp})
	}
}

// LiveHandler returns an http.HandlerFunc for Kubernetes liveness probes.
// It is functionally equivalent to HealthHandler but semantically distinct.
func LiveHandler() http.HandlerFunc {
	return HealthHandler()
}

// ReadyHandler returns an http.HandlerFunc for Kubernetes readiness probes.
// The default implementation always reports up. To supply a readiness check
// that verifies dependencies (database, cache, etc.), do not use this helper —
// register your own handler at /health/ready instead.
func ReadyHandler() http.HandlerFunc {
	return HealthHandler()
}

// RegisterHealth registers /health, /health/live, and /health/ready on the
// given *http.ServeMux using the default handlers.
func RegisterHealth(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", HealthHandler())
	mux.HandleFunc("GET /health/live", LiveHandler())
	mux.HandleFunc("GET /health/ready", ReadyHandler())
}
