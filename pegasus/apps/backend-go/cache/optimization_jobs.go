package cache

import (
	"context"
	"log/slog"
)

// MarkSupplierOptimizationJobActive records a queued optimization job in the
// supplier's active-job Redis set. Nil Redis degrades silently so Spanner
// remains the source of truth.
func MarkSupplierOptimizationJobActive(ctx context.Context, supplierID string, jobID string) error {
	c := GetClient()
	if c == nil || supplierID == "" || jobID == "" {
		return nil
	}

	key := SupplierJobsActiveKey(supplierID)
	pipe := c.Pipeline()
	pipe.SAdd(ctx, key, jobID)
	pipe.Expire(ctx, key, TTLSupplierJobsActive)
	_, err := pipe.Exec(ctx)
	if err != nil {
		slog.WarnContext(ctx, "supplier optimization active-set add failed", "supplier_id", supplierID, "job_id", jobID, "err", err)
	}
	return err
}

// RemoveSupplierOptimizationJobActive removes a terminal optimization job from
// the supplier's active-job Redis set. Nil Redis degrades silently.
func RemoveSupplierOptimizationJobActive(ctx context.Context, supplierID string, jobID string) error {
	c := GetClient()
	if c == nil || supplierID == "" || jobID == "" {
		return nil
	}

	if err := c.SRem(ctx, SupplierJobsActiveKey(supplierID), jobID).Err(); err != nil {
		slog.WarnContext(ctx, "supplier optimization active-set remove failed", "supplier_id", supplierID, "job_id", jobID, "err", err)
		return err
	}
	return nil
}

// ListSupplierOptimizationJobsActive returns the active optimization job ids
// for one supplier from Redis. Nil Redis returns an empty slice so callers can
// explicitly choose their degraded fallback path.
func ListSupplierOptimizationJobsActive(ctx context.Context, supplierID string) ([]string, error) {
	c := GetClient()
	if c == nil || supplierID == "" {
		return []string{}, nil
	}

	jobIDs, err := c.SMembers(ctx, SupplierJobsActiveKey(supplierID)).Result()
	if err != nil {
		return nil, err
	}
	if jobIDs == nil {
		return []string{}, nil
	}
	return jobIDs, nil
}
