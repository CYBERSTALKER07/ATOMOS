package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"

	"optimizercoreadapter/internal/config"
	"optimizercoreadapter/internal/model"
	"optimizercoreadapter/internal/optimizergrpc"

	contract "optimizercontract"
)

const (
	outboxAggregateType = "OptimizationJob"
	outboxEventType     = "OPTIMIZATION_SOLVED"
	outboxTopicMain     = "pegasus-logistics-events"
)

type Worker struct {
	cfg          config.Config
	reader       *kafkago.Reader
	redis        redis.UniversalClient
	spanner      *spanner.Client
	solverClient optimizergrpc.SolverClient
}

func NewWorker(cfg config.Config, spannerClient *spanner.Client, solverClient optimizergrpc.SolverClient, redisClient redis.UniversalClient) *Worker {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  cfg.KafkaBrokers,
		Topic:    cfg.KafkaTopic,
		GroupID:  cfg.KafkaGroupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	return &Worker{
		cfg:          cfg,
		reader:       reader,
		redis:        redisClient,
		spanner:      spannerClient,
		solverClient: solverClient,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		msg, err := w.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("fetch kafka message: %w", err)
		}

		if err := w.handleMessage(ctx, msg); err != nil {
			slog.ErrorContext(ctx, "optimizer worker handler error", "partition", msg.Partition, "offset", msg.Offset, "err", err)
		}

		if err := w.reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("commit kafka message: %w", err)
		}
	}
}

func (w *Worker) Close() error {
	if w == nil {
		return nil
	}
	if w.solverClient != nil {
		_ = w.solverClient.Close()
	}
	if w.reader != nil {
		return w.reader.Close()
	}
	return nil
}

func (w *Worker) handleMessage(ctx context.Context, msg kafkago.Message) error {
	var job contract.OptimizationJobEnvelope
	if err := json.Unmarshal(msg.Value, &job); err != nil {
		return fmt.Errorf("decode optimization job: %w", err)
	}

	if job.JobID == "" {
		return fmt.Errorf("optimization job missing job_id")
	}

	alreadyPublished, err := w.resultAlreadyPublished(ctx, job.JobID)
	if err != nil {
		return fmt.Errorf("check outbox idempotency for job %s: %w", job.JobID, err)
	}
	if alreadyPublished {
		slog.InfoContext(ctx, "optimization job already solved", "job_id", job.JobID)
		return nil
	}

	if err := w.markJobRunning(ctx, job.JobID); err != nil {
		return fmt.Errorf("mark optimization job %s running: %w", job.JobID, err)
	}

	switch job.SolverType {
	case contract.OptimizationSolverTypeVRP:
		req, err := BuildVRPRequest(job)
		if err != nil {
			return w.failJob(ctx, job, "VRP_REQUEST_BUILD_FAILED", fmt.Errorf("build vrp request: %w", err))
		}

		result, err := w.solveVRPWithRetry(ctx, req)
		if err != nil {
			return w.failJob(ctx, job, "VRP_SOLVE_FAILED", fmt.Errorf("solve vrp: %w", err))
		}

		return w.persistSolvedResult(ctx, job, &model.OptimizationSolvedEvent{
			JobID:      job.JobID,
			TraceID:    job.TraceID,
			SupplierID: job.SupplierID,
			SolverType: model.SolverTypeVRP,
			Status:     result.Status,
			TimedOut:   result.TimedOut,
			MatrixSize: result.MatrixSize,
			ProducedAt: time.Now().UTC().Format(time.RFC3339Nano),
			Warnings:   result.Warnings,
			VRP:        result,
		})

	case contract.OptimizationSolverTypeCPSAT:
		req, err := BuildCPSATRequest(job)
		if err != nil {
			return w.failJob(ctx, job, "CPSAT_REQUEST_BUILD_FAILED", fmt.Errorf("build cp-sat request: %w", err))
		}

		result, err := w.solveCPSATWithRetry(ctx, req)
		if err != nil {
			return w.failJob(ctx, job, "CPSAT_SOLVE_FAILED", fmt.Errorf("solve cp-sat: %w", err))
		}

		return w.persistSolvedResult(ctx, job, &model.OptimizationSolvedEvent{
			JobID:      job.JobID,
			TraceID:    job.TraceID,
			SupplierID: job.SupplierID,
			SolverType: model.SolverTypeCPSAT,
			Status:     result.Status,
			TimedOut:   result.TimedOut,
			MatrixSize: result.MatrixSize,
			ProducedAt: time.Now().UTC().Format(time.RFC3339Nano),
			Warnings:   result.Warnings,
			CPSAT:      result,
		})
	default:
		return w.failJob(ctx, job, "UNSUPPORTED_SOLVER_TYPE", fmt.Errorf("unsupported solver_type %q", job.SolverType))
	}
}

