package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"backend-go/fastjson"

	goKafka "github.com/segmentio/kafka-go"
)

// OrderSyncEvent is the immutable event shape emitted by the Desert Protocol.
// Each offline delivery that passes Redis deduplication fires one of these.
// The Treasurer consumer ignores these (listens for ORDER_COMPLETED only);
// a dedicated SyncConsumer should subscribe to TopicDriverSync.
type OrderSyncEvent struct {
	OrderID   string `json:"order_id"`
	DriverID  string `json:"driver_id"`
	NewStatus string `json:"new_status"` // 'DELIVERED' | 'REJECTED_DAMAGED'
	Signature string `json:"signature"`  // Offline SHA-256 from WatermelonDB
	Timestamp int64  `json:"timestamp"`  // Unix ms of the offline scan
}

// syncWriter is the singleton Kafka writer for the driver sync topic.
// Initialised lazily by EmitOrderSyncEvent; set explicitly via InitSyncWriter
// during server startup once cfg.KafkaBrokerAddress is known.
var syncWriter *goKafka.Writer

// InitSyncWriter must be called from main() so the broker address is known.
func InitSyncWriter(brokerAddress string) {
	syncWriter = &goKafka.Writer{
		Addr:     goKafka.TCP(brokerAddress),
		Topic:    TopicDriverSync,
		Balancer: &goKafka.LeastBytes{},
	}
	slog.Info("sync emitter kafka writer armed", "topic", TopicDriverSync)
}

// EmitOrderSyncEvent fires an immutable, append-only event to the driver sync
// topic. On failure, the caller (HandleBatchSync) unlocks the Redis dedup key
// so the driver can retry on the next sync pulse.
func EmitOrderSyncEvent(event OrderSyncEvent) error {
	if syncWriter == nil {
		return fmt.Errorf("sync writer not initialised: call kafka.InitSyncWriter first")
	}

	payload, err := fastjson.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal OrderSyncEvent: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return syncWriter.WriteMessages(ctx, goKafka.Message{
		Key:   []byte(EventOrderSync),
		Value: payload,
	})
}

// PredictionCorrectedEvent is emitted when a retailer edits an AI_PLANNED order,
// feeding the Empathy Engine's RLHF loop.
type PredictionCorrectedEvent struct {
	PredictionID string `json:"prediction_id"`
	RetailerID   string `json:"retailer_id"`
	WarehouseId  string `json:"warehouse_id,omitempty"`
	FieldChanged string `json:"field_changed"` // "amount" | "trigger_date" | "rejected"
	OldValue     string `json:"old_value"`
	NewValue     string `json:"new_value"`
	Timestamp    int64  `json:"timestamp"`
}

// correctionWriter is the Kafka writer for the AI feedback topic.
var correctionWriter *goKafka.Writer

// InitCorrectionWriter must be called from main() alongside InitSyncWriter.
func InitCorrectionWriter(brokerAddress string) {
	correctionWriter = &goKafka.Writer{
		Addr:     goKafka.TCP(brokerAddress),
		Topic:    TopicMain,
		Balancer: &goKafka.LeastBytes{},
	}
	slog.Info("ai feedback kafka writer armed", "topic", TopicMain)
}

// EmitPredictionCorrected fires an RLHF correction event to the logistics topic.
func EmitPredictionCorrected(event PredictionCorrectedEvent) error {
	if correctionWriter == nil {
		return fmt.Errorf("correction writer not initialised: call kafka.InitCorrectionWriter first")
	}

	payload, err := fastjson.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal PredictionCorrectedEvent: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return correctionWriter.WriteMessages(ctx, goKafka.Message{
		Key:   []byte(EventAiPredictionCorrected),
		Value: payload,
	})
}

// ── Granular RLHF Events (Phase 12) ────────────────────────────────────────

// AIPlanDateShiftEvent is emitted when a retailer shifts the suggested delivery
// date on an AI_PLANNED prediction. One event per date change.
type AIPlanDateShiftEvent struct {
	PredictionID string `json:"prediction_id"`
	RetailerID   string `json:"retailer_id"`
	WarehouseId  string `json:"warehouse_id,omitempty"`
	OldDate      string `json:"old_date"` // RFC3339 or ""
	NewDate      string `json:"new_date"` // RFC3339
	Timestamp    int64  `json:"timestamp"`
}

