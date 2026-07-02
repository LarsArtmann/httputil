package httpspec

import (
	"net/http"
	"slices"
	"strings"
	"testing"
)

// --- Test handlers -------------------------------------------------------

func newGoodHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
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

func newTraceEchoingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodTrace {
			w.Header().Set("Content-Type", "message/http")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("TRACE response echoing request headers"))

			return
		}

		w.WriteHeader(http.StatusNotFound)
	})
}

func newServerVersionLeakingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Server", "nginx/1.21.3")
		w.WriteHeader(http.StatusNotFound)
	})
}

func newPoweredByHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Powered-By", "Express")
		w.WriteHeader(http.StatusNotFound)
	})
}

func newRedirectWithoutLocationHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusFound)
	})

	return mux
}

// --- Run-level tests -----------------------------------------------------

func TestRunAllSpecsPassForGoodHandler(t *testing.T) {
	t.Parallel()
	Run(t, newGoodHandler())
}

func TestRunSerialAllSpecsPassForGoodHandler(t *testing.T) {
	t.Parallel()
	RunSerial(t, newGoodHandler())
}

func TestRunSerialWorksWithSkip(t *testing.T) {
	t.Parallel()
	RunSerial(
		t, newSPAHandler(),
		SkipSpec(SpecNameUnknownPathReturns404),
		SkipSpec(SpecNamePostUnknownNotServerError),
		SkipSpec(SpecNameTraceNotEnabled),
		SkipSpec(SpecNameConnectRejected),
		SkipSpec(SpecNameXContentTypeOptions),
	)
}

// --- Option tests --------------------------------------------------------

func TestSkipSpecExcludesSpec(t *testing.T) {
	t.Parallel()
	Run(
		t, newSPAHandler(),
		SkipSpec(SpecNameUnknownPathReturns404),
		SkipSpec(SpecNamePostUnknownNotServerError),
		SkipSpec(SpecNameTraceNotEnabled),
		SkipSpec(SpecNameConnectRejected),
		SkipSpec(SpecNameXContentTypeOptions),
	)
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
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("app"))
	})

	Run(t, mux, WithIndexPath("/app"))
}

// --- Individual check tests: routing -------------------------------------

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

// --- Individual check tests: methods -------------------------------------

