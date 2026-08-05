package httputil

import (
	"errors"
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

// TestSecurityHeaders_AllSet covers every header branch including HSTS,
// which the existing tests omit.
func TestSecurityHeaders_AllSet(t *testing.T) {
	t.Parallel()

	cfg := SecurityHeadersConfig{
		ContentTypeNosniff:      true,
		FrameOptions:            "SAMEORIGIN",
		ReferrerPolicy:          "no-referrer",
		ContentSecurityPolicy:   "default-src 'self'",
		StrictTransportSecurity: "max-age=31536000",
	}

	handler := SecurityHeaders(cfg)(newNoOpHandler())
	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, "X-Content-Type-Options", "nosniff")
	assertHeader(t, rec, "X-Frame-Options", "SAMEORIGIN")
	assertHeader(t, rec, "Referrer-Policy", "no-referrer")
	assertHeader(t, rec, "Content-Security-Policy", "default-src 'self'")
	assertHeader(t, rec, "Strict-Transport-Security", "max-age=31536000")
}

// TestSecurityHeaders_EmptyConfig covers the all-false/empty branch where
// no security headers are set.
func TestSecurityHeaders_EmptyConfig(t *testing.T) {
	t.Parallel()

	cfg := SecurityHeadersConfig{}
	handler := SecurityHeaders(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	for _, header := range []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Content-Security-Policy",
		"Strict-Transport-Security",
	} {
		if got := rec.Header().Get(header); got != "" {
			t.Errorf("%s = %q, want empty", header, got)
		}
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

func TestSecurityHeadersConfig_Validate_RejectsInvalidFrameOptions(t *testing.T) {
	t.Parallel()

	cfg := SecurityHeadersConfig{FrameOptions: "ALLOW-FROM https://evil.example"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for invalid FrameOptions")
	}

	if !errors.Is(err, errSecurityInvalidFrameOptions) {
		t.Errorf("Validate() error = %v, want errSecurityInvalidFrameOptions", err)
	}
}

func TestSecurityHeadersConfig_Validate_AcceptsValidFrameOptions(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "DENY", "SAMEORIGIN"} {
		cfg := SecurityHeadersConfig{FrameOptions: value}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Validate() for FrameOptions=%q: error = %v, want nil", value, err)
		}
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
