package httputil

import (
	"bytes"
	"fmt"
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

func TestReadyHandlerWithProbeReady(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	ReadyHandlerWithProbe(func() bool { return true }).ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, "Content-Type", "application/json")
	assertBody(t, rec, `{"status":"up"}`+"\n")
}

func TestReadyHandlerWithProbeNotReady(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	ReadyHandlerWithProbe(func() bool { return false }).ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusServiceUnavailable)
	assertHeader(t, rec, "Content-Type", "application/json")
	assertBody(t, rec, `{"status":"down"}`+"\n")
}

// TestHealthHandler_ExactBytes asserts the response body is exactly the
// expected byte sequence. This guards against future JSON library changes
// that might alter whitespace, field ordering, or trailing newline behavior.
func TestHealthHandler_ExactBytes(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	HealthHandler().ServeHTTP(rec, req)

	expected := []byte(`{"status":"up"}` + "\n")

	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Errorf("body = %q, want exact bytes %q", rec.Body.String(), string(expected))
	}

	const expectedLen = 16 // len(`{"status":"up"}`) + 1 (newline)

	if rec.Body.Len() != expectedLen {
		t.Errorf("body length = %d, want %d", rec.Body.Len(), expectedLen)
	}
}

func FuzzHealthHandler(f *testing.F) {
	f.Add("")
	f.Add("GET")
	f.Add("/health/ready")

	f.Fuzz(func(t *testing.T, path string) {
		t.Parallel()

		handler := HealthHandler()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/"+path, nil)

		handler.ServeHTTP(rec, req)

		// Primary assertion: no panic. Secondary: 200 for the health handler.
		if rec.Code != http.StatusOK {
			// HealthHandler always returns 200 regardless of path.
			t.Errorf("HealthHandler status = %d, want 200", rec.Code)
		}
	})
}

func ExampleReadyHandlerWithProbe() {
	handler := ReadyHandlerWithProbe(func() bool { return true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	handler.ServeHTTP(rec, req)

	fmt.Print(rec.Body.String())

	// Output: {"status":"up"}
}
