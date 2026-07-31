package order

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RetailerShopClosedResponse struct {
	Action         string     `json:"action"` // RESCHEDULE | CREDIT_LEAVE | CANCEL | BYPASS
	Note           string     `json:"note"`
	NewSlotID      string     `json:"newSlotId"`
	NewWindowStart *time.Time `json:"newWindowStart"`
	NewWindowEnd   *time.Time `json:"newWindowEnd"`
	PhotoUrl       string     `json:"photoUrl"`
}

func (s *Service) HandleRetailerRespondShopClosed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	orderID := chi.URLParam(r, "orderID")
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id required"})
		return
	}

	var req RetailerShopClosedResponse
	body, err := readLimitedBody(r, 64*1024)
	if err != nil || len(body) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if err := s.RetailerRespondShopClosed(r.Context(), orderID, claims.Subject, req); err != nil {
		if st, ok := status.FromError(err); ok {
			statusCode := http.StatusInternalServerError
			switch st.Code() {
			case codes.PermissionDenied:
				statusCode = http.StatusForbidden
			case codes.FailedPrecondition:
				statusCode = http.StatusConflict
			case codes.InvalidArgument:
				statusCode = http.StatusBadRequest
			case codes.NotFound:
				statusCode = http.StatusNotFound
			}
			writeJSON(w, statusCode, map[string]string{"error": st.Message()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Service) RetailerRespondShopClosed(
	ctx context.Context,
	orderID, retailerID string,
	req RetailerShopClosedResponse,
) error {
	now := s.now().UTC()

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		order, err := s.loadOrderForUpdate(ctx, txn, orderID)
		if err != nil {
			return err
		}

		// Guards
		if order.RetailerID != retailerID {
			return status.Error(codes.PermissionDenied, "not your order")
		}
		if order.Status != StatusShopClosedPending {
			return status.Errorf(codes.FailedPrecondition, "order is already resolved or not waiting for response")
		}

		switch req.Action {
		case "RESCHEDULE":
			return s.resolveReschedule(ctx, txn, order, req, now)
		case "CREDIT_LEAVE":
			return s.resolveCreditLeaveFromRetailer(ctx, txn, order, req, now)
		case "CANCEL":
			return s.resolveCancelFromRetailer(ctx, txn, order, req, now)
		case "BYPASS":
			return s.resolveBypassFromRetailer(ctx, txn, order, req, now)
		default:
			return status.Errorf(codes.InvalidArgument, "unknown action %s", req.Action)
		}
	})

	if err == nil {
		s.invalidateOrderCache(ctx, orderID)
	}
	return err
}

func (s *Service) resolveReschedule(ctx context.Context, txn *spanner.ReadWriteTransaction, order *Order, req RetailerShopClosedResponse, now time.Time) error {
	if req.NewWindowStart == nil || req.NewWindowEnd == nil {
		return status.Error(codes.InvalidArgument, "newWindowStart and newWindowEnd required for RESCHEDULE")
	}
	if req.NewWindowStart.Before(now) {
		return status.Error(codes.InvalidArgument, "new window start must be in the future")
	}

	buf := &spannerTxnBuffer{}
	var mutations []*spanner.Mutation

	newStatus := StatusPending
	mutations = append(mutations, spanner.UpdateMap("Orders", map[string]any{
		"OrderId":               order.OrderID,
		"Status":                string(newStatus),
		"ShopClosedResolution":  "RESCHEDULED",
		"Version":               order.Version + 1,
		"UpdatedAt":             now,
		"ReceivingWindowOpen":   *req.NewWindowStart,
		"ReceivingWindowClose":  *req.NewWindowEnd,
	}))

	logPayload, _ := json.Marshal(req)
	mutations = append(mutations, spanner.InsertMap("OrderShopClosedLog", map[string]any{
		"OrderId":   order.OrderID,
		"EventId":   s.newID(),
		"Actor":     order.RetailerID,
		"Action":    "RESPONDED",
		"Payload":   logPayload,
		"CreatedAt": now,
	}))

	_ = outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    order.OrderID,
		SupplierID: order.SupplierID,
		Status:     string(newStatus),
		Reason:     "retailer_rescheduled",
	})
	_ = outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: "order.shop_closed.retailer_responded", Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    order.OrderID,
		SupplierID: order.SupplierID,
		Resolution: "RESCHEDULE",
	})

	for _, e := range buf.events {
		mutations = append(mutations, outboxMutation(e))
	}
	return txn.BufferWrite(mutations)
}

