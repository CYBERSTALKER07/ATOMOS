package payment

import (
	"fmt"
	"strings"
)

// ProviderClient exposes a normalized provider identity over GatewayClient.
type ProviderClient interface {
	GatewayClient
	GatewayName() string
}

type staticProviderClient struct {
	name   string
	client GatewayClient
}

func (s *staticProviderClient) GatewayName() string {
	return s.name
}

func (s *staticProviderClient) Charge(orderID string, amount int64) error {
	return s.client.Charge(orderID, amount)
}

func (s *staticProviderClient) Refund(orderID string, refundAmount int64) error {
	return s.client.Refund(orderID, refundAmount)
}

func normalizeGateway(gateway string) string {
	return strings.ToUpper(strings.TrimSpace(gateway))
}

// SupportedProviderGateways returns the normalized gateway names accepted by
// the provider resolver.
func SupportedProviderGateways() []string {
	return []string{"GLOBAL_PAY", "ADYEN", "CASH"}
}

// IsSupportedProviderGateway reports whether a gateway can be resolved by the
// provider resolver.
func IsSupportedProviderGateway(gateway string) bool {
	normalized := normalizeGateway(gateway)
	for _, supported := range SupportedProviderGateways() {
		if normalized == supported {
			return true
		}
	}
	return false
}

// NewProviderClient returns a provider-backed client for the gateway string.
func NewProviderClient(gateway string) (ProviderClient, error) {
	normalized := normalizeGateway(gateway)
	switch normalized {
	case "GLOBAL_PAY", "CASH":
		return &staticProviderClient{name: normalized, client: &noopGateway{gateway: normalized}}, nil
	case "ADYEN":
		return &staticProviderClient{name: normalized, client: &adyenGateway{}}, nil
	default:
		return nil, fmt.Errorf("unsupported payment gateway: %s (supported: GLOBAL_PAY, ADYEN, CASH)", strings.TrimSpace(gateway))
	}
}
