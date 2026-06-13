package outbox

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func TestSpannerStore_AppendFetchMarkPublished_Integration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	store := NewSpannerStore(client)

	eventID := fmt.Sprintf("it-%d", time.Now().UnixNano())
	if len(eventID) > 36 {
		eventID = eventID[:36]
	}

	event := Event{
		EventID:       eventID,
		AggregateType: "Order",
		AggregateID:   "order-it",
		TopicName:     "pegasusx-main",
		Payload:       []byte(`{"integration":true}`),
		CreatedAt:     time.Unix(1, 0).UTC(),
	}

	if err := store.Append(ctx, []Event{event}); err != nil {
		t.Fatalf("append integration event: %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.Apply(ctx, []*spanner.Mutation{spanner.Delete("OutboxEvents", spanner.Key{eventID})})
		client.Close()
	})

	fetched, err := store.Fetch(ctx, 10000)
	if err != nil {
		t.Fatalf("fetch integration event: %v", err)
	}
	if !containsEventID(fetched, eventID) {
		t.Fatalf("fetch did not include appended event_id %s", eventID)
	}

	publishAt := time.Now().UTC().Truncate(time.Microsecond)
	if err := store.MarkPublished(ctx, []string{eventID}, publishAt); err != nil {
		t.Fatalf("mark published integration event: %v", err)
	}

	publishedAt, err := lookupPublishedAt(ctx, client, eventID)
	if err != nil {
		t.Fatalf("lookup published_at: %v", err)
	}
	if !publishedAt.Valid {
		t.Fatalf("published_at should be set")
	}
	if publishedAt.Time.Before(publishAt.Add(-time.Second)) {
		t.Fatalf("published_at %s is older than expected %s", publishedAt.Time.UTC(), publishAt)
	}

	remaining, err := countUnpublishedByEventID(ctx, client, eventID)
	if err != nil {
		t.Fatalf("count unpublished: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("unpublished count = %d, want 0", remaining)
	}
}

func newSpannerIntegrationClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()

	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	emulatorHost := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if emulatorHost == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping integration test")
	}

	project := envOrDefault("SPANNER_PROJECT", "pegasusx-local")
	instance := envOrDefault("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOrDefault("SPANNER_DATABASE", "pegasusx-db")
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(emulatorHost),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithInsecure()),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner emulator database unavailable (%s): %v", dbPath, err)
		}
		t.Skipf("spanner emulator database unavailable (%s): %v", dbPath, err)
	}

	if err := ensureOutboxTableExists(ctx, client); err != nil {
		client.Close()
		if requireSpanner {
			t.Fatalf("OutboxEvents not ready in emulator database (%s): %v", dbPath, err)
		}
		t.Skipf("OutboxEvents not ready in emulator database (%s): %v", dbPath, err)
	}

	return client
}

func ensureOutboxTableExists(ctx context.Context, client *spanner.Client) error {
	stmt := spanner.Statement{
		SQL: `
SELECT TABLE_NAME
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_NAME = 'OutboxEvents'
LIMIT 1`,
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return fmt.Errorf("table not found")
	}
	if err != nil {
		return err
	}
	var table string
	if err := row.Columns(&table); err != nil {
		return err
	}
	if table != "OutboxEvents" {
		return fmt.Errorf("unexpected table %q", table)
	}
	return nil
}

func lookupPublishedAt(ctx context.Context, client *spanner.Client, eventID string) (spanner.NullTime, error) {
	stmt := spanner.Statement{
		SQL: `SELECT PublishedAt FROM OutboxEvents WHERE EventId = @event_id`,
		Params: map[string]interface{}{
			"event_id": eventID,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return spanner.NullTime{}, err
	}
	var publishedAt spanner.NullTime
	if err := row.Columns(&publishedAt); err != nil {
		return spanner.NullTime{}, err
	}
	return publishedAt, nil
}

func countUnpublishedByEventID(ctx context.Context, client *spanner.Client, eventID string) (int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(1) FROM OutboxEvents WHERE EventId = @event_id AND PublishedAt IS NULL`,
		Params: map[string]interface{}{
			"event_id": eventID,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, err
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func containsEventID(events []Event, eventID string) bool {
	for _, e := range events {
		if e.EventID == eventID {
			return true
		}
	}
	return false
}

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
