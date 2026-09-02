package ws

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
)

const (
	websocketPongWait     = 30 * time.Second
	websocketPingInterval = 15 * time.Second
	websocketWriteWait    = 5 * time.Second
)

// RegisterConfig carries optional WebSocket subscription hooks.
type RegisterConfig struct {
	// RetailerPromoSuppliers returns supplier IDs whose supplier-promo rooms a
	// retailer connection should join at upgrade (typically cart suppliers).
	RetailerPromoSuppliers func(ctx context.Context, retailerID string) []string
}

// RegisterRoutes mounts the WebSocket upgrade handler for the provided hubs.
// Note: Authentication is enforced upstream by standard middleware. The Upgrade
// handler extracts auth.Claims from context to determine identity and rooms.
func RegisterRoutes(r chi.Router, log *slog.Logger, jwtSecret string,
	platformSvc *platform.Service,
	retailerHub, supplierHub, driverHub, payloadHub, warehouseHub, factoryHub, telemetryHub, platformAdminHub *Hub,
	cfg RegisterConfig,
) {
	if log == nil {
		log = slog.Default()
	}
	hubs := roleHubs{retailerHub, supplierHub, driverHub, payloadHub, warehouseHub, factoryHub, telemetryHub, platformAdminHub}

	wsHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ident, ok := claimsFromRequest(req, jwtSecret)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if !roleHubsHaveCapacity(ident, hubs) {
			log.Warn("ws upgrade rejected: hub at capacity", "role", ident.Role)
			w.Header().Set("Retry-After", strconv.Itoa(CapacityRetryAfterSeconds()))
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Error("websocket upgrade failed", "err", err)
			return
		}

		gConn := &gorillaConn{
			id:       uuid.New().String(),
			ident:    ident,
			conn:     conn,
			joinedAt: time.Now(),
			send:     make(chan []byte, 256),
			stop:     make(chan struct{}),
		}
		go gConn.writePump()

		unsubscribeFuncs, ok := subscribeIdentityRooms(ident, gConn, hubs, cfg)
		if !ok {
			log.Warn("ws upgrade: unrecognized role", "role", ident.Role)
			gConn.close()
			return
		}

		go runConnectionLoop(req, gConn, unsubscribeFuncs, platformSvc, hubs, log)
	})

	r.Get("/v1/ws", wsHandler)
	r.Get("/v1/events", HandleSSEEvents(log, jwtSecret, platformSvc, hubs, cfg))
}

func claimsFromRequest(req *http.Request, jwtSecret string) (auth.Claims, bool) {
	if ident, ok := auth.FromContext(req.Context()); ok {
		return ident, true
	}
	if strings.TrimSpace(jwtSecret) == "" {
		return auth.Claims{}, false
	}
	// Native clients (Android/iOS) send Authorization: Bearer with a session JWT.
	if ident, ok := auth.ParseBearerClaims(req, jwtSecret); ok {
		if auth.IsPendingOrgSelect(ident) {
			return auth.Claims{}, false
		}
		return ident, true
	}
	// Browsers cannot set Authorization on WebSocket(); short-lived tickets only.
	token := strings.TrimSpace(req.URL.Query().Get("token"))
	if token == "" {
		return auth.Claims{}, false
	}
	ident, err := auth.Parse(token, jwtSecret)
	if err != nil {
		return auth.Claims{}, false
	}
	if !auth.IsWSTicket(ident) {
		return auth.Claims{}, false
	}
	jti := strings.TrimSpace(ident.JTI)
	if jti != "" {
		revoked, err := auth.GetRevocationStore().IsRevoked(req.Context(), jti)
		if err != nil || revoked {
			return auth.Claims{}, false
		}
	}
	return ident, true
}

type roleHubs struct {
	retailer      *Hub
	supplier      *Hub
	driver        *Hub
	payload       *Hub
	warehouse     *Hub
	factory       *Hub
	telemetry     *Hub
	platformAdmin *Hub
}

func subscribeIdentityRooms(ident auth.Claims, conn Connection, hubs roleHubs, cfg RegisterConfig) ([]func(), bool) {
	var unsubscribes []func()
	switch ident.Role {
	case auth.RoleRetailer:
		return subscribeRetailerRooms(ident, conn, hubs, unsubscribes, cfg), true
	case auth.RoleAdmin:
		return subscribeSupplierRooms(ident, conn, hubs, unsubscribes), true
	case auth.RoleDriver:
		return subscribeDriverRooms(ident, conn, hubs, unsubscribes), true
	case auth.RolePayload:
		return subscribePayloadRooms(ident, conn, hubs, unsubscribes), true
	case auth.RoleWarehouseAdmin, auth.RoleWarehouse:
		return subscribeWarehouseAdminRooms(ident, conn, hubs, unsubscribes), true
	case auth.RoleFactoryAdmin, auth.RoleFactory:
		return subscribeFactoryAdminRooms(ident, conn, hubs, unsubscribes), true
	case auth.RolePlatformAdmin:
		return subscribePlatformAdminRooms(ident, conn, hubs, unsubscribes), true
	default:
		return nil, false
	}
}

