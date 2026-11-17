package bootstrap

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pegasusx/pegasusx/apps/backend-go/workerlease"
)

// Worker-liveness tracking via Redis heartbeat backed by Cloud Spanner WorkerLeases.
//
// The worker tier refreshes both Redis (when configured) and the distributed
// WorkerLeases row in Spanner so liveness probes never fail open when Redis is
// unavailable. If neither Redis nor Spanner can confirm the absence of a live
// worker, WorkerLiveWithLease fails closed (returns true) to prevent API or
// standby pods from acting as duplicate leaders.

const (
	// workerHeartbeatKey is the Redis key the worker tier refreshes.
	workerHeartbeatKey = "{pegasusx:runtime:worker}:heartbeat"
	// WorkerHeartbeatLeaseName is the Cloud Spanner WorkerLeases row tracking active worker-tier presence.
	WorkerHeartbeatLeaseName = "worker:heartbeat"
	// WorkerHeartbeatTTL is how long a heartbeat stays valid after the worker dies.
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

// StartWorkerHeartbeat refreshes the worker-liveness key in Redis until ctx is done.
func StartWorkerHeartbeat(ctx context.Context, client *redis.Client, log *slog.Logger) {
	StartWorkerHeartbeatWithLease(ctx, client, nil, "", log)
}

// StartWorkerHeartbeatWithLease refreshes worker-liveness state in Redis and/or
// Spanner WorkerLeases until ctx is done.
func StartWorkerHeartbeatWithLease(
	ctx context.Context,
	client *redis.Client,
	leaseMgr workerlease.Manager,
	holderID string,
	log *slog.Logger,
) {
	if client == nil && leaseMgr == nil {
		return
	}
	if log == nil {
		log = slog.Default()
	}
	if holderID == "" {
		holderID = workerlease.DefaultHolderID("worker-heartbeat")
	}

	beat := func() {
		if client != nil {
			c, cancel := context.WithTimeout(ctx, 5*time.Second)
			if err := client.Set(c, workerHeartbeatKey, time.Now().UTC().Format(time.RFC3339), WorkerHeartbeatTTL).Err(); err != nil {
				log.Warn("worker heartbeat redis publish failed", "err", err)
			}
			cancel()
		}
		if leaseMgr != nil {
			c, cancel := context.WithTimeout(ctx, 5*time.Second)
			if _, err := leaseMgr.TryAcquireOrRenew(c, WorkerHeartbeatLeaseName, holderID, WorkerHeartbeatTTL); err != nil {
				log.Warn("worker heartbeat spanner lease publish failed", "err", err)
			}
			cancel()
		}
	}

	beat()
	go func() {
		ticker := time.NewTicker(WorkerHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
				if client != nil {
					if err := client.Del(shutdownCtx, workerHeartbeatKey).Err(); err != nil {
						log.Warn("worker heartbeat redis cleanup failed", "err", err)
					}
				}
				if leaseMgr != nil {
					if err := leaseMgr.Release(shutdownCtx, WorkerHeartbeatLeaseName, holderID); err != nil {
						log.Warn("worker heartbeat spanner lease release failed", "err", err)
					}
				}
				cancel()
				return
			case <-ticker.C:
				beat()
			}
		}
	}()
	log.Info("worker heartbeat started",
		"redis", client != nil,
		"spanner_lease", leaseMgr != nil,
		"interval", WorkerHeartbeatInterval,
	)
}

// WorkerLive reports whether a worker tier heartbeat is active, failing closed
// when Redis is unavailable.
func WorkerLive(ctx context.Context, client *redis.Client) bool {
	return WorkerLiveWithLease(ctx, client, nil)
}

// WorkerLiveWithLease checks Redis first when available and falls back to Spanner
// WorkerLeases whenever Redis is nil or unreachable. If neither backend can verify
// that no worker is running, it fails closed (returns true) so pods do not promote
// themselves into duplicate leaders.
func WorkerLiveWithLease(ctx context.Context, client *redis.Client, leaseMgr workerlease.Manager) bool {
	if client != nil {
		c, cancel := context.WithTimeout(ctx, 3*time.Second)
		n, err := client.Exists(c, workerHeartbeatKey).Result()
		cancel()
		if err == nil {
			if n > 0 {
				return true
			}
			if leaseMgr == nil {
				return false
			}
		}
	}

	if leaseMgr != nil {
		c, cancel := context.WithTimeout(ctx, 3*time.Second)
		held, err := leaseMgr.IsHeld(c, WorkerHeartbeatLeaseName)
		cancel()
		if err == nil {
			return held
		}
	}

	// Fail closed when Redis is unavailable and Spanner cannot prove absence of a worker.
	return true
}
