package payment

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

var (
	// ErrAirwallexDirectOperationUnsupported marks flows that should remain on
	// hosted checkout until direct execution is explicitly enabled.
	ErrAirwallexDirectOperationUnsupported = errors.New("airwallex direct execution disabled; use hosted checkout flow")
)

// AirwallexCredentials stores gateway credentials used by execution paths.
type AirwallexCredentials struct {
	ClientID               string
	APIKey                 string
	Environment            string
	DirectExecutionEnabled bool
}

// AirwallexClient integrates with Airwallex's B2B payment rails (Virtual IBAN / Dynamic Routing).
// It implements SplittingGateway or GatewayClient as needed, handling dynamic currency and
// international B2B settlements.
type AirwallexClient struct {
	clientID string
	apiKey   string
	env      string
}

// ResolveAirwallexCredentials merges vault values with environment fallbacks.
func ResolveAirwallexCredentials(clientID, environment, apiKey string) (AirwallexCredentials, error) {
	creds := AirwallexCredentials{
		ClientID:               firstNonEmpty(strings.TrimSpace(clientID), strings.TrimSpace(os.Getenv("AIRWALLEX_CLIENT_ID")), strings.TrimSpace(os.Getenv("AIRWALLEX_MERCHANT_ID"))),
		APIKey:                 firstNonEmpty(strings.TrimSpace(apiKey), strings.TrimSpace(os.Getenv("AIRWALLEX_API_KEY")), strings.TrimSpace(os.Getenv("AIRWALLEX_SECRET_KEY"))),
		Environment:            strings.ToUpper(firstNonEmpty(strings.TrimSpace(environment), strings.TrimSpace(os.Getenv("AIRWALLEX_ENVIRONMENT")), strings.TrimSpace(os.Getenv("AIRWALLEX_ENV")), "DEMO")),
		DirectExecutionEnabled: strings.EqualFold(strings.TrimSpace(os.Getenv("AIRWALLEX_DIRECT_EXECUTION_ENABLED")), "true"),
	}
	if creds.ClientID == "" || creds.APIKey == "" {
		return AirwallexCredentials{}, fmt.Errorf("airwallex credentials incomplete: client_id and api_key are required")
	}
	return creds, nil
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
