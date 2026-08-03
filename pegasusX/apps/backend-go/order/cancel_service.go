package order

import (
	"context"
	"errors"
	"strings"
	"time"

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
	if s.replanner != nil && current.ManifestID != "" {
		go func(rID, act string) {
			_ = s.replanner.ReplanRoute(context.Background(), rID, "cancel_order", act)
		}(current.ManifestID, actorID)
	}
	return lifecycleResponse(*current, current.Version, false), nil
}
