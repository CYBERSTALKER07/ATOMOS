package migration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

func seedM6CatalogAndCustomerEnv(t *testing.T, sup string, now time.Time) (*m5TestRepo, *MemoryApplyStore, Project, string, string) {
	t.Helper()
	ctx := context.Background()
	repo, store, proj := newM5TestEnv(t, sup, now)

	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-wh-m6", PackWarehouses, []map[string]any{
		{"external_warehouse_id": "WH-MAIN", "name": "Tashkent Hub", "address": "Sergeli 1", "is_default": "true"},
	}, map[string]string{
		"external_warehouse_id": "external_warehouse_id",
		"name":                  "name",
		"address":               "address",
		"is_default":            "is_default",
	}, ApplyModeMerge, false, now)
	if _, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-wh-m6", Now: now}); err != nil {
		t.Fatalf("apply wh: %v", err)
	}

	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-cat-m6", PackProductsPrices, []map[string]any{
		{"sku": "SKU-TEA-1", "name": "Black Tea", "price": "20000", "currency": "UZS", "unit": "PCS", "barcode": "478001", "shelf_life_days": "30"},
	}, map[string]string{
		"sku":             "sku",
		"name":            "name",
		"price":           "price",
		"currency":        "currency",
		"unit":            "unit",
		"barcode":         "barcode",
		"shelf_life_days": "shelf_life_days",
	}, ApplyModeMerge, false, now)
	if _, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-cat-m6", Now: now}); err != nil {
		t.Fatalf("apply catalog: %v", err)
	}

	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-cust-m6", PackCustomers, []map[string]any{
		{"external_customer_id": "CUST-A", "customer_name": "Baraka Market", "phone": "901112233", "inn": "301234567", "address": "Chilonzor"},
	}, map[string]string{
		"external_customer_id": "external_customer_id",
		"customer_name":        "customer_name",
		"phone":                "phone",
		"inn":                  "inn",
		"address":              "address",
	}, ApplyModeMerge, false, now)
	if _, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-cust-m6", Now: now}); err != nil {
		t.Fatalf("apply customers: %v", err)
	}

	prodID := store.IdentityMap[identityKey(sup, PackProductsPrices, "SKU-TEA-1")].InternalID
	retID := store.IdentityMap[identityKey(sup, PackCustomers, "CUST-A")].InternalID
	return repo, store, proj, prodID, retID
}

func TestApplyM6_HistoricalOrders_MultiLine_AndRollback(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	sup := "sup-m6-hist"
	repo, store, proj, _, _ := seedM6CatalogAndCustomerEnv(t, sup, now)

	histMap := map[string]string{
		"external_order_no": "external_order_no",
		"order_date":        "order_date",
		"status":            "status",
		"customer_inn":      "customer_inn",
		"sku":               "sku",
		"quantity":          "quantity",
		"total_amount":      "total_amount",
		"currency":          "currency",
	}

	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-hist-m6", PackHistoricalOrders, []map[string]any{
		{
			"external_order_no": "1C-ORD-100",
			"order_date":        "2026-10-05 14:30:00",
			"status":            "DELIVERED",
			"customer_inn":      "301234567",
			"sku":               "SKU-TEA-1",
			"quantity":          "2",
			"total_amount":      "40000",
			"currency":          "UZS",
		},
		{
			"external_order_no": "1C-ORD-100",
			"order_date":        "2026-10-05 14:30:00",
			"status":            "DELIVERED",
			"customer_inn":      "301234567",
			"sku":               "SKU-EXTRA",
			"quantity":          "1",
			"total_amount":      "15000",
			"currency":          "UZS",
		},
	}, histMap, ApplyModeMerge, false, now)

	res, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-hist-m6", Now: now})
	if err != nil {
		t.Fatalf("apply historical orders: %v", err)
	}
	if res.Created != 1 || res.Updated != 1 {
		t.Fatalf("expected 1 created + 1 updated multi-line historical order, got %+v", res)
	}
	if len(store.Orders) != 1 || len(store.Inventory) != 0 {
		t.Fatalf("historical orders must create 1 order and 0 inventory mutations; orders=%d inv=%d", len(store.Orders), len(store.Inventory))
	}
	var imported order.Order
	for _, o := range store.Orders {
		imported = o
	}
	if imported.Source != order.OrderSourceMigration || imported.FiscalStatus != order.FiscalStatusHistoricalSkipped || len(imported.LineItems) != 2 {
		t.Fatalf("unexpected imported historical order: %+v", imported)
	}

	// Non-terminal status fails closed during apply even if session approved.
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-hist-bad", PackHistoricalOrders, []map[string]any{
		{
			"external_order_no": "1C-ORD-BAD",
			"order_date":        "2026-10-06",
			"status":            "IN_TRANSIT",
			"customer_inn":      "301234567",
			"total_amount":      "10000",
		},
	}, map[string]string{
		"external_order_no": "external_order_no",
		"order_date":        "order_date",
		"status":            "status",
		"customer_inn":      "customer_inn",
		"total_amount":      "total_amount",
	}, ApplyModeMerge, false, now)
	if _, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-hist-bad", Now: now}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected non-terminal historical status failure, got %v", err)
	}

	// 72h Rollback supersedes migrated historical orders.
	rbRes, err := RollbackSession(ctx, repo, sup, "sess-hist-m6", now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("rollback historical session: %v", err)
	}
	if rbRes.SupersededOrders != 1 || !store.Orders[imported.OrderID].SupersededByRollback || store.Orders[imported.OrderID].Status != order.StatusCancelled {
		t.Fatalf("expected SupersededByRollback=true & CANCELLED, got rb=%+v ord=%+v", rbRes, store.Orders[imported.OrderID])
	}
}

