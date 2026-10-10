package replenishment

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// FillRateReplayAPI serves ADMIN fill-rate replay.
type FillRateReplayAPI struct {
	Client *spanner.Client
}

func writeReplayJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// HandleReplay POST /v1/admin/planning/safety-stock/replay — ops/e2e trigger.
func (a *FillRateReplayAPI) HandleReplay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeReplayJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeReplayJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if a == nil || a.Client == nil {
		writeReplayJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "replay_unavailable"})
		return
	}
	days := replayDefaultDays
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= replayMinCalendarDays && n <= 180 {
			days = n
		}
	}
	supplierID := strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	cfg := ReplayConfig{
		SupplierID:  supplierID,
		Days:        days,
		RequireGate: strings.EqualFold(r.URL.Query().Get("require_gate"), "true"),
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("target_service_level")); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil && v > 0 && v < 1 {
			cfg.TargetServiceLevel = v
		}
	}
	result, err := RunFillRateReplay(r.Context(), a.Client, cfg)
	if err != nil {
		// Gate failure still returns body with metrics for ops inspection.
		if result.GateRequired && !result.PassGate {
			writeReplayJSON(w, http.StatusConflict, result)
			return
		}
		writeReplayJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeReplayJSON(w, http.StatusOK, result)
}
