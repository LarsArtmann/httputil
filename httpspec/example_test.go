package httpspec

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

func ExampleExpectStatus() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	check := ExpectStatus(http.MethodGet, "/health", http.StatusOK)
	result := check(mux)
	fmt.Println(result)

	// Output: passed
}

func ExampleExpectNotStatus() {
	handler := newStatusOnlyHandler(http.StatusOK)

	check := ExpectNotStatus(http.MethodGet, "/", http.StatusInternalServerError)
	result := check(handler)
	fmt.Println(result)

	// Output: passed
}

func ExampleExpectHeader() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	check := ExpectHeader(http.MethodGet, "/", "Content-Type", "application/json")
	result := check(handler)
	fmt.Println(result)

	// Output: passed
}

func ExampleExpectHeaderAbsent() {
	handler := newStatusOnlyHandler(http.StatusOK)

	check := ExpectHeaderAbsent(http.MethodGet, "/", "X-Powered-By")
	result := check(handler)
	fmt.Println(result)

	// Output: passed
}

func ExampleExpectBodyContains() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	})

	check := ExpectBodyContains(http.MethodGet, "/", "hello")
	result := check(handler)
	fmt.Println(result)

	// Output: passed
}

func ExamplePass() {
	fmt.Println(Pass())

	// Output: passed
}

func ExampleFail() {
	result := Fail("something went wrong")
	fmt.Println(result)

	// Output: something went wrong
}

// ExampleRun shows the building block Run executes for every specification.
// Run itself spawns subtests and needs a *testing.T, so it is called from a
// test function, not from an example:
//
//	func TestAPIHandler(t *testing.T) {
//		httpspec.Run(t, apiHandler, httpspec.WithExtraSpecs(httpspec.CORSSpecs()...))
//	}
//
// Run validates the standard HTTP conventions (routing, methods, headers,
// security) against the handler in parallel subtests, and [WithExtraSpecs]
// composes the optional families (CORS, rate limits, private network
// preflights) into the same run.
func ExampleRun() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "hello")
	})

	spec := Spec{
		Name:     "index greets the client",
		Category: CategoryHeaders,
		Check: func(h http.Handler) Result {
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if !strings.Contains(rec.Body.String(), "hello") {
				return Fail("body %q does not contain hello", rec.Body.String())
			}

			return Pass()
		},
	}

	fmt.Println(spec.Check(handler))

	// Output: passed
}
