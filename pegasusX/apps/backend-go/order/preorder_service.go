package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// ConfirmAIOrder confirms an AI-created future order for the retailer.
func (s *Service) ConfirmAIOrder(ctx context.Context, retailerID string, req ConfirmAIOrderRequest) (RetailerOrderLifecycleResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return RetailerOrderLifecycleResponse{}, errors.New("order_id required")
	}
	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !ok {
		return RetailerOrderLifecycleResponse{}, ErrOrderNotFound
	}
	if current.RetailerID != strings.TrimSpace(retailerID) {
		return RetailerOrderLifecycleResponse{}, ErrOrderForbidden
	}
	if current.Source != OrderSourceAIPreorder {
		return RetailerOrderLifecycleResponse{}, ErrInvalidStatusTransition
	}
	if current.ConfirmationStatus == ConfirmationStatusConfirmed || current.ConfirmationStatus == ConfirmationStatusAutoConfirmed {
		return lifecycleResponse(current, current.Version, false), nil
	}
	if current.ConfirmationStatus != ConfirmationStatusPending {
		return RetailerOrderLifecycleResponse{}, ErrInvalidStatusTransition
	}
	prevLines := append([]LineItem(nil), current.LineItems...)
	prevTotal := current.TotalMinor
	if len(req.LineItems) > 0 {
		lineItems, _, err := s.normalizeAndQuoteLineItems(ctx, req.LineItems, nil)
		if err != nil {
			return RetailerOrderLifecycleResponse{}, err
		}
		if err := s.validateOrderPackMultiples(ctx, lineItems); err != nil {
			return RetailerOrderLifecycleResponse{}, err
		}
		requestedLines := lineItems
		fulfillable, _, err := s.applyPreorderInventoryGuard(ctx, current.SupplierID, current.WarehouseID, current.LineItems, requestedLines)
		if err != nil {
			return RetailerOrderLifecycleResponse{}, err
		}
		current.LineItems = fulfillable
		tot := totalMinorForLines(fulfillable)
		if tot > 0 && current.DeliveryFeeMinor > 0 {
			tot += current.DeliveryFeeMinor
		}
		current.TotalMinor = tot
	} else {
		if err := s.validateOrderPackMultiples(ctx, current.LineItems); err != nil {
			return RetailerOrderLifecycleResponse{}, err
		}
		if s.spannerClient != nil {
			fulfillable, _, err := s.applyPreorderInventoryGuard(ctx, current.SupplierID, current.WarehouseID, current.LineItems, current.LineItems)
			if err != nil {
				return RetailerOrderLifecycleResponse{}, err
			}
			current.LineItems = fulfillable
			tot := totalMinorForLines(fulfillable)
			if tot > 0 && current.DeliveryFeeMinor > 0 {
				tot += current.DeliveryFeeMinor
			}
			current.TotalMinor = tot
		}
	}
	if req.AutoConfirmOptIn != nil {
		current.AutoConfirmOptIn = *req.AutoConfirmOptIn
	}
	if strings.TrimSpace(req.RequestedDeliveryDate) != "" {
		requestedDeliveryDate, err := parseOptionalRFC3339(req.RequestedDeliveryDate)
		if err != nil {
			return RetailerOrderLifecycleResponse{}, fmt.Errorf("parse requested_delivery_date: %w", err)
		}
		current.RequestedDeliveryDate = requestedDeliveryDate
	}
	current.ConfirmationStatus = ConfirmationStatusConfirmed
	current.AutoConfirmAt = nil
	decisionAt := s.now()
	current.DecisionAt = &decisionAt
	current.DecisionBy = strings.TrimSpace(retailerID)
	current.UpdatedAt = decisionAt
	toConfirm := current
	if err := s.repo.UpdateOrderWithTxn(ctx, toConfirm, nil, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if s.spannerClient != nil {
			if err := reconcilePreorderReservationsInTxn(ctx, txn, toConfirm.SupplierID, toConfirm.WarehouseID, toConfirm.OrderID, toConfirm.Source, prevLines, toConfirm.LineItems); err != nil {
				return err
			}
		}
		if s.credit != nil && txn != nil && toConfirm.TotalMinor != prevTotal {
			if err := s.credit.AdjustReserveInTxn(ctx, txn, toConfirm.OrderID, toConfirm.TotalMinor); err != nil {
				return fmt.Errorf("adjust credit reserve on confirm ai order: %w", err)
			}
		}
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, toConfirm.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:             events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: toConfirm.UpdatedAt.Format(time.RFC3339Nano)},
			OrderID:               toConfirm.OrderID,
			SupplierID:            toConfirm.SupplierID,
			RetailerID:            toConfirm.RetailerID,
			PreviousStatus:        string(toConfirm.Status),
			Status:                string(toConfirm.Status),
			Reason:                "AI_CONFIRMED",
			ActorRole:             string(auth.RoleRetailer),
			ActorID:               retailerID,
			OrderSource:           string(toConfirm.Source),
			ConfirmationStatus:    string(toConfirm.ConfirmationStatus),
			RequestedDeliveryDate: formatOptionalRFC3339(toConfirm.RequestedDeliveryDate),
		})
	}); err != nil {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("confirm ai order %s: %w", orderID, err)
	}
	toConfirm.Version++
	s.afterOrderMutation(ctx, toConfirm)
	return lifecycleResponse(toConfirm, toConfirm.Version, false), nil
}

