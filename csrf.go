package httputil

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/justinas/nosurf"
	errorfamily "github.com/larsartmann/go-error-family"
)

// CSRF (Cross-Site Request Forgery) protection middleware based on
// justinas/nosurf. Implements double-submit cookie CSRF with HTMX awareness,
// trusted proxy support, and configurable header/field names.

const (
	// DefaultCSRFCookieName is the default name of the CSRF cookie.
	DefaultCSRFCookieName = "csrf_token"
	// DefaultCSRFHeaderName is the default request header containing the CSRF
	// token. HTMX sends this header when configured with hx-headers.
	DefaultCSRFHeaderName = "X-CSRF-Token"
	// DefaultCSRFFieldName is the default form field name for the CSRF token.
	DefaultCSRFFieldName = "csrf_token"
	defaultCSRFMaxAge     = 24 * time.Hour
)

const contentTypePlain = "text/plain; charset=utf-8"

// CSRFErrorHandler handles CSRF validation failures.
type CSRFErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// ForbiddenCSRFHandler responds with HTTP 403 Forbidden and no body. Useful for
// tests and consumers who want to handle CSRF failures via a separate middleware
// rather than rendering the underlying nosurf error.
func ForbiddenCSRFHandler(w http.ResponseWriter, _ *http.Request, _ error) {
	w.WriteHeader(http.StatusForbidden)
}

// ErrCSRFInvalid is returned when a CSRF token is missing, malformed, or does
// not match. Uses justinas/nosurf under the hood for token generation and
// validation.
var ErrCSRFInvalid = errorfamily.NewRejection("csrf_invalid", "invalid or missing CSRF token")

// ErrCSRFConfig is returned when the CSRF configuration is invalid or insecure.
var ErrCSRFConfig = errorfamily.NewInfrastructure("csrf_config", "invalid CSRF configuration")

// CSRFConfig configures CSRF protection.
//
// All fields are optional; zero values use secure defaults.
// Uses justinas/nosurf internally for token generation, masking (BREACH mitigation),
// cookie management, and validation.
type CSRFConfig struct {
	// CookieName is the name of the CSRF cookie.
	// Default: "csrf_token"
	CookieName string

	// HeaderName is the request header containing the CSRF token.
	// HTMX sends this header when configured with hx-headers.
	// Default: "X-CSRF-Token"
	HeaderName string

	// FieldName is the form field name containing the CSRF token.
	// Checked as fallback when the header is not present.
	// Default: "csrf_token"
	FieldName string

	// MaxAge is the cookie max age.
	// Default: 24 hours
	MaxAge time.Duration

	// Secure sets the Secure flag on the cookie.
	// Default: false (auto-detected from request scheme)
	Secure bool

	// SameSite sets the SameSite attribute on the cookie.
	// Default: http.SameSiteLaxMode
	SameSite http.SameSite

	// Domain sets the cookie domain.
	// Default: "" (host-only cookie)
	Domain string

	// Path sets the cookie path.
	// Default: "/"
	Path string

	// TrustedOrigins configures origins allowed for cross-domain CSRF.
	// Default: nil (same-origin only)
	TrustedOrigins []string

	// TrustedProxies lists the IP addresses (or CIDR-notation networks) of
	// reverse proxies that may strip/forward X-Forwarded-* and similar headers.
	// Used by the plaintext-HTTP origin bypass: a request with no Origin/
	// Referer/Sec-Fetch-Site header is only auto-marked as same-origin when
	// the RemoteAddr is one of these trusted proxies (or loopback).
	TrustedProxies []string

	// TrustedProxiesCIDR is the parsed form of TrustedProxies CIDR entries.
	TrustedProxiesCIDR []*net.IPNet

	// AllowPlaintextBypass grants the plaintext-HTTP origin bypass to ALL
	// non-TLS requests when no TrustedProxies are configured.
	// It is INSECURE for internet-facing plain-HTTP deployments.
	AllowPlaintextBypass bool

	// ErrorHandler is called when CSRF validation fails.
	// Default: writes 403 Forbidden with plain text
	ErrorHandler CSRFErrorHandler
}

func (c *CSRFConfig) cookieName() string {
	if c.CookieName != "" {
		return c.CookieName
	}

	return DefaultCSRFCookieName
}

func (c *CSRFConfig) headerName() string {
	if c.HeaderName != "" {
		return c.HeaderName
	}

	return DefaultCSRFHeaderName
}

func (c *CSRFConfig) fieldName() string {
	if c.FieldName != "" {
		return c.FieldName
	}

	return DefaultCSRFFieldName
}

