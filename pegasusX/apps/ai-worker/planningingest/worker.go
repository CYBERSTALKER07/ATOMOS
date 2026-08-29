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
	reader *kafka.Reader
	client *spanner.Client
	log    *slog.Logger
}

// NewRuntime builds a planning signal ingest consumer.
func NewRuntime(brokers []string, client *spanner.Client, log *slog.Logger) *Runtime {
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
			Brokers:        clean,
			GroupID:        "pegasusx-ai-worker-planning-ingest",
			Topic:          events.TopicPlanningSignalIngest,
			MinBytes:       1e3,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.LastOffset,
		}),
		client: client,
		log:    log,
	}
}

// Run blocks until ctx is cancelled.
func (r *Runtime) Run(ctx context.Context) {
	if r == nil || r.reader == nil {
		return
	}
	defer r.reader.Close()
	svc := planning.NewService(r.client)
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
		if err := r.handle(ctx, svc, msg.Value); err != nil {
			r.log.Warn("planning ingest project failed", "err", err)
			if ctx.Err() != nil {
				return
			}
			time.Sleep(2 * time.Second)
			continue // Do not commit, let it be retried
		}
		if err := r.reader.CommitMessages(ctx, msg); err != nil {
			r.log.Warn("planning ingest commit failed", "err", err)
		}
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
	return svc.ProjectSignal(ctx, supplierID, in)
}

// Close releases the kafka reader.
func (r *Runtime) Close() error {
	if r == nil || r.reader == nil {
		return nil
	}
	return r.reader.Close()
}
