package httputil

import (
	"context"
	"encoding/json/v2"
	"html"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	// token. HTMX sends this header when configured with hx-headers. The value
	// uses Go's canonical MIME header form; HTTP header names are
	// case-insensitive, so this matches the conventional X-CSRF-Token spelling.
	DefaultCSRFHeaderName = "X-Csrf-Token"
	// DefaultCSRFFieldName is the default form field name for the CSRF token.
	DefaultCSRFFieldName = "csrf_token"
	defaultCSRFMaxAge    = 24 * time.Hour
)

const contentTypePlain = "text/plain; charset=utf-8"

// Header names and values used for origin attestation and validation.
const (
	headerOrigin          = "Origin"
	headerReferer         = "Referer"
	headerSecFetchSite    = "Sec-Fetch-Site"
	attestationSameOrigin = "same-origin"
	originNull            = "null"
)

// ErrorHandler handles CSRF validation failures.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// ForbiddenHandler responds with HTTP 403 Forbidden and no body. Useful for
// tests and consumers who want to handle CSRF failures via a separate middleware
// rather than rendering the underlying nosurf error.
func ForbiddenHandler(w http.ResponseWriter, _ *http.Request, _ error) {
	w.WriteHeader(http.StatusForbidden)
}

// ErrCSRFInvalid is returned when a CSRF token is missing, malformed, or does
// not match. Uses justinas/nosurf under the hood for token generation and
// validation.
var ErrCSRFInvalid = errorfamily.NewRejection(
	string(codeCSRFInvalid),
	"invalid or missing CSRF token",
)

// ErrCSRFConfig is returned when the CSRF configuration is invalid or insecure.
var ErrCSRFConfig = errorfamily.NewInfrastructure(
	string(codeCSRFConfig),
	"invalid CSRF configuration",
)

// ErrCSRFAttestationConflict is returned when a client-supplied
// "Sec-Fetch-Site: same-origin" attestation is contradicted by an Origin
// header from a different, untrusted origin. Browsers never produce this
// combination — they attest cross-origin requests as cross-site — so a
// contradiction means the attestation was forged.
var ErrCSRFAttestationConflict = errorfamily.NewRejection(
	string(codeCSRFAttestationConflict),
	"Sec-Fetch-Site attestation contradicts the Origin header",
)

// Legacy underscore-spelled codes for the exported CSRF sentinels, kept for
// backward compatibility; new codes use the domain.dot format.
const (
	codeCSRFInvalid = Code("csrf_invalid")
	codeCSRFConfig  = Code("csrf_config")
)

// Error codes for CSRFConfig validation, classified as Infrastructure to
// match ErrCSRFConfig (kept for backward compatibility).
const (
	codeCSRFSameSiteInsecure = Code("csrf_samesite_insecure")
	codeCSRFUnsafeOrigin     = Code("csrf_unsafe_origin")
	codeCSRFUnsafeProxy      = Code("csrf_unsafe_proxy")
	codeCSRFInvalidCIDR      = Code("csrf_invalid_cidr")
)

// codeCSRFAttestationConflict classifies requests whose same-origin Fetch
// Metadata attestation is contradicted by their Origin header (Rejection:
// forged client input, never retried).
const codeCSRFAttestationConflict = Code("csrf.origin_attestation_conflict")

// codeCSRFMaxAgeNegative classifies a negative CSRFConfig.MaxAge (Rejection:
// fix the config, never retry). A negative cookie MaxAge deletes the cookie,
// which is never a meaningful CSRF configuration.
const codeCSRFMaxAgeNegative = Code("csrf.max_age_negative")

var errCSRFMaxAgeNegative = codeCSRFMaxAgeNegative.Rejection(
	"CSRFConfig: MaxAge must not be negative; zero uses the 24h default",
)

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
	// Default: "X-Csrf-Token" (case-insensitive on the wire)
	HeaderName string

	// FieldName is the form field name containing the CSRF token.
	// Checked as fallback when the header is not present.
	// Default: "csrf_token"
	FieldName string

	// MaxAge is the cookie max age.
	// Default: 24 hours. A negative value fails Validate
	// (csrf.max_age_negative); zero uses the default.
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
	// CSRFMiddleware populates it from TrustedProxies at construction time;
	// Validate does not mutate the config. It may also be set directly.
	TrustedProxiesCIDR []*net.IPNet

	// AllowPlaintextBypass grants the plaintext-HTTP origin bypass to ALL
	// non-TLS requests when no TrustedProxies are configured.
	// It is INSECURE for internet-facing plain-HTTP deployments.
	AllowPlaintextBypass bool

	// ErrorHandler is called when CSRF validation fails.
	// Default: writes 403 Forbidden with plain text
	ErrorHandler ErrorHandler
}

func (c CSRFConfig) cookieName() string {
	if c.CookieName != "" {
		return c.CookieName
	}

	return DefaultCSRFCookieName
}

func (c CSRFConfig) headerName() string {
	if c.HeaderName != "" {
		return c.HeaderName
	}

	return DefaultCSRFHeaderName
}

