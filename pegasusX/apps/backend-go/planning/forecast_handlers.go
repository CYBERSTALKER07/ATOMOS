package planning

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func writeForecastJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// HandleRunForecastOnce POST /v1/admin/planning/forecast/run-once — ops/e2e trigger.
func (r *ForecastRunner) HandleRunForecastOnce(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeForecastJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(req.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeForecastJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if !ForecastAlgoEnabled() {
		writeForecastJSON(w, http.StatusOK, map[string]any{"ok": true, "skipped": true, "reason": "FORECAST_ALGO_ENABLED off"})
		return
	}
	days := 90
	if raw := strings.TrimSpace(req.URL.Query().Get("days")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 60 && n <= 180 {
			days = n
		}
	}
	supplierID := strings.TrimSpace(req.URL.Query().Get("supplier_id"))
	target := time.Time{}
	if raw := strings.TrimSpace(req.URL.Query().Get("target")); raw != "" {
		if t, err := time.Parse("2006-01-02", raw); err == nil {
			target = t
		}
	}
	written, skipped, err := r.RunForecastPass(req.Context(), supplierID, days, target)
	if err != nil {
		writeForecastJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeForecastJSON(w, http.StatusOK, map[string]any{"ok": true, "written": written, "skipped": skipped})
}
