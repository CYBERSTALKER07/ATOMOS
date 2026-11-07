package platformadmin

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandleListFeatureFlags GET /v1/platform-admin/tenants/{tenantType}/{tenantID}/flags
func (h *Handlers) HandleListFeatureFlags(w http.ResponseWriter, r *http.Request) {
	tenantType := chi.URLParam(r, "tenantType")
	tenantID := chi.URLParam(r, "tenantID")
	flags, err := h.Svc.ListFeatureFlags(r.Context(), tenantType, tenantID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": flags})
}

// HandleSetFeatureFlag PUT /v1/platform-admin/tenants/{tenantType}/{tenantID}/flags/{flagKey}
func (h *Handlers) HandleSetFeatureFlag(w http.ResponseWriter, r *http.Request) {
	tenantType := chi.URLParam(r, "tenantType")
	tenantID := chi.URLParam(r, "tenantID")
	flagKey := chi.URLParam(r, "flagKey")
	
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SetFeatureFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	
	if err := h.Svc.SetFeatureFlag(r.Context(), tenantType, tenantID, flagKey, req, claims.Subject); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
