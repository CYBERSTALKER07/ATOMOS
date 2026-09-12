package outbox

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// BackfillSupplierID stamps OutboxEvents.SupplierId from payload JSON when NULL/empty.
// Rows with no payload supplier receive PlatformSupplierID (required before NOT NULL).
// Paginates via primary key cursor (EventId) and batches updates into a single Spanner Apply RPC.
func BackfillSupplierID(ctx context.Context, client *spanner.Client, limit int) (int, error) {
	updated, _, err := BackfillSupplierIDWithCursor(ctx, client, "", limit)
	return updated, err
}

// BackfillSupplierIDWithCursor paginates using the EventId primary key cursor and batches mutations.
func BackfillSupplierIDWithCursor(ctx context.Context, client *spanner.Client, afterEventID string, limit int) (int, string, error) {
	if client == nil {
		return 0, "", fmt.Errorf("outbox backfill: nil client")
	}
	if limit <= 0 {
		limit = 200
	}
	query := `SELECT EventId, Payload FROM OutboxEvents
	          WHERE (SupplierId IS NULL OR SupplierId = '')`
	params := map[string]any{"limit": limit}
	if afterEventID != "" {
		query += ` AND EventId > @afterEventID`
		params["afterEventID"] = afterEventID
	}
	query += ` ORDER BY EventId ASC LIMIT @limit`

	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    query,
		Params: params,
	})
	defer iter.Stop()

	type row struct {
		eventID string
		payload []byte
	}
	var pending []row
	var lastEventID string
	for {
		r, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, "", fmt.Errorf("outbox backfill query: %w", err)
		}
		var eventID string
		var payload []byte
		if err := r.Columns(&eventID, &payload); err != nil {
			continue
		}
		pending = append(pending, row{eventID: eventID, payload: payload})
		lastEventID = eventID
	}
	if len(pending) == 0 {
		return 0, "", nil
	}

	// Batch all mutations into a single transactional Apply RPC
	mutations := make([]*spanner.Mutation, 0, len(pending))
	for _, p := range pending {
		sid := ResolveSupplierID("", p.payload)
		mutations = append(mutations, spanner.UpdateMap("OutboxEvents", map[string]any{
			"EventId":    p.eventID,
			"SupplierId": sid,
		}))
	}

	if _, err := client.Apply(ctx, mutations); err != nil {
		return 0, "", fmt.Errorf("outbox backfill batch update: %w", err)
	}

	return len(mutations), lastEventID, nil
}
