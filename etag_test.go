package httputil

import (
	"net/http"
	"testing"
)

func TestETag_GeneratesStrongETag(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	etag := rec.Header().Get(headerETag)
	if etag == "" {
		t.Error("ETag header is empty, want generated ETag")
	}

	if etag != `"0d4a1185"` {
		t.Errorf("ETag = %q, want %q", etag, `"0d4a1185"`)
	}
}

func TestETag_GeneratesWeakETag(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{Weak: true}
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get(headerETag)
	if etag != `W/"0d4a1185"` {
		t.Errorf("ETag = %q, want %q", etag, `W/"0d4a1185"`)
	}
}

func TestETag_IfNoneMatch_ListContainsMatch(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, `"other", "0d4a1185", "another"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotModified)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("body length = %d, want 0 for 304", rec.Body.Len())
	}
}

func TestETag_IfNoneMatch_Matches(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, `"0d4a1185"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotModified)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("body length = %d, want 0 for 304", rec.Body.Len())
	}
}

func TestETag_IfNoneMatch_NoMatch(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, `"different"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Body.String(); got != "hello world" {
		t.Errorf("body = %q, want %q", got, "hello world")
	}
}

func TestETag_IfNoneMatch_Star(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, "*")

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotModified)
	}
}

func TestETag_NonGetHead(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodPost, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get(headerETag); got != "" {
		t.Errorf("ETag = %q, want empty for POST", got)
	}
}

func TestETag_201Created_IsCacheable(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get(headerETag)
	if etag == "" {
		t.Error("ETag header is empty, want generated ETag for 201")
	}
}

func TestETag_MemoryLimit_DisablesETag(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: 10}
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("this body exceeds the limit"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get(headerETag); got != "" {
		t.Errorf("ETag = %q, want empty when buffer limit exceeded", got)
	}

	if got := rec.Body.String(); got != "this body exceeds the limit" {
		t.Errorf("body = %q, want %q", got, "this body exceeds the limit")
	}
}

func TestETag_EmptyBody(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get(headerETag)
	if etag == "" {
		t.Error("ETag header is empty, want generated ETag for empty body")
	}

	if etag != `"00000000"` {
		t.Errorf("ETag = %q, want %q", etag, `"00000000"`)
	}
}

func TestETag_Flush(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()
	handler := ETag(cfg)(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write([]byte("partial"))

		if f, ok := resp.(http.Flusher); ok {
			f.Flush()
		}

		_, _ = resp.Write([]byte(" more"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(headerETag); got != "" {
		t.Errorf("ETag = %q, want empty after flush", got)
	}

	if got := rec.Body.String(); got != "partial more" {
		t.Errorf("body = %q, want %q", got, "partial more")
	}
}

func TestETag_HeadRequest(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()
	handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodHead, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get(headerETag)
	if etag == "" {
		t.Error("ETag header is empty, want generated ETag for HEAD")
	}
}