func (c CSRFConfig) fieldName() string {
	if c.FieldName != "" {
		return c.FieldName
	}

	return DefaultCSRFFieldName
}

func (c CSRFConfig) maxAge() time.Duration {
	if c.MaxAge > 0 {
		return c.MaxAge
	}

	return defaultCSRFMaxAge
}

func (c CSRFConfig) path() string {
	if c.Path != "" {
		return c.Path
	}

	return "/"
}

// Validate checks the CSRF configuration for common misconfigurations.
// Returns a non-nil error if the config would produce insecure or broken behavior.
//
// Validate is pure: it neither logs nor mutates the receiver. (Before the
// v1.0 cleanup it parsed TrustedProxies into TrustedProxiesCIDR as a side
// effect and warned about Secure=false; CSRFMiddleware now does both at
// construction time.)
func (c CSRFConfig) Validate() error {
	if c.MaxAge < 0 {
		return errCSRFMaxAgeNegative.WithContextAny("max_age", c.MaxAge)
	}

	if c.SameSite == http.SameSiteNoneMode && !c.Secure {
		return codeCSRFSameSiteInsecure.Infrastructure("SameSite=None requires Secure=true").
			WithCause(ErrCSRFConfig).
			WithContextAny("secure", c.Secure)
	}

	for _, origin := range c.TrustedOrigins {
		if origin == "" || origin == "*" {
			return codeCSRFUnsafeOrigin.Infrastructure(
				"TrustedOrigins contains unsafe entry; use specific domain names only",
			).WithCause(ErrCSRFConfig).
				WithContext("origin", origin)
		}
	}

	for _, p := range c.TrustedProxies {
		if p == "" {
			return codeCSRFUnsafeProxy.Infrastructure(
				"TrustedProxies contains empty entry",
			).WithCause(ErrCSRFConfig)
		}

		if strings.Contains(p, "/") {
			if _, _, err := net.ParseCIDR(p); err != nil {
				return codeCSRFInvalidCIDR.Infrastructure("TrustedProxies contains invalid CIDR").
					WithCause(ErrCSRFConfig).
					WithContext("proxy", p).
					WithContextAny("parse_error", err)
			}
		}
	}

	return nil
}

// withParsedTrustedProxies returns a copy of c with TrustedProxiesCIDR
// populated from its TrustedProxies entries. CSRFMiddleware calls it once at
// construction so the request path never parses CIDRs. Any entry that fails
// to parse leaves the CIDR list empty, matching the validate-and-log
// contract: an invalid config never aborts the middleware but must not
// widen proxy trust either.
func (c CSRFConfig) withParsedTrustedProxies() CSRFConfig {
	out := c
	out.TrustedProxiesCIDR = nil

	cidrs := make([]*net.IPNet, 0, len(c.TrustedProxies))

	for _, p := range c.TrustedProxies {
		if p == "" {
			out.TrustedProxiesCIDR = nil

			return out
		}

		if strings.Contains(p, "/") {
			_, ipnet, err := net.ParseCIDR(p)
			if err != nil {
				out.TrustedProxiesCIDR = nil

				return out
			}

			cidrs = append(cidrs, ipnet)
		}
	}

	out.TrustedProxiesCIDR = cidrs

	return out
}

// ConfigureNosurfHandler applies CSRFConfig settings to a nosurf handler.
func ConfigureNosurfHandler(handler *nosurf.CSRFHandler, cfg CSRFConfig) {
	//nolint:gosec,exhaustruct_v5 // HttpOnly=false required for double-submit
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

	handler.SetFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reason := nosurf.Reason(r)
		if reason != nil {
			slog.Warn(
				"httputil: CSRF validation failed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("reason", reason.Error()),
			)
		}

		handleCSRFRejection(cfg, w, r, ErrCSRFInvalid)
	}))
}

