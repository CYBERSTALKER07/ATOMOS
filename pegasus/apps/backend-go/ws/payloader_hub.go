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

// ─── Payloader WebSocket Hub ───────────────────────────────────────────────────
// Dedicated real-time channel for payload terminal devices.
// Used to push PAYLOAD_READY_TO_SEAL notifications so the payloader knows
// when new orders are ready for sealing without polling.

// PayloaderHub maps supplier IDs to their active payloader WebSocket connections.
// Payloader terminals are scoped to a supplier — they receive events for that supplier's orders.
type PayloaderHub struct {
	mu         sync.RWMutex
	writeMu    sync.Mutex
	clients    map[string][]*websocket.Conn // Key: SupplierId → active connections
	subscribed map[string]bool              // supplierID → relay subscription active
}

// NewPayloaderHub creates a fresh hub instance.
func NewPayloaderHub() *PayloaderHub {
	return &PayloaderHub{
		clients:    make(map[string][]*websocket.Conn),
		subscribed: make(map[string]bool),
	}
}

// HandleConnection upgrades the HTTP request and registers the payloader terminal.
// Expected path: /v1/ws/payloader
// Identifies the supplier from JWT claims or query param.
func (h *PayloaderHub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims == nil || claims.ResolveSupplierID() == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	supplierID := claims.ResolveSupplierID()

	if supplierID == "" {
		http.Error(w, "supplier_id could not be determined from auth token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[PAYLOADER HUB] WebSocket upgrade failed for %s: %v", supplierID, err)
		return
	}
	h.mu.Lock()
	h.clients[supplierID] = append(h.clients[supplierID], conn)
	total := len(h.clients[supplierID])
	h.mu.Unlock()
	h.subscribeRelay(supplierID)

	log.Printf("[PAYLOADER HUB] Supplier %s terminal connected. Active pipes: %d", supplierID, total)

	// Start keepalive ping/pong to detect stale connections
	done := ConfigureKeepalive(conn)

	defer func() {
		close(done)
		h.mu.Lock()
		conns := h.clients[supplierID]
		for i, c := range conns {
			if c == conn {
				h.clients[supplierID] = append(conns[:i], conns[i+1:]...)
				break
			}
		}
		if len(h.clients[supplierID]) == 0 {
			delete(h.clients, supplierID)
			if h.subscribed[supplierID] {
				delete(h.subscribed, supplierID)
				cache.Unsubscribe("ws:payloader:" + supplierID)
			}
		}
		h.mu.Unlock()
		conn.Close()
		log.Printf("[PAYLOADER HUB] Supplier %s terminal disconnected.", supplierID)
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// PushToPayloader sends a payload to all active payloader connections for a supplier.
// Returns true if at least one connection received the payload.
func (h *PayloaderHub) PushToPayloader(supplierID string, payload interface{}) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[PAYLOADER HUB] Failed to marshal payload for %s: %v", supplierID, err)
		return false
	}
	local := h.pushToPayloaderLocal(supplierID, data)
	cache.Publish(context.Background(), "ws:payloader:"+supplierID, data)
	return local
}

func (h *PayloaderHub) pushToPayloaderLocal(supplierID string, data []byte) bool {
	h.mu.RLock()
	conns, exists := h.clients[supplierID]
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
			log.Printf("[PAYLOADER HUB] Write failed for %s — evicting dead pipe: %v", supplierID, err)
			conn.Close()
		} else {
			delivered = true
		}
	}

	return delivered
}

func (h *PayloaderHub) subscribeRelay(supplierID string) {
	h.mu.Lock()
	if h.subscribed[supplierID] {
		h.mu.Unlock()
		return
	}
	h.subscribed[supplierID] = true
	h.mu.Unlock()

	channel := "ws:payloader:" + supplierID
	cache.Subscribe(channel, func(_ string, payload []byte) {
		h.pushToPayloaderLocal(supplierID, payload)
	})
}

// Close gracefully closes all connections in the PayloaderHub.
func (h *PayloaderHub) Close() {
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
	log.Println("[PAYLOADER HUB] All connections closed.")
}
