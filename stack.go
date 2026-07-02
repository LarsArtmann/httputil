package httputil

import (
	"errors"
	"fmt"
	"net/http"
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
	MiddlewareETag            = "etag"
	MiddlewareTimeout         = "timeout"
	MiddlewareClientIP        = "client-ip"
)

var (
	errDuplicateMiddleware = errors.New("middleware with this name is already in the stack")
	errRecoveryNotFirst    = errors.New(
		"recovery middleware must be first (outermost) so it can catch panics from all other middleware",
	)
)

// MiddlewareStack collects named middleware entries, validates their ordering,
// and builds the final handler chain. It prevents accidental duplication and
// enforces that [MiddlewareRecovery] is outermost when present.
type MiddlewareStack struct {
	entries []middlewareEntry
}

type middlewareEntry struct {
	name       string
	middleware Middleware
}

// NewMiddlewareStack returns an empty stack ready for [MiddlewareStack.Add].
func NewMiddlewareStack() *MiddlewareStack {
	return &MiddlewareStack{entries: nil}
}

// Add appends a named middleware to the stack. The first middleware added
// becomes the outermost wrapper when [MiddlewareStack.Build] is called.
// Returns an error if a middleware with the same name is already present.
func (s *MiddlewareStack) Add(name string, middleware Middleware) error {
	for _, e := range s.entries {
		if e.name == name {
			return fmt.Errorf("%w: %q", errDuplicateMiddleware, name)
		}
	}

	s.entries = append(s.entries, middlewareEntry{name: name, middleware: middleware})

	return nil
}

// Names returns the names of all middleware in the stack, in order.
func (s *MiddlewareStack) Names() []string {
	//nolint:makezero // pre-allocated with known length, not append
	names := make([]string, len(s.entries))

	for i, e := range s.entries {
		names[i] = e.name
	}

	return names
}

// Validate checks for common ordering mistakes. Currently enforces that
// [MiddlewareRecovery], when present, is the first (outermost) middleware.
func (s *MiddlewareStack) Validate() error {
	for i, e := range s.entries {
		if e.name == MiddlewareRecovery && i != 0 {
			return fmt.Errorf("%w: found at position %d, expected 0", errRecoveryNotFirst, i)
		}
	}

	return nil
}

// Build applies all middleware to the handler and returns the final handler.
// The first middleware added becomes the outermost wrapper. Does not call
// [MiddlewareStack.Validate]; call it separately to check ordering.
func (s *MiddlewareStack) Build(handler http.Handler) http.Handler {
	//nolint:makezero // pre-allocated with known length, not append
	mws := make([]Middleware, len(s.entries))

	for i, e := range s.entries {
		mws[i] = e.middleware
	}

	return Chain(handler, mws...)
}