// handleCSRFRejection dispatches a CSRF failure to the configured
// ErrorHandler, falling back to a 403 plain-text response.
func handleCSRFRejection(cfg CSRFConfig, w http.ResponseWriter, r *http.Request, err error) {
	if cfg.ErrorHandler != nil {
		cfg.ErrorHandler(w, r, err)

		return
	}

	w.Header().Set("Content-Type", contentTypePlain)
	w.WriteHeader(http.StatusForbidden)

	writeCommittedBody(w, []byte(err.Error()))
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
	//nolint:gosec,exhaustruct_v5 // HttpOnly=false required for double-submit; http.Cookie has many optional fields
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
//   - The X-Csrf-Token header (HTMX default)
//   - A form field named "csrf_token"
//
// # Origin attestation trust model
//
// nosurf (verified at v1.2.0) skips Origin/Referer validation entirely when a
// request carries a literal "Sec-Fetch-Site: same-origin" header. Browsers set
// Sec-Fetch-Site truthfully and JavaScript cannot forge it (forbidden header
// name), so the attestation is reliable against the classic cross-site
// attacker; any non-browser client, however, can send it manually. Two
// deliberate consequences:
//
//   - A client-supplied attestation (or the plaintext-proxy bypass) skips only
//     the origin check — the masked-token check still gates every
//     state-changing request. This is what lets non-browser API clients, which
//     cannot pass Origin/Referer validation, work with valid tokens.
//   - A client-supplied attestation contradicted by an Origin header from a
//     different, untrusted origin is rejected with ErrCSRFAttestationConflict
//     before nosurf sees the request: browsers never produce that combination,
//     so it can only be forged.
func CSRFMiddleware(cfg CSRFConfig) func(http.Handler) http.Handler {
	validateConfig("CSRFConfig", cfg.Validate())

	if !cfg.Secure {
		slog.Warn(
			"httputil: CSRFConfig: Secure is false — CSRF cookies will be sent over plain HTTP",
			slog.String("hint", "set Secure=true in production"),
		)
	}

	// Parse the trusted-proxy CIDRs once here so Validate stays pure and the
	// request path never re-parses configuration.
	cfg = cfg.withParsedTrustedProxies()

	warnEmptyTrustedProxies(cfg)

	trustedOrigins := parseTrustedOriginURLs(cfg.TrustedOrigins)

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
			if origin := contradictedAttestationOrigin(r, trustedOrigins); origin != "" {
				slog.Warn(
					"httputil: CSRF rejected request with forged same-origin attestation",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("origin", origin),
				)

				handleCSRFRejection(
					cfg,
					w,
					r,
					ErrCSRFAttestationConflict.WithContext("origin", origin),
				)

				return
			}

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
//
// It only fills a blank: requests that already carry Sec-Fetch-Site, Origin,
// or Referer pass through untouched. The forged value is exactly what a
// non-browser client could send itself — nosurf v1.2.0 short-circuits origin
// validation on any literal same-origin attestation — so the bypass grants no
// client new power; the masked-token check still applies.
func SetPlaintextHTTPOrigin(r *http.Request, cfg CSRFConfig) {
	if !shouldBypassPlaintextOrigin(r, cfg) {
		return
	}

	r.Header.Set(headerSecFetchSite, attestationSameOrigin)
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
	return r.Header.Get(headerSecFetchSite) != "" ||
		r.Header.Get(headerOrigin) != "" ||
		r.Header.Get(headerReferer) != ""
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

// parseTrustedOriginURLs parses TrustedOrigins entries into URLs for the
// attestation-consistency check. Entries that fail to parse are skipped:
// they cannot match any Origin header, matching how nosurf discards them.
func parseTrustedOriginURLs(origins []string) []*url.URL {
	parsed := make([]*url.URL, 0, len(origins))

	for _, origin := range origins {
		u, parseErr := url.Parse(origin)
		if parseErr != nil {
			continue
		}

		parsed = append(parsed, u)
	}

	return parsed
}

// isUnsafeCSRFMethod mirrors nosurf's safe-method set: origin validation (and
// therefore attestation-consistency checking) applies only to state-changing
// requests.
func isUnsafeCSRFMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	}

	return true
}

// contradictedAttestationOrigin returns the Origin header value when a
// client-supplied "Sec-Fetch-Site: same-origin" attestation is contradicted
// by an Origin header that nosurf itself would reject (unparseable, or from a
// different origin that is neither self nor trusted), and "" otherwise.
//
// nosurf v1.2.0 skips Origin/Referer validation on any literal same-origin
// attestation before it ever consults the Origin header, so without this
// check a forged attestation would smuggle a disallowed Origin past
// validation. Browsers never produce the combination — Sec-Fetch-Site is a
// forbidden header name set by the browser itself, and a cross-origin request
// is attested as cross-site — so a contradiction means the attestation was
// forged. The "null" origin is left to nosurf, which treats it as absent.
func contradictedAttestationOrigin(r *http.Request, trustedOrigins []*url.URL) string {
	if !isUnsafeCSRFMethod(r.Method) {
		return ""
	}

	if r.Header.Get(headerSecFetchSite) != attestationSameOrigin {
		return ""
	}

	originStr := r.Header.Get(headerOrigin)
	if originStr == "" || originStr == originNull {
		return ""
	}

	origin, parseErr := url.Parse(originStr)
	if parseErr != nil {
		return originStr
	}

	if origin.Host == r.Host && origin.Scheme == requestScheme(r) {
		return ""
	}

	for _, trusted := range trustedOrigins {
		if origin.Host == trusted.Host && origin.Scheme == trusted.Scheme {
			return ""
		}
	}

	return originStr
}

// requestScheme reports the URL scheme a client used, based on whether the
// connection is TLS-terminated locally. It mirrors how nosurf builds the self
// origin for same-origin comparisons.
func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}

	return "http"
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
// the X-Csrf-Token response header on every request. This eliminates the need
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

	if origin := contradictedAttestationOrigin(
		r,
		parseTrustedOriginURLs(cfg.TrustedOrigins),
	); origin != "" {
		rec := httptest.NewRecorder()
		handleCSRFRejection(cfg, rec, r, ErrCSRFAttestationConflict.WithContext("origin", origin))

		return false, rec
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
