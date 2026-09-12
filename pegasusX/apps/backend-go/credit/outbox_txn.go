package credit

import (
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func outboxMutations(eventsList []outbox.Event) []*spanner.Mutation {
	mutations := make([]*spanner.Mutation, 0, len(eventsList))
	for _, event := range eventsList {
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(event)))
	}
	return mutations
}
