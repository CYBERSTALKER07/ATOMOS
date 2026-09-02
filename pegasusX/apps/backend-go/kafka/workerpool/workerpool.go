// Package workerpool runs a Kafka consumer with partition-parallel handler
// goroutines while preserving per-partition ordering.
package workerpool

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/segmentio/kafka-go"
)

// ErrSkipCommit tells the pool not to commit the offset (e.g. DLQ write failed).
var ErrSkipCommit = errors.New("workerpool: skip commit")

// Handler processes a single Kafka message.
type Handler func(ctx context.Context, msg kafka.Message) error

// FailureHandler runs when Handler returns a non-nil error other than ErrSkipCommit.
type FailureHandler func(ctx context.Context, msg kafka.Message, handlerErr error)

// MessageSource is the subset of *kafka.Reader the pool depends on.
type MessageSource interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// Config wires a Pool. Source and Handler are required.
type Config struct {
	Source     MessageSource
	Handler    Handler
	OnFailure  FailureHandler
	Workers    int
	QueueDepth int
	Logger     *slog.Logger
	Name       string
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
		if r, ok := cfg.Source.(*kafka.Reader); ok {
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

// Run blocks until ctx is cancelled.
func (p *Pool) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	chans := make([]chan kafka.Message, p.workers)
	var wg sync.WaitGroup
	for i := range chans {
		chans[i] = make(chan kafka.Message, p.queue)
		wg.Add(1)
		go p.runWorker(ctx, cancel, chans[i], &wg)
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

func (p *Pool) runWorker(parent context.Context, cancel context.CancelFunc, in <-chan kafka.Message, wg *sync.WaitGroup) {
	defer wg.Done()
	for m := range in {
		ctx := ContextWithTrace(parent, m)
		err := p.cfg.Handler(ctx, m)
		if err != nil && !errors.Is(err, ErrSkipCommit) {
			p.log.ErrorContext(ctx, "handler error",
				"partition", m.Partition, "offset", m.Offset, "err", err)
			if p.cfg.OnFailure != nil {
				p.cfg.OnFailure(ctx, m, err)
			}
		}
		if errors.Is(err, ErrSkipCommit) {
			p.log.ErrorContext(ctx, "halting workerpool due to unrecoverable message skip",
				"partition", m.Partition, "offset", m.Offset)
			cancel()
			return
		}
		commitCtx := parent
		if commitCtx.Err() != nil {
			var commitCancel context.CancelFunc
			commitCtx, commitCancel = context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
			defer commitCancel()
		}
		if cerr := p.cfg.Source.CommitMessages(commitCtx, m); cerr != nil {
			p.log.ErrorContext(ctx, "commit failed",
				"partition", m.Partition, "offset", m.Offset, "err", cerr)
		}
	}
}

// ContextWithTrace binds trace_id from Kafka headers, then JSON body fallback.
func ContextWithTrace(parent context.Context, m kafka.Message) context.Context {
	traceID := HeaderValue(m.Headers, "trace_id")
	if traceID == "" {
		traceID = TraceIDFromPayload(m.Value)
	}
	if traceID == "" {
		return parent
	}
	return outbox.WithTraceID(parent, traceID)
}

// HeaderValue returns the named header value or "".
func HeaderValue(headers []kafka.Header, key string) string {
	for _, h := range headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

// TraceIDFromPayload reads trace_id from a JSON object envelope.
func TraceIDFromPayload(value []byte) string {
	var envelope struct {
		TraceID string `json:"trace_id"`
	}
	if err := json.Unmarshal(value, &envelope); err != nil {
		return ""
	}
	return envelope.TraceID
}

type backoff struct {
	streak int
}

func (b *backoff) next() time.Duration {
	b.streak++
	d := time.Duration(100*1<<(min(b.streak-1, 10))) * time.Millisecond
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func (b *backoff) reset() { b.streak = 0 }
