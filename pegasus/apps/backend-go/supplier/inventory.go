package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

// ── INVENTORY REPLENISHMENT ENGINE ──────────────────────────────────────────
// Lets suppliers increment/decrement stock with a full audit trail.
// Every mutation is logged to InventoryAuditLog for regulatory compliance.

type InventoryAdjustRequest struct {
	Adjustment int64  `json:"adjustment"` // positive = restock, negative = write-off
	Reason     string `json:"reason"`     // PRODUCTION_RECEIPT | DAMAGE_WRITEOFF | CORRECTION | RETURN_TO_STOCK
}

type InventoryItem struct {
	ProductID         string `json:"product_id"`
	SkuName           string `json:"sku_name"`
	SupplierId        string `json:"supplier_id"`
	QuantityAvailable int64  `json:"quantity_available"`
	UpdatedAt         string `json:"updated_at"`
}

type AuditEntry struct {
	AuditID     string `json:"audit_id"`
	ProductID   string `json:"product_id"`
	SkuName     string `json:"sku_name"`
	SupplierId  string `json:"supplier_id"`
	AdjustedBy  string `json:"adjusted_by"`
	PreviousQty int64  `json:"previous_qty"`
	NewQty      int64  `json:"new_qty"`
	Delta       int64  `json:"delta"`
	Reason      string `json:"reason"`
	AdjustedAt  string `json:"adjusted_at"`
}

var validReasons = map[string]bool{
	"PRODUCTION_RECEIPT": true,
	"DAMAGE_WRITEOFF":    true,
	"CORRECTION":         true,
	"RETURN_TO_STOCK":    true,
}

// HandleInventory supports:
//
//	GET  /v1/supplier/inventory          → list all inventory for this supplier
//	PATCH /v1/supplier/inventory/{sku}   → adjust stock for a specific SKU
func HandleInventory(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierId := claims.ResolveSupplierID()

		path := strings.TrimPrefix(r.URL.Path, "/v1/supplier/inventory")
		skuId := strings.TrimPrefix(path, "/")

		switch r.Method {
		case http.MethodGet:
			handleListInventory(w, r, client, supplierId)
		case http.MethodPatch:
			if skuId == "" {
				http.Error(w, `{"error":"sku_id required in path"}`, http.StatusBadRequest)
				return
			}
			handleAdjustInventory(w, r, client, supplierId, skuId)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleListInventory(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierId string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	q := r.URL.Query()

	limit := 25
	offset := 0
	if raw := q.Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	if raw := q.Get("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	sql := `SELECT sp.SkuId, sp.Name, sp.SupplierId,
		             COALESCE(si.QuantityAvailable, 0) AS QuantityAvailable,
		             si.UpdatedAt
		      FROM SupplierProducts sp
		      LEFT JOIN SupplierInventory si ON si.ProductId = sp.SkuId
		      WHERE sp.SupplierId = @sid`
	params := map[string]interface{}{"sid": supplierId}

	// Apply warehouse scope if present
	if whID := auth.EffectiveWarehouseID(r.Context()); whID != "" {
		sql += " AND si.WarehouseId = @warehouseId"
		params["warehouseId"] = whID
	}

	sql += fmt.Sprintf(" ORDER BY sp.Name LIMIT %d OFFSET %d", limit+1, offset)

	stmt := spanner.Statement{
		SQL:    sql,
		Params: params,
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var items []InventoryItem
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var item InventoryItem
		var updatedAt spanner.NullTime
		if err := row.Columns(&item.ProductID, &item.SkuName, &item.SupplierId, &item.QuantityAvailable, &updatedAt); err != nil {
			log.Printf("[INVENTORY] Row parse error: %v", err)
			continue
		}
		if updatedAt.Valid {
			item.UpdatedAt = updatedAt.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []InventoryItem{}
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":        items,
		"limit":       limit,
		"offset":      offset,
		"has_more":    hasMore,
		"next_offset": offset + len(items),
	})
}

func handleAdjustInventory(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierId string, skuId string) {
	var req InventoryAdjustRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	if req.Adjustment == 0 {
		http.Error(w, `{"error":"adjustment must be non-zero"}`, http.StatusBadRequest)
		return
	}
	if !validReasons[req.Reason] {
		http.Error(w, `{"error":"invalid reason. Must be PRODUCTION_RECEIPT|DAMAGE_WRITEOFF|CORRECTION|RETURN_TO_STOCK"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var previousQty, newQty int64

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Verify this SKU belongs to this supplier (check product ownership)
		prodRow, prodErr := txn.ReadRow(ctx, "SupplierProducts", spanner.Key{skuId}, []string{"SupplierId"})
		if prodErr != nil {
			return fmt.Errorf("SKU %s not found", skuId)
		}
		var ownerSid string
		if err := prodRow.Columns(&ownerSid); err != nil {
			return err
		}
		if ownerSid != supplierId {
			return fmt.Errorf("access denied: SKU belongs to another supplier")
		}

		// Read existing inventory (may not exist for legacy products)
		invRow, invErr := txn.ReadRow(ctx, "SupplierInventory", spanner.Key{skuId}, []string{"QuantityAvailable"})
		if invErr == nil {
			if err := invRow.Columns(&previousQty); err != nil {
				return err
			}
		}
		// invErr != nil → no inventory row → previousQty stays 0

		newQty = previousQty + req.Adjustment
		if newQty < 0 {
			return fmt.Errorf("insufficient stock: current=%d, adjustment=%d", previousQty, req.Adjustment)
		}

		// Upsert inventory (InsertOrUpdate creates missing rows)
		txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdate("SupplierInventory",
				[]string{"ProductId", "SupplierId", "QuantityAvailable", "UpdatedAt"},
				[]interface{}{skuId, supplierId, newQty, spanner.CommitTimestamp},
			),
		})

		// Write audit log
		auditId := fmt.Sprintf("AUD-%s", uuid.New().String()[:8])
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("InventoryAuditLog",
				[]string{"AuditId", "ProductId", "SupplierId", "AdjustedBy", "PreviousQty", "NewQty", "Delta", "Reason", "AdjustedAt"},
				[]interface{}{auditId, skuId, supplierId, supplierId, previousQty, newQty, req.Adjustment, req.Reason, spanner.CommitTimestamp},
			),
		})

		return nil
	})

	if err != nil {
		log.Printf("[INVENTORY] Adjust failed for %s/%s: %v", supplierId, skuId, err)
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "insufficient stock") {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "STOCK_ADJUSTED",
		"sku_id":       skuId,
		"previous_qty": previousQty,
		"new_qty":      newQty,
		"delta":        req.Adjustment,
	})
}

