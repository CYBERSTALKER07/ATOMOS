package planning

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func writeAccuracyJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// HandleRunAccuracyOnce POST /v1/admin/planning/accuracy/run-once — ops/e2e trigger.
func (s *AccuracyService) HandleRunAccuracyOnce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAccuracyJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeAccuracyJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if !ForecastAccuracyEnabled() {
		writeAccuracyJSON(w, http.StatusOK, map[string]any{"ok": true, "skipped": true, "reason": "FORECAST_ACCURACY_ENABLED off"})
		return
	}
	days := 28
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}
	supplierID := strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	written, alerts, err := s.RunAccuracyPass(r.Context(), supplierID, days)
	if err != nil {
		writeAccuracyJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeAccuracyJSON(w, http.StatusOK, map[string]any{"ok": true, "written": written, "alerts": alerts})
}

// HandleListAccuracy GET /v1/admin/planning/accuracy — returns recent
// ForecastAccuracyDaily rows for the forecast-accuracy UI. Admin-only.
func (s *AccuracyService) HandleListAccuracy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAccuracyJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeAccuracyJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	supplierID := strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	if supplierID == "" {
		writeAccuracyJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id is required"})
		return
	}
	days := 28
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}
	rows, err := ListAccuracyRows(r.Context(), s.Client, supplierID,
		strings.TrimSpace(r.URL.Query().Get("warehouse_id")),
		strings.TrimSpace(r.URL.Query().Get("product_id")), days)
	if err != nil {
		writeAccuracyJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if rows == nil {
		rows = []AccuracyDailyRow{}
	}
	writeAccuracyJSON(w, http.StatusOK, map[string]any{"items": rows, "days": days})
}
