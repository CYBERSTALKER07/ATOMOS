package creditnote

import (
	"context"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func outboxMutations(eventsList []outbox.Event) []*spanner.Mutation {
	mutations := make([]*spanner.Mutation, 0, len(eventsList))
	for _, event := range eventsList {
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(event)))
	}
	return mutations
}
