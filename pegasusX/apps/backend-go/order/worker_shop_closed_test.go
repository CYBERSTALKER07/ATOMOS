package order

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"google.golang.org/api/option"
)

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

func TestWorkerShopClosed_ResolvesTimeout(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	orderID := fmt.Sprintf("ord_timeout_%d", time.Now().UnixNano())
	retailerID := "ret-123"
	supplierID := "sup-456"
	now := time.Now().UTC()
	past := now.Add(-10 * time.Minute)

	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("Orders",
			[]string{"OrderId", "RetailerId", "SupplierId", "Status", "TotalMinor", "Version", "ShopClosedGraceEndsAt"},
			[]any{orderID, retailerID, supplierID, string(StatusShopClosedPending), int64(10000), int64(1), past},
		),
		spanner.Insert("RetailerCreditProfiles",
			[]string{"RetailerId", "SupplierId", "CreditLimitMinor", "CurrentBalanceMinor", "AvailableCreditMinor", "Status", "RiskScore", "DelinquencyCount", "Version"},
			[]any{retailerID, supplierID, int64(100000), int64(0), int64(100000), string(credit.StatusActive), int64(800), int64(0), int64(1)},
		),
	})
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	s := &Service{
		spannerClient: client,
		now:           func() time.Time { return now },
		newID:         func() string { return "test-evt-id" },
		log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	err = s.processShopClosedTimeouts(ctx)
	if err != nil {
		t.Fatalf("processShopClosedTimeouts failed: %v", err)
	}

	row, err := client.Single().ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"Status", "ShopClosedResolution"})
	if err != nil {
		t.Fatalf("failed to read order back: %v", err)
	}
	var status string
	var resolution spanner.NullString
	if err := row.Columns(&status, &resolution); err != nil {
		t.Fatalf("failed to parse row: %v", err)
	}

	if status != string(StatusDeliveredOnCredit) {
		t.Errorf("want status %s, got %s", StatusDeliveredOnCredit, status)
	}
	if !resolution.Valid || resolution.StringVal != ShopClosedResolutionCreditLeave {
		t.Errorf("want resolution %s, got %s", ShopClosedResolutionCreditLeave, resolution.StringVal)
	}
}
