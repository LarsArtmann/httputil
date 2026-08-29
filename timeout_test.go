package httputil

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestTimeout_DeadlineExceededObservableInHandler(t *testing.T) {
	t.Parallel()

	mw := Timeout(30 * time.Millisecond)

	deadlineHit := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()

		select {
		case <-deadlineHit:
		default:
			close(deadlineHit)
		}

		if r.Context().Err() == nil {
			t.Error("context should be Done after cancellation")

			return
		}

		if !errors.Is(r.Context().Err(), context.DeadlineExceeded) {
			t.Errorf("context error = %v, want %v", r.Context().Err(), context.DeadlineExceeded)
		}
	})

	mw(handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	select {
	case <-deadlineHit:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never observed the deadline")
	}
}

func TestTimeout_DeadlineBeforeHandlerStarts(t *testing.T) {
	t.Parallel()

	mw := Timeout(1 * time.Nanosecond)

	var ctxErr error
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		ctxErr = r.Context().Err()
	})

	mw(handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !errors.Is(ctxErr, context.DeadlineExceeded) {
		t.Errorf("context error = %v, want %v", ctxErr, context.DeadlineExceeded)
	}
}
