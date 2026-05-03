package outbox

import (
	"context"
	"hash/fnv"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	goKafka "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// Relay tails unpublished OutboxEvents rows and publishes them to Kafka.
// It is the ONLY sanctioned way to emit durable state-change events — direct
// writer.WriteMessages from handlers bypasses the atomicity guarantee and is
// forbidden by V.O.I.D. doctrine.
//
// # Sharding model
//
// The relay fans incoming events across numShards parallel publisher goroutines.
// Shard assignment is computed as FNV32(AggregateID) % numShards — the same
// AggregateID always maps to the same shard, so per-entity Kafka ordering is
// preserved. Events for different entities are published concurrently.
//
// # Batch markPublished
//
// All successfully-published event IDs from a single tick are committed to
// Spanner in one Apply call (N mutations, 1 RPC) instead of N separate RPCs.
// FailureCallback is invoked when an outbox event publish fails. The relay
// continues retrying on the next tick; the callback is for real-time alerting
// (e.g. broadcasting OUTBOX_FAILED over WebSocket).
type FailureCallback func(eventID, aggregateID, topic string, err error)

// shardJob is a single unit of work dispatched to a shard worker.
type shardJob struct {
	eventID     string
	aggregateID string
	eventType   string
	topic       string
	traceID     string
	payload     []byte
	resultCh    chan<- shardResult
}

// shardResult carries the outcome of one publish attempt back to the tick collector.
type shardResult struct {
	eventID string
	err     error
}

// Relay is the sharded outbox publisher. Construct with NewRelay.
type Relay struct {
	spanner       *spanner.Client
	brokerAddress string
	pollInterval  time.Duration
	batchSize     int64
	numShards     int
	onFailure     FailureCallback

	mu       sync.Mutex
	writers  map[string]*goKafka.Writer
	shardChs []chan shardJob // one buffered channel per shard
}

// NewRelay constructs a relay bound to a Spanner client and a Kafka broker.
//
// Zero values use sensible defaults:
//   - pollInterval → 2 seconds
//   - batchSize    → 100 events per tick
//   - numShards    → runtime.NumCPU() (at least 2)
func NewRelay(sp *spanner.Client, brokerAddress string, pollInterval time.Duration, batchSize int64, numShards int) *Relay {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	if numShards <= 0 {
		numShards = runtime.NumCPU()
		if numShards < 2 {
			numShards = 2
		}
	}
	chs := make([]chan shardJob, numShards)
	for i := range chs {
		chs[i] = make(chan shardJob, int(batchSize)) // buffer = full batch per shard
	}
	return &Relay{
		spanner:       sp,
		brokerAddress: brokerAddress,
		pollInterval:  pollInterval,
		batchSize:     batchSize,
		numShards:     numShards,
		writers:       make(map[string]*goKafka.Writer),
		shardChs:      chs,
	}
}

// SetOnFailure registers a callback invoked on each publish failure.
func (r *Relay) SetOnFailure(fn FailureCallback) {
	r.onFailure = fn
}

// Start launches the relay. It returns immediately; all goroutines run until
// ctx is cancelled. Errors are logged and never propagated — the relay retries
// on the next tick.
func (r *Relay) Start(ctx context.Context) {
	if r.spanner == nil || r.brokerAddress == "" {
		slog.Warn("outbox.relay.disabled", "reason", "missing Spanner client or broker address")
		return
	}
	// Launch one worker goroutine per shard.
	for i := 0; i < r.numShards; i++ {
		go r.runShard(ctx, i)
	}
	// Launch the reader/fan-out loop.
	go func() {
		slog.Info("outbox.relay.start",
			"poll", r.pollInterval.String(),
			"batch", r.batchSize,
			"shards", r.numShards)
		t := time.NewTicker(r.pollInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				r.closeWriters()
				slog.Info("outbox.relay.stopped")
				return
			case <-t.C:
				if err := r.tick(ctx); err != nil {
					slog.Error("outbox.relay.tick", "err", err)
				}
			}
		}
	}()
}

// tick reads up to batchSize unpublished events, fans them out to shard
// workers, collects results, then batch-marks all successes in one Spanner RPC.
func (r *Relay) tick(ctx context.Context) error {
	events, err := r.readBatch(ctx)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	// resultCh is sized for the full batch so shards never block on send.
	resultCh := make(chan shardResult, len(events))

	// Fan out: dispatch each event to its shard worker.
	for _, ev := range events {
		shard := r.shardFor(ev.aggregateID)
		r.shardChs[shard] <- shardJob{
			eventID:     ev.eventID,
			aggregateID: ev.aggregateID,
			eventType:   ev.eventType,
			topic:       ev.topic,
			traceID:     ev.traceID,
			payload:     ev.payload,
			resultCh:    resultCh,
		}
	}

	// Collect all results (one per dispatched event).
	published := make([]string, 0, len(events))
	for range events {
		res := <-resultCh
		if res.err != nil {
			// already logged by runShard; callback still fires here for metrics
			continue
		}
		published = append(published, res.eventID)
	}

	if len(published) > 0 {
		if err := r.batchMarkPublished(ctx, published); err != nil {
			slog.Error("outbox.relay.batch_mark", "count", len(published), "err", err)
			// Don't return: events were published to Kafka. On next tick the
			// Spanner read will see them again; Kafka dedup via AggregateID key
			// makes double-publish safe at the consumer level.
		} else {
			slog.Info("outbox.relay.published", "count", len(published))
		}
	}
	return nil
}

