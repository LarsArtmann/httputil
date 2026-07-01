package httpspec

import (
	"net/http"
	"strings"
	"testing"
)

// --- Test handlers -------------------------------------------------------

func newGoodHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	return mux
}

func newAlways404Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
}

func newAlways500Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
}

func newNoContentTypeHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("body without content type"))
	})

	return mux
}

func newSPAHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>SPA fallback</html>"))
	})
}

func newLeakingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(
			[]byte(
				"panic: runtime error: goroutine 1 [running]:\n/usr/local/go/src/runtime/panic.go:123",
			),
		)
	})
}

// --- Run-level tests -----------------------------------------------------

func TestRunAllSpecsPassForGoodHandler(t *testing.T) {
	t.Parallel()
	Run(t, newGoodHandler())
}

// --- Option tests --------------------------------------------------------

func TestSkipSpecExcludesSpec(t *testing.T) {
	t.Parallel()
	Run(t, newSPAHandler(), SkipSpec(SpecNameUnknownPathReturns404))
}

func TestWithExtraSpecsPassingSpec(t *testing.T) {
	t.Parallel()

	custom := Spec{
		Name:     "custom spec that passes",
		Category: CategoryRouting,
		Check:    ExpectStatus(http.MethodGet, "/", http.StatusOK),
	}

	Run(t, newGoodHandler(), WithExtraSpecs(custom))
}

func TestWithIndexPathChangesTestedPath(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/app", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("app"))
	})

	Run(t, mux, WithIndexPath("/app"))
}

// --- Individual check tests ----------------------------------------------

