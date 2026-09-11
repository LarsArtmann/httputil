package httputil

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultServerConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()

	if cfg.Addr != defaultAddr {
		t.Errorf("Addr = %q, want %q", cfg.Addr, defaultAddr)
	}

	if cfg.ReadTimeout != defaultReadTimeoutSeconds*time.Second {
		t.Errorf(
			"ReadTimeout = %v, want %v",
			cfg.ReadTimeout,
			defaultReadTimeoutSeconds*time.Second,
		)
	}

	if cfg.ReadHeaderTimeout != defaultReadHeaderTimeoutSeconds*time.Second {
		t.Errorf(
			"ReadHeaderTimeout = %v, want %v",
			cfg.ReadHeaderTimeout,
			defaultReadHeaderTimeoutSeconds*time.Second,
		)
	}

	if cfg.WriteTimeout != defaultWriteTimeoutSeconds*time.Second {
		t.Errorf(
			"WriteTimeout = %v, want %v",
			cfg.WriteTimeout,
			defaultWriteTimeoutSeconds*time.Second,
		)
	}

	if cfg.IdleTimeout != defaultIdleTimeoutSeconds*time.Second {
		t.Errorf(
			"IdleTimeout = %v, want %v",
			cfg.IdleTimeout,
			defaultIdleTimeoutSeconds*time.Second,
		)
	}

	if cfg.ShutdownTimeout != defaultShutdownTimeoutSeconds*time.Second {
		t.Errorf(
			"ShutdownTimeout = %v, want %v",
			cfg.ShutdownTimeout,
			defaultShutdownTimeoutSeconds*time.Second,
		)
	}
}

func TestServerConfigValidateDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestServerConfigValidateNegativeReadTimeout(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:        defaultAddr,
		ReadTimeout: -1 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative ReadTimeout")
	}

	if !errors.Is(err, errReadTimeoutNegative) {
		t.Errorf("Validate() error = %v, want errReadTimeoutNegative", err)
	}
}

func TestServerConfigValidateNegativeReadHeaderTimeout(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:              defaultAddr,
		ReadHeaderTimeout: -1 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative ReadHeaderTimeout")
	}

	if !errors.Is(err, errReadHeaderTimeoutNegative) {
		t.Errorf("Validate() error = %v, want errReadHeaderTimeoutNegative", err)
	}
}

func TestServerConfigValidateNegativeWriteTimeout(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:         defaultAddr,
		WriteTimeout: -1 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative WriteTimeout")
	}

	if !errors.Is(err, errWriteTimeoutNegative) {
		t.Errorf("Validate() error = %v, want errWriteTimeoutNegative", err)
	}
}

func TestServerConfigValidateNegativeIdleTimeout(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:        defaultAddr,
		IdleTimeout: -1 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative IdleTimeout")
	}

	if !errors.Is(err, errIdleTimeoutNegative) {
		t.Errorf("Validate() error = %v, want errIdleTimeoutNegative", err)
	}
}

func TestServerConfigValidateNegativeShutdownTimeout(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:            defaultAddr,
		ShutdownTimeout: -1 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative ShutdownTimeout")
	}

	if !errors.Is(err, errShutdownTimeoutNegative) {
		t.Errorf("Validate() error = %v, want errShutdownTimeoutNegative", err)
	}
}

func TestServerConfigValidateEmptyAddr(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr: "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for empty Addr")
	}

	if !errors.Is(err, errServerAddrEmpty) {
		t.Errorf("Validate() error = %v, want errServerAddrEmpty", err)
	}
}

func TestServerConfigValidateReadHeaderExceedsRead(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:              defaultAddr,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for ReadHeaderTimeout > ReadTimeout")
	}

	if !errors.Is(err, errServerTimeoutOrdering) {
		t.Errorf("Validate() error = %v, want errServerTimeoutOrdering", err)
	}
}

func TestServerConfigValidateAllowsEqualReadAndHeaderTimeouts(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:              defaultAddr,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil (equal timeouts are valid)", err)
	}
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.Addr = "127.0.0.1:0"

	srv, err := NewServer(cfg, newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	if srv.Addr() != cfg.Addr {
		t.Errorf("Addr() = %q, want %q", srv.Addr(), cfg.Addr)
	}
}

func TestNewServerInvalidConfig(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:        defaultAddr,
		ReadTimeout: -1 * time.Second,
	}

	_, err := NewServer(cfg, http.NotFoundHandler())
	if err == nil {
		t.Fatal("NewServer() error = nil, want error for negative ReadTimeout")
	}
}