func subscribePlatformAdminRooms(_ auth.Claims, conn Connection, hubs roleHubs, unsubscribes []func()) []func() {
	if hubs.platformAdmin != nil {
		unsubscribes = append(unsubscribes, hubs.platformAdmin.Subscribe(PlatformAdminRoom(), conn))
	}
	return unsubscribes
}

func subscribeRetailerRooms(ident auth.Claims, conn Connection, hubs roleHubs, unsubscribes []func(), cfg RegisterConfig) []func() {
	if hubs.retailer != nil {
		orgID := auth.ResolveRetailerOrgID(ident)
		if orgID != "" {
			unsubscribes = append(unsubscribes, hubs.retailer.Subscribe("retailer:"+orgID, conn))
			if cfg.RetailerPromoSuppliers != nil {
				for _, supplierID := range cfg.RetailerPromoSuppliers(context.Background(), orgID) {
					room := SupplierPromoRoom(supplierID)
					unsubscribes = append(unsubscribes, hubs.retailer.Subscribe(room, conn))
				}
			}
		}
	}
	return unsubscribes
}

func subscribeSupplierRooms(ident auth.Claims, conn Connection, hubs roleHubs, unsubscribes []func()) []func() {
	if hubs.supplier != nil && ident.SupplierID != "" {
		unsubscribes = append(unsubscribes, hubs.supplier.Subscribe("supplier:"+ident.SupplierID, conn))
	}
	if hubs.warehouse != nil && ident.SupplierRole == auth.RoleWarehouseAdmin && ident.HomeNodeID != "" {
		unsubscribes = append(unsubscribes, hubs.warehouse.Subscribe("warehouse:"+ident.HomeNodeID, conn))
	}
	if hubs.factory != nil && ident.SupplierRole == auth.RoleFactoryAdmin && ident.HomeNodeID != "" {
		unsubscribes = append(unsubscribes, hubs.factory.Subscribe("factory:"+ident.HomeNodeID, conn))
	}
	return subscribeSupplierTelemetry(ident, conn, hubs, unsubscribes)
}

func subscribeDriverRooms(ident auth.Claims, conn Connection, hubs roleHubs, unsubscribes []func()) []func() {
	if hubs.driver != nil && ident.Subject != "" {
		unsubscribes = append(unsubscribes, hubs.driver.Subscribe("driver:"+ident.Subject, conn))
	}
	if hubs.telemetry != nil && ident.Subject != "" {
		unsubscribes = append(unsubscribes, hubs.telemetry.Subscribe("telemetry:driver:"+ident.Subject, conn))
	}
	return unsubscribes
}

func subscribePayloadRooms(ident auth.Claims, conn Connection, hubs roleHubs, unsubscribes []func()) []func() {
	if hubs.payload == nil {
		return unsubscribes
	}
	seen := make(map[string]struct{}, 2)
	for _, roomID := range []string{ident.Subject, ident.SupplierID} {
		roomID = strings.TrimSpace(roomID)
		if roomID == "" {
			continue
		}
		if _, ok := seen[roomID]; ok {
			continue
		}
		seen[roomID] = struct{}{}
		unsubscribes = append(unsubscribes, hubs.payload.Subscribe("payload:"+roomID, conn))
	}
	return unsubscribes
}

func subscribeWarehouseAdminRooms(ident auth.Claims, conn Connection, hubs roleHubs, unsubscribes []func()) []func() {
	if hubs.warehouse != nil && ident.HomeNodeID != "" {
		unsubscribes = append(unsubscribes, hubs.warehouse.Subscribe("warehouse:"+ident.HomeNodeID, conn))
	}
	if hubs.supplier != nil && ident.SupplierID != "" {
		unsubscribes = append(unsubscribes, hubs.supplier.Subscribe("supplier:"+ident.SupplierID, conn))
	}
	return subscribeSupplierTelemetry(ident, conn, hubs, unsubscribes)
}

func subscribeFactoryAdminRooms(ident auth.Claims, conn Connection, hubs roleHubs, unsubscribes []func()) []func() {
	if hubs.factory != nil {
		seen := make(map[string]struct{}, 2)
		for _, roomID := range []string{ident.HomeNodeID, ident.SupplierID} {
			roomID = strings.TrimSpace(roomID)
			if roomID == "" {
				continue
			}
			if _, ok := seen[roomID]; ok {
				continue
			}
			seen[roomID] = struct{}{}
			unsubscribes = append(unsubscribes, hubs.factory.Subscribe("factory:"+roomID, conn))
		}
	}
	if hubs.supplier != nil && ident.SupplierID != "" {
		unsubscribes = append(unsubscribes, hubs.supplier.Subscribe("supplier:"+ident.SupplierID, conn))
	}
	return subscribeSupplierTelemetry(ident, conn, hubs, unsubscribes)
}

