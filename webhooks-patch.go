package payment

import (
	"context"
	"fmt"
	"time"

	"backend-go/outbox"
	"cloud.google.com/go/spanner"
	"log/slog"
)

func (ws *WebhookService) settleGlobalPayInvoiceTxn(ctx context.Context, session PaymentSession) error {
	var retailerID string
	var total int64
	var state string
	invoiceID := session.InvoiceID

	_, err := ws.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "MasterInvoices", spanner.Key{invoiceID},
			[]string{"RetailerId", "Total", "State"})
		if readErr != nil {
			return fmt.Errorf("invoice not found: %s", invoiceID)
		}

		if colErr := row.Columns(&retailerID, &total, &state); colErr != nil {
			return fmt.Errorf("invoice row parse error: %w", colErr)
		}

		if state == "SETTLED" {
			return fmt.Errorf("already settled")
		}
		if state != "PENDING" {
			return fmt.Errorf("invoice %s in state %s, cannot settle", invoiceID, state)
		}
		if total != session.LockedAmount {
			return fmt.Errorf("amount mismatch: invoice=%d webhook=%d", total, session.LockedAmount)
		}

		muts := []*spanner.Mutation{
			spanner.Update("MasterInvoices",
				[]string{"InvoiceId", "State"},
				[]interface{}{invoiceID, "SETTLED"},
			),
		}

		// Write MasterInvoice update
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}

		// INVOICE_SETTLED (To trigger order status changes or others)
		invoiceEvent := InvoiceSettledEvent{
			InvoiceID:  invoiceID,
			Gateway:    "GLOBAL_PAY",
			Amount:     session.LockedAmount,
			RetailerID: retailerID,
			Timestamp:  time.Now().UTC(),
		}
		if err := outbox.EmitJSON(txn, "MasterInvoice", invoiceID, "lab-logistics-events", invoiceEvent); err != nil {
			return fmt.Errorf("failed to emit INVOICE_SETTLED: %w", err)
		}

		// PAYMENT_SETTLED (Notification Dispatcher)
		if session.OrderID != "" {
			pmtEvent := map[string]interface{}{
				"order_id":    session.OrderID,
				"invoice_id":  invoiceID,
				"retailer_id": retailerID,
				"driver_id":   ws.resolveDriverFromOrder(session.OrderID),
				"gateway":     "GLOBAL_PAY",
				"amount":      session.LockedAmount,
				"timestamp":   time.Now().UTC(),
			}
			if err := outbox.EmitJSON(txn, "Order", session.OrderID, "lab-logistics-events", pmtEvent); err != nil {
				return fmt.Errorf("failed to emit PAYMENT_SETTLED: %w", err)
			}
		}

		return nil
	})

	return err
}
