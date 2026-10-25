package httputil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewServerListener_NilListener_ReturnsListenerNilRejection(t *testing.T) {
	t.Parallel()

	_, err := NewServerListener(nil, DefaultServerConfig(), newNoOpHandler())
	if err == nil {
		t.Fatal("NewServerListener(nil, ...) error = nil, want listener_nil rejection")
	}

	if !errors.Is(err, errServerListenerNil) {
		t.Errorf("NewServerListener(nil, ...) error = %v, want errors.Is(err, errServerListenerNil)", err)
	}
}

func TestNewServerListener_InvalidConfig_ReturnsValidationErrorAndLeavesListenerToCaller(t *testing.T) {
	t.Parallel()

	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	cfg := DefaultServerConfig()
	cfg.ReadTimeout = -time.Second

	_, err = NewServerListener(ln, cfg, newNoOpHandler())
	if err == nil {
		t.Fatal("NewServerListener() error = nil, want error for negative ReadTimeout")
	}

	if !errors.Is(err, errReadTimeoutNegative) {
		t.Errorf("NewServerListener() error = %v, want errors.Is(err, errReadTimeoutNegative)", err)
	}

	// The listener is untouched on failure, so closing it stays the caller's
	// prerogative and must succeed.
	if closeErr := ln.Close(); closeErr != nil {
		t.Fatalf("ln.Close() after failed construction error = %v, want nil (listener must be untouched)", closeErr)
	}
}

func TestNewServerListener_DerivesAddrFromListener(t *testing.T) {
	t.Parallel()

	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	defer func() { _ = ln.Close() }()

	// Zero Addr is acceptable here: the bound listener is the authority, so
	// Addr derivation must fill the field before validation runs.
	cfg := ServerConfig{
		ShutdownTimeout: time.Second,
	}

	srv, err := NewServerListener(ln, cfg, newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServerListener() error = %v", err)
	}

	if got, want := srv.Addr(), ln.Addr().String(); got != want {
		t.Errorf("Addr() = %q, want listener-derived %q", got, want)
	}
}

func TestServerListener_ListenerAddr_ReportsInjectedListenerBeforeStart(t *testing.T) {
	t.Parallel()

	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	defer func() { _ = ln.Close() }()

	srv, err := NewServerListener(ln, DefaultServerConfig(), newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServerListener() error = %v", err)
	}

	addr, ok := srv.ListenerAddr()
	if !ok {
		t.Fatal("ListenerAddr() ok = false before Start, want true for injected listener")
	}

	if addr.String() != ln.Addr().String() {
		t.Errorf("ListenerAddr() = %q, want injected listener address %q", addr.String(), ln.Addr().String())
	}
}

