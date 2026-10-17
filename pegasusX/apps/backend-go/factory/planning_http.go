package factory

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func writePlanningJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

// HandleNetworkMode serves GET/PUT /v1/supplier/network-mode.
func (p *PlanningService) HandleNetworkMode(w http.ResponseWriter, r *http.Request, supplierID string) {
	supplierID = strings.TrimSpace(supplierID)
	if p == nil || p.Spanner == nil {
		writePlanningJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "planning_unavailable"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		mode, _ := p.GetNetworkMode(r.Context(), supplierID)
		writePlanningJSON(w, http.StatusOK, map[string]any{
			"mode":              mode,
			"supplier_id":       supplierID,
			"planning_enabled":  PlanningEnabled(),
		})
	case http.MethodPut:
		claims, ok := auth.FromContext(r.Context())
		if !ok || claims.Role != auth.RoleAdmin {
			writePlanningJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		var req struct {
			Mode   string `json:"mode"`
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writePlanningJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		oldMode, newMode, err := p.SetNetworkMode(r.Context(), supplierID, req.Mode, claims.Subject, req.Reason)
		if err != nil {
			if err.Error() == "invalid_mode" {
				writePlanningJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_mode"})
				return
			}
			writePlanningJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
			return
		}
		writePlanningJSON(w, http.StatusOK, map[string]any{"old_mode": oldMode, "new_mode": newMode, "status": "updated"})
	default:
		writePlanningJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandlePullMatrix serves POST /v1/supplier/planning/pull-matrix.
func (p *PlanningService) HandlePullMatrix(w http.ResponseWriter, r *http.Request, supplierID string) {
	if r.Method != http.MethodPost {
		writePlanningJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if !PlanningEnabled() {
		writePlanningJSON(w, http.StatusConflict, map[string]string{"error": "factory_planning_disabled"})
		return
	}
	transfers, skus, err := p.RunPullMatrixForSupplier(r.Context(), supplierID, "MANUAL")
	if err != nil {
		writePlanningJSON(w, http.StatusInternalServerError, map[string]string{"error": "pull_matrix_failed"})
		return
	}
	writePlanningJSON(w, http.StatusOK, map[string]any{
		"status":     "completed",
		"transfers":  transfers,
		"skus":       skus,
		"source":     "MANUAL",
	})
}

// HandleKillSwitch serves POST /v1/supplier/planning/kill-switch.
func (p *PlanningService) HandleKillSwitch(w http.ResponseWriter, r *http.Request, supplierID string) {
	if r.Method != http.MethodPost {
		writePlanningJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writePlanningJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if strings.TrimSpace(req.Reason) == "" {
		req.Reason = "kill_switch"
	}
	n, err := p.RunKillSwitch(r.Context(), supplierID, claims.Subject, req.Reason)
	if err != nil {
		writePlanningJSON(w, http.StatusInternalServerError, map[string]string{"error": "kill_switch_failed"})
		return
	}
	writePlanningJSON(w, http.StatusOK, map[string]any{
		"status":             "manual_only",
		"cancelled_transfers": n,
		"mode":               NetworkModeManualOnly,
	})
}
