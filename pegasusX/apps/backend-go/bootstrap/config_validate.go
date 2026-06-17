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
