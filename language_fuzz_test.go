package httputil

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// FuzzParseAcceptLanguage pins the parser's invariants over arbitrary
// header bytes: candidates are never empty, never carry entry separators,
// the ordering matches an independent differential oracle, and the full
// middleware path stays panic-free with a charset-safe Content-Language.
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
		"0;q=0.1,0,000",
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

		for i, tag := range tags {
			if tag == "" {
				t.Fatalf("candidate %d is empty for header %q", i, header)
			}

			if strings.ContainsAny(tag, ",;") {
				t.Fatalf("candidate %q contains an entry separator (header %q)", tag, header)
			}
		}

		want := differentialAcceptLanguage(header)
		if strings.Join(tags, ",") != strings.Join(want, ",") {
			t.Fatalf("candidates %v, want oracle %v (header %q)", tags, want, header)
		}

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

// differentialAcceptLanguage re-implements the candidate parse with an
// independent mechanism for the numeric work (strconv.ParseFloat instead
// of the production hand-rolled scanner) and returns the expected order.
// Mirrors production semantics: OWS trim, control-char skip, q-grammar
// subset (int part 0|1, <=3 decimals, optional sign, no q>1), malformed
// q keeps default weight.
func differentialAcceptLanguage(header string) []string {
	type pref struct {
		tag    string
		weight float64
	}

	var prefs []pref

	for entry := range strings.SplitSeq(header, ",") {
		tag, params, _ := strings.Cut(entry, ";")
		tag = trim(tag)

		if tag == "" || tag == "*" || containsControlChar(tag) {
			continue
		}

		weight, known := oracleEntryWeight(params)
		if !known {
			weight = defaultQValue
		}

		if weight <= 0 {
			continue
		}

		prefs = append(prefs, pref{tag: tag, weight: weight})
	}

	sort.SliceStable(prefs, func(i, j int) bool {
		return prefs[i].weight > prefs[j].weight
	})

	tags := make([]string, 0, len(prefs))
	for _, p := range prefs {
		tags = append(tags, p.tag)
	}

	return tags
}

// oracleEntryWeight resolves the q-parameter via strconv. The bool result
// reports whether the parameter was well-formed (false keeps default
// weight, mirroring production's malformed-q posture).
func oracleEntryWeight(params string) (float64, bool) {
	rest, found := strings.CutPrefix(trim(params), qValuePrefix)
	if !found {
		return defaultQValue, false
	}

	s := rest

	neg := strings.HasPrefix(s, "-")
	switch {
	case neg:
		s = s[1:]
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	}

	if !oracleQGrammar(s) {
		return defaultQValue, false
	}

	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultQValue, false
	}

	if neg {
		value = -value
	}

	return value, true
}

// oracleQGrammar reports whether s is in the RFC 7231 q-value subset the
// production parser accepts: integer part 0 or 1, optional dot plus 1-3
// decimals, and no value above 1 (fraction on 1 must be zero).
func oracleQGrammar(s string) bool {
	intPart, frac, _ := strings.Cut(s, ".")

	if intPart != "0" && intPart != "1" {
		return false
	}

	if len(frac) > 3 {
		return false
	}

	for i := range len(frac) {
		if frac[i] < '0' || frac[i] > '9' {
			return false
		}
	}

	if intPart == "1" && frac != "" {
		for i := range len(frac) {
			if frac[i] != '0' {
				return false
			}
		}
	}

	return true
}
