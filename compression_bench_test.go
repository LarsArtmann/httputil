package httputil

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"testing"
)

// FuzzCompression verifies the compression middleware never panics and always
// produces a decodable response for arbitrary bodies and Accept-Encoding
// headers: when the response carries the negotiated gzip encoding, gunzipping
// it must reproduce the handler's body exactly.
//
// The decode-and-compare check relies on Go's gzip reader treating
// concatenated members as a multistream: if the writer ever emitted the
// payload twice as separate gzip members (the exact-fill duplication bug),
// the decoded bytes would be body+body and the byte comparison would fail
// instead of silently passing on the first member.
func FuzzCompression(f *testing.F) {
	f.Add([]byte("hello world"), "gzip")
	f.Add([]byte(strings.Repeat("a", 1024)), "gzip, deflate")
	f.Add([]byte(""), "")

	cfg := DefaultCompressionConfig()

	f.Fuzz(func(t *testing.T, body []byte, acceptEncoding string) {
		handler := Compression(cfg)(http.HandlerFunc(func(
			resp http.ResponseWriter,
			req *http.Request,
		) {
			resp.WriteHeader(http.StatusOK)
			_, _ = resp.Write(body)
		}))

		req := newTestRequest(http.MethodGet, "/", "")
		req.Header.Set(headerAcceptEncoding, acceptEncoding)

		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		assertStatus(t, rec, http.StatusOK)

		if got := rec.Header().Get(headerContentEncoding); got == encodingGzip {
			gzipDecoder, err := gzip.NewReader(rec.Body)
			if err != nil {
				t.Errorf("gzip.NewReader on compressed response: %v", err)

				return
			}

			decoded, err := io.ReadAll(gzipDecoder)
			if err != nil {
				t.Errorf("gzip decode of compressed response: %v", err)

				return
			}

			if !bytes.Equal(decoded, body) {
				t.Errorf(
					"round-trip mismatch: decoded %d bytes, want %d",
					len(decoded),
					len(body),
				)
			}
		}
	})
}

func BenchmarkCompression(b *testing.B) {
	cfg := DefaultCompressionConfig()
	middleware := Compression(cfg)

	body := []byte(strings.Repeat("a", defaultCompressionMinSize*2))

	inner := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write(body)
	})

	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	b.ReportAllocs()

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}
