package warehouse

import (
	"context"
	"log/slog"

	pegasuskafka "github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	kafka "github.com/segmentio/kafka-go"
)

// EventConsumer orchestrates warehouse asynchronous updates.
type EventConsumer struct {
	svc *Service
	log *slog.Logger
}

// NewEventConsumer constructs the warehouse module consumer.
func NewEventConsumer(svc *Service, log *slog.Logger) *EventConsumer {
	return &EventConsumer{
		svc: svc,
		log: log,
	}
}

// HandleEvent unmarshals Kafka messages and routes them.
func (c *EventConsumer) HandleEvent(ctx context.Context, msg kafka.Message) error {
	envelope, err := pegasuskafka.ParseEnvelope(msg.Value)
	if err != nil {
		c.log.Warn("warehouse consumer payload parsing failed", "err", err, "topic", msg.Topic)
		return nil
	}
	if envelope.Type == "SUPPLY_REQUEST_ACCEPTED" {
		return c.svc.HandleSupplyRequestAccepted(ctx, msg.Value)
	}
	return nil
}
