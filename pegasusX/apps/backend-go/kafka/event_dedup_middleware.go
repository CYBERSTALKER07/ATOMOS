package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

// WithEventDedup wraps a handler with cross-pod event deduplication.
func WithEventDedup(store EventDedupStore, handler EventHandler) EventHandler {
	if store == nil || handler == nil {
		return handler
	}
	return func(ctx context.Context, msg kafka.Message) error {
		key := DedupKeyForMessage(msg.Topic, msg.Partition, msg.Offset)
		ok, err := store.ShouldProcess(ctx, key)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		return handler(ctx, msg)
	}
}
