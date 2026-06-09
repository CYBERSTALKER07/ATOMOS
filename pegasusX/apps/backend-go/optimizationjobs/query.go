package optimizationjobs

import (
	"context"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/redis/go-redis/v9"
	"google.golang.org/api/iterator"
)

// ActiveJobProjection represents the read model for an active optimization job.
type ActiveJobProjection struct {
	JobID       string `json:"job_id"`
	SupplierID  string `json:"supplier_id"`
	Status      string `json:"status"`
	RequestType string `json:"request_type"`
}

// GetActiveJobs returns all active optimization jobs for a supplier.
// It uses Redis as the primary read source, with a Spanner fallback.
func GetActiveJobs(ctx context.Context, client spanner.Client, redisClient redis.Cmdable, supplierID string) ([]ActiveJobProjection, error) {
	// 1. Primary: Redis
	jobIDs, err := cache.GetActiveOptimizationJobs(ctx, redisClient, supplierID)
	if err == nil && len(jobIDs) > 0 {
		// Found in Redis. We can fetch full details from Spanner or assume they are RUNNING.
		// For projection simplicity, we'll return basic stubs or perform a batch Spanner read.
		stmt := spanner.Statement{
			SQL: `SELECT JobId, SupplierId, Status, RequestType 
			      FROM OptimizationJobs 
			      WHERE SupplierId = @supplierId AND JobId IN UNNEST(@jobIds)`,
			Params: map[string]interface{}{
				"supplierId": supplierID,
				"jobIds":     jobIDs,
			},
		}
		return executeActiveJobsQuery(ctx, client, stmt)
	}

	// 2. Fallback: Spanner Query (only statuses before terminal)
	stmt := spanner.Statement{
		SQL: `SELECT JobId, SupplierId, Status, RequestType 
		      FROM OptimizationJobs 
		      WHERE SupplierId = @supplierId 
		      AND Status IN ('QUEUED', 'PUBLISHED', 'RUNNING', 'APPLYING')`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
		},
	}
	return executeActiveJobsQuery(ctx, client, stmt)
}

func executeActiveJobsQuery(ctx context.Context, client spanner.Client, stmt spanner.Statement) ([]ActiveJobProjection, error) {
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var jobs []ActiveJobProjection
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var job ActiveJobProjection
		if err := row.Columns(&job.JobID, &job.SupplierID, &job.Status, &job.RequestType); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}
