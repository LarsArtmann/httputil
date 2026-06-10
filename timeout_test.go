package httputil

import (
	"net/http"
	"testing"
	"time"
)

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

func BenchmarkTimeout(b *testing.B) {
	middleware := Timeout(time.Second)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}
