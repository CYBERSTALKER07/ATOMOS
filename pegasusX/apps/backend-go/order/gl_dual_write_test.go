package order

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"github.com/pegasusx/pegasusx/apps/backend-go/tax"
	"google.golang.org/api/iterator"
)

func cleanupOrderGLTestState(t *testing.T, client *spanner.Client, supplierID string, orderIDs []string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, q := range []struct {
			sql   string
			table string
			cols  []string
			param string
			val   any
		}{
			{"SELECT EntityId, EntryId, Line FROM JournalPostings WHERE EntityId = @v", "JournalPostings", []string{"EntityId", "EntryId", "Line"}, "v", supplierID},
			{"SELECT EntityId, EntryId FROM JournalEntries WHERE EntityId = @v", "JournalEntries", []string{"EntityId", "EntryId"}, "v", supplierID},
			{"SELECT EntityId, PeriodId, AccountCode, PartyKey, Shard FROM GlBalances WHERE EntityId = @v", "GlBalances", []string{"EntityId", "PeriodId", "AccountCode", "PartyKey", "Shard"}, "v", supplierID},
			{"SELECT FeeScheduleId FROM BillingFeeSchedules WHERE SupplierId = @v", "BillingFeeSchedules", []string{"FeeScheduleId"}, "v", supplierID},
			{"SELECT EntryId FROM ArLedgerEntries WHERE SupplierId = @v", "ArLedgerEntries", []string{"EntryId"}, "v", supplierID},
			{"SELECT InvoiceId FROM ArInvoices WHERE SupplierId = @v", "ArInvoices", []string{"InvoiceId"}, "v", supplierID},
			{"SELECT RetailerId, SupplierId FROM RetailerCreditProfiles WHERE SupplierId = @v", "RetailerCreditProfiles", []string{"RetailerId", "SupplierId"}, "v", supplierID},
			{"SELECT EventId FROM OutboxEvents WHERE SupplierId = @v", "OutboxEvents", []string{"EventId"}, "v", supplierID},
		} {
			var muts []*spanner.Mutation
			it := client.Single().Query(ctx, spanner.Statement{SQL: q.sql, Params: map[string]any{q.param: q.val}})
			for {
				row, err := it.Next()
				if err != nil {
					break
				}
				vals := make([]*spanner.GenericColumnValue, len(q.cols))
				ptrs := make([]any, len(q.cols))
				for i := range q.cols {
					var sv spanner.GenericColumnValue
					vals[i] = &sv
					ptrs[i] = &sv
				}
				if row.Columns(ptrs...) == nil {
					k := make(spanner.Key, len(q.cols))
					for i, gv := range vals {
						if gv.Type.Code == 2 {
							var iv int64
							_ = gv.Decode(&iv)
							k[i] = iv
						} else {
							var s string
							_ = gv.Decode(&s)
							k[i] = s
						}
					}
					muts = append(muts, spanner.Delete(q.table, k))
				}
			}
			it.Stop()
			if len(muts) > 0 {
				_, _ = client.Apply(ctx, muts)
			}
		}
		if len(orderIDs) > 0 {
			for _, tbl := range []string{
				"OrderPaymentLegs", "OrderFiscalReceipts", "OrderLineFiscalSnapshots",
				"OrderStatusTransitions", "OrderShopClosedLog", "OrderCreditReservations", "InvoiceSettlementSlices",
			} {
				it := client.Single().Query(ctx, spanner.Statement{
					SQL:    fmt.Sprintf("SELECT OrderId FROM %s WHERE OrderId IN UNNEST(@oids)", tbl),
					Params: map[string]any{"oids": orderIDs},
				})
				var muts []*spanner.Mutation
				for {
					r, err := it.Next()
					if err != nil {
						break
					}
					var oid string
					if tbl == "OrderLineFiscalSnapshots" || tbl == "OrderPaymentLegs" || tbl == "OrderFiscalReceipts" || tbl == "OrderStatusTransitions" || tbl == "OrderShopClosedLog" {
						// Delete parent Orders or query PKs directly below
						_ = r
						continue
					}
					if r.Column(0, &oid) == nil {
						muts = append(muts, spanner.Delete(tbl, spanner.Key{oid}))
					}
				}
				it.Stop()
				if len(muts) > 0 {
					_, _ = client.Apply(ctx, muts)
				}
			}
			// Clean standalone tables keyed by OrderId / composite PKs
			for _, tbl := range []string{"OrderDeliveryProofs", "PaymentLedgerEntries"} {
				pk := "ProofId"
				if tbl == "PaymentLedgerEntries" {
					pk = "LedgerEntryId"
				}
				it := client.Single().Query(ctx, spanner.Statement{
					SQL:    fmt.Sprintf("SELECT %s FROM %s WHERE OrderId IN UNNEST(@oids)", pk, tbl),
					Params: map[string]any{"oids": orderIDs},
				})
				var muts []*spanner.Mutation
				for {
					r, err := it.Next()
					if err != nil {
						break
					}
					var id string
					if r.Column(0, &id) == nil {
						muts = append(muts, spanner.Delete(tbl, spanner.Key{id}))
					}
				}
				it.Stop()
				if len(muts) > 0 {
					_, _ = client.Apply(ctx, muts)
				}
			}
			// Clean child rows with composite PKs and Orders
			for _, spec := range []struct{ tbl, col2 string }{
				{"OrderPaymentLegs", "LegId"},
				{"OrderSettlementExceptions", "ExceptionId"},
				{"OrderFiscalReceipts", "AttemptId"},
				{"OrderLineFiscalSnapshots", "OrderLineId"},
				{"OrderStatusTransitions", "TransitionId"},
				{"OrderShopClosedLog", "EventId"},
				{"InvoiceSettlementSlices", "SliceId"},
			} {
				it := client.Single().Query(ctx, spanner.Statement{
					SQL:    fmt.Sprintf("SELECT OrderId, %s FROM %s WHERE OrderId IN UNNEST(@oids)", spec.col2, spec.tbl),
					Params: map[string]any{"oids": orderIDs},
				})
				var muts []*spanner.Mutation
				for {
					r, err := it.Next()
					if err != nil {
						break
					}
					var a, b string
					if r.Columns(&a, &b) == nil {
						muts = append(muts, spanner.Delete(spec.tbl, spanner.Key{a, b}))
					}
				}
				it.Stop()
				if len(muts) > 0 {
					_, _ = client.Apply(ctx, muts)
				}
			}
			var ordMuts []*spanner.Mutation
			for _, id := range orderIDs {
				ordMuts = append(ordMuts, spanner.Delete("Orders", spanner.Key{id}))
			}
			_, _ = client.Apply(ctx, ordMuts)
		}
	})
}

