package gl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestDefaultChartOfAccounts_ContainsAll30SystemAccounts(t *testing.T) {
	expected := []string{
		// 17 from payment/double_entry.go
		"AR:RETAILER",
		"CASH:DRIVER",
		"AR:DRIVER:VARIANCE",
		"CASH:OFFICE",
		"PSP:GATEWAY",
		"BANK:OPERATING",
		"AP:SUPPLIER",
		"ESCROW:ORDER",
		"WALLET:RETAILER",
		"AP:DRIVER:VARIANCE",
		"EQUITY:OPENING_BALANCE",
		"REV:PLATFORM:FEE",
		"EXP:PROMO:DISCOUNT",
		"EXP:CASH:WRITEOFF",
		"REV:CASH:OVERAGE",
		"EXP:CLAIMS:SETTLEMENT",
		"ADJ:CREDITNOTE",
		// 10 from ATOMOS Plan §6.1
		"REV:SALES",
		"TAX:VAT:OUTPUT",
		"TAX:VAT:INPUT",
		"COGS",
		"INVENTORY",
		"AP:VENDOR",
		"GRNI",
		"EXP:SHRINK",
		"AR:SUPPLIER:FEES",
		"FX:ROUNDING",
		// 3 from Wave 4A §6.4 Multi-Entity & FX Revaluation
		"FX:UNREALIZED_GAIN_LOSS",
		"IC:DUE_FROM",
		"IC:DUE_TO",
	}

	coaUZ := DefaultChartOfAccounts("UZ", "")
	if len(coaUZ) != 30 {
		t.Fatalf("DefaultChartOfAccounts(UZ) count = %d, want 30", len(coaUZ))
	}
	byCode := make(map[string]AccountDef, len(coaUZ))
	for _, a := range coaUZ {
		if a.Currency != "UZS" {
			t.Fatalf("account %s currency = %q, want UZS", a.AccountCode, a.Currency)
		}
		if a.CountryPack != "UZ" {
			t.Fatalf("account %s country_pack = %q, want UZ", a.AccountCode, a.CountryPack)
		}
		byCode[a.AccountCode] = a
	}
	for _, code := range expected {
		if _, ok := byCode[code]; !ok {
			t.Fatalf("missing required COA account %q", code)
		}
		if _, ok := LookupSystemAccount(code); !ok {
			t.Fatalf("LookupSystemAccount(%q) = false, want true", code)
		}
	}

	coaKZ := DefaultChartOfAccounts("KZ", "")
	if len(coaKZ) != 30 {
		t.Fatalf("DefaultChartOfAccounts(KZ) count = %d, want 30", len(coaKZ))
	}
	if coaKZ[0].Currency != "KZT" || coaKZ[0].CountryPack != "KZ" {
		t.Fatalf("KZ COA defaults got (%s, %s), want (KZT, KZ)", coaKZ[0].Currency, coaKZ[0].CountryPack)
	}
}

