package order

import (
	"context"
	"fmt"
	"strings"
)

// HandleExternalPaymentFailed reacts to gateway payment failures for in-flight orders.
// Idempotent: no-op when the order is not awaiting payment.
func (s *Service) HandleExternalPaymentFailed(ctx context.Context, orderID, gateway, source string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil
	}
	orderRecord, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if !found {
		s.log.InfoContext(ctx, "skipping payment failed hook, order not found", "order_id", orderID)
		return nil
	}
	if orderRecord.Status != StatusAwaitingPayment {
		s.log.InfoContext(ctx, "skipping payment failed hook, order not awaiting payment",
			"order_id", orderID, "status", orderRecord.Status, "gateway", gateway, "source", source)
		return nil
	}
	s.invalidatePaymentCaches(ctx, orderRecord)
	s.afterOrderMutation(ctx, orderRecord)
	s.log.InfoContext(ctx, "recorded external payment failure",
		"order_id", orderID, "gateway", gateway, "source", source)
	return nil
}

// HandleDeliveryDisputed records chargeback/dispute side effects without rolling back delivery.
func (s *Service) HandleDeliveryDisputed(ctx context.Context, orderID, reason, action string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil
	}
	orderRecord, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("order not found: %s", orderID)
	}
	s.invalidatePaymentCaches(ctx, orderRecord)
	s.afterOrderMutation(ctx, orderRecord)
	s.log.WarnContext(ctx, "delivery disputed for order",
		"order_id", orderID, "reason", reason, "action", action, "status", orderRecord.Status)
	return nil
}

func (s *Service) invalidatePaymentCaches(ctx context.Context, orderRecord Order) {
	if s.cache == nil {
		return
	}
	keys := []string{"payment:order:" + orderRecord.OrderID}
	if orderRecord.RetailerID != "" {
		keys = append(keys, "payment:retailer:"+orderRecord.RetailerID)
	}
	s.cache.Invalidate(ctx, keys...)
}
