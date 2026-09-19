package bootstrap

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Worker-liveness gate (P1-9 / run-mode parity).
//
// Production runs two tiers from the same binary: the public API Deployment with
// PEGASUSX_RUN_MODE=api and a worker Deployment with PEGASUSX_RUN_MODE=worker
// that owns the outbox relay + Kafka consumers (where FCM push and notification
// inbox persistence happen). In that shape nothing is lost.
//
// The gap is a single-tier RUN_MODE=api with no worker (local/dev, or a
// misconfigured deploy): notification consumers never start, so FCM push and
// the polling inbox silently die while WebSocket keeps working. To close that
// without double-firing alongside a live worker, the worker tier publishes a
// short-lived Redis heartbeat; an api-tier starts its own notification consumer
// only when no live heartbeat is present.

const (
	// workerHeartbeatKey is the Redis key the worker tier refreshes.
	workerHeartbeatKey = "pegasusx:runtime:worker:heartbeat"
	// WorkerHeartbeatTTL is how long a heartbeat stays valid after the worker
	// dies. Slightly more than two refresh intervals so one missed tick does
	// not flap the api-tier consumer.
	WorkerHeartbeatTTL = 45 * time.Second
	// WorkerHeartbeatInterval is how often the worker refreshes its heartbeat.
	WorkerHeartbeatInterval = 15 * time.Second
)

func redisClientOrNil(adapter redisRuntimeAdapter) *redis.Client {
	if adapter == nil {
		return nil
	}
	return adapter.Client()
}

// StartWorkerHeartbeat refreshes the worker-liveness key until ctx is done.
// No-op when Redis is unavailable. Called by the worker tier.
func StartWorkerHeartbeat(ctx context.Context, client *redis.Client, log *slog.Logger) {
	if client == nil {
		return
	}
	if log == nil {
		log = slog.Default()
	}
	beat := func() {
		c, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := client.Set(c, workerHeartbeatKey, time.Now().UTC().Format(time.RFC3339Nano), WorkerHeartbeatTTL).Err(); err != nil {
			log.Warn("worker heartbeat publish failed", "err", err)
		}
	}
	beat()
	go func() {
		ticker := time.NewTicker(WorkerHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = client.Del(c, workerHeartbeatKey).Err()
				cancel()
				return
			case <-ticker.C:
				beat()
			}
		}
	}()
	log.Info("worker heartbeat started", "key", workerHeartbeatKey, "interval", WorkerHeartbeatInterval)
}

// WorkerLive reports whether a worker tier has published a heartbeat within the
// TTL. Returns false when Redis is unavailable (fail-open: better to risk a
// duplicate push in a degraded dev setup than to silently drop push in prod).
func WorkerLive(ctx context.Context, client *redis.Client) bool {
	if client == nil {
		return false
	}
	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	n, err := client.Exists(c, workerHeartbeatKey).Result()
	return err == nil && n > 0
}
