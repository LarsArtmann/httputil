package httputil

import (
	"context"
	"net/http"
	"strings"
)

// HTTP language negotiation.
//
// Language() resolves the language a request should be served in, exposes
// it to handlers via context, and stamps the response with the headers
// caches need to keep distinct languages distinct:
// Content-Language (what the client got) and Vary: Accept-Language (what
// the answer depends on).
//
// Typical usage (the sperrmuell-direct funnel shape — German canonical
// unprefixed, English under /en):
//
//	cfg := httputil.DefaultLanguageConfig()
//	cfg.SupportedTags = []string{"de", "en"}
//	cfg.Extractors = []httputil.LanguageExtractor{
//		httputil.LanguageExtractorChain(
//			httputil.LanguageExtractorFromPathPrefix(map[string]string{"/en": "en"}),
//			httputil.LanguageExtractorFromAcceptHeader(),
//		),
//	}
//	stack.Add(httputil.MiddlewareLanguage, httputil.Language(cfg))
//
//	// In a handler:
//	lang := httputil.LanguageFromRequest(r) // "de" | "en"
//
// Detection order is the app's decision: the chain tries legs in order and
// the first leg with candidates decides the negotiation input (path
// beating header is the SEO single-home pattern). Message catalogs,
// plurals, and formatting are out of scope — pair this middleware with
// go-i18n or golang.org/x/text (see docs/integrations/).

// LanguageExtractor extracts language-tag candidates from a request in
// preference order. An empty slice or ok=false means "no opinion" and the
// chain moves to the next extractor.
//
// The list shape (rather than a single tag) is what makes Accept-Language
// negotiation correct: the top candidate may not match the supported set
// while a lower one does ("de-AT,de;q=0.9,en;q=0.8" against supported
// {en} must serve en).
type LanguageExtractor func(r *http.Request) (tags []string, ok bool)

// language tag charset: subtag characters only. The served tag is written
// into the Content-Language response header, so the supported set is a
// header-injection surface and is validated like one.
const languageTagChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"

// Error codes for the language domain, all Rejection: invalid config is
// unacceptable input — fix the config, never retry.
const (
	codeLanguageTagsEmpty          = Code("language.tags_empty")
	codeLanguageTagInvalid         = Code("language.tag_invalid")
	codeLanguageDefaultUnsupported = Code("language.default_tag_unsupported")
	codeLanguageExtractorNil       = Code("language.extractor_nil")
)

var (
	errLanguageTagsEmpty = codeLanguageTagsEmpty.Rejection(
		"LanguageConfig.SupportedTags must list at least one language tag (priority order, first is default)",
	)
	errLanguageTagInvalid = codeLanguageTagInvalid.Rejection(
		"LanguageConfig tags must be non-empty BCP 47 subtags [a-zA-Z0-9-] (they are written into Content-Language)",
	)
	errLanguageDefaultUnsupported = codeLanguageDefaultUnsupported.Rejection(
		"LanguageConfig.DefaultTag must be empty (use the first supported tag) or one of SupportedTags",
	)
	errLanguageExtractorNil = codeLanguageExtractorNil.Rejection(
		"LanguageConfig.Extractors must not contain nil entries",
	)
)

// TagMatcher resolves one candidate tag against the app's supported set
// and returns the supported tag to serve. Return ok=false to pass — the
// middleware then tries the next candidate and finally the default tag.
// The built-in matcher is exact-then-primary-subtag; supply an adapter for
// full BCP 47 / CLDR matching (golang.org/x/text's language.Matcher, see
// docs/integrations/x-text.md). Matchers must be safe for concurrent use.
type TagMatcher func(tag string) (supported string, ok bool)

// LanguageConfig configures the Language middleware.
type LanguageConfig struct {
	// SupportedTags lists the canonical language tags the app serves, in
	// priority order — the first entry is the default when nothing matches.
	// Tags are canonicalized (trimmed, lowercased) and must be non-empty
	// subtags of [a-zA-Z0-9-]. Required.
	SupportedTags []string

	// DefaultTag is served when no extractor yields a matching candidate.
	// Empty means SupportedTags[0].
	DefaultTag string

	// Extractors is the detection chain, tried in order; the first leg
	// with candidates decides the negotiation input. Nil or empty means no
	// extraction at all: every request negotiates to the default tag
	// (valid for single-language sites that still want Content-Language
	// and Vary stamped).
	Extractors []LanguageExtractor

	// Matcher resolves a candidate tag against the supported set and
	// returns the supported tag to serve. Nil uses the built-in matcher:
	// exact case-insensitive match first, then primary-subtag match
	// ("de-AT" serves "de"). Supply a TagMatcher adapter for full BCP 47 /
	// CLDR matching — see docs/integrations/x-text.md.
	Matcher TagMatcher

	// DisableContentLanguage turns off the Content-Language response
	// header. The zero value keeps it on — the safe default.
	DisableContentLanguage bool

	// DisableVary turns off the Vary: Accept-Language response header.
	// The zero value keeps it on (RFC-strict: the response varies with the
	// request header even under cookie/query extraction legs).
	DisableVary bool
}

