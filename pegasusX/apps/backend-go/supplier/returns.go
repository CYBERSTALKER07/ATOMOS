package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// SupplierReturnRow is one quarantine row for the supplier returns queue.
type SupplierReturnRow struct {
	ReturnID        string `json:"return_id"`
	LineItemID      string `json:"line_item_id"`
	OrderID         string `json:"order_id"`
	SkuID           string `json:"sku_id"`
	ProductName     string `json:"product_name"`
	Quantity        int64  `json:"quantity"`
	UnitPrice       int64  `json:"unit_price"`
	Status          string `json:"status"`
	PhysicalStatus  string `json:"physical_status"`
	ReceivedQty     int64  `json:"received_qty"`
	ManifestID      string `json:"manifest_id,omitempty"`
	DriverID        string `json:"driver_id,omitempty"`
	DriverName      string `json:"driver_name,omitempty"`
	Reason          string `json:"reason"`
	DriverNotes     string `json:"driver_notes,omitempty"`
	RetailerID      string `json:"retailer_id"`
	RetailerName    string `json:"retailer_name"`
	CreatedAt       string `json:"created_at"`
}

type resolveReturnRequest struct {
	ReturnID   string `json:"return_id"`
	LineItemID string `json:"line_item_id"`
	Resolution string `json:"resolution"`
	Notes      string `json:"notes"`
}

// HandleReturns serves GET /v1/supplier/returns.
func (s *Service) HandleReturns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "returns_unavailable"})
		return
	}
	supplierID := strings.TrimSpace(s.scopedSupplierID(r))
	if supplierID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	limit := 25
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if statusFilter == "" {
		statusFilter = "PENDING"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sql := `SELECT sr.ReturnId, sr.OrderId, sr.SkuId, COALESCE(p.Name, sr.SkuId), sr.RejectedQty,
	               COALESCE(p.PriceMinor, 0), sr.Status, COALESCE(sr.PhysicalStatus, 'PENDING'),
	               COALESCE(sr.ReceivedQty, 0), COALESCE(sr.ManifestId, ''), COALESCE(sr.DriverId, ''),
	               COALESCE(d.Name, ''), sr.Reason, sr.DriverNotes,
	               o.RetailerId, sr.CreatedAt
	        FROM SupplierReturns sr
	        JOIN Orders o ON sr.OrderId = o.OrderId
	        LEFT JOIN Products p ON p.ProductId = sr.SkuId AND p.SupplierId = o.SupplierId
	        LEFT JOIN Drivers d ON d.DriverId = sr.DriverId
	        WHERE o.SupplierId = @supplier_id
	          AND sr.Status = @status`
	params := map[string]any{
		"supplier_id": supplierID,
		"status":      statusFilter,
	}
	if whID := strings.TrimSpace(auth.EffectiveWarehouseID(r.Context())); whID != "" {
		sql += " AND o.WarehouseId = @warehouse_id"
		params["warehouse_id"] = whID
	}
	sql += fmt.Sprintf(" ORDER BY sr.CreatedAt DESC LIMIT %d OFFSET %d", limit+1, offset)

	iter := s.portalSpanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	rows := make([]SupplierReturnRow, 0, limit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			s.log.ErrorContext(ctx, "supplier returns query failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
			return
		}
		var item SupplierReturnRow
		var driverNotes spanner.NullString
		var createdAt time.Time
		if err := row.Columns(
			&item.ReturnID,
			&item.OrderID,
			&item.SkuID,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.Status,
			&item.PhysicalStatus,
			&item.ReceivedQty,
			&item.ManifestID,
			&item.DriverID,
			&item.DriverName,
			&item.Reason,
			&driverNotes,
			&item.RetailerID,
			&createdAt,
		); err != nil {
			s.log.ErrorContext(ctx, "supplier returns scan failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan_failed"})
			return
		}
		item.LineItemID = item.ReturnID
		item.DriverNotes = driverNotes.StringVal
		item.RetailerName = item.RetailerID
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		rows = append(rows, item)
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":        rows,
		"limit":       limit,
		"offset":      offset,
		"has_more":    hasMore,
		"next_offset": offset + len(rows),
	})
}

