package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type mockReplayConn struct {
	id       string
	ident    auth.Claims
	mu       sync.Mutex
	messages [][]byte
}

func (c *mockReplayConn) ID() string {
	return c.id
}
func (c *mockReplayConn) Identity() auth.Claims {
	return c.ident
}
func (c *mockReplayConn) Send(ctx context.Context, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, append([]byte(nil), payload...))
	return nil
}
func (c *mockReplayConn) getMessages() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([][]byte(nil), c.messages...)
}

func TestHub_ReplaySince_RingBuffer(t *testing.T) {
	hub := NewHub("driver", nil, nil)
	ctx := context.Background()

	// Broadcast 5 events to room driver:drv-1
	hub.Broadcast(ctx, "driver:drv-1", []byte(`{"event_id":"evt-1","type":"ROUTE_ASSIGNED","data":"stop 1"}`))
	hub.Broadcast(ctx, "driver:drv-1", []byte(`{"event_id":"evt-2","type":"STOP_ARRIVED","data":"stop 2"}`))
	hub.Broadcast(ctx, "driver:drv-1", []byte(`{"event_id":"evt-3","type":"ORDER_OFFLOADED","data":"order 3"}`))
	hub.Broadcast(ctx, "driver:drv-1", []byte(`{"event_id":"evt-4","type":"ORDER_PAID","data":"order 4"}`))
	hub.Broadcast(ctx, "driver:drv-1", []byte(`{"event_id":"evt-5","type":"ROUTE_COMPLETED","data":"done"}`))

	// Connect a reconnected driver who last saw evt-3
	conn := &mockReplayConn{
		id:    "conn-1",
		ident: auth.Claims{Role: auth.RoleDriver, Subject: "drv-1"},
	}

	replayed := hub.ReplaySince(ctx, "driver:drv-1", 0, "evt-3", conn)
	if replayed != 2 {
		t.Fatalf("expected 2 replayed events after evt-3, got %d", replayed)
	}

	msgs := conn.getMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages received, got %d", len(msgs))
	}
	if !strings.Contains(string(msgs[0]), "evt-4") {
		t.Fatalf("expected first replayed message to be evt-4, got %s", string(msgs[0]))
	}
	if !strings.Contains(string(msgs[1]), "evt-5") {
		t.Fatalf("expected second replayed message to be evt-5, got %s", string(msgs[1]))
	}
}

func TestReplayMissedEvents_InspectsHeadersAndParams(t *testing.T) {
	driverHub := NewHub("driver", nil, nil)
	hubs := roleHubs{driver: driverHub}
	ctx := context.Background()

	driverHub.Broadcast(ctx, "driver:drv-99", []byte(`{"event_id":"evt-10","type":"PING"}`))
	driverHub.Broadcast(ctx, "driver:drv-99", []byte(`{"event_id":"evt-11","type":"ROUTE_UPDATE"}`))
	driverHub.Broadcast(ctx, "driver:drv-99", []byte(`{"event_id":"evt-12","type":"ROUTE_ALERT"}`))

	ident := auth.Claims{Role: auth.RoleDriver, Subject: "drv-99"}
	conn := &mockReplayConn{id: "conn-99", ident: ident}

	// Test reconnect with Last-Event-ID header
	req := httptest.NewRequest(http.MethodGet, "/v1/ws", nil)
	req.Header.Set("Last-Event-ID", "evt-10")

	replayMissedEvents(req, ident, conn, hubs, RegisterConfig{})

	msgs := conn.getMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 replayed messages (evt-11 and evt-12), got %d", len(msgs))
	}
	if !strings.Contains(string(msgs[0]), "evt-11") {
		t.Fatalf("expected evt-11, got %s", string(msgs[0]))
	}
	if !strings.Contains(string(msgs[1]), "evt-12") {
		t.Fatalf("expected evt-12, got %s", string(msgs[1]))
	}
}
