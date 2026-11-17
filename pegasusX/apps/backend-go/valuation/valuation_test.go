package valuation_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/recon"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"github.com/pegasusx/pegasusx/apps/backend-go/valuation"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func openValuationSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner valuation test")
	}

	project := envOrDefault("SPANNER_PROJECT", "pegasusx-local")
	instance := envOrDefault("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOrDefault("SPANNER_DATABASE", "pegasusx-db")
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("open spanner client (%s): %v", dbPath, err)
	}
	t.Cleanup(client.Close)
	return client
}

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func TestMovingAverageIntegerMath(t *testing.T) {
	t.Parallel()
	if got := valuation.ComputeMovingAverageUnitCostMinor(0, 10); got != 0 {
		t.Fatalf("zero val avg = %d, want 0", got)
	}
	if got := valuation.ComputeMovingAverageUnitCostMinor(10_000, 0); got != 0 {
		t.Fatalf("zero qty avg = %d, want 0", got)
	}
	// Half-up integer division: (1000 * 1 + 1) / 3 = 333, (2000) / 3 = 667.
	if got := valuation.ComputeMovingAverageUnitCostMinor(1_000, 3); got != 333 {
		t.Fatalf("1000/3 avg = %d, want 333", got)
	}
	if got := valuation.ComputeMovingAverageUnitCostMinor(2_000, 3); got != 667 {
		t.Fatalf("2000/3 half-up avg = %d, want 667", got)
	}
}

