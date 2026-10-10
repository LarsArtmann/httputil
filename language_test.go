package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func assertLanguageResponse(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	wantContentLanguage, wantVaryContains string,
) {
	t.Helper()

	if got := rec.Header().Get("Content-Language"); got != wantContentLanguage {
		t.Errorf("Content-Language = %q, want %q", got, wantContentLanguage)
	}

	if vary := rec.Header().Get("Vary"); !strings.Contains(vary, wantVaryContains) {
		t.Errorf("Vary = %q, want it to contain %q", vary, wantVaryContains)
	}
}

func TestLanguage_SecondHeaderCandidateServedWhenFirstUnsupported(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{
		SupportedTags: []string{"en"},
		Extractors:    []LanguageExtractor{LanguageExtractorFromAcceptHeader()},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept-Language", "de-AT,de;q=0.9,en;q=0.8")

	Language(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	assertLanguageResponse(t, rec, "en", "Accept-Language")
}

func TestLanguage_PathLegBeatsHeader(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{
		SupportedTags: []string{"de", "en"},
		Extractors: []LanguageExtractor{LanguageExtractorChain(
			LanguageExtractorFromPathPrefix(map[string]string{"/en": "en"}),
			LanguageExtractorFromAcceptHeader(),
		)},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/en/quote", http.NoBody)
	req.Header.Set("Accept-Language", "de")

	Language(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := LanguageFromContext(r.Context()); got != "en" {
			t.Errorf("negotiated tag = %q, want %q (path leg must beat header)", got, "en")
		}
	})).ServeHTTP(rec, req)
}

func TestLanguage_CookieLegBeatsHeader(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{
		SupportedTags: []string{"de", "en"},
		Extractors: []LanguageExtractor{LanguageExtractorChain(
			LanguageExtractorFromCookie("lang"),
			LanguageExtractorFromAcceptHeader(),
		)},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "lang", Value: "en"})
	req.Header.Set("Accept-Language", "de")

	Language(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := LanguageFromContext(r.Context()); got != "en" {
			t.Errorf("negotiated tag = %q, want %q (cookie leg must beat header)", got, "en")
		}
	})).ServeHTTP(rec, req)
}

func TestLanguage_QueryLegExtractsParam(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{
		SupportedTags: []string{"de", "en"},
		Extractors:    []LanguageExtractor{LanguageExtractorFromQuery("lang")},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?lang=en", http.NoBody)

	Language(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := LanguageFromContext(r.Context()); got != "en" {
			t.Errorf("negotiated tag = %q, want %q", got, "en")
		}
	})).ServeHTTP(rec, req)
}

func TestLanguage_NoMatchFallsBackToDefaultTag(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{
		SupportedTags: []string{"de"},
		Extractors:    []LanguageExtractor{LanguageExtractorFromAcceptHeader()},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept-Language", "fr-CH,fr;q=0.9")

	Language(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := LanguageFromContext(r.Context()); got != "de" {
			t.Errorf("negotiated tag = %q, want default %q", got, "de")
		}
	})).ServeHTTP(rec, req)
}

func TestLanguage_EmptyExtractorsServeDefaultTag(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{SupportedTags: []string{"de", "en"}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept-Language", "en")

	Language(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := LanguageFromContext(r.Context()); got != "de" {
			t.Errorf("negotiated tag = %q, want %q (nil chain = default tag)", got, "de")
		}
	})).ServeHTTP(rec, req)
}

func TestLanguage_NilExtractorEntriesAreDropped(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{
		SupportedTags: []string{"de", "en"},
		Extractors:    []LanguageExtractor{nil, LanguageExtractorFromAcceptHeader()},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept-Language", "en")

	Language(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := LanguageFromContext(r.Context()); got != "en" {
			t.Errorf("negotiated tag = %q, want %q (nil entry must not poison the chain)", got, "en")
		}
	})).ServeHTTP(rec, req)
}