// RejectAIOrder rejects an AI-created future order.
func (s *Service) RejectAIOrder(ctx context.Context, retailerID string, req RejectAIOrderRequest) (RetailerOrderLifecycleResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return RetailerOrderLifecycleResponse{}, errors.New("order_id required")
	}
	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !ok {
		return RetailerOrderLifecycleResponse{}, ErrOrderNotFound
	}
	if current.RetailerID != strings.TrimSpace(retailerID) {
		return RetailerOrderLifecycleResponse{}, ErrOrderForbidden
	}
	if current.Source != OrderSourceAIPreorder || current.ConfirmationStatus != ConfirmationStatusPending {
		return RetailerOrderLifecycleResponse{}, ErrInvalidStatusTransition
	}
	previousStatus := current.Status
	if err := ValidateStatusTransition(string(previousStatus), string(StatusCancelled), TransitionOpts{
		Actor:  strings.TrimSpace(retailerID),
		Reason: strings.TrimSpace(req.Reason),
	}); err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	current.ConfirmationStatus = ConfirmationStatusRejected
	current.Status = StatusCancelled
	decisionAt := s.now()
	current.DecisionAt = &decisionAt
	current.DecisionBy = strings.TrimSpace(retailerID)
	current.AutoConfirmAt = nil
	current.UpdatedAt = decisionAt
	if err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:             events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: current.UpdatedAt.Format(time.RFC3339Nano)},
			OrderID:               current.OrderID,
			SupplierID:            current.SupplierID,
			RetailerID:            current.RetailerID,
			PreviousStatus:        string(previousStatus),
			Status:                string(current.Status),
			Reason:                strings.TrimSpace(req.Reason),
			ActorRole:             string(auth.RoleRetailer),
			ActorID:               retailerID,
			OrderSource:           string(current.Source),
			ConfirmationStatus:    string(current.ConfirmationStatus),
			RequestedDeliveryDate: formatOptionalRFC3339(current.RequestedDeliveryDate),
		})
	}); err != nil {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("reject ai order %s: %w", orderID, err)
	}
	s.afterOrderMutation(ctx, current)
	return lifecycleResponse(current, current.Version+1, false), nil
}

