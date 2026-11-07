package workerpool

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

type fakeSource struct {
	mu       sync.Mutex
	messages []kafka.Message
	idx      int
	commits  int
	closed   bool
}

func (f *fakeSource) FetchMessage(ctx context.Context) (kafka.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.idx >= len(f.messages) {
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}
	m := f.messages[f.idx]
	f.idx++
	return m, nil
}

func (f *fakeSource) CommitMessages(_ context.Context, _ ...kafka.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commits++
	return nil
}

func (f *fakeSource) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func TestPool_PreservesPerPartitionOrdering(t *testing.T) {
	t.Parallel()

	src := &fakeSource{
		messages: []kafka.Message{
			{Partition: 0, Offset: 0, Time: time.Now()},
			{Partition: 0, Offset: 1, Time: time.Now()},
			{Partition: 1, Offset: 0, Time: time.Now()},
			{Partition: 1, Offset: 1, Time: time.Now()},
		},
	}
	var mu sync.Mutex
	seenP0 := make([]int64, 0, 2)
	seenP1 := make([]int64, 0, 2)

	pool, err := New(Config{
		Source:  src,
		Workers: 2,
		Handler: func(_ context.Context, m kafka.Message) error {
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
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = pool.Run(ctx)

	mu.Lock()
	defer mu.Unlock()
	if len(seenP0) != 2 || seenP0[0] != 0 || seenP0[1] != 1 {
		t.Fatalf("partition 0 order: %v", seenP0)
	}
	if len(seenP1) != 2 || seenP1[0] != 0 || seenP1[1] != 1 {
		t.Fatalf("partition 1 order: %v", seenP1)
	}
}

func TestOffsetTracker_Monotonicity(t *testing.T) {
	tracker := NewOffsetTracker()

	// Initial commit for partition 0, offset 10
	if !tracker.ShouldCommit(kafka.Message{Partition: 0, Offset: 10}) {
		t.Error("expected first offset to be accepted")
	}

	// Regressive commit for partition 0, offset 8 (must be rejected)
	if tracker.ShouldCommit(kafka.Message{Partition: 0, Offset: 8}) {
		t.Error("expected regressive offset 8 to be rejected")
	}

	// Equal commit for partition 0, offset 10 (must be rejected)
	if tracker.ShouldCommit(kafka.Message{Partition: 0, Offset: 10}) {
		t.Error("expected duplicate offset 10 to be rejected")
	}

	// Progressive commit for partition 0, offset 11 (must be accepted)
	if !tracker.ShouldCommit(kafka.Message{Partition: 0, Offset: 11}) {
		t.Error("expected progressive offset 11 to be accepted")
	}

	// Partition 1 independent tracking
	if !tracker.ShouldCommit(kafka.Message{Partition: 1, Offset: 5}) {
		t.Error("expected partition 1 offset 5 to be accepted")
	}
}

func TestPool_PoisonPillRetryAndQuarantine(t *testing.T) {
	src := &fakeSource{
		messages: []kafka.Message{
			{Partition: 0, Offset: 0, Value: []byte("poison-pill")},
		},
	}

	var handlerAttempts int
	var failureHandled bool
	var mu sync.Mutex

	pool, err := New(Config{
		Source:     src,
		Workers:    1,
		MaxRetries: 3,
		RetryDelay: 5 * time.Millisecond,
		Handler: func(_ context.Context, m kafka.Message) error {
			mu.Lock()
			defer mu.Unlock()
			handlerAttempts++
			return errors.New("simulated deserialization poison pill")
		},
		OnFailure: func(_ context.Context, m kafka.Message, err error) {
			mu.Lock()
			defer mu.Unlock()
			failureHandled = true
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = pool.Run(ctx)

	mu.Lock()
	defer mu.Unlock()
	if handlerAttempts != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", handlerAttempts)
	}
	if !failureHandled {
		t.Error("expected OnFailure to be invoked for poison pill")
	}
	if src.commits != 1 {
		t.Errorf("expected 1 commit after quarantine, got %d", src.commits)
	}
}
