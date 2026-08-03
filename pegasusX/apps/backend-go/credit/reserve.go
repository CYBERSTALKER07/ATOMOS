package credit

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Reserve holds credit headroom for an order at create (idempotent per order).
func (s *Service) Reserve(ctx context.Context, retailerID, supplierID, orderID string, amountMinor int64) error {
	if amountMinor <= 0 || orderID == "" {
		return nil
	}
	check, err := s.CheckOrder(ctx, retailerID, supplierID, amountMinor)
	if err != nil {
		return err
	}
	if !check.Allowed {
		return fmt.Errorf("%w: %s", ErrLimitBreached, check.Reason)
	}
	return s.repo.ReserveOrder(ctx, OrderReservation{
		OrderID:     orderID,
		RetailerID:  retailerID,
		SupplierID:  supplierID,
		AmountMinor: amountMinor,
		Status:      ReservationReserved,
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateCreditProfile, retailerID, events.TopicMain, events.CreditProfileEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: s.now().Format(time.RFC3339Nano)},
			ProfileID:      profileID(retailerID, supplierID),
			RetailerID:     retailerID,
			SupplierID:     supplierID,
			CurrentBalance: 0,
			RiskTier:       string(RiskTierMedium),
			Reason:         fmt.Sprintf("credit_reserve:%s", orderID),
		})
	})
}

// ReleaseReserve frees a reservation (cancel / cash settlement). Idempotent.
func (s *Service) ReleaseReserve(ctx context.Context, orderID string) error {
	if orderID == "" {
		return nil
	}
	return s.repo.ReleaseOrderReservation(ctx, orderID, func(txn outbox.TxnBuffer) error {
		return nil
	})
}

// ConvertReserve moves reservation into balance (credit leave). Idempotent.
// Prefer ConvertReserveInTxn when already inside an order transition txn.
func (s *Service) ConvertReserve(ctx context.Context, orderID string) error {
	if orderID == "" {
		return nil
	}
	return s.repo.ConvertOrderReservation(ctx, orderID, func(txn outbox.TxnBuffer) error {
		return nil
	})
}

// MarkBalanceInTxn increases balance inside an existing Spanner RW txn (same-txn credit leave).
// If a reservation exists for orderID it is converted; otherwise amountMinor is applied directly.
// Idempotent via OrderCreditReservations status / ledger-style CAS on reservation row.
func (s *Service) MarkBalanceInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, amountMinor int64) error {
	if amountMinor <= 0 || s == nil || s.repo == nil {
		return nil
	}
	if mut, ok := s.repo.(TxnMutator); ok {
		return mut.MarkBalanceInTxn(ctx, txn, retailerID, supplierID, orderID, amountMinor)
	}
	// Fallback: best-effort convert then mark outside (tests / memory).
	_ = s.ConvertReserve(ctx, orderID)
	return s.MarkBalance(ctx, retailerID, supplierID, amountMinor, orderID)
}

// TxnMutator is implemented by Spanner-backed repos for same-txn credit leave.
type TxnMutator interface {
	MarkBalanceInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, amountMinor int64) error
}
