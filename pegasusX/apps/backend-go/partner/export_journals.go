package partner

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
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
	"vat_minor",
	"currency",
	"supplier_id",
	"retailer_id",
	"invoice_id",
	"credit_note_id",
	"order_id",
	"aging_bucket",
	"gateway",
	"memo",
}

func mapARJournalAccounts(coa CoaMap, entryType string) (debit, credit string) {
	c := ResolveCoa(coa, true)
	ar := c.AccountAR
	rev := c.AccountRevenue
	bank := c.AccountBankCash
	arVoid := c.AccountARVoid
	revVoid := c.AccountRevVoid
	revSuspense := c.AccountReversal
	t := strings.ToUpper(strings.TrimSpace(entryType))
	switch t {
	case "OPEN":
		return ar, rev
	case "PAYMENT":
		return bank, ar
	case "CREDIT_NOTE", "CREDITNOTE", "CN":
		return rev, ar
	case "ADVANCE_CREDIT":
		return bank, ar
	case "VOID", "AR_VOID", "INVOICE_VOID":
		return revVoid, arVoid
	case "VOID_REVERSAL", "REVERSAL":
		return revSuspense, arVoid
	case "WRITE_OFF", "WRITEOFF":
		return revVoid, ar
	default:
		if strings.Contains(t, "REVERSAL") {
			return revSuspense, arVoid
		}
		if strings.Contains(t, "VOID") {
			return revVoid, arVoid
		}
		// Unknown non-void AR types: treat as AR increase vs suspense revenue.
		return ar, rev
	}
}

