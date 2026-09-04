package order

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

const (
	cacheKeyPaymentBypassPrefix = "payment_bypass:"
	cacheKeyEarlyCompletePrefix = "early_complete:"
	paymentBypassCacheTTL       = 24 * time.Hour
	earlyCompleteCacheTTL       = 6 * time.Hour
)

type paymentBypassRecord struct {
	Token      string `json:"token"`
	SupplierID string `json:"supplier_id"`
	IssuedBy   string `json:"issued_by"`
	Reason     string `json:"reason,omitempty"`
	IssuedAt   string `json:"issued_at"`
}

type earlyCompleteRecord struct {
	RouteID     string   `json:"route_id,omitempty"`
	OrderIDs    []string `json:"order_ids"`
	WarehouseID string   `json:"warehouse_id,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	Note        string   `json:"note,omitempty"`
	DriverID    string   `json:"driver_id"`
}

// HandleIssuePaymentBypass serves POST /v1/supplier/orders/payment-bypass.
func (s *Service) HandleIssuePaymentBypass(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleAdmin && claims.Role != auth.RoleWarehouseAdmin) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil || s.cache == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payment_bypass_unavailable"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req struct {
		OrderID string `json:"order_id"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id required"})
		return
	}

	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok {
		supplierID = s.resolveSupplierScope(r.Context())
	}
	ctx := r.Context()
	now := s.now()

	var status, warehouseID string
	err = s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT Status, WarehouseId FROM Orders WHERE OrderId = @oid AND SupplierId = @sid`,
		Params: map[string]any{"oid": req.OrderID, "sid": supplierID},
	}).Do(func(row *spanner.Row) error {
		var wCol spanner.NullString
		err := row.Columns(&status, &wCol)
		if wCol.Valid {
			warehouseID = wCol.StringVal
		}
		return err
	})
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if status != string(StatusAwaitingPayment) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "order_must_be_awaiting_payment"})
		return
	}
	if claims.Role == auth.RoleWarehouseAdmin && claims.HomeNodeID != warehouseID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "order_not_in_your_warehouse"})
		return
	}

	token, err := generatePaymentBypassToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token_generation_failed"})
		return
	}

	rec := paymentBypassRecord{
		Token:      token,
		SupplierID: supplierID,
		IssuedBy:   strings.TrimSpace(claims.Subject),
		Reason:     strings.TrimSpace(req.Reason),
		IssuedAt:   now.Format(time.RFC3339Nano),
	}
	raw, _ := json.Marshal(rec)
	if err := s.cache.Set(ctx, cacheKeyPaymentBypassPrefix+req.OrderID, raw, paymentBypassCacheTTL); err != nil {
		s.log.ErrorContext(ctx, "payment bypass cache set failed", "order_id", req.OrderID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cache_write_failed"})
		return
	}

	s.log.InfoContext(ctx, "payment bypass issued", "order_id", req.OrderID, "supplier_id", supplierID)
	resp := map[string]any{
		"status":       "bypass_issued",
		"bypass_token": token,
		"order_id":     req.OrderID,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleConfirmPaymentBypass serves POST /v1/delivery/confirm-payment-bypass.
func (s *Service) HandleConfirmPaymentBypass(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil || s.cache == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payment_bypass_unavailable"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req struct {
		OrderID     string `json:"order_id"`
		BypassToken string `json:"bypass_token"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.BypassToken = strings.TrimSpace(req.BypassToken)
	if req.OrderID == "" || req.BypassToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id and bypass_token required"})
		return
	}

	ctx := r.Context()
	raw, found, err := s.cache.Get(ctx, cacheKeyPaymentBypassPrefix+req.OrderID)
	if err != nil || !found || len(raw) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "bypass_not_found"})
		return
	}
	var rec paymentBypassRecord
	if err := json.Unmarshal(raw, &rec); err != nil || rec.Token != req.BypassToken {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_bypass_token"})
		return
	}

	driverID := strings.TrimSpace(claims.Subject)
	now := s.now()
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
			[]string{"Status", "Version", "DriverId", "SupplierId", "RetailerId", "WarehouseId", "TotalMinor", "Currency"})
		if err != nil {
			return ErrOrderNotFound
		}
		var status, supplierID string
		var version, totalMinor int64
		var driverCol, retailerCol, warehouseCol, currencyCol spanner.NullString
		if err := row.Columns(&status, &version, &driverCol, &supplierID, &retailerCol, &warehouseCol, &totalMinor, &currencyCol); err != nil {
			return err
		}
		if status != string(StatusAwaitingPayment) {
			return fmt.Errorf("order must be AWAITING_PAYMENT, got %s", status)
		}
		if !driverCol.Valid || driverCol.StringVal != driverID {
			return ErrOrderForbidden
		}
		if rec.SupplierID != "" && supplierID != rec.SupplierID {
			return ErrOrderForbidden
		}

		if err := ValidateStatusTransition(status, string(StatusFiscalizing), TransitionOpts{
			Actor:  driverID,
			Reason: "payment_bypass_confirmed",
		}); err != nil {
			return err
		}

		attemptID := ""
		if s.newID != nil {
			attemptID = s.newID()
		}
		if attemptID == "" {
			attemptID = defaultOrderID()
		}
		retailerID := retailerCol.StringVal
		currency := currencyCol.StringVal
		if currency == "" {
			currency = "UZS"
		}

		fiscalRow := FiscalReceiptRow{
			OrderID:       req.OrderID,
			AttemptID:     attemptID,
			SupplierID:    supplierID,
			RetailerID:    retailerID,
			Provider:      s.ProviderName(),
			Status:        FiscalAttemptPending,
			AmountMinor:   totalMinor,
			Currency:      currency,
			PaymentMethod: "BYPASS",
			ReasonCode:    "PAYMENT_BYPASS",
			ActorID:       driverID,
			CreatedAt:     now.UTC(),
			UpdatedAt:     now.UTC(),
		}

		orderRecord := Order{
			OrderID:               req.OrderID,
			SupplierID:            supplierID,
			RetailerID:            retailerID,
			WarehouseID:           warehouseCol.StringVal,
			DriverID:              driverID,
			Status:                StatusFiscalizing,
			TotalMinor:            totalMinor,
			Currency:              currency,
			FiscalStatus:          FiscalStatusPending,
			LatestFiscalAttemptID: attemptID,
			Version:               version + 1,
			UpdatedAt:             now.UTC(),
		}

		buf := &spannerTxnBuffer{}
		if err := emitOrderStatusChanged(ctx, buf, orderStatusEmitParams{
			Claims:         claims,
			Order:          orderRecord,
			PreviousStatus: Status(status),
			Reason:         "payment_bypass_confirmed",
			ActorID:        driverID,
		}); err != nil {
			return err
		}

		if err := emitPaymentCaptureFiscal(ctx, buf, orderRecord, fiscalRow, "BYPASS"); err != nil {
			return err
		}

		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, req.OrderID, events.TopicMain, map[string]any{
			"type":      "PAYMENT_BYPASS_CONFIRMED",
			"order_id":  req.OrderID,
			"driver_id": driverID,
			"timestamp": now.Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":               req.OrderID,
				"Status":                string(StatusFiscalizing),
				"FiscalStatus":          FiscalStatusPending,
				"LatestFiscalAttemptId": attemptID,
				"Version":               version + 1,
				"UpdatedAt":             now.UTC(),
			}),
			spanner.InsertMap("OrderFiscalReceipts", map[string]any{
				"OrderId":             req.OrderID,
				"AttemptId":           attemptID,
				"SupplierId":          supplierID,
				"RetailerId":          nullableString(retailerID),
				"Provider":            s.ProviderName(),
				"Status":              FiscalAttemptPending,
				"FiscalReceiptId":     spanner.NullString{},
				"FiscalQR":            spanner.NullString{},
				"AmountMinor":         totalMinor,
				"Currency":            currency,
				"PaymentMethod":       "BYPASS",
				"ProviderPayloadJSON": []byte(nil),
				"ErrorCode":           spanner.NullString{},
				"ErrorMessage":        spanner.NullString{},
				"ReasonCode":          "PAYMENT_BYPASS",
				"ActorId":             driverID,
				"TraceId":             spanner.NullString{},
				"CreatedAt":           now.UTC(),
				"UpdatedAt":           now.UTC(),
			}),
		}
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		for _, a := range buf.audits {
			mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
			return
		}
		if errors.Is(err, ErrOrderForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "order_forbidden"})
			return
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	s.cache.Invalidate(ctx, cacheKeyPaymentBypassPrefix+req.OrderID)
	s.invalidateOrderCache(ctx, req.OrderID)
	resp := map[string]any{"status": string(StatusFiscalizing), "order_id": req.OrderID}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleApproveEarlyComplete serves POST /v1/supplier/route/approve-early-complete.
