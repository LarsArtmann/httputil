package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	HealthHandler().ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, "Content-Type", "application/json")
	assertBody(t, rec, `{"status":"up"}`+"\n")
}

func TestLiveHandler(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)

	LiveHandler().ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertBody(t, rec, `{"status":"up"}`+"\n")
}

func TestReadyHandler(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	ReadyHandler().ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertBody(t, rec, `{"status":"up"}`+"\n")
}

func TestRegisterHealth(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	RegisterHealth(mux)

	endpoints := []string{"/health", "/health/live", "/health/ready"}

	for _, endpoint := range endpoints {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)

		mux.ServeHTTP(rec, req)

		assertStatus(t, rec, http.StatusOK)
	}
}