func mapPaymentJournalAccounts(coa CoaMap, entryType string) (debit, credit string) {
	c := ResolveCoa(coa, true)
	ar := c.AccountAR
	bank := c.AccountBankCash
	arVoid := c.AccountARVoid
	bankVoid := c.AccountBankVoid
	revSuspense := c.AccountReversal
	t := strings.ToUpper(strings.TrimSpace(entryType))
	if strings.Contains(t, "VOID") && strings.Contains(t, "REVERSAL") {
		return bankVoid, arVoid
	}
	if strings.Contains(t, "VOID") {
		return arVoid, bankVoid
	}
	if strings.Contains(t, "REVERSAL") {
		return bankVoid, revSuspense
	}
	if strings.Contains(t, "REFUND") || strings.Contains(t, "CHARGEBACK") {
		return ar, bank
	}
	// Capture / settle / paid / success → cash from AR.
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

type orderVATSummary struct {
	VATMinor   int64
	GrossMinor int64
}

type journalExportItem struct {
	Row            map[string]any
	Source         string // "ar", "credit_note", "payment"
	EntryID        string
	EntryType      string
	SupplierID     string
	RetailerID     string
	OrderID        string
	InvoiceID      string
	IdempotencyKey string
	ReferenceID    string
	AmountMinor    int64
	VATMinor       int64
	EntryDate      time.Time
}

type journalLineItem struct {
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price_minor"`
	VatRateBps  *int64 `json:"vat_rate_bps"`
	VatMinor    *int64 `json:"vat_minor"`
	TaxCategory string `json:"tax_category"`
}

func computeLineItemsVAT(lineItemsJSON []byte) (vatMinor int64, grossMinor int64, err error) {
	trimmed := strings.TrimSpace(string(lineItemsJSON))
	if trimmed == "" || trimmed == "null" {
		return 0, 0, nil
	}
	var items []journalLineItem
	if err := json.Unmarshal(lineItemsJSON, &items); err != nil {
		return 0, 0, err
	}
	for _, item := range items {
		if item.Quantity <= 0 || item.UnitPrice <= 0 {
			continue
		}
		net := item.Quantity * item.UnitPrice
		var lineVAT int64
		switch {
		case item.VatMinor != nil && *item.VatMinor >= 0:
			lineVAT = *item.VatMinor
		case item.VatRateBps != nil && *item.VatRateBps > 0:
			// Soliq OFD integer half-up rounding: (net * rate_bps + 5000) / 10000
			lineVAT = (net*(*item.VatRateBps) + 5000) / 10000
		}
		vatMinor += lineVAT
		grossMinor += net + lineVAT
	}
	return vatMinor, grossMinor, nil
}

func proportionalVATMinor(entryAmtMinor, baseGrossMinor, totalVATMinor int64) int64 {
	if totalVATMinor <= 0 || entryAmtMinor <= 0 {
		return 0
	}
	if baseGrossMinor <= 0 || entryAmtMinor >= baseGrossMinor {
		return totalVATMinor
	}
	// Overflow-safe integer half-up rounding: (totalVAT * entryAmt + base/2) / base
	num := new(big.Int).Mul(big.NewInt(totalVATMinor), big.NewInt(entryAmtMinor))
	half := new(big.Int).Quo(big.NewInt(baseGrossMinor), big.NewInt(2))
	num.Add(num, half)
	num.Quo(num, big.NewInt(baseGrossMinor))
	return num.Int64()
}

func (w *ExportWorker) batchFetchOrderVAT(ctx context.Context, ro *spanner.ReadOnlyTransaction, orderIDs []string) (map[string]orderVATSummary, error) {
	out := make(map[string]orderVATSummary, len(orderIDs))
	if w.spanner == nil || len(orderIDs) == 0 {
		return out, nil
	}
	seen := make(map[string]struct{}, len(orderIDs))
	uniq := make([]string, 0, len(orderIDs))
	for _, id := range orderIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return out, nil
	}

	if ro == nil {
		ro = w.spanner.ReadOnlyTransaction()
		defer ro.Close()
	}

	const chunkSize = 2000
	for start := 0; start < len(uniq); start += chunkSize {
		end := start + chunkSize
		if end > len(uniq) {
			end = len(uniq)
		}
		chunk := uniq[start:end]

		snapIter := ro.Query(ctx, spanner.Statement{
			SQL: `SELECT OrderId, COALESCE(SUM(VatMinor), 0), COALESCE(SUM(GrossMinor), 0)
				FROM OrderLineFiscalSnapshots
				WHERE OrderId IN UNNEST(@orderIds)
				GROUP BY OrderId`,
			Params: map[string]any{"orderIds": chunk},
		})
		for {
			row, err := snapIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				snapIter.Stop()
				return nil, fmt.Errorf("query order fiscal snapshots: %w", err)
			}
			var orderID string
			var vat, gross int64
			if err := row.Columns(&orderID, &vat, &gross); err != nil {
				snapIter.Stop()
				return nil, fmt.Errorf("scan order fiscal snapshots: %w", err)
			}
			out[orderID] = orderVATSummary{VATMinor: vat, GrossMinor: gross}
		}
		snapIter.Stop()

		missing := make([]string, 0)
		for _, id := range chunk {
			if _, ok := out[id]; !ok {
				missing = append(missing, id)
			}
		}
		if len(missing) == 0 {
			continue
		}

		ordIter := ro.Query(ctx, spanner.Statement{
			SQL: `SELECT OrderId, LineItemsJson, COALESCE(TotalMinor, 0)
				FROM Orders
				WHERE OrderId IN UNNEST(@orderIds)`,
			Params: map[string]any{"orderIds": missing},
		})
		for {
			row, err := ordIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				ordIter.Stop()
				return nil, fmt.Errorf("query order line items for VAT: %w", err)
			}
			var orderID string
			var lineItemsRaw []byte
			var totalMinor int64
			if err := row.Columns(&orderID, &lineItemsRaw, &totalMinor); err != nil {
				ordIter.Stop()
				return nil, fmt.Errorf("scan order line items for VAT: %w", err)
			}
			vat, gross, err := computeLineItemsVAT(lineItemsRaw)
			if err != nil {
				w.log.Warn("invalid order line_items_json while computing VAT", "order_id", orderID, "err", err)
				continue
			}
			if gross <= 0 {
				gross = totalMinor
			}
			if vat > 0 {
				out[orderID] = orderVATSummary{VATMinor: vat, GrossMinor: gross}
			}
		}
		ordIter.Stop()
	}
	return out, nil
}

