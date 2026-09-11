package httputil

import (
	"net/http"
	"sync/atomic"
)

// Well-known middleware names for use with [MiddlewareStack].
// Use these when calling [MiddlewareStack.Add] to enable ordering validation.
const (
	MiddlewareRecovery        = "recovery"
	MiddlewareLogging         = "logging"
	MiddlewareRequestID       = "request-id"
	MiddlewareCORS            = "cors"
	MiddlewareSecurityHeaders = "security-headers"
	MiddlewareCompression     = "compression"
	MiddlewareDecompression   = "decompression"
	MiddlewareTimeout         = "timeout"
	MiddlewareClientIP        = "client-ip"
	MiddlewareCSRF            = "csrf"
	MiddlewareServerTiming    = "server-timing"
	MiddlewareKeyedRateLimit  = "keyed-rate-limit"
	MiddlewareETag            = "etag"
	MiddlewareNonce           = "nonce"
)

// Error codes for MiddlewareStack construction, classified as Rejection.
const (
	codeStackDuplicateMiddleware = Code("stack.duplicate_middleware")
	codeStackRecoveryNotFirst    = Code("stack.recovery_not_first")
	codeStackEmptyName           = Code("stack.name_empty")
)

var (
	errDuplicateMiddleware = codeStackDuplicateMiddleware.Rejection(
		"middleware with this name is already in the stack",
	)
	errRecoveryNotFirst = codeStackRecoveryNotFirst.Rejection(
		"recovery middleware must be first (outermost) so it can catch panics from all other middleware",
	)
	errEmptyMiddlewareName = codeStackEmptyName.Rejection(
		"middleware name must not be empty: an empty name defeats duplicate detection and produces empty diagnostics",
	)
)

// MiddlewareStack collects named middleware entries, validates their ordering,
// and builds the final handler chain. It prevents accidental duplication and
// enforces that [MiddlewareRecovery] is outermost when present.
//
// Add is safe to call concurrently with reads (Names, Validate, Build, and the
// middleware returned by Middleware): every read observes an immutable
// snapshot of the entries published atomically by Add, so a stack can be
// extended while a handler built from an earlier snapshot is serving.
type MiddlewareStack struct {
	entries atomic.Pointer[[]middlewareEntry]
}

type middlewareEntry struct {
	name       string
	middleware Middleware
}

// NewMiddlewareStack returns an empty stack ready for [MiddlewareStack.Add].
func NewMiddlewareStack() *MiddlewareStack {
	empty := []middlewareEntry{}

	s := &MiddlewareStack{}
	s.entries.Store(&empty)

	return s
}

// snapshot returns the current immutable entry slice.
func (s *MiddlewareStack) snapshot() []middlewareEntry {
	if cached := s.entries.Load(); cached != nil {
		return *cached
	}

	return nil
}

// Add appends a named middleware to the stack. The first middleware added
// becomes the outermost wrapper when [MiddlewareStack.Build] is called.
// Returns an error if a middleware with the same name is already present, or
// if name is empty. Safe for concurrent use.
func (s *MiddlewareStack) Add(name string, middleware Middleware) error {
	if name == "" {
		return errEmptyMiddlewareName.WithContext("name", name)
	}

	for {
		current := s.entries.Load()

		var next []middlewareEntry

		if current != nil {
			for _, e := range *current {
				if e.name == name {
					return errDuplicateMiddleware.WithContext("name", name)
				}
			}

			next = make([]middlewareEntry, len(*current)+1)
			copy(next, *current)
		} else {
			next = make([]middlewareEntry, 1)
		}

		next[len(next)-1] = middlewareEntry{name: name, middleware: middleware}

		if s.entries.CompareAndSwap(current, &next) {
			return nil
		}
		// Lost the publication race: retry against the winner's snapshot so
		// duplicate detection sees the entry that interleaved with ours.
	}
}

// Names returns the names of all middleware in the stack, in order.
func (s *MiddlewareStack) Names() []string {
	entries := s.snapshot()

	//nolint:makezero // pre-allocated with known length, not append
	names := make([]string, len(entries))

	for i, e := range entries {
		names[i] = e.name
	}

	return names
}

// Validate checks for common ordering mistakes. Currently enforces that
// [MiddlewareRecovery], when present, is the first (outermost) middleware.
func (s *MiddlewareStack) Validate() error {
	for i, e := range s.snapshot() {
		if e.name == MiddlewareRecovery && i != 0 {
			return errRecoveryNotFirst.WithContextAny("position", i)
		}
	}

	return nil
}

// Build applies all middleware to the handler and returns the final handler.
// The first middleware added becomes the outermost wrapper. Does not call
// [MiddlewareStack.Validate]; call it separately to check ordering.
func (s *MiddlewareStack) Build(handler http.Handler) http.Handler {
	return s.Middleware()(handler)
}

// Middleware returns the stack as a single composable middleware. Ordering
// matches [MiddlewareStack.Build]: the first middleware added is the
// outermost wrapper when the returned middleware is applied to a handler.
// Each application reads an immutable snapshot of the stack's entries, so
// middleware added after this call is included — and a snapshot being applied
// concurrently with an Add observes either the pre- or post-Add state, never
// a torn mixture.
func (s *MiddlewareStack) Middleware() Middleware {
	return func(handler http.Handler) http.Handler {
		entries := s.snapshot()

		//nolint:makezero // pre-allocated with known length, not append
		mws := make([]Middleware, len(entries))

		for i, e := range entries {
			mws[i] = e.middleware
		}

		return Chain(handler, mws...)
	}
}
