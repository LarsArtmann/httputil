package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newClientIPRequest builds a GET request with a specific RemoteAddr and an
// optional spoofable forwarding header.
func newClientIPRequest(remoteAddr, header, headerVal string) *http.Request {
	request := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/",
		nil,
	)

	request.RemoteAddr = remoteAddr
	if header != "" {
		request.Header.Set(header, headerVal)
	}

	return request
}

func TestClientIP_XForwardedForFirstEntryWins(t *testing.T) {
	t.Parallel()

	request := newClientIPRequest("10.0.0.1:1234", "X-Forwarded-For", "1.2.3.4, 5.6.7.8")

	if got := ClientIP(request); got != "1.2.3.4" {
		t.Errorf("ClientIP() = %q, want %q", got, "1.2.3.4")
	}
}

func TestClientIP_XRealIPFallback(t *testing.T) {
	t.Parallel()

	request := newClientIPRequest("10.0.0.1:1234", "X-Real-IP", "9.8.7.6")

	if got := ClientIP(request); got != "9.8.7.6" {
		t.Errorf("ClientIP() = %q, want %q", got, "9.8.7.6")
	}
}

func TestClientIP_RemoteAddrWithPort(t *testing.T) {
	t.Parallel()

	request := newClientIPRequest("10.0.0.1:1234", "", "")

	if got := ClientIP(request); got != "10.0.0.1" {
		t.Errorf("ClientIP() = %q, want %q", got, "10.0.0.1")
	}
}

func TestClientIP_RemoteAddrWithoutPort(t *testing.T) {
	t.Parallel()

	request := newClientIPRequest("10.0.0.1", "", "")

	if got := ClientIP(request); got != "10.0.0.1" {
		t.Errorf("ClientIP() = %q, want %q", got, "10.0.0.1")
	}
}

func BenchmarkClientIP(b *testing.B) {
	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	req.RemoteAddr = "10.0.0.1:1234"

	b.ReportAllocs()

	for b.Loop() {
		ClientIP(req)
	}
}

func FuzzClientIP(f *testing.F) {
	f.Add("1.2.3.4")
	f.Add("1.2.3.4, 5.6.7.8, 9.10.11.12")
	f.Add("")
	f.Add("::1")
	f.Add("not-an-ip")

	f.Fuzz(func(t *testing.T, xff string) {
		req := newTestRequest(http.MethodGet, "/", "")
		req.Header.Set("X-Forwarded-For", xff)
		req.RemoteAddr = "10.0.0.1:1234"

		ClientIP(req)
	})
}

// TestClientIP_TrustsSpoofableHeaders documents the non-obvious contract:
// ClientIP trusts X-Forwarded-For and X-Real-IP blindly. A direct client can
// forge these headers; only deploy behind a reverse proxy that strips or
// overwrites them. This test is the executable form of the AGENTS.md warning.
func TestClientIP_TrustsSpoofableHeaders(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:4444"
	req.Header.Set("X-Forwarded-For", "6.6.6.6")

	if got := ClientIP(req); got != "6.6.6.6" {
		t.Errorf("ClientIP = %q, want the forged 6.6.6.6 (blind trust is the contract)", got)
	}

	req.Header.Set("X-Real-IP", "7.7.7.7")

	if got := ClientIP(req); got != "6.6.6.6" {
		t.Errorf("ClientIP = %q, want 6.6.6.6 (XFF wins over X-Real-IP)", got)
	}

	req.Header.Del("X-Forwarded-For")

	if got := ClientIP(req); got != "7.7.7.7" {
		t.Errorf("ClientIP = %q, want 7.7.7.7 (X-Real-IP wins over RemoteAddr)", got)
	}
}
