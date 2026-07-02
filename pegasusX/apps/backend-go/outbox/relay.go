package outbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// RelayConfig tunes the relay loop. Defaults match the doctrine: 250ms tick,
// batch 100, bounded retry per event with exponential backoff + jitter.
type RelayConfig struct {
	TickInterval     time.Duration
	BatchSize        int
	MaxPublishTries  int
	BaseBackoff      time.Duration
	MaxBackoff       time.Duration
	WatchdogInterval time.Duration
	StuckThreshold   time.Duration
}

func (c *RelayConfig) applyDefaults() {
	if c.TickInterval <= 0 {
		c.TickInterval = 250 * time.Millisecond
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	if c.MaxPublishTries <= 0 {
		c.MaxPublishTries = 5
	}
	if c.BaseBackoff <= 0 {
		c.BaseBackoff = 100 * time.Millisecond
	}
	if c.MaxBackoff <= 0 {
		c.MaxBackoff = 5 * time.Second
	}
	if c.WatchdogInterval <= 0 {
		c.WatchdogInterval = 30 * time.Second
	}
	if c.StuckThreshold <= 0 {
		c.StuckThreshold = 60 * time.Second
	}
}

// Relay drains OutboxEvents into Kafka. Construct with NewRelay then call
// Start(ctx) inside bootstrap.NewApp; it returns when ctx is cancelled.
type Relay struct {
	store     Store
	publisher Publisher
	cfg       RelayConfig
	log       *slog.Logger
}

// NewRelay wires the relay with the given store, publisher and config.
func NewRelay(store Store, publisher Publisher, cfg RelayConfig, log *slog.Logger) *Relay {
	cfg.applyDefaults()
	if log == nil {
		log = slog.Default()
	}
	return &Relay{store: store, publisher: publisher, cfg: cfg, log: log}
}

// Start runs the relay loop until ctx is cancelled. Safe to call once per
// process; do not start multiple instances against the same OutboxEvents
// partition without coordination.
func (r *Relay) Start(ctx context.Context) {
	drainTicker := time.NewTicker(r.cfg.TickInterval)
	defer drainTicker.Stop()
	watchdogTicker := time.NewTicker(r.cfg.WatchdogInterval)
	defer watchdogTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			r.log.Info("outbox relay stopping", "reason", ctx.Err())
			return
		case <-drainTicker.C:
			r.drainOnce(ctx)
		case <-watchdogTicker.C:
			r.watchdogOnce(ctx)
		}
	}
}

func (r *Relay) watchdogOnce(ctx context.Context) {
	count, err := r.store.CountUnpublished(ctx)
	if err != nil {
		r.log.Error("outbox watchdog count failed", "err", err)
	} else {
		SetUnpublishedCount(count)
	}
	events, err := r.store.Fetch(ctx, r.cfg.BatchSize)
	if err != nil {
		r.log.Error("outbox watchdog fetch failed", "err", err)
		return
	}
	now := time.Now().UTC()
	var stuck int
	var oldestID string
	var oldestAt time.Time
	for _, e := range events {
		if now.Sub(e.CreatedAt) <= r.cfg.StuckThreshold {
			continue
		}
		stuck++
		if oldestID == "" || e.CreatedAt.Before(oldestAt) {
			oldestID = e.EventID
			oldestAt = e.CreatedAt
		}
	}
	if stuck == 0 {
		return
	}
	r.log.Error("outbox stuck events detected",
		"count", stuck,
		"oldest_event_id", oldestID,
		"oldest_created_at", oldestAt.UTC().Format(time.RFC3339Nano),
	)
}

func (r *Relay) drainOnce(ctx context.Context) {
	events, err := r.store.Fetch(ctx, r.cfg.BatchSize)
	if err != nil {
		r.log.Error("outbox fetch failed", "err", err)
		return
	}
	if len(events) == 0 {
		return
	}
	published := make([]string, 0, len(events))
	for _, e := range events {
		if err := r.publishWithRetry(ctx, e); err != nil {
			r.log.Error("outbox publish exhausted retries",
				"event_id", e.EventID,
				"aggregate_type", e.AggregateType,
				"aggregate_id", e.AggregateID,
				"topic", e.TopicName,
				"err", err,
			)
			continue
		}
		published = append(published, e.EventID)
	}
	if len(published) == 0 {
		return
	}
	if err := r.store.MarkPublished(ctx, published, time.Now().UTC()); err != nil {
		r.log.Error("outbox mark published failed", "count", len(published), "err", err)
	}
}

// publishWithRetry applies bounded retry with exponential backoff + jitter.
// Returns nil on success, last error on exhaustion. Honours ctx cancellation.
func (r *Relay) publishWithRetry(ctx context.Context, e Event) error {
	topics := events.RelayPublishTopics(e.TopicName, e.Payload)
	var lastErr error
	for attempt := 1; attempt <= r.cfg.MaxPublishTries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		attemptErr := error(nil)
		for _, topic := range topics {
			err := r.publisher.Publish(ctx, topic, granularRoutingKey(e), e.Payload)
			if err != nil {
				attemptErr = err
				break
			}
		}
		if attemptErr == nil {
			return nil
		}
		lastErr = attemptErr
		r.log.Warn("outbox publish attempt failed",
			"event_id", e.EventID,
			"topics", topics,
			"attempt", attempt,
			"max_attempts", r.cfg.MaxPublishTries,
			"err", attemptErr,
		)
		if attempt == r.cfg.MaxPublishTries {
			break
		}
		sleep := backoffWithJitter(r.cfg.BaseBackoff, r.cfg.MaxBackoff, attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleep):
		}
	}
	return lastErr
}

func backoffWithJitter(base, maxBackoff time.Duration, attempt int) time.Duration {
	d := base << (attempt - 1)
	if d <= 0 || d > maxBackoff {
		d = maxBackoff
	}
	// Full jitter: random in [0, d).
	return time.Duration(rand.Int63n(int64(d) + 1))
}

func granularRoutingKey(e Event) []byte {
	key := []byte(e.AggregateID)

	// Fast path: if the payload isn't JSON or doesn't have common sub-entities, just return AggregateID.
	if len(e.Payload) == 0 || e.Payload[0] != '{' {
		return key
	}

	var envelope struct {
		OrderID    string `json:"order_id"`
		ManifestID string `json:"manifest_id"`
		RouteID    string `json:"route_id"`
		DriverID   string `json:"driver_id"`
	}

	// Ignore unmarshal errors, fallback to default key
	if err := json.Unmarshal(e.Payload, &envelope); err == nil {
		if envelope.OrderID != "" {
			return append(key, []byte(":"+envelope.OrderID)...)
		}
		if envelope.ManifestID != "" {
			return append(key, []byte(":"+envelope.ManifestID)...)
		}
		if envelope.RouteID != "" {
			return append(key, []byte(":"+envelope.RouteID)...)
		}
		if envelope.DriverID != "" {
			return append(key, []byte(":"+envelope.DriverID)...)
		}
	}

	return key
}
