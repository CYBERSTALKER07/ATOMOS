package planning

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/segmentio/kafka-go"
)

// SignalIngestInput is the validated body for planning signal ingestion.
type SignalIngestInput struct {
	SignalID   string          `json:"signal_id"`
	Source     string          `json:"source"`
	WarehouseID string         `json:"warehouse_id,omitempty"`
	RetailerID string          `json:"retailer_id,omitempty"`
	Payload    json.RawMessage `json:"payload"`
}

// SignalIngestPublisher publishes planning ingest events to Kafka.
type SignalIngestPublisher interface {
	Publish(ctx context.Context, topic string, key []byte, value []byte) error
}

type kafkaSignalPublisher struct {
	writer *kafka.Writer
}

// NewKafkaSignalPublisher builds a publisher for planning ingest topics.
func NewKafkaSignalPublisher(brokers []string, topic string) SignalIngestPublisher {
	if len(brokers) == 0 || topic == "" {
		return nil
	}
	return &kafkaSignalPublisher{writer: &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}}
}

func (p *kafkaSignalPublisher) Publish(ctx context.Context, topic string, key, value []byte) error {
	if p == nil || p.writer == nil {
		return errors.New("publisher unavailable")
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Key: key, Value: value, Time: time.Now().UTC()})
}

// IngestSignal validates and publishes planning.signal.ingest.v1 without hot-path Spanner write.
func IngestSignal(ctx context.Context, pub SignalIngestPublisher, supplierID string, in SignalIngestInput) (string, error) {
	if pub == nil {
		return "", errors.New("ingest publisher unavailable")
	}
	signalID := strings.TrimSpace(in.SignalID)
	if signalID == "" {
		signalID = uuid.NewString()
	}
	source := strings.TrimSpace(in.Source)
	if source == "" {
		return "", errors.New("source_required")
	}
	envelope := map[string]any{
		"type":        events.EventPlanningSignalIngest,
		"supplier_id": supplierID,
		"signal_id":   signalID,
		"source":      source,
		"warehouse_id": strings.TrimSpace(in.WarehouseID),
		"retailer_id": strings.TrimSpace(in.RetailerID),
		"payload":     json.RawMessage(in.Payload),
		"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
		"v":           1,
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return "", err
	}
	topic := events.TopicPlanningSignalIngest
	if err := pub.Publish(ctx, topic, []byte(supplierID+":"+signalID), raw); err != nil {
		return "", err
	}
	return signalID, nil
}

// DecodeSignalIngest parses an ingest envelope from Kafka.
func DecodeSignalIngest(value []byte) (SignalIngestInput, string, error) {
	var envelope struct {
		Type       string          `json:"type"`
		SupplierID string          `json:"supplier_id"`
		SignalID   string          `json:"signal_id"`
		Source     string          `json:"source"`
		WarehouseID string         `json:"warehouse_id"`
		RetailerID string          `json:"retailer_id"`
		Payload    json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(value, &envelope); err != nil {
		return SignalIngestInput{}, "", err
	}
	return SignalIngestInput{
		SignalID:    envelope.SignalID,
		Source:      envelope.Source,
		WarehouseID: envelope.WarehouseID,
		RetailerID:  envelope.RetailerID,
		Payload:     envelope.Payload,
	}, envelope.SupplierID, nil
}

// IngestTopic returns the Kafka topic for planning signal ingest.
func IngestTopic() string {
	return events.TopicPlanningSignalIngest
}

// ReadSeasonalOverrideBody parses seasonal override POST body.
func ReadSeasonalOverrideBody(r io.Reader) (SeasonalOverrideInput, error) {
	raw, err := io.ReadAll(io.LimitReader(r, 8*1024))
	if err != nil {
		return SeasonalOverrideInput{}, err
	}
	var in SeasonalOverrideInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return SeasonalOverrideInput{}, errors.New("invalid_body")
	}
	return in, nil
}

// ReadSignalIngestBody limits and parses HTTP request body.
func ReadSignalIngestBody(r io.Reader) (SignalIngestInput, error) {
	raw, err := io.ReadAll(io.LimitReader(r, 64*1024))
	if err != nil {
		return SignalIngestInput{}, err
	}
	var in SignalIngestInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return SignalIngestInput{}, errors.New("invalid_body")
	}
	return in, nil
}
