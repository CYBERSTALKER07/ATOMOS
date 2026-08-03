package bootstrap

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/claims"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

// claimPaymentSettler adapts payment.Service to claims.ChargebackSettler.
type claimPaymentSettler struct {
	pay *payment.Service
}

func (s *claimPaymentSettler) SettleClaimChargeback(ctx context.Context, in claims.ClaimSettlement) (claims.SettlementResult, error) {
	if s == nil || s.pay == nil {
		return claims.SettlementResult{
			AmountMinor: in.AmountMinor,
			Currency:    in.Currency,
			Mode:        "LEDGER_PENDING_SETTLER",
		}, nil
	}
	res, err := s.pay.SettleClaimChargeback(ctx, payment.ClaimChargebackInput{
		ClaimID:           in.ClaimID,
		OrderID:           in.OrderID,
		SupplierID:        in.SupplierID,
		RetailerID:        in.RetailerID,
		AmountMinor:       in.AmountMinor,
		Currency:          in.Currency,
		SkipGatewayRefund: in.SkipGatewayRefund,
	})
	if err != nil {
		return claims.SettlementResult{}, err
	}
	return claims.SettlementResult{
		ChargebackID:    res.ChargebackID,
		AmountMinor:     res.AmountMinor,
		Currency:        res.Currency,
		Gateway:         res.Gateway,
		GatewayRefunded: res.GatewayRefunded,
		ProviderRef:     res.ProviderRef,
		Mode:            res.Mode,
	}, nil
}
