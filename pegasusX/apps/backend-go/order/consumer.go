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
	if envelope.Type == events.EventPaymentCleared {
		var payloadData struct {
			OrderID string `json:"order_id"`
			Gateway string `json:"gateway"`
		}
		if err := json.Unmarshal(msg.Value, &payloadData); err != nil {
			c.log.ErrorContext(ctx, "failed to unmarshal payment cleared payload", "err", err)
			return nil
		}
		if payloadData.OrderID != "" {
			return c.service.SettleExternalPayment(ctx, payloadData.OrderID, payloadData.Gateway)
		}
	}
	return nil
}
