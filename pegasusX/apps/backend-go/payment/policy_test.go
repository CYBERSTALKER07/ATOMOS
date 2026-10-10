package payment

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestNormalizeGatewayPolicy_DefaultsCashAndGlobalPay(t *testing.T) {
	t.Parallel()
	policy := NormalizeGatewayPolicy(PaymentAcceptorSupplier, nil, "")
	if policy.DefaultPaymentMethod != DefaultPaymentMethod {
		t.Fatalf("default payment method = %q", policy.DefaultPaymentMethod)
	}
	if policy.DefaultCardGateway != DefaultCardGatewayName {
		t.Fatalf("default card gateway = %q", policy.DefaultCardGateway)
	}
	if len(policy.AllowedGateways) < 2 {
		t.Fatalf("allowed gateways = %#v", policy.AllowedGateways)
	}
}

func TestGatewayPolicy_ValidateCardGateway(t *testing.T) {
	t.Parallel()
	policy := NormalizeGatewayPolicy(PaymentAcceptorSupplier, []string{"GLOBAL_PAY", "CASH"}, "SUPPLIER_DEFAULT")
	if err := policy.ValidateCardGateway("ADYEN"); err == nil {
		t.Fatal("expected policy violation for ADYEN")
	}
	if err := policy.ValidateCardGateway("GLOBAL_PAY"); err != nil {
		t.Fatalf("GLOBAL_PAY should be allowed: %v", err)
	}
}

func TestGatewayPolicy_CardGatewaysExcludesCash(t *testing.T) {
	t.Parallel()
	policy := NormalizeGatewayPolicy(PaymentAcceptorSupplier, []string{"GLOBAL_PAY", "CASH"}, "SUPPLIER_DEFAULT")
	card := policy.CardGateways()
	if len(card) != 1 || card[0] != "GLOBAL_PAY" {
		t.Fatalf("card gateways = %#v", card)
	}
}

func TestApplyPackToGatewayPolicy_DropsUnknownPSP(t *testing.T) {
	t.Parallel()
	pack, ok := auth.ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	policy := NormalizeGatewayPolicy(PaymentAcceptorSupplier, []string{"GLOBAL_PAY", "STRIPE", "CASH"}, "SUPPLIER_DEFAULT")
	got := applyPackToGatewayPolicy(policy, pack)
	if got.PolicySource != "MARKET_PACK" {
		t.Fatalf("source=%s", got.PolicySource)
	}
	if err := got.ValidateCardGateway("STRIPE"); err == nil {
		t.Fatal("STRIPE must be dropped by pack intersect")
	}
	if err := got.ValidateCardGateway("GLOBAL_PAY"); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeGatewayPolicyForPack_EmptyUsesLiveUZ(t *testing.T) {
	t.Parallel()
	pack, ok := auth.ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	policy := NormalizeGatewayPolicyForPack(PaymentAcceptorWarehouse, nil, "", pack)
	if !containsGateway(policy.AllowedGateways, "CASH") || !containsGateway(policy.AllowedGateways, "GLOBAL_PAY") {
		t.Fatalf("allowed=%#v", policy.AllowedGateways)
	}
	if containsGateway(policy.AllowedGateways, "ADYEN") || containsGateway(policy.AllowedGateways, "STRIPE") {
		t.Fatalf("foreign rails leaked: %#v", policy.AllowedGateways)
	}
	if containsGateway(policy.AllowedGateways, "PAYME") {
		t.Fatalf("empty config must not select unkeyed rails: %#v", policy.AllowedGateways)
	}
}

func TestNormalizeGatewayPolicyForPack_RejectsStripeOnUZ(t *testing.T) {
	t.Parallel()
	pack, ok := auth.ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	policy := NormalizeGatewayPolicyForPack(PaymentAcceptorWarehouse, []string{"STRIPE", "ADYEN"}, "WAREHOUSE_CONFIG", pack)
	if containsGateway(policy.AllowedGateways, "STRIPE") || containsGateway(policy.AllowedGateways, "ADYEN") {
		t.Fatalf("allowed=%#v", policy.AllowedGateways)
	}
	if !containsGateway(policy.AllowedGateways, "CASH") || !containsGateway(policy.AllowedGateways, "GLOBAL_PAY") {
		t.Fatalf("fallback live=%#v", policy.AllowedGateways)
	}
}

func TestGatewayPolicy_CardGatewaysDoesNotInventGlobalPay(t *testing.T) {
	t.Parallel()
	policy := NormalizeGatewayPolicy(PaymentAcceptorSupplier, []string{"CASH"}, "cash-only")
	if got := policy.CardGateways(); len(got) != 0 {
		t.Fatalf("card gateways=%#v", got)
	}
}
