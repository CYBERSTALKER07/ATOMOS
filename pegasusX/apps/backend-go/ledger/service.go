package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateAccount(ctx context.Context, id, name string, accountType AccountType, currency string) error {
	act := Account{
		AccountID: id,
		Name:      name,
		Type:      accountType,
		Currency:  currency,
	}
	return s.repo.CreateAccount(ctx, act)
}

// TransferRequest is used to specify debit and credit entries.
type TransferRequest struct {
	ReferenceID string
	Description string
	Debits      []TransferEntry
	Credits     []TransferEntry
}

type TransferEntry struct {
	AccountID   string
	AmountMinor int64 // Positive amount
}

// RecordTransfer records a double-entry transaction. It ensures debits equal credits.
func (s *Service) RecordTransfer(ctx context.Context, req TransferRequest) (string, error) {
	if len(req.Debits) == 0 && len(req.Credits) == 0 {
		return "", fmt.Errorf("transfer requires at least one debit or credit")
	}

	var totalDebit int64
	var totalCredit int64
	
	entries := make([]Entry, 0, len(req.Debits)+len(req.Credits))
	txnID := uuid.New().String()

	for _, d := range req.Debits {
		if d.AmountMinor < 0 {
			return "", fmt.Errorf("debit amount must be non-negative, got %d", d.AmountMinor)
		}
		totalDebit += d.AmountMinor
		entries = append(entries, Entry{
			EntryID:       uuid.New().String(),
			TransactionID: txnID,
			AccountID:     d.AccountID,
			AmountMinor:   d.AmountMinor,
			Type:          EntryTypeDebit,
		})
	}

	for _, c := range req.Credits {
		if c.AmountMinor < 0 {
			return "", fmt.Errorf("credit amount must be non-negative, got %d", c.AmountMinor)
		}
		totalCredit += c.AmountMinor
		entries = append(entries, Entry{
			EntryID:       uuid.New().String(),
			TransactionID: txnID,
			AccountID:     c.AccountID,
			AmountMinor:   c.AmountMinor,
			Type:          EntryTypeCredit,
		})
	}

	if totalDebit != totalCredit {
		return "", fmt.Errorf("transfer unbalanced: total debit %d != total credit %d", totalDebit, totalCredit)
	}

	txn := Transaction{
		TransactionID: txnID,
		ReferenceID:   req.ReferenceID,
		Description:   req.Description,
	}

	if err := s.repo.RecordTransaction(ctx, txn, entries, func(txn outbox.TxnBuffer) error {
		// Emit domain event for the transaction.
		return outbox.EmitJSON(ctx, txn, "ledger_transaction", txnID, "ledger.main", map[string]any{
			"Type":        "LEDGER_TRANSACTION_RECORDED",
			"Timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
			"ReferenceId": req.ReferenceID,
			"Description": req.Description,
			"TotalAmount": totalDebit, // Can just use Debit since Debit == Credit
		})
	}); err != nil {
		return "", err
	}

	return txnID, nil
}

func (s *Service) GetAccountBalance(ctx context.Context, accountID string) (AccountBalance, error) {
	return s.repo.GetAccountBalance(ctx, accountID)
}

func (s *Service) ListTransactions(ctx context.Context, accountID string, limit int) ([]Transaction, error) {
	return s.repo.ListTransactions(ctx, accountID, limit)
}
