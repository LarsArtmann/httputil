package httputil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// --- ETag-specific test helpers ---

// newStatusBodyHandler returns an http.HandlerFunc that writes the given
// status code and body. Used by ETag tests that need non-200 status codes.
func newStatusBodyHandler(status int, body string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)

		_, _ = w.Write([]byte(body))
	})
}

// failingWriteRecorder is an httptest.ResponseRecorder whose Write always
// fails, exercising the streaming-write and overflow-write error branches
// that require the underlying ResponseWriter to reject data.
type failingWriteRecorder struct {
	*httptest.ResponseRecorder
}

func (*failingWriteRecorder) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

// nonHijackableRecorder is a minimal http.ResponseWriter that does NOT
// implement http.Hijacker, exercising the "hijack unsupported" error path.
type nonHijackableRecorder struct {
	header http.Header
	status int
	body   []byte
}

func newNonHijackableRecorder() *nonHijackableRecorder {
	return &nonHijackableRecorder{header: http.Header{}}
}

func (r *nonHijackableRecorder) Header() http.Header { return r.header }

func (r *nonHijackableRecorder) WriteHeader(code int) { r.status = code }

func (r *nonHijackableRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)

	return len(b), nil
}

func assertETag(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()

	if got := rec.Header().Get(headerETag); got != want {
		t.Errorf("ETag = %q, want %q", got, want)
	}
}

func assertETagAbsent(t *testing.T, rec *httptest.ResponseRecorder, msg string) {
	t.Helper()

	if got := rec.Header().Get(headerETag); got != "" {
		t.Errorf("ETag = %q, want empty %s", got, msg)
	}
}

// serveGetWithIfNoneMatch wraps the default ETag middleware around a body
// handler, issues a GET request with the given If-None-Match header value,
// serves it, and returns the recorder.
func serveGetWithIfNoneMatch(t *testing.T, ifNoneMatch string) *httptest.ResponseRecorder {
	t.Helper()

	handler := ETag(DefaultETagConfig())(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, ifNoneMatch)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	return rec
}

// --- ETag generation ---

func TestETag_GeneratesStrongEntityTag(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertETag(t, rec, `"779a65e7023cd2e7"`)
}

func TestETag_GeneratesWeakEntityTag(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{Strength: EntityTagWeak}
	handler := ETag(cfg)(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETag(t, rec, `W/"779a65e7023cd2e7"`)
}

func TestETag_EmptyBody(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newWriteStatusHandler(""))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETag(t, rec, `"cbf29ce484222325"`)
}

// --- If-None-Match ---

func TestETag_IfNoneMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		header  string
		want304 bool
	}{
		{name: "ExactMatch", header: `"779a65e7023cd2e7"`, want304: true},
		{name: "Wildcard", header: "*", want304: true},
		{name: "ListContainsMatch", header: `"other", "779a65e7023cd2e7", "another"`, want304: true},
		{name: "WeakClientStrongServer", header: `W/"779a65e7023cd2e7"`, want304: true},
		{name: "ListContainsWeakMatch", header: `"other", W/"779a65e7023cd2e7", "another"`, want304: true},
		{name: "StrongClientNoMatch", header: `"different"`, want304: false},
		{name: "WeakClientNoMatch", header: `W/"different"`, want304: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := serveGetWithIfNoneMatch(t, tt.header)

			if tt.want304 {
				assertStatus(t, rec, http.StatusNotModified)
				assertBodyEmpty(t, rec, "for 304")
			} else {
				assertStatus(t, rec, http.StatusOK)
				assertBody(t, rec, "hello world")
			}
		})
	}
}

func TestETag_IfNoneMatch_StrongClientWeakServer(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{Strength: EntityTagWeak}
	handler := ETag(cfg)(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)
	assertBodyEmpty(t, rec, "for strong If-None-Match against weak server ETag")
}

func TestETag_IfNoneMatch_MultipleHeaders(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Add(headerIfNoneMatch, `"other"`)
	req.Header.Add(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)
	assertBodyEmpty(t, rec, "for 304 with multiple If-None-Match headers")
}

func TestETag_NoIfNoneMatchHeader(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertBody(t, rec, "hello world")
	assertETag(t, rec, `"779a65e7023cd2e7"`)
}

