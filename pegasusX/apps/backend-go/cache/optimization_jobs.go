package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// AddActiveOptimizationJob performs a best-effort SADD to track an active job.
func AddActiveOptimizationJob(ctx context.Context, client redis.Cmdable, supplierID string, jobID string) error {
	if client == nil {
		return nil
	}
	key := KeyActiveOptimizationJobs(supplierID)
	// Track job ID in the set
	err := client.SAdd(ctx, key, jobID).Err()
	if err != nil {
		return err
	}
	// Expire the set after 24 hours as a fallback
	return client.Expire(ctx, key, 24*time.Hour).Err()
}

// RemoveActiveOptimizationJob performs a best-effort SREM to untrack a resolved job.
func RemoveActiveOptimizationJob(ctx context.Context, client redis.Cmdable, supplierID string, jobID string) error {
	if client == nil {
		return nil
	}
	key := KeyActiveOptimizationJobs(supplierID)
	return client.SRem(ctx, key, jobID).Err()
}

// GetActiveOptimizationJobs returns the set of active job IDs from Redis.
func GetActiveOptimizationJobs(ctx context.Context, client redis.Cmdable, supplierID string) ([]string, error) {
	if client == nil {
		return nil, nil // Redis not available
	}
	key := KeyActiveOptimizationJobs(supplierID)
	return client.SMembers(ctx, key).Result()
}
