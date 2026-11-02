package httputil

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

	seen := make(map[string]bool, len(nonces))

	for _, nonce := range nonces {
		if seen[nonce] {
			t.Errorf(
				"duplicate nonce %q across %d requests, want unique nonces",
				nonce,
				len(nonces),
			)
		}

		seen[nonce] = true
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
		t.Errorf(
			"nonce length = %d, want %d (for %d bytes)",
			len(ctxNonce),
			expectedLen,
			defaultNonceSize,
		)
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
		t.Errorf(
			"nonce length = %d, want %d (for %d bytes)",
			len(ctxNonce),
			expectedLen,
			customSize,
		)
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

func TestNonceConfig_Validate_AcceptsZeroSize(t *testing.T) {
	t.Parallel()

	cfg := NonceConfig{
		Size:       0,
		CSPBuilder: RecommendedCSPWithNonce,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for Size = 0 (use default)", err)
	}
}

func TestNonce_MinSizeMiddlewarePath(t *testing.T) {
	t.Parallel()

	cfg := NonceConfig{
		Size:       minNonceSize,
		CSPBuilder: nil,
	}

	var ctxNonce string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxNonce = NonceFromRequest(r)
	})

	handler := Nonce(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	expectedLen := base64.RawURLEncoding.EncodedLen(minNonceSize)
	if len(ctxNonce) != expectedLen {
		t.Errorf(
			"nonce length = %d, want %d (for %d bytes)",
			len(ctxNonce),
			expectedLen,
			minNonceSize,
		)
	}
}

func TestNonce_BeforeSecurityHeaders_LosesCSP(t *testing.T) {
	t.Parallel()

	secCfg := DefaultSecurityHeadersConfig()
	secCfg.ContentSecurityPolicy = "default-src 'none'"

	// Wrong ordering: Nonce is outer (first in Chain), SecurityHeaders is
	// inner. SecurityHeaders overwrites the nonce-bearing CSP. This test
	// documents the footgun -- always place Nonce after SecurityHeaders.
	handler := Chain(
		newNoOpHandler(),
		Nonce(DefaultNonceConfig()),
		SecurityHeaders(secCfg),
	)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if strings.Contains(csp, "'nonce-") {
		t.Errorf("CSP = %q, should not contain nonce when Nonce is outer (wrong ordering)", csp)
	}
}

func TestNonce_DoesNotLeakBetweenRequests(t *testing.T) {
	t.Parallel()

	cfg := DefaultNonceConfig()

	var firstNonce string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nonce := NonceFromRequest(r)
		if firstNonce == "" {
			firstNonce = nonce
		} else if nonce == firstNonce {
			t.Errorf("nonce leaked: request got same nonce as first request (%q)", nonce)
		}
	})

	handler := Nonce(cfg)(inner)

	for range nonceUniqueTestRequests {
		req := newTestRequest(http.MethodGet, "/", "")
		rec := newRecorder()

		handler.ServeHTTP(rec, req)
	}
}

func TestNonce_CSPBuilderReturnsEmpty(t *testing.T) {
	t.Parallel()

	cfg := NonceConfig{
		Size: defaultNonceSize,
		CSPBuilder: func(_ string) string {
			return ""
		},
	}

	handler := Nonce(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp != "" {
		t.Errorf("CSP = %q, want empty when CSPBuilder returns empty string", csp)
	}
}

func TestNonce_LargeSize(t *testing.T) {
	t.Parallel()

	largeSize := 1024

	cfg := NonceConfig{
		Size:       largeSize,
		CSPBuilder: nil,
	}

	var ctxNonce string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxNonce = NonceFromRequest(r)
	})

	handler := Nonce(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	expectedLen := base64.RawURLEncoding.EncodedLen(largeSize)
	if len(ctxNonce) != expectedLen {
		t.Errorf("nonce length = %d, want %d (for %d bytes)", len(ctxNonce), expectedLen, largeSize)
	}
}

