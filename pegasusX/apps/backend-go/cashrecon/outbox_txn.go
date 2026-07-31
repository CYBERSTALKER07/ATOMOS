package cashrecon

import (
	"context"
	"time"

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
		createdAt := event.CreatedAt.UTC()
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		row := map[string]any{
			"EventId":       event.EventID,
			"AggregateType": event.AggregateType,
			"AggregateId":   event.AggregateID,
			"TopicName":     event.TopicName,
			"Payload":       event.Payload,
			"CreatedAt":     createdAt,
			"PublishedAt":   nil,
		}
		if event.PublishedAt != nil {
			row["PublishedAt"] = event.PublishedAt.UTC()
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
	}
	return mutations
}
