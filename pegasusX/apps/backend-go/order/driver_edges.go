package order

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
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

	var req fleetReorderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.RouteID = strings.TrimSpace(req.RouteID)
	if req.RouteID == "" || len(req.OrderSequence) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "route_id_and_order_sequence_required"})
		return
	}
	driverID := strings.TrimSpace(claims.Subject)
	ctx := r.Context()

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
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
	resp := map[string]any{
		"status":     "REORDERED",
		"route_id":   req.RouteID,
		"stop_count": len(req.OrderSequence),
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
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
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleCreditLeave serves POST /v1/driver/orders/{orderId}/credit-leave.
func (s *Service) HandleCreditLeave(w http.ResponseWriter, r *http.Request) {
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
	orderID := chi.URLParam(r, "orderId")
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	
	var req struct {
		Location DriverTelemetry `json:"location"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if err := req.Location.Validate(100.0); err != nil {
		writeJSON(w, http.StatusPreconditionFailed, map[string]string{"error": err.Error()})
		return
	}

	ctx := r.Context()
	
	// Perform transition to DeliveredOnCredit, enforcing proximity via UpdateOrderWithTxn similar to shop-closed.
	// But let's load the order to get the proximity unlocked status and method, and to perform the update.
	var current Order
	current, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "order_load_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}

	if current.Status != StatusArrived {
		s.writeOrderMutationError(w, "credit leave failed", orderID, fmt.Errorf("%w: order must be ARRIVED for credit leave (current: %s)", ErrInvalidStatusTransition, current.Status))
		return
	}

	// Update status
	current.Status = StatusDeliveredOnCredit
	current.UpdatedAt = s.now().UTC()

	err = s.repo.UpdateOrderWithTxn(ctx, current, nil, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := s.ensureProximityUnlocked(ctx, txn, &current, req.Location.ToLocation(), TransitionOpts{
			Actor:  claims.Subject,
			Reason: "credit_leave",
		}); err != nil {
			return err
		}

		profile, err := s.getProfileForUpdate(ctx, txn, current.RetailerID, current.SupplierID)
		if err != nil {
			return fmt.Errorf("failed to load credit profile: %w", err)
		}
		score, err := getRetailerCreditScore(ctx, txn, current.RetailerID)
		if err != nil {
			s.log.WarnContext(ctx, "failed to load credit score for driver credit leave", "err", err, "retailer_id", current.RetailerID)
		}

		cfg := TimeoutConfig{
			MaxAutoCreditMinor:            50000000,
			MaxRiskTierForAutoCredit:      2,
			AllowForceBypass:              false,
			CreditScoreEnforcementEnabled: s.creditScoreEnforcement,
		}

		if err := CanLeaveOnCredit(&current, profile, score, cfg, cfg.CreditScoreEnforcementEnabled); err != nil {
			return err
		}

		leg := PaymentLeg{
			OrderID:        orderID,
			LegID:          s.newID(),
			Method:         MethodCredit,
			AmountMinor:    current.TotalMinor,
			Status:         PaymentStatusCaptured,
			IdempotencyKey: fmt.Sprintf("credit-leave-%s-%s", orderID, s.newID()),
			CreatedAt:      s.now(),
			CapturedAt:     spanner.NullTime{Time: s.now(), Valid: true},
		}
		if err := s.RecordPaymentLeg(ctx, txn, leg); err != nil {
			return err
		}
		if s.credit != nil && current.TotalMinor > 0 {
			if err := s.credit.MarkBalanceInTxn(ctx, txn, current.RetailerID, current.SupplierID, orderID, current.TotalMinor); err != nil {
				return err
			}
		}
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: s.now().UTC().Format(time.RFC3339Nano)},
			OrderID:    current.OrderID,
			DriverID:   claims.Subject,
			RetailerID: current.RetailerID,
			SupplierID: current.SupplierID,
			Status:     string(current.Status),
		})
	})
	if err != nil {
		s.log.ErrorContext(ctx, "credit leave failed", "order_id", orderID, "err", err)
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	dueAt := ""
	termsDays := int64(0)
	if s.credit != nil {
		if check, cerr := s.credit.CheckCreditPath(ctx, current.RetailerID, current.SupplierID, 0); cerr == nil {
			dueAt = check.DueAt
			termsDays = check.TermsDays
		}
	}
	leaveAt := s.now()
	dueTime := leaveAt.AddDate(0, 0, 30)
	if dueAt != "" {
		if t, perr := time.Parse(time.RFC3339, dueAt); perr == nil {
			dueTime = t
		}
	} else if termsDays > 0 {
		dueTime = leaveAt.AddDate(0, 0, int(termsDays))
		dueAt = dueTime.UTC().Format(time.RFC3339)
	}
	if s.ar != nil && current.TotalMinor > 0 {
		if _, aerr := s.ar.OpenFromCreditLeave(ctx, current.SupplierID, current.RetailerID, orderID,
			current.TotalMinor, termsDays, 0, leaveAt, dueTime); aerr != nil {
			s.log.Error("open AR invoice failed", "order_id", orderID, "err", aerr)
		}
	}
	s.invalidateOrderCache(ctx, orderID)

	var proxUnlocked bool
	var proxMethod string
	if current.ProximityUnlockedAt != nil {
		proxUnlocked = true
		proxMethod = current.ProximityMethod
	}

	resp := map[string]any{
		"order_id":            orderID,
		"status":              string(StatusDeliveredOnCredit),
		"proximity_unlocked":  proxUnlocked,
		"proximity_method":    proxMethod,
		"due_at":              dueAt,
		"terms_days":          termsDays,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
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
		OrderID          string `json:"order_id"`
		PhotoProofURL    string `json:"photo_proof_url"`
		ForceBypassToken string `json:"force_bypass_token,omitempty"`
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

	ctx := r.Context()
	// Proximity gate: credit leave only when unlocked or supervised FORCE_BYPASS.
	if err := s.requireProximityUnlocked(ctx, req.OrderID, req.ForceBypassToken); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": ErrProximityLocked.Error()})
		return
	}

	var leaveRetailerID, leaveSupplierID string
	result, err := s.transitionDriverOrder(ctx, claims, driverTransitionRequest{
		OrderID:    req.OrderID,
		NextStatus: StatusDeliveredOnCredit,
		Reason:     "credit_delivery",
		Precheck: func(o Order) error {
			leaveRetailerID = o.RetailerID
			leaveSupplierID = o.SupplierID
			if o.Status != StatusArrived && o.Status != StatusShopClosedPending {
				return fmt.Errorf("order must be ARRIVED or ARRIVED_SHOP_CLOSED (current: %s)", o.Status)
			}
			// Block if already in fiscal path.
			if o.Status == StatusFiscalizing || o.FiscalStatus == FiscalStatusPending || o.FiscalStatus == FiscalStatusSuccess {
				return fmt.Errorf("credit leave blocked: fiscal state %s", o.FiscalStatus)
			}
			if s.credit != nil && o.TotalMinor > 0 {
				check, cerr := s.credit.CheckCreditPath(ctx, o.RetailerID, o.SupplierID, o.TotalMinor)
				if cerr != nil {
					return cerr
				}
				if !check.Allowed {
					return fmt.Errorf("%w: %s", ErrCreditLimitBreached, check.Reason)
				}
			}
			return nil
		},
		PrepareOrder: func(o *Order, _ Status) {
			if o.ShopClosedResolution == "" {
				o.ShopClosedResolution = ShopClosedResolutionCreditLeave
			}
		},
		EmitExtra: func(txn outbox.TxnBuffer, orderRecord Order, _ Status) error {
			if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, events.CreditDeliveryEvent{
				BaseEvent:  events.BaseEvent{Type: events.EventCreditDeliveryMarked, Timestamp: s.now().UTC().Format(time.RFC3339Nano)},
				OrderID:    orderRecord.OrderID,
				DriverID:   claims.Subject,
				SupplierID: orderRecord.SupplierID,
				RetailerID: orderRecord.RetailerID,
				Status:     string(StatusDeliveredOnCredit),
			}); err != nil {
				return err
			}
			return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, events.OrderEvent{
				BaseEvent:  events.BaseEvent{Type: events.EventCreditLeave, Timestamp: s.now().UTC().Format(time.RFC3339Nano)},
				OrderID:    orderRecord.OrderID,
				DriverID:   claims.Subject,
				SupplierID: orderRecord.SupplierID,
				RetailerID: orderRecord.RetailerID,
				Status:     string(StatusDeliveredOnCredit),
				Resolution: ShopClosedResolutionCreditLeave,
			})
		},
		InTxn: func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
			delivered, err := s.getDeliveredGrossMinor(txnCtx, req.OrderID)
			if err != nil {
				return err
			}
			leg := PaymentLeg{
				OrderID:        req.OrderID,
				LegID:          s.newID(),
				Method:         MethodCredit,
				AmountMinor:    delivered,
				Status:         PaymentStatusCaptured,
				IdempotencyKey: fmt.Sprintf("credit-leave-%s-%s", req.OrderID, s.newID()),
				CreatedAt:      s.now(),
				CapturedAt:     spanner.NullTime{Time: s.now(), Valid: true},
			}
			if err := s.RecordPaymentLeg(txnCtx, txn, leg); err != nil {
				return err
			}
			// Same-txn MarkBalance / convert reserve (P0 integrity).
			if s.credit != nil && delivered > 0 {
				if markErr := s.credit.MarkBalanceInTxn(txnCtx, txn, leaveRetailerID, leaveSupplierID, req.OrderID, delivered); markErr != nil {
					return markErr
				}
			}
			return nil
		},
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	dueAt := ""
	termsDays := int64(0)
	if s.credit != nil && result.Order.TotalMinor > 0 {
		if check, cerr := s.credit.CheckCreditPath(ctx, result.Order.RetailerID, result.Order.SupplierID, 0); cerr == nil {
			dueAt = check.DueAt
			termsDays = check.TermsDays
		}
		leaveAt := s.now()
		dueTime := leaveAt.AddDate(0, 0, 30)
		if dueAt != "" {
			if t, perr := time.Parse(time.RFC3339, dueAt); perr == nil {
				dueTime = t
			}
		} else if termsDays > 0 {
			dueTime = leaveAt.AddDate(0, 0, int(termsDays))
			dueAt = dueTime.UTC().Format(time.RFC3339)
		}
		if s.ar != nil {
			if _, aerr := s.ar.OpenFromCreditLeave(ctx, result.Order.SupplierID, result.Order.RetailerID, result.Order.OrderID,
				result.Order.TotalMinor, termsDays, 0, leaveAt, dueTime); aerr != nil {
				s.log.Error("open AR invoice failed", "order_id", result.Order.OrderID, "err", aerr)
			}
		}
	}
	s.invalidateOrderCache(ctx, req.OrderID)
	resp := map[string]any{
		"status":     result.Order.Status,
		"order_id":   result.Order.OrderID,
		"due_at":     dueAt,
		"terms_days": termsDays,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// ExceptionReportItem is one OS&D line on a driver exception report.
// Wire accepts both canonical and driver-app legacy field names.
type ExceptionReportItem struct {
	SKU      string `json:"sku"`
	Quantity int64  `json:"quantity"`
	Reason   string `json:"reason"` // MISSING | DAMAGED | WRONG_ITEM | OTHER
	PhotoURL string `json:"photo_url"`
}

// UnmarshalJSON accepts sku/sku_id and quantity/missing_qty (driver iOS/Android legacy).
func (it *ExceptionReportItem) UnmarshalJSON(data []byte) error {
	var raw struct {
		SKU        string `json:"sku"`
		SKUID      string `json:"sku_id"`
		Quantity   int64  `json:"quantity"`
		MissingQty int64  `json:"missing_qty"`
		Reason     string `json:"reason"`
		PhotoURL   string `json:"photo_url"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	it.SKU = strings.TrimSpace(raw.SKU)
	if it.SKU == "" {
		it.SKU = strings.TrimSpace(raw.SKUID)
	}
	it.Quantity = raw.Quantity
	if it.Quantity <= 0 {
		it.Quantity = raw.MissingQty
	}
	it.Reason = raw.Reason
	it.PhotoURL = raw.PhotoURL
	return nil
}

// exceptionReportRequest is the POST body for exception-report / missing-items.
type exceptionReportRequest struct {
	OrderID  string
	Note     string
	Items    []ExceptionReportItem
	PhotoURL string
}

// parseExceptionReportBody accepts canonical {items} and legacy driver {missing_items}.
func parseExceptionReportBody(body []byte) (exceptionReportRequest, error) {
	var raw struct {
		OrderID      string                `json:"order_id"`
		Note         string                `json:"note"`
		Items        []ExceptionReportItem `json:"items"`
		MissingItems []ExceptionReportItem `json:"missing_items"`
		PhotoURL     string                `json:"photo_url"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return exceptionReportRequest{}, err
	}
	items := raw.Items
	if len(items) == 0 {
		items = raw.MissingItems
	}
	// Legacy missing-items default reason MISSING when empty.
	for i := range items {
		if strings.TrimSpace(items[i].Reason) == "" {
			items[i].Reason = "MISSING"
		}
	}
	return exceptionReportRequest{
		OrderID:  strings.TrimSpace(raw.OrderID),
		Note:     raw.Note,
		Items:    items,
		PhotoURL: raw.PhotoURL,
	}, nil
}

// HandleMissingItems serves POST /v1/delivery/missing-items (compat alias).
func (s *Service) HandleMissingItems(w http.ResponseWriter, r *http.Request) {
	s.HandleExceptionReport(w, r)
}

// HandleExceptionReport serves POST /v1/delivery/exception-report (and missing-items).
// Drivers report MISSING/DAMAGED quantities with optional visual proof URLs.
// DAMAGED lines require photo_url so adjudication is possible.
//
// Wire compatibility: driver apps send legacy missing_items[{sku_id, missing_qty}];
// canonical clients send items[{sku, quantity, reason, photo_url}].
func (s *Service) HandleExceptionReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleDriver && claims.Role != auth.RolePayload) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	body, err := readLimitedBody(r, 256*1024)
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

	req, err := parseExceptionReportBody(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	current, okRow, err := s.repo.GetOrder(r.Context(), req.OrderID)
	if err != nil || !okRow {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if claims.Role == auth.RoleDriver {
		if strings.TrimSpace(current.DriverID) == "" || strings.TrimSpace(current.DriverID) != strings.TrimSpace(claims.Subject) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
	}
	if len(req.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "items_required"})
		return
	}

	origQtyBySKU := make(map[string]int64, len(current.LineItems))
	for _, line := range current.LineItems {
		origQtyBySKU[strings.TrimSpace(line.SKU)] = line.Quantity
	}
	amendItems := make([]AmendItemRequest, 0, len(req.Items))
	photoURIs := make([]string, 0, len(req.Items)+1)
	if u := strings.TrimSpace(req.PhotoURL); u != "" {
		photoURIs = append(photoURIs, u)
	}
	for _, item := range req.Items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_exception_item"})
			return
		}
		reason := normalizeAmendReason(item.Reason)
		if reason == "" {
			reason = "MISSING"
		}
		if err := validateAmendReason(reason, ""); err != nil && reason != "OTHER" {
			// OTHER without custom reason still allowed for driver edges — map note.
			if reason != "OTHER" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_reason", "message": err.Error()})
				return
			}
		}
		// Visual proof required for damage / wrong-item during handshake.
		if reason == "DAMAGED" || reason == "WRONG_ITEM" {
			photo := strings.TrimSpace(item.PhotoURL)
			if photo == "" {
				photo = strings.TrimSpace(req.PhotoURL)
			}
			if photo == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "photo_url_required_for_damage"})
				return
			}
			photoURIs = append(photoURIs, photo)
		} else if u := strings.TrimSpace(item.PhotoURL); u != "" {
			photoURIs = append(photoURIs, u)
		}
		origQty, ok := origQtyBySKU[sku]
		if !ok || item.Quantity > origQty {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("exception quantity exceeds item quantity for sku %s", sku)})
			return
		}
		amendItems = append(amendItems, AmendItemRequest{
			ProductID:   sku,
			AcceptedQty: origQty - item.Quantity,
			RejectedQty: item.Quantity,
			Reason:      reason,
		})
	}

	amendResp, err := s.applyOrderAmendments(r.Context(), current, AmendOrderRequest{
		OrderID:     req.OrderID,
		Items:       amendItems,
		DriverNotes: req.Note,
	}, missingItemsDriverID(claims, current))
	if err != nil {
		s.writeOrderMutationError(w, "exception report amend failed", req.OrderID, err)
		return
	}

	updated, okRow, err := s.repo.GetOrder(r.Context(), req.OrderID)
	if err != nil || !okRow {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	driverID := missingItemsDriverID(claims, updated)
	if err := s.emitDriverEdgeEvent(r.Context(), updated, map[string]any{
		"type":        events.EventMissingItemsReported, // TODO maybe rename to EventExceptionReported
		"order_id":    req.OrderID,
		"driver_id":   driverID,
		"supplier_id": updated.SupplierID,
		"retailer_id": updated.RetailerID,
		"note":        req.Note,
		"image_url":   req.PhotoURL,
		"items":       req.Items,
		"photo_urls":  photoURIs,
		"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		s.log.ErrorContext(r.Context(), "exception report event failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "event_failed"})
		return
	}

	// Logistics exceptions topic (plus main via dual emit for reverse logistics).
	if err := s.emitExceptionTopics(r.Context(), updated, map[string]any{
		"type":        events.EventLogisticsExceptionReported,
		"order_id":    req.OrderID,
		"driver_id":   driverID,
		"supplier_id": updated.SupplierID,
		"retailer_id": updated.RetailerID,
		"note":        req.Note,
		"items":       req.Items,
		"photo_urls":  photoURIs,
		"timestamp":   s.now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		s.log.ErrorContext(r.Context(), "logistics exception topic emit failed", "err", err)
	}

	if err := s.emitExceptionTopics(r.Context(), updated, map[string]any{
		"type":         events.EventReverseLogisticsRequired,
		"order_id":     req.OrderID,
		"warehouse_id": updated.WarehouseID,
		"supplier_id":  updated.SupplierID,
		"retailer_id":  updated.RetailerID,
		"items":        req.Items,
		"photo_urls":   photoURIs,
		"timestamp":    s.now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		s.log.ErrorContext(r.Context(), "reverse logistics event failed", "err", err)
	}

	// Optional claims bridge when service is wired (driver OS&D → claim).
	if s.claimsBridge != nil {
		if err := s.claimsBridge.OnDriverException(r.Context(), updated, driverID, req.Items, photoURIs, req.Note); err != nil {
			s.log.WarnContext(r.Context(), "claims bridge failed", "err", err)
		}
	}

	idemCommitted = true
	// Include order_id + reported status aliases for driver iOS/Android wire contracts.
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"status":          "reported",
		"order_id":        req.OrderID,
		"adjusted_total":  amendResp.AdjustedTotal,
		"original_amount": orderOriginalAmountMinor(updated),
		"photo_count":     len(photoURIs),
	})
}

// claimsBridge is optional — set from bootstrap to open Claims rows from driver OS&D.
type claimsBridge interface {
	OnDriverException(ctx context.Context, o Order, driverID string, items []ExceptionReportItem, photos []string, note string) error
	GetRemainingClaimable(ctx context.Context, orderID string) (RemainingClaimableMinor int64, DeliveredGrossMinor int64, err error)
}

// SetClaimsBridge wires the claims domain bridge (optional).
func (s *Service) SetClaimsBridge(b claimsBridge) {
	if s != nil {
		s.claimsBridge = b
	}
}

func (s *Service) emitExceptionTopics(ctx context.Context, o Order, payload map[string]any) error {
	if s.spannerClient == nil {
		return fmt.Errorf("spanner_unavailable")
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		// New logistics exceptions topic.
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, o.OrderID, events.TopicExceptions, payload); err != nil {
			return err
		}
		// Keep main so existing notification consumers still see reverse logistics.
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, o.OrderID, events.TopicMain, payload); err != nil {
			return err
		}
		mutations := make([]*spanner.Mutation, 0, len(buf.events))
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	return err
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
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

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
	idemCommitted = true
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
	if status != string(StatusShopClosedPending) {
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
	if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.ShopClosedBypassOffloadEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventShopClosedBypassOffload, Timestamp: now.UTC().Format(time.RFC3339Nano)},
		OrderID:    orderID,
		DriverID:   driverID,
		SupplierID: supplierID,
		RetailerID: retailerID,
		Status:     string(StatusAwaitingPayment),
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
