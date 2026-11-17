package roi_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openROISpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST unset; skipping live Spanner ROI test")
	}
	envOr := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	dbPath := "projects/" + envOr("SPANNER_PROJECT", "pegasusx-local") +
		"/instances/" + envOr("SPANNER_INSTANCE", "pegasusx-instance") +
		"/databases/" + envOr("SPANNER_DATABASE", "pegasusx-db")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner dial: %v", err)
		}
		t.Skipf("spanner unavailable: %v", err)
	}
	ro := c.ReadOnlyTransaction()
	_, pErr := ro.ReadRow(ctx, "RoiBaselines", spanner.Key{"__probe__", "__probe__"}, []string{"BaselineId"})
	ro.Close()
	if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
		c.Close()
		if requireSpanner {
			t.Fatalf("RoiBaselines unavailable: %v", pErr)
		}
		t.Skipf("RoiBaselines unavailable: %v", pErr)
	}
	t.Cleanup(c.Close)
	return c
}

func TestROI_LiveSpanner_BaselineEventsBillingFeesAndOutbox(t *testing.T) {
	client := openROISpannerClient(t)
	ctx := context.Background()
	tid := "sup-roi-sp-" + uuid.NewString()[:8]
	now := time.Now().UTC()

	// Seed ArInvoices (overdue bad debt), StockLots (expired), BillingMeterEvents + BillingFeeSchedules.
	invID := "inv-roi-" + uuid.NewString()[:8]
	lotID := "lot-roi-" + uuid.NewString()[:8]
	prodID := "prod-roi-" + uuid.NewString()[:8]
	schedID := "sched-roi-" + uuid.NewString()[:8]
	bmID := "bm-roi-" + uuid.NewString()[:8]

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("ArInvoices", spanner.Key{invID}),
			spanner.Delete("StockLots", spanner.Key{lotID}),
			spanner.Delete("Products", spanner.Key{prodID}),
			spanner.Delete("BillingFeeSchedules", spanner.Key{schedID}),
			spanner.Delete("BillingMeterEvents", spanner.Key{bmID}),
			spanner.Delete("RoiBaselines", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("RoiEvents", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
		})
	})

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Products", map[string]any{
			"ProductId": prodID, "SupplierId": tid, "CategoryId": "cat-1",
			"Name": "Cheese 250g", "PriceMinor": int64(15000), "Currency": "UZS",
			"StockQuantity": int64(100), "IsActive": true, "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ArInvoices", map[string]any{
			"InvoiceId": invID, "SupplierId": tid, "RetailerId": "ret-1", "OrderId": "ord-1",
			"Status": "OPEN", "PrincipalMinor": int64(800000), "BalanceMinor": int64(500000),
			"Currency": "UZS", "CreditLeaveAt": now.Add(-60 * 24 * time.Hour), "DueAt": now.Add(-45 * 24 * time.Hour),
			"TermsDays": int64(15), "GracePeriodDays": int64(0), "DunningStep": int64(2),
			"Version": int64(1), "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotID, "SupplierId": tid, "WarehouseId": "wh-1", "ProductId": prodID, "LocationId": "loc-1",
			"ExpiryDate": civil.DateOf(now.Add(-2 * 24 * time.Hour)), "QuantityOnHand": int64(10),
			"QuantityReserved": int64(0), "ReceivedAt": spanner.CommitTimestamp, "Status": "EXPIRED",
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("BillingFeeSchedules", map[string]any{
			"FeeScheduleId": schedID, "SupplierId": tid, "Tier": "STANDARD",
			"PerOrderMinor": int64(25000), "GmvBps": int64(100), "MonthlySubscriptionMinor": int64(100000),
			"Currency": "UZS", "EffectiveFrom": now.Add(-24 * time.Hour), "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("BillingMeterEvents", map[string]any{
			"EventId": bmID, "SupplierId": tid, "OrderId": "ord-bm-1",
			"MeterType": "ORDER_COMPLETED", "Amount": int64(1000000), "ProcessedAt": now.Add(-time.Hour),
		}),
	}); err != nil {
		t.Fatalf("seed live Spanner ROI data: %v", err)
	}

	svc := roi.NewService(client)
	base, err := svc.CaptureBaseline(ctx, roi.CaptureBaselineInput{
		TenantID:              tid,
		WindowDays:            90,
		BaselineLabourHours:   20,
		SurveyHourlyWageMinor: 30000,
		CapturedBy:            "owner-sp-1",
	})
	if err != nil {
		t.Fatalf("CaptureBaseline Spanner: %v", err)
	}
	if base.BaselineArMinor != 800000 || base.BaselineBadDebtMinor != 500000 || base.BaselineExpiredMinor != 150000 {
		t.Fatalf("unexpected Spanner historical baseline: %+v", base)
	}

	ev, err := svc.RecordEvent(ctx, roi.RecordEventInput{
		TenantID: tid, Category: roi.CategoryExpirySaved, SavingsMinor: 405000,
		EvidenceRefType: "StockLot", EvidenceRefID: lotID, RecordedBy: "owner-sp-1",
		IdempotencyKey: "sp-roi-idem-" + tid,
	})
	if err != nil {
		t.Fatalf("RecordEvent Spanner: %v", err)
	}
	if ev.EventID == "" {
		t.Fatalf("expected non-empty EventID")
	}

	// Verify outbox row roi.event.recorded written atomically.
	ro := client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT AggregateType, Payload FROM OutboxEvents WHERE AggregateId = @eid LIMIT 1`,
		Params: map[string]any{"eid": ev.EventID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		t.Fatalf("expected OutboxEvents row for roi.event.recorded: %v", err)
	}
	var aggType string
	var payload []byte
	if err := row.Columns(&aggType, &payload); err != nil {
		t.Fatalf("scan OutboxEvents row: %v", err)
	}
	if aggType != "RoiEvent" || !strings.Contains(string(payload), "roi.event.recorded") {
		t.Fatalf("expected outbox RoiEvent roi.event.recorded, got agg=%q payload=%s", aggType, string(payload))
	}

	sum, err := svc.GetSummary(ctx, tid, 90)
	if err != nil {
		t.Fatalf("GetSummary Spanner: %v", err)
	}
	// Fee: 25,000 per order + 10,000 (1% of 1,000,000) + 3 * 100,000 (90 days / 30 = 3 months) = 335,000
	if sum.PlatformFeesMinor != 335000 || sum.TotalSavingsMinor != 405000 || sum.NetSavingsMinor != 70000 {
		t.Fatalf("unexpected Spanner ROI summary fees/savings: %+v", sum)
	}
}
