package tenantreg

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

type nopBuf struct{}

func (nopBuf) BufferOutbox(context.Context, outbox.Event) error { return nil }

type fakeRegistry struct {
	profiles map[string]supplier.Profile
	phones   map[string]supplier.SupplierAuthRecord
	last     supplier.Profile
}

func (f *fakeRegistry) GetAuthByPhone(_ context.Context, phone string) (supplier.SupplierAuthRecord, bool, error) {
	rec, ok := f.phones[phone]
	return rec, ok, nil
}

func (f *fakeRegistry) GetProfile(_ context.Context, supplierID string) (supplier.Profile, bool, error) {
	p, ok := f.profiles[supplierID]
	return p, ok, nil
}

func (f *fakeRegistry) UpdateProfile(_ context.Context, p supplier.Profile, emit func(outbox.TxnBuffer) error) error {
	if f.profiles == nil {
		f.profiles = map[string]supplier.Profile{}
	}
	f.profiles[p.SupplierID] = p
	f.last = p
	if emit != nil {
		return emit(nopBuf{})
	}
	return nil
}

func testService(repo *fakeRegistry) *Service {
	return NewService(Config{
		Repo:           repo,
		SeedSupplierID: "seed-1",
		JWTSecret:      "test-secret",
		JWTIssuer:      "t",
		NewID:          func() string { return "minted-uuid-1" },
	})
}

func TestRegister_MintsNewUUIDNotSeed(t *testing.T) {
	repo := &fakeRegistry{
		profiles: map[string]supplier.Profile{
			"seed-1": {SupplierID: "seed-1", IsRegistered: false},
		},
	}
	svc := testService(repo)
	resp, err := svc.Register(context.Background(), Request{
		LegalName:  "Acme",
		Phone:      "+998901111111",
		Password:   "secret12",
		MarketCode: "uz",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.SupplierID == "seed-1" {
		t.Fatal("T1 must never write the seed row")
	}
	if resp.SupplierID != "minted-uuid-1" {
		t.Fatalf("id=%s", resp.SupplierID)
	}
	if resp.MarketCode != "UZ" || resp.HomeCell != "cell-uz" || resp.Source != auth.MarketSourceProfile {
		t.Fatalf("pack=%+v", resp)
	}
	got := repo.profiles["minted-uuid-1"]
	if got.MarketCode != "UZ" || got.HomeCell != "cell-uz" {
		t.Fatalf("persisted=%+v", got)
	}
	if _, seedUnchanged := repo.profiles["seed-1"]; !seedUnchanged {
		t.Fatal("seed row must remain")
	}
	if repo.profiles["seed-1"].LegalName != "" {
		t.Fatal("seed must not be overwritten")
	}
}

func TestRegister_PlannedPack404(t *testing.T) {
	svc := testService(&fakeRegistry{})
	_, err := svc.Register(context.Background(), Request{
		LegalName: "Acme", Phone: "+1", Password: "x", MarketCode: "EU",
	})
	if !errors.Is(err, ErrMarketNotShipped) {
		t.Fatalf("err=%v", err)
	}
}

func TestRegister_UnknownPack404(t *testing.T) {
	svc := testService(&fakeRegistry{})
	_, err := svc.Register(context.Background(), Request{
		LegalName: "Acme", Phone: "+1", Password: "x", MarketCode: "XX",
	})
	if !errors.Is(err, ErrUnknownMarket) {
		t.Fatalf("err=%v", err)
	}
}

func TestRegister_EmptyMarketNotSilentUZ(t *testing.T) {
	svc := testService(&fakeRegistry{})
	_, err := svc.Register(context.Background(), Request{
		LegalName: "Acme", Phone: "+1", Password: "x",
	})
	if !errors.Is(err, ErrMarketCodeRequired) {
		t.Fatalf("empty market must not default to UZ: %v", err)
	}
}

func TestRegister_PhoneTaken(t *testing.T) {
	svc := testService(&fakeRegistry{
		phones: map[string]supplier.SupplierAuthRecord{
			"+998": {SupplierID: "other", Phone: "+998"},
		},
	})
	_, err := svc.Register(context.Background(), Request{
		LegalName: "Acme", Phone: "+998", Password: "x", MarketCode: "UZ",
	})
	if !errors.Is(err, ErrPhoneTaken) {
		t.Fatalf("err=%v", err)
	}
}

func TestRegister_SkipsSeedID(t *testing.T) {
	n := 0
	svc := NewService(Config{
		Repo:           &fakeRegistry{},
		SeedSupplierID: "seed-1",
		NewID: func() string {
			n++
			if n == 1 {
				return "seed-1"
			}
			return "fresh-2"
		},
	})
	resp, err := svc.Register(context.Background(), Request{
		LegalName: "Acme", Phone: "+1", Password: "x", MarketCode: "UZ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.SupplierID != "fresh-2" {
		t.Fatalf("id=%s", resp.SupplierID)
	}
}

func TestRegister_SourceHasNoSeedResolver(t *testing.T) {
	for _, name := range []string{"service.go", "handlers.go"} {
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(src), "resolveRegistrationSupplierID(") {
			t.Fatalf("%s must not call the seed register resolver", name)
		}
	}
}
