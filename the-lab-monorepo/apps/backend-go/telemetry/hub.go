package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/proximity"
	wsEvents "backend-go/ws"

	"cloud.google.com/go/spanner"
	"github.com/gorilla/websocket"
)

// In production, restrict CheckOrigin to your Next.js and React Native domains
var upgrader = websocket.Upgrader{
	CheckOrigin: wsEvents.CheckWSOrigin,
}

// GPSPayload defines the exact packet fired by the React Native app
type GPSPayload struct {
	DriverID  string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

// clientMeta holds per-connection metadata for supplier-scoped broadcasting
type clientMeta struct {
	Role       string // "DRIVER" or "ADMIN"
	SupplierID string // For ADMIN: claims.UserID. For DRIVER: resolved from Drivers.SupplierId
	DriverID   string // Only set for DRIVER role
}

// Hub manages the active WebSocket connections
type Hub struct {
	sync.RWMutex
	Clients         map[*websocket.Conn]*clientMeta // Maps connection to metadata
	ProximityEngine *proximity.Engine               // Phase 2: Redis GEO proximity detector (nil-safe)
	Spanner         *spanner.Client                 // For resolving Driver→Supplier ownership
	Buffer          *GPSBuffer                      // GPS buffer for batched flush (nil = direct broadcast)
	// driverSupplierCache maps driverID → supplierID to avoid repeated Spanner reads
	driverSupplierCache sync.Map
}

var FleetHub = &Hub{
	Clients: make(map[*websocket.Conn]*clientMeta),
}

// resolveDriverSupplier looks up which supplier owns a driver (cached in-memory)
func (h *Hub) resolveDriverSupplier(ctx context.Context, driverID string) string {
	if cached, ok := h.driverSupplierCache.Load(driverID); ok {
		return cached.(string)
	}
	if h.Spanner == nil {
		return ""
	}
	row, err := h.Spanner.Single().ReadRow(ctx, "Drivers", spanner.Key{driverID}, []string{"SupplierId"})
	if err != nil {
		return ""
	}
	var sid spanner.NullString
	if err := row.Columns(&sid); err != nil || !sid.Valid {
		return ""
	}
	h.driverSupplierCache.Store(driverID, sid.StringVal)
	return sid.StringVal
}

// HandleConnection upgrades the HTTP request to a persistent WebSocket
func (h *Hub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// 1. Authenticate and Extract Role via Context Auth Guard
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || (claims.Role != "DRIVER" && claims.Role != "ADMIN") {
		http.Error(w, "Unauthorized telemetry access", http.StatusUnauthorized)
		return
	}
	role := claims.Role

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("[TELEMETRY_FAULT] Protocol upgrade failed: %v\n", err)
		return
	}
	defer ws.Close()

	// 2. Build metadata with supplier scoping
	meta := &clientMeta{Role: role}
	if role == "ADMIN" {
		// For ADMIN/SUPPLIER: use ResolveSupplierID to get the actual org-level supplier ID
		meta.SupplierID = claims.ResolveSupplierID()
	} else if role == "DRIVER" {
		meta.DriverID = claims.UserID
		meta.SupplierID = h.resolveDriverSupplier(r.Context(), claims.UserID)
	}

	// 3. Register the connection
	h.Lock()
	h.Clients[ws] = meta
	h.Unlock()

	// 3a. Subscribe to Redis Pub/Sub relay for this supplier (cross-pod broadcast)
	if meta.SupplierID != "" {
		h.SubscribeSupplierRelay(meta.SupplierID)
	}

	// 3b. Start keepalive (Phase 5.3)
	ws.SetReadDeadline(time.Now().Add(65 * time.Second))
	ws.SetPongHandler(func(appData string) error {
		ws.SetReadDeadline(time.Now().Add(65 * time.Second))
		return nil
	})
	pingDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-pingDone:
				return
			case <-ticker.C:
				ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
					return
				}
			}
		}
	}()

	fmt.Printf("[TELEMETRY] %s (supplier=%s) linked to the matrix.\n", role, meta.SupplierID)

	// Send TOKEN_REFRESH_NEEDED if operating in grace period (A-4)
	if claims.GracePeriod {
		refreshMsg, _ := json.Marshal(map[string]string{
			"type":    wsEvents.EventTokenRefreshNeeded,
			"message": "Your authentication token has expired. Please refresh to maintain full access.",
		})
		ws.WriteMessage(websocket.TextMessage, refreshMsg)
	}

	// 4. The Event Loop
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			// Connection dropped (e.g., driver went into a tunnel)
			close(pingDone)
			h.Lock()
			delete(h.Clients, ws)
			h.Unlock()
			fmt.Printf("[TELEMETRY] %s connection severed.\n", role)
			break
		}

		// 5. Ingest and Route
		// Drivers are the only ones permitted to emit coordinates
		if role == "DRIVER" {
			var payload GPSPayload
			if err := json.Unmarshal(msg, &payload); err == nil {
				// Route GPS through buffer for batched flush (95% packet reduction)
				if h.Buffer != nil {
					h.Buffer.Ingest(GPSEntry{
						DriverID:   payload.DriverID,
						Latitude:   payload.Latitude,
						Longitude:  payload.Longitude,
						Timestamp:  payload.Timestamp,
						SupplierID: meta.SupplierID,
					})
				} else {
					// Fallback: direct broadcast (buffer not initialized)
					h.BroadcastToSupplier(meta.SupplierID, msg)
				}
				// Phase 2: Feed the proximity engine (non-blocking goroutine)
				if h.ProximityEngine != nil {
					go h.ProximityEngine.ProcessPing(context.Background(), payload.DriverID, payload.Latitude, payload.Longitude)
				}
			}
		}
	}
}

