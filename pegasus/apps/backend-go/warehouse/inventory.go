package warehouse

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/spannerx"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Inventory ────────────────────────────────────────────────────────────────
// Warehouse-scoped inventory view and adjustment.

type InventoryItem struct {
	SkuID            string  `json:"sku_id"`
	ProductID        string  `json:"product_id"`
	SKU              string  `json:"sku"`
	ProductName      string  `json:"product_name"`
	Quantity         int64   `json:"quantity"`
	ReorderThreshold int64   `json:"reorder_threshold"`
	VolumeVU         float64 `json:"volume_vu,omitempty"`
	CategoryID       string  `json:"category_id,omitempty"`
	IsLowStock       bool    `json:"is_low_stock"`
	LastUpdated      string  `json:"last_updated,omitempty"`
}

func inventorySearchTerm(r *http.Request) string {
	if query := strings.TrimSpace(r.URL.Query().Get("q")); query != "" {
		return query
	}
	return strings.TrimSpace(r.URL.Query().Get("search"))
}

func inventoryMutationSKU(skuID, productID string) string {
	if value := strings.TrimSpace(skuID); value != "" {
		return value
	}
	return strings.TrimSpace(productID)
}

func normalizeInventoryItemAliases(item *InventoryItem) {
	if item == nil {
		return
	}
	if item.ProductID == "" {
		item.ProductID = item.SkuID
	}
	if item.SKU == "" {
		item.SKU = item.SkuID
	}
}

func inventoryResponsePayload(items []InventoryItem) map[string]interface{} {
	return map[string]interface{}{
		"inventory": items,
		"items":     items,
		"total":     len(items),
	}
}

// HandleOpsInventory — GET/PATCH for /v1/warehouse/ops/inventory

func HandleOpsInventory(spannerClient *spanner.Client, rc *cache.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listOpsInventory(w, r, spannerClient)
		case http.MethodPatch:
			adjustOpsInventory(w, r, spannerClient, rc)
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
	if q := inventorySearchTerm(r); q != "" {
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
		normalizeInventoryItemAliases(&item)
		item.LastUpdated = updatedAt.Format(time.RFC3339)
		item.IsLowStock = item.Quantity <= item.ReorderThreshold && item.ReorderThreshold > 0
		items = append(items, item)
	}
	if items == nil {
		items = []InventoryItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inventoryResponsePayload(items))
}

func adjustOpsInventory(w http.ResponseWriter, r *http.Request, client *spanner.Client, rc *cache.Cache) {
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
		ProductID        string `json:"product_id"`
		Quantity         *int64 `json:"quantity,omitempty"`
		ReorderThreshold *int64 `json:"reorder_threshold,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	skuID := inventoryMutationSKU(req.SkuID, req.ProductID)
	if skuID == "" {
		http.Error(w, `{"error":"sku_id or product_id required"}`, http.StatusBadRequest)
		return
	}

	cols := []string{"ProductId", "SupplierId", "WarehouseId"}
	vals := []interface{}{skuID, ops.SupplierID, ops.WarehouseID}
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
	if _, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		log.Printf("[WH INVENTORY] upsert error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if rc != nil {
		rc.Invalidate(r.Context(), cache.WarehouseGeoMember(ops.WarehouseID), cache.PrefixWarehouseDetail+ops.WarehouseID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated", "sku_id": skuID, "product_id": skuID})
}