func TestPostUnknownNotServerErrorPassesFor404(t *testing.T) {
	t.Parallel()

	result := postUnknownNotServerErrorCheck()(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestPostUnknownNotServerErrorFailsFor500(t *testing.T) {
	t.Parallel()

	result := postUnknownNotServerErrorCheck()(newAlways500Handler())
	if result.OK {
		t.Error("expected failure when POST to unknown path returns 500")
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

func TestTraceNotEnabledPassesFor404(t *testing.T) {
	t.Parallel()

	result := traceNotEnabledCheck()(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestTraceNotEnabledPassesFor405(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	result := traceNotEnabledCheck()(handler)
	if !result.OK {
		t.Errorf("expected pass for 405, got: %s", result.Message)
	}
}

func TestTraceNotEnabledFailsFor200(t *testing.T) {
	t.Parallel()

	result := traceNotEnabledCheck()(newTraceEchoingHandler())
	if result.OK {
		t.Error("expected failure when TRACE returns 200 (XST vulnerability)")
	}
}

// --- Individual check tests: headers -------------------------------------

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

func TestErrorResponsesHaveContentTypePasses(t *testing.T) {
	t.Parallel()

	result := errorResponsesHaveContentTypeCheck()(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestErrorResponsesHaveContentTypeFailsWhenMissing(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not Found"))
	})

	result := errorResponsesHaveContentTypeCheck()(handler)
	if result.OK {
		t.Error("expected failure when error response body lacks Content-Type")
	}
}

func TestErrorResponsesHaveContentTypePassesForEmptyBody(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	result := errorResponsesHaveContentTypeCheck()(handler)
	if !result.OK {
		t.Errorf("expected pass for empty error body, got: %s", result.Message)
	}
}

func TestRedirectHasLocationPasses(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/new")
		w.WriteHeader(http.StatusMovedPermanently)
	})

	result := redirectHasLocationCheck("/")(handler)
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestRedirectHasLocationPassesForNonRedirect(t *testing.T) {
	t.Parallel()

	result := redirectHasLocationCheck("/")(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass for non-redirect, got: %s", result.Message)
	}
}

func TestRedirectHasLocationFailsWhenMissing(t *testing.T) {
	t.Parallel()

	result := redirectHasLocationCheck("/")(newRedirectWithoutLocationHandler())
	if result.OK {
		t.Error("expected failure when redirect lacks Location header")
	}
}

// --- Individual check tests: security ------------------------------------

func TestNoServerVersionHeaderPasses(t *testing.T) {
	t.Parallel()

	result := noServerVersionHeaderCheck()(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestNoServerVersionHeaderPassesForBareServerName(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Server", "nginx")
		w.WriteHeader(http.StatusNotFound)
	})

	result := noServerVersionHeaderCheck()(handler)
	if !result.OK {
		t.Errorf("expected pass for bare Server name, got: %s", result.Message)
	}
}

func TestNoServerVersionHeaderFailsForVersionLeak(t *testing.T) {
	t.Parallel()

	result := noServerVersionHeaderCheck()(newServerVersionLeakingHandler())
	if result.OK {
		t.Error("expected failure when Server header leaks version")
	}
}

func TestNoPoweredByHeaderPasses(t *testing.T) {
	t.Parallel()

	result := noPoweredByHeaderCheck()(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestNoPoweredByHeaderFailsWhenPresent(t *testing.T) {
	t.Parallel()

	result := noPoweredByHeaderCheck()(newPoweredByHandler())
	if result.OK {
		t.Error("expected failure when X-Powered-By header is present")
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

// --- Builder tests -------------------------------------------------------

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

func TestExpectHeaderPasses(t *testing.T) {
	t.Parallel()

	check := ExpectHeader(http.MethodGet, "/", "Content-Type", "text/plain; charset=utf-8")
	result := check(newGoodHandler())

	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestExpectHeaderFails(t *testing.T) {
	t.Parallel()

	check := ExpectHeader(http.MethodGet, "/", "Content-Type", "application/json")
	result := check(newGoodHandler())

	if result.OK {
		t.Error("expected failure when header value does not match")
	}
}

func TestExpectHeaderAbsentPasses(t *testing.T) {
	t.Parallel()

	check := ExpectHeaderAbsent(http.MethodGet, "/", "X-Powered-By")
	result := check(newGoodHandler())

	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestExpectHeaderAbsentFails(t *testing.T) {
	t.Parallel()

	check := ExpectHeaderAbsent(http.MethodGet, "/", "X-Powered-By")
	result := check(newPoweredByHandler())

	if result.OK {
		t.Error("expected failure when header is present")
	}
}

func TestExpectBodyContainsPasses(t *testing.T) {
	t.Parallel()

	check := ExpectBodyContains(http.MethodGet, "/", "hello")
	result := check(newGoodHandler())

	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestExpectBodyContainsFails(t *testing.T) {
	t.Parallel()

	check := ExpectBodyContains(http.MethodGet, "/", "goodbye")
	result := check(newGoodHandler())

	if result.OK {
		t.Error("expected failure when body does not contain substring")
	}
}

// --- Internal helper tests -----------------------------------------------

func TestStandardSpecsHasAllExpectedNames(t *testing.T) {
	t.Parallel()

	specs := standardSpecs(newConfig())

	expected := map[string]bool{
		SpecNameIndexNot404:                   false,
		SpecNameIndexNotServerError:           false,
		SpecNameUnknownPathReturns404:         false,
		SpecNamePostUnknownNotServerError:     false,
		SpecNameBodyHasContentType:            false,
		SpecNameErrorResponsesHaveContentType: false,
		SpecNameHeadHandled:                   false,
		SpecNameOptionsHandled:                false,
		SpecNameTraceNotEnabled:               false,
		SpecNameRedirectHasLocation:           false,
		SpecNameNoServerVersionHeader:         false,
		SpecNameNoPoweredByHeader:             false,
		SpecNameNoLeakedInternals:             false,
		SpecNameXContentTypeOptions:           false,
		SpecNameNoDuplicateHeaders:            false,
		SpecNameConnectRejected:               false,
		SpecNameRespectsAcceptHeader:          false,
		SpecNameLongURLHandled:                false,
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
	found := slices.ContainsFunc(patterns, func(p string) bool {
		return strings.Contains(body, p)
	})

	if !found {
		t.Error("expected at least one pattern to match a typical panic output")
	}
}

func TestHasVersionLeakDetectsVersionPattern(t *testing.T) {
	t.Parallel()

	cases := []struct {
		server string
		leak   bool
	}{
		{"nginx/1.21.3", true},
		{"Apache/2.4.41 (Ubuntu)", true},
		{"nginx", false},
		{"", false},
		{"MyServer", false},
	}

	for _, tc := range cases {
		if got := hasVersionLeak(tc.server); got != tc.leak {
			t.Errorf("hasVersionLeak(%q) = %v, want %v", tc.server, got, tc.leak)
		}
	}
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

// --- Test handlers for new specs ----------------------------------------

func newNoSniffHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	return mux
}

func newDuplicateHeaderHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header()["Set-Cookie"] = []string{"a=1", "b=2"}
		w.WriteHeader(http.StatusOK)
	})
}

func newConnectAllowingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			w.WriteHeader(http.StatusOK)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	})
}

func newLongURLCrashingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) > 1000 {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	})
}

// --- ExpectNotStatus tests ----------------------------------------------

func TestExpectNotStatusPasses(t *testing.T) {
	t.Parallel()

	check := ExpectNotStatus(http.MethodGet, "/", http.StatusInternalServerError)
	result := check(newGoodHandler())

	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestExpectNotStatusFails(t *testing.T) {
	t.Parallel()

	check := ExpectNotStatus(http.MethodGet, "/", http.StatusOK)
	result := check(newGoodHandler())

	if result.OK {
		t.Error("expected failure when status matches the excluded code")
	}
}

// --- X-Content-Type-Options tests ---------------------------------------

func TestXContentTypeOptionsPasses(t *testing.T) {
	t.Parallel()

	result := xContentTypeOptionsCheck("/")(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestXContentTypeOptionsFailsWhenMissing(t *testing.T) {
	t.Parallel()

	result := xContentTypeOptionsCheck("/")(newNoSniffHandler())
	if result.OK {
		t.Error("expected failure when X-Content-Type-Options is missing")
	}
}

// --- No duplicate headers tests -----------------------------------------

func TestNoDuplicateHeadersPasses(t *testing.T) {
	t.Parallel()

	result := noDuplicateHeadersCheck("/")(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestNoDuplicateHeadersFailsWhenDuplicates(t *testing.T) {
	t.Parallel()

	result := noDuplicateHeadersCheck("/")(newDuplicateHeaderHandler())
	if result.OK {
		t.Error("expected failure when response has duplicate headers")
	}
}

// --- CONNECT method tests -----------------------------------------------

func TestConnectRejectedPassesFor404(t *testing.T) {
	t.Parallel()

	result := connectRejectedCheck()(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestConnectRejectedPassesFor405(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	result := connectRejectedCheck()(handler)
	if !result.OK {
		t.Errorf("expected pass for 405, got: %s", result.Message)
	}
}

func TestConnectRejectedFailsFor200(t *testing.T) {
	t.Parallel()

	result := connectRejectedCheck()(newConnectAllowingHandler())
	if result.OK {
		t.Error("expected failure when CONNECT returns 200")
	}
}

// --- Accept header tests ------------------------------------------------

func TestRespectsAcceptHeaderPasses(t *testing.T) {
	t.Parallel()

	result := respectsAcceptHeaderCheck("/")(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestRespectsAcceptHeaderFailsFor500(t *testing.T) {
	t.Parallel()

	result := respectsAcceptHeaderCheck("/")(newAlways500Handler())
	if result.OK {
		t.Error("expected failure when Accept header causes 500")
	}
}

// --- Long URL tests -----------------------------------------------------

func TestLongURLHandledPasses(t *testing.T) {
	t.Parallel()

	result := longURLHandledCheck()(newGoodHandler())
	if !result.OK {
		t.Errorf("expected pass, got: %s", result.Message)
	}
}

func TestLongURLHandledFailsFor500(t *testing.T) {
	t.Parallel()

	result := longURLHandledCheck()(newLongURLCrashingHandler())
	if result.OK {
		t.Error("expected failure when long URL causes 500")
	}
}
