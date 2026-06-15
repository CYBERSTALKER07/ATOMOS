package ws

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type reapTrackingConn struct {
	id     string
	reaped bool
	sent   chan []byte
}

func (c *reapTrackingConn) ID() string { return c.id }

func (c *reapTrackingConn) Identity() auth.Claims { return auth.Claims{} }

func (c *reapTrackingConn) Send(_ context.Context, payload []byte) error {
	c.sent <- append([]byte(nil), payload...)
	return nil
}

func (c *reapTrackingConn) Reap() { c.reaped = true }

func TestSubscribeShedsOldestWhenRoomExceedsLimit(t *testing.T) {
	t.Parallel()

	hub := NewHubWithLimits("driver", nil, nil, HubLimits{MaxPerRoom: 2, MaxTotal: 0})
	c1 := &reapTrackingConn{id: "c1", sent: make(chan []byte, 1)}
	c2 := &reapTrackingConn{id: "c2", sent: make(chan []byte, 1)}
	c3 := &reapTrackingConn{id: "c3", sent: make(chan []byte, 1)}

	hub.Subscribe("driver:drv-1", c1)
	time.Sleep(2 * time.Millisecond)
	hub.Subscribe("driver:drv-1", c2)
	time.Sleep(2 * time.Millisecond)
	hub.Subscribe("driver:drv-1", c3)

	if !c1.reaped {
		t.Fatal("expected oldest connection c1 to be shed")
	}
	if c2.reaped || c3.reaped {
		t.Fatalf("expected c2/c3 to remain; c2=%v c3=%v", c2.reaped, c3.reaped)
	}
	if stats := hub.Stats(); stats.Connections != 2 || stats.ShedCount != 1 {
		t.Fatalf("stats=%+v want connections=2 shed=1", stats)
	}
}

func TestHasCapacityRespectsMaxTotal(t *testing.T) {
	t.Parallel()

	hub := NewHubWithLimits("retailer", nil, nil, HubLimits{MaxPerRoom: 0, MaxTotal: 2})
	if !hub.HasCapacity() {
		t.Fatal("empty hub should have capacity")
	}
	hub.Subscribe("retailer:r1", &reapTrackingConn{id: "a", sent: make(chan []byte, 1)})
	hub.Subscribe("retailer:r2", &reapTrackingConn{id: "b", sent: make(chan []byte, 1)})
	if hub.HasCapacity() {
		t.Fatal("hub at max total should reject new connections")
	}
}

func TestReconnectDelayWithJitterWithinBounds(t *testing.T) {
	t.Parallel()

	for attempt := 0; attempt < 8; attempt++ {
		delay := ReconnectDelay(attempt, 2*time.Second, 60*time.Second)
		base := minDuration(2*time.Second<<minInt(attempt, 10), 60*time.Second)
		if delay < base || delay > base+base/2 {
			t.Fatalf("attempt=%d delay=%s base=%s out of jitter range", attempt, delay, base)
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func TestReconnectDelayUsesDeterministicRandWhenInjected(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(42))
	delay := ReconnectDelayWithRand(3, 2*time.Second, 60*time.Second, rng)
	if delay <= 0 {
		t.Fatalf("delay=%s", delay)
	}
}

func TestReconnectDelayWithRetryAfterHonorsServerHint(t *testing.T) {
	t.Parallel()

	backoff := ReconnectDelayWithRetryAfter(0, 2*time.Second, 60*time.Second, 0)
	hinted := ReconnectDelayWithRetryAfter(0, 2*time.Second, 60*time.Second, 45*time.Second)
	if hinted < 45*time.Second {
		t.Fatalf("hinted=%s want >= 45s (backoff alone was %s)", hinted, backoff)
	}
}

func TestParseRetryAfterSeconds(t *testing.T) {
	t.Parallel()

	if sec, ok := ParseRetryAfterSeconds("30"); !ok || sec != 30 {
		t.Fatalf("parse 30 = %d %v", sec, ok)
	}
	if _, ok := ParseRetryAfterSeconds(""); ok {
		t.Fatal("empty should not parse")
	}
}
