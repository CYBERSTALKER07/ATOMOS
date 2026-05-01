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

// ─── Warehouse WebSocket Hub ──────────────────────────────────────────────────
// Dedicated real-time channel for warehouse terminals and warehouse portal.
// Used to push supply request updates, demand alerts, and dispatch lock events
// so warehouse staff sees live operational state without polling.

// WarehouseHub maps warehouse IDs to their active WebSocket connections.
type WarehouseHub struct {
	mu         sync.RWMutex
	writeMu    sync.Mutex
	clients    map[string][]*websocket.Conn // Key: WarehouseId → active connections
	subscribed map[string]bool              // warehouseID → relay subscription active
}

// NewWarehouseHub creates a fresh hub instance.
func NewWarehouseHub() *WarehouseHub {
	return &WarehouseHub{
		clients:    make(map[string][]*websocket.Conn),
		subscribed: make(map[string]bool),
	}
}

// HandleConnection upgrades the HTTP request and registers the warehouse terminal.
// Expected path: /ws/warehouse
// Identifies the warehouse from JWT claims or query param.
func (h *WarehouseHub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims == nil || claims.WarehouseID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	warehouseID := claims.WarehouseID

	if warehouseID == "" {
		http.Error(w, "warehouse_id could not be determined from auth token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WAREHOUSE HUB] WebSocket upgrade failed for %s: %v", warehouseID, err)
		return
	}
	h.mu.Lock()
	h.clients[warehouseID] = append(h.clients[warehouseID], conn)
	total := len(h.clients[warehouseID])
	h.mu.Unlock()
	h.subscribeRelay(warehouseID)

	log.Printf("[WAREHOUSE HUB] Warehouse %s terminal connected. Active pipes: %d", warehouseID, total)

	// Start keepalive ping/pong to detect stale connections
	done := ConfigureKeepalive(conn)

	defer func() {
		close(done)
		h.mu.Lock()
		conns := h.clients[warehouseID]
		for i, c := range conns {
			if c == conn {
				h.clients[warehouseID] = append(conns[:i], conns[i+1:]...)
				break
			}
		}
		if len(h.clients[warehouseID]) == 0 {
			delete(h.clients, warehouseID)
			if h.subscribed[warehouseID] {
				delete(h.subscribed, warehouseID)
				cache.Unsubscribe("ws:warehouse:" + warehouseID)
			}
		}
		h.mu.Unlock()
		conn.Close()
		log.Printf("[WAREHOUSE HUB] Warehouse %s terminal disconnected.", warehouseID)
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// PushToWarehouse sends a payload to all active warehouse connections.
// Returns true if at least one connection received the payload.
func (h *WarehouseHub) PushToWarehouse(warehouseID string, payload interface{}) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[WAREHOUSE HUB] Failed to marshal payload for %s: %v", warehouseID, err)
		return false
	}
	local := h.pushToWarehouseLocal(warehouseID, data)
	cache.Publish(context.Background(), "ws:warehouse:"+warehouseID, data)
	return local
}

func (h *WarehouseHub) pushToWarehouseLocal(warehouseID string, data []byte) bool {
	h.mu.RLock()
	conns, exists := h.clients[warehouseID]
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
			log.Printf("[WAREHOUSE HUB] Write failed for %s — evicting dead pipe: %v", warehouseID, err)
			conn.Close()
		} else {
			delivered = true
		}
	}

	return delivered
}

func (h *WarehouseHub) subscribeRelay(warehouseID string) {
	h.mu.Lock()
	if h.subscribed[warehouseID] {
		h.mu.Unlock()
		return
	}
	h.subscribed[warehouseID] = true
	h.mu.Unlock()

	channel := "ws:warehouse:" + warehouseID
	cache.Subscribe(channel, func(_ string, payload []byte) {
		h.pushToWarehouseLocal(warehouseID, payload)
	})
}

// BroadcastSupplyRequestUpdate pushes a supply request state change to the warehouse.
func (h *WarehouseHub) BroadcastSupplyRequestUpdate(warehouseID, requestID, state string) {
	h.PushToWarehouse(warehouseID, map[string]interface{}{
		"type":       EventSupplyRequestUpdate,
		"request_id": requestID,
		"state":      state,
	})
}

// BroadcastDispatchLockChange pushes a dispatch lock state change to the warehouse.
func (h *WarehouseHub) BroadcastDispatchLockChange(warehouseID, lockID, action string) {
	h.PushToWarehouse(warehouseID, map[string]interface{}{
		"type":    EventDispatchLockChange,
		"lock_id": lockID,
		"action":  action, // ACQUIRED | RELEASED
	})
}

// BroadcastOutboxFailure pushes an OUTBOX_FAILED event to ALL connected
// warehouse clients. Used by the outbox relay failure callback to surface
// publish errors on the admin portal in real time.
func (h *WarehouseHub) BroadcastOutboxFailure(eventID, aggregateID, topic, reason string) {
	payload := map[string]interface{}{
		"type":         "OUTBOX_FAILED",
		"event_id":     eventID,
		"aggregate_id": aggregateID,
		"topic":        topic,
		"reason":       reason,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}
	h.mu.RLock()
	ids := make([]string, 0, len(h.clients))
	for wid := range h.clients {
		ids = append(ids, wid)
	}
	h.mu.RUnlock()
	for _, wid := range ids {
		h.PushToWarehouse(wid, payload)
	}
}

// PushDelta sends a compact DeltaEvent to all active warehouse connections.
// Fields are auto-compressed using the V.O.I.D. short-key dictionary.
func (h *WarehouseHub) PushDelta(warehouseID string, event DeltaEvent) bool {
	if event.TS == 0 {
		event.TS = time.Now().Unix()
	}
	event.D = CompressDelta(event.D)
	return h.PushToWarehouse(warehouseID, event)
}

// Close gracefully closes all connections in the WarehouseHub.
func (h *WarehouseHub) Close() {
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
	log.Println("[WAREHOUSE HUB] All connections closed.")
}
