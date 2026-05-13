package payment

import (
	"errors"
	"testing"
)

func TestNewGatewayClient_SupportedGateways(t *testing.T) {
	gateways := []string{"GLOBAL_PAY", "CASH", "ADYEN", " adyen "}
	for _, gateway := range gateways {
		client, err := NewGatewayClient(gateway)
		if err != nil {
			t.Fatalf("gateway %s should resolve: %v", gateway, err)
		}
		if client == nil {
			t.Fatalf("gateway %s returned nil client", gateway)
		}
	}
}

func TestNewGatewayClient_UnsupportedGateway(t *testing.T) {
	client, err := NewGatewayClient("UNKNOWN_GATEWAY")
	if err == nil {
		t.Fatalf("expected error for unsupported gateway, got client=%v", client)
	}
}

func TestAdyenGateway_DirectOperationsUnsupported(t *testing.T) {
	client, err := NewGatewayClient("ADYEN")
	if err != nil {
		t.Fatalf("adyen resolver should succeed: %v", err)
	}

	if err := client.Charge("order-1", 1000); !errors.Is(err, ErrAdyenDirectOperationUnsupported) {
		t.Fatalf("adyen charge error = %v, want ErrAdyenDirectOperationUnsupported", err)
	}
	if err := client.Refund("order-1", 250); !errors.Is(err, ErrAdyenDirectOperationUnsupported) {
		t.Fatalf("adyen refund error = %v, want ErrAdyenDirectOperationUnsupported", err)
	}
}
