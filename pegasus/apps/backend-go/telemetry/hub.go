package telemetry

import (
	"context"
	"encoding/json"
	"log"
	"math"
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
	writeMu         sync.Mutex
	subscribed      map[string]bool // supplierID → relay subscription active
	// driverSupplierCache maps driverID → supplierID to avoid repeated Spanner reads
	driverSupplierCache sync.Map
}

var FleetHub = &Hub{
	Clients:    make(map[*websocket.Conn]*clientMeta),
	subscribed: make(map[string]bool),
}

// Grace-mode telemetry connections are intentionally short-lived to force
// token refresh + reconnect while still giving clients a small handoff window.
var graceReconnectCloseAfter = 30 * time.Second

func graceReconnectDelay(claims *auth.LabClaims) time.Duration {
	closeAfter := graceReconnectCloseAfter
	if claims != nil && !claims.GraceDeadline.IsZero() {
		remaining := time.Until(claims.GraceDeadline)
		if remaining < closeAfter {
			closeAfter = remaining
		}
	}
	if closeAfter <= 0 {
		return 1 * time.Second
	}
	return closeAfter
}

func (h *Hub) startGraceReconnectEnforcer(ws *websocket.Conn, claims *auth.LabClaims, done <-chan struct{}) {
	closeAfter := graceReconnectDelay(claims)

	refreshMsg, _ := json.Marshal(map[string]interface{}{
		"type":             wsEvents.EventTokenRefreshNeeded,
		"message":          "Authentication token is in grace mode. Refresh token and reconnect telemetry.",
		"required_action":  "REFRESH_TOKEN_AND_RECONNECT",
		"close_in_seconds": int(math.Ceil(closeAfter.Seconds())),
	})
	h.writeMu.Lock()
	_ = ws.WriteMessage(websocket.TextMessage, refreshMsg)
	h.writeMu.Unlock()

	go func() {
		timer := time.NewTimer(closeAfter)
		defer timer.Stop()
		select {
		case <-done:
			return
		case <-timer.C:
			h.writeMu.Lock()
			_ = ws.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "token refresh required"),
				time.Now().Add(wsEvents.WriteWait),
			)
			h.writeMu.Unlock()
			_ = ws.Close()
		}
	}()
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

func normalizeTelemetryRole(role string) (string, bool) {
	switch role {
	case "ADMIN", "SUPPLIER":
		return "ADMIN", true
	case "DRIVER":
		return "DRIVER", true
	default:
		return "", false
	}
}

func (h *Hub) removeClient(conn *websocket.Conn, meta *clientMeta) {
	shouldUnsubscribe := false
	channel := ""

	h.Lock()
	delete(h.Clients, conn)
	if meta != nil && meta.Role == "ADMIN" && meta.SupplierID != "" {
		if h.subscribed[meta.SupplierID] && !h.hasSupplierAdminLocked(meta.SupplierID) {
			delete(h.subscribed, meta.SupplierID)
			shouldUnsubscribe = true
			channel = "ws:supplier:" + meta.SupplierID
		}
	}
	h.Unlock()

	if shouldUnsubscribe {
		cache.Unsubscribe(channel)
	}
}

func (h *Hub) hasSupplierAdminLocked(supplierID string) bool {
	for _, meta := range h.Clients {
		if meta.Role == "ADMIN" && meta.SupplierID == supplierID {
			return true
		}
	}
	return false
}

// HandleConnection upgrades the HTTP request to a persistent WebSocket
func (h *Hub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized telemetry access", http.StatusUnauthorized)
		return
	}

	role, allowed := normalizeTelemetryRole(claims.Role)
	if !allowed {
		http.Error(w, "Unauthorized telemetry access", http.StatusUnauthorized)
		return
	}

	meta := &clientMeta{Role: role}
	if role == "ADMIN" {
		meta.SupplierID = claims.ResolveSupplierID()
		if meta.SupplierID == "" {
			http.Error(w, "Unauthorized telemetry access", http.StatusUnauthorized)
			return
		}
	} else {
		if claims.UserID == "" {
			http.Error(w, "Unauthorized telemetry access", http.StatusUnauthorized)
			return
		}
		meta.DriverID = claims.UserID
		meta.SupplierID = h.resolveDriverSupplier(r.Context(), claims.UserID)
		if meta.SupplierID == "" {
			http.Error(w, "Driver supplier scope unavailable", http.StatusUnauthorized)
			return
		}
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[TELEMETRY_FAULT] Protocol upgrade failed: %v", err)
		return
	}

	h.Lock()
	h.Clients[ws] = meta
	h.Unlock()

	if meta.Role == "ADMIN" && meta.SupplierID != "" {
		h.SubscribeSupplierRelay(meta.SupplierID)
	}

	done := wsEvents.ConfigureKeepalive(ws)
	defer func() {
		close(done)
		h.removeClient(ws, meta)
		ws.Close()
		log.Printf("[TELEMETRY] %s (supplier=%s) disconnected.", meta.Role, meta.SupplierID)
	}()

	log.Printf("[TELEMETRY] %s (supplier=%s) linked to the matrix.", meta.Role, meta.SupplierID)

	// Grace-mode DRIVER connections must refresh + reconnect quickly.
	if claims.GracePeriod {
		h.startGraceReconnectEnforcer(ws, claims, done)
	}

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			break
		}

		if meta.Role == "DRIVER" {
			var payload GPSPayload
			if err := json.Unmarshal(msg, &payload); err != nil {
				continue
			}
			if payload.DriverID == "" {
				payload.DriverID = meta.DriverID
			}
			if payload.DriverID != meta.DriverID {
				log.Printf("[TELEMETRY] Driver spoof attempt blocked: token_driver=%s payload_driver=%s", meta.DriverID, payload.DriverID)
				continue
			}
			if payload.Timestamp == 0 {
				payload.Timestamp = time.Now().Unix()
			}

			if h.Buffer != nil {
				h.Buffer.Ingest(GPSEntry{
					DriverID:   meta.DriverID,
					Latitude:   payload.Latitude,
					Longitude:  payload.Longitude,
					Timestamp:  payload.Timestamp,
					SupplierID: meta.SupplierID,
				})
			} else {
				h.BroadcastToSupplier(meta.SupplierID, msg)
			}
			if h.ProximityEngine != nil {
				go h.ProximityEngine.ProcessPing(context.Background(), meta.DriverID, payload.Latitude, payload.Longitude)
			}
		}
	}
}

