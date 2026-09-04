package payment

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewService_DoesNotInventWebhookSecrets(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "local")
	svc := NewService(ServiceConfig{SupplierID: "seed"})
	if svc.globalPayWebhookSecret != "" ||
		svc.adyenWebhookSecret != "" ||
		svc.stripeWebhookSecret != "" ||
		svc.paymeWebhookSecret != "" ||
		svc.clickWebhookSecret != "" {
		t.Fatalf("NewService must not invent webhook secrets; got gp=%q adyen=%q stripe=%q payme=%q click=%q",
			svc.globalPayWebhookSecret, svc.adyenWebhookSecret, svc.stripeWebhookSecret, svc.paymeWebhookSecret, svc.clickWebhookSecret)
	}
}

func TestNewService_StripsDevSecretsInProduction(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	svc := NewService(ServiceConfig{
		SupplierID:             "seed",
		GlobalPayWebhookSecret: "dev-global-pay-secret",
		AdyenWebhookSecret:     "dev-adyen-secret",
		StripeWebhookSecret:    "dev-stripe-secret",
		PaymeWebhookSecret:     "dev-payme-secret",
		ClickWebhookSecret:     "dev-click-secret",
	})
	if svc.globalPayWebhookSecret != "" ||
		svc.adyenWebhookSecret != "" ||
		svc.stripeWebhookSecret != "" ||
		svc.paymeWebhookSecret != "" ||
		svc.clickWebhookSecret != "" {
		t.Fatalf("production must strip known-weak webhook secrets; got gp=%q adyen=%q stripe=%q payme=%q click=%q",
			svc.globalPayWebhookSecret, svc.adyenWebhookSecret, svc.stripeWebhookSecret, svc.paymeWebhookSecret, svc.clickWebhookSecret)
	}
}

func TestNewService_KeepsNonDevSecretsInProduction(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	svc := NewService(ServiceConfig{
		SupplierID:             "seed",
		GlobalPayWebhookSecret: "prod-global-pay-secret",
		StripeWebhookSecret:    "whsec_live_example",
	})
	if svc.globalPayWebhookSecret != "prod-global-pay-secret" {
		t.Fatalf("gp secret = %q", svc.globalPayWebhookSecret)
	}
	if svc.stripeWebhookSecret != "whsec_live_example" {
		t.Fatalf("stripe secret = %q", svc.stripeWebhookSecret)
	}
}

func TestGlobalPayWebhook_FailClosedWithoutSecret(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "local")
	svc := NewService(ServiceConfig{SupplierID: "seed"})
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/global-pay", strings.NewReader(`{"transaction_id":"t1","status":"SUCCESS"}`))
	req.Header.Set("Authorization", "Basic ZGV2LWdsb2JhbC1wYXktc2VjcmV0Og==") // "dev-global-pay-secret:"
	rr := httptest.NewRecorder()
	svc.HandleGlobalPayWebhook(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 when no webhook secret configured", rr.Code)
	}
}
