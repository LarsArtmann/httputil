package httputil

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
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

	waitForServerStart(t, errChan, 100*time.Millisecond)

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

	waitForServerStart(t, errChan, 100*time.Millisecond)

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

	waitForServerStart(t, errChan, 100*time.Millisecond)

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

	freePort := reserveFreePort(t)
	certPEM, keyPEM := newSelfSignedCert(t)
	certPath, keyPath := filepath.Join(
		t.TempDir(),
		"cert.pem",
	), filepath.Join(
		t.TempDir(),
		"key.pem",
	)
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
		Addr:              fmt.Sprintf("127.0.0.1:%d", freePort),
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

	select {
	case e := <-errChan:
		t.Fatalf("StartTLS error: %v", e)
	default:
	}
	waitForTLS(t, srv, tlsCfg)

	clientTLS := tlsCfg.Clone()
	clientTLS.NextProtos = []string{"http/1.1"}

	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: clientTLS, ForceAttemptHTTP2: false},
	}
	resp, err := client.Get("https://" + srv.Addr() + "/")
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

func newSelfSignedCert(t *testing.T) ([]byte, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(
		&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)},
	)

	return certPEM, keyPEM
}

func waitForTLS(t *testing.T, srv *Server, cfg *tls.Config) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) {
		conn, err := tls.DialWithDialer(
			&net.Dialer{Timeout: 100 * time.Millisecond},
			"tcp",
			srv.Addr(),
			cfg,
		)
		if err == nil {
			_ = conn.Close()

			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("TLS server did not become ready within 3s")
}

func reserveFreePort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected listener address type: %T", l.Addr())
	}

	return addr.Port
}
