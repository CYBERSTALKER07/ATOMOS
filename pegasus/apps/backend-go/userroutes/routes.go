// Package userroutes owns the cross-role /v1/user/* surface — endpoints
// shared by every authenticated principal (retailer, driver, supplier,
// payloader, admin) rather than scoped to one domain role.
package userroutes

import (
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/notifications"
	"backend-go/ws"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to register /v1/user routes.
type Deps struct {
	Spanner         *spanner.Client
	DeviceTokenSvc  *notifications.DeviceTokenService
	DriverHub       *ws.DriverHub
	RetailerHub     *ws.RetailerHub
	PayloaderHub    *ws.PayloaderHub
	SupplierHub     *ws.SupplierHub
	CommandRegistry *ws.CommandRegistry
	Log             Middleware
}

// RegisterRoutes mounts the shared user surface:
//
//	POST/DELETE /v1/user/device-token       — FCM/APNs token lifecycle
//	GET         /v1/user/notifications      — notification inbox
//	POST        /v1/user/notifications/read — mark notifications read
//	POST        /v1/ws/command/dispatch     — desktop/native command initiation
//	POST        /v1/ws/ack                  — native command acknowledgment
func RegisterRoutes(r chi.Router, d Deps) {
	s := d.Spanner
	log := d.Log
	allRoles := []string{"RETAILER", "DRIVER", "SUPPLIER", "PAYLOADER"}
	inboxRoles := []string{"RETAILER", "DRIVER", "SUPPLIER", "ADMIN", "PAYLOADER"}
	ackRoles := []string{"RETAILER", "DRIVER", "SUPPLIER", "PAYLOADER", "ADMIN", "FACTORY", "WAREHOUSE"}
	dispatchRoles := []string{"SUPPLIER", "ADMIN", "FACTORY", "WAREHOUSE"}

	r.HandleFunc("/v1/user/device-token",
		auth.RequireRole(allRoles, log(handleDeviceToken(d.DeviceTokenSvc))))
	r.HandleFunc("/v1/user/notifications",
		auth.RequireRole(inboxRoles, log(notifications.HandleNotificationInbox(s))))
	r.HandleFunc("/v1/user/notifications/read",
		auth.RequireRole(inboxRoles, log(notifications.HandleMarkNotificationRead(s))))
	r.HandleFunc("/v1/ws/command/dispatch",
		auth.RequireRole(dispatchRoles, log(handleDispatchCommand(d))))
	r.HandleFunc("/v1/ws/ack",
		auth.RequireRole(ackRoles, log(handleWSAck(d))))
}

// handleDeviceToken adapts the DeviceTokenService into an http.HandlerFunc.
// Behaviour preserved verbatim from the inline closure it replaced.
func handleDeviceToken(svc *notifications.DeviceTokenService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
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

type dispatchCommandRequest struct {
	TargetRole string                 `json:"target_role"`
	TargetID   string                 `json:"target_id"`
	Type       string                 `json:"type"`
	TraceID    string                 `json:"trace_id,omitempty"`
	SupplierID string                 `json:"supplier_id,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
}

type wsAckRequest struct {
	CommandID string `json:"command_id"`
	TraceID   string `json:"trace_id,omitempty"`
}

func handleDispatchCommand(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if d.CommandRegistry == nil {
			http.Error(w, "Command registry unavailable", http.StatusServiceUnavailable)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req dispatchCommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		req.TargetRole = strings.ToUpper(strings.TrimSpace(req.TargetRole))
		req.TargetID = strings.TrimSpace(req.TargetID)
		req.Type = strings.TrimSpace(req.Type)
		if req.TargetRole == "" || req.TargetID == "" || req.Type == "" {
			http.Error(w, "target_role, target_id, and type are required", http.StatusBadRequest)
			return
		}

		payload := cloneMap(req.Payload)
		if payload == nil {
			payload = map[string]interface{}{}
		}
		payload["type"] = req.Type
		supplierID := strings.TrimSpace(req.SupplierID)
		if supplierID == "" {
			supplierID = claims.ResolveSupplierID()
		}
		if supplierID != "" {
			payload["supplier_id"] = supplierID
		}

		cmd, err := d.CommandRegistry.RegisterDispatch(r.Context(), req.TargetRole, req.TargetID, supplierID, req.Type, strings.TrimSpace(req.TraceID), payload)
		if err != nil {
			slog.ErrorContext(r.Context(), "ws command dispatch failed",
				"target_role", req.TargetRole,
				"target_id", req.TargetID,
				"event_type", req.Type,
				"trace_id", req.TraceID,
				"error", err,
			)
			http.Error(w, "Failed to register command", http.StatusInternalServerError)
			return
		}
		cmd, err = d.CommandRegistry.MarkDispatched(r.Context(), cmd.CommandID, cmd.TraceID)
		if err != nil {
			http.Error(w, "Failed to mark command dispatched", http.StatusInternalServerError)
			return
		}

		payload["command_id"] = cmd.CommandID
		payload["command_state"] = cmd.State
		payload["initiated_at"] = cmd.InitiatedAt.Format(time.RFC3339Nano)
		payload["dispatched_at"] = cmd.DispatchedAt.Format(time.RFC3339Nano)
		payload["trace_id"] = cmd.TraceID

		var dispatched bool
		switch req.TargetRole {
		case "DRIVER":
			dispatched = d.DriverHub != nil && d.DriverHub.PushToDriver(req.TargetID, payload)
		case "RETAILER":
			dispatched = d.RetailerHub != nil && d.RetailerHub.PushToRetailer(req.TargetID, payload)
		case "PAYLOADER":
			dispatched = d.PayloaderHub != nil && d.PayloaderHub.PushToPayloader(req.TargetID, payload)
		default:
			http.Error(w, "Unsupported target_role", http.StatusBadRequest)
			return
		}

		if !dispatched {
			slog.WarnContext(r.Context(), "ws command dispatched without active local recipients",
				"command_id", cmd.CommandID,
				"target_role", req.TargetRole,
				"target_id", req.TargetID,
				"trace_id", cmd.TraceID,
			)
		}

		if d.SupplierHub != nil && supplierID != "" {
			d.SupplierHub.PushToSupplier(supplierID, map[string]interface{}{
				"type":          ws.EventCommandDispatched,
				"command_id":    cmd.CommandID,
				"command_state": cmd.State,
				"event_type":    cmd.EventType,
				"target_role":   cmd.TargetRole,
				"target_id":     cmd.TargetID,
				"trace_id":      cmd.TraceID,
				"timestamp":     cmd.UpdatedAt.Format(time.RFC3339Nano),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":        "dispatched",
			"command_id":    cmd.CommandID,
			"command_state": cmd.State,
			"trace_id":      cmd.TraceID,
		})
	}
}

func handleWSAck(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if d.CommandRegistry == nil {
			http.Error(w, "Command registry unavailable", http.StatusServiceUnavailable)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req wsAckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		req.CommandID = strings.TrimSpace(req.CommandID)
		req.TraceID = strings.TrimSpace(req.TraceID)
		if req.CommandID == "" {
			http.Error(w, "command_id is required", http.StatusBadRequest)
			return
		}

		received, err := d.CommandRegistry.MarkReceived(r.Context(), req.CommandID, claims.UserID, claims.Role, req.TraceID)
		if err != nil {
			if errors.Is(err, ws.ErrCommandNotFound) {
				http.Error(w, "command not found", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to mark command received", http.StatusInternalServerError)
			return
		}

		settled, err := d.CommandRegistry.MarkSettled(r.Context(), req.CommandID, claims.UserID, claims.Role, req.TraceID)
		if err != nil {
			http.Error(w, "failed to settle command", http.StatusInternalServerError)
			return
		}

		if d.SupplierHub != nil && settled.SupplierID != "" {
			d.SupplierHub.PushToSupplier(settled.SupplierID, map[string]interface{}{
				"type":           ws.EventCommandReceived,
				"command_id":     received.CommandID,
				"command_state":  received.State,
				"event_type":     received.EventType,
				"target_role":    received.TargetRole,
				"target_id":      received.TargetID,
				"ack_by_user_id": received.AckByUserID,
				"ack_by_role":    received.AckByRole,
				"trace_id":       received.TraceID,
				"timestamp":      received.UpdatedAt.Format(time.RFC3339Nano),
			})
			d.SupplierHub.PushToSupplier(settled.SupplierID, map[string]interface{}{
				"type":           ws.EventCommandSettled,
				"command_id":     settled.CommandID,
				"command_state":  settled.State,
				"event_type":     settled.EventType,
				"target_role":    settled.TargetRole,
				"target_id":      settled.TargetID,
				"ack_by_user_id": settled.AckByUserID,
				"ack_by_role":    settled.AckByRole,
				"trace_id":       settled.TraceID,
				"timestamp":      settled.UpdatedAt.Format(time.RFC3339Nano),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "settled",
			"command_id":     settled.CommandID,
			"command_state":  settled.State,
			"received_state": received.State,
			"trace_id":       settled.TraceID,
		})
	}
}

func cloneMap(in map[string]interface{}) map[string]interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
