package gl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openGLSpannerTestClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Spanner GL integration test")
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
			t.Fatalf("spanner emulator connect (%s): %v", dbPath, err)
		}
		t.Skipf("spanner emulator unavailable (%s): %v", dbPath, err)
	}

	_, probeErr := client.Single().ReadRow(ctx, "JournalEntries", spanner.Key{"__gl_probe__", "__gl_probe__"}, []string{"EntryId"})
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("spanner JournalEntries unavailable (%s): %v", dbPath, probeErr)
		}
		t.Skipf("spanner GL tables unavailable (%s): %v", dbPath, probeErr)
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

func uniqueGLEntity(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func cleanupGLEntityState(t *testing.T, client *spanner.Client, entityID string, pleIDs []string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var muts []*spanner.Mutation

		// Delete child JournalPostings before parent JournalEntries (ON DELETE NO ACTION).
		muts = append(muts,
			spanner.Delete("JournalPostings", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("JournalEntries", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlBalances", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlAccounts", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
		)
		for _, pleID := range pleIDs {
			muts = append(muts, spanner.Delete("PaymentLedgerEntries", spanner.Key{pleID}))
		}

		// Delete OutboxEvents emitted for entityID.
		iter := client.Single().Query(ctx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": entityID},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var eid string
			if row.Column(0, &eid) == nil {
				muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		iter.Stop()

		_, _ = client.Apply(ctx, muts)
	})
}

func TestSpannerGL_SeedAccountsAndRules_UZAndKZ(t *testing.T) {
	client := openGLSpannerTestClient(t)
	ctx := context.Background()
	entityID := uniqueGLEntity("sup_seed")
	cleanupGLEntityState(t, client, entityID, nil)

	svc := NewService(client)
	if err := svc.SeedDefaults(ctx, entityID, "UZ", "UZS"); err != nil {
		t.Fatalf("SeedDefaults failed: %v", err)
	}
	// Second run must be idempotent (0 inserted).
	n, err := SeedDefaultAccounts(ctx, client, entityID, "UZ", "UZS")
	if err != nil || n != 0 {
		t.Fatalf("second SeedDefaultAccounts = (%d, %v), want (0, nil)", n, err)
	}

	rulesUZ, err := svc.ListPostingRules(ctx, "UZ", "")
	if err != nil || len(rulesUZ) < 15 {
		t.Fatalf("ListPostingRules(UZ) = (%d, %v), want >= 15 rules", len(rulesUZ), err)
	}
	rulesKZ, err := svc.ListPostingRules(ctx, "KZ", "")
	if err != nil || len(rulesKZ) < 15 {
		t.Fatalf("ListPostingRules(KZ) = (%d, %v), want >= 15 rules", len(rulesKZ), err)
	}
}

func TestSpannerGL_AppendOnlyJournal_HashChain_Idempotency_Outbox(t *testing.T) {
	client := openGLSpannerTestClient(t)
	ctx := context.Background()
	entityID := uniqueGLEntity("sup_chain")
	cleanupGLEntityState(t, client, entityID, nil)

	svc := NewService(client)
	if _, err := SeedDefaultAccounts(ctx, client, entityID, "UZ", "UZS"); err != nil {
		t.Fatalf("seed accounts: %v", err)
	}

	t1 := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	r1, err := svc.PostEvent(ctx, RuleEventOrderDelivered, RuleEvalContext{
		EntityID: entityID, SourceID: "ord-1", OrderID: "ord-1", DriverID: "drv-1",
		PaymentMode: "CASH", Currency: "UZS", IdempotencyKey: "k-1", OccurredAt: t1,
		NetMinor: 100000, VatMinor: 12000, GrossMinor: 112000, CostMinor: 70000,
	})
	if err != nil {
		t.Fatalf("post entry 1: %v", err)
	}
	if r1.PrevHash != GenesisHash || r1.AlreadyPosted {
		t.Fatalf("entry 1 unexpected result: %+v", r1)
	}

	r2, err := svc.PostEvent(ctx, RuleEventOrderDelivered, RuleEvalContext{
		EntityID: entityID, SourceID: "ord-2", OrderID: "ord-2", RetailerID: "ret-1",
		PaymentMode: "CREDIT", Currency: "UZS", IdempotencyKey: "k-2", OccurredAt: t1.Add(time.Minute),
		NetMinor: 50000, VatMinor: 6000, GrossMinor: 56000,
	})
	if err != nil {
		t.Fatalf("post entry 2: %v", err)
	}
	if r2.PrevHash != r1.Hash || r2.AlreadyPosted {
		t.Fatalf("entry 2 prev_hash=%s want=%s", r2.PrevHash, r1.Hash)
	}

	// Idempotent replay of k-2: must NOT increment GlBalances or emit another outbox event.
	r2Dup, err := svc.PostEvent(ctx, RuleEventOrderDelivered, RuleEvalContext{
		EntityID: entityID, SourceID: "ord-2", OrderID: "ord-2", RetailerID: "ret-1",
		PaymentMode: "CREDIT", Currency: "UZS", IdempotencyKey: "k-2", OccurredAt: t1.Add(time.Minute),
		NetMinor: 50000, VatMinor: 6000, GrossMinor: 56000,
	})
	if err != nil {
		t.Fatalf("idempotent replay failed: %v", err)
	}
	if !r2Dup.AlreadyPosted || r2Dup.Hash != r2.Hash || r2Dup.EntryID != r2.EntryID {
		t.Fatalf("idempotent replay mismatch: got %+v want EntryID=%s Hash=%s", r2Dup, r2.EntryID, r2.Hash)
	}

	r3, err := svc.PostEvent(ctx, RuleEventARPaymentReceived, RuleEvalContext{
		EntityID: entityID, SourceID: "inv-1", RetailerID: "ret-1",
		PaymentMode: "BANK", Currency: "UZS", IdempotencyKey: "k-3", OccurredAt: t1.Add(2 * time.Minute),
		GrossMinor: 56000,
	})
	if err != nil {
		t.Fatalf("post entry 3: %v", err)
	}
	if r3.PrevHash != r2.Hash {
		t.Fatalf("entry 3 prev_hash=%s want=%s", r3.PrevHash, r2.Hash)
	}

	// Verify cryptographic hash chain end-to-end.
	verify, err := VerifyEntityHashChain(ctx, client, entityID)
	if err != nil {
		t.Fatalf("VerifyEntityHashChain error: %v", err)
	}
	if !verify.Valid || verify.EntriesCount != 3 || verify.HeadHash != r3.Hash {
		t.Fatalf("chain verification failed: %+v", verify)
	}

	// Verify OutboxEvents emitted exactly 3 rows on TopicLedger.
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT AggregateType, AggregateId, TopicName, Payload FROM OutboxEvents WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": entityID},
	})
	defer iter.Stop()
	outboxCount := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("query outbox: %v", err)
		}
		var aggType, aggID, topic string
		var payload []byte
		if err := row.Columns(&aggType, &aggID, &topic, &payload); err != nil {
			t.Fatalf("scan outbox: %v", err)
		}
		outboxCount++
		if aggType != events.AggregateJournalEntry {
			t.Fatalf("outbox AggregateType=%q want=%q", aggType, events.AggregateJournalEntry)
		}
		if topic != events.TopicLedger {
			t.Fatalf("outbox TopicName=%q want=%q", topic, events.TopicLedger)
		}
		var ev events.GLEntryPostedEvent
		if err := json.Unmarshal(payload, &ev); err != nil {
			t.Fatalf("unmarshal GLEntryPostedEvent: %v", err)
		}
		if ev.Type != events.EventGLEntryPosted || ev.EntityID != entityID || ev.TotalDebitMinor != ev.TotalCreditMinor {
			t.Fatalf("invalid outbox payload: %+v", ev)
		}
	}
	if outboxCount != 3 {
		t.Fatalf("outbox count = %d, want 3 (no duplicate emit on replay)", outboxCount)
	}

	// Verify Trial Balance for 2026-11 is balanced and AR:RETAILER nets to 0 (56k Dr - 56k Cr).
	tb, err := svc.TrialBalance(ctx, entityID, "2026-11")
	if err != nil {
		t.Fatalf("TrialBalance error: %v", err)
	}
	if !tb.Balanced || tb.TotalDebitsMinor != 294000 || tb.TotalCreditsMinor != 294000 {
		t.Fatalf("trial balance unexpected: %+v", tb)
	}
}

