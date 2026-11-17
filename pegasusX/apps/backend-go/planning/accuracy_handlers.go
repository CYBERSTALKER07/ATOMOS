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
	if !ok || (claims.Role != auth.RoleAdmin && claims.Role != auth.RolePlatformAdmin) {
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
	supplierID, code := resolveAccuracySupplierID(r, claims)
	if supplierID == "" {
		writeAccuracyJSON(w, code, map[string]string{"error": "supplier_scope_required"})
		return
	}
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
	if !ok || (claims.Role != auth.RoleAdmin && claims.Role != auth.RolePlatformAdmin) {
		writeAccuracyJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	supplierID, code := resolveAccuracySupplierID(r, claims)
	if supplierID == "" {
		writeAccuracyJSON(w, code, map[string]string{"error": "supplier_scope_required"})
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
	writeAccuracyJSON(w, http.StatusOK, map[string]any{
		"items":                  rows,
		"days":                   days,
		"demote_enabled":         ForecastDemoteEnabled(),
		"demote_wape28_max":      ForecastDemoteWape28Max(),
		"demote_min_sample_days": ForecastDemoteMinSample(),
	})
}

// resolveAccuracySupplierID uses TenantContext/claims for supplier ADMIN.
// PLATFORM_ADMIN may pass ?supplier_id= for break-glass; query is ignored otherwise.
func resolveAccuracySupplierID(r *http.Request, claims auth.Claims) (string, int) {
	claimed := strings.TrimSpace(auth.PreferTenantSupplierID(r.Context(), claims.SupplierID))
	queried := strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	if claims.Role == auth.RolePlatformAdmin {
		if queried != "" {
			return queried, 0
		}
		if claimed != "" {
			return claimed, 0
		}
		return "", http.StatusBadRequest
	}
	if claimed == "" {
		return "", http.StatusForbidden
	}
	return claimed, 0
}
