package httputil

import (
	"bytes"
	"encoding/json"
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

// FuzzHealthResponse_Encoding fuzzes arbitrary HealthStatus values through the
// JSON encoder to verify the response is always valid, parseable JSON that
// round-trips cleanly and never panics — regardless of the status string. This
// guards the encoding contract that Kubernetes probes and monitoring tools
// depend on. It replaces an earlier version that fuzzed request paths, which the
// health handler ignores entirely.
func FuzzHealthResponse_Encoding(f *testing.F) {
	f.Add("up")
	f.Add("down")
	f.Add("")
	f.Add("status with spaces")
	f.Add(`value"with"quotes`)
	f.Add("unicode: ✓ café")

	f.Fuzz(func(t *testing.T, status string) {
		t.Parallel()

		var buf bytes.Buffer

		err := json.NewEncoder(&buf).Encode(HealthResponse{Status: HealthStatus(status)})
		if err != nil {
			t.Fatalf("Encode error = %v, want nil for status %q", err, status)
		}

		// The encoded output must always be valid, parseable JSON. The exact
		// round-trip value is not asserted because the JSON encoder normalizes
		// invalid UTF-8 (e.g. lone continuation bytes) to U+FFFD.
		var resp HealthResponse

		err = json.Unmarshal(buf.Bytes(), &resp)
		if err != nil {
			t.Fatalf("Unmarshal error = %v, want nil for encoded %q", err, buf.String())
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
