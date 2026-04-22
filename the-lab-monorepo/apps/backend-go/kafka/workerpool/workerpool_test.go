package workerpool

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"backend-go/telemetry"

	goKafka "github.com/segmentio/kafka-go"
)

// fakeSource implements MessageSource for tests. It serves a fixed slice of
// messages once, then blocks on FetchMessage until ctx is cancelled.
type fakeSource struct {
	mu        sync.Mutex
	pending   []goKafka.Message
	committed []goKafka.Message
	fetchErr  error
	served    int32
	closed    int32
}

func (f *fakeSource) FetchMessage(ctx context.Context) (goKafka.Message, error) {
	for {
		f.mu.Lock()
		if f.fetchErr != nil {
			err := f.fetchErr
			f.fetchErr = nil
			f.mu.Unlock()
			return goKafka.Message{}, err
		}
		if len(f.pending) > 0 {
			m := f.pending[0]
			f.pending = f.pending[1:]
			f.mu.Unlock()
			atomic.AddInt32(&f.served, 1)
			return m, nil
		}
		f.mu.Unlock()
		select {
		case <-ctx.Done():
			return goKafka.Message{}, ctx.Err()
		case <-time.After(2 * time.Millisecond):
		}
	}
}

func (f *fakeSource) CommitMessages(_ context.Context, msgs ...goKafka.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.committed = append(f.committed, msgs...)
	return nil
}

func (f *fakeSource) Close() error {
	atomic.StoreInt32(&f.closed, 1)
	return nil
}

func (f *fakeSource) committedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.committed)
}

func TestNew_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"missing source", Config{Handler: func(context.Context, goKafka.Message) error { return nil }}, true},
		{"missing handler", Config{Source: &fakeSource{}}, true},
		{"valid", Config{Source: &fakeSource{}, Handler: func(context.Context, goKafka.Message) error { return nil }}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestPool_DeliversAndCommits(t *testing.T) {
	src := &fakeSource{
		pending: []goKafka.Message{
			{Partition: 0, Offset: 1, Value: []byte("a")},
			{Partition: 1, Offset: 1, Value: []byte("b")},
			{Partition: 2, Offset: 1, Value: []byte("c")},
			{Partition: 3, Offset: 1, Value: []byte("d")},
		},
	}
	var seen int32
	pool, err := New(Config{
		Source:  src,
		Workers: 4,
		Handler: func(_ context.Context, _ goKafka.Message) error {
			atomic.AddInt32(&seen, 1)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = pool.Run(ctx)

	if got := atomic.LoadInt32(&seen); got != 4 {
		t.Fatalf("handler invocations = %d, want 4", got)
	}
	if got := src.committedCount(); got != 4 {
		t.Fatalf("commits = %d, want 4", got)
	}
	if atomic.LoadInt32(&src.closed) != 1 {
		t.Fatal("source not closed on shutdown")
	}
}

func TestPool_PreservesPerPartitionOrder(t *testing.T) {
	// 8 messages across 2 partitions; per-partition ordering must hold even
	// with multiple workers.
	src := &fakeSource{}
	for i := int64(0); i < 4; i++ {
		src.pending = append(src.pending,
			goKafka.Message{Partition: 0, Offset: i, Value: []byte("p0")},
			goKafka.Message{Partition: 1, Offset: i, Value: []byte("p1")},
		)
	}

	var mu sync.Mutex
	seenP0, seenP1 := []int64{}, []int64{}
	pool, err := New(Config{
		Source:  src,
		Workers: 4,
		Handler: func(_ context.Context, m goKafka.Message) error {
			mu.Lock()
			defer mu.Unlock()
			if m.Partition == 0 {
				seenP0 = append(seenP0, m.Offset)
			} else {
				seenP1 = append(seenP1, m.Offset)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = pool.Run(ctx)

	for i, want := range []int64{0, 1, 2, 3} {
		if seenP0[i] != want {
			t.Errorf("partition 0 order: got %v, want sequential", seenP0)
			break
		}
		if seenP1[i] != want {
			t.Errorf("partition 1 order: got %v, want sequential", seenP1)
			break
		}
	}
}

func TestPool_FailureHandlerInvokedAndCommitsAnyway(t *testing.T) {
	src := &fakeSource{pending: []goKafka.Message{
		{Partition: 0, Offset: 1, Value: []byte("poison")},
	}}
	handlerErr := errors.New("boom")
	var failureSeen int32
	pool, err := New(Config{
		Source:  src,
		Workers: 1,
		Handler: func(context.Context, goKafka.Message) error { return handlerErr },
		OnFailure: func(_ context.Context, _ goKafka.Message, err error) {
			if !errors.Is(err, handlerErr) {
				t.Errorf("OnFailure got %v, want %v", err, handlerErr)
			}
			atomic.AddInt32(&failureSeen, 1)
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_ = pool.Run(ctx)

	if atomic.LoadInt32(&failureSeen) != 1 {
		t.Fatal("OnFailure not invoked")
	}
	if got := src.committedCount(); got != 1 {
		t.Fatalf("commits = %d, want 1 (commit-on-failure to avoid stuck partition)", got)
	}
}

func TestPool_PropagatesTraceID(t *testing.T) {
	src := &fakeSource{pending: []goKafka.Message{
		{Partition: 0, Offset: 1, Value: []byte("v"), Headers: []goKafka.Header{
			{Key: "trace_id", Value: []byte("trace-xyz")},
		}},
	}}
	var got string
	var done sync.WaitGroup
	done.Add(1)
	pool, err := New(Config{
		Source:  src,
		Workers: 1,
		Handler: func(ctx context.Context, _ goKafka.Message) error {
			got = telemetry.TraceIDFromContext(ctx)
			done.Done()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	go pool.Run(ctx)
	done.Wait()
	if got != "trace-xyz" {
		t.Fatalf("trace_id propagated = %q, want trace-xyz", got)
	}
}

func TestPool_FetchErrorBackoffThenRecovers(t *testing.T) {
	src := &fakeSource{
		pending:  []goKafka.Message{{Partition: 0, Offset: 1, Value: []byte("v")}},
		fetchErr: io.ErrUnexpectedEOF,
	}
	var seen int32
	pool, err := New(Config{
		Source:  src,
		Workers: 1,
		Handler: func(context.Context, goKafka.Message) error {
			atomic.AddInt32(&seen, 1)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = pool.Run(ctx)

	if atomic.LoadInt32(&seen) != 1 {
		t.Fatalf("handler invocations after recovery = %d, want 1", seen)
	}
}

func TestPool_GracefulShutdownReturnsCtxError(t *testing.T) {
	src := &fakeSource{}
	pool, err := New(Config{
		Source:  src,
		Workers: 2,
		Handler: func(context.Context, goKafka.Message) error { return nil },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := pool.Run(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run err = %v, want DeadlineExceeded", err)
	}
	if atomic.LoadInt32(&src.closed) != 1 {
		t.Fatal("source not closed on shutdown")
	}
}
