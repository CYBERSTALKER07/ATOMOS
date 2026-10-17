package promotion

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SpannerTxnBuffer exports the outbox transaction buffer for Repository test doubles.
type SpannerTxnBuffer = spannerTxnBuffer

type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}
