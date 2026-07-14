package dashboard

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/szporwolik/cqops/assets"
)

// NewMux builds the HTTP handler tree for the dashboard.
func NewMux(state *State, hub *Hub) *http.ServeMux {
	mux := http.NewServeMux()

	// Static files served directly from the embedded filesystem.
	// go:embed static/* embeds files at "static/index.html", etc.
	// We read the embed.FS by those exact paths to avoid FileServer
	// redirect/prefix issues.
	mux.HandleFunc("/", serveStaticFile("static/index.html", "text/html; charset=utf-8"))
	mux.HandleFunc("/index.html", serveStaticFile("static/index.html", "text/html; charset=utf-8"))
	mux.HandleFunc("/app.js", serveStaticFile("static/app.js", "application/javascript"))
	mux.HandleFunc("/style.css", serveStaticFile("static/style.css", "text/css"))
	mux.HandleFunc("/leaflet.js", serveStaticFile("static/leaflet.js", "application/javascript"))
	mux.HandleFunc("/leaflet.css", serveStaticFile("static/leaflet.css", "text/css"))
	mux.HandleFunc("/terminator.js", serveStaticFile("static/terminator.js", "application/javascript"))
	mux.HandleFunc("/favicon.ico", serveStaticFile("static/favicon.ico", "image/x-icon"))
	mux.HandleFunc("/images/marker-icon.png", serveStaticFile("static/images/marker-icon.png", "image/png"))
	mux.HandleFunc("/images/marker-icon-2x.png", serveStaticFile("static/images/marker-icon-2x.png", "image/png"))
	mux.HandleFunc("/images/marker-shadow.png", serveStaticFile("static/images/marker-shadow.png", "image/png"))

	// APRS symbol sprite sheets (hessu/aprs-symbols, 24px base with @2x for retina).
	mux.HandleFunc("/images/symbols/aprs-symbols-24-0.png", serveStaticFile("static/images/symbols/aprs-symbols-24-0.png", "image/png"))
	mux.HandleFunc("/images/symbols/aprs-symbols-24-1.png", serveStaticFile("static/images/symbols/aprs-symbols-24-1.png", "image/png"))
	mux.HandleFunc("/images/symbols/aprs-symbols-24-2.png", serveStaticFile("static/images/symbols/aprs-symbols-24-2.png", "image/png"))

	// API endpoints.
	mux.HandleFunc("/api/snapshot", handleSnapshot(state))
	mux.HandleFunc("/api/recent", handleRecent(state))
	mux.HandleFunc("/api/today", handleToday(state))
	mux.HandleFunc("/api/stats", handleStats(state))
	mux.HandleFunc("/api/map", handleMap(state))
	mux.HandleFunc("/api/aprs", handleAPRS(state))
	mux.HandleFunc("/api/events", handleEvents(state, hub))
	mux.HandleFunc("/healthz", handleHealthz())

	// Embedded logo — served from binary, no internet required.
	mux.HandleFunc("/logo.png", handleLogo())

	// Embedded world map — offline fallback for Leaflet when tiles are unavailable.
	mux.HandleFunc("/api/map-earth", handleMapEarth())

	// Radar tile proxy — avoids CORS/ORB issues with RainViewer CDN.
	mux.HandleFunc("/radar-proxy/", handleRadarProxy())

	return mux
}

func handleSnapshot(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, state.Snapshot())
	}
}

func handleRecent(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		snap := state.Snapshot()
		recent := snap.Recent
		if recent == nil {
			recent = []QSOView{}
		}
		writeJSON(w, http.StatusOK, recent)
	}
}

func handleToday(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		snap := state.Snapshot()
		today := snap.Today
		if today == nil {
			today = []QSOView{}
		}
		writeJSON(w, http.StatusOK, today)
	}
}

func handleStats(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		snap := state.Snapshot()
		writeJSON(w, http.StatusOK, snap.Stats)
	}
}

func handleMap(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		snap := state.Snapshot()
		writeJSON(w, http.StatusOK, snap.Map)
	}
}

func handleAPRS(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		snap := state.Snapshot()
		aprs := snap.APRS
		if aprs == nil {
			aprs = []APRSStation{}
		}
		writeJSON(w, http.StatusOK, aprs)
	}
}

func handleHealthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}
}

// handleLogo serves the embedded CQOps logo PNG from the binary.
func handleLogo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(assets.Logo)
	}
}

// handleMapEarth serves the embedded equirectangular world map JPEG for
// offline Leaflet fallback when tile servers are unreachable.
func handleMapEarth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(assets.WorldMap)
	}
}

// handleEvents serves the SSE stream.
func handleEvents(state *State, hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		// Disable the server-wide WriteTimeout for this SSE stream.
		// The Go HTTP server's WriteTimeout normally kills idle writes;
		// for a never-ending SSE response we manage liveness ourselves via
		// heartbeats and client disconnect detection.
		rc := http.NewResponseController(w)
		rc.SetWriteDeadline(time.Time{})

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		// Send initial snapshot immediately.
		ev := Event{
			ID:        0,
			Type:      string(EventSnapshot),
			Timestamp: timeNow(),
			Payload:   state.Snapshot(),
		}
		writeSSE(w, ev)
		flusher.Flush()

		// Subscribe to hub.
		ch := hub.Subscribe()
		defer hub.Unsubscribe(ch)

		// Heartbeat ticker — must fire well within any proxy/CDN/browser
		// idle timeout. 8s is safe for typical 10–15s write timeouts.
		heartbeat := time.NewTicker(8 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-heartbeat.C:
				ev := Event{
					ID:        0,
					Type:      string(EventHeartbeat),
					Timestamp: timeNow(),
				}
				writeSSE(w, ev)
				flusher.Flush()
			case ev, ok := <-ch:
				if !ok {
					return
				}
				writeSSE(w, ev)
				flusher.Flush()
			}
		}
	}
}

// writeSSE writes a single event in SSE wire format.
func writeSSE(w io.Writer, ev Event) {
	fmt.Fprintf(w, "id: %d\n", ev.ID)
	fmt.Fprintf(w, "event: %s\n", ev.Type)
	data, _ := json.Marshal(ev)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// writeJSON writes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Best-effort — headers already sent.
	}
}

// serveStaticFile returns a handler that serves a single file from the
// embedded filesystem with the given content type.
func serveStaticFile(embedPath, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := staticFS.ReadFile(embedPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}

// handleRadarProxy proxies radar tile requests to the RainViewer CDN.
// This avoids browser CORS/ORB blocking and the 429 rate-limiting that
// occurs when many no-cors tile requests flood the CDN from a single client.
func handleRadarProxy() http.HandlerFunc {
	client := &http.Client{
		Timeout: 12 * time.Second,
	}
	upstream := "https://tilecache.rainviewer.com"

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/radar-proxy")
		if path == r.URL.Path || path == "" || path == "/" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), r.Method, upstream+path, nil)
		if err != nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Only cache successful tile responses; 429/5xx should not be cached.
		if resp.StatusCode == http.StatusOK {
			w.Header().Set("Cache-Control", "public, max-age=600")
		}
		if ct := resp.Header.Get("Content-Type"); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}
