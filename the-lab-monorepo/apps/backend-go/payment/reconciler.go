package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"time"

	"backend-go/kafka/workerpool"
	"config"

	"cloud.google.com/go/spanner"
	"github.com/segmentio/kafka-go"
)

// ReconciliationEvent is the canonical payload emitted to the
// payment_reconciliation Kafka topic for every financial state change.
type ReconciliationEvent struct {
	EventType  string    `json:"event_type"` // FULL_CAPTURE | PARTIAL_REFUND
	OrderID    string    `json:"order_id"`
	RetailerID string    `json:"retailer_id"`
	Amount     int64     `json:"amount"`
	Timestamp  time.Time `json:"timestamp"`
	Gateway    string    `json:"gateway"`
}

type ReconcilerService struct {
	kafkaReader   *kafka.Reader
	kafkaWriter   *kafka.Writer
	spannerClient *spanner.Client
}

func NewReconcilerService(cfg *config.EnvConfig, spannerClient *spanner.Client) *ReconcilerService {
	return &ReconcilerService{
		kafkaReader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  []string{cfg.KafkaBrokerAddress},
			GroupID:  "payment-reconciliation-group",
			Topic:    "lab-logistics-events",
			MinBytes: 10e3,
			MaxBytes: 10e6,
		}),
		kafkaWriter: &kafka.Writer{
			Addr:     kafka.TCP(cfg.KafkaBrokerAddress),
			Topic:    "payment_reconciliation",
			Balancer: &kafka.LeastBytes{},
		},
		spannerClient: spannerClient,
	}
}

// emitReconciliationEvent publishes a ReconciliationEvent to the canonical
// payment_reconciliation topic. This is the Kafka handshake required by the
// Financial Integrity rule — no transaction is complete without it.
func (s *ReconcilerService) emitReconciliationEvent(ctx context.Context, evt ReconciliationEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal reconciliation event: %w", err)
	}
	err = s.kafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(evt.OrderID),
		Value: data,
	})
	if err != nil {
		return fmt.Errorf("kafka emit failed for order %s: %w", evt.OrderID, err)
	}
	log.Printf("[KAFKA] Emitted %s event for order %s", evt.EventType, evt.OrderID)
	return nil
}

// ExecuteFullCapture performs a one-shot full capture immediately at order
// creation time. This eliminates the pre-auth escrow race condition where
// Payme/Click would release funds after 24–72h if delivery was delayed.
func (s *ReconcilerService) ExecuteFullCapture(ctx context.Context, orderID, retailerID, gateway string, amount int64) error {
	gw, err := NewGatewayClient(gateway)
	if err != nil {
		return fmt.Errorf("gateway init failed: %w", err)
	}

	if err := gw.Charge(orderID, amount); err != nil {
		return fmt.Errorf("full capture failed: %w", err)
	}

	// Financial Integrity — emit Kafka handshake
	return s.emitReconciliationEvent(ctx, ReconciliationEvent{
		EventType:  "FULL_CAPTURE",
		OrderID:    orderID,
		RetailerID: retailerID,
		Amount:     amount,
		Timestamp:  time.Now().UTC(),
		Gateway:    gateway,
	})
}

// ExecutePartialRefund is triggered when the driver's amend submission contains
// REJECTED_DAMAGED line items. The refund delta is calculated by the caller
// (AmendOrder in order/service.go) and passed in directly.
func (s *ReconcilerService) ExecutePartialRefund(ctx context.Context, orderID, retailerID, gateway string, refundAmount int64) error {
	if refundAmount <= 0 {
		log.Printf("[PAYMENT] Zero refund requested for order %s — skipping", orderID)
		return nil
	}

	gw, err := NewGatewayClient(gateway)
	if err != nil {
		return fmt.Errorf("gateway init failed for refund: %w", err)
	}

	if err := gw.Refund(orderID, refundAmount); err != nil {
		return fmt.Errorf("partial refund failed: %w", err)
	}

	// Financial Integrity — emit Kafka handshake for refund too
	return s.emitReconciliationEvent(ctx, ReconciliationEvent{
		EventType:  "PARTIAL_REFUND",
		OrderID:    orderID,
		RetailerID: retailerID,
		Amount:     refundAmount,
		Timestamp:  time.Now().UTC(),
		Gateway:    gateway,
	})
}

