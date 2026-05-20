package optimizationjobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const defaultActiveListLimit = 100

func GetByID(ctx context.Context, client *spanner.Client, supplierID string, jobID string) (Job, bool, error) {
	row, err := client.Single().ReadRow(ctx, "OptimizationJobs", spanner.Key{jobID}, []string{
		"JobId",
		"SupplierId",
		"JobType",
		"SolverType",
		"Status",
		"TraceId",
		"IdempotencyKey",
		"SourceEventType",
		"Payload",
		"ResultPayload",
		"FailureCode",
		"FailureMessage",
		"AttemptCount",
		"RequestedAt",
		"PublishedAt",
		"StartedAt",
		"CompletedAt",
		"AppliedAt",
		"UpdatedAt",
	})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return Job{}, false, nil
		}
		return Job{}, false, fmt.Errorf("read optimization job %s: %w", jobID, err)
	}

	job, err := scanJob(row)
	if err != nil {
		return Job{}, false, err
	}
	if job.SupplierID != supplierID {
		return Job{}, false, nil
	}
	return job, true, nil
}

func ListSummariesByJobIDs(ctx context.Context, client *spanner.Client, supplierID string, jobIDs []string) ([]Summary, error) {
	if len(jobIDs) == 0 {
		return []Summary{}, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT JobId, SupplierId, JobType, SolverType, Status, TraceId, RequestedAt, StartedAt, CompletedAt, UpdatedAt
			FROM OptimizationJobs
			WHERE SupplierId = @supplierId
			  AND JobId IN UNNEST(@jobIds)
			ORDER BY UpdatedAt DESC`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"jobIds":     jobIDs,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var summaries []Summary
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query optimization jobs by ids: %w", err)
		}

		summary, err := scanSummary(row)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	if summaries == nil {
		return []Summary{}, nil
	}
	return summaries, nil
}

func ListActiveSummariesBySupplier(ctx context.Context, client *spanner.Client, supplierID string, limit int) ([]Summary, error) {
	if limit <= 0 {
		limit = defaultActiveListLimit
	}

	stmt := spanner.Statement{
		SQL: `SELECT JobId, SupplierId, JobType, SolverType, Status, TraceId, RequestedAt, StartedAt, CompletedAt, UpdatedAt
			FROM OptimizationJobs@{FORCE_INDEX=Idx_OptimizationJobs_BySupplierStatus}
			WHERE SupplierId = @supplierId
			  AND Status IN UNNEST(@statuses)
			ORDER BY UpdatedAt DESC
			LIMIT @limit`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"statuses":   []string{string(StatusQueued), string(StatusPublished), string(StatusRunning), string(StatusApplying)},
			"limit":      int64(limit),
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var summaries []Summary
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query active optimization jobs: %w", err)
		}

		summary, err := scanSummary(row)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	if summaries == nil {
		return []Summary{}, nil
	}
	return summaries, nil
}

func scanJob(row *spanner.Row) (Job, error) {
	var job Job
	var solverType string
	var status string
	var traceID spanner.NullString
	var idempotencyKey spanner.NullString
	var sourceEventType spanner.NullString
	var payload []byte
	var resultPayload []byte
	var failureCode spanner.NullString
	var failureMessage spanner.NullString
	var publishedAt spanner.NullTime
	var startedAt spanner.NullTime
	var completedAt spanner.NullTime
	var appliedAt spanner.NullTime

	if err := row.Columns(
		&job.JobID,
		&job.SupplierID,
		&job.JobType,
		&solverType,
		&status,
		&traceID,
		&idempotencyKey,
		&sourceEventType,
		&payload,
		&resultPayload,
		&failureCode,
		&failureMessage,
		&job.AttemptCount,
		&job.RequestedAt,
		&publishedAt,
		&startedAt,
		&completedAt,
		&appliedAt,
		&job.UpdatedAt,
	); err != nil {
		return Job{}, fmt.Errorf("scan optimization job row: %w", err)
	}

	job.SolverType = SolverType(solverType)
	job.Status = Status(status)
	job.TraceID = traceID.StringVal
	job.IdempotencyKey = idempotencyKey.StringVal
	job.SourceEventType = sourceEventType.StringVal
	job.Payload = cloneRawMessage(payload)
	job.ResultPayload = cloneRawMessage(resultPayload)
	job.FailureCode = failureCode.StringVal
	job.FailureMessage = failureMessage.StringVal
	if publishedAt.Valid {
		value := publishedAt.Time
		job.PublishedAt = &value
	}
	if startedAt.Valid {
		value := startedAt.Time
		job.StartedAt = &value
	}
	if completedAt.Valid {
		value := completedAt.Time
		job.CompletedAt = &value
	}
	if appliedAt.Valid {
		value := appliedAt.Time
		job.AppliedAt = &value
	}

	return job, nil
}

func scanSummary(row *spanner.Row) (Summary, error) {
	var summary Summary
	var solverType string
	var status string
	var traceID spanner.NullString
	var startedAt spanner.NullTime
	var completedAt spanner.NullTime

	if err := row.Columns(
		&summary.JobID,
		&summary.SupplierID,
		&summary.JobType,
		&solverType,
		&status,
		&traceID,
		&summary.RequestedAt,
		&startedAt,
		&completedAt,
		&summary.UpdatedAt,
	); err != nil {
		return Summary{}, fmt.Errorf("scan optimization job summary row: %w", err)
	}

	summary.SolverType = SolverType(solverType)
	summary.Status = Status(status)
	summary.TraceID = traceID.StringVal
	if startedAt.Valid {
		value := startedAt.Time
		summary.StartedAt = &value
	}
	if completedAt.Valid {
		value := completedAt.Time
		summary.CompletedAt = &value
	}

	return summary, nil
}

func cloneRawMessage(data []byte) json.RawMessage {
	if len(data) == 0 {
		return nil
	}
	cloned := make(json.RawMessage, len(data))
	copy(cloned, data)
	return cloned
}
