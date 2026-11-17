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
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
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

	suffix := time.Now().UnixNano()
	orderID := fmt.Sprintf("ord_timeout_%d", suffix)
	retailerID := fmt.Sprintf("ret-123-%d", suffix)
	supplierID := fmt.Sprintf("sup-456-%d", suffix)
	// Lagged clock: the emulator container clock can trail the host, and
	// Spanner rejects client timestamps in the future.
	now := time.Now().UTC().Add(-2 * time.Minute)
	past := now.Add(-10 * time.Minute)

	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("Orders",
			[]string{"OrderId", "RetailerId", "SupplierId", "Status", "OrderSource", "ConfirmationStatus", "LineItemsJson", "TotalMinor", "Currency", "Version", "ShopClosedGraceEndsAt", "CreatedAt", "UpdatedAt"},
			[]any{orderID, retailerID, supplierID, string(StatusShopClosedPending), string(OrderSourceManual), string(ConfirmationStatusConfirmed), []byte("[]"), int64(10000), "UZS", int64(1), past, past, past},
		),
		spanner.Insert("RetailerCreditProfiles",
			[]string{"RetailerId", "SupplierId", "CreditLimitMinor", "CurrentBalanceMinor", "AvailableCreditMinor", "Status", "RiskScore", "DelinquencyCount", "Version", "CreatedAt", "UpdatedAt"},
			[]any{retailerID, supplierID, int64(100000), int64(0), int64(100000), string(credit.StatusActive), int64(800), int64(0), int64(1), past, past},
		),
	})
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	t.Setenv("AR_INVOICES_ENABLED", "1")
	creditSvc := credit.NewService(credit.NewSpannerRepository(client))
	creditSvc.SetNow(func() time.Time { return now })
	seq := 0
	s := &Service{
		spannerClient:      client,
		credit:             creditSvc,
		ar:                 ar.NewService(ar.NewSpannerRepository(client)),
		commissionResolver: billing.NewFeeScheduleResolver(client),
		now:                func() time.Time { return now },
		newID: func() string {
			seq++
			return fmt.Sprintf("test-evt-id-%d-%d", suffix, seq)
		},
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
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

func TestSpannerOrder_ListRetailerOrdersHandlesNullWarehouseAndGeo(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	orderID := fmt.Sprintf("ord_nullcols_%d", suffix)
	retailerID := fmt.Sprintf("ret_nullcols_%d", suffix)
	supplierID := fmt.Sprintf("sup_nullcols_%d", suffix)
	past := time.Now().UTC().Add(-2 * time.Minute)

	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("Orders",
			[]string{"OrderId", "RetailerId", "SupplierId", "Status", "OrderSource", "ConfirmationStatus", "LineItemsJson", "TotalMinor", "Currency", "Version", "CreatedAt", "UpdatedAt"},
			[]any{orderID, retailerID, supplierID, "DELIVERED", "MIGRATION", string(ConfirmationStatusConfirmed), []byte("[]"), int64(25000), "UZS", int64(1), past, past},
		),
	})
	if err != nil {
		t.Fatalf("insert nullable-cols order: %v", err)
	}

	repo := NewSpannerRepository(client)
	orders, err := repo.ListRetailerOrders(ctx, retailerID, 10)
	if err != nil {
		t.Fatalf("ListRetailerOrders failed on NULL WarehouseId/H3Cell/Lat/Lng: %v", err)
	}
	if len(orders) != 1 || orders[0].OrderID != orderID {
		t.Fatalf("got orders %+v want 1 order %q", orders, orderID)
	}
	if orders[0].WarehouseID != "" || orders[0].H3Cell != "" || orders[0].Lat != 0 || orders[0].Lng != 0 {
		t.Fatalf("expected empty/zero defaults for NULL columns, got %+v", orders[0])
	}
}