func isPaymentCaptureEntryType(entryType string) bool {
	t := strings.ToUpper(strings.TrimSpace(entryType))
	if t == "" {
		return false
	}
	if strings.Contains(t, "VOID") || strings.Contains(t, "REVERSAL") ||
		strings.Contains(t, "REFUND") || strings.Contains(t, "CHARGEBACK") ||
		strings.Contains(t, "FAILED") || strings.Contains(t, "CANCEL") ||
		strings.Contains(t, "PENDING") {
		return false
	}
	return true
}

// deduplicateJournalItems removes duplicate PAYMENT exports with default MaxExportRows cap.
func deduplicateJournalItems(arItems, cnItems, payItems []journalExportItem) []map[string]any {
	return deduplicateJournalItemsWithLimit(arItems, cnItems, payItems, MaxExportRows)
}

// deduplicateJournalItemsWithLimit removes duplicate PAYMENT exports in O(N) using hash indexes.
// maxRows <= 0 disables truncation (unlimited streaming exports).
func deduplicateJournalItemsWithLimit(arItems, cnItems, payItems []journalExportItem, maxRows int) []map[string]any {
	suppressedAR := make([]bool, len(arItems))

	type pairKey struct {
		sup string
		ret string
		ref string
	}
	type orderAmtKey struct {
		sup string
		ret string
		ord string
		amt int64
	}

	arByExactRef := make(map[pairKey][]int, len(arItems))
	arByPrefixRef := make(map[pairKey][]int, len(arItems))
	arByOrderAmt := make(map[orderAmtKey][]int, len(arItems))

	for i := range arItems {
		a := arItems[i]
		if strings.ToUpper(strings.TrimSpace(a.EntryType)) != "PAYMENT" {
			continue
		}
		if a.IdempotencyKey != "" {
			ek := pairKey{sup: a.SupplierID, ret: a.RetailerID, ref: a.IdempotencyKey}
			arByExactRef[ek] = append(arByExactRef[ek], i)
			if colon := strings.IndexByte(a.IdempotencyKey, ':'); colon > 0 {
				for idx := colon; idx < len(a.IdempotencyKey); idx++ {
					if a.IdempotencyKey[idx] == ':' {
						pk := pairKey{sup: a.SupplierID, ret: a.RetailerID, ref: a.IdempotencyKey[:idx]}
						arByPrefixRef[pk] = append(arByPrefixRef[pk], i)
					}
				}
			}
		}
		if a.OrderID != "" {
			ok := orderAmtKey{sup: a.SupplierID, ret: a.RetailerID, ord: a.OrderID, amt: a.AmountMinor}
			arByOrderAmt[ok] = append(arByOrderAmt[ok], i)
		}
	}

	propagateARFieldsToPayment := func(p *journalExportItem, invID string, vatMinor int64) {
		if vatMinor > 0 && p.VATMinor == 0 {
			p.VATMinor = vatMinor
			p.Row["vat_minor"] = vatMinor
		}
		if invID != "" && strings.TrimSpace(p.InvoiceID) == "" {
			p.InvoiceID = invID
			p.Row["invoice_id"] = invID
		}
	}

	// Pass 1: exact ReferenceId <-> ArLedgerEntries.IdempotencyKey match (single or FIFO split slices).
	matchedPay := make([]bool, len(payItems))
	for pi := range payItems {
		p := &payItems[pi]
		if !isPaymentCaptureEntryType(p.EntryType) || p.ReferenceID == "" {
			continue
		}
		matchedSingle := false
		for _, ai := range arByExactRef[pairKey{sup: p.SupplierID, ret: p.RetailerID, ref: p.ReferenceID}] {
			if suppressedAR[ai] {
				continue
			}
			a := arItems[ai]
			if a.AmountMinor == p.AmountMinor {
				suppressedAR[ai] = true
				propagateARFieldsToPayment(p, a.InvoiceID, a.VATMinor)
				matchedPay[pi] = true
				matchedSingle = true
				break
			}
		}
		if matchedSingle {
			continue
		}
		// Check prefix slices (IdempotencyKey starting with ReferenceId + ":").
		prefCandidates := arByPrefixRef[pairKey{sup: p.SupplierID, ret: p.RetailerID, ref: p.ReferenceID}]
		for _, ai := range prefCandidates {
			if suppressedAR[ai] {
				continue
			}
			a := arItems[ai]
			if a.AmountMinor == p.AmountMinor {
				suppressedAR[ai] = true
				propagateARFieldsToPayment(p, a.InvoiceID, a.VATMinor)
				matchedPay[pi] = true
				matchedSingle = true
				break
			}
		}
		if matchedSingle {
			continue
		}
		var sliceIdx []int
		var sumAmt, sumVAT int64
		for _, ai := range prefCandidates {
			if suppressedAR[ai] {
				continue
			}
			a := arItems[ai]
			sliceIdx = append(sliceIdx, ai)
			sumAmt += a.AmountMinor
			sumVAT += a.VATMinor
		}
		if len(sliceIdx) > 0 && sumAmt == p.AmountMinor {
			invID := ""
			for _, ai := range sliceIdx {
				suppressedAR[ai] = true
				if invID == "" {
					invID = arItems[ai].InvoiceID
				}
			}
			propagateARFieldsToPayment(p, invID, sumVAT)
			matchedPay[pi] = true
		}
	}

	// Pass 2: deterministic (SupplierId, RetailerId, OrderId, AmountMinor) multiset match.
	for pi := range payItems {
		if matchedPay[pi] {
			continue
		}
		p := &payItems[pi]
		if !isPaymentCaptureEntryType(p.EntryType) || p.OrderID == "" {
			continue
		}
		for _, ai := range arByOrderAmt[orderAmtKey{sup: p.SupplierID, ret: p.RetailerID, ord: p.OrderID, amt: p.AmountMinor}] {
			if suppressedAR[ai] {
				continue
			}
			a := arItems[ai]
			suppressedAR[ai] = true
			propagateARFieldsToPayment(p, a.InvoiceID, a.VATMinor)
			matchedPay[pi] = true
			break
		}
	}

	combined := make([]journalExportItem, 0, len(arItems)+len(cnItems)+len(payItems))
	seenIDs := make(map[string]struct{}, len(arItems)+len(cnItems)+len(payItems))
	addUnique := func(it journalExportItem) {
		if it.EntryID != "" {
			k := it.Source + "|" + it.EntryID
			if _, ok := seenIDs[k]; ok {
				return
			}
			seenIDs[k] = struct{}{}
		}
		combined = append(combined, it)
	}
	for i, it := range arItems {
		if suppressedAR[i] {
			continue
		}
		addUnique(it)
	}
	for _, it := range cnItems {
		addUnique(it)
	}
	for _, it := range payItems {
		addUnique(it)
	}

	sort.Slice(combined, func(i, j int) bool {
		if !combined[i].EntryDate.Equal(combined[j].EntryDate) {
			return combined[i].EntryDate.After(combined[j].EntryDate)
		}
		if combined[i].Source != combined[j].Source {
			return combined[i].Source < combined[j].Source
		}
		return combined[i].EntryID < combined[j].EntryID
	})

	limit := len(combined)
	if maxRows > 0 && limit > maxRows {
		limit = maxRows
	}
	out := make([]map[string]any, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, combined[i].Row)
	}
	return out
}

