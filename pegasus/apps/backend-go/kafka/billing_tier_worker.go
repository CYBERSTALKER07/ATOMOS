package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"backend-go/internal/services/billing"

	"cloud.google.com/go/spanner"
	goKafka "github.com/segmentio/kafka-go"

	"backend-go/kafka/workerpool"
)

type BillingTierDeps struct {
	SpannerClient *spanner.Client
}

// StartBillingTierWorker starts the consumer that listens for ORDER_FINALIZED events
// and maintains idempotent billing meters plus fee-milestone adjustments.
func StartBillingTierWorker(ctx context.Context, deps BillingTierDeps, brokerAddress string) {
	meterSvc := billing.NewMeterWorker(deps.SpannerClient)

	reader := goKafka.NewReader(goKafka.ReaderConfig{
		Brokers:  []string{brokerAddress},
		Topic:    TopicMain,
		GroupID:  "pegasus-billing-tier-worker-group",
		MinBytes: 1,
		MaxBytes: 10 << 20,
	})

	pool, err := workerpool.New(workerpool.Config{
		Source: reader,
		Name:   "billing-tier-worker",
		Logger: slog.Default(),
		Handler: func(ctx context.Context, m goKafka.Message) error {
			eventType := EventType(m.Headers, m.Key)
			if eventType != EventOrderFinalized {
				return nil
			}
			return handleOrderFinalizedForBilling(ctx, meterSvc, m.Value)
		},
		OnFailure: func(ctx context.Context, m goKafka.Message, handlerErr error) {
			eventType := EventType(m.Headers, m.Key)
			orderID := extractOrderID(m.Value)
			slog.ErrorContext(ctx, "billing_tier_worker.handler_error",
				"event", eventType,
				"order_id", orderID,
				"partition", m.Partition,
				"offset", m.Offset,
				"err", handlerErr,
			)
			RouteToDLQ(LogisticsEvent{
				EventName: eventType,
				OrderId:   orderID,
				Timestamp: time.Now().UTC(),
			}, handlerErr.Error())
		},
	})

	if err != nil {
		slog.ErrorContext(ctx, "failed to init billing tier worker pool", "err", err)
		return
	}

	go func() {
		defer reader.Close()
		if err := pool.Run(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "billing_tier_worker: pool exited", "err", err)
		}
	}()
}

func handleOrderFinalizedForBilling(ctx context.Context, meterSvc *billing.MeterWorker, value []byte) error {
	var ev OrderFinalizedEvent
	if err := json.Unmarshal(value, &ev); err != nil {
		return fmt.Errorf("billing_tier: unmarshal ORDER_FINALIZED: %w", err)
	}

	if ev.OrderID == "" {
		return nil
	}

	return meterSvc.ProcessFinalizedOrder(ctx, billing.FinalizedOrderInput{
		OrderID:    ev.OrderID,
		InvoiceID:  ev.InvoiceID,
		SupplierID: ev.SupplierID,
		RetailerID: ev.RetailerID,
		Timestamp:  ev.Timestamp,
	})
}

func extractOrderID(payload []byte) string {
	var body struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return ""
	}
	return body.OrderID
}