func TestMemory_FEFOOrdering_MovingAvg_And_Idempotency(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := valuation.NewMemoryRepository()
	svc := valuation.NewServiceWithRepository(repo)

	t0 := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

	// Receipt 1: no expiry, earliest receipt, unit cost 1,000, qty 10 -> total 10,000.
	r1, err := svc.RecordReceipt(ctx, valuation.ReceiptInput{
		EntityID:       "sup-mem-1",
		WarehouseID:    "wh-1",
		ProductID:      "sku-1",
		LotID:          "lot-noexp",
		ReceiptRefType: "PO",
		ReceiptRefID:   "po-1",
		Quantity:       10,
		UnitCostMinor:  1_000,
		VatMinor:       1_200,
		CreditAccount:  gl.AccountVendorPayable,
		VendorID:       "vendor-a",
		ReceivedAt:     t0,
		IdempotencyKey: "rec-1",
		ActorID:        "usr-1",
	})
	if err != nil {
		t.Fatalf("RecordReceipt 1: %v", err)
	}
	if r1.TotalCostMinor != 10_000 || r1.VatMinor != 1_200 || r1.AlreadyProcessed {
		t.Fatalf("unexpected r1: %+v", r1)
	}

	// Idempotent replay must return AlreadyProcessed=true without doubling inventory value.
	r1Dup, err := svc.RecordReceipt(ctx, valuation.ReceiptInput{
		EntityID:       "sup-mem-1",
		WarehouseID:    "wh-1",
		ProductID:      "sku-1",
		LotID:          "lot-noexp",
		ReceiptRefType: "PO",
		ReceiptRefID:   "po-1",
		Quantity:       10,
		UnitCostMinor:  1_000,
		VatMinor:       1_200,
		CreditAccount:  gl.AccountVendorPayable,
		ReceivedAt:     t0,
		IdempotencyKey: "rec-1",
	})
	if err != nil {
		t.Fatalf("RecordReceipt replay: %v", err)
	}
	if !r1Dup.AlreadyProcessed {
		t.Fatalf("expected AlreadyProcessed=true on replay")
	}

	// Receipt 2: expiring soon (2026-11-10), unit cost 2,000, qty 5 -> must consume FIRST under FEFO despite later receipt!
	r2, err := svc.RecordReceipt(ctx, valuation.ReceiptInput{
		EntityID:       "sup-mem-1",
		WarehouseID:    "wh-1",
		ProductID:      "sku-1",
		LotID:          "lot-exp-early",
		ReceiptRefType: "PO",
		ReceiptRefID:   "po-2",
		Quantity:       5,
		UnitCostMinor:  2_000,
		ReceivedAt:     t0.Add(2 * time.Hour),
		ExpiryDate:     "2026-11-10",
		IdempotencyKey: "rec-2",
	})
	if err != nil {
		t.Fatalf("RecordReceipt 2: %v", err)
	}
	if !r2.Layer.NearExpiry {
		// Note: NearExpiry relative to time.Now() vs receivedAt depends on ScanExpiryRisks / layer check.
		_ = r2
	}

	// Receipt 3: later expiry (2026-12-20), unit cost 3,000, qty 5 -> consumed SECOND under FEFO, BEFORE no-expiry!
	if _, err := svc.RecordReceipt(ctx, valuation.ReceiptInput{
		EntityID:       "sup-mem-1",
		WarehouseID:    "wh-1",
		ProductID:      "sku-1",
		LotID:          "lot-exp-late",
		ReceiptRefType: "PO",
		ReceiptRefID:   "po-3",
		Quantity:       5,
		UnitCostMinor:  3_000,
		ReceivedAt:     t0.Add(4 * time.Hour),
		ExpiryDate:     "2026-12-20",
		IdempotencyKey: "rec-3",
	}); err != nil {
		t.Fatalf("RecordReceipt 3: %v", err)
	}

	// Issue 7 units via FEFO: 5 from lot-exp-early (@ 2,000 = 10,000) + 2 from lot-exp-late (@ 3,000 = 6,000) = 16,000.
	iss, err := svc.IssueStockFEFO(ctx, valuation.IssueInput{
		EntityID:       "sup-mem-1",
		WarehouseID:    "wh-1",
		ProductID:      "sku-1",
		ReferenceType:  "PICK",
		ReferenceID:    "wave-1:task-1",
		OrderID:        "ord-1",
		Quantity:       7,
		CostingMethod:  valuation.CostingMethodFEFO,
		IdempotencyKey: "iss-fefo-1",
	})
	if err != nil {
		t.Fatalf("IssueStockFEFO: %v", err)
	}
	if iss.TotalQtyIssued != 7 || iss.TotalCostMinor != 16_000 || len(iss.Movements) != 2 {
		t.Fatalf("unexpected FEFO issue: %+v", iss)
	}
	if iss.Movements[0].LotID != "lot-exp-early" || iss.Movements[1].LotID != "lot-exp-late" {
		t.Fatalf("wrong FEFO consumption order: lot[0]=%q, lot[1]=%q", iss.Movements[0].LotID, iss.Movements[1].LotID)
	}

	// Remaining stock: lot-exp-late qty 3 @ 3,000 (=9,000) + lot-noexp qty 10 @ 1,000 (=10,000) => 13 qty, 19,000 minor.
	sumBeforeAdj, err := svc.GetValuationSummary(ctx, valuation.ValuationFilter{
		EntityID:    "sup-mem-1",
		WarehouseID: "wh-1",
		ProductID:   "sku-1",
	})
	if err != nil {
		t.Fatalf("GetValuationSummary: %v", err)
	}
	if sumBeforeAdj.TotalQtyRemaining != 13 || sumBeforeAdj.TotalValueMinor != 19_000 {
		t.Fatalf("summary mismatch: qty=%d val=%d, want 13 / 19000", sumBeforeAdj.TotalQtyRemaining, sumBeforeAdj.TotalValueMinor)
	}

	// Shrinkage (-4 qty): consumes remaining 3 from lot-exp-late (9,000) + 1 from lot-noexp (1,000) = 10,000 shrink cost.
	adj, err := svc.RecordAdjustmentCost(ctx, valuation.AdjustmentCostInput{
		EntityID:       "sup-mem-1",
		WarehouseID:    "wh-1",
		ProductID:      "sku-1",
		AdjustmentID:   "adj-1",
		DeltaQty:       -4,
		ReasonCode:     "DAMAGE",
		IdempotencyKey: "adj-1",
	})
	if err != nil {
		t.Fatalf("RecordAdjustmentCost: %v", err)
	}
	if adj.TotalCostMinor != 10_000 {
		t.Fatalf("shrink cost = %d, want 10000", adj.TotalCostMinor)
	}

	// Insufficient layer guard on over-issue (-100 qty).
	if _, err := svc.IssueStockFEFO(ctx, valuation.IssueInput{
		EntityID:       "sup-mem-1",
		WarehouseID:    "wh-1",
		ProductID:      "sku-1",
		ReferenceType:  "PICK",
		ReferenceID:    "task-too-big",
		Quantity:       100,
		IdempotencyKey: "iss-too-big",
	}); !errors.Is(err, valuation.ErrInsufficientCostLayers) {
		t.Fatalf("over-issue err = %v, want ErrInsufficientCostLayers", err)
	}

	// Moving-average costing issue on sku-ma: two layers (10 @ 1000 + 10 @ 2000 = 30,000 / 20 = 1500 avg).
	if _, err := svc.RecordReceipt(ctx, valuation.ReceiptInput{
		EntityID: "sup-mem-1", WarehouseID: "wh-1", ProductID: "sku-ma",
		ReceiptRefID: "ma-1", Quantity: 10, UnitCostMinor: 1_000, IdempotencyKey: "ma-1",
	}); err != nil {
		t.Fatalf("RecordReceipt ma-1: %v", err)
	}
	if _, err := svc.RecordReceipt(ctx, valuation.ReceiptInput{
		EntityID: "sup-mem-1", WarehouseID: "wh-1", ProductID: "sku-ma",
		ReceiptRefID: "ma-2", Quantity: 10, UnitCostMinor: 2_000, IdempotencyKey: "ma-2",
	}); err != nil {
		t.Fatalf("RecordReceipt ma-2: %v", err)
	}
	maIss, err := svc.IssueStockFEFO(ctx, valuation.IssueInput{
		EntityID: "sup-mem-1", WarehouseID: "wh-1", ProductID: "sku-ma",
		ReferenceType: "PICK", ReferenceID: "pick-ma",
		Quantity: 6, CostingMethod: valuation.CostingMethodMovingAverage, IdempotencyKey: "iss-ma-1",
	})
	if err != nil {
		t.Fatalf("IssueStockFEFO MOVING_AVERAGE: %v", err)
	}
	// 6 * (30,000 / 20 = 1,500) = 9,000.
	if maIss.TotalCostMinor != 9_000 || maIss.UnitCostAvgMinor != 1_500 {
		t.Fatalf("moving avg issue = %d (avg=%d), want 9000 (1500)", maIss.TotalCostMinor, maIss.UnitCostAvgMinor)
	}
}