func TestApplyM6_NasiyaAR_DoubleEntry_AtomicAbort_AndRollback(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	sup := "sup-m6-nasiya"
	repo, store, proj, _, retID := seedM6CatalogAndCustomerEnv(t, sup, now)

	nasiyaCols := map[string]string{
		"external_ar_id":  "external_ar_id",
		"customer_inn":    "customer_inn",
		"balance_due":     "balance_due",
		"original_amount": "original_amount",
		"currency":        "currency",
		"issued_date":     "issued_date",
		"due_date":        "due_date",
	}

	// 1. Fail-closed journal imbalance must abort the chunk atomically leaving zero AR / credit / journal mutations.
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-ar-abort", PackNasiyaAR, []map[string]any{
		{
			"external_ar_id":  "AR-BAD-01",
			"customer_inn":    "301234567",
			"balance_due":     "300000",
			"original_amount": "300000",
			"currency":        "UZS",
			"issued_date":     "2026-10-01",
			"due_date":        "2026-10-20",
		},
	}, nasiyaCols, ApplyModeMerge, false, now)

	_, err := ApplySession(ctx, repo, store, ApplyOptions{
		SupplierID: sup,
		SessionID:  "sess-ar-abort",
		Now:        now,
		JournalOverrideFn: func(in ar.MigrationJournalInput) (*payment.JournalEntry, error) {
			imbalanced := &payment.JournalEntry{
				EntryID:       in.EntryID,
				TransactionID: "bad:" + in.EntryID,
				OrderID:       in.OrderID,
				SupplierID:    in.SupplierID,
				RetailerID:    in.RetailerID,
				Currency:      in.Currency,
				OccurredAt:    now,
				Postings: []payment.Posting{
					{AccountCode: "AR:" + in.SupplierID, AccountType: "ASSET", Direction: payment.PostingDebit, AmountMinor: in.AmountMinor, Currency: in.Currency},
					{AccountCode: payment.AccountOpeningBalanceEquity, AccountType: "EQUITY", Direction: payment.PostingCredit, AmountMinor: in.AmountMinor - 100, Currency: in.Currency},
				},
			}
			return imbalanced, imbalanced.Validate()
		},
	})
	if err == nil {
		t.Fatal("expected imbalanced journal entry to fail chunk apply")
	}
	if len(store.CreditProfiles) != 0 || len(store.JournalEntries) != 0 {
		t.Fatalf("aborted chunk leaked AR state: profiles=%d journals=%d", len(store.CreditProfiles), len(store.JournalEntries))
	}

	// 2. Valid balanced nasiya apply using production payment.BuildMigrationOpeningBalanceJournalEntry.
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-ar-ok", PackNasiyaAR, []map[string]any{
		{
			"external_ar_id":  "AR-OK-01",
			"customer_inn":    "301234567",
			"balance_due":     "1250000",
			"original_amount": "1500000",
			"currency":        "UZS",
			"issued_date":     "2026-10-01",
			"due_date":        "2026-10-20",
		},
	}, nasiyaCols, ApplyModeMerge, false, now)

	res, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-ar-ok", Now: now})
	if err != nil {
		t.Fatalf("apply nasiya AR: %v", err)
	}
	if res.Created != 1 || len(res.JournalEntries) != 1 {
		t.Fatalf("expected 1 created + 1 balanced journal entry, got %+v", res)
	}
	je := res.JournalEntries[0]
	if err := je.Validate(); err != nil || len(je.Postings) != 2 || je.Postings[1].AccountCode != payment.AccountOpeningBalanceEquity {
		t.Fatalf("unexpected double-entry journal %+v err=%v", je, err)
	}
	cp := store.CreditProfiles[sup+"|"+retID]
	if cp.CurrentBalanceMinor != 125000000 {
		t.Fatalf("credit profile CurrentBalanceMinor got %d want 125000000", cp.CurrentBalanceMinor)
	}

	invs, err := store.ARRepo.ListMigrationOpeningBalances(ctx, retID)
	if err != nil || len(invs) != 1 {
		t.Fatalf("expected 1 migrated AR invoice for retailer, got %+v err=%v", invs, err)
	}
	if invs[0].Source != ar.InvoiceSourceMigration || invs[0].AcknowledgedAt != nil || !invs[0].IsMigrationDunningFrozen() {
		t.Fatalf("expected unacknowledged MIGRATION AR invoice with frozen dunning, got %+v", invs[0])
	}

	// Paired synthetic order carries OrderSource=MIGRATION & FiscalStatus=HISTORICAL_SKIPPED.
	synOrd, ok := store.Orders[invs[0].OrderID]
	if !ok || synOrd.Source != order.OrderSourceMigration || synOrd.FiscalStatus != order.FiscalStatusHistoricalSkipped {
		t.Fatalf("missing or invalid synthetic order: ok=%v ord=%+v", ok, synOrd)
	}

	// 3. 72h Rollback voids migrated AR invoice, emits balanced reversal journal entry, and zeroes credit balance.
	rbRes, err := RollbackSession(ctx, repo, sup, "sess-ar-ok", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("rollback nasiya session: %v", err)
	}
	if rbRes.ReversedInvoices != 1 || rbRes.ReversedAmountMinor != 125000000 || len(rbRes.ReversalEntryIDs) != 1 {
		t.Fatalf("unexpected nasiya rollback result: %+v", rbRes)
	}
	revJE := store.JournalEntries[rbRes.ReversalEntryIDs[0]]
	if revJE == nil || revJE.Validate() != nil || !strings.HasPrefix(revJE.EntryID, "jentry_migration_reversal_") {
		t.Fatalf("expected valid reversal journal entry, got %+v", revJE)
	}
	voidInv, err := store.ARRepo.GetByID(ctx, invs[0].InvoiceID)
	if err != nil || voidInv.Status != ar.StatusVoid || voidInv.BalanceMinor != 0 {
		t.Fatalf("expected VOID invoice with zero balance after rollback, got %+v err=%v", voidInv, err)
	}
	if store.CreditProfiles[sup+"|"+retID].CurrentBalanceMinor != 0 {
		t.Fatalf("credit balance not restored to 0 on rollback: %+v", store.CreditProfiles[sup+"|"+retID])
	}
}