func TestDecideShopClosedTimeout_ForceBypass(t *testing.T) {
	o := &Order{
		OrderID:    "ord-force-bypass-1",
		Status:     StatusShopClosedPending,
		TotalMinor: 50000,
	}
	// Profile with insufficient credit and AllowForceBypass enabled
	p := &credit.Profile{
		CreditLimitMinor:     10000,
		AvailableCreditMinor: 0,
		Status:               credit.StatusActive,
	}
	cfg := TimeoutConfig{
		AllowForceBypass: true,
	}

	decision := DecideShopClosedTimeout(o, p, cfg)
	if decision != DecisionForceBypass {
		t.Fatalf("expected DecisionForceBypass, got %s", decision)
	}

	// Without AllowForceBypass, should fall back to DecisionReturnToWarehouse
	cfg.AllowForceBypass = false
	decisionDefault := DecideShopClosedTimeout(o, p, cfg)
	if decisionDefault != DecisionReturnToWarehouse {
		t.Fatalf("expected DecisionReturnToWarehouse, got %s", decisionDefault)
	}
}

func TestWorkerShopClosed_ForceBypassResolution(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	orderID := fmt.Sprintf("ord_bypass_%d", suffix)
	retailerID := fmt.Sprintf("ret-bypass-%d", suffix)
	supplierID := fmt.Sprintf("sup-bypass-%d", suffix)
	now := time.Now().UTC().Add(-2 * time.Minute)
	past := now.Add(-10 * time.Minute)

	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("Orders",
			[]string{"OrderId", "RetailerId", "SupplierId", "Status", "OrderSource", "ConfirmationStatus", "LineItemsJson", "TotalMinor", "Currency", "Version", "ShopClosedGraceEndsAt", "CreatedAt", "UpdatedAt"},
			[]any{orderID, retailerID, supplierID, string(StatusShopClosedPending), string(OrderSourceManual), string(ConfirmationStatusConfirmed), []byte("[]"), int64(100000), "UZS", int64(1), past, past, past},
		),
		spanner.Insert("RetailerCreditProfiles",
			[]string{"RetailerId", "SupplierId", "CreditLimitMinor", "CurrentBalanceMinor", "AvailableCreditMinor", "Status", "RiskScore", "DelinquencyCount", "Version", "CreatedAt", "UpdatedAt"},
			[]any{retailerID, supplierID, int64(0), int64(0), int64(0), string(credit.StatusFrozen), int64(300), int64(5), int64(1), past, past},
		),
	})
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	seq := 0
	s := &Service{
		spannerClient: client,
		now:           func() time.Time { return now },
		timeoutCfg: TimeoutConfig{
			AllowForceBypass: true,
		},
		newID: func() string {
			seq++
			return fmt.Sprintf("test-bypass-id-%d-%d", suffix, seq)
		},
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	err = s.processShopClosedTimeouts(ctx)
	if err != nil {
		t.Fatalf("processShopClosedTimeouts failed: %v", err)
	}

	row, err := client.Single().ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"Status", "ShopClosedResolution", "ShopClosedGraceEndsAt", "ProximityUnlockedAt"})
	if err != nil {
		t.Fatalf("failed to read order back: %v", err)
	}
	var status string
	var resolution spanner.NullString
	var graceEndsAt spanner.NullTime
	var proximityUnlockedAt spanner.NullTime
	if err := row.Columns(&status, &resolution, &graceEndsAt, &proximityUnlockedAt); err != nil {
		t.Fatalf("failed to parse row: %v", err)
	}

	if status != string(StatusArrived) {
		t.Errorf("want status %s, got %s", StatusArrived, status)
	}
	if !resolution.Valid || resolution.StringVal != ShopClosedResolutionBypass {
		t.Errorf("want resolution %s, got %s", ShopClosedResolutionBypass, resolution.StringVal)
	}
	if graceEndsAt.Valid {
		t.Errorf("expected ShopClosedGraceEndsAt to be nil, got %v", graceEndsAt.Time)
	}
	if !proximityUnlockedAt.Valid {
		t.Errorf("expected ProximityUnlockedAt to be stamped")
	}
}
