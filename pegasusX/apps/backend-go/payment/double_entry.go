package payment

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Chart of accounts standard classifications
const (
	AccountClassAsset     = "ASSET"
	AccountClassLiability = "LIABILITY"
	AccountClassEquity    = "EQUITY"
	AccountClassRevenue   = "REVENUE"
	AccountClassExpense   = "EXPENSE"
)

// Standard system accounts
const (
	// Assets
	AccountAccountsReceivable     = "AR:RETAILER"       // Retailer outstanding invoices
	AccountCashInTransit          = "CASH:DRIVER"       // Driver collected cash on delivery
	AccountCardPSPTransit         = "PSP:GATEWAY"       // In-flight funds with Payme/Click/Adyen
	AccountBankOperating          = "BANK:OPERATING"    // Supplier bank account

	// Liabilities
	AccountSupplierPayable        = "AP:SUPPLIER"       // Net due to supplier
	AccountPlatformEscrow         = "ESCROW:ORDER"      // Buyer funds held before fulfillment
	AccountRetailerCredit         = "WALLET:RETAILER"   // Customer deposit / credit wallet

	// Equity / Revenue / Expense
	AccountPlatformFeeRevenue     = "REV:PLATFORM:FEE"  // Pegasus platform marketplace cut
	AccountDiscountsAndAllowances = "EXP:PROMO:DISCOUNT"// Promo code / marketing subsidy
)

// PostingDirection designates Debit or Credit in double-entry bookkeeping.
type PostingDirection string

const (
	PostingDebit  PostingDirection = "DEBIT"
	PostingCredit PostingDirection = "CREDIT"
)

// Posting is an atomic financial movement on a specific ledger account.
type Posting struct {
	AccountCode string           `json:"account_code"`
	AccountType string           `json:"account_type"` // ASSET, LIABILITY, REVENUE, EXPENSE, EQUITY
	Direction   PostingDirection `json:"direction"`    // DEBIT or CREDIT
	AmountMinor int64            `json:"amount_minor"` // strictly positive
	Currency    string           `json:"currency"`
	PartyID     string           `json:"party_id,omitempty"`
	Description string           `json:"description,omitempty"`
}

// JournalEntry is an immutable, balanced set of postings representing a financial transaction.
type JournalEntry struct {
	EntryID       string    `json:"entry_id"`
	TransactionID string    `json:"transaction_id"`
	OrderID       string    `json:"order_id,omitempty"`
	SessionID     string    `json:"session_id,omitempty"`
	SupplierID    string    `json:"supplier_id"`
	RetailerID    string    `json:"retailer_id,omitempty"`
	Currency      string    `json:"currency"`
	Memo          string    `json:"memo"`
	OccurredAt    time.Time `json:"occurred_at"`
	Postings      []Posting `json:"postings"`
}

// Invariant errors
var (
	ErrUnbalancedLedgerEntry   = errors.New("ledger: sum of debits must strictly equal sum of credits")
	ErrZeroAmountPosting       = errors.New("ledger: posting amount must be strictly greater than zero")
	ErrInsufficientPostings    = errors.New("ledger: journal entry must contain at least 2 postings")
	ErrCurrencyMismatchPosting = errors.New("ledger: posting currency does not match journal entry currency")
)

