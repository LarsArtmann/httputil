package httpspec

import (
	"net/http"
	"net/textproto"
	"slices"
	"strings"
)

const (
	unknownPath   = "/httpspec-nonexistent-a7f3e2d1c9b4"
	longURLLength = 8192
)

// leakPatterns are substrings that indicate an error response is leaking
// internal details such as stack traces, source file paths, or runtime errors.
func leakPatterns() []string {
	return []string{
		"goroutine ",
		"/usr/local/go/",
		"/home/",
		"/.go/src/",
		".go:",
		"panic:",
		"runtime error",
	}
}

func standardSpecs(cfg config) []Spec {
	return slices.Concat(
		routingSpecs(cfg),
		methodSpecs(cfg),
		headerSpecs(cfg),
		securitySpecs(cfg),
	)
}

func routingSpecs(cfg config) []Spec {
	return []Spec{
		{
			Name:     SpecNameIndexNot404,
			Category: CategoryRouting,
			Check:    indexNot404Check(cfg.indexPath),
		},
		{
			Name:     SpecNameIndexNotServerError,
			Category: CategoryRouting,
			Check:    indexNotServerErrorCheck(cfg.indexPath),
		},
		{
			Name:     SpecNameUnknownPathReturns404,
			Category: CategoryRouting,
			Check:    unknownPathCheck(),
		},
		{
			Name:     SpecNameLongURLHandled,
			Category: CategoryRouting,
			Check:    longURLHandledCheck(),
		},
	}
}

func methodSpecs(cfg config) []Spec {
	return []Spec{
		{
			Name:     SpecNamePostUnknownNotServerError,
			Category: CategoryMethods,
			Check:    postUnknownNotServerErrorCheck(),
		},
		{
			Name:     SpecNameHeadHandled,
			Category: CategoryMethods,
			Check:    headHandledCheck(cfg.indexPath),
		},
		{
			Name:     SpecNameOptionsHandled,
			Category: CategoryMethods,
			Check:    optionsHandledCheck(cfg.indexPath),
		},
		{
			Name:     SpecNameTraceNotEnabled,
			Category: CategorySecurity,
			Check:    traceNotEnabledCheck(),
		},
		{
			Name:     SpecNameConnectRejected,
			Category: CategorySecurity,
			Check:    connectRejectedCheck(),
		},
	}
}

func headerSpecs(cfg config) []Spec {
	return []Spec{
		{
			Name:     SpecNameBodyHasContentType,
			Category: CategoryHeaders,
			Check:    bodyHasContentTypeCheck(cfg.indexPath),
		},
		{
			Name:     SpecNameErrorResponsesHaveContentType,
			Category: CategoryHeaders,
			Check:    errorResponsesHaveContentTypeCheck(),
		},
		{
			Name:     SpecNameRedirectHasLocation,
			Category: CategoryHeaders,
			Check:    redirectHasLocationCheck(cfg.indexPath),
		},
		{
			Name:     SpecNameNoDuplicateHeaders,
			Category: CategoryHeaders,
			Check:    noDuplicateHeadersCheck(cfg.indexPath),
		},
		{
			Name:     SpecNameRespectsAcceptHeader,
			Category: CategoryHeaders,
			Check:    respectsAcceptHeaderCheck(cfg.indexPath),
		},
	}
}

func securitySpecs(cfg config) []Spec {
	return []Spec{
		{
			Name:     SpecNameNoServerVersionHeader,
			Category: CategorySecurity,
			Check:    noServerVersionHeaderCheck(),
		},
		{
			Name:     SpecNameNoPoweredByHeader,
			Category: CategorySecurity,
			Check:    noPoweredByHeaderCheck(),
		},
		{
			Name:     SpecNameNoLeakedInternals,
			Category: CategorySecurity,
			Check:    noLeakedInternalsCheck(),
		},
		{
			Name:     SpecNameXContentTypeOptions,
			Category: CategorySecurity,
			Check:    xContentTypeOptionsCheck(cfg.indexPath),
		},
	}
}

func indexNot404Check(indexPath string) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodGet, indexPath))

		if rec.Code == http.StatusNotFound {
			return Fail(
				"GET %s returned 404 Not Found, but the index page should be reachable",
				indexPath,
			)
		}

		return Pass()
	}
}

func indexNotServerErrorCheck(indexPath string) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodGet, indexPath))

		if rec.Code >= http.StatusInternalServerError {
			return Fail(
				"GET %s returned status %d, but the index page should not return a server error",
				indexPath, rec.Code,
			)
		}

		return Pass()
	}
}

func unknownPathCheck() Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodGet, unknownPath))

		isNotFound := rec.Code == http.StatusNotFound
		isRedirect := rec.Code >= http.StatusMultipleChoices && rec.Code < http.StatusBadRequest

		if !isNotFound && !isRedirect {
			return Fail(
				"GET %s returned status %d, want 404 Not Found or a 3xx redirect for unknown paths",
				unknownPath, rec.Code,
			)
		}

		return Pass()
	}
}

func postUnknownNotServerErrorCheck() Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodPost, unknownPath))

		if rec.Code >= http.StatusInternalServerError {
			return Fail(
				"POST %s returned status %d, but unknown paths should not trigger server errors",
				unknownPath, rec.Code,
			)
		}

		return Pass()
	}
}

func bodyHasContentTypeCheck(indexPath string) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodGet, indexPath))

		if rec.Body.Len() == 0 {
			return Pass()
		}

		if rec.Header().Get("Content-Type") == "" {
			return Fail("GET %s returned a response body without a Content-Type header", indexPath)
		}

		return Pass()
	}
}

