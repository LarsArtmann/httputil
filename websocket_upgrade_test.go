package httputil

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// errMalformedHeader signals a header line without a colon during upgrade
// response parsing in the WebSocket integration test.
var errMalformedHeader = errors.New("malformed HTTP header line in upgrade response")

// RFC 6455 (section 4.2.2) worked-example handshake values, reused verbatim
// so the test asserts a genuine WebSocket Upgrade rather than an echo of
// arbitrary strings.
const (
	wsExampleClientKey   = "dGhlIHNhbXBsZSBub25jZQ=="
	wsExampleAcceptValue = "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
)

// TestCompressionETag_WebSocketUpgrade_Passthrough drives a real HTTP
// connection through the Compression + ETag chain and performs a
// WebSocket-style Upgrade. It proves the buffering middleware does not
// corrupt the 101 Switching Protocols handshake or the raw byte stream that
// follows the hijack.
func TestCompressionETag_WebSocketUpgrade_Passthrough(t *testing.T) {
	t.Parallel()

	const echoPayload = "hello-over-hijacked-conn"

	// upgradeHandler mirrors the canonical WebSocket server pattern: set the
	// response headers, hijack the connection, then write the 101 line and
	// headers directly through the raw bufio.ReadWriter before exchanging
	// frames.
	upgradeHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Hijacker after Compression + ETag")
		}

		conn, bufrw, err := hijacker.Hijack()
		if err != nil {
			t.Fatalf("Hijack() through Compression + ETag failed: %v", err)
		}

		defer func() {
			_ = conn.Close()
		}()

		response := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + wsExampleAcceptValue + "\r\n" +
			"\r\n"

		_, err = bufrw.WriteString(response)
		if err != nil {
			t.Fatalf("write 101 response failed: %v", err)
		}

		err = bufrw.Flush()
		if err != nil {
			t.Fatalf("flush 101 response failed: %v", err)
		}

		// Raw frame echo: read one line from the client and return it. This
		// proves bytes flow bidirectionally after the hijack.
		line, err := bufrw.ReadString('\n')
		if err != nil {
			t.Fatalf("read client frame failed: %v", err)
		}

		_, err = bufrw.WriteString(strings.TrimSpace(line) + "\n")
		if err != nil {
			t.Fatalf("write echo frame failed: %v", err)
		}

		err = bufrw.Flush()
		if err != nil {
			t.Fatalf("flush echo frame failed: %v", err)
		}
	})

	handler := Chain(
		upgradeHandler,
		Compression(DefaultCompressionConfig()),
		ETag(DefaultETagConfig()),
	)

	server := httptest.NewServer(handler)
	defer server.Close()

	conn, err := net.Dial("tcp", server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial server failed: %v", err)
	}

	defer func() {
		_ = conn.Close()
	}()

	// Send a WebSocket-style Upgrade request. Accept-Encoding: gzip is
	// included deliberately to confirm compression negotiation never injects a
	// Content-Encoding header into the Upgrade response.
	request := "GET /ws HTTP/1.1\r\n" +
		"Host: " + server.Listener.Addr().String() + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + wsExampleClientKey + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Accept-Encoding: gzip\r\n" +
		"\r\n"

	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("write upgrade request failed: %v", err)
	}

	reader := bufio.NewReader(conn)

	// The very first bytes must be the 101 status line. Any buffered body or
	// header written by the middleware would corrupt it.
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read status line failed: %v", err)
	}

	if !strings.HasPrefix(statusLine, "HTTP/1.1 101") {
		t.Fatalf("status line = %q, want 101 Switching Protocols", statusLine)
	}

	headers, err := readUpgradeHeaders(reader)
	if err != nil {
		t.Fatalf("read upgrade headers failed: %v", err)
	}

	if v := headers[headerContentEncoding]; v != "" {
		t.Errorf(
			"Content-Encoding = %q on upgrade response, want none (compression must not engage)",
			v,
		)
	}

	if v := headers[headerETag]; v != "" {
		t.Errorf(
			"ETag = %q on upgrade response, want none (ETag must not stamp a hijacked stream)",
			v,
		)
	}

	if v := headers[http.CanonicalHeaderKey("Sec-WebSocket-Accept")]; v != wsExampleAcceptValue {
		t.Errorf("Sec-WebSocket-Accept = %q, want %q", v, wsExampleAcceptValue)
	}

	// Verify the raw post-hijack byte stream is intact.
	_, err = conn.Write([]byte(echoPayload + "\n"))
	if err != nil {
		t.Fatalf("write echo payload failed: %v", err)
	}

	echo, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read echo response failed: %v", err)
	}

	if strings.TrimSpace(echo) != echoPayload {
		t.Errorf("post-hijack echo = %q, want %q", echo, echoPayload)
	}
}

// readUpgradeHeaders reads HTTP headers from r until the terminating blank
// line and returns them keyed by canonical header name.
func readUpgradeHeaders(r *bufio.Reader) (map[string]string, error) {
	headers := make(map[string]string)

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read upgrade header line: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			return headers, nil
		}

		name, value, found := strings.Cut(line, ":")
		if !found {
			return nil, errMalformedHeader
		}

		headers[http.CanonicalHeaderKey(name)] = strings.TrimSpace(value)
	}
}
