# `Language()` middleware — design note (T05–T09)

Status: EXECUTED design (owner green-light 2026-10-10, "get shit done — the
WHOLE TODO LIST"; D1–D6 rulings adopted per the consolidated plan's own
recommendations). Plan of record:
[planning/2026-10-10_02-48-SUPERB-i18n-consolidated-plan.html](2026-10-10_02-48-SUPERB-i18n-consolidated-plan.html).
Evidence base: ROADMAP §42, the sperrmuell-direct consumer study, the
go-appkit sibling doc, and the 2026-10-10 fleet demand sweep (N≥4 repos:
sperrmuell-direct, go-website-template, artmann-technologies-website,
webphone — all hand-rolling negotiation; none import go-appkit).

## Rulings (D1–D6, as adopted)

| Gate              | Ruling                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| D1 home           | Negotiation lives once in httputil core (`language.go`); go-appkit keeps the content lane (Translator/guards/Alternates), parked on demand (N=1 appkit consumer, sweep-verified).                                                                                                                                                                                                                                                                                                                             |
| D2 Vary           | RFC-strict default: `Vary: Accept-Language` on every response the middleware wraps, `DisableVary` opt-out. URL/cookie legs make content vary per-visitor regardless of what the header said; emitting the header unconditionally is the cache-honest superb default.                                                                                                                                                                                                                                          |
| D3 matcher        | Zero-dep primary-subtag matcher by default (policy-conformant: no new core deps). Full BCP 47 / CLDR matching stays a plugin surface (`TagMatcher`); `docs/integrations/x-text.md` carries the adapter pattern. The pt/zh script-class misroute is conceded real; the plugin hatch + its discoverability (doc + config-doc comment) is the mitigation.                                                                                                                                                        |
| D4 406/confidence | No `RejectOnNoMatch` axis in v1; unmatched requests silently serve the default tag. Rationale: a language preference is a hint, never a contract — 406 on a preference header breaks crawlers/clients that send stale headers, and every surveyed middleware falls back silently. Confidence is not exposed (the built-in matcher is exact-or-primary; there is nothing honest to expose). Additive evolution path: a future `RejectOnNoMatch bool` + `language.no_match` Transient code, no breaking change. |
| D5 cookbook       | sperrmuell shape (German canonical unprefixed + `/en`) is the worked example; EN-canonical global-SaaS is a config variant, documented as such.                                                                                                                                                                                                                                                                                                                                                               |
| D6 cross-repo     | Granted and executed: sperrmuell-direct `/api/*` now sends `Vary: Accept-Language` + `Content-Language` (T03, all 7 gates green), TODO row T70 records the Matcher-upgrade trigger (T04).                                                                                                                                                                                                                                                                                                                     |

## Deviation from the plan sketch (recorded deliberately)

The plan sketched `LanguageExtractor func(*http.Request) (tag string, ok
bool)`. A single-tag extractor cannot negotiate correctly: the top
Accept-Language candidate may not match the supported set while the header's
second candidate does (`de-AT,de;q=0.9,en;q=0.8` against supported
`{en}` must serve `en`). The shipped signature is therefore
`func(*http.Request) ([]string, ok bool)` — an ordered candidate list per
leg (q-desc, tie = listed order, `q=0` dropped, `*` yields no candidates).
Chain semantics stay first-wins at the LEG level: the first leg with any
candidates decides the negotiation input (sperrmuell's path-fixes-journey
behavior); within the leg, candidates are tried in order. Leg veto is
deliberately NOT a thing: an unsupported path tag falls through to the leg's
remaining candidates, then to the default — preference order, not veto
order.

## API surface (v1)

```go
type LanguageExtractor func(r *http.Request) (tags []string, ok bool)

func LanguageExtractorFromAcceptHeader() LanguageExtractor
func LanguageExtractorFromQuery(param string) LanguageExtractor
func LanguageExtractorFromCookie(name string) LanguageExtractor
func LanguageExtractorFromPathPrefix(prefixes map[string]string) LanguageExtractor
func LanguageExtractorChain(extractors ...LanguageExtractor) LanguageExtractor

type TagMatcher func(tag string) (supported string, ok bool)

type LanguageConfig struct {
    SupportedTags       []string          // required; priority order; first = default
    DefaultTag          string            // "" → SupportedTags[0]
    Extractors          []LanguageExtractor // nil/empty → no extraction (always default tag)
    Matcher             TagMatcher        // nil → built-in exact-then-primary matcher
    DisableContentLanguage bool           // zero value keeps the header ON (safe default)
    DisableVary         bool              // zero value keeps Vary ON (safe default)
}

func DefaultLanguageConfig() LanguageConfig // {"en"}, AcceptHeader, headers on
func (c LanguageConfig) Validate() error
func Language(cfg LanguageConfig) Middleware
func WithLanguage(ctx context.Context, tag string) context.Context
func LanguageFromContext(ctx context.Context) string // "" when absent
```

