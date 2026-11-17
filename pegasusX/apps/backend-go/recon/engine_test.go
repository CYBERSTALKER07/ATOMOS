package recon

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEvaluateSnapshot_CleanGreenRun(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

	snap := Snapshot{
		EntityID:                              "sup-green",
		PeriodID:                              "2026-11",
		Currency:                              "UZS",
		AsOf:                                  now,
		DeliveredCreditOrderCount:             2,
		DeliveredCreditOrderMinor:             500_000,
		OpenARInvoiceCount:                    1,
		OpenARBalanceMinor:                    300_000,
		GLARRetailerNetMinor:                  300_000,
		WarehousesChecked:                     1,
		CostedStockValuedMinor:                1_200_000,
		GLInventoryNetMinor:                   1_200_000,
		SettledPSPSessionsMinor:               400_000,
		DriverCashCollectedMinor:              200_000,
		CashReconAcceptedShortfallRepaidMinor: 5_000,
		VerifiedARBankGLMinor:                 150_000,
		VerifiedCreditTopupBankMinor:          50_000, // AR_ADVANCE overpayment into BANK:OPERATING
		GLPSPGatewayNetMinor:                  400_000,
		GLCashDriverNetMinor:                  195_000,
		GLCashOfficeNetMinor:                  5_000,
		GLBankOperatingNetMinor:               200_000,
		FiscalizedOrderCount:                  3,
		SnapshotVatMinor:                      72_000,
		CreditNoteVatMinor:                    12_000,
		GLVatOutputCreditMinor:                60_000,
	}

	run, breaks := EvaluateSnapshot(snap, nil, now)
	if run.Status != RunStatusGreen {
		t.Fatalf("run.Status = %q, want %q (breaks=%+v)", run.Status, RunStatusGreen, breaks)
	}
	if run.BreakCount != 0 || len(breaks) != 0 {
		t.Fatalf("BreakCount = %d (%d), want 0", run.BreakCount, len(breaks))
	}
	if run.ExposureMinor != 0 {
		t.Fatalf("ExposureMinor = %d, want 0", run.ExposureMinor)
	}
	if run.Summary.GreenDimensions != 4 || run.Summary.BreakDimensions != 0 {
		t.Fatalf("Summary = %+v, want 4 green / 0 break", run.Summary)
	}
	for _, dim := range AllDimensions {
		chk, ok := run.Checks[dim]
		if !ok {
			t.Fatalf("missing dimension check %s", dim)
		}
		if chk.Status != DimStatusGreen || chk.DeltaMinor != 0 || chk.BreakCount != 0 {
			t.Fatalf("check %s = %+v, want GREEN with 0 delta/breaks", dim, chk)
		}
	}
}

