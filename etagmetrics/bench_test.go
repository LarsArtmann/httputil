package etagmetrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	etag "github.com/larsartmann/go-etag/server"

	"github.com/larsartmann/httputil/etagmetrics"
)

func BenchmarkETagMetricsHookOverhead(b *testing.B) {
	cfg, _ := etagmetrics.Attach(etag.DefaultETagConfig())
	handler := etag.New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("benchmark payload"))
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkETagPlainNoHooks(b *testing.B) {
	handler := etag.New(
		etag.DefaultETagConfig(),
	)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("benchmark payload"))
		}),
	)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
