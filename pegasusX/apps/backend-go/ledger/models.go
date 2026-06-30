package ledger

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// AccountType defines the standard double-entry account classifications.
type AccountType string

const (
	AccountTypeAsset     AccountType = "ASSET"
	AccountTypeLiability AccountType = "LIABILITY"
	AccountTypeEquity    AccountType = "EQUITY"
	AccountTypeRevenue   AccountType = "REVENUE"
	AccountTypeExpense   AccountType = "EXPENSE"
)

type EntryType string

const (
	EntryTypeDebit  EntryType = "DEBIT"
	EntryTypeCredit EntryType = "CREDIT"
)

// Account represents a financial account.
type Account struct {
	AccountID string
	Name      string
	Type      AccountType
	Currency  string
	CreatedAt time.Time
}

// Transaction represents an atomic double-entry accounting transaction.
type Transaction struct {
	TransactionID string
	ReferenceID   string
	Description   string
	CreatedAt     time.Time
}

// Entry is a single movement of funds into or out of an account.
type Entry struct {
	EntryID       string
	TransactionID string
	AccountID     string
	AmountMinor   int64 // Positive integer
	Type          EntryType
	CreatedAt     time.Time
}

// Balance Query Result
type AccountBalance struct {
	AccountID   string
	Currency    string
	Balance     int64 // Calculated balance based on normal balances (Assets/Expenses are Debit-normal; Liab/Eq/Rev are Credit-normal)
}

// Repository defines the contract for persisting double-entry ledger state.
type Repository interface {
	CreateAccount(ctx context.Context, act Account) error
	RecordTransaction(ctx context.Context, txn Transaction, entries []Entry, emit func(outbox.TxnBuffer) error) error
	GetAccountBalance(ctx context.Context, accountID string) (AccountBalance, error)
	ListTransactions(ctx context.Context, accountID string, limit int) ([]Transaction, error)
}