// HandleResolveReturn serves POST /v1/supplier/returns/resolve.
func (s *Service) HandleResolveReturn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "returns_unavailable"})
		return
	}
	supplierID := strings.TrimSpace(s.scopedSupplierID(r))
	if supplierID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, ok := readMutationBody(w, r, 32*1024)
	if !ok {
		return
	}
	idemKey, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

	var req resolveReturnRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	returnID := strings.TrimSpace(req.ReturnID)
	if returnID == "" {
		returnID = strings.TrimSpace(req.LineItemID)
	}
	if returnID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "return_id_required"})
		return
	}
	resolution := strings.ToUpper(strings.TrimSpace(req.Resolution))
	if resolution != "WRITE_OFF" && resolution != "RETURN_TO_STOCK" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_resolution"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var orderID, skuID, warehouseID string
	var qty int64
	newStatus := "WRITE_OFF"
	if resolution == "RETURN_TO_STOCK" {
		newStatus = "RETURNED_TO_STOCK"
	}

	_, err := s.portalSpanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &supplierSpannerTxnBuf{}
		row, err := txn.ReadRow(ctx, "SupplierReturns", spanner.Key{returnID},
			[]string{"OrderId", "SkuId", "RejectedQty", "Status", "PhysicalStatus", "ReceivedQty"})
		if err != nil {
			return fmt.Errorf("return not found")
		}
		var status, physicalStatus string
		var receivedQty int64
		if err := row.Columns(&orderID, &skuID, &qty, &status, &physicalStatus, &receivedQty); err != nil {
			return err
		}
		if status != "PENDING" {
			return fmt.Errorf("return %s already resolved", returnID)
		}
		if resolution == "RETURN_TO_STOCK" {
			switch physicalStatus {
			case "RESTOCKED":
				// Gate already credited inventory; supplier ack only.
			case "RECEIVING", "ARRIVED":
				if receivedQty <= 0 {
					return fmt.Errorf("return %s not yet scanned at warehouse gate", returnID)
				}
			default:
				return fmt.Errorf("return %s must be received at warehouse gate before restock", returnID)
			}
		}

		orderRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID},
			[]string{"SupplierId", "WarehouseId"})
		if err != nil {
			return fmt.Errorf("order not found")
		}
		var orderSupplierID string
		var warehouseNull spanner.NullString
		if err := orderRow.Columns(&orderSupplierID, &warehouseNull); err != nil {
			return err
		}
		if strings.TrimSpace(orderSupplierID) != supplierID {
			return fmt.Errorf("access denied")
		}
		if warehouseNull.Valid {
			warehouseID = warehouseNull.StringVal
		}
		if whID := strings.TrimSpace(auth.EffectiveWarehouseID(r.Context())); whID != "" && warehouseID != whID {
			return fmt.Errorf("warehouse scope denied")
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("SupplierReturns", map[string]any{
				"ReturnId":        returnID,
				"Status":          newStatus,
				"ResolvedAt":      spanner.CommitTimestamp,
				"ResolutionNotes": strings.TrimSpace(req.Notes),
			}),
		}

		if resolution == "RETURN_TO_STOCK" && warehouseID != "" && qty > 0 && physicalStatus != "RESTOCKED" {
			invRow, err := txn.ReadRow(ctx, "SupplierInventoryV2",
				spanner.Key{supplierID, warehouseID, skuID},
				[]string{"QuantityOnHand"})
			if err != nil {
				if spanner.ErrCode(err) != codes.NotFound {
					return fmt.Errorf("inventory read failed: %w", err)
				}
				mutations = append(mutations, spanner.InsertMap("SupplierInventoryV2", map[string]any{
					"SupplierId":       supplierID,
					"WarehouseId":      warehouseID,
					"ProductId":        skuID,
					"QuantityOnHand":   qty,
					"QuantityReserved": int64(0),
					"UpdatedAt":        spanner.CommitTimestamp,
				}))
			} else {
				var onHand int64
				if err := invRow.Columns(&onHand); err != nil {
					return err
				}
				mutations = append(mutations, spanner.UpdateMap("SupplierInventoryV2", map[string]any{
					"SupplierId":     supplierID,
					"WarehouseId":    warehouseID,
					"ProductId":      skuID,
					"QuantityOnHand": onHand + qty,
					"UpdatedAt":      spanner.CommitTimestamp,
				}))
			}
		}

		eventPayload := map[string]any{
			"type":        events.EventSupplierReturnResolved,
			"return_id":   returnID,
			"order_id":    orderID,
			"sku_id":      skuID,
			"quantity":    qty,
			"resolution":  resolution,
			"supplier_id": supplierID,
			"notes":       strings.TrimSpace(req.Notes),
			"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, eventPayload); err != nil {
			return err
		}
		for _, e := range buf.events {
			mutations = append(mutations, portalOutboxMutation(e))
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if s.cache != nil {
		s.cache.Invalidate(ctx, supplierCacheKey(supplierID), "supplier:inventory:"+supplierID)
	}

	resp := map[string]any{
		"status":     "RESOLVED",
		"return_id":  returnID,
		"resolution": resolution,
	}
	respBytes, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
	s.storeMutationReplay(r.Context(), idemKey, body, http.StatusOK, respBytes)
}
