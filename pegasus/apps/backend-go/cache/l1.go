// Package cache — in-process L1 cache.
//
// A sync.Map-backed layer that absorbs reads before they touch Redis (L2).
// Entries carry an explicit expiry so expired items are evicted lazily on
// the next read. No background sweeper — at typical key counts (< 10 k
// hot keys) lazy eviction keeps memory bounded without added goroutines.
//
// Invalidation: Invalidate() calls l1Evict so both L1 and L2 are purged
// atomically from the caller's perspective. Cross-pod invalidations arrive
// via the Pub/Sub channel and are dispatched through OnInvalidate hooks,
// which also call l1Evict on the receiving pod.
package cache

import (
	"sync"
	"time"
)

type l1Entry struct {
	val string
	exp time.Time
}

// l1store is the process-level in-memory cache.
var l1store sync.Map // string → *l1Entry

// l1Get returns the cached value and true when the key is present and not expired.
// Expired entries are deleted lazily before returning false.
func l1Get(key string) (string, bool) {
	v, ok := l1store.Load(key)
	if !ok {
		return "", false
	}
	e := v.(*l1Entry)
	if time.Now().After(e.exp) {
		l1store.Delete(key)
		return "", false
	}
	return e.val, true
}

// l1Set stores val in the in-process L1 cache with the given TTL.
// A zero or negative TTL is a no-op (ephemeral writes must not pollute L1).
func l1Set(key, val string, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	l1store.Store(key, &l1Entry{val: val, exp: time.Now().Add(ttl)})
}

// l1Evict removes the given keys from L1. Called by Invalidate and the
// cross-pod Pub/Sub subscriber so all pods stay coherent.
func l1Evict(keys []string) {
	for _, k := range keys {
		l1store.Delete(k)
	}
}
