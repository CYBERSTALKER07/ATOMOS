package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/pegasusx/pegasusx/apps/backend-go/workerlease"
)

type failingLeaseManager struct{}

func (failingLeaseManager) TryAcquireOrRenew(context.Context, string, string, time.Duration) (bool, error) {
	return false, errors.New("spanner unavailable")
}

func (failingLeaseManager) Release(context.Context, string, string) error {
	return errors.New("spanner unavailable")
}

func (failingLeaseManager) IsHeld(context.Context, string) (bool, error) {
	return false, errors.New("spanner unavailable")
}

func TestWorkerLive_NilClient_FailsClosed(t *testing.T) {
	if !WorkerLive(context.Background(), nil) {
		t.Fatal("WorkerLive(nil) must fail closed (true) when neither Redis nor Spanner WorkerLeases is available")
	}
}

func TestWorkerLiveWithLease_RedisDownFallsBackToSpannerLease(t *testing.T) {
	ctx := context.Background()
	leaseMgr := workerlease.NewMemoryManager()

	// Redis nil + Spanner lease explicitly unheld -> false.
	if WorkerLiveWithLease(ctx, nil, leaseMgr) {
		t.Fatal("expected false when Redis is nil and Spanner lease is not held")
	}

	// Worker acquires Spanner WorkerLease -> true even without Redis.
	ok, err := leaseMgr.TryAcquireOrRenew(ctx, WorkerHeartbeatLeaseName, "worker-pod-1", 30*time.Second)
	if err != nil || !ok {
		t.Fatalf("acquire heartbeat lease failed: ok=%v err=%v", ok, err)
	}
	if !WorkerLiveWithLease(ctx, nil, leaseMgr) {
		t.Fatal("expected true when Spanner WorkerLeases holds heartbeat")
	}

	// Redis unreachable (closed miniredis) falls back to Spanner WorkerLeases cleanly.
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		if err := rdb.Close(); err != nil {
			t.Logf("close redis client: %v", err)
		}
	})
	mr.Close()

	if !WorkerLiveWithLease(ctx, rdb, leaseMgr) {
		t.Fatal("expected true via Spanner fallback when Redis is unreachable and lease is held")
	}

	if err := leaseMgr.Release(ctx, WorkerHeartbeatLeaseName, "worker-pod-1"); err != nil {
		t.Fatalf("release lease failed: %v", err)
	}
	if WorkerLiveWithLease(ctx, rdb, leaseMgr) {
		t.Fatal("expected false when Redis is unreachable and Spanner explicitly confirms lease unheld")
	}

	// Both Redis down and Spanner failing -> fail closed (true).
	if !WorkerLiveWithLease(ctx, rdb, failingLeaseManager{}) {
		t.Fatal("expected fail closed (true) when both Redis and Spanner fail")
	}
}

func TestWorkerLiveWithLease_RedisHealthyReportsPresence(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		if err := rdb.Close(); err != nil {
			t.Logf("close redis client: %v", err)
		}
	})

	if WorkerLive(ctx, rdb) {
		t.Fatal("expected false when Redis is healthy and key is absent")
	}

	hbCtx, cancel := context.WithCancel(ctx)
	leaseMgr := workerlease.NewMemoryManager()
	StartWorkerHeartbeatWithLease(hbCtx, rdb, leaseMgr, "worker-1", nil)
	if !WorkerLiveWithLease(ctx, rdb, leaseMgr) {
		t.Fatal("expected true after StartWorkerHeartbeatWithLease starts")
	}

	cancel()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !WorkerLiveWithLease(ctx, rdb, leaseMgr) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected heartbeat cleanup after context cancellation")
}

func TestStartWorkerHeartbeat_NilClient_NoOp(t *testing.T) {
	StartWorkerHeartbeat(context.Background(), nil, nil)
}

func TestRedisClientOrNil(t *testing.T) {
	if redisClientOrNil(nil) != nil {
		t.Fatal("redisClientOrNil(nil) must return nil")
	}
}
