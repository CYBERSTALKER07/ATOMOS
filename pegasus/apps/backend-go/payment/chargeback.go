package payment

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/spanner"
)

// ChargebackService handles provider-initiated chargebacks and reversals.
// Even if gateways don't actively push chargebacks yet, having the handler
// ready means we can wire it immediately when needed.
type ChargebackService struct {
	spanner *spanner.Client
}

func NewChargebackService(sc *spanner.Client) *ChargebackService {
	return &ChargebackService{spanner: sc}
}

// HandleChargeback records a provider-initiated chargeback as a LedgerAnomaly.
// Uses the existing LedgerAnomalies table (PK=OrderId) with Status=CHARGEBACK.
func (cs *ChargebackService) HandleChargeback(ctx context.Context, orderID, retailerID, gateway string, amountUZS int64) error {
	now := time.Now()

	// InsertOrUpdate because LedgerAnomalies PK is OrderId — one anomaly per order
	_, err := cs.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertOrUpdate("LedgerAnomalies",
			[]string{"OrderId", "RetailerId", "SpannerAmount", "GatewayAmount", "Currency", "GatewayProvider", "Status", "DetectedAt"},
			[]interface{}{orderID, retailerID, amountUZS, amountUZS, "UZS", gateway, "CHARGEBACK", now},
		)
		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	if err != nil {
		return fmt.Errorf("record chargeback: %w", err)
	}

	log.Printf("[CHARGEBACK] Recorded anomaly for order %s: %d UZS (gateway: %s)", orderID, amountUZS, gateway)
	return nil
}

// HandleReversal processes a gateway-initiated session reversal.
// This typically happens when a payment that was thought to be settled
// is reversed by the provider (e.g., insufficient funds discovered later).
func (cs *ChargebackService) HandleReversal(ctx context.Context, sessionID string) error {
	var orderID, retailerID, gateway string
	var lockedAmount int64
	now := time.Now()
	alreadyReversed := false

	_, err := cs.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "PaymentSessions",
			spanner.Key{sessionID},
			[]string{"SessionId", "OrderId", "RetailerId", "LockedAmount", "Status", "Gateway"})
		if err != nil {
			return fmt.Errorf("read session %s: %w", sessionID, err)
		}

		var sessID, status string
		if err := row.Columns(&sessID, &orderID, &retailerID, &lockedAmount, &status, &gateway); err != nil {
			return fmt.Errorf("parse session: %w", err)
		}
		if sessID == "" {
			return fmt.Errorf("session %s missing identifier", sessionID)
		}
		if status == SessionReversed {
			alreadyReversed = true
			return nil
		}
		if status != SessionSettled {
			return fmt.Errorf("session %s is %s, not SETTLED — cannot reverse", sessionID, status)
		}

		// Mark session as reversed
		m1 := spanner.Update("PaymentSessions",
			[]string{"SessionId", "Status", "UpdatedAt"},
			[]interface{}{sessionID, SessionReversed, spanner.CommitTimestamp},
		)

		// Record anomaly
		m2 := spanner.InsertOrUpdate("LedgerAnomalies",
			[]string{"OrderId", "RetailerId", "SpannerAmount", "GatewayAmount", "Currency", "GatewayProvider", "Status", "DetectedAt"},
			[]interface{}{orderID, retailerID, lockedAmount, int64(0), "UZS", gateway, "REVERSAL", now},
		)

		return txn.BufferWrite([]*spanner.Mutation{m1, m2})
	})
	if err != nil {
		return fmt.Errorf("reversal transaction: %w", err)
	}
	if alreadyReversed {
		log.Printf("[CHARGEBACK] Session %s already reversed", sessionID)
		return nil
	}

	log.Printf("[CHARGEBACK] Session %s reversed for order %s: %d UZS", sessionID, orderID, lockedAmount)
	return nil
}
