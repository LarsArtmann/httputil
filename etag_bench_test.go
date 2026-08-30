package httputil

import (
	"net/http"
	"testing"

	etag "github.com/larsartmann/go-etag/server"
)

// BenchmarkETagAdapterOverhead proves the deprecated httputil.ETag adapter is
// a zero-cost passthrough over etag.New: construction is the same expression
// and the serving path is byte-identical, so the adapter exists only for
// composition ergonomics. The bare baseline shows the middleware's absolute
// per-request cost for context.
func BenchmarkETagAdapterOverhead(b *testing.B) {
	cases := []struct {
		name  string
		build func() Middleware
	}{
		{
			name:  "baselineNoMiddleware",
			build: nil,
		},
		{
			name:  "directEtagNew",
			build: func() Middleware { return etag.New(etag.DefaultETagConfig()) },
		},
		{
			name:  "httputilAdapter",
			build: func() Middleware { return ETag(etag.DefaultETagConfig()) },
		},
	}

	inner := newWriteStatusHandler("etag adapter benchmark body")

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			handler := http.Handler(inner)
			if tc.build != nil {
				handler = tc.build()(inner)
			}

			req := newTestRequest(http.MethodGet, "/", "")

			b.ReportAllocs()

			for b.Loop() {
				rec := newRecorder()
				handler.ServeHTTP(rec, req)

				if rec.Code != http.StatusOK {
					b.Fatalf("unexpected status %d", rec.Code)
				}
			}
		})
	}
}
