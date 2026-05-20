package optimizationjobs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusQueued    Status = "QUEUED"
	StatusPublished Status = "PUBLISHED"
	StatusRunning   Status = "RUNNING"
	StatusSolved    Status = "SOLVED"
	StatusApplying  Status = "APPLYING"
	StatusApplied   Status = "APPLIED"
	StatusFailed    Status = "FAILED"
	StatusCancelled Status = "CANCELLED"
)

type SolverType string

const (
	SolverTypeVRP   SolverType = "VRP"
	SolverTypeCPSAT SolverType = "CP_SAT"
)

const (
	JobTypeAutoDispatch  = "AUTO_DISPATCH"
	JobTypeDispatchQueue = "DISPATCH_QUEUE"
)

type Job struct {
	JobID           string
	SupplierID      string
	JobType         string
	SolverType      SolverType
	Status          Status
	TraceID         string
	IdempotencyKey  string
	SourceEventType string
	Payload         json.RawMessage
	ResultPayload   json.RawMessage
	FailureCode     string
	FailureMessage  string
	AttemptCount    int64
	RequestedAt     time.Time
	PublishedAt     *time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
	AppliedAt       *time.Time
	UpdatedAt       time.Time
}

type Summary struct {
	JobID       string
	SupplierID  string
	JobType     string
	SolverType  SolverType
	Status      Status
	TraceID     string
	RequestedAt time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	UpdatedAt   time.Time
}

type CreateParams struct {
	JobID           string
	SupplierID      string
	JobType         string
	SolverType      SolverType
	TraceID         string
	IdempotencyKey  string
	SourceEventType string
	Payload         json.RawMessage
	RequestedAt     time.Time
}

func New(params CreateParams) (Job, error) {
	if params.SupplierID == "" {
		return Job{}, fmt.Errorf("optimizationjobs: supplier_id is required")
	}
	if params.JobType == "" {
		return Job{}, fmt.Errorf("optimizationjobs: job_type is required")
	}
	if !params.SolverType.Valid() {
		return Job{}, fmt.Errorf("optimizationjobs: invalid solver_type %q", params.SolverType)
	}
	if params.SourceEventType == "" {
		return Job{}, fmt.Errorf("optimizationjobs: source_event_type is required")
	}
	if len(params.Payload) == 0 {
		return Job{}, fmt.Errorf("optimizationjobs: payload is required")
	}

	requestedAt := params.RequestedAt.UTC()
	if params.RequestedAt.IsZero() {
		requestedAt = time.Now().UTC()
	}

	payload := make(json.RawMessage, len(params.Payload))
	copy(payload, params.Payload)

	jobID := params.JobID
	if jobID == "" {
		jobID = uuid.NewString()
	}

	return Job{
		JobID:           jobID,
		SupplierID:      params.SupplierID,
		JobType:         params.JobType,
		SolverType:      params.SolverType,
		Status:          StatusQueued,
		TraceID:         params.TraceID,
		IdempotencyKey:  params.IdempotencyKey,
		SourceEventType: params.SourceEventType,
		Payload:         payload,
		AttemptCount:    0,
		RequestedAt:     requestedAt,
		UpdatedAt:       requestedAt,
	}, nil
}

func (s Status) Valid() bool {
	switch s {
	case StatusQueued, StatusPublished, StatusRunning, StatusSolved, StatusApplying, StatusApplied, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

func (s Status) IsActive() bool {
	switch s {
	case StatusQueued, StatusPublished, StatusRunning, StatusApplying:
		return true
	default:
		return false
	}
}

func (s SolverType) Valid() bool {
	switch s {
	case SolverTypeVRP, SolverTypeCPSAT:
		return true
	default:
		return false
	}
}
