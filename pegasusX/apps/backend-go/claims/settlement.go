package claims

import "context"

// ClaimSettlement is the input for supplier chargeback + optional card refund.
type ClaimSettlement struct {
	ClaimID           string
	OrderID           string
	SupplierID        string
	RetailerID        string
	AmountMinor       int64
	Currency          string
	LineItems         []ClaimLine
	SkipGatewayRefund bool
}

// ChargebackSettler records the marketplace-style supplier debit and optional
// PSP refund to the retailer. Implemented by payment.Service.
//
// Implementations MUST use a deterministic chargeback id derived from ClaimID
// so approve retries are idempotent at the ledger layer.
type ChargebackSettler interface {
	SettleClaimChargeback(ctx context.Context, in ClaimSettlement) (SettlementResult, error)
}
