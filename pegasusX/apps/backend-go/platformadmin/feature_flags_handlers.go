package platformadmin

import (
	"encoding/json"
	"errors"
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

// HandleApproveFeatureFlag POST /v1/platform-admin/tenants/{tenantType}/{tenantID}/flags/{flagKey}/approve
func (h *Handlers) HandleApproveFeatureFlag(w http.ResponseWriter, r *http.Request) {
	tenantType := chi.URLParam(r, "tenantType")
	tenantID := chi.URLParam(r, "tenantID")
	flagKey := chi.URLParam(r, "flagKey")

	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.Svc.ApproveFeatureFlag(r.Context(), tenantType, tenantID, flagKey, claims.Subject); err != nil {
		if errors.Is(err, ErrApproverMustDiffer) {
			writeErr(w, http.StatusForbidden, err.Error())
			return
		}
		if errors.Is(err, ErrFeatureFlagNotFound) {
			writeErr(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, ErrFeatureFlagNotPending) {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "flag_status": "ACTIVE"})
}