func (w *ExportWorker) exportJournals(ctx context.Context, tenantType, tenantID string, from, to time.Time, unlimited bool) ([]map[string]any, error) {
	if w.spanner == nil {
		return []map[string]any{}, nil
	}
	coa := w.resolveTenantCoa(ctx, tenantType, tenantID)
	ro := w.spanner.ReadOnlyTransaction()
	defer ro.Close()

	arItems, err := w.exportJournalsAR(ctx, ro, tenantType, tenantID, from, to, unlimited, coa)
	if err != nil {
		return nil, fmt.Errorf("export journals ar: %w", err)
	}
	cnItems, err := w.exportJournalsCreditNotes(ctx, ro, tenantType, tenantID, from, to, unlimited, coa)
	if err != nil {
		return nil, fmt.Errorf("export journals credit notes: %w", err)
	}
	payItems, err := w.exportJournalsPayment(ctx, ro, tenantType, tenantID, from, to, unlimited, coa)
	if err != nil {
		return nil, fmt.Errorf("export journals payment: %w", err)
	}
	maxRows := MaxExportRows
	if unlimited {
		maxRows = 0
	}
	return deduplicateJournalItemsWithLimit(arItems, cnItems, payItems, maxRows), nil
}

type rawARJournalRow struct {
	entryID        string
	invID          string
	supplierID     string
	retailerID     string
	entryType      string
	amount         int64
	refOrder       string
	idempotencyKey string
	created        time.Time
	currency       string
	orderID        string
	aging          string
	invStatus      string
	principalMinor int64
}