func TestSpannerGL_BatchInTxn_ChainingAndBalanceCoalescing(t *testing.T) {
	client := openGLSpannerTestClient(t)
	ctx := context.Background()
	entityID := uniqueGLEntity("sup_batch")
	cleanupGLEntityState(t, client, entityID, nil)

	now := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	var batchRes []JournalEntryResult

	// Post 4 items in one RW transaction (including an intra-batch duplicate at index 2).
	err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		in1, err := BuildARPaymentInput(RuleEvalContext{
			EntityID: entityID, SourceID: "inv-b1", RetailerID: "ret-1", PaymentMode: "BANK",
			Currency: "UZS", IdempotencyKey: "batch-1", OccurredAt: now, GrossMinor: 30000,
		})
		if err != nil {
			return err
		}
		in2, err := BuildARPaymentInput(RuleEvalContext{
			EntityID: entityID, SourceID: "inv-b2", RetailerID: "ret-1", PaymentMode: "BANK",
			Currency: "UZS", IdempotencyKey: "batch-2", OccurredAt: now.Add(time.Second), GrossMinor: 20000,
		})
		if err != nil {
			return err
		}
		in3, err := BuildPerOrderFeeInput(RuleEvalContext{
			EntityID: entityID, SourceID: "ord-b3", OrderID: "ord-b3",
			Currency: "UZS", IdempotencyKey: "batch-3", OccurredAt: now.Add(2 * time.Second), FeeMinor: 2500,
		})
		if err != nil {
			return err
		}
		res, err := BufferJournalEntriesBatchInTxn(ctx, txn, nil, []JournalEntryInput{in1, in2, in1, in3})
		if err != nil {
			return err
		}
		batchRes = res
		return nil
	})
	if err != nil {
		t.Fatalf("batch txn failed: %v", err)
	}
	if len(batchRes) != 4 {
		t.Fatalf("len(batchRes) = %d, want 4", len(batchRes))
	}
	if batchRes[0].PrevHash != GenesisHash || batchRes[1].PrevHash != batchRes[0].Hash || !batchRes[2].AlreadyPosted || batchRes[3].PrevHash != batchRes[1].Hash {
		t.Fatalf("unexpected intra-batch hash chain: %+v", batchRes)
	}

	// Next sequential transaction MUST pick up batchRes[3].Hash as PrevHash even though all 3 batch rows share the same PostedAt!
	svc := NewService(client)
	afterRes, err := svc.PostEvent(ctx, RuleEventFeePerOrder, RuleEvalContext{
		EntityID: entityID, SourceID: "ord-b4", OrderID: "ord-b4",
		Currency: "UZS", IdempotencyKey: "after-batch", OccurredAt: now.Add(3 * time.Second), FeeMinor: 1500,
	})
	if err != nil {
		t.Fatalf("post after batch: %v", err)
	}
	if afterRes.PrevHash != batchRes[3].Hash {
		t.Fatalf("after-batch PrevHash=%s want=%s", afterRes.PrevHash, batchRes[3].Hash)
	}

	verify, err := VerifyEntityHashChain(ctx, client, entityID)
	if err != nil || !verify.Valid || verify.EntriesCount != 4 || verify.HeadHash != afterRes.Hash {
		t.Fatalf("VerifyEntityHashChain after batch failed: %+v (err=%v)", verify, err)
	}
}