// DefaultLanguageConfig returns a LanguageConfig with a working baseline:
// English only, Accept-Language extraction, both response headers on.
// Override SupportedTags and Extractors for real deployments.
func DefaultLanguageConfig() LanguageConfig {
	return LanguageConfig{
		SupportedTags:          []string{"en"},
		DefaultTag:             "",
		Extractors:             []LanguageExtractor{LanguageExtractorFromAcceptHeader()},
		Matcher:                nil,
		DisableContentLanguage: false,
		DisableVary:            false,
	}
}

// Validate checks the LanguageConfig for invalid values. The constructor
// ([Language]) calls Validate after applying defaults, following the
// validate-and-log pattern: invalid configs are logged and the middleware
// falls back to safe values instead of aborting construction.
func (c LanguageConfig) Validate() error {
	if len(c.SupportedTags) == 0 {
		return errLanguageTagsEmpty
	}

	for _, tag := range c.SupportedTags {
		if !validLanguageTag(tag) {
			return errLanguageTagInvalid.WithContextAny("tag", tag)
		}
	}

	if c.DefaultTag != "" && !c.supports(c.DefaultTag) {
		return errLanguageDefaultUnsupported.WithContextAny("default_tag", c.DefaultTag)
	}

	for _, extractor := range c.Extractors {
		if extractor == nil {
			return errLanguageExtractorNil
		}
	}

	return nil
}

// supports reports whether tag is a member of the supported set
// (case-insensitive exact comparison).
func (c LanguageConfig) supports(tag string) bool {
	for _, supported := range c.SupportedTags {
		if strings.EqualFold(supported, tag) {
			return true
		}
	}

	return false
}

// validLanguageTag reports whether tag is a non-empty string of subtag
// characters. It is an injection guard for the Content-Language header
// value, not a full BCP 47 grammar check.
func validLanguageTag(tag string) bool {
	if tag == "" {
		return false
	}

	for i := range len(tag) {
		if !strings.ContainsRune(languageTagChars, rune(tag[i])) {
			return false
		}
	}

	return true
}

// languageMatcher resolves candidate tags against the supported set: exact
// case-insensitive match first, then primary-subtag match in supported
// priority order. Precompiled at construction (compression-negotiator
// pattern) so the per-request path is table lookups only.
type languageMatcher struct {
	// supported are the canonical tags in priority order.
	supported []string
	// primaries maps each tag's primary subtag to the first supported tag
	// carrying it (e.g. "de" → "de-AT" when supported is ["de-AT", "en"]).
	primaries map[string]string
}

// buildLanguageMatcher canonicalizes the supported set and precompiles the
// primary-subtab table. Entries failing the charset guard are dropped;
// a nil matcher is only returned when nothing survived (the constructor
// then falls back to DefaultLanguageConfig).
func buildLanguageMatcher(tags []string) *languageMatcher {
	supported := make([]string, 0, len(tags))
	seen := make(map[string]bool, len(tags))
	primaries := make(map[string]string, len(tags))

	for _, tag := range tags {
		canonical := strings.ToLower(trim(tag))
		if !validLanguageTag(canonical) || seen[canonical] {
			continue
		}

		seen[canonical] = true
		supported = append(supported, canonical)

		primary, _, _ := strings.Cut(canonical, "-")
		if _, exists := primaries[primary]; !exists {
			primaries[primary] = canonical
		}
	}

	if len(supported) == 0 {
		return nil
	}

	return &languageMatcher{supported: supported, primaries: primaries}
}

// match resolves one candidate: exact match first, then primary subtag.
func (m *languageMatcher) match(tag string) (string, bool) {
	candidate := strings.ToLower(trim(tag))

	for _, supported := range m.supported {
		if supported == candidate {
			return supported, true
		}
	}

	primary, _, _ := strings.Cut(candidate, "-")
	supported, ok := m.primaries[primary]

	return supported, ok
}

