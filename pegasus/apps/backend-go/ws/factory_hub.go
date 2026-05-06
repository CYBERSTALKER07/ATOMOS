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

// FactoryHub maps factory IDs to active realtime websocket connections.
// Factory portal and native factory apps consume this channel for live updates.
type FactoryHub struct {
	mu         sync.RWMutex
	writeMu    sync.Mutex
	clients    map[string][]*websocket.Conn
	subscribed map[string]bool
}

// NewFactoryHub creates a fresh factory websocket hub.
func NewFactoryHub() *FactoryHub {
	return &FactoryHub{
		clients:    make(map[string][]*websocket.Conn),
		subscribed: make(map[string]bool),
	}
}

// HandleConnection upgrades and registers a FACTORY-scoped client.
// Expected path: /v1/ws/factory.
func (h *FactoryHub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	factoryID := auth.EffectiveFactoryID(r.Context())
	if factoryID == "" {
		factoryID = claims.FactoryID
	}
	if factoryID == "" {
		http.Error(w, "factory_id could not be determined from auth token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[FACTORY HUB] WebSocket upgrade failed for %s: %v", factoryID, err)
		return
	}

	h.mu.Lock()
	h.clients[factoryID] = append(h.clients[factoryID], conn)
	total := len(h.clients[factoryID])
	h.mu.Unlock()
	h.subscribeRelay(factoryID)

	log.Printf("[FACTORY HUB] Factory %s connected. Active pipes: %d", factoryID, total)

	done := ConfigureKeepalive(conn)

	defer func() {
		close(done)
		h.mu.Lock()
		conns := h.clients[factoryID]
		for i, c := range conns {
			if c == conn {
				h.clients[factoryID] = append(conns[:i], conns[i+1:]...)
				break
			}
		}
		if len(h.clients[factoryID]) == 0 {
			delete(h.clients, factoryID)
			if h.subscribed[factoryID] {
				delete(h.subscribed, factoryID)
				cache.Unsubscribe("ws:factory:" + factoryID)
			}
		}
		h.mu.Unlock()
		conn.Close()
		log.Printf("[FACTORY HUB] Factory %s disconnected.", factoryID)
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// PushToFactory sends payload to all active connections for a factory.
// Returns true if at least one connection received the payload.
func (h *FactoryHub) PushToFactory(factoryID string, payload interface{}) bool {
	if factoryID == "" {
		return false
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[FACTORY HUB] Failed to marshal payload for %s: %v", factoryID, err)
		return false
	}

	local := h.pushToFactoryLocal(factoryID, data)
	cache.Publish(context.Background(), "ws:factory:"+factoryID, data)
	return local
}

func (h *FactoryHub) pushToFactoryLocal(factoryID string, data []byte) bool {
	h.mu.RLock()
	conns, exists := h.clients[factoryID]
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
			log.Printf("[FACTORY HUB] Write failed for %s — evicting dead pipe: %v", factoryID, err)
			conn.Close()
		} else {
			delivered = true
		}
	}

	return delivered
}

func (h *FactoryHub) subscribeRelay(factoryID string) {
	h.mu.Lock()
	if h.subscribed[factoryID] {
		h.mu.Unlock()
		return
	}
	h.subscribed[factoryID] = true
	h.mu.Unlock()

	channel := "ws:factory:" + factoryID
	cache.Subscribe(channel, func(_ string, payload []byte) {
		h.pushToFactoryLocal(factoryID, payload)
	})
}

// BroadcastSupplyRequestUpdate sends a supply-request state change to a factory room.
func (h *FactoryHub) BroadcastSupplyRequestUpdate(factoryID, requestID, warehouseID, state, action, supplierID string) {
	h.PushToFactory(factoryID, map[string]interface{}{
		"type":         EventFactorySupplyRequestUpdate,
		"factory_id":   factoryID,
		"supplier_id":  supplierID,
		"warehouse_id": warehouseID,
		"request_id":   requestID,
		"state":        state,
		"action":       action,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	})
}

// BroadcastTransferUpdate sends transfer lifecycle changes to a factory room.
func (h *FactoryHub) BroadcastTransferUpdate(factoryID, transferID, warehouseID, manifestID, fromState, toState, action, supplierID string) {
	h.PushToFactory(factoryID, map[string]interface{}{
		"type":         EventFactoryTransferUpdate,
		"factory_id":   factoryID,
		"supplier_id":  supplierID,
		"transfer_id":  transferID,
		"warehouse_id": warehouseID,
		"manifest_id":  manifestID,
		"from_state":   fromState,
		"to_state":     toState,
		"action":       action,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	})
}

// BroadcastManifestUpdate sends manifest lifecycle changes to a factory room.
func (h *FactoryHub) BroadcastManifestUpdate(factoryID, manifestID, state, action, reason, supplierID string, transferIDs []string) {
	h.PushToFactory(factoryID, map[string]interface{}{
		"type":         EventFactoryManifestUpdate,
		"factory_id":   factoryID,
		"supplier_id":  supplierID,
		"manifest_id":  manifestID,
		"state":        state,
		"action":       action,
		"reason":       reason,
		"transfer_ids": transferIDs,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	})
}

// Close gracefully closes all connections in the FactoryHub.
func (h *FactoryHub) Close() {
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
	log.Println("[FACTORY HUB] All connections closed.")
}
