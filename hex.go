package httputil

// hexDigitsLower is the lowercase hex alphabet (0-9, a-f) used by the
// request-ID generator and ETag encoder. Indexed by a 4-bit nibble to
// produce a single hex character byte.
const hexDigitsLower = "0123456789abcdef"

const (
	hexNibbleShift  = 4
	hexNibbleMask   = 0x0f
	hexBitsPerByte  = 8
	hexCharsPerByte = 2
	hexUint64Bytes  = 8
	hexUint64Chars  = hexUint64Bytes * hexCharsPerByte // 16.
)

// hexEncodeUint64 returns the lowercase hex encoding of v as a 16-character
// string. The encoding is written into a stack-allocated array and converted
// to a string in a single allocation, avoiding the overhead of strings.Builder.
func hexEncodeUint64(v uint64) string {
	var buf [hexUint64Chars]byte

	for i := hexUint64Bytes - 1; i >= 0; i-- {
		buf[i*hexCharsPerByte+1] = hexDigitsLower[v&hexNibbleMask]
		buf[i*hexCharsPerByte] = hexDigitsLower[(v>>hexNibbleShift)&hexNibbleMask]
		v >>= hexBitsPerByte
	}

	return string(buf[:])
}
