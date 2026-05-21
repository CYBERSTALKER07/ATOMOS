package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	consumerLag = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "void",
		Subsystem: "kafka",
		Name:      "consumer_lag_seconds",
		Help:      "Kafka consumer lag in seconds",
	}, []string{"topic", "partition"})

	consumerErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "kafka",
		Name:      "consumer_errors_total",
		Help:      "Total number of consumer errors",
	}, []string{"topic", "partition"})
)

type EventHandler func(ctx context.Context, msg kafka.Message) error

type ConsumerDeps struct {
	Brokers []string
	GroupID string
	Topic   string
	Handler EventHandler
	DLQWriter *kafka.Writer
	MaxAttempts int
}

type Consumer struct {
	reader *kafka.Reader
	deps   ConsumerDeps
}

func NewConsumer(deps ConsumerDeps) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  deps.Brokers,
		GroupID:  deps.GroupID,
		Topic:    deps.Topic,
		MaxBytes: 10e6, // 10MB
		CommitInterval: time.Second, // Flush commits to Kafka every second
	})

	if deps.MaxAttempts <= 0 {
		deps.MaxAttempts = 3
	}

	return &Consumer{
		reader: r,
		deps:   deps,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	slog.Info("kafka consumer starting", "topic", c.deps.Topic, "group_id", c.deps.GroupID)

	// Target one goroutine per partition, bounded by GOMAXPROCS.
	// Since we are reading from the generic reader, segmentio load balances.
	// We'll spawn GOMAXPROCS workers to process messages.
	numWorkers := runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			c.workerLoop(ctx, workerID)
		}(i)
	}

	wg.Wait()
	slog.Info("kafka consumer stopped", "topic", c.deps.Topic, "group_id", c.deps.GroupID)
}

func (c *Consumer) workerLoop(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			slog.Error("kafka fetch error", "err", err, "topic", c.deps.Topic, "worker_id", workerID)
			time.Sleep(1 * time.Second)
			continue
		}

		lag := time.Since(m.Time).Seconds()
		consumerLag.WithLabelValues(c.deps.Topic, fmt.Sprintf("%d", m.Partition)).Set(lag)

		// Process message with retries
		err = c.processWithRetries(ctx, m)
		if err != nil {
			slog.Error("kafka message processing failed, sending to DLQ",
				"err", err, "topic", c.deps.Topic, "partition", m.Partition, "offset", m.Offset)
			c.sendToDLQ(ctx, m, err)
		}

		// Commit the message after processing (or DLQ routing)
		if err := c.reader.CommitMessages(ctx, m); err != nil {
			slog.Error("kafka commit error", "err", err, "topic", c.deps.Topic, "partition", m.Partition)
		}
	}
}

func (c *Consumer) processWithRetries(ctx context.Context, m kafka.Message) error {
	var err error
	for attempt := 1; attempt <= c.deps.MaxAttempts; attempt++ {
		err = c.deps.Handler(ctx, m)
		if err == nil {
			return nil
		}
		
		slog.Warn("kafka handler error", 
			"err", err, "attempt", attempt, "max", c.deps.MaxAttempts,
			"topic", c.deps.Topic, "partition", m.Partition)

		if attempt < c.deps.MaxAttempts {
			// Exponential backoff with jitter
			backoff := time.Duration(1<<attempt) * 100 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	consumerErrors.WithLabelValues(c.deps.Topic, fmt.Sprintf("%d", m.Partition)).Inc()
	return err
}

func (c *Consumer) sendToDLQ(ctx context.Context, m kafka.Message, reason error) {
	if c.deps.DLQWriter == nil {
		slog.Warn("no DLQ configured, message dropped", "topic", c.deps.Topic)
		return
	}

	dlqMsg := kafka.Message{
		Key:   m.Key,
		Value: m.Value,
		Headers: append(m.Headers, kafka.Header{
			Key:   "dlq_reason",
			Value: []byte(reason.Error()),
		}),
	}

	if err := c.deps.DLQWriter.WriteMessages(ctx, dlqMsg); err != nil {
		slog.Error("failed to write to DLQ", "err", err, "topic", c.deps.Topic)
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
