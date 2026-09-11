package httputil

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
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

func TestMiddlewareStack_Add_RejectsEmptyName(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()

	err := stack.Add("", func(next http.Handler) http.Handler { return next })
	if err == nil {
		t.Fatal("Add with empty name error = nil, want error")
	}

	if !errors.Is(err, errEmptyMiddlewareName) {
		t.Errorf("error = %v, want errEmptyMiddlewareName", err)
	}

	if got := stack.Names(); len(got) != 0 {
		t.Errorf("Names() = %v, want empty (rejected entry must not be added)", got)
	}
}

func TestMiddlewareStack_ConcurrentAddAndRead_RaceFree(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()

	const writers = 8

	const reads = 8

	const iterations = 200

	var wg sync.WaitGroup

	for w := range writers {
		wg.Go(func() {
			for iteration := range iterations {
				err := stack.Add(
					fmt.Sprintf("middleware-%d-%d", w, iteration),
					func(next http.Handler) http.Handler { return next },
				)
				if err != nil {
					t.Errorf("Add: %v", err)

					return
				}
			}
		})
	}

	for range reads {
		wg.Go(func() {
			for range iterations {
				_ = stack.Names()
				_ = stack.Validate()
				_ = stack.Build(newNoOpHandler())
			}
		})
	}

	wg.Wait()

	if got := len(stack.Names()); got != writers*iterations {
		t.Errorf("len(Names()) = %d, want %d (no Add may be lost)", got, writers*iterations)
	}
}
