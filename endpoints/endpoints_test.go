package endpoints

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestHandle_RegistersHandlerOnMux(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registry := NewRegistry()

	err := registry.HandleFunc(mux, "GET /hello", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}, "Says hello")

	if err != nil {
		t.Fatalf("HandleFunc returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Body.String(); got != "hello" {
		t.Errorf("body = %q, want %q", got, "hello")
	}
}

func TestHandle_RecordsRouteWithParsedMethod(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	err := registry.Handle(http.NewServeMux(), "GET /users/{id}", http.NotFoundHandler(), "Fetch one user")

	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	routes := registry.Routes()

	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1", len(routes))
	}

	want := Route{Method: "GET", Pattern: "GET /users/{id}", Summary: "Fetch one user"}

	if routes[0] != want {
		t.Errorf("routes[0] = %+v, want %+v", routes[0], want)
	}
}

func TestHandle_PatternWithoutMethod_RecordsEmptyMethod(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	err := registry.Handle(http.NewServeMux(), "/health", http.NotFoundHandler(), "")
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	routes := registry.Routes()

	if len(routes) != 1 || routes[0].Method != "" {
		t.Errorf("routes = %+v, want one route with empty Method", routes)
	}
}

func TestHandle_HostPattern_RecordsEmptyMethod(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	err := registry.Handle(http.NewServeMux(), "example.com/{$}", http.NotFoundHandler(), "")
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	routes := registry.Routes()

	if len(routes) != 1 || routes[0].Method != "" {
		t.Errorf("routes = %+v, want one route with empty Method", routes)
	}
}

func TestHandle_MethodAndHostPattern_RecordsMethod(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	err := registry.Handle(http.NewServeMux(), "GET example.com/x", http.NotFoundHandler(), "")
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	routes := registry.Routes()

	if len(routes) != 1 || routes[0].Method != "GET" {
		t.Errorf("routes = %+v, want one route with Method GET", routes)
	}
}

func TestHandle_TabSeparatedMethod_RecordsMethod(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	err := registry.Handle(http.NewServeMux(), "GET\t/users", http.NotFoundHandler(), "")
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	routes := registry.Routes()

	if len(routes) != 1 || routes[0].Method != "GET" {
		t.Errorf("routes = %+v, want one route with Method GET (tab separator)", routes)
	}
}

func TestMethodOf_FirstFieldWithSlash_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	// Unreachable through valid ServeMux patterns (a path cannot contain an
	// unescaped space), but methodOf is a pure helper whose contract keeps
	// the guard: a first field containing '/' is host/path, never a method.
	if got := methodOf("example.com/x y"); got != "" {
		t.Errorf("methodOf(%q) = %q, want %q", "example.com/x y", got, "")
	}
}

func TestMethodOf_NoWhitespace_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	if got := methodOf("/plain/path"); got != "" {
		t.Errorf("methodOf(%q) = %q, want %q", "/plain/path", got, "")
	}
}

func TestHandle_NilMux_ReturnsErrMuxNil(t *testing.T) {
	t.Parallel()

	err := NewRegistry().Handle(nil, "GET /x", http.NotFoundHandler(), "")

	if !errors.Is(err, errMuxNil) {
		t.Errorf("err = %v, want errMuxNil", err)
	}
}

func TestHandle_NilHandler_ReturnsErrHandlerNil(t *testing.T) {
	t.Parallel()

	err := NewRegistry().Handle(http.NewServeMux(), "GET /x", nil, "")

	if !errors.Is(err, errHandlerNil) {
		t.Errorf("err = %v, want errHandlerNil", err)
	}
}

func TestHandle_EmptyPattern_ReturnsErrPatternEmpty(t *testing.T) {
	t.Parallel()

	err := NewRegistry().Handle(http.NewServeMux(), "", http.NotFoundHandler(), "")

	if !errors.Is(err, errPatternEmpty) {
		t.Errorf("err = %v, want errPatternEmpty", err)
	}
}

