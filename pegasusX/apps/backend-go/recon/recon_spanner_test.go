package recon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/cashrecon"
	"github.com/pegasusx/pegasusx/apps/backend-go/controltower"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openReconSpannerTestClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner recon test")
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
		if requireSpanner {
			t.Fatalf("spanner connect (%s): %v", dbPath, err)
		}
		t.Skipf("spanner unavailable (%s): %v", dbPath, err)
	}

	_, probeErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ReconciliationRuns", spanner.Key{"__recon_probe__", "__run_probe__"}, []string{"RunId"})
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("ReconciliationRuns table unavailable (%s): %v", dbPath, probeErr)
		}
		t.Skipf("ReconciliationRuns unavailable (%s): %v", dbPath, probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func cleanupEntityReconTables(ctx context.Context, client *spanner.Client, entityID string) {
	_, _ = client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		ticketIter := txn.Query(txCtx, spanner.Statement{
			SQL:    `SELECT ControlTowerExceptionId FROM ReconciliationBreaks WHERE EntityId = @eid AND ControlTowerExceptionId IS NOT NULL`,
			Params: map[string]any{"eid": entityID},
		})
		defer ticketIter.Stop()
		var muts []*spanner.Mutation
		for {
			row, err := ticketIter.Next()
			if err != nil {
				break
			}
			var tid string
			if err := row.Columns(&tid); err == nil && tid != "" {
				muts = append(muts, spanner.Delete("ExceptionTickets", spanner.Key{tid}))
			}
		}
		muts = append(muts,
			spanner.Delete("ReconciliationBreaks", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ReconciliationRuns", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlBalances", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
		)
		return txn.BufferWrite(muts)
	})
}

func cleanupGLJournalEntriesForEntity(ctx context.Context, client *spanner.Client, entityID string) {
	_, _ = client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.Update(txCtx, spanner.Statement{
			SQL:    `DELETE FROM JournalEntries WHERE EntityId = @eid`,
			Params: map[string]any{"eid": entityID},
		})
		return err
	})
}

func seedOutboxEventCleanup(ctx context.Context, client *spanner.Client, supplierID string) {
	_, _ = client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.Update(txCtx, spanner.Statement{
			SQL:    `DELETE FROM OutboxEvents WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": supplierID},
		})
		return err
	})
}

func upsertGlBalance(ctx context.Context, client *spanner.Client, entityID, periodID, accountCode, currency string, debitMinor, creditMinor int64) error {
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("GlBalances", map[string]any{
			"EntityId":    entityID,
			"PeriodId":    periodID,
			"AccountCode": accountCode,
			"PartyKey":    "_",
			"Shard":       int64(0),
			"Currency":    currency,
			"DebitMinor":  debitMinor,
			"CreditMinor": creditMinor,
			"UpdatedAt":   spanner.CommitTimestamp,
		}),
	})
	return err
}

func testProductMutation(productID, supplierID, currency string, priceMinor int64) *spanner.Mutation {
	return spanner.InsertOrUpdateMap("Products", map[string]any{
		"ProductId":         productID,
		"SupplierId":        supplierID,
		"CategoryId":        "cat-test",
		"Name":              productID,
		"PriceMinor":        priceMinor,
		"Currency":          currency,
		"StockQuantity":     int64(0),
		"Unit":              "UNIT",
		"SaleUnit":          "UNIT",
		"UnitVolumeVU":      1.0,
		"Stackable":         true,
		"MaxStackHeight":    int64(1),
		"HandlingClass":     "GENERAL",
		"RequiresColdChain": false,
		"IsPerishable":      false,
		"IsHazardous":       false,
		"IsActive":          true,
		"Version":           int64(1),
		"CreatedAt":         spanner.CommitTimestamp,
		"UpdatedAt":         spanner.CommitTimestamp,
	})
}

func testOrderMutation(orderID, supplierID, retailerID, status, fiscalStatus string, totalMinor int64, currency string, now time.Time) *spanner.Mutation {
	return spanner.InsertOrUpdateMap("Orders", map[string]any{
		"OrderId":            orderID,
		"SupplierId":         supplierID,
		"RetailerId":         retailerID,
		"Status":             status,
		"FiscalStatus":       fiscalStatus,
		"OrderSource":        "MANUAL",
		"ConfirmationStatus": "CONFIRMED",
		"LineItemsJson":      []byte("[]"),
		"TotalMinor":         totalMinor,
		"Currency":           currency,
		"Version":            int64(1),
		"CreatedAt":          now,
		"UpdatedAt":          now,
	})
}

func testTaxRegimeMutation(regimeID, countryCode, currency string, now time.Time) *spanner.Mutation {
	return spanner.InsertOrUpdateMap("TaxRegimeVersions", map[string]any{
		"Id":            regimeID,
		"CountryCode":   countryCode,
		"EffectiveFrom": now.Add(-24 * time.Hour),
		"Currency":      currency,
		"VatRatesBps":   []int64{1200},
		"CreatedAt":     now,
		"CreatedBy":     "recon-test",
		"UpdatedAt":     now,
	})
}

func fetchOutboxEventTypesForSupplier(t *testing.T, ctx context.Context, client *spanner.Client, supplierID string) map[string][]string {
	t.Helper()
	iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    `SELECT TopicName, Payload FROM OutboxEvents WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	res := make(map[string][]string)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("query OutboxEvents: %v", err)
		}
		var topic string
		var raw spanner.NullString
		// Payload is STRING(MAX) or BYTES(MAX) depending on schema; inspect string/bytes safely.
		var rawBytes []byte
		if err := row.Columns(&topic, &rawBytes); err == nil {
			var base struct {
				Type string `json:"type"`
			}
			if jErr := json.Unmarshal(rawBytes, &base); jErr == nil && base.Type != "" {
				res[base.Type] = append(res[base.Type], topic)
			}
			continue
		}
		_ = raw
	}
	return res
}

