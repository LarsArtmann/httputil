package endpoints

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

func ExampleRegister() {
	routes := NewRegistry()
	mux := http.NewServeMux()

	err := routes.HandleFunc(mux, "GET /users/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, "Fetch one user")
	if err != nil {
		// Wiring errors come back as values (never panics): a nil mux, an
		// empty pattern, or a pattern ServeMux rejects.
		fmt.Println("registration error:", err)

		return
	}

	err = Register(mux, routes) // mounts "GET /endpoints"
	if err != nil {
		fmt.Println("registration error:", err)

		return
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/endpoints", nil))
	fmt.Println(rec.Body.String())

	// Output:
	// {"endpoints":[{"method":"GET","pattern":"GET /endpoints","summary":"Endpoint listing (this page)"},{"method":"GET","pattern":"GET /users/{id}","summary":"Fetch one user"}]}
}

func ExampleRegistry_Handler() {
	// A browser's Accept header selects the HTML table: text/html wins over
	// the */* fallback, so the same URL serves humans and machines.
	routes := NewRegistry()
	mux := http.NewServeMux()

	if err := routes.HandleFunc(
		mux,
		"GET /users/{id}",
		func(w http.ResponseWriter, _ *http.Request) {},
		"Fetch one user",
	); err != nil {
		fmt.Println("registration error:", err)

		return
	}

	if err := Register(mux, routes); err != nil {
		fmt.Println("registration error:", err)

		return
	}

	browserAccept := "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
	req := httptest.NewRequest(http.MethodGet, "/endpoints", nil)
	req.Header.Set("Accept", browserAccept)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	fmt.Println(strings.Contains(rec.Body.String(), "<td>GET /users/{id}</td>"))

	// Output:
	// true
}

func ExampleRegistry_Routes() {
	routes := NewRegistry()

	if err := routes.Handle(
		http.NewServeMux(),
		"GET /users/{id}",
		http.NotFoundHandler(),
		"Fetch one user",
	); err != nil {
		fmt.Println("registration error:", err)

		return
	}

	if err := routes.Handle(http.NewServeMux(), "example.com/{$}", http.NotFoundHandler(), "Host root"); err != nil {
		fmt.Println("registration error:", err)

		return
	}

	for _, route := range routes.Routes() {
		fmt.Printf("%q %q %q\n", route.Method, route.Pattern, route.Summary)
	}

	// Output:
	// "GET" "GET /users/{id}" "Fetch one user"
	// "" "example.com/{$}" "Host root"
}
