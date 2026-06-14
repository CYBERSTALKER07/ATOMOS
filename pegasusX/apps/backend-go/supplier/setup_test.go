package supplier

import (
	"context"
	"testing"
)

func TestCompleteBusinessSetupMarksRegistered(t *testing.T) {
	repo := &countingSupplierRepo{
		profiles: map[string]Profile{
			"sup-1": {SupplierID: "sup-1", LegalName: "Acme", Country: "UZ", Currency: "UZS"},
		},
	}
	svc := NewService(ServiceConfig{
		Repo:       repo,
		SupplierID: "sup-1",
		Country:    "UZ",
		Currency:   "UZS",
	})

	resp, err := svc.CompleteBusinessSetup(context.Background(), "sup-1", BusinessSetupRequest{
		TaxID:               "123456789",
		RegistrationNumber:  "REG-1",
		HeadquartersAddress: "1 Main St",
		City:                "Tashkent",
		PostalCode:          "100000",
	})
	if err != nil {
		t.Fatalf("CompleteBusinessSetup: %v", err)
	}
	if !resp.IsRegistered {
		t.Fatal("expected is_registered true")
	}
	if resp.NextStep != "/setup/billing" {
		t.Fatalf("next_step=%q want /setup/billing", resp.NextStep)
	}

	profile := repo.profiles["sup-1"]
	if !profile.IsRegistered {
		t.Fatal("profile.IsRegistered=false")
	}
	if profile.TaxID != "123456789" {
		t.Fatalf("tax_id=%q", profile.TaxID)
	}
	if profile.BillingAddress == "" {
		t.Fatal("expected billing address from HQ")
	}
}

func TestRegisterMinimalLeavesUnregistered(t *testing.T) {
	repo := &countingSupplierRepo{
		count: 1,
		profiles: map[string]Profile{
			"seed-1": {SupplierID: "seed-1", IsRegistered: false},
		},
	}
	svc := NewService(ServiceConfig{
		Repo:           repo,
		SupplierID:     "seed-1",
		SeedSupplierID: "seed-1",
		MaxSuppliers:   10,
		Country:        "UZ",
		Currency:       "UZS",
	})

	resp, err := svc.Register(context.Background(), RegisterRequest{
		Account: AccountStep{
			LegalName:   "Acme LLC",
			ContactName: "Boss",
			Email:       "boss@acme.test",
			Country:     "UZ",
			Phone:       "+998901234567",
		},
		Phone: "+998901234567",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.IsRegistered {
		t.Fatal("minimal register should leave is_registered false")
	}
	if resp.NextStep != "/setup/business" {
		t.Fatalf("next_step=%q want /setup/business", resp.NextStep)
	}
}

func TestRegisterFullPayloadMarksRegistered(t *testing.T) {
	repo := &countingSupplierRepo{
		count: 1,
		profiles: map[string]Profile{
			"seed-1": {SupplierID: "seed-1", IsRegistered: false},
		},
	}
	svc := NewService(ServiceConfig{
		Repo:           repo,
		SupplierID:     "seed-1",
		SeedSupplierID: "seed-1",
		MaxSuppliers:   10,
		Country:        "UZ",
		Currency:       "UZS",
	})

	resp, err := svc.Register(context.Background(), RegisterRequest{
		Account: AccountStep{
			LegalName:   "Acme LLC",
			ContactName: "Boss",
			Email:       "boss@acme.test",
			Country:     "UZ",
			Phone:       "+998901234567",
		},
		Location: LocationStep{
			Warehouse: AddressStep{Address: "Warehouse 1", Lat: 41.3, Lng: 69.2},
		},
		Business: BusinessStep{TaxID: "TAX-99"},
		Phone:    "+998901234567",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !resp.IsRegistered {
		t.Fatal("full register should set is_registered true")
	}
	if resp.NextStep != "/setup/billing" {
		t.Fatalf("next_step=%q want /setup/billing", resp.NextStep)
	}
}