// EditPreorder updates a scheduled manual preorder.
func (s *Service) EditPreorder(ctx context.Context, retailerID string, req EditPreorderRequest) (RetailerOrderLifecycleResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return RetailerOrderLifecycleResponse{}, errors.New("order_id required")
	}
	lineItems, _, err := s.normalizeAndQuoteLineItems(ctx, req.LineItems, nil)
	if err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	if err := s.validateOrderPackMultiples(ctx, lineItems); err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	requestedDeliveryDate, err := parseOptionalRFC3339(req.RequestedDeliveryDate)
	if err != nil || requestedDeliveryDate == nil {
		if err == nil {
			err = errors.New("requested_delivery_date required")
		}
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("parse requested_delivery_date: %w", err)
	}
	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !ok {
		return RetailerOrderLifecycleResponse{}, ErrOrderNotFound
	}
	if current.RetailerID != strings.TrimSpace(retailerID) {
		return RetailerOrderLifecycleResponse{}, ErrOrderForbidden
	}
	if current.Source != OrderSourceManualPreorder || current.Status != StatusScheduled {
		return RetailerOrderLifecycleResponse{}, ErrInvalidStatusTransition
	}
	loc, locErr := resolveCalendarLocation(ctx, current.SupplierID, current.Timezone)
	if locErr != nil {
		return RetailerOrderLifecycleResponse{}, locErr
	}
	if PreorderEditLocked(s.now(), loc, current) {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("%w: preorder edit locked within %d days of delivery", ErrInvalidStatusTransition, PreorderEditLockDays)
	}
	if current.ConfirmationStatus == ConfirmationStatusRejected {
		return RetailerOrderLifecycleResponse{}, ErrInvalidStatusTransition
	}
	if req.AutoConfirmOptIn != nil {
		current.AutoConfirmOptIn = *req.AutoConfirmOptIn
	}
	current.RequestedDeliveryDate = requestedDeliveryDate
	current.UpdatedAt = s.now()
	toSave := current
	updated, err := s.updatePreorderLines(ctx, toSave, lineItems, func(txn outbox.TxnBuffer, updated Order) error {
		return emitPreorderEvent(ctx, txn, events.EventPreOrderEdited, updated, string(auth.RoleRetailer), retailerID)
	})
	if err != nil {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("edit preorder %s: %w", orderID, err)
	}
	updated.Version++
	s.afterOrderMutation(ctx, updated)
	return lifecycleResponse(updated, updated.Version, false), nil
}

// ConfirmPreorder confirms a draft manual preorder.
func (s *Service) ConfirmPreorder(ctx context.Context, retailerID string, req ConfirmPreorderRequest) (RetailerOrderLifecycleResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return RetailerOrderLifecycleResponse{}, errors.New("order_id required")
	}
	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !ok {
		return RetailerOrderLifecycleResponse{}, ErrOrderNotFound
	}
	if current.RetailerID != strings.TrimSpace(retailerID) {
		return RetailerOrderLifecycleResponse{}, ErrOrderForbidden
	}
	if current.Source != OrderSourceManualPreorder || current.Status != StatusScheduled {
		return RetailerOrderLifecycleResponse{}, ErrInvalidStatusTransition
	}
	if current.ConfirmationStatus == ConfirmationStatusConfirmed || current.ConfirmationStatus == ConfirmationStatusAutoConfirmed {
		return lifecycleResponse(current, current.Version, false), nil
	}
	if current.ConfirmationStatus != ConfirmationStatusDraft && current.ConfirmationStatus != ConfirmationStatusPending {
		return RetailerOrderLifecycleResponse{}, ErrInvalidStatusTransition
	}
	decisionAt := s.now()
	current.ConfirmationStatus = ConfirmationStatusConfirmed
	current.DecisionAt = &decisionAt
	current.DecisionBy = strings.TrimSpace(retailerID)
	current.UpdatedAt = decisionAt
	if err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		return emitPreorderEvent(ctx, txn, events.EventPreOrderConfirmed, current, string(auth.RoleRetailer), retailerID)
	}); err != nil {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("confirm preorder %s: %w", orderID, err)
	}
	s.afterOrderMutation(ctx, current)
	return lifecycleResponse(current, current.Version+1, false), nil
}

// ListRetailerAIPredictions returns pending AI preorders for retailer review.
func (s *Service) ListRetailerAIPredictions(ctx context.Context, retailerID string, limit int) ([]RetailerAIPrediction, error) {
	if limit <= 0 {
		limit = 25
	}
	orders, err := s.repo.ListRetailerOrders(ctx, strings.TrimSpace(retailerID), limit*4)
	if err != nil {
		return nil, fmt.Errorf("list retailer orders for ai predictions: %w", err)
	}
	items := make([]RetailerAIPrediction, 0, limit)
	for _, orderRecord := range orders {
		if orderRecord.Source != OrderSourceAIPreorder || orderRecord.ConfirmationStatus != ConfirmationStatusPending {
			continue
		}
		items = append(items, retailerPrediction(orderRecord))
		if len(items) >= limit {
			break
		}
	}
	return items, nil
}