func subscribeSupplierTelemetry(ident auth.Claims, conn Connection, hubs roleHubs, unsubscribes []func()) []func() {
	if hubs.telemetry != nil && ident.SupplierID != "" {
		unsubscribes = append(unsubscribes, hubs.telemetry.Subscribe("telemetry:supplier:"+ident.SupplierID, conn))
	}
	return unsubscribes
}

func roleHubsHaveCapacity(ident auth.Claims, hubs roleHubs) bool {
	for _, hub := range roleHubsForIdentity(ident, hubs) {
		if hub != nil && !hub.HasCapacity() {
			return false
		}
	}
	return true
}

func roleHubsForIdentity(ident auth.Claims, hubs roleHubs) []*Hub {
	switch ident.Role {
	case auth.RoleRetailer:
		return []*Hub{hubs.retailer}
	case auth.RoleAdmin:
		return []*Hub{hubs.supplier, hubs.warehouse, hubs.factory, hubs.telemetry}
	case auth.RoleDriver:
		return []*Hub{hubs.driver, hubs.telemetry}
	case auth.RolePayload:
		return []*Hub{hubs.payload}
	case auth.RoleWarehouseAdmin, auth.RoleWarehouse:
		return []*Hub{hubs.warehouse, hubs.supplier, hubs.telemetry}
	case auth.RoleFactoryAdmin, auth.RoleFactory:
		return []*Hub{hubs.factory, hubs.supplier, hubs.telemetry}
	case auth.RolePlatformAdmin:
		return []*Hub{hubs.platformAdmin}
	default:
		return nil
	}
}

func runConnectionLoop(req *http.Request, conn *gorillaConn, unsubscribes []func(), platformSvc *platform.Service, hubs roleHubs, log *slog.Logger) {
	maybeSendOutdated(req, conn, platformSvc, log)
	done := make(chan struct{})

	// Touch presence on initial connect (use Background — req.Context() is
	// cancelled by the HTTP server after the WebSocket upgrade completes).
	for _, h := range roleHubsForIdentity(conn.Identity(), hubs) {
		if h != nil {
			h.TouchPresence(context.Background(), conn)
		}
	}

	// startPingLoop(conn, done, log) // handled by writePump now
	defer func() {
		close(done)
		for _, unsubscribe := range unsubscribes {
			unsubscribe()
		}
		// Clear presence on disconnect (req.Context() is long dead here).
		for _, h := range roleHubsForIdentity(conn.Identity(), hubs) {
			if h != nil {
				h.ClearPresence(context.Background(), conn)
			}
		}
		conn.close()
	}()
	conn.conn.SetReadLimit(1024)
	_ = conn.conn.SetReadDeadline(time.Now().Add(websocketPongWait))
	conn.conn.SetPongHandler(func(string) error {
		// Touch presence on pong
		for _, h := range roleHubsForIdentity(conn.Identity(), hubs) {
			if h != nil {
				h.TouchPresence(context.Background(), conn)
			}
		}
		return conn.conn.SetReadDeadline(time.Now().Add(websocketPongWait))
	})

	limiter := NewIngressRateLimiter()
	for {
		if _, _, err := conn.conn.ReadMessage(); err != nil {
			return
		}
		if !limiter.Allow() {
			log.Warn("websocket ingress rate limit exceeded", "conn_id", conn.ID())
			return
		}
	}
}

func maybeSendOutdated(req *http.Request, conn *gorillaConn, platformSvc *platform.Service, log *slog.Logger) {
	if platformSvc == nil {
		return
	}
	version := strings.TrimSpace(req.URL.Query().Get("version"))
	if version == "" {
		version = strings.TrimSpace(req.Header.Get("X-App-Version"))
	}
	platformName := strings.TrimSpace(req.URL.Query().Get("platform"))
	channel := strings.TrimSpace(req.URL.Query().Get("channel"))
	ident := conn.ident
	traceID := outbox.TraceIDFromContext(req.Context())
	payload, send, err := platformSvc.OutdatedWSPayload(
		req.Context(),
		platform.ClaimsRoleForPolicy(ident),
		platformName,
		channel,
		version,
		platform.ClaimsActorID(ident),
		traceID,
	)
	if err != nil {
		log.WarnContext(req.Context(), "outdated ws check failed", "err", err)
		return
	}
	if !send {
		return
	}
	if err := conn.Send(context.Background(), payload); err != nil {
		log.WarnContext(req.Context(), "outdated ws send failed", "err", err)
	}
}

func startPingLoop(conn *gorillaConn, done <-chan struct{}, log *slog.Logger) {
	ticker := time.NewTicker(websocketPingInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := conn.ping(websocketWriteWait); err != nil {
					log.Debug("websocket ping failed", "conn_id", conn.ID(), "err", err)
					conn.close()
					return
				}
			}
		}
	}()
}
