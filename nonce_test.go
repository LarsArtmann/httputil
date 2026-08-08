package httputil

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"
)

const nonceUniqueTestRequests = 10

func TestNonce_GeneratesAndStoresInContext(t *testing.T) {
	t.Parallel()

	cfg := DefaultNonceConfig()

	var ctxNonce string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxNonce = NonceFromContext(r.Context())
	})

	handler := Nonce(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if ctxNonce == "" {
		t.Fatal("NonceFromContext returned empty, want a generated nonce")
	}
}

func TestNonce_SetsDefaultCSPHeader(t *testing.T) {
	t.Parallel()

	cfg := DefaultNonceConfig()

	var ctxNonce string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxNonce = NonceFromContext(r.Context())
	})

	handler := Nonce(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header is empty, want a nonce-bearing policy")
	}

	expectedNonce := "'nonce-" + ctxNonce + "'"
	if !strings.Contains(csp, expectedNonce) {
		t.Errorf("CSP = %q, want it to contain %q", csp, expectedNonce)
	}
}

func TestNonce_NilCSPBuilderSkipsHeader(t *testing.T) {
	t.Parallel()

	cfg := NonceConfig{
		Size:       defaultNonceSize,
		CSPBuilder: nil,
	}

	handler := Nonce(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if csp := rec.Header().Get("Content-Security-Policy"); csp != "" {
		t.Errorf("Content-Security-Policy = %q, want empty when CSPBuilder is nil", csp)
	}
}

func TestNonce_UniquePerRequest(t *testing.T) {
	t.Parallel()

	cfg := DefaultNonceConfig()

	var nonces []string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nonces = append(nonces, NonceFromContext(r.Context()))
	})

	handler := Nonce(cfg)(inner)

	for range nonceUniqueTestRequests {
		req := newTestRequest(http.MethodGet, "/", "")
		rec := newRecorder()

		handler.ServeHTTP(rec, req)
	}

	for i := 1; i < len(nonces); i++ {
		if nonces[i] == nonces[0] {
			t.Errorf("nonce[%d] == nonce[0] (%q), want unique nonces", i, nonces[0])
		}
	}
}

func TestNonce_DefaultSizeWhenZero(t *testing.T) {
	t.Parallel()

	cfg := NonceConfig{
		Size:       0,
		CSPBuilder: nil,
	}

	var ctxNonce string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxNonce = NonceFromContext(r.Context())
	})

	handler := Nonce(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	expectedLen := base64.RawURLEncoding.EncodedLen(defaultNonceSize)
	if len(ctxNonce) != expectedLen {
		t.Errorf("nonce length = %d, want %d (for %d bytes)", len(ctxNonce), expectedLen, defaultNonceSize)
	}
}

func TestNonce_CustomSize(t *testing.T) {
	t.Parallel()

	customSize := 32

	cfg := NonceConfig{
		Size:       customSize,
		CSPBuilder: nil,
	}

	var ctxNonce string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxNonce = NonceFromContext(r.Context())
	})

	handler := Nonce(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	expectedLen := base64.RawURLEncoding.EncodedLen(customSize)
	if len(ctxNonce) != expectedLen {
		t.Errorf("nonce length = %d, want %d (for %d bytes)", len(ctxNonce), expectedLen, customSize)
	}
}

func TestNonce_CustomCSPBuilder(t *testing.T) {
	t.Parallel()

	customCSP := "default-src 'none'; script-src 'nonce-test'"

	cfg := NonceConfig{
		Size: defaultNonceSize,
		CSPBuilder: func(_ string) string {
			return customCSP
		},
	}

	handler := Nonce(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, "Content-Security-Policy", customCSP)
}

func TestNonce_NonceIsBase64Encoded(t *testing.T) {
	t.Parallel()

	cfg := DefaultNonceConfig()

	var ctxNonce string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxNonce = NonceFromContext(r.Context())
	})

	handler := Nonce(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	decoded, err := base64.RawURLEncoding.DecodeString(ctxNonce)
	if err != nil {
		t.Fatalf("nonce %q is not valid base64: %v", ctxNonce, err)
	}

	if len(decoded) != defaultNonceSize {
		t.Errorf("decoded nonce length = %d, want %d", len(decoded), defaultNonceSize)
	}
}

func TestNonce_NonceFromRequest(t *testing.T) {
	t.Parallel()

	cfg := DefaultNonceConfig()

	var requestNonce string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		requestNonce = NonceFromRequest(r)
	})

	handler := Nonce(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if requestNonce == "" {
		t.Fatal("NonceFromRequest returned empty, want a generated nonce")
	}

	expectedLen := base64.RawURLEncoding.EncodedLen(defaultNonceSize)
	if len(requestNonce) != expectedLen {
		t.Errorf("NonceFromRequest length = %d, want %d", len(requestNonce), expectedLen)
	}
}