func TestLanguage_ZeroValueConfigFallsBackToDefault(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept-Language", "en")

	Language(LanguageConfig{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := LanguageFromContext(r.Context()); got != "en" {
			t.Errorf("negotiated tag = %q, want %q (zero config must fall back to DefaultLanguageConfig)", got, "en")
		}
	})).ServeHTTP(rec, req)

	assertLanguageResponse(t, rec, "en", "Accept-Language")
}

func TestLanguage_ZeroValueDisableFlagsKeepHeadersOn(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{SupportedTags: []string{"de"}}

	rec := httptest.NewRecorder()
	Language(
		cfg,
	)(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	assertLanguageResponse(t, rec, "de", "Accept-Language")
}

func TestLanguage_DisableFlagsTurnHeadersOff(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{
		SupportedTags:          []string{"de"},
		DisableContentLanguage: true,
		DisableVary:            true,
	}

	rec := httptest.NewRecorder()
	Language(
		cfg,
	)(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	if got := rec.Header().Get("Content-Language"); got != "" {
		t.Errorf("Content-Language = %q, want empty (DisableContentLanguage)", got)
	}

	if got := rec.Header().Get("Vary"); got != "" {
		t.Errorf("Vary = %q, want empty (DisableVary)", got)
	}
}

func TestLanguage_VaryNotDuplicatedWhenOuterMiddlewareAlreadyAddedIt(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{SupportedTags: []string{"de"}}

	outer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Vary", "Accept-Language")

			next.ServeHTTP(w, r)
		})
	}

	rec := httptest.NewRecorder()
	mw := Language(cfg)
	outer(
		mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})),
	).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	vary := rec.Header().Values("Vary")

	if len(vary) != 1 {
		t.Fatalf("Vary values = %v, want one value (duplicate token must be skipped)", vary)
	}
}

func TestLanguage_VaryComposesWithCompressionStyleAdd(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{SupportedTags: []string{"de"}}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Vary", "Accept-Encoding")
	})

	rec := httptest.NewRecorder()
	Language(cfg)(handler).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	vary := strings.Join(rec.Header().Values("Vary"), ", ")
	if !strings.Contains(vary, "Accept-Encoding") || !strings.Contains(vary, "Accept-Language") {
		t.Errorf("Vary values = %q, want both Accept-Encoding and Accept-Language", vary)
	}
}

func TestLanguage_SupportedTagsKeepDeclaredSpelling(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{
		SupportedTags: []string{"zh-Hans", "en"},
		Extractors:    []LanguageExtractor{LanguageExtractorFromAcceptHeader()},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept-Language", "ZH-hans")

	Language(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := LanguageFromContext(r.Context()); got != "zh-Hans" {
			t.Errorf(
				"negotiated tag = %q, want declared spelling %q (matching is case-insensitive, serving is not)",
				got,
				"zh-Hans",
			)
		}
	})).ServeHTTP(rec, req)
}

func TestLanguage_CustomMatcherPluginIsUsed(t *testing.T) {
	t.Parallel()

	matcher := func(tag string) (string, bool) {
		if strings.HasPrefix(tag, "zh") {
			return "zh-Hans", true
		}

		return "", false
	}

	cfg := LanguageConfig{
		SupportedTags: []string{"zh-Hans"},
		Extractors:    []LanguageExtractor{LanguageExtractorFromAcceptHeader()},
		Matcher:       matcher,
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept-Language", "zh-Hant,zh;q=0.9")

	Language(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := LanguageFromContext(r.Context()); got != "zh-Hans" {
			t.Errorf("negotiated tag = %q, want %q (custom matcher must decide)", got, "zh-Hans")
		}
	})).ServeHTTP(rec, req)
}

