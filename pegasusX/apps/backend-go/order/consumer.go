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
	ctx = pegasuskafka.WithTraceFromMessage(ctx, msg)
	envelope, err := pegasuskafka.EnvelopeFromMessage(msg)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to parse event envelope", "err", err, "topic", msg.Topic, "offset", msg.Offset)
		return err
	}
	switch envelope.Type {
	case events.EventPaymentCleared:
		var payload struct {
			OrderID     string `json:"order_id"`
			Gateway     string `json:"gateway"`
			AmountMinor int64  `json:"amount_minor"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			c.log.ErrorContext(ctx, "failed to unmarshal payment cleared payload", "err", err)
			return err
		}
		if payload.OrderID != "" {
			return c.service.SettleExternalPayment(ctx, payload.OrderID, payload.Gateway, payload.AmountMinor)
		}
	case events.EventFiscalReceiptRequested:
		var payload events.FiscalReceiptEvent
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			c.log.ErrorContext(ctx, "failed to unmarshal fiscal receipt requested payload", "err", err)
			return err
		}
		if payload.OrderID != "" && payload.AttemptID != "" {
			return c.service.ApplyFiscalWorkerResult(ctx, payload.OrderID, payload.AttemptID)
		}
	case events.EventPaymentFailed:
		var fin events.FinanceEvent
		if err := json.Unmarshal(msg.Value, &fin); err != nil {
			c.log.ErrorContext(ctx, "failed to unmarshal payment failed payload", "err", err)
			return err
		}
		if fin.OrderID != "" {
			return c.service.HandleExternalPaymentFailed(ctx, fin.OrderID, fin.Gateway, fin.Source)
		}
	case events.EventDeliveryDisputed:
		var disputed events.OrderEvent
		if err := json.Unmarshal(msg.Value, &disputed); err != nil {
			c.log.ErrorContext(ctx, "failed to unmarshal delivery disputed payload", "err", err)
			return err
		}
		if disputed.OrderID != "" {
			return c.service.HandleDeliveryDisputed(ctx, disputed.OrderID, disputed.Reason, disputed.Action)
		}
	}
	return nil
}
