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
	"sync"
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
//
// Classification runs CONCURRENTLY behind a bounded slot pool: an idle
// connection burns only its own deadline instead of serializing every later
// client on the sole Accept path.
type hybridListener struct {
	net.Listener
	tlsConfig *tls.Config

	results chan acceptResult // classified connections, one per accepted conn
	slots   chan struct{}     // bounds concurrent classification
	wg      sync.WaitGroup    // accept loop + in-flight classifications

	closed    chan struct{} // closed by Close — cancels slot acquisition and result delivery
	closeOnce sync.Once
	closeErr  error
}

// acceptResult pairs a classified connection with a terminal accept error.
type acceptResult struct {
	conn net.Conn
	err  error
}

// hybridPeekTimeout bounds how long classification waits for a connection's
// first byte. An individual connection must never fail the listener: an idle
// client hits the peek deadline, a vanished client returns EOF — both become
// ordinary plain-HTTP connections and are handled (and cleaned up)
// per-connection by the http server, whose own read deadline applies from
// there. A var so tests can shorten the shutdown/classification deadline.
var hybridPeekTimeout = 3 * time.Second

// hybridClassifySlots bounds the number of connections being classified at
// once, so a flood of idle connections can neither grow goroutines without
// limit nor starve the accept loop forever.
const hybridClassifySlots = 64

func newHybridListener(ln net.Listener, tlsConfig *tls.Config) net.Listener {
	hl := &hybridListener{
		Listener:  ln,
		tlsConfig: tlsConfig,
		results:   make(chan acceptResult, hybridClassifySlots),
		slots:     make(chan struct{}, hybridClassifySlots),
		closed:    make(chan struct{}),
	}
	hl.wg.Add(1)
	go hl.acceptLoop()
	return hl
}

// Close shuts the listener down explicitly: slot acquisition and result
// delivery are cancelled, connections that never reached a consumer are
// closed, and the accept loop and classification workers are joined before
// the result channel closes. The inherited Listener.Close alone left those
// in flight — idle clients saw a hang instead of a close, and workers
// blocked forever on saturated delivery with no consumer.
func (hl *hybridListener) Close() error {
	hl.closeOnce.Do(func() {
		close(hl.closed)
		hl.closeErr = hl.Listener.Close() // unblock the raw Accept
		hl.wg.Wait()                      // join accept loop + classify workers
		// Connections already classified but never dispatched have no
		// consumer — close them so clients see the shutdown.
		for {
			select {
			case res := <-hl.results:
				if res.conn != nil {
					res.conn.Close()
				}
				continue
			default:
			}
			break
		}
		close(hl.results)
	})
	return hl.closeErr
}

// Accept returns the next classified connection. Classification is performed
// by worker goroutines, so Accept never blocks on a client's first byte.
func (hl *hybridListener) Accept() (net.Conn, error) {
	res, ok := <-hl.results
	if !ok {
		return nil, net.ErrClosed
	}
	return res.conn, res.err
}

// acceptLoop keeps accepting raw connections and hands each one to a bounded
// classification goroutine. The slot pool caps resource usage under a stall
// flood; connections beyond the cap simply wait in the kernel accept queue.
// Every blocking point also selects on hl.closed so shutdown can cancel it.
func (hl *hybridListener) acceptLoop() {
	defer hl.wg.Done()
	for {
		select {
		case <-hl.closed:
			return
		case hl.slots <- struct{}{}: // block while classification is saturated
		}
		c, err := hl.Listener.Accept()
		if err != nil {
			<-hl.slots
			// The raw listener is gone (shutdown). Deliver one terminal
			// error so http.Server.Serve can exit; if nobody is reading
			// anymore or shutdown is cancelling delivery, just return.
			select {
			case hl.results <- acceptResult{err: err}:
			case <-hl.closed:
			default:
			}
			return
		}
		// A connection accepted after shutdown began has no consumer —
		// close it immediately instead of leaking it.
		select {
		case <-hl.closed:
			c.Close()
			<-hl.slots
			return
		default:
		}
		hl.wg.Add(1)
		go func(c net.Conn) {
			defer hl.wg.Done()
			defer func() { <-hl.slots }()
			// Shutdown started while this worker waited — close without
			// classifying.
			select {
			case <-hl.closed:
				c.Close()
				return
			default:
			}
			cl := hl.classify(c)
			select {
			case hl.results <- acceptResult{conn: cl}:
			case <-hl.closed:
				// Shutdown cancels delivery — no consumer will ever read
				// this connection. Close it so the client observes the
				// shutdown instead of hanging on an open socket.
				cl.Close()
			}
		}(c)
	}
}

// classify determines whether a connection speaks TLS by peeking its first
// byte, bounded by the classification deadline.
func (hl *hybridListener) classify(c net.Conn) net.Conn {
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
			return tls.Server(&prefixedConn{Conn: c, r: br}, hl.tlsConfig)
		}
		// Clear the classification deadline before handing the connection
		// to the http server.
		c.SetReadDeadline(time.Time{})
	}
	return &prefixedConn{Conn: c, r: br}
}

// prefixedConn reinserts already-buffered bytes so the http server reads
// the stream exactly as the client sent it.
type prefixedConn struct {
	net.Conn
	r *bufio.Reader
}

func (pc *prefixedConn) Read(p []byte) (int, error) { return pc.r.Read(p) }
