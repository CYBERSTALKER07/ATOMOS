package bootstrap

import (
	"context"
	"log/slog"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type outboxEventAppender interface {
	Append(ctx context.Context, events []outbox.Event) error
}

// loggingOutboxPublisher is a scaffold publisher seam; production wiring swaps
// this for Kafka writer-backed Publisher.
type loggingOutboxPublisher struct {
	log *slog.Logger
}

func (p *loggingOutboxPublisher) Publish(_ context.Context, topic string, key []byte, value []byte) error {
	if p.log != nil {
		p.log.Debug("outbox published",
			"topic", topic,
			"aggregate_id", string(key),
			"payload_bytes", len(value),
		)
	}
	return nil
}
