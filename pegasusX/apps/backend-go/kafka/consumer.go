package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"runtime"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/kafka/workerpool"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/segmentio/kafka-go"
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

	consumerPoisonSkipped = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "kafka",
		Name:      "consumer_poison_skipped_total",
		Help:      "Malformed envelopes skipped without DLQ (empty type only)",
	}, []string{"topic"})
)

type EventHandler func(ctx context.Context, msg kafka.Message) error

type DLQWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type ConsumerDeps struct {
	Brokers     []string
	GroupID     string
	Topic       string
	Topics      []string
	Handler     EventHandler
	DLQWriter   DLQWriter
	MaxAttempts int
	// Auth: empty = local plaintext; GCP_MANAGED_OAUTH for Managed Kafka.
	Auth kafkautil.ClientAuth
}

type Consumer struct {
	reader consumerReader
	deps   ConsumerDeps
	pool   *workerpool.Pool
}

type consumerReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

func NewConsumer(deps ConsumerDeps) *Consumer {
	if deps.MaxAttempts <= 0 {
		deps.MaxAttempts = 3
	}
	dialer, err := kafkautil.Dialer(deps.Auth)
	if err != nil {
		// Fall back to plaintext dialer so local tests still construct; Start will fail on auth errors.
		dialer = &kafka.Dialer{Timeout: 15 * time.Second, DualStack: true}
		slog.Error("kafka consumer dialer", "err", err)
	}
	// CommitInterval=0 → CommitMessages is synchronous. workerpool commits only
	// after handler success or successful DLQ routing (see dispatch + ErrSkipCommit).
	// Never use ReadMessage auto-commit path; FetchMessage + manual commit only.
	reader := kafka.NewReader(readerConfig(deps, dialer))
	return &Consumer{
		reader: reader,
		deps:   deps,
	}
}

// readerConfig builds a group consumer that only advances offsets via CommitMessages.
func readerConfig(deps ConsumerDeps, dialer *kafka.Dialer) kafka.ReaderConfig {
	return kafka.ReaderConfig{
		Brokers:               deps.Brokers,
		GroupID:               deps.GroupID,
		Topic:                 deps.Topic,
		MaxBytes:              10e6,
		CommitInterval:        0, // sync commit after success/DLQ
		WatchPartitionChanges: true,
		Dialer:                dialer,
	}
}

// Start runs the partition-parallel consumer until ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) {
	slog.Info("kafka consumer starting",
		"topic", c.deps.Topic,
		"group_id", c.deps.GroupID,
		"workers", runtime.GOMAXPROCS(0),
	)

	pool, err := workerpool.New(workerpool.Config{
		Source:  c.reader,
		Handler: c.dispatch,
		Name:    c.deps.Topic,
		Logger:  slog.Default(),
		OnFailure: func(ctx context.Context, m kafka.Message, handlerErr error) {
			slog.ErrorContext(ctx, "kafka handler exhausted retries",
				"err", handlerErr,
				"topic", c.deps.Topic,
				"partition", m.Partition,
				"offset", m.Offset,
				"trace_id", TraceIDFromMessage(m),
			)
		},
	})
	if err != nil {
		slog.Error("kafka consumer pool init failed", "topic", c.deps.Topic, "err", err)
		return
	}
	c.pool = pool
	if runErr := pool.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
		slog.Error("kafka consumer stopped with error", "topic", c.deps.Topic, "err", runErr)
	}
	slog.Info("kafka consumer stopped", "topic", c.deps.Topic, "group_id", c.deps.GroupID)
}

func (c *Consumer) dispatch(ctx context.Context, m kafka.Message) error {
	lag := time.Since(m.Time).Seconds()
	consumerLag.WithLabelValues(c.deps.Topic, fmt.Sprintf("%d", m.Partition)).Set(lag)

	err := c.processWithRetries(ctx, m)
	if err == nil {
		return nil
	}
	if dlqErr := c.sendToDLQ(ctx, m, err); dlqErr != nil {
		slog.ErrorContext(ctx, "kafka dlq routing failed; leaving offset uncommitted",
			"err", dlqErr,
			"topic", c.deps.Topic,
			"partition", m.Partition,
			"offset", m.Offset,
			"trace_id", TraceIDFromMessage(m),
		)
		return workerpool.ErrSkipCommit
	}
	return nil
}

func (c *Consumer) processWithRetries(ctx context.Context, m kafka.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in kafka handler: %v", r)
			slog.ErrorContext(ctx, "panic recovered in kafka consumer", "err", err, "trace_id", TraceIDFromMessage(m))
			consumerErrors.WithLabelValues(c.deps.Topic, fmt.Sprintf("%d", m.Partition)).Inc()
		}
	}()

	for attempt := 1; attempt <= c.deps.MaxAttempts; attempt++ {
		err = c.deps.Handler(ctx, m)
		if err == nil {
			return nil
		}
		slog.WarnContext(ctx, "kafka handler error",
			"err", err,
			"attempt", attempt,
			"max", c.deps.MaxAttempts,
			"topic", c.deps.Topic,
			"partition", m.Partition,
			"trace_id", TraceIDFromMessage(m),
		)
		if attempt < c.deps.MaxAttempts {
			backoff := retryBackoffWithJitter(attempt)
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

func (c *Consumer) sendToDLQ(ctx context.Context, m kafka.Message, reason error) error {
	if c.deps.DLQWriter == nil {
		return fmt.Errorf("dlq writer not configured for topic %s", c.deps.Topic)
	}
	dlqMsg := kafka.Message{
		Key:   m.Key,
		Value: m.Value,
		Headers: append(m.Headers,
			kafka.Header{Key: "dlq_reason", Value: []byte(reason.Error())},
			kafka.Header{Key: "original_topic", Value: []byte(c.deps.Topic)},
			kafka.Header{Key: "original_partition", Value: []byte(fmt.Sprintf("%d", m.Partition))},
			kafka.Header{Key: "original_offset", Value: []byte(fmt.Sprintf("%d", m.Offset))},
		),
	}
	if err := c.deps.DLQWriter.WriteMessages(ctx, dlqMsg); err != nil {
		return fmt.Errorf("write dlq message: %w", err)
	}
	return nil
}

func retryBackoffWithJitter(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	base := time.Duration(1<<(attempt-1)) * 100 * time.Millisecond
	maxJitter := base / 2
	if maxJitter <= 0 {
		return base
	}
	return base + time.Duration(retryJitterInt63n(int64(maxJitter)))
}

var retryJitterInt63n = rand.Int63n

func (c *Consumer) Close() error {
	if c.pool != nil {
		return c.reader.Close()
	}
	return c.reader.Close()
}
