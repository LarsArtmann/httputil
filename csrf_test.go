package httputil

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCSRFMiddleware_GETSetsTokenInContext(t *testing.T) {
	t.Parallel()

	var ctxToken string

	mw := CSRFMiddleware(CSRFConfig{})

	handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxToken = CSRFTokenFromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if ctxToken == "" {
		t.Fatal("CSRF token not set in context for GET request")
	}

	// Cookie should be set.
	var hasCookie bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == DefaultCSRFCookieName {
			hasCookie = true

			break
		}
	}

	if !hasCookie {
		t.Fatal("CSRF cookie not set")
	}
}

func TestCSRFMiddleware_POSTWithoutTokenRejected(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST without token should be rejected with 403, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_POSTWithValidTokenAccepted(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{})

	// First, make a GET to get token + cookie.
	token, cookie := CSRFTestToken(mw)
	if token == "" {
		t.Fatal("CSRFTestToken returned empty token")
	}

	if cookie == nil {
		t.Fatal("CSRFTestToken returned nil cookie")
	}

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set(DefaultCSRFHeaderName, token)
	req.AddCookie(cookie)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST with valid token should be accepted, got %d", rec.Code)
	}
}

func TestCSRFResponseHeaderMiddleware_SetsHeader(t *testing.T) {
	t.Parallel()

	stack := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		CSRFMiddleware(CSRFConfig{}),
		CSRFResponseHeaderMiddleware,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	stack.ServeHTTP(rec, req)

	if rec.Header().Get(DefaultCSRFHeaderName) == "" {
		t.Fatal("X-CSRF-Token response header not set")
	}
}

func TestCSRFTokenFromContext_EmptyWhenNotSet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	if token := CSRFTokenFromContext(ctx); token != "" {
		t.Fatalf("expected empty token, got %q", token)
	}
}

func TestWithCSRFToken_RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "test-token")
	if got := CSRFTokenFromContext(ctx); got != "test-token" {
		t.Fatalf("got %q, want %q", got, "test-token")
	}
}

func TestCSRFConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{}
	if cfg.cookieName() != DefaultCSRFCookieName {
		t.Errorf("cookieName = %q, want %q", cfg.cookieName(), DefaultCSRFCookieName)
	}

	if cfg.headerName() != DefaultCSRFHeaderName {
		t.Errorf("headerName = %q, want %q", cfg.headerName(), DefaultCSRFHeaderName)
	}

	if cfg.fieldName() != DefaultCSRFFieldName {
		t.Errorf("fieldName = %q, want %q", cfg.fieldName(), DefaultCSRFFieldName)
	}

	if cfg.path() != "/" {
		t.Errorf("path = %q, want %q", cfg.path(), "/")
	}
}

func TestCSRFConfig_Validate_SameSiteNoneWithoutSecure(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{
		SameSite: http.SameSiteNoneMode,
		Secure:   false,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for SameSite=None without Secure")
	}
}

func TestForbiddenHandler_RespondsWith403(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	ForbiddenHandler(rec, nil, nil)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestInvalidateCSRFCookie_SetsExpiredCookie(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	InvalidateCSRFCookie(rec, CSRFConfig{})

	var found bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == DefaultCSRFCookieName {
			found = true
			if c.MaxAge != -1 {
				t.Errorf("MaxAge = %d, want -1", c.MaxAge)
			}
		}
	}

	if !found {
		t.Fatal("invalidation cookie not set")
	}
}

