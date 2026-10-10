package billing

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

func TestNewMeterWorker(t *testing.T) {
	t.Parallel()

	worker := NewMeterWorker(nil)
	if worker == nil {
		t.Fatal("NewMeterWorker(nil) returned nil")
	}
	if worker.client != nil {
		t.Fatalf("worker.client = %v, want nil", worker.client)
	}
}

func newSpannerIntegrationClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()

	emulatorHost := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if emulatorHost == "" {
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping integration test")
	}

	project := "pegasusx-local"
	if p := os.Getenv("SPANNER_PROJECT"); p != "" {
		project = p
	}
	instance := "pegasusx-instance"
	if i := os.Getenv("SPANNER_INSTANCE"); i != "" {
		instance = i
	}
	database := "pegasusx-db"
	if d := os.Getenv("SPANNER_DATABASE"); d != "" {
		database = d
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(emulatorHost),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("failed to create spanner client: %v", err)
	}
	return client
}

func TestMeterWorker_ProcessOrderFinalized_Integration(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	worker := NewMeterWorker(client)
	orderID := "ord_meter_" + uuid.NewString()
	supplierID := "sup_meter_" + uuid.NewString()
	const amountMinor int64 = 1500000 // 15,000 UZS in minor tiyin

	// First processing
	err := worker.ProcessOrderFinalized(ctx, orderID, amountMinor, supplierID)
	if err != nil {
		t.Fatalf("ProcessOrderFinalized failed: %v", err)
	}

	// Verify BillingSupplierMeters
	row, err := client.Single().ReadRow(ctx, "BillingSupplierMeters", spanner.Key{supplierID, int64(0)}, []string{"CurrentValue"})
	if err != nil {
		t.Fatalf("read BillingSupplierMeters failed: %v", err)
	}
	var currentValue int64
	if err := row.Column(0, &currentValue); err != nil {
		t.Fatalf("decode CurrentValue failed: %v", err)
	}
	if currentValue != amountMinor {
		t.Fatalf("CurrentValue = %d, want %d", currentValue, amountMinor)
	}

	// Verify BillingMeterEvents
	stmt := spanner.Statement{
		SQL: `SELECT Amount FROM BillingMeterEvents WHERE OrderId = @orderId LIMIT 1`,
		Params: map[string]interface{}{
			"orderId": orderID,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	eventRow, err := iter.Next()
	if err != nil {
		t.Fatalf("read BillingMeterEvents failed: %v", err)
	}
	var eventAmount int64
	if err := eventRow.Column(0, &eventAmount); err != nil {
		t.Fatalf("decode Amount failed: %v", err)
	}
	if eventAmount != amountMinor {
		t.Fatalf("Event Amount = %d, want %d", eventAmount, amountMinor)
	}

	// Idempotency check: second processing of same order should be a no-op
	err = worker.ProcessOrderFinalized(ctx, orderID, amountMinor, supplierID)
	if err != nil {
		t.Fatalf("idempotent ProcessOrderFinalized failed: %v", err)
	}

	// CurrentValue should not have changed
	row, err = client.Single().ReadRow(ctx, "BillingSupplierMeters", spanner.Key{supplierID, int64(0)}, []string{"CurrentValue"})
	if err != nil {
		t.Fatalf("read BillingSupplierMeters second time failed: %v", err)
	}
	if err := row.Column(0, &currentValue); err != nil {
		t.Fatalf("decode CurrentValue second time failed: %v", err)
	}
	if currentValue != amountMinor {
		t.Fatalf("CurrentValue after replay = %d, want %d", currentValue, amountMinor)
	}
}
