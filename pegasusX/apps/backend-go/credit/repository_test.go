package credit

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func newEmulatorClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if os.Getenv("PARITY_REQUIRE_SPANNER") == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping integration test")
	}
	envOr := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"))
	client, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host), option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithInsecure()))
	if err != nil {
		t.Fatalf("spanner client: %v", err)
	}
	return client
}

func TestClearBalanceInTxn_ValidationGuards(t *testing.T) {
	ctx := context.Background()
	repo := &SpannerRepository{}

	// Nil txn
	if err := repo.ClearBalanceInTxn(ctx, nil, "ret-1", "sup-1", "ord-1", 100); err == nil {
		t.Fatal("expected error on nil txn")
	}

	// Empty orderID is no-op
	if err := repo.ClearBalanceInTxn(ctx, &spanner.ReadWriteTransaction{}, "ret-1", "sup-1", "", 100); err != nil {
		t.Fatalf("expected nil on empty order ID, got %v", err)
	}

	// Zero or negative amount is no-op
	if err := repo.ClearBalanceInTxn(ctx, &spanner.ReadWriteTransaction{}, "ret-1", "sup-1", "ord-1", 0); err != nil {
		t.Fatalf("expected nil on 0 amount, got %v", err)
	}
	if err := repo.ClearBalanceInTxn(ctx, &spanner.ReadWriteTransaction{}, "ret-1", "sup-1", "ord-1", -50); err != nil {
		t.Fatalf("expected nil on negative amount, got %v", err)
	}
}

func TestClearBalanceInTxn_MigratedOrderClearsBalance_Emulator(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	retailerID := fmt.Sprintf("ret-mig-%d", suffix)
	supplierID := fmt.Sprintf("sup-mig-%d", suffix)
	orderID := fmt.Sprintf("ord-mig-%d", suffix)
	now := time.Now().UTC()

	// Seed RetailerCreditProfiles with 300,000 balance and Orders row with OrderSource="MIGRATION".
	// Notice: OrderCreditReservations is intentionally NOT created, simulating legacy migrated AR.
	muts := []*spanner.Mutation{
		spanner.InsertMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":           retailerID,
			"SupplierId":           supplierID,
			"CreditLimitMinor":     int64(1000000),
			"CurrentBalanceMinor":  int64(300000),
			"ReservedMinor":        int64(0),
			"AvailableCreditMinor": int64(700000),
			"Status":               "ACTIVE",
			"RiskScore":            int64(50),
			"DelinquencyCount":     int64(0),
			"Version":              int64(1),
			"LastEvaluatedAt":      now,
			"CreatedAt":            now,
			"UpdatedAt":            now,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            orderID,
			"SupplierId":         supplierID,
			"RetailerId":         retailerID,
			"Status":             "DELIVERED",
			"OrderSource":        "MIGRATION",
			"ConfirmationStatus": "CONFIRMED",
			"LineItemsJson":      []byte("[]"),
			"TotalMinor":         int64(300000),
			"OriginalTotalMinor": int64(300000),
			"Currency":           "UZS",
			"DeliveryPriority":   "STANDARD",
			"Version":            int64(1),
			"CreatedAt":          now,
			"UpdatedAt":          now,
		}),
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("seed spanner: %v", err)
	}

	repo := NewSpannerRepository(client)

	// Clear balance on the migrated order
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return repo.ClearBalanceInTxn(ctx, txn, retailerID, supplierID, orderID, 300000)
	})
	if err != nil {
		t.Fatalf("ClearBalanceInTxn: %v", err)
	}

	// Verify RetailerCreditProfiles CurrentBalanceMinor was decremented to 0
	pRow, err := client.Single().ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID},
		[]string{"CurrentBalanceMinor", "AvailableCreditMinor"})
	if err != nil {
		t.Fatalf("read credit profile: %v", err)
	}
	var bal, avail int64
	if err := pRow.Columns(&bal, &avail); err != nil {
		t.Fatal(err)
	}
	if bal != 0 {
		t.Fatalf("CurrentBalanceMinor = %d, want 0", bal)
	}
	if avail != 1000000 {
		t.Fatalf("AvailableCreditMinor = %d, want 1000000", avail)
	}

	// Verify OrderCreditReservations now has a CLEARED row
	rRow, err := client.Single().ReadRow(ctx, "OrderCreditReservations", spanner.Key{orderID},
		[]string{"Status", "AmountMinor"})
	if err != nil {
		t.Fatalf("read order reservation: %v", err)
	}
	var resStatus string
	var resAmount int64
	if err := rRow.Columns(&resStatus, &resAmount); err != nil {
		t.Fatal(err)
	}
	if resStatus != "CLEARED" {
		t.Fatalf("reservation Status = %q, want CLEARED", resStatus)
	}

	// Call again: must be idempotent no-op
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return repo.ClearBalanceInTxn(ctx, txn, retailerID, supplierID, orderID, 300000)
	})
	if err != nil {
		t.Fatalf("idempotent retry failed: %v", err)
	}
}

