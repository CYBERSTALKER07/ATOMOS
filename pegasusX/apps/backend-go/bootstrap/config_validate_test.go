package bootstrap

import (
	"os"
	"testing"
)

func TestValidateProductionProfile_RejectsDisabledTenantEnforcement(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("TENANT_CONTEXT_ENFORCED", "false")
	cfg := testConfig()
	cfg.RequireInfraAdapters = true
	cfg.AllowMemoryFallback = false
	cfg.JWTSecret = "prod-jwt-secret-value"
	cfg.GlobalPayWebhookSecret = "prod-global-pay-secret"
	cfg.AdyenWebhookSecret = "prod-adyen-secret"
	cfg.StripeWebhookSecret = "prod-stripe-secret"
	cfg.PaymeWebhookSecret = "prod-payme-secret"
	cfg.ClickWebhookSecret = "prod-click-secret"
	cfg.UpdatesBaseURL = "https://cdn.void.example"

	if err := cfg.ValidateProductionProfile(); err == nil {
		t.Fatal("expected TENANT_CONTEXT_ENFORCED=false to fail production profile")
	}
}

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
	cfg.UpdatesBaseURL = "https://cdn.void.example"

	if err := cfg.ValidateProductionProfile(); err == nil {
		t.Fatal("expected production profile to require infra adapters")
	}
}

func TestValidateProductionProfile_RequiresUpdatesBaseURL(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	cfg := testConfig()
	cfg.RequireInfraAdapters = true
	cfg.AllowMemoryFallback = false
	cfg.JWTSecret = "prod-jwt-secret-value"
	cfg.GlobalPayWebhookSecret = "prod-global-pay-secret"
	cfg.AdyenWebhookSecret = "prod-adyen-secret"
	cfg.StripeWebhookSecret = "prod-stripe-secret"
	cfg.PaymeWebhookSecret = "prod-payme-secret"
	cfg.ClickWebhookSecret = "prod-click-secret"
	cfg.UpdatesBaseURL = ""

	if err := cfg.ValidateProductionProfile(); err == nil {
		t.Fatal("expected missing UPDATES_BASE_URL to fail production profile")
	}
}

func TestEnsureMemoryFallbackAllowed_BlocksByDefault(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "ssmr")
	cfg := testConfig()
	cfg.TestingMode = false
	cfg.RequireInfraAdapters = true
	cfg.AllowMemoryFallback = false
	if err := cfg.ensureMemoryFallbackAllowed("order repository"); err == nil {
		t.Fatal("expected memory fallback blocked under strict infra")
	}

	cfg.RequireInfraAdapters = false
	cfg.AllowMemoryFallback = true
	if err := cfg.ensureMemoryFallbackAllowed("order repository"); err != nil {
		t.Fatalf("expected memory fallback allowed: %v", err)
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
	setenvFiscalMySoliqProd(t)
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

func setenvFiscalMySoliqProd(t *testing.T) {
	t.Helper()
	t.Setenv("FISCAL_PROVIDER", "MY_SOLIQ")
	t.Setenv("FISCAL_MY_SOLIQ_BASE_URL", "https://ofd.example/api")
	t.Setenv("FISCAL_MY_SOLIQ_API_KEY", "k")
	t.Setenv("FISCAL_MY_SOLIQ_TIN", "123")
	t.Setenv("FISCAL_MY_SOLIQ_SIGNER", "pkcs12")
	t.Setenv("FISCAL_MY_SOLIQ_PKCS12_FILE", "/secrets/eds.p12")
	t.Setenv("FIREBASE_PROJECT_ID", "pegasus-test")
}

func TestValidateProductionProfile_RequiresMySoliqCreds(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("TENANT_CONTEXT_ENFORCED", "true")
	t.Setenv("FISCAL_PROVIDER", "MY_SOLIQ")
	t.Setenv("FISCAL_MY_SOLIQ_BASE_URL", "")
	t.Setenv("FISCAL_MY_SOLIQ_API_KEY", "")
	t.Setenv("FISCAL_MY_SOLIQ_TIN", "")
	cfg := testConfig()
	cfg.RequireInfraAdapters = true
	cfg.AllowMemoryFallback = false
	cfg.PlatformAdminMFARequired = true
	cfg.JWTSecret = "prod-jwt-secret-value"
	cfg.GlobalPayWebhookSecret = "prod-global-pay-secret"
	cfg.AdyenWebhookSecret = "prod-adyen-secret"
	cfg.StripeWebhookSecret = "prod-stripe-secret"
	cfg.PaymeWebhookSecret = "prod-payme-secret"
	cfg.ClickWebhookSecret = "prod-click-secret"
	cfg.UpdatesBaseURL = "https://cdn.void.example"
	if err := cfg.ValidateProductionProfile(); err == nil {
		t.Fatal("expected MY_SOLIQ creds required in production")
	}
}

func TestValidateProductionProfile_PegasusRequiresCommercialAllow(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("TENANT_CONTEXT_ENFORCED", "true")
	t.Setenv("FISCAL_PROVIDER", "PEGASUS")
	t.Setenv("FISCAL_ALLOW_COMMERCIAL_RECEIPTS", "")
	t.Setenv("FIREBASE_PROJECT_ID", "pegasus-test")
	cfg := testConfig()
	cfg.RequireInfraAdapters = true
	cfg.AllowMemoryFallback = false
	cfg.PlatformAdminMFARequired = true
	cfg.JWTSecret = "prod-jwt-secret-value"
	cfg.GlobalPayWebhookSecret = "prod-global-pay-secret"
	cfg.AdyenWebhookSecret = "prod-adyen-secret"
	cfg.StripeWebhookSecret = "prod-stripe-secret"
	cfg.PaymeWebhookSecret = "prod-payme-secret"
	cfg.ClickWebhookSecret = "prod-click-secret"
	cfg.UpdatesBaseURL = "https://cdn.void.example"
	if err := cfg.ValidateProductionProfile(); err == nil {
		t.Fatal("expected PEGASUS commercial path to require FISCAL_ALLOW_COMMERCIAL_RECEIPTS")
	}
}

func TestValidateProductionProfile_RequiresFirebaseOrFCMAllow(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("TENANT_CONTEXT_ENFORCED", "true")
	setenvFiscalMySoliqProd(t)
	t.Setenv("FIREBASE_PROJECT_ID", "")
	t.Setenv("FIREBASE_CREDENTIALS_PATH", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	t.Setenv("FCM_ALLOW_NOOP", "")
	cfg := testConfig()
	cfg.RequireInfraAdapters = true
	cfg.AllowMemoryFallback = false
	cfg.PlatformAdminMFARequired = true
	cfg.JWTSecret = "prod-jwt-secret-value"
	cfg.GlobalPayWebhookSecret = "prod-global-pay-secret"
	cfg.AdyenWebhookSecret = "prod-adyen-secret"
	cfg.StripeWebhookSecret = "prod-stripe-secret"
	cfg.PaymeWebhookSecret = "prod-payme-secret"
	cfg.ClickWebhookSecret = "prod-click-secret"
	cfg.UpdatesBaseURL = "https://cdn.void.example"
	if err := cfg.ValidateProductionProfile(); err == nil {
		t.Fatal("expected missing Firebase config to fail production profile")
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
