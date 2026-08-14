package httputil

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html"
	"net/http"
)

// CSP (Content Security Policy) nonce generation and propagation.
//
// Generates a cryptographically random, per-request nonce that allows specific
// inline <script> and <style> elements to execute while blocking all others.
// The browser matches the nonce in the Content-Security-Policy header against
// the nonce attribute on inline elements — any mismatched inline content is
// blocked. This eliminates the need for 'unsafe-inline' in CSP directives.
//
// Typical usage:
//
//	stack := httputil.NewMiddlewareStack()
//	stack.Add(httputil.MiddlewareNonce, httputil.Nonce(httputil.DefaultNonceConfig()))
//
//	// In a handler or template:
//	attr := httputil.NonceAttr(r) // returns nonce="abc123"
//	// <script {{ NonceAttr }}>...</script>
//
// For a stricter policy, use ProductionCSPWithNonce as the CSPBuilder.
// Responses with per-request nonces must not be cached — set
// Cache-Control: no-store in your handler or caching middleware.

const (
	// defaultNonceSize is the number of random bytes (before base64 encoding)
	// generated per request. 20 bytes = 160 bits, exceeding the CSP Level 3
	// recommendation of at least 128 bits.
	defaultNonceSize = 20

	// minNonceSize is the minimum allowed nonce size per CSP Level 3
	// (128 bits). Validate rejects anything smaller.
	minNonceSize = 16
)

// codeNonceTooSmall classifies an undersized nonce as Rejection.
const codeNonceTooSmall = Code("nonce.size_too_small")

var errNonceTooSmall = codeNonceTooSmall.Rejection(
	"NonceConfig.Size must be 0 (use default) or at least 16 (128 bits per CSP Level 3 recommendation)",
)

// NonceConfig configures per-request CSP nonce generation and propagation.
type NonceConfig struct {
	// Size is the number of random bytes generated per request before base64
	// encoding. A value of 0 uses the default (20 bytes / 160 bits). Minimum
	// recommended: 16 (128 bits per CSP Level 3).
	Size int

	// CSPBuilder takes the generated base64-encoded nonce and returns the
	// Content-Security-Policy header value. If nil, no CSP header is set —
	// the nonce is only available via NonceFromContext/NonceFromRequest for
	// template use. Default: RecommendedCSPWithNonce.
	CSPBuilder func(nonce string) string
}

// DefaultNonceConfig returns a NonceConfig with sensible defaults: 20 random
// bytes (160 bits) and a CSP policy that allows self plus the nonce for
// script-src and style-src.
func DefaultNonceConfig() NonceConfig {
	return NonceConfig{
		Size:       defaultNonceSize,
		CSPBuilder: RecommendedCSPWithNonce,
	}
}

// RecommendedCSPWithNonce returns a Content-Security-Policy header value that
// extends RecommendedCSP with per-request nonces for script-src and style-src.
// Use this as NonceConfig.CSPBuilder, or call it directly if you prefer to set
// the CSP header yourself.
func RecommendedCSPWithNonce(nonce string) string {
	return fmt.Sprintf(
		"default-src 'self'; script-src 'self' 'nonce-%[1]s'; style-src 'self' 'nonce-%[1]s'",
		nonce,
	)
}

// ProductionCSPWithNonce returns a stricter Content-Security-Policy for
// production deployments: adds object-src 'none' (blocks Flash/plugins),
// base-uri 'self' (prevents base-tag hijacking), and frame-ancestors 'none'
// (clickjacking defense via CSP instead of X-Frame-Options).
// Use this as NonceConfig.CSPBuilder when you want defense-in-depth beyond the
// baseline RecommendedCSPWithNonce.
func ProductionCSPWithNonce(nonce string) string {
	return fmt.Sprintf(
		"default-src 'self'; script-src 'self' 'nonce-%[1]s'; "+
			"style-src 'self' 'nonce-%[1]s'; object-src 'none'; "+
			"base-uri 'self'; frame-ancestors 'none'",
		nonce,
	)
}

// Validate checks the NonceConfig for invalid values. A Size of 0 means
// "use default" and is always valid. Any non-zero Size below 16 (128 bits)
// is rejected per the CSP Level 3 recommendation.
func (c NonceConfig) Validate() error {
	if c.Size != 0 && c.Size < minNonceSize {
		return errNonceTooSmall.WithContextAny("size", c.Size)
	}

	return nil
}

// generateNonce generates a cryptographically random nonce of the given byte
// size and returns it base64-encoded (URL-safe, no padding). Panics only if
// crypto/rand fails, which should never happen on a healthy system.
func generateNonce(size int) string {
	//nolint:makezero // pre-allocated for crypto/rand to fill, not append
	buf := make([]byte, size)

	_, err := rand.Read(buf)
	if err != nil {
		panic("httputil: crypto/rand.Read failed: " + err.Error())
	}

	return base64.RawURLEncoding.EncodeToString(buf)
}

// nonceKey is the context key for storing the per-request CSP nonce.
type nonceKey struct{}

// Nonce returns middleware that generates a per-request CSP nonce, stores it
// in the request context for template access, and optionally sets the
// Content-Security-Policy response header.
func Nonce(cfg NonceConfig) Middleware {
	validateConfig("NonceConfig", cfg.Validate())

	size := cfg.Size
	if size < minNonceSize {
		size = defaultNonceSize
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nonce := generateNonce(size)

			if cfg.CSPBuilder != nil {
				w.Header().Set("Content-Security-Policy", cfg.CSPBuilder(nonce))
			}

			ctx := WithNonce(r.Context(), nonce)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WithNonce stores the nonce in the request context. Retrieve it with
// NonceFromContext or NonceFromRequest.
func WithNonce(parent context.Context, nonce string) context.Context {
	return context.WithValue(parent, nonceKey{}, nonce)
}

// NonceFromContext retrieves the nonce from the context. Returns an empty
// string if no nonce was stored.
func NonceFromContext(ctx context.Context) string {
	nonce, _ := ctx.Value(nonceKey{}).(string)

	return nonce
}

// NonceFromRequest retrieves the nonce from the request context. Returns an
// empty string if no nonce was stored. Convenience wrapper around
// NonceFromContext.
func NonceFromRequest(r *http.Request) string {
	return NonceFromContext(r.Context())
}

// NonceAttr returns an HTML nonce attribute (e.g. `nonce="abc123"`) suitable
// for use in <script> and <style> tags within Go templates:
//
//	<script {{ NonceAttr }}>...</script>
//	<style {{ NonceAttr }}>...</style>
//
// Returns an empty string when no nonce is present, so templates render
// `<script >...</script>` (harmless) instead of `<script nonce="">` (invalid).
// The nonce value is HTML-escaped for defense in depth, though base64
// URL-safe encoding only produces [A-Za-z0-9_-] which requires no escaping.
func NonceAttr(r *http.Request) string {
	nonce := NonceFromRequest(r)
	if nonce == "" {
		return ""
	}

	return `nonce="` + html.EscapeString(nonce) + `"`
}