func TestOrder_DeliveryVATPerOrderFeeCreditLeaveAndSplitTender_GLDualWrite_LiveSpanner(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	suffix := uuid.NewString()[:8]
	supID := "sup_ord_gl_" + suffix
	retID := "ret_ord_gl_" + suffix
	drvID := "drv_ord_gl_" + suffix
	ordCash := "ord_gl_cash_" + suffix
	ordCredit := "ord_gl_cred_" + suffix
	ordSplit := "ord_gl_split_" + suffix
	feeID := "fee_ord_gl_" + suffix
	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)

	cleanupOrderGLTestState(t, client, supID, []string{ordCash, ordCredit, ordSplit})

	// Seed BillingFeeSchedules for PerOrderMinor = 2500.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("BillingFeeSchedules", map[string]any{
			"FeeScheduleId":            feeID,
			"SupplierId":               supID,
			"Tier":                     "STANDARD",
			"PerOrderMinor":            int64(2500),
			"GmvBps":                   int64(0),
			"MonthlySubscriptionMinor": int64(0),
			"Currency":                 "UZS",
			"EffectiveFrom":            now.Add(-time.Hour),
			"CreatedAt":                now,
		}),
	}); err != nil {
		t.Fatalf("seed BillingFeeSchedules: %v", err)
	}

	// Ensure active UZ 12% VAT regime exists (do not disturb if already present).
	taxRepo := tax.NewSpannerRepository(client)
	var regimeCreated string
	if _, found, err := taxRepo.GetActiveRegime(ctx, nil, "UZ", time.Now().UTC()); err != nil || !found {
		regimeCreated = "reg_ord_gl_" + suffix
		if _, err := client.Apply(ctx, []*spanner.Mutation{
			spanner.InsertMap("TaxRegimeVersions", map[string]any{
				"Id":            regimeCreated,
				"CountryCode":   "UZ",
				"EffectiveFrom": now.Add(-24 * time.Hour),
				"Currency":      "UZS",
				"VatRatesBps":   []int64{1200},
				"CreatedAt":     now,
				"CreatedBy":     "test",
				"UpdatedAt":     now,
			}),
		}); err != nil {
			t.Fatalf("seed TaxRegimeVersions: %v", err)
		}
		t.Cleanup(func() {
			_, _ = client.Apply(context.Background(), []*spanner.Mutation{
				spanner.Delete("TaxRegimeVersions", spanner.Key{regimeCreated}),
			})
		})
	}

	// Build real Spanner-backed order service with AR + Credit + Tax + FeeScheduleResolver.
	orderRepo := NewSpannerRepository(client)
	creditSvc := credit.NewService(credit.NewSpannerRepository(client))
	creditSvc.SetNow(func() time.Time { return now })
	arSvc := ar.NewService(ar.NewSpannerRepository(client))
	arSvc.SetNow(func() time.Time { return now })
	feeResolver := billing.NewFeeScheduleResolver(client)

	svc := NewService(ServiceConfig{
		Repo:               orderRepo,
		Credit:             creditSvc,
		CommissionResolver: feeResolver,
		SpannerClient:      client,
		Now:                func() time.Time { return now },
		Log:                slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	svc.SetARService(arSvc)
	svc.SetTaxService(tax.NewService(taxRepo, nil, slog.New(slog.NewTextHandler(io.Discard, nil))))
	svc.SetPerOrderFeeResolver(func(ctx context.Context, supplierID string) (int64, string, error) {
		sched, err := feeResolver.Resolve(ctx, supplierID)
		if err != nil {
			return 0, "", err
		}
		return sched.PerOrderMinor, sched.Currency, nil
	})

	lineItems, _ := json.Marshal([]LineItem{{SKU: "SKU-1", Quantity: 1, UnitPrice: 100000}})
	lineSplit, _ := json.Marshal([]LineItem{{SKU: "SKU-2", Quantity: 1, UnitPrice: 100000}})

	// Seed orders:
	// 1. ordCash in FISCAL_FAILED ready for ForceCompleteOrder (cash 112k w/ 12% VAT on 100k line)
	// 2. ordCredit in SHOP_CLOSED_PENDING past grace ready for ResolveShopClosedTimeouts (credit leave 80k)
	// 3. ordSplit in FISCAL_FAILED ready for RecordSplitTenderTxn + ForceCompleteOrder
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordCash, "RetailerId": retID, "SupplierId": supID, "DriverId": drvID, "WarehouseId": "wh_gl_" + suffix,
			"Status": string(StatusFiscalFailed), "OrderSource": string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed), "LineItemsJson": lineItems,
			"TotalMinor": int64(112000), "Currency": "UZS", "FiscalStatus": FiscalStatusFailed,
			"Version": int64(1), "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordCredit, "RetailerId": retID, "SupplierId": supID, "DriverId": drvID, "WarehouseId": "wh_gl_" + suffix,
			"Status": string(StatusShopClosedPending), "OrderSource": string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed), "LineItemsJson": []byte("[]"),
			"TotalMinor": int64(80000), "Currency": "UZS", "ShopClosedGraceEndsAt": now.Add(-5 * time.Minute),
			"Version": int64(1), "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordSplit, "RetailerId": retID, "SupplierId": supID, "DriverId": drvID, "WarehouseId": "wh_gl_" + suffix,
			"Status": string(StatusFiscalFailed), "OrderSource": string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed), "LineItemsJson": lineSplit,
			"TotalMinor": int64(100000), "Currency": "UZS", "FiscalStatus": FiscalStatusFailed,
			"Version": int64(1), "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": retID, "SupplierId": supID, "CreditLimitMinor": int64(500000),
			"CurrentBalanceMinor": int64(0), "AvailableCreditMinor": int64(500000),
			"Status": string(credit.StatusActive), "RiskScore": int64(800),
			"DelinquencyCount": int64(0), "Version": int64(1), "CreatedAt": now, "UpdatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed orders: %v", err)
	}

	adminClaims := auth.Claims{Subject: "adm_gl", Role: auth.RoleAdmin, SupplierID: supID}

	// (A) Complete cash order -> ORDER_DELIVERED (100k REV:SALES + 12k VAT / 112k CASH:DRIVER) + FEE_PER_ORDER (2.5k).
	if _, err := svc.ForceCompleteOrder(ctx, adminClaims, ordCash, ForceReasonOFDDown); err != nil {
		t.Fatalf("ForceCompleteOrder(cash): %v", err)
	}

	// (B) Worker shop-closed credit leave -> AR invoice open + ORDER_DELIVERED (CREDIT 80k) inside same txn.
	if err := svc.resolveOneShopClosedTimeout(ctx, ordCredit); err != nil {
		t.Fatalf("resolveOneShopClosedTimeout(credit): %v", err)
	}
	inv, found, err := arSvc.GetByOrder(ctx, ordCredit)
	if err != nil || !found || inv.BalanceMinor != 80000 {
		t.Fatalf("expected AR invoice for %s (found=%v inv=%+v err=%v)", ordCredit, found, inv, err)
	}

	// Attempting ForceCompleteOrder later on same orderID or idempotent replay must NOT double-post gl:order_delivered:<id>.
	if err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return svc.recordOrderDeliveredGLInTxn(ctx, txn, nil, &Order{
			OrderID: ordCredit, SupplierID: supID, RetailerID: retID, DriverID: drvID, Currency: "UZS", TotalMinor: 80000, Status: StatusCompleted,
		}, 0, 0, 80000, nil, "CREDIT", now)
	}); err != nil {
		t.Fatalf("replay order delivered GL: %v", err)
	}

	// (C) Split tender (WALLET 20k + CARD 30k + CASH 50k) + ForceCompleteOrder (escrow release without double debit).
	splitOrd, _, err := orderRepo.GetOrder(ctx, ordSplit)
	if err != nil {
		t.Fatalf("GetOrder split: %v", err)
	}
	if err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return svc.RecordSplitTenderTxn(ctx, txn, splitOrd, SplitTenderPlan{
			WalletMinor: 20000,
			CardMinor:   30000,
			CashMinor:   50000,
		})
	}); err != nil {
		t.Fatalf("RecordSplitTenderTxn: %v", err)
	}
	// Mark tax off for split order if we want exact 100k escrow match, OR tax adds 12k VAT (net 100k + vat 12k = 112k).
	// Clear taxSvc so ordSplit stays exactly 100,000 matching its 100,000 split escrow release cleanly!
	svc.SetTaxService(nil)
	if _, err := svc.ForceCompleteOrder(ctx, adminClaims, ordSplit, ForceReasonOFDDown); err != nil {
		t.Fatalf("ForceCompleteOrder(split): %v", err)
	}

	// Verify GL Trial Balance & Hash Chain across ALL posted entries for supplier.
	glSvc := gl.NewService(client)
	periodID := gl.ComputePeriodID(ctx, supID, "UZ", now)
	tb, err := glSvc.TrialBalance(ctx, supID, periodID)
	if err != nil {
		t.Fatalf("gl.TrialBalance: %v", err)
	}
	// Cash order: 112k + 2.5k fee; Credit order: 80k; Split tender: 100k; Split delivery escrow release: 100k + 2.5k fee => total 397,000!
	if !tb.Balanced || tb.TotalDebitsMinor != 397000 || tb.TotalCreditsMinor != 397000 {
		t.Fatalf("order GL trial balance mismatch: %+v", tb)
	}
	byAcct := make(map[string]int64, len(tb.Accounts))
	for _, a := range tb.Accounts {
		byAcct[a.AccountCode] = a.NetMinor
	}
	// ESCROW:ORDER credited 100k in split.tender and debited 100k on delivery -> net 0!
	if byAcct[gl.AccountPlatformEscrow] != 0 {
		t.Fatalf("expected ESCROW:ORDER net 0 after split delivery release, got %d (accounts=%+v)", byAcct[gl.AccountPlatformEscrow], byAcct)
	}
	// REV:SALES = 100k (cash) + 80k (credit) + 100k (split) = 280k; TAX:VAT:OUTPUT = 12k; AR:SUPPLIER:FEES = 5k (2x 2.5k)
	if byAcct[gl.AccountSalesRevenue] != 280000 || byAcct[gl.AccountVatOutput] != 12000 || byAcct[gl.AccountSupplierFeesReceivable] != 5000 || byAcct[gl.AccountAccountsReceivable] != 80000 {
		t.Fatalf("unexpected order GL account nets: %+v", byAcct)
	}

	chain, err := gl.VerifyEntityHashChain(ctx, client, supID)
	if err != nil || !chain.Valid || chain.EntriesCount != 6 {
		t.Fatalf("VerifyEntityHashChain got (%+v, %v), want Valid=true EntriesCount=6", chain, err)
	}

	var glOutbox int64
	it := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM OutboxEvents WHERE SupplierId = @sid AND AggregateType = 'JournalEntry'`,
		Params: map[string]any{"sid": supID},
	})
	defer it.Stop()
	if r, err := it.Next(); err == nil {
		_ = r.Column(0, &glOutbox)
	} else if err != iterator.Done {
		t.Fatalf("query gl outbox: %v", err)
	}
	if glOutbox != 6 {
		t.Fatalf("gl outbox count = %d, want 6 (got strings=%s)", glOutbox, strings.TrimSpace(supID))
	}
}

func TestOrder_CashShortfallAndCreditLeaveShortfall_GLDualWrite_LiveSpanner(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	suffix := uuid.NewString()[:8]
	supReg := "sup_sf_reg_" + suffix
	supCred := "sup_sf_cred_" + suffix
	retID := "ret_sf_" + suffix
	drvID := "drv_sf_" + suffix
	ordReg := "ord_sf_reg_" + suffix
	ordCred := "ord_sf_cred_" + suffix
	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)

	cleanupOrderGLTestState(t, client, supReg, []string{ordReg})
	cleanupOrderGLTestState(t, client, supCred, []string{ordCred})

	orderRepo := NewSpannerRepository(client)
	creditSvc := credit.NewService(credit.NewSpannerRepository(client))
	creditSvc.SetNow(func() time.Time { return now })
	arSvc := ar.NewService(ar.NewSpannerRepository(client))
	arSvc.SetNow(func() time.Time { return now })
	feeResolver := billing.NewFeeScheduleResolver(client)

	svc := NewService(ServiceConfig{
		Repo:               orderRepo,
		Credit:             creditSvc,
		CommissionResolver: feeResolver,
		SpannerClient:      client,
		Now:                func() time.Time { return now },
		Log:                slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	svc.SetARService(arSvc)
	svc.SetDoorstepShortfall(func(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID, supplierID, driverID, currency string, expectedMinor, receivedMinor int64, occurredAt time.Time) error {
		_, err := gl.PostEventInTxn(ctx, txn, nil, gl.RuleEventDoorstepShortfall, gl.RuleEvalContext{
			EntityID:       supplierID,
			SupplierID:     supplierID,
			DriverID:       driverID,
			OrderID:        orderID,
			SourceID:       orderID,
			Currency:       currency,
			ExpectedMinor:  expectedMinor,
			ReceivedMinor:  receivedMinor,
			IdempotencyKey: "gl:shortfall:" + orderID,
			Memo:           "Doorstep cash shortfall order " + orderID,
			OccurredAt:     occurredAt,
		})
		return err
	})

	// Seed:
	// 1. Regular cash order in PENDING_CASH_COLLECTION (100k, no VAT items -> 100k TotalMinor)
	// 2. Credit-leave order in SHOP_CLOSED_PENDING (100k)
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordReg, "RetailerId": retID, "SupplierId": supReg, "DriverId": drvID, "WarehouseId": "wh_sf_" + suffix,
			"Status": string(StatusPendingCashCollection), "OrderSource": string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed), "LineItemsJson": []byte("[]"),
			"TotalMinor": int64(100000), "Currency": "UZS", "Lat": 41.311, "Lng": 69.279,
			"Version": int64(1), "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordCred, "RetailerId": retID, "SupplierId": supCred, "DriverId": drvID, "WarehouseId": "wh_sf_" + suffix,
			"Status": string(StatusShopClosedPending), "OrderSource": string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed), "LineItemsJson": []byte("[]"),
			"TotalMinor": int64(100000), "Currency": "UZS", "Lat": 41.311, "Lng": 69.279,
			"ShopClosedGraceEndsAt": now.Add(-5 * time.Minute),
			"Version":               int64(1), "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("BillingFeeSchedules", map[string]any{
			"FeeScheduleId": "fee_sf_reg_" + suffix, "SupplierId": supReg, "Tier": "STANDARD",
			"PerOrderMinor": int64(0), "GmvBps": int64(0), "MonthlySubscriptionMinor": int64(0),
			"Currency": "UZS", "EffectiveFrom": now.Add(-time.Hour), "CreatedAt": now,
		}),
		spanner.InsertMap("BillingFeeSchedules", map[string]any{
			"FeeScheduleId": "fee_sf_cred_" + suffix, "SupplierId": supCred, "Tier": "STANDARD",
			"PerOrderMinor": int64(0), "GmvBps": int64(0), "MonthlySubscriptionMinor": int64(0),
			"Currency": "UZS", "EffectiveFrom": now.Add(-time.Hour), "CreatedAt": now,
		}),
		spanner.InsertMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": retID, "SupplierId": supCred, "CreditLimitMinor": int64(500000),
			"CurrentBalanceMinor": int64(0), "AvailableCreditMinor": int64(500000),
			"Status": string(credit.StatusActive), "RiskScore": int64(800),
			"DelinquencyCount": int64(0), "Version": int64(1), "CreatedAt": now, "UpdatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed shortfall test orders: %v", err)
	}

	drvClaims := auth.Claims{Subject: drvID, Role: auth.RoleDriver}
	admReg := auth.Claims{Subject: "adm_sf_reg", Role: auth.RoleAdmin, SupplierID: supReg}
	admCred := auth.Claims{Subject: "adm_sf_cred", Role: auth.RoleAdmin, SupplierID: supCred}
	got80k := int64(80000)

	// (A) Regular cash order with 20k shortfall (expected 100k, received 80k) -> ForceCompleteOrder.
	// CollectCash posts gl:shortfall:<ordReg> (DR CASH:DRIVER 80k + DR DRIVER:VARIANCE 20k / CR ESCROW:ORDER 100k).
	// ForceCompleteOrder must release DR ESCROW:ORDER 100k -> CR REV:SALES 100k without double-debiting CASH:DRIVER.
	if _, err := svc.CollectCash(ctx, drvClaims, CollectCashRequest{
		OrderID:             ordReg,
		Latitude:            41.311,
		Longitude:           69.279,
		AmountReceivedMinor: &got80k,
	}); err != nil {
		t.Fatalf("CollectCash(regular shortfall): %v", err)
	}
	if _, err := svc.ForceCompleteOrder(ctx, admReg, ordReg, ForceReasonOFDDown); err != nil {
		t.Fatalf("ForceCompleteOrder(regular shortfall): %v", err)
	}

	glSvc := gl.NewService(client)
	periodID := gl.ComputePeriodID(ctx, supReg, "UZ", now)
	tbReg, err := glSvc.TrialBalance(ctx, supReg, periodID)
	if err != nil {
		t.Fatalf("TrialBalance(supReg): %v", err)
	}
	if !tbReg.Balanced || tbReg.TotalDebitsMinor != 200000 || tbReg.TotalCreditsMinor != 200000 {
		t.Fatalf("regular shortfall trial balance mismatch: %+v", tbReg)
	}
	acctReg := make(map[string]int64, len(tbReg.Accounts))
	for _, a := range tbReg.Accounts {
		acctReg[a.AccountCode] = a.NetMinor
	}
	if acctReg[gl.AccountCashInTransit] != 80000 {
		t.Fatalf("regular shortfall CASH:DRIVER = %d, want 80000", acctReg[gl.AccountCashInTransit])
	}
	if acctReg[gl.AccountDriverReceivableVariance] != 20000 {
		t.Fatalf("regular shortfall DRIVER:VARIANCE = %d, want 20000", acctReg[gl.AccountDriverReceivableVariance])
	}
	if acctReg[gl.AccountPlatformEscrow] != 0 {
		t.Fatalf("regular shortfall ESCROW:ORDER = %d, want 0 (released on completion)", acctReg[gl.AccountPlatformEscrow])
	}
	if acctReg[gl.AccountSalesRevenue] != 100000 {
		t.Fatalf("regular shortfall REV:SALES = %d, want 100000", acctReg[gl.AccountSalesRevenue])
	}
	chainReg, err := gl.VerifyEntityHashChain(ctx, client, supReg)
	if err != nil || !chainReg.Valid || chainReg.EntriesCount != 2 {
		t.Fatalf("supReg hash chain (%+v, %v), want Valid=true EntriesCount=2", chainReg, err)
	}

	// (B) Credit-leave order (100k DR AR:RETAILER / CR REV:SALES) -> CollectCash 80k shortfall -> ForceCompleteOrder.
	// Must pay down AR by 80k via CASH:DRIVER (not double-debit CASH:DRIVER, not strand CR ESCROW:ORDER).
	if err := svc.resolveOneShopClosedTimeout(ctx, ordCred); err != nil {
		t.Fatalf("resolveOneShopClosedTimeout(credit): %v", err)
	}
	if _, err := svc.CollectCash(ctx, drvClaims, CollectCashRequest{
		OrderID:             ordCred,
		Latitude:            41.311,
		Longitude:           69.279,
		AmountReceivedMinor: &got80k,
	}); err != nil {
		t.Fatalf("CollectCash(credit-leave shortfall): %v", err)
	}
	if _, err := svc.ForceCompleteOrder(ctx, admCred, ordCred, ForceReasonOFDDown); err != nil {
		t.Fatalf("ForceCompleteOrder(credit-leave shortfall): %v", err)
	}

	inv, found, err := arSvc.GetByOrder(ctx, ordCred)
	if err != nil || !found || inv.BalanceMinor != 20000 || inv.Status != ar.StatusPartial {
		t.Fatalf("credit-leave AR invoice after 80k CollectCash = (%+v, found=%v, err=%v), want 20k PARTIAL", inv, found, err)
	}

	tbCred, err := glSvc.TrialBalance(ctx, supCred, periodID)
	if err != nil {
		t.Fatalf("TrialBalance(supCred): %v", err)
	}
	if !tbCred.Balanced || tbCred.TotalDebitsMinor != 180000 || tbCred.TotalCreditsMinor != 180000 {
		t.Fatalf("credit-leave shortfall trial balance mismatch: %+v", tbCred)
	}
	acctCred := make(map[string]int64, len(tbCred.Accounts))
	for _, a := range tbCred.Accounts {
		acctCred[a.AccountCode] = a.NetMinor
	}
	if acctCred[gl.AccountCashInTransit] != 80000 {
		t.Fatalf("credit-leave CASH:DRIVER = %d, want 80000 (must not double-debit)", acctCred[gl.AccountCashInTransit])
	}
	if acctCred[gl.AccountPlatformEscrow] != 0 {
		t.Fatalf("credit-leave ESCROW:ORDER = %d, want 0 (must not strand)", acctCred[gl.AccountPlatformEscrow])
	}
	if acctCred[gl.AccountAccountsReceivable] != 20000 {
		t.Fatalf("credit-leave AR:RETAILER = %d, want 20000 (must match ArInvoices.BalanceMinor)", acctCred[gl.AccountAccountsReceivable])
	}
	if acctCred[gl.AccountDriverReceivableVariance] != 0 {
		t.Fatalf("credit-leave DRIVER:VARIANCE = %d, want 0", acctCred[gl.AccountDriverReceivableVariance])
	}
	if acctCred[gl.AccountSalesRevenue] != 100000 {
		t.Fatalf("credit-leave REV:SALES = %d, want 100000", acctCred[gl.AccountSalesRevenue])
	}
	chainCred, err := gl.VerifyEntityHashChain(ctx, client, supCred)
	if err != nil || !chainCred.Valid || chainCred.EntriesCount != 2 {
		t.Fatalf("supCred hash chain (%+v, %v), want Valid=true EntriesCount=2", chainCred, err)
	}
}