// Validate enforces the fundamental double-entry invariant: sum(Debits) == sum(Credits).
func (je *JournalEntry) Validate() error {
	if len(je.Postings) < 2 {
		return ErrInsufficientPostings
	}
	var sumDebits int64
	var sumCredits int64

	for i, p := range je.Postings {
		if p.AmountMinor <= 0 {
			return fmt.Errorf("%w at posting index %d (account %s)", ErrZeroAmountPosting, i, p.AccountCode)
		}
		if strings.TrimSpace(p.AccountCode) == "" {
			return fmt.Errorf("ledger: empty account_code at posting index %d", i)
		}
		if p.Currency != "" && je.Currency != "" && !strings.EqualFold(p.Currency, je.Currency) {
			return fmt.Errorf("%w: posting %s has %s want %s", ErrCurrencyMismatchPosting, p.AccountCode, p.Currency, je.Currency)
		}
		switch p.Direction {
		case PostingDebit:
			sumDebits += p.AmountMinor
		case PostingCredit:
			sumCredits += p.AmountMinor
		default:
			return fmt.Errorf("ledger: invalid posting direction %q at index %d", p.Direction, i)
		}
	}

	if sumDebits != sumCredits {
		return fmt.Errorf("%w: sum(debits)=%d != sum(credits)=%d (discrepancy=%d minor)",
			ErrUnbalancedLedgerEntry, sumDebits, sumCredits, sumDebits-sumCredits)
	}
	return nil
}

// TenderSplit represents one leg of a split-tender payment (e.g. Card, Cash, Wallet).
type TenderSplit struct {
	Method      string `json:"method"`       // "CARD", "CASH", "WALLET"
	Gateway     string `json:"gateway"`      // "PAYME", "CLICK", "ADYEN", "MANUAL"
	AmountMinor int64  `json:"amount_minor"` // positive amount
	ReferenceID string `json:"reference_id,omitempty"`
}

// BuildSplitTenderJournalEntry creates and validates a balanced JournalEntry for an order paid via multiple tenders.
// Invariant: sum(tenders.AmountMinor) + discountMinor == totalOrderMinor
func BuildSplitTenderJournalEntry(
	orderID, supplierID, retailerID, currency string,
	totalOrderMinor int64,
	tenders []TenderSplit,
	discountMinor int64,
	memo string,
) (*JournalEntry, error) {
	if totalOrderMinor <= 0 {
		return nil, fmt.Errorf("ledger: total order minor must be positive, got %d", totalOrderMinor)
	}

	je := &JournalEntry{
		EntryID:    fmt.Sprintf("jentry_order_%s_%d", orderID, time.Now().UTC().UnixNano()),
		OrderID:    orderID,
		SupplierID: supplierID,
		RetailerID: retailerID,
		Currency:   strings.ToUpper(currency),
		Memo:       memo,
		OccurredAt: time.Now().UTC(),
		Postings:   make([]Posting, 0, len(tenders)+2),
	}

	// Debits: asset accounts collecting funds
	for _, t := range tenders {
		if t.AmountMinor <= 0 {
			continue
		}
		accountCode := AccountCardPSPTransit + ":" + strings.ToUpper(t.Gateway)
		if strings.EqualFold(t.Method, "CASH") {
			accountCode = AccountCashInTransit
		} else if strings.EqualFold(t.Method, "WALLET") {
			accountCode = AccountRetailerCredit + ":" + retailerID
		}

		je.Postings = append(je.Postings, Posting{
			AccountCode: accountCode,
			AccountType: AccountClassAsset,
			Direction:   PostingDebit,
			AmountMinor: t.AmountMinor,
			Currency:    je.Currency,
			PartyID:     retailerID,
			Description: fmt.Sprintf("Split tender %s via %s ref:%s", t.Method, t.Gateway, t.ReferenceID),
		})
	}

	// Promotional discount leg (Debit Marketing/Promo Expense)
	if discountMinor > 0 {
		je.Postings = append(je.Postings, Posting{
			AccountCode: AccountDiscountsAndAllowances,
			AccountType: AccountClassExpense,
			Direction:   PostingDebit,
			AmountMinor: discountMinor,
			Currency:    je.Currency,
			PartyID:     supplierID,
			Description: "Promotional discount subsidy",
		})
	}

	// Credit: order escrow liability awaiting delivery
	je.Postings = append(je.Postings, Posting{
		AccountCode: AccountPlatformEscrow + ":" + orderID,
		AccountType: AccountClassLiability,
		Direction:   PostingCredit,
		AmountMinor: totalOrderMinor,
		Currency:    je.Currency,
		PartyID:     supplierID,
		Description: fmt.Sprintf("Order fulfillment escrow obligation for %s", orderID),
	})

	if err := je.Validate(); err != nil {
		return nil, fmt.Errorf("split tender ledger validation failed: %w", err)
	}

	return je, nil
}