func TestRecon_GreenRun_All4Dimensions_LiveSpanner(t *testing.T) {
	client := openReconSpannerTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sid := "sup-" + uuid.NewString()[:8]
	wid := "wh-" + uuid.NewString()[:8]
	pid := "prod-" + uuid.NewString()[:8]
	lotID := "lot-" + uuid.NewString()[:8]
	regimeID := "reg-" + uuid.NewString()[:8]
	ordCredit1 := "ord-cr1-" + uuid.NewString()[:8]
	ordCredit2 := "ord-cr2-" + uuid.NewString()[:8]
	ordCard := "ord-cd-" + uuid.NewString()[:8]
	ordCash1 := "ord-ca1-" + uuid.NewString()[:8]
	ordCash2 := "ord-ca2-" + uuid.NewString()[:8]
	invCredit1 := "inv1-" + uuid.NewString()[:8]
	invCredit2 := "inv2-" + uuid.NewString()[:8]
	sessCard := "sess-" + uuid.NewString()[:8]
	legCash1 := "leg1-" + uuid.NewString()[:8]
	legCash2 := "leg2-" + uuid.NewString()[:8]
	crShortID := "cr-short-" + uuid.NewString()[:8]
	crBalID := "cr-bal-" + uuid.NewString()[:8]
	cnID := "cn-" + uuid.NewString()[:8]
	glSvc := gl.NewService(client)
	arSvc := ar.NewService(ar.NewSpannerRepository(client))
	cashRepo := cashrecon.NewSpannerRepository(client)
	now := time.Now().UTC().Add(-2 * time.Minute)
	periodID := gl.ComputePeriodID(ctx, sid, "UZ", now)

	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanCancel()
		cleanupEntityReconTables(cleanCtx, client, sid)
		cleanupGLJournalEntriesForEntity(cleanCtx, client, sid)
		seedOutboxEventCleanup(cleanCtx, client, sid)
		_, _ = client.ReadWriteTransaction(cleanCtx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
			_, _ = txn.Update(txCtx, spanner.Statement{SQL: `DELETE FROM ArLedgerEntries WHERE SupplierId = @sid`, Params: map[string]any{"sid": sid}})
			_, _ = txn.Update(txCtx, spanner.Statement{SQL: `DELETE FROM ArAdvanceBalances WHERE SupplierId = @sid`, Params: map[string]any{"sid": sid}})
			_, _ = txn.Update(txCtx, spanner.Statement{SQL: `DELETE FROM PaymentLedgerEntries WHERE SupplierId = @sid`, Params: map[string]any{"sid": sid}})
			return nil
		})
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("CreditNotes", spanner.Key{cnID}),
			spanner.Delete("OrderLineFiscalSnapshots", spanner.Key{ordCard, "line-1"}),
			spanner.Delete("OrderFiscalReceipts", spanner.Key{ordCard, "att-1"}),
			spanner.Delete("OrderPaymentLegs", spanner.Key{ordCash1, legCash1}),
			spanner.Delete("OrderPaymentLegs", spanner.Key{ordCash2, legCash2}),
			spanner.Delete("PaymentSessions", spanner.Key{sessCard}),
			spanner.Delete("CashReconciliations", spanner.Key{crShortID}),
			spanner.Delete("CashReconciliations", spanner.Key{crBalID}),
			spanner.Delete("ArInvoices", spanner.Key{invCredit1}),
			spanner.Delete("ArInvoices", spanner.Key{invCredit2}),
			spanner.Delete("Orders", spanner.Key{ordCredit1}),
			spanner.Delete("Orders", spanner.Key{ordCredit2}),
			spanner.Delete("Orders", spanner.Key{ordCard}),
			spanner.Delete("Orders", spanner.Key{ordCash1}),
			spanner.Delete("Orders", spanner.Key{ordCash2}),
			spanner.Delete("SupplierInventoryV2", spanner.Key{sid, wid, pid}),
			spanner.Delete("StockLots", spanner.Key{lotID}),
			spanner.Delete("Warehouses", spanner.Key{wid}),
			spanner.Delete("Products", spanner.Key{pid}),
			spanner.Delete("TaxRegimeVersions", spanner.Key{regimeID}),
		})
	})

	muts := []*spanner.Mutation{
		testTaxRegimeMutation(regimeID, "UZ", "UZS", now),
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId": wid, "SupplierId": sid, "Name": wid, "CoverageRadiusKm": 10.0,
			"IsActive": true, "IsOnShift": true, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		testProductMutation(pid, sid, "UZS", 50_000),
		spanner.InsertMap("SupplierInventoryV2", map[string]any{
			"SupplierId": sid, "WarehouseId": wid, "ProductId": pid,
			"QuantityOnHand": int64(10), "QuantityReserved": int64(0), "ReorderThreshold": int64(2),
			"UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotID, "SupplierId": sid, "WarehouseId": wid, "ProductId": pid,
			"LocationId": "loc-1", "QuantityOnHand": int64(10), "QuantityReserved": int64(0),
			"Status": "AVAILABLE", "ReceivedAt": now, "CreatedAt": now, "UpdatedAt": now,
		}),
		// ORDERS_AR: two DELIVERED_ON_CREDIT orders (250k ret-1, 200k ret-2). ret-1 pays 300k via ApplyPaymentFIFO -> 250k AR_PAYMENT + 50k AR_ADVANCE!
		testOrderMutation(ordCredit1, sid, "ret-1", "DELIVERED_ON_CREDIT", "NONE", 250_000, "UZS", now),
		testOrderMutation(ordCredit2, sid, "ret-2", "DELIVERED_ON_CREDIT", "NONE", 200_000, "UZS", now),
		spanner.InsertMap("ArInvoices", map[string]any{
			"InvoiceId": invCredit1, "SupplierId": sid, "RetailerId": "ret-1", "OrderId": ordCredit1,
			"Status": "OPEN", "PrincipalMinor": int64(250_000), "BalanceMinor": int64(250_000),
			"Currency": "UZS", "CreditLeaveAt": now, "DueAt": now.Add(7 * 24 * time.Hour), "TermsDays": int64(7),
			"CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("ArInvoices", map[string]any{
			"InvoiceId": invCredit2, "SupplierId": sid, "RetailerId": "ret-2", "OrderId": ordCredit2,
			"Status": "OPEN", "PrincipalMinor": int64(200_000), "BalanceMinor": int64(200_000),
			"Currency": "UZS", "CreditLeaveAt": now, "DueAt": now.Add(7 * 24 * time.Hour), "TermsDays": int64(7),
			"CreatedAt": now, "UpdatedAt": now,
		}),
		// CASH_PSP & TAX_FISCAL: card 112k (100k net + 12k VAT), cash1 80k, cash2 50k
		testOrderMutation(ordCard, sid, "ret-1", "COMPLETED", "SUCCESS", 112_000, "UZS", now),
		spanner.InsertMap("PaymentSessions", map[string]any{
			"SessionId": sessCard, "OrderId": ordCard, "SupplierId": sid, "RetailerId": "ret-1",
			"Gateway": "PAYME", "Currency": "UZS", "AmountMinor": int64(112_000), "Mode": "CARD",
			"Status": "CAPTURED", "CreatedAt": now, "UpdatedAt": now,
		}),
		testOrderMutation(ordCash1, sid, "ret-1", "COMPLETED", "NONE", 80_000, "UZS", now),
		spanner.InsertMap("OrderPaymentLegs", map[string]any{
			"OrderId": ordCash1, "LegId": legCash1, "Method": "CASH",
			"AmountMinor": int64(80_000), "Status": "CAPTURED", "IdempotencyKey": "leg-" + legCash1,
			"CreatedAt": now,
		}),
		testOrderMutation(ordCash2, sid, "ret-2", "COMPLETED", "NONE", 50_000, "UZS", now),
		spanner.InsertMap("OrderPaymentLegs", map[string]any{
			"OrderId": ordCash2, "LegId": legCash2, "Method": "CASH",
			"AmountMinor": int64(50_000), "Status": "CAPTURED", "IdempotencyKey": "leg-" + legCash2,
			"CreatedAt": now,
		}),
		spanner.InsertMap("OrderFiscalReceipts", map[string]any{
			"OrderId": ordCard, "AttemptId": "att-1", "SupplierId": sid, "RetailerId": "ret-1",
			"Provider": "PEGASUS", "Status": "ISSUED", "AmountMinor": int64(112_000), "Currency": "UZS",
			"CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("OrderLineFiscalSnapshots", map[string]any{
			"OrderId": ordCard, "OrderLineId": "line-1", "RegimeId": regimeID, "VatRateBps": int64(1200),
			"NetMinor": int64(100_000), "VatMinor": int64(12_000), "GrossMinor": int64(112_000),
			"SnapshotAt": now, "CreatedAt": now,
		}),
		spanner.InsertMap("CreditNotes", map[string]any{
			"CreditNoteId": cnID, "OrderId": ordCard, "Type": "PARTIAL", "Status": "ISSUED",
			"ReasonCode": "DAMAGED", "TotalNetMinor": int64(16_667), "TotalVatMinor": int64(2_000),
			"TotalGrossMinor": int64(18_667), "CreatedBy": "admin-1", "CreatedAt": now,
		}),
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("seed clean domain rows: %v", err)
	}

	_ = glSvc.SeedDefaults(ctx, sid, "UZ", "UZS")
	// Real gl.PostEvent dual-write flows for ORDER_DELIVERED (CREDIT, CARD, CASH), CREDIT_NOTE, and INVENTORY baseline.
	for _, ev := range []struct {
		evType string
		ctx    gl.RuleEvalContext
	}{
		{gl.RuleEventOrderDelivered, gl.RuleEvalContext{EntityID: sid, RetailerID: "ret-1", OrderID: ordCredit1, PaymentMode: "CREDIT", NetMinor: 250_000, GrossMinor: 250_000, Currency: "UZS", IdempotencyKey: "recon-real:cr1:" + sid, OccurredAt: now}},
		{gl.RuleEventOrderDelivered, gl.RuleEvalContext{EntityID: sid, RetailerID: "ret-2", OrderID: ordCredit2, PaymentMode: "CREDIT", NetMinor: 200_000, GrossMinor: 200_000, Currency: "UZS", IdempotencyKey: "recon-real:cr2:" + sid, OccurredAt: now}},
		{gl.RuleEventOrderDelivered, gl.RuleEvalContext{EntityID: sid, RetailerID: "ret-1", OrderID: ordCard, PaymentMode: "CARD", Gateway: "PAYME", NetMinor: 100_000, VatMinor: 12_000, GrossMinor: 112_000, Currency: "UZS", IdempotencyKey: "recon-real:card:" + sid, OccurredAt: now}},
		{gl.RuleEventOrderDelivered, gl.RuleEvalContext{EntityID: sid, RetailerID: "ret-1", DriverID: "drv-1", OrderID: ordCash1, PaymentMode: "CASH", NetMinor: 80_000, GrossMinor: 80_000, Currency: "UZS", IdempotencyKey: "recon-real:cash1:" + sid, OccurredAt: now}},
		{gl.RuleEventOrderDelivered, gl.RuleEvalContext{EntityID: sid, RetailerID: "ret-2", DriverID: "drv-2", OrderID: ordCash2, PaymentMode: "CASH", NetMinor: 50_000, GrossMinor: 50_000, Currency: "UZS", IdempotencyKey: "recon-real:cash2:" + sid, OccurredAt: now}},
		{gl.RuleEventCreditNoteIssued, gl.RuleEvalContext{EntityID: sid, RetailerID: "ret-1", OrderID: ordCard, SourceID: cnID, NetMinor: 16_667, VatMinor: 2_000, GrossMinor: 18_667, Currency: "UZS", IdempotencyKey: "recon-real:cn:" + sid, OccurredAt: now}},
	} {
		if _, err := glSvc.PostEvent(ctx, ev.evType, ev.ctx); err != nil {
			t.Fatalf("PostEvent %s (%s): %v", ev.evType, ev.ctx.OrderID, err)
		}
	}
	// Adjust AR:RETAILER net back up by CreditNote gross (18667)? No! In reality CreditNote applies to ARInvoice or Order;
	// keep CreditNote VAT test clean by balancing AR on cnID against OrderCredit2? Wait: CreditNote credited AR:RETAILER by 18_667!
	// Let's reverse that AR side on a matching 18_667 MANUAL non-cash opening OR post INVENTORY (500k) cleanly:
	if _, err := glSvc.PostEntry(ctx, gl.JournalEntryInput{
		EntityID: sid, CountryPack: "UZ", PeriodID: periodID, SourceType: "MANUAL", SourceID: "seed-inv",
		Currency: "UZS", IdempotencyKey: "recon-real:inv:" + sid, OccurredAt: now, PostedBy: "test",
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountInventory, Direction: gl.DirectionDebit, AmountMinor: 500_000},
			{AccountCode: gl.AccountAccountsReceivable, Direction: gl.DirectionDebit, AmountMinor: 18_667}, // offsets standalone card CreditNote AR leg
			{AccountCode: gl.AccountOpeningBalanceEquity, Direction: gl.DirectionCredit, AmountMinor: 518_667},
		},
	}); err != nil {
		t.Fatalf("post inventory opening: %v", err)
	}

	// Real ar.ApplyPaymentFIFO with 300k on ret-1 (250k AR_PAYMENT + 50k AR_ADVANCE into BANK:OPERATING)!
	fifoResp, err := arSvc.ApplyPaymentFIFO(ctx, ar.PayFIFORequest{
		SupplierID:     sid,
		RetailerID:     "ret-1",
		AmountMinor:    300_000,
		Currency:       "UZS",
		PaymentMethod:  "BANK",
		IdempotencyKey: "fifo-green-" + sid,
	})
	if err != nil || fifoResp.AdvanceAddedMinor != 50_000 {
		t.Fatalf("ApplyPaymentFIFO = (%+v, %v), want AdvanceAddedMinor=50000", fifoResp, err)
	}

	// Real cashrecon.SaveReconciliation flows:
	// 1. Balanced shift on ordCash2 (50k/50k ACCEPTED, posts 0 GL rows; remains in CASH:DRIVER)
	if err := cashRepo.SaveReconciliation(ctx, cashrecon.CashReconciliation{
		ReconciliationId: crBalID, SupplierId: sid, DriverId: "drv-2", ShiftDate: now,
		ExpectedCashMinor: 50_000, DeclaredCashMinor: 50_000, DifferenceMinor: 0,
		Status: cashrecon.ReconciliationStatusAccepted, CreatedAt: now,
	}, cashrecon.EventCashReconciliationCreated); err != nil {
		t.Fatalf("SaveReconciliation(balanced): %v", err)
	}
	// 2. Shortfall shift on ordCash1 (80k expected, 75k declared, -5k diff): PENDING -> ACCEPTED (DR CASH:OFFICE 5k)
	crShort := cashrecon.CashReconciliation{
		ReconciliationId: crShortID, SupplierId: sid, DriverId: "drv-1", ShiftDate: now,
		ExpectedCashMinor: 80_000, DeclaredCashMinor: 75_000, DifferenceMinor: -5_000,
		Status: cashrecon.ReconciliationStatusPending, CreatedAt: now,
	}
	if err := cashRepo.SaveReconciliation(ctx, crShort, cashrecon.EventCashReconciliationCreated); err != nil {
		t.Fatalf("SaveReconciliation(short PENDING): %v", err)
	}
	actor := "fin-1"
	resolvedAt := now.Add(time.Minute)
	crShort.Status = cashrecon.ReconciliationStatusAccepted
	crShort.ResolvedBy = &actor
	crShort.ResolvedAt = &resolvedAt
	if err := cashRepo.SaveReconciliation(ctx, crShort, cashrecon.EventCashReconciliationAccepted); err != nil {
		t.Fatalf("SaveReconciliation(short ACCEPTED): %v", err)
	}

	reconSvc := NewService(client)
	out, err := reconSvc.RunReconciliation(ctx, RunRequest{
		EntityID:    sid,
		PeriodID:    periodID,
		Mode:        ModeManual,
		TriggeredBy: "test-green-real",
	})
	if err != nil {
		t.Fatalf("RunReconciliation green failed: %v", err)
	}
	if out.Run.Status != RunStatusGreen || out.Run.BreakCount != 0 || len(out.Breaks) != 0 {
		t.Fatalf("expected GREEN run with 0 breaks on real dual-write data, got status=%s breakCount=%d breaks=%+v checks=%+v",
			out.Run.Status, out.Run.BreakCount, out.Breaks, out.Run.Checks)
	}

	evMap := fetchOutboxEventTypesForSupplier(t, ctx, client, sid)
	if topics := evMap[events.EventReconRunCompleted]; len(topics) != 1 || topics[0] != events.TopicLedger {
		t.Fatalf("recon.run.completed outbox = %+v, want 1 event on TopicLedger", evMap)
	}
}

