# Message Lookup with go-i18n

httputil's `Language` middleware negotiates the response language and stamps `Content-Language` + `Vary: Accept-Language`; it deliberately does not render strings. [go-i18n](https://pkg.go.dev/github.com/nicksnyder/go-i18n/v2/i18n) owns message catalogs and plural rules. The bridge is one line: read the negotiated tag from the request context and hand it to a `Localizer`. No core dependency required.

## Pattern

Load the bundle once at startup; construct the `Localizer` per request from the negotiated tag.

```go
// This example illustrates the Localizer bridge. It references
// github.com/nicksnyder/go-i18n/v2, which is NOT a dependency of httputil —
// add it to your go.mod to compile.
package main

import (
    "net/http"
    "os"

    "github.com/larsartmann/httputil"
    "github.com/nicksnyder/go-i18n/v2/i18n"
    "golang.org/x/text/language"
)

func main() {
    bundle := i18n.NewBundle(language.English)
    bundle.LoadMessageFileFS(os.DirFS("locales"), "active.en.json")
    bundle.LoadMessageFileFS(os.DirFS("locales"), "active.de.json")

    negotiate := httputil.Language(httputil.DefaultLanguageConfig())

    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        localizer := i18n.NewLocalizer(bundle, httputil.LanguageFromRequest(r))

        greeting, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: "greeting"})
        if err != nil {
            http.Error(w, "translation missing", http.StatusInternalServerError)
            return
        }

        w.Write([]byte(greeting))
    })

    _ = http.ListenAndServe(":8080", negotiate(handler))
}
```

## Why the bridge is exact

`LanguageFromRequest(r)` returns the served tag — a member of your `SupportedTags`, in the spelling you declared. Passing that exact string to `i18n.NewLocalizer` bypasses go-i18n's own Accept-Language parsing and its matcher-ordering caveats: the bundle lookup targets precisely the language you negotiated, and `Content-Language` on the wire matches the message catalog that produced the body.

## Cache correctness is already handled

Because `Language()` stamps `Vary: Accept-Language` by default, caches revalidate per request header; because the negotiated tag also decides the body (through this bridge), header and body can never disagree. Do not disable `Vary` when the response body depends on the negotiated language.

## Localizer lifetime

`i18n.NewLocalizer` allocates a matcher per call. Per-request construction keeps the code stateless and is the shape shown in go-i18n's docs; if profiling ever shows it matters, cache one `Localizer` per supported tag — the set is closed (`SupportedTags`), so the cache is bounded and pre-warmable at startup.

## Guard test: make missing translations visible in CI

The deployment failure mode of this bridge is silent: a bundle missing `active.de.json` falls back to the default language and German users read English — no error anywhere. Catch it with one CI test that renders every supported tag and asserts the output is actually the requested language, not the fallback. The shape (from the sperrmuell-direct deployment's guard pattern):

```go
// This example illustrates the guard-test pattern for one handler; wire in
// your own handler and marker strings. One standalone test per deployment.
func TestEverySupportedTagRendersItsLanguage(t *testing.T) {
	t.Parallel()

	markers := map[string]string{"en": "Welcome", "de": "Willkommen"}

	for tag, marker := range markers {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req.WithContext(httputil.WithLanguage(req.Context(), tag)))

		body := rec.Body.String()
		if !strings.Contains(body, marker) {
			t.Errorf("tag %s: rendered body misses the %q marker — catalog entry missing or fallback served: %.120q", tag, marker, body)
		}

		if got := rec.Header().Get("Content-Language"); got != tag {
			t.Errorf("tag %s: Content-Language = %q", tag, got)
		}
	}
}
```

A missing catalog file fails the test with the exact tag named — the translation gap becomes a red build instead of a user report. Run it before the release gate; pair it with `httpspec.LanguageSpecs()` for the header contract.

Related: the [language cookbook](language-cookbook.md) shows the German-canonical + `/en` deployment shape this bridge is typically deployed in.

## Locale formatting (out of middleware scope, by design)

go-i18n owns message catalogs and plurals; `httputil.Language()` owns negotiation. Neither owns **formatting** (dates, numbers, currency) — that is `golang.org/x/text/message`'s job. Route the negotiated tag into a per-request `message.Printer`:

```go
func handle(w http.ResponseWriter, r *http.Request) {
	printer := message.NewPrinter(language.Make(string(httputil.LanguageFromRequest(r))))

	price := printer.Sprintf("%.2f", 1234.5) // locale-correct decimal separator
	_ = price
}
```

Build the printer per request (it is cheap); do not cache one per locale in a global unless you also pin its behavior under concurrent use. The middleware's contract ends at `Content-Language`/context — everything user-visible that depends on locale conventions belongs to the formatter layer.
