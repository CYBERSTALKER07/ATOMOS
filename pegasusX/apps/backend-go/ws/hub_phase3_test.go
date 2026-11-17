package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

type slowConnection struct {
	id       string
	claims   auth.Claims
	delay    time.Duration
	received atomic.Int64
}

func (c *slowConnection) ID() string             { return c.id }
func (c *slowConnection) Identity() auth.Claims { return c.claims }
func (c *slowConnection) Send(ctx context.Context, payload []byte) error {
	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	c.received.Add(1)
	return nil
}

func TestHub_GloballyUniqueEventID_FormatAndDecoupledSeq(t *testing.T) {
	backend := cache.NewInMemoryBackend()
	hubA := NewHub("driver", backend, nil)
	hubB := NewHub("driver", backend, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hubB.StartRelaySubscriber(ctx)
	time.Sleep(20 * time.Millisecond)

	connB := newTestConnection("client-b")
	hubB.Subscribe("driver:drv-1", connB)

	// Broadcast from Hub A without event_id
	payload := []byte(`{"type":"ORDER_UPDATED","order_id":"ord-1"}`)
	hubA.Broadcast(ctx, "driver:drv-1", payload)

	// Hub B should receive and fan out to connB
	select {
	case received := <-connB.sent:
		var parsed map[string]any
		if err := json.Unmarshal(received, &parsed); err != nil {
			t.Fatalf("unmarshal received message: %v", err)
		}

		eventID, ok := parsed["event_id"].(string)
		if !ok || eventID == "" {
			t.Fatalf("expected non-empty event_id in %s", string(received))
		}

		// Verify eventID has format <podID>-<nanos>-<randHex>
		parts := strings.Split(eventID, "-")
		if len(parts) < 3 {
			t.Fatalf("event_id %q does not match podID-nanos-randHex format", eventID)
		}

		// Verify seq was assigned locally on Hub B
		seqVal, hasSeq := parsed["seq"]
		if !hasSeq {
			t.Fatalf("expected local seq on Hub B, got %v", parsed)
		}
		if seqFloat, ok := seqVal.(float64); !ok || int64(seqFloat) <= 0 {
			t.Fatalf("expected positive local seq on Hub B, got %v", seqVal)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event on Hub B")
	}
}

func TestHub_BoundedRoomHistory_CappedAt2048(t *testing.T) {
	hub := NewHub("test", nil, nil)

	// Fill 2050 rooms
	const totalRooms = 2050
	for i := 0; i < totalRooms; i++ {
		room := fmt.Sprintf("room-%d", i)
		hub.recordHistory(room, []byte(`{"event_id":"evt","msg":"hello"}`), "evt")
	}

	hub.historyMu.RLock()
	roomCount := len(hub.history)
	hub.historyMu.RUnlock()

	if roomCount > maxHistoryRooms {
		t.Fatalf("history rooms = %d, expected cap at %d", roomCount, maxHistoryRooms)
	}
}

func TestHub_ReplaySince_WSResyncRequired_WhenUnknownLastEventID(t *testing.T) {
	hub := NewHub("retailer", nil, nil)
	ctx := context.Background()

	// Fill room with events evt-10, evt-11
	hub.Broadcast(ctx, "retailer:ret-1", []byte(`{"event_id":"evt-10","data":"10"}`))
	hub.Broadcast(ctx, "retailer:ret-1", []byte(`{"event_id":"evt-11","data":"11"}`))

	conn := &mockReplayConn{
		id:    "conn-recon",
		ident: auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"},
	}

	// Reconnect with unknown last_event_id (e.g. from an ancient session)
	replayed := hub.ReplaySince(ctx, "retailer:ret-1", 0, "unknown-ancient-evt", conn)
	if replayed != 0 {
		t.Fatalf("expected 0 replayed events, got %d", replayed)
	}

	msgs := conn.getMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message (WS_RESYNC_REQUIRED), got %d", len(msgs))
	}

	var resync map[string]string
	if err := json.Unmarshal(msgs[0], &resync); err != nil {
		t.Fatalf("unmarshal resync message: %v", err)
	}
	if resync["type"] != "WS_RESYNC_REQUIRED" {
		t.Fatalf("expected type WS_RESYNC_REQUIRED, got %q", resync["type"])
	}
	if resync["room"] != "retailer:ret-1" {
		t.Fatalf("expected room retailer:ret-1, got %q", resync["room"])
	}
	if resync["last_event_id"] != "unknown-ancient-evt" {
		t.Fatalf("expected last_event_id unknown-ancient-evt, got %q", resync["last_event_id"])
	}
}

func TestHub_ReplaySince_WSResyncRequired_WhenBufferEmpty(t *testing.T) {
	hub := NewHub("warehouse", nil, nil)
	ctx := context.Background()

	conn := &mockReplayConn{
		id:    "conn-wh",
		ident: auth.Claims{Role: auth.RoleWarehouse, Subject: "wh-1"},
	}

	// Reconnect to a room with no history in the buffer
	replayed := hub.ReplaySince(ctx, "warehouse:wh-1", 0, "evt-last", conn)
	if replayed != 0 {
		t.Fatalf("expected 0 replayed events, got %d", replayed)
	}

	msgs := conn.getMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var resync map[string]string
	if err := json.Unmarshal(msgs[0], &resync); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resync["type"] != "WS_RESYNC_REQUIRED" {
		t.Fatalf("expected WS_RESYNC_REQUIRED, got %q", resync["type"])
	}
	if resync["last_event_id"] != "evt-last" {
		t.Fatalf("expected last_event_id evt-last, got %q", resync["last_event_id"])
	}
}

func TestHub_DecoupledCrossPodFanout_SlowWorkerDoesNotBlockRedis(t *testing.T) {
	backend := cache.NewInMemoryBackend()
	hub := NewHub("telemetry", backend, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Room 1 has a very slow client (delay 500ms)
	slowConn := &slowConnection{id: "slow-c", delay: 500 * time.Millisecond}
	hub.Subscribe("telemetry:room-slow", slowConn)

	// Room 2 has a fast client
	fastConn := newTestConnection("fast-c")
	hub.Subscribe("telemetry:room-fast", fastConn)

	go hub.StartRelaySubscriber(ctx)
	time.Sleep(20 * time.Millisecond)

	// Publish to room-slow via Redis channel
	envSlow, _ := json.Marshal(relayEnvelope{
		EventID: "evt-slow",
		Source:  "other-instance",
		Room:    "telemetry:room-slow",
		Payload: []byte(`{"type":"SLOW"}`),
	})
	_ = backend.Publish(ctx, hub.relayChannel(), envSlow)

	// Immediately publish to room-fast via Redis channel
	envFast, _ := json.Marshal(relayEnvelope{
		EventID: "evt-fast",
		Source:  "other-instance",
		Room:    "telemetry:room-fast",
		Payload: []byte(`{"type":"FAST"}`),
	})
	_ = backend.Publish(ctx, hub.relayChannel(), envFast)

	// Fast connection MUST receive its message quickly without waiting 500ms for slowConn!
	select {
	case received := <-fastConn.sent:
		if !strings.Contains(string(received), "FAST") {
			t.Fatalf("expected FAST payload, got %s", string(received))
		}
	case <-time.After(150 * time.Millisecond):
		t.Fatal("fast room delivery was blocked by slow room worker!")
	}
}
