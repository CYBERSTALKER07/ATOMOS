package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	segmentkafka "github.com/segmentio/kafka-go"
)

type kafkaDLQWriter struct {
	writer *segmentkafka.Writer
}

func NewDLQWriterFromCSV(brokersCSV, topic string) (DLQWriter, error) {
	brokers := splitAndTrimBrokers(brokersCSV)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka dlq writer: at least one broker required")
	}
	trimmedTopic := strings.TrimSpace(topic)
	if trimmedTopic == "" {
		return nil, fmt.Errorf("kafka dlq writer: topic required")
	}
	return &kafkaDLQWriter{
		writer: &segmentkafka.Writer{
			Addr:         segmentkafka.TCP(brokers...),
			Topic:        trimmedTopic,
			RequiredAcks: segmentkafka.RequireAll,
			BatchTimeout: 250 * time.Millisecond,
			MaxAttempts:  5,
			Balancer:     &segmentkafka.Hash{},
			Async:        false,
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

func splitAndTrimBrokers(brokersCSV string) []string {
	parts := strings.Split(brokersCSV, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		brokers = append(brokers, trimmed)
	}
	return brokers
}
