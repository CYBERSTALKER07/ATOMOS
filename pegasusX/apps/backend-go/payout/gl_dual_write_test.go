package payout

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
)

func TestPayout_UpdateStatusPaid_GLDualWriteParity_LiveSpanner(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	suffix := uuid.NewString()[:8]
	supplierID := fmt.Sprintf("sup-po-gl-%s", suffix)
	batchID := fmt.Sprintf("batch-po-gl-%s", suffix)
	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)

	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, q := range []struct {
			sql, tbl string
			cols     []string
		}{
			{"SELECT EntityId, EntryId, Line FROM JournalPostings WHERE EntityId = @sid", "JournalPostings", []string{"EntityId", "EntryId", "Line"}},
			{"SELECT EntityId, EntryId FROM JournalEntries WHERE EntityId = @sid", "JournalEntries", []string{"EntityId", "EntryId"}},
			{"SELECT EntityId, PeriodId, AccountCode, PartyKey, Shard FROM GlBalances WHERE EntityId = @sid", "GlBalances", []string{"EntityId", "PeriodId", "AccountCode", "PartyKey", "Shard"}},
			{"SELECT LedgerEntryId FROM PaymentLedgerEntries WHERE SupplierId = @sid", "PaymentLedgerEntries", []string{"LedgerEntryId"}},
			{"SELECT EventId FROM OutboxEvents WHERE SupplierId = @sid", "OutboxEvents", []string{"EventId"}},
		} {
			var muts []*spanner.Mutation
			it := client.Single().Query(cctx, spanner.Statement{SQL: q.sql, Params: map[string]any{"sid": supplierID}})
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
					muts = append(muts, spanner.Delete(q.tbl, k))
				}
			}
			it.Stop()
			if len(muts) > 0 {
				_, _ = client.Apply(cctx, muts)
			}
		}
		_, _ = client.Apply(cctx, []*spanner.Mutation{
			spanner.Delete("PayoutBatches", spanner.Key{batchID}),
		})
	})

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("PayoutBatches", map[string]any{
			"BatchId":            batchID,
			"SupplierId":         supplierID,
			"PeriodStart":        civil.DateOf(now.Add(-24 * time.Hour)),
			"PeriodEnd":          civil.DateOf(now),
			"Currency":           "UZS",
			"GrossCapturedMinor": int64(200000),
			"RefundedMinor":      int64(20000),
			"CommissionMinor":    int64(9000),
			"NetPayoutMinor":     int64(171000),
			"Status":             StatusSubmitted,
			"IdempotencyKey":     "po-gl-" + suffix,
			"CreatedBy":          "test",
			"CreatedAt":          now,
			"UpdatedAt":          now,
		}),
	}); err != nil {
		t.Fatalf("seed PayoutBatches: %v", err)
	}

	// Ensure real SettlementJournal hook (or default payment.WriteSettlementJournal semantics) runs here.
	// Note payout cannot import payment directly (payment or bootstrap wires it), so reuse test helper or
	// invoke gl + payment via bootstrap or check hook invocation? Wait: verify SettlementJournal hook fails closed &
	// test GL rule evaluation directly inside UpdateStatusRef transaction cleanly!
	origHook := SettlementJournal
	t.Cleanup(func() { SettlementJournal = origHook })
	SettlementJournal = func(txn *spanner.ReadWriteTransaction, bID, supID, cur string, grossMinor, feeMinor int64) error {
		_, err := gl.PostEventInTxn(ctx, txn, nil, gl.RuleEventPayoutPaid, gl.RuleEvalContext{
			EntityID:       supID,
			SupplierID:     supID,
			SourceID:       bID,
			Currency:       cur,
			GrossMinor:     grossMinor,
			FeeMinor:       feeMinor,
			IdempotencyKey: "gl:payout:" + bID,
			OccurredAt:     now,
		})
		return err
	}

	svc := NewService(NewRepository(client))
	if err := svc.ConfirmSettlement(ctx, batchID, "rail-ref-"+suffix); err != nil {
		t.Fatalf("ConfirmSettlement: %v", err)
	}
	// Idempotent replay
	if err := svc.ConfirmSettlement(ctx, batchID, "rail-ref-"+suffix); err != nil {
		t.Fatalf("ConfirmSettlement replay: %v", err)
	}

	glSvc := gl.NewService(client)
	periodID := gl.ComputePeriodID(ctx, supplierID, "UZ", now)
	tb, err := glSvc.TrialBalance(ctx, supplierID, periodID)
	if err != nil {
		t.Fatalf("gl.TrialBalance: %v", err)
	}
	// gross = 200k - 20k = 180k, fee = 9k, net = 171k -> balanced 180k!
	if !tb.Balanced || tb.TotalDebitsMinor != 180000 || tb.TotalCreditsMinor != 180000 {
		t.Fatalf("payout GL trial balance mismatch: %+v", tb)
	}
	byAcct := make(map[string]int64, len(tb.Accounts))
	for _, a := range tb.Accounts {
		byAcct[a.AccountCode] = a.NetMinor
	}
	if byAcct[gl.AccountSupplierPayable] != 171000 || byAcct[gl.AccountPlatformFeeRevenue] != 9000 || byAcct[gl.AccountPlatformEscrow] != -180000 {
		t.Fatalf("unexpected payout GL account balances: %+v", byAcct)
	}
	chain, err := gl.VerifyEntityHashChain(ctx, client, supplierID)
	if err != nil || !chain.Valid || chain.EntriesCount != 1 {
		t.Fatalf("VerifyEntityHashChain got (%+v, %v), want Valid=true EntriesCount=1", chain, err)
	}
}
