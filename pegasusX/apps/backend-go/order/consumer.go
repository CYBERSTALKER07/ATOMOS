package order

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	pegasuskafka "github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	kafka "github.com/segmentio/kafka-go"
)

type EventConsumer struct {
	service *Service
	log     *slog.Logger
}

func NewEventConsumer(service *Service, log *slog.Logger) *EventConsumer {
	return &EventConsumer{
		service: service,
		log:     log,
	}
}

func (c *EventConsumer) HandleEvent(ctx context.Context, msg kafka.Message) error {
	envelope, err := pegasuskafka.ParseEnvelope(msg.Value)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to parse event envelope", "err", err)
		return nil // poison pill
	}
	switch envelope.Type {
	case events.EventPaymentCleared:
		var payload struct {
			OrderID string `json:"order_id"`
			Gateway string `json:"gateway"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			c.log.ErrorContext(ctx, "failed to unmarshal payment cleared payload", "err", err)
			return nil
		}
		if payload.OrderID != "" {
			return c.service.SettleExternalPayment(ctx, payload.OrderID, payload.Gateway)
		}
	case events.EventPaymentFailed:
		var fin events.FinanceEvent
		if err := json.Unmarshal(msg.Value, &fin); err != nil {
			c.log.ErrorContext(ctx, "failed to unmarshal payment failed payload", "err", err)
			return nil
		}
		if fin.OrderID != "" {
			return c.service.HandleExternalPaymentFailed(ctx, fin.OrderID, fin.Gateway, fin.Source)
		}
	case events.EventDeliveryDisputed:
		var disputed events.OrderEvent
		if err := json.Unmarshal(msg.Value, &disputed); err != nil {
			c.log.ErrorContext(ctx, "failed to unmarshal delivery disputed payload", "err", err)
			return nil
		}
		if disputed.OrderID != "" {
			return c.service.HandleDeliveryDisputed(ctx, disputed.OrderID, disputed.Reason, disputed.Action)
		}
	}
	return nil
}
