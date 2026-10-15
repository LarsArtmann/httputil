package httputil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// languageBenchHandler is a no-op terminal handler; the benchmark measures
// the middleware, not the handler.
func languageBenchHandler(http.ResponseWriter, *http.Request) {}

// languageStaticWriter is a minimal ResponseWriter over io.Discard: a bare
// header map with no recorder machinery. It exists to isolate the
// middleware's own cost from httptest's synthetic-request/recorder
// overhead (measured at ~750 ns and ~5 KB per construction in
// docs/research/2026-08-30_httptest-newrequest-profiling.md; the recorder's
// buffered body adds its own allocations per write).
type languageStaticWriter struct{ header http.Header }

func (w *languageStaticWriter) Header() http.Header { return w.header }

func (w *languageStaticWriter) Write(p []byte) (int, error) {
	n, err := io.Discard.Write(p)

	return n, err
}

func (w *languageStaticWriter) WriteHeader(int) {}

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

// BenchmarkLanguageStaticRequest runs the same funnel shape as
// BenchmarkLanguage against the static writer: the delta between the two
// (same request object, same handler chain) is the httptest recorder
// overhead, so the static numbers are the middleware's own per-request
// cost. Headers are cleared each iteration so stamping cost stays in the
// measurement without growing the map across the loop.
func BenchmarkLanguageStaticRequest(b *testing.B) {
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
	w := &languageStaticWriter{header: make(http.Header, 2)}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		clear(w.header)
		handler.ServeHTTP(w, req)
	}
}
