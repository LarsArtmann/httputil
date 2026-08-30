package httputil

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestLogging_RecordsRequest(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := Logging(logger)(inner)

	req := newTestRequest(http.MethodGet, "/test", "")
	req.RemoteAddr = "10.0.0.1:1234"

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	out := buf.String()
	if out == "" {
		t.Fatal("expected log output, got none")
	}

	for _, want := range []string{"status=201", "method=GET", "path=/test"} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q: %s", want, out)
		}
	}
}

// TestLogging_DefaultStatus covers the status==0 branch: when the handler
// does not call WriteHeader, Logging defaults to 200.
func TestLogging_DefaultStatus(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	handler := Logging(logger)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	output := buf.String()
	if !strings.Contains(output, "status=200") {
		t.Errorf("log output does not contain status=200: %s", output)
	}
}

func BenchmarkLogging(b *testing.B) {
	logger := newTestLogger()
	middleware := Logging(logger)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// TestLogging_IncludesRequestID covers issue #2: when RequestID middleware
// ran upstream of Logging, the request log must carry the request_id so a
// log line can be correlated with the X-Request-ID response header.
func TestLogging_IncludesRequestID(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		RequestID(DefaultRequestIDConfig()),
		Logging(logger),
	)

	handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/test", ""))

	out := buf.String()
	if !strings.Contains(out, "request_id=") {
		t.Errorf("log output missing request_id: %s", out)
	}
}

// TestLogging_OmitsRequestIDWhenAbsent covers the inverse: without the
// RequestID middleware, no empty request_id field may be emitted.
func TestLogging_OmitsRequestIDWhenAbsent(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/test", ""))

	if out := buf.String(); strings.Contains(out, "request_id") {
		t.Errorf("log output should omit request_id when absent: %s", out)
	}
}