// Start begins consuming ORDER_COMPLETED events via a partition-parallel worker
// pool. Blocks until ctx is cancelled.
func (s *ReconcilerService) Start(ctx context.Context) {
	pool, err := workerpool.New(workerpool.Config{
		Source: s.kafkaReader,
		Name:   "payment-reconciler",
		Handler: func(ctx context.Context, msg kafka.Message) error {
			eventType := messageEventType(msg)
			if eventType != eventOrderCompleted {
				return nil
			}
			var event struct {
				OrderID    string `json:"order_id"`
				RetailerID string `json:"retailer_id"`
			}
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				return fmt.Errorf("payment reconciler: invalid payload: %w", err)
			}
			if err := s.markCompleted(ctx, event.OrderID); err != nil {
				return fmt.Errorf("payment reconciler: markCompleted %s: %w", event.OrderID, err)
			}
			return nil
		},
		OnFailure: func(ctx context.Context, msg kafka.Message, handlerErr error) {
			slog.ErrorContext(ctx, "payment_reconciler.handler_error",
				"partition", msg.Partition, "offset", msg.Offset, "err", handlerErr)
		},
	})
	if err != nil {
		slog.Error("payment_reconciler: pool init failed", "err", err)
		return
	}
	slog.Info("[PAYMENT] reconciler consumer started")
	if err := pool.Run(ctx); err != nil && ctx.Err() == nil {
		slog.Error("payment_reconciler: pool exited", "err", err)
	}
}

func messageEventType(msg kafka.Message) string {
	for _, h := range msg.Headers {
		if h.Key == "event_type" {
			return string(h.Value)
		}
	}
	// Legacy fallback: direct producers used event type as message key.
	return string(msg.Key)
}

// markCompleted performs an idempotent state transition to COMPLETED.
// Accepts orders in ARRIVED, AWAITING_PAYMENT, or PENDING_CASH_COLLECTION.
// This is a reconciliation fallback — primary completion happens in order/service.go.
//
// For GLOBAL_PAY orders, the reconciler requires a settled payment session
// before projecting PaymentStatus = PAID. If no settled session exists, the
// order is not transitioned — the Global Pay sweeper or manual reconciliation
// must resolve the payment first.
func (s *ReconcilerService) markCompleted(ctx context.Context, orderID string) error {
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    `SELECT State, RetailerId, Amount, PaymentGateway FROM Orders WHERE OrderId = @id LIMIT 1`,
			Params: map[string]interface{}{"id": orderID},
		}
		iter := txn.Query(ctx, stmt)
		row, err := iter.Next()
		iter.Stop()
		if err != nil {
			return fmt.Errorf("query failed: %w", err)
		}

		var state string
		var retailerID string
		var amount int64
		var gatewayNull spanner.NullString
		if err := row.Columns(&state, &retailerID, &amount, &gatewayNull); err != nil {
			return fmt.Errorf("column scan failed: %w", err)
		}

		if state == "COMPLETED" {
			log.Printf("[PAYMENT] order %s already COMPLETED — idempotent skip", orderID)
			return nil
		}
		// Allow transition from any late-lifecycle state
		validStates := map[string]bool{"ARRIVED": true, "AWAITING_PAYMENT": true, "PENDING_CASH_COLLECTION": true}
		if !validStates[state] {
			return fmt.Errorf("order %s cannot transition from %s to COMPLETED", orderID, state)
		}

		gateway := gatewayNull.StringVal

		// GLOBAL_PAY guard: do not blindly project PAID — require a settled session
		if gateway == "GLOBAL_PAY" {
			sessionStmt := spanner.Statement{
				SQL: `SELECT Status FROM PaymentSessions
				      WHERE OrderId = @orderId AND Gateway = 'GLOBAL_PAY'
				      ORDER BY CreatedAt DESC LIMIT 1`,
				Params: map[string]interface{}{"orderId": orderID},
			}
			sessionIter := txn.Query(ctx, sessionStmt)
			sessionRow, sessionErr := sessionIter.Next()
			sessionIter.Stop()

			if sessionErr != nil {
				log.Printf("[PAYMENT] GLOBAL_PAY order %s has no payment session — skipping COMPLETED projection", orderID)
				return fmt.Errorf("GLOBAL_PAY order %s requires a settled payment session before completion", orderID)
			}

			var sessionStatus string
			if colErr := sessionRow.Columns(&sessionStatus); colErr != nil {
				return fmt.Errorf("session status scan failed: %w", colErr)
			}

			if sessionStatus != SessionSettled {
				log.Printf("[PAYMENT] GLOBAL_PAY order %s session is %s, not SETTLED — skipping COMPLETED projection", orderID, sessionStatus)
				return fmt.Errorf("GLOBAL_PAY order %s session is %s; awaiting settlement before completion", orderID, sessionStatus)
			}
		}

		updateStmt := spanner.Statement{
			SQL:    `UPDATE Orders SET State = 'COMPLETED', PaymentStatus = 'PAID' WHERE OrderId = @id`,
			Params: map[string]interface{}{"id": orderID},
		}
		_, err = txn.Update(ctx, updateStmt)
		return err
	})
	return err
}

func (s *ReconcilerService) Close() error {
	_ = s.kafkaWriter.Close()
	return s.kafkaReader.Close()
}
