package spannerutils

import (
	"context"

	"cloud.google.com/go/spanner"
)

// nil-client path returns ErrNilSpannerClient (see retry.go).

// Chunker configuration limits.
// Spanner limit is 80,000 mutations per transaction.
// A safe chunk size depends on the number of columns being written.
// We default to a safe 2,000 entities per chunk if 1 entity = ~20 columns.
const DefaultChunkSize = 2000

// RunChunkedTransaction splits a large slice of work into multiple
// ReadWriteTransactions to avoid hitting Spanner's 80,000 mutation limit.
// T is the slice element type.
// Caution: This breaks strict ACID guarantees for the *entire* collection of items.
// If chunk N fails, chunks 1..N-1 have already committed.
func RunChunkedTransaction[T any](ctx context.Context, client *spanner.Client, items []T, chunkSize int, fn func(context.Context, *spanner.ReadWriteTransaction, []T) error) error {
	if client == nil {
		return ErrNilSpannerClient
	}
	if len(items) == 0 {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunk := items[i:end]

		if err := RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return fn(ctx, txn, chunk)
		}); err != nil {
			// A failure in a later chunk does NOT rollback earlier chunks.
			// Callers must implement idempotency.
			return err
		}
	}

	return nil
}
