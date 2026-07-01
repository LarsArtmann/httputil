package httpspec

import (
	"net/http"
	"strings"
)

// unknownPath is a path that no reasonable handler should serve.
// It includes a random-looking suffix to avoid accidental matches.
const unknownPath = "/httpspec-nonexistent-a7f3e2d1c9b4"

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
			Name:     SpecNameBodyHasContentType,
			Category: CategoryHeaders,
			Check:    bodyHasContentTypeCheck(cfg.indexPath),
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
			Name:     SpecNameNoLeakedInternals,
			Category: CategorySecurity,
			Check:    noLeakedInternalsCheck(),
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
