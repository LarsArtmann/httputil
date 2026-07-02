package httpspec

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Category groups related specs for reporting and filtering.
type Category string

const (
	// CategoryRouting covers path resolution and status code correctness.
	CategoryRouting Category = "routing"

	// CategoryHeaders covers response header compliance.
	CategoryHeaders Category = "headers"

	// CategoryMethods covers HTTP method handling.
	CategoryMethods Category = "methods"

	// CategorySecurity covers information leakage in error responses.
	CategorySecurity Category = "security"
)

// Spec is a named behavioral check that can be run against any [http.Handler].
type Spec struct {
	Name     string
	Category Category
	Check    Check
}

// Check examines an [http.Handler] and returns a [Result] describing whether
// the handler conforms to the expected behavior.
type Check func(handler http.Handler) Result

// Result describes the outcome of a [Check].
type Result struct {
	// OK is true when the handler passed the specification.
	OK bool

	// Message describes the failure when OK is false. It is empty when OK is true.
	Message string
}

// String returns a human-readable description of the result.
func (r Result) String() string {
	if r.OK {
		return "passed"
	}

	return r.Message
}

// Pass returns a passing [Result].
func Pass() Result {
	return Result{OK: true, Message: ""}
}

// Fail returns a failing [Result] with a formatted message.
// The format string and args follow [fmt.Sprintf] conventions.
func Fail(format string, args ...any) Result {
	return Result{OK: false, Message: fmt.Sprintf(format, args...)}
}

// SpecName constants identify each standard spec for use with [SkipSpec].
const (
	SpecNameIndexNot404                   = "the index page should not return 404 Not Found"
	SpecNameIndexNotServerError           = "the index page should not return a server error"
	SpecNameUnknownPathReturns404         = "unknown paths should return 404 or a redirect"
	SpecNamePostUnknownNotServerError     = "POST to unknown paths should not trigger server errors"
	SpecNameBodyHasContentType            = "responses with a body should include Content-Type"
	SpecNameErrorResponsesHaveContentType = "error responses with a body should include Content-Type"
	SpecNameHeadHandled                   = "HEAD requests should be handled without a server error"
	SpecNameOptionsHandled                = "OPTIONS requests should be handled without a server error"
	SpecNameTraceNotEnabled               = "TRACE method should be disabled to prevent Cross-Site Tracing"
	SpecNameRedirectHasLocation           = "redirect responses should include a Location header"
	SpecNameNoServerVersionHeader         = "Server header should not leak version information"
	SpecNameNoPoweredByHeader             = "X-Powered-By header should not be present"
	SpecNameNoLeakedInternals             = "error responses should not leak internal details"
	SpecNameXContentTypeOptions           = "responses should include X-Content-Type-Options: nosniff"
	SpecNameNoDuplicateHeaders            = "responses should not contain duplicate headers"
	SpecNameConnectRejected               = "CONNECT method should be rejected to prevent tunneling"
	SpecNameRespectsAcceptHeader          = "servers should respect Accept header for content negotiation"
	SpecNameLongURLHandled                = "servers should handle very long URLs without server errors"
)

// Option configures the spec runner.
type Option func(*config)

type config struct {
	indexPath  string
	skipSpecs  map[string]bool
	extraSpecs []Spec
}

const defaultIndexPath = "/"

func newConfig() config {
	return config{
		indexPath:  defaultIndexPath,
		skipSpecs:  make(map[string]bool),
		extraSpecs: nil,
	}
}

// WithIndexPath sets the path treated as the index page for spec assertions.
// The default is "/".
func WithIndexPath(path string) Option {
	return func(c *config) {
		c.indexPath = path
	}
}

// SkipSpec excludes the spec with the given name from execution.
// Use the [SpecName] constants for standard specs.
func SkipSpec(name string) Option {
	return func(c *config) {
		c.skipSpecs[name] = true
	}
}

// WithExtraSpecs adds custom specifications alongside the standard suite.
func WithExtraSpecs(specs ...Spec) Option {
	return func(c *config) {
		c.extraSpecs = append(c.extraSpecs, specs...)
	}
}

// ExpectStatus returns a [Check] that verifies a request to the given path
// with the given method returns the expected HTTP status code.
// Use it to build custom specs for application-specific routes.
func ExpectStatus(method, path string, expectedCode int) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(method, path))

		if rec.Code != expectedCode {
			return Fail("%s %s returned status %d, want %d", method, path, rec.Code, expectedCode)
		}

		return Pass()
	}
}

// ExpectNotStatus returns a [Check] that verifies a request to the given path
// with the given method does NOT return the specified HTTP status code.
// Useful for asserting that certain endpoints avoid specific error codes.
func ExpectNotStatus(method, path string, notCode int) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(method, path))

		if rec.Code == notCode {
			return Fail("%s %s returned status %d, which should be avoided", method, path, notCode)
		}

		return Pass()
	}
}

// ExpectHeader returns a [Check] that verifies a response header equals the
// expected value.
func ExpectHeader(method, path, header, expectedValue string) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(method, path))

		if got := rec.Header().Get(header); got != expectedValue {
			return Fail(
				"%s %s header %q = %q, want %q",
				method, path, header, got, expectedValue,
			)
		}

		return Pass()
	}
}

// ExpectHeaderAbsent returns a [Check] that verifies a response header is not
// present.
func ExpectHeaderAbsent(method, path, header string) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(method, path))

		if got := rec.Header().Get(header); got != "" {
			return Fail("%s %s should not set header %q, but got %q", method, path, header, got)
		}

		return Pass()
	}
}

// ExpectBodyContains returns a [Check] that verifies the response body
// contains the given substring.
func ExpectBodyContains(method, path, substring string) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(method, path))

		if !strings.Contains(rec.Body.String(), substring) {
			return Fail(
				"%s %s response body does not contain %q",
				method, path, substring,
			)
		}

		return Pass()
	}
}

// Run executes all standard HTTP behavior specifications against the given
// handler. Each spec runs as a parallel subtest, producing output that reads
// like a behavior specification document.
//
// The handler must not be nil. For handlers with shared mutable state that
// cannot handle concurrent requests, use [RunSerial] instead.
func Run(t *testing.T, handler http.Handler, opts ...Option) {
	t.Helper()

	runSpecs(t, handler, true, opts...)
}

// RunSerial is like [Run] but runs each spec sequentially instead of in
// parallel. Use this when the handler has shared mutable state that cannot
// safely handle concurrent requests.
func RunSerial(t *testing.T, handler http.Handler, opts ...Option) {
	t.Helper()

	runSpecs(t, handler, false, opts...)
}

func runSpecs(t *testing.T, handler http.Handler, parallel bool, opts ...Option) {
	t.Helper()

	if handler == nil {
		t.Fatal("httpspec: handler must not be nil")
	}

	cfg := newConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	specs := standardSpecs(cfg)
	specs = append(specs, cfg.extraSpecs...)

	for _, spec := range specs {
		if cfg.skipSpecs[spec.Name] {
			continue
		}

		t.Run(spec.Name, func(t *testing.T) {
			if parallel {
				t.Parallel()
			}

			result := spec.Check(handler)
			if !result.OK {
				t.Error(result.Message)
			}
		})
	}
}

func mustRequest(method, target string) *http.Request {
	req, err := http.NewRequestWithContext(context.Background(), method, target, nil)
	if err != nil {
		// This is unreachable for the valid methods and paths used by httpspec.
		// A failure here means a bug in the spec definitions themselves.
		panic("httpspec: failed to create " + method + " " + target + ": " + err.Error())
	}

	return req
}

func serve(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}
