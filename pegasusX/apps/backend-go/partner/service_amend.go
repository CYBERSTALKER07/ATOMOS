package partner

import (
	"context"
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

// CancelOrder cancels a given order. Only valid for suppliers or partners with write access.
func (s *Service) CancelOrder(ctx context.Context, p Principal, orderID, reason string) (order.UpdateStatusResponse, error) {
	if s.orders == nil {
		return order.UpdateStatusResponse{}, fmt.Errorf("orders_unavailable")
	}

	o, ok, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return order.UpdateStatusResponse{}, err
	}
	if !ok {
		return order.UpdateStatusResponse{}, errNotFound("order")
	}
	if !s.canAccessOrder(p, o) {
		return order.UpdateStatusResponse{}, errNotFound("order")
	}

	if p.TenantType == TenantRetailer {
		return order.UpdateStatusResponse{}, fmt.Errorf("retailer_cannot_cancel_via_partner_api")
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "partner_cancel"
	}

	return s.orders.PartnerCancelOrder(ctx, orderID, reason)
}

// UpdateOrderStatus transitions a given order's status. Only valid for suppliers or partners with write access.
func (s *Service) UpdateOrderStatus(ctx context.Context, p Principal, orderID string, req order.UpdateStatusRequest) (order.UpdateStatusResponse, error) {
	if s.orders == nil {
		return order.UpdateStatusResponse{}, fmt.Errorf("orders_unavailable")
	}

	o, ok, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return order.UpdateStatusResponse{}, err
	}
	if !ok {
		return order.UpdateStatusResponse{}, errNotFound("order")
	}
	if !s.canAccessOrder(p, o) {
		return order.UpdateStatusResponse{}, errNotFound("order")
	}

	if p.TenantType == TenantRetailer {
		return order.UpdateStatusResponse{}, fmt.Errorf("retailer_cannot_update_status_via_partner_api")
	}

	return s.orders.PartnerUpdateStatus(ctx, orderID, req)
}