func TestValidateJournalInput_StrictInvariantsAndEntityTimezonePeriod(t *testing.T) {
	ctx := context.Background()

	// 2026-10-31 20:30:00 UTC == 2026-11-01 01:30:00 Asia/Tashkent (UTC+5) -> period "2026-11"
	occurredAt := time.Date(2026, 10, 31, 20, 30, 0, 123456789, time.UTC)
	validIn := JournalEntryInput{
		EntityID:       "sup-1",
		SourceType:     "MANUAL",
		SourceID:       "man-1",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: "idem-1",
		OccurredAt:     occurredAt,
		Postings: []PostingInput{
			{AccountCode: "AR:RETAILER", Direction: DirectionDebit, AmountMinor: 112000, PartyType: "RETAILER", PartyID: "ret-1"},
			{AccountCode: "REV:SALES", Direction: DirectionCredit, AmountMinor: 100000},
			{AccountCode: "TAX:VAT:OUTPUT", Direction: DirectionCredit, AmountMinor: 12000},
		},
	}
	td, tc, err := ValidateJournalInput(ctx, &validIn)
	if err != nil {
		t.Fatalf("valid entry failed: %v", err)
	}
	if td != 112000 || tc != 112000 {
		t.Fatalf("totals = (%d, %d), want (112000, 112000)", td, tc)
	}
	if validIn.PeriodID != "2026-11" {
		t.Fatalf("period_id = %q, want 2026-11 in Asia/Tashkent", validIn.PeriodID)
	}
	if validIn.OccurredAt.Nanosecond()%1000 != 0 {
		t.Fatalf("occurred_at sub-microsecond ns=%d, want microsecond-truncated", validIn.OccurredAt.Nanosecond())
	}

	// Verify KZ timezone (Asia/Almaty UTC+5 since 2024): 2026-01-31 20:00Z -> 2026-02
	kzCtx := auth.WithClaims(ctx, auth.Claims{SupplierID: "sup-kz", MarketCode: "KZ"})
	kzPeriod := ComputePeriodID(kzCtx, "sup-kz", "KZ", time.Date(2026, 1, 31, 20, 0, 0, 0, time.UTC))
	if kzPeriod != "2026-02" {
		t.Fatalf("KZ period_id = %q, want 2026-02", kzPeriod)
	}

	cases := []struct {
		name string
		mut  func(in *JournalEntryInput)
		want error
	}{
		{
			name: "fewer_than_2_postings",
			mut:  func(in *JournalEntryInput) { in.Postings = in.Postings[:1] },
			want: ErrInsufficientPostings,
		},
		{
			name: "zero_amount",
			mut:  func(in *JournalEntryInput) { in.Postings[2].AmountMinor = 0 },
			want: ErrInvalidPostingAmount,
		},
		{
			name: "negative_amount",
			mut:  func(in *JournalEntryInput) { in.Postings[0].AmountMinor = -100 },
			want: ErrInvalidPostingAmount,
		},
		{
			name: "unbalanced_txn_amounts",
			mut:  func(in *JournalEntryInput) { in.Postings[2].AmountMinor = 11999 },
			want: ErrUnbalancedEntry,
		},
		{
			name: "unbalanced_functional_amounts",
			mut: func(in *JournalEntryInput) {
				in.Currency = "USD"
				in.Postings = []PostingInput{
					{AccountCode: "AR:RETAILER", Direction: DirectionDebit, AmountMinor: 100, Currency: "USD", FunctionalAmountMinor: 1280000, FxRateID: "fx-1"},
					{AccountCode: "REV:SALES", Direction: DirectionCredit, AmountMinor: 100, Currency: "USD", FunctionalAmountMinor: 1279999, FxRateID: "fx-1"},
				}
			},
			want: ErrUnbalancedEntry,
		},
		{
			name: "currency_mismatch",
			mut:  func(in *JournalEntryInput) { in.Postings[1].Currency = "USD" },
			want: ErrCurrencyMismatch,
		},
		{
			name: "invalid_direction",
			mut:  func(in *JournalEntryInput) { in.Postings[0].Direction = "SIDEWAYS" },
			want: ErrInvalidPostingDirection,
		},
	}

	for _, tcCase := range cases {
		t.Run(tcCase.name, func(t *testing.T) {
			cp := validIn
			cp.Postings = append([]PostingInput(nil), validIn.Postings...)
			tcCase.mut(&cp)
			if _, _, err := ValidateJournalInput(ctx, &cp); !errors.Is(err, tcCase.want) {
				t.Fatalf("got err=%v, want errors.Is(_, %v)", err, tcCase.want)
			}
		})
	}
}

