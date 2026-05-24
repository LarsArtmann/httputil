package httputil

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestItoaZero(t *testing.T) {
	t.Parallel()

	if got := itoa(0); got != "0" {
		t.Errorf("itoa(0) = %q, want %q", got, "0")
	}
}

func TestItoaPositive(t *testing.T) {
	t.Parallel()

	if got := itoa(42); got != "42" {
		t.Errorf("itoa(42) = %q, want %q", got, "42")
	}
}

func TestItoaOneDigit(t *testing.T) {
	t.Parallel()

	if got := itoa(7); got != "7" {
		t.Errorf("itoa(7) = %q, want %q", got, "7")
	}
}

func TestItoaLargePositive(t *testing.T) {
	t.Parallel()

	if got := itoa(123456789); got != "123456789" {
		t.Errorf("itoa(123456789) = %q, want %q", got, "123456789")
	}
}

func TestItoaNegative(t *testing.T) {
	t.Parallel()

	if got := itoa(-42); got != "-42" {
		t.Errorf("itoa(-42) = %q, want %q", got, "-42")
	}
}

func TestItoaNegativeOne(t *testing.T) {
	t.Parallel()

	if got := itoa(-1); got != "-1" {
		t.Errorf("itoa(-1) = %q, want %q", got, "-1")
	}
}

func TestItoaLargeNegative(t *testing.T) {
	t.Parallel()

	if got := itoa(-123456789); got != "-123456789" {
		t.Errorf("itoa(-123456789) = %q, want %q", got, "-123456789")
	}
}

func TestItoaMaxInt(t *testing.T) {
	t.Parallel()

	want := "9223372036854775807"
	if got := itoa(math.MaxInt); got != want {
		t.Errorf("itoa(math.MaxInt) = %q, want %q", got, want)
	}
}

func TestItoaMinInt(t *testing.T) {
	t.Parallel()

	want := "-9223372036854775808"
	if got := itoa(math.MinInt); got != want {
		t.Errorf("itoa(math.MinInt) = %q, want %q", got, want)
	}
}

func BenchmarkItoa(b *testing.B) {
	for b.Loop() {
		itoa(200)
		itoa(86400)
		itoa(math.MaxInt)
		itoa(math.MinInt)
	}
}

func BenchmarkItoa_Strconv(b *testing.B) {
	for b.Loop() {
		strconv.Itoa(200)
		strconv.Itoa(86400)
		strconv.Itoa(math.MaxInt)
		strconv.Itoa(math.MinInt)
	}
}

func BenchmarkJoin(b *testing.B) {
	input := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}

	for b.Loop() {
		join(input)
	}
}

func BenchmarkJoin_StringsJoin(b *testing.B) {
	input := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	sep := ", "

	for b.Loop() {
		strings.Join(input, sep)
	}
}
