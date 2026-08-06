package partner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// journalColumnOrder is the stable CSV/XML attribute order for journals exports.
var journalColumnOrder = []string{
	"entry_date",
	"source",
	"entry_id",
	"entry_type",
	"debit_account",
	"credit_account",
	"amount_minor",
	"currency",
	"supplier_id",
	"retailer_id",
	"invoice_id",
	"order_id",
	"aging_bucket",
	"gateway",
	"memo",
}

func mapARJournalAccounts(coa CoaMap, entryType string) (debit, credit string) {
	ar := coa.AccountAR
	rev := coa.AccountRevenue
	bank := coa.AccountBankCash
	switch strings.ToUpper(strings.TrimSpace(entryType)) {
	case "OPEN":
		return ar, rev
	case "PAYMENT":
		return bank, ar
	default:
		// Unknown AR types: treat as AR increase vs suspense revenue.
		return ar, rev
	}
}

func mapPaymentJournalAccounts(coa CoaMap, entryType string) (debit, credit string) {
	ar := coa.AccountAR
	bank := coa.AccountBankCash
	t := strings.ToUpper(strings.TrimSpace(entryType))
	if strings.Contains(t, "REFUND") || strings.Contains(t, "CHARGEBACK") ||
		(strings.Contains(t, "VOID") && !strings.Contains(t, "REVERSAL")) {
		return ar, bank
	}
	// Capture / settle / paid / success / reversal of chargeback → cash from AR.
	return bank, ar
}

