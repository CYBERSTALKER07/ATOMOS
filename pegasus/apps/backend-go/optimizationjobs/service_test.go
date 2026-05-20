package optimizationjobs

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewDefaultsQueuedStatus(t *testing.T) {
	requestedAt := time.Date(2026, time.May, 20, 12, 0, 0, 0, time.UTC)
	job, err := New(CreateParams{
		SupplierID:      "supplier-1",
		JobType:         JobTypeAutoDispatch,
		SolverType:      SolverTypeVRP,
		TraceID:         "trace-1",
		SourceEventType: "OPTIMIZATION_JOB_QUEUED",
		Payload:         json.RawMessage(`{"scope":"dispatch"}`),
		RequestedAt:     requestedAt,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if job.JobID == "" {
		t.Fatal("expected job id to be generated")
	}
	if job.Status != StatusQueued {
		t.Fatalf("expected queued status, got %q", job.Status)
	}
	if !job.RequestedAt.Equal(requestedAt) {
		t.Fatalf("expected requested_at %s, got %s", requestedAt, job.RequestedAt)
	}
	if string(job.Payload) != `{"scope":"dispatch"}` {
		t.Fatalf("expected payload copy, got %s", string(job.Payload))
	}
	if job.AttemptCount != 0 {
		t.Fatalf("expected attempt_count=0, got %d", job.AttemptCount)
	}
}

func TestNewRejectsInvalidSolver(t *testing.T) {
	_, err := New(CreateParams{
		SupplierID:      "supplier-1",
		JobType:         JobTypeAutoDispatch,
		SolverType:      SolverType("BAD"),
		SourceEventType: "OPTIMIZATION_JOB_QUEUED",
		Payload:         json.RawMessage(`{"scope":"dispatch"}`),
	})
	if err == nil {
		t.Fatal("expected invalid solver type error")
	}
}

func TestNewUsesProvidedJobID(t *testing.T) {
	job, err := New(CreateParams{
		JobID:           "job-123",
		SupplierID:      "supplier-1",
		JobType:         JobTypeAutoDispatch,
		SolverType:      SolverTypeVRP,
		SourceEventType: "OPTIMIZATION_JOB_QUEUED",
		Payload:         json.RawMessage(`{"scope":"dispatch"}`),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if job.JobID != "job-123" {
		t.Fatalf("expected provided job id, got %q", job.JobID)
	}
}
