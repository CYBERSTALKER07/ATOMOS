package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"backend-go/cache"

	"github.com/segmentio/kafka-go"
)

var dlqWriter *kafka.Writer

// InitDLQ boots the failsafe Dead Letter Queue transmitter.
func InitDLQ(brokerAddress string) {
	dlqWriter = &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Topic:    "lab-logistics-events-dlq",
		Balancer: &kafka.LeastBytes{},
	}
	log.Println("[DLQ SYSTEM] Dead Letter Queue Transmitter Online.")
}

// RouteToDLQ catches dropped transactions and permanently stores them.
func RouteToDLQ(event LogisticsEvent, failReason string) {
	if dlqWriter == nil {
		log.Println("[CRITICAL FAULT] DLQ Writer not initialized. Money lost!")
		return
	}

	// Attach the reason so Admin can debug in the Next.js Portal
	payload, _ := json.Marshal(map[string]interface{}{
		"failed_at":  time.Now().Format(time.RFC3339),
		"reason":     failReason,
		"event_data": event,
	})

	dlqCtx, dlqCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dlqCancel()
	err := dlqWriter.WriteMessages(dlqCtx,
		kafka.Message{
			Key:   []byte(event.OrderId),
			Value: payload,
		},
	)

	if err != nil {
		log.Printf("[CRITICAL FAULT] Failed to write to DLQ for Order %s: %v", event.OrderId, err)
	} else {
		log.Printf("[DLQ SYSTEM] Event for %s safely secured in Dead Letter Queue.", event.OrderId)
	}
}

// DLQMessage is the wire format for a single DLQ entry read back from Kafka.
type DLQMessage struct {
	Offset    int64           `json:"offset"`
	Key       string          `json:"key"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// ListDLQMessages opens a short-lived consumer window and returns up to
// maxMessages entries from the Dead Letter Queue topic.
// It times out after readTimeout if the topic has fewer messages than cap.
func ListDLQMessages(brokerAddress string, maxMessages int, offset int) ([]DLQMessage, error) {
	if maxMessages <= 0 {
		maxMessages = 50
	}
	if maxMessages > 500 {
		maxMessages = 500
	}
	if offset < 0 {
		offset = 0
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{brokerAddress},
		Topic:     "lab-logistics-events-dlq",
		GroupID:   "", // No consumer group — read from offset 0 every call (observer pattern)
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  10 << 20, // 10 MB
		MaxWait:   500 * time.Millisecond,
	})
	// Always seek to the beginning so Admins see ALL trapped events
	if err := reader.SetOffset(kafka.FirstOffset); err != nil {
		reader.Close()
		return nil, fmt.Errorf("DLQ seek error: %w", err)
	}
	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Batch-fetch the Resolution Ledger BEFORE reading Kafka — single round-trip.
	// Any offset present in this set was already successfully replayed; hide it.
	resolvedOffsets := cache.ResolvedOffsets(ctx)

	maxRead := offset + maxMessages
	var rawMessages []DLQMessage
	for len(rawMessages) < maxRead {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			// Timeout or EOF — stop gracefully
			break
		}
		rawMessages = append(rawMessages, DLQMessage{
			Offset:    msg.Offset,
			Key:       string(msg.Key),
			Timestamp: msg.Time,
			Payload:   json.RawMessage(msg.Value),
		})
	}

	// Filter: remove any offset that has been marked as resolved in Redis.
	var filteredMessages []DLQMessage
	for _, msg := range rawMessages {
		offsetStr := fmt.Sprintf("%d", msg.Offset)
		if !resolvedOffsets[offsetStr] {
			filteredMessages = append(filteredMessages, msg)
		}
	}

	if offset >= len(filteredMessages) {
		return []DLQMessage{}, nil
	}

	end := offset + maxMessages
	if end > len(filteredMessages) {
		end = len(filteredMessages)
	}
	paged := filteredMessages[offset:end]

	log.Printf("[DLQ SYSTEM] Inspector: %d total, %d resolved (hidden), %d active, returning %d from offset %d.",
		len(rawMessages), len(rawMessages)-len(filteredMessages), len(filteredMessages), len(paged), offset)
	return paged, nil
}

// ReplayDLQMessage reads the payload at a specific offset from the DLQ and
// re-emits it onto the main 'lab-logistics-events' topic so the payment
// reconciliation consumer can reprocess it without data loss.
func ReplayDLQMessage(brokerAddress string, offset int64) error {
	// 1. Open a partition reader exactly at the target offset
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{brokerAddress},
		Topic:     "lab-logistics-events-dlq",
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  10 << 20,
		MaxWait:   1 * time.Second,
	})
	if err := reader.SetOffset(offset); err != nil {
		reader.Close()
		return fmt.Errorf("DLQ replay seek error: %w", err)
	}
	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		return fmt.Errorf("DLQ replay read error: %w", err)
	}

	// 2. Extract the original event_data field from the DLQ envelope
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		return fmt.Errorf("DLQ envelope parse error: %w", err)
	}

	originalPayload, ok := envelope["event_data"]
	if !ok {
		return fmt.Errorf("DLQ message at offset %d has no event_data field", offset)
	}

	// 3. Re-emit onto the live reconciliation topic
	mainWriter := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Topic:    "lab-logistics-events",
		Balancer: &kafka.LeastBytes{},
	}
	defer mainWriter.Close()

	writeCtx, writeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer writeCancel()

	err = mainWriter.WriteMessages(writeCtx, kafka.Message{
		Key:   msg.Key,
		Value: originalPayload,
	})
	if err != nil {
		return fmt.Errorf("DLQ replay re-emit error: %w", err)
	}

	// ── Resolution Ledger: mark this offset as permanently replayed ──────────
	// CRITICAL: we write to Redis ONLY after the Kafka emit succeeds.
	// If SAdd fails we return an error — but the event IS on the main topic.
	// The operator will see the offset next poll; a second replay attempt will
	// be a harmless duplicate FROM Kafka's perspective, NOT a financial issue,
	// because the payment-reconciliation consumer is idempotent on OrderId.
	// The Redis failure is still surfaced so ops can investigate the cache issue.
	markCtx, markCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer markCancel()
	if redisErr := cache.MarkResolved(markCtx, offset); redisErr != nil {
		log.Printf("[DLQ SYSTEM] WARNING: Replay succeeded but Resolution Ledger write failed for offset %d: %v", offset, redisErr)
		return fmt.Errorf("replay succeeded but failed to mark offset %d as resolved: %w", offset, redisErr)
	}

	log.Printf("[DLQ SYSTEM] Replayed message at offset %d (key=%s) → main topic. Resolution Ledger updated.", offset, string(msg.Key))
	return nil
}
