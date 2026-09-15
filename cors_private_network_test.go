package httputil

import (
	"net/http"
	"testing"
)

// newPrivateNetworkPreflight builds an OPTIONS request shaped like Chrome's
// Private Network Access / Local Network Access preflight: an Origin, the
// Access-Control-Request-Method marker, and the private-network request header
// Chrome sends before a page may fetch subresources in a more-private address
// space.
func newPrivateNetworkPreflight() *http.Request {
	req := newTestRequest(http.MethodOptions, "/test", "https://public.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Allow-Private-Network", "true")

	return req
}

func TestCORS_Preflight_PrivateNetworkAllowed_WhenConfigured(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	cfg.AllowPrivateNetwork = true

	inner := newNoOpHandler()
	rec := newRecorder()

	CORS(cfg)(inner).ServeHTTP(rec, newPrivateNetworkPreflight())

	assertStatus(t, rec, http.StatusNoContent)
	assertHeader(t, rec, "Access-Control-Allow-Private-Network", "true")
}

func TestCORS_Preflight_PrivateNetworkOmitted_ByDefault(t *testing.T) {
	t.Parallel()

	inner := newNoOpHandler()
	rec := newRecorder()

	CORS(DefaultCORSConfig())(inner).ServeHTTP(rec, newPrivateNetworkPreflight())

	assertStatus(t, rec, http.StatusNoContent)

	if got := rec.Header().Get("Access-Control-Allow-Private-Network"); got != "" {
		t.Errorf("Allow-Private-Network = %q, want absent (default off)", got)
	}
}

func TestCORS_ActualRequest_PrivateNetworkOmitted_WhenConfigured(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	cfg.AllowPrivateNetwork = true

	called := false
	inner := newCountingHandler(&called)
	rec := newRecorder()

	req := newTestRequest(http.MethodGet, "/test", "https://public.example.com")

	CORS(cfg)(inner).ServeHTTP(rec, req)

	if !called {
		t.Error("next handler should be called for actual request")
	}

	if got := rec.Header().Get("Access-Control-Allow-Private-Network"); got != "" {
		t.Errorf("Allow-Private-Network = %q, want absent on actual request", got)
	}
}

func TestCORS_PreflightPassthrough_PrivateNetworkOmitted_WhenConfigured(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	cfg.AllowPrivateNetwork = true
	cfg.OptionsPassthrough = true

	called := false
	inner := newCountingHandler(&called)
	rec := newRecorder()

	CORS(cfg)(inner).ServeHTTP(rec, newPrivateNetworkPreflight())

	if !called {
		t.Error("next handler should be called for preflight passthrough")
	}

	if got := rec.Header().Get("Access-Control-Allow-Private-Network"); got != "" {
		t.Errorf("Allow-Private-Network = %q, want absent (handler owns the preflight)", got)
	}
}
