package outbox

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// DLQEntry is an outbox row moved to the dead-letter queue after repeated failures.
type DLQEntry struct {
	EventID       string    `json:"event_id"`
	AggregateType string    `json:"aggregate_type"`
	AggregateID   string    `json:"aggregate_id"`
	EventType     string    `json:"event_type"`
	TopicName     string    `json:"topic_name"`
	Payload       string    `json:"payload"`
	TraceID       string    `json:"trace_id,omitempty"`
	RetryCount    int64     `json:"retry_count"`
	LastError     string    `json:"last_error,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	MovedAt       time.Time `json:"moved_at"`
}

// ListDLQ returns recent OutboxDLQ rows for admin inspection.
func ListDLQ(ctx context.Context, client *spanner.Client, limit int64) ([]DLQEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	stmt := spanner.Statement{
		SQL: `SELECT EventId, AggregateType, AggregateId, EventType, TopicName, Payload, TraceID,
		             RetryCount, LastError, CreatedAt, MovedAt
		      FROM OutboxDLQ
		      ORDER BY MovedAt DESC
		      LIMIT @lim`,
		Params: map[string]interface{}{"lim": limit},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var entries []DLQEntry
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var e DLQEntry
		var payload []byte
		var traceID spanner.NullString
		var lastError spanner.NullString
		if err := row.Columns(&e.EventID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.TopicName, &payload, &traceID, &e.RetryCount, &lastError, &e.CreatedAt, &e.MovedAt); err != nil {
			return nil, err
		}
		e.Payload = string(payload)
		entries = append(entries, e)
	}
	return entries, nil
}

// ReplayDLQ re-inserts a DLQ event into OutboxEvents for another relay attempt.
func ReplayDLQ(ctx context.Context, client *spanner.Client, eventID string) error {
	row, err := client.Single().ReadRow(ctx, "OutboxDLQ", spanner.Key{eventID},
		[]string{"EventId", "AggregateType", "AggregateId", "EventType", "TopicName", "Payload", "TraceID"})
	if err != nil {
		return err
	}
	var evID, aggType, aggID, evType, topic string
	var payload []byte
	var traceID spanner.NullString
	if err := row.Columns(&evID, &aggType, &aggID, &evType, &topic, &payload, &traceID); err != nil {
		return err
	}

	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{
			spanner.Insert("OutboxEvents",
				[]string{"EventId", "AggregateType", "AggregateId", "EventType", "TopicName", "Payload", "TraceID", "RetryCount", "CreatedAt"},
				[]interface{}{evID, aggType, aggID, evType, topic, payload, traceID.StringVal, int64(0), spanner.CommitTimestamp},
			),
			spanner.Delete("OutboxDLQ", spanner.Key{eventID}),
		}
		return txn.BufferWrite(muts)
	})
	return err
}

// MarshalDLQEntries JSON-encodes entries for API responses.
func MarshalDLQEntries(entries []DLQEntry) ([]byte, error) {
	type row struct {
		DLQEntry
		Payload map[string]interface{} `json:"payload_json,omitempty"`
	}
	out := make([]row, 0, len(entries))
	for _, e := range entries {
		r := row{DLQEntry: e}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(e.Payload), &parsed); err == nil {
			r.Payload = parsed
		}
		out = append(out, r)
	}
	return json.Marshal(out)
}
