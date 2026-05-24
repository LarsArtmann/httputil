package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithClientIP_StoresIP(t *testing.T) {
	t.Parallel()

	ctx := WithClientIP(context.Background(), "1.2.3.4")

	got := ClientIPFromContext(ctx)
	if got != "1.2.3.4" {
		t.Errorf("ClientIPFromContext() = %q, want %q", got, "1.2.3.4")
	}
}

func TestClientIPFromContext_Empty(t *testing.T) {
	t.Parallel()

	got := ClientIPFromContext(context.Background())
	if got != "" {
		t.Errorf("ClientIPFromContext() = %q, want empty", got)
	}
}

func TestClientIPMiddleware_StoresIPInContext(t *testing.T) {
	t.Parallel()

	var extracted string

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		extracted = ClientIPFromContext(r.Context())
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "9.8.7.6")
	req.RemoteAddr = "10.0.0.1:1234"

	rec := httptest.NewRecorder()

	ClientIPMiddleware(inner).ServeHTTP(rec, req)

	if extracted != "9.8.7.6" {
		t.Errorf("extracted IP = %q, want %q", extracted, "9.8.7.6")
	}
}

func TestResponseRecorder_HeaderSnapshot(t *testing.T) {
	t.Parallel()

	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)

	inner.Header().Set("X-Custom", "value")
	inner.Header().Add("X-Multi", "a")
	inner.Header().Add("X-Multi", "b")

	snapshot := recorder.HeaderSnapshot()

	if got := snapshot.Get("X-Custom"); got != "value" {
		t.Errorf("snapshot[X-Custom] = %q, want %q", got, "value")
	}

	if got := snapshot.Values("X-Multi"); len(got) != 2 {
		t.Errorf("snapshot[X-Multi] = %v, want 2 values", got)
	}

	inner.Header().Set("X-Custom", "changed")

	if snapshot.Get("X-Custom") != "value" {
		t.Error("snapshot should be isolated from later changes")
	}
}
