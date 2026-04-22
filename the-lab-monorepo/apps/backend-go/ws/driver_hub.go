package ws

import (
	"backend-go/auth"
	"backend-go/cache"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ─── Driver WebSocket Hub ──────────────────────────────────────────────────────
// Dedicated real-time channel for driver devices.
// Used to push payment settlement confirmations so the driver knows
// when to enable the "Completed" button after offload.

// DriverPayload is the wire format pushed to the driver's native app.
type DriverPayload struct {
	Type          string `json:"type"` // PAYMENT_SETTLED | PAYMENT_FAILED
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"` // Settled amount
	Message       string `json:"message"`
	WarehouseId   string `json:"warehouse_id,omitempty"`
	WarehouseName string `json:"warehouse_name,omitempty"`
}

// DriverHub maps driver IDs to their active WebSocket connections.
type DriverHub struct {
	mu         sync.RWMutex
	writeMu    sync.Mutex
	clients    map[string][]*websocket.Conn // Key: DriverId → active connections
	subscribed map[string]bool              // driverID → relay subscription active
}

// NewDriverHub creates a fresh hub instance.
func NewDriverHub() *DriverHub {
	return &DriverHub{
		clients:    make(map[string][]*websocket.Conn),
		subscribed: make(map[string]bool),
	}
}

// HandleConnection upgrades the HTTP request and registers the driver.
// Expected path: /v1/ws/driver
func (h *DriverHub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims == nil || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	driverID := claims.UserID

	if driverID == "" {
		http.Error(w, "driver_id could not be determined from auth token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[DRIVER HUB] WebSocket upgrade failed for %s: %v", driverID, err)
		return
	}
	h.mu.Lock()
	h.clients[driverID] = append(h.clients[driverID], conn)
	total := len(h.clients[driverID])
	h.mu.Unlock()
	h.subscribeRelay(driverID)

	log.Printf("[DRIVER HUB] %s connected. Active pipes: %d", driverID, total)

	// Start keepalive ping/pong
	done := ConfigureKeepalive(conn)

	defer func() {
		close(done)
		h.mu.Lock()
		conns := h.clients[driverID]
		for i, c := range conns {
			if c == conn {
				h.clients[driverID] = append(conns[:i], conns[i+1:]...)
				break
			}
		}
		if len(h.clients[driverID]) == 0 {
			delete(h.clients, driverID)
			if h.subscribed[driverID] {
				delete(h.subscribed, driverID)
				cache.Unsubscribe("ws:driver:" + driverID)
			}
		}
		h.mu.Unlock()
		conn.Close()
		log.Printf("[DRIVER HUB] %s disconnected.", driverID)
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// PushToDriver sends a DriverPayload to all active connections for a driver.
// Returns true if at least one connection received the payload.
func (h *DriverHub) PushToDriver(driverID string, payload interface{}) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[DRIVER HUB] Failed to marshal payload for %s: %v", driverID, err)
		return false
	}
	local := h.pushToDriverLocal(driverID, data)
	cache.Publish(context.Background(), "ws:driver:"+driverID, data)
	return local
}

func (h *DriverHub) pushToDriverLocal(driverID string, data []byte) bool {
	h.mu.RLock()
	conns, exists := h.clients[driverID]
	if !exists || len(conns) == 0 {
		h.mu.RUnlock()
		return false
	}
	snapshot := make([]*websocket.Conn, len(conns))
	copy(snapshot, conns)
	h.mu.RUnlock()

	delivered := false
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	for _, conn := range snapshot {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[DRIVER HUB] Write failed for %s — evicting dead pipe: %v", driverID, err)
			conn.Close()
		} else {
			delivered = true
		}
	}

	return delivered
}

func (h *DriverHub) subscribeRelay(driverID string) {
	h.mu.Lock()
	if h.subscribed[driverID] {
		h.mu.Unlock()
		return
	}
	h.subscribed[driverID] = true
	h.mu.Unlock()

	channel := "ws:driver:" + driverID
	cache.Subscribe(channel, func(_ string, payload []byte) {
		h.pushToDriverLocal(driverID, payload)
	})
}

// PushDelta sends a compact DeltaEvent to all active connections for a driver.
// Fields are auto-compressed using the V.O.I.D. short-key dictionary.
func (h *DriverHub) PushDelta(driverID string, event DeltaEvent) bool {
	if event.TS == 0 {
		event.TS = time.Now().Unix()
	}
	event.D = CompressDelta(event.D)
	return h.PushToDriver(driverID, event)
}

// Close gracefully closes all connections in the DriverHub.
func (h *DriverHub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	for id, conns := range h.clients {
		for _, conn := range conns {
			conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down"),
				time.Now().Add(WriteWait))
			conn.Close()
		}
		delete(h.clients, id)
	}
	log.Println("[DRIVER HUB] All connections closed.")
}