// BroadcastToSupplier fans out the GPS payload to ADMIN connections belonging
// to the same supplier on THIS pod, AND publishes to Redis Pub/Sub so other
// pods can relay to their local connections.
func (h *Hub) BroadcastToSupplier(supplierID string, message []byte) {
	h.broadcastToSupplierLocal(supplierID, message)
	h.publishSupplierRelay(supplierID, message)
}

func (h *Hub) publishSupplierRelay(supplierID string, message []byte) {
	if supplierID == "" {
		return
	}
	rc := cache.GetClient()
	if rc == nil {
		return
	}
	channel := "ws:supplier:" + supplierID
	if err := rc.Publish(context.Background(), channel, message).Err(); err != nil {
		WSPubSubFailures.WithLabelValues("fleet").Inc()
		log.Printf("[TELEMETRY] relay publish failed for supplier %s: %v", supplierID, err)
	}
}

// broadcastToSupplierLocal sends to local connections only (called by both
// the direct path and the Redis Pub/Sub relay handler).
func (h *Hub) broadcastToSupplierLocal(supplierID string, message []byte) {
	if supplierID == "" {
		return
	}

	h.RLock()
	targets := make([]*websocket.Conn, 0, len(h.Clients))
	for client, meta := range h.Clients {
		if meta.Role == "ADMIN" && meta.SupplierID == supplierID {
			targets = append(targets, client)
		}
	}
	h.RUnlock()
	if len(targets) == 0 {
		return
	}

	h.writeMu.Lock()
	dead := make([]*websocket.Conn, 0)
	for _, client := range targets {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			client.Close()
			dead = append(dead, client)
		}
	}
	h.writeMu.Unlock()

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
	if supplierID == "" {
		return
	}

	h.Lock()
	if h.subscribed == nil {
		h.subscribed = make(map[string]bool)
	}
	if h.subscribed[supplierID] {
		h.Unlock()
		return
	}
	h.subscribed[supplierID] = true
	h.Unlock()

	channel := "ws:supplier:" + supplierID
	cache.Subscribe(channel, func(_ string, payload []byte) {
		// Relay to local connections only (no re-publish to avoid loops)
		h.broadcastToSupplierLocal(supplierID, payload)
	})
}

// BroadcastToAdmins fans out GPS to all connected control towers (kept for backward compat / internal use)
func (h *Hub) BroadcastToAdmins(message []byte) {
	h.RLock()
	targets := make([]*websocket.Conn, 0, len(h.Clients))
	for client, meta := range h.Clients {
		if meta.Role == "ADMIN" {
			targets = append(targets, client)
		}
	}
	h.RUnlock()
	if len(targets) == 0 {
		return
	}

	h.writeMu.Lock()
	dead := make([]*websocket.Conn, 0)
	for _, client := range targets {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			client.Close()
			dead = append(dead, client)
		}
	}
	h.writeMu.Unlock()

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
	clients := make([]*websocket.Conn, 0, len(h.Clients))
	for client := range h.Clients {
		clients = append(clients, client)
	}
	suppliers := make([]string, 0, len(h.subscribed))
	for supplierID := range h.subscribed {
		suppliers = append(suppliers, supplierID)
	}
	h.Clients = make(map[*websocket.Conn]*clientMeta)
	h.subscribed = make(map[string]bool)
	h.Unlock()

	h.writeMu.Lock()
	for _, client := range clients {
		client.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutting down"),
			time.Now().Add(10*time.Second))
		client.Close()
	}
	h.writeMu.Unlock()

	for _, supplierID := range suppliers {
		cache.Unsubscribe("ws:supplier:" + supplierID)
	}
	log.Println("[TELEMETRY HUB] All connections closed.")
}
