package periods_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/periods"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openPeriodsSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping periods live Spanner tests")
	}
	project := envOr("SPANNER_PROJECT", "pegasusx-local")
	instance := envOr("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOr("SPANNER_DATABASE", "pegasusx-db")
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := spanner.NewClient(
		ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner connect: %v", err)
		}
		t.Skipf("spanner unavailable: %v", err)
	}
	ro := client.ReadOnlyTransaction()
	_, probeErr := ro.ReadRow(ctx, "Periods", spanner.Key{"__probe__", "2026-01"}, []string{"Status"})
	ro.Close()
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("Periods table probe failed: %v", probeErr)
		}
		t.Skipf("Periods table unavailable: %v", probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func envOr(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}

type recordedPeriodAudit struct {
	events []periods.AuditRecordInput
}

func (r *recordedPeriodAudit) RecordEvent(_ context.Context, ev periods.AuditRecordInput) error {
	r.events = append(r.events, ev)
	return nil
}

func (r *recordedPeriodAudit) RecordEventInTxn(_ context.Context, _ *spanner.ReadWriteTransaction, ev periods.AuditRecordInput) error {
	r.events = append(r.events, ev)
	return nil
}

func TestCutoffCalculation_TimezoneBoundaries(t *testing.T) {
	// Asia/Tashkent is UTC+05:00.
	// 2026-10-31 19:00:00Z = 2026-11-01 00:00:00+05:00 -> November!
	startUTC, endUTC, cutoffLocal, tz, err := periods.ComputePeriodWindow("2026-10", "Asia/Tashkent")
	if err != nil {
		t.Fatalf("ComputePeriodWindow: %v", err)
	}
	if tz != "Asia/Tashkent" {
		t.Fatalf("tz = %q, want Asia/Tashkent", tz)
	}
	wantStart := time.Date(2026, 9, 30, 19, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 10, 31, 19, 0, 0, 0, time.UTC)
	if !startUTC.Equal(wantStart) {
		t.Fatalf("startUTC = %v, want %v", startUTC, wantStart)
	}
	if !endUTC.Equal(wantEnd) {
		t.Fatalf("endUTC = %v, want %v", endUTC, wantEnd)
	}
	if !strings.HasPrefix(cutoffLocal, "2026-10-31T23:59:59+05:00") {
		t.Fatalf("unexpected cutoffLocal %q", cutoffLocal)
	}

	if _, _, _, err := periods.ValidatePeriodID("2026-13"); !errors.Is(err, periods.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for month 13, got %v", err)
	}
	if _, _, _, err := periods.ValidatePeriodID("2026-00"); !errors.Is(err, periods.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for month 00, got %v", err)
	}

	t.Setenv("PEGASUSX_ENV", "production")
	if _, err := periods.NewLocalSnapshotSink(t.TempDir()); !errors.Is(err, periods.ErrSnapshotWormRequired) {
		t.Fatalf("expected FailClosed ErrSnapshotWormRequired in prod, got %v", err)
	}
	if _, err := periods.DefaultSnapshotSink(nil).WriteSnapshot(context.Background(), periods.CloseSnapshot{}, []byte("{}"), []byte("a,b\n")); !errors.Is(err, periods.ErrSnapshotWormRequired) {
		t.Fatalf("expected DefaultSnapshotSink(nil) fail-closed in prod, got %v", err)
	}
}

func TestSpannerPeriods_Rollover_ManualLock_Checklist_Snapshot_Reopen_Outbox(t *testing.T) {
	client := openPeriodsSpannerClient(t)
	ctx := context.Background()

	// Wire CheckPeriodInTxn into gl for this test and restore on cleanup.
	gl.SetPeriodLockChecker(periods.CheckPeriodInTxn)
	t.Cleanup(func() { gl.SetPeriodLockChecker(nil) })

	entityID := "sup-p1-" + uuid.NewString()[:8]
	warehouseID := "wh-" + uuid.NewString()[:8]
	orderID := "ord-" + uuid.NewString()[:8]
	cashID := "cr-" + uuid.NewString()[:8]
	esfID := "esf-" + uuid.NewString()[:8]
	sliceID := "sl-" + uuid.NewString()[:8]
	adjID := "adj-" + uuid.NewString()[:8]
	invID := "inv-" + uuid.NewString()[:8]
	layerID := "lyr-" + uuid.NewString()[:8]

	glSvc := gl.NewService(client)
	if err := glSvc.SeedDefaults(ctx, entityID, "UZ", "UZS"); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}

	periodSvc := periods.NewService(client)
	wormDir := t.TempDir()
	sink, err := periods.NewLocalSnapshotSink(wormDir)
	if err != nil {
		t.Fatalf("NewLocalSnapshotSink: %v", err)
	}
	periodSvc.SetSnapshotSink(sink)
	audRec := &recordedPeriodAudit{}
	periodSvc.SetAuditRecorder(audRec)

	// Open 2026-10 and 2026-11.
	pOct, err := periodSvc.OpenPeriod(ctx, entityID, "2026-10", "Asia/Tashkent", "fin-lead-1", auth.ActorTypeHuman, auth.SupplierRoleFinanceManager)
	if err != nil {
		t.Fatalf("OpenPeriod 2026-10: %v", err)
	}
	if pOct.Status != periods.StatusOpen || pOct.Timezone != "Asia/Tashkent" {
		t.Fatalf("unexpected opened 2026-10 period: %+v", pOct)
	}
	if _, err := periodSvc.OpenPeriod(ctx, entityID, "2026-11", "Asia/Tashkent", "fin-lead-1", auth.ActorTypeHuman, auth.SupplierRoleFinanceManager); err != nil {
		t.Fatalf("OpenPeriod 2026-11: %v", err)
	}

	// Soft-close 2026-10.
	pOctSoft, err := periodSvc.SoftClose(ctx, entityID, "2026-10", "", "fin-lead-1", auth.ActorTypeHuman, auth.SupplierRoleFinanceManager)
	if err != nil {
		t.Fatalf("SoftClose 2026-10: %v", err)
	}
	if pOctSoft.Status != periods.StatusSoftClosed || pOctSoft.SoftClosedBy != "fin-lead-1" {
		t.Fatalf("expected SOFT_CLOSED, got %+v", pOctSoft)
	}

	// 1. Operational event in SOFT_CLOSED 2026-10 must roll into earliest OPEN period (2026-11) with OriginalPeriodId = 2026-10.
	lateOrderTime := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	opPostings := []gl.PostingInput{
		{AccountCode: gl.AccountAccountsReceivable, PartyType: "RETAILER", PartyID: "ret-1", Direction: gl.DirectionDebit, AmountMinor: 112000, Currency: "UZS"},
		{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 100000, Currency: "UZS"},
		{AccountCode: gl.AccountVatOutput, Direction: gl.DirectionCredit, AmountMinor: 12000, Currency: "UZS"},
	}
	resRollSoft, err := glSvc.PostEntry(ctx, gl.JournalEntryInput{
		EntityID:       entityID,
		SourceType:     "ORDER_DELIVERED",
		SourceID:       "ord-late-soft",
		OrderID:        "ord-late-soft",
		Memo:           "late order delivery after soft close",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: "roll-soft-1",
		PostedBy:       "worker",
		OccurredAt:     lateOrderTime,
		Postings:       opPostings,
	})
	if err != nil {
		t.Fatalf("PostEntry operational into SOFT_CLOSED: %v", err)
	}
	if resRollSoft.PeriodID != "2026-11" || resRollSoft.OriginalPeriodID != "2026-10" {
		t.Fatalf("expected rollover to 2026-11 from OriginalPeriodId=2026-10, got period=%q orig=%q", resRollSoft.PeriodID, resRollSoft.OriginalPeriodID)
	}

	// 2. Manual adjustment in SOFT_CLOSED 2026-10 must be ALLOWED directly into 2026-10!
	resManualSoft, err := glSvc.PostEntry(ctx, gl.JournalEntryInput{
		EntityID:       entityID,
		PeriodID:       "2026-10",
		SourceType:     "MANUAL",
		SourceID:       "adj-oct-1",
		Memo:           "period-end accrual & VAT adjustment",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: "manual-soft-oct-1",
		PostedBy:       "fin-lead-1",
		OccurredAt:     lateOrderTime,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountAccountsReceivable, PartyType: "RETAILER", PartyID: "ret-1", Direction: gl.DirectionDebit, AmountMinor: 112000, Currency: "UZS"},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 100000, Currency: "UZS"},
			{AccountCode: gl.AccountVatOutput, Direction: gl.DirectionCredit, AmountMinor: 12000, Currency: "UZS"},
		},
	})
	if err != nil {
		t.Fatalf("PostEntry MANUAL into SOFT_CLOSED: %v", err)
	}
	if resManualSoft.PeriodID != "2026-10" || resManualSoft.OriginalPeriodID != "" {
		t.Fatalf("expected manual entry in 2026-10 without rollover, got period=%q orig=%q", resManualSoft.PeriodID, resManualSoft.OriginalPeriodID)
	}

	// 3. Seed real blockers in live Spanner tables for 2026-10 (Asia/Tashkent UTC window: Sep 30 19Z..Oct 31 19Z).
	inWindowTS := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	_, err = client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("Warehouses", map[string]any{
				"WarehouseId":      warehouseID,
				"SupplierId":       entityID,
				"Name":             "Tashkent Central",
				"CoverageRadiusKm": 25.0,
				"IsActive":         true,
				"IsOnShift":        true,
				"CreatedAt":        spanner.CommitTimestamp,
				"UpdatedAt":        spanner.CommitTimestamp,
			}),
			spanner.InsertMap("CashReconciliations", map[string]any{
				"ReconciliationId":  cashID,
				"SupplierId":        entityID,
				"DriverId":          "drv-1",
				"ShiftDate":         civil.Date{Year: 2026, Month: 10, Day: 15},
				"ExpectedCashMinor": int64(50000),
				"DeclaredCashMinor": int64(48000),
				"DifferenceMinor":   int64(-2000),
				"Status":            "DISPUTED",
				"CreatedAt":         inWindowTS,
			}),
			spanner.InsertMap("EsfDocuments", map[string]any{
				"EntityId":        entityID,
				"EsfId":           esfID,
				"Direction":       "OUTBOUND",
				"CounterpartyTin": "301234567",
				"TotalMinor":      int64(112000),
				"VatMinor":        int64(12000),
				"VatRateBps":      int64(1200),
				"Status":          "SUBMITTED",
				"Operator":        "SOLIQ_SERVIS",
				"PeriodId":        "2026-10",
				"CreatedAt":       spanner.CommitTimestamp,
				"UpdatedAt":       spanner.CommitTimestamp,
			}),
			spanner.InsertMap("InvoiceSettlementSlices", map[string]any{
				"SliceId":         sliceID,
				"OrderId":         orderID,
				"SupplierId":      entityID,
				"CapturedLegId":   "leg-1",
				"GrossMinor":      int64(100000),
				"CommissionMinor": int64(5000),
				"NetPayoutMinor":  int64(95000),
				"Currency":        "UZS",
				"Status":          "UNSETTLED",
				"CreatedAt":       inWindowTS,
			}),
			spanner.InsertMap("InventoryAdjustments", map[string]any{
				"AdjustmentId": adjID,
				"WarehouseId":  warehouseID,
				"ProductId":    "prod-1",
				"DeltaQty":     int64(-2),
				"Status":       "PENDING",
				"CreatedAt":    inWindowTS,
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
			spanner.InsertMap("ArInvoices", map[string]any{
				"InvoiceId":      invID,
				"SupplierId":     entityID,
				"RetailerId":     "ret-1",
				"OrderId":        orderID,
				"Status":         "OPEN",
				"PrincipalMinor": int64(112000),
				"BalanceMinor":   int64(112000),
				"Currency":       "UZS",
				"CreditLeaveAt":  inWindowTS,
				"DueAt":          inWindowTS.Add(14 * 24 * time.Hour),
				"TermsDays":      int64(14),
				"Version":        int64(1),
				"CreatedAt":      inWindowTS,
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
			spanner.InsertMap("InventoryCostLayers", map[string]any{
				"EntityId":      entityID,
				"WarehouseId":   warehouseID,
				"ProductId":     "prod-1",
				"LayerId":       layerID,
				"QtyRemaining":  int64(20),
				"OriginalQty":   int64(20),
				"UnitCostMinor": int64(15000),
				"Currency":      "UZS",
				"ReceivedAt":    inWindowTS,
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
		})
	})
	if err != nil {
		t.Fatalf("seed blockers & snapshot data: %v", err)
	}

	failRes, closeErr := periodSvc.Close(ctx, entityID, "2026-10", "", "fin-lead-1", auth.ActorTypeHuman, auth.SupplierRoleFinanceManager)
	if !errors.Is(closeErr, periods.ErrChecklistFailed) {
		t.Fatalf("expected ErrChecklistFailed, got err=%v", closeErr)
	}
	if failRes.Checklist.BlockingCount != 4 || failRes.Checklist.ReadyToClose {
		t.Fatalf("expected 4 blockers (cash, esf, payout, inv adj), got %+v", failRes.Checklist)
	}

	// Resolve blockers.
	_, err = client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("CashReconciliations", map[string]any{
				"ReconciliationId": cashID,
				"Status":           "ACCEPTED",
			}),
			spanner.UpdateMap("EsfDocuments", map[string]any{
				"EntityId":  entityID,
				"EsfId":     esfID,
				"Status":    "ACCEPTED",
				"UpdatedAt": spanner.CommitTimestamp,
			}),
			spanner.UpdateMap("InvoiceSettlementSlices", map[string]any{
				"SliceId": sliceID,
				"Status":  "BATCHED",
			}),
			spanner.UpdateMap("InventoryAdjustments", map[string]any{
				"AdjustmentId": adjID,
				"Status":       "POSTED",
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
		})
	})
	if err != nil {
		t.Fatalf("resolve blockers: %v", err)
	}

	// 4. Close 2026-10 cleanly -> generates WORM snapshot JSON+CSV + SHA256.
	closeRes, err := periodSvc.Close(ctx, entityID, "2026-10", "", "fin-lead-1", auth.ActorTypeHuman, auth.SupplierRoleFinanceManager)
	if err != nil {
		t.Fatalf("Close 2026-10: %v", err)
	}
	if closeRes.Period.Status != periods.StatusClosed || closeRes.Period.SnapshotHash == "" || closeRes.Period.SnapshotURI == "" {
		t.Fatalf("unexpected closed period: %+v", closeRes.Period)
	}
	if !closeRes.Checklist.ReadyToClose || closeRes.Checklist.BlockingCount != 0 {
		t.Fatalf("expected clean checklist, got %+v", closeRes.Checklist)
	}
	// Verify SHA-256 on written JSON artifact matches SnapshotHash bit-for-bit!
	jsonPath := strings.TrimPrefix(closeRes.Period.SnapshotURI, "file://")
	rawJSON, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read snapshot file %s: %v", jsonPath, err)
	}
	sum := sha256.Sum256(rawJSON)
	if hex.EncodeToString(sum[:]) != closeRes.Period.SnapshotHash {
		t.Fatalf("snapshot SHA-256 mismatch: file=%s row=%s", hex.EncodeToString(sum[:]), closeRes.Period.SnapshotHash)
	}
	csvPath := strings.TrimSuffix(jsonPath, ".json") + ".csv"
	rawCSV, err := os.ReadFile(csvPath)
	if err != nil || !strings.Contains(string(rawCSV), "VALUATION_LINE") || !strings.Contains(string(rawCSV), "TRIAL_BALANCE") {
		t.Fatalf("expected frozen CSV file at %s with VALUATION_LINE & TRIAL_BALANCE, err=%v", csvPath, err)
	}
	if closeRes.Snapshot.ValuationSummary.TotalValueMinor != 300000 || closeRes.Snapshot.ARSummary.OpenAmountMinor != 112000 {
		t.Fatalf("unexpected snapshot numbers: val=%+v ar=%+v", closeRes.Snapshot.ValuationSummary, closeRes.Snapshot.ARSummary)
	}
	if closeRes.Snapshot.VATSummary.GLOutputVATMinor != 12000 || closeRes.Snapshot.VATSummary.ESFOutboundVATMinor != 12000 || closeRes.Snapshot.VATSummary.VarianceMinor != 0 {
		t.Fatalf("unexpected VAT summary: %+v", closeRes.Snapshot.VATSummary)
	}

	// 5. Hard close lock: MANUAL posting into CLOSED 2026-10 MUST return gl.ErrPeriodClosed!
	_, err = glSvc.PostEntry(ctx, gl.JournalEntryInput{
		EntityID:       entityID,
		PeriodID:       "2026-10",
		SourceType:     "MANUAL",
		SourceID:       "manual-closed-blocked",
		Memo:           "should fail on CLOSED period",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: "manual-closed-blocked-1",
		PostedBy:       "fin-lead-1",
		OccurredAt:     lateOrderTime,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountCashOffice, Direction: gl.DirectionDebit, AmountMinor: 1000, Currency: "UZS"},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 1000, Currency: "UZS"},
		},
	})
	if !errors.Is(err, gl.ErrPeriodClosed) {
		t.Fatalf("expected gl.ErrPeriodClosed on MANUAL into CLOSED 2026-10, got %v", err)
	}

	// 6. Late operational event occurring in CLOSED 2026-10 rolls forward to 2026-11 stamping OriginalPeriodId!
	resRollClosed, err := glSvc.PostEntry(ctx, gl.JournalEntryInput{
		EntityID:       entityID,
		SourceType:     "AR_PAYMENT",
		SourceID:       "pay-late-closed",
		Memo:           "late payment settled in Nov for Oct timestamp",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: "roll-closed-1",
		PostedBy:       "worker",
		OccurredAt:     lateOrderTime,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountBankOperating, Direction: gl.DirectionDebit, AmountMinor: 50000, Currency: "UZS"},
			{AccountCode: gl.AccountAccountsReceivable, PartyType: "RETAILER", PartyID: "ret-1", Direction: gl.DirectionCredit, AmountMinor: 50000, Currency: "UZS"},
		},
	})
	if err != nil {
		t.Fatalf("PostEntry AR_PAYMENT into CLOSED period should roll forward, got %v", err)
	}
	if resRollClosed.PeriodID != "2026-11" || resRollClosed.OriginalPeriodID != "2026-10" {
		t.Fatalf("expected rollover from CLOSED 2026-10 -> 2026-11, got period=%q orig=%q", resRollClosed.PeriodID, resRollClosed.OriginalPeriodID)
	}

	// Verify GL hash chain remains 100% valid across rolled-over entries!
	if chainRes, err := gl.VerifyEntityHashChain(ctx, client, entityID); err != nil || !chainRes.Valid {
		t.Fatalf("VerifyEntityHashChain failed: res=%+v err=%v", chainRes, err)
	}

	// 7. Reopen requires non-empty reason, forbids agent, and requires 2 distinct approvers!
	if _, err := periodSvc.Reopen(ctx, periods.ReopenInput{
		EntityID: entityID, PeriodID: "2026-10", Reason: "", ActorID: "admin-1", CoApproverID: "admin-2",
	}); !errors.Is(err, periods.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput on empty reason, got %v", err)
	}
	if _, err := periodSvc.Reopen(ctx, periods.ReopenInput{
		EntityID: entityID, PeriodID: "2026-10", Reason: "agent fix", ActorID: "agent-bot", ActorType: auth.ActorTypeAgent, CoApproverID: "admin-2",
	}); !errors.Is(err, periods.ErrSoDViolation) {
		t.Fatalf("expected ErrSoDViolation for agent reopen, got %v", err)
	}
	if _, err := periodSvc.Reopen(ctx, periods.ReopenInput{
		EntityID: entityID, PeriodID: "2026-10", Reason: "need adj", ActorID: "admin-1", ActorType: auth.ActorTypeHuman, CoApproverID: "admin-1",
	}); !errors.Is(err, periods.ErrApprovalRequired) {
		t.Fatalf("expected ErrApprovalRequired on single approver, got %v", err)
	}

	reopenRes, err := periodSvc.Reopen(ctx, periods.ReopenInput{
		EntityID: entityID, PeriodID: "2026-10", Reason: "Late customs VAT invoice", ActorID: "admin-1", ActorType: auth.ActorTypeHuman, CoApproverID: "admin-2",
	})
	if err != nil {
		t.Fatalf("Reopen 2026-10 with 2 approvers: %v", err)
	}
	if reopenRes.Period == nil || reopenRes.Period.Status != periods.StatusReopened || reopenRes.Period.ReopenReason != "Late customs VAT invoice" {
		t.Fatalf("unexpected reopened period: %+v", reopenRes.Period)
	}

	// Confirm OutboxEvents emitted period.soft_closed, period.closed, period.reopened on TopicLedger!
	ro := client.ReadOnlyTransaction()
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT TopicName, Payload
		      FROM OutboxEvents
		      WHERE AggregateType = @agg AND AggregateId = @aggId
		      ORDER BY CreatedAt ASC`,
		Params: map[string]any{
			"agg":   events.AggregatePeriod,
			"aggId": entityID + ":2026-10",
		},
	})
	defer iter.Stop()
	var topics []string
	var types []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("query OutboxEvents: %v", err)
		}
		var top string
		var payloadBytes []byte
		if err := row.Columns(&top, &payloadBytes); err != nil {
			t.Fatalf("scan OutboxEvents: %v", err)
		}
		var env struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(payloadBytes, &env); err != nil {
			t.Fatalf("unmarshal OutboxEvents payload: %v", err)
		}
		topics = append(topics, top)
		types = append(types, env.Type)
	}
	if len(types) != 3 || types[0] != events.EventPeriodSoftClosed || types[1] != events.EventPeriodClosed || types[2] != events.EventPeriodReopened {
		t.Fatalf("unexpected period outbox sequence: types=%v topics=%v", types, topics)
	}
	for _, top := range topics {
		if top != events.TopicLedger {
			t.Fatalf("expected TopicLedger (%s), got %q", events.TopicLedger, top)
		}
	}
	if len(audRec.events) < 4 {
		t.Fatalf("expected >=4 recorded period audit events, got %d", len(audRec.events))
	}
}
