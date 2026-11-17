package payment

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func assertInsertOnlyLedgerMutation(t *testing.T, m *spanner.Mutation) {
	t.Helper()
	if m == nil {
		t.Fatal("expected non-nil Spanner mutation")
	}
	v := reflect.ValueOf(m).Elem()
	tableField := v.FieldByName("table")
	opField := v.FieldByName("op")
	if !tableField.IsValid() || !opField.IsValid() {
		t.Fatalf("unexpected spanner.Mutation layout: tableValid=%v opValid=%v", tableField.IsValid(), opField.IsValid())
	}
	if gotTable := tableField.String(); gotTable != "PaymentLedgerEntries" {
		t.Fatalf("mutation table = %q, want %q", gotTable, "PaymentLedgerEntries")
	}

	insertRef := reflect.ValueOf(spanner.InsertMap("PaymentLedgerEntries", map[string]any{"k": "v"})).Elem().FieldByName("op")
	upsertRef := reflect.ValueOf(spanner.InsertOrUpdateMap("PaymentLedgerEntries", map[string]any{"k": "v"})).Elem().FieldByName("op")
	gotOp := fmt.Sprint(opField)
	if gotOp == fmt.Sprint(upsertRef) {
		t.Fatalf("PaymentLedgerEntries mutation must be insert-only, got insertOrUpdate (%s)", gotOp)
	}
	if gotOp != fmt.Sprint(insertRef) {
		t.Fatalf("PaymentLedgerEntries mutation op = %s, want insert (%s)", gotOp, fmt.Sprint(insertRef))
	}
}

func TestLedgerMutation_UsesAppendOnlyInsertOp(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 16, 1, 0, 0, 0, time.UTC)
	single := LedgerMutation(LedgerEntryRecord{
		LedgerEntryID: "pledger_unit_1",
		OrderID:       "ord_unit_1",
		SupplierID:    "sup_unit_1",
		RetailerID:    "ret_unit_1",
		Gateway:       "payme",
		EntryType:     "webhook_paid",
		AmountMinor:   150000,
		Currency:      "uzs",
		ReferenceID:   "tx_unit_1",
		Source:        "unit.test",
		OccurredAt:    now,
		CreatedAt:     now,
	})
	assertInsertOnlyLedgerMutation(t, single)

	jeSettle, err := BuildSettlementJournalEntry("batch_unit_1", "sup_unit_1", "UZS", 100000, 5000)
	if err != nil {
		t.Fatalf("BuildSettlementJournalEntry: %v", err)
	}
	settleMuts, err := JournalEntryMutations(jeSettle)
	if err != nil {
		t.Fatalf("JournalEntryMutations(settle): %v", err)
	}
	if len(settleMuts) == 0 {
		t.Fatal("expected non-empty settlement journal mutations")
	}
	for i, m := range settleMuts {
		t.Run(fmt.Sprintf("settle_leg_%d", i), func(t *testing.T) {
			assertInsertOnlyLedgerMutation(t, m)
		})
	}

	jeShortfall, err := BuildDoorstepCashShortfallJournalEntry("ord_unit_sf", "sup_unit_1", "drv_unit_1", "UZS", 100000, 75000, now)
	if err != nil {
		t.Fatalf("BuildDoorstepCashShortfallJournalEntry: %v", err)
	}
	sfMuts, err := JournalEntryMutations(jeShortfall)
	if err != nil {
		t.Fatalf("JournalEntryMutations(shortfall): %v", err)
	}
	if len(sfMuts) == 0 {
		t.Fatal("expected non-empty shortfall journal mutations")
	}
	for i, m := range sfMuts {
		t.Run(fmt.Sprintf("shortfall_leg_%d", i), func(t *testing.T) {
			assertInsertOnlyLedgerMutation(t, m)
		})
	}
}

