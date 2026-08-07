package outbox

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

type relayTestStore struct {
	events    []Event
	fetchErr  error
	markIDs   []string
	markAt    time.Time
	markCalls int
}

func (s *relayTestStore) Fetch(_ context.Context, limit int) ([]Event, error) {
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	if limit <= 0 || limit >= len(s.events) {
		return append([]Event(nil), s.events...), nil
	}
	return append([]Event(nil), s.events[:limit]...), nil
}

func (s *relayTestStore) MarkPublished(_ context.Context, eventIDs []string, at time.Time) error {
	s.markCalls++
	s.markIDs = append([]string(nil), eventIDs...)
	s.markAt = at
	return nil
}

func (s *relayTestStore) CountUnpublished(_ context.Context) (int64, error) {
	return int64(len(s.events) - len(s.markIDs)), nil
}

func (s *relayTestStore) RecordPublishFailures(_ context.Context, eventIDs []string, _ string, _ int64) ([]string, error) {
	return nil, nil
}

type relayTestPublisher struct {
	errorsByCall []error
	callCount    int
}

func (p *relayTestPublisher) Publish(_ context.Context, _ string, _ []byte, _ []byte) error {
	p.callCount++
	idx := p.callCount - 1
	if idx >= 0 && idx < len(p.errorsByCall) {
		return p.errorsByCall[idx]
	}
	return nil
}

func TestRelayDrainOnceMarksPublishedOnSuccess(t *testing.T) {
	t.Parallel()

	store := &relayTestStore{events: []Event{{EventID: "e1", AggregateID: "a1", TopicName: "t1", Payload: []byte("p1")}}}
	pub := &relayTestPublisher{}
	relay := NewRelay(store, pub, RelayConfig{MaxPublishTries: 2, BaseBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}, nil)

	relay.drainOnce(context.Background())

	if pub.callCount != 1 {
		t.Fatalf("publish call count = %d, want 1", pub.callCount)
	}
	if store.markCalls != 1 {
		t.Fatalf("mark call count = %d, want 1", store.markCalls)
	}
	if !reflect.DeepEqual(store.markIDs, []string{"e1"}) {
		t.Fatalf("mark ids = %v, want [e1]", store.markIDs)
	}
	if store.markAt.IsZero() {
		t.Fatalf("mark timestamp should be set")
	}
}

func TestRelayDrainOnceRetriesAndMarksPublished(t *testing.T) {
	t.Parallel()

	store := &relayTestStore{events: []Event{{EventID: "e2", AggregateID: "a2", TopicName: "t2", Payload: []byte("p2")}}}
	pub := &relayTestPublisher{errorsByCall: []error{errors.New("first attempt failed"), nil}}
	relay := NewRelay(store, pub, RelayConfig{MaxPublishTries: 2, BaseBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}, nil)

	relay.drainOnce(context.Background())

	if pub.callCount != 2 {
		t.Fatalf("publish call count = %d, want 2", pub.callCount)
	}
	if store.markCalls != 1 {
		t.Fatalf("mark call count = %d, want 1", store.markCalls)
	}
	if !reflect.DeepEqual(store.markIDs, []string{"e2"}) {
		t.Fatalf("mark ids = %v, want [e2]", store.markIDs)
	}
}

func TestRelayDrainOnceSkipsMarkWhenPublishExhausted(t *testing.T) {
	t.Parallel()

	store := &relayTestStore{events: []Event{{EventID: "e3", AggregateID: "a3", TopicName: "t3", Payload: []byte("p3")}}}
	pub := &relayTestPublisher{errorsByCall: []error{errors.New("fail-1"), errors.New("fail-2")}}
	relay := NewRelay(store, pub, RelayConfig{MaxPublishTries: 2, BaseBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}, nil)

	relay.drainOnce(context.Background())

	if pub.callCount != 2 {
		t.Fatalf("publish call count = %d, want 2", pub.callCount)
	}
	if store.markCalls != 0 {
		t.Fatalf("mark call count = %d, want 0", store.markCalls)
	}
	if len(store.markIDs) != 0 {
		t.Fatalf("mark ids = %v, want empty", store.markIDs)
	}
}

// blockingPublisher simulates a wedged broker write: blocks until ctx deadline.
type blockingPublisher struct {
	calls int32
}

func (p *blockingPublisher) Publish(ctx context.Context, _ string, _ []byte, _ []byte) error {
	atomic.AddInt32(&p.calls, 1)
	<-ctx.Done()
	return ctx.Err()
}

// Regression: before PublishTimeout, a wedged WriteMessages blocked the
// single-threaded drain loop indefinitely (observed as multi-minute stalls of
// ALL event delivery in SSMR with no relay logs). The drain must fail the
// event within MaxPublishTries × PublishTimeout and move on.
func TestRelayDrainOnceBoundsWedgedPublisher(t *testing.T) {
	t.Parallel()

	store := &relayTestStore{events: []Event{
		{EventID: "wedged", AggregateID: "a-w", TopicName: "t1", Payload: []byte("p1")},
		{EventID: "healthy", AggregateID: "a-h", TopicName: "t2", Payload: []byte("p2")},
	}}
	pub := &blockingPublisher{}
	relay := NewRelay(store, pub, RelayConfig{
		MaxPublishTries: 2,
		BaseBackoff:     time.Millisecond,
		MaxBackoff:      2 * time.Millisecond,
		PublishTimeout:  50 * time.Millisecond,
	}, nil)

	// Both events hit the blocking publisher; drain must finish in bounded time.
	start := time.Now()
	relay.drainOnce(context.Background())
	elapsed := time.Since(start)

	// 2 events × 2 tries × 50ms + small backoffs = ~400ms upper bound; allow margin.
	if elapsed > 5*time.Second {
		t.Fatalf("drainOnce took %v, want bounded by per-attempt publish timeout", elapsed)
	}
	if got := atomic.LoadInt32(&pub.calls); got != 4 {
		t.Fatalf("publish call count = %d, want 4 (2 events × 2 tries)", got)
	}
	if store.markCalls != 0 {
		t.Fatalf("no event should be marked published, got %d mark calls", store.markCalls)
	}
}
