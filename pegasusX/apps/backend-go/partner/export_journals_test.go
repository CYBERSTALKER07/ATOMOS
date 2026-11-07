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
