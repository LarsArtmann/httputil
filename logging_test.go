package httputil

import (
	"bytes"
	"context"
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

// TestLogging_IncludesRequestIDMatchingResponseHeader covers issue #2: when
// RequestID middleware ran upstream of Logging, the logged request_id must
// equal the X-Request-Id response header value, so a log line correlates
// with what the client received.
func TestLogging_IncludesRequestIDMatchingResponseHeader(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	rec := newRecorder()

	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		RequestID(DefaultRequestIDConfig()),
		Logging(logger),
	)

	handler.ServeHTTP(rec, newTestRequest(http.MethodGet, "/test", ""))

	headerID := rec.Header().Get("X-Request-Id")
	if headerID == "" {
		t.Fatal("response missing X-Request-Id header")
	}

	out := buf.String()
	if !strings.Contains(out, "request_id="+headerID) {
		t.Errorf("log output missing request_id=%s: %s", headerID, out)
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

// cancelDroppingHandler is an slog.Handler that refuses records whose log
// context is already canceled, standing in for cancellation-aware handlers.
type cancelDroppingHandler struct {
	wrote bool
}

func (h *cancelDroppingHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *cancelDroppingHandler) Handle(ctx context.Context, _ slog.Record) error {
	if ctx.Err() == nil {
		h.wrote = true
	}

	return nil
}

func (h *cancelDroppingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *cancelDroppingHandler) WithGroup(string) slog.Handler { return h }

// TestLogging_EmitsWhenRequestContextCanceled pins the WithoutCancel
// behavior: when the request context dies before logging (Timeout deadline,
// client disconnect), the request log must still be emitted.
func TestLogging_EmitsWhenRequestContextCanceled(t *testing.T) {
	t.Parallel()

	dropping := &cancelDroppingHandler{}
	logger := slog.New(dropping)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := newTestRequest(http.MethodGet, "/test", "").WithContext(ctx)

	Logging(logger)(newNoOpHandler()).ServeHTTP(newRecorder(), req)

	if !dropping.wrote {
		t.Error("request log was dropped because the request context was canceled")
	}
}