// BroadcastToSupplier fans out the GPS payload to ADMIN connections belonging
// to the same supplier on THIS pod, AND publishes to Redis Pub/Sub so other
// pods can relay to their local connections.
func (h *Hub) BroadcastToSupplier(supplierID string, message []byte) {
	h.broadcastToSupplierLocal(supplierID, message)

	// Cross-pod relay via Redis Pub/Sub (non-blocking, fail-open)
	cache.Publish(context.Background(), "ws:supplier:"+supplierID, message)
}

// broadcastToSupplierLocal sends to local connections only (called by both
// the direct path and the Redis Pub/Sub relay handler).
func (h *Hub) broadcastToSupplierLocal(supplierID string, message []byte) {
	h.RLock()
	var dead []*websocket.Conn
	for client, meta := range h.Clients {
		if meta.Role == "ADMIN" && meta.SupplierID == supplierID {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				client.Close()
				dead = append(dead, client)
			}
		}
	}
	h.RUnlock()

	if len(dead) > 0 {
		h.Lock()
		for _, c := range dead {
			delete(h.Clients, c)
		}
		h.Unlock()
	}
}

// SubscribeSupplierRelay registers a Redis Pub/Sub handler so that messages
// published by OTHER pods for this supplier are relayed to local connections.
func (h *Hub) SubscribeSupplierRelay(supplierID string) {
	channel := "ws:supplier:" + supplierID
	cache.Subscribe(channel, func(_ string, payload []byte) {
		// Relay to local connections only (no re-publish to avoid loops)
		h.broadcastToSupplierLocal(supplierID, payload)
	})
}

// BroadcastToAdmins fans out GPS to all connected control towers (kept for backward compat / internal use)
func (h *Hub) BroadcastToAdmins(message []byte) {
	h.RLock()
	var dead []*websocket.Conn
	for client, meta := range h.Clients {
		if meta.Role == "ADMIN" {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				client.Close()
				dead = append(dead, client)
			}
		}
	}
	h.RUnlock()

	if len(dead) > 0 {
		h.Lock()
		for _, c := range dead {
			delete(h.Clients, c)
		}
		h.Unlock()
	}
}

// ── Order State Change Push ─────────────────────────────────────────────────
// BroadcastOrderStateChange pushes an ORDER_STATE_CHANGED event to all admin
// WebSocket connections belonging to the specified supplier. This enables
// instant UI refresh on the admin portal instead of relying on REST polling.
func (h *Hub) BroadcastOrderStateChange(supplierID, orderID, newState, driverID string) {
	payload := map[string]interface{}{
		"type":      wsEvents.EventOrderStateChanged,
		"order_id":  orderID,
		"state":     newState,
		"driver_id": driverID,
		"timestamp": time.Now().UnixMilli(),
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[TELEMETRY] Failed to marshal ORDER_STATE_CHANGED for %s: %v", orderID, err)
		return
	}
	h.BroadcastToSupplier(supplierID, msg)
}

