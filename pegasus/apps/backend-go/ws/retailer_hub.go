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

// ─── Retailer WebSocket Hub ────────────────────────────────────────────────────
// Dedicated real-time channel for retailer devices.
// The Proximity Engine's DRIVER_APPROACHING Kafka consumer pushes QR popup
// payloads through this hub before falling back to FCM.

// ApproachPayload is the wire format pushed to the retailer's native app
// to trigger the QR code popup instantly.
type ApproachPayload struct {
	Type            string  `json:"type"` // Always "DRIVER_APPROACHING"
	OrderID         string  `json:"order_id"`
	SupplierID      string  `json:"supplier_id"`
	SupplierName    string  `json:"supplier_name"`
	WarehouseId     string  `json:"warehouse_id,omitempty"`
	WarehouseName   string  `json:"warehouse_name,omitempty"`
	RetailerID      string  `json:"retailer_id"`
	DeliveryToken   string  `json:"delivery_token"` // The token to encode into the QR
	DriverLatitude  float64 `json:"driver_latitude"`
	DriverLongitude float64 `json:"driver_longitude"`
}

// RetailerHub maps retailer IDs to their active WebSocket connections.
// A single retailer can have multiple connections (phone + tablet).
type RetailerHub struct {
	mu         sync.RWMutex
	writeMu    sync.Mutex
	clients    map[string][]*websocket.Conn // Key: RetailerId → active connections
	subscribed map[string]bool              // retailerID → relay subscription active
}

// NewRetailerHub creates a fresh hub instance.
func NewRetailerHub() *RetailerHub {
	return &RetailerHub{
		clients:    make(map[string][]*websocket.Conn),
		subscribed: make(map[string]bool),
	}
}

// HandleConnection upgrades the HTTP request and registers the retailer.
// Expected path: /v1/ws/retailer
func (h *RetailerHub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims == nil || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	retailerID := claims.UserID

	if retailerID == "" {
		http.Error(w, "retailer_id could not be determined from auth token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[RETAILER HUB] WebSocket upgrade failed for %s: %v", retailerID, err)
		return
	}
	h.mu.Lock()
	h.clients[retailerID] = append(h.clients[retailerID], conn)
	total := len(h.clients[retailerID])
	h.mu.Unlock()

	log.Printf("[RETAILER HUB] %s connected. Active pipes: %d", retailerID, total)

	// Subscribe to Redis Pub/Sub relay for cross-pod delivery
	h.subscribeRelay(retailerID)

	// Start keepalive ping/pong to detect stale connections
	done := ConfigureKeepalive(conn)

	defer func() {
		close(done)
		h.mu.Lock()
		conns := h.clients[retailerID]
		for i, c := range conns {
			if c == conn {
				h.clients[retailerID] = append(conns[:i], conns[i+1:]...)
				break
			}
		}
		if len(h.clients[retailerID]) == 0 {
			delete(h.clients, retailerID)
			if h.subscribed[retailerID] {
				delete(h.subscribed, retailerID)
				cache.Unsubscribe("ws:retailer:" + retailerID)
			}
		}
		h.mu.Unlock()
		conn.Close()
		log.Printf("[RETAILER HUB] %s disconnected.", retailerID)
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break // Connection lost
		}
	}
}

// PushToRetailer sends a payload to all active connections for a retailer on
// this pod AND publishes to Redis Pub/Sub for cross-pod relay.
// Returns true if at least one LOCAL connection received the payload.
func (h *RetailerHub) PushToRetailer(retailerID string, payload interface{}) bool {
	local := h.pushToRetailerLocal(retailerID, payload)

	// Cross-pod relay (non-blocking, fail-open)
	data, err := json.Marshal(payload)
	if err == nil {
		cache.Publish(context.Background(), "ws:retailer:"+retailerID, data)
	}

	return local
}

// pushToRetailerLocal sends to local connections only.
func (h *RetailerHub) pushToRetailerLocal(retailerID string, payload interface{}) bool {
	h.mu.RLock()
	conns, exists := h.clients[retailerID]
	if !exists || len(conns) == 0 {
		h.mu.RUnlock()
		return false
	}
	// Snapshot the connections under read lock
	snapshot := make([]*websocket.Conn, len(conns))
	copy(snapshot, conns)
	h.mu.RUnlock()

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[RETAILER HUB] Failed to marshal payload for %s: %v", retailerID, err)
		return false
	}

	delivered := false
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	for _, conn := range snapshot {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[RETAILER HUB] Write failed for %s — evicting dead pipe: %v", retailerID, err)
			conn.Close()
		} else {
			delivered = true
		}
	}

	return delivered
}

// subscribeRelay registers a Redis Pub/Sub handler so messages from other pods
// are delivered to local connections.
func (h *RetailerHub) subscribeRelay(retailerID string) {
	h.mu.Lock()
	if h.subscribed[retailerID] {
		h.mu.Unlock()
		return
	}
	h.subscribed[retailerID] = true
	h.mu.Unlock()

	channel := "ws:retailer:" + retailerID
	cache.Subscribe(channel, func(_ string, payload []byte) {
		// Decode and relay to local connections only
		var msg interface{}
		if err := json.Unmarshal(payload, &msg); err != nil {
			return
		}
		h.pushToRetailerLocal(retailerID, msg)
	})
}

// PushDelta sends a compact DeltaEvent to all active connections for a retailer.
// Fields are auto-compressed using the V.O.I.D. short-key dictionary.
// Includes cross-pod Redis relay for multi-instance deployments.
func (h *RetailerHub) PushDelta(retailerID string, event DeltaEvent) bool {
	if event.TS == 0 {
		event.TS = time.Now().Unix()
	}
	event.D = CompressDelta(event.D)
	return h.PushToRetailer(retailerID, event)
}

// IsConnected returns true if the retailer has at least one active WebSocket pipe.
func (h *RetailerHub) IsConnected(retailerID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conns, exists := h.clients[retailerID]
	return exists && len(conns) > 0
}

// Close gracefully closes all connections in the RetailerHub.
func (h *RetailerHub) Close() {
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
	log.Println("[RETAILER HUB] All connections closed.")
}
