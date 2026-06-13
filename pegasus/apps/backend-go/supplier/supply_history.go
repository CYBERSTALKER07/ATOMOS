package supplier

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SupplyRequestHistoryRow is the supplier CEO read model for supply-request audit.
type SupplyRequestHistoryRow struct {
	RequestID             string     `json:"request_id"`
	WarehouseID           string     `json:"warehouse_id"`
	WarehouseName         string     `json:"warehouse_name"`
	FactoryID             string     `json:"factory_id"`
	FactoryName           string     `json:"factory_name"`
	State                 string     `json:"state"`
	Priority              string     `json:"priority"`
	TotalVolumeVU         float64    `json:"total_volume_vu"`
	TransferOrderID       string     `json:"transfer_order_id,omitempty"`
	TransferState         string     `json:"transfer_state,omitempty"`
	LaneType              string     `json:"lane_type,omitempty"`
	IsNearbyFactory       bool       `json:"is_nearby_factory"`
	RequestedDeliveryDate *time.Time `json:"requested_delivery_date,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             *time.Time `json:"updated_at,omitempty"`
}

// HandleSupplyRequestHistory returns enriched supply-request rows for the supplier portal.
// GET /v1/supplier/supply-requests/history?state=FULFILLED&warehouse_id=&factory_id=
func HandleSupplyRequestHistory(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		sql := `SELECT sr.RequestId, sr.WarehouseId, w.Name, sr.FactoryId, f.Name,
		               sr.State, sr.Priority, sr.TotalVolumeVU, sr.TransferOrderId,
		               COALESCE(ito.State, ''), COALESCE(ito.LaneType, 'TRUCK'),
		               COALESCE(w.IsNearbyFactory, false),
		               sr.RequestedDeliveryDate, sr.CreatedAt, sr.UpdatedAt
		        FROM SupplyRequests sr
		        JOIN Warehouses w ON sr.WarehouseId = w.WarehouseId
		        JOIN Factories f ON sr.FactoryId = f.FactoryId
		        LEFT JOIN InternalTransferOrders ito ON sr.TransferOrderId = ito.TransferId
		        WHERE sr.SupplierId = @supplierId`
		params := map[string]interface{}{"supplierId": supplierID}

		if state := r.URL.Query().Get("state"); state != "" {
			sql += " AND sr.State = @state"
			params["state"] = state
		}
		if warehouseID := r.URL.Query().Get("warehouse_id"); warehouseID != "" {
			sql += " AND sr.WarehouseId = @warehouseId"
			params["warehouseId"] = warehouseID
		}
		if factoryID := r.URL.Query().Get("factory_id"); factoryID != "" {
			sql += " AND sr.FactoryId = @factoryId"
			params["factoryId"] = factoryID
		}
		sql += " ORDER BY sr.CreatedAt DESC LIMIT 200"

		iter := client.Single().Query(r.Context(), spanner.Statement{SQL: sql, Params: params})
		defer iter.Stop()

		rows := make([]SupplyRequestHistoryRow, 0)
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[SUPPLY HISTORY] query error supplier=%s: %v", supplierID, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var entry SupplyRequestHistoryRow
			var transferID spanner.NullString
			var reqDelivery spanner.NullTime
			var updatedAt spanner.NullTime
			if err := row.Columns(
				&entry.RequestID, &entry.WarehouseID, &entry.WarehouseName,
				&entry.FactoryID, &entry.FactoryName, &entry.State, &entry.Priority,
				&entry.TotalVolumeVU, &transferID, &entry.TransferState, &entry.LaneType,
				&entry.IsNearbyFactory, &reqDelivery, &entry.CreatedAt, &updatedAt,
			); err != nil {
				log.Printf("[SUPPLY HISTORY] row parse error: %v", err)
				continue
			}
			if transferID.Valid {
				entry.TransferOrderID = transferID.StringVal
			}
			if reqDelivery.Valid {
				entry.RequestedDeliveryDate = &reqDelivery.Time
			}
			if updatedAt.Valid {
				entry.UpdatedAt = &updatedAt.Time
			}
			rows = append(rows, entry)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": rows,
			"count": len(rows),
		})
	}
}
