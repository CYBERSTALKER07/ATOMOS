package supplier

import "context"

// ListOrdersForPulse returns recent supplier orders for the network pulse timeline.
func (s *Service) ListOrdersForPulse(ctx context.Context, supplierID string, limit int) ([]SupplierOrder, error) {
	reader, ok := s.repo.(supplierOrderReader)
	if !ok {
		return nil, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	orders, err := reader.ListOrders(ctx, supplierID, limit, 0)
	if err != nil {
		return nil, err
	}
	return orders, nil
}
