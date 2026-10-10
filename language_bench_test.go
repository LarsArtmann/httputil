package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// languageBenchHandler is a no-op terminal handler; the benchmark measures
// the middleware, not the handler.
func languageBenchHandler(http.ResponseWriter, *http.Request) {}

// BenchmarkLanguage measures the full per-request path: extraction chain,
// matching, and header writes, with the two default legs (path miss +
// header hit) of the sperrmuell funnel shape.
func BenchmarkLanguage(b *testing.B) {
	cfg := LanguageConfig{
		SupportedTags: []string{"de", "en"},
		Extractors: []LanguageExtractor{LanguageExtractorChain(
			LanguageExtractorFromPathPrefix(map[string]string{"/en": "en"}),
			LanguageExtractorFromAcceptHeader(),
		)},
	}

	handler := Language(cfg)(http.HandlerFunc(languageBenchHandler))
	req := httptest.NewRequest(http.MethodGet, "/quote", http.NoBody)
	req.Header.Set("Accept-Language", "de-AT,de;q=0.9,en;q=0.8")
	rec := httptest.NewRecorder()

	b.ReportAllocs()

	for b.Loop() {
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkLanguageHeaderOnly isolates the Accept-Language leg without the
// path-prefix miss, the cheapest realistic configuration.
func BenchmarkLanguageHeaderOnly(b *testing.B) {
	cfg := LanguageConfig{
		SupportedTags: []string{"de", "en"},
		Extractors:    []LanguageExtractor{LanguageExtractorFromAcceptHeader()},
	}

	handler := Language(cfg)(http.HandlerFunc(languageBenchHandler))
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept-Language", "de-AT,de;q=0.9,en;q=0.8")
	rec := httptest.NewRecorder()

	b.ReportAllocs()

	for b.Loop() {
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkParseAcceptLanguageCandidates measures the header grammar parse
// alone (the reuse foundation other consumers may build on).
func BenchmarkParseAcceptLanguageCandidates(b *testing.B) {
	const header = "de-AT,de;q=0.9,en;q=0.8"

	b.ReportAllocs()

	for b.Loop() {
		if _, ok := parseAcceptLanguageCandidates(header); !ok {
			b.Fatal("parse failed")
		}
	}
}
