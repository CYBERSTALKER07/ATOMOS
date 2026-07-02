package outbox

import (
	"context"
	"errors"
	"reflect"
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
