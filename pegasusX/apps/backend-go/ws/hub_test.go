package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
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

type benchmarkConnection struct {
	id string
}

func (c *benchmarkConnection) ID() string { return c.id }

func (c *benchmarkConnection) Identity() auth.Claims { return auth.Claims{} }

func (c *benchmarkConnection) Send(_ context.Context, _ []byte) error { return nil }

type countingConnection struct {
	id    string
	count atomic.Int64
}

func (c *countingConnection) ID() string { return c.id }

func (c *countingConnection) Identity() auth.Claims { return auth.Claims{} }

func (c *countingConnection) Send(_ context.Context, _ []byte) error {
	c.count.Add(1)
	return nil
}

func (c *countingConnection) Delivered() int64 { return c.count.Load() }

func waitForRelayReady(t *testing.T, ctx context.Context, publisher *Hub, room string, conn *countingConnection) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		before := conn.Delivered()
		publisher.Broadcast(ctx, room, []byte("relay-ready"))
		probeDeadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(probeDeadline) {
			if conn.Delivered() > before {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	t.Fatal("relay subscriber not ready")
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
	time.Sleep(20 * time.Millisecond) // Wait for subscriber to attach

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

func TestStartRelaySubscriberDeliversBurstIntegrity(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	backend := cache.NewInMemoryBackend()
	publisherHub := NewHub("telemetry", backend, nil)
	receiverHub := NewHub("telemetry", backend, nil)
	room := "telemetry:supplier:sup-1"
	conn := &countingConnection{id: "counting"}
	receiverHub.Subscribe(room, conn)

	go receiverHub.StartRelaySubscriber(ctx)
	time.Sleep(20 * time.Millisecond)
	waitForRelayReady(t, ctx, publisherHub, room, conn)

	const burst = 32
	for i := 0; i < burst; i++ {
		payload := []byte(fmt.Sprintf("relay-burst-%d", i))
		publisherHub.Broadcast(ctx, room, payload)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if conn.Delivered() >= burst+1 { // +1 for relay-ready probe
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("delivered=%d want=%d", conn.Delivered(), burst+1)
}

func BenchmarkHubBroadcastTelemetryFanout(b *testing.B) {
	payload := []byte(`{"type":"DRIVER_LOCATION_UPDATED","trace_id":"bench-trace","data":{"driver_id":"drv-1","supplier_id":"sup-1","lat":41.3,"lng":69.2}}`)

	for _, subscribers := range []int{1, 10, 100, 500} {
		b.Run(fmt.Sprintf("subscribers_%d", subscribers), func(b *testing.B) {
			hub := NewHub("telemetry", nil, nil)
			room := "telemetry:supplier:sup-1"
			for i := 0; i < subscribers; i++ {
				hub.Subscribe(room, &benchmarkConnection{id: fmt.Sprintf("c-%d", i)})
			}

			ctx := context.Background()
			b.ReportAllocs()
			b.SetBytes(int64(len(payload)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				hub.Broadcast(ctx, room, payload)
			}
		})
	}
}

func BenchmarkHubBroadcastTelemetryFanoutRelay(b *testing.B) {
	payload := []byte(`{"type":"DRIVER_LOCATION_UPDATED","trace_id":"bench-trace","data":{"driver_id":"drv-1","supplier_id":"sup-1","lat":41.3,"lng":69.2}}`)

	for _, subscribers := range []int{1, 10, 100, 500} {
		b.Run(fmt.Sprintf("relay_subscribers_%d", subscribers), func(b *testing.B) {
			backend := cache.NewInMemoryBackend()
			publisherHub := NewHub("telemetry", backend, nil)
			receiverHub := NewHub("telemetry", backend, nil)
			room := "telemetry:supplier:sup-1"
			for i := 0; i < subscribers; i++ {
				receiverHub.Subscribe(room, &benchmarkConnection{id: fmt.Sprintf("relay-c-%d", i)})
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go receiverHub.StartRelaySubscriber(ctx)
			time.Sleep(10 * time.Millisecond)

			b.ReportAllocs()
			b.SetBytes(int64(len(payload)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				publisherHub.Broadcast(ctx, room, payload)
			}
		})
	}
}
