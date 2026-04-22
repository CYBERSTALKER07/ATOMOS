package payment

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	goKafka "github.com/segmentio/kafka-go"
)

// Refund status constants matching the Refunds table CHECK constraint.
const (
	RefundPending        = "PENDING"
	RefundSettled        = "SETTLED"
	RefundFailed         = "FAILED"
	RefundManualRequired = "MANUAL_REQUIRED"
)

// RefundRequest is the input for initiating a supplier-triggered refund.
type RefundRequest struct {
	OrderID   string `json:"order_id"`
	Reason    string `json:"reason"`
	AmountUZS int64  `json:"amount_uzs"` // 0 = full refund (uses session LockedAmount)
}

// RefundResult is the output after a refund attempt.
type RefundResult struct {
	RefundID         string `json:"refund_id"`
	Status           string `json:"status"`
	AmountUZS        int64  `json:"amount_uzs"`
	Gateway          string `json:"gateway"`
	ProviderRefundID string `json:"provider_refund_id,omitempty"`
}

// RefundService handles the refund lifecycle: validation, gateway call,
// ledger reversal, and Kafka event emission.
type RefundService struct {
	spanner     *spanner.Client
	kafkaWriter KafkaWriter
	feeBP       int64 // Platform fee in basis points (0 = zero-fee era)
}

// KafkaWriter is a minimal interface for emitting events.
type KafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...goKafka.Message) error
}

// NewRefundService creates a refund service. feeBP is the platform commission
// in basis points (e.g., 500 = 5%). Pass 0 for zero-fee era.
func NewRefundService(sc *spanner.Client, kw KafkaWriter, feeBP int64) *RefundService {
	return &RefundService{spanner: sc, kafkaWriter: kw, feeBP: feeBP}
}

