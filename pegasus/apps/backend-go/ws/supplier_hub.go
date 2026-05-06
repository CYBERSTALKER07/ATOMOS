package ws

import (
	"backend-go/auth"
	"backend-go/cache"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ─── Supplier WebSocket Hub ───────────────────────────────────────────────────
// Dedicated real-time channel for supplier administrative clients (Admin Portal).
// Used to push notifications, live dashboard updates, and other real-time events without polling.

// SupplierHub maps supplier IDs to their active Admin Portal WebSocket connections.
// Supplier connections are scoped to a supplier — they receive events for that supplier's data.
type SupplierHub struct {
	mu         sync.RWMutex
	writeMu    sync.Mutex
	clients    map[string][]*websocket.Conn // Key: SupplierId → active connections
	subscribed map[string]bool              // supplierID → relay subscription active
}

// NewSupplierHub creates a fresh hub instance.
func NewSupplierHub() *SupplierHub {
	return &SupplierHub{
		clients:    make(map[string][]*websocket.Conn),
		subscribed: make(map[string]bool),
	}
}

// HandleConnection upgrades the HTTP request and registers the supplier connection.
// Expected path: /v1/ws/supplier
// Identifies the supplier from JWT claims or query param.
func (h *SupplierHub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims == nil || claims.ResolveSupplierID() == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	supplierID := claims.ResolveSupplierID()
	traceID := r.Header.Get("X-Trace-Id")
	if traceID == "" {
		traceID = r.Header.Get("X-Request-Id")
	}

	if supplierID == "" {
		http.Error(w, "supplier_id could not be determined from auth token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "supplier hub websocket upgrade failed",
			"hub", "supplier",
			"supplier_id", supplierID,
			"trace_id", traceID,
			"error", err,
		)
		return
	}
	h.mu.Lock()
	h.clients[supplierID] = append(h.clients[supplierID], conn)
	total := len(h.clients[supplierID])
	h.mu.Unlock()
	h.subscribeRelay(supplierID)

	slog.InfoContext(r.Context(), "supplier hub client connected",
		"hub", "supplier",
		"supplier_id", supplierID,
		"active_pipes", total,
		"trace_id", traceID,
	)

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
				cache.Unsubscribe("ws:supplier:" + supplierID)
			}
		}
		h.mu.Unlock()
		conn.Close()
		slog.InfoContext(r.Context(), "supplier hub client disconnected",
			"hub", "supplier",
			"supplier_id", supplierID,
			"trace_id", traceID,
		)
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// PushToSupplier sends a payload to all active supplier connections for a supplier.
// Returns true if at least one connection received the payload.
func (h *SupplierHub) PushToSupplier(supplierID string, payload interface{}) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("supplier hub payload marshal failed",
			"hub", "supplier",
			"supplier_id", supplierID,
			"error", err,
		)
		return false
	}
	local := h.pushToSupplierLocal(supplierID, data)
	cache.Publish(context.Background(), "ws:supplier:"+supplierID, data)
	return local
}

func (h *SupplierHub) pushToSupplierLocal(supplierID string, data []byte) bool {
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
			slog.Warn("supplier hub write failed; evicting dead connection",
				"hub", "supplier",
				"supplier_id", supplierID,
				"error", err,
			)
			conn.Close()
		} else {
			delivered = true
		}
	}

	return delivered
}

func (h *SupplierHub) subscribeRelay(supplierID string) {
	h.mu.Lock()
	if h.subscribed[supplierID] {
		h.mu.Unlock()
		return
	}
	h.subscribed[supplierID] = true
	h.mu.Unlock()

	channel := "ws:supplier:" + supplierID
	cache.Subscribe(channel, func(_ string, payload []byte) {
		h.pushToSupplierLocal(supplierID, payload)
	})
}

// Close gracefully closes all connections in the SupplierHub.
func (h *SupplierHub) Close() {
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
	slog.Info("supplier hub closed all connections", "hub", "supplier")
}
