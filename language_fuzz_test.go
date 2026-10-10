package httputil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// FuzzParseAcceptLanguage pins the parser's invariants over arbitrary
// header bytes: candidates are never empty strings, never contain
// separators that would smuggle extra entries, ordering is weight
// non-increasing, and the full middleware path stays panic-free with the
// negotiated tag charset-safe for the Content-Language header.
func FuzzParseAcceptLanguage(f *testing.F) {
	seeds := []string{
		"",
		"en",
		"de-AT,de;q=0.9,en;q=0.8",
		"en;q=0.5,de;q=0.9,fr",
		"*,en;q=0",
		"de;q=abc",
		"de;q=0.5000",
		"en;q=1.0,en;q=0.9",
		";;;",
		"  de  ;  q=0.5  , en",
		"de;q=-0.5",
		"de;q=2",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, header string) {
		t.Parallel()

		tags, ok := parseAcceptLanguageCandidates(header)

		if !ok {
			return
		}

		if len(tags) == 0 {
			t.Fatal("ok=true with zero candidates")
		}

		prevWeight := 2.0

		for i, tag := range tags {
			if tag == "" {
				t.Fatalf("candidate %d is empty for header %q", i, header)
			}

			if strings.ContainsAny(tag, ",;") {
				t.Fatalf("candidate %q contains an entry separator (header %q)", tag, header)
			}

			weight, found := candidateWeight(header, tag)
			if found && weight > prevWeight+1e-9 {
				t.Fatalf("candidates %v not in non-increasing weight order (header %q)", tags, header)
			}

			if found {
				prevWeight = weight
			}
		}

		// Full middleware path must stay panic-free and produce a
		// charset-safe Content-Language value for any supported set.
		cfg := LanguageConfig{
			SupportedTags: []string{"de", "en"},
			Extractors:    []LanguageExtractor{LanguageExtractorFromAcceptHeader()},
		}

		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.Header.Set("Accept-Language", header)

		rec := httptest.NewRecorder()

		Language(cfg)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rec, req)

		served := rec.Header().Get("Content-Language")
		if served != "de" && served != "en" {
			t.Fatalf("Content-Language = %q, want a supported tag (header %q)", served, header)
		}
	})
}

// candidateWeight recovers the weight the parser assigned to tag by
// re-walking the header entries (fuzz-oracle helper; bounded by the header
// length the fuzz engine supplies).
func candidateWeight(header, tag string) (float64, bool) {
	for _, entry := range strings.Split(header, ",") {
		candidate, params, _ := strings.Cut(entry, ";")
		if trim(candidate) != tag {
			continue
		}

		weight := defaultQValue

		if qPart, found := strings.CutPrefix(trim(params), qValuePrefix); found {
			if parsed, err := parseQValue(qPart); err == nil {
				weight = parsed
			}
		}

		return weight, true
	}

	return 0, false
}
