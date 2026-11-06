package endpoints

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"unicode/utf8"
)

// DefaultPattern is the ServeMux pattern [Register] mounts the listing handler at.
const DefaultPattern = "GET /endpoints"

const (
	allowGetHead        = "GET, HEAD"
	cacheControlNoStore = "no-store"
	// headerAccept names the request header negotiated on; it is also the
	// Vary value, since the two representations share one URL.
	headerAccept   = "Accept"
	listingSummary = "Endpoint listing (this page)"
)

// Route describes one endpoint in the registry listing.
type Route struct {
	// Method is the HTTP method part of the ServeMux pattern ("GET" for
	// "GET /users/{id}"), or "" when the pattern matches every method.
	Method string `json:"method"`

	// Pattern is the full ServeMux pattern as registered, including any
	// method and host segments, e.g. "GET /users/{id}".
	Pattern string `json:"pattern"`

	// Summary is an optional one-line human description shown in the
	// listing; "" omits it from the JSON representation.
	Summary string `json:"summary,omitempty"`
}

// Registry records HTTP routes at registration time and serves a
// content-negotiated listing of them. Because *http.ServeMux cannot enumerate
// its patterns, the registry is the source of truth: routes registered through
// [Registry.Handle] / [Registry.HandleFunc] are mounted on the mux AND
// recorded in the listing in one declaration.
//
// The zero value is an empty registry ready for [Registry.Handle];
// [NewRegistry] exists for explicitness at call sites.
//
// A Registry is safe for concurrent use: Routes and the listing handler read
// immutable snapshots published atomically by Handle, so routes can be added
// while a listing is serving (each request then sees either the pre- or
// post-add state, never a torn mixture).
type Registry struct {
	routes atomic.Pointer[[]Route]
}

// muxRejectedError reports a pattern the ServeMux refused to register,
// translated from the panic ServeMux raises for malformed or duplicate
// patterns so this module keeps a never-panic API surface.
type muxRejectedError struct {
	pattern string
	reason  any
}

// Error describes the rejected pattern and the ServeMux's reason.
func (e *muxRejectedError) Error() string {
	return fmt.Sprintf("serve mux rejected pattern %q: %v", e.pattern, e.reason)
}

var (
	// errMuxNil is returned when no mux is provided to register on.
	errMuxNil = errors.New("mux must not be nil")

	// errHandlerNil is returned when no handler is provided to mount.
	errHandlerNil = errors.New("handler must not be nil")

	// errPatternEmpty mirrors MiddlewareStack's empty-name rejection: an
	// empty pattern defeats the listing's purpose and hides wiring bugs.
	errPatternEmpty = errors.New("pattern must not be empty")

	// errPatternInvalidUTF8 rejects patterns that are not valid UTF-8.
	errPatternInvalidUTF8 = errors.New("pattern must be valid UTF-8")

	// errSummaryInvalidUTF8 rejects summaries that are not valid UTF-8;
	// encoding/json v2 refuses to marshal invalid UTF-8, and rejecting at
	// registration time keeps the listing renderable by construction.
	errSummaryInvalidUTF8 = errors.New("summary must be valid UTF-8")
)

// NewRegistry returns an empty registry ready for [Registry.Handle].
func NewRegistry() *Registry {
	empty := []Route{}

	registry := &Registry{
		routes: atomic.Pointer[[]Route]{},
	}
	registry.routes.Store(&empty)

	return registry
}

// Handle registers handler on mux under pattern and records the route in the
// listing. pattern is a ServeMux pattern ("GET /users/{id}", "/health",
// "example.com/{$}", ...); the mux remains the pattern authority — a pattern
// it rejects (malformed, or already registered) comes back as an error instead
// of the panic ServeMux would raise. summary is optional.
//
// Safe for concurrent use.
func (r *Registry) Handle(mux *http.ServeMux, pattern string, handler http.Handler, summary string) error {
	if mux == nil {
		return errMuxNil
	}

	if handler == nil {
		return errHandlerNil
	}

	if pattern == "" {
		return errPatternEmpty
	}

	if !utf8.ValidString(pattern) {
		return fmt.Errorf("%w: %q", errPatternInvalidUTF8, pattern)
	}

	if !utf8.ValidString(summary) {
		return fmt.Errorf("%w: %q", errSummaryInvalidUTF8, summary)
	}

	if err := registerOnMux(mux, pattern, handler); err != nil {
		return err
	}

	r.record(Route{Method: methodOf(pattern), Pattern: pattern, Summary: summary})

	return nil
}

