// Package workerpool runs a Kafka consumer with partition-parallel handler
// goroutines while preserving per-partition ordering.
//
// The pool fetches from a single segmentio/kafka-go reader and shards each
// fetched message to a per-worker buffered channel keyed on partition. This
// gives N-way parallelism across partitions while every message from a given
// partition is always processed by the same goroutine — preserving the
// per-aggregate ordering guarantee Kafka makes by partition key.
//
// Delivery is at-least-once: the offset is committed AFTER the handler
// returns. A handler error is logged and routed to the optional FailureHandler
// (mirrors RouteToDLQ); the offset is still committed to avoid head-of-line
// blocking on a poison message.
//
// Shutdown: cancel the parent context. The pool drains its in-flight queues,
// waits for workers, and closes the underlying reader.
//
// This package MUST NOT import backend-go/kafka (the kafka package depends on
// this primitive). The headerValue helper is intentionally duplicated.
package workerpool

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"backend-go/telemetry"

	goKafka "github.com/segmentio/kafka-go"
)

// Handler processes a single Kafka message. Returning an error logs the
// failure and (if configured) invokes OnFailure; the offset is still
// committed to avoid blocking the partition.
type Handler func(ctx context.Context, msg goKafka.Message) error

// FailureHandler is invoked when Handler returns a non-nil error. It runs
// inside the worker goroutine before commit.
type FailureHandler func(ctx context.Context, msg goKafka.Message, handlerErr error)

// MessageSource is the subset of *goKafka.Reader the pool depends on.
// Defined as an interface so tests can substitute a fake.
type MessageSource interface {
	FetchMessage(ctx context.Context) (goKafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...goKafka.Message) error
	Close() error
}

// Config wires a Pool. Source and Handler are required.
type Config struct {
	Source     MessageSource
	Handler    Handler
	OnFailure  FailureHandler
	Workers    int          // defaults to GOMAXPROCS(0)
	QueueDepth int          // per-worker buffer; defaults to 32
	Logger     *slog.Logger // defaults to slog.Default()
	Name       string       // log tag; defaults to reader topic if discoverable
}

// Pool is a partition-parallel Kafka consumer.
type Pool struct {
	cfg     Config
	workers int
	queue   int
	log     *slog.Logger
}

// New validates cfg and returns a ready Pool.
func New(cfg Config) (*Pool, error) {
	if cfg.Source == nil {
		return nil, errors.New("workerpool: Source is required")
	}
	if cfg.Handler == nil {
		return nil, errors.New("workerpool: Handler is required")
	}
	workers := cfg.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	queue := cfg.QueueDepth
	if queue <= 0 {
		queue = 32
	}
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}
	name := cfg.Name
	if name == "" {
		if r, ok := cfg.Source.(*goKafka.Reader); ok {
			name = r.Config().Topic
		} else {
			name = "workerpool"
		}
	}
	return &Pool{
		cfg:     Config{Source: cfg.Source, Handler: cfg.Handler, OnFailure: cfg.OnFailure, Name: name},
		workers: workers,
		queue:   queue,
		log:     log.With("consumer", name, "workers", workers),
	}, nil
}

// Run blocks until ctx is cancelled or an unrecoverable error occurs.
// Returns ctx.Err() on graceful shutdown.
func (p *Pool) Run(ctx context.Context) error {
	chans := make([]chan goKafka.Message, p.workers)
	var wg sync.WaitGroup
	for i := range chans {
		chans[i] = make(chan goKafka.Message, p.queue)
		wg.Add(1)
		go p.runWorker(ctx, chans[i], &wg)
	}
	defer func() {
		for _, c := range chans {
			close(c)
		}
		wg.Wait()
		_ = p.cfg.Source.Close()
	}()

	bo := &backoff{}
	for {
		m, err := p.cfg.Source.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			d := bo.next()
			p.log.Error("fetch failed; backing off",
				"err", err, "streak", bo.streak, "backoff", d)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d):
			}
			continue
		}
		bo.reset()
		idx := int(uint(m.Partition)) % p.workers
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chans[idx] <- m:
		}
	}
}

// runWorker processes messages from one channel sequentially. Single-goroutine
// drain per channel preserves per-partition ordering.
func (p *Pool) runWorker(parent context.Context, in <-chan goKafka.Message, wg *sync.WaitGroup) {
	defer wg.Done()
	for m := range in {
		traceID := headerValue(m.Headers, "trace_id")
		ctx := parent
		if traceID != "" {
			ctx = telemetry.WithTraceID(parent, traceID)
		}
		err := p.cfg.Handler(ctx, m)
		if err != nil {
			p.log.ErrorContext(ctx, "handler error",
				"partition", m.Partition, "offset", m.Offset, "err", err)
			if p.cfg.OnFailure != nil {
				p.cfg.OnFailure(ctx, m, err)
			}
		}
		// Commit even on handler error to avoid head-of-line blocking on a
		// poison message. OnFailure (or DLQ) is the durable retry mechanism.
		commitCtx := parent
		if commitCtx.Err() != nil {
			// Parent cancelled mid-flight; use a detached context so the
			// in-flight commit can still land before we exit.
			var cancel context.CancelFunc
			commitCtx, cancel = context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
			defer cancel()
		}
		if cerr := p.cfg.Source.CommitMessages(commitCtx, m); cerr != nil {
			p.log.ErrorContext(ctx, "commit failed",
				"partition", m.Partition, "offset", m.Offset, "err", cerr)
		}
	}
}

// headerValue returns the value of the named header (case-sensitive), or "".
// Duplicated here so this package does not import backend-go/kafka.
func headerValue(headers []goKafka.Header, key string) string {
	for _, h := range headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

// backoff implements the same exponential schedule as kafka.ConsumerBackoff
// (100ms → 30s cap), duplicated here to keep this package import-clean.
type backoff struct {
	streak int
}

func (b *backoff) next() time.Duration {
	b.streak++
	d := time.Duration(100*1<<min(b.streak-1, 10)) * time.Millisecond
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func (b *backoff) reset() { b.streak = 0 }