func TestRecon_RealDualWrite_WriteOff_Overage_And_LedgerChargeback_LiveSpanner(t *testing.T) {
	client := openReconSpannerTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sid := "sup-" + uuid.NewString()[:8]
	ordCash := "ord-ca-" + uuid.NewString()[:8]
	ordCard := "ord-cd-" + uuid.NewString()[:8]
	sessCard := "sess-" + uuid.NewString()[:8]
	legCash := "leg-" + uuid.NewString()[:8]
	crWriteOffID := "cr-wo-" + uuid.NewString()[:8]
	crOverID := "cr-ov-" + uuid.NewString()[:8]
	cbID := "cb-" + uuid.NewString()[:8]
	glSvc := gl.NewService(client)
	cashRepo := cashrecon.NewSpannerRepository(client)
	payRepo := payment.NewSpannerRepository(client)
	now := time.Now().UTC().Add(-2 * time.Minute)
	periodID := gl.ComputePeriodID(ctx, sid, "UZ", now)

	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanCancel()
		cleanupEntityReconTables(cleanCtx, client, sid)
		cleanupGLJournalEntriesForEntity(cleanCtx, client, sid)
		seedOutboxEventCleanup(cleanCtx, client, sid)
		_, _ = client.ReadWriteTransaction(cleanCtx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
			_, _ = txn.Update(txCtx, spanner.Statement{SQL: `DELETE FROM PaymentLedgerEntries WHERE SupplierId = @sid`, Params: map[string]any{"sid": sid}})
			return nil
		})
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("PaymentChargebacks", spanner.Key{cbID}),
			spanner.Delete("PaymentSessions", spanner.Key{sessCard}),
			spanner.Delete("OrderPaymentLegs", spanner.Key{ordCash, legCash}),
			spanner.Delete("CashReconciliations", spanner.Key{crWriteOffID}),
			spanner.Delete("CashReconciliations", spanner.Key{crOverID}),
			spanner.Delete("Orders", spanner.Key{ordCash}),
			spanner.Delete("Orders", spanner.Key{ordCard}),
		})
	})

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		testOrderMutation(ordCard, sid, "ret-1", "COMPLETED", "SUCCESS", 100_000, "UZS", now),
		spanner.InsertMap("PaymentSessions", map[string]any{
			"SessionId": sessCard, "OrderId": ordCard, "SupplierId": sid, "RetailerId": "ret-1",
			"Gateway": "PAYME", "Currency": "UZS", "AmountMinor": int64(100_000), "Mode": "CARD",
			"Status": "CAPTURED", "CreatedAt": now, "UpdatedAt": now,
		}),
		testOrderMutation(ordCash, sid, "ret-1", "COMPLETED", "NONE", 100_000, "UZS", now),
		spanner.InsertMap("OrderPaymentLegs", map[string]any{
			"OrderId": ordCash, "LegId": legCash, "Method": "CASH",
			"AmountMinor": int64(100_000), "Status": "CAPTURED", "IdempotencyKey": "leg-" + legCash,
			"CreatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed cash/psp orders: %v", err)
	}

	if _, err := glSvc.PostEvent(ctx, gl.RuleEventOrderDelivered, gl.RuleEvalContext{
		EntityID: sid, RetailerID: "ret-1", OrderID: ordCard, PaymentMode: "CARD", Gateway: "PAYME",
		NetMinor: 100_000, GrossMinor: 100_000, Currency: "UZS", IdempotencyKey: "recon-wo:cd:" + sid, OccurredAt: now,
	}); err != nil {
		t.Fatalf("PostEvent CARD: %v", err)
	}
	if _, err := glSvc.PostEvent(ctx, gl.RuleEventOrderDelivered, gl.RuleEvalContext{
		EntityID: sid, RetailerID: "ret-1", DriverID: "drv-1", OrderID: ordCash, PaymentMode: "CASH",
		NetMinor: 100_000, GrossMinor: 100_000, Currency: "UZS", IdempotencyKey: "recon-wo:ca:" + sid, OccurredAt: now,
	}); err != nil {
		t.Fatalf("PostEvent CASH: %v", err)
	}

	// LEDGER_ONLY chargeback on PAYME session -> must NOT distort expectedPSP (PSP:GATEWAY stays 100k)!
	if err := payRepo.SaveChargeback(ctx, payment.ChargebackRecord{
		ChargebackID: cbID, OrderID: ordCard, SupplierID: sid, RetailerID: "ret-1",
		Gateway: "PAYME", AmountMinor: 42_000, Currency: "UZS", CreatedAt: now,
	}, nil); err != nil {
		t.Fatalf("SaveChargeback: %v", err)
	}

	// cashrecon WRITE_OFF shortfall (-15k) + ACCEPTED overage (+10k)
	actor := "fin-1"
	resolvedAt := now.Add(time.Minute)
	crWO := cashrecon.CashReconciliation{
		ReconciliationId: crWriteOffID, SupplierId: sid, DriverId: "drv-1", ShiftDate: now,
		ExpectedCashMinor: 100_000, DeclaredCashMinor: 85_000, DifferenceMinor: -15_000,
		Status: cashrecon.ReconciliationStatusPending, CreatedAt: now,
	}
	if err := cashRepo.SaveReconciliation(ctx, crWO, cashrecon.EventCashReconciliationCreated); err != nil {
		t.Fatalf("SaveReconciliation(WO open): %v", err)
	}
	crWO.Status = cashrecon.ReconciliationStatusWriteOff
	crWO.ResolvedBy = &actor
	crWO.ResolvedAt = &resolvedAt
	if err := cashRepo.SaveReconciliation(ctx, crWO, cashrecon.EventCashReconciliationWrittenOff); err != nil {
		t.Fatalf("SaveReconciliation(WO): %v", err)
	}

	crOv := cashrecon.CashReconciliation{
		ReconciliationId: crOverID, SupplierId: sid, DriverId: "drv-2", ShiftDate: now,
		ExpectedCashMinor: 0, DeclaredCashMinor: 10_000, DifferenceMinor: 10_000,
		Status: cashrecon.ReconciliationStatusPending, CreatedAt: now,
	}
	if err := cashRepo.SaveReconciliation(ctx, crOv, cashrecon.EventCashReconciliationCreated); err != nil {
		t.Fatalf("SaveReconciliation(OV open): %v", err)
	}
	crOv.Status = cashrecon.ReconciliationStatusAccepted
	crOv.ResolvedBy = &actor
	crOv.ResolvedAt = &resolvedAt
	if err := cashRepo.SaveReconciliation(ctx, crOv, cashrecon.EventCashReconciliationAccepted); err != nil {
		t.Fatalf("SaveReconciliation(OV accept): %v", err)
	}

	svc := NewService(client)
	out, err := svc.RunReconciliation(ctx, RunRequest{
		EntityID:   sid,
		PeriodID:   periodID,
		Dimensions: []string{DimensionCashPSP},
	})
	if err != nil {
		t.Fatalf("RunReconciliation CASH_PSP failed: %v", err)
	}
	if out.Run.Status != RunStatusGreen || out.Run.BreakCount != 0 {
		t.Fatalf("expected GREEN CASH_PSP on WRITE_OFF + OVERAGE + LEDGER_ONLY chargeback, got status=%s breaks=%+v check=%+v",
			out.Run.Status, out.Breaks, out.Run.Checks[DimensionCashPSP])
	}
}