### Zero-value semantics (each pinned by an execution probe)

- `SupportedTags` empty → Validate rejects `language.tags_empty`; the
  constructor falls back to `DefaultLanguageConfig()` as a whole (there is
  no partial fallback for "which languages do you serve").
- `DefaultTag` empty → `SupportedTags[0]` (the documented "empty means
  first"). Non-empty but unsupported → logged `language.default_tag_unsupported`,
  falls back to `SupportedTags[0]`.
- `Extractors` nil/empty → valid; every request negotiates to the default
  tag (single-language sites). Nil entries inside the slice → logged
  `language.extractor_nil`, dropped.
- `Disable*` flags inverted so the zero value keeps both headers ON — the
  bare-literal trap (`LanguageConfig{SupportedTags: …}` silently losing
  cache correctness) cannot happen.
- Tag charset `[a-zA-Z0-9-]+` enforced (trim + charset validation at
  construction; commas/whitespace/CRLF are `language.tag_invalid`) —
  the served tag is written into `Content-Language`, so the supported set
  is a header-injection surface and is validated like one. Declared
  spellings are preserved verbatim, not canonicalized to lowercase:
  `zh-Hans` is served as `zh-Hans`. Duplicate detection is
  case-insensitive (`EqualFold`) and keeps the first declared spelling.

### Matching (built-in)

Pass 1: exact case-insensitive full-tag match against `SupportedTags`
(priority order). Pass 2: primary-subtag match (`de-AT` → `de`), first
supported tag whose primary subtag equals the candidate's. First match
wins; none → default. The matcher is precompiled at construction
(supported-primary table), mirroring the compression negotiator pattern.
The negotiated value served to callers (context and `Content-Language`)
is the declared spelling from `SupportedTags`, never a lowercased form —
only comparisons are lowercased (precompiled `lowered` slice + primary
table).

## Ordering rules (T06)

- **Language outside CSRF** — CSRF's 403 page should localize, which needs
  the language context already resolved. Recommended chain position:
  outer to CSRF, inner to CORS (CORS must stay outer for preflight).
- **CORS preflight** — no code special-case: OPTIONS requests negotiate
  like any other (harmless; a 204 preflight carrying `Content-Language` is
  noise but correct). Documented guidance, not machinery.
- **Recovery** — unchanged: still outermost when present
  (`MiddlewareStack.Validate` untouched).

## Vary composition (T07)

Language runs `Add("Vary", "Accept-Language")` — never `Set` — and skips
the add when any existing `Vary` value already lists the token
(comma-aware, case-insensitive; zero allocations on the common empty-Vary
path). Chained with `Compression`, the response carries two Vary lines
(`Accept-Encoding` from compression, `Accept-Language` from Language) —
RFC 9110 §12.5.5-valid, and both major CDN families combine fields.
Compression keeps its shipped plain-Add contract; this middleware is the
composing citizen. **ETag caveat (documented in README + integrations):**
validators are per-representation — an ETag computed before language
selection lets a cache serve language A's body to language B's conditional
request. Compute ETags after the language middleware (or mix the tag into
the ETag input); `Vary: Accept-Language` covers header-keyed caches only.

## Error domain (T08)

`language.*` domain, all Rejection family (invalid config = fix the
config): `language.tags_empty`, `language.tag_invalid`,
`language.default_tag_unsupported`, `language.extractor_nil`. Sentinels
live in `language.go`; templates + `allHTTputilErrorCodes` + domain test
follow the documented adding-a-code sweep (T18).

## Honest-silence audit (F47)

Header writes are map sets before `WriteHeader` — they cannot fail, so
there is nothing to discard. The middleware adds no post-commit write
paths; `writeCommittedBody` is not needed here.

## Out of scope (unchanged from ROADMAP §42)

Message catalogs/plurals/formatting (go-i18n's lane), hreflang/sitemap
alternates (route-registry lane), 406 rejection, confidence exposure,
ICU MessageFormat, date localization (no x/text/date exists).
