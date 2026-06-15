package order

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

type fleetReorderRequest struct {
	RouteID       string   `json:"route_id"`
	OrderSequence []string `json:"order_sequence"`
}

// HandleFleetRouteReorder serves POST /v1/fleet/route/reorder.
func (s *Service) HandleFleetRouteReorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "route_reorder_unavailable"})
		return
	}

	var req fleetReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()
	req.RouteID = strings.TrimSpace(req.RouteID)
	if req.RouteID == "" || len(req.OrderSequence) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "route_id_and_order_sequence_required"})
		return
	}
	driverID := strings.TrimSpace(claims.Subject)
	ctx := r.Context()

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := assertDriverOwnsRoute(ctx, txn, driverID, req.RouteID); err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRoute, req.RouteID, events.TopicMain, map[string]any{
			"type":           events.EventRouteReordered,
			"route_id":       req.RouteID,
			"driver_id":      driverID,
			"supplier_id":    claims.SupplierID,
			"order_sequence": req.OrderSequence,
			"timestamp":      s.now().UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}
		mutations := make([]*spanner.Mutation, 0, len(req.OrderSequence)+len(buf.events))
		for i, orderID := range req.OrderSequence {
			orderID = strings.TrimSpace(orderID)
			if orderID == "" {
				return fmt.Errorf("empty order_id in sequence")
			}
			n, err := txn.Update(ctx, spanner.Statement{
				SQL: `UPDATE Orders SET SequenceIndex = @seq, UpdatedAt = PENDING_COMMIT_TIMESTAMP()
				      WHERE OrderId = @oid AND RouteId = @rid AND DriverId = @did`,
				Params: map[string]any{
					"seq": int64(i + 1),
					"oid": orderID,
					"rid": req.RouteID,
					"did": driverID,
				},
			})
			if err != nil {
				return fmt.Errorf("update sequence %s: %w", orderID, err)
			}
			if n == 0 {
				return fmt.Errorf("order %s not on route %s", orderID, req.RouteID)
			}
		}
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		if strings.Contains(err.Error(), "not on route") || strings.Contains(err.Error(), "not assigned") {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
		s.log.ErrorContext(ctx, "fleet route reorder failed", "err", err, "driver_id", driverID)
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	if s.manifestStore != nil {
		if geomErr := s.manifestStore.PersistRouteGeometryForDriverRoute(ctx, driverID, req.RouteID, "route_reordered"); geomErr != nil {
			s.log.WarnContext(ctx, "route geometry refresh after reorder failed",
				"route_id", req.RouteID, "driver_id", driverID, "err", geomErr)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "REORDERED",
		"route_id":   req.RouteID,
		"stop_count": len(req.OrderSequence),
	})
}

// HandleBypassOffload serves POST /v1/delivery/bypass-offload (shop-closed bypass).
func (s *Service) HandleBypassOffload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "bypass_offload_unavailable"})
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_and_bypass_token_required"})
		return
	}

	driverID := strings.TrimSpace(claims.Subject)
	ctx := r.Context()
	now := s.now()
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return applyShopClosedBypassOffload(ctx, txn, driverID, req.OrderID, req.BypassToken, now)
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	s.invalidateOrderCache(ctx, req.OrderID)
	resp := map[string]string{"status": "AWAITING_PAYMENT"}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleCreditDelivery serves POST /v1/delivery/credit-delivery.
