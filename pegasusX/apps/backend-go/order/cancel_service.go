package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// RejectPreorder lets a retailer cancel a draft or scheduled manual pre-order.
func (s *Service) RejectPreorder(ctx context.Context, retailerID string, req RejectPreorderRequest) (RetailerOrderLifecycleResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return RetailerOrderLifecycleResponse{}, errors.New("order_id required")
	}
	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	if !ok {
		return RetailerOrderLifecycleResponse{}, ErrOrderNotFound
	}
	if current.RetailerID != strings.TrimSpace(retailerID) {
		return RetailerOrderLifecycleResponse{}, ErrOrderForbidden
	}
	if current.Source != OrderSourceManualPreorder {
		return RetailerOrderLifecycleResponse{}, ErrInvalidStatusTransition
	}
	switch current.Status {
	case StatusScheduled, StatusAutoAccepted:
	default:
		return RetailerOrderLifecycleResponse{}, ErrInvalidStatusTransition
	}
	if current.ConfirmationStatus == ConfirmationStatusPendingWarehouse {
		return RetailerOrderLifecycleResponse{}, ErrDeliveryProposalPending
	}
	if PreorderCancelLocked(s.now(), current) {
		return RetailerOrderLifecycleResponse{}, ErrOrderCancelLocked
	}
	return s.cancelOrderWithReason(ctx, &current, retailerID, string(auth.RoleRetailer), strings.TrimSpace(req.Reason), events.EventPreOrderCancelled)
}

func (s *Service) cancelOrderWithReason(ctx context.Context, current *Order, actorID, actorRole, reason, eventType string) (RetailerOrderLifecycleResponse, error) {
	if current.Status == StatusCancelled {
		return lifecycleResponse(*current, current.Version, false), nil
	}
	prevStatus := current.Status
	if err := ValidateStatusTransition(string(prevStatus), string(StatusCancelled), TransitionOpts{Actor: actorID, Reason: reason}); err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	current.Status = StatusCancelled
	current.ConfirmationStatus = ConfirmationStatusRejected
	clearDeliveryProposal(current)
	current.UpdatedAt = s.now()
	if reason != "" {
		current.WarehouseNotes = reason
	}
	if err := s.repo.UpdateOrder(ctx, *current, nil, func(txn outbox.TxnBuffer) error {
		if eventType == events.EventPreOrderCancelled || eventType == events.EventPreOrderDateRejected {
			return emitPreorderEvent(ctx, txn, eventType, *current, actorRole, actorID)
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:             events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: current.UpdatedAt.UTC().Format(time.RFC3339Nano)},
			OrderID:               current.OrderID,
			SupplierID:            current.SupplierID,
			RetailerID:            current.RetailerID,
			WarehouseID:           current.WarehouseID,
			PreviousStatus:        string(prevStatus),
			Status:                string(current.Status),
			Reason:                reason,
			ActorRole:             actorRole,
			ActorID:               actorID,
			OrderSource:           string(current.Source),
			ConfirmationStatus:    string(current.ConfirmationStatus),
			RequestedDeliveryDate: formatOptionalRFC3339(current.RequestedDeliveryDate),
		})
	}); err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	s.afterOrderMutation(ctx, *current)
	if s.credit != nil {
		if err := s.credit.ReleaseReserve(ctx, current.OrderID); err != nil {
			s.log.Warn("credit reserve release on cancel failed", "order_id", current.OrderID, "err", err)
			_ = s.enqueueCreditReleaseCompensation(ctx, current, reason, err)
		}
	}
	if s.spannerClient != nil {
		if err := s.ReleaseAllocation(ctx, current.OrderID); err != nil {
			s.log.Warn("release allocation on cancel failed", "order_id", current.OrderID, "err", err)
		}
	}
	if current.ManifestID != "" {
		s.replanAsync(current.ManifestID, "cancel_order", actorID)
	}
	return lifecycleResponse(*current, current.Version, false), nil
}

// CreditReleaseCompensationPayload describes the outbox event payload when credit reservation release fails.
type CreditReleaseCompensationPayload struct {
	OrderID    string `json:"order_id"`
	SupplierID string `json:"supplier_id"`
	RetailerID string `json:"retailer_id"`
	Reason     string `json:"reason"`
	Error      string `json:"error"`
}

// CompensationOutboxHook is an optional hook for testing compensating outbox events.
var CompensationOutboxHook func(ctx context.Context, e outbox.Event) error

func (s *Service) enqueueCreditReleaseCompensation(ctx context.Context, current *Order, reason string, releaseErr error) error {
	payload := CreditReleaseCompensationPayload{
		OrderID:    current.OrderID,
		SupplierID: current.SupplierID,
		RetailerID: current.RetailerID,
		Reason:     reason,
		Error:      releaseErr.Error(),
	}

	if CompensationOutboxHook != nil {
		raw, _ := json.Marshal(payload)
		return CompensationOutboxHook(ctx, outbox.Event{
			EventID:       current.OrderID + "-credit-comp",
			AggregateType: events.AggregateOrder,
			AggregateID:   current.OrderID,
			TopicName:     events.TopicMain,
			Payload:       raw,
			CreatedAt:     s.now(),
			SupplierID:    current.SupplierID,
		})
	}

	if s.spannerClient != nil {
		_, writeErr := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			buf := outbox.NewSpannerTxnBuffer(txn)
			if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, current.OrderID, events.TopicMain, payload); err != nil {
				return err
			}
			return buf.Flush(ctx)
		})
		return writeErr
	}
	return nil
}

// RequestCancelOrderRequest is the payload for requesting order cancellation.
type RequestCancelOrderRequest struct {
	Reason string `json:"reason"`
}

// RequestCancelOrder transitions an order to CANCEL_REQUESTED (retailer or admin).
func (s *Service) RequestCancelOrder(ctx context.Context, claims auth.Claims, orderID string, req RequestCancelOrderRequest) (UpdateStatusResponse, error) {
	return s.UpdateStatus(ctx, claims, orderID, UpdateStatusRequest{
		Status: string(StatusCancelRequested),
		Reason: req.Reason,
	})
}

// HandleRequestCancel is POST /v1/order/{orderID}/cancel-request.
func (s *Service) HandleRequestCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := chi.URLParam(r, "orderID")
	if strings.TrimSpace(orderID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	var req RequestCancelOrderRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	resp, err := s.RequestCancelOrder(r.Context(), claims, orderID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrderNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		case errors.Is(err, ErrOrderForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		case errors.Is(err, ErrInvalidStatusTransition):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_status_transition"})
		case errors.Is(err, ErrOrderCancelLocked):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "order_cancel_locked"})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