func TestSpannerGL_ConcurrentWriters_HashChainIntegrity(t *testing.T) {
	client := openGLSpannerTestClient(t)
	ctx := context.Background()
	entityID := uniqueGLEntity("sup_conc")
	cleanupGLEntityState(t, client, entityID, nil)

	svc := NewService(client)
	const workers = 6
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.PostEvent(ctx, RuleEventFeePerOrder, RuleEvalContext{
				EntityID: entityID, SourceID: fmt.Sprintf("ord-c-%d", i),
				Currency: "UZS", IdempotencyKey: fmt.Sprintf("conc-%d", i),
				FeeMinor: 1000,
			})
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent GL write failed: %v", err)
	}

	verify, err := VerifyEntityHashChain(ctx, client, entityID)
	if err != nil {
		t.Fatalf("VerifyEntityHashChain failed: %v", err)
	}
	if !verify.Valid || verify.EntriesCount != workers {
		t.Fatalf("concurrent chain verification failed: %+v", verify)
	}
}

func TestSpannerGL_PostingRulesDBOverrideAndAccountGuards(t *testing.T) {
	client := openGLSpannerTestClient(t)
	ctx := context.Background()
	entityID := uniqueGLEntity("sup_kz_ovr")
	cleanupGLEntityState(t, client, entityID, nil)

	// Insert custom account and v2 KZ rule override into Spanner.
	customAcct := "REV:KZ:CUSTOM_FEE"
	inactiveAcct := "EXP:CLOSED_ACCT"
	now := time.Now().UTC()
	past := now.Add(-24 * time.Hour)
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("GlAccounts", map[string]any{
			"EntityId": entityID, "AccountCode": customAcct, "Class": ClassRevenue,
			"Name": "KZ Custom Fee Revenue", "Currency": "KZT", "IsControl": false,
			"CountryPack": "KZ", "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("GlAccounts", map[string]any{
			"EntityId": entityID, "AccountCode": inactiveAcct, "Class": ClassExpense,
			"Name": "Closed Account", "Currency": "KZT", "IsControl": false,
			"ActiveTo": past, "CountryPack": "KZ", "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("PostingRules", map[string]any{
			"CountryPack": "KZ", "RuleId": "kz_fee_per_order_accrual", "Version": int64(2),
			"EventType": RuleEventFeePerOrder, "ConditionExpr": "fee_minor > 0",
			"DebitAccount": AccountSupplierFeesReceivable, "CreditAccount": customAcct,
			"AmountExpr": "fee_minor", "Description": "KZ v2 custom fee override",
			"Active": true, "UpdatedAt": spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		t.Fatalf("seed custom rule/accounts: %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.Apply(context.Background(), []*spanner.Mutation{
			spanner.Delete("PostingRules", spanner.Key{"KZ", "kz_fee_per_order_accrual", int64(2)}),
		})
	})

	svc := NewService(client)
	res, err := svc.PostEvent(ctx, RuleEventFeePerOrder, RuleEvalContext{
		EntityID: entityID, CountryPack: "KZ", SourceID: "ord-kz-ovr", Currency: "KZT",
		IdempotencyKey: "kz-ovr-1", FeeMinor: 2500, OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("PostEvent KZ override failed: %v", err)
	}
	rec, err := svc.GetEntry(ctx, entityID, res.EntryID)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if rec.Postings[1].AccountCode != customAcct {
		t.Fatalf("credit account = %q, want DB v2 rule %q", rec.Postings[1].AccountCode, customAcct)
	}

	// Unknown account fails closed.
	_, err = svc.PostEntry(ctx, JournalEntryInput{
		EntityID: entityID, SourceType: "MANUAL", SourceID: "m-bad", Currency: "KZT",
		IdempotencyKey: "bad-acct", OccurredAt: now,
		Postings: []PostingInput{
			{AccountCode: "EXP:DOES_NOT_EXIST", Direction: DirectionDebit, AmountMinor: 100},
			{AccountCode: AccountCashOffice, Direction: DirectionCredit, AmountMinor: 100},
		},
	})
	if !errors.Is(err, ErrUnknownAccount) {
		t.Fatalf("got err=%v, want ErrUnknownAccount", err)
	}

	// Inactive account fails closed.
	_, err = svc.PostEntry(ctx, JournalEntryInput{
		EntityID: entityID, SourceType: "MANUAL", SourceID: "m-inact", Currency: "KZT",
		IdempotencyKey: "inact-acct", OccurredAt: now,
		Postings: []PostingInput{
			{AccountCode: inactiveAcct, Direction: DirectionDebit, AmountMinor: 100},
			{AccountCode: AccountCashOffice, Direction: DirectionCredit, AmountMinor: 100},
		},
	})
	if !errors.Is(err, ErrInactiveAccount) {
		t.Fatalf("got err=%v, want ErrInactiveAccount", err)
	}
}