func (w *Worker) solveVRPWithRetry(ctx context.Context, req *model.VRPRequestEnvelope) (*model.VRPResultEnvelope, error) {
	var lastErr error

	for attempt := 1; attempt <= w.cfg.SolverMaxAttempts; attempt++ {
		result, err := w.solverClient.CalculateRoute(ctx, req)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if attempt == w.cfg.SolverMaxAttempts {
			break
		}
		if err := sleepWithBackoff(ctx, w.cfg.SolverRetryBaseTime, attempt); err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("vrp retries exhausted: %w", lastErr)
}

func (w *Worker) solveCPSATWithRetry(ctx context.Context, req *model.CPSATRequestEnvelope) (*model.CPSATResultEnvelope, error) {
	var lastErr error

	for attempt := 1; attempt <= w.cfg.SolverMaxAttempts; attempt++ {
		result, err := w.solverClient.ResolveConstraint(ctx, req)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if attempt == w.cfg.SolverMaxAttempts {
			break
		}
		if err := sleepWithBackoff(ctx, w.cfg.SolverRetryBaseTime, attempt); err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("cp-sat retries exhausted: %w", lastErr)
}

func (w *Worker) resultAlreadyPublished(ctx context.Context, jobID string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT EventId FROM OutboxEvents
			WHERE AggregateType = @aggregate_type
			  AND AggregateId = @aggregate_id
			  AND EventType = @event_type
			LIMIT 1`,
		Params: map[string]interface{}{
			"aggregate_type": outboxAggregateType,
			"aggregate_id":   jobID,
			"event_type":     outboxEventType,
		},
	}

	iter := w.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (w *Worker) markJobRunning(ctx context.Context, jobID string) error {
	startedAt := time.Now().UTC()

	_, err := w.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutation := spanner.Update("OptimizationJobs",
			[]string{"JobId", "Status", "PublishedAt", "StartedAt", "AttemptCount", "FailureCode", "FailureMessage", "UpdatedAt"},
			[]interface{}{jobID, string(contract.OptimizationJobStatusRunning), startedAt, startedAt, int64(1), "", "", spanner.CommitTimestamp},
		)
		return txn.BufferWrite([]*spanner.Mutation{mutation})
	})
	if err != nil {
		return fmt.Errorf("mark optimization job running: %w", err)
	}

	return nil
}

func (w *Worker) persistSolvedResult(ctx context.Context, job contract.OptimizationJobEnvelope, event *model.OptimizationSolvedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal optimization solved event: %w", err)
	}
	completedAt := time.Now().UTC()

	_, err = w.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		jobMutation := spanner.Update("OptimizationJobs",
			[]string{"JobId", "Status", "ResultPayload", "FailureCode", "FailureMessage", "CompletedAt", "UpdatedAt"},
			[]interface{}{job.JobID, string(contract.OptimizationJobStatusSolved), payload, "", "", completedAt, spanner.CommitTimestamp},
		)
		outboxMutation := spanner.Insert("OutboxEvents",
			[]string{"EventId", "AggregateType", "AggregateId", "EventType", "TopicName", "Payload", "CreatedAt", "TraceID"},
			[]interface{}{uuid.NewString(), outboxAggregateType, job.JobID, outboxEventType, outboxTopicMain, payload, spanner.CommitTimestamp, job.TraceID},
		)
		return txn.BufferWrite([]*spanner.Mutation{jobMutation, outboxMutation})
	})
	if err != nil {
		return fmt.Errorf("write outbox result for job %s: %w", job.JobID, err)
	}
	if err := w.removeActiveJob(ctx, job.SupplierID, job.JobID); err != nil {
		slog.WarnContext(ctx, "optimizer worker active-set remove failed after solve", "job_id", job.JobID, "supplier_id", job.SupplierID, "err", err)
	}

	return nil
}

func (w *Worker) failJob(ctx context.Context, job contract.OptimizationJobEnvelope, failureCode string, cause error) error {
	if err := w.recordJobFailure(ctx, job.SupplierID, job.JobID, failureCode, cause.Error()); err != nil {
		return fmt.Errorf("record failure for job %s after %s: %w", job.JobID, failureCode, err)
	}

	slog.ErrorContext(ctx, "optimizer worker job failed", "job_id", job.JobID, "failure_code", failureCode, "err", cause)
	return nil
}

func (w *Worker) recordJobFailure(ctx context.Context, supplierID string, jobID string, failureCode string, failureMessage string) error {
	completedAt := time.Now().UTC()

	_, err := w.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutation := spanner.Update("OptimizationJobs",
			[]string{"JobId", "Status", "ResultPayload", "FailureCode", "FailureMessage", "CompletedAt", "UpdatedAt"},
			[]interface{}{jobID, string(contract.OptimizationJobStatusFailed), []byte{}, failureCode, failureMessage, completedAt, spanner.CommitTimestamp},
		)
		return txn.BufferWrite([]*spanner.Mutation{mutation})
	})
	if err != nil {
		return fmt.Errorf("mark optimization job failed: %w", err)
	}
	if err := w.removeActiveJob(ctx, supplierID, jobID); err != nil {
		slog.WarnContext(ctx, "optimizer worker active-set remove failed after failure", "job_id", jobID, "supplier_id", supplierID, "err", err)
	}

	return nil
}

func (w *Worker) removeActiveJob(ctx context.Context, supplierID string, jobID string) error {
	if w.redis == nil || supplierID == "" || jobID == "" {
		return nil
	}
	return w.redis.SRem(ctx, supplierActiveJobsKey(supplierID), jobID).Err()
}

func supplierActiveJobsKey(supplierID string) string {
	return "supplier:" + supplierID + ":jobs:active"
}

func sleepWithBackoff(ctx context.Context, base time.Duration, attempt int) error {
	if attempt < 1 {
		attempt = 1
	}

	backoff := base * time.Duration(1<<(attempt-1))
	jitter := time.Duration(rand.Int63n(int64(base)))
	wait := backoff + jitter

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