func (s *Service) resolveCreditLeaveFromRetailer(ctx context.Context, txn *spanner.ReadWriteTransaction, order *Order, req RetailerShopClosedResponse, now time.Time) error {
	profile, err := s.getProfileForUpdate(ctx, txn, order.RetailerID, order.SupplierID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get credit profile: %v", err)
	}

	score, err := getRetailerCreditScore(ctx, txn, order.RetailerID)
	if err != nil {
		s.log.WarnContext(ctx, "failed to load credit score for credit leave", "err", err, "retailer_id", order.RetailerID)
	}

	cfg := TimeoutConfig{
		MaxAutoCreditMinor:            50000000,
		MaxRiskTierForAutoCredit:      2,
		AllowForceBypass:              false,
		CreditScoreEnforcementEnabled: s.creditScoreEnforcement,
	}

	if err := CanLeaveOnCredit(order, profile, score, cfg, cfg.CreditScoreEnforcementEnabled); err != nil {
		return err
	}

	buf := &spannerTxnBuffer{}
	var mutations []*spanner.Mutation

	newBalance := profile.CurrentBalanceMinor + order.TotalMinor
	mutations = append(mutations, spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
		"RetailerId":           profile.RetailerID,
		"SupplierId":           profile.SupplierID,
		"CurrentBalanceMinor":  newBalance,
		"AvailableCreditMinor": max(0, profile.CreditLimitMinor-newBalance),
		"Version":              profile.Version + 1,
		"UpdatedAt":            spanner.CommitTimestamp,
	}))

	newStatus := StatusDeliveredOnCredit
	mutations = append(mutations, spanner.UpdateMap("Orders", map[string]any{
		"OrderId":              order.OrderID,
		"Status":               string(newStatus),
		"ShopClosedResolution": ShopClosedResolutionCreditLeave,
		"Version":              order.Version + 1,
		"UpdatedAt":            now,
	}))

	logPayload, _ := json.Marshal(req)
	mutations = append(mutations, spanner.InsertMap("OrderShopClosedLog", map[string]any{
		"OrderId":   order.OrderID,
		"EventId":   s.newID(),
		"Actor":     order.RetailerID,
		"Action":    "RESPONDED",
		"Payload":   logPayload,
		"CreatedAt": now,
	}))

	_ = outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    order.OrderID,
		SupplierID: order.SupplierID,
		Status:     string(newStatus),
	})
	_ = outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: "order.shop_closed.retailer_responded", Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    order.OrderID,
		SupplierID: order.SupplierID,
		Resolution: "CREDIT_LEAVE",
	})

	for _, e := range buf.events {
		mutations = append(mutations, outboxMutation(e))
	}
	return txn.BufferWrite(mutations)
}

func (s *Service) resolveCancelFromRetailer(ctx context.Context, txn *spanner.ReadWriteTransaction, order *Order, req RetailerShopClosedResponse, now time.Time) error {
	buf := &spannerTxnBuffer{}
	var mutations []*spanner.Mutation

	if err := releaseOrderReservationsInTxn(ctx, txn, order); err != nil {
		return err
	}

	newStatus := StatusCancelled
	mutations = append(mutations, spanner.UpdateMap("Orders", map[string]any{
		"OrderId":              order.OrderID,
		"Status":               string(newStatus),
		"ShopClosedResolution": "CANCELLED",
		"Version":              order.Version + 1,
		"UpdatedAt":            now,
	}))

	logPayload, _ := json.Marshal(req)
	mutations = append(mutations, spanner.InsertMap("OrderShopClosedLog", map[string]any{
		"OrderId":   order.OrderID,
		"EventId":   s.newID(),
		"Actor":     order.RetailerID,
		"Action":    "RESPONDED",
		"Payload":   logPayload,
		"CreatedAt": now,
	}))

	_ = outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    order.OrderID,
		SupplierID: order.SupplierID,
		Status:     string(newStatus),
		Reason:     "retailer_cancelled",
	})
	_ = outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: "order.shop_closed.retailer_responded", Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    order.OrderID,
		SupplierID: order.SupplierID,
		Resolution: "CANCEL",
	})

	for _, e := range buf.events {
		mutations = append(mutations, outboxMutation(e))
	}
	return txn.BufferWrite(mutations)
}

func (s *Service) resolveBypassFromRetailer(ctx context.Context, txn *spanner.ReadWriteTransaction, order *Order, req RetailerShopClosedResponse, now time.Time) error {
	if req.PhotoUrl == "" {
		return status.Error(codes.InvalidArgument, "photoUrl required for BYPASS")
	}

	buf := &spannerTxnBuffer{}
	var mutations []*spanner.Mutation

	newStatus := StatusAwaitingPayment
	mutations = append(mutations, spanner.UpdateMap("Orders", map[string]any{
		"OrderId":              order.OrderID,
		"Status":               string(newStatus),
		"ShopClosedResolution": ShopClosedResolutionBypass,
		"Version":              order.Version + 1,
		"UpdatedAt":            now,
	}))

	logPayload, _ := json.Marshal(req)
	mutations = append(mutations, spanner.InsertMap("OrderShopClosedLog", map[string]any{
		"OrderId":   order.OrderID,
		"EventId":   s.newID(),
		"Actor":     order.RetailerID,
		"Action":    "RESPONDED",
		"Payload":   logPayload,
		"CreatedAt": now,
	}))

	_ = outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    order.OrderID,
		SupplierID: order.SupplierID,
		Status:     string(newStatus),
		Reason:     "retailer_bypass",
	})
	_ = outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: "order.shop_closed.retailer_responded", Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    order.OrderID,
		SupplierID: order.SupplierID,
		Resolution: "BYPASS",
	})

	for _, e := range buf.events {
		mutations = append(mutations, outboxMutation(e))
	}
	return txn.BufferWrite(mutations)
}
