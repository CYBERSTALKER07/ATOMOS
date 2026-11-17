package order

import (
	"context"

	"cloud.google.com/go/spanner"
)

// CapturedPaymentStatement exposes capturedPaymentStatement for testing.
func CapturedPaymentStatement(orderID string) spanner.Statement {
	return capturedPaymentStatement(orderID)
}

// CheckSettlementTenderSufficiency exposes checkSettlementTenderSufficiency for testing.
func CheckSettlementTenderSufficiency(orderID string, paid, exceptions, delivered int64) error {
	return checkSettlementTenderSufficiency(orderID, paid, exceptions, delivered)
}

// ShouldClearCreditAndAR exposes shouldClearCreditAndAR for testing.
func ShouldClearCreditAndAR(leg *PaymentLeg) bool {
	return shouldClearCreditAndAR(leg)
}

// SettleOutstandingCardPayment exposes settleOutstandingCardPayment for testing.
func (s *Service) SettleOutstandingCardPayment(ctx context.Context, orderRecord Order) error {
	return s.settleOutstandingCardPayment(ctx, orderRecord)
}

// AggregateSKUQuantities exposes aggregateSKUQuantities for testing.
func AggregateSKUQuantities(lineItems []LineItem) map[string]int64 {
	return aggregateSKUQuantities(lineItems)
}
