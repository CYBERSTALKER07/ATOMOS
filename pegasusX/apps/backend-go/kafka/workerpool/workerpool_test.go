package workerpool

import (
	"context"
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