func (s *Service) HandleCreditDelivery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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
		OrderID       string `json:"order_id"`
		PhotoProofURL string `json:"photo_proof_url"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	result, err := s.transitionDriverOrder(r.Context(), claims, driverTransitionRequest{
		OrderID:    req.OrderID,
		NextStatus: StatusDeliveredOnCredit,
		Reason:     "credit_delivery",
		Precheck: func(o Order) error {
			if o.Status != StatusArrived && o.Status != StatusArrivedShopClosed {
				return fmt.Errorf("order must be ARRIVED or ARRIVED_SHOP_CLOSED (current: %s)", o.Status)
			}
			return nil
		},
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	resp := map[string]any{
		"status":   result.Order.Status,
		"order_id": result.Order.OrderID,
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleMissingItems serves POST /v1/delivery/missing-items.
func (s *Service) HandleMissingItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleDriver && claims.Role != auth.RolePayload) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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
		Note    string `json:"note"`
		Items   []struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	current, okRow, err := s.repo.GetOrder(r.Context(), req.OrderID)
	if err != nil || !okRow {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if err := s.emitDriverEdgeEvent(r.Context(), current, map[string]any{
		"type":        events.EventMissingItemsReported,
		"order_id":    req.OrderID,
		"driver_id":   missingItemsDriverID(claims, current),
		"supplier_id": current.SupplierID,
		"retailer_id": current.RetailerID,
		"note":        req.Note,
		"items":       req.Items,
		"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		s.log.ErrorContext(r.Context(), "missing items event failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "event_failed"})
		return
	}
	
	// Phase 3: Emit Reverse Logistics requirement
	if err := s.emitDriverEdgeEvent(r.Context(), current, map[string]any{
		"type":        "REVERSE_LOGISTICS_REQUIRED",
		"order_id":    req.OrderID,
		"warehouse_id": current.WarehouseID,
		"supplier_id": current.SupplierID,
		"retailer_id": current.RetailerID,
		"items":       req.Items,
		"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		s.log.ErrorContext(r.Context(), "reverse logistics event failed", "err", err)
	}

	idemCommitted = true
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]string{"status": "reported"})
}

// HandleSplitPayment serves POST /v1/delivery/split-payment.
func (s *Service) HandleSplitPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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
	var req struct {
		OrderID   string `json:"order_id"`
		CashMinor int64  `json:"cash_minor"`
		CardMinor int64  `json:"card_minor"`
		Currency  string `json:"currency"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" || req.CashMinor+req.CardMinor <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_split_amounts"})
		return
	}

	current, okRow, err := s.repo.GetOrder(r.Context(), req.OrderID)
	if err != nil || !okRow {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = current.Currency
	}
	if err := s.emitDriverEdgeEvent(r.Context(), current, map[string]any{
		"type":        events.EventSplitPaymentCreated,
		"order_id":    req.OrderID,
		"driver_id":   claims.Subject,
		"supplier_id": current.SupplierID,
		"retailer_id": current.RetailerID,
		"cash_minor":  req.CashMinor,
		"card_minor":  req.CardMinor,
		"currency":    currency,
		"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		s.log.ErrorContext(r.Context(), "split payment event failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "event_failed"})
		return
	}
	resp := map[string]string{"status": "split_recorded"}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func (s *Service) emitDriverEdgeEvent(ctx context.Context, o Order, payload map[string]any) error {
	if s.spannerClient == nil {
		return fmt.Errorf("spanner_unavailable")
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, o.OrderID, events.TopicMain, payload); err != nil {
			return err
		}
		mutations := make([]*spanner.Mutation, 0, len(buf.events))
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return err
	}
	s.afterOrderMutation(ctx, o)
	return nil
}

func assertDriverOwnsRoute(ctx context.Context, txn *spanner.ReadWriteTransaction, driverID, routeID string) error {
	stmt := spanner.Statement{
		SQL:    `SELECT OrderId FROM Orders WHERE DriverId = @did AND RouteId = @rid LIMIT 1`,
		Params: map[string]any{"did": driverID, "rid": routeID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	_, err := iter.Next()
	if err == iterator.Done {
		return fmt.Errorf("route %s not assigned to driver %s", routeID, driverID)
	}
	return err
}

func applyShopClosedBypassOffload(ctx context.Context, txn *spanner.ReadWriteTransaction, driverID, orderID, token string, now time.Time) error {
	row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID},
		[]string{"Status", "Version", "DriverId", "SupplierId", "RetailerId"})
	if err != nil {
		return ErrOrderNotFound
	}
	var status string
	var version int64
	var did spanner.NullString
	var supplierID, retailerID string
	if err := row.Columns(&status, &version, &did, &supplierID, &retailerID); err != nil {
		return err
	}
	if status != string(StatusArrivedShopClosed) {
		return fmt.Errorf("order must be ARRIVED_SHOP_CLOSED (current: %s)", status)
	}
	if !did.Valid || did.StringVal != driverID {
		return ErrOrderForbidden
	}

	stmt := spanner.Statement{
		SQL: `SELECT AttemptId, BypassToken FROM ShopClosedAttempts
		      WHERE OrderId = @oid AND Resolution = 'BYPASS_ISSUED'
		      ORDER BY ReportedAt DESC LIMIT 1`,
		Params: map[string]any{"oid": orderID},
	}
	iter := txn.Query(ctx, stmt)
	arow, err := iter.Next()
	iter.Stop()
	if err != nil {
		return fmt.Errorf("no bypass token for order %s", orderID)
	}
	var attemptID string
	var stored spanner.NullString
	if err := arow.Columns(&attemptID, &stored); err != nil {
		return err
	}
	if !stored.Valid || stored.StringVal != token {
		return fmt.Errorf("invalid bypass token")
	}

	buf := &spannerTxnBuffer{}
	if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, map[string]any{
		"type":        "SHOP_CLOSED_BYPASS_OFFLOAD",
		"order_id":    orderID,
		"driver_id":   driverID,
		"supplier_id": supplierID,
		"retailer_id": retailerID,
		"status":      string(StatusAwaitingPayment),
		"timestamp":   now.UTC().Format(time.RFC3339Nano),
	}); err != nil {
		return err
	}
	mutations := []*spanner.Mutation{
		spanner.UpdateMap("Orders", map[string]any{
			"OrderId":   orderID,
			"Status":    string(StatusAwaitingPayment),
			"Version":   version + 1,
			"UpdatedAt": now.UTC(),
		}),
		spanner.UpdateMap("ShopClosedAttempts", map[string]any{
			"AttemptId":  attemptID,
			"ResolvedAt": now.UTC(),
		}),
	}
	for _, e := range buf.events {
		mutations = append(mutations, outboxMutation(e))
	}
	return txn.BufferWrite(mutations)
}

func missingItemsDriverID(claims auth.Claims, order Order) string {
	if claims.Role == auth.RoleDriver {
		return claims.Subject
	}
	if order.DriverID != "" {
		return order.DriverID
	}
	return claims.Subject
}