// BroadcastDriverApproaching pushes a DRIVER_APPROACHING event to the supplier's
// admin portal so the control tower sees the approach alongside the retailer.
func (h *Hub) BroadcastDriverApproaching(supplierID, orderID string, driverLat, driverLng float64) {
	payload := map[string]interface{}{
		"type":             wsEvents.EventDriverApproaching,
		"order_id":         orderID,
		"driver_latitude":  driverLat,
		"driver_longitude": driverLng,
		"timestamp":        time.Now().UnixMilli(),
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[TELEMETRY] Failed to marshal DRIVER_APPROACHING for %s: %v", orderID, err)
		return
	}
	h.BroadcastToSupplier(supplierID, msg)
}

// BroadcastETAUpdate pushes an ETA_UPDATED event to the supplier's admin portal
// so the fleet page refreshes driver/order ETAs in real time without polling.
func (h *Hub) BroadcastETAUpdate(supplierID, driverID string) {
	payload := map[string]interface{}{
		"type":      wsEvents.EventETAUpdated,
		"driver_id": driverID,
		"timestamp": time.Now().UnixMilli(),
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[TELEMETRY] Failed to marshal ETA_UPDATED for driver %s: %v", driverID, err)
		return
	}
	h.BroadcastToSupplier(supplierID, msg)
}

// BroadcastDriverAvailability pushes a DRIVER_AVAILABILITY_CHANGED event to the
// supplier's admin portal so the fleet page reflects online/offline state in real time.
func (h *Hub) BroadcastDriverAvailability(supplierID, driverID string, available bool, reason string) {
	payload := map[string]interface{}{
		"type":      wsEvents.EventDriverAvailabilityChanged,
		"driver_id": driverID,
		"available": available,
		"reason":    reason,
		"timestamp": time.Now().UnixMilli(),
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[TELEMETRY] Failed to marshal DRIVER_AVAILABILITY_CHANGED for driver %s: %v", driverID, err)
		return
	}
	h.BroadcastToSupplier(supplierID, msg)
}

// BroadcastOrderReassigned pushes an ORDER_REASSIGNED event to the supplier's admin portal.
func (h *Hub) BroadcastOrderReassigned(supplierID, orderID, oldDriverID, newDriverID string) {
	payload := map[string]interface{}{
		"type":          wsEvents.EventOrderReassigned,
		"order_id":      orderID,
		"old_driver_id": oldDriverID,
		"new_driver_id": newDriverID,
		"timestamp":     time.Now().UnixMilli(),
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[TELEMETRY] Failed to marshal ORDER_REASSIGNED for order %s: %v", orderID, err)
		return
	}
	h.BroadcastToSupplier(supplierID, msg)
}

// ── Delta-Sync Broadcast ────────────────────────────────────────────────────
// BroadcastDelta prepares and sends a compressed DeltaEvent to a specific topic
// (supplier channel). Fields are auto-compressed using the V.O.I.D. short-key
// dictionary for ~90% bandwidth reduction.
func (h *Hub) BroadcastDelta(topic string, event wsEvents.DeltaEvent) {
	if event.TS == 0 {
		event.TS = time.Now().Unix()
	}
	event.D = wsEvents.CompressDelta(event.D)
	msg, err := json.Marshal(event)
	if err != nil {
		log.Printf("[TELEMETRY] Failed to marshal DeltaEvent %s for %s: %v", event.T, event.I, err)
		return
	}
	h.BroadcastToSupplier(topic, msg)
}

// BroadcastOrderDelta is a convenience wrapper that emits an ORD_UP delta for
// an order state change. Only the changed fields are transmitted.
func (h *Hub) BroadcastOrderDelta(topic, orderID string, changedFields map[string]interface{}) {
	h.BroadcastDelta(topic, wsEvents.NewDelta(wsEvents.DeltaOrderUpdate, orderID, changedFields))
}

// BroadcastDriverDelta emits a DRV_UP delta for a driver field change.
func (h *Hub) BroadcastDriverDelta(topic, driverID string, changedFields map[string]interface{}) {
	h.BroadcastDelta(topic, wsEvents.NewDelta(wsEvents.DeltaDriverUpdate, driverID, changedFields))
}

// Close gracefully closes all connections in the telemetry Hub.
func (h *Hub) Close() {
	h.Lock()
	defer h.Unlock()
	for client := range h.Clients {
		client.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down"),
			time.Now().Add(10*time.Second))
		client.Close()
		delete(h.Clients, client)
	}
	log.Println("[TELEMETRY HUB] All connections closed.")
}
