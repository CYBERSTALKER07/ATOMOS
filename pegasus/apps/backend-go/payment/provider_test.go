package payment

import "testing"

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

func TestAdyenGateway_FailsClosed(t *testing.T) {
	client, err := NewGatewayClient("ADYEN")
	if err != nil {
		t.Fatalf("adyen resolver should succeed: %v", err)
	}

	if err := client.Charge("order-1", 1000); err == nil {
		t.Fatal("adyen charge should fail until provider adapter is configured")
	}
	if err := client.Refund("order-1", 250); err == nil {
		t.Fatal("adyen refund should fail until provider adapter is configured")
	}
}
