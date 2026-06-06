package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaPublisherConfig tunes writer behavior for outbox delivery.
type KafkaPublisherConfig struct {
	BatchTimeout time.Duration
	MaxAttempts  int
}

func (c *KafkaPublisherConfig) applyDefaults() {
	if c.BatchTimeout <= 0 {
		c.BatchTimeout = 250 * time.Millisecond
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 5
	}
}

// KafkaPublisher publishes outbox events to Kafka with required acks.
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisherFromCSV builds a Kafka publisher from a comma-separated
// broker list.
func NewKafkaPublisherFromCSV(brokersCSV string, cfg KafkaPublisherConfig) (*KafkaPublisher, error) {
	parts := strings.Split(brokersCSV, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			brokers = append(brokers, trimmed)
		}
	}
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka publisher: at least one broker required")
	}
	cfg.applyDefaults()
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		RequiredAcks: kafka.RequireAll,
		BatchTimeout: cfg.BatchTimeout,
		MaxAttempts:  cfg.MaxAttempts,
		Balancer:     &kafka.Hash{},
		Async:        false,
	}
	return &KafkaPublisher{writer: writer}, nil
}

// Publish writes a single message to a topic. Key should be aggregate root id
// bytes to preserve per-entity ordering.
func (p *KafkaPublisher) Publish(ctx context.Context, topic string, key []byte, value []byte) error {
	if p == nil || p.writer == nil {
		return fmt.Errorf("kafka publisher: nil writer")
	}
	if strings.TrimSpace(topic) == "" {
		return fmt.Errorf("kafka publisher: topic required")
	}
	msg := kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
		Time:  time.Now().UTC(),
	}
	if traceID := traceIDFromPayload(value); traceID != "" {
		msg.Headers = []kafka.Header{{Key: "trace_id", Value: []byte(traceID)}}
	}
	return p.writer.WriteMessages(ctx, msg)
}

func traceIDFromPayload(value []byte) string {
	var envelope struct {
		TraceID string `json:"trace_id"`
	}
	if err := json.Unmarshal(value, &envelope); err != nil {
		return ""
	}
	return strings.TrimSpace(envelope.TraceID)
}

// Close releases writer resources.
func (p *KafkaPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
