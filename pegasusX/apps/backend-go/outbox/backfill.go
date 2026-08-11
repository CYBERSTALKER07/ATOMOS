package outbox

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// BackfillSupplierID stamps OutboxEvents.SupplierId from payload JSON when NULL/empty.
// Rows with no payload supplier receive PlatformSupplierID (required before NOT NULL).
func BackfillSupplierID(ctx context.Context, client *spanner.Client, limit int) (int, error) {
	if client == nil {
		return 0, fmt.Errorf("outbox backfill: nil client")
	}
	if limit <= 0 {
		limit = 200
	}
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT EventId, Payload FROM OutboxEvents
		      WHERE (SupplierId IS NULL OR SupplierId = '')
		      ORDER BY CreatedAt DESC
		      LIMIT @limit`,
		Params: map[string]any{"limit": limit},
	})
	defer iter.Stop()

	type row struct {
		eventID string
		payload []byte
	}
	var pending []row
	for {
		r, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("outbox backfill query: %w", err)
		}
		var eventID string
		var payload []byte
		if err := r.Columns(&eventID, &payload); err != nil {
			continue
		}
		pending = append(pending, row{eventID: eventID, payload: payload})
	}
	updated := 0
	for _, p := range pending {
		sid := ResolveSupplierID("", p.payload)
		_, err := client.Apply(ctx, []*spanner.Mutation{
			spanner.UpdateMap("OutboxEvents", map[string]any{
				"EventId":    p.eventID,
				"SupplierId": sid,
			}),
		})
		if err != nil {
			return updated, fmt.Errorf("outbox backfill update %s: %w", p.eventID, err)
		}
		updated++
	}
	return updated, nil
}
