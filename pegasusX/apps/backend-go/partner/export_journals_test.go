package partner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMapARJournalAccounts(t *testing.T) {
	coa := DefaultCoa()
	d, c := mapARJournalAccounts(coa, "OPEN")
	if d != DefaultCoaAR || c != DefaultCoaRevenue {
		t.Fatalf("OPEN: debit=%s credit=%s", d, c)
	}
	d, c = mapARJournalAccounts(coa, "PAYMENT")
	if d != DefaultCoaBankCash || c != DefaultCoaAR {
		t.Fatalf("PAYMENT: debit=%s credit=%s", d, c)
	}
}

func TestMapPaymentJournalAccounts(t *testing.T) {
	coa := DefaultCoa()
	d, c := mapPaymentJournalAccounts(coa, "SESSION_CAPTURED")
	if d != DefaultCoaBankCash || c != DefaultCoaAR {
		t.Fatalf("capture: debit=%s credit=%s", d, c)
	}
	d, c = mapPaymentJournalAccounts(coa, "GATEWAY_REFUND")
	if d != DefaultCoaAR || c != DefaultCoaBankCash {
		t.Fatalf("refund: debit=%s credit=%s", d, c)
	}
	d, c = mapPaymentJournalAccounts(coa, "CHARGEBACK_RECORDED")
	if d != DefaultCoaAR || c != DefaultCoaBankCash {
		t.Fatalf("chargeback: debit=%s credit=%s", d, c)
	}
}

func TestEncodeExportXMLJournal(t *testing.T) {
	rows := []map[string]any{
		journalRow(map[string]any{
			"entry_date":     time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
			"source":         "ar",
			"entry_id":       "e1",
			"entry_type":     "OPEN",
			"debit_account":  DefaultCoaAR,
			"credit_account": DefaultCoaRevenue,
			"amount_minor":   int64(1000),
			"currency":       "UZS",
			"supplier_id":    "sup1",
			"retailer_id":    "ret1",
			"invoice_id":     "inv1",
			"order_id":       "ord1",
			"aging_bucket":   "CURRENT",
			"gateway":        "",
			"memo":           `AR OPEN "quote" & amp`,
		}),
	}
	body, ct, err := encodeExport(rows, ExportFormatXML)
	if err != nil {
		t.Fatal(err)
	}
	if ct != "application/xml" {
		t.Fatalf("content-type=%s", ct)
	}
	s := string(body)
	if !strings.Contains(s, `<Journal version="1" dialect="1c">`) {
		t.Fatalf("missing journal root: %s", s)
	}
	if !strings.Contains(s, `debit_account="62.01"`) || !strings.Contains(s, `credit_account="90.01"`) {
		t.Fatalf("missing accounts: %s", s)
	}
	if !strings.Contains(s, `&quot;`) || !strings.Contains(s, `&amp;`) {
		t.Fatalf("expected XML attr escape: %s", s)
	}
}

func TestCreateExportJobJournalsXML(t *testing.T) {
	t.Setenv("PARTNER_EXPORTS_ENABLED", "true")
	exports := NewMemoryExportRepository()
	svc := NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)
	svc.SetExportRepos(exports, NewMemorySftpConfigRepository())
	p := Principal{TenantType: TenantSupplier, TenantID: "sup-j", Scopes: []string{ScopeExportsRead}}
	j, err := svc.CreateExportJob(context.Background(), p, ExportResourceJournals, ExportFormatXML, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if j.Resource != ExportResourceJournals || j.Format != ExportFormatXML {
		t.Fatalf("job=%+v", j)
	}
}

func TestExportJournalsWorkerEmptySpanner(t *testing.T) {
	t.Setenv("PARTNER_EXPORTS_ENABLED", "true")
	t.Setenv("PARTNER_SFTP_ENABLED", "false")
	root := t.TempDir()
	t.Setenv("PARTNER_EXPORT_LOCAL_ROOT", root)

	exports := NewMemoryExportRepository()
	sftp := NewMemorySftpConfigRepository()
	coa := NewMemoryCoaRepository()
	svc := NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)
	svc.SetExportRepos(exports, sftp)
	svc.SetCoaRepository(coa)
	worker := NewExportWorker(exports, sftp, nil, nil)
	worker.SetCoaRepository(coa)

	p := Principal{TenantType: TenantSupplier, TenantID: "sup-j2", Scopes: []string{ScopeExportsRead}}
	j, err := svc.CreateExportJob(context.Background(), p, ExportResourceJournals, ExportFormatCSV, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	n, err := worker.RunOnce(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("worker n=%d err=%v", n, err)
	}
	got, _, err := svc.GetExportJob(context.Background(), p, j.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ExportStatusSucceeded {
		t.Fatalf("status=%s err=%s", got.Status, got.Error)
	}
	if !strings.HasSuffix(got.ObjectPath, ".csv") {
		t.Fatalf("path=%s", got.ObjectPath)
	}
}

func TestJournalCurrency_EmptyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if got := journalCurrency(context.Background(), "sup-j", ""); got != "UZS" {
		t.Fatalf("got %q want UZS from pack", got)
	}
}