func TestCanonicalHashChain_Determinism(t *testing.T) {
	p1 := CanonicalEntryPayload{
		EntityID:       "sup-1",
		EntryID:        "gle-1",
		PeriodID:       "2026-11",
		SourceType:     "ORDER_DELIVERED",
		SourceID:       "ord-1",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: "k1",
		PostedBy:       "system",
		OccurredAt:     "2026-11-16T10:00:00Z",
		Postings: []CanonicalPosting{
			{Line: 2, AccountCode: "REV:SALES", Direction: "CREDIT", AmountMinor: 1000, Currency: "UZS", FunctionalAmountMinor: 1000},
			{Line: 1, AccountCode: "CASH:DRIVER", Direction: "DEBIT", AmountMinor: 1000, Currency: "UZS", FunctionalAmountMinor: 1000},
		},
	}
	p2 := p1
	p2.Postings = []CanonicalPosting{p1.Postings[1], p1.Postings[0]}

	b1, err := CanonicalEntryBytes(p1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := CanonicalEntryBytes(p2)
	if err != nil {
		t.Fatal(err)
	}
	if string(b1) != string(b2) {
		t.Fatalf("canonical json must sort by Line: %s != %s", string(b1), string(b2))
	}

	gotHash := ComputeEntryHash(GenesisHash, b1)
	raw := sha256.Sum256(append([]byte(GenesisHash), b1...))
	wantHash := hex.EncodeToString(raw[:])
	if gotHash != wantHash || len(gotHash) != 64 {
		t.Fatalf("hash mismatch: got=%s want=%s", gotHash, wantHash)
	}
}

func TestPostingRules_AllM02EventsBalanced(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		build     func() (JournalEntryInput, error)
		wantLines int
		wantTotal int64
	}{
		{
			name: "order_delivered_cash_vat_cogs",
			build: func() (JournalEntryInput, error) {
				return BuildOrderDeliveredInput(RuleEvalContext{
					EntityID: "sup-1", SourceID: "ord-1", OrderID: "ord-1", DriverID: "drv-1",
					WarehouseID: "wh-1", ProductID: "sku-1", LotID: "lot-1",
					PaymentMode: "CASH", Currency: "UZS", IdempotencyKey: "k-od-cash", OccurredAt: now,
					GrossMinor: 112000, NetMinor: 100000, VatMinor: 12000, CostMinor: 70000,
				})
			},
			wantLines: 5, // Dr CASH:DRIVER 112k, Dr COGS 70k, Cr REV:SALES 100k, Cr TAX:VAT:OUTPUT 12k, Cr INVENTORY 70k
			wantTotal: 182000,
		},
		{
			name: "order_delivered_nasiya_kz",
			build: func() (JournalEntryInput, error) {
				return BuildOrderDeliveredInput(RuleEvalContext{
					EntityID: "sup-kz", CountryPack: "KZ", SourceID: "ord-kz-1", OrderID: "ord-kz-1",
					RetailerID: "ret-1", PaymentMode: "CREDIT", Currency: "KZT",
					IdempotencyKey: "k-od-kz", OccurredAt: now,
					GrossMinor: 112000, NetMinor: 100000, VatMinor: 12000,
				})
			},
			wantLines: 3,
			wantTotal: 112000,
		},
		{
			name: "order_delivered_card",
			build: func() (JournalEntryInput, error) {
				return BuildOrderDeliveredInput(RuleEvalContext{
					EntityID: "sup-1", SourceID: "ord-card", PaymentMode: "CARD", Gateway: "PAYME",
					Currency: "UZS", IdempotencyKey: "k-od-card", OccurredAt: now, GrossMinor: 50000,
				})
			},
			wantLines: 2,
			wantTotal: 50000,
		},
		{
			name: "ar_payment_received",
			build: func() (JournalEntryInput, error) {
				return BuildARPaymentInput(RuleEvalContext{
					EntityID: "sup-1", SourceID: "inv-1", RetailerID: "ret-1",
					PaymentMode: "BANK", Currency: "UZS", IdempotencyKey: "k-ar-pay", OccurredAt: now,
					GrossMinor: 85000,
				})
			},
			wantLines: 2,
			wantTotal: 85000,
		},
		{
			name: "payout_paid_with_fee",
			build: func() (JournalEntryInput, error) {
				return BuildPayoutPaidInput(RuleEvalContext{
					EntityID: "sup-1", SourceID: "batch-1", Currency: "UZS",
					IdempotencyKey: "k-po-1", OccurredAt: now, GrossMinor: 100000, FeeMinor: 5000,
				})
			},
			wantLines: 3,
			wantTotal: 100000,
		},
		{
			name: "creditnote_issued_with_vat",
			build: func() (JournalEntryInput, error) {
				return BuildCreditNoteInput(RuleEvalContext{
					EntityID: "sup-1", SourceID: "cn-1", OrderID: "ord-1", RetailerID: "ret-1",
					Currency: "UZS", IdempotencyKey: "k-cn-1", OccurredAt: now,
					NetMinor: 20000, VatMinor: 2400, GrossMinor: 22400,
				})
			},
			wantLines: 3,
			wantTotal: 22400,
		},
		{
			name: "cash_variance_shortfall_and_writeoff",
			build: func() (JournalEntryInput, error) {
				return BuildCashVarianceInput(RuleEvalContext{
					EntityID: "sup-1", SourceID: "rec-1", DriverID: "drv-1",
					Action: "WRITE_OFF", Currency: "UZS", IdempotencyKey: "k-cv-wo", OccurredAt: now,
					ExpectedMinor: 100000, ReceivedMinor: 92000,
				})
			},
			wantLines: 2,
			wantTotal: 8000,
		},
		{
			name: "fee_per_order",
			build: func() (JournalEntryInput, error) {
				return BuildPerOrderFeeInput(RuleEvalContext{
					EntityID: "sup-1", SourceID: "ord-1", OrderID: "ord-1",
					Currency: "UZS", IdempotencyKey: "k-fee-1", OccurredAt: now, FeeMinor: 1500,
				})
			},
			wantLines: 2,
			wantTotal: 1500,
		},
		{
			name: "split_tender",
			build: func() (JournalEntryInput, error) {
				return BuildSplitTenderInput(RuleEvalContext{
					EntityID: "sup-1", SourceID: "ord-split", OrderID: "ord-split", RetailerID: "ret-1", DriverID: "drv-1",
					Currency: "UZS", IdempotencyKey: "k-split-1", OccurredAt: now,
					GrossMinor: 100000, DiscountMinor: 10000,
					Tenders: []TenderComponent{
						{Method: "CARD", Gateway: "PAYME", AmountMinor: 60000, ReferenceID: "p1"},
						{Method: "CASH", Gateway: "MANUAL", AmountMinor: 30000, ReferenceID: "c1"},
					},
				})
			},
			wantLines: 4,
			wantTotal: 100000,
		},
		{
			name: "claim_dispute_store_credit",
			build: func() (JournalEntryInput, error) {
				return BuildClaimDisputeInput(RuleEvalContext{
					EntityID: "sup-1", SourceID: "clm-1", OrderID: "ord-1", RetailerID: "ret-1",
					SettlementMode: "STORE_CREDIT", Currency: "UZS", IdempotencyKey: "k-clm-1", OccurredAt: now,
					GrossMinor: 12000,
				})
			},
			wantLines: 2,
			wantTotal: 12000,
		},
		{
			name: "doorstep_shortfall",
			build: func() (JournalEntryInput, error) {
				return BuildDoorstepShortfallInput(RuleEvalContext{
					EntityID: "sup-1", SourceID: "ord-sf-1", OrderID: "ord-sf-1", DriverID: "drv-1",
					Currency: "UZS", IdempotencyKey: "k-sf-1", OccurredAt: now,
					ExpectedMinor: 100000, ReceivedMinor: 80000,
				})
			},
			wantLines: 3,
			wantTotal: 100000,
		},
		{
			name: "migration_opening_balance",
			build: func() (JournalEntryInput, error) {
				return BuildMigrationOpeningBalanceInput(RuleEvalContext{
					EntityID: "sup-1", SourceID: "mig-1", RetailerID: "ret-1",
					Currency: "UZS", IdempotencyKey: "k-mig-1", OccurredAt: now, GrossMinor: 450000,
				})
			},
			wantLines: 2,
			wantTotal: 450000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in, err := tc.build()
			if err != nil {
				t.Fatalf("build failed: %v", err)
			}
			if len(in.Postings) != tc.wantLines {
				t.Fatalf("posting count = %d, want %d", len(in.Postings), tc.wantLines)
			}
			td, _, err := ValidateJournalInput(context.Background(), &in)
			if err != nil {
				t.Fatalf("validate failed: %v", err)
			}
			if td != tc.wantTotal {
				t.Fatalf("total = %d, want %d", td, tc.wantTotal)
			}
		})
	}
}