func TestApplyM6_OpenOrders_HoldAndSelectiveReleaseWithFEFO(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	sup := "sup-m6-open"
	prevLots := stocklots.LotsEnabled()
	stocklots.SetLotsEnabled(true)
	t.Cleanup(func() { stocklots.SetLotsEnabled(prevLots) })

	repo, store, proj, prodID, _ := seedM6CatalogAndCustomerEnv(t, sup, now)

	// Apply two expiry-dated lots (LOT-EARLY expires first: FEFO must reserve from it first).
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-lots-m6", PackLots, []map[string]any{
		{
			"sku": "SKU-TEA-1", "warehouse_id": "WH-MAIN", "lot_number": "LOT-LATE",
			"quantity_on_hand": "6", "expiry_date": "2027-06-01",
		},
		{
			"sku": "SKU-TEA-1", "warehouse_id": "WH-MAIN", "lot_number": "LOT-EARLY",
			"quantity_on_hand": "4", "expiry_date": "2027-01-15",
		},
	}, map[string]string{
		"sku": "sku", "warehouse_id": "warehouse_id", "lot_number": "lot_number",
		"quantity_on_hand": "quantity_on_hand", "expiry_date": "expiry_date",
	}, ApplyModeMerge, false, now)
	if _, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-lots-m6", Now: now}); err != nil {
		t.Fatalf("apply lots: %v", err)
	}

	// Apply two open orders: ORD-TOO-BIG (needs 20 > 10 available) and ORD-FIT (needs 6 <= 10 available).
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-open-m6", PackOpenOrders, []map[string]any{
		{
			"external_order_no": "ORD-FIT",
			"customer_inn":      "301234567",
			"warehouse_id":      "WH-MAIN",
			"sku":               "SKU-TEA-1",
			"quantity":          "6",
			"status":            "SCHEDULED",
			"delivery_date":     "2026-11-20",
			"total_amount":      "120000",
		},
		{
			"external_order_no": "ORD-TOO-BIG",
			"customer_inn":      "301234567",
			"warehouse_id":      "WH-MAIN",
			"sku":               "SKU-TEA-1",
			"quantity":          "20",
			"delivery_date":     "2026-11-21",
			"total_amount":      "400000",
		},
	}, map[string]string{
		"external_order_no": "external_order_no",
		"customer_inn":      "customer_inn",
		"warehouse_id":      "warehouse_id",
		"sku":               "sku",
		"quantity":          "quantity",
		"status":            "status",
		"delivery_date":     "delivery_date",
		"total_amount":      "total_amount",
	}, ApplyModeMerge, false, now)

	if _, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-open-m6", Now: now}); err != nil {
		t.Fatalf("apply open orders: %v", err)
	}

	fitID := store.IdentityMap[identityKey(sup, PackOpenOrders, "ORD-FIT")].InternalID
	bigID := store.IdentityMap[identityKey(sup, PackOpenOrders, "ORD-TOO-BIG")].InternalID
	if store.Orders[fitID].Status != order.StatusMigrationOpen || store.Orders[bigID].Status != order.StatusMigrationOpen {
		t.Fatalf("open orders must remain held in MIGRATION_OPEN on apply")
	}

	// Verify ZERO inventory or lot reservation occurred during apply.
	whID := store.IdentityMap[identityKey(sup, PackWarehouses, "WH-MAIN")].InternalID
	if stk := store.Inventory[inventoryKey(sup, whID, prodID)]; stk.QuantityOnHand != 10 || stk.QuantityReserved != 0 {
		t.Fatalf("expected QOH=10 QR=0 before release, got %+v", stk)
	}

	// Try releasing both: ORD-FIT succeeds (PENDING override), ORD-TOO-BIG fails with insufficient stock.
	relRes, err := store.ReleaseOpenOrders(ctx, sup, []order.ReleaseOpenOrderItem{
		{OrderID: fitID, TargetStatus: order.StatusScheduled},
		{OrderID: bigID, TargetStatus: order.StatusPending},
	}, "admin-1", now)
	if err != nil {
		t.Fatalf("ReleaseOpenOrders error: %v", err)
	}
	if len(relRes.ReleasedOrderIDs) != 1 || relRes.ReleasedOrderIDs[0] != fitID {
		t.Fatalf("expected only ORD-FIT released, got %+v", relRes)
	}
	if len(relRes.SkippedOrderIDs) != 1 || relRes.SkippedOrderIDs[0] != bigID || relRes.Errors[bigID] == "" {
		t.Fatalf("expected ORD-TOO-BIG skipped with insufficient stock error, got %+v", relRes)
	}

	// Verify released order transitioned to MANUAL / SCHEDULED and reserved 6 units (FEFO: 4 from LOT-EARLY + 2 from LOT-LATE).
	releasedOrd := store.Orders[fitID]
	if releasedOrd.Status != order.StatusScheduled || releasedOrd.Source != order.OrderSourceManual {
		t.Fatalf("unexpected released order state: %+v", releasedOrd)
	}
	if store.Orders[bigID].Status != order.StatusMigrationOpen {
		t.Fatalf("failed order must remain in MIGRATION_OPEN, got %s", store.Orders[bigID].Status)
	}
	if stk := store.Inventory[inventoryKey(sup, whID, prodID)]; stk.QuantityReserved != 6 {
		t.Fatalf("expected QuantityReserved=6 after FEFO release, got %+v", stk)
	}
	for _, lot := range store.Lots {
		switch lot.LotCode {
		case "LOT-EARLY":
			if lot.QuantityReserved != 4 {
				t.Fatalf("FEFO expected LOT-EARLY fully reserved (4), got %d", lot.QuantityReserved)
			}
		case "LOT-LATE":
			if lot.QuantityReserved != 2 {
				t.Fatalf("FEFO expected LOT-LATE partial reserved (2), got %d", lot.QuantityReserved)
			}
		}
	}
}

