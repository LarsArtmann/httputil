package httputil

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func assertItoa(t *testing.T, input int, want string) {
	t.Helper()

	if got := itoa(input); got != want {
		t.Errorf("itoa(%d) = %q, want %q", input, got, want)
	}
}

func TestItoaZero(t *testing.T) {
	t.Parallel()

	assertItoa(t, 0, "0")
}

func TestItoaPositive(t *testing.T) {
	t.Parallel()

	assertItoa(t, 42, "42")
}

func TestItoaOneDigit(t *testing.T) {
	t.Parallel()

	assertItoa(t, 7, "7")
}

func TestItoaLargePositive(t *testing.T) {
	t.Parallel()

	assertItoa(t, 123456789, "123456789")
}

func TestItoaNegative(t *testing.T) {
	t.Parallel()

	assertItoa(t, -42, "-42")
}

func TestItoaNegativeOne(t *testing.T) {
	t.Parallel()

	assertItoa(t, -1, "-1")
}

func TestItoaLargeNegative(t *testing.T) {
	t.Parallel()

	assertItoa(t, -123456789, "-123456789")
}

func TestItoaMaxInt(t *testing.T) {
	t.Parallel()

	want := "9223372036854775807"
	assertItoa(t, math.MaxInt, want)
}

func TestItoaMinInt(t *testing.T) {
	t.Parallel()

	want := "-9223372036854775808"
	assertItoa(t, math.MinInt, want)
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
