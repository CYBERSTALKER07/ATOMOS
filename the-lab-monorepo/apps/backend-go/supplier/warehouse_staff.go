package supplier

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// HandleWarehouseStaff lists warehouse staff for a given warehouse.
// GET /v1/supplier/warehouse-staff?warehouse_id=X → { staff: [...] }
// Supplier-scoped: verifies the warehouse belongs to the caller's supplier.
func HandleWarehouseStaff(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		scope := auth.GetWarehouseScope(r.Context())
		if scope == nil || scope.SupplierId == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		warehouseID := r.URL.Query().Get("warehouse_id")
		if warehouseID == "" {
			http.Error(w, `{"error":"warehouse_id query parameter is required"}`, http.StatusBadRequest)
			return
		}

		// Verify the warehouse belongs to this supplier
		whRow, err := spannerClient.Single().ReadRow(r.Context(), "Warehouses",
			spanner.Key{warehouseID}, []string{"SupplierId"})
		if err != nil {
			http.Error(w, `{"error":"warehouse not found"}`, http.StatusNotFound)
			return
		}
		var ownerSupplierId string
		if err := whRow.Columns(&ownerSupplierId); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if ownerSupplierId != scope.SupplierId {
			http.Error(w, `{"error":"warehouse does not belong to your organization"}`, http.StatusForbidden)
			return
		}

		// Query warehouse staff
		stmt := spanner.Statement{
			SQL: `SELECT WorkerId, Name, Phone, Role, IsActive, CreatedAt
			      FROM WarehouseStaff
			      WHERE WarehouseId = @warehouseId AND SupplierId = @supplierId
			      ORDER BY CreatedAt DESC`,
			Params: map[string]interface{}{
				"warehouseId": warehouseID,
				"supplierId":  scope.SupplierId,
			},
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		type staffItem struct {
			WorkerID  string `json:"worker_id"`
			Name      string `json:"name"`
			Phone     string `json:"phone"`
			Role      string `json:"role"`
			IsActive  bool   `json:"is_active"`
			CreatedAt string `json:"created_at"`
		}

		var staff []staffItem
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[WAREHOUSE STAFF] query error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var item staffItem
			var roleNull spanner.NullString
			var ts spanner.NullTime
			if err := row.Columns(&item.WorkerID, &item.Name, &item.Phone, &roleNull, &item.IsActive, &ts); err != nil {
				log.Printf("[WAREHOUSE STAFF] parse error: %v", err)
				continue
			}
			if roleNull.Valid {
				item.Role = roleNull.StringVal
			} else {
				item.Role = "WAREHOUSE_STAFF"
			}
			if ts.Valid {
				item.CreatedAt = ts.Time.Format("2006-01-02T15:04:05Z")
			}
			staff = append(staff, item)
		}

		if staff == nil {
			staff = []staffItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"staff": staff,
		})
	}
}

// HandleWarehouseStaffToggle toggles the IsActive status of a warehouse staff member.
// PATCH /v1/supplier/warehouse-staff/{worker_id} → { warehouse_id, is_active }
func HandleWarehouseStaffToggle(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		scope := auth.GetWarehouseScope(r.Context())
		if scope == nil || scope.SupplierId == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Extract worker ID from path: /v1/supplier/warehouse-staff/{worker_id}
		workerID := strings.TrimPrefix(r.URL.Path, "/v1/supplier/warehouse-staff/")
		if workerID == "" {
			http.Error(w, `{"error":"worker_id is required"}`, http.StatusBadRequest)
			return
		}

		var req struct {
			WarehouseID string `json:"warehouse_id"`
			IsActive    bool   `json:"is_active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.WarehouseID == "" {
			http.Error(w, `{"error":"warehouse_id is required"}`, http.StatusBadRequest)
			return
		}

		// Verify ownership
		whRow, err := spannerClient.Single().ReadRow(r.Context(), "Warehouses",
			spanner.Key{req.WarehouseID}, []string{"SupplierId"})
		if err != nil {
			http.Error(w, `{"error":"warehouse not found"}`, http.StatusNotFound)
			return
		}
		var ownerSupplierId string
		if err := whRow.Columns(&ownerSupplierId); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if ownerSupplierId != scope.SupplierId {
			http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
			return
		}

		// Verify worker belongs to this warehouse
		staffRow, err := spannerClient.Single().ReadRow(r.Context(), "WarehouseStaff",
			spanner.Key{workerID}, []string{"WarehouseId"})
		if err != nil {
			http.Error(w, `{"error":"worker not found"}`, http.StatusNotFound)
			return
		}
		var staffWarehouseId string
		if err := staffRow.Columns(&staffWarehouseId); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if staffWarehouseId != req.WarehouseID {
			http.Error(w, `{"error":"worker does not belong to this warehouse"}`, http.StatusForbidden)
			return
		}

		// Update IsActive
		m := spanner.Update("WarehouseStaff",
			[]string{"WorkerId", "IsActive"},
			[]interface{}{workerID, req.IsActive},
		)
		if _, err := spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
			log.Printf("[WAREHOUSE STAFF TOGGLE] spanner error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"worker_id": workerID,
			"is_active": req.IsActive,
		})
	}
}
