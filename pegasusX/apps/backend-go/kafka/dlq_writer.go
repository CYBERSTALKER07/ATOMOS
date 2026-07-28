package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	segmentkafka "github.com/segmentio/kafka-go"
)

type kafkaDLQWriter struct {
	writer *segmentkafka.Writer
}

func NewDLQWriterFromCSV(brokersCSV, topic string) (DLQWriter, error) {
	return NewDLQWriterFromCSVWithAuth(brokersCSV, topic, kafkautil.ClientAuth{})
}

// NewDLQWriterFromCSVWithAuth builds a DLQ writer with optional GCP Managed Kafka auth.
func NewDLQWriterFromCSVWithAuth(brokersCSV, topic string, auth kafkautil.ClientAuth) (DLQWriter, error) {
	brokers := kafkautil.SplitBrokers(brokersCSV)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka dlq writer: at least one broker required")
	}
	trimmedTopic := strings.TrimSpace(topic)
	if trimmedTopic == "" {
		return nil, fmt.Errorf("kafka dlq writer: topic required")
	}
	transport, err := kafkautil.Transport(auth)
	if err != nil {
		return nil, fmt.Errorf("kafka dlq writer: transport: %w", err)
	}
	return &kafkaDLQWriter{
		writer: &segmentkafka.Writer{
			Addr:                   segmentkafka.TCP(brokers...),
			Topic:                  trimmedTopic,
			RequiredAcks:           segmentkafka.RequireAll,
			BatchTimeout:           250 * time.Millisecond,
			MaxAttempts:            1 << 20, // high bound; WriteTimeout caps wait
			WriteTimeout:           30 * time.Second,
			ReadTimeout:            10 * time.Second,
			Balancer:               &segmentkafka.Hash{},
			Async:                  false,
			AllowAutoTopicCreation: false,
			Transport:              transport,
		},
	}, nil
}

func (w *kafkaDLQWriter) WriteMessages(ctx context.Context, msgs ...segmentkafka.Message) error {
	if w == nil || w.writer == nil {
		return fmt.Errorf("kafka dlq writer: nil writer")
	}
	return w.writer.WriteMessages(ctx, msgs...)
}

func (w *kafkaDLQWriter) Close() error {
	if w == nil || w.writer == nil {
		return nil
	}
	return w.writer.Close()
}