func (w *ExportWorker) exportJournalsAR(ctx context.Context, ro *spanner.ReadOnlyTransaction, tenantType, tenantID string, from, to time.Time, unlimited bool, coa CoaMap) ([]journalExportItem, error) {
	var sql string
	switch tenantType {
	case TenantSupplier:
		sql = `SELECT e.EntryId, e.InvoiceId, e.SupplierId, e.RetailerId, e.EntryType, e.AmountMinor,
			COALESCE(e.RefOrderId, ''), COALESCE(e.IdempotencyKey, ''), e.CreatedAt,
			COALESCE(i.Currency, ''), COALESCE(i.OrderId, ''), COALESCE(i.AgingBucket, ''), COALESCE(i.Status, ''), COALESCE(i.PrincipalMinor, 0)
			FROM ArLedgerEntries e
			LEFT JOIN ArInvoices i ON i.InvoiceId = e.InvoiceId
			WHERE e.SupplierId = @tid AND e.CreatedAt >= @from AND e.CreatedAt <= @to
			AND (@cursor = '' OR e.EntryId > @cursor)
			ORDER BY e.EntryId ASC LIMIT @lim`
	case TenantRetailer:
		sql = `SELECT e.EntryId, e.InvoiceId, e.SupplierId, e.RetailerId, e.EntryType, e.AmountMinor,
			COALESCE(e.RefOrderId, ''), COALESCE(e.IdempotencyKey, ''), e.CreatedAt,
			COALESCE(i.Currency, ''), COALESCE(i.OrderId, ''), COALESCE(i.AgingBucket, ''), COALESCE(i.Status, ''), COALESCE(i.PrincipalMinor, 0)
			FROM ArLedgerEntries e
			LEFT JOIN ArInvoices i ON i.InvoiceId = e.InvoiceId
			WHERE e.RetailerId = @tid AND e.CreatedAt >= @from AND e.CreatedAt <= @to
			AND (@cursor = '' OR e.EntryId > @cursor)
			ORDER BY e.EntryId ASC LIMIT @lim`
	default:
		return nil, fmt.Errorf("invalid_tenant")
	}

	batchSize := w.pageLimit()
	cursor := ""
	rawRows := make([]rawARJournalRow, 0)
	orderIDs := make([]string, 0)
	for {
		lim := nextBatchSize(len(rawRows), batchSize, unlimited)
		if lim <= 0 {
			break
		}
		iter := ro.Query(ctx, spanner.Statement{
			SQL:    sql,
			Params: map[string]any{"tid": tenantID, "from": from, "to": to, "cursor": cursor, "lim": int64(lim)},
		})
		count := 0
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return nil, err
			}
			var rr rawARJournalRow
			if err := row.Columns(&rr.entryID, &rr.invID, &rr.supplierID, &rr.retailerID, &rr.entryType, &rr.amount,
				&rr.refOrder, &rr.idempotencyKey, &rr.created, &rr.currency, &rr.orderID, &rr.aging, &rr.invStatus, &rr.principalMinor); err != nil {
				iter.Stop()
				return nil, err
			}
			if rr.orderID == "" {
				rr.orderID = rr.refOrder
			}
			if rr.orderID != "" {
				orderIDs = append(orderIDs, rr.orderID)
			}
			rawRows = append(rawRows, rr)
			cursor = rr.entryID
			count++
		}
		iter.Stop()
		if count < lim {
			break
		}
	}

	vatByOrder, err := w.batchFetchOrderVAT(ctx, ro, orderIDs)
	if err != nil {
		return nil, err
	}

	out := make([]journalExportItem, 0, len(rawRows))
	for _, rr := range rawRows {
		entryUpper := strings.ToUpper(strings.TrimSpace(rr.entryType))
		debit, credit := mapARJournalAccounts(coa, entryUpper)
		amt := absMinor(rr.amount)
		var vatMinor int64
		if summary, ok := vatByOrder[rr.orderID]; ok && summary.VATMinor > 0 {
			baseGross := rr.principalMinor
			if baseGross <= 0 {
				baseGross = summary.GrossMinor
			}
			vatMinor = proportionalVATMinor(amt, baseGross, summary.VATMinor)
		}
		memo := fmt.Sprintf("AR %s invoice=%s", entryUpper, rr.invID)
		if rr.invStatus != "" {
			memo += " status=" + rr.invStatus
		}
		rowMap := journalRow(map[string]any{
			"entry_date":     rr.created.UTC().Format(time.RFC3339),
			"source":         "ar",
			"entry_id":       rr.entryID,
			"entry_type":     entryUpper,
			"debit_account":  debit,
			"credit_account": credit,
			"amount_minor":   amt,
			"vat_minor":      vatMinor,
			"currency":       journalCurrency(ctx, rr.supplierID, rr.currency),
			"supplier_id":    rr.supplierID,
			"retailer_id":    rr.retailerID,
			"invoice_id":     rr.invID,
			"credit_note_id": "",
			"order_id":       rr.orderID,
			"aging_bucket":   rr.aging,
			"gateway":        "",
			"memo":           memo,
		})
		out = append(out, journalExportItem{
			Row:            rowMap,
			Source:         "ar",
			EntryID:        rr.entryID,
			EntryType:      entryUpper,
			SupplierID:     rr.supplierID,
			RetailerID:     rr.retailerID,
			OrderID:        rr.orderID,
			InvoiceID:      rr.invID,
			IdempotencyKey: rr.idempotencyKey,
			AmountMinor:    amt,
			VATMinor:       vatMinor,
			EntryDate:      rr.created.UTC(),
		})
	}
	return out, nil
}

