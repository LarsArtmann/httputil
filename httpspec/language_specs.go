package httpspec

import (
	"net/http"
	"net/http/httptest"
	"strings"
)

// Language spec names identify each language-negotiation behavior check for
// use with [SkipSpec].
const (
	SpecNameLanguageContentLanguagePresent = "language-aware responses should include Content-Language"
	SpecNameLanguageVaryAcceptLanguage     = "language-aware responses should set Vary: Accept-Language"
)

// LanguageSpecs returns language-negotiation behavior specs that can be
// composed into a spec run via [WithExtraSpecs]. They verify the two header
// contracts every language-aware handler owes its clients: Content-Language
// names the language of this response, and Vary lists Accept-Language so
// shared caches do not serve one locale's body to another locale's request.
//
// The returned specs assume the handler negotiates a response language
// (e.g. wrapped with httputil's Language middleware). For handlers that do
// not negotiate, omit these specs. Use [SkipSpec] for individual specs that
// are not applicable to your deployment.
func LanguageSpecs() []Spec {
	return []Spec{
		{
			Name:     SpecNameLanguageContentLanguagePresent,
			Category: CategoryHeaders,
			Check:    contentLanguagePresentCheck(),
		},
		{
			Name:     SpecNameLanguageVaryAcceptLanguage,
			Category: CategoryHeaders,
			Check:    varyAcceptLanguageCheck(),
		},
	}
}

// contentLanguagePresentCheck verifies that a language-negotiated response
// carries Content-Language. RFC 7231 permits a language list, so the check
// asserts presence only, not the number of tags.
func contentLanguagePresentCheck() Check {
	return func(handler http.Handler) Result {
		req := mustRequest(http.MethodGet, "/")
		req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")

		rec := serve(handler, req)

		if rec.Header().Get("Content-Language") == "" {
			return Fail(
				"response to a request with Accept-Language has no Content-Language header; " +
					"clients and caches cannot tell which language the body is in",
			)
		}

		return Pass()
	}
}

// varyAcceptLanguageCheck verifies that the response varies with the request
// header the negotiation reads. Vary may be comma-separated on one line or
// repeated across lines; both spellings are accepted.
func varyAcceptLanguageCheck() Check {
	return func(handler http.Handler) Result {
		req := mustRequest(http.MethodGet, "/")
		req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")

		rec := serve(handler, req)

		if !varyListsToken(rec, "Accept-Language") {
			return Fail(
				"response negotiates on Accept-Language but Vary = %v does not list it; "+
					"shared caches may serve one locale's body to another locale's request",
				rec.Header().Values("Vary"),
			)
		}

		return Pass()
	}
}

// varyListsToken reports whether any Vary header line (or comma-separated
// entry within a line) names token, case-insensitively.
func varyListsToken(rec *httptest.ResponseRecorder, token string) bool {
	for _, line := range rec.Header().Values("Vary") {
		for entry := range strings.SplitSeq(line, ",") {
			if strings.EqualFold(strings.TrimSpace(entry), token) {
				return true
			}
		}
	}

	return false
}
