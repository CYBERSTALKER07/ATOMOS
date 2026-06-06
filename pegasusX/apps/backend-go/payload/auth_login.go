package payload

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandlePayloaderLogin authenticates payload terminal staff (phone + PIN).
// POST /v1/auth/payloader/login
func (s *Service) HandlePayloaderLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	defer r.Body.Close()

	var req struct {
		Phone string `json:"phone"`
		PIN   string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	phone := strings.TrimSpace(req.Phone)
	pin := strings.TrimSpace(req.PIN)
	if phone == "" || pin == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone_and_pin_required"})
		return
	}

	expectPhone := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_PHONE"))
	if expectPhone == "" {
		expectPhone = "+998901110022"
	}
	expectPIN := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_PIN"))
	if expectPIN == "" {
		expectPIN = "33333333"
	}
	if phone != expectPhone || pin != expectPIN {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
		return
	}

	warehouseID := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_WAREHOUSE_ID"))
	if warehouseID == "" {
		warehouseID = "warehouse-demo-1"
	}
	warehouseName := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_WAREHOUSE_NAME"))
	if warehouseName == "" {
		warehouseName = "PegasusX Demo Warehouse"
	}
	workerID := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_WORKER_ID"))
	if workerID == "" {
		workerID = "payloader-demo-1"
	}
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}

	claims := auth.Claims{
		Subject:      workerID,
		Role:         auth.RolePayload,
		SupplierID:   s.supplierID,
		SupplierRole: auth.RoleWarehouseAdmin,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   warehouseID,
		IsConfigured: true,
	}
	token, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 24 * time.Hour})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_token_failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":          token,
		"worker_id":      workerID,
		"supplier_id":    s.supplierID,
		"role":           string(auth.RolePayload),
		"name":           "Demo Payloader",
		"warehouse_id":   warehouseID,
		"warehouse_name": warehouseName,
		"warehouse_lat":  41.3111,
		"warehouse_lng":  69.2797,
	})
}