func TestClearBalanceInTxn_MigratedWithConvertedReservation_Emulator(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	retailerID := fmt.Sprintf("ret-conv-%d", suffix)
	supplierID := fmt.Sprintf("sup-conv-%d", suffix)
	orderID := fmt.Sprintf("ord-conv-%d", suffix)
	now := time.Now().UTC()

	// Seed with CONVERTED reservation as created by PrepareMigrationOpeningBalance
	muts := []*spanner.Mutation{
		spanner.InsertMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":           retailerID,
			"SupplierId":           supplierID,
			"CreditLimitMinor":     int64(500000),
			"CurrentBalanceMinor":  int64(200000),
			"ReservedMinor":        int64(0),
			"AvailableCreditMinor": int64(300000),
			"Status":               "ACTIVE",
			"RiskScore":            int64(50),
			"DelinquencyCount":     int64(0),
			"Version":              int64(1),
			"LastEvaluatedAt":      now,
			"CreatedAt":            now,
			"UpdatedAt":            now,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            orderID,
			"SupplierId":         supplierID,
			"RetailerId":         retailerID,
			"Status":             "DELIVERED",
			"OrderSource":        "MIGRATION",
			"ConfirmationStatus": "CONFIRMED",
			"LineItemsJson":      []byte("[]"),
			"TotalMinor":         int64(200000),
			"OriginalTotalMinor": int64(200000),
			"Currency":           "UZS",
			"DeliveryPriority":   "STANDARD",
			"Version":            int64(1),
			"CreatedAt":          now,
			"UpdatedAt":          now,
		}),
		spanner.InsertMap("OrderCreditReservations", map[string]any{
			"OrderId":     orderID,
			"RetailerId":  retailerID,
			"SupplierId":  supplierID,
			"AmountMinor": int64(200000),
			"Status":      "CONVERTED",
			"CreatedAt":   now,
			"UpdatedAt":   now,
		}),
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("seed spanner: %v", err)
	}

	repo := NewSpannerRepository(client)

	// Clear balance
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return repo.ClearBalanceInTxn(ctx, txn, retailerID, supplierID, orderID, 200000)
	})
	if err != nil {
		t.Fatalf("ClearBalanceInTxn: %v", err)
	}

	// Verify balance decremented to 0 and reservation status is CLEARED
	pRow, err := client.Single().ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID},
		[]string{"CurrentBalanceMinor"})
	if err != nil {
		t.Fatal(err)
	}
	var bal int64
	if err := pRow.Column(0, &bal); err != nil || bal != 0 {
		t.Fatalf("balance = %d, want 0", bal)
	}

	rRow, err := client.Single().ReadRow(ctx, "OrderCreditReservations", spanner.Key{orderID},
		[]string{"Status"})
	if err != nil {
		t.Fatal(err)
	}
	var st string
	if err := rRow.Column(0, &st); err != nil || st != "CLEARED" {
		t.Fatalf("status = %q, want CLEARED", st)
	}
}