func (c *CSRFConfig) maxAge() time.Duration {
	if c.MaxAge > 0 {
		return c.MaxAge
	}

	return defaultCSRFMaxAge
}

func (c *CSRFConfig) path() string {
	if c.Path != "" {
		return c.Path
	}

	return "/"
}

// Validate checks the CSRF configuration for common misconfigurations.
// Returns a non-nil error if the config would produce insecure or broken behavior.
func (c *CSRFConfig) Validate() error {
	if c.SameSite == http.SameSiteNoneMode && !c.Secure {
		return errorfamily.NewInfrastructure("csrf_samesite_insecure", "SameSite=None requires Secure=true").
			WithCause(ErrCSRFConfig)
	}

	for _, origin := range c.TrustedOrigins {
		if origin == "" || origin == "*" {
			return errorfamily.NewInfrastructure("csrf_unsafe_origin",
				fmt.Sprintf("TrustedOrigins contains unsafe entry %q — use specific domain names only",
					origin)).WithCause(ErrCSRFConfig)
		}
	}

	if !c.Secure {
		slog.Warn("httputil: CSRFConfig.Validate: Secure is false — CSRF cookies will be sent over plain HTTP",
			slog.String("hint", "set Secure=true in production"))
	}

	// Parse TrustedProxies CIDR entries.
	c.TrustedProxiesCIDR = nil
	for _, p := range c.TrustedProxies {
		if p == "" {
			return errorfamily.NewInfrastructure("csrf_unsafe_proxy",
				"TrustedProxies contains empty entry").WithCause(ErrCSRFConfig)
		}

		if strings.Contains(p, "/") {
			_, ipnet, err := net.ParseCIDR(p)
			if err != nil {
				return errorfamily.NewInfrastructure("csrf_invalid_cidr",
					fmt.Sprintf("TrustedProxies contains invalid CIDR %q: %v", p, err)).
					WithCause(ErrCSRFConfig)
			}

			c.TrustedProxiesCIDR = append(c.TrustedProxiesCIDR, ipnet)
		}
	}

	return nil
}

// ConfigureNosurfHandler applies CSRFConfig settings to a nosurf handler.
func ConfigureNosurfHandler(handler *nosurf.CSRFHandler, cfg CSRFConfig) {
	//nolint:gosec,exhaustruct // HttpOnly=false required for double-submit
	cookie := http.Cookie{
		Name:     cfg.cookieName(),
		Path:     cfg.path(),
		Secure:   cfg.Secure,
		HttpOnly: false,
		SameSite: cfg.SameSite,
		MaxAge:   int(cfg.maxAge().Seconds()),
	}
	if cfg.Domain != "" {
		cookie.Domain = cfg.Domain
	}

	handler.SetBaseCookie(cookie)

	handler.SetIsTLSFunc(func(r *http.Request) bool {
		return r.TLS != nil
	})

	if len(cfg.TrustedOrigins) > 0 {
		origins, err := nosurf.StaticOrigins(cfg.TrustedOrigins...)
		if err != nil {
			slog.Error(
				"httputil: invalid TrustedOrigins",
				slog.String("error", err.Error()),
			)
		} else {
			handler.SetIsAllowedOriginFunc(origins)
		}
	}

	failureHandler := cfg.ErrorHandler
	if failureHandler == nil {
		failureHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", contentTypePlain)
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(err.Error())) //nolint:gosec // text/plain prevents HTML rendering
		}
	}

	handler.SetFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reason := nosurf.Reason(r); reason != nil {
			slog.Warn(
				"httputil: CSRF validation failed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("reason", reason.Error()),
			)
		}

		failureHandler(w, r, ErrCSRFInvalid)
	}))
}

// ---------------------------------------------------------------------------
// Context helpers
// ---------------------------------------------------------------------------

type csrfKey struct{}

// WithCSRFToken stores a CSRF token in the context.
func WithCSRFToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, csrfKey{}, token)
}

// CSRFTokenFromContext retrieves the CSRF token stored by CSRFMiddleware.
// Returns an empty string if no token is present.
func CSRFTokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(csrfKey{}).(string)

	return token
}

// CSRFTokenFromRequest extracts the CSRF token from either the nosurf
// request context or the httputil context. Returns "" if no token is present.
func CSRFTokenFromRequest(r *http.Request) string {
	if token := nosurf.Token(r); token != "" {
		return token
	}

	return CSRFTokenFromContext(r.Context())
}

