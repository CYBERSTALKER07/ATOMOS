package payment

import (
	"errors"
	"testing"
	"time"
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

func TestDoubleEntry_DeterministicIdempotencyKeys(t *testing.T) {
	tenders := []TenderSplit{
		{Method: "CARD", Gateway: "PAYME", AmountMinor: 60000, ReferenceID: "payme_tx_123"},
		{Method: "CASH", Gateway: "MANUAL", AmountMinor: 40000, ReferenceID: "cash_cod_123"},
	}

	// Repeated invocations for the exact same order must produce IDENTICAL EntryID and ReferenceID
	je1, err := BuildSplitTenderJournalEntry("ord-repeat-99", "sup-1", "ret-1", "UZS", 100000, tenders, 0, "Test checkout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	je2, err := BuildSplitTenderJournalEntry("ord-repeat-99", "sup-1", "ret-1", "UZS", 100000, tenders, 0, "Test checkout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if je1.EntryID != je2.EntryID {
		t.Fatalf("expected deterministic EntryID: got %s vs %s", je1.EntryID, je2.EntryID)
	}
	if je1.EntryID != "jentry_order_ord-repeat-99" {
		t.Fatalf("expected EntryID 'jentry_order_ord-repeat-99', got %s", je1.EntryID)
	}

	recs1 := je1.ToLedgerEntryRecords()
	recs2 := je2.ToLedgerEntryRecords()

	if len(recs1) != len(recs2) {
		t.Fatalf("record lengths differ: %d vs %d", len(recs1), len(recs2))
	}

	for i := range recs1 {
		if recs1[i].LedgerEntryID != recs2[i].LedgerEntryID {
			t.Errorf("posting %d LedgerEntryID mismatch: %s vs %s", i, recs1[i].LedgerEntryID, recs2[i].LedgerEntryID)
		}
		if recs1[i].ReferenceID != recs2[i].ReferenceID {
			t.Errorf("posting %d ReferenceID mismatch: %s vs %s", i, recs1[i].ReferenceID, recs2[i].ReferenceID)
		}
		if recs1[i].ReferenceID != "jentry_order_ord-repeat-99" {
			t.Errorf("posting %d ReferenceID unexpected: %s", i, recs1[i].ReferenceID)
		}
	}

	// Also verify settlement journal entry determinism
	s1, err := BuildSettlementJournalEntry("ord-settle-det-1", "sup-1", "UZS", 100000, 5000)
	if err != nil {
		t.Fatalf("unexpected settlement error: %v", err)
	}
	s2, err := BuildSettlementJournalEntry("ord-settle-det-1", "sup-1", "UZS", 100000, 5000)
	if err != nil {
		t.Fatalf("unexpected settlement error: %v", err)
	}

	if s1.EntryID != s2.EntryID || s1.EntryID != "jentry_settle_ord-settle-det-1" {
		t.Fatalf("expected deterministic settlement EntryID: %s vs %s", s1.EntryID, s2.EntryID)
	}

	sRecs1 := s1.ToLedgerEntryRecords()
	sRecs2 := s2.ToLedgerEntryRecords()
	for i := range sRecs1 {
		if sRecs1[i].LedgerEntryID != sRecs2[i].LedgerEntryID {
			t.Errorf("settle posting %d LedgerEntryID mismatch", i)
		}
		if sRecs1[i].ReferenceID != sRecs2[i].ReferenceID || sRecs1[i].ReferenceID != "jentry_settle_ord-settle-det-1" {
			t.Errorf("settle posting %d ReferenceID mismatch: %s", i, sRecs1[i].ReferenceID)
		}
	}
}

func TestDoubleEntry_CashVarianceShortfall(t *testing.T) {
	// Driver expected to collect 100,000 UZS, declared 80,000 UZS -> Shortfall 20,000 UZS
	je, err := BuildCashVarianceJournalEntry("recon-short-1", "sup-1", "driver-1", "route-1", "UZS", 100000, 80000, time.Time{}, "Driver cash shortfall")
	if err != nil {
		t.Fatalf("expected valid cash variance shortfall, got: %v", err)
	}

	if err := je.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if len(je.Postings) != 2 {
		t.Fatalf("expected 2 postings, got %d", len(je.Postings))
	}

	// Posting 0: Debit AR:DRIVER:VARIANCE:driver-1 for 20000
	if je.Postings[0].Direction != PostingDebit || je.Postings[0].AccountCode != "AR:DRIVER:VARIANCE:driver-1" || je.Postings[0].AmountMinor != 20000 {
		t.Errorf("unexpected debit posting: %+v", je.Postings[0])
	}

	// Posting 1: Credit CASH:DRIVER:driver-1 for 20000
	if je.Postings[1].Direction != PostingCredit || je.Postings[1].AccountCode != "CASH:DRIVER:driver-1" || je.Postings[1].AmountMinor != 20000 {
		t.Errorf("unexpected credit posting: %+v", je.Postings[1])
	}

	muts, err := JournalEntryMutations(je)
	if err != nil {
		t.Fatalf("mutations failed: %v", err)
	}
	if len(muts) != 2 {
		t.Fatalf("expected 2 spanner mutations, got %d", len(muts))
	}
}

func TestDoubleEntry_CashVarianceOverage(t *testing.T) {
	// Driver expected to collect 100,000 UZS, declared 115,000 UZS -> Overage 15,000 UZS
	je, err := BuildCashVarianceJournalEntry("recon-over-1", "sup-1", "driver-2", "route-1", "UZS", 100000, 115000, time.Time{}, "Driver cash overage")
	if err != nil {
		t.Fatalf("expected valid cash variance overage, got: %v", err)
	}

	if err := je.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// Posting 0: Debit CASH:DRIVER:driver-2 for 15000
	if je.Postings[0].Direction != PostingDebit || je.Postings[0].AccountCode != "CASH:DRIVER:driver-2" || je.Postings[0].AmountMinor != 15000 {
		t.Errorf("unexpected debit posting: %+v", je.Postings[0])
	}

	// Posting 1: Credit AP:DRIVER:VARIANCE:driver-2 for 15000
	if je.Postings[1].Direction != PostingCredit || je.Postings[1].AccountCode != "AP:DRIVER:VARIANCE:driver-2" || je.Postings[1].AmountMinor != 15000 {
		t.Errorf("unexpected credit posting: %+v", je.Postings[1])
	}
}

func TestDoubleEntry_CashResolutionAcceptAndWriteOff(t *testing.T) {
	// Resolve shortfall with ACCEPT (driver repays cash into office vault)
	jeAccept, err := BuildCashResolutionJournalEntry("recon-res-1", "sup-1", "driver-1", "ACCEPT", "UZS", -20000, time.Time{}, "Driver repaid shortfall")
	if err != nil {
		t.Fatalf("expected valid accept, got: %v", err)
	}
	if err := jeAccept.Validate(); err != nil {
		t.Fatalf("accept validation failed: %v", err)
	}
	if jeAccept.Postings[0].AccountCode != "CASH:OFFICE" || jeAccept.Postings[1].AccountCode != "AR:DRIVER:VARIANCE:driver-1" {
		t.Errorf("unexpected accept accounts: %+v, %+v", jeAccept.Postings[0], jeAccept.Postings[1])
	}

	// Resolve shortfall with WRITE_OFF (company absorbs loss)
	jeWriteOff, err := BuildCashResolutionJournalEntry("recon-res-2", "sup-1", "driver-1", "WRITE_OFF", "UZS", -20000, time.Time{}, "Finance write-off")
	if err != nil {
		t.Fatalf("expected valid write-off, got: %v", err)
	}
	if err := jeWriteOff.Validate(); err != nil {
		t.Fatalf("write-off validation failed: %v", err)
	}
	if jeWriteOff.Postings[0].AccountCode != "EXP:CASH:WRITEOFF" || jeWriteOff.Postings[1].AccountCode != "AR:DRIVER:VARIANCE:driver-1" {
		t.Errorf("unexpected write-off accounts: %+v, %+v", jeWriteOff.Postings[0], jeWriteOff.Postings[1])
	}

	// Resolve overage with ACCEPT (recognized as misc revenue)
	jeOverage, err := BuildCashResolutionJournalEntry("recon-res-3", "sup-1", "driver-2", "ACCEPT", "UZS", 15000, time.Time{}, "Recognize surplus")
	if err != nil {
		t.Fatalf("expected valid overage resolution, got: %v", err)
	}
	if err := jeOverage.Validate(); err != nil {
		t.Fatalf("overage resolution validation failed: %v", err)
	}
	if jeOverage.Postings[0].AccountCode != "AP:DRIVER:VARIANCE:driver-2" || jeOverage.Postings[1].AccountCode != "REV:CASH:OVERAGE" {
		t.Errorf("unexpected overage resolution accounts: %+v, %+v", jeOverage.Postings[0], jeOverage.Postings[1])
	}
}

func TestDoubleEntry_DoorstepCashShortfall(t *testing.T) {
	// Order total is 100,000 UZS, driver collected 90,000 UZS, shortfall 10,000 UZS
	je, err := BuildDoorstepCashShortfallJournalEntry("ord-doorstep-1", "sup-1", "driver-1", "UZS", 100000, 90000, time.Time{})
	if err != nil {
		t.Fatalf("expected valid doorstep shortfall, got: %v", err)
	}
	if err := je.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if len(je.Postings) != 3 {
		t.Fatalf("expected 3 postings, got %d", len(je.Postings))
	}
	// Debits: CASH:DRIVER (90000) + AR:DRIVER:VARIANCE (10000) = 100000
	// Credit: ESCROW:ORDER (100000)
	var sumDebit, sumCredit int64
	for _, p := range je.Postings {
		if p.Direction == PostingDebit {
			sumDebit += p.AmountMinor
		} else {
			sumCredit += p.AmountMinor
		}
	}
	if sumDebit != 100000 || sumCredit != 100000 {
		t.Errorf("debits (%d) != credits (%d)", sumDebit, sumCredit)
	}
}

func TestDoubleEntry_ClaimDisputeSettlements(t *testing.T) {
	// 1) Store credit settlement mode
	jeStoreCredit, err := BuildClaimDisputeJournalEntry("clm-1", "ord-1", "sup-1", "ret-1", "UZS", "STORE_CREDIT", 45000, time.Time{}, "Damaged items")
	if err != nil {
		t.Fatalf("expected valid store credit claim, got: %v", err)
	}
	if err := jeStoreCredit.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	if jeStoreCredit.Postings[0].AccountCode != "EXP:CLAIMS:SETTLEMENT" || jeStoreCredit.Postings[1].AccountCode != "WALLET:RETAILER:ret-1" {
		t.Errorf("unexpected accounts for store credit: %+v", jeStoreCredit.Postings)
	}

	// 2) Gateway refund settlement mode
	jeRefund, err := BuildClaimDisputeJournalEntry("clm-2", "ord-2", "sup-1", "ret-1", "UZS", "GATEWAY_REFUND", 30000, time.Time{}, "Missing items")
	if err != nil {
		t.Fatalf("expected valid gateway refund claim, got: %v", err)
	}
	if err := jeRefund.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	if jeRefund.Postings[0].AccountCode != "AP:SUPPLIER:sup-1" || jeRefund.Postings[1].AccountCode != "PSP:GATEWAY" {
		t.Errorf("unexpected accounts for gateway refund: %+v", jeRefund.Postings)
	}

	// 3) Ledger only settlement mode
	jeLedger, err := BuildClaimDisputeJournalEntry("clm-3", "ord-3", "sup-1", "ret-1", "UZS", "LEDGER_ONLY", 25000, time.Time{}, "Expired goods")
	if err != nil {
		t.Fatalf("expected valid ledger claim, got: %v", err)
	}
	if err := jeLedger.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	if jeLedger.Postings[0].AccountCode != "AP:SUPPLIER:sup-1" || jeLedger.Postings[1].AccountCode != "AR:RETAILER:ret-1" {
		t.Errorf("unexpected accounts for ledger claim: %+v", jeLedger.Postings)
	}
}

func TestDoubleEntry_CreditNoteAdjustment(t *testing.T) {
	je, err := BuildCreditNoteJournalEntry("cn-123", "ord-555", "sup-1", "ret-1", "UZS", 50000, time.Time{}, "Partial return credit note")
	if err != nil {
		t.Fatalf("expected valid credit note, got: %v", err)
	}
	if err := je.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	if len(je.Postings) != 2 {
		t.Fatalf("expected 2 postings, got %d", len(je.Postings))
	}
	if je.Postings[0].AccountCode != "ADJ:CREDITNOTE:sup-1" || je.Postings[0].Direction != PostingDebit {
		t.Errorf("unexpected debit posting: %+v", je.Postings[0])
	}
	if je.Postings[1].AccountCode != "AR:RETAILER:ret-1" || je.Postings[1].Direction != PostingCredit {
		t.Errorf("unexpected credit posting: %+v", je.Postings[1])
	}

	muts, err := JournalEntryMutations(je)
	if err != nil {
		t.Fatalf("JournalEntryMutations failed: %v", err)
	}
	if len(muts) != 2 {
		t.Fatalf("expected 2 mutations, got %d", len(muts))
	}
}