func TestHandle_InvalidUTF8Pattern_ReturnsErrorAndRecordsNothing(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	err := registry.Handle(http.NewServeMux(), "GET /\xff{x}", http.NotFoundHandler(), "")

	if !errors.Is(err, errPatternInvalidUTF8) {
		t.Errorf("err = %v, want errPatternInvalidUTF8", err)
	}

	if routes := registry.Routes(); len(routes) != 0 {
		t.Errorf("routes = %+v, want none recorded", routes)
	}
}

func TestHandle_InvalidUTF8Summary_ReturnsErrorAndMountsNothing(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mux := http.NewServeMux()

	err := registry.Handle(mux, "GET /x", http.NotFoundHandler(), "bad\xffsummary")

	if !errors.Is(err, errSummaryInvalidUTF8) {
		t.Errorf("err = %v, want errSummaryInvalidUTF8", err)
	}

	if routes := registry.Routes(); len(routes) != 0 {
		t.Errorf("routes = %+v, want none recorded", routes)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (pattern must not be mounted)", rec.Code, http.StatusNotFound)
	}
}

func TestHandle_RejectedPattern_ReturnsErrorNotPanic(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	// "GET" alone has no path; ServeMux panics on this at registration.
	err := registry.Handle(http.NewServeMux(), "GET", http.NotFoundHandler(), "")

	var rejected *muxRejectedError

	if !errors.As(err, &rejected) {
		t.Fatalf("err = %v, want *muxRejectedError", err)
	}

	if !strings.Contains(err.Error(), `"GET"`) {
		t.Errorf("err = %v, want the rejected pattern quoted", err)
	}
}

func TestHandle_DuplicatePattern_ReturnsErrorAndRecordsOnce(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mux := http.NewServeMux()

	if err := registry.Handle(mux, "GET /dup", http.NotFoundHandler(), ""); err != nil {
		t.Fatalf("first Handle returned error: %v", err)
	}

	err := registry.Handle(mux, "GET /dup", http.NotFoundHandler(), "")
	if err == nil {
		t.Fatal("second Handle returned nil error, want duplicate rejection")
	}

	if routes := registry.Routes(); len(routes) != 1 {
		t.Errorf("len(routes) = %d, want 1 (duplicate must not be recorded)", len(routes))
	}
}

func TestHandleFunc_NilHandler_ReturnsErrHandlerNil(t *testing.T) {
	t.Parallel()

	err := NewRegistry().HandleFunc(http.NewServeMux(), "GET /x", nil, "")

	if !errors.Is(err, errHandlerNil) {
		t.Errorf("err = %v, want errHandlerNil", err)
	}
}

func TestRoutes_ReturnsSortedSnapshot(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mux := http.NewServeMux()

	for _, pattern := range []string{"/zebra", "/alpha", "GET /mid"} {
		if err := registry.Handle(mux, pattern, http.NotFoundHandler(), ""); err != nil {
			t.Fatalf("Handle(%q) returned error: %v", pattern, err)
		}
	}

	// Sorted by the full pattern string: '/' (0x2f) sorts before 'G'
	// (0x47), so the method-less patterns come first.
	got := registry.Routes()
	want := []string{"/alpha", "/zebra", "GET /mid"}

	if len(got) != len(want) {
		t.Fatalf("len(routes) = %d, want %d", len(got), len(want))
	}

	for i, pattern := range want {
		if got[i].Pattern != pattern {
			t.Errorf("routes[%d].Pattern = %q, want %q", i, got[i].Pattern, pattern)
		}
	}
}

func TestRoutes_ReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	if err := registry.Handle(http.NewServeMux(), "/x", http.NotFoundHandler(), ""); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	first := registry.Routes()
	first[0] = Route{Method: "CORRUPTED", Pattern: "CORRUPTED", Summary: "CORRUPTED"}

	second := registry.Routes()

	if second[0].Pattern == "CORRUPTED" {
		t.Error("mutating the returned slice changed the registry state")
	}
}

func TestRoutes_ZeroValueRegistry_ReturnsNil(t *testing.T) {
	t.Parallel()

	var registry Registry

	if routes := registry.Routes(); routes != nil {
		t.Errorf("routes = %+v, want nil", routes)
	}
}

