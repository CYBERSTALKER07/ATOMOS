package pulse

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
)

// Handlers serves role-scoped pulse endpoints.
type Handlers struct {
	Service *Service
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

// HandleRetailerPulse serves GET /v1/retailer/pulse.
func (h *Handlers) HandleRetailerPulse(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	recipient := notifications.RecipientIDFromClaims(claims)
	resp, err := h.Service.ListForRecipient(r.Context(), recipient, string(claims.Role), claims.Subject, 40)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pulse_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleSupplierPulse serves GET /v1/supplier/pulse.
func (h *Handlers) HandleSupplierPulse(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	sid := strings.TrimSpace(claims.SupplierID)
	recipient := notifications.RecipientIDFromClaims(claims)
	resp, err := h.Service.ListForRecipient(r.Context(), recipient, string(claims.Role), sid, 40)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pulse_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleWarehousePulse serves GET /v1/warehouse/ops/pulse.
func (h *Handlers) HandleWarehousePulse(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleWarehouse && claims.Role != auth.RoleWarehouseAdmin) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	wh := strings.TrimSpace(claims.HomeNodeID)
	recipient := notifications.RecipientIDFromClaims(claims)
	resp, err := h.Service.ListForRecipient(r.Context(), recipient, string(claims.Role), wh, 40)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pulse_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleDriverPulse serves GET /v1/driver/pulse.
func (h *Handlers) HandleDriverPulse(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	recipient := notifications.RecipientIDFromClaims(claims)
	resp, err := h.Service.ListForRecipient(r.Context(), recipient, string(claims.Role), claims.Subject, 40)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pulse_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandlePayloaderPulse serves GET /v1/payloader/pulse.
func (h *Handlers) HandlePayloaderPulse(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RolePayload {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	sid := strings.TrimSpace(claims.SupplierID)
	recipient := notifications.RecipientIDFromClaims(claims)
	resp, err := h.Service.ListForRecipient(r.Context(), recipient, string(claims.Role), sid, 40)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pulse_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleFactoryPulse serves GET /v1/factory/pulse.
func (h *Handlers) HandleFactoryPulse(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleFactoryAdmin {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	scope := strings.TrimSpace(claims.HomeNodeID)
	recipient := notifications.RecipientIDFromClaims(claims)
	resp, err := h.Service.ListForRecipient(r.Context(), recipient, string(claims.Role), scope, 40)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pulse_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
