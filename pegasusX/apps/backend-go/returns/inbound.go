package returns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// InboundReturnRow is the gate queue read model.
type InboundReturnRow struct {
	ReturnID              string `json:"return_id"`
	OrderID               string `json:"order_id"`
	SkuID                 string `json:"sku_id"`
	ProductName           string `json:"product_name"`
	ImageURL              string `json:"image_url,omitempty"`
	Barcode               string `json:"barcode,omitempty"`
	ExpectedQty           int64  `json:"expected_qty"`
	ReceivedQty           int64  `json:"received_qty"`
	Reason                string `json:"reason"`
	DriverNotes           string `json:"driver_notes,omitempty"`
	PhysicalStatus        string `json:"physical_status"`
	ManifestID            string `json:"manifest_id,omitempty"`
	DriverID              string `json:"driver_id,omitempty"`
	DriverName            string `json:"driver_name,omitempty"`
	SuggestedDisposition  string `json:"suggested_disposition"`
	CreatedAt             string `json:"created_at"`
}

// HandleInboundList serves GET /v1/returns/inbound.
func (s *Service) HandleInboundList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "returns_unavailable"})
		return
	}
	warehouseID, ok := s.resolveGateWarehouseID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "warehouse_scope_required"})
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	manifestID := strings.TrimSpace(r.URL.Query().Get("manifest_id"))
	driverID := strings.TrimSpace(r.URL.Query().Get("driver_id"))
	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("physical_status")))
	if statusFilter == "" {
		statusFilter = PhysicalArrived
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if statusFilter == PhysicalArrived {
		_ = s.MarkArrivedForWarehouse(ctx, warehouseID)
	}

	sql := `SELECT sr.ReturnId, sr.OrderId, sr.SkuId, COALESCE(p.Name, sr.SkuId),
	               COALESCE(p.ImageURL, ''), COALESCE(p.Barcode, ''),
	               COALESCE(sr.ExpectedQty, sr.RejectedQty), sr.ReceivedQty,
	               sr.Reason, sr.DriverNotes, sr.PhysicalStatus,
	               COALESCE(sr.ManifestId, ''), COALESCE(sr.DriverId, ''),
	               COALESCE(d.Name, ''), sr.CreatedAt
	        FROM SupplierReturns sr
	        JOIN Orders o ON sr.OrderId = o.OrderId
	        LEFT JOIN Products p ON p.ProductId = sr.SkuId AND p.SupplierId = o.SupplierId
	        LEFT JOIN Drivers d ON d.DriverId = sr.DriverId
	        WHERE sr.WarehouseId = @warehouse_id
	          AND sr.PhysicalStatus = @physical_status
	          AND sr.Status = @fin_pending`
	params := map[string]any{
		"warehouse_id":    warehouseID,
		"physical_status": statusFilter,
		"fin_pending":     FinancialPending,
	}
	if manifestID != "" {
		sql += " AND sr.ManifestId = @manifest_id"
		params["manifest_id"] = manifestID
	}
	if driverID != "" {
		sql += " AND sr.DriverId = @driver_id"
		params["driver_id"] = driverID
	}
	sql += fmt.Sprintf(" ORDER BY sr.CreatedAt ASC LIMIT %d", limit)

	rows, err := s.queryInboundRows(ctx, sql, params)
	if err != nil {
		s.log.ErrorContext(ctx, "inbound returns query failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rows, "warehouse_id": warehouseID})
}

// HandleBarcodeLookup serves GET /v1/catalog/barcode/{ean}.
func (s *Service) HandleBarcodeLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "catalog_unavailable"})
		return
	}
	raw := strings.TrimSpace(chi.URLParam(r, "ean"))
	if raw == "" {
		raw = strings.TrimPrefix(r.URL.Path, "/v1/catalog/barcode/")
	}
	barcode, err := NormalizeBarcode(raw)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	supplierID := ""
	if claims, ok := auth.FromContext(r.Context()); ok {
		supplierID = strings.TrimSpace(claims.SupplierID)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sql := `SELECT ProductId, Name, ImageURL, Barcode, PriceMinor, Currency, SupplierId
	        FROM Products WHERE Barcode = @barcode AND IsActive = TRUE`
	params := map[string]any{"barcode": barcode}
	if supplierID != "" {
		sql += " AND SupplierId = @supplier_id"
		params["supplier_id"] = supplierID
	}
	sql += " LIMIT 1"

	iter := s.spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup_failed"})
		return
	}
	var productID, name, imageURL, bc, currency, prodSupplier string
	var price int64
	if err := row.Columns(&productID, &name, &imageURL, &bc, &price, &currency, &prodSupplier); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"product_id":  productID,
		"sku_id":      productID,
		"name":        name,
		"image_url":   imageURL,
		"barcode":     bc,
		"price_minor": price,
		"currency":    currency,
		"supplier_id": prodSupplier,
	})
}

