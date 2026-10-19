package order

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// FiscalSweeperBackoffFn is the algorithm used to determine retry delay.
// Exported for test overrides.
var FiscalSweeperBackoffFn = CalculateFiscalBackoff

// CalculateFiscalBackoff returns full-jitter exponential backoff:
// random(0, min(300s, 2^attempt * 500ms))
func CalculateFiscalBackoff(failedAttempts int64) time.Duration {
	if failedAttempts <= 0 {
		return 0
	}
	base := 500 * time.Millisecond
	maxCap := 300 * time.Second

	shift := failedAttempts
	if shift > 10 { // 2^10 * 500ms = 512s > 300s maxCap
		shift = 10
	}
	ceiling := (int64(1) << shift) * int64(base)
	if ceiling > int64(maxCap) {
		ceiling = int64(maxCap)
	}
	if ceiling <= 0 {
		return 0
	}
	// Full jitter: random interval in [0, ceiling]
	return time.Duration(rand.Int63n(ceiling))
}

// StartFiscalSweeper launches the background daemon that continuously sweeps
// FISCAL_FAILED orders and retries them with jittered backoff or moves to DLQ.
func StartFiscalSweeper(svc *Service) {
	if svc == nil {
		return
	}
	svc.log.Info("fiscal retry sweeper started")
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			if err := svc.RunFiscalSweeperOnce(ctx); err != nil {
				svc.log.Warn("fiscal sweeper iteration failed", "err", err)
			}
			cancel()
		}
	}()
}

// RunFiscalSweeperOnce performs one sweep across pending FISCAL_FAILED orders.
func (s *Service) RunFiscalSweeperOnce(ctx context.Context) error {
	orders, err := s.repo.ListOrdersByStatus(ctx, "", string(StatusFiscalFailed), 100)
	if err != nil {
		return fmt.Errorf("list fiscal failed orders: %w", err)
	}
	now := s.now().UTC()

	for _, o := range orders {
		// If already dead-lettered / quarantined, skip
		if o.FiscalStatus == FiscalStatusDeadLetter {
			continue
		}

		failedCount, countErr := s.repo.CountFiscalAttemptsByStatus(ctx, o.OrderID, FiscalAttemptFailed)
		if countErr != nil {
			s.log.WarnContext(ctx, "failed counting fiscal attempts", "order_id", o.OrderID, "err", countErr)
			continue
		}

		// 1. If exhausted maximum failed attempts -> Quarantine to OutboxDeadLetters and alert Control Tower
		if failedCount >= FiscalMaxFailedAttempts {
			if qErr := s.quarantineFiscalOrderToDLQ(ctx, o, failedCount); qErr != nil {
				s.log.ErrorContext(ctx, "failed quarantining fiscal order to DLQ", "order_id", o.OrderID, "err", qErr)
			}
			continue
		}

		// 2. If retry budget remains -> check full-jitter exponential backoff window
		requiredBackoff := FiscalSweeperBackoffFn(failedCount)
		elapsed := now.Sub(o.UpdatedAt)
		if elapsed < requiredBackoff {
			// Backoff window has not elapsed yet
			continue
		}

		// Trigger automatic fiscal retry
		claims := auth.Claims{
			Subject:    "system:fiscal-sweeper",
			Role:       auth.RoleAdmin,
			SupplierID: o.SupplierID,
		}
		if _, rErr := s.RetryFiscal(ctx, claims, o.OrderID); rErr != nil {
			s.log.WarnContext(ctx, "fiscal sweeper auto-retry failed", "order_id", o.OrderID, "attempt", failedCount+1, "err", rErr)
		} else {
			s.log.InfoContext(ctx, "fiscal sweeper auto-retry initiated", "order_id", o.OrderID, "attempt", failedCount+1)
		}
	}

	return nil
}

func (s *Service) quarantineFiscalOrderToDLQ(ctx context.Context, orderRecord Order, failedAttempts int64) error {
	now := s.now().UTC()
	orderRecord.FiscalStatus = FiscalStatusDeadLetter
	orderRecord.UpdatedAt = now

	payload := map[string]any{
		"order_id":        orderRecord.OrderID,
		"supplier_id":     orderRecord.SupplierID,
		"retailer_id":     orderRecord.RetailerID,
		"total_minor":     orderRecord.TotalMinor,
		"currency":        orderRecord.Currency,
		"failed_attempts": failedAttempts,
		"reason":          "fiscal_max_attempts_exhausted",
		"quarantined_at":  now.Format(time.RFC3339),
	}
	payloadBytes, _ := json.Marshal(payload)

	var inTxn func(context.Context, *spanner.ReadWriteTransaction) error
	if s.spannerClient != nil {
		inTxn = func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			dlqRow := map[string]any{
				"EventId":        uuid.New().String(),
				"AggregateType":  events.AggregateOrder,
				"AggregateId":    orderRecord.OrderID,
				"TopicName":      "fiscal.dlq",
				"Payload":        payloadBytes,
				"CreatedAt":      orderRecord.CreatedAt,
				"DeadLetteredAt": spanner.CommitTimestamp,
				"Attempts":       failedAttempts,
				"LastError":      fmt.Sprintf("fiscalization exhausted %d attempts", failedAttempts),
				"SupplierId":     orderRecord.SupplierID,
			}
			return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("OutboxDeadLetters", dlqRow)})
		}
	}

	err := s.repo.UpdateOrderWithTxn(ctx, orderRecord, nil, inTxn, func(txn outbox.TxnBuffer) error {
		return emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Order:          orderRecord,
			PreviousStatus: StatusFiscalFailed,
			Reason:         "fiscal_dead_lettered",
			ActorID:        "system:fiscal-sweeper",
		})
	})
	if err != nil {
		return fmt.Errorf("update order to dead letter: %w", err)
	}

	s.afterOrderMutation(ctx, orderRecord)

	// Alert Control Tower / Supplier Hub
	if s.supplierHub != nil {
		s.supplierHub.Broadcast(ctx, "supplier:"+orderRecord.SupplierID, payloadBytes)
	}

	s.log.ErrorContext(ctx, "order fiscalization exhausted and quarantined to DLQ",
		"order_id", orderRecord.OrderID,
		"supplier_id", orderRecord.SupplierID,
		"failed_attempts", failedAttempts)

	return nil
}
