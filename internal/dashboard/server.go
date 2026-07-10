package dashboard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
)

// Server wraps the net/http server and dashboard state for the CQOps Live
// browser dashboard.
type Server struct {
	addr   string
	http   *http.Server
	state  *State
	hub    *Hub
	status chan bool

	mu  sync.Mutex // guards err and http
	err error
}

// New creates a Server that will listen on the given address and port once
// Start is called.
func New(address, port string) *Server {
	hub := NewHub()
	return &Server{
		addr:   net.JoinHostPort(address, port),
		state:  NewState(hub),
		hub:    hub,
		status: make(chan bool, 4),
	}
}

// State returns the thread-safe dashboard state. The TUI writes to it;
// HTTP handlers read from it.
func (s *Server) State() *State { return s.state }

// Hub returns the SSE event hub.
func (s *Server) Hub() *Hub { return s.hub }

// Start begins listening in a background goroutine.
// Safe to call multiple times — subsequent calls are no-ops.
func (s *Server) Start() {
	s.mu.Lock()
	if s.http != nil {
		s.mu.Unlock()
		applog.Debug("dashboard: Start called but already running", "addr", s.addr)
		return
	}
	s.mu.Unlock()

	applog.Info("dashboard: starting", "addr", s.addr)

	mux := NewMux(s.state, s.hub)
	handler := securityHeaders(mux)

	srv := &http.Server{
		Addr:         s.addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // disabled — SSE streams never finish writing
		IdleTimeout:  120 * time.Second,
	}

	s.mu.Lock()
	s.http = srv
	s.mu.Unlock()

	go func() {
		ln, err := net.Listen("tcp", s.addr)
		if err != nil {
			applog.Error("dashboard: cannot bind", "addr", s.addr, "error", err)
			s.mu.Lock()
			s.err = err
			s.http = nil
			s.mu.Unlock()
			s.status <- false
			return
		}
		s.status <- true
		applog.Info("dashboard: listening", "addr", s.addr, "url", fmt.Sprintf("http://%s", s.addr))

		err = srv.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			applog.Error("dashboard: serve error", "addr", s.addr, "error", err)
			s.mu.Lock()
			s.err = err
			s.mu.Unlock()
		}
		s.status <- false
	}()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() {
	s.mu.Lock()
	srv := s.http
	s.mu.Unlock()

	if srv == nil {
		return
	}
	applog.Info("dashboard: stopping", "addr", s.addr)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		applog.Warn("dashboard: shutdown error", "addr", s.addr, "error", err)
	}

	s.mu.Lock()
	s.http = nil
	s.err = nil
	s.mu.Unlock()
}

// Status returns a receive-only channel that reports server state changes.
func (s *Server) Status() <-chan bool { return s.status }

// Error returns the last error encountered, or nil.
func (s *Server) Error() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Addr returns the address the server was configured to listen on.
func (s *Server) Addr() string { return s.addr }

// securityHeaders wraps an http.Handler with basic hardening headers suitable
// for a same-origin LAN dashboard. Not a substitute for a reverse proxy in
// untrusted environments.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Restrict to same-origin by default; style-src 'unsafe-inline'
		// needed for Leaflet. If your deployment needs external map tiles
		// or weather APIs, adjust img-src and connect-src accordingly.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: https:; "+
				"connect-src 'self' https:; "+
				"font-src 'self'")
		next.ServeHTTP(w, r)
	})
}
