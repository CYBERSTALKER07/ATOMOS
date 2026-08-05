package partner

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/segmentio/kafka-go"
)

// EventConsumer enqueues outbound webhook deliveries from Kafka domain events.
type EventConsumer struct {
	svc *Service
	log *slog.Logger
}

func NewEventConsumer(svc *Service, log *slog.Logger) *EventConsumer {
	if log == nil {
		log = slog.Default()
	}
	return &EventConsumer{svc: svc, log: log}
}

var allowlistedWebhookEvents = map[string]bool{
	events.EventOrderCreated:       true,
	events.EventOrderStatusChanged: true,
	events.EventClaimFiled:         true,
	events.EventPaymentCleared:     true,
}

// HandleEvent is a kafka.EventHandler.
func (c *EventConsumer) HandleEvent(ctx context.Context, msg kafka.Message) error {
	if c == nil || c.svc == nil {
		return nil
	}
	var envelope map[string]any
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		return nil
	}
	eventType, _ := envelope["type"].(string)
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		eventType = strings.TrimSpace(headerString(msg, "event_type"))
	}
	if !allowlistedWebhookEvents[eventType] {
		return nil
	}
	eventID := strings.TrimSpace(headerString(msg, "event_id"))
	if eventID == "" {
		if v, ok := envelope["event_id"].(string); ok {
			eventID = strings.TrimSpace(v)
		}
	}
	if eventID == "" {
		if v, ok := envelope["order_id"].(string); ok && v != "" {
			eventID = eventType + ":" + v
		} else {
			eventID = eventType + ":" + string(msg.Key)
		}
	}
	if err := c.svc.EnqueueEvent(ctx, eventID, eventType, envelope); err != nil {
		c.log.Warn("partner webhook enqueue", "err", err, "event_type", eventType)
	}
	c.svc.EnqueueEdiFromEvent(ctx, eventType, envelope)
	return nil
}

func headerString(msg kafka.Message, key string) string {
	for _, h := range msg.Headers {
		if strings.EqualFold(h.Key, key) {
			return string(h.Value)
		}
	}
	return ""
}
