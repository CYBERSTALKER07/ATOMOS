package warehouse

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// HandleInventoryPolicy serves PATCH /v1/warehouse/ops/inventory/{productID}/policy.
func (s *Service) HandleInventoryPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	productID := strings.TrimSpace(chi.URLParam(r, "productID"))
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing_product_id"})
		return
	}
	if s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inventory_unavailable"})
		return
	}

	body, ok := readMutationBody(w, r, 16*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), key)
		}
	}()

	var req struct {
		OutOfStockPolicy *string `json:"out_of_stock_policy"`
		ReorderThreshold *int64  `json:"reorder_threshold"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if req.OutOfStockPolicy == nil && req.ReorderThreshold == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no_fields_to_update"})
		return
	}

	policy := ""
	if req.OutOfStockPolicy != nil {
		policy = strings.ToUpper(strings.TrimSpace(*req.OutOfStockPolicy))
		switch policy {
		case OutOfStockPolicyInherit, OutOfStockPolicyReject, OutOfStockPolicyAcceptBackorder:
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_out_of_stock_policy"})
			return
		}
	}

	err := s.repo.UpdateInventoryPolicy(r.Context(), whID, productID, policy, req.ReorderThreshold, func(buf outbox.TxnBuffer) error {
		return nil
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_update_inventory_policy"})
		return
	}

	claims, _ := auth.FromContext(r.Context())
	s.log.InfoContext(r.Context(), "warehouse.inventory.policy.updated",
		"warehouse_id", whID,
		"product_id", productID,
		"actor", strings.TrimSpace(claims.Subject),
	)

	resp := map[string]string{"status": "updated"}
	respBytes, _ := json.Marshal(resp)
	s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSON(w, http.StatusOK, resp)
}
