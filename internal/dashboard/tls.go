package dashboard

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// GenerateSelfSignedCert writes a self-signed TLS certificate and its
// private key to certPath and keyPath (PEM). The certificate is valid for
// three years and lists the given hosts as Subject Alternative Names —
// modern browsers reject certificates identified only by Common Name, so
// the SAN list must cover every address the dashboard is reached by.
func GenerateSelfSignedCert(certPath, keyPath string, hosts []string) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	if err != nil {
		return err
	}

	now := time.Now()
	tpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "CQOps Dashboard"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(3, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	if len(hosts) == 0 {
		hosts = []string{"localhost"}
	}
	for _, h := range hosts {
		if h == "" {
			continue
		}
		if ip := net.ParseIP(h); ip != nil {
			tpl.IPAddresses = append(tpl.IPAddresses, ip)
		} else {
			tpl.DNSNames = append(tpl.DNSNames, h)
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(certPath), 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return err
	}

	certOut, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := certOut.Write(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})); err != nil {
		certOut.Close()
		return err
	}
	if err := certOut.Close(); err != nil {
		return err
	}

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := keyOut.Write(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})); err != nil {
		keyOut.Close()
		return err
	}
	return keyOut.Close()
}

// redirectToHTTPS answers plain-HTTP requests with a permanent redirect to
// the equivalent https:// URL when the dashboard serves TLS. Without this a
// browser pointed at http://host:port gets the raw "Client sent an HTTP
// request to an HTTPS server" error instead of being sent to the right URL.
func redirectToHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil {
			next.ServeHTTP(w, r)
			return
		}
		host := r.Host
		if host == "" {
			host = r.URL.Host
		}
		http.Redirect(w, r, "https://"+host+r.URL.RequestURI(), http.StatusMovedPermanently)
	})
}

// hybridListener routes connections on a single port by their first byte:
// TLS handshakes (record type 0x16) are wrapped in *tls.Conn and served as
// HTTPS, anything else is plain HTTP and reaches the redirect handler.
type hybridListener struct {
	net.Listener
	tlsConfig *tls.Config
}

// hybridPeekTimeout bounds how long Accept waits for a connection's first
// byte while classifying it. Without a deadline a single idle TCP connection
// would block Accept and starve every subsequent client.
const hybridPeekTimeout = 3 * time.Second

func newHybridListener(ln net.Listener, tlsConfig *tls.Config) net.Listener {
	return &hybridListener{Listener: ln, tlsConfig: tlsConfig}
}

func (hl *hybridListener) Accept() (net.Conn, error) {
	c, err := hl.Listener.Accept()
	if err != nil {
		return nil, err
	}
	br := bufio.NewReader(c)

	// Classify by the first byte. An individual connection must never fail
	// the listener: an idle client hits the peek deadline, a vanished
	// client returns EOF — both become ordinary plain-HTTP connections and
	// are handled (and cleaned up) per-connection by the http server, whose
	// own read deadline applies from here. Returning those errors from
	// Accept would terminate http.Server.Serve for everyone.
	if err := c.SetReadDeadline(time.Now().Add(hybridPeekTimeout)); err == nil {
		if first, err := br.Peek(1); err == nil && first[0] == 0x16 { // TLS handshake record
			c.SetReadDeadline(time.Time{}) // clear the classification deadline
			return tls.Server(&prefixedConn{Conn: c, r: br}, hl.tlsConfig), nil
		}
		// Clear the classification deadline before handing the connection
		// to the http server.
		c.SetReadDeadline(time.Time{})
	}
	return &prefixedConn{Conn: c, r: br}, nil
}

// prefixedConn reinserts already-buffered bytes so the http server reads
// the stream exactly as the client sent it.
type prefixedConn struct {
	net.Conn
	r *bufio.Reader
}

func (pc *prefixedConn) Read(p []byte) (int, error) { return pc.r.Read(p) }
