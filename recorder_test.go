package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseRecorder_DefaultStatus(t *testing.T) {
	t.Parallel()

	recorder := NewResponseRecorder(httptest.NewRecorder())
	if recorder.Status() != 0 {
		t.Errorf("Status() = %d before WriteHeader, want 0", recorder.Status())
	}
}

func TestResponseRecorder_WriteHeader(t *testing.T) {
	t.Parallel()

	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)
	recorder.WriteHeader(http.StatusNotFound)

	if recorder.Status() != http.StatusNotFound {
		t.Errorf("Status() = %d, want %d", recorder.Status(), http.StatusNotFound)
	}

	if inner.Code != http.StatusNotFound {
		t.Errorf("inner.Code = %d, want %d", inner.Code, http.StatusNotFound)
	}
}

func TestResponseRecorder_WriteSetsStatusOK(t *testing.T) {
	t.Parallel()

	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)

	_, err := recorder.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}

	if recorder.Status() != http.StatusOK {
		t.Errorf("Status() = %d after Write, want %d", recorder.Status(), http.StatusOK)
	}
}

func TestResponseRecorder_WriteHeaderOnlyOnce(t *testing.T) {
	t.Parallel()

	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)
	recorder.WriteHeader(http.StatusCreated)
	recorder.WriteHeader(http.StatusInternalServerError)

	if recorder.Status() != http.StatusCreated {
		t.Errorf(
			"Status() = %d, want %d (first WriteHeader wins)",
			recorder.Status(),
			http.StatusCreated,
		)
	}
}

func TestChain(t *testing.T) {
	t.Parallel()

	var order []string

	middleware := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)

				next.ServeHTTP(w, r)
			})
		}
	}

	handler := newAppendingHandler(&order, "handler")

	Chain(
		handler,
		middleware("a"),
		middleware("b"),
		middleware("c"),
	).ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

	want := []string{"a", "b", "c", "handler"}
	assertSliceEqual(t, order, want)
}

func TestChain_ZeroMiddleware(t *testing.T) {
	t.Parallel()

	called := false
	handler := newCountingHandler(&called)

	Chain(
		handler,
	).ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

	if !called {
		t.Error("handler should be called with zero middleware")
	}
}

func TestChain_SingleMiddleware(t *testing.T) {
	t.Parallel()

	var order []string

	wrapper := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw")

			next.ServeHTTP(w, r)
		})
	}

	handler := newAppendingHandler(&order, "handler")

	Chain(
		handler,
		wrapper,
	).ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

	want := []string{"mw", "handler"}
	assertSliceEqual(t, order, want)
}

func TestResponseRecorder_WriteAfterWriteHeader(t *testing.T) {
	t.Parallel()

	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)

	recorder.WriteHeader(http.StatusBadRequest)

	_, err := recorder.Write([]byte("error"))
	if err != nil {
		t.Errorf("Write() error = %v, want nil", err)
	}

	if recorder.Status() != http.StatusBadRequest {
		t.Errorf(
			"Status() = %d, want %d (WriteHeader wins)",
			recorder.Status(),
			http.StatusBadRequest,
		)
	}
}

func BenchmarkResponseRecorder(b *testing.B) {
	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)
	body := []byte("hello world benchmark test data")

	for b.Loop() {
		_, _ = recorder.Write(body)
	}
}

func TestResponseRecorder_WroteHeader(t *testing.T) {
	t.Parallel()

	recorder := NewResponseRecorder(httptest.NewRecorder())

	if recorder.WroteHeader() {
		t.Error("WroteHeader() = true before WriteHeader, want false")
	}

	recorder.WriteHeader(http.StatusOK)

	if !recorder.WroteHeader() {
		t.Error("WroteHeader() = false after WriteHeader, want true")
	}
}

func TestResponseRecorder_Flush_NoFlusher(t *testing.T) {
	t.Parallel()

	recorder := NewResponseRecorder(httptest.NewRecorder())
	recorder.Flush()
}

func TestResponseRecorder_Hijack_Unsupported(t *testing.T) {
	t.Parallel()

	recorder := NewResponseRecorder(httptest.NewRecorder())

	conn, rw, err := recorder.Hijack()
	if conn != nil || rw != nil {
		t.Error("expected nil conn and rw for non-hijacker")
	}

	if err == nil {
		t.Fatal("expected error for non-hijacker")
	}
}
