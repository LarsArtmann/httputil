package httputil

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSecurityHeaders_DefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultSecurityHeadersConfig()
	handler := SecurityHeaders(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	tests := []struct{ header, want string }{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		if got := rec.Header().Get(tt.header); got != tt.want {
			t.Errorf("%s = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestSecurityHeaders_CustomCSP(t *testing.T) {
	t.Parallel()

	cfg := SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'",
	}
	handler := SecurityHeaders(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, "Content-Security-Policy", "default-src 'self'")
}

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

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "existing-id-123")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, "X-Request-ID", "existing-id-123")

	if ctxID != "existing-id-123" {
		t.Errorf("context ID = %q, want %q", ctxID, "existing-id-123")
	}
}

func TestRecovery_CatchesPanic(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestRecovery_PassesThroughNormal(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	called := false

	handler := Recovery(logger)(newCountingHandler(&called))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called")
	}
}

func TestTimeout_SetsDeadline(t *testing.T) {
	t.Parallel()

	handler := Timeout(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Error("context has no deadline")
		}

		if deadline.IsZero() {
			t.Error("deadline is zero")
		}
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)
}

func TestLogging_RecordsRequest(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := Logging(logger)(inner)

	req := newTestRequest(http.MethodGet, "/test", "")
	req.RemoteAddr = "10.0.0.1:1234"

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if buf.Len() == 0 {
		t.Error("expected log output")
	}
}
