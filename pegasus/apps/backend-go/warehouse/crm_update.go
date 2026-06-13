package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"backend-go/auth"
	"backend-go/proximity"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
)

type updateCRMRetailerRequest struct {
	ReceivingWindowOpen    string  `json:"receiving_window_open"`
	ReceivingWindowClose   string  `json:"receiving_window_close"`
	AccessType             string  `json:"access_type"`
	StorageCeilingHeightCM float64 `json:"storage_ceiling_height_cm"`
}

// HandleOpsCRMRetailer updates retailer delivery-receiving metadata for warehouse CRM.
// PATCH /v1/warehouse/ops/crm/{retailer_id}
func HandleOpsCRMRetailer(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		ops := auth.GetWarehouseOps(r.Context())
		if ops == nil {
			http.Error(w, "Warehouse scope required", http.StatusForbidden)
			return
		}

		retailerID := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/ops/crm/")
		retailerID = strings.Trim(retailerID, "/")
		if retailerID == "" {
			http.Error(w, `{"error":"retailer_id required"}`, http.StatusBadRequest)
			return
		}

		var req updateCRMRetailerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		openWindow := strings.TrimSpace(req.ReceivingWindowOpen)
		closeWindow := strings.TrimSpace(req.ReceivingWindowClose)
		if openWindow == "" && closeWindow == "" && req.AccessType == "" && req.StorageCeilingHeightCM <= 0 {
			http.Error(w, `{"error":"at least one field required"}`, http.StatusBadRequest)
			return
		}

		if openWindow != "" {
			canon, err := proximity.ValidateReceivingWindow(openWindow)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
				return
			}
			openWindow = canon
		}
		if closeWindow != "" {
			canon, err := proximity.ValidateReceivingWindow(closeWindow)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
				return
			}
			closeWindow = canon
		}

		_, err := spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			if err := assertRetailerServedByWarehouse(ctx, txn, ops.SupplierID, ops.WarehouseID, retailerID); err != nil {
				return err
			}

			cols := []string{"RetailerId"}
			vals := []interface{}{retailerID}
			if openWindow != "" {
				cols = append(cols, "ReceivingWindowOpen")
				vals = append(vals, openWindow)
			}
			if closeWindow != "" {
				cols = append(cols, "ReceivingWindowClose")
				vals = append(vals, closeWindow)
			}
			if access := strings.TrimSpace(req.AccessType); access != "" {
				cols = append(cols, "AccessType")
				vals = append(vals, access)
			}
			if req.StorageCeilingHeightCM > 0 {
				cols = append(cols, "StorageCeilingHeightCM")
				vals = append(vals, req.StorageCeilingHeightCM)
			}

			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Retailers", cols, vals),
			})
		})
		if err != nil {
			if strings.Contains(err.Error(), "not linked") {
				http.Error(w, `{"error":"retailer not linked to this warehouse"}`, http.StatusForbidden)
				return
			}
			log.Printf("[WH CRM] update retailer=%s trace=%s err=%v",
				retailerID, telemetry.TraceIDFromContext(r.Context()), err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":      "ok",
			"retailer_id": retailerID,
		})
	}
}

func assertRetailerServedByWarehouse(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	supplierID, warehouseID, retailerID string,
) error {
	stmt := spanner.Statement{
		SQL: `SELECT 1
		      FROM Orders
		      WHERE SupplierId = @sid AND WarehouseId = @wh AND RetailerId = @rid
		      LIMIT 1`,
		Params: map[string]interface{}{
			"sid": supplierID,
			"wh":  warehouseID,
			"rid": retailerID,
		},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	_, err := iter.Next()
	if err != nil {
		return fmt.Errorf("retailer not linked to warehouse")
	}
	return nil
}
