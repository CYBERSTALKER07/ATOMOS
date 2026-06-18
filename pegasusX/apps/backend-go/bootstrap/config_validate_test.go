package bootstrap

import (
	"os"
	"testing"
)

func TestValidateProductionProfile_RejectsDevSecrets(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	cfg := testConfig()
	cfg.GlobalPayWebhookSecret = "dev-global-pay-secret"
	cfg.AdyenWebhookSecret = "dev-adyen-secret"
	cfg.StripeWebhookSecret = "dev-stripe-secret"

	if err := cfg.ValidateProductionProfile(); err == nil {
		t.Fatal("expected production profile validation error")
	}
}

func TestValidateProductionProfile_RequiresInfraAdapters(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	cfg := testConfig()
	cfg.RequireInfraAdapters = false
	cfg.GlobalPayWebhookSecret = "prod-global-pay-secret"
	cfg.AdyenWebhookSecret = "prod-adyen-secret"
	cfg.StripeWebhookSecret = "prod-stripe-secret"
	cfg.PaymeWebhookSecret = "prod-payme-secret"
	cfg.ClickWebhookSecret = "prod-click-secret"

	if err := cfg.ValidateProductionProfile(); err == nil {
		t.Fatal("expected production profile to require infra adapters")
	}
}

func TestValidateProductionProfile_AllowsDevSecretsOutsideProduction(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "ssmr")
	cfg := testConfig()
	cfg.GlobalPayWebhookSecret = "dev-global-pay-secret"
	cfg.RequireInfraAdapters = true

	if err := cfg.ValidateProductionProfile(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestLoadConfig_ProductionProfileFailsClosed(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("GLOBAL_PAY_WEBHOOK_SECRET", "dev-global-pay-secret")
	t.Setenv("ADYEN_WEBHOOK_SECRET", "dev-adyen-secret")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "dev-stripe-secret")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected LoadConfig to fail on dev webhook secrets in production")
	}
}

func TestValidateProductionProfile_RequiresGlobalPayCredentialsInProductionEnv(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	cfg := testConfig()
	cfg.GlobalPayWebhookSecret = "prod-global-pay-secret"
	cfg.AdyenWebhookSecret = "prod-adyen-secret"
	cfg.StripeWebhookSecret = "prod-stripe-secret"
	cfg.PaymeWebhookSecret = "prod-payme-secret"
	cfg.ClickWebhookSecret = "prod-click-secret"
	cfg.GlobalPayEnv = "production"
	cfg.GlobalPayUsername = ""
	cfg.GlobalPayPassword = "secret"

	if err := cfg.ValidateProductionProfile(); err == nil {
		t.Fatal("expected missing GLOBAL_PAY_USERNAME to fail production profile")
	}
}

func TestLoadConfig_SSMRDevSecretsAllowed(t *testing.T) {
	_ = os.Unsetenv("PEGASUSX_ENV")
	t.Setenv("GLOBAL_PAY_WEBHOOK_SECRET", "dev-global-pay-secret")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.GlobalPayWebhookSecret != "dev-global-pay-secret" {
		t.Fatalf("secret = %q", cfg.GlobalPayWebhookSecret)
	}
}
