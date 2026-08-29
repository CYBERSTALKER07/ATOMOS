package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

// WithEventDedup wraps a handler with cross-pod event deduplication scoped to
// consumerGroup so parallel consumer groups on the same topic stay independent.
func WithEventDedup(store EventDedupStore, consumerGroup string, handler EventHandler) EventHandler {
	if store == nil || handler == nil {
		return handler
	}
	return func(ctx context.Context, msg kafka.Message) error {
		key := DedupKeyForConsumerGroup(consumerGroup, msg.Topic, msg.Partition, msg.Offset)
		if eid := headerValue(msg, "event_id"); eid != "" {
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
		if h.Key == name {
			return string(h.Value)
		}
	}
	return ""
}