func TestIndexNot404PassesFor200(t *testing.T) {
	t.Parallel()

	result := indexNot404Check("/")(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestIndexNot404FailsFor404(t *testing.T) {
	t.Parallel()

	result := indexNot404Check("/")(newAlways404Handler())
	if result.OK {
		t.Error("expected failure for handler returning 404 on index")
	}
}

func TestIndexNotServerErrorPassesFor200(t *testing.T) {
	t.Parallel()

	result := indexNotServerErrorCheck("/")(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestIndexNotServerErrorFailsFor500(t *testing.T) {
	t.Parallel()

	result := indexNotServerErrorCheck("/")(newAlways500Handler())
	if result.OK {
		t.Error("expected failure for handler returning 500 on index")
	}
}

func TestUnknownPathReturns404PassesFor404(t *testing.T) {
	t.Parallel()

	result := unknownPathCheck()(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestUnknownPathReturns404PassesForRedirect(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)
	})

	result := unknownPathCheck()(handler)
	if !result.OK {
		t.Errorf("expected pass for redirect, got: %s", result.Message)
	}
}

func TestUnknownPathReturns404FailsFor200(t *testing.T) {
	t.Parallel()

	result := unknownPathCheck()(newSPAHandler())
	if result.OK {
		t.Error("expected failure when unknown path returns 200")
	}
}

func TestBodyHasContentTypePasses(t *testing.T) {
	t.Parallel()

	result := bodyHasContentTypeCheck("/")(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestBodyHasContentTypeFailsWhenMissing(t *testing.T) {
	t.Parallel()

	result := bodyHasContentTypeCheck("/")(newNoContentTypeHandler())
	if result.OK {
		t.Error("expected failure when body lacks Content-Type")
	}
}

func TestBodyHasContentTypePassesForEmptyBody(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	result := bodyHasContentTypeCheck("/")(handler)
	if !result.OK {
		t.Errorf("expected pass for empty body, got: %s", result.Message)
	}
}

func TestHeadHandledPasses(t *testing.T) {
	t.Parallel()

	result := headHandledCheck("/")(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestHeadHandledFailsFor500(t *testing.T) {
	t.Parallel()

	result := headHandledCheck("/")(newAlways500Handler())
	if result.OK {
		t.Error("expected failure when HEAD returns 500")
	}
}

func TestOptionsHandledPasses(t *testing.T) {
	t.Parallel()

	result := optionsHandledCheck("/")(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestOptionsHandledFailsFor500(t *testing.T) {
	t.Parallel()

	result := optionsHandledCheck("/")(newAlways500Handler())
	if result.OK {
		t.Error("expected failure when OPTIONS returns 500")
	}
}

func TestNoLeakedInternalsPasses(t *testing.T) {
	t.Parallel()

	result := noLeakedInternalsCheck()(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestNoLeakedInternalsFailsForLeak(t *testing.T) {
	t.Parallel()

	result := noLeakedInternalsCheck()(newLeakingHandler())
	if result.OK {
		t.Error("expected failure when error response contains internal details")
	}
}

// --- Helper tests --------------------------------------------------------

func TestPassReturnsPassingResult(t *testing.T) {
	t.Parallel()

	result := Pass()
	if !result.OK {
		t.Error("Pass() should return OK=true")
	}

	if result.Message != "" {
		t.Error("Pass() should return empty message")
	}
}

func TestFailReturnsFailingResult(t *testing.T) {
	t.Parallel()

	result := Fail("something went wrong")
	if result.OK {
		t.Error("Fail() should return OK=false")
	}

	if result.Message != "something went wrong" {
		t.Errorf("Fail() message = %q, want %q", result.Message, "something went wrong")
	}
}

func TestFailFormatsMessage(t *testing.T) {
	t.Parallel()

	result := Fail("status %d for %s", http.StatusNotFound, "/")
	want := "status 404 for /"

	if result.Message != want {
		t.Errorf("Fail() message = %q, want %q", result.Message, want)
	}
}

func TestResultStringPassed(t *testing.T) {
	t.Parallel()

	if Pass().String() != "passed" {
		t.Error("Pass().String() should return 'passed'")
	}
}

func TestResultStringFailed(t *testing.T) {
	t.Parallel()

	result := Fail("broken")
	if result.String() != "broken" {
		t.Errorf("got %q, want %q", result.String(), "broken")
	}
}

func TestExpectStatusPasses(t *testing.T) {
	t.Parallel()

	check := ExpectStatus(http.MethodGet, "/", http.StatusOK)
	result := check(newGoodHandler())

	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestExpectStatusFails(t *testing.T) {
	t.Parallel()

	check := ExpectStatus(http.MethodGet, "/", http.StatusTeapot)
	result := check(newGoodHandler())

	if result.OK {
		t.Error("expected failure when status does not match")
	}
}

func TestStandardSpecsHasAllExpectedNames(t *testing.T) {
	t.Parallel()

	specs := standardSpecs(newConfig())

	expected := map[string]bool{
		SpecNameIndexNot404:           false,
		SpecNameIndexNotServerError:   false,
		SpecNameUnknownPathReturns404: false,
		SpecNameBodyHasContentType:    false,
		SpecNameHeadHandled:           false,
		SpecNameOptionsHandled:        false,
		SpecNameNoLeakedInternals:     false,
	}

	for _, s := range specs {
		if _, ok := expected[s.Name]; !ok {
			t.Errorf("unexpected spec name: %s", s.Name)
		}

		expected[s.Name] = true
	}

	for name, found := range expected {
		if !found {
			t.Errorf("expected spec %q was not found", name)
		}
	}
}

func TestLeakPatternsCoversCommonLeaks(t *testing.T) {
	t.Parallel()

	patterns := leakPatterns()
	if len(patterns) == 0 {
		t.Fatal("expected non-empty leak patterns")
	}

	body := "panic: goroutine 1 [running]:\n/usr/local/go/src/runtime/panic.go:123"
	for _, p := range patterns {
		if strings.Contains(body, p) {
			return
		}
	}

	t.Error("expected at least one pattern to match a typical panic output")
}

func TestServeReturnsRecorder(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	rec := serve(handler, mustRequest(http.MethodGet, "/"))
	if rec.Code != http.StatusTeapot {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusTeapot)
	}
}

func TestMustRequestCreatesValidRequest(t *testing.T) {
	t.Parallel()

	req := mustRequest(http.MethodGet, "/test")
	if req.Method != http.MethodGet {
		t.Errorf("got method %q, want %q", req.Method, http.MethodGet)
	}

	if req.URL.Path != "/test" {
		t.Errorf("got path %q, want %q", req.URL.Path, "/test")
	}
}
