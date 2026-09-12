package supplier

import (
	"context"
	"errors"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestBillingSetupRequest_Validate_Structural(t *testing.T) {
	t.Parallel()
	req := BillingSetupRequest{
		BankName: "NBU", AccountHolder: "Co", AccountNumber: "1", SwiftBic: "NBFAUZ2X",
		SelectedGateways: []string{"STRIPE"},
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("structural validate should not pack-check: %v", err)
	}
}

func TestConfigureBilling_RejectsStripeOnUZ(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := &Service{supplierID: "sup-1"}
	_, err := svc.ConfigureBilling(context.Background(), BillingSetupRequest{
		BankName: "NBU", AccountHolder: "Co", AccountNumber: "1", SwiftBic: "NBFAUZ2X",
		SelectedGateways: []string{"STRIPE"},
	})
	if !errors.Is(err, auth.ErrPackGatewayForbidden) {
		t.Fatalf("err=%v", err)
	}
}

func TestConfigureBilling_RejectsAdyenOnUZ(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := &Service{supplierID: "sup-1"}
	_, err := svc.ConfigureBilling(context.Background(), BillingSetupRequest{
		BankName: "NBU", AccountHolder: "Co", AccountNumber: "1", SwiftBic: "NBFAUZ2X",
		SelectedGateways: []string{"ADYEN"},
	})
	if !errors.Is(err, auth.ErrPackGatewayForbidden) {
		t.Fatalf("err=%v", err)
	}
}
