package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
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
	if len(req.LineItems) > 0 {
		lineItems, total, err := s.normalizeAndQuoteLineItems(ctx, req.LineItems)
		if err != nil {
			return RetailerOrderLifecycleResponse{}, err
		}
		current.LineItems = lineItems
		current.TotalMinor = total
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
	if err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:             events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: current.UpdatedAt.Format(time.RFC3339Nano)},
			OrderID:               current.OrderID,
			SupplierID:            current.SupplierID,
			RetailerID:            current.RetailerID,
			PreviousStatus:        string(current.Status),
			Status:                string(current.Status),
			Reason:                "AI_CONFIRMED",
			ActorRole:             string(auth.RoleRetailer),
			ActorID:               retailerID,
			OrderSource:           string(current.Source),
			ConfirmationStatus:    string(current.ConfirmationStatus),
			RequestedDeliveryDate: formatOptionalRFC3339(current.RequestedDeliveryDate),
		})
	}); err != nil {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("confirm ai order %s: %w", orderID, err)
	}
	s.afterOrderMutation(ctx, current)
	return lifecycleResponse(current, current.Version+1, false), nil
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
			PreviousStatus:        string(StatusPending),
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
	lineItems, total, err := s.normalizeAndQuoteLineItems(ctx, req.LineItems)
	if err != nil {
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
	loc := proximity.TashkentLocation
	if current.Timezone != "" {
		if l, err := time.LoadLocation(current.Timezone); err == nil {
			loc = l
		}
	}
	if PreorderEditLocked(s.now(), loc, current) {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("%w: preorder edit locked within %d days of delivery", ErrInvalidStatusTransition, PreorderEditLockDays)
	}
	if current.ConfirmationStatus == ConfirmationStatusRejected {
		return RetailerOrderLifecycleResponse{}, ErrInvalidStatusTransition
	}
	current.LineItems = lineItems
	current.TotalMinor = total
	current.RequestedDeliveryDate = requestedDeliveryDate
	current.UpdatedAt = s.now()
	if err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		return emitPreorderEvent(ctx, txn, events.EventPreOrderEdited, current, string(auth.RoleRetailer), retailerID)
	}); err != nil {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("edit preorder %s: %w", orderID, err)
	}
	s.afterOrderMutation(ctx, current)
	return lifecycleResponse(current, current.Version+1, false), nil
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

// AutoConfirmDueOrders promotes due AI preorders to auto-confirmed.
func (s *Service) AutoConfirmDueOrders(ctx context.Context, limit int) error {
	if limit <= 0 {
		limit = 50
	}
	orders, err := s.repo.ListDueAutoConfirmOrders(ctx, s.now(), limit)
	if err != nil {
		return fmt.Errorf("list due auto-confirm orders: %w", err)
	}
	for _, orderRecord := range orders {
		if orderRecord.Source != OrderSourceAIPreorder || orderRecord.ConfirmationStatus != ConfirmationStatusPending {
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