func TestRecon_InjectedBreaks_All4Dimensions_PlusOrphanLot_LiveSpanner(t *testing.T) {
	client := openReconSpannerTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sid := "sup-" + uuid.NewString()[:8]
	wid := "wh-" + uuid.NewString()[:8]
	pidCosted := "prod-c-" + uuid.NewString()[:8]
	lotOrphan := "lot-orph-" + uuid.NewString()[:8]
	lotUncosted := "lot-unc-" + uuid.NewString()[:8]
	ordNoAR := "ord-noar-" + uuid.NewString()[:8]
	ordCard := "ord-card-" + uuid.NewString()[:8]
	sessCard := "sess-" + uuid.NewString()[:8]
	crID := "cr-" + uuid.NewString()[:8]
	regimeID := "reg-" + uuid.NewString()[:8]
	now := time.Now().UTC()
	periodID := gl.ComputePeriodID(ctx, sid, "UZ", now)

	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanCancel()
		cleanupEntityReconTables(cleanCtx, client, sid)
		seedOutboxEventCleanup(cleanCtx, client, sid)
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("OrderLineFiscalSnapshots", spanner.Key{ordCard, "line-1"}),
			spanner.Delete("OrderFiscalReceipts", spanner.Key{ordCard, "att-1"}),
			spanner.Delete("PaymentSessions", spanner.Key{sessCard}),
			spanner.Delete("CashReconciliations", spanner.Key{crID}),
			spanner.Delete("Orders", spanner.Key{ordNoAR}),
			spanner.Delete("Orders", spanner.Key{ordCard}),
			spanner.Delete("StockLots", spanner.Key{lotOrphan}),
			spanner.Delete("StockLots", spanner.Key{lotUncosted}),
			spanner.Delete("Warehouses", spanner.Key{wid}),
			spanner.Delete("Products", spanner.Key{pidCosted}),
			spanner.Delete("TaxRegimeVersions", spanner.Key{regimeID}),
		})
	})

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		testTaxRegimeMutation(regimeID, "UZ", "UZS", now),
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId": wid, "SupplierId": sid, "Name": wid, "CoverageRadiusKm": 10.0,
			"IsActive": true, "IsOnShift": true, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		testProductMutation(pidCosted, sid, "UZS", 40_000),
		// Dimension 1 (ORDERS_AR): DELIVERED_ON_CREDIT order without ArInvoices + extra GL AR drift
		testOrderMutation(ordNoAR, sid, "ret-1", "DELIVERED_ON_CREDIT", "NONE", 300_000, "UZS", now),
		// Dimension 2 (STOCK_GL): Orphan AVAILABLE lot (no SupplierInventoryV2 row) 7 * 40k = 280k + uncosted lot
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotOrphan, "SupplierId": sid, "WarehouseId": wid, "ProductId": pidCosted,
			"LocationId": "loc-orph", "QuantityOnHand": int64(7), "QuantityReserved": int64(0),
			"Status": "AVAILABLE", "ReceivedAt": now, "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotUncosted, "SupplierId": sid, "WarehouseId": wid, "ProductId": "missing-sku-nocost",
			"LocationId": "loc-unc", "QuantityOnHand": int64(3), "QuantityReserved": int64(0),
			"Status": "AVAILABLE", "ReceivedAt": now, "CreatedAt": now, "UpdatedAt": now,
		}),
		// Dimension 3 (CASH_PSP): Settled card session 200k vs GL PSP 140k + ACCEPTED CashRecon 90k vs GL CASH:OFFICE 0
		testOrderMutation(ordCard, sid, "ret-1", "COMPLETED", "SUCCESS", 200_000, "UZS", now),
		spanner.InsertMap("PaymentSessions", map[string]any{
			"SessionId": sessCard, "OrderId": ordCard, "SupplierId": sid, "RetailerId": "ret-1",
			"Gateway": "PAYME", "Currency": "UZS", "AmountMinor": int64(200_000), "Mode": "CARD",
			"Status": "SETTLED", "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("CashReconciliations", map[string]any{
			"ReconciliationId": crID, "SupplierId": sid, "DriverId": "drv-2",
			"ShiftDate": "2026-11-16", "ExpectedCashMinor": int64(100_000), "DeclaredCashMinor": int64(80_000),
			"DifferenceMinor": int64(-20_000), "Status": "ACCEPTED", "CreatedAt": now,
		}),
		// Dimension 4 (TAX_FISCAL): Registered fiscal VAT 24k vs GL TAX:VAT:OUTPUT 10k
		spanner.InsertMap("OrderFiscalReceipts", map[string]any{
			"OrderId": ordCard, "AttemptId": "att-1", "SupplierId": sid, "RetailerId": "ret-1",
			"Provider": "PEGASUS", "Status": "REGISTERED", "AmountMinor": int64(200_000), "Currency": "UZS",
			"CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("OrderLineFiscalSnapshots", map[string]any{
			"OrderId": ordCard, "OrderLineId": "line-1", "RegimeId": regimeID, "VatRateBps": int64(1200),
			"NetMinor": int64(200_000), "VatMinor": int64(24_000), "GrossMinor": int64(224_000),
			"SnapshotAt": now, "CreatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed broken domain rows: %v", err)
	}

	// Inject GL drift rows directly in GlBalances
	_ = upsertGlBalance(ctx, client, sid, periodID, gl.AccountAccountsReceivable, "UZS", 50_000, 0)
	_ = upsertGlBalance(ctx, client, sid, periodID, gl.AccountInventory, "UZS", 100_000, 0)
	_ = upsertGlBalance(ctx, client, sid, periodID, gl.AccountCardPSPTransit, "UZS", 140_000, 0)
	_ = upsertGlBalance(ctx, client, sid, periodID, gl.AccountVatOutput, "UZS", 0, 10_000)

	svc := NewService(client)
	out, err := svc.RunReconciliation(ctx, RunRequest{EntityID: sid, PeriodID: periodID, TriggeredBy: "test-breaks"})
	if err != nil {
		t.Fatalf("RunReconciliation failed: %v", err)
	}
	if out.Run.Status != RunStatusBreaksFound || out.Run.Summary.BreakDimensions != 4 {
		t.Fatalf("expected all 4 dimensions in break, got status=%s summary=%+v checks=%+v", out.Run.Status, out.Run.Summary, out.Run.Checks)
	}

	var sawOrderAR, sawARGL, sawOrphanV2, sawUncosted, sawStockGL, sawPSP, sawOffice, sawTax bool
	for _, b := range out.Breaks {
		switch {
		case b.Dimension == DimensionOrdersAR && b.ObjectType == ObjectOrder && b.ObjectID == ordNoAR:
			sawOrderAR = true
		case b.Dimension == DimensionOrdersAR && b.ObjectType == ObjectDimension:
			sawARGL = true
		case b.Dimension == DimensionStockGL && b.ObjectType == ObjectStockLot && b.ObjectID == wid+":"+pidCosted:
			sawOrphanV2 = true
		case b.Dimension == DimensionStockGL && b.ObjectType == ObjectStockLotUncosted && b.ObjectID == lotUncosted:
			sawUncosted = true
		case b.Dimension == DimensionStockGL && b.ObjectType == ObjectDimension:
			sawStockGL = true
		case b.Dimension == DimensionCashPSP && b.ObjectType == ObjectGLAccount && b.ObjectID == "PSP:GATEWAY":
			sawPSP = true
		case b.Dimension == DimensionCashPSP && b.ObjectType == ObjectGLAccount && b.ObjectID == "CASH:OFFICE":
			sawOffice = true
		case b.Dimension == DimensionTaxFiscal && b.ObjectType == ObjectDimension && b.DeltaMinor == -14_000:
			sawTax = true
		}
	}
	if !sawOrderAR || !sawARGL || !sawOrphanV2 || !sawUncosted || !sawStockGL || !sawPSP || !sawOffice || !sawTax {
		t.Fatalf("missing injected breaks: orderAR=%t arGL=%t orphanV2=%t uncosted=%t stockGL=%t psp=%t office=%t tax=%t\nbreaks=%+v",
			sawOrderAR, sawARGL, sawOrphanV2, sawUncosted, sawStockGL, sawPSP, sawOffice, sawTax, out.Breaks)
	}
}

