package endpoints

import (
	"strings"
	"testing"
)

func TestRenderJSON_ExactBytes(t *testing.T) {
	t.Parallel()

	got := renderJSON([]Route{
		{Method: "GET", Pattern: "GET /endpoints", Summary: listingSummary},
		{Method: "GET", Pattern: "GET /health", Summary: ""},
	})

	want := `{"endpoints":[{"method":"GET","pattern":"GET /endpoints","summary":"Endpoint listing (this page)"},` +
		`{"method":"GET","pattern":"GET /health"}]}` + "\n"

	if got != want {
		t.Errorf("got  = %q\nwant = %q", got, want)
	}
}

func TestRenderJSON_EmptyRegistry(t *testing.T) {
	t.Parallel()

	if got, want := renderJSON(nil), "{\"endpoints\":[]}\n"; got != want {
		t.Errorf("got = %q, want %q", got, want)
	}
}

func TestRenderHTML_ExactDocument(t *testing.T) {
	t.Parallel()

	got := renderHTML([]Route{
		{Method: "GET", Pattern: "GET /endpoints", Summary: listingSummary},
		{Method: "", Pattern: "/health", Summary: ""},
	})

	want := `<!DOCTYPE html>
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
<tr><td>GET</td><td>GET /endpoints</td><td>Endpoint listing (this page)</td></tr>
<tr><td>*</td><td>/health</td><td></td></tr>
</tbody>
</table>
</body>
</html>
`

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderHTML_EscapesRouteFields(t *testing.T) {
	t.Parallel()

	hostile := Route{
		Method:  `G<ET`,
		Pattern: `/<script>alert("x")</script>`,
		Summary: `&amp; injection 'attempt'`,
	}

	page := renderHTML([]Route{hostile})

	if strings.Contains(page, "<script>") {
		t.Errorf("page contains a raw <script> element:\n%s", page)
	}

	if !strings.Contains(page, "&lt;script&gt;") {
		t.Errorf("page does not contain the escaped pattern:\n%s", page)
	}

	if !strings.Contains(page, "&amp;amp;") {
		t.Errorf("page does not double-escape the literal ampersand entity:\n%s", page)
	}
}

func TestRenderHTML_ContainsNoStyleOrScriptElements(t *testing.T) {
	t.Parallel()

	routes := []Route{
		{Method: "GET", Pattern: "GET /a", Summary: "<style>body{}</style>"},
		{Method: "POST", Pattern: "POST /b", Summary: "<script src=evil>"},
	}

	page := renderHTML(routes)

	if strings.Contains(page, "<style") || strings.Contains(page, "<script") {
		t.Errorf("page contains style/script markup; it must render under a strict CSP:\n%s", page)
	}
}

func TestRenderHTML_AnyMethodRoute_ShowsStar(t *testing.T) {
	t.Parallel()

	page := renderHTML([]Route{{Method: "", Pattern: "/everything", Summary: ""}})

	if !strings.Contains(page, "<td>*</td>") {
		t.Errorf("page does not show the * label for an any-method route:\n%s", page)
	}
}
