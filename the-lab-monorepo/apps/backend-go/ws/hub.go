package ws

import (
	"log"
	"net/http"
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

	allowed := map[string]bool{
		"http://localhost:3000":  true,
		"http://localhost:3001":  true,
		"http://localhost:3002":  true,
		"http://localhost:8081":  true,
		"http://localhost:19006": true,
	}
	if allowed[origin] {
		return true
	}
	// Dynamic tunnel/Expo/LAN origins for mobile dev
	if strings.HasSuffix(origin, ".ngrok-free.app") || strings.HasSuffix(origin, ".expo.dev") || strings.HasPrefix(origin, "http://192.168.") || strings.HasPrefix(origin, "http://10.0.") {
		return true
	}
	log.Printf("[WS] Rejected WebSocket upgrade from origin: %s", origin)
	return false
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
			log.Printf("[FLEET HUB] Admin pipe broken — evicting client: %v", err)
			client.Close()
			delete(h.clients, client)
		}
	}
}

// HandleConnection upgrades an HTTP request to a permanent WebSocket pipe.
// Works for both Admin (reads) and Driver (writes) clients via the same endpoint.
func (h *FleetHub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[FLEET HUB] WebSocket upgrade failed: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	clientCount := len(h.clients)
	h.mu.Unlock()

	log.Printf("[FLEET HUB] New telemetry pipe opened. Active connections: %d", clientCount)

	// Start keepalive ping/pong
	done := ConfigureKeepalive(conn)

	defer func() {
		close(done)
		h.mu.Lock()
		delete(h.clients, conn)
		remaining := len(h.clients)
		h.mu.Unlock()
		conn.Close()
		log.Printf("[FLEET HUB] Pipe closed. Active connections: %d", remaining)
	}()

	// Listen for incoming GPS pings — Drivers write, Admins only read
	for {
		var update LocationUpdate
		if err := conn.ReadJSON(&update); err != nil {
			// Normal disconnect (app closed, network dropped, etc.)
			break
		}

		log.Printf("[FLEET GPS] %s → Lat: %.6f, Lng: %.6f", update.DriverID, update.Latitude, update.Longitude)
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
	log.Println("[FLEET HUB] All connections closed.")
}