func TestServerStartAndShutdown(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.Addr = "127.0.0.1:0"

	srv, err := NewServer(cfg, newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	errChan := srv.Start()

	if _, ok := waitForListenerAddr(t, srv, errChan); !ok {
		t.Fatal("listener address did not resolve after Start")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestServerShutdownWithBackgroundContext(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.Addr = "127.0.0.1:0"
	cfg.ShutdownTimeout = 5 * time.Second

	srv, err := NewServer(cfg, newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	errChan := srv.Start()

	if _, ok := waitForListenerAddr(t, srv, errChan); !ok {
		t.Fatal("listener address did not resolve after Start")
	}

	err = srv.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown() with background ctx error = %v", err)
	}
}

func TestServerStartError(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.Addr = "invalid-address"

	srv, err := NewServer(cfg, http.NotFoundHandler())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	errChan := srv.Start()

	select {
	case err := <-errChan:
		if err == nil {
			t.Fatal("expected startup error, got nil")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected startup error, got none")
	}
}

func TestServerServesRequests(t *testing.T) {
	t.Parallel()

	// Bind a concrete port so the test can issue a real HTTP request through
	// the running server (a :0 address never exposes the resolved port).
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}

	addr := listener.Addr().String()

	if err := listener.Close(); err != nil {
		t.Fatalf("listener.Close: %v", err)
	}

	cfg := DefaultServerConfig()
	cfg.Addr = addr

	srv, err := NewServer(cfg, newWriteStatusHandler("hello"))
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	errChan := srv.Start()

	if _, ok := waitForListenerAddr(t, srv, errChan); !ok {
		t.Fatal("listener address did not resolve after Start")
	}

	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("GET http://%s/: %v", addr, err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if string(body) != "hello" {
		t.Errorf("body = %q, want %q", string(body), "hello")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestServerShutdownReturnsErrorOnContextExpiry(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}

	blockCh := make(chan struct{})
	handlerReached := make(chan struct{})

	srv := &Server{
		httpServer: &http.Server{
			Handler: http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				close(handlerReached)
				<-blockCh
			}),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}

	go func() {
		_ = srv.httpServer.Serve(listener)
	}()

	t.Cleanup(func() {
		close(blockCh)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = srv.Shutdown(ctx)
	})

	addr := listener.Addr().String()

	go func() {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get("http://" + addr + "/")
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	<-handlerReached

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = srv.Shutdown(ctx)
	if err == nil {
		t.Fatal("expected error when context expires with active connections")
	}
}

func TestServerConfigValidateTLSInsecureMinVersion(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.TLSConfig = &tls.Config{ //nolint:gosec // G402: intentionally insecure MinVersion for validation test
		MinVersion: tls.VersionTLS10,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for TLS 1.0 MinVersion")
	}

	if !errors.Is(err, errTLSMinVersionInsecure) {
		t.Errorf("Validate() error = %v, want errTLSMinVersionInsecure", err)
	}
}

func TestServerConfigValidateTLSInsecureMinVersion11(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.TLSConfig = &tls.Config{ //nolint:gosec // G402: intentionally insecure MinVersion for validation test
		MinVersion: tls.VersionTLS11,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for TLS 1.1 MinVersion")
	}

	if !errors.Is(err, errTLSMinVersionInsecure) {
		t.Errorf("Validate() error = %v, want errTLSMinVersionInsecure", err)
	}
}

func TestServerConfigValidateTLSMinVersion12(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for TLS 1.2 MinVersion", err)
	}
}

func TestServerConfigValidateTLSMinVersion13(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for TLS 1.3 MinVersion", err)
	}
}

func TestServerConfigValidateTLSZeroMinVersion(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.TLSConfig = &tls.Config{}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for zero MinVersion (defaults to TLS 1.2)", err)
	}
}

func TestNewServerWiresTLSConfig(t *testing.T) {
	t.Parallel()

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	cfg := DefaultServerConfig()
	cfg.TLSConfig = tlsCfg

	srv, err := NewServer(cfg, newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	if srv.httpServer.TLSConfig != tlsCfg {
		t.Error("NewServer() did not wire TLSConfig to the underlying http.Server")
	}
}

func TestServerStartTLSServesHTTPSWithSelfSignedCert(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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

	srv, err := NewServer(ServerConfig{
		Addr:              "127.0.0.1:0",
		ReadTimeout:       time.Second,
		ReadHeaderTimeout: time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
		ShutdownTimeout:   time.Second,
		TLSConfig:         tlsCfg.Clone(), // the server mutates its config (h2 ALPN); give it a private copy
	}, handler)
	if err != nil {
		t.Fatal(err)
	}

	errChan := srv.StartTLS(certPath, keyPath)
	listenAddr, ok := waitForListenerAddr(t, srv, errChan)
	if !ok {
		t.Fatal("listener address did not resolve after StartTLS")
	}

	waitForTLS(t, errChan, listenAddr, tlsCfg)

	clientTLS := tlsCfg.Clone()
	clientTLS.NextProtos = []string{"http/1.1"}

	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: clientTLS, ForceAttemptHTTP2: false},
	}
	resp, err := client.Get("https://" + listenAddr + "/")
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if resp.TLS == nil || resp.TLS.Version < tls.VersionTLS12 {
		t.Errorf("connection should use TLS 1.2+, got %v", resp.TLS)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("shutdown failed: %v", err)
	}

	select {
	case err := <-errChan:
		t.Errorf("unexpected server error: %v", err)
	default:
	}
}

// waitForListenerAddr polls Server.ListenerAddr until the started listener
// resolves or the deadline elapses, failing the test on a startup error from
// errChan.
func waitForListenerAddr(t *testing.T, srv *Server, errChan <-chan error) (string, bool) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) {
		select {
		case err := <-errChan:
			t.Fatalf("server failed to start: %v", err)
		case <-time.After(10 * time.Millisecond):
		}

		addr, ok := srv.ListenerAddr()
		if ok {
			return addr.String(), true
		}
	}

	return "", false
}

