// Package simulator_test provides end-to-end payment and pricing simulation tests.
package simulator_test

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

// ─────────────────────────────────────────────────────────────────────────────
// 1. Gateway Policy simulation
// ─────────────────────────────────────────────────────────────────────────────

func TestSimGatewayPolicy_DefaultsCashAndGlobalPay(t *testing.T) {
	policy := payment.NormalizeGatewayPolicy("test-supplier", nil, "default")

	// Cash is always included in AllowedGateways
	hasCash := false
	for _, g := range policy.AllowedGateways {
		if g == "CASH" {
			hasCash = true
		}
	}
	if !hasCash {
		t.Error("CASH should be in AllowedGateways by default")
	}

	gateways := policy.CardGateways()
	if len(gateways) == 0 {
		t.Error("should have at least one card gateway by default")
	}

	t.Logf("default policy: allowed=%v, card_gateways=%v, default_card=%s",
		policy.AllowedGateways, gateways, policy.DefaultCardGateway)
}

func TestSimGatewayPolicy_CardGatewaysExcludesCash(t *testing.T) {
	policy := payment.NormalizeGatewayPolicy("test-supplier", []string{"CASH", "GLOBAL_PAY", "CLICK"}, "test")
	gateways := policy.CardGateways()

	for _, g := range gateways {
		if g == "CASH" {
			t.Error("card gateways should not include 'CASH'")
		}
	}
}

func TestSimGatewayPolicy_ValidateCardGateway(t *testing.T) {
	policy := payment.NormalizeGatewayPolicy("test-supplier", []string{"GLOBAL_PAY", "CLICK"}, "test")

	if err := policy.ValidateCardGateway("GLOBAL_PAY"); err != nil {
		t.Errorf("GLOBAL_PAY should be valid: %v", err)
	}

	if err := policy.ValidateCardGateway("CLICK"); err != nil {
		t.Errorf("CLICK should be valid: %v", err)
	}

	if err := policy.ValidateCardGateway("NONEXISTENT_GATEWAY"); err == nil {
		t.Error("nonexistent gateway should fail validation")
	}
}

func TestSimGatewayPolicy_ResolveCardGateway(t *testing.T) {
	policy := payment.NormalizeGatewayPolicy("test-supplier", []string{"GLOBAL_PAY", "CLICK"}, "test")

	// Requesting a known gateway should return it
	resolved := policy.ResolveCardGateway("CLICK")
	if resolved != "CLICK" {
		t.Errorf("expected 'CLICK', got '%s'", resolved)
	}

	// Requesting unknown should return default
	resolved = policy.ResolveCardGateway("UNKNOWN")
	if resolved == "" {
		t.Error("unknown gateway should resolve to a default, not empty")
	}
	t.Logf("unknown gateway resolved to: %s", resolved)
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. Gateway policy edge cases
// ─────────────────────────────────────────────────────────────────────────────

func TestSimGatewayPolicy_EmptyGateways(t *testing.T) {
	policy := payment.NormalizeGatewayPolicy("test-supplier", []string{}, "empty")

	// Even with empty gateways, cash should be present
	hasCash := false
	for _, g := range policy.AllowedGateways {
		if g == "CASH" {
			hasCash = true
		}
	}
	if !hasCash {
		t.Error("CASH should be in AllowedGateways even with empty gateway list")
	}
}

func TestSimGatewayPolicy_CashOnlyConfig(t *testing.T) {
	policy := payment.NormalizeGatewayPolicy("test-supplier", []string{"CASH"}, "cash-only")

	hasCash := false
	for _, g := range policy.AllowedGateways {
		if g == "CASH" {
			hasCash = true
		}
	}
	if !hasCash {
		t.Error("CASH should be in AllowedGateways")
	}

	cardGateways := policy.CardGateways()
	t.Logf("cash-only config: card_gateways=%v (defaults applied)", cardGateways)
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. Payer model validation
// ─────────────────────────────────────────────────────────────────────────────

func TestSimPayer_ModelFields(t *testing.T) {
	phone := "+998901234567"
	addr := "123 Billing St, Tashkent"
	taxID := "UZ123456789"

	p := payment.Payer{
		PayerID:        "payer-001",
		Name:           "Test Company LLC",
		Email:          "billing@testco.uz",
		Phone:          &phone,
		BillingAddress: &addr,
		TaxID:          &taxID,
		IsActive:       true,
	}

	if p.PayerID == "" {
		t.Error("PayerID should be set")
	}
	if p.Name == "" {
		t.Error("Name should be set")
	}
	if p.Email == "" {
		t.Error("Email should be set")
	}
	if p.Phone == nil || *p.Phone == "" {
		t.Error("Phone should be set")
	}
	if p.BillingAddress == nil || *p.BillingAddress == "" {
		t.Error("BillingAddress should be set")
	}
	if p.TaxID == nil || *p.TaxID == "" {
		t.Error("TaxID should be set")
	}
	if !p.IsActive {
		t.Error("IsActive should be true")
	}
	t.Logf("payer model: %s (%s) ✓", p.Name, p.Email)
}

func TestSimPayer_OptionalFields(t *testing.T) {
	p := payment.Payer{
		PayerID:  "payer-002",
		Name:     "Solo Retailer",
		Email:    "solo@example.uz",
		IsActive: true,
	}

	if p.Phone != nil {
		t.Error("Phone should be nil when not set")
	}
	if p.BillingAddress != nil {
		t.Error("BillingAddress should be nil when not set")
	}
	if p.TaxID != nil {
		t.Error("TaxID should be nil when not set")
	}
	t.Log("payer optional fields: nil by default ✓")
}
