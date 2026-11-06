package endpoints

import (
	"html"
	"strings"
	"testing"
)

// FuzzNegotiateRepresentation verifies the Accept negotiation is total and
// deterministic for any input: no panic, always one of the two
// representations, and the same answer twice. The oracle deliberately models
// the documented lenient postures (garbage entries skipped, broken q-values
// fall back to the default weight) rather than the RFC grammar — a stricter
// oracle than the parser is noise, not a find (the 2026-10-10 lesson).
func FuzzNegotiateRepresentation(f *testing.F) {
	f.Add("")
	f.Add("*/*")
	f.Add("application/json")
	f.Add("text/html")
	f.Add("text/html;q=0.5, application/json;q=0.9")
	f.Add("text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	f.Add("text/*;q=0.3, text/html;q=0.7, text/html;level=1, */*;q=0.5")
	f.Add("garbage;;, ;;, ,,")
	f.Add("text/html;q=")
	f.Add("text/html;q=9.9.9.9")
	f.Add("TEXT/HTML;Q=0\t")
	f.Add(strings.Repeat("text/html,", 100) + "application/json")

	f.Fuzz(func(t *testing.T, accept string) {
		got := negotiateRepresentation(accept)

		if got != representationJSON && got != representationHTML {
			t.Errorf("negotiateRepresentation(%q) = %v, want one of the two representations", accept, got)
		}

		if again := negotiateRepresentation(accept); again != got {
			t.Errorf("negotiateRepresentation(%q) is not deterministic: %v then %v", accept, got, again)
		}
	})
}

// FuzzParseQValue pins the q-value parser's bound invariant: any input that
// parses yields a value in [-1, 1] (the integer part is restricted to 0 or 1,
// and the sign is preserved), and parsing is deterministic.
func FuzzParseQValue(f *testing.F) {
	f.Add("0")
	f.Add("1")
	f.Add("0.5")
	f.Add("0.123")
	f.Add("1.000")
	f.Add("0.")
	f.Add("-0.5")
	f.Add("+1")
	f.Add("2")
	f.Add("1.5")
	f.Add("0.1234")
	f.Add("abc")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		got, err := parseQValue(input)
		if err != nil {
			return
		}

		if got < -1 || got > 1 {
			t.Errorf("parseQValue(%q) = %v, want within [-1, 1]", input, got)
		}

		if again, againErr := parseQValue(input); again != got || againErr != nil {
			t.Errorf("parseQValue(%q) is not deterministic: (%v, %v) then (%v, %v)", input, got, err, again, againErr)
		}
	})
}

// FuzzRenderHTML verifies the HTML rendering is injection-safe for any route
// fields: HTML-significant field values never appear raw in the output, and
// the document is always complete.
func FuzzRenderHTML(f *testing.F) {
	f.Add("GET", "/users/{id}", "Fetch one user")
	f.Add("", "", "")
	f.Add("G<ET", "/<script>alert(1)</script>", "<style>x{}</style>")
	f.Add("GET", "/x", `&"'<>`)
	f.Add("GET", "/\x00\xff", "invalid utf8 \xff")
	f.Add("GET", strings.Repeat("a{", 200), strings.Repeat("b}", 200))

	f.Fuzz(func(t *testing.T, method, pattern, summary string) {
		page := renderHTML([]Route{{Method: method, Pattern: pattern, Summary: summary}})

		for _, field := range []string{method, pattern, summary} {
			// Positive oracle: every field must appear ESCAPED. A mutation
			// that emits a field raw fails this for HTML-significant values,
			// and the inverse containment check is impossible to write:
			// the document's own markup legitimately contains ", <, and >.
			if !strings.Contains(page, html.EscapeString(field)) {
				t.Errorf("field %q does not appear escaped in the page:\n%s", field, page)
			}
		}

		if !strings.HasSuffix(page, "</html>\n") {
			t.Errorf("page is not a complete document:\n%s", page)
		}
	})
}
