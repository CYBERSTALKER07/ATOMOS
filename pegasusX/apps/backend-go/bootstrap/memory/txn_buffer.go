package memory

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// inMemoryTxnBuffer collects outbox events for the scaffold path. A future
// Spanner adapter will replace this with a real BufferWrite-backed buffer.
type inMemoryTxnBuffer struct {
	events []outbox.Event
	audits []outbox.AuditEntry
}

func (b *inMemoryTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func (b *inMemoryTxnBuffer) BufferAudit(_ context.Context, e outbox.AuditEntry) error {
	b.audits = append(b.audits, e)
	return nil
}
