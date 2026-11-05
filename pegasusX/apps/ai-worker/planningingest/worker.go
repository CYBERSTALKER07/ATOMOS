package planningingest

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/segmentio/kafka-go"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
)

// Runtime consumes planning.signal.ingest.v1 and projects to Spanner.
type Runtime struct {
	reader    *kafka.Reader
	dlqWriter *kafka.Writer
	client    *spanner.Client
	log       *slog.Logger
}

// NewRuntime builds a planning signal ingest consumer.
func NewRuntime(brokers []string, client *spanner.Client, log *slog.Logger) *Runtime {
	return NewRuntimeWithDialer(brokers, nil, client, log)
}

// NewRuntimeWithDialer builds a planning signal ingest consumer with an optional SASL/TLS dialer.
func NewRuntimeWithDialer(brokers []string, dialer *kafka.Dialer, client *spanner.Client, log *slog.Logger) *Runtime {
	return NewRuntimeWithDLQ(brokers, dialer, nil, client, log)
}

// NewRuntimeWithDLQ builds a planning signal ingest consumer with SASL/TLS dialer and DLQ routing.
func NewRuntimeWithDLQ(brokers []string, dialer *kafka.Dialer, dlqWriter *kafka.Writer, client *spanner.Client, log *slog.Logger) *Runtime {
	clean := make([]string, 0, len(brokers))
	for _, b := range brokers {
		if t := strings.TrimSpace(b); t != "" {
			clean = append(clean, t)
		}
	}
	if len(clean) == 0 || client == nil {
		return nil
	}
	if log == nil {
		log = slog.Default()
	}
	return &Runtime{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:               clean,
			Dialer:                dialer,
			GroupID:               "pegasusx-ai-worker-planning-ingest",
			Topic:                 events.TopicPlanningSignalIngest,
			MinBytes:              1e3,
			MaxBytes:              10e6,
			CommitInterval:        0,
			StartOffset:           kafka.FirstOffset,
			WatchPartitionChanges: true,
		}),
		dlqWriter: dlqWriter,
		client:    client,
		log:       log,
	}
}

// Run blocks until ctx is cancelled.
func (r *Runtime) Run(ctx context.Context) {
	if r == nil || r.reader == nil {
		return
	}
	defer r.reader.Close()
	svc := planning.NewService(r.client)
	const maxAttempts = 3
	for {
		msg, err := r.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			r.log.Warn("planning ingest fetch failed", "err", err)
			time.Sleep(250 * time.Millisecond)
			continue
		}

		var lastErr error
		var isPoisonPill bool
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			in, supplierID, decErr := planning.DecodeSignalIngest(msg.Value)
			if decErr != nil {
				lastErr = decErr
				isPoisonPill = true
				break
			}
			if supplierID == "" {
				lastErr = nil
				break
			}
			_, projErr := svc.ProjectSignal(ctx, supplierID, in)
			if projErr == nil {
				lastErr = nil
				break
			}
			lastErr = projErr
			r.log.Warn("planning ingest project attempt failed",
				"attempt", attempt,
				"max_attempts", maxAttempts,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"err", projErr,
			)
			if attempt < maxAttempts {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(1<<(attempt-1)) * 200 * time.Millisecond):
				}
			}
		}

		if lastErr != nil {
			if r.dlqWriter != nil {
				dlqReason := "project_signal_exhausted: " + lastErr.Error()
				if isPoisonPill {
					dlqReason = "decode_poison_pill: " + lastErr.Error()
				}
				dlqMsg := kafka.Message{
					Key:   msg.Key,
					Value: msg.Value,
					Headers: append(msg.Headers,
						kafka.Header{Key: "dlq_original_topic", Value: []byte(events.TopicPlanningSignalIngest)},
						kafka.Header{Key: "original_topic", Value: []byte(events.TopicPlanningSignalIngest)},
						kafka.Header{Key: "dlq_reason", Value: []byte(dlqReason)},
						kafka.Header{Key: "dlq_time", Value: []byte(time.Now().UTC().Format(time.RFC3339Nano))},
					),
				}
				dlqCtx, cancelDLQ := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Second)
				dlqErr := r.dlqWriter.WriteMessages(dlqCtx, dlqMsg)
				cancelDLQ()
				if dlqErr != nil {
					r.log.Error("planning ingest dlq write failed; leaving offset uncommitted", "err", dlqErr)
					if ctx.Err() != nil {
						return
					}
					time.Sleep(2 * time.Second)
					continue
				}
				r.log.Warn("planning ingest routed failed message to DLQ",
					"partition", msg.Partition,
					"offset", msg.Offset,
					"reason", dlqReason,
				)
			} else if !isPoisonPill {
				if ctx.Err() != nil {
					return
				}
				time.Sleep(2 * time.Second)
				continue
			} else {
				r.log.Error("planning ingest skipping poison pill without dlqWriter",
					"partition", msg.Partition,
					"offset", msg.Offset,
					"err", lastErr,
				)
			}
		}

		commitCtx, cancelCommit := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		if err := r.reader.CommitMessages(commitCtx, msg); err != nil {
			r.log.Warn("planning ingest commit failed", "err", err)
		}
		cancelCommit()
	}
}

func (r *Runtime) handle(ctx context.Context, svc *planning.Service, value []byte) error {
	in, supplierID, err := planning.DecodeSignalIngest(value)
	if err != nil {
		return err
	}
	if supplierID == "" {
		return nil
	}
	_, err = svc.ProjectSignal(ctx, supplierID, in)
	return err
}

// Close releases the kafka reader.
func (r *Runtime) Close() error {
	if r == nil || r.reader == nil {
		return nil
	}
	return r.reader.Close()
}