// InitiateRefund validates the order state, calls the gateway, creates the
// Refunds row, writes reversal ledger entries, and emits PAYMENT_REFUNDED.
func (rs *RefundService) InitiateRefund(ctx context.Context, req RefundRequest, initiatedBy string) (*RefundResult, error) {
	// 1. Read order + session in single snapshot
	row, err := rs.spanner.Single().ReadRow(ctx, "Orders",
		spanner.Key{req.OrderID},
		[]string{"OrderId", "SupplierId", "RetailerId", "State", "PaymentGateway", "Amount"})
	if err != nil {
		return nil, fmt.Errorf("read order: %w", err)
	}

	var orderID, supplierID, retailerID, state, gateway string
	var orderAmount int64
	if err := row.Columns(&orderID, &supplierID, &retailerID, &state, &gateway, &orderAmount); err != nil {
		return nil, fmt.Errorf("parse order: %w", err)
	}

	// Only COMPLETED or CANCELLED orders can be refunded
	if state != "COMPLETED" && state != "CANCELLED" {
		return nil, fmt.Errorf("order %s is in state %s — only COMPLETED or CANCELLED orders can be refunded", orderID, state)
	}

	// 2. Find settled payment session for this order
	stmt := spanner.Statement{
		SQL:    `SELECT SessionId, Gateway, LockedAmount, PaidAmount FROM PaymentSessions WHERE OrderId = @orderID AND Status = 'SETTLED' LIMIT 1`,
		Params: map[string]interface{}{"orderID": orderID},
	}
	iter := rs.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	sessRow, err := iter.Next()
	if err != nil {
		return nil, fmt.Errorf("no settled payment session for order %s: %w", orderID, err)
	}

	var sessionID, sessGateway string
	var lockedAmount spanner.NullInt64
	var paidAmount spanner.NullInt64
	if err := sessRow.Columns(&sessionID, &sessGateway, &lockedAmount, &paidAmount); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}

	// Determine refund amount
	refundAmount := req.AmountUZS
	if refundAmount <= 0 {
		if paidAmount.Valid && paidAmount.Int64 > 0 {
			refundAmount = paidAmount.Int64
		} else {
			refundAmount = lockedAmount.Int64
		}
	}

	// Guard: refund cannot exceed paid amount
	maxRefundable := lockedAmount.Int64
	if paidAmount.Valid && paidAmount.Int64 > 0 {
		maxRefundable = paidAmount.Int64
	}
	if refundAmount > maxRefundable {
		return nil, fmt.Errorf("refund amount %d exceeds maximum refundable %d", refundAmount, maxRefundable)
	}

	// 3. Call payment gateway for refund
	refundID := uuid.New().String()
	providerRefundID := ""
	refundStatus := RefundPending

	if sessGateway == "CASH" {
		// Cash refunds are manual — mark for manual processing
		refundStatus = RefundManualRequired
		log.Printf("[REFUND] Cash refund for order %s — requires manual processing", orderID)
	} else {
		gw, gwErr := NewGatewayClient(sessGateway)
		if gwErr != nil {
			refundStatus = RefundManualRequired
			log.Printf("[REFUND] Cannot create gateway client (%s) for refund on order %s: %v", sessGateway, orderID, gwErr)
		} else {
			if refErr := gw.Refund(orderID, refundAmount); refErr != nil {
				refundStatus = RefundFailed
				log.Printf("[REFUND] Gateway refund failed for order %s via %s: %v", orderID, sessGateway, refErr)
			} else {
				refundStatus = RefundSettled
				providerRefundID = fmt.Sprintf("GW-%s-%s", sessGateway, refundID[:8])
				log.Printf("[REFUND] Gateway refund succeeded for order %s via %s: %d UZS", orderID, sessGateway, refundAmount)
			}
		}
	}

	// 4. Persist refund + reversal ledger entries in single transaction
	now := time.Now()
	_, txErr := rs.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Insert Refunds row
		refundMut := spanner.Insert("Refunds",
			[]string{"RefundId", "OrderId", "SessionId", "Gateway", "AmountUZS", "Reason", "Status", "ProviderRefundId", "InitiatedBy", "CreatedAt"},
			[]interface{}{refundID, orderID, sessionID, sessGateway, refundAmount, req.Reason, refundStatus, providerRefundID, initiatedBy, now},
		)

		mutations := []*spanner.Mutation{refundMut}

		// Only create reversal ledger entries if gateway actually refunded
		if refundStatus == RefundSettled {
			// Reversal: debit supplier account + debit lab commission account
			// Compute in tiyin using basis-point math — matches ComputeSplitRecipients exactly
			totalTiyin := refundAmount * 100
			labReversal := totalTiyin * rs.feeBP / 10000
			supplierReversal := totalTiyin - labReversal

			labTxnID := fmt.Sprintf("TXN-REFUND-LAB-%s", refundID[:8])
			supTxnID := fmt.Sprintf("TXN-REFUND-SUP-%s", refundID[:8])

			// Debit The Lab (reverse the commission)
			mutations = append(mutations, spanner.Insert("LedgerEntries",
				[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType", "CreatedAt"},
				[]interface{}{labTxnID, orderID, "ACC-THE-LAB", -labReversal, "DEBIT_REFUND", now},
			))

			// Debit the supplier (reverse the payout)
			mutations = append(mutations, spanner.Insert("LedgerEntries",
				[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType", "CreatedAt"},
				[]interface{}{supTxnID, orderID, supplierID, -supplierReversal, "DEBIT_REFUND", now},
			))

			// Update session status to reflect refund
			mutations = append(mutations, spanner.Update("PaymentSessions",
				[]string{"SessionId", "Status", "UpdatedAt"},
				[]interface{}{sessionID, "REFUNDED", spanner.CommitTimestamp},
			))
		}

		return txn.BufferWrite(mutations)
	})

	if txErr != nil {
		return nil, fmt.Errorf("refund transaction failed: %w", txErr)
	}

	// 5. Emit Kafka event for notification dispatcher
	if rs.kafkaWriter != nil {
		rs.emitRefundEvent(orderID, retailerID, supplierID, refundID, refundAmount, refundStatus)
	}

	log.Printf("[REFUND] Refund %s for order %s: %d UZS → %s", refundID, orderID, refundAmount, refundStatus)

	return &RefundResult{
		RefundID:         refundID,
		Status:           refundStatus,
		AmountUZS:        refundAmount,
		Gateway:          sessGateway,
		ProviderRefundID: providerRefundID,
	}, nil
}

// emitRefundEvent publishes a PAYMENT_REFUNDED event for the notification dispatcher.
func (rs *RefundService) emitRefundEvent(orderID, retailerID, supplierID, refundID string, amount int64, status string) {
	payload := fmt.Sprintf(`{"order_id":"%s","retailer_id":"%s","supplier_id":"%s","refund_id":"%s","amount":%d,"status":"%s","timestamp":"%s"}`,
		orderID, retailerID, supplierID, refundID, amount, status, time.Now().Format(time.RFC3339))

	msg := goKafka.Message{
		Key:   []byte("PAYMENT_REFUNDED"),
		Value: []byte(payload),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rs.kafkaWriter.WriteMessages(ctx, msg); err != nil {
		log.Printf("[REFUND] Failed to emit PAYMENT_REFUNDED event for order %s: %v", orderID, err)
	}
}

// GetRefundsByOrder returns all refunds for a given order.
func (rs *RefundService) GetRefundsByOrder(ctx context.Context, orderID string) ([]RefundResult, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT RefundId, Status, AmountUZS, Gateway, ProviderRefundId FROM Refunds WHERE OrderId = @orderID ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"orderID": orderID},
	}
	iter := rs.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []RefundResult
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var r RefundResult
		var providerRef spanner.NullString
		if err := row.Columns(&r.RefundID, &r.Status, &r.AmountUZS, &r.Gateway, &providerRef); err != nil {
			continue
		}
		r.ProviderRefundID = providerRef.StringVal
		results = append(results, r)
	}

	return results, nil
}
