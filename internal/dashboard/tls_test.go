package dashboard

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// freePort grabs an ephemeral port and releases it so the server can bind
// the same port immediately. Slightly racy, but adequate for tests.
func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return strconv.Itoa(port)
}

func TestGenerateSelfSignedCert(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	if err := GenerateSelfSignedCert(certPath, keyPath,
		[]string{"localhost", "127.0.0.1", "192.168.1.50"}); err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cert: %v", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("cert file does not contain a CERTIFICATE PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}

	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		t.Errorf("cert not currently valid: %v .. %v", cert.NotBefore, cert.NotAfter)
	}
	if len(cert.DNSNames) == 0 {
		t.Error("cert has no DNS SANs")
	}
	foundIP := false
	for _, ip := range cert.IPAddresses {
		if ip.String() == "192.168.1.50" {
			foundIP = true
		}
	}
	if !foundIP {
		t.Errorf("cert SANs missing 192.168.1.50: %v", cert.IPAddresses)
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read key: %v", err)
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		t.Fatalf("key file does not contain a PEM block")
	}
	if _, err := x509.ParseECPrivateKey(keyBlock.Bytes); err != nil {
		t.Fatalf("parse key: %v", err)
	}
}

// TestServerTLSIdleConnDoesNotBlockOtherClients is the regression test for
// one idle TCP connection blocking the dashboard: Accept must classify with
// a deadline, so a connection that never sends a byte cannot starve every
// subsequent client.
func TestServerTLSIdleConnDoesNotBlockOtherClients(t *testing.T) {
	srv := startTLSServer(t)

	// Connect and never send anything.
	idle, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("dial idle conn: %v", err)
	}
	defer idle.Close()

	client := testTLSClient()
	start := time.Now()
	resp, err := client.Get("https://" + srv.Addr() + "/")
	if err != nil {
		t.Fatalf("GET behind an idle connection: %v", err)
	}
	resp.Body.Close()

	// The request must complete shortly after the classification deadline
	// (3 s) — a generous watchdog catches the pre-fix hang.
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("idle connection blocked the dashboard for %v", elapsed)
	}
}

// TestServerTLSManyIdleConnsDoNotSerialize is the regression test for the
// serialized-classification bug: each idle connection used to burn its 3 s
// deadline on the sole Accept path, so ten of them delayed a legitimate
// browser by ~30 s. Classification now runs concurrently behind a bounded
// slot pool, so idle clients cost only their own slot.
func TestServerTLSManyIdleConnsDoNotSerialize(t *testing.T) {
	srv := startTLSServer(t)

	const idleConns = 10
	var idle []net.Conn
	for i := 0; i < idleConns; i++ {
		c, err := net.Dial("tcp", srv.Addr())
		if err != nil {
			t.Fatalf("dial idle conn %d: %v", i, err)
		}
		idle = append(idle, c)
	}
	defer func() {
		for _, c := range idle {
			c.Close()
		}
	}()

	client := testTLSClient()
	start := time.Now()
	resp, err := client.Get("https://" + srv.Addr() + "/")
	if err != nil {
		t.Fatalf("GET behind %d idle connections: %v", idleConns, err)
	}
	resp.Body.Close()

	// With concurrent classification the request completes almost
	// immediately; serially it would take ~30 s (10 × 3 s deadline).
	if elapsed := time.Since(start); elapsed > 8*time.Second {
		t.Fatalf("%d idle connections blocked the dashboard for %v", idleConns, elapsed)
	}
}

// TestServerTLSPeerDisconnectDoesNotKillServer is the regression test for
// a peer's EOF terminating Serve: a connect-then-hang-up must be treated as
// an ordinary per-connection failure, not a fatal listener error.
func TestServerTLSPeerDisconnectDoesNotKillServer(t *testing.T) {
	srv := startTLSServer(t)

	// Connect and immediately disconnect.
	c, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	c.Close()

	// Give the server a moment to process the dead connection.
	time.Sleep(200 * time.Millisecond)

	// The dashboard must still serve HTTPS.
	resp, err := testTLSClient().Get("https://" + srv.Addr() + "/")
	if err != nil {
		t.Fatalf("GET after peer disconnect: %v", err)
	}
	resp.Body.Close()
}

// TestHybridListenerAcceptSurvivesPeerEOF checks the listener contract
// directly: a vanished peer must never surface as an Accept error.
func TestHybridListenerAcceptSurvivesPeerEOF(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	hl := newHybridListener(ln, &tls.Config{})

	// Connect and immediately hang up.
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	c.Close()

	got, err := hl.Accept()
	if err != nil {
		t.Fatalf("Accept returned a fatal error for a vanished peer: %v", err)
	}
	got.Close()
}