func TestServerListener_Start_ServesInjectedListenerWithoutRebinding(t *testing.T) {
	t.Parallel()

	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv, err := NewServerListener(ln, DefaultServerConfig(), newWriteStatusHandler("injected"))
	if err != nil {
		t.Fatalf("NewServerListener() error = %v", err)
	}

	errChan := srv.Start()

	// Addr is derived from the listener the socket already holds; a rebind
	// would fail with address-in-use, so a served request proves Start served
	// the injected listener instead of binding again.
	listenAddr, ok := waitForListenerAddr(t, srv, errChan)
	if !ok {
		t.Fatal("listener address did not resolve after Start")
	}

	if listenAddr != ln.Addr().String() {
		t.Errorf("ListenerAddr() = %q, want injected listener address %q", listenAddr, ln.Addr().String())
	}

	resp, err := http.DefaultClient.Do(newTestGetRequest(t, "http://"+listenAddr+"/"))
	if err != nil {
		t.Fatalf("GET http://%s/: %v", listenAddr, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestServerListener_Start_ServesUnixSocketListener(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "httputil.sock")

	ln, err := new(net.ListenConfig).Listen(context.Background(), "unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}

	srv, err := NewServerListener(ln, DefaultServerConfig(), newWriteStatusHandler("unix"))
	if err != nil {
		t.Fatalf("NewServerListener() error = %v", err)
	}

	errChan := srv.Start()

	select {
	case err := <-errChan:
		t.Fatalf("server failed to start: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not start within 3s")
	default:
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var dialer net.Dialer

				return dialer.DialContext(ctx, "unix", socketPath)
			},
		},
	}

	resp, err := client.Do(newTestGetRequest(t, "http://unix/"))
	if err != nil {
		t.Fatalf("GET over unix socket: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	select {
	case err := <-errChan:
		t.Errorf("unexpected server error: %v", err)
	default:
	}
}

func TestServerListener_StartTLS_ServesHTTPSOnInjectedListener(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	certPEM, keyPEM := newSelfSignedCert(t)
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	srv, err := NewServerListener(ln, DefaultServerConfig(), handler)
	if err != nil {
		t.Fatalf("NewServerListener() error = %v", err)
	}

	errChan := srv.StartTLS(certPath, keyPath)
	listenAddr, ok := waitForListenerAddr(t, srv, errChan)
	if !ok {
		t.Fatal("listener address did not resolve after StartTLS")
	}

	if listenAddr != ln.Addr().String() {
		t.Errorf("ListenerAddr() = %q, want injected listener address %q", listenAddr, ln.Addr().String())
	}

	// A plain HTTP request against the TLS listener must fail; a successful
	// TLS request then proves ServeTLS wrapped the injected listener.
	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(cert)

	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		RootCAs:            caPool,
		ClientAuth:         tls.NoClientCert,
		InsecureSkipVerify: false,
		ServerName:         "localhost",
	}

	plainClient := &http.Client{Timeout: 2 * time.Second}
	if _, err := plainClient.Do(newTestGetRequest(t, "http://"+listenAddr+"/")); err == nil {
		t.Error("plain HTTP request against TLS listener succeeded, want failure")
	}

	clientTLS := tlsCfg.Clone()
	clientTLS.NextProtos = []string{"http/1.1"}

	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: clientTLS, ForceAttemptHTTP2: false},
	}
	resp, err := client.Do(newTestGetRequest(t, "https://"+listenAddr+"/"))
	if err != nil {
		t.Fatalf("HTTPS request over injected listener failed: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if resp.TLS == nil || resp.TLS.Version < tls.VersionTLS12 {
		t.Errorf("connection should use TLS 1.2+, got %v", resp.TLS)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestServerListener_Start_TwiceFailsSecondStart(t *testing.T) {
	t.Parallel()

	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv, err := NewServerListener(ln, DefaultServerConfig(), newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServerListener() error = %v", err)
	}

	firstChan := srv.Start()

	if _, ok := waitForListenerAddr(t, srv, firstChan); !ok {
		t.Fatal("listener address did not resolve after Start")
	}

	secondChan := srv.Start()

	select {
	case err := <-secondChan:
		if !errors.Is(err, errServerAlreadyStarted) {
			t.Errorf("second Start() error = %v, want errors.Is(err, errServerAlreadyStarted)", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("second Start() delivered no error, want already_started")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func waitForServerSettled(t *testing.T, srv *Server) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) {
		if !srv.started.Load() {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatal("server did not settle within 3s")
}

func TestServerListener_RestartAfterShutdown_ReturnsListenerConsumed(t *testing.T) {
	t.Parallel()

	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv, err := NewServerListener(ln, DefaultServerConfig(), newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServerListener() error = %v", err)
	}

	errChan := srv.Start()

	if _, ok := waitForListenerAddr(t, srv, errChan); !ok {
		t.Fatal("listener address did not resolve after Start")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	waitForServerSettled(t, srv)

	if _, ok := srv.ListenerAddr(); ok {
		t.Error("ListenerAddr() ok = true after Shutdown, want false")
	}

	retryChan := srv.Start()

	select {
	case err := <-retryChan:
		if !errors.Is(err, errServerListenerConsumed) {
			t.Errorf("retried Start() error = %v, want errors.Is(err, errServerListenerConsumed)", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("retried Start() delivered no error, want listener_consumed")
	}
}

func TestServerListener_RestartTLSAfterShutdown_ReturnsListenerConsumed(t *testing.T) {
	t.Parallel()

	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	certPEM, keyPEM := newSelfSignedCert(t)
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	srv, err := NewServerListener(ln, DefaultServerConfig(), newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServerListener() error = %v", err)
	}

	errChan := srv.StartTLS(certPath, keyPath)

	if _, ok := waitForListenerAddr(t, srv, errChan); !ok {
		t.Fatal("listener address did not resolve after StartTLS")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	waitForServerSettled(t, srv)

	retryChan := srv.StartTLS(certPath, keyPath)

	select {
	case err := <-retryChan:
		if !errors.Is(err, errServerListenerConsumed) {
			t.Errorf("retried StartTLS() error = %v, want errors.Is(err, errServerListenerConsumed)", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("retried StartTLS() delivered no error, want listener_consumed")
	}
}

func TestServerListener_ShutdownClosesInjectedListener(t *testing.T) {
	t.Parallel()

	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv, err := NewServerListener(ln, DefaultServerConfig(), newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServerListener() error = %v", err)
	}

	errChan := srv.Start()

	if _, ok := waitForListenerAddr(t, srv, errChan); !ok {
		t.Fatal("listener address did not resolve after Start")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	waitForServerSettled(t, srv)

	// Ownership transferred to the Server: Serve closes the listener when
	// serving ends, so a caller-side Accept must report the closed socket.
	if _, acceptErr := ln.Accept(); acceptErr == nil {
		t.Error("ln.Accept() after Shutdown error = nil, want closed-listener error")
	}
}