func TestNonceFromContext_EmptyWhenMissing(t *testing.T) {
	t.Parallel()

	nonce := NonceFromContext(context.Background())
	if nonce != "" {
		t.Errorf("NonceFromContext() = %q, want empty for context without nonce", nonce)
	}
}

func TestNonceFromRequest_EmptyWhenMissing(t *testing.T) {
	t.Parallel()

	req := newTestRequest(http.MethodGet, "/", "")

	nonce := NonceFromRequest(req)
	if nonce != "" {
		t.Errorf("NonceFromRequest() = %q, want empty for request without nonce", nonce)
	}
}

func TestRecommendedCSPWithNonce(t *testing.T) {
	t.Parallel()

	nonce := "abc123test"

	csp := RecommendedCSPWithNonce(nonce)

	if !strings.Contains(csp, "'nonce-"+nonce+"'") {
		t.Errorf("CSP = %q, want it to contain 'nonce-%s'", csp, nonce)
	}

	if !strings.Contains(csp, "script-src") {
		t.Errorf("CSP = %q, want it to contain script-src", csp)
	}

	if !strings.Contains(csp, "style-src") {
		t.Errorf("CSP = %q, want it to contain style-src", csp)
	}
}

func TestNonceConfig_Validate_ValidDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultNonceConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestNonceConfig_Validate_RejectsTooSmall(t *testing.T) {
	t.Parallel()

	cfg := NonceConfig{
		Size:       minNonceSize - 1,
		CSPBuilder: RecommendedCSPWithNonce,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for Size < 16")
	}

	if !errors.Is(err, errNonceTooSmall) {
		t.Errorf("Validate() error = %v, want errNonceTooSmall", err)
	}
}

func TestNonceConfig_Validate_AcceptsMinimumSize(t *testing.T) {
	t.Parallel()

	cfg := NonceConfig{
		Size:       minNonceSize,
		CSPBuilder: RecommendedCSPWithNonce,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for Size = 16", err)
	}
}

func BenchmarkNonce(b *testing.B) {
	cfg := DefaultNonceConfig()
	middleware := Nonce(cfg)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func TestProductionCSPWithNonce(t *testing.T) {
	t.Parallel()

	nonce := "abc123test"

	csp := ProductionCSPWithNonce(nonce)

	if !strings.Contains(csp, "'nonce-"+nonce+"'") {
		t.Errorf("CSP = %q, want it to contain 'nonce-%s'", csp, nonce)
	}

	for _, expected := range []string{
		"object-src 'none'",
		"base-uri 'self'",
		"frame-ancestors 'none'",
	} {
		if !strings.Contains(csp, expected) {
			t.Errorf("CSP = %q, want it to contain %q", csp, expected)
		}
	}
}

func TestNonce_ProductionCSPBuilder(t *testing.T) {
	t.Parallel()

	cfg := NonceConfig{
		Size:       defaultNonceSize,
		CSPBuilder: ProductionCSPWithNonce,
	}

	var ctxNonce string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxNonce = NonceFromRequest(r)
	})

	handler := Nonce(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")

	if !strings.Contains(csp, "object-src 'none'") {
		t.Errorf("CSP = %q, want production directives", csp)
	}

	if !strings.Contains(csp, "'nonce-"+ctxNonce+"'") {
		t.Errorf("CSP = %q, want nonce %q", csp, ctxNonce)
	}
}

func TestNonceAttr(t *testing.T) {
	t.Parallel()

	cfg := DefaultNonceConfig()

	var attr, nonce string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nonce = NonceFromRequest(r)
		attr = NonceAttr(r)
	})

	handler := Nonce(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	expected := `nonce="` + nonce + `"`
	if attr != expected {
		t.Errorf("NonceAttr = %q, want %q", attr, expected)
	}
}

func TestNonceAttr_EmptyWhenMissing(t *testing.T) {
	t.Parallel()

	req := newTestRequest(http.MethodGet, "/", "")

	attr := NonceAttr(req)
	if attr != "" {
		t.Errorf("NonceAttr() = %q, want empty for request without nonce", attr)
	}
}

func TestNonce_OverwritesStaticCSP(t *testing.T) {
	t.Parallel()

	secCfg := DefaultSecurityHeadersConfig()
	secCfg.ContentSecurityPolicy = "default-src 'none'"

	// SecurityHeaders is outermost, Nonce is inner. The inner middleware
	// sets Content-Security-Policy after the outer one, so the nonce-bearing
	// CSP wins. This is the correct ordering.
	handler := Chain(
		newNoOpHandler(),
		SecurityHeaders(secCfg),
		Nonce(DefaultNonceConfig()),
	)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "'nonce-") {
		t.Errorf(
			"CSP = %q, want nonce-bearing policy (Nonce middleware must overwrite SecurityHeaders' static CSP)",
			csp,
		)
	}
}

func BenchmarkGenerateNonce(b *testing.B) {
	for b.Loop() {
		_ = generateNonce(defaultNonceSize)
	}
}
