package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