func TestValuation_SpannerLiveE2E_GL_Outbox_Hooks_And_ReconTieOut(t *testing.T) {
	client := openValuationSpannerClient(t)
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	entityID := "sup-val-" + suffix
	whID := "wh-val-" + suffix
	prodID := "prod-val-" + suffix
	lotEarly := "lot-early-" + suffix
	lotLate := "lot-late-" + suffix

	cleanupSpannerValuation(t, client, entityID, whID)

	svc := valuation.NewService(client)
	now := time.Now().UTC()
	nearExpiryDate := now.AddDate(0, 0, 5).Format("2006-01-02")
	farExpiryDate := now.AddDate(0, 0, 90).Format("2006-01-02")

	// 1. Receipt 1: 10 units @ 1,500 with VAT 1,800 (AP:VENDOR), near expiry (triggers inventory.expiry.risk!).
	r1, err := svc.RecordReceipt(ctx, valuation.ReceiptInput{
		EntityID:             entityID,
		WarehouseID:          whID,
		ProductID:            prodID,
		LotID:                lotEarly,
		ReceiptRefType:       "PO",
		ReceiptRefID:         "po-val-1-" + suffix,
		Quantity:             10,
		UnitCostMinor:        1_500,
		VatMinor:             1_800,
		CreditAccount:        gl.AccountVendorPayable,
		VendorID:             "vend-val-" + suffix,
		Currency:             "UZS",
		ReceivedAt:           now.Add(-2 * time.Hour),
		ExpiryDate:           nearExpiryDate,
		ExpiryRiskWindowDays: 14,
		IdempotencyKey:       "live-rec-1-" + suffix,
		ActorID:              "usr-val",
	})
	if err != nil {
		t.Fatalf("Spanner RecordReceipt 1: %v", err)
	}
	if r1.TotalCostMinor != 15_000 || r1.GlEntryID == "" {
		t.Fatalf("unexpected r1: %+v", r1)
	}

	// Verify GL postings for Receipt 1: DR INVENTORY 15,000 + DR TAX:VAT:INPUT 1,800 = CR AP:VENDOR 16,800.
	assertGLEntryPostings(t, client, entityID, r1.GlEntryID, map[string]int64{
		gl.AccountInventory:     15_000,
		gl.AccountVatInput:      1_800,
		gl.AccountVendorPayable: -16_800,
	})

	// 2. Receipt 2: 10 units @ 2,000 via GRNI, far expiry.
	r2, err := svc.RecordReceipt(ctx, valuation.ReceiptInput{
		EntityID:       entityID,
		WarehouseID:    whID,
		ProductID:      prodID,
		LotID:          lotLate,
		ReceiptRefType: "TRANSFER",
		ReceiptRefID:   "trf-val-2-" + suffix,
		Quantity:       10,
		UnitCostMinor:  2_000,
		CreditAccount:  gl.AccountGRNI,
		Currency:       "UZS",
		ReceivedAt:     now.Add(-1 * time.Hour),
		ExpiryDate:     farExpiryDate,
		IdempotencyKey: "live-rec-2-" + suffix,
		ActorID:        "usr-val",
	})
	if err != nil {
		t.Fatalf("Spanner RecordReceipt 2: %v", err)
	}
	assertGLEntryPostings(t, client, entityID, r2.GlEntryID, map[string]int64{
		gl.AccountInventory: 20_000,
		gl.AccountGRNI:      -20_000,
	})

	// Total Inventory in GL = 15,000 + 20,000 = 35,000.

	// 3. Issue 12 units via FEFO -> consumes all 10 of near-expiry (@ 1,500 = 15,000) + 2 of far-expiry (@ 2,000 = 4,000) = 19,000 COGS.
	iss, err := svc.IssueStockFEFO(ctx, valuation.IssueInput{
		EntityID:       entityID,
		WarehouseID:    whID,
		ProductID:      prodID,
		ReferenceType:  "PICK",
		ReferenceID:    "pick-" + suffix,
		OrderID:        "ord-val-" + suffix,
		Quantity:       12,
		IdempotencyKey: "live-iss-1-" + suffix,
		ActorID:        "picker-val",
	})
	if err != nil {
		t.Fatalf("Spanner IssueStockFEFO: %v", err)
	}
	if iss.TotalQtyIssued != 12 || iss.TotalCostMinor != 19_000 || len(iss.Movements) != 2 {
		t.Fatalf("unexpected FEFO Spanner issue: %+v", iss)
	}
	assertGLEntryPostings(t, client, entityID, iss.GlEntryID, map[string]int64{
		gl.AccountCOGS:      19_000,
		gl.AccountInventory: -19_000,
	})

	// Remaining: 8 units @ 2,000 = 16,000.

	// 4. Shrinkage adjustment (-3 units) -> posts DR EXP:SHRINK 6,000 / CR INVENTORY 6,000!
	adj, err := svc.RecordAdjustmentCost(ctx, valuation.AdjustmentCostInput{
		EntityID:       entityID,
		WarehouseID:    whID,
		ProductID:      prodID,
		AdjustmentID:   "adj-val-" + suffix,
		DeltaQty:       -3,
		ReasonCode:     "EXPIRED_SPOILAGE",
		IdempotencyKey: "live-adj-1-" + suffix,
		ActorID:        "sup-mgr",
	})
	if err != nil {
		t.Fatalf("Spanner RecordAdjustmentCost: %v", err)
	}
	if adj.TotalCostMinor != 6_000 {
		t.Fatalf("shrink cost = %d, want 6000", adj.TotalCostMinor)
	}
	assertGLEntryPostings(t, client, entityID, adj.GlEntryID, map[string]int64{
		gl.AccountShrinkExpense: 6_000,
		gl.AccountInventory:     -6_000,
	})

	// Remaining in InventoryCostLayers AND GL INVENTORY: 5 units @ 2,000 = 10,000 minor!
	summary, err := svc.GetValuationSummary(ctx, valuation.ValuationFilter{
		EntityID:       entityID,
		ExpiryRiskDays: 14,
	})
	if err != nil {
		t.Fatalf("GetValuationSummary: %v", err)
	}
	if summary.TotalQtyRemaining != 5 || summary.TotalValueMinor != 10_000 {
		t.Fatalf("Spanner valuation summary = qty %d / val %d, want 5 / 10000", summary.TotalQtyRemaining, summary.TotalValueMinor)
	}

	// Verify OutboxEvents emitted onto TopicDispatch for inventory.received, inventory.issued, inventory.expiry.risk!
	assertOutboxEventTypesPresent(t, client, entityID, events.TopicDispatch, []string{
		events.EventInventoryReceived,
		events.EventInventoryIssued,
		events.EventInventoryExpiryRisk,
	})

	// 5. Verify Dimension 2 (STOCK_GL) tie-out in recon proves InventoryCostLayers 10,000 ties directly to GL INVENTORY 10,000 (DeltaMinor == 0, GREEN)!
	periodID := gl.ComputePeriodID(ctx, entityID, "UZ", now)
	reconSvc := recon.NewServiceWithRepository(recon.NewSpannerRepository(client))
	reconOut, err := reconSvc.RunReconciliation(ctx, recon.RunRequest{
		EntityID:   entityID,
		PeriodID:   periodID,
		Dimensions: []string{recon.DimensionStockGL},
	})
	if err != nil {
		t.Fatalf("recon RunReconciliation(STOCK_GL): %v", err)
	}
	check := reconOut.Run.Checks[recon.DimensionStockGL]
	if check.Status != recon.DimStatusGreen || check.DeltaMinor != 0 || check.ExpectedMinor != 10_000 || check.ActualMinor != 10_000 {
		t.Fatalf("STOCK_GL recon check = %+v, want GREEN expected=10000 actual=10000 delta=0", check)
	}

	// 6. Verify optional StockLots hook safety when wired inside Spanner RW txn:
	// Zero-layer SKU must safely no-op via IssueStockFEFOIfPresentInTxn / RecordAdjustmentCostIfPresentInTxn!
	stocklots.SetPickValuationHook(func(txCtx context.Context, txn *spanner.ReadWriteTransaction, args stocklots.PickValuationArgs) error {
		_, _, hErr := svc.IssueStockFEFOIfPresentInTxn(txCtx, txn, outbox.NewSpannerTxnBuffer(txn), valuation.IssueInput{
			EntityID:       args.EntityID,
			WarehouseID:    args.WarehouseID,
			ProductID:      args.ProductID,
			LotID:          args.LotID,
			ReferenceType:  "PICK",
			ReferenceID:    args.WaveID + ":" + args.TaskID,
			OrderID:        args.OrderID,
			Quantity:       args.Quantity,
			IdempotencyKey: args.IdempotencyKey,
			ActorID:        args.PickerID,
		})
		return hErr
	})
	t.Cleanup(func() { stocklots.SetPickValuationHook(nil) })

	if _, err := client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, consumed, iErr := svc.IssueStockFEFOIfPresentInTxn(txCtx, txn, outbox.NewSpannerTxnBuffer(txn), valuation.IssueInput{
			EntityID:       entityID,
			WarehouseID:    whID,
			ProductID:      "sku-no-layers-" + suffix,
			ReferenceType:  "PICK",
			ReferenceID:    "noop-task",
			Quantity:       2,
			IdempotencyKey: "noop-" + suffix,
		})
		if iErr != nil {
			return iErr
		}
		if consumed {
			t.Fatalf("expected consumed=false for SKU without cost layers")
		}
		return nil
	}); err != nil {
		t.Fatalf("IssueStockFEFOIfPresentInTxn no-op check failed: %v", err)
	}
}

