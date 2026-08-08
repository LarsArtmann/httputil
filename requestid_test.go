package httputil

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID_GeneratesID(t *testing.T) {
	t.Parallel()

	cfg := DefaultRequestIDConfig()
	handler := RequestID(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	got := rec.Header().Get("X-Request-ID")
	if got == "" {
		t.Error("X-Request-ID header is empty, want generated ID")
	}
}

func TestRequestID_ForwardsExistingID(t *testing.T) {
	t.Parallel()

	cfg := DefaultRequestIDConfig()

	var ctxID string

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = RequestIDFromContext(r.Context())
	})

	handler := RequestID(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("X-Request-ID", "existing-id-123")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, "X-Request-ID", "existing-id-123")

	if ctxID != "existing-id-123" {
		t.Errorf("context ID = %q, want %q", ctxID, "existing-id-123")
	}
}

func FuzzRequestID(f *testing.F) {
	f.Add("existing-id-123")
	f.Add("")
	f.Add("a")
	f.Add("x-request-id-with-dashes-and-numbers-42")

	f.Fuzz(func(t *testing.T, headerValue string) {
		cfg := DefaultRequestIDConfig()
		handler := RequestID(cfg)(newNoOpHandler())

		req := newTestRequest(http.MethodGet, "/", "")
		req.Header.Set(cfg.IncomingHeader, headerValue)

		rec := newRecorder()
		handler.ServeHTTP(rec, req)

		got := rec.Header().Get(cfg.ResponseHeader)
		if got == "" {
			t.Error("request ID header is empty")
		}
	})
}

func TestRequestIDConfig_Validate_ValidDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultRequestIDConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestRequestIDConfig_Validate_NilGenerateID(t *testing.T) {
	t.Parallel()

	cfg := RequestIDConfig{
		ResponseHeader: "X-Request-ID",
		IncomingHeader: "X-Request-ID",
		GenerateID:     nil,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for nil GenerateID")
	}

	if !errors.Is(err, errNilGenerateID) {
		t.Errorf("Validate() error = %v, want errNilGenerateID", err)
	}
}

func TestRequestIDConfig_Validate_EmptyResponseHeader(t *testing.T) {
	t.Parallel()

	cfg := newRequestIDConfigForTest("", "X-Request-ID")

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for empty ResponseHeader")
	}

	if !errors.Is(err, errEmptyResponseHeader) {
		t.Errorf("Validate() error = %v, want errEmptyResponseHeader", err)
	}
}

func TestRequestIDConfig_Validate_EmptyIncomingHeader(t *testing.T) {
	t.Parallel()

	cfg := newRequestIDConfigForTest("X-Request-ID", "")

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for empty IncomingHeader")
	}

	if !errors.Is(err, errEmptyIncomingHeader) {
		t.Errorf("Validate() error = %v, want errEmptyIncomingHeader", err)
	}
}

// TestRequestID_InvalidConfigLogsAndContinues verifies that an invalid
// RequestID config (empty ResponseHeader with a valid GenerateID) is logged
// via slog but does not prevent the middleware from serving requests.
func TestRequestID_InvalidConfigLogsAndContinues(t *testing.T) {
	t.Parallel()

	// GenerateID is set (avoids nil-call panic at request time), but
	// ResponseHeader is empty so Validate returns errEmptyResponseHeader.
	cfg := RequestIDConfig{
		GenerateID: func() string { return "test-id" },
	}

	var called bool

	handler := RequestID(cfg)(newCountingHandler(&called))
	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called (invalid config should log and continue)")
	}
}

func BenchmarkRequestID(b *testing.B) {
	cfg := DefaultRequestIDConfig()
	middleware := RequestID(cfg)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}
