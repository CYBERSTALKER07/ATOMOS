package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/segmentio/kafka-go"
)

// Default publisher reliability knobs.
// MaxAttempts is intentionally very high; WriteTimeout bounds total wait on
// transient broker/network failures (kafka-go retries within that envelope).
const (
	defaultKafkaPublishMaxAttempts = math.MaxInt32
	defaultKafkaPublishWriteTimeout = 30 * time.Second
	defaultKafkaPublishReadTimeout  = 10 * time.Second
	defaultKafkaPublishBatchTimeout = 250 * time.Millisecond
)

// KafkaPublisherConfig tunes writer behavior for outbox delivery.
type KafkaPublisherConfig struct {
	BatchTimeout time.Duration
	MaxAttempts  int
	WriteTimeout time.Duration
	ReadTimeout  time.Duration
	// Auth: empty = local plaintext; GCP_MANAGED_OAUTH = Managed Kafka SASL_SSL.
	Auth kafkautil.ClientAuth
}

func (c *KafkaPublisherConfig) applyDefaults() {
	if c.BatchTimeout <= 0 {
		c.BatchTimeout = defaultKafkaPublishBatchTimeout
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = defaultKafkaPublishMaxAttempts
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = defaultKafkaPublishWriteTimeout
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = defaultKafkaPublishReadTimeout
	}
}

// KafkaPublisher publishes outbox events to Kafka with required acks.
//
// Reliability model (at-least-once, outbox-backed):
//   - RequiredAcks=all (broker ISR ack)
//   - High MaxAttempts + WriteTimeout (retry until delivery window expires)
//   - Hash balancer on aggregate key (per-entity order)
//   - Sync writes (Async=false)
//   - AllowAutoTopicCreation=false (topics owned by Strimzi CRDs)
//
// Note: segmentio/kafka-go Writer does not expose enable.idempotence. Duplicate
// prevention for the outbox path is handled by Relay MarkPublished + consumer
// event dedup middleware. True broker-side idempotent producers require a
// transactional client; do not weaken RequireAll / sync writes.
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisherFromCSV builds a Kafka publisher from a comma-separated
// broker list.
func NewKafkaPublisherFromCSV(brokersCSV string, cfg KafkaPublisherConfig) (*KafkaPublisher, error) {
	brokers := kafkautil.SplitBrokers(brokersCSV)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka publisher: at least one broker required")
	}
	cfg.applyDefaults()
	transport, err := kafkautil.Transport(cfg.Auth)
	if err != nil {
		return nil, fmt.Errorf("kafka publisher: transport: %w", err)
	}
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		RequiredAcks:           kafka.RequireAll,
		BatchTimeout:           cfg.BatchTimeout,
		MaxAttempts:            cfg.MaxAttempts,
		WriteTimeout:           cfg.WriteTimeout,
		ReadTimeout:            cfg.ReadTimeout,
		Balancer:               &kafka.Hash{},
		Async:                  false,
		AllowAutoTopicCreation: false,
		Transport:              transport,
	}
	return &KafkaPublisher{writer: writer}, nil
}

// Publish writes a single message to a topic. Key should be aggregate root id
// bytes to preserve per-entity ordering.
func (p *KafkaPublisher) Publish(ctx context.Context, topic string, key []byte, value []byte) error {
	return p.PublishWithHeaders(ctx, topic, key, value, nil)
}

// PublishWithHeaders writes with optional Kafka headers (event_id for consumer dedupe).
func (p *KafkaPublisher) PublishWithHeaders(ctx context.Context, topic string, key []byte, value []byte, headers map[string][]byte) error {
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
	var hdrs []kafka.Header
	if traceID := traceIDFromPayload(value); traceID != "" {
		hdrs = append(hdrs, kafka.Header{Key: "trace_id", Value: []byte(traceID)})
	}
	for k, v := range headers {
		if strings.TrimSpace(k) == "" || len(v) == 0 {
			continue
		}
		hdrs = append(hdrs, kafka.Header{Key: k, Value: v})
	}
	msg.Headers = hdrs
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
