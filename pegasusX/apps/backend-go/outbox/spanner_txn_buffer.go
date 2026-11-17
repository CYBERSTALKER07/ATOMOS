package outbox

import (
	"context"

	"cloud.google.com/go/spanner"
)

// SpannerTxnBuffer collects outbox events during a Spanner read-write
// transaction and flushes them as OutboxEvents mutations on the same txn, so a
// domain state change and its event commit atomically. This is the shared
// adapter for domains (ar, payout, ...) whose repositories write directly via
// spanner rather than through an order-style EmitJSON-in-txn seam.
type SpannerTxnBuffer struct {
	txn    *spanner.ReadWriteTransaction
	events []Event
}

func NewSpannerTxnBuffer(txn *spanner.ReadWriteTransaction) *SpannerTxnBuffer {
	return &SpannerTxnBuffer{txn: txn}
}

func (b *SpannerTxnBuffer) BufferOutbox(_ context.Context, e Event) error {
	b.events = append(b.events, e)
	return nil
}

// Flush appends OutboxEvents insert mutations for every buffered event onto the
// underlying transaction. Call once, after the domain's own mutations are
// buffered, so all writes land in a single commit.
func (b *SpannerTxnBuffer) Flush(ctx context.Context) error {
	if b == nil || b.txn == nil || len(b.events) == 0 {
		return nil
	}
	mutations := make([]*spanner.Mutation, 0, len(b.events))
	for _, e := range b.events {
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", EventRowMap(e)))
	}
	return b.txn.BufferWrite(mutations)
}
