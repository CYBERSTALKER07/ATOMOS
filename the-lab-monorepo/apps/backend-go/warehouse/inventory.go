package warehouse

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/spannerx"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Inventory ────────────────────────────────────────────────────────────────
// Warehouse-scoped inventory view and adjustment.

type InventoryItem struct {
	SkuID            string  `json:"sku_id"`
	ProductName      string  `json:"product_name"`
	Quantity         int64   `json:"quantity"`
	ReorderThreshold int64   `json:"reorder_threshold"`
	VolumeVU         float64 `json:"volume_vu,omitempty"`
	CategoryID       string  `json:"category_id,omitempty"`
	IsLowStock       bool    `json:"is_low_stock"`
	LastUpdated      string  `json:"last_updated,omitempty"`
}

// HandleOpsInventory — GET/PATCH for /v1/warehouse/ops/inventory
func HandleOpsInventory(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listOpsInventory(w, r, spannerClient)
		case http.MethodPatch:
			adjustOpsInventory(w, r, spannerClient)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func listOpsInventory(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	ops := auth.GetWarehouseOps(r.Context())
	if ops == nil {
		http.Error(w, "Warehouse scope required", http.StatusForbidden)
		return
	}

	sql := `SELECT sp.SkuId, sp.Name, COALESCE(si.Quantity, 0),
	               COALESCE(si.ReorderThreshold, 0), COALESCE(sp.VolumeVU, 0),
	               COALESCE(sp.CategoryId, ''), COALESCE(si.UpdatedAt, sp.CreatedAt)
	        FROM SupplierProducts sp
	        LEFT JOIN SupplierInventory si ON sp.SkuId = si.ProductId
	             AND si.SupplierId = @sid AND si.WarehouseId = @whId
	        WHERE sp.SupplierId = @sid`

	params := map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID}

	// Search
	if q := r.URL.Query().Get("q"); q != "" {
		sql += " AND LOWER(sp.Name) LIKE @search"
		params["search"] = "%" + strings.ToLower(q) + "%"
	}
	// Low stock filter
	if r.URL.Query().Get("low_stock") == "true" {
		sql += " AND si.Quantity <= si.ReorderThreshold"
	}

	sql += " ORDER BY sp.Name LIMIT 500"

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := spannerx.StaleQuery(r.Context(), client, stmt)
	defer iter.Stop()

	var items []InventoryItem
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[WH INVENTORY] list error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var item InventoryItem
		var updatedAt time.Time
		if err := row.Columns(&item.SkuID, &item.ProductName, &item.Quantity,
			&item.ReorderThreshold, &item.VolumeVU, &item.CategoryID, &updatedAt); err != nil {
			log.Printf("[WH INVENTORY] parse: %v", err)
			continue
		}
		item.LastUpdated = updatedAt.Format(time.RFC3339)
		item.IsLowStock = item.Quantity <= item.ReorderThreshold && item.ReorderThreshold > 0
		items = append(items, item)
	}
	if items == nil {
		items = []InventoryItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"inventory": items, "total": len(items)})
}

func adjustOpsInventory(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	ops := auth.GetWarehouseOps(r.Context())
	if ops == nil {
		http.Error(w, "Warehouse scope required", http.StatusForbidden)
		return
	}
	if ops.WarehouseRole != "WAREHOUSE_ADMIN" {
		http.Error(w, `{"error":"warehouse admin required"}`, http.StatusForbidden)
		return
	}

	var req struct {
		SkuID            string `json:"sku_id"`
		Quantity         *int64 `json:"quantity,omitempty"`
		ReorderThreshold *int64 `json:"reorder_threshold,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.SkuID == "" {
		http.Error(w, `{"error":"sku_id required"}`, http.StatusBadRequest)
		return
	}

	cols := []string{"ProductId", "SupplierId", "WarehouseId"}
	vals := []interface{}{req.SkuID, ops.SupplierID, ops.WarehouseID}
	if req.Quantity != nil {
		cols = append(cols, "Quantity")
		vals = append(vals, *req.Quantity)
	}
	if req.ReorderThreshold != nil {
		cols = append(cols, "ReorderThreshold")
		vals = append(vals, *req.ReorderThreshold)
	}
	cols = append(cols, "UpdatedAt")
	vals = append(vals, spanner.CommitTimestamp)

	m := spanner.InsertOrUpdate("SupplierInventory", cols, vals)
	if _, err := client.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
		log.Printf("[WH INVENTORY] upsert error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated", "sku_id": req.SkuID})
}