func TestRecon_ControlTowerLink_BreakResolution_And_NightlyWorker_LiveSpanner(t *testing.T) {
	client := openReconSpannerTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sid := "sup-" + uuid.NewString()[:8]
	now := time.Now().UTC()
	periodID := gl.ComputePeriodID(ctx, sid, "UZ", now)

	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanCancel()
		cleanupEntityReconTables(cleanCtx, client, sid)
		seedOutboxEventCleanup(cleanCtx, client, sid)
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("Suppliers", spanner.Key{sid}),
		})
	})

	// Seed active supplier + two breaks (AR:RETAILER drift + TAX:VAT:OUTPUT drift).
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Suppliers", map[string]any{
			"SupplierId": sid, "Name": "Test Recon Supplier", "CountryCode": "UZ", "Currency": "UZS",
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed supplier: %v", err)
	}
	_ = upsertGlBalance(ctx, client, sid, periodID, gl.AccountAccountsReceivable, "UZS", 120_000, 0)
	_ = upsertGlBalance(ctx, client, sid, periodID, gl.AccountVatOutput, "UZS", 0, 15_000)

	svc := NewService(client)
	out, err := svc.RunReconciliation(ctx, RunRequest{EntityID: sid, PeriodID: periodID, TriggeredBy: "ct-test"})
	if err != nil || len(out.Breaks) != 2 {
		t.Fatalf("RunReconciliation = (%d breaks, %v), want 2", len(out.Breaks), err)
	}

	// Verify Control Tower surfaces both breaks in ListOpenScored + ListSupplierIDsWithOpenExceptions.
	ctRepo := controltower.NewSpannerRepository(client)
	ctSvc := controltower.NewService(ctRepo, nil, controltower.Config{Enabled: true})
	openScored, err := ctSvc.ListOpenScored(ctx, sid, 50)
	if err != nil {
		t.Fatalf("ctSvc.ListOpenScored (open): %v", err)
	}
	var ctReconCount int
	for _, item := range openScored {
		if strings.HasPrefix(item.Type, "RECON_") {
			ctReconCount++
		}
	}
	if ctReconCount != 2 {
		t.Fatalf("expected 2 RECON_ exceptions in Control Tower ListOpenScored, got %d (%+v)", ctReconCount, openScored)
	}
	for _, b := range out.Breaks {
		if b.ControlTowerExceptionID == "" {
			t.Fatalf("break %s missing ControlTowerExceptionID", b.BreakID)
		}
		var tStatus string
		row, rErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ExceptionTickets", spanner.Key{b.ControlTowerExceptionID}, []string{"Status"})
		if rErr != nil {
			t.Fatalf("read linked ExceptionTickets %s: %v", b.ControlTowerExceptionID, rErr)
		}
		if err := row.Columns(&tStatus); err != nil || tStatus != BreakStatusOpen {
			t.Fatalf("ExceptionTickets status = %q (%v), want OPEN", tStatus, err)
		}
	}

	// Acknowledge + Resolve break 0, verify ticket + outbox recon.break.resolved emission.
	b0 := out.Breaks[0]
	ack, err := svc.PatchBreak(ctx, sid, b0.BreakID, PatchBreakRequest{Action: ActionAcknowledge, ActorID: "fin-ops"})
	if err != nil || ack.Status != BreakStatusAcknowledged {
		t.Fatalf("acknowledge break = (%+v, %v)", ack, err)
	}
	resolved, err := svc.PatchBreak(ctx, sid, b0.BreakID, PatchBreakRequest{Action: ActionResolve, Note: "cleared adjustment", ActorID: "fin-ops"})
	if err != nil || resolved.Status != BreakStatusResolved {
		t.Fatalf("resolve break = (%+v, %v)", resolved, err)
	}

	// Verify linked ExceptionTicket synced to RESOLVED.
	var tStatus string
	if row, err := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ExceptionTickets", spanner.Key{b0.ControlTowerExceptionID}, []string{"Status"}); err != nil || row.Columns(&tStatus) != nil || tStatus != BreakStatusResolved {
		t.Fatalf("linked ExceptionTickets status after resolve = %q (%v), want RESOLVED", tStatus, err)
	}

	// Waive break 1 with note.
	b1 := out.Breaks[1]
	waived, err := svc.PatchBreak(ctx, sid, b1.BreakID, PatchBreakRequest{Action: ActionWaive, Note: "immaterial test variance", ActorID: "cfo-1"})
	if err != nil || waived.Status != BreakStatusWaived {
		t.Fatalf("waive break = (%+v, %v)", waived, err)
	}

	// Outbox should have recon.run.completed (1), recon.break.opened (2), recon.break.resolved (2) on TopicLedger.
	evMap := fetchOutboxEventTypesForSupplier(t, ctx, client, sid)
	if len(evMap[events.EventReconBreakOpened]) != 2 || len(evMap[events.EventReconBreakResolved]) != 2 {
		t.Fatalf("outbox events = %+v, want 2 opened + 2 resolved", evMap)
	}

	// Control Tower queue after resolving all breaks should contain 0 breaks for this entity.
	scored, err := ctSvc.ListOpenScored(ctx, sid, 50)
	if err != nil {
		t.Fatalf("ctSvc.ListOpenScored: %v", err)
	}
	for _, ex := range scored {
		if strings.HasPrefix(ex.Type, "RECON_") {
			t.Fatalf("unexpected open RECON exception remaining in Control Tower: %+v", ex)
		}
	}

	// Verify Nightly Worker picks up seeded Suppliers row and persists a NIGHTLY run.
	batch, err := svc.RunNightlyBatch(ctx, now)
	if err != nil {
		t.Fatalf("RunNightlyBatch: %v", err)
	}
	if batch.RunsCompleted == 0 {
		t.Fatalf("expected >= 1 nightly run completed, got %+v", batch)
	}
	runs, err := svc.ListRuns(ctx, ListRunsFilter{EntityID: sid})
	if err != nil || len(runs) < 2 {
		t.Fatalf("ListRuns = (%d, %v), want >= 2 runs", len(runs), err)
	}
	var sawNightly bool
	for _, r := range runs {
		if r.Mode == ModeNightly {
			sawNightly = true
		}
	}
	if !sawNightly {
		t.Fatalf("expected at least one NIGHTLY run in ListRuns: %+v", runs)
	}
}

