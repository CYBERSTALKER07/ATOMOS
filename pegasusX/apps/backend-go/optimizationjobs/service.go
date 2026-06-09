package optimizationjobs

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	contract "github.com/pegasusx/pegasusx/packages/optimizer-contract"
)

// EnqueueJob creates a job record and a transactional outbox event
// to dispatch the job to the optimizer worker via Kafka TopicOptimizerJobs.
func EnqueueJob(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxTxn outbox.TxnBuffer,
	jobID string,
	supplierID string,
	requestType contract.OptimizationJobType,
	payload *contract.OptimizationJobEnvelope,
	idempotencyKey string,
) error {
	if jobID == "" {
		return errors.New("jobID cannot be empty")
	}

	payload.JobID = jobID
	payload.SupplierID = supplierID
	payload.JobType = requestType
	payload.Status = contract.OptimizationJobStatusQueued

	mut, err := InsertJobMutation(
		jobID,
		supplierID,
		contract.OptimizationJobStatusQueued,
		requestType,
		payload,
		idempotencyKey,
	)
	if err != nil {
		return err
	}
	err = txn.BufferWrite([]*spanner.Mutation{mut})
	if err != nil {
		return err
	}

	// TopicOptimizerJobs is the topic used for optimizer routing.
	err = outbox.EmitJSON(
		ctx,
		outboxTxn,
		"OptimizationJob",
		jobID,
		"TopicOptimizerJobs",
		payload,
	)
	if err != nil {
		return fmt.Errorf("failed to write outbox event: %w", err)
	}

	return nil
}

// MarkJobStatus updates the status of an existing job in Spanner.
func MarkJobStatus(jobID string, status contract.OptimizationJobStatus) *spanner.Mutation {
	return spanner.Update(
		"OptimizationJobs",
		[]string{"JobId", "Status", "UpdatedAt"},
		[]interface{}{jobID, string(status), spanner.CommitTimestamp},
	)
}