// LanguageExtractorChain bundles extractors into one: legs run in order
// and the first leg that yields candidates decides the negotiation input.
// Nil legs are skipped, so a chain never panics on a dropped extractor.
func LanguageExtractorChain(extractors ...LanguageExtractor) LanguageExtractor {
	return func(r *http.Request) ([]string, bool) {
		for _, extractor := range extractors {
			if extractor == nil {
				continue
			}

			if tags, ok := extractor(r); ok && len(tags) > 0 {
				return tags, true
			}
		}

		return nil, false
	}
}

// LanguageExtractorFromAcceptHeader parses the Accept-Language header with
// the RFC 7231 q-value grammar (the same allocation-free parser the
// compression negotiator uses) and yields the header's candidates in
// preference order: q descending, ties in listed order, q=0 excluded, "*"
// yielding nothing (the default-tag fallback covers it). A missing header
// yields no candidates.
func LanguageExtractorFromAcceptHeader() LanguageExtractor {
	return func(r *http.Request) ([]string, bool) {
		return parseAcceptLanguageCandidates(r.Header.Get(headerAcceptLanguage))
	}
}

type languagePreference struct {
	tag    string
	weight float64
	order  int
}

// parseAcceptLanguageCandidates splits an Accept-Language header into
// ordered tag candidates. Malformed q-values keep the tag at default
// weight (the compression parser's lenient posture); entries parse
// per-entry — one broken entry never poisons the rest.
func parseAcceptLanguageCandidates(header string) ([]string, bool) {
	if trim(header) == "" {
		return nil, false
	}

	entries := strings.Split(header, ",")
	prefs := make([]languagePreference, 0, len(entries))

	for i, entry := range entries {
		if pref, ok := parseAcceptLanguageEntry(entry, i); ok {
			prefs = append(prefs, pref)
		}
	}

	if len(prefs) == 0 {
		return nil, false
	}

	// Insertion sort by weight descending; equal weights keep listed order
	// (the tie-break clause keeps the sort stable without relying on
	// sort.SliceStable's guarantees for tiny n). n is small (header entries).
	for i := 1; i < len(prefs); i++ {
		for j := i; j > 0 && prefs[j].weight > prefs[j-1].weight; j-- {
			prefs[j-1], prefs[j] = prefs[j], prefs[j-1]
		}
	}

	tags := make([]string, 0, len(prefs))
	for _, p := range prefs {
		tags = append(tags, p.tag)
	}

	return tags, true
}

// parseAcceptLanguageEntry parses one comma-separated Accept-Language
// entry ("de" or "de;q=0.8") into its preference record. Wildcards and
// q=0 entries yield nothing; malformed q-values keep default weight.
func parseAcceptLanguageEntry(entry string, order int) (languagePreference, bool) {
	tag, params, _ := strings.Cut(entry, ";")
	tag = trim(tag)

	if tag == "" || tag == "*" {
		return languagePreference{}, false
	}

	weight := defaultQValue
	if qPart, found := strings.CutPrefix(trim(params), qValuePrefix); found {
		if parsed, err := parseQValue(qPart); err == nil {
			weight = parsed
		}
	}

	if weight <= 0 {
		return languagePreference{}, false
	}

	return languagePreference{tag: tag, weight: weight, order: order}, true
}

// LanguageExtractorFromQuery yields the tag in the given URL query
// parameter (e.g. "lang"), when present and non-empty.
func LanguageExtractorFromQuery(param string) LanguageExtractor {
	return func(r *http.Request) ([]string, bool) {
		tag := trim(r.URL.Query().Get(param))
		if tag == "" {
			return nil, false
		}

		return []string{tag}, true
	}
}

// LanguageExtractorFromCookie yields the tag stored in the named cookie,
// when present and non-empty. Cookie reading can fail per-cookie; a failed
// read is "no opinion", not an error — the chain continues.
func LanguageExtractorFromCookie(name string) LanguageExtractor {
	return func(r *http.Request) ([]string, bool) {
		cookie, err := r.Cookie(name)
		if err != nil || trim(cookie.Value) == "" {
			return nil, false
		}

		return []string{cookie.Value}, true
	}
}

