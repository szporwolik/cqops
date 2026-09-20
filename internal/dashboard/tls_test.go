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