// InvalidateCSRFCookie invalidates the current CSRF cookie, forcing a new token
// to be generated on the next request. Call this on login/logout to prevent
// CSRF fixation attacks.
func InvalidateCSRFCookie(w http.ResponseWriter, cfg CSRFConfig) {
	//nolint:gosec,exhaustruct // HttpOnly=false required for double-submit; http.Cookie has many optional fields
	cookie := &http.Cookie{
		Name:     cfg.cookieName(),
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Now().Add(-time.Hour),
		Path:     cfg.path(),
		Domain:   cfg.Domain,
		Secure:   cfg.Secure,
		HttpOnly: false,
		SameSite: cfg.SameSite,
	}
	http.SetCookie(w, cookie)
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

// CSRFMiddleware returns HTTP middleware that implements double-submit cookie
// CSRF protection with HTMX awareness.
//
// Uses justinas/nosurf internally for:
//   - Cryptographically secure token generation (crypto/rand)
//   - Per-request token masking (BREACH attack mitigation)
//   - Same-origin validation via Origin/Referer/Sec-Fetch-Site headers
//   - Trusted origins support for cross-domain use cases
//
// For GET/HEAD/OPTIONS/TRACE requests, the middleware ensures a CSRF token
// cookie exists and stores the masked token in context for use in templates.
//
// For state-changing methods (POST/PUT/PATCH/DELETE), it validates that the
// request includes a matching token in either:
//   - The X-CSRF-Token header (HTMX default)
//   - A form field named "csrf_token"
func CSRFMiddleware(cfg CSRFConfig) func(http.Handler) http.Handler {
	if err := cfg.Validate(); err != nil {
		slog.Error("httputil: CSRFConfig validation failed", slog.String("error", err.Error()))
	}

	warnEmptyTrustedProxies(cfg)

	return func(next http.Handler) http.Handler {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := nosurf.Token(r); token != "" {
				r = r.WithContext(WithCSRFToken(r.Context(), token))
			}

			next.ServeHTTP(w, r)
		})

		handler := nosurf.New(inner)
		ConfigureNosurfHandler(handler, cfg)

		needsTranslation := cfg.headerName() != DefaultCSRFHeaderName ||
			cfg.fieldName() != DefaultCSRFFieldName

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			SetPlaintextHTTPOrigin(r, cfg)

			if needsTranslation {
				TranslateCSRFHeaders(r, cfg)
			}

			handler.ServeHTTP(w, r)
		})
	}
}

func warnEmptyTrustedProxies(cfg CSRFConfig) {
	if !cfg.AllowPlaintextBypass {
		return
	}

	if len(cfg.TrustedProxies) == 0 && len(cfg.TrustedProxiesCIDR) == 0 {
		slog.Warn(
			"httputil: CSRFConfig.AllowPlaintextBypass is enabled with no TrustedProxies — " +
				"ALL non-TLS requests bypass origin validation. Set TrustedProxies or remove " +
				"AllowPlaintextBypass in production",
		)
	}
}

// SetPlaintextHTTPOrigin sets the Sec-Fetch-Site header to "same-origin" for
// plain HTTP requests without origin headers. This allows nosurf to skip
// origin validation for HTTP deployments behind trusted proxies.
func SetPlaintextHTTPOrigin(r *http.Request, cfg CSRFConfig) {
	if !shouldBypassPlaintextOrigin(r, cfg) {
		return
	}

	r.Header.Set("Sec-Fetch-Site", "same-origin")
}

func shouldBypassPlaintextOrigin(r *http.Request, cfg CSRFConfig) bool {
	if r.TLS != nil {
		return false
	}

	if hasOriginHeader(r) {
		return false
	}

	remoteHost, remoteIP := remoteHostAndIP(r.RemoteAddr)
	if isLoopback(remoteIP) {
		return true
	}

	return isTrustedProxy(remoteHost, remoteIP, r.RemoteAddr, cfg)
}

func hasOriginHeader(r *http.Request) bool {
	return r.Header.Get("Sec-Fetch-Site") != "" ||
		r.Header.Get("Origin") != "" ||
		r.Header.Get("Referer") != ""
}

func remoteHostAndIP(remoteAddr string) (string, net.IP) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	return host, net.ParseIP(host)
}

func isLoopback(ip net.IP) bool {
	return ip != nil && ip.IsLoopback()
}

func isTrustedProxy(remoteHost string, remoteIP net.IP, remoteAddr string, cfg CSRFConfig) bool {
	if len(cfg.TrustedProxies) == 0 && len(cfg.TrustedProxiesCIDR) == 0 {
		return cfg.AllowPlaintextBypass
	}

	if remoteIP != nil {
		for _, cidr := range cfg.TrustedProxiesCIDR {
			if cidr.Contains(remoteIP) {
				return true
			}
		}
	}

	for _, trusted := range cfg.TrustedProxies {
		if trusted == remoteHost || trusted == remoteAddr {
			return true
		}
	}

	return false
}

