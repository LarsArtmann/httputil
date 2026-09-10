package httputil

import (
	"net/http"
	"strings"
	"testing"
)

func TestCompose_AppliesMiddlewareInDeclarationOrder(t *testing.T) {
	t.Parallel()

	var order []string

	outer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "outer-before")

			next.ServeHTTP(w, r)

			order = append(order, "outer-after")
		})
	}

	inner := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "inner-before")

			next.ServeHTTP(w, r)

			order = append(order, "inner-after")
		})
	}

	handler := Compose(outer, inner)(newAppendingHandler(&order, "handler"))

	handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/", ""))

	want := []string{"outer-before", "inner-before", "handler", "inner-after", "outer-after"}

	assertSliceEqual(t, order, want)
}

func TestCompose_EmptyListReturnsIdentityMiddleware(t *testing.T) {
	t.Parallel()

	var order []string

	terminal := newAppendingHandler(&order, "terminal")

	handler := Compose()(terminal)

	handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/", ""))

	if len(order) != 1 || order[0] != "terminal" {
		t.Errorf("order = %v, want [terminal]", order)
	}
}

func TestCompose_SingleMiddlewareWrapsHandler(t *testing.T) {
	t.Parallel()

	var order []string

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw-before")

			next.ServeHTTP(w, r)

			order = append(order, "mw-after")
		})
	}

	handler := Compose(mw)(newAppendingHandler(&order, "handler"))

	handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/", ""))

	want := []string{"mw-before", "handler", "mw-after"}

	assertSliceEqual(t, order, want)
}

func TestCompose_ComposedBundlesNestAsSingleMiddleware(t *testing.T) {
	t.Parallel()

	var order []string

	recorder := func(label string, next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, label+"-before")

			next.ServeHTTP(w, r)

			order = append(order, label+"-after")
		})
	}

	bundle := Compose(
		func(next http.Handler) http.Handler { return recorder("bundle-outer", next) },
		func(next http.Handler) http.Handler { return recorder("bundle-inner", next) },
	)

	handler := Compose(
		func(next http.Handler) http.Handler { return recorder("app-outer", next) },
		bundle,
	)(newAppendingHandler(&order, "handler"))

	handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/", ""))

	want := []string{
		"app-outer-before",
		"bundle-outer-before",
		"bundle-inner-before",
		"handler",
		"bundle-inner-after",
		"bundle-outer-after",
		"app-outer-after",
	}

	assertSliceEqual(t, order, want)
}

func TestMiddlewareFunc_Then_WrapsHandler(t *testing.T) {
	t.Parallel()

	var order []string

	var mw MiddlewareFunc = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw-before")

			next.ServeHTTP(w, r)

			order = append(order, "mw-after")
		})
	}

	handler := mw.Then(newAppendingHandler(&order, "handler"))

	handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/", ""))

	want := []string{"mw-before", "handler", "mw-after"}

	assertSliceEqual(t, order, want)
}

func TestMiddlewareFunc_Then_NilHandlerServesInternalServerError(t *testing.T) {
	t.Parallel()

	var mw MiddlewareFunc

	handler := mw.Then(nil)

	rec := newRecorder()
	handler.ServeHTTP(rec, newTestRequest(http.MethodGet, "/", ""))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Then(nil) status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	if body := rec.Body.String(); !strings.Contains(body, "nil handler") {
		t.Errorf("Then(nil) body = %q, want it to name the wiring mistake", body)
	}
}

func TestMiddlewareStack_Middleware_FirstAddedIsOutermost(t *testing.T) {
	t.Parallel()

	var order []string

	recorder := func(label string, next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, label+"-before")

			next.ServeHTTP(w, r)

			order = append(order, label+"-after")
		})
	}

	stack := NewMiddlewareStack()

	err := stack.Add(
		"first",
		func(next http.Handler) http.Handler { return recorder("first", next) },
	)
	if err != nil {
		t.Fatalf("Add first: %v", err)
	}

	err = stack.Add(
		"second",
		func(next http.Handler) http.Handler { return recorder("second", next) },
	)
	if err != nil {
		t.Fatalf("Add second: %v", err)
	}

	handler := stack.Middleware()(newAppendingHandler(&order, "handler"))

	handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/", ""))

	want := []string{
		"first-before",
		"second-before",
		"handler",
		"second-after",
		"first-after",
	}

	assertSliceEqual(t, order, want)
}

func TestMiddlewareStack_Middleware_EmptyStackPassesHandlerThrough(t *testing.T) {
	t.Parallel()

	var order []string

	handler := NewMiddlewareStack().Middleware()(newAppendingHandler(&order, "terminal"))

	handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/", ""))

	if len(order) != 1 || order[0] != "terminal" {
		t.Errorf("order = %v, want [terminal]", order)
	}
}

func TestMiddlewareStack_Middleware_MatchesBuild(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()

	err := stack.Add("logging", Logging(newTestLogger()))
	if err != nil {
		t.Fatalf("Add logging: %v", err)
	}

	err = stack.Add("request-id", RequestID(DefaultRequestIDConfig()))
	if err != nil {
		t.Fatalf("Add request-id: %v", err)
	}

	terminal := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	fromBuild := stack.Build(terminal)
	fromMiddleware := stack.Middleware()(terminal)

	req := newTestRequest(http.MethodGet, "/", "")

	gotBuild := newRecorder()
	fromBuild.ServeHTTP(gotBuild, req)

	gotMiddleware := newRecorder()
	fromMiddleware.ServeHTTP(gotMiddleware, newTestRequest(http.MethodGet, "/", ""))

	if gotMiddleware.Code != gotBuild.Code {
		t.Errorf("Middleware() status = %d, want %d", gotMiddleware.Code, gotBuild.Code)
	}
}

func TestMiddlewareStack_Middleware_IncludesMiddlewareAddedAfterCall(t *testing.T) {
	t.Parallel()

	var order []string

	stack := NewMiddlewareStack()

	err := stack.Add("outer", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "outer-before")

			next.ServeHTTP(w, r)

			order = append(order, "outer-after")
		})
	})
	if err != nil {
		t.Fatalf("Add outer: %v", err)
	}

	mw := stack.Middleware()

	err = stack.Add("inner", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "inner-before")

			next.ServeHTTP(w, r)

			order = append(order, "inner-after")
		})
	})
	if err != nil {
		t.Fatalf("Add inner: %v", err)
	}

	handler := mw(newAppendingHandler(&order, "handler"))

	handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/", ""))

	want := []string{"outer-before", "inner-before", "handler", "inner-after", "outer-after"}

	assertSliceEqual(t, order, want)
}
