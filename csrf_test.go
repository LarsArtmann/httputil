package httputil

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json/v2"
	"errors"
	"log/slog"
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

func TestInvalidateCSRFCookie_NoneWithoutSecure_FallsBackToSecure(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	InvalidateCSRFCookie(rec, CSRFConfig{SameSite: http.SameSiteNoneMode, Secure: false})

	var found bool

	for _, c := range rec.Result().Cookies() {
		if c.Name == DefaultCSRFCookieName {
			found = true

			if !c.Secure {
				t.Errorf("c.Secure = false, want true (deletion cookie must match the fallback cookie)")
			}

			if c.SameSite != http.SameSiteNoneMode {
				t.Errorf("c.SameSite = %v, want %v", c.SameSite, http.SameSiteNoneMode)
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

func TestCSRFMiddleware_NoneWithoutSecure_FallsBackToSecureCookie(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{SameSite: http.SameSiteNoneMode, Secure: false})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	handler.ServeHTTP(rec, req)

	cookie := csrfTokenCookie(t, rec)

	if !cookie.Secure {
		t.Errorf("cookie.Secure = false, want true (rfc6265bis fallback)")
	}

	if cookie.SameSite != http.SameSiteNoneMode {
		t.Errorf("cookie.SameSite = %v, want %v", cookie.SameSite, http.SameSiteNoneMode)
	}
}

func TestCSRFMiddleware_NoneWithoutSecure_OptOutKeepsVerbatim(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{
		SameSite:                  http.SameSiteNoneMode,
		AllowInsecureSameSiteNone: true,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	handler.ServeHTTP(rec, req)

	cookie := csrfTokenCookie(t, rec)

	if cookie.Secure {
		t.Errorf("cookie.Secure = true, want false (opt-out keeps the config verbatim)")
	}

	if cookie.SameSite != http.SameSiteNoneMode {
		t.Errorf("cookie.SameSite = %v, want %v", cookie.SameSite, http.SameSiteNoneMode)
	}
}

func TestCSRFMiddleware_NoneWithSecure_Unchanged(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{SameSite: http.SameSiteNoneMode, Secure: true})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	handler.ServeHTTP(rec, req)

	cookie := csrfTokenCookie(t, rec)

	if !cookie.Secure {
		t.Errorf("cookie.Secure = false, want true")
	}

	if cookie.SameSite != http.SameSiteNoneMode {
		t.Errorf("cookie.SameSite = %v, want %v", cookie.SameSite, http.SameSiteNoneMode)
	}
}

func TestCSRFMiddleware_LaxWithoutSecure_StaysUnchanged(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{SameSite: http.SameSiteLaxMode, Secure: false})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	handler.ServeHTTP(rec, req)

	cookie := csrfTokenCookie(t, rec)

	if cookie.Secure {
		t.Errorf("cookie.Secure = true, want false (fallback applies only to SameSite=None)")
	}

	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("cookie.SameSite = %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
	}
}

// csrfTokenCookie returns the CSRF cookie set on rec, failing the test when
// absent. The middleware sets it on the first request via nosurf's token
// regeneration.
func csrfTokenCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	for _, c := range rec.Result().Cookies() {
		if c.Name == DefaultCSRFCookieName {
			return c
		}
	}

	t.Fatalf("no %s cookie in Set-Cookie headers", DefaultCSRFCookieName)

	return nil
}

// captureCSRFConstructorLogs runs fn with slog's default logger swapped for a
// JSON handler writing to a buffer, and returns every decoded log record in
// order. Use for construction-path log assertions that need more than the
// first record.
func captureCSRFConstructorLogs(t *testing.T, fn func()) []map[string]any {
	t.Helper()

	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	fn()

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	records := make([]map[string]any, 0, len(lines))

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var decoded map[string]any

		err := json.Unmarshal(line, &decoded)
		if err != nil {
			t.Fatalf("decoding log line %q: %v", line, err)
		}

		records = append(records, decoded)
	}

	return records
}