// LanguageExtractorFromPathPrefix matches the request path against the
// given prefix→tag pairs ("/en" → "en") and yields the first pair whose
// prefix the path starts with. Match exactly the prefix or prefix+"/" so
// "/english" does not match "/en". The path leg is how SEO single-home
// funnels pin the journey regardless of headers.
func LanguageExtractorFromPathPrefix(prefixes map[string]string) LanguageExtractor {
	return func(r *http.Request) ([]string, bool) {
		for prefix, tag := range prefixes {
			prefix = trim(prefix)

			rest, found := strings.CutPrefix(r.URL.Path, prefix)
			if !found || (rest != "" && !strings.HasPrefix(rest, "/")) {
				continue
			}

			if tag = trim(tag); tag != "" {
				return []string{tag}, true
			}
		}

		return nil, false
	}
}

// languageKey is the context key for the negotiated language tag.
type languageKey struct{}

// Language returns middleware that negotiates the request language,
// stores the served tag in the request context, and stamps
// Content-Language and Vary: Accept-Language (both on by default).
// Invalid configs follow the validate-and-log pattern: they are logged
// and the middleware falls back to safe values (see LanguageConfig).
func Language(cfg LanguageConfig) Middleware {
	if len(cfg.SupportedTags) == 0 {
		validateConfig("LanguageConfig", errLanguageTagsEmpty)

		cfg = DefaultLanguageConfig()
	}

	matcher := buildLanguageMatcher(cfg.SupportedTags)
	if matcher == nil {
		validateConfig("LanguageConfig", errLanguageTagInvalid)

		cfg = DefaultLanguageConfig()
		matcher = buildLanguageMatcher(cfg.SupportedTags)
	}

	defaultTag := strings.ToLower(trim(cfg.DefaultTag))
	if defaultTag == "" {
		defaultTag = matcher.supported[0]
	} else if _, ok := matcher.match(defaultTag); !ok {
		validateConfig(
			"LanguageConfig",
			errLanguageDefaultUnsupported.WithContextAny("default_tag", cfg.DefaultTag),
		)

		defaultTag = matcher.supported[0]
	}

	extractors := make([]LanguageExtractor, 0, len(cfg.Extractors))
	for _, extractor := range cfg.Extractors {
		if extractor == nil {
			validateConfig("LanguageConfig", errLanguageExtractorNil)

			continue
		}

		extractors = append(extractors, extractor)
	}

	chain := LanguageExtractorChain(extractors...)

	resolver := cfg.Matcher
	if resolver == nil {
		resolver = matcher.match
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			served := defaultTag

			if candidates, ok := chain(r); ok {
				if resolved, matched := resolveTag(resolver, candidates); matched {
					served = resolved
				}
			}

			if !cfg.DisableVary {
				addVaryToken(w.Header(), headerAcceptLanguage)
			}

			if !cfg.DisableContentLanguage {
				w.Header().Set(headerContentLanguage, served)
			}

			next.ServeHTTP(w, r.WithContext(WithLanguage(r.Context(), served)))
		})
	}
}

// resolveTag applies the matcher to the candidates in order and reports
// whether one resolved. A matcher panic is not caught: matchers are
// app-supplied configuration, and a panicking config is a
// construction-time bug, not a per-request condition.
func resolveTag(resolver TagMatcher, candidates []string) (string, bool) {
	for _, candidate := range candidates {
		if supported, ok := resolver(candidate); ok {
			return supported, true
		}
	}

	return "", false
}

// addVaryToken adds token to the Vary header unless any existing Vary
// value already lists it (comma-aware, case-insensitive). Add (never Set)
// preserves Vary values from outer middleware such as Compression; the
// membership check avoids stacking a duplicate token. The common
// empty-Vary path allocates nothing.
func addVaryToken(h http.Header, token string) {
	for _, value := range h.Values(headerVary) {
		for part := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(trim(part), token) {
				return
			}
		}
	}

	h.Add(headerVary, token)
}

// WithLanguage stores the negotiated language tag in the context.
// Retrieve it with LanguageFromContext or LanguageFromRequest.
func WithLanguage(parent context.Context, tag string) context.Context {
	return context.WithValue(parent, languageKey{}, tag)
}

// LanguageFromContext retrieves the negotiated language tag. Returns an
// empty string when no language was stored.
func LanguageFromContext(ctx context.Context) string {
	tag, _ := ctx.Value(languageKey{}).(string)

	return tag
}

// LanguageFromRequest retrieves the negotiated language tag from the
// request context. Returns an empty string when no language was stored.
func LanguageFromRequest(r *http.Request) string {
	return LanguageFromContext(r.Context())
}
