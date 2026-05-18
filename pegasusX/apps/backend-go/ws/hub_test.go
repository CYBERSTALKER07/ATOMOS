package ws

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

type testConnection struct {
	id     string
	claims auth.Claims
	fail   bool
	sent   chan []byte
}

func newTestConnection(id string) *testConnection {
	return &testConnection{id: id, sent: make(chan []byte, 8)}
}

func (c *testConnection) ID() string { return c.id }

func (c *testConnection) Identity() auth.Claims { return c.claims }

func (c *testConnection) Send(_ context.Context, payload []byte) error {
	if c.fail {
		return errors.New("send failed")
	}
	copyPayload := append([]byte(nil), payload...)
	c.sent <- copyPayload
	return nil
}

type publishFailBackend struct{}

func (b *publishFailBackend) Get(_ context.Context, _ string) ([]byte, bool, error) {
	return nil, false, nil
}
func (b *publishFailBackend) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}
func (b *publishFailBackend) Delete(_ context.Context, _ ...string) error { return nil }
func (b *publishFailBackend) Publish(_ context.Context, _ string, _ []byte) error {
	return errors.New("relay publish failure")
}
func (b *publishFailBackend) Subscribe(_ context.Context, _ string) (<-chan []byte, func(), error) {
	ch := make(chan []byte)
	cancel := func() { close(ch) }
	return ch, cancel, nil
}

func expectPayload(t *testing.T, ch <-chan []byte, want string) {
	t.Helper()
	select {
	case got := <-ch:
		if string(got) != want {
			t.Fatalf("payload = %q, want %q", string(got), want)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for payload %q", want)
	}
}

func TestBroadcastFailOpenOnPublishErrorStillFansOutLocal(t *testing.T) {
	t.Parallel()

	hub := NewHub("retailer", &publishFailBackend{}, nil)
	conn := newTestConnection("c1")
	hub.Subscribe("room:1", conn)

	hub.Broadcast(context.Background(), "room:1", []byte("hello"))

	expectPayload(t, conn.sent, "hello")
	if stats := hub.Stats(); stats.PubFailures != 1 {
		t.Fatalf("pub failure count = %d, want 1", stats.PubFailures)
	}
}

func TestBroadcastReapsDeadConnections(t *testing.T) {
	t.Parallel()

	hub := NewHub("retailer", cache.NewInMemoryBackend(), nil)
	alive := newTestConnection("alive")
	dead := newTestConnection("dead")
	dead.fail = true

	hub.Subscribe("room:2", alive)
	hub.Subscribe("room:2", dead)

	hub.Broadcast(context.Background(), "room:2", []byte("event"))

	expectPayload(t, alive.sent, "event")
	if stats := hub.Stats(); stats.Connections != 1 {
		t.Fatalf("connections = %d, want 1", stats.Connections)
	}
}

func TestStartRelaySubscriberFansOutPeerEventsAndSuppressesSelf(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	backend := cache.NewInMemoryBackend()
	hub := NewHub("retailer", backend, nil)
	conn := newTestConnection("c1")
	hub.Subscribe("room:3", conn)

	go hub.StartRelaySubscriber(ctx)

	peerEnvelope, err := json.Marshal(relayEnvelope{Source: "peer-instance", Room: "room:3", Payload: []byte("peer-event")})
	if err != nil {
		t.Fatalf("marshal peer envelope: %v", err)
	}
	if err := backend.Publish(ctx, hub.relayChannel(), peerEnvelope); err != nil {
		t.Fatalf("publish peer envelope: %v", err)
	}

	expectPayload(t, conn.sent, "peer-event")

	selfEnvelope, err := json.Marshal(relayEnvelope{Source: hub.instance, Room: "room:3", Payload: []byte("self-event")})
	if err != nil {
		t.Fatalf("marshal self envelope: %v", err)
	}
	if err := backend.Publish(ctx, hub.relayChannel(), selfEnvelope); err != nil {
		t.Fatalf("publish self envelope: %v", err)
	}

	select {
	case got := <-conn.sent:
		t.Fatalf("expected self-source suppression, got payload %q", string(got))
	case <-time.After(200 * time.Millisecond):
	}
}