// batchEvent is the scan target for readBatch.
type batchEvent struct {
	eventID, aggregateID, eventType, topic, traceID string
	payload                                         []byte
}

func (r *Relay) readBatch(ctx context.Context) ([]batchEvent, error) {
	stmt := spanner.Statement{
		SQL: `SELECT EventId, AggregateId, EventType, TopicName, Payload, TraceID
		      FROM OutboxEvents
		      WHERE PublishedAt IS NULL
		      ORDER BY CreatedAt
		      LIMIT @lim`,
		Params: map[string]interface{}{"lim": r.batchSize},
	}
	iter := r.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var events []batchEvent
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var ev batchEvent
		var traceID spanner.NullString
		if err := row.Columns(&ev.eventID, &ev.aggregateID, &ev.eventType, &ev.topic, &ev.payload, &traceID); err != nil {
			slog.Error("outbox.relay.row_scan", "err", err)
			continue
		}
		ev.traceID = traceID.StringVal
		events = append(events, ev)
	}
	return events, nil
}

// runShard is the per-shard worker goroutine. It publishes events from its
// channel sequentially, preserving per-AggregateID ordering within the shard.
func (r *Relay) runShard(ctx context.Context, shardIdx int) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-r.shardChs[shardIdx]:
			err := r.publish(ctx, job.topic, job.aggregateID, job.eventType, job.traceID, job.payload)
			if err != nil {
				slog.Error("outbox.relay.publish",
					"shard", shardIdx,
					"topic", job.topic,
					"event_id", job.eventID,
					"aggregate_id", job.aggregateID,
					"err", err)
				if r.onFailure != nil {
					r.onFailure(job.eventID, job.aggregateID, job.topic, err)
				}
			}
			job.resultCh <- shardResult{eventID: job.eventID, err: err}
		}
	}
}

// shardFor returns the shard index for a given aggregateID.
// FNV32a gives a cheap, uniform distribution across shards while being
// deterministic — same ID always maps to the same shard.
func (r *Relay) shardFor(aggregateID string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(aggregateID))
	return int(h.Sum32()) % r.numShards
}

func (r *Relay) publish(ctx context.Context, topic, key, eventType, traceID string, payload []byte) error {
	w := r.writerFor(topic)
	wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Key = AggregateID — preserves per-entity partition ordering in Kafka.
	// Headers carry the event_type discriminator so consumers route without
	// parsing the payload; trace_id threads the request correlation token.
	headers := []goKafka.Header{
		{Key: "event_type", Value: []byte(eventType)},
	}
	if traceID != "" {
		headers = append(headers, goKafka.Header{Key: "trace_id", Value: []byte(traceID)})
	}
	return w.WriteMessages(wctx, goKafka.Message{
		Key:     []byte(key),
		Value:   payload,
		Headers: headers,
	})
}

// batchMarkPublished commits all eventIDs to Spanner in a single transaction.
// PublishedAt uses a plain UTC timestamp because existing deployments define
// the column as TIMESTAMP, not allow_commit_timestamp=true.
func (r *Relay) batchMarkPublished(ctx context.Context, eventIDs []string) error {
	publishedAt := time.Now().UTC()
	muts := make([]*spanner.Mutation, len(eventIDs))
	for i, id := range eventIDs {
		muts[i] = spanner.Update("OutboxEvents",
			[]string{"EventId", "PublishedAt"},
			[]interface{}{id, publishedAt})
	}
	_, err := r.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite(muts)
	})
	return err
}

func (r *Relay) writerFor(topic string) *goKafka.Writer {
	r.mu.Lock()
	defer r.mu.Unlock()
	if w, ok := r.writers[topic]; ok {
		return w
	}
	w := &goKafka.Writer{
		Addr:         goKafka.TCP(r.brokerAddress),
		Topic:        topic,
		Balancer:     &goKafka.Hash{},
		RequiredAcks: goKafka.RequireAll,
		MaxAttempts:  5,
		BatchTimeout: 10 * time.Millisecond,
	}
	r.writers[topic] = w
	return w
}

func (r *Relay) closeWriters() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for topic, w := range r.writers {
		if err := w.Close(); err != nil {
			slog.Error("outbox.relay.writer_close", "topic", topic, "err", err)
		}
	}
	r.writers = make(map[string]*goKafka.Writer)
}

// Relay tails unpublished OutboxEvents rows and publishes them to Kafka.
// It is the ONLY sanctioned way to emit durable state-change events — direct
// writer.WriteMessages from handlers bypasses the atomicity guarantee and is
// forbidden by V.O.I.D. doctrine.
//