type startSessionRequest struct {
	ManifestID string `json:"manifest_id"`
	DriverID   string `json:"driver_id"`
}

// HandleStartReceiveSession serves POST /v1/returns/inbound/sessions.
func (s *Service) HandleStartReceiveSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "returns_unavailable"})
		return
	}
	warehouseID, ok := s.resolveGateWarehouseID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req startSessionRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	sessionID := s.newID()
	role := string(claims.Role)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_, err := s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("ReturnReceiveSessions", map[string]any{
			"SessionId":    sessionID,
			"WarehouseId":  warehouseID,
			"ManifestId":   nullableString(req.ManifestID),
			"DriverId":     nullableString(req.DriverID),
			"OperatorId":   claims.Subject,
			"OperatorRole": role,
			"Status":       SessionStatusOpen,
			"StartedAt":    spanner.CommitTimestamp,
		})})
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "session_create_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session_id": sessionID, "status": SessionStatusOpen})
}

type scanRequest struct {
	Barcode   string `json:"barcode"`
	Qty       int64  `json:"qty"`
	ReturnID  string `json:"return_id"`
	SessionID string `json:"session_id"`
	ManifestID string `json:"manifest_id"`
}

type scanResponse struct {
	Matched    bool             `json:"matched"`
	ReturnID   string           `json:"return_id,omitempty"`
	Product    map[string]any   `json:"product,omitempty"`
	Line       *InboundReturnRow  `json:"line,omitempty"`
	Variance   bool             `json:"variance"`
	Message    string           `json:"message,omitempty"`
}

// HandleInboundScan serves POST /v1/returns/inbound/scan.
func (s *Service) HandleInboundScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "returns_unavailable"})
		return
	}
	warehouseID, ok := s.resolveGateWarehouseID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req scanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	qty := req.Qty
	if qty <= 0 {
		qty = 1
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	returnID := strings.TrimSpace(req.ReturnID)
	var skuID string
	if returnID == "" {
		barcode, err := NormalizeBarcode(req.Barcode)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		skuID, err = s.lookupSKUByBarcode(ctx, barcode, warehouseID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "product_not_found"})
			return
		}
		returnID, err = s.matchReturnLine(ctx, warehouseID, skuID, req.ManifestID, req.SessionID)
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]any{
				"matched": false,
				"message": err.Error(),
			})
			return
		}
	}

	var resp scanResponse
	_, err := s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "SupplierReturns", spanner.Key{returnID},
			[]string{"SkuId", "ExpectedQty", "RejectedQty", "ReceivedQty", "PhysicalStatus", "WarehouseId", "Reason"})
		if err != nil {
			return fmt.Errorf("return_not_found")
		}
		var sku, reason, physical string
		var expectedNull spanner.NullInt64
		var rejected, received int64
		var whNull spanner.NullString
		if err := row.Columns(&sku, &expectedNull, &rejected, &received, &physical, &whNull, &reason); err != nil {
			return err
		}
		if whNull.Valid && whNull.StringVal != warehouseID {
			return fmt.Errorf("warehouse_mismatch")
		}
		expected := rejected
		if expectedNull.Valid {
			expected = expectedNull.Int64
		}
		newReceived := received + qty
		newPhysical := PhysicalReceiving
		if newReceived >= expected {
			newPhysical = PhysicalReceiving
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("SupplierReturns", map[string]any{
			"ReturnId":         returnID,
			"ReceivedQty":      newReceived,
			"PhysicalStatus":   newPhysical,
			"ReceiveSessionId": nullableString(req.SessionID),
			"ReceivedBy":       claims.Subject,
		})}); err != nil {
			return err
		}
		resp = scanResponse{
			Matched:  true,
			ReturnID: returnID,
			Variance: newReceived != expected,
			Message:  fmt.Sprintf("scanned %d of %d", newReceived, expected),
		}
		if newReceived > expected {
			resp.Message = fmt.Sprintf("variance: scanned %d exceeds expected %d", newReceived, expected)
		}
		return nil
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

type confirmLine struct {
	ReturnID    string `json:"return_id"`
	Disposition string `json:"disposition"`
	Qty         int64  `json:"qty"`
	Notes       string `json:"notes"`
}

