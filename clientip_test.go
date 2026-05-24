package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP_XForwardedFor(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	r.RemoteAddr = "10.0.0.1:1234"

	got := ClientIP(r)
	if got != "1.2.3.4" {
		t.Errorf("ClientIP() = %q, want %q", got, "1.2.3.4")
	}
}

func TestClientIP_XRealIP(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Real-IP", "9.8.7.6")
	r.RemoteAddr = "10.0.0.1:1234"

	got := ClientIP(r)
	if got != "9.8.7.6" {
		t.Errorf("ClientIP() = %q, want %q", got, "9.8.7.6")
	}
}

func TestClientIP_RemoteAddr(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"

	got := ClientIP(r)
	if got != "10.0.0.1" {
		t.Errorf("ClientIP() = %q, want %q", got, "10.0.0.1")
	}
}

func TestClientIP_RemoteAddrNoPort(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1"

	got := ClientIP(r)
	if got != "10.0.0.1" {
		t.Errorf("ClientIP() = %q, want %q", got, "10.0.0.1")
	}
}
