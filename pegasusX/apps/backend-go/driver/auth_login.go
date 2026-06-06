package driver

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandleDriverLogin authenticates drivers for native clients.
// POST /v1/auth/driver/login  body: { "phone", "pin" }
func (s *Service) HandleDriverLogin(w http.ResponseWriter, r *http.Request) {
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

	expectPhone := strings.TrimSpace(os.Getenv("DRIVER_DEMO_PHONE"))
	if expectPhone == "" {
		expectPhone = "+998901000066"
	}
	expectPIN := strings.TrimSpace(os.Getenv("DRIVER_DEMO_PIN"))
	if expectPIN == "" {
		expectPIN = strings.TrimSpace(os.Getenv("DRIVER_DEMO_PASSWORD"))
	}
	if expectPIN == "" {
		expectPIN = "1234"
	}
	if phone != expectPhone || pin != expectPIN {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
		return
	}
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}

	driverID := strings.TrimSpace(os.Getenv("DRIVER_DEMO_ID"))
	if driverID == "" {
		driverID = "drv_factory_1"
	}
	homeNodeID := strings.TrimSpace(os.Getenv("DRIVER_DEMO_HOME_NODE_ID"))
	if homeNodeID == "" {
		homeNodeID = "factory-demo-1"
	}
	homeNodeType := strings.TrimSpace(os.Getenv("DRIVER_DEMO_HOME_NODE_TYPE"))
	if homeNodeType == "" {
		homeNodeType = string(auth.HomeNodeFactory)
	}

	claims := auth.Claims{
		Subject:      driverID,
		Role:         auth.RoleDriver,
		SupplierID:   s.supplierID,
		HomeNodeType: auth.HomeNodeType(homeNodeType),
		HomeNodeID:   homeNodeID,
	}
	token, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: 24 * time.Hour})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_token_failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":          token,
		"user_id":        driverID,
		"driver_id":      driverID,
		"role":           string(auth.RoleDriver),
		"name":           "PegasusX Demo Driver",
		"vehicle_type":   "VAN",
		"license_plate":  "01D777AA",
		"supplier_id":    s.supplierID,
		"vehicle_id":     "veh_factory_1",
		"vehicle_class":  "MEDIUM",
		"max_volume_vu":  120.0,
		"warehouse_id":     "",
		"warehouse_name":   "",
		"warehouse_lat":    0.0,
		"warehouse_lng":    0.0,
		"home_node_type":   homeNodeType,
		"home_node_id":     homeNodeID,
		"driver_mode":      "FACTORY",
		"factory_id":       homeNodeID,
		"factory_name":     "PegasusX Demo Factory",
		"factory_lat":      41.311,
		"factory_lng":      69.241,
	})
}
