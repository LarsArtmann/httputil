package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	t.Parallel()

	const defaultRemoteAddr = "10.0.0.1:1234"

	tests := []struct {
		name       string
		remoteAddr string
		header     string
		headerVal  string
		want       string
	}{
		{
			name:      "X-Forwarded-For first entry wins",
			header:    "X-Forwarded-For",
			headerVal: "1.2.3.4, 5.6.7.8",
			want:      "1.2.3.4",
		},
		{
			name:      "X-Real-IP fallback",
			header:    "X-Real-IP",
			headerVal: "9.8.7.6",
			want:      "9.8.7.6",
		},
		{
			name: "RemoteAddr with port",
			want: "10.0.0.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "10.0.0.1",
			want:       "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			remoteAddr := tt.remoteAddr
			if remoteAddr == "" {
				remoteAddr = defaultRemoteAddr
			}

			request := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				"/",
				nil,
			)

			request.RemoteAddr = remoteAddr
			if tt.header != "" {
				request.Header.Set(tt.header, tt.headerVal)
			}

			got := ClientIP(request)
			if got != tt.want {
				t.Errorf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
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
