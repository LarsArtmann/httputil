package httputil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

// newQueryRequest builds a GET request whose query string is the given text.
func newQueryRequest(query string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/?"+query, nil)
}

func TestParseUintQuery_Valid(t *testing.T) {
	t.Parallel()

	if got := ParseUintQuery(newQueryRequest("page=42"), "page"); got != 42 {
		t.Errorf("ParseUintQuery(page) = %d, want 42", got)
	}
}

func TestParseUintQuery_MissingKey(t *testing.T) {
	t.Parallel()

	if got := ParseUintQuery(newQueryRequest(""), "page"); got != 0 {
		t.Errorf("ParseUintQuery(page) = %d, want 0", got)
	}
}

func TestParseUintQuery_EmptyValue(t *testing.T) {
	t.Parallel()

	if got := ParseUintQuery(newQueryRequest("page="), "page"); got != 0 {
		t.Errorf("ParseUintQuery(page) = %d, want 0", got)
	}
}

func TestParseUintQuery_Negative(t *testing.T) {
	t.Parallel()

	if got := ParseUintQuery(newQueryRequest("page=-1"), "page"); got != 0 {
		t.Errorf("ParseUintQuery(page) = %d, want 0", got)
	}
}

func TestParseUintQuery_NonNumeric(t *testing.T) {
	t.Parallel()

	if got := ParseUintQuery(newQueryRequest("page=abc"), "page"); got != 0 {
		t.Errorf("ParseUintQuery(page) = %d, want 0", got)
	}
}

func TestParseUintQuery_Zero(t *testing.T) {
	t.Parallel()

	if got := ParseUintQuery(newQueryRequest("page=0"), "page"); got != 0 {
		t.Errorf("ParseUintQuery(page) = %d, want 0", got)
	}
}

func TestParseUintQuery_LargeValid(t *testing.T) {
	t.Parallel()

	if got := ParseUintQuery(newQueryRequest("page=4294967295"), "page"); got != 4294967295 {
		t.Errorf("ParseUintQuery(page) = %d, want 4294967295", got)
	}
}

func TestParseUintQuery_Overflow32Bit(t *testing.T) {
	t.Parallel()

	if got := ParseUintQuery(newQueryRequest("page=4294967296"), "page"); got != 0 {
		t.Errorf("ParseUintQuery(page) = %d, want 0", got)
	}
}

func TestParseUintQuery_DifferentKey(t *testing.T) {
	t.Parallel()

	if got := ParseUintQuery(newQueryRequest("size=10&page=3"), "page"); got != 3 {
		t.Errorf("ParseUintQuery(page) = %d, want 3", got)
	}
}

func TestParseUintQuery_Float(t *testing.T) {
	t.Parallel()

	if got := ParseUintQuery(newQueryRequest("page=1.5"), "page"); got != 0 {
		t.Errorf("ParseUintQuery(page) = %d, want 0", got)
	}
}

func TestParseUintQuery_HexNotation(t *testing.T) {
	t.Parallel()

	if got := ParseUintQuery(newQueryRequest("page=0x10"), "page"); got != 0 {
		t.Errorf("ParseUintQuery(page) = %d, want 0", got)
	}
}

func TestParseUintQuery_PlusSign(t *testing.T) {
	t.Parallel()

	if got := ParseUintQuery(newQueryRequest("page=+5"), "page"); got != 0 {
		t.Errorf("ParseUintQuery(page) = %d, want 0", got)
	}
}

func TestParseUintQueryMultipleParams(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/?page=2&page_size=50", nil)
	if got := ParseUintQuery(req, "page"); got != 2 {
		t.Errorf("page = %d, want 2", got)
	}

	if got := ParseUintQuery(req, "page_size"); got != 50 {
		t.Errorf("page_size = %d, want 50", got)
	}
}

func BenchmarkParseUintQuery(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/?page=42&page_size=20", nil)

	b.ReportAllocs()

	for b.Loop() {
		_ = ParseUintQuery(req, "page")
	}
}

func FuzzParseUintQuery(f *testing.F) {
	f.Add("42")
	f.Add("")
	f.Add("-1")
	f.Add("abc")
	f.Add("4294967296")
	f.Add("0x10")
	f.Add("1.5")

	f.Fuzz(func(t *testing.T, value string) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/?page="+url.QueryEscape(value), nil)

		got := ParseUintQuery(req, "page")

		if value == "" {
			if got != 0 {
				t.Errorf("empty value should return 0, got %d", got)
			}

			return
		}

		// ParseUintQuery either parses successfully or returns 0. When it
		// returns non-zero, the value must agree with an independent
		// strconv.ParseUint of the same input (beyond the 32-bit cap).
		if got != 0 {
			ref, refErr := strconv.ParseUint(value, 10, 32)
			if refErr == nil && got != uint(ref) {
				t.Errorf("ParseUintQuery = %d, want %d for input %q", got, ref, value)
			}
		}
	})
}

func ExampleParseUintQuery() {
	req := httptest.NewRequest(http.MethodGet, "/?page=3&limit=20", nil)

	page := ParseUintQuery(req, "page")
	fmt.Println("page:", page)

	// Output: page: 3
}