func TestLanguage_ContextIsPerRequest(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{
		SupportedTags: []string{"de", "en"},
		Extractors:    []LanguageExtractor{LanguageExtractorFromAcceptHeader()},
	}

	mw := Language(cfg)

	deReq := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	deReq.Header.Set("Accept-Language", "de")
	enReq := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	enReq.Header.Set("Accept-Language", "en")

	var got []string

	collect := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = append(got, LanguageFromContext(r.Context()))
	})

	done := make(chan struct{})

	go func() {
		defer close(done)

		mw(collect).ServeHTTP(httptest.NewRecorder(), deReq)
		mw(collect).ServeHTTP(httptest.NewRecorder(), enReq)
	}()

	<-done

	if len(got) != 2 || got[0] != "de" || got[1] != "en" {
		t.Errorf("per-request tags = %v, want [de en]", got)
	}
}

func TestLanguageFromContext_ReturnsEmptyWithoutMiddleware(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	if got := LanguageFromRequest(req); got != "" {
		t.Errorf("LanguageFromRequest = %q, want empty string without middleware", got)
	}
}

func TestWithLanguage_RoundTripsThroughContext(t *testing.T) {
	t.Parallel()

	ctx := WithLanguage(context.Background(), "fr")

	if got := LanguageFromContext(ctx); got != "fr" {
		t.Errorf("LanguageFromContext = %q, want %q", got, "fr")
	}
}

func TestLanguageExtractorFromPathPrefix_MatchesExactPrefixAndSlashContinuation(t *testing.T) {
	t.Parallel()

	extractor := LanguageExtractorFromPathPrefix(map[string]string{"/en": "en"})

	req := httptest.NewRequest(http.MethodGet, "/en", http.NoBody)
	if tags, ok := extractor(req); !ok || tags[0] != "en" {
		t.Errorf("extractor(/en) = (%v, %v), want ([en], true)", tags, ok)
	}
}

func TestLanguageExtractorFromPathPrefix_MatchesPrefixSlashPath(t *testing.T) {
	t.Parallel()

	extractor := LanguageExtractorFromPathPrefix(map[string]string{"/en": "en"})

	req := httptest.NewRequest(http.MethodGet, "/en/quote", http.NoBody)
	if tags, ok := extractor(req); !ok || tags[0] != "en" {
		t.Errorf("extractor(/en/quote) = (%v, %v), want ([en], true)", tags, ok)
	}
}

func TestLanguageExtractorFromPathPrefix_RejectsSharedPrefixWords(t *testing.T) {
	t.Parallel()

	extractor := LanguageExtractorFromPathPrefix(map[string]string{"/en": "en"})

	req := httptest.NewRequest(http.MethodGet, "/english", http.NoBody)
	if _, ok := extractor(req); ok {
		t.Error("extractor(/english) must not match the /en prefix")
	}
}

func TestLanguageExtractorFromPathPrefix_IgnoresUnrelatedPaths(t *testing.T) {
	t.Parallel()

	extractor := LanguageExtractorFromPathPrefix(map[string]string{"/en": "en"})

	for _, path := range []string{"/de", "/"} {
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		if _, ok := extractor(req); ok {
			t.Errorf("extractor(%q) must not match", path)
		}
	}
}

func TestLanguageExtractorFromAcceptHeader_EmptyHeaderYieldsNothing(t *testing.T) {
	t.Parallel()

	extractor := LanguageExtractorFromAcceptHeader()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	if tags, ok := extractor(req); ok || tags != nil {
		t.Errorf("empty Accept-Language = (%v, %v), want (nil, false)", tags, ok)
	}
}

func TestLanguageExtractorChain_FirstLegWithCandidatesWins(t *testing.T) {
	t.Parallel()

	headerLeg := LanguageExtractorFromAcceptHeader()
	chain := LanguageExtractorChain(
		func(_ *http.Request) ([]string, bool) { return nil, false },
		func(_ *http.Request) ([]string, bool) { return []string{"it"}, true },
		headerLeg,
	)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept-Language", "de")

	tags, ok := chain(req)
	if !ok || len(tags) != 1 || tags[0] != "it" {
		t.Errorf("chain = (%v, %v), want ([it], true) — second leg must win, header must not run", tags, ok)
	}
}

