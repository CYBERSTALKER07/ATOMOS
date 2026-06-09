package optimizationjobs

import (
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	contract "github.com/pegasusx/pegasusx/packages/optimizer-contract"
)

// JobRecord represents a durable row in OptimizationJobs.
type JobRecord struct {
	JobID          string
	SupplierID     string
	Status         string
	RequestType    string
	PayloadJSON    []byte
	IdempotencyKey spanner.NullString
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// InsertJobMutation creates a Spanner mutation to insert a new OptimizationJob.
func InsertJobMutation(
	jobID string,
	supplierID string,
	status contract.OptimizationJobStatus,
	requestType contract.OptimizationJobType,
	payload *contract.OptimizationJobEnvelope,
	idempotencyKey string,
) (*spanner.Mutation, error) {
	if jobID == "" {
		jobID = uuid.New().String()
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize payload: %w", err)
	}

	var idemKey spanner.NullString
	if idempotencyKey != "" {
		idemKey = spanner.NullString{StringVal: idempotencyKey, Valid: true}
	}

	return spanner.Insert(
		"OptimizationJobs",
		[]string{
			"JobId",
			"SupplierId",
			"Status",
			"RequestType",
			"PayloadJson",
			"IdempotencyKey",
			"CreatedAt",
			"UpdatedAt",
		},
		[]interface{}{
			jobID,
			supplierID,
			string(status),
			string(requestType),
			payloadBytes,
			idemKey,
			spanner.CommitTimestamp,
			spanner.CommitTimestamp,
		},
	), nil
}
