package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// PartnerCancelOrder cancels an order on behalf of a partner/supplier.
// The partner layer has already authenticated and authorized the caller.
func (s *Service) PartnerCancelOrder(ctx context.Context, orderID string, reason string) (UpdateStatusResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return UpdateStatusResponse{}, errors.New("order_id required")
	}

	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return UpdateStatusResponse{}, fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !ok {
		return UpdateStatusResponse{}, ErrOrderNotFound
	}

	resp, err := s.cancelOrderWithReason(ctx, &current, "partner_api", "PARTNER", reason, events.EventOrderStatusChanged)
	if err != nil {
		return UpdateStatusResponse{}, err
	}

	return UpdateStatusResponse{
		OrderID:        resp.OrderID,
		PreviousStatus: current.Status,
		Status:         resp.Status,
		Version:        resp.Version,
		UpdatedAt:      resp.UpdatedAt,
		EventType:      events.EventOrderStatusChanged,
	}, nil
}

// PartnerUpdateStatus changes an order's status on behalf of a partner/supplier.
// Reuses the same mutation pattern as UpdateStatus but bypasses claims-role checks.
// Partners may NOT soft-complete (COMPLETED requires fiscal flow).
func (s *Service) PartnerUpdateStatus(ctx context.Context, orderID string, req UpdateStatusRequest) (UpdateStatusResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return UpdateStatusResponse{}, errors.New("order_id required")
	}

	nextStatus, err := req.Validate()
	if err != nil {
		return UpdateStatusResponse{}, err
	}

	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return UpdateStatusResponse{}, fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !ok {
		return UpdateStatusResponse{}, ErrOrderNotFound
	}

	if err := ValidateStatusTransition(string(current.Status), string(nextStatus), TransitionOpts{}); err != nil {
		return UpdateStatusResponse{}, err
	}

	// Partners cannot soft-complete; fiscal hard-gate is mandatory.
	if nextStatus == StatusCompleted {
		return UpdateStatusResponse{}, fmt.Errorf(
			"%w: partner_cannot_soft_complete; use the fiscal flow",
			ErrInvalidStatusTransition,
		)
	}

	// If status transition is a cancel, delegate to the richer cancel path.
	if nextStatus == StatusCancelled {
		return s.PartnerCancelOrder(ctx, orderID, req.Reason)
	}

	if current.Status == nextStatus {
		return UpdateStatusResponse{
			OrderID:        current.OrderID,
			PreviousStatus: current.Status,
			Status:         current.Status,
			Version:        current.Version,
			UpdatedAt:      current.UpdatedAt.Format(time.RFC3339Nano),
			EventType:      events.EventOrderStatusChanged,
		}, nil
	}

	prevStatus := current.Status
	previousDriverID := strings.TrimSpace(current.DriverID)
	current.Status = nextStatus
	current.UpdatedAt = s.now()
	s.applyHandoffLifecycle(&current, prevStatus, previousDriverID)

	current.TransitionReason = req.Reason
	current.TransitionActorRole = "PARTNER"
	current.TransitionActorID = "partner_api"

	err = s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: current.UpdatedAt.Format(time.RFC3339Nano)},
			OrderID:        current.OrderID,
			SupplierID:     current.SupplierID,
			RetailerID:     current.RetailerID,
			DriverID:       current.DriverID,
			PreviousStatus: string(prevStatus),
			Status:         string(current.Status),
			Reason:         strings.TrimSpace(req.Reason),
			ActorRole:      "PARTNER",
			ActorID:        "partner_api",
		}); err != nil {
			return err
		}
		if ab, ok := txn.(outbox.AuditBuffer); ok {
			return outbox.WriteAudit(ctx, ab, current.SupplierID, "partner_api", "PARTNER", "ORDER_STATUS_CHANGED", "Order", current.OrderID, map[string]string{"from": string(prevStatus), "to": string(current.Status)})
		}
		return nil
	})
	if err != nil {
		return UpdateStatusResponse{}, fmt.Errorf("apply partner update status %s: %w", orderID, err)
	}

	if s.cache != nil {
		s.cache.Invalidate(ctx, append(
			[]string{
				retailerOrdersKey(current.RetailerID),
				supplierOrdersKey(current.SupplierID),
			},
			dashboardCacheKeys(current.SupplierID, current.WarehouseID)...,
		)...)
	}

	s.log.Info("order status updated by partner",
		"order_id", current.OrderID,
		"supplier_id", current.SupplierID,
		"retailer_id", current.RetailerID,
		"prev_status", prevStatus,
		"status", current.Status,
	)

	return UpdateStatusResponse{
		OrderID:        current.OrderID,
		PreviousStatus: prevStatus,
		Status:         current.Status,
		Version:        current.Version + 1,
		UpdatedAt:      current.UpdatedAt.Format(time.RFC3339Nano),
		EventType:      events.EventOrderStatusChanged,
	}, nil
}