// BuildSettlementJournalEntry creates and validates a balanced JournalEntry when order escrow is settled to supplier.
func BuildSettlementJournalEntry(
	orderID, supplierID, currency string,
	grossOrderMinor, platformFeeMinor int64,
) (*JournalEntry, error) {
	if grossOrderMinor <= 0 {
		return nil, fmt.Errorf("ledger: gross minor must be positive, got %d", grossOrderMinor)
	}
	if platformFeeMinor < 0 || platformFeeMinor > grossOrderMinor {
		return nil, fmt.Errorf("ledger: platform fee minor (%d) invalid for gross (%d)", platformFeeMinor, grossOrderMinor)
	}
	netSupplierMinor := grossOrderMinor - platformFeeMinor

	je := &JournalEntry{
		EntryID:    fmt.Sprintf("jentry_settle_%s_%d", orderID, time.Now().UTC().UnixNano()),
		OrderID:    orderID,
		SupplierID: supplierID,
		Currency:   strings.ToUpper(currency),
		Memo:       fmt.Sprintf("Escrow release and settlement for order %s", orderID),
		OccurredAt: time.Now().UTC(),
		Postings:   make([]Posting, 0, 3),
	}

	// Debit: release escrow liability
	je.Postings = append(je.Postings, Posting{
		AccountCode: AccountPlatformEscrow + ":" + orderID,
		AccountType: AccountClassLiability,
		Direction:   PostingDebit,
		AmountMinor: grossOrderMinor,
		Currency:    je.Currency,
		PartyID:     supplierID,
		Description: "Release order escrow obligation",
	})

	// Credit: net due to supplier
	je.Postings = append(je.Postings, Posting{
		AccountCode: AccountSupplierPayable + ":" + supplierID,
		AccountType: AccountClassLiability,
		Direction:   PostingCredit,
		AmountMinor: netSupplierMinor,
		Currency:    je.Currency,
		PartyID:     supplierID,
		Description: "Net payout payable to supplier",
	})

	// Credit: platform fee revenue (if any)
	if platformFeeMinor > 0 {
		je.Postings = append(je.Postings, Posting{
			AccountCode: AccountPlatformFeeRevenue,
			AccountType: AccountClassRevenue,
			Direction:   PostingCredit,
			AmountMinor: platformFeeMinor,
			Currency:    je.Currency,
			Description: "Platform marketplace commission",
		})
	}

	if err := je.Validate(); err != nil {
		return nil, fmt.Errorf("settlement ledger validation failed: %w", err)
	}

	return je, nil
}

// ToLedgerEntryRecords transforms a balanced JournalEntry into durable LedgerEntryRecords for persistence.
func (je *JournalEntry) ToLedgerEntryRecords() []LedgerEntryRecord {
	records := make([]LedgerEntryRecord, 0, len(je.Postings))
	for idx, p := range je.Postings {
		entryType := fmt.Sprintf("GL_%s_%s", p.Direction, strings.ReplaceAll(p.AccountCode, ":", "_"))
		records = append(records, LedgerEntryRecord{
			LedgerEntryID: fmt.Sprintf("%s_p%d", je.EntryID, idx),
			SessionID:     je.SessionID,
			OrderID:       je.OrderID,
			SupplierID:    je.SupplierID,
			RetailerID:    je.RetailerID,
			Gateway:       "GENERAL_LEDGER",
			EntryType:     entryType,
			AmountMinor:   p.AmountMinor,
			Currency:      p.Currency,
			ReferenceID:   je.EntryID,
			Source:        "ledger.double_entry",
			OccurredAt:    je.OccurredAt,
			CreatedAt:     je.OccurredAt,
		})
	}
	return records
}
