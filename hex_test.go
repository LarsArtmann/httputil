package httputil

import (
	"testing"
)

func TestHexEncodeUint64_Zero(t *testing.T) {
	t.Parallel()

	if got := hexEncodeUint64(0); got != "0000000000000000" {
		t.Errorf("hexEncodeUint64(0) = %q, want %q", got, "0000000000000000")
	}
}

func TestHexEncodeUint64_MaxUint64(t *testing.T) {
	t.Parallel()

	if got := hexEncodeUint64(0xffffffffffffffff); got != "ffffffffffffffff" {
		t.Errorf("hexEncodeUint64(max) = %q, want %q", got, "ffffffffffffffff")
	}
}

func TestHexEncodeUint64_KnownValue(t *testing.T) {
	t.Parallel()

	if got := hexEncodeUint64(0x779a65e7023cd2e7); got != "779a65e7023cd2e7" {
		t.Errorf("hexEncodeUint64(0x779a65e7023cd2e7) = %q, want %q", got, "779a65e7023cd2e7")
	}
}

func TestHexEncodeUint64_SmallValue(t *testing.T) {
	t.Parallel()

	if got := hexEncodeUint64(255); got != "00000000000000ff" {
		t.Errorf("hexEncodeUint64(255) = %q, want %q", got, "00000000000000ff")
	}
}

func TestHexEncodeUint64_AlwaysSixteenChars(t *testing.T) {
	t.Parallel()

	values := []uint64{
		0, 1, 0xff, 0x100, 0xffff, 0x10000, 0xffffffff, 0x100000000,
		0xffffffffffffffff, 0x123456789abcdef0,
	}

	for _, v := range values {
		got := hexEncodeUint64(v)
		if len(got) != hexUint64Chars {
			t.Errorf("hexEncodeUint64(%#x) length = %d, want %d", v, len(got), hexUint64Chars)
		}
	}
}