// HandleInventoryAuditLog returns the audit trail for this supplier's inventory changes.
func HandleInventoryAuditLog(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		q := r.URL.Query()

		limit := 25
		offset := 0
		if raw := q.Get("limit"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
				limit = parsed
			}
		}
		if raw := q.Get("offset"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		sql := `SELECT a.AuditId, a.ProductId, sp.Name, a.SupplierId, a.AdjustedBy,
			             a.PreviousQty, a.NewQty, a.Delta, a.Reason, a.AdjustedAt
			      FROM InventoryAuditLog a
			      JOIN SupplierProducts sp ON a.ProductId = sp.SkuId
			      WHERE a.SupplierId = @sid`
		params := map[string]interface{}{"sid": claims.ResolveSupplierID()}

		// Apply warehouse scope if present
		if whID := auth.EffectiveWarehouseID(r.Context()); whID != "" {
			sql += " AND a.WarehouseId = @warehouseId"
			params["warehouseId"] = whID
		}

		sql += fmt.Sprintf(" ORDER BY a.AdjustedAt DESC LIMIT %d OFFSET %d", limit+1, offset)

		stmt := spanner.Statement{
			SQL:    sql,
			Params: params,
		}

		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()

		var entries []AuditEntry
		for {
			row, err := iter.Next()
			if err != nil {
				break
			}
			var e AuditEntry
			var adjustedAt spanner.NullTime
			if err := row.Columns(&e.AuditID, &e.ProductID, &e.SkuName, &e.SupplierId, &e.AdjustedBy, &e.PreviousQty, &e.NewQty, &e.Delta, &e.Reason, &adjustedAt); err != nil {
				continue
			}
			if adjustedAt.Valid {
				e.AdjustedAt = adjustedAt.Time.Format(time.RFC3339)
			}
			entries = append(entries, e)
		}
		if entries == nil {
			entries = []AuditEntry{}
		}
		hasMore := len(entries) > limit
		if hasMore {
			entries = entries[:limit]
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":        entries,
			"limit":       limit,
			"offset":      offset,
			"has_more":    hasMore,
			"next_offset": offset + len(entries),
		})
	}
}