func creditNotesJournalQuery(tenantType string) (string, error) {
	switch tenantType {
	case TenantSupplier:
		return `SELECT cn.CreditNoteId, cn.OrderId, o.SupplierId, o.RetailerId,
			cn.TotalNetMinor, cn.TotalVatMinor, cn.TotalGrossMinor, COALESCE(o.Currency, ''), cn.CreatedAt
			FROM CreditNotes cn
			JOIN Orders o ON o.OrderId = cn.OrderId
			WHERE o.SupplierId = @tid AND cn.CreatedAt >= @from AND cn.CreatedAt <= @to
			AND (@cursor = '' OR cn.CreditNoteId > @cursor)
			ORDER BY cn.CreditNoteId ASC LIMIT @lim`, nil
	case TenantRetailer:
		return `SELECT cn.CreditNoteId, cn.OrderId, o.SupplierId, o.RetailerId,
			cn.TotalNetMinor, cn.TotalVatMinor, cn.TotalGrossMinor, COALESCE(o.Currency, ''), cn.CreatedAt
			FROM CreditNotes cn
			JOIN Orders o ON o.OrderId = cn.OrderId
			WHERE o.RetailerId = @tid AND cn.CreatedAt >= @from AND cn.CreatedAt <= @to
			AND (@cursor = '' OR cn.CreditNoteId > @cursor)
			ORDER BY cn.CreditNoteId ASC LIMIT @lim`, nil
	default:
		return "", fmt.Errorf("invalid_tenant")
	}
}

