package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
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
	SkuID             string `json:"sku_id"`
	ProductName       string `json:"product_name"`
	SkuName           string `json:"sku_name"`
	SupplierId        string `json:"supplier_id"`
	QuantityAvailable int64  `json:"quantity_available"`
	UpdatedAt         string `json:"updated_at"`
}

type AuditEntry struct {
	AuditID     string `json:"audit_id"`
	ProductID   string `json:"product_id"`
	SkuID       string `json:"sku_id"`
	ProductName string `json:"product_name"`
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

var (
	errInventoryProductNotFound = errors.New("inventory product not found")
	errInventoryAccessDenied    = errors.New("inventory access denied")
)

type errInventoryInsufficientStock struct {
	current    int64
	adjustment int64
}

func (e errInventoryInsufficientStock) Error() string {
	return fmt.Sprintf("insufficient stock: current=%d, adjustment=%d", e.current, e.adjustment)
}

func inventoryMutationSKU(pathSKU, bodySKU, productID string) string {
	if value := strings.TrimSpace(pathSKU); value != "" {
		return value
	}
	if value := strings.TrimSpace(bodySKU); value != "" {
		return value
	}
	return strings.TrimSpace(productID)
}

func normalizeInventoryItemAliases(item *InventoryItem) {
	if item == nil {
		return
	}
	if item.SkuID == "" {
		item.SkuID = item.ProductID
	}
	if item.ProductName == "" {
		item.ProductName = item.SkuName
	}
}

func normalizeInventoryAuditAliases(entry *AuditEntry) {
	if entry == nil {
		return
	}
	if entry.SkuID == "" {
		entry.SkuID = entry.ProductID
	}
	if entry.ProductName == "" {
		entry.ProductName = entry.SkuName
	}
}

// HandleInventory supports:
//
//	GET  /v1/supplier/inventory          → list all inventory for this supplier
//	PATCH /v1/supplier/inventory         → adjust stock for a specific SKU
//	PATCH /v1/supplier/inventory/{sku}   → legacy path alias for stock adjustment
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
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[INVENTORY] list query error for supplier %s: %v", supplierId, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var item InventoryItem
		var updatedAt spanner.NullTime
		if err := row.Columns(&item.ProductID, &item.SkuName, &item.SupplierId, &item.QuantityAvailable, &updatedAt); err != nil {
			log.Printf("[INVENTORY] row parse error for supplier %s: %v", supplierId, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if updatedAt.Valid {
			item.UpdatedAt = updatedAt.Time.Format(time.RFC3339)
		}
		normalizeInventoryItemAliases(&item)
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
	var req struct {
		InventoryAdjustRequest
		SkuID     string `json:"sku_id"`
		ProductID string `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	targetSKU := inventoryMutationSKU(skuId, req.SkuID, req.ProductID)
	if targetSKU == "" {
		http.Error(w, `{"error":"sku_id or product_id required"}`, http.StatusBadRequest)
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
		prodRow, prodErr := txn.ReadRow(ctx, "SupplierProducts", spanner.Key{targetSKU}, []string{"SupplierId"})
		if prodErr != nil {
			if spanner.ErrCode(prodErr) == codes.NotFound {
				return errInventoryProductNotFound
			}
			return fmt.Errorf("read inventory product %s: %w", targetSKU, prodErr)
		}
		var ownerSid string
		if err := prodRow.Columns(&ownerSid); err != nil {
			return fmt.Errorf("parse inventory product %s owner: %w", targetSKU, err)
		}
		if ownerSid != supplierId {
			return errInventoryAccessDenied
		}

		// Read existing inventory (may not exist for legacy products)
		invRow, invErr := txn.ReadRow(ctx, "SupplierInventory", spanner.Key{targetSKU}, []string{"QuantityAvailable"})
		if invErr == nil {
			if err := invRow.Columns(&previousQty); err != nil {
				return fmt.Errorf("parse inventory %s quantity: %w", targetSKU, err)
			}
		} else if spanner.ErrCode(invErr) != codes.NotFound {
			return fmt.Errorf("read inventory %s: %w", targetSKU, invErr)
		}

		newQty = previousQty + req.Adjustment
		if newQty < 0 {
			return errInventoryInsufficientStock{current: previousQty, adjustment: req.Adjustment}
		}

		// Upsert inventory (InsertOrUpdate creates missing rows)
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdate("SupplierInventory",
				[]string{"ProductId", "SupplierId", "QuantityAvailable", "UpdatedAt"},
				[]interface{}{targetSKU, supplierId, newQty, spanner.CommitTimestamp},
			),
		}); err != nil {
			return fmt.Errorf("buffer inventory %s update: %w", targetSKU, err)
		}

		// Write audit log
		auditId := fmt.Sprintf("AUD-%s", uuid.New().String()[:8])
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("InventoryAuditLog",
				[]string{"AuditId", "ProductId", "SupplierId", "AdjustedBy", "PreviousQty", "NewQty", "Delta", "Reason", "AdjustedAt"},
				[]interface{}{auditId, targetSKU, supplierId, supplierId, previousQty, newQty, req.Adjustment, req.Reason, spanner.CommitTimestamp},
			),
		}); err != nil {
			return fmt.Errorf("buffer inventory %s audit: %w", targetSKU, err)
		}

		return nil
	})

	if err != nil {
		log.Printf("[INVENTORY] Adjust failed for %s/%s: %v", supplierId, skuId, err)
		if errors.Is(err, errInventoryProductNotFound) {
			http.Error(w, `{"error":"SKU not found"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, errInventoryAccessDenied) {
			http.Error(w, `{"error":"access denied: SKU belongs to another supplier"}`, http.StatusForbidden)
			return
		}
		var insufficient errInventoryInsufficientStock
		if errors.As(err, &insufficient) {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, insufficient.Error()), http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "STOCK_ADJUSTED",
		"sku_id":       targetSKU,
		"product_id":   targetSKU,
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
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[INVENTORY] audit query error for supplier %s: %v", claims.ResolveSupplierID(), err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var e AuditEntry
			var adjustedAt spanner.NullTime
			if err := row.Columns(&e.AuditID, &e.ProductID, &e.SkuName, &e.SupplierId, &e.AdjustedBy, &e.PreviousQty, &e.NewQty, &e.Delta, &e.Reason, &adjustedAt); err != nil {
				log.Printf("[INVENTORY] audit parse error for supplier %s: %v", claims.ResolveSupplierID(), err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if adjustedAt.Valid {
				e.AdjustedAt = adjustedAt.Time.Format(time.RFC3339)
			}
			normalizeInventoryAuditAliases(&e)
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