func TestRecon_StockGL_CostLayers_LiveSpanner(t *testing.T) {
	ctx := context.Background()
	client := openReconSpannerTestClient(t)

	suffix := uuid.NewString()[:8]
	sid := "sup-rcl-" + suffix
	whID := "wh-rcl-" + suffix
	prodID := "prod-rcl-" + suffix
	layer1 := "vcl-1-" + suffix
	layer2 := "vcl-2-" + suffix
	now := time.Now().UTC()
	periodID := gl.ComputePeriodID(ctx, sid, "UZ", now)

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cleanupEntityReconTables(cleanCtx, client, sid)
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("InventoryCostLayers", spanner.KeyRange{Start: spanner.Key{sid}, End: spanner.Key{sid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("JournalPostings", spanner.KeyRange{Start: spanner.Key{sid}, End: spanner.Key{sid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("JournalEntries", spanner.KeyRange{Start: spanner.Key{sid}, End: spanner.Key{sid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlBalances", spanner.KeyRange{Start: spanner.Key{sid}, End: spanner.Key{sid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("OutboxEvents", spanner.KeyRange{Start: spanner.Key{sid}, End: spanner.Key{sid}, Kind: spanner.ClosedClosed}),
		})
	})

	// Seed GL INVENTORY = 30,000 + matching InventoryCostLayers (10 * 3,000 = 30,000).
	glSvc := gl.NewService(client)
	if _, err := glSvc.PostEntry(ctx, gl.JournalEntryInput{
		EntityID:       sid,
		SourceType:     "INVENTORY_RECEIPT",
		SourceID:       "po-" + suffix,
		Memo:           "recon cost layers test receipt",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: "gl-rcl-" + suffix,
		PostedBy:       "test",
		OccurredAt:     now,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountInventory, Direction: gl.DirectionDebit, AmountMinor: 30_000, Currency: "UZS"},
			{AccountCode: gl.AccountGRNI, Direction: gl.DirectionCredit, AmountMinor: 30_000, Currency: "UZS"},
		},
	}); err != nil {
		t.Fatalf("glSvc.PostEntry: %v", err)
	}
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("InventoryCostLayers", map[string]any{
			"EntityId": sid, "WarehouseId": whID, "ProductId": prodID, "LayerId": layer1,
			"QtyRemaining": int64(10), "OriginalQty": int64(10), "UnitCostMinor": int64(3_000),
			"Currency": "UZS", "ReceivedAt": now, "CreatedAt": now, "UpdatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed InventoryCostLayers 1: %v", err)
	}

	svc := NewServiceWithRepository(NewSpannerRepository(client))
	outGreen, err := svc.RunReconciliation(ctx, RunRequest{
		EntityID:   sid,
		PeriodID:   periodID,
		Dimensions: []string{DimensionStockGL},
	})
	if err != nil {
		t.Fatalf("RunReconciliation GREEN: %v", err)
	}
	chk := outGreen.Run.Checks[DimensionStockGL]
	if chk.Status != DimStatusGreen || chk.DeltaMinor != 0 || chk.ExpectedMinor != 30_000 || chk.ActualMinor != 30_000 {
		t.Fatalf("expected STOCK_GL GREEN 30000/30000 delta=0, got %+v", chk)
	}

	// Inject unposted cost layer (5 * 2,000 = +10,000) -> STOCK_GL must detect -10,000 break.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("InventoryCostLayers", map[string]any{
			"EntityId": sid, "WarehouseId": whID, "ProductId": prodID, "LayerId": layer2,
			"QtyRemaining": int64(5), "OriginalQty": int64(5), "UnitCostMinor": int64(2_000),
			"Currency": "UZS", "ReceivedAt": now, "CreatedAt": now, "UpdatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed InventoryCostLayers 2: %v", err)
	}
	outBreak, err := svc.RunReconciliation(ctx, RunRequest{
		EntityID:   sid,
		PeriodID:   periodID,
		Dimensions: []string{DimensionStockGL},
	})
	if err != nil {
		t.Fatalf("RunReconciliation BREAK: %v", err)
	}
	if outBreak.Run.Checks[DimensionStockGL].Status != DimStatusBreak || outBreak.Run.Checks[DimensionStockGL].DeltaMinor != -10_000 {
		t.Fatalf("expected STOCK_GL BREAK delta=-10000, got %+v", outBreak.Run.Checks[DimensionStockGL])
	}
}
