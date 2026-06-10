package httputil

import (
	"net/http"
	"testing"
)

func TestSecurityHeaders_DefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultSecurityHeadersConfig()
	handler := SecurityHeaders(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	tests := []struct{ header, want string }{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		if got := rec.Header().Get(tt.header); got != tt.want {
			t.Errorf("%s = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestSecurityHeaders_CustomCSP(t *testing.T) {
	t.Parallel()

	cfg := SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'",
	}

	handler := SecurityHeaders(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, "Content-Security-Policy", "default-src 'self'")
}

func TestSecurityHeadersConfig_Validate_ValidDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultSecurityHeadersConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestSecurityHeadersConfig_Validate_EmptyConfig(t *testing.T) {
	t.Parallel()

	cfg := SecurityHeadersConfig{}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for empty config", err)
	}
}

func BenchmarkSecurityHeaders(b *testing.B) {
	cfg := DefaultSecurityHeadersConfig()
	middleware := SecurityHeaders(cfg)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}