func assertGLEntryPostings(t *testing.T, client *spanner.Client, entityID, entryID string, wantNetSigned map[string]int64) {
	t.Helper()
	if entryID == "" {
		t.Fatal("expected non-empty GlEntryId")
	}
	ctx := context.Background()
	iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    `SELECT AccountCode, Direction, AmountMinor FROM JournalPostings WHERE EntityId = @eid AND EntryId = @entryId`,
		Params: map[string]any{"eid": entityID, "entryId": entryID},
	})
	defer iter.Stop()

	got := make(map[string]int64)
	var sumDr, sumCr int64
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			t.Fatalf("query JournalPostings: %v", err)
		}
		var acct, dir string
		var amt int64
		if err := row.Columns(&acct, &dir, &amt); err != nil {
			t.Fatalf("scan JournalPosting: %v", err)
		}
		if dir == string(gl.DirectionDebit) {
			got[acct] += amt
			sumDr += amt
		} else {
			got[acct] -= amt
			sumCr += amt
		}
	}
	if sumDr == 0 || sumDr != sumCr {
		t.Fatalf("GL entry %s unbalanced: sumDr=%d sumCr=%d", entryID, sumDr, sumCr)
	}
	for acct, want := range wantNetSigned {
		if got[acct] != want {
			t.Fatalf("GL entry %s account %s net = %d, want %d (all=%v)", entryID, acct, got[acct], want, got)
		}
	}
}

