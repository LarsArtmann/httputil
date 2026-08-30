package httputil

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	etag "github.com/larsartmann/go-etag/server"
)

const (
	// hijackTestTimeout bounds every blocking I/O operation in the real-
	// connection upgrade tests so a regression fails instead of hanging.
	hijackTestTimeout = 5 * time.Second

	// hijackEchoPayload is the client payload echoed back over the hijacked
	// connection in the byte-integrity test.
	hijackEchoPayload = "raw-tcp-bytes-after-hijack!"
)

// upgradeHandshake is the minimal 101 response a handler writes directly to
// the connection after hijacking. These raw bytes are the entire response the
// client sees — the middleware chain must not inject headers into them.
const upgradeHandshake = "HTTP/1.1 101 Switching Protocols\r\n" +
	"Upgrade: websocket\r\n" +
	"Connection: Upgrade\r\n" +
	"\r\n"

// newHijackChain builds the Compression+ETag chain around inner. This is the
// composition whose Hijack passthrough the tests in this file pin.
func newHijackChain(inner http.Handler) http.Handler {
	return Chain(
		inner,
		Compression(DefaultCompressionConfig()),
		ETag(etag.DefaultETagConfig()),
	)
}

// TestChain_CompressionETag_PreservesHijacker verifies that the inner handler
// still sees an http.Hijacker through the Compression+ETag wrapper chain and
// that Hijack reaches the underlying writer (interface delegation only; the
// real-connection behavior is covered by the upgrade tests below).
func TestChain_CompressionETag_PreservesHijacker(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		hijacker, ok := rw.(http.Hijacker)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Hijacker")
		}

		if _, _, err := hijacker.Hijack(); err != nil {
			t.Fatalf("Hijack() error = %v, want nil", err)
		}
	})

	rec := newHijackRecorder()

	newHijackChain(inner).ServeHTTP(rec, newTestRequest(http.MethodGet, "/", ""))

	if !rec.hijacked {
		t.Error("underlying writer was not hijacked")
	}
}

// TestChain_HijackUpgrade_ResponseType is the light real-connection test: a
// handler that hijacks must produce a 101 response with its own headers and
// neither Content-Encoding (Compression) nor ETag (ETag) injected.
func TestChain_HijackUpgrade_ResponseType(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		hijacker, ok := rw.(http.Hijacker)
		if !ok {
			http.Error(rw, "hijacker unavailable", http.StatusInternalServerError)

			return
		}

		conn, bufRW, err := hijacker.Hijack()
		if err != nil {
			return
		}

		defer conn.Close() //nolint:errcheck // best-effort cleanup of the hijacked connection
		_ = conn.SetDeadline(time.Now().Add(hijackTestTimeout))

		_, _ = bufRW.WriteString(upgradeHandshake)
		_ = bufRW.Flush()
	})

	srv := httptest.NewServer(newHijackChain(inner))
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("upgrade request failed: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusSwitchingProtocols)
	}

	if got := resp.Header.Get("Upgrade"); got != "websocket" {
		t.Errorf("Upgrade header = %q, want %q", got, "websocket")
	}

	if got := resp.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want empty (upgrade responses must not be compressed)", got)
	}

	if got := resp.Header.Get("ETag"); got != "" {
		t.Errorf("ETag = %q, want empty (upgrade responses must not be tagged)", got)
	}
}

// TestChain_HijackUpgrade_BytesFlowThroughHijackedConn is the full real-
// connection test: after the 101 handshake, client bytes written to the
// connection reach the handler's hijacked reader and the handler's raw
// response bytes reach the client uncorrupted, proving the hijacked conn is
// the live TCP socket end to end.
func TestChain_HijackUpgrade_BytesFlowThroughHijackedConn(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		hijacker, ok := rw.(http.Hijacker)
		if !ok {
			http.Error(rw, "hijacker unavailable", http.StatusInternalServerError)

			return
		}

		conn, bufRW, err := hijacker.Hijack()
		if err != nil {
			return
		}

		defer conn.Close() //nolint:errcheck // best-effort cleanup of the hijacked connection
		_ = conn.SetDeadline(time.Now().Add(hijackTestTimeout))

		if _, err := bufRW.WriteString(upgradeHandshake); err != nil {
			return
		}

		if err := bufRW.Flush(); err != nil {
			return
		}

		echo := make([]byte, len(hijackEchoPayload)) //nolint:makezero // filled via io.ReadFull below

		if _, err := io.ReadFull(bufRW, echo); err != nil {
			return
		}

		if _, err := bufRW.Write(echo); err != nil {
			return
		}

		_ = bufRW.Flush()
	})

	srv := httptest.NewServer(newHijackChain(inner))
	t.Cleanup(srv.Close)

	bodyReader, bodyWriter := io.Pipe()
	// Unblock the pipe-writing goroutine if the request fails before the
	// transport consumes the body (otherwise it blocks forever on Write).
	t.Cleanup(func() { _ = bodyWriter.Close() })

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	req.ContentLength = int64(len(hijackEchoPayload))
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	// The pipe write runs concurrently with the request: Do returns as soon
	// as the handshake arrives, and the payload bytes reach the hijacked
	// connection as the transport streams the request body.
	go func() {
		_, _ = bodyWriter.Write([]byte(hijackEchoPayload))
		_ = bodyWriter.Close()
	}()

	resp, doErr := srv.Client().Do(req)
	if doErr != nil {
		t.Fatalf("upgrade request failed: %v", doErr)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusSwitchingProtocols)
	}

	echo := make([]byte, len(hijackEchoPayload)) //nolint:makezero // filled via io.ReadFull below

	readErr := make(chan error, 1)

	go func() {
		_, err := io.ReadFull(resp.Body, echo)
		readErr <- err
	}()

	select {
	case err := <-readErr:
		if err != nil {
			t.Fatalf("read echo: %v", err)
		}
	case <-time.After(hijackTestTimeout):
		t.Fatal("timed out waiting for echoed bytes")
	}

	if string(echo) != hijackEchoPayload {
		t.Errorf(
			"echoed bytes = %q, want %q (post-hijack payload must flow through uncorrupted)",
			string(echo),
			hijackEchoPayload,
		)
	}
}
