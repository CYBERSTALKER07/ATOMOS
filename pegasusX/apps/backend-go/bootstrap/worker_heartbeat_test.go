package bootstrap

import (
	"context"
	"testing"
)

// The api-tier safety net must fail OPEN when Redis is unavailable: better to
// risk a duplicate push in a degraded dev setup than to silently drop push in a
// single-tier api deployment with no reachable Redis.
func TestWorkerLive_NilClient_FailsOpen(t *testing.T) {
	if WorkerLive(context.Background(), nil) {
		t.Fatal("WorkerLive(nil) must be false so the api-tier consumer starts")
	}
}

// Heartbeat with a nil client must be a safe no-op (no panic, no goroutine leak
// blocking on a nil client).
func TestStartWorkerHeartbeat_NilClient_NoOp(t *testing.T) {
	StartWorkerHeartbeat(context.Background(), nil, nil)
}

func TestRedisClientOrNil(t *testing.T) {
	if redisClientOrNil(nil) != nil {
		t.Fatal("redisClientOrNil(nil) must return nil")
	}
}
