package httputil

import (
	"encoding/base64"
	"strings"
	"testing"
)

func FuzzNonce(f *testing.F) {
	f.Add(minNonceSize)
	f.Add(defaultNonceSize)
	f.Add(32)
	f.Add(64)

	f.Fuzz(func(t *testing.T, size int) {
		t.Parallel()

		if size < 1 || size > 1024 {
			t.Skip()
		}

		nonce := generateNonce(size)

		decoded, err := base64.RawURLEncoding.DecodeString(nonce)
		if err != nil {
			t.Fatalf("nonce %q is not valid base64: %v", nonce, err)
		}

		if len(decoded) != size {
			t.Fatalf("decoded length %d, want %d", len(decoded), size)
		}

		for _, csp := range []string{
			RecommendedCSPWithNonce(nonce),
			ProductionCSPWithNonce(nonce),
		} {
			if strings.ContainsAny(csp, "\r\n") {
				t.Fatalf("CSP contains CRLF (header injection): %q", csp)
			}
		}
	})
}