//nolint:paralleltest // swaps the global default logger; cannot run in parallel
func TestCSRFMiddleware_NoneWithoutSecure_FallbackLogsRemediation(t *testing.T) {
	records := captureCSRFConstructorLogs(t, func() {
		CSRFMiddleware(CSRFConfig{SameSite: http.SameSiteNoneMode, Secure: false})
	})

	if len(records) < 2 {
		t.Fatalf("records = %d, want at least 2 (Validate rejection + remediation fallback)", len(records))
	}

	if records[0]["level"] != "ERROR" || records[0]["code"] != "csrf_samesite_insecure" {
		t.Errorf("validateConfig record = %v/%v, want ERROR/csrf_samesite_insecure", records[0]["level"], records[0]["code"])
	}

	if records[1]["level"] != "WARN" {
		t.Errorf("fallback level = %v, want WARN", records[1]["level"])
	}

	msg, _ := records[1]["msg"].(string)
	if !strings.Contains(msg, "fell back to Secure=true") {
		t.Errorf("fallback msg = %q, want it to name the Secure=true fallback", msg)
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

func TestCSRFConfig_Validate_TrustedOriginNotParseable(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedOrigins: []string{"https://app.example.com/%zz"}}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for a TrustedOrigins entry url.Parse rejects")
	}

	if !errors.Is(err, errCSRFInvalidOrigin) {
		t.Errorf("Validate() error = %v, want errCSRFInvalidOrigin", err)
	}

	if !errors.Is(err, ErrCSRFConfig) {
		t.Errorf("Validate() error = %v, want ErrCSRFConfig in the cause chain", err)
	}

	code, ok := CodeOf(err)
	if !ok || code != codeCSRFInvalidOrigin {
		t.Errorf("CodeOf(err) = %q, %v; want %q", code, ok, codeCSRFInvalidOrigin)
	}
}

func TestCSRFConfig_Validate_TrustedOriginMissingScheme(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedOrigins: []string{"app.example.com"}}
	err := cfg.Validate()
	if !errors.Is(err, errCSRFInvalidOrigin) {
		t.Errorf("Validate() error = %v, want errCSRFInvalidOrigin for a scheme-less origin", err)
	}
}

func TestCSRFConfig_Validate_TrustedOriginMissingHost(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedOrigins: []string{"https://"}}
	err := cfg.Validate()
	if !errors.Is(err, errCSRFInvalidOrigin) {
		t.Errorf("Validate() error = %v, want errCSRFInvalidOrigin for a host-less origin", err)
	}
}

func TestCSRFConfig_Validate_TrustedOriginWellFormedAccepted(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedOrigins: []string{
		"https://app.example.com",
		"http://other.example.net:8443/path",
	}}

	if err := cfg.Validate(); err != nil {
		t.Fatalf(
			"Validate() error = %v, want nil for scheme://host origins (extra path parts are ignored)",
			err,
		)
	}
}

func TestParseTrustedOriginURLs_AllOrNothing(t *testing.T) {
	t.Parallel()

	broken := parseTrustedOriginURLs([]string{"https://good.example", "https://bad.example/%zz"})
	if broken != nil {
		t.Errorf(
			"parseTrustedOriginURLs with any unparseable entry = %v, want nil (all-or-nothing, mirroring nosurf.StaticOrigins)",
			broken,
		)
	}

	good := parseTrustedOriginURLs([]string{"https://good.example"})
	if len(good) != 1 {
		t.Errorf("parseTrustedOriginURLs with only valid entries length = %d, want 1", len(good))
	}
}

func TestCSRFMiddleware_UnparseableTrustedOriginFailsClosed(t *testing.T) {
	t.Parallel()

	var captured error

	cfg := CSRFConfig{
		TrustedOrigins: []string{"https://trusted.example", "https://broken.example/%zz"},
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
	req.Header.Set("Origin", "https://trusted.example")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf(
			"request should be rejected when TrustedOrigins contain an unparseable entry, got %d",
			rec.Code,
		)
	}

	if !errors.Is(captured, ErrCSRFAttestationConflict) {
		t.Errorf(
			"ErrorHandler error = %v, want ErrCSRFAttestationConflict: with an unparseable entry no origin may be trusted, not even a well-formed sibling",
			captured,
		)
	}
}

func TestCSRFMiddleware_AllowsTrustedOriginWithoutAttestationHeader(t *testing.T) {
	t.Parallel()

	mw := CSRFMiddleware(CSRFConfig{
		TrustedOrigins: []string{"https://trusted.example"},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newValidCSRFPost(t, mw)
	req.Header.Set("Origin", "https://trusted.example")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"cross-origin request with valid token and a listed Origin should be accepted without a Sec-Fetch-Site attestation, got %d",
			rec.Code,
		)
	}
}