func TestLanguageExtractorChain_NilLegsAreSkipped(t *testing.T) {
	t.Parallel()

	chain := LanguageExtractorChain(nil, LanguageExtractorFromAcceptHeader())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept-Language", "de")

	tags, ok := chain(req)
	if !ok || len(tags) != 1 || tags[0] != "de" {
		t.Errorf("chain = (%v, %v), want ([de], true) — nil leg must not poison the chain", tags, ok)
	}
}

func TestParseAcceptLanguageCandidates_OrdersByWeightThenListedOrder(t *testing.T) {
	t.Parallel()

	tags, ok := parseAcceptLanguageCandidates("en;q=0.5,de;q=0.9,fr")
	if !ok {
		t.Fatal("parseAcceptLanguageCandidates reported no candidates for a valid header")
	}

	want := []string{"fr", "de", "en"}

	if strings.Join(tags, ",") != strings.Join(want, ",") {
		t.Errorf("candidates = %v, want %v (weight desc — fr defaults to q=1 — ties in listed order)", tags, want)
	}
}

func TestParseAcceptLanguageCandidates_TieKeepsListedOrder(t *testing.T) {
	t.Parallel()

	tags, ok := parseAcceptLanguageCandidates("en,de")
	if !ok {
		t.Fatal("parseAcceptLanguageCandidates reported no candidates for a valid header")
	}

	if tags[0] != "en" {
		t.Errorf("tie order = %v, want en first (listed order)", tags)
	}
}

func TestParseAcceptLanguageCandidates_DropsQZeroAndWildcard(t *testing.T) {
	t.Parallel()

	tags, ok := parseAcceptLanguageCandidates("de;q=0,*,en;q=1.0")
	if !ok {
		t.Fatal("parseAcceptLanguageCandidates reported no candidates")
	}

	if strings.Join(tags, ",") != "en" {
		t.Errorf("candidates = %v, want [en] (q=0 and * must be dropped)", tags)
	}
}

func TestParseAcceptLanguageCandidates_MalformedQKeepsDefaultWeight(t *testing.T) {
	t.Parallel()

	tags, ok := parseAcceptLanguageCandidates("de;q=abc,en")
	if !ok {
		t.Fatal("parseAcceptLanguageCandidates reported no candidates")
	}

	if strings.Join(tags, ",") != "de,en" {
		t.Errorf("candidates = %v, want [de en] (malformed q keeps default weight, listed order)", tags)
	}
}

func TestParseAcceptLanguageCandidates_GarbageYieldsNothing(t *testing.T) {
	t.Parallel()

	if tags, ok := parseAcceptLanguageCandidates(";;;"); ok || tags != nil {
		t.Errorf("garbage header = (%v, %v), want (nil, false)", tags, ok)
	}
}

func TestParseAcceptLanguageCandidates_FourDecimalsTruncatedToThree(t *testing.T) {
	t.Parallel()

	tags, ok := parseAcceptLanguageCandidates("de;q=0.5000,en;q=0.4000")
	if !ok {
		t.Fatal("parseAcceptLanguageCandidates reported no candidates")
	}

	if strings.Join(tags, ",") != "de,en" {
		t.Errorf("candidates = %v, want [de en] (RFC 7231 caps q at 3 decimals)", tags)
	}
}

func TestLanguageMatcher_ExactBeatsPrimary(t *testing.T) {
	t.Parallel()

	matcher := buildLanguageMatcher([]string{"de", "de-AT", "en"})

	supported, ok := matcher.match("de-AT")
	if !ok || supported != "de-AT" {
		t.Errorf("match(de-AT) = (%q, %v), want (de-AT, true) — exact match must beat primary", supported, ok)
	}
}

func TestLanguageMatcher_PrimarySubtagMatchesRegionalCandidate(t *testing.T) {
	t.Parallel()

	matcher := buildLanguageMatcher([]string{"de", "en"})

	supported, ok := matcher.match("de-AT")
	if !ok || supported != "de" {
		t.Errorf("match(de-AT) = (%q, %v), want (de, true)", supported, ok)
	}
}