func TestLedgerWriters_NilGuardsFailClosed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var nilRepo *SpannerRepository
	if err := nilRepo.SaveChargeback(ctx, ChargebackRecord{ChargebackID: "cb_nil"}, nil); err == nil {
		t.Fatal("expected error from nil *SpannerRepository.SaveChargeback")
	}

	emptyRepo := &SpannerRepository{}
	if err := emptyRepo.SaveChargeback(ctx, ChargebackRecord{ChargebackID: "cb_empty"}, nil); err == nil {
		t.Fatal("expected error from empty SpannerRepository.SaveChargeback")
	}
	if err := emptyRepo.SaveWebhook(ctx, WebhookRecord{WebhookID: "wh_empty"}, nil); err == nil {
		t.Fatal("expected error from empty SpannerRepository.SaveWebhook")
	}
	if err := emptyRepo.SaveReversal(ctx, ReversalRecord{ReversalID: "rev_empty"}, nil); err == nil {
		t.Fatal("expected error from empty SpannerRepository.SaveReversal")
	}
	if err := emptyRepo.CreateSession(ctx, SessionRecord{SessionID: "sess_empty"}, nil); err == nil {
		t.Fatal("expected error from empty SpannerRepository.CreateSession")
	}
	if err := WriteSettlementJournal(nil, "batch_nil", "sup_nil", "UZS", 10000, 500); err == nil {
		t.Fatal("expected error from WriteSettlementJournal(nil)")
	}
	if err := WriteDoorstepShortfall(ctx, nil, "ord_nil", "sup_nil", "drv_nil", "UZS", 10000, 5000, time.Now().UTC()); err == nil {
		t.Fatal("expected error from WriteDoorstepShortfall(nil)")
	}
}

func openLedgerSpannerTestClient(t *testing.T) *spanner.Client {
	t.Helper()

	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Spanner emulator ledger test")
	}

	project := envOrDefault("SPANNER_PROJECT", "pegasusx-local")
	instance := envOrDefault("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOrDefault("SPANNER_DATABASE", "pegasusx-db")
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
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

	_, probeErr := client.Single().ReadRow(ctx, "PaymentLedgerEntries", spanner.Key{"__t8_probe__"}, []string{"LedgerEntryId"})
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("spanner PaymentLedgerEntries unavailable (%s): %v", dbPath, probeErr)
		}
		t.Skipf("spanner emulator unreachable or unmigrated (%s): %v", dbPath, probeErr)
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

func uniqueT8Suffix(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("t8_%d", time.Now().UnixNano()%10_000_000_000)
}

func cleanupPaymentLedgerTestState(t *testing.T, client *spanner.Client, ledgerIDs, sessionIDs, chargebackIDs, reversalIDs, webhookIDs, outboxOrderIDs []string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var muts []*spanner.Mutation
		for _, id := range ledgerIDs {
			if id != "" {
				muts = append(muts, spanner.Delete("PaymentLedgerEntries", spanner.Key{id}))
			}
		}
		for _, id := range sessionIDs {
			if id != "" {
				muts = append(muts, spanner.Delete("PaymentSessions", spanner.Key{id}))
			}
		}
		for _, id := range chargebackIDs {
			if id != "" {
				muts = append(muts, spanner.Delete("PaymentChargebacks", spanner.Key{id}))
			}
		}
		for _, id := range reversalIDs {
			if id != "" {
				muts = append(muts, spanner.Delete("PaymentReversals", spanner.Key{id}))
			}
		}
		for _, id := range webhookIDs {
			if id != "" {
				muts = append(muts, spanner.Delete("PaymentWebhooks", spanner.Key{id}))
			}
		}
		_, _ = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			if len(outboxOrderIDs) > 0 {
				iter := txn.Query(ctx, spanner.Statement{
					SQL:    `SELECT EventId FROM OutboxEvents WHERE AggregateId IN UNNEST(@aggIds)`,
					Params: map[string]any{"aggIds": outboxOrderIDs},
				})
				defer iter.Stop()
				for {
					row, err := iter.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						break
					}
					var eventID string
					if err := row.Column(0, &eventID); err == nil && eventID != "" {
						muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eventID}))
					}
				}
			}
			if len(muts) == 0 {
				return nil
			}
			return txn.BufferWrite(muts)
		})
	})
}

