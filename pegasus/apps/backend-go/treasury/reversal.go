package treasury

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
)

// CreateReversalEntry creates a negative-amount LedgerEntry that references
// the original entry. Used by the refund flow and chargeback handler.
//
// The reversal entry has a negative Amount and an EntryType of "DEBIT_REVERSAL".
// AccountId is copied from the original entry.
func CreateReversalEntry(ctx context.Context, client *spanner.Client, originalTxnID, reason string) error {
	// Read the original entry
	row, err := client.Single().ReadRow(ctx, "LedgerEntries",
		spanner.Key{originalTxnID},
		[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType"})
	if err != nil {
		return fmt.Errorf("read original ledger entry %s: %w", originalTxnID, err)
	}

	var txnID, orderID, accountID, entryType string
	var amount int64
	if err := row.Columns(&txnID, &orderID, &accountID, &amount, &entryType); err != nil {
		return fmt.Errorf("parse original entry: %w", err)
	}

	// Create reversal entry with negative amount
	reversalTxnID := fmt.Sprintf("REV-%s", originalTxnID)
	now := time.Now()

	_, txErr := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.Insert("LedgerEntries",
			[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType", "CreatedAt"},
			[]interface{}{reversalTxnID, orderID, accountID, -amount, "DEBIT_REVERSAL", now},
		)
		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	if txErr != nil {
		return fmt.Errorf("reversal transaction: %w", txErr)
	}

	slog.Info("treasury.reversal_created", "original_txn_id", originalTxnID, "amount", amount, "account_id", accountID, "reason", reason)
	return nil
}

// GetLedgerBalance returns the net balance for an account.
func GetLedgerBalance(ctx context.Context, client *spanner.Client, accountID string) (int64, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT SUM(Amount) FROM LedgerEntries WHERE AccountId = @accountId`,
		Params: map[string]interface{}{"accountId": accountID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("ledger balance query: %w", err)
	}

	var balance spanner.NullInt64
	if err := row.Columns(&balance); err != nil {
		return 0, fmt.Errorf("parse balance: %w", err)
	}

	return balance.Int64, nil
}
