package httpspec

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkCheck(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

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

			for range b.N {
				_ = bc.check(mux)
			}
		})
	}
}

func BenchmarkCheckServesRequest(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()

	for range b.N {
		mux.ServeHTTP(rec, req)
	}
}
