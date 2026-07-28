package httputil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseUintQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
		key   string
		want  uint
	}{
		{"valid", "page=42", "page", 42},
		{"missing key", "", "page", 0},
		{"empty value", "page=", "page", 0},
		{"negative", "page=-1", "page", 0},
		{"non-numeric", "page=abc", "page", 0},
		{"zero", "page=0", "page", 0},
		{"large valid", "page=4294967295", "page", 4294967295},
		{"overflow 32-bit", "page=4294967296", "page", 0},
		{"different key", "size=10&page=3", "page", 3},
		{"float", "page=1.5", "page", 0},
		{"hex notation", "page=0x10", "page", 0},
		{"plus sign", "page=+5", "page", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)

			got := ParseUintQuery(req, tt.key)
			if got != tt.want {
				t.Errorf("ParseUintQuery(%q) = %d, want %d", tt.key, got, tt.want)
			}
		})
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

		req := httptest.NewRequest(http.MethodGet, "/?page="+value, nil)

		got := ParseUintQuery(req, "page")

		if value == "" {
			if got != 0 {
				t.Errorf("empty value should return 0, got %d", got)
			}

			return
		}

		// For any input, result must be a valid uint (no panic, no negative).
		// ParseUintQuery either parses successfully or returns 0.
		_ = got
	})
}

func ExampleParseUintQuery() {
	req := httptest.NewRequest(http.MethodGet, "/?page=3&limit=20", nil)

	page := ParseUintQuery(req, "page")
	fmt.Println("page:", page)

	// Output: page: 3
}
