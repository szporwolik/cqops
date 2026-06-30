package dashboard

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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
		writeJSON(w, http.StatusOK, snap.APRS)
	}
}

func handleHealthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
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
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}
