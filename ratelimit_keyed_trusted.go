package httputil

import (
	"net"
	"net/http"
	"slices"
	"strings"
)

// Trusted-proxy-gated client-IP resolution for rate-limit keying.
//
// [KeyExtractorFromClientIP] trusts X-Forwarded-For / X-Real-IP blindly,
// which is only safe behind a proxy that strips or overwrites those headers.
// This file adds the gated variant: forwarded headers are honored only when
// the socket peer is inside a trusted CIDR set; otherwise the key comes from
// the socket address. Design note:
// docs/planning/2026-10-10_05-50_trusted-client-ip-extractor-design.md.

const (
	xffHeaderName = "X-Forwarded-For"
	xriHeaderName = "X-Real-IP"
)

// defaultTrustedProxyCIDRs is the loopback-only trust set substituted when
// ParseTrustedProxies receives no entries: local reverse proxies and
// httptest clients connect from loopback, so default behavior honors
// forwarded headers exactly where local deployments and tests need it.
func defaultTrustedProxyCIDRs() []string {
	return []string{"127.0.0.0/8", "::1/128"}
}

// codeRatelimitTrustedProxyCIDRInvalid classifies a CIDR entry that
// net.ParseCIDR rejects while building a TrustedProxySet.
const codeRatelimitTrustedProxyCIDRInvalid = Code("ratelimit.trusted_proxy_cidr_invalid")

// errTrustedProxyCIDRInvalid is a Rejection: an invalid trust set is a
// configuration error, never worth retrying.
var errTrustedProxyCIDRInvalid = codeRatelimitTrustedProxyCIDRInvalid.Rejection(
	"TrustedProxySet contains an invalid CIDR entry",
)

// TrustedProxySet is a parsed, immutable set of trusted proxy networks.
// A value from [ParseTrustedProxies] with no entries trusts loopback only;
// the zero value trusts nothing (no peer's forwarded headers are honored).
// The value is safe for concurrent use: the parsed networks never change
// after ParseTrustedProxies returns.
type TrustedProxySet struct {
	networks []*net.IPNet
}

// ParseTrustedProxies parses CIDR entries into a [TrustedProxySet]. An empty
// or nil slice substitutes the loopback-only default (127.0.0.0/8, ::1/128),
// so local proxies and httptest clients work without configuration; remote
// trusts require explicit entries. An entry net.ParseCIDR rejects fails the
// whole call (classified Rejection) instead of silently narrowing the trust
// set: a typo'd CIDR would otherwise collapse every client onto the proxy's
// address, one shared rate-limit bucket.
func ParseTrustedProxies(cidrs []string) (TrustedProxySet, error) {
	if len(cidrs) == 0 {
		cidrs = defaultTrustedProxyCIDRs()
	}

	networks := make([]*net.IPNet, 0, len(cidrs))

	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return TrustedProxySet{}, errTrustedProxyCIDRInvalid.
				WithContext("cidr", cidr).
				WithContextAny("parse_error", err)
		}

		networks = append(networks, network)
	}

	return TrustedProxySet{networks: networks}, nil
}

// Contains reports whether ip is inside the trust set. The zero
// TrustedProxySet contains nothing, so it trusts no peer.
func (s TrustedProxySet) Contains(ip net.IP) bool {
	for _, network := range s.networks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// KeyExtractor returns a [KeyExtractor] keyed on the trusted-proxy-gated
// client IP ([TrustedProxySet.ClientIP]). The extractor never returns "",
// so a miswired trust set cannot collapse all clients into one bucket.
func (s TrustedProxySet) KeyExtractor() KeyExtractor {
	return s.ClientIP
}

// ClientIP resolves the client IP with the trust gate applied. Forwarded
// headers are read only when the socket peer is inside the trust set;
// otherwise the RemoteAddr host wins. The X-Forwarded-For list is walked
// right-to-left, skipping trusted proxies, so an attacker-supplied leftmost
// entry cannot outrank the entry an honest appending proxy placed after it.
// A malformed list is distrusted as a whole. The result is never empty when
// RemoteAddr is set.
func (s TrustedProxySet) ClientIP(r *http.Request) string {
	peerHost, peerIP := remoteHostAndIP(r.RemoteAddr)

	peer := peerHost
	if peer == "" {
		peer = r.RemoteAddr
	}

	if !s.Contains(peerIP) {
		return peer
	}

	if ip := s.forwardedIP(r); ip != "" {
		return ip
	}

	return peer
}

// forwardedIP resolves the client from the forwarded headers of a trusted
// peer: the rightmost-untrusted X-Forwarded-For entry first, then
// X-Real-IP, or "" when neither carries a parseable candidate.
func (s TrustedProxySet) forwardedIP(r *http.Request) string {
	if xff := r.Header.Get(xffHeaderName); xff != "" {
		if ip := s.clientFromXFF(xff); ip != "" {
			return ip
		}
	}

	xri := strings.TrimSpace(r.Header.Get(xriHeaderName))
	if xri != "" && net.ParseIP(xri) != nil {
		return xri
	}

	return ""
}

// clientFromXFF walks the comma-separated X-Forwarded-For entries from the
// right, skipping entries inside the trust set; the first untrusted entry
// is the client as the honest appending chain reported it. When every entry
// is trusted, the leftmost entry wins. Any unparseable entry aborts the
// walk ("" returned): the chain is not the documented honest-append shape,
// so the whole header is distrusted rather than skipping past garbage.
func (s TrustedProxySet) clientFromXFF(xff string) string {
	entries := strings.Split(xff, ",")

	for _, raw := range slices.Backward(entries) {
		entry := strings.TrimSpace(raw)

		ip := net.ParseIP(entry)
		if ip == nil {
			return ""
		}

		if !s.Contains(ip) {
			return entry
		}
	}

	return strings.TrimSpace(entries[0])
}

// KeyExtractorFromTrustedClientIP returns a [KeyExtractor] that keys on the
// client IP with forwarded headers gated by the trusted-proxy CIDR set
// (see [ParseTrustedProxies] for the empty-slice default and the fail-loud
// CIDR contract). Use it instead of [KeyExtractorFromClientIP] whenever the
// deployment cannot guarantee that an outer proxy strips client-supplied
// forwarding headers.
func KeyExtractorFromTrustedClientIP(cidrs []string) (KeyExtractor, error) {
	set, err := ParseTrustedProxies(cidrs)
	if err != nil {
		return nil, err
	}

	return set.KeyExtractor(), nil
}