func TestJournalCurrency_StoredWins(t *testing.T) {
	if got := journalCurrency(context.Background(), "sup-j", "eur"); got != "EUR" {
		t.Fatalf("got %q want EUR", got)
	}
}

func TestJournalCurrency_PlannedDoesNotInvent(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "EU")
	if got := journalCurrency(context.Background(), "sup-j", ""); got != "" {
		t.Fatalf("planned pack must not invent UZS, got %q", got)
	}
}

func TestCreditNotesJournalQuery_EmptyCurrencyNoInvent(t *testing.T) {
	for _, tenant := range []string{TenantSupplier, TenantRetailer} {
		sql, err := creditNotesJournalQuery(tenant)
		if err != nil {
			t.Fatalf("%s: %v", tenant, err)
		}
		if strings.Contains(sql, "'UZS'") || strings.Contains(sql, `"UZS"`) {
			t.Fatalf("%s SQL must not invent UZS: %s", tenant, sql)
		}
		if !strings.Contains(sql, "COALESCE(o.Currency, '')") {
			t.Fatalf("%s want empty COALESCE, got %s", tenant, sql)
		}
	}
	if _, err := creditNotesJournalQuery("bogus"); err == nil {
		t.Fatal("expected invalid_tenant")
	}
}

func TestMapARJournalAccounts_VoidAndReversalDistinct(t *testing.T) {
	coa := DefaultCoa()
	for _, et := range []string{"VOID", "AR_VOID", "INVOICE_VOID"} {
		d, c := mapARJournalAccounts(coa, et)
		if d == DefaultCoaAR || d == DefaultCoaRevenue || d == DefaultCoaBankCash ||
			c == DefaultCoaAR || c == DefaultCoaRevenue || c == DefaultCoaBankCash {
			t.Fatalf("%s leaked into normal accounts: debit=%s credit=%s", et, d, c)
		}
		if d != DefaultCoaRevenueReversal || c != DefaultCoaARReversal {
			t.Fatalf("%s: got %s/%s want %s/%s", et, d, c, DefaultCoaRevenueReversal, DefaultCoaARReversal)
		}
	}
	for _, et := range []string{"REVERSAL", "VOID_REVERSAL"} {
		d, c := mapARJournalAccounts(coa, et)
		if d == DefaultCoaAR || d == DefaultCoaRevenue || c == DefaultCoaAR || c == DefaultCoaRevenue {
			t.Fatalf("%s leaked into normal accounts: debit=%s credit=%s", et, d, c)
		}
	}
}

func TestMapPaymentJournalAccounts_VoidAndReversalDistinct(t *testing.T) {
	coa := DefaultCoa()
	for _, et := range []string{"VOID", "SESSION_VOID", "PAYMENT_VOIDED"} {
		d, c := mapPaymentJournalAccounts(coa, et)
		if d == DefaultCoaBankCash || c == DefaultCoaBankCash || d == DefaultCoaAR || c == DefaultCoaAR ||
			d == DefaultCoaRevenue || c == DefaultCoaRevenue {
			t.Fatalf("%s leaked into normal payment/AR/rev accounts: debit=%s credit=%s", et, d, c)
		}
	}
	for _, et := range []string{"CHARGEBACK_REVERSAL_RECORDED", "PAYMENT_REVERSAL", "VOID_REVERSAL"} {
		d, c := mapPaymentJournalAccounts(coa, et)
		if d == DefaultCoaBankCash && c == DefaultCoaAR {
			t.Fatalf("%s must remain strictly distinct from captured payment (%s/%s)", et, d, c)
		}
		if d == DefaultCoaAR || c == DefaultCoaAR || d == DefaultCoaBankCash || c == DefaultCoaBankCash {
			t.Fatalf("%s should not use normal AR/Bank cash accounts: debit=%s credit=%s", et, d, c)
		}
	}
}

func TestComputeLineItemsVATAndProportionalVAT(t *testing.T) {
	raw := []byte(`[
		{"sku":"A","quantity":2,"unit_price_minor":50000,"vat_rate_bps":1200},
		{"sku":"B","quantity":1,"unit_price_minor":20000,"tax_category":"EXEMPT"}
	]`)
	vat, gross, err := computeLineItemsVAT(raw)
	if err != nil {
		t.Fatal(err)
	}
	if vat != 12000 || gross != 132000 {
		t.Fatalf("got vat=%d gross=%d want vat=12000 gross=132000", vat, gross)
	}
	if part := proportionalVATMinor(66000, gross, vat); part != 6000 {
		t.Fatalf("proportional half VAT got %d want 6000", part)
	}
	if full := proportionalVATMinor(132000, gross, vat); full != 12000 {
		t.Fatalf("full VAT got %d want 12000", full)
	}
	explicit := []byte(`[{"sku":"C","quantity":1,"unit_price_minor":10000,"vat_minor":1200}]`)
	vat2, gross2, err := computeLineItemsVAT(explicit)
	if err != nil || vat2 != 1200 || gross2 != 11200 {
		t.Fatalf("explicit vat got vat=%d gross=%d err=%v", vat2, gross2, err)
	}
}

