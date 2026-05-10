package telemetryaudit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"backend-go/fastjson"
	internalKafka "backend-go/kafka"
	"backend-go/telemetry"

	goKafka "github.com/segmentio/kafka-go"
)

const (
	journalQueueDepth = 1024
)

var errJournalBackpressure = errors.New("telemetry audit journal queue full")
var errJournalClosed = errors.New("telemetry audit journal closed")

// Journal asynchronously writes telemetry audit events to Kafka so the
// WebSocket ingress path remains non-blocking.
type Journal struct {
	writer *goKafka.Writer
	queue  chan telemetry.AuditEvent
	done   chan struct{}
	once   sync.Once
	mu     sync.RWMutex
	closed bool
	err    error
}

// NewJournal arms the background writer loop for telemetry audit events.
func NewJournal(brokerAddress string) *Journal {
	j := &Journal{
		writer: &goKafka.Writer{
			Addr:         goKafka.TCP(brokerAddress),
			Topic:        internalKafka.TopicTelemetryRaw,
			Balancer:     &goKafka.LeastBytes{},
			BatchTimeout: 50 * time.Millisecond,
			RequiredAcks: goKafka.RequireOne,
			MaxAttempts:  5,
		},
		queue: make(chan telemetry.AuditEvent, journalQueueDepth),
		done:  make(chan struct{}),
	}
	go j.run()
	slog.Info("telemetry audit journal armed", "topic", internalKafka.TopicTelemetryRaw, "queue_depth", journalQueueDepth)
	return j
}

// Emit enqueues an audit event without blocking the telemetry ingress path.
func (j *Journal) Emit(ctx context.Context, event telemetry.AuditEvent) error {
	if j == nil {
		return nil
	}
	j.mu.RLock()
	if j.closed {
		j.mu.RUnlock()
		return errJournalClosed
	}
	select {
	case j.queue <- event:
		j.mu.RUnlock()
		return nil
	case <-ctx.Done():
		j.mu.RUnlock()
		return ctx.Err()
	default:
		j.mu.RUnlock()
		return errJournalBackpressure
	}
}

// Close drains the queue and closes the underlying Kafka writer.
func (j *Journal) Close() error {
	if j == nil {
		return nil
	}
	j.once.Do(func() {
		j.mu.Lock()
		j.closed = true
		close(j.queue)
		j.mu.Unlock()
		<-j.done
		if err := j.writer.Close(); err != nil {
			j.err = fmt.Errorf("close telemetry audit writer: %w", err)
		}
	})
	return j.err
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
				"topic", internalKafka.TopicTelemetryRaw,
				"error", err,
			)
		}
	}
}
