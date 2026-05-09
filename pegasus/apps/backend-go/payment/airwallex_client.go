package payment

import (
	"log/slog"
)

// AirwallexClient integrates with Airwallex's B2B payment rails (Virtual IBAN / Dynamic Routing).
// It implements SplittingGateway or GatewayClient as needed, handling dynamic currency and
// international B2B settlements.
type AirwallexClient struct {
	clientID string
	apiKey   string
	env      string
}

// NewAirwallexClient configures a new Airwallex API client using environment secrets.
func NewAirwallexClient(clientID, apiKey, env string) *AirwallexClient {
	return &AirwallexClient{
		clientID: clientID,
		apiKey:   apiKey,
		env:      env,
	}
}

// Charge places a hold or instant capture for a B2B invoice depending on configuration.
func (c *AirwallexClient) Charge(orderID string, amount int64) error {
	slog.Info("airwallex: mocked charge", "order_id", orderID, "amount", amount)
	// Example integration boundary: perform API request to /api/v1/pa/payment_intents/create
	return nil
}

// Refund reverts an accepted B2B settlement back to the virtual IBAN.
func (c *AirwallexClient) Refund(orderID string, refundAmount int64) error {
	slog.Info("airwallex: mocked refund", "order_id", orderID, "amount", refundAmount)
	return nil
}

// Authorize holds funds on a card or account.
func (c *AirwallexClient) Authorize(orderID string, amount int64) (string, error) {
	slog.Info("airwallex: mocked authorize", "order_id", orderID, "amount", amount)
	return "auth_simulated_id", nil
}

// Capture finalizes a held authorization.
func (c *AirwallexClient) Capture(authorizationID string, captureAmount int64) error {
	slog.Info("airwallex: mocked capture", "auth_id", authorizationID, "amount", captureAmount)
	return nil
}

// Void attempts to cancel a held authorization.
func (c *AirwallexClient) Void(authorizationID string) error {
	slog.Info("airwallex: mocked void", "auth_id", authorizationID)
	return nil
}

// Verify it meets standard interface definitions.
var _ SplittingGateway = (*AirwallexClient)(nil)
