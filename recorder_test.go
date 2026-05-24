package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseRecorder_DefaultStatus(t *testing.T) {
	t.Parallel()
	rr := NewResponseRecorder(httptest.NewRecorder())
	if rr.Status() != 0 {
		t.Errorf("Status() = %d before WriteHeader, want 0", rr.Status())
	}
}

func TestResponseRecorder_WriteHeader(t *testing.T) {
	t.Parallel()
	inner := httptest.NewRecorder()
	rr := NewResponseRecorder(inner)
	rr.WriteHeader(http.StatusNotFound)

	if rr.Status() != http.StatusNotFound {
		t.Errorf("Status() = %d, want %d", rr.Status(), http.StatusNotFound)
	}
	if inner.Code != http.StatusNotFound {
		t.Errorf("inner.Code = %d, want %d", inner.Code, http.StatusNotFound)
	}
}

func TestResponseRecorder_WriteSetsStatusOK(t *testing.T) {
	t.Parallel()
	inner := httptest.NewRecorder()
	rr := NewResponseRecorder(inner)
	rr.Write([]byte("hello"))

	if rr.Status() != http.StatusOK {
		t.Errorf("Status() = %d after Write, want %d", rr.Status(), http.StatusOK)
	}
}

func TestResponseRecorder_WriteHeaderOnlyOnce(t *testing.T) {
	t.Parallel()
	inner := httptest.NewRecorder()
	rr := NewResponseRecorder(inner)
	rr.WriteHeader(http.StatusCreated)
	rr.WriteHeader(http.StatusInternalServerError)

	if rr.Status() != http.StatusCreated {
		t.Errorf("Status() = %d, want %d (first WriteHeader wins)", rr.Status(), http.StatusCreated)
	}
}

func TestChain(t *testing.T) {
	t.Parallel()
	var order []string
	mw := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	Chain(handler, mw("a"), mw("b"), mw("c")).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"a", "b", "c", "handler"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i, v := range want {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}
