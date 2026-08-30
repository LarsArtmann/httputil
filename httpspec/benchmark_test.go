package httpspec

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkCheck(b *testing.B) {
	mux := newTypedHelloMux()

	checks := []struct {
		name  string
		check Check
	}{
		{"index_not_404", indexNot404Check("/")},
		{"unknown_path_404", unknownPathCheck()},
		{"body_has_content_type", bodyHasContentTypeCheck("/")},
		{"no_leaked_internals", noLeakedInternalsCheck()},
		{"long_url_handled", longURLHandledCheck()},
		{"expect_status", ExpectStatus(http.MethodGet, "/", http.StatusOK)},
	}

	for _, bc := range checks {
		b.Run(bc.name, func(b *testing.B) {
			b.ResetTimer()

			for b.Loop() {
				_ = bc.check(mux)
			}
		})
	}
}

func BenchmarkCheckServesRequest(b *testing.B) {
	mux := newTypedHelloMux()

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}