func TestETag_IfNoneMatch_EscapedQuoteEndToEnd(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, `"a\"b", "c"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertBody(t, rec, "hello world")
}

// --- HTTP methods ---

func TestETag_NonGetHead(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodPost, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertETagAbsent(t, rec, "for POST")
}

func TestETag_HeadRequest_GeneratesETag(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodHead, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get(headerETag)
	if etag == "" {
		t.Error("ETag header is empty, want generated ETag for HEAD")
	}
}

// RFC 7230 §3.3: HEAD responses MUST NOT include a message body.
func TestETag_HeadRequest_NoBody(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodHead, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertBodyEmpty(t, rec, "HEAD must not have a body")

	// Content-Length should reflect the representation size.
	if cl := rec.Header().Get(headerContentLength); cl != "11" {
		t.Errorf("Content-Length = %q, want %q for HEAD", cl, "11")
	}
}

func TestETag_HeadRequest_IfNoneMatch(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodHead, "/", "")
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)
	assertBodyEmpty(t, rec, "for HEAD 304")
}

// --- Status codes ---

func TestETag_Status201_IsCacheable(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newStatusBodyHandler(http.StatusCreated, "created"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get(headerETag)
	if etag == "" {
		t.Error("ETag header is empty, want generated ETag for 201")
	}
}

// Non-cacheable status codes (3xx+) must never return 304, even when
// If-None-Match matches the body hash. The ETag is still set for client
// caching, but the full response is always sent.
func TestETag_NonCacheableStatus_NeverReturns304(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
	}{
		{name: "301 Moved Permanently", status: http.StatusMovedPermanently},
		{name: "302 Found", status: http.StatusFound},
		{name: "304 Not Modified", status: http.StatusNotModified},
		{name: "400 Bad Request", status: http.StatusBadRequest},
		{name: "404 Not Found", status: http.StatusNotFound},
		{name: "500 Internal Server Error", status: http.StatusInternalServerError},
		{name: "503 Service Unavailable", status: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := ETag(DefaultETagConfig())(newStatusBodyHandler(tt.status, "hello world"))

			req := newTestRequest(http.MethodGet, "/", "")
			req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

			rec := newRecorder()
			handler.ServeHTTP(rec, req)

			assertStatus(t, rec, tt.status)
			assertBody(t, rec, "hello world")
		})
	}
}

// Boundary: 299 is the last cacheable status (200-299 range).
func TestETag_Status299_IsCacheable(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newStatusBodyHandler(status299Cacheable, "hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)
	assertBodyEmpty(t, rec, "299 is cacheable")
}

func TestETag_Status300_NotCacheable(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newStatusBodyHandler(http.StatusMultipleChoices, "hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusMultipleChoices)
	assertBody(t, rec, "hello world")
}

// --- Buffer overflow ---

func TestETag_MemoryLimit_DisablesETag(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: etagOverflowTestLimit}
	handler := ETag(cfg)(newWriteStatusHandler("this body exceeds the limit"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertETagAbsent(t, rec, "when buffer limit exceeded")
	assertBody(t, rec, "this body exceeds the limit")
}

func TestETag_Overflow_IfNoneMatch(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: etagOverflowTestLimit}
	handler := ETag(cfg)(newWriteStatusHandler("this body exceeds the limit"))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, `"anything"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertBody(t, rec, "this body exceeds the limit")
	assertETagAbsent(t, rec, "when buffer limit exceeded")
}

// Zero-value ETagConfig must not cause unbounded buffering.
func TestETag_ZeroValueConfig_ClampsBufferSize(t *testing.T) {
	t.Parallel()

	// ETagConfig{} has MaxBufferSize=0, which must be clamped to the default.
	// Without clamping, the overflow guard in Write would be disabled.
	cfg := ETagConfig{}
	handler := ETag(cfg)(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	// A clamped config generates an ETag; an unbounded one would too, but the
	// important thing is that it doesn't panic or misbehave.
	etag := rec.Header().Get(headerETag)
	if etag == "" {
		t.Error("ETag header is empty with zero-value config, want clamped default behavior")
	}
}

// --- 304 response correctness ---

// RFC 7232 §4.1: 304 Not Modified MUST NOT include Content-Length.
func TestETag_304_ExcludesContentLength(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentLength, "11")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)

	if cl := rec.Header().Get(headerContentLength); cl != "" {
		t.Errorf("Content-Length = %q on 304, want empty", cl)
	}
}

func TestETag_304_IncludesETagHeader(t *testing.T) {
	t.Parallel()

	rec := serveGetWithIfNoneMatch(t, `"779a65e7023cd2e7"`)

	assertStatus(t, rec, http.StatusNotModified)

	if rec.Header().Get(headerETag) == "" {
		t.Error("ETag header is empty on 304, want the generated ETag")
	}
}

// --- Flush / Hijack ---

func TestETag_Flush(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(newFlushHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETagAbsent(t, rec, "after flush")
	assertBody(t, rec, "partial more")
}

func TestETag_FlushAlreadyFlushed(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("data"))

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Flusher")
		}

		flusher.Flush()
		flusher.Flush()
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)
}

func TestETag_Hijack_SetsFlushedMode(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()
	writer := newETagWriter(newHijackRecorder(), cfg)

	_, _, err := writer.Hijack()
	if err != nil {
		t.Fatalf("Hijack() error = %v, want nil", err)
	}

	if !writer.flushed {
		t.Error("Hijack should set flushed mode")
	}
}

