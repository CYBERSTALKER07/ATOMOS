package migration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

func cleanupMigrationGLTestState(t *testing.T, client *spanner.Client, supplierID, sessionID string) {
	t.Helper()
	cleanupMigrationSpannerTestData(t, client, supplierID, nil)
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
			{"SELECT supplier_id, session_id FROM SupplierImportSessions WHERE supplier_id = @sid", "SupplierImportSessions", []string{"supplier_id", "session_id"}},
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
		_ = sessionID
	})
}

func TestMigration_NasiyaApplyAndRollback_GLDualWriteAndArLedgerAppendOnly_LiveSpanner(t *testing.T) {
	client := requireSpannerEmulatorForMigrationTest(t)
	ctx := context.Background()
	suffix := uuid.NewString()[:10]
	supplierID := "sup_mig_gl_" + suffix
	retailerID := "ret_mig_gl_" + suffix
	sessionID := "ses_mig_gl_" + suffix
	cleanupMigrationGLTestState(t, client, supplierID, sessionID)

	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Suppliers", map[string]any{
			"SupplierId":  supplierID,
			"Name":        "Migration GL Supplier",
			"CountryCode": "UZ",
			"Currency":    "UZS",
			"CreatedAt":   now,
			"UpdatedAt":   now,
		}),
		spanner.InsertMap("SupplierImportSessions", map[string]any{
			"supplier_id": supplierID,
			"session_id":  sessionID,
			"pack":        string(PackNasiyaAR),
			"file_name":   "nasiya.xlsx",
			"status":      string(SessionStatusApplied),
			"created_at":  now,
			"updated_at":  now,
		}),
	}); err != nil {
		t.Fatalf("seed migration parent rows: %v", err)
	}

	recorder := buildPaymentMigrationJournalRecorder(nil)
	var invIDs [2]string
	err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		cache := ar.NewMigrationTxnCache()
		for i, amt := range []int64{120000, 80000} {
			out, err := ar.OpenFromMigrationInTxn(ctx, txn, cache, ar.OpenFromMigrationRequest{
				SessionID:        sessionID,
				SupplierID:       supplierID,
				RetailerID:       retailerID,
				ExternalID:       fmt.Sprintf("ext-mig-%s-%d", suffix, i),
				EntryID:          fmt.Sprintf("jentry_mig_%s_%d", suffix, i),
				SupplierCurrency: "UZS",
				Currency:         "UZS",
				PrincipalMinor:   amt,
				BalanceMinor:     amt,
				CreditLeaveAt:    now,
				DueAt:            now.Add(14 * 24 * time.Hour),
				TermsDays:        14,
				Now:              now,
				JournalRecorder:  recorder,
			})
			if err != nil {
				return err
			}
			invIDs[i] = out.Invoice.InvoiceID
		}
		return ar.FlushMigrationCreditProfilesAndGLInTxn(ctx, txn, nil, cache, now)
	})
	if err != nil {
		t.Fatalf("chunked migration apply with GL: %v", err)
	}

	glSvc := gl.NewService(client)
	periodID := gl.ComputePeriodID(ctx, supplierID, "UZ", now)
	tbAfterOpen, err := glSvc.TrialBalance(ctx, supplierID, periodID)
	if err != nil {
		t.Fatalf("TrialBalance after open: %v", err)
	}
	if !tbAfterOpen.Balanced || tbAfterOpen.TotalDebitsMinor != 200000 || tbAfterOpen.TotalCreditsMinor != 200000 {
		t.Fatalf("unexpected migration open TB: %+v", tbAfterOpen)
	}

	// ExecuteSessionRollback reverses both invoices + ArLedgerEntries (InsertMap) + batched GL rollbacks.
	repo := NewSpannerRepository(client)
	snapshots := []MigrationSnapshotRecord{
		{
			SupplierID:  supplierID,
			SessionID:   sessionID,
			SnapshotSeq: 1,
			TargetTable: "ArInvoices",
			RowKeyJSON:  fmt.Sprintf(`{"InvoiceId":%q}`, invIDs[0]),
			OpKind:      SnapshotOpInsert,
			CreatedAt:   now,
		},
		{
			SupplierID:  supplierID,
			SessionID:   sessionID,
			SnapshotSeq: 2,
			TargetTable: "ArInvoices",
			RowKeyJSON:  fmt.Sprintf(`{"InvoiceId":%q}`, invIDs[1]),
			OpKind:      SnapshotOpInsert,
			CreatedAt:   now,
		},
	}
	rbRes, err := repo.ExecuteSessionRollback(ctx, supplierID, sessionID, snapshots, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ExecuteSessionRollback: %v", err)
	}
	if rbRes.ReversedInvoices != 2 || rbRes.ReversedAmountMinor != 200000 {
		t.Fatalf("unexpected rollback result: %+v", rbRes)
	}

	// Idempotent second rollback replay must not add duplicate rows.
	if _, err := repo.ExecuteSessionRollback(ctx, supplierID, sessionID, snapshots, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("idempotent ExecuteSessionRollback replay: %v", err)
	}

	chain, err := gl.VerifyEntityHashChain(ctx, client, supplierID)
	if err != nil || !chain.Valid || chain.EntriesCount != 4 {
		t.Fatalf("VerifyEntityHashChain got (%+v, %v), want Valid=true EntriesCount=4", chain, err)
	}

	tbAfterRollback, err := glSvc.TrialBalance(ctx, supplierID, periodID)
	if err != nil {
		t.Fatalf("TrialBalance after rollback: %v", err)
	}
	if !tbAfterRollback.Balanced || tbAfterRollback.TotalDebitsMinor != 400000 || tbAfterRollback.TotalCreditsMinor != 400000 {
		t.Fatalf("expected balanced 400k totals after rollback, got %+v", tbAfterRollback)
	}
	for _, acct := range tbAfterRollback.Accounts {
		if acct.NetMinor != 0 {
			t.Fatalf("expected net zero on %s after rollback, got %+v", acct.AccountCode, acct)
		}
	}

	var arRows int64
	it := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM ArLedgerEntries WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": supplierID},
	})
	defer it.Stop()
	if row, err := it.Next(); err == nil {
		_ = row.Column(0, &arRows)
	} else if err != iterator.Done {
		t.Fatalf("count ArLedgerEntries: %v", err)
	}
	if arRows != 4 {
		t.Fatalf("ArLedgerEntries row count = %d, want 4 (2 OPENING_BALANCE + 2 MIGRATION_ROLLBACK)", arRows)
	}
}
