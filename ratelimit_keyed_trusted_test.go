package httputil

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTrustedIPRequest builds a request with a controlled RemoteAddr and the
// given forwarding headers (empty argument skips the header).
func newTrustedIPRequest(t *testing.T, remoteAddr, xff, xri string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set(xffHeaderName, xff)
	}

	if xri != "" {
		req.Header.Set(xriHeaderName, xri)
	}

	return req
}

func TestParseTrustedProxies_EmptySubstitutesLoopbackDefault(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies(nil)
	if err != nil {
		t.Fatalf("ParseTrustedProxies(nil) = error %v, want nil", err)
	}

	local := newTrustedIPRequest(t, "127.0.0.1:8443", "203.0.113.7", "")
	if got := set.ClientIP(local); got != "203.0.113.7" {
		t.Errorf("loopback default, trusted peer: got = %q, want %q", got, "203.0.113.7")
	}

	remote := newTrustedIPRequest(t, "192.0.2.1:5555", "203.0.113.7", "")
	if got := set.ClientIP(remote); got != "192.0.2.1" {
		t.Errorf("loopback default, untrusted peer: got = %q, want %q", got, "192.0.2.1")
	}
}

func TestParseTrustedProxies_RejectsInvalidCIDR(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies([]string{"10.0.0.0/33"})
	if err == nil {
		t.Fatalf("ParseTrustedProxies invalid CIDR = (%v, nil), want error", set)
	}

	if !errors.Is(err, errTrustedProxyCIDRInvalid) {
		t.Errorf("errors.Is(err, errTrustedProxyCIDRInvalid) = false, err = %v", err)
	}

	if !InDomain(err, "ratelimit") {
		t.Errorf("InDomain(err, ratelimit) = false, err = %v", err)
	}
}

func TestParseTrustedProxies_RejectsEmptyEntry(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies([]string{"10.0.0.0/8", ""})
	if err == nil {
		t.Fatalf("ParseTrustedProxies empty entry = (%v, nil), want error", set)
	}

	if !errors.Is(err, errTrustedProxyCIDRInvalid) {
		t.Errorf("errors.Is(err, errTrustedProxyCIDRInvalid) = false, err = %v", err)
	}
}

func TestTrustedProxySet_ClientIP_UntrustedPeerIgnoresForwardedHeaders(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	req := newTrustedIPRequest(t, "192.0.2.1:5555", "203.0.113.7", "203.0.113.9")

	if got := set.ClientIP(req); got != "192.0.2.1" {
		t.Errorf("untrusted peer: got = %q, want %q", got, "192.0.2.1")
	}
}

func TestTrustedProxySet_ClientIP_TrustedPeerHonorsXFF(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	req := newTrustedIPRequest(t, "10.0.0.5:443", "203.0.113.7", "")

	if got := set.ClientIP(req); got != "203.0.113.7" {
		t.Errorf("trusted peer: got = %q, want %q", got, "203.0.113.7")
	}
}

func TestTrustedProxySet_ClientIP_ForgedLeftmostEntryIgnored(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies(nil)
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	req := newTrustedIPRequest(t, "127.0.0.1:8443", "9.9.9.9, 203.0.113.7", "")

	if got := set.ClientIP(req); got != "203.0.113.7" {
		t.Errorf("forged leftmost: got = %q, want %q", got, "203.0.113.7")
	}
}

func TestTrustedProxySet_ClientIP_MultiHopTrustedChainResolvesClient(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies([]string{"10.0.0.0/8", "127.0.0.0/8"})
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	req := newTrustedIPRequest(t, "127.0.0.1:8443", "203.0.113.7, 10.0.0.5", "")

	if got := set.ClientIP(req); got != "203.0.113.7" {
		t.Errorf("multi-hop chain: got = %q, want %q", got, "203.0.113.7")
	}
}

func TestTrustedProxySet_ClientIP_AllEntriesTrustedFallsBackToLeftmost(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	req := newTrustedIPRequest(t, "10.0.0.5:443", "10.0.0.9, 10.0.0.7", "")

	if got := set.ClientIP(req); got != "10.0.0.9" {
		t.Errorf("all trusted: got = %q, want leftmost %q", got, "10.0.0.9")
	}
}

func TestTrustedProxySet_ClientIP_MalformedRightmostXFFDistrustsWholeHeader(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies(nil)
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	req := newTrustedIPRequest(t, "127.0.0.1:8443", "203.0.113.7, garbage", "")

	if got := set.ClientIP(req); got != "127.0.0.1" {
		t.Errorf("malformed rightmost XFF: got = %q, want peer host %q", got, "127.0.0.1")
	}
}

func TestTrustedProxySet_ClientIP_GarbageLeftOfUntrustedEntryIgnored(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies(nil)
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	req := newTrustedIPRequest(t, "127.0.0.1:8443", "garbage, 203.0.113.7", "")

	if got := set.ClientIP(req); got != "203.0.113.7" {
		t.Errorf("garbage left of honest entry: got = %q, want %q", got, "203.0.113.7")
	}
}

func TestTrustedProxySet_ClientIP_XRIFallbackWhenNoXFF(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies(nil)
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	req := newTrustedIPRequest(t, "127.0.0.1:8443", "", "203.0.113.7")

	if got := set.ClientIP(req); got != "203.0.113.7" {
		t.Errorf("XRI fallback: got = %q, want %q", got, "203.0.113.7")
	}
}

func TestTrustedProxySet_ClientIP_UnparseableXRIFallsBackToPeerHost(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies(nil)
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	req := newTrustedIPRequest(t, "127.0.0.1:8443", "", "not-an-ip")

	if got := set.ClientIP(req); got != "127.0.0.1" {
		t.Errorf("unparseable XRI: got = %q, want peer host %q", got, "127.0.0.1")
	}
}