func TestETag_Hijack_NoETag(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		hj, ok := rw.(http.Hijacker)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Hijacker")
		}

		_, _, _ = hj.Hijack()
	}))

	rec := newHijackRecorder()
	req := newTestRequest(http.MethodGet, "/", "")

	handler.ServeHTTP(rec, req)

	assertETagAbsent(t, rec.ResponseRecorder, "after hijack")
}

// --- SkipIfPresent ---

func TestETag_SkipIfPresent_RespectsHandlerETag(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{SkipIfPresent: true}
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerETag, `"my-revision-42"`)
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETag(t, rec, `"my-revision-42"`)
}

func TestETag_SkipIfPresent_Allows304WithHandlerETag(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{SkipIfPresent: true}
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerETag, `"my-revision-42"`)
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, `"my-revision-42"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)
}

func TestETag_SkipIfPresent_FallsBackWhenNoHandlerETag(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{SkipIfPresent: true}
	handler := ETag(cfg)(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	// No handler-set ETag, so middleware computes one.
	assertETag(t, rec, `"779a65e7023cd2e7"`)
}

func TestETag_SkipIfPresent_MalformedHandlerETag_FallsBack(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{SkipIfPresent: true}
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerETag, "not-a-valid-etag")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	// Malformed handler ETag cannot be parsed, so middleware computes one.
	assertETag(t, rec, `"779a65e7023cd2e7"`)
}

func TestETag_SkipIfPresent_EmptyHandlerETag_FallsBack(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{SkipIfPresent: true}
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerETag, "")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETag(t, rec, `"779a65e7023cd2e7"`)
}

// --- Skip predicate ---

func TestETag_SkipPredicate(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{
		Skip: func(r *http.Request) bool {
			return r.URL.Path == "/stream"
		},
	}
	handler := ETag(cfg)(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/stream", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETagAbsent(t, rec, "when Skip returns true")
	assertBody(t, rec, "hello world")
}

func TestETag_SkipPredicate_False(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{
		Skip: func(r *http.Request) bool {
			return false
		},
	}
	handler := ETag(cfg)(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETag(t, rec, `"779a65e7023cd2e7"`)
}

// --- Validate ---

func TestETagConfig_Validate_ValidDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestETagConfig_Validate_ZeroMaxBufferSize(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: 0}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for zero MaxBufferSize")
	}

	if !errors.Is(err, ErrETagConfig) {
		t.Errorf("Validate() error = %v, want ErrETagConfig", err)
	}
}

func TestETagConfig_Validate_NegativeMaxBufferSize(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: -1}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative MaxBufferSize")
	}

	if !errors.Is(err, ErrETagConfig) {
		t.Errorf("Validate() error = %v, want ErrETagConfig", err)
	}
}

func TestETagConfig_Validate_InvalidStrength(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{Strength: EntityTagStrength(strengthInvalidValue)}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for invalid Strength")
	}

	if !errors.Is(err, ErrETagConfig) {
		t.Errorf("Validate() error = %v, want ErrETagConfig", err)
	}
}

// --- Error handling ---

func TestETag_OnError_StreamingWriteFailure(t *testing.T) {
	t.Parallel()

	var capturedErr *errorfamily.Error

	cfg := ETagConfig{
		OnError: func(e *errorfamily.Error) {
			capturedErr = e
		},
	}

	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("first chunk"))

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Flusher")
		}

		flusher.Flush()

		_, _ = w.Write([]byte("second chunk"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := &failingWriteRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.ServeHTTP(rec, req)

	if capturedErr == nil {
		t.Fatal("OnError was not called on streaming write failure")
	}

	if capturedErr.ErrorFamily() != errorfamily.Transient {
		t.Errorf("OnError error family = %s, want Transient", capturedErr.ErrorFamily())
	}
}

func TestETag_OverflowWriteError(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: etagOverflowTestLimit}

	var writeErr error
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, writeErr = w.Write([]byte(strings.Repeat("x", etagOverflowBodySize)))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := &failingWriteRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.ServeHTTP(rec, req)

	if writeErr == nil {
		t.Fatal("Write returned nil error for overflow to failing writer")
	}

	var classified *errorfamily.Error
	if !errors.As(writeErr, &classified) {
		t.Fatalf("Write error is %T, want *errorfamily.Error", writeErr)
	}

	if classified.Code() != ErrCodeETagWriteFailed {
		t.Errorf("Write error code = %q, want %q", classified.Code(), ErrCodeETagWriteFailed)
	}
}

func TestETag_OverflowWriteError_NilOnError_DoesNotPanic(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: etagOverflowTestLimit}

	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(strings.Repeat("x", etagOverflowBodySize)))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := &failingWriteRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.ServeHTTP(rec, req)
}

func TestETag_FlushWriteError_NilOnError_DoesNotPanic(t *testing.T) {
	t.Parallel()

	handler := ETag(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Flusher")
		}

		flusher.Flush()

		_, _ = w.Write([]byte("after flush"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := &failingWriteRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.ServeHTTP(rec, req)
}

// --- Custom HashFunc ---

func TestETag_CustomHashFunc(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{
		HashFunc: func(_ []byte) string {
			return "custom-hash"
		},
	}
	handler := ETag(cfg)(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETag(t, rec, `"custom-hash"`)
}

func TestETag_CustomHashFunc_ReceivesBody(t *testing.T) {
	t.Parallel()

	var receivedBody []byte
	cfg := ETagConfig{
		HashFunc: func(b []byte) string {
			receivedBody = b

			return "from-body"
		},
	}
	handler := ETag(cfg)(newWriteStatusHandler("the actual body"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if string(receivedBody) != "the actual body" {
		t.Errorf("HashFunc received %q, want %q", string(receivedBody), "the actual body")
	}
}

func TestETag_EmptyHandler_NoETag(t *testing.T) {
	t.Parallel()

	// A handler that writes nothing and never calls WriteHeader hits the
	// computeETag early return for empty body + no buffered header.
	handler := ETag(DefaultETagConfig())(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertETagAbsent(t, rec, "for handler that writes nothing")
	assertBodyEmpty(t, rec, "for handler that writes nothing")
}

func TestETag_HandlerWriteHeaderOnly_WithBody(t *testing.T) {
	t.Parallel()

	// A handler that calls WriteHeader but writes no body should still get
	// an ETag (the empty body hash).
	handler := ETag(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	// Empty body still produces an ETag (FNV-64a of empty string).
	assertETag(t, rec, `"cbf29ce484222325"`)
}

// --- Fuzz ---

func FuzzETag(f *testing.F) {
	f.Add([]byte("hello world"), "")
	f.Add([]byte(strings.Repeat("a", defaultETagMaxBufferSize)), `"abc123"`)
	f.Add([]byte(""), "*")
	f.Add([]byte("response body"), `W/"779a65e7023cd2e7"`)
	f.Add([]byte("response body"), `"779a65e7023cd2e7", W/"deadbeef"`)
	f.Add([]byte("body"), `"a\"b"`)
	f.Add([]byte("body"), `"a\"b", "c"`)
	f.Add([]byte("body"), `"a,b\"c"`)

	cfg := DefaultETagConfig()
	inner := newStatusBodyHandler(http.StatusOK, "response body")

	f.Fuzz(func(t *testing.T, body []byte, ifNoneMatch string) {
		handler := ETag(cfg)(inner)

		req := newTestRequest(http.MethodGet, "/", "")
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.Header.Set(headerIfNoneMatch, ifNoneMatch)

		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK && rec.Code != http.StatusNotModified {
			t.Errorf("status = %d, want 200 or 304", rec.Code)
		}
	})
}

// --- Benchmarks ---

func BenchmarkETag(b *testing.B) {
	handler := ETag(DefaultETagConfig())(newWriteBodyHandler([]byte("hello world benchmark test data")))

	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkETag_IfNoneMatch(b *testing.B) {
	handler := ETag(DefaultETagConfig())(newWriteBodyHandler([]byte("hello world benchmark test data")))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkMatchesIfNoneMatch(b *testing.B) {
	single := `"779a65e7023cd2e7"`
	multi := `"abc123", W/"def456", "779a65e7023cd2e7"`
	weak := `W/"779a65e7023cd2e7"`
	tag := NewEntityTag("779a65e7023cd2e7", EntityTagStrong)

	b.Run("single", func(b *testing.B) {
		for b.Loop() {
			_ = MatchesIfNoneMatch(tag, single)
		}
	})

	b.Run("multi", func(b *testing.B) {
		for b.Loop() {
			_ = MatchesIfNoneMatch(tag, multi)
		}
	})

	b.Run("weak", func(b *testing.B) {
		for b.Loop() {
			_ = MatchesIfNoneMatch(tag, weak)
		}
	})
}

// --- Examples ---

func ExampleETag() {
	handler := ETag(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("ETag") != "")

	// Output: true
}

func ExampleEntityTag() {
	tag := NewEntityTag("abc123", EntityTagStrong)

	fmt.Println(tag)
	fmt.Println(tag.IsWeak())

	weak := NewEntityTag("abc123", EntityTagWeak)
	fmt.Println(weak)

	// Output:
	// "abc123"
	// false
	// W/"abc123"
}

// --- Test constants ---

const (
	status299Cacheable    = 299
	etagOverflowTestLimit = 10
	etagOverflowBodySize  = 100
	strengthInvalidValue  = 42
)
