package payout

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

type fixedCommission struct{ minor int64 }

func (f fixedCommission) CommissionMinor(context.Context, string, int64, string) (int64, error) {
	return f.minor, nil
}

func seedLegs(t *testing.T, ctx context.Context, client *spanner.Client, supplierID, orderPrefix string, ts time.Time, legs []map[string]any) {
	t.Helper()
	muts := []*spanner.Mutation{
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            orderPrefix,
			"RetailerId":         "ret-po",
			"SupplierId":         supplierID,
			"Status":             "COMPLETED",
			"OrderSource":        "MANUAL",
			"ConfirmationStatus": "CONFIRMED",
			"LineItemsJson":      []byte("[]"),
			"TotalMinor":         int64(100000),
			"Currency":           "UZS",
			"Version":            int64(1),
			"CreatedAt":          ts,
			"UpdatedAt":          ts,
		}),
	}
	for _, leg := range legs {
		leg["OrderId"] = orderPrefix
		muts = append(muts, spanner.InsertMap("OrderPaymentLegs", leg))
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("seed legs: %v", err)
	}
}

func TestPayoutBatch_NetMathAndIdempotency(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	supplierID := fmt.Sprintf("sup-po-%d", suffix)
	// Period covers "now-ish": use lagged timestamps for emulator clock skew.
	legTime := time.Now().UTC().Add(-2 * time.Minute)
	start := legTime.Add(-24 * time.Hour)
	end := legTime.Add(time.Hour)

	seedLegs(t, ctx, client, supplierID, fmt.Sprintf("ord-po-%d-a", suffix), legTime, []map[string]any{
		{"LegId": "l1", "Method": "CARD", "AmountMinor": int64(100000), "Status": "CAPTURED", "IdempotencyKey": fmt.Sprintf("cap-a-%d", suffix), "CreatedAt": legTime, "CapturedAt": legTime},
		{"LegId": "l2", "Method": "REFUND", "AmountMinor": int64(20000), "Status": "CAPTURED", "IdempotencyKey": fmt.Sprintf("refund-card:a-%d", suffix), "CreatedAt": legTime, "CapturedAt": legTime},
	})
	seedLegs(t, ctx, client, supplierID, fmt.Sprintf("ord-po-%d-b", suffix), legTime, []map[string]any{
		{"LegId": "l1", "Method": "CASH", "AmountMinor": int64(50000), "Status": "CAPTURED", "IdempotencyKey": fmt.Sprintf("cap-b-%d", suffix), "CreatedAt": legTime, "CapturedAt": legTime},
	})

	svc := NewService(NewRepository(client))
	svc.SetCommissionResolver(fixedCommission{minor: 5000})
	b, err := svc.GenerateBatch(ctx, supplierID, start, end, "admin-1", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if b.GrossCapturedMinor != 150000 || b.RefundedMinor != 20000 || b.CommissionMinor != 5000 {
		t.Fatalf("batch math = %d/%d/%d, want 150000/20000/5000", b.GrossCapturedMinor, b.RefundedMinor, b.CommissionMinor)
	}
	if b.NetPayoutMinor != 125000 {
		t.Fatalf("net = %d, want 125000", b.NetPayoutMinor)
	}

	// Replay: same period returns the existing batch.
	b2, err := svc.GenerateBatch(ctx, supplierID, start, end, "admin-1", "")
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if b2.BatchID != b.BatchID {
		t.Fatalf("replay created new batch %s, want %s", b2.BatchID, b.BatchID)
	}
}

func TestPayoutExport_FailClosedBankDetails(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	supplierID := fmt.Sprintf("sup-po-nobank-%d", suffix)
	legTime := time.Now().UTC().Add(-2 * time.Minute)
	seedLegs(t, ctx, client, supplierID, fmt.Sprintf("ord-po-nb-%d", suffix), legTime, []map[string]any{
		{"LegId": "l1", "Method": "CARD", "AmountMinor": int64(100000), "Status": "CAPTURED", "IdempotencyKey": fmt.Sprintf("cap-nb-%d", suffix), "CreatedAt": legTime, "CapturedAt": legTime},
	})

	svc := NewService(NewRepository(client))
	b, err := svc.GenerateBatch(ctx, supplierID, legTime.Add(-time.Hour), legTime.Add(time.Hour), "a", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// No SupplierProfiles row: bank details resolution must fail, no file.
	if _, _, err := svc.ExportBankFile(ctx, b.BatchID); err == nil {
		t.Fatal("export without bank details must fail closed")
	}
}