type confirmRequest struct {
	SessionID string        `json:"session_id"`
	Lines     []confirmLine `json:"lines"`
}

// HandleInboundConfirm serves POST /v1/returns/inbound/confirm.
func (s *Service) HandleInboundConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "returns_unavailable"})
		return
	}
	warehouseID, ok := s.resolveGateWarehouseID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req confirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if len(req.Lines) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lines_required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	confirmed := make([]string, 0, len(req.Lines))

	_, err := s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		for _, line := range req.Lines {
			returnID := strings.TrimSpace(line.ReturnID)
			if returnID == "" {
				continue
			}
			disposition := strings.ToUpper(strings.TrimSpace(line.Disposition))
			if disposition != DispositionRestock && disposition != DispositionWriteOff {
				return fmt.Errorf("invalid disposition for %s", returnID)
			}
			qty := line.Qty
			if qty <= 0 {
				qty = 1
			}

			row, err := txn.ReadRow(ctx, "SupplierReturns", spanner.Key{returnID},
				[]string{"OrderId", "SkuId", "RejectedQty", "ReceivedQty", "Status", "PhysicalStatus", "WarehouseId", "Reason"})
			if err != nil {
				return fmt.Errorf("return %s not found", returnID)
			}
			var orderID, skuID, finStatus, physical, reason string
			var rejected, received int64
			var whNull spanner.NullString
			if err := row.Columns(&orderID, &skuID, &rejected, &received, &finStatus, &physical, &whNull, &reason); err != nil {
				return err
			}
			if finStatus != FinancialPending {
				return fmt.Errorf("return %s already resolved", returnID)
			}
			if whNull.Valid && whNull.StringVal != warehouseID {
				return fmt.Errorf("warehouse mismatch for %s", returnID)
			}
			if received <= 0 {
				received = qty
			}
			creditQty := received
			if line.Qty > 0 {
				creditQty = line.Qty
			}

			newPhysical := PhysicalWrittenOff
			newFin := FinancialWriteOff
			if disposition == DispositionRestock {
				newPhysical = PhysicalRestocked
				newFin = FinancialReturnedToStock
			}

			orderRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"SupplierId"})
			if err != nil {
				return err
			}
			var supplierID string
			if err := orderRow.Columns(&supplierID); err != nil {
				return err
			}

			mutations := []*spanner.Mutation{spanner.UpdateMap("SupplierReturns", map[string]any{
				"ReturnId":        returnID,
				"PhysicalStatus":  newPhysical,
				"Status":          newFin,
				"ReceivedQty":     received,
				"ReceivedAt":      spanner.CommitTimestamp,
				"ReceivedBy":      claims.Subject,
				"ResolvedAt":      spanner.CommitTimestamp,
				"ResolutionNotes": strings.TrimSpace(line.Notes),
			})}

			if disposition == DispositionRestock && creditQty > 0 {
				if err := inventory.CreditSupplierInventoryV2InTxn(ctx, txn, supplierID, warehouseID, skuID, creditQty); err != nil {
					return err
				}
			}

			payload, err := json.Marshal(map[string]any{
				"type":            events.EventReturnReceivedAtWarehouse,
				"return_id":       returnID,
				"order_id":        orderID,
				"sku_id":          skuID,
				"quantity":        creditQty,
				"disposition":     disposition,
				"warehouse_id":    warehouseID,
				"supplier_id":     supplierID,
				"operator_id":     claims.Subject,
				"physical_status": newPhysical,
				"timestamp":       s.now().UTC().Format(time.RFC3339Nano),
			})
			if err != nil {
				return err
			}
			eventID := returnID
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
				"EventId":       eventID,
				"AggregateType": events.AggregateOrder,
				"AggregateId":   orderID,
				"TopicName":     events.TopicMain,
				"Payload":       payload,
				"CreatedAt":     s.now().UTC(),
				"PublishedAt":   nil,
			}))
			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}
			confirmed = append(confirmed, returnID)
		}
		if sessionID := strings.TrimSpace(req.SessionID); sessionID != "" {
			if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("ReturnReceiveSessions", map[string]any{
				"SessionId":   sessionID,
				"Status":      SessionStatusCompleted,
				"CompletedAt": spanner.CommitTimestamp,
			})}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "warehouse:inventory:"+warehouseID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "confirmed",
		"return_ids": confirmed,
	})
}