func TestRegistry_ConcurrentHandleAndRoutes_AreRaceFree(t *testing.T) {
	t.Parallel()

	const writers = 8
	const perWriter = 25

	registry := NewRegistry()
	mux := http.NewServeMux()

	var wg sync.WaitGroup

	for writer := range writers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := range perWriter {
				pattern := fmt.Sprintf("GET /w%d-%d", writer, i)

				err := registry.Handle(mux, pattern, http.NotFoundHandler(), "")
				if err != nil {
					t.Errorf("Handle(%q) returned error: %v", pattern, err)

					return
				}
			}
		}()
	}

	for range writers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range perWriter {
				_ = registry.Routes()
			}
		}()
	}

	wg.Wait()

	if got := len(registry.Routes()); got != writers*perWriter {
		t.Errorf("len(Routes()) = %d, want %d", got, writers*perWriter)
	}
}

func TestRegister_MountsListingThatIncludesItself(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mux := http.NewServeMux()

	if err := registry.HandleFunc(mux, "GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, "Health"); err != nil {
		t.Fatalf("HandleFunc returned error: %v", err)
	}

	if err := Register(mux, registry); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/endpoints", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if !strings.Contains(rec.Body.String(), `"pattern":"GET /endpoints"`) {
		t.Errorf("listing body does not include itself:\n%s", rec.Body.String())
	}
}

func TestHandler_MethodNotAllowed_IncludesAllowHeaderAndNoBody(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()

	NewRegistry().Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/endpoints", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}

	if allow := rec.Header().Get("Allow"); allow != allowGetHead {
		t.Errorf("Allow = %q, want %q", allow, allowGetHead)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestHandler_DefaultsToJSON(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()

	NewRegistry().Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/endpoints", nil))

	if ct := rec.Header().Get("Content-Type"); ct != contentTypeJSON {
		t.Errorf("Content-Type = %q, want %q", ct, contentTypeJSON)
	}
}

func TestHandler_BrowserAccept_SelectsHTML(t *testing.T) {
	t.Parallel()

	browserAccept := "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"
	req := httptest.NewRequest(http.MethodGet, "/endpoints", nil)
	req.Header.Set("Accept", browserAccept)

	rec := httptest.NewRecorder()

	NewRegistry().Handler().ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != contentTypeHTML {
		t.Errorf("Content-Type = %q, want %q", ct, contentTypeHTML)
	}

	if !strings.HasPrefix(rec.Body.String(), "<!DOCTYPE html>") {
		t.Errorf("body does not start with the HTML doctype:\n%s", rec.Body.String())
	}
}

func TestHandler_ExplicitJSONAccept_SelectsJSON(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/endpoints", nil)
	req.Header.Set("Accept", "application/json")

	rec := httptest.NewRecorder()

	NewRegistry().Handler().ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != contentTypeJSON {
		t.Errorf("Content-Type = %q, want %q", ct, contentTypeJSON)
	}
}

func TestHandler_RespondsNoStoreAndVaryAcceptInBothRepresentations(t *testing.T) {
	t.Parallel()

	for _, accept := range []string{"application/json", "text/html"} {
		req := httptest.NewRequest(http.MethodGet, "/endpoints", nil)
		req.Header.Set("Accept", accept)

		rec := httptest.NewRecorder()

		NewRegistry().Handler().ServeHTTP(rec, req)

		if cc := rec.Header().Get("Cache-Control"); cc != cacheControlNoStore {
			t.Errorf("Accept %q: Cache-Control = %q, want %q", accept, cc, cacheControlNoStore)
		}

		if vary := rec.Header().Get("Vary"); vary != headerAccept {
			t.Errorf("Accept %q: Vary = %q, want %q", accept, vary, headerAccept)
		}
	}
}

func TestHandler_ZeroValueRegistry_ListsEmptyJSON(t *testing.T) {
	t.Parallel()

	var registry Registry

	rec := httptest.NewRecorder()

	registry.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/endpoints", nil))

	if got, want := rec.Body.String(), "{\"endpoints\":[]}\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestHandler_HeadRequest_ServesHeadersWithoutBody(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mux := http.NewServeMux()

	if err := Register(mux, registry); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodHead,
		server.URL+"/endpoints",
		nil,
	)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}

	if len(body) != 0 {
		t.Errorf("HEAD body = %q, want empty", body)
	}
}
