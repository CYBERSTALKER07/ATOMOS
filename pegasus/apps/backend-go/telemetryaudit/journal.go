package telemetryaudit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"backend-go/fastjson"
	"backend-go/telemetry"

	goKafka "github.com/segmentio/kafka-go"
)

const (
	// TopicRaw is the additive Kafka journal for live telemetry ingress.
	TopicRaw = "pegasus-telemetry-raw"

	journalQueueDepth = 1024
)

var errJournalBackpressure = errors.New("telemetry audit journal queue full")

// Journal asynchronously writes telemetry audit events to Kafka so the
// WebSocket ingress path remains non-blocking.
type Journal struct {
	writer *goKafka.Writer
	queue  chan telemetry.AuditEvent
	done   chan struct{}
	once   sync.Once
}

// NewJournal arms the background writer loop for telemetry audit events.
func NewJournal(brokerAddress string) *Journal {
	j := &Journal{
		writer: &goKafka.Writer{
			Addr:         goKafka.TCP(brokerAddress),
			Topic:        TopicRaw,
			Balancer:     &goKafka.LeastBytes{},
			BatchTimeout: 50 * time.Millisecond,
			RequiredAcks: goKafka.RequireOne,
			MaxAttempts:  5,
		},
		queue: make(chan telemetry.AuditEvent, journalQueueDepth),
		done:  make(chan struct{}),
	}
	go j.run()
	slog.Info("telemetry audit journal armed", "topic", TopicRaw, "queue_depth", journalQueueDepth)
	return j
}

// Emit enqueues an audit event without blocking the telemetry ingress path.
func (j *Journal) Emit(ctx context.Context, event telemetry.AuditEvent) error {
	if j == nil {
		return nil
	}
	select {
	case j.queue <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return errJournalBackpressure
	}
}

// Close drains the queue and closes the underlying Kafka writer.
func (j *Journal) Close() error {
	if j == nil {
		return nil
	}
	j.once.Do(func() {
		close(j.queue)
		<-j.done
	})
	if err := j.writer.Close(); err != nil {
		return fmt.Errorf("close telemetry audit writer: %w", err)
	}
	return nil
}

func (j *Journal) run() {
	defer close(j.done)
	for event := range j.queue {
		body, err := fastjson.Marshal(event)
		if err != nil {
			slog.Error("telemetry audit marshal failed",
				"trace_id", event.TraceID,
				"driver_id", event.DriverID,
				"error", err,
			)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = j.writer.WriteMessages(ctx, goKafka.Message{
			Key:   []byte(event.DriverID),
			Value: body,
			Headers: []goKafka.Header{
				{Key: "trace_id", Value: []byte(event.TraceID)},
				{Key: "supplier_id", Value: []byte(event.SupplierID)},
			},
		})
		cancel()
		if err != nil {
			slog.Error("telemetry audit publish failed",
				"trace_id", event.TraceID,
				"driver_id", event.DriverID,
				"supplier_id", event.SupplierID,
				"topic", TopicRaw,
				"error", err,
			)
		}
	}
}
