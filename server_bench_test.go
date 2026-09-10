package httputil

import (
	"crypto/tls"
	"testing"
)

// BenchmarkServerConfigValidateWithTLS measures ServerConfig validation with
// a populated TLSConfig, the heaviest Validate path.
func BenchmarkServerConfigValidateWithTLS(b *testing.B) {
	cfg := DefaultServerConfig()
	cfg.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS13}

	b.ReportAllocs()

	for b.Loop() {
		if err := cfg.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}
