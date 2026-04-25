// Package userroutes owns the cross-role /v1/user/* surface — endpoints
// shared by every authenticated principal (retailer, driver, supplier,
// payloader, admin) rather than scoped to one domain role.
package userroutes

import (
	"encoding/json"
	"log"
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/notifications"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to register /v1/user routes.
type Deps struct {
	Spanner        *spanner.Client
	DeviceTokenSvc *notifications.DeviceTokenService
	Log            Middleware
}

// RegisterRoutes mounts the shared user surface:
//
//	POST/DELETE /v1/user/device-token       — FCM/APNs token lifecycle
//	GET         /v1/user/notifications      — notification inbox
//	POST        /v1/user/notifications/read — mark notifications read
func RegisterRoutes(r chi.Router, d Deps) {
	s := d.Spanner
	log := d.Log
	allRoles := []string{"RETAILER", "DRIVER", "SUPPLIER", "PAYLOADER"}
	inboxRoles := []string{"RETAILER", "DRIVER", "SUPPLIER", "ADMIN", "PAYLOADER"}

	r.HandleFunc("/v1/user/device-token",
		auth.RequireRole(allRoles, log(handleDeviceToken(d.DeviceTokenSvc))))
	r.HandleFunc("/v1/user/notifications",
		auth.RequireRole(inboxRoles, log(notifications.HandleNotificationInbox(s))))
	r.HandleFunc("/v1/user/notifications/read",
		auth.RequireRole(inboxRoles, log(notifications.HandleMarkNotificationRead(s))))
}

// handleDeviceToken adapts the DeviceTokenService into an http.HandlerFunc.
// Behaviour preserved verbatim from the inline closure it replaced.
func handleDeviceToken(svc *notifications.DeviceTokenService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		switch r.Method {
		case http.MethodPost:
			var req notifications.RegisterTokenRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON body", http.StatusBadRequest)
				return
			}
			if err := svc.RegisterToken(r.Context(), claims.UserID, claims.Role, req); err != nil {
				http.Error(w, "Failed to register device token", http.StatusInternalServerError)
				log.Printf("[DEVICE_TOKEN] Registration failed for %s: %v", claims.UserID, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"registered"}`))

		case http.MethodDelete:
			platform := r.URL.Query().Get("platform")
			if err := svc.UnregisterToken(r.Context(), claims.UserID, platform); err != nil {
				http.Error(w, "Failed to unregister device token", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"unregistered"}`))

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}
