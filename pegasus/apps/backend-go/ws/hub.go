package ws

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// LocationUpdate is the GPS payload from a Driver
type LocationUpdate struct {
	DriverID  string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: CheckWSOrigin,
}

// CheckWSOrigin validates the Origin header against the allowlist.
// Mobile native apps (no Origin header) are allowed through.
// Exported so telemetry/hub.go can share the same origin policy.
func CheckWSOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// No Origin header — native mobile clients or same-origin
		return true
	}

	allowed := parseWSAllowlist()
	if allowed[origin] {
		return true
	}
	if isPatternAllowed(origin) {
		return true
	}
	slog.Warn("websocket origin rejected", "hub", "fleet", "origin", origin)
	return false
}

func parseWSAllowlist() map[string]bool {
	allowlist := make(map[string]bool)
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw != "" {
		for _, entry := range strings.Split(raw, ",") {
			origin := strings.TrimSpace(entry)
			if origin != "" {
				allowlist[origin] = true
			}
		}
	}

	if len(allowlist) == 0 {
		environment := strings.ToLower(strings.TrimSpace(os.Getenv("ENVIRONMENT")))
		if environment != "production" {
			allowlist["http://localhost:3000"] = true
			allowlist["http://localhost:3001"] = true
			allowlist["http://localhost:3002"] = true
			allowlist["http://localhost:8081"] = true
			allowlist["http://localhost:19006"] = true
		}
	}

	return allowlist
}

func isPatternAllowed(origin string) bool {
	// Dynamic tunnel/Expo/LAN origins for mobile development.
	return strings.HasSuffix(origin, ".ngrok-free.app") ||
		strings.HasSuffix(origin, ".expo.dev") ||
		strings.HasPrefix(origin, "http://192.168.") ||
		strings.HasPrefix(origin, "http://10.0.")
}

// FleetHub holds all connected WebSocket clients (Admin Portals + Drivers)
type FleetHub struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}

func NewFleetHub() *FleetHub {
	return &FleetHub{
		clients: make(map[*websocket.Conn]bool),
	}
}

// Broadcast sends the GPS ping to every connected Admin client
func (h *FleetHub) Broadcast(update LocationUpdate) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		err := client.WriteJSON(update)
		if err != nil {
			slog.Warn("fleet hub write failed; evicting connection",
				"hub", "fleet",
				"error", err,
			)
			client.Close()
			delete(h.clients, client)
		}
	}
}

// HandleConnection upgrades an HTTP request to a permanent WebSocket pipe.
// Works for both Admin (reads) and Driver (writes) clients via the same endpoint.
func (h *FleetHub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	traceID := r.Header.Get("X-Trace-Id")
	if traceID == "" {
		traceID = r.Header.Get("X-Request-Id")
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "fleet hub websocket upgrade failed",
			"hub", "fleet",
			"trace_id", traceID,
			"error", err,
		)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	clientCount := len(h.clients)
	h.mu.Unlock()

	slog.InfoContext(r.Context(), "fleet hub client connected",
		"hub", "fleet",
		"active_connections", clientCount,
		"trace_id", traceID,
	)

	// Start keepalive ping/pong
	done := ConfigureKeepalive(conn)

	defer func() {
		close(done)
		h.mu.Lock()
		delete(h.clients, conn)
		remaining := len(h.clients)
		h.mu.Unlock()
		conn.Close()
		slog.InfoContext(r.Context(), "fleet hub client disconnected",
			"hub", "fleet",
			"active_connections", remaining,
			"trace_id", traceID,
		)
	}()

	// Listen for incoming GPS pings — Drivers write, Admins only read
	for {
		var update LocationUpdate
		if err := conn.ReadJSON(&update); err != nil {
			// Normal disconnect (app closed, network dropped, etc.)
			break
		}

		slog.Info("fleet gps update received",
			"hub", "fleet",
			"driver_id", update.DriverID,
			"latitude", update.Latitude,
			"longitude", update.Longitude,
		)
		// Immediately fan the ping out to all connected Admin dashboards
		h.Broadcast(update)
	}
}

// Close gracefully closes all connections in the FleetHub.
func (h *FleetHub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		client.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down"),
			time.Now().Add(WriteWait))
		client.Close()
		delete(h.clients, client)
	}
	slog.Info("fleet hub closed all connections", "hub", "fleet")
}