// AIPlanSkuModifiedEvent is emitted per-SKU when a retailer edits quantity or
// rejects a predicted line item. The Empathy Engine consumes these for
// item-level RLHF gradient updates.
type AIPlanSkuModifiedEvent struct {
	PredictionID string `json:"prediction_id"`
	RetailerID   string `json:"retailer_id"`
	WarehouseId  string `json:"warehouse_id,omitempty"`
	SkuID        string `json:"sku_id"`
	Field        string `json:"field"` // "amount" | "rejected"
	OldValue     string `json:"old_value"`
	NewValue     string `json:"new_value"`
	Timestamp    int64  `json:"timestamp"`
}

// EmitDateShift fires a granular AI_PLAN_DATE_SHIFT event.
func EmitDateShift(event AIPlanDateShiftEvent) error {
	if correctionWriter == nil {
		return fmt.Errorf("correction writer not initialised: call kafka.InitCorrectionWriter first")
	}
	payload, err := fastjson.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal AIPlanDateShiftEvent: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return correctionWriter.WriteMessages(ctx, goKafka.Message{
		Key:   []byte(EventAiPlanDateShift),
		Value: payload,
	})
}

// EmitSkuModified fires a granular AI_PLAN_SKU_MODIFIED event per SKU delta.
func EmitSkuModified(event AIPlanSkuModifiedEvent) error {
	if correctionWriter == nil {
		return fmt.Errorf("correction writer not initialised: call kafka.InitCorrectionWriter first")
	}
	payload, err := fastjson.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal AIPlanSkuModifiedEvent: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return correctionWriter.WriteMessages(ctx, goKafka.Message{
		Key:   []byte(EventAiPlanSkuModified),
		Value: payload,
	})
}

// ── Notification Fan-Out Writer ───────────────────────────────────────────
// notificationWriter publishes best-effort notification events to TopicMain
// keyed by EventType so the notification_dispatcher consumer can route them.
// This runs in parallel to the transactional outbox: the outbox remains the
// source of truth for durable state transitions (AggregateID-keyed, ordered
// per entity); this writer is a UX-tier fan-out for push/WS/Telegram.
// Fire-and-forget — errors are logged, not returned.
var notificationWriter *goKafka.Writer

// InitNotificationWriter must be called from bootstrap alongside InitSyncWriter.
func InitNotificationWriter(brokerAddress string) {
	notificationWriter = &goKafka.Writer{
		Addr:     goKafka.TCP(brokerAddress),
		Topic:    TopicMain,
		Balancer: &goKafka.LeastBytes{},
	}
	slog.Info("notification kafka writer armed", "topic", TopicMain)
}

// EmitNotification publishes eventType-keyed payload to TopicMain. Intended to
// be called after a successful Spanner commit by handlers that need real-time
// notification fan-out. Safe to call before initialisation — logs and returns.
func EmitNotification(eventType string, payload any) {
	if notificationWriter == nil {
		slog.Warn("notification kafka writer not initialised; skipping event", "event_type", eventType)
		return
	}
	body, err := fastjson.Marshal(payload)
	if err != nil {
		slog.Error("notification event marshal failed", "event_type", eventType, "error", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := notificationWriter.WriteMessages(ctx, goKafka.Message{
		Key:   []byte(eventType),
		Value: body,
	}); err != nil {
		slog.Error("notification event publish failed", "event_type", eventType, "error", err)
	}
}

// CloseWriters gracefully shuts down all singleton Kafka writers.
// Call from main() on SIGTERM / SIGINT before process exit.
func CloseWriters() {
	if syncWriter != nil {
		if err := syncWriter.Close(); err != nil {
			slog.Error("sync writer close failed", "error", err)
		} else {
			slog.Info("sync writer closed")
		}
	}
	if correctionWriter != nil {
		if err := correctionWriter.Close(); err != nil {
			slog.Error("correction writer close failed", "error", err)
		} else {
			slog.Info("correction writer closed")
		}
	}
	if notificationWriter != nil {
		if err := notificationWriter.Close(); err != nil {
			slog.Error("notification writer close failed", "error", err)
		} else {
			slog.Info("notification writer closed")
		}
	}
	if dlqWriter != nil {
		if err := dlqWriter.Close(); err != nil {
			slog.Error("dlq writer close failed", "error", err)
		} else {
			slog.Info("dlq writer closed")
		}
	}
}
