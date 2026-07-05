package httputil

// hexDigitsLower is the lowercase hex alphabet (0-9, a-f) shared by the ETag
// encoder and the request-ID generator. Indexed by a 4-bit nibble to produce
// a single hex character byte.
const hexDigitsLower = "0123456789abcdef"