// TranslateCSRFHeaders maps custom header/field names to nosurf's default
// header name. nosurf hardcodes its header and field names, so we translate
// before passing the request to nosurf.
func TranslateCSRFHeaders(r *http.Request, cfg CSRFConfig) {
	if cfg.headerName() != DefaultCSRFHeaderName {
		if token := r.Header.Get(cfg.headerName()); token != "" {
			r.Header.Set(DefaultCSRFHeaderName, token)

			return
		}
	}

	if cfg.fieldName() != DefaultCSRFFieldName {
		if token := r.PostFormValue(cfg.fieldName()); token != "" {
			r.Header.Set(DefaultCSRFHeaderName, token)
		}
	}
}

// CSRFResponseHeaderMiddleware returns HTTP middleware that automatically sets
// the X-CSRF-Token response header on every request. This eliminates the need
// for individual handlers to manually set the token.
//
// Place this AFTER CSRFMiddleware in the chain so the token is already in context.
func CSRFResponseHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token := CSRFTokenFromRequest(r); token != "" {
			w.Header().Set(DefaultCSRFHeaderName, token)
		}

		next.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------------------
// HTML helpers
// ---------------------------------------------------------------------------

func csrfTokenFormatted(r *http.Request, format func(escaped string) string) string {
	token := CSRFTokenFromRequest(r)
	if token == "" {
		return ""
	}

	return format(html.EscapeString(token))
}

// CSRFTokenHTMLMeta returns an HTML meta tag containing the CSRF token.
func CSRFTokenHTMLMeta(r *http.Request) string {
	return csrfTokenFormatted(r, func(tok string) string {
		return `<meta name="csrf-token" content="` + tok + `">`
	})
}

// CSRFTokenHXHeaders returns an HTMX hx-headers attribute with the CSRF token.
func CSRFTokenHXHeaders(r *http.Request) string {
	token := CSRFTokenFromRequest(r)
	if token == "" {
		return ""
	}

	jsonVal, err := json.Marshal(map[string]string{DefaultCSRFHeaderName: token})
	if err != nil {
		return ""
	}

	return `hx-headers='` + string(jsonVal) + `'`
}

// CSRFTokenFormField returns a hidden input HTML element containing the CSRF token.
func CSRFTokenFormField(r *http.Request) string {
	return csrfTokenFormatted(r, func(tok string) string {
		return `<input type="hidden" name="` + html.EscapeString(
			DefaultCSRFFieldName,
		) + `" value="` + tok + `">`
	})
}

// ---------------------------------------------------------------------------
// Testing helper
// ---------------------------------------------------------------------------

// CSRFTestToken extracts a valid CSRF token AND cookie by making a GET request
// through the given middleware chain. The middleware must include CSRFMiddleware.
// CSRFResponseHeaderMiddleware is optional — without it, the token is
// extracted from the request context instead of the response header.
//
// nosurf uses token masking: the cookie value is NOT the same as the valid
// header token. A masked token is derived from the cookie per-request.
// This helper handles that dance automatically, returning both the masked
// token and the cookie that nosurf set.
func CSRFTestToken(middleware func(http.Handler) http.Handler) (string, *http.Cookie) {
	var ctxToken string

	handler := middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxToken = CSRFTokenFromContext(r.Context())
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	var cookie *http.Cookie

	for _, c := range w.Result().Cookies() {
		if c.Name == DefaultCSRFCookieName {
			cookie = c

			break
		}
	}

	if hdr := w.Header().Get(DefaultCSRFHeaderName); hdr != "" {
		return hdr, cookie
	}

	return ctxToken, cookie
}

// ValidateCSRF checks whether a request passes CSRF validation against the
// given config. Returns (true, nil) when valid, or (false, rec) when the
// request fails validation — the recorder contains the failure response
// (headers, status code, body) that should be copied to the real ResponseWriter.
//
// If the request already has a valid nosurf token (global CSRFMiddleware already
// ran), returns (true, nil) without re-validating.
func ValidateCSRF(r *http.Request, cfg CSRFConfig) (bool, *httptest.ResponseRecorder) {
	if nosurf.Token(r) != "" {
		return true, nil
	}

	var validated bool

	dummy := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		validated = true
	})

	handler := nosurf.New(dummy)
	ConfigureNosurfHandler(handler, cfg)

	SetPlaintextHTTPOrigin(r, cfg)

	needsTranslation := cfg.headerName() != DefaultCSRFHeaderName ||
		cfg.fieldName() != DefaultCSRFFieldName
	if needsTranslation {
		TranslateCSRFHeaders(r, cfg)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r)

	return validated, rec
}
