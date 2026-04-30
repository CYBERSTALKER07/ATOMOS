package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"

	"backend-go/kafka/workerpool"
	"config"

	"cloud.google.com/go/spanner"
	"github.com/segmentio/kafka-go"
)

type ReconcilerService struct {
	kafkaReader   *kafka.Reader
	spannerClient *spanner.Client
}

const (
	reconcilerTopicMain = "pegasus-logistics-events"
)

func NewReconcilerService(cfg *config.EnvConfig, spannerClient *spanner.Client) *ReconcilerService {
	return &ReconcilerService{
		kafkaReader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  []string{cfg.KafkaBrokerAddress},
			GroupID:  "payment-reconciliation-group",
			Topic:    reconcilerTopicMain,
			MinBytes: 10e3,
			MaxBytes: 10e6,
		}),
		spannerClient: spannerClient,
	}
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
	return s.kafkaReader.Close()
}