func TestCSRFTokenFormField_ReturnsEmptyWhenNoToken(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := CSRFTokenFormField(req); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestCSRFTokenHTMLMeta_ReturnsEmptyWhenNoToken(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := CSRFTokenHTMLMeta(req); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Coverage closure tests for CSRF
// ---------------------------------------------------------------------------

func TestValidateCSRF_RejectsRequestWithoutToken(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	ok, rec := ValidateCSRF(req, CSRFConfig{})
	if ok {
		t.Fatal("expected ok=false for request without token")
	}

	if rec == nil {
		t.Fatal("expected non-nil recorder")
	} else if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestValidateCSRF_AlreadyValidatedReturnsTrue(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{})

	var captured *http.Request

	handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = r
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	handler.ServeHTTP(rec, req)

	if captured == nil {
		t.Fatal("handler did not run")
	}

	ok, rec2 := ValidateCSRF(captured, CSRFConfig{})
	if !ok {
		t.Fatal("expected ok=true when nosurf token already present")
	}

	if rec2 != nil {
		t.Fatal("expected nil recorder when already validated")
	}
}

func TestValidateCSRF_ValidRequestPasses(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{}
	mw := CSRFMiddleware(cfg)
	token, cookie := CSRFTestToken(mw)
	if token == "" {
		t.Fatal("CSRFTestToken returned empty token")
	}

	if cookie == nil {
		t.Fatal("CSRFTestToken returned nil cookie")
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set(DefaultCSRFHeaderName, token)
	req.AddCookie(cookie)

	ok, _ := ValidateCSRF(req, cfg)
	if !ok {
		t.Fatal("expected ok=true for valid request with token+cookie")
	}
}

func TestCSRFMiddleware_InvalidConfigContinues(t *testing.T) {
	t.Parallel()

	// Invalid config triggers slog.Error but middleware must still work.
	cfg := CSRFConfig{SameSite: http.SameSiteNoneMode, Secure: false}
	mw := CSRFMiddleware(cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 despite invalid config, got %d", rec.Code)
	}
}

func TestTranslateCSRFHeaders_CustomHeaderName(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{HeaderName: "X-Custom-Csrf"}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Custom-Csrf", "my-token")

	TranslateCSRFHeaders(req, cfg)

	if got := req.Header.Get(DefaultCSRFHeaderName); got != "my-token" {
		t.Fatalf("expected default header to have token, got %q", got)
	}
}

func TestTranslateCSRFHeaders_CustomFieldName(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{FieldName: "custom_field"}

	form := url.Values{}
	form.Set("custom_field", "form-token")

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	TranslateCSRFHeaders(req, cfg)

	if got := req.Header.Get(DefaultCSRFHeaderName); got != "form-token" {
		t.Fatalf("expected default header to have form token, got %q", got)
	}
}

func TestCSRFTokenHXHeaders_WithToken(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithCSRFToken(context.Background(), "abc123"))

	got := CSRFTokenHXHeaders(req)
	if !strings.HasPrefix(got, "hx-headers='") {
		t.Fatalf("expected hx-headers prefix, got %q", got)
	}

	if !strings.Contains(got, "abc123") {
		t.Fatalf("expected token in output, got %q", got)
	}
}

func TestCSRFTokenHTMLMeta_WithToken(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithCSRFToken(context.Background(), "test-token"))

	got := CSRFTokenHTMLMeta(req)
	if !strings.Contains(got, "test-token") {
		t.Fatalf("expected token in meta tag, got %q", got)
	}

	if !strings.Contains(got, `<meta name="csrf-token"`) {
		t.Fatalf("expected meta tag format, got %q", got)
	}
}

func TestCSRFTokenFormField_WithToken(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithCSRFToken(context.Background(), "form-token"))

	got := CSRFTokenFormField(req)
	if !strings.Contains(got, "form-token") {
		t.Fatalf("expected token in form field, got %q", got)
	}

	if !strings.Contains(got, `<input type="hidden"`) {
		t.Fatalf("expected hidden input format, got %q", got)
	}
}

func TestCSRFConfig_CustomValues(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{
		CookieName: "custom_cookie",
		HeaderName: "X-Custom-Hdr",
		FieldName:  "custom_field",
		MaxAge:     2 * time.Hour,
		Path:       "/api",
	}

	if got := cfg.cookieName(); got != "custom_cookie" {
		t.Errorf("cookieName = %q", got)
	}

	if got := cfg.headerName(); got != "X-Custom-Hdr" {
		t.Errorf("headerName = %q", got)
	}

	if got := cfg.fieldName(); got != "custom_field" {
		t.Errorf("fieldName = %q", got)
	}

	if got := cfg.maxAge(); got != 2*time.Hour {
		t.Errorf("maxAge = %v", got)
	}

	if got := cfg.path(); got != "/api" {
		t.Errorf("path = %q", got)
	}
}

func TestCSRFConfig_Validate_UnsafeOriginEmpty(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedOrigins: []string{""}}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty trusted origin")
	}
}

func TestCSRFConfig_Validate_UnsafeOriginWildcard(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedOrigins: []string{"*"}}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for wildcard trusted origin")
	}
}

func TestCSRFConfig_Validate_EmptyProxyEntry(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedProxies: []string{""}}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty trusted proxy entry")
	}
}

