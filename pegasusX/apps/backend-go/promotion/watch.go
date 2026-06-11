package promotion

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

type watchSupplierRequest struct {
	SupplierID string `json:"supplier_id"`
}

// BindRetailerHub wires the retailer WebSocket hub for live promotion watch.
func (s *Service) BindRetailerHub(hub *ws.Hub) {
	if s == nil {
		return
	}
	s.retailerHub = hub
}

// HandleWatchSupplierPromotions serves POST /v1/retailer/promotions/watch.
func (s *Service) HandleWatchSupplierPromotions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" || claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req watchSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()
	supplierID := strings.TrimSpace(req.SupplierID)
	if supplierID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id_required"})
		return
	}
	if s.retailerHub != nil {
		s.retailerHub.AttachRoomsForRetailer(claims.Subject, []string{ws.SupplierPromoRoom(supplierID)})
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "watching"})
}