func TestCutoverM6_PreflightChecklistAndRollbackLock(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	sup := "sup-m6-cutover"
	repo, store, proj, _, _ := seedM6CatalogAndCustomerEnv(t, sup, now)

	// Stage one unapplied approved session -> cutover must FAIL checklist.
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-unapplied", PackWarehouses, []map[string]any{
		{"external_warehouse_id": "WH-2", "name": "Branch 2"},
	}, map[string]string{
		"external_warehouse_id": "external_warehouse_id",
		"name":                  "name",
	}, ApplyModeMerge, false, now)

	resFail, err := ExecuteProjectCutover(ctx, repo, CutoverOptions{SupplierID: sup, ProjectID: proj.ProjectID, Now: now})
	if !errors.Is(err, ErrCutoverChecklistFailed) || len(resFail.Checklist) == 0 {
		t.Fatalf("expected ErrCutoverChecklistFailed when unapplied session exists, got err=%v checklist=%+v", err, resFail.Checklist)
	}

	// Apply remaining session.
	if _, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-unapplied", Now: now}); err != nil {
		t.Fatalf("apply remaining session: %v", err)
	}

	// Soliq failure blocks cutover.
	if _, err := ExecuteProjectCutover(ctx, repo, CutoverOptions{
		SupplierID:      sup,
		ProjectID:       proj.ProjectID,
		Now:             now,
		SoliqHealthPing: func(context.Context, string) error { return errors.New("soliq timeout") },
	}); !errors.Is(err, ErrCutoverChecklistFailed) {
		t.Fatalf("expected cutover failure on unhealthy soliq ping, got %v", err)
	}

	// Expired dry-run (>24h) blocks cutover.
	if _, err := ExecuteProjectCutover(ctx, repo, CutoverOptions{
		SupplierID: sup,
		ProjectID:  proj.ProjectID,
		Now:        now.Add(25 * time.Hour),
	}); !errors.Is(err, ErrCutoverChecklistFailed) {
		t.Fatalf("expected cutover failure on expired >24h dry-run report, got %v", err)
	}

	// Healthy cutover completes cleanly and locks rollback forever.
	resOK, err := ExecuteProjectCutover(ctx, repo, CutoverOptions{
		SupplierID:           sup,
		ProjectID:            proj.ProjectID,
		Now:                  now,
		SoliqHealthPing:      func(context.Context, string) error { return nil },
		ReleaseLotsGateCheck: func(context.Context, string) error { return nil },
	})
	if err != nil {
		t.Fatalf("ExecuteProjectCutover: %v", err)
	}
	if resOK.Project.Status != ProjectStatusCutoverComplete {
		t.Fatalf("expected project CUTOVER_COMPLETE, got %s", resOK.Project.Status)
	}
	for _, s := range resOK.Sessions {
		if s.Status != SessionStatusCutoverComplete {
			t.Fatalf("expected session %s CUTOVER_COMPLETE, got %s", s.SessionID, s.Status)
		}
	}
	if _, err := RollbackSession(ctx, repo, sup, "sess-unapplied", now.Add(time.Minute)); !errors.Is(err, ErrRollbackNotAllowed) {
		t.Fatalf("expected ErrRollbackNotAllowed after CUTOVER_COMPLETE, got %v", err)
	}
}
