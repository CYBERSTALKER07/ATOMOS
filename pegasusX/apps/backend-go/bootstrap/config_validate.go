package bootstrap

import (
	"fmt"
	"os"
	"strings"
)

// ValidateProductionProfile rejects dev-default webhook secrets when
// PEGASUSX_ENV=production. SSMR/local stacks keep dev-* secrets in .env.ssmr.
func (c *Config) ValidateProductionProfile() error {
	if c == nil || !isProductionEnv() {
		return nil
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
	return nil
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

func isDevWebhookSecret(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(trimmed), "dev-")
}
