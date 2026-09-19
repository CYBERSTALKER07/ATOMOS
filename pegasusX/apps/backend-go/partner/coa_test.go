package partner

import (
	"context"
	"testing"
)

func TestMapARJournalAccounts_DefaultCoa(t *testing.T) {
	coa := DefaultCoa()
	d, c := mapARJournalAccounts(coa, "OPEN")
	if d != DefaultCoaAR || c != DefaultCoaRevenue {
		t.Fatalf("OPEN: %s/%s", d, c)
	}
	d, c = mapARJournalAccounts(coa, "PAYMENT")
	if d != DefaultCoaBankCash || c != DefaultCoaAR {
		t.Fatalf("PAYMENT: %s/%s", d, c)
	}
}

func TestMapPaymentJournalAccounts_DefaultCoa(t *testing.T) {
	coa := DefaultCoa()
	d, c := mapPaymentJournalAccounts(coa, "SESSION_CAPTURED")
	if d != DefaultCoaBankCash || c != DefaultCoaAR {
		t.Fatalf("capture: %s/%s", d, c)
	}
	d, c = mapPaymentJournalAccounts(coa, "GATEWAY_REFUND")
	if d != DefaultCoaAR || c != DefaultCoaBankCash {
		t.Fatalf("refund: %s/%s", d, c)
	}
	d, c = mapPaymentJournalAccounts(coa, "CHARGEBACK_RECORDED")
	if d != DefaultCoaAR || c != DefaultCoaBankCash {
		t.Fatalf("chargeback: %s/%s", d, c)
	}
}

func TestMapARJournalAccounts_CustomCoa(t *testing.T) {
	coa := CoaMap{AccountAR: "62.02", AccountRevenue: "90.05", AccountBankCash: "51.99"}
	d, c := mapARJournalAccounts(coa, "OPEN")
	if d != "62.02" || c != "90.05" {
		t.Fatalf("custom OPEN: %s/%s", d, c)
	}
	d, c = mapPaymentJournalAccounts(coa, "SESSION_CAPTURED")
	if d != "51.99" || c != "62.02" {
		t.Fatalf("custom capture: %s/%s", d, c)
	}
}

func TestResolveCoa_DefaultsAndMerge(t *testing.T) {
	got := ResolveCoa(CoaMap{}, false)
	if !got.UsingDefaults || got.AccountAR != DefaultCoaAR {
		t.Fatalf("%+v", got)
	}
	got = ResolveCoa(CoaMap{AccountAR: "62.99"}, true)
	if got.AccountAR != "62.99" || got.AccountRevenue != DefaultCoaRevenue {
		t.Fatalf("partial merge %+v", got)
	}
}

func TestValidateCoaAccounts(t *testing.T) {
	if err := ValidateCoaAccounts(CoaMap{AccountAR: "62.01"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCoaAccounts(CoaMap{AccountAR: "bad account!"}); err == nil {
		t.Fatal("expected invalid")
	}
}

func TestUpsertAndGetCoa(t *testing.T) {
	svc := NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)
	repo := NewMemoryCoaRepository()
	svc.SetCoaRepository(repo)
	m, err := svc.UpsertCoa(context.Background(), TenantSupplier, "sup-1", "user-1", CoaMap{
		AccountAR: "62.10", AccountRevenue: "90.10", AccountBankCash: "51.10",
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.AccountAR != "62.10" || m.UsingDefaults {
		t.Fatalf("%+v", m)
	}
	got, err := svc.GetCoa(context.Background(), TenantSupplier, "sup-1")
	if err != nil || got.AccountBankCash != "51.10" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestExportWorkerUsesTenantCoa(t *testing.T) {
	repo := NewMemoryCoaRepository()
	_ = repo.Upsert(context.Background(), CoaMap{
		TenantType: TenantSupplier, TenantID: "sup-coa",
		AccountAR: "62.77", AccountRevenue: "90.77", AccountBankCash: "51.77",
	})
	w := NewExportWorker(NewMemoryExportRepository(), NewMemorySftpConfigRepository(), nil, nil)
	w.SetCoaRepository(repo)
	coa := w.resolveTenantCoa(context.Background(), TenantSupplier, "sup-coa")
	if coa.AccountAR != "62.77" {
		t.Fatalf("%+v", coa)
	}
	d, c := mapARJournalAccounts(coa, "OPEN")
	if d != "62.77" || c != "90.77" {
		t.Fatalf("%s/%s", d, c)
	}
}