func newSelfSignedCert(t *testing.T) ([]byte, []byte) {
	t.Helper()

	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, key.Public(), key)
	if err != nil {
		t.Fatal(err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM
}

func TestServer_ListenerAddr_NotListening_ReturnsFalse(t *testing.T) {
	t.Parallel()

	srv, err := NewServer(DefaultServerConfig(), http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}

	if addr, ok := srv.ListenerAddr(); ok {
		t.Errorf("ListenerAddr before Start = (%v, true), want (nil, false)", addr)
	}
}

func TestServer_Start_EphemeralAddr_ListenerAddrResolvesPort(t *testing.T) {
	t.Parallel()

	srv, err := NewServer(ServerConfig{
		Addr:              "127.0.0.1:0",
		ReadTimeout:       time.Second,
		ReadHeaderTimeout: time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
		ShutdownTimeout:   time.Second,
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatal(err)
	}

	errChan := srv.Start()

	listenAddr, ok := waitForListenerAddr(t, srv, errChan)
	if !ok {
		t.Fatal("listener address did not resolve after Start")
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", listenAddr)
	if err != nil {
		t.Fatal(err)
	}

	if tcpAddr.Port == 0 {
		t.Errorf("resolved port = 0, want the OS-assigned ephemeral port")
	}

	resp, err := http.Get("http://" + listenAddr + "/")
	if err != nil {
		t.Fatalf("request to resolved address failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	if addr, ok := srv.ListenerAddr(); ok {
		t.Errorf("ListenerAddr after Shutdown = (%v, true), want (nil, false)", addr)
	}
}

func TestServer_Start_TwiceFailsSecondStart(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.Addr = "127.0.0.1:0"

	srv, err := NewServer(cfg, newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	errChan := srv.Start()

	if _, ok := waitForListenerAddr(t, srv, errChan); !ok {
		t.Fatal("listener address did not resolve after Start")
	}

	secondErrChan := srv.Start()

	select {
	case err := <-secondErrChan:
		if !errors.Is(err, errServerAlreadyStarted) {
			t.Errorf("second Start() error = %v, want errServerAlreadyStarted", err)
		}
	case <-time.After(time.Second):
		t.Fatal("second Start() delivered no error")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestServer_StartTLS_ListenerClearedOnCertFailure(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.Addr = "127.0.0.1:0"

	srv, err := NewServer(cfg, newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	errChan := srv.StartTLS("/nonexistent/cert.pem", "/nonexistent/key.pem")

	select {
	case err := <-errChan:
		if err == nil {
			t.Fatal("StartTLS with unreadable certificate files delivered nil error")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("StartTLS with unreadable certificate files delivered no error")
	}

	if addr, ok := srv.ListenerAddr(); ok {
		t.Errorf("ListenerAddr after failed StartTLS = (%v, true), want (nil, false)", addr)
	}
}