// WarehouseDemandForecast projects future-dated demand for one warehouse.
func (s *Service) WarehouseDemandForecast(ctx context.Context, warehouseID string, start time.Time, days int) ([]WarehouseDemandDay, error) {
	if days <= 0 {
		days = 7
	}
	from := start.UTC().Truncate(24 * time.Hour)
	to := from.AddDate(0, 0, days)
	orders, err := s.repo.ListWarehouseOrdersByDeliveryWindow(ctx, strings.TrimSpace(warehouseID), from, to, 500)
	if err != nil {
		return nil, fmt.Errorf("list warehouse orders by delivery window: %w", err)
	}
	buckets := make(map[string]*WarehouseDemandDay, days)
	for i := 0; i < days; i++ {
		date := from.AddDate(0, 0, i).Format("2006-01-02")
		buckets[date] = &WarehouseDemandDay{Date: date, Currency: s.currency}
	}
	for _, orderRecord := range orders {
		if orderRecord.RequestedDeliveryDate == nil {
			continue
		}
		date := orderRecord.RequestedDeliveryDate.UTC().Format("2006-01-02")
		bucket, ok := buckets[date]
		if !ok {
			continue
		}
		var units int64
		for _, item := range orderRecord.LineItems {
			units += item.Quantity
		}
		bucket.ProjectedUnits += units
		bucket.ProjectedRevenue += orderRecord.TotalMinor
		switch orderRecord.ConfirmationStatus {
		case ConfirmationStatusConfirmed, ConfirmationStatusAutoConfirmed:
			bucket.CommittedUnits += units
		case ConfirmationStatusDraft, ConfirmationStatusPending:
			bucket.PendingConfirmationUnits += units
		}
	}
	series := make([]WarehouseDemandDay, 0, days)
	for i := 0; i < days; i++ {
		date := from.AddDate(0, 0, i).Format("2006-01-02")
		series = append(series, *buckets[date])
	}
	return series, nil
}

// AutoConfirmDueOrders promotes due AI/auto-order proposals to auto-confirmed
// ONLY when AutoConfirmOptIn == true; unconfirmed opt-out proposals expire cleanly.
func (s *Service) AutoConfirmDueOrders(ctx context.Context, limit int) error {
	if limit <= 0 {
		limit = 50
	}
	orders, err := s.repo.ListDueAutoConfirmOrders(ctx, s.now(), limit)
	if err != nil {
		return fmt.Errorf("list due auto-confirm orders: %w", err)
	}
	for _, orderRecord := range orders {
		if (orderRecord.Source != OrderSourceAIPreorder && orderRecord.Source != OrderSourceAutoOrder) ||
			orderRecord.ConfirmationStatus != ConfirmationStatusPending {
			continue
		}
		if orderRecord.Status == StatusCancelled || orderRecord.Status == StatusCompleted {
			continue
		}
		if !orderRecord.AutoConfirmOptIn {
			if _, expErr := s.expireUnconfirmedProposal(ctx, orderRecord, s.now(), "AUTO_CONFIRM_OPT_OUT_EXPIRED", "system:auto_confirm_opt_out_expired"); expErr != nil {
				s.log.Warn("expire opt-out preorder failed", "order_id", orderRecord.OrderID, "err", expErr)
			}
			continue
		}
		updated := orderRecord
		decisionAt := s.now()
		updated.ConfirmationStatus = ConfirmationStatusAutoConfirmed
		updated.DecisionAt = &decisionAt
		updated.DecisionBy = "SYSTEM"
		updated.AutoConfirmAt = nil
		updated.UpdatedAt = decisionAt
		if updateErr := s.repo.UpdateOrder(ctx, updated, nil, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(ctx, txn, events.AggregateOrder, updated.OrderID, events.TopicMain, events.OrderEvent{
				BaseEvent:             events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: updated.UpdatedAt.Format(time.RFC3339Nano)},
				OrderID:               updated.OrderID,
				SupplierID:            updated.SupplierID,
				RetailerID:            updated.RetailerID,
				PreviousStatus:        string(updated.Status),
				Status:                string(updated.Status),
				Reason:                "PREORDER_AUTO_CONFIRMED",
				ActorRole:             "SYSTEM",
				ActorID:               "system:auto_confirm",
				OrderSource:           string(updated.Source),
				ConfirmationStatus:    string(updated.ConfirmationStatus),
				RequestedDeliveryDate: formatOptionalRFC3339(updated.RequestedDeliveryDate),
			})
		}); updateErr != nil {
			s.log.Warn("auto confirm preorder failed", "order_id", updated.OrderID, "err", updateErr)
			continue
		}
		s.afterOrderMutation(ctx, updated)
	}
	return nil
}

