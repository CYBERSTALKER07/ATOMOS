package optimizationjobs

import "cloud.google.com/go/spanner"

func InsertMutation(job Job) *spanner.Mutation {
	return spanner.Insert(
		"OptimizationJobs",
		[]string{
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
		},
		[]interface{}{
			job.JobID,
			job.SupplierID,
			job.JobType,
			string(job.SolverType),
			string(job.Status),
			job.TraceID,
			job.IdempotencyKey,
			job.SourceEventType,
			[]byte(job.Payload),
			[]byte(job.ResultPayload),
			job.FailureCode,
			job.FailureMessage,
			job.AttemptCount,
			job.RequestedAt,
			job.PublishedAt,
			job.StartedAt,
			job.CompletedAt,
			job.AppliedAt,
			spanner.CommitTimestamp,
		},
	)
}