func TestSpannerGL_Reversal_ImmutabilityAndDoubleReverseGuard(t *testing.T) {
	client := openGLSpannerTestClient(t)
	ctx := context.Background()
	entityID := uniqueGLEntity("sup_rev")
	cleanupGLEntityState(t, client, entityID, nil)

	svc := NewService(client)
	origRes, err := svc.PostEntry(ctx, JournalEntryInput{
		EntityID: entityID, SourceType: "MANUAL", SourceID: "man-orig", Currency: "UZS",
		Memo: "Original manual adjustment", IdempotencyKey: "orig-1",
		Postings: []PostingInput{
			{AccountCode: AccountCashOffice, Direction: DirectionDebit, AmountMinor: 42000},
			{AccountCode: AccountSalesRevenue, Direction: DirectionCredit, AmountMinor: 42000},
		},
	})
	if err != nil {
		t.Fatalf("post orig: %v", err)
	}

	revRes, err := svc.ReverseEntry(ctx, entityID, origRes.EntryID, "rev-key-1", "Incorrect posting", "admin-1")
	if err != nil {
		t.Fatalf("ReverseEntry failed: %v", err)
	}
	if revRes.AlreadyPosted || revRes.PrevHash != origRes.Hash {
		t.Fatalf("unexpected reversal result: %+v", revRes)
	}

	// Original entry remains untouched.
	origRec, err := svc.GetEntry(ctx, entityID, origRes.EntryID)
	if err != nil || origRec.EntryHash != origRes.Hash || len(origRec.Postings) != 2 {
		t.Fatalf("original entry altered or unreadable: %+v (err=%v)", origRec, err)
	}

	// Reversal entry flips directions.
	revRec, err := svc.GetEntry(ctx, entityID, revRes.EntryID)
	if err != nil {
		t.Fatalf("GetEntry reversal: %v", err)
	}
	if revRec.ReversesEntryID != origRes.EntryID ||
		revRec.Postings[0].Direction != DirectionCredit ||
		revRec.Postings[1].Direction != DirectionDebit {
		t.Fatalf("reversal postings not inverted: %+v", revRec)
	}

	// Idempotent retry with same key succeeds cleanly.
	revDup, err := svc.ReverseEntry(ctx, entityID, origRes.EntryID, "rev-key-1", "Incorrect posting", "admin-1")
	if err != nil || !revDup.AlreadyPosted || revDup.EntryID != revRes.EntryID {
		t.Fatalf("reversal idempotent retry mismatch: %+v (err=%v)", revDup, err)
	}

	// Second reversal attempt with a DIFFERENT key must fail with ErrAlreadyReversed.
	if _, err := svc.ReverseEntry(ctx, entityID, origRes.EntryID, "rev-key-2", "Duplicate", "admin-2"); !errors.Is(err, ErrAlreadyReversed) {
		t.Fatalf("got err=%v, want ErrAlreadyReversed", err)
	}

	// Trial balance nets to zero on both accounts.
	tb, err := svc.TrialBalance(ctx, entityID, origRec.PeriodID)
	if err != nil {
		t.Fatalf("TrialBalance failed: %v", err)
	}
	for _, line := range tb.Accounts {
		if line.NetMinor != 0 {
			t.Fatalf("expected net 0 after reversal on %s, got %+v", line.AccountCode, line)
		}
	}

	// Verify OutboxEvents emitted EventGLEntryReversed on TopicLedger.
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT Payload FROM OutboxEvents WHERE SupplierId = @sid AND AggregateId = @aid`,
		Params: map[string]any{"sid": entityID, "aid": revRes.EntryID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		t.Fatalf("missing reversal outbox event: %v", err)
	}
	var payload []byte
	if err := row.Column(0, &payload); err != nil {
		t.Fatal(err)
	}
	var ev events.GLEntryReversedEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		t.Fatalf("unmarshal GLEntryReversedEvent: %v", err)
	}
	if ev.Type != events.EventGLEntryReversed || ev.ReversesEntryID != origRes.EntryID {
		t.Fatalf("unexpected GLEntryReversedEvent: %+v", ev)
	}
}

func TestSpannerGL_BackfillFromPaymentLedgerEntries_Idempotent(t *testing.T) {
	client := openGLSpannerTestClient(t)
	ctx := context.Background()
	entityID := uniqueGLEntity("sup_bf")
	pleIDs := []string{
		entityID + "_p0",
		entityID + "_p1",
		entityID + "_wh",
		entityID + "_init",
		entityID + "_orphan",
	}
	cleanupGLEntityState(t, client, entityID, pleIDs)

	now := time.Now().UTC().Add(-30 * time.Second).Truncate(time.Microsecond)
	refDouble := "jentry_bf_" + entityID
	refWebhook := "txn_bf_" + entityID
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("PaymentLedgerEntries", map[string]any{
			"LedgerEntryId": pleIDs[0], "OrderId": "ord-1", "SupplierId": entityID, "RetailerId": "ret-1",
			"Gateway": "GENERAL_LEDGER", "EntryType": "GL_DEBIT_AR_RETAILER_ret-1",
			"AmountMinor": int64(75000), "Currency": "UZS", "ReferenceId": refDouble,
			"Source": "ledger.double_entry", "OccurredAt": now, "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("PaymentLedgerEntries", map[string]any{
			"LedgerEntryId": pleIDs[1], "OrderId": "ord-1", "SupplierId": entityID, "RetailerId": "ret-1",
			"Gateway": "GENERAL_LEDGER", "EntryType": "GL_CREDIT_ESCROW_ORDER_ord-1",
			"AmountMinor": int64(75000), "Currency": "UZS", "ReferenceId": refDouble,
			"Source": "ledger.double_entry", "OccurredAt": now, "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("PaymentLedgerEntries", map[string]any{
			"LedgerEntryId": pleIDs[2], "OrderId": "ord-2", "SupplierId": entityID,
			"Gateway": "PAYME", "EntryType": "WEBHOOK_CAPTURED",
			"AmountMinor": int64(25000), "Currency": "UZS", "ReferenceId": refWebhook,
			"Source": "payment.webhook", "OccurredAt": now.Add(time.Second), "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("PaymentLedgerEntries", map[string]any{
			"LedgerEntryId": pleIDs[3], "OrderId": "ord-3", "SupplierId": entityID,
			"Gateway": "PAYME", "EntryType": "SESSION_INITIATED",
			"AmountMinor": int64(10000), "Currency": "UZS", "ReferenceId": "sess_" + entityID,
			"Source": "payment.session", "OccurredAt": now.Add(2 * time.Second), "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("PaymentLedgerEntries", map[string]any{
			"LedgerEntryId": pleIDs[4], "SupplierId": entityID,
			"Gateway": "GENERAL_LEDGER", "EntryType": "GL_DEBIT_CASH_OFFICE",
			"AmountMinor": int64(5000), "Currency": "UZS", "ReferenceId": "orphan_" + entityID,
			"Source": "ledger.double_entry", "OccurredAt": now.Add(3 * time.Second), "CreatedAt": spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		t.Fatalf("seed PaymentLedgerEntries: %v", err)
	}

	svc := NewService(client)
	rep1, err := svc.BackfillFromPaymentLedgerEntries(ctx, entityID, BackfillOptions{Limit: 100})
	if err != nil {
		t.Fatalf("Backfill run 1 failed: %v", err)
	}
	if rep1.RowsScanned != 5 || rep1.Posted != 2 || rep1.AlreadyPosted != 0 || rep1.SkippedUnbalanced != 1 || rep1.SkippedNonPosting != 1 {
		t.Fatalf("unexpected Backfill run 1 report: %+v", rep1)
	}

	// Idempotent second run: zero new rows posted.
	rep2, err := svc.BackfillFromPaymentLedgerEntries(ctx, entityID, BackfillOptions{Limit: 100})
	if err != nil {
		t.Fatalf("Backfill run 2 failed: %v", err)
	}
	if rep2.Posted != 0 || rep2.AlreadyPosted != 2 {
		t.Fatalf("unexpected Backfill run 2 report: %+v", rep2)
	}

	verify, err := VerifyEntityHashChain(ctx, client, entityID)
	if err != nil || !verify.Valid || verify.EntriesCount != 2 {
		t.Fatalf("hash chain invalid after backfill: %+v (err=%v)", verify, err)
	}
}

// Keep outbox import used for verifying SpannerTxnBuffer compatibility directly.
var _ outbox.TxnBuffer = (*outbox.SpannerTxnBuffer)(nil)

type countingOutboxBuf struct {
	events []outbox.Event
}

func (b *countingOutboxBuf) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func TestSpannerGL_ExternalOutboxBufferAndIdempotencyNoEmit(t *testing.T) {
	client := openGLSpannerTestClient(t)
	ctx := context.Background()
	entityID := uniqueGLEntity("sup_outbox")
	cleanupGLEntityState(t, client, entityID, nil)

	now := time.Now().UTC().Add(-10 * time.Second).Truncate(time.Microsecond)
	in := JournalEntryInput{
		EntityID:       entityID,
		SourceType:     "ORDER_DELIVERED",
		SourceID:       "ord-ob-1",
		OrderID:        "ord-ob-1",
		Currency:       "UZS",
		IdempotencyKey: "ob-idem-1",
		OccurredAt:     now,
		Postings: []PostingInput{
			{AccountCode: AccountCashInTransit, Direction: DirectionDebit, AmountMinor: 15000, PartyType: "DRIVER", PartyID: "drv-1"},
			{AccountCode: AccountSalesRevenue, Direction: DirectionCredit, AmountMinor: 15000},
		},
	}

	var buf1 countingOutboxBuf
	err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf1.events = nil
		res, err := BufferJournalEntryInTxn(ctx, txn, &buf1, in)
		if err != nil {
			return err
		}
		if res.AlreadyPosted {
			t.Fatalf("expected fresh post, got AlreadyPosted=true")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("first external outboxBuffer txn failed: %v", err)
	}
	if len(buf1.events) != 1 || buf1.events[0].SupplierID != entityID || buf1.events[0].TopicName != events.TopicLedger {
		t.Fatalf("unexpected external outbox events: %+v", buf1.events)
	}

	// Second txn with same IdempotencyKey must NOT emit any outbox event onto outboxBuf.
	var buf2 countingOutboxBuf
	err = spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf2.events = nil
		res, err := BufferJournalEntryInTxn(ctx, txn, &buf2, in)
		if err != nil {
			return err
		}
		if !res.AlreadyPosted {
			t.Fatalf("expected AlreadyPosted=true on replay")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("second external outboxBuffer txn failed: %v", err)
	}
	if len(buf2.events) != 0 {
		t.Fatalf("idempotent replay emitted %d outbox events, want 0", len(buf2.events))
	}
}

func TestSpannerGL_HashChainDetectsTamper(t *testing.T) {
	client := openGLSpannerTestClient(t)
	ctx := context.Background()
	entityID := uniqueGLEntity("sup_tamper")
	cleanupGLEntityState(t, client, entityID, nil)

	svc := NewService(client)
	res, err := svc.PostEvent(ctx, RuleEventFeePerOrder, RuleEvalContext{
		EntityID: entityID, SourceID: "ord-t1", Currency: "UZS", IdempotencyKey: "t-1", FeeMinor: 5000,
	})
	if err != nil {
		t.Fatalf("post entry: %v", err)
	}

	// Simulate an out-of-band DB tamper on Memo (test-only verification of cryptographic detection).
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.UpdateMap("JournalEntries", map[string]any{
			"EntityId": entityID,
			"EntryId":  res.EntryID,
			"Memo":     "TAMPERED_MEMO",
		}),
	}); err != nil {
		t.Fatalf("test tamper update: %v", err)
	}

	verify, err := VerifyEntityHashChain(ctx, client, entityID)
	if err != nil {
		t.Fatalf("VerifyEntityHashChain error: %v", err)
	}
	if verify.Valid || verify.BrokenEntryID != res.EntryID {
		t.Fatalf("expected tamper detection on %s, got %+v", res.EntryID, verify)
	}
}
