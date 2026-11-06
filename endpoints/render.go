package endpoints

import (
	"bytes"
	"encoding/json/v2"
	"html"
	"strings"
)

const (
	contentTypeJSON = "application/json"
	contentTypeHTML = "text/html; charset=utf-8"
)

// anyMethodLabel stands in for a route whose pattern matches every method.
const anyMethodLabel = "*"

// Response is the JSON representation of the listing: every registered route
// sorted by pattern (then method).
type Response struct {
	Endpoints []Route `json:"endpoints"`
}

// renderJSON renders the listing as JSON followed by a trailing newline
// (terminal-friendly, the root module's health-body convention).
//
// MarshalWrite cannot fail: every Route field was UTF-8-validated at
// registration ([Registry.Handle] rejects invalid UTF-8 patterns and
// summaries precisely because encoding/json v2 refuses to marshal them), so
// the discarded error is honest silence.
func renderJSON(routes []Route) string {
	var buf bytes.Buffer

	_ = json.MarshalWrite(&buf, Response{Endpoints: routes})
	buf.WriteString("\n")

	return buf.String()
}

// renderHTML renders the listing as a minimal HTML5 document: one semantic
// table, no inline styles, no scripts — the page renders under a strict CSP
// (the root module's SecurityHeaders defaults) without any nonce. Every
// dynamic field is escaped with [html.EscapeString].
func renderHTML(routes []Route) string {
	var page strings.Builder

	page.Grow(len(htmlDocumentPrefix) + len(routes)*htmlRowEstimate)

	page.WriteString(htmlDocumentPrefix)

	for _, route := range routes {
		method := route.Method
		if method == "" {
			method = anyMethodLabel
		}

		page.WriteString("<tr><td>")
		page.WriteString(html.EscapeString(method))
		page.WriteString("</td><td>")
		page.WriteString(html.EscapeString(route.Pattern))
		page.WriteString("</td><td>")
		page.WriteString(html.EscapeString(route.Summary))
		page.WriteString("</td></tr>\n")
	}

	page.WriteString(htmlDocumentSuffix)

	return page.String()
}

// htmlRowEstimate is the preallocation hint per rendered table row.
const htmlRowEstimate = 128

const htmlDocumentPrefix = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Endpoints</title>
</head>
<body>
<h1>Endpoints</h1>
<table>
<thead><tr><th scope="col">Method</th><th scope="col">Pattern</th><th scope="col">Summary</th></tr></thead>
<tbody>
`

const htmlDocumentSuffix = `</tbody>
</table>
</body>
</html>
`