func (w *ExportWorker) exportJournalsCreditNotes(ctx context.Context, ro *spanner.ReadOnlyTransaction, tenantType, tenantID string, from, to time.Time, unlimited bool, coa CoaMap) ([]journalExportItem, error) {
	sql, err := creditNotesJournalQuery(tenantType)
	if err != nil {
		return nil, err
	}
	batchSize := w.pageLimit()
	cursor := ""
	out := make([]journalExportItem, 0)
	for {
		lim := nextBatchSize(len(out), batchSize, unlimited)
		if lim <= 0 {
			break
		}
		iter := ro.Query(ctx, spanner.Statement{
			SQL:    sql,
			Params: map[string]any{"tid": tenantID, "from": from, "to": to, "cursor": cursor, "lim": int64(lim)},
		})
		count := 0
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return nil, err
			}
			var cnID, orderID, supplierID, retailerID, currency string
			var net, vat, gross int64
			var created time.Time
			if err := row.Columns(&cnID, &orderID, &supplierID, &retailerID, &net, &vat, &gross, &currency, &created); err != nil {
				iter.Stop()
				return nil, err
			}
			debit, credit := mapARJournalAccounts(coa, "CREDIT_NOTE")
			amt := absMinor(gross)
			vatAbs := absMinor(vat)
			rowMap := journalRow(map[string]any{
				"entry_date":     created.UTC().Format(time.RFC3339),
				"source":         "credit_note",
				"entry_id":       cnID,
				"entry_type":     "CREDIT_NOTE",
				"debit_account":  debit,
				"credit_account": credit,
				"amount_minor":   amt,
				"vat_minor":      vatAbs,
				"currency":       journalCurrency(ctx, supplierID, currency),
				"supplier_id":    supplierID,
				"retailer_id":    retailerID,
				"invoice_id":     "",
				"credit_note_id": cnID,
				"order_id":       orderID,
				"aging_bucket":   "",
				"gateway":        "",
				"memo":           fmt.Sprintf("Credit note net=%d vat=%d", net, vat),
			})
			out = append(out, journalExportItem{
				Row:         rowMap,
				Source:      "credit_note",
				EntryID:     cnID,
				EntryType:   "CREDIT_NOTE",
				SupplierID:  supplierID,
				RetailerID:  retailerID,
				OrderID:     orderID,
				AmountMinor: amt,
				VATMinor:    vatAbs,
				EntryDate:   created.UTC(),
			})
			cursor = cnID
			count++
		}
		iter.Stop()
		if count < lim {
			break
		}
	}
	return out, nil
}

