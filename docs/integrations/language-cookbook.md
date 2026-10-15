# Language Cookbook — The Sperrmuell Shape (German Canonical + `/en`)

_Worked example for the D5 ruling (i18n consolidated plan): German-canonical-unprefixed + `/en` prefix, the SEO shape proven by the sperrmuell-direct deployment (`/api/*` config landed 2026-10-10). The EN-canonical global-SaaS variant follows at the end._

## The shape

German is the canonical language: URLs without a prefix serve German (the default), `/en/...` serves English. SEO keeps one canonical host; crawlers see `Content-Language` on every response and the hreflang alternates live in the sitemap (app lane, see the caveat below).

```go
cfg := httputil.DefaultLanguageConfig()
cfg.SupportedTags = []string{"de", "en"} // first entry = default
cfg.DefaultTag = ""                      // empty = first supported tag ("de")

negotiate := httputil.LanguageExtractorChain(
    httputil.LanguageExtractorFromPathPrefix(map[string]string{"/en": "en"}),
    httputil.LanguageExtractorFromCookie("lang"),
    httputil.LanguageExtractorFromAcceptHeader(),
)
cfg.Extractors = []httputil.LanguageExtractor{negotiate}

handler := httputil.Language(cfg)(mux)
```

Resolution order per request: `/en` path prefix wins, then the `lang` cookie (language switcher), then `Accept-Language`, then the default (`de`). The first extractor producing a supported tag decides — that is why the path leg is first: a bookmarked `/en/pricing` must serve English even for a German browser.

What the middleware stamps on every response:

- `Content-Language: de` or `en` — the served tag, in the declared spelling.
- `Vary: Accept-Language` — caches revalidate per header (RFC-strict default, kept even when a cookie or the path leg decided; `DisableVary` opts out for cookie/query-only chains).
- The tag lands in the request context (`LanguageFromRequest(r)`) for handlers and templates.

## German-canonical specifics

- `DefaultTag` stays empty (or `"de"`): zero-value requests with no headers, no cookie, and no prefix serve German.
- The `/en` prefix map keys on the exact path prefix — mount the middleware before your router so `/en/...` routes resolve after negotiation.
- Spellings are declared once in `SupportedTags` and preserved verbatim on the wire (`de`, not `de-DE`).

## EN-canonical global-SaaS variant

The config flips, nothing else changes: `SupportedTags = []string{"en", "de"}` (English canonical, unprefixed), and the prefix map covers the non-canonical locales (`/de` → `de`). Same extractor chain, same header behavior.

## hreflang caveat (app lane, deliberately out of library scope)

`Content-Language` and `Vary` are per-response HTTP facts; hreflang alternates (`<link rel="alternate" hreflang>` or sitemap `xhtml:link` entries) bind URL variants together and stay an application route-registry concern — recorded as out-of-scope in ROADMAP. Point the alternates at the canonical host's variant URLs and keep them consistent with the prefix map above; Google's localized-versions documentation is the reference for the annotation formats.

## Related

- [Message Lookup with go-i18n](go-i18n.md) — rendering strings from the negotiated tag.
- [x/text Matcher integration](x-text.md) — full BCP 47 matching via `TagMatcher`.
- `httpspec.LanguageSpecs()` — opt-in contract checks that a handler stamps both headers.
