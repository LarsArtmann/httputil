package httputil

import "net/http"

// Compose combines middleware into a single middleware. The first middleware
// in the list becomes the outermost wrapper when the composed middleware is
// applied to a handler, matching [Chain]'s ordering contract. Composing an
// empty list yields the identity middleware, which returns the handler
// unchanged.
func Compose(middlewares ...Middleware) Middleware {
	return func(handler http.Handler) http.Handler {
		return Chain(handler, middlewares...)
	}
}

// MiddlewareFunc adapts an ordinary function to the [Middleware] signature so
// it can carry composition helpers. [Middleware] is an alias for a plain
// function type and therefore cannot declare methods; define reusable
// middleware functions as MiddlewareFunc to chain them fluently.
type MiddlewareFunc func(http.Handler) http.Handler

// Then applies the middleware to next, returning the wrapped handler. A nil
// next wires a fallback handler that serves 500 for every request, so wiring
// mistakes surface as a diagnosable response instead of a panic — the
// library never panics by design.
func (mw MiddlewareFunc) Then(next http.Handler) http.Handler {
	if next == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(
				w,
				"httputil: nil handler wired into MiddlewareFunc.Then",
				http.StatusInternalServerError,
			)
		})
	}

	return mw(next)
}
