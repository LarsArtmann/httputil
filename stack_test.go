package httputil

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareStackAddAndBuild(t *testing.T) {
	t.Parallel()

	var order []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1-before")

			next.ServeHTTP(w, r)

			order = append(order, "mw1-after")
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2-before")

			next.ServeHTTP(w, r)

			order = append(order, "mw2-after")
		})
	}

	stack := NewMiddlewareStack()

	err := stack.Add("mw1", mw1)
	if err != nil {
		t.Fatalf("Add mw1: %v", err)
	}

	err = stack.Add("mw2", mw2)
	if err != nil {
		t.Fatalf("Add mw2: %v", err)
	}

	handler := stack.Build(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")

		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	want := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}

	for i, w := range want {
		if i >= len(order) {
			t.Errorf("order[%d]: missing, want %q", i, w)

			continue
		}

		if order[i] != w {
			t.Errorf("order[%d] = %q, want %q", i, order[i], w)
		}
	}
}

func TestMiddlewareStackRejectsDuplicateNames(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()

	err := stack.Add("cors", CORS(DefaultCORSConfig()))
	if err != nil {
		t.Fatalf("first Add: %v", err)
	}

	err = stack.Add("cors", CORS(DefaultCORSConfig()))
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}

	if !errors.Is(err, errDuplicateMiddleware) {
		t.Errorf("error = %v, want errDuplicateMiddleware", err)
	}
}

func TestMiddlewareStackValidateEmptyPasses(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()

	err := stack.Validate()
	if err != nil {
		t.Errorf("empty stack should validate: %v", err)
	}
}

func TestMiddlewareStackValidateRecoveryFirstPasses(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()

	_ = stack.Add(MiddlewareRecovery, Recovery(slog.Default()))
	_ = stack.Add(MiddlewareCORS, CORS(DefaultCORSConfig()))
	_ = stack.Add(MiddlewareCompression, Compression(DefaultCompressionConfig()))

	err := stack.Validate()
	if err != nil {
		t.Errorf("valid stack should validate: %v", err)
	}
}

func TestMiddlewareStackValidateRecoveryNotFirstFails(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()

	_ = stack.Add(MiddlewareCORS, CORS(DefaultCORSConfig()))
	_ = stack.Add(MiddlewareRecovery, Recovery(slog.Default()))

	err := stack.Validate()
	if err == nil {
		t.Fatal("expected error when recovery is not first, got nil")
	}

	if !errors.Is(err, errRecoveryNotFirst) {
		t.Errorf("error = %v, want errRecoveryNotFirst", err)
	}
}

func TestMiddlewareStackNames(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()

	_ = stack.Add(MiddlewareRecovery, Recovery(slog.Default()))
	_ = stack.Add(MiddlewareCORS, CORS(DefaultCORSConfig()))

	names := stack.Names()

	if len(names) != 2 {
		t.Fatalf("Names() returned %d entries, want 2", len(names))
	}

	if names[0] != MiddlewareRecovery {
		t.Errorf("names[0] = %q, want %q", names[0], MiddlewareRecovery)
	}

	if names[1] != MiddlewareCORS {
		t.Errorf("names[1] = %q, want %q", names[1], MiddlewareCORS)
	}
}
