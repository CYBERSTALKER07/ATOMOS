package payment

import (
	"context"
	"errors"
	"testing"
)

func TestAdyenExecutionClient_RefundPayment_Gate(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		wantErr bool
	}{
		{
			name:    "returns ErrAdyenDirectOperationUnsupported when disabled",
			enabled: false,
			wantErr: true,
		},
		{
			// When enabled, it should try to create client and might fail on credentials or just return an error from unmocked http call,
			// but importantly it won't be ErrAdyenDirectOperationUnsupported.
			name:    "does not return ErrAdyenDirectOperationUnsupported when enabled",
			enabled: true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &adyenExecutionClient{
				creds: AdyenCredentials{
					DirectExecutionEnabled: tt.enabled,
				},
			}

			_, err := client.RefundPayment(context.Background(), ProviderRefundRequest{
				OrderID: "order123",
			})

			if tt.wantErr {
				if !errors.Is(err, ErrAdyenDirectOperationUnsupported) {
					t.Errorf("expected ErrAdyenDirectOperationUnsupported, got %v", err)
				}
			} else {
				if errors.Is(err, ErrAdyenDirectOperationUnsupported) {
					t.Errorf("did not expect ErrAdyenDirectOperationUnsupported, got %v", err)
				}
			}
		})
	}
}
