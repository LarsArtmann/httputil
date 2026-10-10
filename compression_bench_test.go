package httputil

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
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
	f.Add([]byte("hello world"), "gzip", "", false)
	f.Add([]byte(strings.Repeat("a", 1024)), "gzip, deflate", "", false)
	f.Add([]byte(""), "", "", false)
	f.Add([]byte("range passthrough"), "gzip", "bytes=0-4", false)
	f.Add([]byte("range passthrough"), "", "bytes=-5", false)
	f.Add([]byte(strings.Repeat("a", 2048)), "", "", true)
	f.Add([]byte(strings.Repeat("a", 2048)), "gzip", "", true)

	cfg := DefaultCompressionConfig()
	legacyCfg := DefaultCompressionConfig()
	legacyCfg.AbsentEncoding = AbsentEncodingFirstConfigured

	f.Fuzz(func(t *testing.T, body []byte, acceptEncoding, rangeHeader string, absentFirst bool) {
		activeCfg := cfg
		if absentFirst {
			activeCfg = legacyCfg
		}

		handler := Compression(activeCfg)(http.HandlerFunc(func(
			resp http.ResponseWriter,
			req *http.Request,
		) {
			resp.WriteHeader(http.StatusOK)
			_, _ = resp.Write(body)
		}))

		req := newTestRequest(http.MethodGet, "/", "")

		if acceptEncoding != "" {
			req.Header.Set(headerAcceptEncoding, acceptEncoding)
		}

		if rangeHeader != "" {
			req.Header.Set(headerRange, rangeHeader)
		}

		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		assertStatus(t, rec, http.StatusOK)

		if rangeHeader != "" {
			if got := rec.Header().Get(headerContentEncoding); got != "" {
				t.Fatalf("Range request Content-Encoding = %q, want empty passthrough", got)
			}

			if !bytes.Equal(rec.Body.Bytes(), body) {
				t.Errorf(
					"Range passthrough mismatch: got %d bytes, want the verbatim %d-byte body",
					rec.Body.Len(),
					len(body),
				)
			}

			return
		}

		if acceptEncoding == "" {
			assertHeaderlessAbsentEncoding(t, rec, body, absentFirst)

			return
		}

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

// assertHeaderlessAbsentEncoding pins the AbsentEncoding invariant for
// header-less requests: compression happens exactly under the legacy
// FirstConfigured policy, and only for bodies above the MinSize threshold;
// compressed bytes must round-trip.
func assertHeaderlessAbsentEncoding(t *testing.T, rec *httptest.ResponseRecorder, body []byte, absentFirst bool) {
	t.Helper()

	if len(body) <= defaultCompressionMinSize {
		return
	}

	gotCE := rec.Header().Get(headerContentEncoding)
	if absentFirst && gotCE != encodingGzip {
		t.Fatalf("AbsentEncodingFirstConfigured header-less Content-Encoding = %q, want gzip", gotCE)
	}

	if !absentFirst && gotCE != "" {
		t.Fatalf("AbsentEncodingIdentity header-less Content-Encoding = %q, want empty", gotCE)
	}

	if gotCE != encodingGzip {
		return
	}

	gzipDecoder, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader on header-less compressed response: %v", err)
	}

	defer func() { _ = gzipDecoder.Close() }()

	decoded, err := io.ReadAll(gzipDecoder)
	if err != nil {
		t.Fatalf("gzip decode of header-less compressed response: %v", err)
	}

	if !bytes.Equal(decoded, body) {
		t.Errorf("header-less round-trip mismatch: decoded %d bytes, want %d", len(decoded), len(body))
	}
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
