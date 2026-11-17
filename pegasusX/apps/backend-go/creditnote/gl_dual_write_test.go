package creditnote

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"google.golang.org/api/iterator"
)

func cleanupCreditNoteGLState(t *testing.T, client *spanner.Client, supplierID string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, q := range []struct {
			sql   string
			table string
			cols  []string
		}{
			{"SELECT EntityId, EntryId, Line FROM JournalPostings WHERE EntityId = @sid", "JournalPostings", []string{"EntityId", "EntryId", "Line"}},
			{"SELECT EntityId, EntryId FROM JournalEntries WHERE EntityId = @sid", "JournalEntries", []string{"EntityId", "EntryId"}},
			{"SELECT EntityId, PeriodId, AccountCode, PartyKey, Shard FROM GlBalances WHERE EntityId = @sid", "GlBalances", []string{"EntityId", "PeriodId", "AccountCode", "PartyKey", "Shard"}},
			{"SELECT EventId FROM OutboxEvents WHERE SupplierId = @sid", "OutboxEvents", []string{"EventId"}},
		} {
			var muts []*spanner.Mutation
			it := client.Single().Query(ctx, spanner.Statement{SQL: q.sql, Params: map[string]any{"sid": supplierID}})
			for {
				row, err := it.Next()
				if err != nil {
					break
				}
				vals := make([]any, len(q.cols))
				ptrs := make([]any, len(q.cols))
				for i := range q.cols {
					var sv spanner.GenericColumnValue
					ptrs[i] = &sv
					vals[i] = &sv
				}
				if err := row.Columns(ptrs...); err == nil {
					k := make(spanner.Key, len(q.cols))
					for i := range q.cols {
						gv := vals[i].(*spanner.GenericColumnValue)
						switch gv.Type.Code {
						case 2:
							var iv int64
							_ = gv.Decode(&iv)
							k[i] = iv
						default:
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
	})
}

func TestCreditNote_GLDualWrite_ZeroAndPositiveVAT_LiveSpanner(t *testing.T) {
	client := openCreditNoteSpannerTestClient(t)
	ctx := context.Background()
	repo := NewSpannerRepository(client)
	suffix := uuid.NewString()[:8]
	supID := "sup_cn_gl_" + suffix
	retID := "ret_cn_gl_" + suffix
	ordZero := "ord_cnz_" + suffix
	ordVat := "ord_cnv_" + suffix
	cnZero := "cn_z_" + suffix
	cnVat := "cn_v_" + suffix
	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)

	cleanupCreditNoteGLState(t, client, supID)
	cleanupCreditNoteSpannerState(t, client, []string{cnZero, cnVat}, []string{ordZero, ordVat}, []string{
		"jentry_cn_" + cnZero + "_p0", "jentry_cn_" + cnZero + "_p1",
		"jentry_cn_" + cnVat + "_p0", "jentry_cn_" + cnVat + "_p1",
	})
	seedTestOrder(t, ctx, client, ordZero, supID, retID, now)
	seedTestOrder(t, ctx, client, ordVat, supID, retID, now)

	// 1. VatMinor == 0
	if err := repo.SaveCreditNote(ctx, CreditNote{
		CreditNoteId:    cnZero,
		OrderId:         ordZero,
		Type:            CreditNoteTypeFullReturn,
		Status:          CreditNoteStatusFiscalPending,
		ReasonCode:      "DAMAGED",
		TotalNetMinor:   50000,
		TotalVatMinor:   0,
		TotalGrossMinor: 50000,
		CreatedBy:       "op",
		CreatedAt:       now,
	}, EventCreditNoteCreated); err != nil {
		t.Fatalf("SaveCreditNote zero VAT: %v", err)
	}

	// 2. VatMinor > 0 (net 100k + VAT 12k = gross 112k) + idempotent replay check.
	vatNote := CreditNote{
		CreditNoteId:    cnVat,
		OrderId:         ordVat,
		Type:            CreditNoteTypePartial,
		Status:          CreditNoteStatusFiscalPending,
		ReasonCode:      "SHORTAGE",
		TotalNetMinor:   100000,
		TotalVatMinor:   12000,
		TotalGrossMinor: 112000,
		CreatedBy:       "op",
		CreatedAt:       now,
	}
	if err := repo.SaveCreditNote(ctx, vatNote, EventCreditNoteCreated); err != nil {
		t.Fatalf("SaveCreditNote with VAT: %v", err)
	}
	if err := repo.SaveCreditNote(ctx, vatNote, EventCreditNoteCreated); err != nil {
		t.Fatalf("SaveCreditNote idempotent replay: %v", err)
	}

	glSvc := gl.NewService(client)
	periodID := gl.ComputePeriodID(ctx, supID, "UZ", time.Now().UTC())
	tb, err := glSvc.TrialBalance(ctx, supID, periodID)
	if err != nil {
		t.Fatalf("gl.TrialBalance: %v", err)
	}
	if !tb.Balanced || tb.TotalDebitsMinor != 162000 || tb.TotalCreditsMinor != 162000 {
		t.Fatalf("creditnote GL trial balance mismatch: %+v", tb)
	}

	byAcct := make(map[string]int64, len(tb.Accounts))
	for _, a := range tb.Accounts {
		byAcct[a.AccountCode] = a.NetMinor
	}
	if byAcct[gl.AccountCreditNoteAdjustment] != 150000 || byAcct[gl.AccountVatOutput] != -12000 || byAcct[gl.AccountAccountsReceivable] != -162000 {
		t.Fatalf("unexpected account balances: %+v", byAcct)
	}

	chain, err := gl.VerifyEntityHashChain(ctx, client, supID)
	if err != nil || !chain.Valid || chain.EntriesCount != 2 {
		t.Fatalf("VerifyEntityHashChain got (%+v, %v), want Valid=true EntriesCount=2", chain, err)
	}

	var glOutbox int64
	it := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM OutboxEvents WHERE SupplierId = @sid AND AggregateType = @agg AND TopicName = @topic`,
		Params: map[string]any{"sid": supID, "agg": events.AggregateJournalEntry, "topic": events.TopicLedger},
	})
	defer it.Stop()
	if r, err := it.Next(); err == nil {
		_ = r.Column(0, &glOutbox)
	} else if err != iterator.Done {
		t.Fatalf("query gl outbox: %v", err)
	}
	if glOutbox != 2 {
		t.Fatalf("gl.entry.posted outbox count = %d, want 2", glOutbox)
	}
}
