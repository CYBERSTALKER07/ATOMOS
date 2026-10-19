package kafka

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
)

// SpannerEventDedup uses Spanner's ConsumerInbox table for persistent cross-pod idempotency.
type SpannerEventDedup struct {
	client *spanner.Client
}

// NewSpannerEventDedup builds a Spanner-backed dedup store.
func NewSpannerEventDedup(client *spanner.Client) *SpannerEventDedup {
	return &SpannerEventDedup{client: client}
}

// ShouldProcess returns false when the event key already exists in ConsumerInbox.
func (s *SpannerEventDedup) ShouldProcess(ctx context.Context, key string) (bool, error) {
	if s == nil || s.client == nil || key == "" {
		return true, nil
	}
	if len(key) > 256 {
		key = key[:256]
	}

	var inserted bool
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.ReadRow(ctx, "ConsumerInbox", spanner.Key{key}, []string{"ProcessedAt"})
		if err == nil {
			inserted = false
			return nil
		}
		if spanner.ErrCode(err) != codes.NotFound {
			return err
		}

		err = txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("ConsumerInbox", []string{"DedupKey", "ProcessedAt"}, []interface{}{key, spanner.CommitTimestamp}),
		})
		if err != nil {
			return err
		}
		inserted = true
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("spanner event dedup: %w", err)
	}
	return inserted, nil
}

// Release deletes the key from ConsumerInbox, allowing future retries.
func (s *SpannerEventDedup) Release(ctx context.Context, key string) error {
	if s == nil || s.client == nil || key == "" {
		return nil
	}
	if len(key) > 256 {
		key = key[:256]
	}
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Delete("ConsumerInbox", spanner.Key{key}),
		})
	})
	return err
}