func readLedgerAmountAndRef(t *testing.T, ctx context.Context, client *spanner.Client, entryID string) (int64, string) {
	t.Helper()
	row, err := client.Single().ReadRow(ctx, "PaymentLedgerEntries", spanner.Key{entryID}, []string{"AmountMinor", "ReferenceId"})
	if err != nil {
		t.Fatalf("ReadRow(%s): %v", entryID, err)
	}
	var amount int64
	var ref spanner.NullString
	if err := row.Columns(&amount, &ref); err != nil {
		t.Fatalf("Columns(%s): %v", entryID, err)
	}
	return amount, ref.StringVal
}

func countLedgerRowsByOrder(t *testing.T, ctx context.Context, client *spanner.Client, orderID string) int {
	t.Helper()
	var count int64
	err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		iter := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT COUNT(*) FROM PaymentLedgerEntries WHERE OrderId = @oid`,
			Params: map[string]any{"oid": orderID},
		})
		defer iter.Stop()
		row, err := iter.Next()
		if err != nil {
			return err
		}
		return row.Column(0, &count)
	})
	if err != nil {
		t.Fatalf("countLedgerRowsByOrder(%s): %v", orderID, err)
	}
	return int(count)
}

func TestSpannerLedger_DuplicatePKAndUniqueIndexFailClosed(t *testing.T) {
	client := openLedgerSpannerTestClient(t)
	ctx := context.Background()
	s := uniqueT8Suffix(t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	pkEntryID := "pk_" + s
	idxEntry1 := "idx1_" + s
	idxEntry2 := "idx2_" + s
	cleanupPaymentLedgerTestState(t, client, []string{pkEntryID, idxEntry1, idxEntry2}, nil, nil, nil, nil, nil)

	first := LedgerEntryRecord{
		LedgerEntryID: pkEntryID,
		OrderID:       "ord_" + s,
		SupplierID:    "sup_t8",
		Gateway:       "PAYME",
		EntryType:     "WEBHOOK_PAID",
		AmountMinor:   15000,
		Currency:      "UZS",
		ReferenceID:   "ref_pk_" + s,
		Source:        "t8.test",
		OccurredAt:    now,
		CreatedAt:     now,
	}
	if _, err := client.Apply(ctx, []*spanner.Mutation{LedgerMutation(first)}); err != nil {
		t.Fatalf("initial LedgerMutation apply: %v", err)
	}

	dupPK := first
	dupPK.AmountMinor = 99999
	dupPK.ReferenceID = "ref_pk_diff_" + s
	if _, err := client.Apply(ctx, []*spanner.Mutation{LedgerMutation(dupPK)}); spanner.ErrCode(err) != codes.AlreadyExists {
		t.Fatalf("duplicate PK LedgerMutation err = %v (code=%v), want AlreadyExists", err, spanner.ErrCode(err))
	}

	gotAmt, gotRef := readLedgerAmountAndRef(t, ctx, client, pkEntryID)
	if gotAmt != 15000 || gotRef != first.ReferenceID {
		t.Fatalf("original row mutated after collision: amount=%d ref=%q", gotAmt, gotRef)
	}

	idxRec1 := LedgerEntryRecord{
		LedgerEntryID: idxEntry1,
		OrderID:       "ord_idx_" + s,
		SupplierID:    "sup_t8",
		Gateway:       "CLICK",
		EntryType:     "WEBHOOK_PAID",
		AmountMinor:   25000,
		Currency:      "UZS",
		ReferenceID:   "ref_shared_" + s,
		Source:        "t8.test",
		OccurredAt:    now,
		CreatedAt:     now,
	}
	if _, err := client.Apply(ctx, []*spanner.Mutation{LedgerMutation(idxRec1)}); err != nil {
		t.Fatalf("initial unique index apply: %v", err)
	}
	idxRec2 := idxRec1
	idxRec2.LedgerEntryID = idxEntry2
	idxRec2.AmountMinor = 77777
	if _, err := client.Apply(ctx, []*spanner.Mutation{LedgerMutation(idxRec2)}); spanner.ErrCode(err) != codes.AlreadyExists {
		t.Fatalf("duplicate unique index LedgerMutation err = %v (code=%v), want AlreadyExists", err, spanner.ErrCode(err))
	}
}

func TestSpannerLedger_SettlementJournalCollisionFailsClosed(t *testing.T) {
	client := openLedgerSpannerTestClient(t)
	ctx := context.Background()
	s := uniqueT8Suffix(t)
	batchID := "b_" + s
	p0 := "jentry_settle_" + batchID + "_p0"
	p1 := "jentry_settle_" + batchID + "_p1"
	p2 := "jentry_settle_" + batchID + "_p2"
	cleanupPaymentLedgerTestState(t, client, []string{p0, p1, p2}, nil, nil, nil, nil, nil)

	err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return WriteSettlementJournal(txn, batchID, "sup_t8", "UZS", 100000, 5000)
	})
	if err != nil {
		t.Fatalf("first WriteSettlementJournal: %v", err)
	}

	err = spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return WriteSettlementJournal(txn, batchID, "sup_t8", "UZS", 250000, 9000)
	})
	if spanner.ErrCode(err) != codes.AlreadyExists {
		t.Fatalf("duplicate WriteSettlementJournal err = %v (code=%v), want AlreadyExists", err, spanner.ErrCode(err))
	}

	gotAmt0, _ := readLedgerAmountAndRef(t, ctx, client, p0)
	if gotAmt0 != 100000 {
		t.Fatalf("original settlement p0 overwritten: got %d, want 100000", gotAmt0)
	}
}

func TestSpannerLedger_SessionAndReversalCollisionsFailClosed(t *testing.T) {
	client := openLedgerSpannerTestClient(t)
	ctx := context.Background()
	repo := NewSpannerRepository(client)
	s := uniqueT8Suffix(t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	sessID := "sess_" + s
	revID := "rev_" + s
	sessLedgerID := "pledger_session_" + sessID
	revLedgerID := "pledger_reversal_" + revID
	cleanupPaymentLedgerTestState(t, client, []string{sessLedgerID, revLedgerID}, []string{sessID}, nil, []string{revID}, nil, nil)

	sess := SessionRecord{
		SessionID:   sessID,
		OrderID:     "ord_s_" + s,
		SupplierID:  "sup_t8",
		RetailerID:  "ret_t8",
		Gateway:     "PAYME",
		Currency:    "UZS",
		AmountMinor: 88000,
		Mode:        "ONLINE",
		Status:      "AWAITING_PAYMENT",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.CreateSession(ctx, sess, nil); err != nil {
		t.Fatalf("first CreateSession: %v", err)
	}
	dupSess := sess
	dupSess.AmountMinor = 999999
	if err := repo.CreateSession(ctx, dupSess, nil); spanner.ErrCode(err) != codes.AlreadyExists {
		t.Fatalf("duplicate CreateSession err = %v (code=%v), want AlreadyExists", err, spanner.ErrCode(err))
	}
	gotSessAmt, _ := readLedgerAmountAndRef(t, ctx, client, sessLedgerID)
	if gotSessAmt != 88000 {
		t.Fatalf("session ledger entry overwritten: got %d, want 88000", gotSessAmt)
	}

	rev := ReversalRecord{
		ReversalID:  revID,
		SessionID:   sessID,
		SupplierID:  "sup_t8",
		Gateway:     "PAYME",
		AmountMinor: 44000,
		Currency:    "UZS",
		CreatedAt:   now,
	}
	if err := repo.SaveReversal(ctx, rev, nil); err != nil {
		t.Fatalf("first SaveReversal: %v", err)
	}
	dupRev := rev
	dupRev.AmountMinor = 11111
	if err := repo.SaveReversal(ctx, dupRev, nil); spanner.ErrCode(err) != codes.AlreadyExists {
		t.Fatalf("duplicate SaveReversal err = %v (code=%v), want AlreadyExists", err, spanner.ErrCode(err))
	}
	gotRevAmt, _ := readLedgerAmountAndRef(t, ctx, client, revLedgerID)
	if gotRevAmt != 44000 {
		t.Fatalf("reversal ledger entry overwritten: got %d, want 44000", gotRevAmt)
	}
}

func TestSpannerLedger_SaveWebhookIdempotentRetryAndDirectPKCollision(t *testing.T) {
	client := openLedgerSpannerTestClient(t)
	ctx := context.Background()
	repo := NewSpannerRepository(client)
	s := uniqueT8Suffix(t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	wh1 := WebhookRecord{
		WebhookID:      "wh1_" + s,
		Gateway:        "PAYME",
		TransactionID:  "txn1_" + s,
		OrderID:        "ord_wh_" + s,
		SupplierID:     "sup_t8",
		RetailerID:     "ret_t8",
		Status:         "PAID",
		AmountMinor:    55000,
		Currency:       "UZS",
		ReceivedAt:     now,
		SignatureValid: true,
	}
	wh2 := WebhookRecord{
		WebhookID:      "wh2_" + s,
		Gateway:        "UZUM",
		TransactionID:  "txn2_" + s,
		OrderID:        "ord_wh2_" + s,
		SupplierID:     "sup_t8",
		RetailerID:     "ret_t8",
		Status:         "PAID",
		AmountMinor:    66000,
		Currency:       "UZS",
		ReceivedAt:     now,
		SignatureValid: true,
	}
	wh1Ledger := "pledger_webhook_" + wh1.WebhookID
	wh2Ledger := "pledger_webhook_" + wh2.WebhookID
	cleanupPaymentLedgerTestState(t, client, []string{wh1Ledger, wh2Ledger}, nil, nil, nil, []string{wh1.WebhookID, wh2.WebhookID}, []string{wh1.OrderID, wh2.OrderID})

	var emitCount atomic.Int32
	emit := func(buf outbox.TxnBuffer) error {
		emitCount.Add(1)
		return buf.BufferOutbox(ctx, outbox.Event{
			EventID:       "ev_wh_" + s,
			AggregateType: "PaymentWebhook",
			AggregateID:   wh1.OrderID,
			TopicName:     "payments.events",
			Payload:       []byte(`{"ok":true}`),
			CreatedAt:     now,
			SupplierID:    wh1.SupplierID,
		})
	}
	if err := repo.SaveWebhook(ctx, wh1, emit); err != nil {
		t.Fatalf("first SaveWebhook: %v", err)
	}
	if err := repo.SaveWebhook(ctx, wh1, emit); err != nil {
		t.Fatalf("idempotent retry SaveWebhook: %v", err)
	}
	if got := emitCount.Load(); got != 1 {
		t.Fatalf("webhook emit count after retry = %d, want 1", got)
	}

	// Pre-insert pledger_webhook_<wh2.WebhookID> with distinct (Gateway, EntryType, ReferenceId)
	// so pre-checks do not match and InsertMap must fail closed on LedgerEntryId PK collision.
	if _, err := client.Apply(ctx, []*spanner.Mutation{LedgerMutation(LedgerEntryRecord{
		LedgerEntryID: wh2Ledger,
		OrderID:       "ord_seeded_" + s,
		SupplierID:    "sup_t8",
		Gateway:       "CLICK",
		EntryType:     "SEEDED_ENTRY",
		AmountMinor:   12345,
		Currency:      "UZS",
		ReferenceID:   "seeded_ref_" + s,
		Source:        "t8.seed",
		OccurredAt:    now,
		CreatedAt:     now,
	})}); err != nil {
		t.Fatalf("seed colliding webhook ledger row: %v", err)
	}
	if err := repo.SaveWebhook(ctx, wh2, nil); spanner.ErrCode(err) != codes.AlreadyExists {
		t.Fatalf("SaveWebhook on direct LedgerEntryId collision err = %v (code=%v), want AlreadyExists", err, spanner.ErrCode(err))
	}
	gotAmt, _ := readLedgerAmountAndRef(t, ctx, client, wh2Ledger)
	if gotAmt != 12345 {
		t.Fatalf("seeded ledger entry overwritten on webhook PK collision: got %d, want 12345", gotAmt)
	}
}

func TestSpannerLedger_SaveChargebackIdempotentRetryAndDirectPKCollision(t *testing.T) {
	client := openLedgerSpannerTestClient(t)
	ctx := context.Background()
	repo := NewSpannerRepository(client)
	s := uniqueT8Suffix(t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	cb1 := ChargebackRecord{
		ChargebackID: "cb1_" + s,
		OrderID:      "ord_cb1_" + s,
		SupplierID:   "sup_t8",
		RetailerID:   "ret_t8",
		Gateway:      "LEDGER",
		AmountMinor:  42000,
		Currency:     "UZS",
		CreatedAt:    now,
		Source:       "claims.settle_chargeback",
	}
	cb2 := ChargebackRecord{
		ChargebackID: "cb2_" + s,
		OrderID:      "ord_cb2_" + s,
		SupplierID:   "sup_t8",
		RetailerID:   "ret_t8",
		Gateway:      "LEDGER",
		AmountMinor:  90000,
		Currency:     "UZS",
		CreatedAt:    now,
	}
	cb1Ledger := "pledger_chargeback_" + cb1.ChargebackID
	cb1J0 := "jentry_claim_" + cb1.ChargebackID + "_p0"
	cb1J1 := "jentry_claim_" + cb1.ChargebackID + "_p1"
	cb2Ledger := "pledger_chargeback_" + cb2.ChargebackID
	cleanupPaymentLedgerTestState(t, client, []string{cb1Ledger, cb1J0, cb1J1, cb2Ledger}, nil, []string{cb1.ChargebackID, cb2.ChargebackID}, nil, nil, []string{cb1.OrderID})

	var emitCount atomic.Int32
	emit := func(buf outbox.TxnBuffer) error {
		emitCount.Add(1)
		return buf.BufferOutbox(ctx, outbox.Event{
			EventID:       "ev_cb_" + s,
			AggregateType: "PaymentChargeback",
			AggregateID:   cb1.OrderID,
			TopicName:     "payments.events",
			Payload:       []byte(`{"cb":true}`),
			CreatedAt:     now,
			SupplierID:    cb1.SupplierID,
		})
	}
	if err := repo.SaveChargeback(ctx, cb1, emit); err != nil {
		t.Fatalf("first SaveChargeback: %v", err)
	}
	if err := repo.SaveChargeback(ctx, cb1, emit); err != nil {
		t.Fatalf("idempotent retry SaveChargeback: %v", err)
	}
	if got := emitCount.Load(); got != 1 {
		t.Fatalf("chargeback emit count after retry = %d, want 1", got)
	}
	if rows := countLedgerRowsByOrder(t, ctx, client, cb1.OrderID); rows != 3 {
		t.Fatalf("chargeback ledger rows = %d, want 3", rows)
	}

	// Pre-insert pledger_chargeback_<cb2.ChargebackID> directly without PaymentChargebacks row.
	if _, err := client.Apply(ctx, []*spanner.Mutation{LedgerMutation(LedgerEntryRecord{
		LedgerEntryID: cb2Ledger,
		OrderID:       "ord_cb2_seed_" + s,
		SupplierID:    "sup_t8",
		Gateway:       "PAYME",
		EntryType:     "SEEDED_CB",
		AmountMinor:   22222,
		Currency:      "UZS",
		ReferenceID:   "seed_cb_" + s,
		Source:        "t8.seed",
		OccurredAt:    now,
		CreatedAt:     now,
	})}); err != nil {
		t.Fatalf("seed colliding chargeback ledger row: %v", err)
	}
	if err := repo.SaveChargeback(ctx, cb2, nil); spanner.ErrCode(err) != codes.AlreadyExists {
		t.Fatalf("SaveChargeback on direct LedgerEntryId collision err = %v (code=%v), want AlreadyExists", err, spanner.ErrCode(err))
	}
	gotAmt, _ := readLedgerAmountAndRef(t, ctx, client, cb2Ledger)
	if gotAmt != 22222 {
		t.Fatalf("seeded chargeback ledger entry overwritten: got %d, want 22222", gotAmt)
	}
}

func TestSpannerLedger_WriteDoorstepShortfallIdempotentAndCollision(t *testing.T) {
	client := openLedgerSpannerTestClient(t)
	ctx := context.Background()
	s := uniqueT8Suffix(t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	ord1 := "ord_sf1_" + s
	ord2 := "ord_sf2_" + s
	sf1P0 := "jentry_shortfall_" + ord1 + "_p0"
	sf1P1 := "jentry_shortfall_" + ord1 + "_p1"
	sf1P2 := "jentry_shortfall_" + ord1 + "_p2"
	sf2P0 := "jentry_shortfall_" + ord2 + "_p0"
	sf2P1 := "jentry_shortfall_" + ord2 + "_p1"
	sf2P2 := "jentry_shortfall_" + ord2 + "_p2"
	cleanupPaymentLedgerTestState(t, client, []string{sf1P0, sf1P1, sf1P2, sf2P0, sf2P1, sf2P2}, nil, nil, nil, nil, nil)

	if err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return WriteDoorstepShortfall(ctx, txn, ord1, "sup_t8", "drv_t8", "UZS", 100000, 80000, now)
	}); err != nil {
		t.Fatalf("first WriteDoorstepShortfall: %v", err)
	}

	if err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return WriteDoorstepShortfall(ctx, txn, ord1, "sup_t8", "drv_t8", "UZS", 100000, 80000, now)
	}); err != nil {
		t.Fatalf("idempotent retry WriteDoorstepShortfall: %v", err)
	}
	if rows := countLedgerRowsByOrder(t, ctx, client, ord1); rows != 3 {
		t.Fatalf("doorstep shortfall ledger rows = %d, want 3", rows)
	}

	// Pre-insert jentry_shortfall_<ord2>_p0 under a different OrderId so pre-check misses and InsertMap must fail closed.
	if _, err := client.Apply(ctx, []*spanner.Mutation{LedgerMutation(LedgerEntryRecord{
		LedgerEntryID: sf2P0,
		OrderID:       "ord_other_" + s,
		SupplierID:    "sup_t8",
		Gateway:       "CASH",
		EntryType:     "SEEDED_SF",
		AmountMinor:   33333,
		Currency:      "UZS",
		ReferenceID:   "seed_sf_" + s,
		Source:        "t8.seed",
		OccurredAt:    now,
		CreatedAt:     now,
	})}); err != nil {
		t.Fatalf("seed colliding shortfall ledger row: %v", err)
	}

	err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return WriteDoorstepShortfall(ctx, txn, ord2, "sup_t8", "drv_t8", "UZS", 95000, 70000, now)
	})
	if spanner.ErrCode(err) != codes.AlreadyExists {
		t.Fatalf("WriteDoorstepShortfall on direct LedgerEntryId collision err = %v (code=%v), want AlreadyExists", err, spanner.ErrCode(err))
	}
	gotAmt, _ := readLedgerAmountAndRef(t, ctx, client, sf2P0)
	if gotAmt != 33333 {
		t.Fatalf("seeded shortfall ledger row overwritten: got %d, want 33333", gotAmt)
	}
}
