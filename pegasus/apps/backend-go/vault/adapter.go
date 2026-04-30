package vault

import (
	"context"

	"backend-go/payment"
)

// PaymentVaultAdapter wraps vault.Service to implement payment.VaultResolver.
type PaymentVaultAdapter struct {
	Svc *Service
}

// GetDecryptedConfigByOrder implements payment.VaultResolver.
func (a *PaymentVaultAdapter) GetDecryptedConfigByOrder(ctx context.Context, orderId, gatewayName string) (*payment.VaultConfig, error) {
	cfg, err := a.Svc.GetDecryptedConfigByOrder(ctx, orderId, gatewayName)
	if err != nil {
		return nil, err
	}
	return &payment.VaultConfig{
		SecretKey:   cfg.SecretKey,
		MerchantId:  cfg.MerchantId,
		ServiceId:   cfg.ServiceId,
		RecipientId: cfg.RecipientId,
	}, nil
}
