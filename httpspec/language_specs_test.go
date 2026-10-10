package httpspec

import (
	"net/http"
	"testing"
)

// newLanguageAwareHandler mimics a handler wrapped with a language
// negotiation middleware: it stamps Content-Language and Vary per request.
func newLanguageAwareHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept-Language") != "" {
			w.Header().Set("Content-Language", "de")
			w.Header().Add("Vary", "Accept-Language")
		}

		w.WriteHeader(http.StatusOK)
	})
}

func TestLanguageSpecs_PassWithLanguageAwareHandler(t *testing.T) {
	t.Parallel()

	handler := newLanguageAwareHandler()

	for _, spec := range LanguageSpecs() {
		t.Run(spec.Name, func(t *testing.T) {
			t.Parallel()

			result := spec.Check(handler)
			if !result.OK {
				t.Errorf("spec %q failed: %s", spec.Name, result.Message)
			}
		})
	}
}

func TestLanguageSpecs_ContentLanguagePresent_FailsWithoutHeader(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	spec := lookupSpec(t, LanguageSpecs(), SpecNameLanguageContentLanguagePresent)

	result := spec.Check(handler)
	if result.OK {
		t.Errorf("expected Content-Language spec to fail without the header")
	}
}

func TestLanguageSpecs_ContentLanguagePresent_FailsWithEmptyHeader(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Language", "")
		w.WriteHeader(http.StatusOK)
	})

	spec := lookupSpec(t, LanguageSpecs(), SpecNameLanguageContentLanguagePresent)

	result := spec.Check(handler)
	if result.OK {
		t.Errorf("expected Content-Language spec to fail with an empty header value")
	}
}

func TestLanguageSpecs_VaryAcceptLanguage_FailsWithoutVary(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Language", "de")
		w.WriteHeader(http.StatusOK)
	})

	spec := lookupSpec(t, LanguageSpecs(), SpecNameLanguageVaryAcceptLanguage)

	result := spec.Check(handler)
	if result.OK {
		t.Errorf("expected Vary spec to fail without Vary")
	}
}

func TestLanguageSpecs_VaryAcceptLanguage_FailsWithUnrelatedVary(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Language", "de")
		w.Header().Set("Vary", "Accept-Encoding")
		w.WriteHeader(http.StatusOK)
	})

	spec := lookupSpec(t, LanguageSpecs(), SpecNameLanguageVaryAcceptLanguage)

	result := spec.Check(handler)
	if result.OK {
		t.Errorf("expected Vary spec to fail when Vary names only Accept-Encoding")
	}
}

func TestLanguageSpecs_VaryAcceptLanguage_PassesWithCommaSeparatedLine(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Language", "de")
		w.Header().Set("Vary", "Origin, Accept-Language")
		w.WriteHeader(http.StatusOK)
	})

	spec := lookupSpec(t, LanguageSpecs(), SpecNameLanguageVaryAcceptLanguage)

	result := spec.Check(handler)
	if !result.OK {
		t.Errorf("comma-separated Vary line should pass: %s", result.Message)
	}
}

func TestLanguageSpecs_VaryAcceptLanguage_PassesWithRepeatedLines(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Language", "de")
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Accept-Language")
		w.WriteHeader(http.StatusOK)
	})

	spec := lookupSpec(t, LanguageSpecs(), SpecNameLanguageVaryAcceptLanguage)

	result := spec.Check(handler)
	if !result.OK {
		t.Errorf("repeated Vary lines should pass: %s", result.Message)
	}
}

func TestLanguageSpecs_NamesAreRegistered(t *testing.T) {
	t.Parallel()

	specs := LanguageSpecs()
	if len(specs) != 2 {
		t.Fatalf("len(LanguageSpecs()) = %d, want 2", len(specs))
	}

	if specs[0].Name != SpecNameLanguageContentLanguagePresent {
		t.Errorf("specs[0].Name = %q, want %q", specs[0].Name, SpecNameLanguageContentLanguagePresent)
	}

	if specs[1].Name != SpecNameLanguageVaryAcceptLanguage {
		t.Errorf("specs[1].Name = %q, want %q", specs[1].Name, SpecNameLanguageVaryAcceptLanguage)
	}
}
