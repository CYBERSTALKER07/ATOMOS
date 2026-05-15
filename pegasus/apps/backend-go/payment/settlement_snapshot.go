package payment

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SupplierSettlementSnapshot captures additive fee-snapshot details for one
// order + supplier pair.
type SupplierSettlementSnapshot struct {
	InvoiceID        string
	SupplierID       string
	GrossAmount      int64
	FeeAmount        int64
	NetPayoutAmount  int64
	PayoutOwnerType  string
	PayoutOwnerID    string
	FeePolicyVersion string
	Currency         string
}

// LoadSupplierSettlementSnapshot resolves an order's supplier settlement slice.
// Returns (snapshot, false, nil) when no slice exists so callers can use legacy
// fallback behavior while the cutover completes.
func LoadSupplierSettlementSnapshot(ctx context.Context, client *spanner.Client, orderID, supplierID string) (SupplierSettlementSnapshot, bool, error) {
	if client == nil {
		return SupplierSettlementSnapshot{}, false, fmt.Errorf("spanner client is nil")
	}

	invoiceStmt := spanner.Statement{
		SQL: `SELECT InvoiceId
		      FROM MasterInvoices
		      WHERE OrderId = @orderId
		      ORDER BY CreatedAt DESC
		      LIMIT 1`,
		Params: map[string]interface{}{"orderId": orderID},
	}

	invoiceIter := client.Single().Query(ctx, invoiceStmt)
	defer invoiceIter.Stop()

	invoiceRow, invoiceErr := invoiceIter.Next()
	if invoiceErr == iterator.Done {
		return SupplierSettlementSnapshot{}, false, nil
	}
	if invoiceErr != nil {
		return SupplierSettlementSnapshot{}, false, fmt.Errorf("load invoice for order %s: %w", orderID, invoiceErr)
	}

	var invoiceID string
	if err := invoiceRow.Columns(&invoiceID); err != nil {
		return SupplierSettlementSnapshot{}, false, fmt.Errorf("parse invoice for order %s: %w", orderID, err)
	}

	aggStmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(GrossAmount), 0),
		             COALESCE(SUM(FeeAmount), 0),
		             COALESCE(SUM(NetPayoutAmount), 0)
		      FROM InvoiceSettlementSlices
		      WHERE InvoiceId = @invoiceId AND SupplierId = @supplierId`,
		Params: map[string]interface{}{
			"invoiceId":  invoiceID,
			"supplierId": supplierID,
		},
	}

	aggIter := client.Single().Query(ctx, aggStmt)
	defer aggIter.Stop()

	aggRow, aggErr := aggIter.Next()
	if aggErr != nil {
		if aggErr == iterator.Done {
			return SupplierSettlementSnapshot{}, false, nil
		}
		return SupplierSettlementSnapshot{}, false, fmt.Errorf("aggregate settlement slices for invoice %s supplier %s: %w", invoiceID, supplierID, aggErr)
	}

	var grossAmount int64
	var feeAmount int64
	var netPayoutAmount int64
	if err := aggRow.Columns(&grossAmount, &feeAmount, &netPayoutAmount); err != nil {
		return SupplierSettlementSnapshot{}, false, fmt.Errorf("parse aggregate settlement slices for invoice %s supplier %s: %w", invoiceID, supplierID, err)
	}
	if grossAmount <= 0 {
		return SupplierSettlementSnapshot{}, false, nil
	}

	detailStmt := spanner.Statement{
		SQL: `SELECT PayoutOwnerType, PayoutOwnerId, FeePolicyVersion, Currency
		      FROM InvoiceSettlementSlices
		      WHERE InvoiceId = @invoiceId AND SupplierId = @supplierId
		      ORDER BY CreatedAt DESC
		      LIMIT 1`,
		Params: map[string]interface{}{
			"invoiceId":  invoiceID,
			"supplierId": supplierID,
		},
	}

	detailIter := client.Single().Query(ctx, detailStmt)
	defer detailIter.Stop()

	detailRow, detailErr := detailIter.Next()
	if detailErr != nil {
		if detailErr == iterator.Done {
			return SupplierSettlementSnapshot{
				InvoiceID:       invoiceID,
				SupplierID:      supplierID,
				GrossAmount:     grossAmount,
				FeeAmount:       feeAmount,
				NetPayoutAmount: netPayoutAmount,
			}, true, nil
		}
		return SupplierSettlementSnapshot{}, false, fmt.Errorf("load settlement slice detail for invoice %s supplier %s: %w", invoiceID, supplierID, detailErr)
	}

	var payoutOwnerType spanner.NullString
	var payoutOwnerID spanner.NullString
	var feePolicyVersion spanner.NullString
	var currency spanner.NullString
	if err := detailRow.Columns(&payoutOwnerType, &payoutOwnerID, &feePolicyVersion, &currency); err != nil {
		return SupplierSettlementSnapshot{}, false, fmt.Errorf("parse settlement slice detail for invoice %s supplier %s: %w", invoiceID, supplierID, err)
	}

	snapshot := SupplierSettlementSnapshot{
		InvoiceID:       invoiceID,
		SupplierID:      supplierID,
		GrossAmount:     grossAmount,
		FeeAmount:       feeAmount,
		NetPayoutAmount: netPayoutAmount,
	}
	if payoutOwnerType.Valid {
		snapshot.PayoutOwnerType = payoutOwnerType.StringVal
	}
	if payoutOwnerID.Valid {
		snapshot.PayoutOwnerID = payoutOwnerID.StringVal
	}
	if feePolicyVersion.Valid {
		snapshot.FeePolicyVersion = feePolicyVersion.StringVal
	}
	if currency.Valid {
		snapshot.Currency = currency.StringVal
	}

	return snapshot, true, nil
}
