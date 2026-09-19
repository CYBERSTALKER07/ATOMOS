import re

with open("apps/backend-go/supplier/orders_vet.go", "r") as f:
    content = f.read()

replacement = """
func orderPaymentClearedInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT
		        EXISTS (
		          SELECT 1
		          FROM PaymentLedgerEntries ple
		          WHERE ple.OrderId = @orderId
		            AND ple.EntryType IN UNNEST(@clearedEntryTypes)
		        ) AS ledger_cleared,
		        (
		          SELECT 
		            CASE 
		              WHEN ps.Gateway IN ('CASH', 'CREDIT', 'B2B_CREDIT', 'INTERNAL') THEN true
		              WHEN ps.Status IN UNNEST(@clearedSessionStatuses) THEN true
		              ELSE false
		            END
		          FROM PaymentSessions ps
		          WHERE ps.OrderId = @orderId
		          ORDER BY ps.CreatedAt DESC
		          LIMIT 1
		        ) AS session_cleared`,
		Params: map[string]any{
			"orderId":                orderID,
			"clearedEntryTypes":      []string{"WEBHOOK_PAID", "CASH_COLLECTED", "SETTLEMENT_CREDIT"},
			"clearedSessionStatuses": []string{"PAID", "CAPTURED", "SETTLED", "SUCCESS", "AUTHORIZED"},
		},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return false, fmt.Errorf("query payment clearance: %w", err)
	}
	var ledgerCleared bool
	var sessionCleared spanner.NullBool
	if err := row.Columns(&ledgerCleared, &sessionCleared); err != nil {
		return false, fmt.Errorf("scan payment clearance: %w", err)
	}
	
	sessionIsCleared := true
	if sessionCleared.Valid {
		sessionIsCleared = sessionCleared.Bool
	}
	return ledgerCleared || sessionIsCleared, nil
}
"""

pattern = re.compile(r'\nfunc orderPaymentClearedInTxn.*?return ledgerCleared \|\| sessionCleared, nil\n\}', re.DOTALL)
content = pattern.sub(replacement, content)

with open("apps/backend-go/supplier/orders_vet.go", "w") as f:
    f.write(content)