func TestTrustedProxySet_ClientIP_IPv6LoopbackPeerTrusted(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies(nil)
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	req := newTrustedIPRequest(t, "[::1]:8443", "2001:db8::1", "")

	if got := set.ClientIP(req); got != "2001:db8::1" {
		t.Errorf("IPv6 loopback peer: got = %q, want %q", got, "2001:db8::1")
	}
}

func TestTrustedProxySet_ClientIP_NeverReturnsEmptyForOddRemoteAddr(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies(nil)
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	req := newTrustedIPRequest(t, "weird-addr", "203.0.113.7", "")

	if got := set.ClientIP(req); got == "" {
		t.Errorf("ClientIP = %q, want non-empty for RemoteAddr %q", got, req.RemoteAddr)
	}
}

func TestTrustedProxySet_ZeroValueTrustsNothing(t *testing.T) {
	t.Parallel()

	var set TrustedProxySet

	req := newTrustedIPRequest(t, "127.0.0.1:8443", "203.0.113.7", "")

	if got := set.ClientIP(req); got != "127.0.0.1" {
		t.Errorf("zero value: got = %q, want peer host %q", got, "127.0.0.1")
	}

	if set.Contains(net.ParseIP("127.0.0.1")) {
		t.Errorf("zero value Contains(loopback) = true, want false")
	}
}

func TestTrustedProxySet_Contains_IPv4AndIPv6(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies([]string{"10.0.0.0/8", "::1/128"})
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	if !set.Contains(net.ParseIP("10.1.2.3")) {
		t.Errorf("Contains(10.1.2.3) = false, want true")
	}

	if !set.Contains(net.ParseIP("::1")) {
		t.Errorf("Contains(::1) = false, want true")
	}

	if set.Contains(net.ParseIP("192.0.2.1")) {
		t.Errorf("Contains(192.0.2.1) = true, want false")
	}

	if set.Contains(nil) {
		t.Errorf("Contains(nil) = true, want false")
	}
}

func TestTrustedProxySet_KeyExtractor_MatchesClientIP(t *testing.T) {
	t.Parallel()

	set, err := ParseTrustedProxies(nil)
	if err != nil {
		t.Fatalf("ParseTrustedProxies = %v, want nil error", err)
	}

	extractor := set.KeyExtractor()

	req := newTrustedIPRequest(t, "127.0.0.1:8443", "203.0.113.7", "")

	if got, want := extractor(req), set.ClientIP(req); got != want {
		t.Errorf("extractor(req) = %q, want %q", got, want)
	}
}

func TestKeyExtractorFromTrustedClientIP_ReturnsWorkingExtractor(t *testing.T) {
	t.Parallel()

	extractor, err := KeyExtractorFromTrustedClientIP([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("KeyExtractorFromTrustedClientIP = %v, want nil error", err)
	}

	req := newTrustedIPRequest(t, "10.0.0.5:443", "203.0.113.7", "")

	if got := extractor(req); got != "203.0.113.7" {
		t.Errorf("extractor(req) = %q, want %q", got, "203.0.113.7")
	}
}

func TestKeyExtractorFromTrustedClientIP_PropagatesCIDRError(t *testing.T) {
	t.Parallel()

	extractor, err := KeyExtractorFromTrustedClientIP([]string{"10.0.0.0/8", "bogus"})
	if err == nil {
		t.Fatalf("KeyExtractorFromTrustedClientIP bogus CIDR = (%v, nil), want error", extractor)
	}

	if extractor != nil {
		t.Errorf("extractor on error = %v, want nil", extractor)
	}

	if !errors.Is(err, errTrustedProxyCIDRInvalid) {
		t.Errorf("errors.Is(err, errTrustedProxyCIDRInvalid) = false, err = %v", err)
	}
}

func FuzzTrustedProxySetClientIP(f *testing.F) {
	seeds := []struct {
		cidr       string
		remoteAddr string
		xff        string
		xri        string
	}{
		{"10.0.0.0/8", "10.0.0.5:443", "203.0.113.7", ""},
		{"10.0.0.0/8", "192.0.2.1:5555", "203.0.113.7", "203.0.113.9"},
		{"127.0.0.0/8", "127.0.0.1:8443", "9.9.9.9, 203.0.113.7", ""},
		{"127.0.0.0/8", "[::1]:8443", "2001:db8::1", ""},
		{"10.0.0.0/8", "10.0.0.5:443", "10.0.0.9, 10.0.0.7", ""},
		{"10.0.0.0/8", "10.0.0.5:443", "garbage, 203.0.113.7", ""},
		{"10.0.0.0/8", "10.0.0.5:443", "203.0.113.7, garbage", ""},
		{"bogus", "10.0.0.5:443", "203.0.113.7", ""},
		{"10.0.0.0/8", "", "", ""},
	}

	for _, seed := range seeds {
		f.Add(seed.cidr, seed.remoteAddr, seed.xff, seed.xri)
	}

	f.Fuzz(func(t *testing.T, cidr, remoteAddr, xff, xri string) {
		set, err := ParseTrustedProxies([]string{cidr})
		if err != nil {
			return
		}

		req := newTrustedIPRequest(t, remoteAddr, xff, xri)

		got := set.ClientIP(req)
		if got == "" && remoteAddr != "" {
			t.Errorf("ClientIP = %q for RemoteAddr %q, want non-empty", got, remoteAddr)
		}

		if !strings.Contains(got, ":") && net.ParseIP(got) == nil && got != remoteAddr {
			peerHost, _ := remoteHostAndIP(remoteAddr)
			if got != peerHost {
				t.Errorf("ClientIP = %q, want a parseable IP or the peer host %q", got, peerHost)
			}
		}
	})
}
