package payload

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

// OrderExpectationReader loads order rows for delivery-expectation enrichment.
type OrderExpectationReader interface {
	GetOrder(ctx context.Context, orderID string) (order.Order, bool, error)
}

// SetOrderExpectationReader wires order lookup for manifest detail enrichment.
func (s *Service) SetOrderExpectationReader(reader OrderExpectationReader) {
	s.orderReader = reader
}

func (s *Service) enrichManifestWireExpectations(ctx context.Context, wire manifest.Wire) manifest.Wire {
	if s == nil || s.orderReader == nil || len(wire.Orders) == 0 {
		return wire
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now()
	}
	for i := range wire.Orders {
		o, ok, err := s.orderReader.GetOrder(ctx, wire.Orders[i].OrderID)
		if err != nil || !ok {
			continue
		}
		exp := order.ComputeDeliveryExpectation(now, o)
		w := manifest.DeliveryExpectationWire{
			Kind:                 exp.Kind,
			TargetDate:           exp.TargetDate,
			TargetLabel:          exp.TargetLabel,
			ModeLabel:            exp.ModeLabel,
			ReceivingWindowOpen:  exp.ReceivingWindowOpen,
			ReceivingWindowClose: exp.ReceivingWindowClose,
			Delayed:              exp.Delayed,
			DelayReason:          exp.DelayReason,
			Urgency:              exp.Urgency,
			BadgeLabel:           exp.BadgeLabel,
		}
		wire.Orders[i].DeliveryExpectation = &w
	}
	return wire
}
