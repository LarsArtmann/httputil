package endpoints

import (
	"errors"
	"testing"
)

func TestParseQValue_Zero(t *testing.T) {
	t.Parallel()

	assertQValue(t, "0", 0)
}

func TestParseQValue_One(t *testing.T) {
	t.Parallel()

	assertQValue(t, "1", 1)
}

func TestParseQValue_Fraction(t *testing.T) {
	t.Parallel()

	assertQValue(t, "0.5", 0.5)
}

func TestParseQValue_ThreeFractionDigits(t *testing.T) {
	t.Parallel()

	assertQValue(t, "0.123", 0.123)
}

func TestParseQValue_TrailingFractionZeroes(t *testing.T) {
	t.Parallel()

	assertQValue(t, "1.000", 1)
}

func TestParseQValue_EmptyFraction(t *testing.T) {
	t.Parallel()

	// Lenient posture (the 2026-10-10 root-parser lesson): "0." is accepted
	// as 0.0 rather than rejected.
	assertQValue(t, "0.", 0)
}

func TestParseQValue_PlusSignAccepted(t *testing.T) {
	t.Parallel()

	assertQValue(t, "+1", 1)
}

func TestParseQValue_MinusSignPreserved(t *testing.T) {
	t.Parallel()

	assertQValue(t, "-0.5", -0.5)
}

func TestParseQValue_EmptyInput_ReturnsErrEmptyQValue(t *testing.T) {
	t.Parallel()

	_, err := parseQValue("")

	if !errors.Is(err, errEmptyQValue) {
		t.Errorf("err = %v, want errEmptyQValue", err)
	}
}

func TestParseQValue_IntegerAboveOne_ReturnsErrInvalidQInt(t *testing.T) {
	t.Parallel()

	_, err := parseQValue("2")

	if !errors.Is(err, errInvalidQInt) {
		t.Errorf("err = %v, want errInvalidQInt", err)
	}
}

func TestParseQValue_LetterIntegerPart_ReturnsErrInvalidQInt(t *testing.T) {
	t.Parallel()

	_, err := parseQValue("a")

	if !errors.Is(err, errInvalidQInt) {
		t.Errorf("err = %v, want errInvalidQInt", err)
	}
}

func TestParseQValue_FractionAboveOne_ReturnsErrQValueTooLarge(t *testing.T) {
	t.Parallel()

	_, err := parseQValue("1.5")

	if !errors.Is(err, errQValueTooLarge) {
		t.Errorf("err = %v, want errQValueTooLarge", err)
	}
}

func TestParseQValue_TrailingCharacters_ReturnsErrTrailingQChars(t *testing.T) {
	t.Parallel()

	_, err := parseQValue("0.5x")

	if !errors.Is(err, errTrailingQChars) {
		t.Errorf("err = %v, want errTrailingQChars", err)
	}
}

func TestParseQValue_FourthFractionDigit_ReturnsErrTrailingQChars(t *testing.T) {
	t.Parallel()

	// Only three fraction digits are consumed; the fourth is trailing.
	_, err := parseQValue("0.1234")

	if !errors.Is(err, errTrailingQChars) {
		t.Errorf("err = %v, want errTrailingQChars", err)
	}
}

func TestParseQValue_DoubleDot_ReturnsErrTrailingQChars(t *testing.T) {
	t.Parallel()

	_, err := parseQValue("0.1.2")

	if !errors.Is(err, errTrailingQChars) {
		t.Errorf("err = %v, want errTrailingQChars", err)
	}
}

// assertQValue parses input and asserts it yields want without error.
func assertQValue(t *testing.T, input string, want float64) {
	t.Helper()

	got, err := parseQValue(input)
	if err != nil {
		t.Fatalf("parseQValue(%q) returned error: %v", input, err)
	}

	if got != want {
		t.Errorf("parseQValue(%q) = %v, want %v", input, got, want)
	}
}