func TestCSRFConfig_Validate_InvalidCIDR(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedProxies: []string{"not-a-cidr/999"}}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid CIDR")
	}
}

func TestCSRFConfig_Validate_ValidCIDR(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedProxies: []string{"10.0.0.0/8"}}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error for valid CIDR: %v", err)
	}

	if len(cfg.TrustedProxiesCIDR) != 1 {
		t.Fatalf("expected 1 parsed CIDR, got %d", len(cfg.TrustedProxiesCIDR))
	}
}

func TestCSRFConfig_Validate_SecureFalseLogsWarning(t *testing.T) {
	t.Parallel()

	// Secure=false with no other issues: exercises the slog.Warn path.
	cfg := CSRFConfig{}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error for default config: %v", err)
	}
}

func TestIsTrustedProxy_CIDRMatch(t *testing.T) {
	t.Parallel()

	_, ipnet, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatalf("failed to parse CIDR: %v", err)
	}

	cfg := CSRFConfig{
		TrustedProxies:     []string{"10.0.0.0/8"},
		TrustedProxiesCIDR: []*net.IPNet{ipnet},
	}

	ip := net.ParseIP("10.1.2.3")
	if !isTrustedProxy("10.1.2.3", ip, "10.1.2.3:1234", cfg) {
		t.Fatal("expected 10.1.2.3 to be trusted within 10.0.0.0/8")
	}
}

func TestIsTrustedProxy_ExactMatch(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedProxies: []string{"1.2.3.4"}}

	ip := net.ParseIP("1.2.3.4")
	if !isTrustedProxy("1.2.3.4", ip, "1.2.3.4:5678", cfg) {
		t.Fatal("expected exact IP match to be trusted")
	}
}

func TestIsTrustedProxy_NoMatch(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedProxies: []string{"1.2.3.4"}}

	ip := net.ParseIP("5.6.7.8")
	if isTrustedProxy("5.6.7.8", ip, "5.6.7.8:1234", cfg) {
		t.Fatal("expected non-listed IP to not be trusted")
	}
}

func TestShouldBypassPlaintextOrigin_TLSRequest(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}

	if shouldBypassPlaintextOrigin(req, CSRFConfig{}) {
		t.Fatal("TLS request should not bypass plaintext origin")
	}
}

func TestShouldBypassPlaintextOrigin_LoopbackBypass(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	if !shouldBypassPlaintextOrigin(req, CSRFConfig{}) {
		t.Fatal("loopback plaintext request should bypass")
	}
}

func TestShouldBypassPlaintextOrigin_HasOriginHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.RemoteAddr = "127.0.0.1:1234"

	if shouldBypassPlaintextOrigin(req, CSRFConfig{}) {
		t.Fatal("request with Origin header should not bypass")
	}
}

func TestRemoteHostAndIP_NoPort(t *testing.T) {
	t.Parallel()

	host, ip := remoteHostAndIP("1.2.3.4")
	if host != "1.2.3.4" {
		t.Errorf("host = %q, want %q", host, "1.2.3.4")
	}

	if ip == nil {
		t.Fatal("expected non-nil IP")
	}
}

func TestWarnEmptyTrustedProxies_AllowBypassNoProxies(t *testing.T) {
	t.Parallel()

	// Exercises the warning path. Cannot assert on slog output easily.
	warnEmptyTrustedProxies(CSRFConfig{AllowPlaintextBypass: true})
}

func TestCSRFMiddleware_CustomErrorHandler(t *testing.T) {
	t.Parallel()

	var called bool

	cfg := CSRFConfig{
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			called = true
			w.WriteHeader(http.StatusTeapot)
		},
	}

	mw := CSRFMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("custom error handler was not called")
	}

	if rec.Code != http.StatusTeapot {
		t.Fatalf("expected 418 from custom handler, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_WithDomain(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{Domain: "example.com"}

	mw := CSRFMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	handler.ServeHTTP(rec, req)

	var found bool

	for _, c := range rec.Result().Cookies() {
		if c.Name == DefaultCSRFCookieName {
			found = true

			if c.Domain != "example.com" {
				t.Errorf("cookie Domain = %q, want %q", c.Domain, "example.com")
			}
		}
	}

	if !found {
		t.Fatal("CSRF cookie not set")
	}
}

func TestCSRFMiddleware_WithCustomHeaderName(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{HeaderName: "X-Custom-Csrf"}

	mw := CSRFMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	token, cookie := CSRFTestToken(mw)
	if token == "" {
		t.Fatal("CSRFTestToken returned empty token")
	}

	if cookie == nil {
		t.Fatal("CSRFTestToken returned nil cookie")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Custom-Csrf", token)
	req.AddCookie(cookie)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token in custom header, got %d", rec.Code)
	}
}

func TestForbiddenHandler_WritesStatusOnly(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	ForbiddenHandler(rec, nil, nil)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusForbidden)
	}

	if got := rec.Header().Get("Content-Type"); got != "" {
		t.Errorf("ForbiddenHandler contract is status-only; got unexpected Content-Type %q", got)
	}
}