func TestNonce_WithRecoveryComposition(t *testing.T) {
	t.Parallel()

	panickingHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	})

	// Recovery is outermost, Nonce is inner. CSP header should be set
	// even when the handler panics, because Nonce middleware runs before
	// dispatching to the handler.
	handler := Chain(
		panickingHandler,
		Recovery(slog.New(slog.DiscardHandler)),
		Nonce(DefaultNonceConfig()),
	)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "'nonce-") {
		t.Errorf("CSP = %q, want nonce-bearing CSP even on panic-recovery 500", csp)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestNonce_InvalidConfigLogsAndFallsBack(t *testing.T) {
	t.Parallel()

	// Size 8 is below minNonceSize (16) and non-zero, so Validate() returns
	// errNonceTooSmall. The constructor logs via slog.Error and continues with
	// the default size — it must NOT panic or produce a broken middleware.
	cfg := NonceConfig{
		Size:       minNonceSize / 2,
		CSPBuilder: nil,
	}

	var ctxNonce string

	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctxNonce = NonceFromRequest(r)
	})

	handler := Nonce(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	// Invalid Size falls back to defaultNonceSize.
	expectedLen := base64.RawURLEncoding.EncodedLen(defaultNonceSize)
	if len(ctxNonce) != expectedLen {
		t.Errorf(
			"nonce length = %d, want %d (invalid Size should fall back to default)",
			len(ctxNonce),
			expectedLen,
		)
	}
}

func BenchmarkNonceAttr(b *testing.B) {
	nonce := generateNonce(defaultNonceSize)
	req := newTestRequest(http.MethodGet, "/", "")
	req = req.WithContext(WithNonce(req.Context(), nonce))

	var attr string

	for b.Loop() {
		attr = NonceAttr(req)
	}

	_ = attr
}

// captureNonceRequestLogs runs fn with slog's default logger swapped for a
// JSON handler writing to a buffer, and returns every decoded log record in
// order (the csrf captureCSRFConstructorLogs pattern, request-path edition).
func captureNonceRequestLogs(t *testing.T, fn func()) []map[string]any {
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

		var record map[string]any

		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatalf("decode log record %q: %v", line, err)
		}

		records = append(records, record)
	}

	return records
}

// TestNonce_CSPConflict_OuterCSPOverwrittenWithWarning pins the
// security.csp_conflict prototype: SecurityHeaders (with an explicit CSP)
// wrapping Nonce (with a CSPBuilder) must keep the documented overwrite
// behavior — the nonce-bearing CSP wins — AND warn loudly that a static CSP
// was clobbered, so the double-writer misconfiguration stops being silent.
// No t.Parallel: the slog default swap races constructor tests that log.
//
//nolint:paralleltest // swaps the global default logger; cannot run in parallel
func TestNonce_CSPConflict_OuterCSPOverwrittenWithWarning(t *testing.T) {
	outerCSP := "default-src 'self'; script-src 'self'"

	staticHeaders := SecurityHeadersConfig{ContentSecurityPolicy: outerCSP}
	nonceCfg := DefaultNonceConfig()

	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		SecurityHeaders(staticHeaders),
		Nonce(nonceCfg),
	)

	var records []map[string]any

	rec := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	records = captureNonceRequestLogs(t, func() {
		handler.ServeHTTP(rec, request)
	})

	served := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(served, "nonce-") {
		t.Fatalf(
			"served CSP %q does not contain a nonce directive, want the nonce-bearing CSP to win unchanged",
			served,
		)
	}

	if served == outerCSP {
		t.Fatalf("served CSP %q equals the outer static CSP; the overwrite behavior changed", served)
	}

	found := false

	for _, record := range records {
		if record["code"] == string(codeSecurityCSPConflict) {
			found = true

			if record["outer_csp"] != outerCSP {
				t.Errorf("warning outer_csp = %v, want %q", record["outer_csp"], outerCSP)
			}
		}
	}

	if !found {
		t.Fatalf("no %s warning logged; records: %v", codeSecurityCSPConflict, records)
	}
}

// TestNonce_NoOuterCSP_NoConflictWarning: a lone Nonce (the normal case)
// must not emit the conflict warning — the detector fires only when a CSP
// header is already present.
//
//nolint:paralleltest // swaps the global default logger; cannot run in parallel
func TestNonce_NoOuterCSP_NoConflictWarning(t *testing.T) {
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		Nonce(DefaultNonceConfig()),
	)

	var records []map[string]any

	rec := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	records = captureNonceRequestLogs(t, func() {
		handler.ServeHTTP(rec, request)
	})

	for _, record := range records {
		if record["code"] == string(codeSecurityCSPConflict) {
			t.Fatalf("conflict warning logged without an outer CSP: %v", record)
		}
	}
}