func TestEvaluateSnapshot_FourDimensionsAndUncostedOrphanBreaks(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

	snap := Snapshot{
		EntityID: "sup-breaks",
		PeriodID: "2026-11",
		Currency: "UZS",
		AsOf:     now,

		// ORDERS_AR: missing ArInvoice + GL divergence
		DeliveredCreditAnomalies: []OrderARAnomaly{
			{OrderID: "ord-missing-ar", OrderTotalMinor: 250_000, InvoicePrincipalMinor: 0, InvoiceStatus: ""},
		},
		OpenARBalanceMinor:   500_000,
		GLARRetailerNetMinor: 450_000,

		// STOCK_GL: orphan V2 mismatch + uncosted lot + GL INVENTORY divergence
		WarehousesChecked: 1,
		V2Mismatches: []StockV2Mismatch{
			{
				WarehouseID:    "wh-1",
				ProductID:      "sku-orphan",
				MissingV2:      true,
				LotAvailable:   5,
				V2Available:    0,
				DeltaAvailable: -5,
				UnitCostMinor:  40_000,
			},
		},
		UncostedLots: []UncostedLot{
			{LotID: "lot-uncosted-1", WarehouseID: "wh-1", ProductID: "sku-nocost", Quantity: 3},
		},
		CostedStockValuedMinor: 200_000,
		GLInventoryNetMinor:    0,

		// CASH_PSP: PSP shortfall + CASH:OFFICE shortfall + DISPUTED CashRecon
		SettledPSPSessionsMinor:               300_000,
		GLPSPGatewayNetMinor:                  250_000,
		CashReconAcceptedShortfallRepaidMinor: 25_000,
		DriverCashCollectedMinor:              100_000,
		GLCashDriverNetMinor:                  75_000,
		GLCashOfficeNetMinor:                  10_000,
		DisputedCashRecons: []DisputedCashRecon{
			{ReconciliationID: "cr-disp-1", DriverID: "drv-1", ExpectedCashMinor: 50_000, DeclaredCashMinor: 35_000, DifferenceMinor: -15_000},
		},

		// TAX_FISCAL: missing fiscal snapshot + VAT output credit divergence
		SnapshotVatMinor:       60_000,
		CreditNoteVatMinor:     10_000,
		GLVatOutputCreditMinor: 20_000,
		MissingSnapshotOrders: []MissingFiscalSnapshotOrder{
			{OrderID: "ord-nosnap-1", TotalMinor: 112_000},
		},
	}

	run, breaks := EvaluateSnapshot(snap, nil, now)
	if run.Status != RunStatusBreaksFound {
		t.Fatalf("run.Status = %q, want %q", run.Status, RunStatusBreaksFound)
	}
	if run.Summary.BreakDimensions != 4 || run.Summary.GreenDimensions != 0 {
		t.Fatalf("Summary = %+v, want 4 break / 0 green", run.Summary)
	}
	if int64(len(breaks)) != run.BreakCount || run.BreakCount < 9 {
		t.Fatalf("breaks len=%d BreakCount=%d, want >= 9 breaks across 4 dimensions", len(breaks), run.BreakCount)
	}

	var foundUncosted, foundOrphanLot, foundOrderAR, foundPSP, foundTax bool
	for _, b := range breaks {
		if b.SlaDueAt == nil || !b.SlaDueAt.After(now) {
			t.Fatalf("break %s missing future SlaDueAt: %+v", b.BreakID, b)
		}
		if b.DeltaMinor != b.ActualMinor-b.ExpectedMinor {
			t.Fatalf("break %s violated DeltaMinor = ActualMinor - ExpectedMinor: %+v", b.BreakID, b)
		}
		switch {
		case b.ObjectType == ObjectStockLotUncosted && b.ObjectID == "lot-uncosted-1":
			foundUncosted = true
			if b.DeltaMinor != 0 || b.Severity != SeverityMedium {
				t.Fatalf("uncosted lot break = %+v, want DeltaMinor=0 MEDIUM", b)
			}
		case b.ObjectType == ObjectStockLot && b.ObjectID == "wh-1:sku-orphan":
			foundOrphanLot = true
			if b.DeltaMinor != -200_000 {
				t.Fatalf("orphan lot delta = %d, want -200000", b.DeltaMinor)
			}
		case b.Dimension == DimensionOrdersAR && b.ObjectType == ObjectOrder && b.ObjectID == "ord-missing-ar":
			foundOrderAR = true
		case b.Dimension == DimensionCashPSP && b.ObjectType == ObjectGLAccount && b.ObjectID == "PSP:GATEWAY":
			foundPSP = true
			if b.DeltaMinor != -50_000 {
				t.Fatalf("PSP delta = %d, want -50000", b.DeltaMinor)
			}
		case b.Dimension == DimensionTaxFiscal && b.ObjectType == ObjectDimension:
			foundTax = true
			if b.DeltaMinor != -30_000 {
				t.Fatalf("TAX delta = %d, want -30000", b.DeltaMinor)
			}
		}
	}
	if !foundUncosted || !foundOrphanLot || !foundOrderAR || !foundPSP || !foundTax {
		t.Fatalf("missing expected break flags: uncosted=%t orphan=%t orderAR=%t psp=%t tax=%t",
			foundUncosted, foundOrphanLot, foundOrderAR, foundPSP, foundTax)
	}
}

func TestService_PatchTransitionsAndNightlyBatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := NewMemoryRepository()
	repo.SeedSnapshot(Snapshot{
		EntityID:               "sup-1",
		PeriodID:               "2026-11",
		Currency:               "UZS",
		OpenARBalanceMinor:     100_000,
		GLARRetailerNetMinor:   75_000,
		CostedStockValuedMinor: 0,
		GLInventoryNetMinor:    0,
	})
	svc := NewServiceWithRepository(repo)

	if _, err := svc.RunReconciliation(ctx, RunRequest{EntityID: "sup-1", PeriodID: "2026-13"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid period_id err = %v, want ErrInvalidInput", err)
	}
	if _, err := svc.ListRuns(ctx, ListRunsFilter{EntityID: "sup-1", PeriodID: "bad"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid filter period_id err = %v, want ErrInvalidInput", err)
	}

	report, err := svc.RunNightlyBatch(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("RunNightlyBatch: %v", err)
	}
	if report.SuppliersScanned != 1 || report.RunsCompleted != 1 || report.BreaksOpened != 1 {
		t.Fatalf("unexpected nightly report: %+v", report)
	}

	breaks, err := svc.ListBreaks(ctx, ListBreaksFilter{EntityID: "sup-1", Status: BreakStatusOpen})
	if err != nil || len(breaks) != 1 {
		t.Fatalf("ListBreaks = (%d, %v), want 1 break", len(breaks), err)
	}
	bid := breaks[0].BreakID

	// WAIVE without note must fail closed.
	if _, err := svc.PatchBreak(ctx, "sup-1", bid, PatchBreakRequest{Action: ActionWaive, ActorID: "admin-1"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("WAIVE without note err = %v, want ErrInvalidInput", err)
	}

	// ACKNOWLEDGE -> RESOLVE succeeds, second resolve conflicts.
	ack, err := svc.PatchBreak(ctx, "sup-1", bid, PatchBreakRequest{Action: ActionAcknowledge, Note: "investigating", ActorID: "admin-1"})
	if err != nil || ack.Status != BreakStatusAcknowledged {
		t.Fatalf("acknowledge = (%+v, %v)", ack, err)
	}
	res, err := svc.PatchBreak(ctx, "sup-1", bid, PatchBreakRequest{Action: ActionResolve, Note: "journal posted", ActorID: "admin-1"})
	if err != nil || res.Status != BreakStatusResolved {
		t.Fatalf("resolve = (%+v, %v)", res, err)
	}
	if _, err := svc.PatchBreak(ctx, "sup-1", bid, PatchBreakRequest{Action: ActionResolve, Note: "again", ActorID: "admin-1"}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("double resolve err = %v, want ErrInvalidTransition", err)
	}
}

func TestReconEngine_StockGL_CostLayersTieOutAndBreak(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// 1. When HasCostLayers=true matches GLInventoryNetMinor, STOCK_GL must be GREEN even if StockLots unvalued=0.
	repoGreen := NewMemoryRepository()
	repoGreen.SeedSnapshot(Snapshot{
		EntityID:               "sup-cl-1",
		PeriodID:               "2026-11",
		Currency:               "UZS",
		HasCostLayers:          true,
		CostLayersCount:        2,
		CostLayersQtyRemaining: 15,
		CostLayersValuedMinor:  45_000,
		GLInventoryNetMinor:    45_000,
	})
	outGreen, err := NewServiceWithRepository(repoGreen).RunReconciliation(ctx, RunRequest{
		EntityID:   "sup-cl-1",
		PeriodID:   "2026-11",
		Dimensions: []string{DimensionStockGL},
	})
	if err != nil {
		t.Fatalf("RunReconciliation green: %v", err)
	}
	chk := outGreen.Run.Checks[DimensionStockGL]
	if chk.Status != DimStatusGreen || chk.DeltaMinor != 0 || chk.ExpectedMinor != 45_000 || chk.ActualMinor != 45_000 {
		t.Fatalf("unexpected GREEN check: %+v", chk)
	}
	if chk.Metrics["cost_layers_valued_minor"] != 45_000 || chk.Metrics["cost_layers_count"] != 2 {
		t.Fatalf("unexpected cost layer metrics: %+v", chk.Metrics)
	}

	// 2. When InventoryCostLayers value diverges from GL INVENTORY, emit STOCK_GL break.
	repoBreak := NewMemoryRepository()
	repoBreak.SeedSnapshot(Snapshot{
		EntityID:               "sup-cl-2",
		PeriodID:               "2026-11",
		Currency:               "UZS",
		HasCostLayers:          true,
		CostLayersCount:        1,
		CostLayersQtyRemaining: 10,
		CostLayersValuedMinor:  50_000,
		GLInventoryNetMinor:    35_000,
	})
	outBreak, err := NewServiceWithRepository(repoBreak).RunReconciliation(ctx, RunRequest{
		EntityID:   "sup-cl-2",
		PeriodID:   "2026-11",
		Dimensions: []string{DimensionStockGL},
	})
	if err != nil {
		t.Fatalf("RunReconciliation break: %v", err)
	}
	if outBreak.Run.Status != RunStatusBreaksFound || len(outBreak.Breaks) != 1 {
		t.Fatalf("expected 1 STOCK_GL break, got run=%+v breaks=%+v", outBreak.Run, outBreak.Breaks)
	}
	if outBreak.Breaks[0].DeltaMinor != -15_000 || outBreak.Breaks[0].Dimension != DimensionStockGL {
		t.Fatalf("unexpected break: %+v", outBreak.Breaks[0])
	}
}