func TestDeduplicateJournalItems(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	mkRow := func(src, id, et, inv, ord string, amt, vat int64) map[string]any {
		return journalRow(map[string]any{
			"entry_date":     now.Format(time.RFC3339),
			"source":         src,
			"entry_id":       id,
			"entry_type":     et,
			"debit_account":  DefaultCoaBankCash,
			"credit_account": DefaultCoaAR,
			"amount_minor":   amt,
			"vat_minor":      vat,
			"currency":       "UZS",
			"supplier_id":    "sup-1",
			"retailer_id":    "ret-1",
			"invoice_id":     inv,
			"credit_note_id": "",
			"order_id":       ord,
		})
	}
	arItems := []journalExportItem{
		// Exact ref match with pay-1 -> must be suppressed, carrying vat_minor=6000 to pay-1.
		{Row: mkRow("ar", "ar-pay-1", "PAYMENT", "inv-1", "ord-1", 56000, 6000), Source: "ar", EntryID: "ar-pay-1", EntryType: "PAYMENT", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-1", InvoiceID: "inv-1", IdempotencyKey: "gw-ref-1:inv-1", AmountMinor: 56000, VATMinor: 6000, EntryDate: now},
		// Order+amount match with pay-2 (both ord-2 payments of 25000 must stay count=2, not 4 and not 1).
		{Row: mkRow("ar", "ar-pay-2a", "PAYMENT", "inv-2", "ord-2", 25000, 0), Source: "ar", EntryID: "ar-pay-2a", EntryType: "PAYMENT", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-2", InvoiceID: "inv-2", IdempotencyKey: "k2a", AmountMinor: 25000, EntryDate: now},
		{Row: mkRow("ar", "ar-pay-2b", "PAYMENT", "inv-2", "ord-2", 25000, 0), Source: "ar", EntryID: "ar-pay-2b", EntryType: "PAYMENT", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-2", InvoiceID: "inv-2", IdempotencyKey: "k2b", AmountMinor: 25000, EntryDate: now},
		// Manual AR-only cash settlement with no PaymentLedgerEntries row -> must stay!
		{Row: mkRow("ar", "ar-pay-manual", "PAYMENT", "inv-3", "ord-3", 10000, 0), Source: "ar", EntryID: "ar-pay-manual", EntryType: "PAYMENT", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-3", InvoiceID: "inv-3", IdempotencyKey: "manual-cash", AmountMinor: 10000, EntryDate: now},
		// Invoice OPEN must never be suppressed.
		{Row: mkRow("ar", "ar-open-1", "OPEN", "inv-1", "ord-1", 56000, 6000), Source: "ar", EntryID: "ar-open-1", EntryType: "OPEN", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-1", InvoiceID: "inv-1", AmountMinor: 56000, VATMinor: 6000, EntryDate: now},
	}
	payItems := []journalExportItem{
		{Row: mkRow("payment", "pay-1", "SESSION_CAPTURED", "", "ord-1", 56000, 0), Source: "payment", EntryID: "pay-1", EntryType: "SESSION_CAPTURED", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-1", ReferenceID: "gw-ref-1", AmountMinor: 56000, EntryDate: now},
		{Row: mkRow("payment", "pay-2a", "WEBHOOK_PAID", "", "ord-2", 25000, 0), Source: "payment", EntryID: "pay-2a", EntryType: "WEBHOOK_PAID", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-2", ReferenceID: "gw-ref-2a", AmountMinor: 25000, EntryDate: now},
		{Row: mkRow("payment", "pay-2b", "WEBHOOK_PAID", "", "ord-2", 25000, 0), Source: "payment", EntryID: "pay-2b", EntryType: "WEBHOOK_PAID", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-2", ReferenceID: "gw-ref-2b", AmountMinor: 25000, EntryDate: now},
	}
	rows := deduplicateJournalItems(arItems, nil, payItems)
	// Expected surviving entries: pay-1, pay-2a, pay-2b, ar-pay-manual, ar-open-1 = 5 rows.
	if len(rows) != 5 {
		t.Fatalf("got %d rows want 5: %+v", len(rows), rows)
	}
	seen := map[string]map[string]any{}
	for _, r := range rows {
		seen[r["entry_id"].(string)] = r
	}
	if _, dup := seen["ar-pay-1"]; dup {
		t.Fatal("duplicate AR payment ar-pay-1 should be suppressed")
	}
	if _, dup := seen["ar-pay-2a"]; dup {
		t.Fatal("duplicate AR payment ar-pay-2a should be suppressed")
	}
	if _, dup := seen["ar-pay-2b"]; dup {
		t.Fatal("duplicate AR payment ar-pay-2b should be suppressed")
	}
	if _, ok := seen["ar-pay-manual"]; !ok {
		t.Fatal("unmatched AR payment ar-pay-manual must survive")
	}
	if seen["pay-1"]["vat_minor"] != int64(6000) || seen["pay-1"]["invoice_id"] != "inv-1" {
		t.Fatalf("expected pay-1 to inherit vat/invoice from matched AR row, got %+v", seen["pay-1"])
	}
}