func TestLanguageMatcher_UnknownLanguageFails(t *testing.T) {
	t.Parallel()

	matcher := buildLanguageMatcher([]string{"de", "en"})

	if _, ok := matcher.match("pt-BR"); ok {
		t.Error("match(pt-BR) must not match a {de,en} set")
	}
}

func TestLanguageMatcher_UnsupportedCharsetEntriesAreDropped(t *testing.T) {
	t.Parallel()

	if matcher := buildLanguageMatcher([]string{"de\nda"}); matcher != nil {
		t.Error("matcher must be nil when every entry fails the charset guard")
	}
}

func TestLanguageConfig_ValidateRejectsEmptyTags(t *testing.T) {
	t.Parallel()

	err := LanguageConfig{SupportedTags: nil}.Validate()
	if err == nil || !InDomain(err, Domain("language")) {
		t.Errorf("Validate() = %v, want a language.* Rejection", err)
	}
}

func TestLanguageConfig_ValidateRejectsInvalidTagCharset(t *testing.T) {
	t.Parallel()

	err := LanguageConfig{SupportedTags: []string{"de,en"}}.Validate()
	if err == nil || !InDomain(err, Domain("language")) {
		t.Errorf("Validate() = %v, want a language.* Rejection (comma is not a subtag character)", err)
	}
}

func TestLanguageConfig_ValidateRejectsUnsupportedDefaultTag(t *testing.T) {
	t.Parallel()

	err := LanguageConfig{SupportedTags: []string{"de"}, DefaultTag: "fr"}.Validate()
	if err == nil || !InDomain(err, Domain("language")) {
		t.Errorf("Validate() = %v, want a language.* Rejection", err)
	}
}

func TestLanguageConfig_ValidateRejectsNilExtractor(t *testing.T) {
	t.Parallel()

	err := LanguageConfig{
		SupportedTags: []string{"de"},
		Extractors:    []LanguageExtractor{nil},
	}.Validate()
	if err == nil || !InDomain(err, Domain("language")) {
		t.Errorf("Validate() = %v, want a language.* Rejection", err)
	}
}

func TestLanguageConfig_ValidateAcceptsDefaultConfig(t *testing.T) {
	t.Parallel()

	if err := DefaultLanguageConfig().Validate(); err != nil {
		t.Errorf("Validate(DefaultLanguageConfig()) = %v, want nil", err)
	}
}

func TestLanguage_EmptyTagsConfigServesDefaultConfigLanguage(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	Language(
		LanguageConfig{SupportedTags: nil},
	)(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	assertLanguageResponse(t, rec, "en", "Accept-Language")
}

func TestLanguage_AllInvalidTagsFallBackToDefaultConfig(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	Language(
		LanguageConfig{SupportedTags: []string{"a,b", "c;d"}},
	)(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	assertLanguageResponse(t, rec, "en", "Accept-Language")
}

func TestLanguage_UnsupportedDefaultTagFallsBackToFirst(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{
		SupportedTags: []string{"de", "en"},
		DefaultTag:    "fr",
	}

	rec := httptest.NewRecorder()
	Language(
		cfg,
	)(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	if got := rec.Header().Get("Content-Language"); got != "de" {
		t.Errorf("Content-Language = %q, want %q (unsupported default falls back to first supported)", got, "de")
	}
}

func TestLanguage_ConcurrentUseUnderRace(t *testing.T) {
	t.Parallel()

	cfg := LanguageConfig{
		SupportedTags: []string{"de", "en"},
		Extractors:    []LanguageExtractor{LanguageExtractorFromAcceptHeader()},
	}

	mw := Language(cfg)
	handler := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	done := make(chan struct{})

	for range 8 {
		go func() {
			defer func() { done <- struct{}{} }()

			for range 50 {
				req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
				req.Header.Set("Accept-Language", "de,en;q=0.5")
				handler.ServeHTTP(httptest.NewRecorder(), req)
			}
		}()
	}

	for range 8 {
		<-done
	}
}
