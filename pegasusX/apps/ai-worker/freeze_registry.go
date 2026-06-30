package main

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

const defaultFreezeLockTTL = 5 * time.Minute

// freezeRegistry tracks entities the AI worker must not auto-touch until TTL expiry.
type freezeRegistry struct {
	mu    sync.RWMutex
	until map[string]time.Time
}

func newFreezeRegistry() *freezeRegistry {
	return &freezeRegistry{until: make(map[string]time.Time)}
}

func (r *freezeRegistry) RunCleanup(ctx context.Context) {
	if r == nil {
		return
	}
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			r.mu.Lock()
			for k, exp := range r.until {
				if now.After(exp) {
					delete(r.until, k)
				}
			}
			r.mu.Unlock()
		}
	}
}

func (r *freezeRegistry) isFrozen(entityType, entityID string) bool {
	if r == nil {
		return false
	}
	entityType = strings.TrimSpace(entityType)
	entityID = strings.TrimSpace(entityID)
	if entityType == "" || entityID == "" {
		return false
	}
	key := entityType + ":" + entityID
	now := time.Now()
	r.mu.RLock()
	exp, ok := r.until[key]
	r.mu.RUnlock()
	return ok && now.Before(exp)
}

func (r *freezeRegistry) applyPayload(payload []byte) {
	var evt events.DispatchLockEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return
	}
	r.applyEvent(evt)
}

func (r *freezeRegistry) applyEvent(evt events.DispatchLockEvent) {
	if r == nil {
		return
	}
	keys := freezeKeysFromEvent(evt)
	if len(keys) == 0 {
		return
	}
	switch strings.TrimSpace(evt.Type) {
	case events.EventFreezeLockAcquired:
		ttl := time.Duration(evt.TTLSeconds) * time.Second
		if ttl <= 0 {
			ttl = defaultFreezeLockTTL
		}
		exp := time.Now().Add(ttl)
		r.mu.Lock()
		for _, key := range keys {
			r.until[key] = exp
		}
		r.mu.Unlock()
	case events.EventFreezeLockReleased:
		r.mu.Lock()
		for _, key := range keys {
			delete(r.until, key)
		}
		r.mu.Unlock()
	}
}

func freezeKeysFromEvent(evt events.DispatchLockEvent) []string {
	keys := make([]string, 0, 3)
	if entityType := strings.TrimSpace(evt.EntityType); entityType != "" {
		if entityID := strings.TrimSpace(evt.EntityID); entityID != "" {
			keys = append(keys, entityType+":"+entityID)
		}
	}
	if warehouseID := strings.TrimSpace(evt.WarehouseID); warehouseID != "" {
		keys = append(keys, "WAREHOUSE:"+warehouseID)
	}
	if factoryID := strings.TrimSpace(evt.FactoryID); factoryID != "" {
		keys = append(keys, "FACTORY:"+factoryID)
	}
	return keys
}
