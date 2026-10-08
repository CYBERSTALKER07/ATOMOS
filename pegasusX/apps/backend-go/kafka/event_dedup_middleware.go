package kafka

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/segmentio/kafka-go"
)

// WithEventDedup wraps a handler with cross-pod event deduplication scoped to
// consumerGroup so parallel consumer groups on the same topic stay independent.
// It also binds distributed trace context from message headers/envelope onto ctx.
func WithEventDedup(store EventDedupStore, consumerGroup string, handler EventHandler) EventHandler {
	if store == nil || handler == nil {
		return handler
	}
	return func(ctx context.Context, msg kafka.Message) error {
		ctx = WithTraceFromMessage(ctx, msg)
		key := DedupKeyForConsumerGroup(consumerGroup, msg.Topic, msg.Partition, msg.Offset)
		eid := headerValue(msg, "event_id")
		if eid == "" {
			eid = headerValue(msg, "event-id")
		}
		if eid == "" {
			eid = eventIDFromPayload(msg.Value)
		}
		if eid != "" {
			if k := DedupKeyForEventID(consumerGroup, msg.Topic, eid); k != "" {
				key = k
			}
		}
		ok, err := store.ShouldProcess(ctx, key)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		err = handler(ctx, msg)
		if err != nil {
			_ = store.Release(ctx, key)
			return err
		}
		return nil
	}
}

func headerValue(msg kafka.Message, name string) string {
	for _, h := range msg.Headers {
		if strings.EqualFold(h.Key, name) {
			return string(h.Value)
		}
	}
	return ""
}

func eventIDFromPayload(value []byte) string {
	if len(value) == 0 || value[0] != '{' {
		return ""
	}
	var env struct {
		EventID string `json:"event_id"`
		ID      string `json:"id"`
	}
	if err := json.Unmarshal(value, &env); err == nil {
		if env.EventID != "" {
			return strings.TrimSpace(env.EventID)
		}
		if env.ID != "" {
			return strings.TrimSpace(env.ID)
		}
	}
	return ""
}