type txnOrderMutator interface {
	mutateOrderWithTxn(
		ctx context.Context,
		orderID string,
		expectedVersion int64,
		fn func(context.Context, *spanner.ReadWriteTransaction, *Order) (func(outbox.TxnBuffer) error, error),
	) (Order, error)
}

func (s *Service) expireUnconfirmedProposal(ctx context.Context, o Order, now time.Time, reason, actorID string) (Order, error) {
	if o.Status == StatusCancelled {
		return o, nil
	}
	applyExpiry := func(target *Order) (Status, bool, error) {
		if target.Status == StatusCancelled {
			return StatusCancelled, false, nil
		}
		if target.AutoConfirmOptIn ||
			target.ConfirmationStatus == ConfirmationStatusConfirmed ||
			target.ConfirmationStatus == ConfirmationStatusAutoConfirmed {
			return target.Status, false, nil
		}
		prev := target.Status
		if err := ValidateStatusTransition(string(prev), string(StatusCancelled), TransitionOpts{
			Actor:  actorID,
			Reason: reason,
		}); err != nil {
			return "", false, err
		}
		decAt := now
		target.Status = StatusCancelled
		target.ConfirmationStatus = ConfirmationStatusRejected
		target.AutoConfirmAt = nil
		target.DecisionAt = &decAt
		target.DecisionBy = actorID
		target.UpdatedAt = now
		target.TransitionReason = reason
		target.TransitionActorRole = "SYSTEM"
		target.TransitionActorID = actorID
		target.TransitionEventKind = "EXPIRE"
		return prev, true, nil
	}
	emitExpiry := func(txn outbox.TxnBuffer, prev Status, updated Order) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, updated.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:             events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: updated.UpdatedAt.Format(time.RFC3339Nano)},
			OrderID:               updated.OrderID,
			SupplierID:            updated.SupplierID,
			RetailerID:            updated.RetailerID,
			WarehouseID:           updated.WarehouseID,
			PreviousStatus:        string(prev),
			Status:                string(updated.Status),
			Reason:                reason,
			ActorRole:             "SYSTEM",
			ActorID:               actorID,
			OrderSource:           string(updated.Source),
			ConfirmationStatus:    string(updated.ConfirmationStatus),
			RequestedDeliveryDate: formatOptionalRFC3339(updated.RequestedDeliveryDate),
		})
	}

	if mut, ok := s.repo.(txnOrderMutator); ok {
		changed := false
		updated, err := mut.mutateOrderWithTxn(ctx, o.OrderID, 0, func(txCtx context.Context, txn *spanner.ReadWriteTransaction, curr *Order) (func(outbox.TxnBuffer) error, error) {
			prev, didChange, err := applyExpiry(curr)
			if err != nil || !didChange {
				return nil, err
			}
			changed = true
			if s.credit != nil {
				if err := s.credit.AdjustReserveInTxn(txCtx, txn, curr.OrderID, 0); err != nil {
					return nil, fmt.Errorf("release credit reserve on proposal expiry: %w", err)
				}
			}
			snapshot := *curr
			return func(buf outbox.TxnBuffer) error {
				return emitExpiry(buf, prev, snapshot)
			}, nil
		})
		if err != nil {
			return o, err
		}
		if changed {
			s.afterOrderMutation(ctx, updated)
		}
		return updated, nil
	}

	prev, didChange, err := applyExpiry(&o)
	if err != nil || !didChange {
		return o, err
	}
	if err := s.repo.UpdateOrder(ctx, o, nil, func(txn outbox.TxnBuffer) error {
		return emitExpiry(txn, prev, o)
	}); err != nil {
		return o, err
	}
	if s.credit != nil {
		if cerr := s.credit.ReleaseReserve(ctx, o.OrderID); cerr != nil {
			s.log.Warn("credit reserve release on proposal expiry failed", "order_id", o.OrderID, "err", cerr)
		}
	}
	o.Version++
	s.afterOrderMutation(ctx, o)
	return o, nil
}
