package httpspec

import "net/http"

// newStatusOnlyHandler returns an http.HandlerFunc that writes only the given
// status code, without a body. Used by check tests that assert on status
// without caring about response content.
func newStatusOnlyHandler(status int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	})
}

// newBareServerNameHandler returns an http.HandlerFunc that sets the Server
// header to name and writes StatusNotFound. Used to verify that "nginx" (bare
// name) does not trigger a version-leak spec failure.
func newBareServerNameHandler(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Server", name)
		w.WriteHeader(http.StatusNotFound)
	})
}

// newHeaderNotFoundHandler returns an http.HandlerFunc that sets a single
// header (key to value) and writes StatusNotFound. Used to exercise the
// server-version and powered-by header-leak specs.
func newHeaderNotFoundHandler(key, value string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(key, value)
		w.WriteHeader(http.StatusNotFound)
	})
}

// newTypedMux returns an http.ServeMux that serves a single response on path.
// The response carries the standard security + content-type headers and writes
// body with StatusOK. Used to construct "good" handlers for spec and benchmark
// tests where the same shape must be reused with different paths or bodies.
func newTypedMux(path, contentType, body string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})

	return mux
}

// newTypedHelloMux returns an http.ServeMux whose "/" handler responds with a
// text/plain content type, an X-Content-Type-Options header, and the literal
// "hello" body. It is the canonical "good handler" used to verify that the
// full httpspec suite passes.
func newTypedHelloMux() http.Handler {
	return newTypedMux("/{$}", "text/plain; charset=utf-8", "hello")
}

// newTypedBodyHandler returns an http.HandlerFunc that sets Content-Type and
// writes body with StatusOK. Used to construct simple content-typed handlers
// for tests that don't need a full mux (e.g. SPA fallback, no-sniff checks).
func newTypedBodyHandler(contentType, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})
}