// TestHybridListener_CloseClosesIdleClients reproduces the shutdown leak:
// clients connected but not yet sending their first byte used to stay open
// after Close because the inherited Listener.Close never managed in-flight
// classification. Close must return promptly and every client must observe
// the connection closing.
func TestHybridListener_CloseClosesIdleClients(t *testing.T) {
	// Shorten classification so the shutdown join is fast and deterministic.
	orig := hybridPeekTimeout
	hybridPeekTimeout = 100 * time.Millisecond
	t.Cleanup(func() { hybridPeekTimeout = orig })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	hl := newHybridListener(ln, &tls.Config{})

	// Clients connect and never send a byte (no Accept reader drains them).
	const idleConns = 8
	var conns []net.Conn
	for i := 0; i < idleConns; i++ {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatalf("dial idle conn %d: %v", i, err)
		}
		conns = append(conns, c)
	}
	// Let every connection reach classification.
	time.Sleep(50 * time.Millisecond)

	closed := make(chan struct{})
	go func() { hl.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close hung with idle clients connected")
	}

	// Every client must observe the shutdown as a close, not a hang.
	for i, c := range conns {
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, err := c.Read(make([]byte, 1)); err == nil {
			t.Errorf("client %d: read succeeded — connection left open after Close", i)
		}
		c.Close()
	}
	// The listener reports itself closed; a second Close is a no-op.
	if _, err := hl.Accept(); err == nil {
		t.Error("Accept after Close should return an error")
	}
	if err := hl.Close(); err != nil {
		t.Errorf("second Close = %v, want no error", err)
	}
}

// TestHybridListener_CloseWithSaturatedResultsQueue verifies shutdown when
// the result channel is full and workers are blocked delivering — the accept
// loop is also saturated on the slot pool. Close must unblock everyone and
// close every connection, not hang forever.
func TestHybridListener_CloseWithSaturatedResultsQueue(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	hl := newHybridListener(ln, &tls.Config{})

	// More clients than result-buffer + slot capacity; each sends a byte
	// immediately so classification completes instantly and the result
	// queue saturates while nobody reads Accept.
	const n = 80
	var conns []net.Conn
	for i := 0; i < n; i++ {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatalf("dial conn %d: %v", i, err)
		}
		if _, err := c.Write([]byte("G")); err != nil {
			t.Fatalf("write conn %d: %v", i, err)
		}
		conns = append(conns, c)
	}
	time.Sleep(100 * time.Millisecond) // let workers saturate the queue

	closed := make(chan struct{})
	go func() { hl.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close hung with a saturated results queue")
	}

	for i, c := range conns {
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, err := c.Read(make([]byte, 1)); err == nil {
			t.Errorf("client %d: read succeeded — connection left open after Close", i)
		}
		c.Close()
	}
	if _, err := hl.Accept(); err == nil {
		t.Error("Accept after Close should return an error")
	}
}

// startTLSServer boots a TLS dashboard on an ephemeral port and fails the
// test if it does not come online.
func startTLSServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := GenerateSelfSignedCert(certPath, keyPath, []string{"localhost", "127.0.0.1"}); err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}

	srv := NewWithTLS("127.0.0.1", freePort(t), certPath, keyPath)
	srv.Start()
	t.Cleanup(srv.Stop)

	select {
	case online := <-srv.Status():
		if !online {
			t.Fatalf("server failed to start: %v", srv.Error())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server to start")
	}
	return srv
}

func testTLSClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func TestServerServesTLS(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := GenerateSelfSignedCert(certPath, keyPath, []string{"localhost", "127.0.0.1"}); err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}

	srv := NewWithTLS("127.0.0.1", freePort(t), certPath, keyPath)
	srv.Start()
	defer srv.Stop()

	select {
	case online := <-srv.Status():
		if !online {
			t.Fatalf("server failed to start: %v", srv.Error())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server to start")
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Get("https://" + srv.Addr() + "/")
	if err != nil {
		t.Fatalf("GET https://%s: %v", srv.Addr(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// TestServerRedirectsHTTPToHTTPS: a browser that still uses http:// on the
// TLS port must receive a 301 redirect to the https:// URL instead of the
// raw "Client sent an HTTP request to an HTTPS server" handshake error.
func TestServerRedirectsHTTPToHTTPS(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := GenerateSelfSignedCert(certPath, keyPath, []string{"localhost", "127.0.0.1"}); err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}

	srv := NewWithTLS("127.0.0.1", freePort(t), certPath, keyPath)
	srv.Start()
	defer srv.Stop()

	select {
	case online := <-srv.Status():
		if !online {
			t.Fatalf("server failed to start: %v", srv.Error())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server to start")
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get("http://" + srv.Addr() + "/")
	if err != nil {
		t.Fatalf("GET http://%s: %v", srv.Addr(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("status = %d, want 301", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "https://"+srv.Addr()+"/" {
		t.Errorf("Location = %q, want %q", loc, "https://"+srv.Addr()+"/")
	}
}