func TestCSRFMiddleware_UnparseableTrustedOriginFallsBackToSameOriginOnly(t *testing.T) {
	t.Parallel()

	var captured error

	cfg := CSRFConfig{
		TrustedOrigins: []string{"https://trusted.example", "https://broken.example/%zz"},
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
	req.Header.Set("Origin", "https://trusted.example")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf(
			"cross-origin request with valid token should be rejected once any TrustedOrigins entry is unparseable (same-origin-only fallback), got %d",
			rec.Code,
		)
	}

	if !errors.Is(captured, ErrCSRFInvalid) {
		t.Errorf(
			"ErrorHandler error = %v, want ErrCSRFInvalid: the nosurf token path, not the attestation check, must reject the now-untrusted origin",
			captured,
		)
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

	// Validate is pure: parsing happens in withParsedTrustedProxies, so the
	// receiver must be untouched.
	if len(cfg.TrustedProxiesCIDR) != 0 {
		t.Fatalf(
			"Validate must not mutate the config, got %d parsed CIDRs",
			len(cfg.TrustedProxiesCIDR),
		)
	}
}

func TestCSRFConfig_WithParsedTrustedProxies_PopulatesCIDR(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedProxies: []string{"10.0.0.0/8", "192.168.0.1"}}

	parsed := cfg.withParsedTrustedProxies()

	if len(parsed.TrustedProxiesCIDR) != 1 {
		t.Fatalf("expected 1 parsed CIDR (bare IP ignored), got %d", len(parsed.TrustedProxiesCIDR))
	}

	if _, subnet, _ := net.ParseCIDR(
		"10.0.0.0/8",
	); !parsed.TrustedProxiesCIDR[0].IP.Equal(
		subnet.IP,
	) {
		t.Errorf("parsed CIDR = %v, want the 10.0.0.0/8 network", parsed.TrustedProxiesCIDR[0])
	}
}

func TestCSRFConfig_WithParsedTrustedProxies_InvalidCIDRYieldsEmpty(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{TrustedProxies: []string{"10.0.0.0/8", "10.0.0.0/99"}}

	parsed := cfg.withParsedTrustedProxies()

	if len(parsed.TrustedProxiesCIDR) != 0 {
		t.Fatalf(
			"invalid entry must yield an empty CIDR list, got %d",
			len(parsed.TrustedProxiesCIDR),
		)
	}
}

func TestCSRFConfig_Validate_SecureFalseIsNotAnError(t *testing.T) {
	t.Parallel()

	// Secure=false alone is a constructor warning, not a validation error.
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

func TestCSRFMiddleware_AllowsCaseDifferingHostWithSameOriginAttestation(t *testing.T) {
	t.Parallel()

	// DNS hosts are case-insensitive: Origin http://EXAMPLE.COM against
	// Host example.com is the same origin, not a forged attestation.
	mw := CSRFMiddleware(CSRFConfig{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newValidCSRFPost(t, mw)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "http://EXAMPLE.COM")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"case-differing same host should not be treated as a forged attestation, got %d",
			rec.Code,
		)
	}
}

func TestCSRFMiddleware_TrustedProxyForwardedProtoAllowsHTTPSOrigin(t *testing.T) {
	t.Parallel()

	// TLS termination: the browser sends Origin https://example.com, the
	// trusted proxy forwards plaintext. X-Forwarded-Proto from the trusted
	// proxy restores the client-facing scheme, so the truthful attestation
	// is consistent instead of "forged".
	mw := CSRFMiddleware(CSRFConfig{TrustedProxies: []string{"10.0.0.1"}})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newValidCSRFPost(t, mw)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("X-Forwarded-Proto", "https")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"https Origin behind a trusted TLS-terminating proxy should be accepted, got %d",
			rec.Code,
		)
	}
}

func TestCSRFMiddleware_UntrustedForwardedProtoStillRejected(t *testing.T) {
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

	// X-Forwarded-Proto from a NON-trusted address is attacker-controlled
	// and must not rescue a scheme-contradicted attestation.
	req := newValidCSRFPost(t, mw)
	req.RemoteAddr = "1.2.3.4:1234"
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("X-Forwarded-Proto", "https")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf(
			"X-Forwarded-Proto from an untrusted address must not excuse the contradiction, got %d",
			rec.Code,
		)
	}

	if !errors.Is(captured, ErrCSRFAttestationConflict) {
		t.Errorf("ErrorHandler error = %v, want ErrCSRFAttestationConflict", captured)
	}
}

func TestCSRFMiddleware_TrustedProxyDoesNotExcuseDifferentHost(t *testing.T) {
	t.Parallel()

	var captured error

	cfg := CSRFConfig{
		TrustedProxies: []string{"10.0.0.1"},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			captured = err
			w.WriteHeader(http.StatusForbidden)
		},
	}

	mw := CSRFMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// A trusted proxy widens only the scheme view; a different host is
	// still a forged attestation.
	req := newValidCSRFPost(t, mw)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("X-Forwarded-Proto", "https")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("different host behind a trusted proxy must still be rejected, got %d", rec.Code)
	}

	if !errors.Is(captured, ErrCSRFAttestationConflict) {
		t.Errorf("ErrorHandler error = %v, want ErrCSRFAttestationConflict", captured)
	}
}