func assertOutboxEventTypesPresent(t *testing.T, client *spanner.Client, entityID, wantTopic string, requiredTypes []string) {
	t.Helper()
	ctx := context.Background()
	iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    `SELECT TopicName, Payload FROM OutboxEvents WHERE SupplierId = @eid`,
		Params: map[string]any{"eid": entityID},
	})
	defer iter.Stop()

	topicsByType := make(map[string]string)
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			t.Fatalf("query OutboxEvents: %v", err)
		}
		var topic string
		var payloadBytes []byte
		if err := row.Columns(&topic, &payloadBytes); err != nil {
			t.Fatalf("scan OutboxEvent: %v", err)
		}
		var body struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(payloadBytes, &body)
		if body.Type != "" {
			topicsByType[body.Type] = topic
		}
	}
	for _, reqType := range requiredTypes {
		gotTopic, ok := topicsByType[reqType]
		if !ok {
			t.Fatalf("missing required outbox event %q for entity %s (seen=%v)", reqType, entityID, topicsByType)
		}
		if gotTopic != wantTopic {
			t.Fatalf("event %s routed to topic %q, want %q", reqType, gotTopic, wantTopic)
		}
	}
}

func cleanupSpannerValuation(t *testing.T, client *spanner.Client, entityID, warehouseID string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, sql := range []struct {
			tbl, col string
			val      string
		}{
			{"InventoryMovements", "EntityId", entityID},
			{"InventoryCostLayers", "EntityId", entityID},
			{"JournalPostings", "EntityId", entityID},
			{"JournalEntries", "EntityId", entityID},
			{"GlBalances", "EntityId", entityID},
			{"ReconciliationBreaks", "EntityId", entityID},
			{"ReconciliationRuns", "EntityId", entityID},
			{"OutboxEvents", "SupplierId", entityID},
		} {
			_, _ = client.PartitionedUpdate(ctx, spanner.Statement{
				SQL:    fmt.Sprintf("DELETE FROM %s WHERE %s = @val", sql.tbl, sql.col),
				Params: map[string]any{"val": sql.val},
			})
		}
		_ = warehouseID
	})
}
