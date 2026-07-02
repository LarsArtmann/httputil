package httpspec

import (
	"fmt"
	"net/http"
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
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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
