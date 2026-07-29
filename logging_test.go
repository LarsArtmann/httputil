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

	if buf.Len() == 0 {
		t.Error("expected log output")
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