// HandleReturnsHistory serves GET /v1/returns/history.
func (s *Service) HandleReturnsHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "returns_unavailable"})
		return
	}

	limit := 25
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			offset = n
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sql := `SELECT sr.ReturnId, sr.OrderId, sr.SkuId, COALESCE(p.Name, sr.SkuId),
	               COALESCE(p.ImageURL, ''), COALESCE(p.Barcode, ''),
	               COALESCE(sr.ExpectedQty, sr.RejectedQty), sr.ReceivedQty,
	               sr.Reason, sr.DriverNotes, sr.PhysicalStatus, sr.Status,
	               COALESCE(sr.ManifestId, ''), COALESCE(sr.DriverId, ''),
	               COALESCE(d.Name, ''), sr.ReceivedAt, sr.ResolutionNotes
	        FROM SupplierReturns sr
	        JOIN Orders o ON sr.OrderId = o.OrderId
	        LEFT JOIN Products p ON p.ProductId = sr.SkuId AND p.SupplierId = o.SupplierId
	        LEFT JOIN Drivers d ON d.DriverId = sr.DriverId
	        WHERE sr.PhysicalStatus IN (@restocked, @written_off)`
	params := map[string]any{
		"restocked":  PhysicalRestocked,
		"written_off": PhysicalWrittenOff,
	}

	if whID, ok := s.resolveGateWarehouseID(r); ok && whID != "" {
		sql += " AND sr.WarehouseId = @warehouse_id"
		params["warehouse_id"] = whID
	} else if claims, ok := auth.FromContext(r.Context()); ok {
		if supplierID := strings.TrimSpace(claims.SupplierID); supplierID != "" {
			sql += " AND o.SupplierId = @supplier_id"
			params["supplier_id"] = supplierID
		}
	}
	sql += fmt.Sprintf(" ORDER BY sr.ReceivedAt DESC LIMIT %d OFFSET %d", limit+1, offset)

	iter := s.spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	type historyRow struct {
		InboundReturnRow
		FinancialStatus string `json:"financial_status"`
		ReceivedAt      string `json:"received_at,omitempty"`
		ResolutionNotes string `json:"resolution_notes,omitempty"`
	}
	rows := make([]historyRow, 0, limit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
			return
		}
		var item historyRow
		var driverNotes spanner.NullString
		var receivedAt spanner.NullTime
		var resolutionNotes spanner.NullString
		if err := row.Columns(
			&item.ReturnID, &item.OrderID, &item.SkuID, &item.ProductName,
			&item.ImageURL, &item.Barcode, &item.ExpectedQty, &item.ReceivedQty,
			&item.Reason, &driverNotes, &item.PhysicalStatus, &item.FinancialStatus,
			&item.ManifestID, &item.DriverID, &item.DriverName,
			&receivedAt, &resolutionNotes,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan_failed"})
			return
		}
		item.DriverNotes = driverNotes.StringVal
		item.SuggestedDisposition = SuggestedDisposition(item.Reason)
		if receivedAt.Valid {
			item.ReceivedAt = receivedAt.Time.UTC().Format(time.RFC3339)
		}
		item.ResolutionNotes = resolutionNotes.StringVal
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

