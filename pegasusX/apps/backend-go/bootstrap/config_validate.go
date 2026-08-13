package bootstrap

import (
	"fmt"
	"os"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// ValidateProductionProfile rejects dev-default webhook secrets when
// PEGASUSX_ENV=production. SSMR/local stacks keep dev-* secrets in .env.ssmr.
func (c *Config) ValidateProductionProfile() error {
	if c == nil || !isProductionEnv() {
		return nil
	}
	if !auth.TenantContextEnforced() {
		return fmt.Errorf("TENANT_CONTEXT_ENFORCED must be true when PEGASUSX_ENV=production (or unset to use production default)")
	}
	if !c.RequireInfraAdapters {
		return fmt.Errorf("REQUIRE_INFRA_ADAPTERS must be true when PEGASUSX_ENV=production")
	}
	if c.AllowMemoryFallback {
		return fmt.Errorf("ALLOW_MEMORY_FALLBACK must be false when PEGASUSX_ENV=production")
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.JWTSecret)), "dev-") ||
		strings.EqualFold(strings.TrimSpace(c.JWTSecret), "dev-only-change-me") {
		return fmt.Errorf("JWT_SECRET must be a non-dev value when PEGASUSX_ENV=production")
	}
	if !c.PlatformAdminMFARequired {
		return fmt.Errorf("PLATFORM_ADMIN_MFA_REQUIRED must be true when PEGASUSX_ENV=production")
	}
	if err := c.validateUpdatesBaseURL(true); err != nil {
		return err
	}
	checks := map[string]string{
		"GLOBAL_PAY_WEBHOOK_SECRET": c.GlobalPayWebhookSecret,
		"ADYEN_WEBHOOK_SECRET":      c.AdyenWebhookSecret,
		"STRIPE_WEBHOOK_SECRET":     c.StripeWebhookSecret,
		"PAYME_WEBHOOK_SECRET":      c.PaymeWebhookSecret,
		"CLICK_WEBHOOK_SECRET":      c.ClickWebhookSecret,
	}
	for name, value := range checks {
		if isDevWebhookSecret(value) {
			return fmt.Errorf("%s must be set to a non-dev value when PEGASUSX_ENV=production", name)
		}
	}
	gpEnv := strings.ToLower(strings.TrimSpace(c.GlobalPayEnv))
	if gpEnv == "production" || gpEnv == "staging" {
		if strings.TrimSpace(c.GlobalPayUsername) == "" || strings.TrimSpace(c.GlobalPayPassword) == "" {
			return fmt.Errorf("GLOBAL_PAY_USERNAME and GLOBAL_PAY_PASSWORD must be set when GLOBAL_PAY_ENV=%s in production profile", gpEnv)
		}
		if strings.TrimSpace(c.GlobalPayServiceID) == "" {
			return fmt.Errorf("GLOBAL_PAY_SERVICE_ID must be set when GLOBAL_PAY_ENV=%s in production profile", gpEnv)
		}
	}
	// G1.B: production tax path — MY_SOLIQ (default when unset) must have OFD + EDS; PEGASUS commercial needs explicit allow.
	if err := validateProductionFiscalProfile(); err != nil {
		return err
	}
	// G1.D: production push must not be silently no-op without explicit allow.
	if err := validateProductionFCMProfile(); err != nil {
		return err
	}
	return nil
}

// validateProductionFCMProfile requires Firebase config unless FCM_ALLOW_NOOP=true.
func validateProductionFCMProfile() error {
	if envTruthyFiscal(os.Getenv("FCM_ALLOW_NOOP")) {
		return nil
	}
	if strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID")) == "" &&
		strings.TrimSpace(os.Getenv("FIREBASE_CREDENTIALS_PATH")) == "" &&
		strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")) == "" {
		return fmt.Errorf("FIREBASE_PROJECT_ID or FIREBASE_CREDENTIALS_PATH required in production (or set FCM_ALLOW_NOOP=true for explicit push_degraded)")
	}
	return nil
}

