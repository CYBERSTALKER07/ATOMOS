package cashrecon

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openCashReconSpannerTestClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live cashrecon GL test")
	}

	envOrDefault := func(k, fb string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return fb
	}
	dbPath := fmt.Sprintf(
		"projects/%s/instances/%s/databases/%s",
		envOrDefault("SPANNER_PROJECT", "pegasusx-local"),
		envOrDefault("SPANNER_INSTANCE", "pegasusx-instance"),
		envOrDefault("SPANNER_DATABASE", "pegasusx-db"),
	)

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
		t.Fatalf("open spanner client: %v", err)
	}
	if _, err := client.Single().ReadRow(ctx, "CashReconciliations", spanner.Key{"__cr_probe__"}, []string{"ReconciliationId"}); err != nil && spanner.ErrCode(err) != codes.NotFound {
		client.Close()
		t.Fatalf("probe CashReconciliations: %v", err)
	}
	t.Cleanup(client.Close)
	return client
}

func cleanupCashReconGLState(t *testing.T, client *spanner.Client, supplierID string, crIDs []string) {
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
			{"SELECT LedgerEntryId FROM PaymentLedgerEntries WHERE SupplierId = @sid", "PaymentLedgerEntries", []string{"LedgerEntryId"}},
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
		var crMuts []*spanner.Mutation
		for _, id := range crIDs {
			crMuts = append(crMuts, spanner.Delete("CashReconciliations", spanner.Key{id}))
		}
		if len(crMuts) > 0 {
			_, _ = client.Apply(ctx, crMuts)
		}
	})
}

func TestCashRecon_GLDualWrite_OpenAcceptWriteOff_LiveSpanner(t *testing.T) {
	client := openCashReconSpannerTestClient(t)
	ctx := context.Background()
	repo := NewSpannerRepository(client)
	suffix := uuid.NewString()[:8]
	supID := "sup_cr_gl_" + suffix
	drvID := "drv_cr_gl_" + suffix
	crShort := "cr_s_" + suffix
	crOver := "cr_o_" + suffix
	cleanupCashReconGLState(t, client, supID, []string{crShort, crOver})

	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)
	resolved := now.Add(time.Minute)
	by := "fin_admin"

	// 1. Shortfall OPEN (-15,000) then WRITE_OFF
	short := CashReconciliation{
		ReconciliationId:  crShort,
		SupplierId:        supID,
		DriverId:          drvID,
		ShiftDate:         now,
		ExpectedCashMinor: 100000,
		DeclaredCashMinor: 85000,
		DifferenceMinor:   -15000,
		Status:            ReconciliationStatusPending,
		CreatedAt:         now,
	}
	if err := repo.SaveReconciliation(ctx, short, EventCashReconciliationCreated); err != nil {
		t.Fatalf("SaveReconciliation(short OPEN): %v", err)
	}
	short.Status = ReconciliationStatusWriteOff
	short.ResolvedAt = &resolved
	short.ResolvedBy = &by
	if err := repo.SaveReconciliation(ctx, short, EventCashReconciliationWrittenOff); err != nil {
		t.Fatalf("SaveReconciliation(short WRITE_OFF): %v", err)
	}

	// 2. Overage OPEN (+10,000) then ACCEPT
	over := CashReconciliation{
		ReconciliationId:  crOver,
		SupplierId:        supID,
		DriverId:          drvID,
		ShiftDate:         now,
		ExpectedCashMinor: 50000,
		DeclaredCashMinor: 60000,
		DifferenceMinor:   10000,
		Status:            ReconciliationStatusPending,
		CreatedAt:         now,
	}
	if err := repo.SaveReconciliation(ctx, over, EventCashReconciliationCreated); err != nil {
		t.Fatalf("SaveReconciliation(over OPEN): %v", err)
	}
	over.Status = ReconciliationStatusAccepted
	over.ResolvedAt = &resolved
	over.ResolvedBy = &by
	if err := repo.SaveReconciliation(ctx, over, EventCashReconciliationAccepted); err != nil {
		t.Fatalf("SaveReconciliation(over ACCEPT): %v", err)
	}

	glSvc := gl.NewService(client)
	periodID := gl.ComputePeriodID(ctx, supID, "UZ", now)
	tb, err := glSvc.TrialBalance(ctx, supID, periodID)
	if err != nil {
		t.Fatalf("gl.TrialBalance: %v", err)
	}
	if !tb.Balanced || tb.TotalDebitsMinor != 50000 || tb.TotalCreditsMinor != 50000 {
		t.Fatalf("cashrecon GL trial balance mismatch: %+v", tb)
	}

	byAcct := make(map[string]int64, len(tb.Accounts))
	for _, a := range tb.Accounts {
		byAcct[a.AccountCode] = a.NetMinor
	}
	if byAcct[gl.AccountDriverReceivableVariance] != 0 || byAcct[gl.AccountDriverPayableVariance] != 0 ||
		byAcct[gl.AccountCashWriteOffExpense] != 15000 || byAcct[gl.AccountCashOverShortRevenue] != 10000 {
		t.Fatalf("unexpected cashrecon net account balances: %+v", byAcct)
	}

	chain, err := gl.VerifyEntityHashChain(ctx, client, supID)
	if err != nil || !chain.Valid || chain.EntriesCount != 4 {
		t.Fatalf("VerifyEntityHashChain got (%+v, %v), want Valid=true EntriesCount=4", chain, err)
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
	if glOutbox != 4 {
		t.Fatalf("gl.entry.posted outbox count = %d, want 4", glOutbox)
	}
}