// DriverReturnGoodsLine is one SKU on the truck for depot summary.
type DriverReturnGoodsLine struct {
	ReturnID    string `json:"return_id"`
	OrderID     string `json:"order_id"`
	SkuID       string `json:"sku_id"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
	Reason      string `json:"reason"`
}

// HandleDriverReturnGoods serves GET /v1/driver/return-goods.
func (s *Service) HandleDriverReturnGoods(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.spanner == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "total_units": 0})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	driverID := strings.TrimSpace(claims.Subject)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	stmt := spanner.Statement{
		SQL: `SELECT sr.ReturnId, sr.OrderId, sr.SkuId, COALESCE(p.Name, sr.SkuId),
		             COALESCE(sr.ExpectedQty, sr.RejectedQty), sr.Reason
		      FROM SupplierReturns sr
		      LEFT JOIN Orders o ON sr.OrderId = o.OrderId
		      LEFT JOIN Products p ON p.ProductId = sr.SkuId AND p.SupplierId = o.SupplierId
		      WHERE sr.DriverId = @driver_id
		        AND sr.PhysicalStatus IN (@pending, @on_truck, @arrived, @receiving)
		        AND sr.Status = @fin_pending
		      ORDER BY sr.CreatedAt ASC`,
		Params: map[string]any{
			"driver_id":  driverID,
			"pending":    PhysicalPending,
			"on_truck":   PhysicalOnTruck,
			"arrived":    PhysicalArrived,
			"receiving":  PhysicalReceiving,
			"fin_pending": FinancialPending,
		},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	items := make([]DriverReturnGoodsLine, 0, 8)
	var total int64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
			return
		}
		var item DriverReturnGoodsLine
		if err := row.Columns(&item.ReturnID, &item.OrderID, &item.SkuID, &item.ProductName, &item.Quantity, &item.Reason); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan_failed"})
			return
		}
		total += item.Quantity
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":       items,
		"total_units": total,
		"line_count":  len(items),
	})
}

func (s *Service) resolveGateWarehouseID(r *http.Request) (string, bool) {
	if wh := strings.TrimSpace(auth.EffectiveWarehouseID(r.Context())); wh != "" {
		return wh, true
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return "", false
	}
	if claims.Role == auth.RolePayload && strings.TrimSpace(claims.HomeNodeID) != "" {
		return strings.TrimSpace(claims.HomeNodeID), true
	}
	if claims.Role == auth.RoleWarehouseAdmin && strings.TrimSpace(claims.HomeNodeID) != "" {
		return strings.TrimSpace(claims.HomeNodeID), true
	}
	return "", false
}

func (s *Service) queryInboundRows(ctx context.Context, sql string, params map[string]any) ([]InboundReturnRow, error) {
	iter := s.spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	rows := make([]InboundReturnRow, 0, 16)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
		var item InboundReturnRow
		var driverNotes spanner.NullString
		var createdAt time.Time
		if err := row.Columns(
			&item.ReturnID, &item.OrderID, &item.SkuID, &item.ProductName,
			&item.ImageURL, &item.Barcode, &item.ExpectedQty, &item.ReceivedQty,
			&item.Reason, &driverNotes, &item.PhysicalStatus,
			&item.ManifestID, &item.DriverID, &item.DriverName, &createdAt,
		); err != nil {
			return nil, err
		}
		item.DriverNotes = driverNotes.StringVal
		item.SuggestedDisposition = SuggestedDisposition(item.Reason)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		rows = append(rows, item)
	}
}

func (s *Service) lookupSKUByBarcode(ctx context.Context, barcode, warehouseID string) (string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT p.ProductId FROM Products p
		      JOIN Warehouses w ON w.SupplierId = p.SupplierId
		      WHERE p.Barcode = @barcode AND w.WarehouseId = @warehouse_id AND p.IsActive = TRUE
		      LIMIT 1`,
		Params: map[string]any{"barcode": barcode, "warehouse_id": warehouseID},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", fmt.Errorf("product_not_found")
	}
	if err != nil {
		return "", err
	}
	var sku string
	if err := row.Columns(&sku); err != nil {
		return "", err
	}
	return sku, nil
}

func (s *Service) matchReturnLine(ctx context.Context, warehouseID, skuID, manifestID, sessionID string) (string, error) {
	sql := `SELECT ReturnId FROM SupplierReturns
	        WHERE WarehouseId = @warehouse_id AND SkuId = @sku_id
	          AND PhysicalStatus IN (@arrived, @receiving, @on_truck)
	          AND Status = @fin_pending`
	params := map[string]any{
		"warehouse_id": warehouseID,
		"sku_id":       skuID,
		"arrived":      PhysicalArrived,
		"receiving":    PhysicalReceiving,
		"on_truck":     PhysicalOnTruck,
		"fin_pending":  FinancialPending,
	}
	if manifestID != "" {
		sql += " AND ManifestId = @manifest_id"
		params["manifest_id"] = manifestID
	}
	sql += " ORDER BY CreatedAt ASC LIMIT 1"
	iter := s.spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", fmt.Errorf("no_matching_return_line")
	}
	if err != nil {
		return "", err
	}
	var returnID string
	if err := row.Columns(&returnID); err != nil {
		return "", err
	}
	_ = sessionID
	return returnID, nil
}

// MarkArrivedOnTruck promotes ON_TRUCK returns to ARRIVED when inbound list is opened without geofence.
func (s *Service) MarkArrivedForWarehouse(ctx context.Context, warehouseID string) error {
	if s.spanner == nil || warehouseID == "" {
		return nil
	}
	_, err := s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `UPDATE SupplierReturns SET PhysicalStatus = @arrived
			      WHERE WarehouseId = @warehouse_id AND PhysicalStatus = @on_truck`,
			Params: map[string]any{
				"arrived":      PhysicalArrived,
				"warehouse_id": warehouseID,
				"on_truck":     PhysicalOnTruck,
			},
		}
		_, err := txn.Update(ctx, stmt)
		if err != nil && spanner.ErrCode(err) != codes.OK {
			return err
		}
		return nil
	})
	return err
}