func (s *Service) HandleApproveEarlyComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleAdmin && claims.Role != auth.RoleWarehouseAdmin) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil || s.cache == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "early_complete_unavailable"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req struct {
		DriverID       string     `json:"driver_id"`
		Action         string     `json:"action"` // CANCEL | RESCHEDULE
		NewWindowStart *time.Time `json:"newWindowStart,omitempty"`
		NewWindowEnd   *time.Time `json:"newWindowEnd,omitempty"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.DriverID = strings.TrimSpace(req.DriverID)
	if req.DriverID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "driver_id required"})
		return
	}
	if req.Action == "" {
		req.Action = "CANCEL"
	}
	if req.Action == "RESCHEDULE" && (req.NewWindowStart == nil || req.NewWindowEnd == nil) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "newWindowStart and newWindowEnd required for RESCHEDULE"})
		return
	}

	ctx := r.Context()
	raw, found, err := s.cache.Get(ctx, cacheKeyEarlyCompletePrefix+req.DriverID)
	if err != nil || !found || len(raw) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_pending_early_complete"})
		return
	}
	var rec earlyCompleteRecord
	if err := json.Unmarshal(raw, &rec); err != nil || len(rec.OrderIDs) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_pending_early_complete"})
		return
	}
	if claims.Role == auth.RoleWarehouseAdmin && claims.HomeNodeID != rec.WarehouseID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "driver_not_in_your_warehouse"})
		return
	}

	supplierID, _ := auth.ResolveSupplierID(ctx)
	if supplierID == "" {
		supplierID = s.resolveSupplierScope(ctx)
	}
	now := s.now()
	approved := 0

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		for _, orderID := range rec.OrderIDs {
			orderID = strings.TrimSpace(orderID)
			if orderID == "" {
				continue
			}
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID},
				[]string{"Status", "Version", "SupplierId", "LineItemsJson", "WarehouseId", "Source"})
			if err != nil {
				continue
			}
			var status, oidSupplier, warehouseID, source string
			var version int64
			var lineItemsRaw []byte
			if err := row.Columns(&status, &version, &oidSupplier, &lineItemsRaw, &warehouseID, &source); err != nil {
				continue
			}
			if oidSupplier != supplierID {
				continue
			}
			if isTerminalStatus(Status(status)) {
				continue
			}

			newStatus := StatusCancelled
			if req.Action == "RESCHEDULE" {
				newStatus = StatusPending
			}

			if err := ValidateStatusTransition(status, string(newStatus), TransitionOpts{
				Actor:  claims.Subject,
				Reason: "early_route_complete_approved",
			}); err != nil {
				continue
			}

			if newStatus == StatusCancelled {
				if err := ReleaseReservationsFromOrderFields(ctx, txn, supplierID, warehouseID, source, lineItemsRaw); err != nil {
					return err
				}
			}

			buf := &spannerTxnBuffer{}
			_ = outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.OrderEvent{
				BaseEvent:      events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: now.Format(time.RFC3339Nano)},
				OrderID:        orderID,
				SupplierID:     supplierID,
				PreviousStatus: status,
				Status:         string(newStatus),
				Reason:         "early_route_complete_" + strings.ToLower(req.Action),
				ActorRole:      string(claims.Role),
				ActorID:        claims.Subject,
				Version:        version + 1,
			})

			upd := map[string]any{
				"OrderId":   orderID,
				"Status":    string(newStatus),
				"Version":   version + 1,
				"UpdatedAt": now.UTC(),
			}
			if req.Action == "RESCHEDULE" {
				upd["ReceivingWindowOpen"] = *req.NewWindowStart
				upd["ReceivingWindowClose"] = *req.NewWindowEnd
				upd["DriverId"] = spanner.NullString{} // unassign driver
			}

			mutations := []*spanner.Mutation{
				spanner.UpdateMap("Orders", upd),
			}
			for _, e := range buf.events {
				mutations = append(mutations, outboxMutation(e))
			}
			for _, a := range buf.audits {
				mutations = append(mutations, spanner.InsertMap("AuditLog", a.AuditRowMap()))
			}
			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}
			approved++
		}
		return nil
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "early_complete_failed"})
		return
	}
	s.cache.Invalidate(ctx, cacheKeyEarlyCompletePrefix+req.DriverID)
	resp := map[string]any{
		"status":          "approved",
		"driver_id":       req.DriverID,
		"orders_approved": approved,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// StoreEarlyCompleteRequest caches a driver early-complete request for supplier approval.
func (s *Service) StoreEarlyCompleteRequest(ctx context.Context, rec earlyCompleteRecord) error {
	if s.cache == nil {
		return errors.New("cache_unavailable")
	}
	rec.DriverID = strings.TrimSpace(rec.DriverID)
	if rec.DriverID == "" {
		return errors.New("driver_id required")
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return s.cache.Set(ctx, cacheKeyEarlyCompletePrefix+rec.DriverID, raw, earlyCompleteCacheTTL)
}

// HandleRequestEarlyComplete serves POST /v1/fleet/route/request-early-complete.
func (s *Service) HandleRequestEarlyComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil || s.cache == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "early_complete_unavailable"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req struct {
		RouteID string `json:"route_id"`
		Reason  string `json:"reason"`
		Note    string `json:"note"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	driverID := strings.TrimSpace(claims.Subject)
	ctx := r.Context()
	var orderIDs []string
	var warehouseID string
	err = s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, WarehouseId FROM Orders
		      WHERE DriverId = @did AND Status NOT IN ('COMPLETED', 'CANCELLED')
		      ORDER BY CreatedAt DESC`,
		Params: map[string]any{"did": driverID},
	}).Do(func(row *spanner.Row) error {
		var oid string
		var wCol spanner.NullString
		if err := row.Columns(&oid, &wCol); err != nil {
			return err
		}
		if wCol.Valid && warehouseID == "" {
			warehouseID = wCol.StringVal
		}
		orderIDs = append(orderIDs, oid)
		return nil
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_route_orders_failed"})
		return
	}
	if len(orderIDs) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_active_orders"})
		return
	}

	rec := earlyCompleteRecord{
		RouteID:     req.RouteID,
		OrderIDs:    orderIDs,
		WarehouseID: warehouseID,
		Reason:      strings.TrimSpace(req.Reason),
		Note:        strings.TrimSpace(req.Note),
		DriverID:    driverID,
	}
	if err := s.StoreEarlyCompleteRequest(ctx, rec); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cache_write_failed"})
		return
	}

	buf := &spannerTxnBuffer{}
	_ = outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, warehouseID, events.TopicMain, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: "order.early_complete.requested", Timestamp: s.now().Format(time.RFC3339Nano)},
		DriverID:   driverID,
		Reason:     req.Reason,
	})
	if len(buf.events) > 0 {
		var mutations []*spanner.Mutation
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		_, _ = s.spannerClient.Apply(ctx, mutations)
	}

	resp := map[string]any{
		"status":      "REQUESTED",
		"order_count": len(orderIDs),
		"order_ids":   orderIDs,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleGetEarlyCompleteRequest serves GET /v1/warehouse/ops/orders/early-complete/{driverID}.
func (s *Service) HandleGetEarlyCompleteRequest(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleAdmin && claims.Role != auth.RoleWarehouseAdmin) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	
	driverID := chi.URLParam(r, "driverID")
	if driverID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "driver_id required"})
		return
	}
	
	raw, found, err := s.cache.Get(r.Context(), cacheKeyEarlyCompletePrefix+driverID)
	if err != nil || !found || len(raw) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	
	var rec earlyCompleteRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	
	if claims.Role == auth.RoleWarehouseAdmin && claims.HomeNodeID != rec.WarehouseID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "driver_not_in_your_warehouse"})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Write(raw)
}

// HandleListEarlyCompleteRequests serves GET /v1/warehouse/ops/orders/early-complete.
func (s *Service) HandleListEarlyCompleteRequests(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleAdmin && claims.Role != auth.RoleWarehouseAdmin) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// In a real production environment, you should avoid SCAN for large datasets, 
	// but since this is for active requests (a very small set), it's acceptable.
	
	// Assuming redis client is accessible or we can use the cache interface if it supports iteration
	// But our cache interface might not support iteration.
	// Let's use the DB instead. 
	// Wait, we can query Orders for Status != CANCELLED and DriverId assigned, 
	// but there's no "early_complete" status on the Order until approved. 
	// Oh, I can just rely on the UI making the user type the Driver ID if I don't have a backend list, OR we can add a method to cache.
	
	// Since cache.Get doesn't support list, let's just make the UI accept a DriverID.
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not_implemented"})
}

func generatePaymentBypassToken() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}
