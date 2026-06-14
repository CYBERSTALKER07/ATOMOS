package main

import (
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestFreezeRegistry_AcquireAndRelease(t *testing.T) {
	reg := newFreezeRegistry()
	reg.applyEvent(events.DispatchLockEvent{
		BaseEvent:   events.BaseEvent{Type: events.EventFreezeLockAcquired},
		WarehouseID: "wh-1",
		EntityType:  "ORDER",
		EntityID:    "ord-1",
		TTLSeconds:  60,
	})
	if !reg.isFrozen("WAREHOUSE", "wh-1") {
		t.Fatal("expected warehouse frozen")
	}
	if !reg.isFrozen("ORDER", "ord-1") {
		t.Fatal("expected order frozen")
	}

	reg.applyEvent(events.DispatchLockEvent{
		BaseEvent:   events.BaseEvent{Type: events.EventFreezeLockReleased},
		WarehouseID: "wh-1",
		EntityType:  "ORDER",
		EntityID:    "ord-1",
	})
	if reg.isFrozen("WAREHOUSE", "wh-1") || reg.isFrozen("ORDER", "ord-1") {
		t.Fatal("expected locks released")
	}
}

func TestFreezeRegistry_ExpiresAfterTTL(t *testing.T) {
	reg := newFreezeRegistry()
	reg.applyEvent(events.DispatchLockEvent{
		BaseEvent:   events.BaseEvent{Type: events.EventFreezeLockAcquired},
		EntityType:  "ORDER",
		EntityID:    "ord-expire",
		TTLSeconds:  0,
	})
	reg.mu.Lock()
	for key := range reg.until {
		reg.until[key] = time.Now().Add(-time.Second)
	}
	reg.mu.Unlock()
	if reg.isFrozen("ORDER", "ord-expire") {
		t.Fatal("expected expired lock to be inactive")
	}
}