func (w *ExportWorker) exportJournalsPayment(ctx context.Context, ro *spanner.ReadOnlyTransaction, tenantType, tenantID string, from, to time.Time, unlimited bool, coa CoaMap) ([]journalExportItem, error) {
	var sql string
	switch tenantType {
	case TenantSupplier:
		sql = `SELECT LedgerEntryId, COALESCE(OrderId, ''), SupplierId, RetailerId, COALESCE(Gateway, ''),
			EntryType, AmountMinor, COALESCE(Currency, ''), COALESCE(ReferenceId, ''), COALESCE(Source, ''), OccurredAt
			FROM PaymentLedgerEntries
			WHERE SupplierId = @tid AND OccurredAt >= @from AND OccurredAt <= @to
			AND (@cursor = '' OR LedgerEntryId > @cursor)
			ORDER BY LedgerEntryId ASC LIMIT @lim`
	case TenantRetailer:
		sql = `SELECT LedgerEntryId, COALESCE(OrderId, ''), SupplierId, RetailerId, COALESCE(Gateway, ''),
			EntryType, AmountMinor, COALESCE(Currency, ''), COALESCE(ReferenceId, ''), COALESCE(Source, ''), OccurredAt
			FROM PaymentLedgerEntries
			WHERE RetailerId = @tid AND OccurredAt >= @from AND OccurredAt <= @to
			AND (@cursor = '' OR LedgerEntryId > @cursor)
			ORDER BY LedgerEntryId ASC LIMIT @lim`
	default:
		return nil, fmt.Errorf("invalid_tenant")
	}

	batchSize := w.pageLimit()
	cursor := ""
	out := make([]journalExportItem, 0)
	for {
		lim := nextBatchSize(len(out), batchSize, unlimited)
		if lim <= 0 {
			break
		}
		iter := ro.Query(ctx, spanner.Statement{
			SQL:    sql,
			Params: map[string]any{"tid": tenantID, "from": from, "to": to, "cursor": cursor, "lim": int64(lim)},
		})
		count := 0
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return nil, err
			}
			var entryID, orderID, supplierID, retailerID, gateway, entryType, currency, refID, source string
			var amount int64
			var occurred time.Time
			if err := row.Columns(&entryID, &orderID, &supplierID, &retailerID, &gateway, &entryType,
				&amount, &currency, &refID, &source, &occurred); err != nil {
				iter.Stop()
				return nil, err
			}
			entryUpper := strings.ToUpper(strings.TrimSpace(entryType))
			debit, credit := mapPaymentJournalAccounts(coa, entryUpper)
			amt := absMinor(amount)
			memo := fmt.Sprintf("Payment %s", entryUpper)
			if gateway != "" {
				memo += " via " + gateway
			}
			if refID != "" {
				memo += " ref=" + refID
			}
			if source != "" {
				memo += " src=" + source
			}
			rowMap := journalRow(map[string]any{
				"entry_date":     occurred.UTC().Format(time.RFC3339),
				"source":         "payment",
				"entry_id":       entryID,
				"entry_type":     entryUpper,
				"debit_account":  debit,
				"credit_account": credit,
				"amount_minor":   amt,
				"vat_minor":      int64(0),
				"currency":       journalCurrency(ctx, supplierID, currency),
				"supplier_id":    supplierID,
				"retailer_id":    retailerID,
				"invoice_id":     "",
				"credit_note_id": "",
				"order_id":       orderID,
				"aging_bucket":   "",
				"gateway":        gateway,
				"memo":           memo,
			})
			out = append(out, journalExportItem{
				Row:         rowMap,
				Source:      "payment",
				EntryID:     entryID,
				EntryType:   entryUpper,
				SupplierID:  supplierID,
				RetailerID:  retailerID,
				OrderID:     orderID,
				ReferenceID: refID,
				AmountMinor: amt,
				EntryDate:   occurred.UTC(),
			})
			cursor = entryID
			count++
		}
		iter.Stop()
		if count < lim {
			break
		}
	}
	return out, nil
}

// journalCurrency is empty-currency law for partner journals: stored ISO, else
// the shipped pack. Planned/unknown packs stay empty — never invent UZS.
func journalCurrency(ctx context.Context, supplierID, stored string) string {
	c, err := auth.CoalesceCurrency(ctx, supplierID, stored)
	if err != nil {
		return strings.ToUpper(strings.TrimSpace(stored))
	}
	return c
}
