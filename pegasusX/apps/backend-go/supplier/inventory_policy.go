package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const (
	oosPolicyInherit         = "INHERIT"
	oosPolicyReject          = "REJECT"
	oosPolicyAcceptBackorder = "ACCEPT_BACKORDER"
)

func resolveSupplierOutOfStockPolicy(warehouseDefault, productOverride string) string {
	override := strings.ToUpper(strings.TrimSpace(productOverride))
	switch override {
	case oosPolicyReject, oosPolicyAcceptBackorder:
		return override
	}
	def := strings.ToUpper(strings.TrimSpace(warehouseDefault))
	if def == oosPolicyAcceptBackorder {
		return oosPolicyAcceptBackorder
	}
	return oosPolicyReject
}

func (s *Service) listSupplierInventoryV2(ctx context.Context, supplierID string) ([]InventoryLevelView, error) {
	if s.portalSpanner == nil {
		return nil, nil
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return nil, fmt.Errorf("supplier_id required")
	}
	stmt := spanner.Statement{
		SQL: `SELECT si.ProductId, si.WarehouseId, si.SupplierId,
		             si.QuantityOnHand, si.QuantityReserved,
		             COALESCE(si.ReorderThreshold, 0),
		             COALESCE(si.OutOfStockPolicy, ''),
		             COALESCE(w.DefaultOutOfStockPolicy, 'REJECT')
		      FROM SupplierInventoryV2 si
		      INNER JOIN Warehouses w ON si.WarehouseId = w.WarehouseId AND si.SupplierId = w.SupplierId
		      WHERE si.SupplierId = @sid
		      ORDER BY si.WarehouseId, si.ProductId`,
		Params: map[string]any{"sid": supplierID},
	}
	iter := s.portalSpanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var levels []InventoryLevelView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query supplier inventory v2: %w", err)
		}
		var (
			productID, warehouseID, sid, productPolicy, warehousePolicy string
			qoh, qr, reorder                                            int64
		)
		if err := row.Columns(&productID, &warehouseID, &sid, &qoh, &qr, &reorder, &productPolicy, &warehousePolicy); err != nil {
			return nil, err
		}
		effective := resolveSupplierOutOfStockPolicy(warehousePolicy, productPolicy)
		levels = append(levels, InventoryLevelView{
			InventoryID:      fmt.Sprintf("%s:%s:%s", sid, warehouseID, productID),
			ProductID:        productID,
			WarehouseID:      warehouseID,
			SupplierID:       sid,
			QuantityOnHand:   qoh,
			QuantityReserved: qr,
			ReorderThreshold: reorder,
			OutOfStockPolicy: strings.ToUpper(strings.TrimSpace(productPolicy)),
			EffectivePolicy:  effective,
			AcceptsBackorder: effective == oosPolicyAcceptBackorder,
		})
	}
	return levels, nil
}

// HandleInventoryPolicy serves PATCH /v1/supplier/inventory/policy.
func (s *Service) HandleInventoryPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inventory_unavailable"})
		return
	}
	sid := s.scopedSupplierID(r)
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
			s.releaseMutationReplay(r.Context(), r)
		}
	}()

	var req struct {
		WarehouseID      string  `json:"warehouse_id"`
		ProductID        string  `json:"product_id"`
		SKUID            string  `json:"sku_id"`
		OutOfStockPolicy *string `json:"out_of_stock_policy"`
		ReorderThreshold *int64  `json:"reorder_threshold"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	warehouseID := strings.TrimSpace(req.WarehouseID)
	productID := strings.TrimSpace(req.ProductID)
	if productID == "" {
		productID = strings.TrimSpace(req.SKUID)
	}
	if warehouseID == "" || productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_and_product_id_required"})
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
		case oosPolicyInherit, oosPolicyReject, oosPolicyAcceptBackorder:
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_out_of_stock_policy"})
			return
		}
	}

	row, err := s.portalSpanner.Single().ReadRow(r.Context(), "Warehouses", spanner.Key{warehouseID}, []string{"SupplierId"})
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "warehouse_not_found"})
		return
	}
	var whSupplierID string
	if err := row.Column(0, &whSupplierID); err != nil || strings.TrimSpace(whSupplierID) != sid {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_forbidden"})
		return
	}

	update := map[string]any{
		"SupplierId":  sid,
		"WarehouseId": warehouseID,
		"ProductId":   productID,
		"UpdatedAt":   spanner.CommitTimestamp,
	}
	if policy != "" {
		update["OutOfStockPolicy"] = policy
	}
	if req.ReorderThreshold != nil {
		update["ReorderThreshold"] = *req.ReorderThreshold
	}
	if _, err := s.portalSpanner.Apply(r.Context(), []*spanner.Mutation{spanner.InsertOrUpdateMap("SupplierInventoryV2", update)}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_update_inventory_policy"})
		return
	}

	resp := map[string]string{"status": "updated"}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	writeJSON(w, http.StatusOK, resp)
}
