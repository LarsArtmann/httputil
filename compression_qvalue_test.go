package httputil

import "testing"

func assertQValue(t *testing.T, input string, want float64) {
	t.Helper()

	got, err := parseQValue(input)
	if err != nil {
		t.Errorf("parseQValue(%q) error = %v, want nil", input, err)

		return
	}

	diff := got - want
	if diff < -0.001 || diff > 0.001 {
		t.Errorf("parseQValue(%q) = %f, want %f", input, got, want)
	}
}

func assertQValueError(t *testing.T, input string) {
	t.Helper()

	_, err := parseQValue(input)
	if err == nil {
		t.Errorf("parseQValue(%q) error = nil, want error", input)
	}
}

// TestParseQValue_One verifies q-value "1" parses correctly.
func TestParseQValue_One(t *testing.T) {
	t.Parallel()

	assertQValue(t, "1", 1.0)
}

// TestParseQValue_Zero verifies q-value "0" parses correctly.
func TestParseQValue_Zero(t *testing.T) {
	t.Parallel()

	assertQValue(t, "0", 0.0)
}

// TestParseQValue_Half verifies q-value "0.5" parses correctly.
func TestParseQValue_Half(t *testing.T) {
	t.Parallel()

	assertQValue(t, "0.5", 0.5)
}

// TestParseQValue_NineTenths verifies q-value "0.9" parses correctly.
func TestParseQValue_NineTenths(t *testing.T) {
	t.Parallel()

	assertQValue(t, "0.9", 0.9)
}

// TestParseQValue_OnePointZero verifies q-value "1.0" parses correctly.
func TestParseQValue_OnePointZero(t *testing.T) {
	t.Parallel()

	assertQValue(t, "1.0", 1.0)
}

// TestParseQValue_ThreeDecimals verifies q-value "0.001" parses correctly.
func TestParseQValue_ThreeDecimals(t *testing.T) {
	t.Parallel()

	assertQValue(t, "0.001", 0.001)
}

// TestParseQValue_EmptyError verifies empty q-value returns an error.
func TestParseQValue_EmptyError(t *testing.T) {
	t.Parallel()

	assertQValueError(t, "")
}

// TestParseQValue_TwoError verifies q-value "2" returns an error.
func TestParseQValue_TwoError(t *testing.T) {
	t.Parallel()

	assertQValueError(t, "2")
}

// TestParseQValue_OnePointFiveError verifies q-value "1.5" returns an error.
func TestParseQValue_OnePointFiveError(t *testing.T) {
	t.Parallel()

	assertQValueError(t, "1.5")
}

// TestParseQValue_AlphabeticError verifies q-value "abc" returns an error.
func TestParseQValue_AlphabeticError(t *testing.T) {
	t.Parallel()

	assertQValueError(t, "abc")
}

// TestParseQValue_DoubleDecimalError verifies q-value "0.5.5" returns an error.
func TestParseQValue_DoubleDecimalError(t *testing.T) {
	t.Parallel()

	assertQValueError(t, "0.5.5")
}

// TestParseQValue_TrailingCharsError verifies q-value "1.0a" returns an error.
func TestParseQValue_TrailingCharsError(t *testing.T) {
	t.Parallel()

	assertQValueError(t, "1.0a")
}
