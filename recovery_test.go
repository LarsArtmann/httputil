package httputil

import (
	"net/http"
	"testing"
)

func TestRecovery_CatchesPanic(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	handler := Recovery(logger)(newPanicHandler("test panic"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusInternalServerError)
}

func TestRecovery_PassesThroughNormal(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	called := false

	handler := Recovery(logger)(newCountingHandler(&called))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called")
	}
}

func BenchmarkRecovery(b *testing.B) {
	logger := newTestLogger()
	middleware := Recovery(logger)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}
