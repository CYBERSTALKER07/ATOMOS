package billing

import (
	"crypto/sha1"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func testDiscardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

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

func TestMonthlyFee_Math(t *testing.T) {
	s := FeeSchedule{PerOrderMinor: 500, GmvBps: 150, MonthlySubscriptionMinor: 100000, Currency: "UZS"}
	// 100 orders, GMV 2,000,000 minor: 500*100 + 2,000,000*150/10000 + 100000
	want := int64(500*100 + 30000 + 100000)
	if got := s.MonthlyFee(100, 2000000); got != want {
		t.Fatalf("fee = %d, want %d", got, want)
	}
}

func TestZeroSchedule_EmptyDoesNotInvent(t *testing.T) {
	s := ZeroSchedule("")
	if s.Currency != "" {
		t.Fatalf("empty must not invent UZS, got %q", s.Currency)
	}
}

func TestZeroSchedule_NormalizesCallerCurrency(t *testing.T) {
	if s := ZeroSchedule("usd"); s.Currency != "USD" {
		t.Fatalf("got %q want USD", s.Currency)
	}
}

func TestPackCurrencyOrEmpty_ShippedUZ(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if got := packCurrencyOrEmpty(context.Background(), ""); got != "UZS" {
		t.Fatalf("got %q want UZS from pack", got)
	}
}

func TestPackCurrencyOrEmpty_PlannedEmpty(t *testing.T) {
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"})
	if got := packCurrencyOrEmpty(ctx, "sup-1"); got != "" {
		t.Fatalf("planned pack must not invent UZS, got %q", got)
	}
}

func TestFeeScheduleResolve_SupplierOverridesTier(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	supplierID := fmt.Sprintf("sup-bill-%d", suffix)
	now := time.Now().UTC().Add(-2 * time.Minute)

	muts := []*spanner.Mutation{
		spanner.InsertMap("SupplierProfiles", map[string]any{
			"SupplierId": supplierID, "Tier": "VOLUME",
			"RegisteredAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("BillingFeeSchedules", map[string]any{
			"FeeScheduleId": fmt.Sprintf("fs-std-%d", suffix), "SupplierId": "", "Tier": "STANDARD",
			"PerOrderMinor": int64(100), "GmvBps": int64(100), "MonthlySubscriptionMinor": int64(0),
			"Currency": "UZS", "EffectiveFrom": now.Add(-24 * time.Hour), "CreatedAt": now,
		}),
		spanner.InsertMap("BillingFeeSchedules", map[string]any{
			"FeeScheduleId": fmt.Sprintf("fs-vol-%d", suffix), "SupplierId": "", "Tier": "VOLUME",
			"PerOrderMinor": int64(50), "GmvBps": int64(75), "MonthlySubscriptionMinor": int64(0),
			"Currency": "UZS", "EffectiveFrom": now.Add(-24 * time.Hour), "CreatedAt": now,
		}),
		spanner.InsertMap("BillingFeeSchedules", map[string]any{
			"FeeScheduleId": fmt.Sprintf("fs-sup-%d", suffix), "SupplierId": supplierID, "Tier": "CUSTOM",
			"PerOrderMinor": int64(10), "GmvBps": int64(25), "MonthlySubscriptionMinor": int64(999),
			"Currency": "UZS", "EffectiveFrom": now.Add(-24 * time.Hour), "CreatedAt": now,
		}),
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("seed: %v", err)
	}

	resolver := NewFeeScheduleResolver(client)
	sched, err := resolver.Resolve(ctx, supplierID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if sched.FeeScheduleID != fmt.Sprintf("fs-sup-%d", suffix) {
		t.Fatalf("resolved %s, want supplier-specific", sched.FeeScheduleID)
	}
	commission, err := resolver.CommissionMinor(ctx, supplierID, 1000000, "UZS")
	if err != nil {
		t.Fatalf("commission: %v", err)
	}
	if commission != 2500 { // 1,000,000 * 25 bps / 10000
		t.Fatalf("commission = %d, want 2500", commission)
	}
}

func TestInvoiceWorker_MonthlyBillingOpensARItem(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	t.Setenv("AR_INVOICES_ENABLED", "1")
	suffix := time.Now().UnixNano()
	supplierID := fmt.Sprintf("sup-bw-%d", suffix)
	now := time.Now().UTC().Add(-2 * time.Minute)
	month := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	inMonth := month.AddDate(0, 0, 10)

	muts := []*spanner.Mutation{
		spanner.InsertMap("BillingFeeSchedules", map[string]any{
			"FeeScheduleId": fmt.Sprintf("fs-bw-%d", suffix), "SupplierId": supplierID, "Tier": "CUSTOM",
			"PerOrderMinor": int64(1000), "GmvBps": int64(0), "MonthlySubscriptionMinor": int64(5000),
			"Currency": "UZS", "EffectiveFrom": now.Add(-720 * time.Hour), "CreatedAt": now,
		}),
	}
	for i := 0; i < 3; i++ {
		muts = append(muts, spanner.InsertMap("BillingMeterEvents", map[string]any{
			"EventId":     fmt.Sprintf("evt-bw-%d-%d", suffix, i),
			"SupplierId":  supplierID,
			"OrderId":     fmt.Sprintf("ord-bw-%d-%d", suffix, i),
			"MeterType":   "ORDER_GMV",
			"Amount":      100.0,
			"ProcessedAt": inMonth,
		}))
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("seed: %v", err)
	}

	arSvc := ar.NewService(ar.NewSpannerRepository(client))
	arSvc.SetNow(func() time.Time { return now })
	worker := NewInvoiceWorker(client, arSvc, NewFeeScheduleResolver(client), testDiscardLogger())

	invoiceID, err := worker.GenerateMonthlyInvoice(ctx, supplierID, month)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if invoiceID == "" {
		t.Fatal("expected an invoice")
	}
	// fee = 3*1000 + 0 + 5000 = 8000; GMV = 3*100.00 major = 30000 minor
	sum := sha1.Sum([]byte(supplierID))
	inv, found, err := ar.NewSpannerRepository(client).GetByOrder(ctx, fmt.Sprintf("bill-%x-202607", sum[:6]))
	if err != nil || !found {
		t.Fatalf("billing AR invoice missing: found=%v err=%v", found, err)
	}
	if inv.PrincipalMinor != 8000 {
		t.Fatalf("principal = %d, want 8000", inv.PrincipalMinor)
	}
	if inv.SupplierID != "PLATFORM" || inv.RetailerID != supplierID {
		t.Fatalf("invoice direction = %s->%s, want PLATFORM->%s", inv.SupplierID, inv.RetailerID, supplierID)
	}

	// Idempotent: second run replays the same invoice.
	again, err := worker.GenerateMonthlyInvoice(ctx, supplierID, month)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if again != invoiceID {
		t.Fatalf("re-run opened %s, want replay of %s", again, invoiceID)
	}
}