// HandleFunc is [Registry.Handle] for a bare handler function.
func (r *Registry) HandleFunc(
	mux *http.ServeMux,
	pattern string,
	handler func(http.ResponseWriter, *http.Request),
	summary string,
) error {
	if handler == nil {
		return errHandlerNil
	}

	return r.Handle(mux, pattern, http.HandlerFunc(handler), summary)
}

// Routes returns the registered routes sorted by pattern (then method). The
// result is a defensive copy; callers may mutate it.
func (r *Registry) Routes() []Route {
	current := r.routes.Load()
	if current == nil {
		return nil
	}

	routes := make([]Route, len(*current))
	copy(routes, *current)

	slices.SortFunc(routes, func(a, b Route) int {
		return cmp.Or(
			strings.Compare(a.Pattern, b.Pattern),
			strings.Compare(a.Method, b.Method),
		)
	})

	return routes
}

// Handler returns the listing handler. It serves GET and HEAD (other methods
// get 405 with an Allow header) and negotiates the representation from the
// Accept request header: text/html renders a plain HTML table,
// application/json (and */*, and absent Accept headers) render JSON. Every
// request reads a fresh snapshot, so routes added after Handler was called
// still appear.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(r.serveListing)
}

// Register mounts the listing handler on mux at [DefaultPattern]
// ("GET /endpoints"), recording it in the registry so the listing includes
// itself.
func Register(mux *http.ServeMux, registry *Registry) error {
	return registry.Handle(mux, DefaultPattern, registry.Handler(), listingSummary)
}

// serveListing writes the negotiated listing representation.
func (r *Registry) serveListing(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		w.Header().Set("Allow", allowGetHead)
		w.WriteHeader(http.StatusMethodNotAllowed)

		return
	}

	routes := r.Routes()

	w.Header().Set("Cache-Control", cacheControlNoStore)
	w.Header().Add("Vary", headerAccept)

	if negotiateRepresentation(req.Header.Get(headerAccept)) == representationHTML {
		w.Header().Set("Content-Type", contentTypeHTML)
		w.WriteHeader(http.StatusOK)
		writeCommitted(w, renderHTML(routes))

		return
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	writeCommitted(w, renderJSON(routes))
}

// registerOnMux registers handler on mux, translating the panic ServeMux
// raises for a malformed or duplicate pattern into a returned error.
func registerOnMux(mux *http.ServeMux, pattern string, handler http.Handler) error {
	var muxErr error

	func() {
		defer func() {
			if rejected := recover(); rejected != nil {
				muxErr = &muxRejectedError{
					pattern: pattern,
					reason:  rejected,
				}
			}
		}()

		mux.Handle(pattern, handler)
	}()

	return muxErr
}

// record publishes route as a new immutable snapshot, retrying when a
// concurrent record won the publication race.
func (r *Registry) record(route Route) {
	for {
		current := r.routes.Load()

		var currentRoutes []Route

		if current != nil {
			currentRoutes = *current
		}

		next := make([]Route, 0, len(currentRoutes)+1)
		next = append(next, currentRoutes...)
		next = append(next, route)

		if r.routes.CompareAndSwap(current, &next) {
			return
		}
	}
}

// methodOf extracts the method segment of a ServeMux pattern: the first
// whitespace-delimited field when it contains no '/'. Patterns without a
// method ("example.com/" or "/users/{id}") return "".
func methodOf(pattern string) string {
	sep := strings.IndexAny(pattern, " \t")
	if sep < 1 {
		return ""
	}

	method := pattern[:sep]
	if strings.ContainsRune(method, '/') {
		return ""
	}

	return method
}

// writeCommitted writes a response body after the status line is committed;
// write failures cannot be surfaced to the client ("honest silence", the same
// convention as the root module's health and recovery body writes).
func writeCommitted(w io.Writer, body string) {
	_, _ = io.WriteString(w, body)
}