func errorResponsesHaveContentTypeCheck() Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodGet, unknownPath))

		if rec.Body.Len() == 0 {
			return Pass()
		}

		if rec.Code >= http.StatusBadRequest && rec.Header().Get("Content-Type") == "" {
			return Fail(
				"GET %s returned error status %d with a body but no Content-Type header",
				unknownPath, rec.Code,
			)
		}

		return Pass()
	}
}

func headHandledCheck(indexPath string) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodHead, indexPath))

		if rec.Code >= http.StatusInternalServerError {
			return Fail(
				"HEAD %s returned status %d, server errors indicate HEAD is not handled",
				indexPath,
				rec.Code,
			)
		}

		return Pass()
	}
}

func optionsHandledCheck(indexPath string) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodOptions, indexPath))

		if rec.Code >= http.StatusInternalServerError {
			return Fail(
				"OPTIONS %s returned status %d, server errors indicate OPTIONS is not handled",
				indexPath, rec.Code,
			)
		}

		return Pass()
	}
}

func traceNotEnabledCheck() Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodTrace, unknownPath))

		if rec.Code == http.StatusOK {
			return Fail(
				"TRACE %s returned 200 OK, TRACE should be disabled to prevent Cross-Site Tracing (XST)",
				unknownPath,
			)
		}

		return Pass()
	}
}

func redirectHasLocationCheck(indexPath string) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodGet, indexPath))

		isRedirect := rec.Code >= http.StatusMultipleChoices && rec.Code < http.StatusBadRequest
		if !isRedirect {
			return Pass()
		}

		if rec.Header().Get("Location") == "" {
			return Fail(
				"GET %s returned redirect status %d without a Location header",
				indexPath, rec.Code,
			)
		}

		return Pass()
	}
}

func noServerVersionHeaderCheck() Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodGet, unknownPath))

		server := rec.Header().Get("Server")

		if hasVersionLeak(server) {
			return Fail(
				"Server header %q leaks version information, which helps attackers fingerprint the stack",
				server,
			)
		}

		return Pass()
	}
}

func noPoweredByHeaderCheck() Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodGet, unknownPath))

		if rec.Header().Get("X-Powered-By") != "" {
			return Fail(
				"X-Powered-By header is set to %q, this leaks framework information",
				rec.Header().Get("X-Powered-By"),
			)
		}

		return Pass()
	}
}

func noLeakedInternalsCheck() Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodGet, unknownPath))
		body := rec.Body.String()

		for _, pattern := range leakPatterns() {
			if strings.Contains(body, pattern) {
				return Fail(
					"error response body for GET %s contains potentially sensitive pattern %q",
					unknownPath, pattern,
				)
			}
		}

		return Pass()
	}
}

// hasVersionLeak checks if a Server header value contains a version number
// pattern like "nginx/1.21.3" or "Apache/2.4.41 (Ubuntu)".
func hasVersionLeak(server string) bool {
	for i := range len(server) - 1 {
		if server[i] == '/' && i+1 < len(server) && server[i+1] >= '0' && server[i+1] <= '9' {
			return true
		}
	}

	return false
}

func connectRejectedCheck() Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodConnect, unknownPath))

		if rec.Code == http.StatusOK {
			return Fail(
				"CONNECT %s returned 200 OK, CONNECT should be rejected to prevent HTTP tunneling",
				unknownPath,
			)
		}

		return Pass()
	}
}

func xContentTypeOptionsCheck(indexPath string) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodGet, indexPath))

		if rec.Header().Get("X-Content-Type-Options") == "" {
			return Fail(
				"GET %s does not set X-Content-Type-Options: nosniff, which prevents MIME-type sniffing attacks",
				indexPath,
			)
		}

		return Pass()
	}
}

func noDuplicateHeadersCheck(indexPath string) Check {
	return func(handler http.Handler) Result {
		rec := serve(handler, mustRequest(http.MethodGet, indexPath))

		seenCanonical := make(map[string]string)

		for key, values := range rec.Header() {
			if len(values) > 1 {
				return Fail(
					"GET %s response has duplicate %q headers (%d values), which can confuse clients and proxies",
					indexPath,
					key,
					len(values),
				)
			}

			canonical := textproto.CanonicalMIMEHeaderKey(key)

			if original, exists := seenCanonical[canonical]; exists {
				return Fail(
					"GET %s response sets the same header under two casings (%q and %q); clients that read canonical keys may miss one",
					indexPath,
					original,
					key,
				)
			}

			seenCanonical[canonical] = key
		}

		return Pass()
	}
}

func respectsAcceptHeaderCheck(indexPath string) Check {
	return func(handler http.Handler) Result {
		req := mustRequest(http.MethodGet, indexPath)
		req.Header.Set("Accept", "application/json")

		rec := serve(handler, req)

		if rec.Code >= http.StatusInternalServerError {
			return Fail(
				"GET %s with Accept: application/json returned status %d, servers should handle Accept headers without crashing",
				indexPath,
				rec.Code,
			)
		}

		return Pass()
	}
}

func longURLHandledCheck() Check {
	return func(handler http.Handler) Result {
		longPath := "/" + strings.Repeat("a", longURLLength)
		rec := serve(handler, mustRequest(http.MethodGet, longPath))

		if rec.Code >= http.StatusInternalServerError {
			return Fail(
				"GET with a %d-byte URL returned status %d, long URLs should not trigger server errors",
				longURLLength,
				rec.Code,
			)
		}

		return Pass()
	}
}