// validateProductionFiscalProfile enforces fiscal product truth at boot (G1.B).
func validateProductionFiscalProfile() error {
	name := resolveProductionFiscalProviderName()
	switch name {
	case "MY_SOLIQ", "MYSOLIQ", "SOLIQ", "OFD":
		for _, key := range []string{"FISCAL_MY_SOLIQ_BASE_URL", "FISCAL_MY_SOLIQ_API_KEY", "FISCAL_MY_SOLIQ_TIN"} {
			if strings.TrimSpace(os.Getenv(key)) == "" {
				return fmt.Errorf("%s required when FISCAL_PROVIDER resolves to MY_SOLIQ in production", key)
			}
		}
		signer := strings.ToLower(strings.TrimSpace(os.Getenv("FISCAL_MY_SOLIQ_SIGNER")))
		if signer == "" {
			return fmt.Errorf("FISCAL_MY_SOLIQ_SIGNER required when MY_SOLIQ in production (use pkcs12 with EDS)")
		}
		if signer == "dev-hmac" {
			return fmt.Errorf("FISCAL_MY_SOLIQ_SIGNER=dev-hmac is forbidden in production (use pkcs12)")
		}
		if signer == "pkcs12" && strings.TrimSpace(os.Getenv("FISCAL_MY_SOLIQ_PKCS12_FILE")) == "" {
			return fmt.Errorf("FISCAL_MY_SOLIQ_PKCS12_FILE required for pkcs12 signer in production")
		}
	case "PEGASUS", "COMMERCIAL", "PLATFORM":
		if !envTruthyFiscal(os.Getenv("FISCAL_ALLOW_COMMERCIAL_RECEIPTS")) {
			return fmt.Errorf("FISCAL_PROVIDER=PEGASUS in production requires FISCAL_ALLOW_COMMERCIAL_RECEIPTS=true (commercial tax_ofd=false path — not Soliq OFD)")
		}
	case "FAKE":
		return fmt.Errorf("FISCAL_PROVIDER=FAKE is forbidden when PEGASUSX_ENV=production")
	}
	return nil
}

// resolveProductionFiscalProviderName mirrors order.ResolveFiscalProviderName without an order import cycle.
func resolveProductionFiscalProviderName() string {
	raw := strings.ToUpper(strings.TrimSpace(os.Getenv("FISCAL_PROVIDER")))
	switch raw {
	case "MY_SOLIQ", "MYSOLIQ", "SOLIQ", "OFD":
		return "MY_SOLIQ"
	case "FAKE":
		return "FAKE"
	case "GLOBAL_PAY":
		return "GLOBAL_PAY"
	case "PEGASUS", "COMMERCIAL", "PLATFORM":
		return "PEGASUS"
	case "":
		// Tax default in production (G1.B).
		return "MY_SOLIQ"
	default:
		return "MY_SOLIQ"
	}
}

func envTruthyFiscal(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// allowsRepoMemoryFallback gates in-memory domain repos / outbox when Spanner is down.
// Unit tests (TestingMode) may use memory; production never does; local SSMR only with
// ALLOW_MEMORY_FALLBACK=true and REQUIRE_INFRA_ADAPTERS=false.
func (c *Config) allowsRepoMemoryFallback() bool {
	if c == nil || isProductionEnv() {
		return false
	}
	if c.TestingMode {
		return true
	}
	return c.AllowMemoryFallback && !c.RequireInfraAdapters
}

// ensureMemoryFallbackAllowed gates silent in-memory repository / outbox paths.
func (c *Config) ensureMemoryFallbackAllowed(component string) error {
	if c == nil {
		return fmt.Errorf("%s: config nil", component)
	}
	if c.allowsRepoMemoryFallback() {
		return nil
	}
	return fmt.Errorf("%s: in-memory fallback blocked (set ALLOW_MEMORY_FALLBACK=true only for local/SSMR with REQUIRE_INFRA_ADAPTERS=false)", component)
}

// validateUpdatesBaseURL ensures OTA origin is a real HTTPS public base in production.
func (c *Config) validateUpdatesBaseURL(require bool) error {
	base := strings.TrimSpace(c.UpdatesBaseURL)
	if base == "" {
		if require {
			return fmt.Errorf("UPDATES_BASE_URL must be set when PEGASUSX_ENV=production")
		}
		return nil
	}
	lower := strings.ToLower(base)
	if strings.Contains(lower, "example.com") || strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") {
		if require {
			return fmt.Errorf("UPDATES_BASE_URL must not use example/localhost hosts in production (got %q)", base)
		}
	}
	if require && !strings.HasPrefix(lower, "https://") {
		return fmt.Errorf("UPDATES_BASE_URL must use https:// in production (got %q)", base)
	}
	return nil
}

func isProductionEnv() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "production")
}

// fcmAllowNoOp reports whether FCM may degrade to no-op push (G1.D).
// Tax/ops envs (production, staging) refuse silent no-op unless FCM_ALLOW_NOOP=true.
// Local/ssmr/dev allow no-op by default so developers are not blocked.
func fcmAllowNoOp() bool {
	if envTruthyFiscal(os.Getenv("FCM_ALLOW_NOOP")) {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PEGASUSX_ENV"))) {
	case "production", "prod", "staging":
		return false
	default:
		return true
	}
}

func isDevWebhookSecret(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(trimmed), "dev-")
}
