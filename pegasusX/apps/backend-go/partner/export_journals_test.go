package partner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMapARJournalAccounts(t *testing.T) {
	d, c := mapARJournalAccounts("OPEN")
	if d != coaAR || c != coaRevenue {
		t.Fatalf("OPEN: debit=%s credit=%s", d, c)
	}
	d, c = mapARJournalAccounts("PAYMENT")
	if d != coaBankCash || c != coaAR {
		t.Fatalf("PAYMENT: debit=%s credit=%s", d, c)
	}
}

func TestMapPaymentJournalAccounts(t *testing.T) {
	d, c := mapPaymentJournalAccounts("SESSION_CAPTURED")
	if d != coaBankCash || c != coaAR {
		t.Fatalf("capture: debit=%s credit=%s", d, c)
	}
	d, c = mapPaymentJournalAccounts("GATEWAY_REFUND")
	if d != coaAR || c != coaBankCash {
		t.Fatalf("refund: debit=%s credit=%s", d, c)
	}
	d, c = mapPaymentJournalAccounts("CHARGEBACK_RECORDED")
	if d != coaAR || c != coaBankCash {
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
			"debit_account":  coaAR,
			"credit_account": coaRevenue,
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
	svc := NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)
	svc.SetExportRepos(exports, sftp)
	worker := NewExportWorker(exports, sftp, nil, nil)

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
