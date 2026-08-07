package httputil

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestChain_RecoveryLoggingCORS(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	corsCfg := DefaultCORSConfig()

	inner := newWriteStatusHandler(http.StatusOK, "ok")

	handler := Chain(inner, CORS(corsCfg), Recovery(logger), Logging(logger))

	req := newTestRequest(http.MethodGet, "/", "http://example.com")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	assertHeader(t, rec, "Access-Control-Allow-Origin", "*")
}

func TestChain_RecoveryCatchesPanicWithLogging(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	corsCfg := DefaultCORSConfig()

	inner := newPanicHandler("integration test panic")

	handler := Chain(inner, CORS(corsCfg), Recovery(logger), Logging(logger))

	req := newTestRequest(http.MethodGet, "/", "http://example.com")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusInternalServerError)
}

func TestChain_RequestIDSecurityHeaders(t *testing.T) {
	t.Parallel()

	reqCfg := DefaultRequestIDConfig()
	secCfg := DefaultSecurityHeadersConfig()

	inner := newWriteStatusHandler(http.StatusOK, "")

	handler := Chain(inner, SecurityHeaders(secCfg), RequestID(reqCfg))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header is empty")
	}

	assertHeader(t, rec, "X-Content-Type-Options", "nosniff")
}

func TestChain_TimeoutThenRecovery(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Deadline()
		if !ok {
			t.Error("context has no deadline")
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := Chain(inner, Recovery(logger), Timeout(time.Second))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
}

func BenchmarkChain(b *testing.B) {
	logger := newTestLogger()
	corsCfg := DefaultCORSConfig()
	reqCfg := DefaultRequestIDConfig()
	secCfg := DefaultSecurityHeadersConfig()

	inner := newWriteStatusHandler(http.StatusOK, "")

	handler := Chain(
		inner,
		SecurityHeaders(secCfg),
		RequestID(reqCfg),
		Recovery(logger),
		Logging(logger),
		CORS(corsCfg),
	)

	req := newTestRequest(http.MethodGet, "/", "http://example.com")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}