func absMinor(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func journalRow(cols map[string]any) map[string]any {
	out := make(map[string]any, len(journalColumnOrder))
	for _, k := range journalColumnOrder {
		if v, ok := cols[k]; ok {
			out[k] = v
		} else {
			out[k] = ""
		}
	}
	return out
}

func (w *ExportWorker) resolveTenantCoa(ctx context.Context, tenantType, tenantID string) CoaMap {
	if w == nil || w.coa == nil {
		return DefaultCoa()
	}
	stored, found, err := w.coa.Get(ctx, tenantType, tenantID)
	if err != nil {
		w.log.Debug("coa load failed; using defaults", "tenant", tenantID, "err", err)
		return DefaultCoa()
	}
	return ResolveCoa(stored, found)
}

func (w *ExportWorker) exportJournals(ctx context.Context, tenantType, tenantID string, from, to time.Time) ([]map[string]any, error) {
	out := make([]map[string]any, 0)
	if w.spanner == nil {
		return out, nil
	}
	coa := w.resolveTenantCoa(ctx, tenantType, tenantID)
	arRows, err := w.exportJournalsAR(ctx, tenantType, tenantID, from, to, MaxExportRows, coa)
	if err != nil {
		// Best-effort: schema drift → continue with payment side.
		arRows = nil
	}
	out = append(out, arRows...)
	remain := MaxExportRows - len(out)
	if remain <= 0 {
		return out[:MaxExportRows], nil
	}
	payRows, err := w.exportJournalsPayment(ctx, tenantType, tenantID, from, to, remain, coa)
	if err != nil {
		return out, nil
	}
	out = append(out, payRows...)
	if len(out) > MaxExportRows {
		out = out[:MaxExportRows]
	}
	return out, nil
}

func (w *ExportWorker) exportJournalsAR(ctx context.Context, tenantType, tenantID string, from, to time.Time, lim int, coa CoaMap) ([]map[string]any, error) {
	if lim <= 0 {
		return nil, nil
	}
	var sql string
	params := map[string]any{"tid": tenantID, "from": from, "to": to, "lim": int64(lim)}
	switch tenantType {
	case TenantSupplier:
		sql = `SELECT e.EntryId, e.InvoiceId, e.SupplierId, e.RetailerId, e.EntryType, e.AmountMinor,
			COALESCE(e.RefOrderId, ''), e.CreatedAt,
			COALESCE(i.Currency, ''), COALESCE(i.OrderId, ''), COALESCE(i.AgingBucket, ''), COALESCE(i.Status, '')
			FROM ArLedgerEntries e
			LEFT JOIN ArInvoices i ON i.InvoiceId = e.InvoiceId
			WHERE e.SupplierId = @tid AND e.CreatedAt >= @from AND e.CreatedAt <= @to
			ORDER BY e.CreatedAt DESC LIMIT @lim`
	case TenantRetailer:
		sql = `SELECT e.EntryId, e.InvoiceId, e.SupplierId, e.RetailerId, e.EntryType, e.AmountMinor,
			COALESCE(e.RefOrderId, ''), e.CreatedAt,
			COALESCE(i.Currency, ''), COALESCE(i.OrderId, ''), COALESCE(i.AgingBucket, ''), COALESCE(i.Status, '')
			FROM ArLedgerEntries e
			LEFT JOIN ArInvoices i ON i.InvoiceId = e.InvoiceId
			WHERE e.RetailerId = @tid AND e.CreatedAt >= @from AND e.CreatedAt <= @to
			ORDER BY e.CreatedAt DESC LIMIT @lim`
	default:
		return nil, fmt.Errorf("invalid_tenant")
	}
	iter := w.spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := make([]map[string]any, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var entryID, invID, supplierID, retailerID, entryType, refOrder string
		var amount int64
		var created time.Time
		var currency, orderID, aging, invStatus string
		if err := row.Columns(&entryID, &invID, &supplierID, &retailerID, &entryType, &amount,
			&refOrder, &created, &currency, &orderID, &aging, &invStatus); err != nil {
			return nil, err
		}
		if orderID == "" {
			orderID = refOrder
		}
		debit, credit := mapARJournalAccounts(coa, entryType)
		amt := absMinor(amount)
		memo := fmt.Sprintf("AR %s invoice=%s", strings.ToUpper(entryType), invID)
		if invStatus != "" {
			memo += " status=" + invStatus
		}
		out = append(out, journalRow(map[string]any{
			"entry_date":     created.UTC().Format(time.RFC3339),
			"source":         "ar",
			"entry_id":       entryID,
			"entry_type":     strings.ToUpper(entryType),
			"debit_account":  debit,
			"credit_account": credit,
			"amount_minor":   amt,
			"currency":       currency,
			"supplier_id":    supplierID,
			"retailer_id":    retailerID,
			"invoice_id":     invID,
			"order_id":       orderID,
			"aging_bucket":   aging,
			"gateway":        "",
			"memo":           memo,
		}))
	}
	return out, nil
}

func (w *ExportWorker) exportJournalsPayment(ctx context.Context, tenantType, tenantID string, from, to time.Time, lim int, coa CoaMap) ([]map[string]any, error) {
	if lim <= 0 {
		return nil, nil
	}
	var sql string
	params := map[string]any{"tid": tenantID, "from": from, "to": to, "lim": int64(lim)}
	switch tenantType {
	case TenantSupplier:
		sql = `SELECT LedgerEntryId, COALESCE(OrderId, ''), SupplierId, RetailerId, COALESCE(Gateway, ''),
			EntryType, AmountMinor, COALESCE(Currency, ''), COALESCE(ReferenceId, ''), COALESCE(Source, ''), OccurredAt
			FROM PaymentLedgerEntries
			WHERE SupplierId = @tid AND OccurredAt >= @from AND OccurredAt <= @to
			ORDER BY OccurredAt DESC LIMIT @lim`
	case TenantRetailer:
		sql = `SELECT LedgerEntryId, COALESCE(OrderId, ''), SupplierId, RetailerId, COALESCE(Gateway, ''),
			EntryType, AmountMinor, COALESCE(Currency, ''), COALESCE(ReferenceId, ''), COALESCE(Source, ''), OccurredAt
			FROM PaymentLedgerEntries
			WHERE RetailerId = @tid AND OccurredAt >= @from AND OccurredAt <= @to
			ORDER BY OccurredAt DESC LIMIT @lim`
	default:
		return nil, fmt.Errorf("invalid_tenant")
	}
	iter := w.spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := make([]map[string]any, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var entryID, orderID, supplierID, retailerID, gateway, entryType, currency, refID, source string
		var amount int64
		var occurred time.Time
		if err := row.Columns(&entryID, &orderID, &supplierID, &retailerID, &gateway, &entryType,
			&amount, &currency, &refID, &source, &occurred); err != nil {
			return nil, err
		}
		debit, credit := mapPaymentJournalAccounts(coa, entryType)
		amt := absMinor(amount)
		memo := fmt.Sprintf("Payment %s", strings.ToUpper(entryType))
		if gateway != "" {
			memo += " via " + gateway
		}
		if refID != "" {
			memo += " ref=" + refID
		}
		_ = source
		out = append(out, journalRow(map[string]any{
			"entry_date":     occurred.UTC().Format(time.RFC3339),
			"source":         "payment",
			"entry_id":       entryID,
			"entry_type":     strings.ToUpper(entryType),
			"debit_account":  debit,
			"credit_account": credit,
			"amount_minor":   amt,
			"currency":       currency,
			"supplier_id":    supplierID,
			"retailer_id":    retailerID,
			"invoice_id":     "",
			"order_id":       orderID,
			"aging_bucket":   "",
			"gateway":        gateway,
			"memo":           memo,
		}))
	}
	return out, nil
}
