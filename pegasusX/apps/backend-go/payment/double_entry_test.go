package payment

import (
	"errors"
	"testing"
)

func TestDoubleEntry_BalancedSplitTenderPasses(t *testing.T) {
	// 100,000 UZS total = 60,000 Payme Card + 30,000 Cash + 10,000 Promo Discount
	tenders := []TenderSplit{
		{Method: "CARD", Gateway: "PAYME", AmountMinor: 60000, ReferenceID: "payme_tx_123"},
		{Method: "CASH", Gateway: "MANUAL", AmountMinor: 30000, ReferenceID: "cash_cod_123"},
	}
	discountMinor := int64(10000)
	totalOrderMinor := int64(100000)

	je, err := BuildSplitTenderJournalEntry("ord-split-1", "sup-1", "ret-1", "UZS", totalOrderMinor, tenders, discountMinor, "Split tender order checkout")
	if err != nil {
		t.Fatalf("expected balanced split tender to pass, got: %v", err)
	}

	if len(je.Postings) != 4 {
		t.Fatalf("expected 4 postings (2 tenders + 1 discount + 1 escrow), got %d", len(je.Postings))
	}

	records := je.ToLedgerEntryRecords()
	if len(records) != 4 {
		t.Fatalf("expected 4 ledger records, got %d", len(records))
	}
}

func TestDoubleEntry_UnbalancedSplitTenderFails(t *testing.T) {
	// Imbalance: total is 100,000 UZS, but tenders only sum to 99,999 UZS (1 minor off)
	tenders := []TenderSplit{
		{Method: "CARD", Gateway: "PAYME", AmountMinor: 60000},
		{Method: "CASH", Gateway: "MANUAL", AmountMinor: 29999},
	}
	discountMinor := int64(10000)
	totalOrderMinor := int64(100000)

	_, err := BuildSplitTenderJournalEntry("ord-split-2", "sup-1", "ret-1", "UZS", totalOrderMinor, tenders, discountMinor, "Unbalanced test")
	if err == nil {
		t.Fatal("expected error for unbalanced split tender, got nil")
	}
	if !errors.Is(err, ErrUnbalancedLedgerEntry) {
		t.Fatalf("expected ErrUnbalancedLedgerEntry, got %v", err)
	}
}

func TestDoubleEntry_ZeroOrNegativeAmountFails(t *testing.T) {
	je := JournalEntry{
		Currency: "UZS",
		Postings: []Posting{
			{AccountCode: "AR:RETAILER", Direction: PostingDebit, AmountMinor: 0, Currency: "UZS"},
			{AccountCode: "ESCROW:ORDER", Direction: PostingCredit, AmountMinor: 0, Currency: "UZS"},
		},
	}
	err := je.Validate()
	if !errors.Is(err, ErrZeroAmountPosting) {
		t.Fatalf("expected ErrZeroAmountPosting, got %v", err)
	}
}

func TestDoubleEntry_InsufficientPostingsFails(t *testing.T) {
	je := JournalEntry{
		Currency: "UZS",
		Postings: []Posting{
			{AccountCode: "AR:RETAILER", Direction: PostingDebit, AmountMinor: 5000, Currency: "UZS"},
		},
	}
	err := je.Validate()
	if !errors.Is(err, ErrInsufficientPostings) {
		t.Fatalf("expected ErrInsufficientPostings, got %v", err)
	}
}

func TestDoubleEntry_CurrencyMismatchFails(t *testing.T) {
	je := JournalEntry{
		Currency: "UZS",
		Postings: []Posting{
			{AccountCode: "AR:RETAILER", Direction: PostingDebit, AmountMinor: 5000, Currency: "USD"},
			{AccountCode: "ESCROW:ORDER", Direction: PostingCredit, AmountMinor: 5000, Currency: "UZS"},
		},
	}
	err := je.Validate()
	if !errors.Is(err, ErrCurrencyMismatchPosting) {
		t.Fatalf("expected ErrCurrencyMismatchPosting, got %v", err)
	}
}

func TestDoubleEntry_SettlementRelease(t *testing.T) {
	// Gross: 100,000 UZS, Platform Fee: 5,000 UZS, Net Supplier: 95,000 UZS
	je, err := BuildSettlementJournalEntry("ord-settle-1", "sup-1", "UZS", 100000, 5000)
	if err != nil {
		t.Fatalf("expected valid settlement, got: %v", err)
	}

	if len(je.Postings) != 3 {
		t.Fatalf("expected 3 postings, got %d", len(je.Postings))
	}

	records := je.ToLedgerEntryRecords()
	if len(records) != 3 {
		t.Fatalf("expected 3 ledger records, got %d", len(records))
	}

	// Invalid fee exceeding gross
	_, err = BuildSettlementJournalEntry("ord-settle-2", "sup-1", "UZS", 100000, 150000)
	if err == nil {
		t.Fatal("expected error for fee > gross, got nil")
	}
}