func TestCSRFMiddleware_RejectionSetsPlainTextContentType(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST without token should be rejected with 403, got %d", rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); got != contentTypePlain {
		t.Errorf("rejection Content-Type = %q, want %q", got, contentTypePlain)
	}

	if rec.Body.Len() == 0 {
		t.Error("rejection body should name the failure reason")
	}
}

// newValidCSRFPost builds a POST request carrying the given middleware's valid
// masked token and session cookie, so a test exercises only the origin
// attestation logic instead of token validation.
func newValidCSRFPost(t *testing.T, mw func(http.Handler) http.Handler) *http.Request {
	t.Helper()

	token, cookie := CSRFTestToken(mw)
	if token == "" {
		t.Fatal("CSRFTestToken returned empty token")
	}

	if cookie == nil {
		t.Fatal("CSRFTestToken returned nil cookie")
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(DefaultCSRFHeaderName, token)
	req.AddCookie(cookie)

	return req
}

func TestCSRFMiddleware_RejectsContradictedSecFetchSiteAttestation(t *testing.T) {
	t.Parallel()

	var captured error

	cfg := CSRFConfig{
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			captured = err
			w.WriteHeader(http.StatusForbidden)
		},
	}

	mw := CSRFMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newValidCSRFPost(t, mw)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "https://evil.example")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf(
			"forged same-origin attestation with cross-origin Origin should be rejected with 403, got %d",
			rec.Code,
		)
	}

	if !errors.Is(captured, ErrCSRFAttestationConflict) {
		t.Fatalf("ErrorHandler error should match ErrCSRFAttestationConflict, got %v", captured)
	}
}

func TestCSRFMiddleware_AllowsConsistentSameOriginAttestation(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newValidCSRFPost(t, mw)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "http://example.com")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("consistent same-origin attestation should be accepted, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_AllowsTrustedOriginWithSameOriginAttestation(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{
		TrustedOrigins: []string{"https://trusted.example"},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newValidCSRFPost(t, mw)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "https://trusted.example")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("same-origin attestation with trusted Origin should be accepted, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_AllowsSecFetchSiteAttestationWithoutOriginHeaders(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newValidCSRFPost(t, mw)
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"client-supplied same-origin attestation with no Origin/Referer should pass nosurf (documented v1.2.0 short-circuit), got %d",
			rec.Code,
		)
	}
}

func TestCSRFMiddleware_SafeMethodIgnoresContradictedAttestation(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "https://evil.example")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("safe methods skip origin validation and attestation checks, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_RejectsCrossSiteAttestationWithCrossOrigin(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newValidCSRFPost(t, mw)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Origin", "https://evil.example")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf(
			"cross-site attestation with disallowed Origin should fail nosurf origin validation with 403, got %d",
			rec.Code,
		)
	}
}

func TestValidateCSRF_RejectsContradictedAttestation(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "https://evil.example")

	valid, rec := ValidateCSRF(req, CSRFConfig{})

	if valid {
		t.Fatal("ValidateCSRF should reject a contradicted same-origin attestation")
	}

	if rec == nil || rec.Code != http.StatusForbidden {
		t.Fatalf("ValidateCSRF rejection should carry a 403 response, got %v", rec)
	}
}

func TestCSRFConfig_Validate_RejectsNegativeMaxAge(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{MaxAge: -time.Hour}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error for negative MaxAge")
	} else if !errors.Is(err, errCSRFMaxAgeNegative) {
		t.Errorf("Validate() error = %v, want errCSRFMaxAgeNegative", err)
	}
}

func TestCSRFConfig_Validate_AcceptsZeroMaxAge(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{MaxAge: 0}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil (zero MaxAge uses the 24h default)", err)
	}
}
