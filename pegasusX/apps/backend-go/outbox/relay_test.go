package outbox

import (
	"context"
	"errors"
	"reflect"
	"sync"
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

	recordFailuresCalls      int
	recordFailureIDs         []string
	recordFailureLastErr     string
	recordFailureMaxAttempts int64
	mockDeadLettered         []string
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

func (s *relayTestStore) RecordPublishFailures(_ context.Context, eventIDs []string, lastErr string, maxAttempts int64) ([]string, error) {
	s.recordFailuresCalls++
	s.recordFailureIDs = append([]string(nil), eventIDs...)
	s.recordFailureLastErr = lastErr
	s.recordFailureMaxAttempts = maxAttempts
	return s.mockDeadLettered, nil
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

func TestRelayDrainOnce_RecordsPublishFailuresAndDeadLetters(t *testing.T) {
	t.Parallel()

	store := &relayTestStore{
		events: []Event{
			{EventID: "fail-1", AggregateID: "a-fail", TopicName: "t-fail", Payload: []byte("p")},
		},
		mockDeadLettered: []string{"fail-1"},
	}
	pub := &relayTestPublisher{
		errorsByCall: []error{errors.New("pub-err-1"), errors.New("pub-err-2")},
	}
	relay := NewRelay(store, pub, RelayConfig{
		MaxPublishTries:  2,
		BaseBackoff:      time.Millisecond,
		MaxBackoff:       2 * time.Millisecond,
		MaxTotalAttempts: 5,
	}, nil)

	relay.drainOnce(context.Background())

	if store.recordFailuresCalls != 1 {
		t.Fatalf("recordFailuresCalls = %d, want 1", store.recordFailuresCalls)
	}
	if !reflect.DeepEqual(store.recordFailureIDs, []string{"fail-1"}) {
		t.Fatalf("recordFailureIDs = %v, want [fail-1]", store.recordFailureIDs)
	}
	if store.recordFailureMaxAttempts != 5 {
		t.Fatalf("recordFailureMaxAttempts = %d, want 5", store.recordFailureMaxAttempts)
	}
	if store.recordFailureLastErr != "pub-err-2" {
		t.Fatalf("recordFailureLastErr = %q, want 'pub-err-2'", store.recordFailureLastErr)
	}
}

type statefulOutboxStore struct {
	mu           sync.Mutex
	unpublished  map[string]Event
	attempts     map[string]int64
	deadLettered map[string]bool
	published    map[string]bool
}

func newTestStatefulStore(events ...Event) *statefulOutboxStore {
	s := &statefulOutboxStore{
		unpublished:  make(map[string]Event),
		attempts:     make(map[string]int64),
		deadLettered: make(map[string]bool),
		published:    make(map[string]bool),
	}
	for _, e := range events {
		s.unpublished[e.EventID] = e
	}
	return s
}

func (s *statefulOutboxStore) Add(events ...Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range events {
		s.unpublished[e.EventID] = e
	}
}

func (s *statefulOutboxStore) Fetch(_ context.Context, limit int) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := make([]Event, 0, len(s.unpublished))
	for _, e := range s.unpublished {
		res = append(res, e)
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}

func (s *statefulOutboxStore) MarkPublished(_ context.Context, eventIDs []string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range eventIDs {
		s.published[id] = true
		delete(s.unpublished, id)
	}
	return nil
}

func (s *statefulOutboxStore) CountUnpublished(_ context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return int64(len(s.unpublished)), nil
}

func (s *statefulOutboxStore) RecordPublishFailures(_ context.Context, eventIDs []string, _ string, maxAttempts int64) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var dled []string
	for _, id := range eventIDs {
		s.attempts[id]++
		if s.attempts[id] >= maxAttempts {
			s.deadLettered[id] = true
			delete(s.unpublished, id)
			dled = append(dled, id)
		}
	}
	return dled, nil
}

type selectiveErrorPublisher struct {
	failingAggregateID string
	err                error
}

func (p *selectiveErrorPublisher) Publish(_ context.Context, _ string, key []byte, _ []byte) error {
	if string(key) == p.failingAggregateID {
		return p.err
	}
	return nil
}

// TestRelayDrainOnce_PoisonPillIsolation_MultiTickQuarantine validates that:
// 1. A poisoned event that repeatedly fails Kafka produce attempts never blocks healthy events in the same or future batches.
// 2. The poisoned event accumulates attempts across ticks.
// 3. Once MaxTotalAttempts is exhausted, the event is atomically quarantined into OutboxDeadLetters and purged from unpublished.
// 4. Healthy events continue streaming without head-of-line blocking or relay stalls.
func TestRelayDrainOnce_PoisonPillIsolation_MultiTickQuarantine(t *testing.T) {
	t.Parallel()

	poisonEvent := Event{
		EventID:       "evt-poison-666",
		AggregateType: "Order",
		AggregateID:   "bad-payload-order",
		TopicName:     "orders.v1",
		Payload:       []byte(`{"corrupt": true}`),
	}
	healthyEvent1 := Event{
		EventID:       "evt-healthy-1",
		AggregateType: "Order",
		AggregateID:   "good-order-1",
		TopicName:     "orders.v1",
		Payload:       []byte(`{"order_id": "good-1"}`),
	}

	store := newTestStatefulStore(poisonEvent, healthyEvent1)
	pub := &selectiveErrorPublisher{
		failingAggregateID: "bad-payload-order",
		err:                errors.New("kafka: schema validation rejected message"),
	}

	relay := NewRelay(store, pub, RelayConfig{
		MaxPublishTries:  2,
		BaseBackoff:      time.Millisecond,
		MaxBackoff:       2 * time.Millisecond,
		MaxTotalAttempts: 3, // Poison pill will be dead-lettered after 3 failed ticks
	}, nil)

	ctx := context.Background()

	// --- Tick 1: Mixed batch [poison, healthy1] ---
	relay.drainOnce(ctx)

	store.mu.Lock()
	if !store.published["evt-healthy-1"] {
		t.Fatal("healthy event 1 should be marked published even though poison event failed")
	}
	if store.attempts["evt-poison-666"] != 1 {
		t.Fatalf("poison event attempt count = %d, want 1", store.attempts["evt-poison-666"])
	}
	if store.deadLettered["evt-poison-666"] {
		t.Fatal("poison event should not be dead-lettered yet after 1 tick")
	}
	store.mu.Unlock()

	// --- Tick 2: Enqueue healthy event 2 alongside the retry of poison event ---
	healthyEvent2 := Event{
		EventID:       "evt-healthy-2",
		AggregateType: "Order",
		AggregateID:   "good-order-2",
		TopicName:     "orders.v1",
		Payload:       []byte(`{"order_id": "good-2"}`),
	}
	store.Add(healthyEvent2)

	relay.drainOnce(ctx)

	store.mu.Lock()
	if !store.published["evt-healthy-2"] {
		t.Fatal("healthy event 2 should be marked published on tick 2")
	}
	if store.attempts["evt-poison-666"] != 2 {
		t.Fatalf("poison event attempt count = %d, want 2", store.attempts["evt-poison-666"])
	}
	if store.deadLettered["evt-poison-666"] {
		t.Fatal("poison event should not be dead-lettered yet after 2 ticks")
	}
	store.mu.Unlock()

	// --- Tick 3: Enqueue healthy event 3; poison event reaches MaxTotalAttempts (3) ---
	healthyEvent3 := Event{
		EventID:       "evt-healthy-3",
		AggregateType: "Order",
		AggregateID:   "good-order-3",
		TopicName:     "orders.v1",
		Payload:       []byte(`{"order_id": "good-3"}`),
	}
	store.Add(healthyEvent3)

	relay.drainOnce(ctx)

	store.mu.Lock()
	if !store.published["evt-healthy-3"] {
		t.Fatal("healthy event 3 should be marked published on tick 3")
	}
	if store.attempts["evt-poison-666"] != 3 {
		t.Fatalf("poison event attempt count = %d, want 3", store.attempts["evt-poison-666"])
	}
	if !store.deadLettered["evt-poison-666"] {
		t.Fatal("poison event MUST be dead-lettered after reaching MaxTotalAttempts (3)")
	}
	if _, stillUnpublished := store.unpublished["evt-poison-666"]; stillUnpublished {
		t.Fatal("poison event must be purged from unpublished store upon quarantine")
	}
	remainingCount := len(store.unpublished)
	store.mu.Unlock()

	if remainingCount != 0 {
		t.Fatalf("unpublished queue should be completely empty, got count = %d", remainingCount)
	}

	// --- Tick 4: Post-quarantine drain confirms system is idle and stable ---
	relay.drainOnce(ctx)
}

