package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkHealthHandler(b *testing.B) {
	handler := HealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkLiveHandler(b *testing.B) {
	handler := LiveHandler()

	req := httptest.NewRequest(http.MethodGet, "/live", nil)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkReadyHandler(b *testing.B) {
	handler := ReadyHandler()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkMetricsMiddleware(b *testing.B) {
	recorder := &benchMetricsRecorder{}
	cfg := DefaultMetricsConfig()
	cfg.Recorder = recorder

	handler := Metrics(cfg)(newStatusOnlyHandler(http.StatusOK))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkMetricsMiddlewareWithBody(b *testing.B) {
	recorder := &benchMetricsRecorder{}
	cfg := DefaultMetricsConfig()
	cfg.Recorder = recorder

	handler := Metrics(cfg)(newWriteBodyHandler([]byte("hello world")))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkMetricsMiddlewareWithCustomPath(b *testing.B) {
	recorder := &benchMetricsRecorder{}
	cfg := DefaultMetricsConfig()
	cfg.Recorder = recorder
	cfg.PathFunc = func(_ *http.Request) string {
		return "/api/test"
	}

	handler := Metrics(cfg)(newStatusOnlyHandler(http.StatusOK))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

type benchMetricsRecorder struct{}

func (r *benchMetricsRecorder) Record(_, _ string, _ int, _ time.Duration) {}
