package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"backend-go/auth"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── DISPUTE & RETURNS QUEUE ─────────────────────────────────────────────────
// Tracks REJECTED_DAMAGED line items. Supplier can acknowledge returns and
// either process write-off or return-to-stock (inventory restoration).

type ReturnItem struct {
	LineItemID   string `json:"line_item_id"`
	OrderID      string `json:"order_id"`
	SkuID        string `json:"sku_id"`
	ProductName  string `json:"product_name"`
	Quantity     int64  `json:"quantity"`
	UnitPrice    int64  `json:"unit_price"`
	Status       string `json:"status"`
	RetailerName string `json:"retailer_name"`
	CreatedAt    string `json:"created_at"`
}

type ResolveReturnRequest struct {
	LineItemID string `json:"line_item_id"`
	Resolution string `json:"resolution"` // WRITE_OFF | RETURN_TO_STOCK
	Notes      string `json:"notes"`
}

type ReturnsService struct {
	Client   *spanner.Client
	Producer *kafka.Writer
}

func NewReturnsService(client *spanner.Client, producer *kafka.Writer) *ReturnsService {
	return &ReturnsService{Client: client, Producer: producer}
}

// HandleReturns — GET: list REJECTED_DAMAGED line items for this supplier
func (s *ReturnsService) HandleReturns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierId := claims.ResolveSupplierID()

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sql := `SELECT li.LineItemId, li.OrderId, li.SkuId, sp.Name, li.Quantity,
	               COALESCE(li.UnitPrice, 0), li.Status,
	               COALESCE(ret.Name, 'Unknown'), o.CreatedAt
	        FROM OrderLineItems li
	        JOIN Orders o ON li.OrderId = o.OrderId
	        JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
	        LEFT JOIN Retailers ret ON o.RetailerId = ret.RetailerId
	        WHERE o.SupplierId = @sid
	          AND li.Status = 'REJECTED_DAMAGED'`

	params := map[string]interface{}{"sid": supplierId}
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

	// Apply warehouse scope if present
	if whID := auth.EffectiveWarehouseID(r.Context()); whID != "" {
		sql += " AND o.WarehouseId = @warehouseId"
		params["warehouseId"] = whID
	}

	sql += fmt.Sprintf(" ORDER BY o.CreatedAt DESC LIMIT %d OFFSET %d", limit+1, offset)

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var items []ReturnItem
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[RETURNS] Query error: %v", err)
			break
		}
		var item ReturnItem
		var createdAt spanner.NullTime
		if err := row.Columns(&item.LineItemID, &item.OrderID, &item.SkuID, &item.ProductName,
			&item.Quantity, &item.UnitPrice, &item.Status, &item.RetailerName, &createdAt); err != nil {
			log.Printf("[RETURNS] Row parse error: %v", err)
			continue
		}
		if createdAt.Valid {
			item.CreatedAt = createdAt.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []ReturnItem{}
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

// HandleResolveReturn — POST: supplier acknowledges a return and decides disposition
func (s *ReturnsService) HandleResolveReturn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierId := claims.ResolveSupplierID()

	var req ResolveReturnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.LineItemID == "" {
		http.Error(w, `{"error":"line_item_id required"}`, http.StatusBadRequest)
		return
	}
	if req.Resolution != "WRITE_OFF" && req.Resolution != "RETURN_TO_STOCK" {
		http.Error(w, `{"error":"resolution must be WRITE_OFF or RETURN_TO_STOCK"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var skuId string
	var qty int64
	var orderId string

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Read line item + verify supplier ownership
		liStmt := spanner.Statement{
			SQL: `SELECT li.LineItemId, li.OrderId, li.SkuId, li.Quantity, li.Status, o.SupplierId
			      FROM OrderLineItems li
			      JOIN Orders o ON li.OrderId = o.OrderId
			      WHERE li.LineItemId = @lid`,
			Params: map[string]interface{}{"lid": req.LineItemID},
		}
		liIter := txn.Query(ctx, liStmt)
		defer liIter.Stop()

		row, err := liIter.Next()
		if err != nil {
			return fmt.Errorf("line item not found: %s", req.LineItemID)
		}
		var liId, liStatus, liSupplierId string
		if err := row.Columns(&liId, &orderId, &skuId, &qty, &liStatus, &liSupplierId); err != nil {
			return err
		}
		if liSupplierId != supplierId {
			return fmt.Errorf("access denied: line item belongs to another supplier")
		}
		if liStatus != "REJECTED_DAMAGED" {
			return fmt.Errorf("line item %s is %s, not REJECTED_DAMAGED", req.LineItemID, liStatus)
		}

		// Mark line item as resolved
		newStatus := "WRITE_OFF"
		if req.Resolution == "RETURN_TO_STOCK" {
			newStatus = "RETURNED_TO_STOCK"
		}

		txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("OrderLineItems",
				[]string{"LineItemId", "Status"},
				[]interface{}{req.LineItemID, newStatus},
			),
		})

		// If returning to stock, restore inventory
		if req.Resolution == "RETURN_TO_STOCK" {
			invRow, err := txn.ReadRow(ctx, "SupplierInventory", spanner.Key{skuId}, []string{"QuantityAvailable"})
			if err != nil {
				return fmt.Errorf("inventory record not found for SKU %s", skuId)
			}
			var currentQty int64
			invRow.Columns(&currentQty)
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("SupplierInventory",
					[]string{"ProductId", "QuantityAvailable", "UpdatedAt"},
					[]interface{}{skuId, currentQty + qty, spanner.CommitTimestamp},
				),
			})
		}

		return nil
	})

	if err != nil {
		log.Printf("[RETURNS] Resolve failed for %s: %v", req.LineItemID, err)
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Emit Kafka event for audit trail
	event := map[string]interface{}{
		"type":         ws.EventReturnResolved,
		"line_item_id": req.LineItemID,
		"order_id":     orderId,
		"sku_id":       skuId,
		"quantity":     qty,
		"resolution":   req.Resolution,
		"supplier_id":  supplierId,
		"notes":        req.Notes,
		"timestamp":    time.Now().UnixMilli(),
	}
	eventBytes, _ := json.Marshal(event)
	go func() {
		err := s.Producer.WriteMessages(context.Background(), kafka.Message{
			Key:   []byte(orderId),
			Value: eventBytes,
		})
		if err != nil {
			log.Printf("[RETURNS] Kafka emit failed for %s: %v", req.LineItemID, err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "RESOLVED",
		"line_item_id": req.LineItemID,
		"resolution":   req.Resolution,
	})
}
