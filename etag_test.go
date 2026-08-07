package httputil

import (
	"net/http"
	"testing"

	etag "github.com/larsartmann/go-etag"
)

func TestETag_GeneratesHeader(t *testing.T) {
	t.Parallel()

	handler := ETag(etag.DefaultETagConfig())(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	if rec.Header().Get("ETag") == "" {
		t.Error("ETag header is empty, want generated ETag")
	}
}

func TestETag_IfNoneMatch_Returns304(t *testing.T) {
	t.Parallel()

	handler := ETag(etag.DefaultETagConfig())(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("If-None-Match", `"779a65e7023cd2e7"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)
}

func TestETag_PostRequest_NoETag(t *testing.T) {
	t.Parallel()

	handler := ETag(etag.DefaultETagConfig())(newWriteStatusHandler("hello world"))

	req := newTestRequest(http.MethodPost, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	if rec.Header().Get("ETag") != "" {
		t.Error("ETag header is set for POST, want empty")
	}
}

func TestMiddlewareETag_Constant(t *testing.T) {
	t.Parallel()

	if MiddlewareETag != "etag" {
		t.Errorf("MiddlewareETag = %q, want %q", MiddlewareETag, "etag")
	}
}

func TestETag_WorksInChain(t *testing.T) {
	t.Parallel()

	inner := newWriteStatusHandler("hello world")

	handler := Chain(inner, ETag(etag.DefaultETagConfig()))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	if rec.Header().Get("ETag") == "" {
		t.Error("ETag header is empty through Chain")
	}
}

func TestETag_WorksInMiddlewareStack(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()

	err := stack.Add(MiddlewareETag, ETag(etag.DefaultETagConfig()))
	if err != nil {
		t.Fatalf("stack.Add failed: %v", err)
	}

	if stack.Names()[0] != MiddlewareETag {
		t.Errorf("stack.Names()[0] = %q, want %q", stack.Names()[0], MiddlewareETag)
	}
}
