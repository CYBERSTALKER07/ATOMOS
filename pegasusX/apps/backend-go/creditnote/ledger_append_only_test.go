package creditnote

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func TestCreditNoteSpannerRepo_NilGuards(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var nilRepo *SpannerRepository
	if err := nilRepo.SaveCreditNote(ctx, CreditNote{CreditNoteId: "cn_nil"}, ""); err == nil {
		t.Fatal("expected error from nil *SpannerRepository.SaveCreditNote")
	}
	if _, err := nilRepo.GetCreditNote(ctx, "cn_nil"); err == nil {
		t.Fatal("expected error from nil *SpannerRepository.GetCreditNote")
	}

	emptyRepo := &SpannerRepository{}
	if err := emptyRepo.SaveCreditNote(ctx, CreditNote{CreditNoteId: "cn_empty"}, ""); err == nil {
		t.Fatal("expected error from empty SpannerRepository.SaveCreditNote")
	}
}

func openCreditNoteSpannerTestClient(t *testing.T) *spanner.Client {
	t.Helper()

	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Spanner emulator creditnote test")
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

	_, probeErr := client.Single().ReadRow(ctx, "CreditNotes", spanner.Key{"__t8_cn_probe__"}, []string{"CreditNoteId"})
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("spanner CreditNotes unavailable (%s): %v", dbPath, probeErr)
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

func seedTestOrder(t *testing.T, ctx context.Context, client *spanner.Client, orderID, supplierID, retailerID string, now time.Time) {
	t.Helper()
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("Orders", map[string]any{
			"OrderId":            orderID,
			"SupplierId":         supplierID,
			"RetailerId":         retailerID,
			"Status":             "CONFIRMED",
			"OrderSource":        "RETAILER_APP",
			"ConfirmationStatus": "CONFIRMED",
			"LineItemsJson":      []byte(`[]`),
			"TotalMinor":         int64(100000),
			"Currency":           "UZS",
			"Version":            int64(1),
			"CreatedAt":          spanner.CommitTimestamp,
			"UpdatedAt":          spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		t.Fatalf("seed Orders(%s): %v", orderID, err)
	}
}

func cleanupCreditNoteSpannerState(t *testing.T, client *spanner.Client, cnIDs, orderIDs, ledgerIDs []string) {
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
		for _, id := range cnIDs {
			if id != "" {
				muts = append(muts, spanner.Delete("CreditNotes", spanner.Key{id}))
			}
		}
		for _, id := range orderIDs {
			if id != "" {
				muts = append(muts, spanner.Delete("Orders", spanner.Key{id}))
			}
		}
		_, _ = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			if len(orderIDs) > 0 {
				iter := txn.Query(ctx, spanner.Statement{
					SQL:    `SELECT EventId FROM OutboxEvents WHERE AggregateId IN UNNEST(@aggIds)`,
					Params: map[string]any{"aggIds": orderIDs},
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

func TestSpannerCreditNote_AppendOnlyLedgerAndIdempotentRetry(t *testing.T) {
	client := openCreditNoteSpannerTestClient(t)
	ctx := context.Background()
	repo := NewSpannerRepository(client)
	suffix := fmt.Sprintf("t8_%d", time.Now().UnixNano()%10_000_000_000)
	now := time.Now().UTC().Truncate(time.Microsecond)

	orderID := "ord_cn_" + suffix
	cnID := "cn_" + suffix
	refID := "jentry_cn_" + cnID
	p0 := refID + "_p0"
	p1 := refID + "_p1"
	cleanupCreditNoteSpannerState(t, client, []string{cnID}, []string{orderID}, []string{p0, p1})

	seedTestOrder(t, ctx, client, orderID, "sup_t8", "ret_t8", now)

	draft := CreditNote{
		CreditNoteId:    cnID,
		OrderId:         orderID,
		Type:            CreditNoteTypeFullReturn,
		Status:          CreditNoteStatusDraft,
		ReasonCode:      "DAMAGED",
		TotalNetMinor:   50000,
		TotalVatMinor:   0,
		TotalGrossMinor: 50000,
		CreatedBy:       "op_t8",
		CreatedAt:       now,
	}
	if err := repo.SaveCreditNote(ctx, draft, ""); err != nil {
		t.Fatalf("SaveCreditNote(DRAFT): %v", err)
	}
	if _, err := client.Single().ReadRow(ctx, "PaymentLedgerEntries", spanner.Key{p0}, []string{"LedgerEntryId"}); spanner.ErrCode(err) != codes.NotFound {
		t.Fatalf("DRAFT creditnote should not write ledger entry %s, got err=%v", p0, err)
	}

	issued := draft
	issued.Status = CreditNoteStatusFiscalPending
	issued.IssuedAt = &now
	if err := repo.SaveCreditNote(ctx, issued, ""); err != nil {
		t.Fatalf("SaveCreditNote(FISCAL_PENDING): %v", err)
	}
	row0, err := client.Single().ReadRow(ctx, "PaymentLedgerEntries", spanner.Key{p0}, []string{"AmountMinor", "EntryType"})
	if err != nil {
		t.Fatalf("read p0 after issue: %v", err)
	}
	var amt0 int64
	var et0 string
	if err := row0.Columns(&amt0, &et0); err != nil {
		t.Fatalf("scan p0: %v", err)
	}
	if amt0 != 50000 || et0 != "GL_DEBIT_ADJ_CREDITNOTE" {
		t.Fatalf("unexpected p0 row: amount=%d entryType=%q", amt0, et0)
	}

	// Retry SaveCreditNote while already FISCAL_PENDING must succeed without overwriting ledger rows.
	retryIssued := issued
	retryIssued.TotalGrossMinor = 88888
	if err := repo.SaveCreditNote(ctx, retryIssued, ""); err != nil {
		t.Fatalf("idempotent retry SaveCreditNote(FISCAL_PENDING): %v", err)
	}
	row0After, err := client.Single().ReadRow(ctx, "PaymentLedgerEntries", spanner.Key{p0}, []string{"AmountMinor"})
	if err != nil {
		t.Fatalf("read p0 after retry: %v", err)
	}
	var amtAfter int64
	if err := row0After.Column(0, &amtAfter); err != nil {
		t.Fatalf("scan p0 after retry: %v", err)
	}
	if amtAfter != 50000 {
		t.Fatalf("credit note retry overwrote ledger p0 amount: got %d, want 50000", amtAfter)
	}
}

func TestSpannerCreditNote_DirectLedgerCollisionFailsClosed(t *testing.T) {
	client := openCreditNoteSpannerTestClient(t)
	ctx := context.Background()
	repo := NewSpannerRepository(client)
	suffix := fmt.Sprintf("t8c_%d", time.Now().UnixNano()%10_000_000_000)
	now := time.Now().UTC().Truncate(time.Microsecond)

	orderID := "ord_cnc_" + suffix
	cnID := "cnc_" + suffix
	p0 := "jentry_cn_" + cnID + "_p0"
	p1 := "jentry_cn_" + cnID + "_p1"
	cleanupCreditNoteSpannerState(t, client, []string{cnID}, []string{orderID}, []string{p0, p1})

	seedTestOrder(t, ctx, client, orderID, "sup_t8", "ret_t8", now)

	draft := CreditNote{
		CreditNoteId:    cnID,
		OrderId:         orderID,
		Type:            CreditNoteTypeFullReturn,
		Status:          CreditNoteStatusDraft,
		ReasonCode:      "SHORT_DELIVERY",
		TotalNetMinor:   30000,
		TotalVatMinor:   0,
		TotalGrossMinor: 30000,
		CreatedBy:       "op_t8",
		CreatedAt:       now,
	}
	if err := repo.SaveCreditNote(ctx, draft, ""); err != nil {
		t.Fatalf("SaveCreditNote(DRAFT): %v", err)
	}

	// Pre-insert conflicting PaymentLedgerEntries row before FISCAL_PENDING transition.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("PaymentLedgerEntries", map[string]any{
			"LedgerEntryId": p0,
			"OrderId":       "ord_seeded_" + suffix,
			"SupplierId":    "sup_t8",
			"RetailerId":    "ret_t8",
			"Gateway":       "GENERAL_LEDGER",
			"EntryType":     "SEEDED_CN",
			"AmountMinor":   int64(11111),
			"Currency":      "UZS",
			"ReferenceId":   "seed_cn_" + suffix,
			"Source":        "t8.seed",
			"OccurredAt":    now,
			"CreatedAt":     now,
		}),
	}); err != nil {
		t.Fatalf("seed conflicting creditnote ledger row: %v", err)
	}

	issued := draft
	issued.Status = CreditNoteStatusFiscalPending
	issued.IssuedAt = &now
	if err := repo.SaveCreditNote(ctx, issued, ""); spanner.ErrCode(err) != codes.AlreadyExists {
		t.Fatalf("SaveCreditNote on direct LedgerEntryId collision err = %v (code=%v), want AlreadyExists", err, spanner.ErrCode(err))
	}

	row, err := client.Single().ReadRow(ctx, "PaymentLedgerEntries", spanner.Key{p0}, []string{"AmountMinor"})
	if err != nil {
		t.Fatalf("read colliding p0 row: %v", err)
	}
	var gotAmt int64
	if err := row.Column(0, &gotAmt); err != nil {
		t.Fatalf("scan p0 amount: %v", err)
	}
	if gotAmt != 11111 {
		t.Fatalf("colliding ledger row overwritten: got %d, want 11111", gotAmt)
	}
}
