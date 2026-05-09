package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	
	"cloud.google.com/go/spanner"
	goKafka "github.com/segmentio/kafka-go"
	
	"backend-go/kafka/workerpool"
)

type BillingTierDeps struct {
	SpannerClient *spanner.Client
}

// StartBillingTierWorker starts the consumer that listens for ORDER_COMPLETED events
// to adjust supplier take rates depending on monthly volume tiers.
func StartBillingTierWorker(ctx context.Context, deps BillingTierDeps, brokerAddress string) {
	reader := goKafka.NewReader(goKafka.ReaderConfig{
		Brokers:  []string{brokerAddress},
		Topic:    TopicMain,
		GroupID:  "pegasus-billing-tier-worker-group",
		MinBytes: 1,
		MaxBytes: 10 << 20,
	})

	pool, err := workerpool.New(workerpool.Config{
		Source:  reader,
		Name:    "billing-tier-worker",
		Logger:  slog.Default(),
		Handler: func(ctx context.Context, m goKafka.Message) error {
			eventType := EventType(m.Headers, m.Key)
			if eventType == EventOrderCompleted {
				return handleOrderCompletedForBilling(ctx, deps, m.Value)
			}
			return nil
		},
	})

	if err != nil {
		slog.ErrorContext(ctx, "failed to init billing tier worker pool", "err", err)
		return
	}

	go func() {
		defer reader.Close()
		if err := pool.Run(ctx); err != nil && ctx.Err() == nil { slog.ErrorContext(ctx, "billing_tier_worker: pool exited", "err", err) }
	}()
}

func handleOrderCompletedForBilling(ctx context.Context, deps BillingTierDeps, value []byte) error {
	// Fast-fail if not an OrderCompletedEvent envelope, though we could unmarshal the exact struct
	// Let's use the explicit OrderCompletedEvent struct from events.go
	var ev OrderCompletedEvent
	if err := json.Unmarshal(value, &ev); err != nil {
		return fmt.Errorf("billing_tier: unmarshal event: %w", err)
	}
	
	if ev.SupplierId == "" || ev.Amount <= 0 {
		return nil
	}

	// This is a minimal blueprint. A full implementation would aggregate the supplier's
	// rolling monthly volume, fetch their SupplierBillingTiers, and conditionally
	// update their Suppliers.TakeRateBasisPts via a spanner.ReadWriteTransaction.
	slog.InfoContext(ctx, "billing_tier_worker.evaluating_take_rate",
		"supplier_id", ev.SupplierId,
		"order_amount", ev.Amount,
		"order_id", ev.OrderID,
	)

	return nil
}
