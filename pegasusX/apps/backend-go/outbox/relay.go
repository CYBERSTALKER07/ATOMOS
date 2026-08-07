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
	// MaxTotalAttempts is the persistent per-event publish budget across ticks;
	// on exhaustion the event moves to the dead-letter sink. Default 20.
	MaxTotalAttempts int64
	// PublishTimeout bounds one broker produce attempt. Without it a wedged
	// publish blocks the single-threaded drain loop indefinitely (observed as
	// multi-minute system-wide event delays with no logs). Default 10s.
	PublishTimeout time.Duration
	// StoreTimeout bounds Fetch/MarkPublished/RecordPublishFailures calls for
	// the same reason. Default 15s.
	StoreTimeout time.Duration
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
	if c.MaxTotalAttempts <= 0 {
		c.MaxTotalAttempts = 20
	}
	if c.PublishTimeout <= 0 {
		c.PublishTimeout = 10 * time.Second
	}
	if c.StoreTimeout <= 0 {
		c.StoreTimeout = 15 * time.Second
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
	started := time.Now()
	fetchCtx, cancelFetch := context.WithTimeout(ctx, r.cfg.StoreTimeout)
	events, err := r.store.Fetch(fetchCtx, r.cfg.BatchSize)
	cancelFetch()
	if err != nil {
		r.log.Error("outbox fetch failed", "err", err)
		return
	}
	if len(events) == 0 {
		return
	}
	published := make([]string, 0, len(events))
	failed := make(map[string]error)
	for _, e := range events {
		if err := r.publishWithRetry(ctx, e); err != nil {
			r.log.Error("outbox publish exhausted retries",
				"event_id", e.EventID,
				"aggregate_type", e.AggregateType,
				"aggregate_id", e.AggregateID,
				"topic", e.TopicName,
				"err", err,
			)
			failed[e.EventID] = err
			continue
		}
		published = append(published, e.EventID)
	}
	if len(published) > 0 {
		markCtx, cancelMark := context.WithTimeout(ctx, r.cfg.StoreTimeout)
		err := r.store.MarkPublished(markCtx, published, time.Now().UTC())
		cancelMark()
		if err != nil {
			r.log.Error("outbox mark published failed", "count", len(published), "err", err)
		}
	}
	if len(failed) > 0 {
		ids := make([]string, 0, len(failed))
		var firstErr error
		for id, ferr := range failed {
			ids = append(ids, id)
			if firstErr == nil {
				firstErr = ferr
			}
		}
		errText := "outbox publish failed"
		if firstErr != nil {
			errText = firstErr.Error()
		}
		recordCtx, cancelRecord := context.WithTimeout(ctx, r.cfg.StoreTimeout)
		deadLettered, derr := r.store.RecordPublishFailures(recordCtx, ids, errText, r.cfg.MaxTotalAttempts)
		cancelRecord()
		if derr != nil {
			r.log.Error("outbox record publish failures failed", "count", len(ids), "err", derr)
			return
		}
		if len(deadLettered) > 0 {
			IncDeadLettered(len(deadLettered))
			r.log.Error("outbox events dead-lettered after exhausting attempts",
				"event_ids", deadLettered,
				"max_total_attempts", r.cfg.MaxTotalAttempts,
				"err", errText,
			)
		}
	}
	if dur := time.Since(started); dur > 5*time.Second {
		r.log.Warn("outbox drain slow",
			"duration", dur.String(),
			"batch", len(events),
			"published", len(published),
			"failed", len(failed),
		)
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
			pubCtx, cancel := context.WithTimeout(ctx, r.cfg.PublishTimeout)
			err := publishOutboxEvent(pubCtx, r.publisher, topic, granularRoutingKey(e), e)
			cancel()
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

func publishOutboxEvent(ctx context.Context, pub Publisher, topic string, key []byte, e Event) error {
	headers := map[string][]byte{"event_id": []byte(e.EventID)}
	if hp, ok := pub.(HeaderPublisher); ok {
		return hp.PublishWithHeaders(ctx, topic, key, e.Payload, headers)
	}
	return pub.Publish(ctx, topic, key, e.Payload)
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