func TestBackfillParser_LegacyFormatsAndOrphanFiltering(t *testing.T) {
	now := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	rows := []pleRow{
		// 1. Balanced GL_* pair from payment.BuildSplitTenderJournalEntry
		{LedgerEntryID: "e1_p0", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-1", EntryType: "GL_DEBIT_PSP_GATEWAY_PAYME", AmountMinor: 60000, Currency: "UZS", ReferenceID: "jentry_order_ord-1", Source: "ledger.double_entry", OccurredAt: now},
		{LedgerEntryID: "e1_p1", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-1", EntryType: "GL_DEBIT_CASH_DRIVER", AmountMinor: 40000, Currency: "UZS", ReferenceID: "jentry_order_ord-1", Source: "ledger.double_entry", OccurredAt: now},
		{LedgerEntryID: "e1_p2", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-1", EntryType: "GL_CREDIT_ESCROW_ORDER_ord-1", AmountMinor: 100000, Currency: "UZS", ReferenceID: "jentry_order_ord-1", Source: "ledger.double_entry", OccurredAt: now},
		// 2. CreditNote pair (without party suffix)
		{LedgerEntryID: "cn_p0", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-2", EntryType: "GL_DEBIT_ADJ_CREDITNOTE", AmountMinor: 15000, Currency: "UZS", ReferenceID: "jentry_cn_cn1", Source: "ledger.creditnote", OccurredAt: now},
		{LedgerEntryID: "cn_p1", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-2", EntryType: "GL_CREDIT_AR_RETAILER", AmountMinor: 15000, Currency: "UZS", ReferenceID: "jentry_cn_cn1", Source: "ledger.creditnote", OccurredAt: now},
		// 3. Legacy single-leg settled webhook
		{LedgerEntryID: "wh_1", SupplierID: "sup-1", OrderID: "ord-3", Gateway: "PAYME", EntryType: "WEBHOOK_PAID", AmountMinor: 50000, Currency: "UZS", ReferenceID: "payme_txn_99", Source: "payment.webhook", OccurredAt: now},
		// 4. Non-posting initiated session (should be skipped)
		{LedgerEntryID: "sess_init", SupplierID: "sup-1", OrderID: "ord-4", Gateway: "PAYME", EntryType: "SESSION_AWAITING_PAYMENT", AmountMinor: 50000, Currency: "UZS", ReferenceID: "sess_4", Source: "payment.session", OccurredAt: now},
		// 5. Unbalanced orphan GL_* row (should be skipped safely)
		{LedgerEntryID: "orphan_1", SupplierID: "sup-1", EntryType: "GL_DEBIT_AR_RETAILER_ret-9", AmountMinor: 9000, Currency: "UZS", ReferenceID: "jentry_broken", Source: "ledger.double_entry", OccurredAt: now},
	}

	inputs, stats := buildBackfillInputs("sup-1", "gl_backfill", rows)
	if len(inputs) != 3 {
		t.Fatalf("got %d backfill entries, want 3 (gl group + cn group + webhook)", len(inputs))
	}
	if stats.SkippedUnbalanced != 1 || stats.SkippedNonPosting != 1 {
		t.Fatalf("unexpected skip stats: %+v", stats)
	}
	for i := range inputs {
		if _, _, err := ValidateJournalInput(context.Background(), &inputs[i]); err != nil {
			t.Fatalf("backfill input[%d] failed validation: %v", i, err)
		}
	}
}
