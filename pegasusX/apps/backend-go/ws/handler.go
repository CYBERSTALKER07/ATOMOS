package ws

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// RegisterRoutes mounts the WebSocket upgrade handler for the provided hubs.
// Note: Authentication is enforced upstream by standard middleware. The Upgrade
// handler extracts auth.Claims from context to determine identity and rooms.
func RegisterRoutes(r chi.Router, log *slog.Logger, firebaseAuthEnabled bool, verifier auth.FirebaseVerifier,
	retailerHub, supplierHub, driverHub, payloadHub, warehouseHub, factoryHub, telemetryHub *Hub) {

	wsHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ident, ok := auth.FromContext(req.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Error("websocket upgrade failed", "err", err)
			return
		}

		gConn := &gorillaConn{
			id:    uuid.New().String(),
			ident: ident,
			conn:  conn,
		}

		var unsubscribeFuncs []func()

		// Assign rooms based on role
		switch ident.Role {
		case auth.RoleRetailer:
			if retailerHub != nil && ident.Subject != "" {
				unsubscribeFuncs = append(unsubscribeFuncs, retailerHub.Subscribe("retailer:"+ident.Subject, gConn))
			}
		case auth.RoleAdmin:
			if supplierHub != nil && ident.SupplierID != "" {
				unsubscribeFuncs = append(unsubscribeFuncs, supplierHub.Subscribe("supplier:"+ident.SupplierID, gConn))
			}
			if warehouseHub != nil && ident.SupplierRole == auth.RoleWarehouseAdmin && ident.HomeNodeID != "" {
				unsubscribeFuncs = append(unsubscribeFuncs, warehouseHub.Subscribe("warehouse:"+ident.HomeNodeID, gConn))
			}
			if factoryHub != nil && ident.SupplierRole == auth.RoleFactoryAdmin && ident.HomeNodeID != "" {
				unsubscribeFuncs = append(unsubscribeFuncs, factoryHub.Subscribe("factory:"+ident.HomeNodeID, gConn))
			}
		case auth.RoleDriver:
			if driverHub != nil && ident.Subject != "" {
				unsubscribeFuncs = append(unsubscribeFuncs, driverHub.Subscribe("driver:"+ident.Subject, gConn))
			}
		case auth.RolePayload:
			if payloadHub != nil && ident.Subject != "" {
				unsubscribeFuncs = append(unsubscribeFuncs, payloadHub.Subscribe("payload:"+ident.Subject, gConn))
			}
		case auth.RoleWarehouseAdmin:
			if warehouseHub != nil && ident.HomeNodeID != "" {
				unsubscribeFuncs = append(unsubscribeFuncs, warehouseHub.Subscribe("warehouse:"+ident.HomeNodeID, gConn))
			}
			if supplierHub != nil && ident.SupplierID != "" {
				unsubscribeFuncs = append(unsubscribeFuncs, supplierHub.Subscribe("supplier:"+ident.SupplierID, gConn))
			}
		case auth.RoleFactoryAdmin:
			if factoryHub != nil && ident.HomeNodeID != "" {
				unsubscribeFuncs = append(unsubscribeFuncs, factoryHub.Subscribe("factory:"+ident.HomeNodeID, gConn))
			}
			if supplierHub != nil && ident.SupplierID != "" {
				unsubscribeFuncs = append(unsubscribeFuncs, supplierHub.Subscribe("supplier:"+ident.SupplierID, gConn))
			}
		default:
			log.Warn("ws upgrade: unrecognized role", "role", ident.Role)
			conn.Close()
			return
		}

		// Keep connection alive, listen for close/pings
		go func() {
			defer func() {
				for _, unsub := range unsubscribeFuncs {
					unsub()
				}
				conn.Close()
			}()
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}
		}()
	})

	if firebaseAuthEnabled && verifier != nil {
		r.With(auth.FirebaseAuth(verifier)).Get("/v1/ws", wsHandler)
	} else {
		r.Get("/v1/ws", wsHandler)
	}
}
