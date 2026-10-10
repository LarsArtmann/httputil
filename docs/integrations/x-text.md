# Full BCP 47 Matching with golang.org/x/text

The built-in `TagMatcher` resolves candidates with exact-then-primary-subtag matching: `de-AT` serves `de`, `zh-Hans` matches `zh-Hans` exactly or `zh` by primary subtag. That covers most sites. Applications that want CLDR-aware matching — macro languages, mutual intelligibility (`nb` matching `no`), script awareness, deprecated-tag canonicalization — can plug in [golang.org/x/text](https://pkg.go.dev/golang.org/x/text/language)'s `language.Matcher` through the `TagMatcher` plugin interface. No core dependency required.

## Pattern

Build the `language.Matcher` once at construction and resolve each candidate through it. The adapter keeps the declared spellings from `SupportedTags` for serving, so `Content-Language` keeps the exact spelling you declared even though x/text canonicalizes internally.

```go
// This example illustrates the TagMatcher pattern. It references
// golang.org/x/text, which is NOT a dependency of httputil — add it to
// your go.mod to compile.
package main

import (
    "net/http"

    "github.com/larsartmann/httputil"
    "golang.org/x/text/language"
)

func newXTextMatcher(served []string, tags []language.Tag) httputil.TagMatcher {
    matcher := language.NewMatcher(tags)

    return func(tag string) (string, bool) {
        parsed, err := language.Parse(tag)
        if err != nil {
            return "", false // unparsable candidate: try the next candidate
        }

        _, idx, conf := matcher.Match(parsed)
        if conf == language.No {
            return "", false // no supported language matches: keep looking
        }

        return served[idx], true
    }
}

func main() {
    served := []string{"en", "de"}
    tags := []language.Tag{language.English, language.German}

    mw := httputil.Language(httputil.LanguageConfig{
        SupportedTags: served,
        Extractors: []httputil.LanguageExtractor{
            httputil.LanguageExtractorFromAcceptHeader(),
        },
        Matcher: newXTextMatcher(served, tags),
    })

    _ = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("served as " + httputil.LanguageFromRequest(r)))
        })).ServeHTTP(w, r)
    })
}
```

## Confidence gating

`Matcher.Match` always returns its best supported tag plus a `language.Confidence` (`Exact`, `High`, `Low`, `No`). `language.No` means nothing matched; returning `ok=false` in that branch makes the middleware fall through to the next candidate and finally `DefaultTag`. Accepting `Low` matches (the default in the example) is what enables mutual-intelligibility pairs; gate on `Exact`/`High` only if serving a low-confidence substitute is worse than serving the default.

## Keep the built-in extractor

`LanguageExtractorFromAcceptHeader` already parses `Accept-Language` with q-value ordering, and the middleware iterates candidates until one resolves. Swapping the matcher is the only change most apps need. For whole-header matching in one step, x/text also offers `language.MatchStrings(matcher, r.Header.Get("Accept-Language"))`, which you can wrap as a custom `LanguageExtractor` returning the matched supported tag as a single candidate.

## Concurrency

`httputil.TagMatcher` implementations must be safe for concurrent use across requests. Build the `language.Matcher` once at construction and treat it as immutable thereafter — the matcher performs read-only matching after `NewMatcher`, which is the standard shape for this requirement.

## Parse vs Make

Use `language.Parse` (returns an error) inside the per-request adapter so garbage candidates fail the leg cleanly. `language.Make` never returns an error — it silently substitutes a sensible default — which would turn unparsable input into a surprising match.
