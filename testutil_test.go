package httputil

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// newNoOpHandler returns an http.HandlerFunc that does nothing.
func newNoOpHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

// newCountingHandler returns an http.HandlerFunc that sets called to true.
func newCountingHandler(called *bool) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
	})
}

// newTestRequest creates an httptest.Request with context and Origin header set.
func newTestRequest(method, path, origin string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}

	return req
}

// newRecorder creates a new httptest.ResponseRecorder.
func newRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

// newTestLogger returns a slog.Logger that writes to a discard buffer.
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

// newAppendingHandler returns an http.HandlerFunc that appends val to s.
func newAppendingHandler(s *[]string, val string) http.HandlerFunc {
	return http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		*s = append(*s, val)
	})
}

// assertHeader checks that a response recorder has the expected header value.
func assertHeader(t *testing.T, rec *httptest.ResponseRecorder, key, want string) {
	t.Helper()

	if got := rec.Header().Get(key); got != want {
		t.Errorf("%s = %q, want %q", key, got, want)
	}
}

// newWriteStatusHandler returns an http.HandlerFunc that writes StatusOK and body.
func newWriteStatusHandler(body string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})
}

// newTypedBodyHandler returns an http.HandlerFunc that sets Content-Type and
// writes body with StatusOK. Used to construct content-typed handlers for
// compression tests that need a specific Content-Type to drive the
// incompressible-type filter.
func newTypedBodyHandler(contentType, body string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})
}

// newStatusOnlyHandler returns an http.HandlerFunc that writes only the given
// status code, without a body. Useful when a test asserts on rate-limit /
// status-only middleware behavior and the body is irrelevant.
func newStatusOnlyHandler(status int) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	})
}

// newWriteLargeBodyHandler returns an http.HandlerFunc that writes StatusOK
// and a body of size defaultCompressionMinSize+1, which is just above the
// compression threshold used in middleware tests.
func newWriteLargeBodyHandler() http.HandlerFunc {
	return newWriteStatusHandler(
		strings.Repeat("a", defaultCompressionMinSize+1),
	)
}

// newWriteBodyHandler returns an http.HandlerFunc that writes OK status and body.
func newWriteBodyHandler(body []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

// newFlushHandler returns an http.HandlerFunc that writes "partial", flushes
// if the ResponseWriter implements http.Flusher, then writes " more".
func newFlushHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		_, _ = w.Write([]byte(" more"))
	})
}

// hijackRecorder is an httptest.ResponseRecorder that also implements http.Hijacker.
type hijackRecorder struct {
	*httptest.ResponseRecorder

	hijacked bool
}

func newHijackRecorder() *hijackRecorder {
	return &hijackRecorder{ResponseRecorder: httptest.NewRecorder()}
}

func (r *hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	r.hijacked = true

	return nil, nil, nil
}

// errHijackFailed is the sentinel returned by failingHijacker.Hijack.
var errHijackFailed = errors.New("hijack failed")

// failingHijacker is an http.ResponseWriter that implements http.Hijacker
// whose Hijack always fails. Used to exercise the ErrCodeHijackFailed path.
type failingHijacker struct {
	*httptest.ResponseRecorder
}

func newFailingHijacker() failingHijacker {
	return failingHijacker{ResponseRecorder: httptest.NewRecorder()}
}

func (failingHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errHijackFailed
}

// assertStatus checks that a response recorder has the expected status code.
func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()

	if rec.Code != want {
		t.Errorf("status = %d, want %d", rec.Code, want)
	}
}

// assertBody checks that a response recorder body matches the expected string.
func assertBody(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()

	if got := rec.Body.String(); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

// assertBodyEmpty checks that a response recorder has no body, formatted with
// msg to clarify the test intent (e.g. "for 304").
func assertBodyEmpty(t *testing.T, rec *httptest.ResponseRecorder, msg string) {
	t.Helper()

	if rec.Body.Len() != 0 {
		t.Errorf("body length = %d, want 0 %s", rec.Body.Len(), msg)
	}
}

// waitForServerStart blocks until errChan receives an error or the timeout
// elapses. It fails the test if an error is received and silently returns on
// timeout, indicating the server started successfully.
func waitForServerStart(t *testing.T, errChan <-chan error, timeout time.Duration) {
	t.Helper()

	select {
	case err := <-errChan:
		t.Fatalf("server failed to start: %v", err)
	case <-time.After(timeout):
		// Server started successfully.
	}
}

// assertNegotiatedEncoding runs the negotiator on header, fails the test on
// negotiation failure, and verifies the result matches wantEncoding. The
// contextMsg is appended to the failure message to clarify the test intent
// (e.g. "gzip should be disabled by q=0").
func assertNegotiatedEncoding(
	t *testing.T,
	neg *negotiator,
	header, wantEncoding, contextMsg string,
) {
	t.Helper()

	encoding, _, ok := neg.negotiateEncoding(header)
	if !ok {
		t.Fatalf("negotiation failed: %s", contextMsg)
	}

	if encoding != wantEncoding {
		t.Errorf("encoding = %q, want %q (%s)", encoding, wantEncoding, contextMsg)
	}
}

// newTestNegotiator returns a negotiator built from the default writer
// factories, the standard fixture for negotiation tests.
func newTestNegotiator() *negotiator {
	return buildNegotiator(DefaultWriterFactories())
}

// assertClassified verifies err belongs to wantFamily and that retryability
// matches wantRetryable. Used by error-classification tests across the
// recorder, hijack, and compression paths.
func assertClassified(t *testing.T, err error, wantFamily errorfamily.Family, wantRetryable bool) {
	t.Helper()

	if got := errorfamily.Classify(err); got != wantFamily {
		t.Errorf("Classify(err) = %v, want %v", got, wantFamily)
	}

	if got := errorfamily.IsRetryable(err); got != wantRetryable {
		t.Errorf("IsRetryable(err) = %v, want %v", got, wantRetryable)
	}
}

// newRequestIDConfigForTest returns a RequestIDConfig with a stub ID
// generator and the supplied header names. Used by validation tests where
// the exact header field under test is the only varying input.
func newRequestIDConfigForTest(responseHeader, incomingHeader string) RequestIDConfig {
	return RequestIDConfig{
		ResponseHeader: responseHeader,
		IncomingHeader: incomingHeader,
		GenerateID:     func() string { return "id" },
	}
}

// newPanicHandler returns an http.HandlerFunc that panics with msg.
func newPanicHandler(msg string) http.HandlerFunc {
	return http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic(msg)
	})
}

// assertSliceEqual checks that two string slices are element-wise equal.
func assertSliceEqual(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	for i, v := range want {
		if got[i] != v {
			t.Errorf("got[%d] = %q, want %q", i, got[i], v)
		}
	}
}
