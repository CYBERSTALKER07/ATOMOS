package adapter

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"optimizercoreadapter/internal/config"
	"optimizercoreadapter/internal/model"
)

type fakeSolverClient struct {
	calculateRouteFn    func(context.Context, *model.VRPRequestEnvelope) (*model.VRPResultEnvelope, error)
	resolveConstraintFn func(context.Context, *model.CPSATRequestEnvelope) (*model.CPSATResultEnvelope, error)
}

func (f *fakeSolverClient) CalculateRoute(ctx context.Context, req *model.VRPRequestEnvelope) (*model.VRPResultEnvelope, error) {
	if f.calculateRouteFn == nil {
		return nil, errors.New("calculate route handler is not configured")
	}
	return f.calculateRouteFn(ctx, req)
}

func (f *fakeSolverClient) ResolveConstraint(ctx context.Context, req *model.CPSATRequestEnvelope) (*model.CPSATResultEnvelope, error) {
	if f.resolveConstraintFn == nil {
		return nil, errors.New("resolve constraint handler is not configured")
	}
	return f.resolveConstraintFn(ctx, req)
}

func (f *fakeSolverClient) Close() error {
	return nil
}

func TestSolveVRPWithRetryRecoversAfterTransientFailures(t *testing.T) {
	attempts := 0

	worker := &Worker{
		cfg: config.Config{
			SolverMaxAttempts:   3,
			SolverRetryBaseTime: time.Millisecond,
		},
		solverClient: &fakeSolverClient{
			calculateRouteFn: func(context.Context, *model.VRPRequestEnvelope) (*model.VRPResultEnvelope, error) {
				attempts++
				if attempts < 3 {
					return nil, status.Error(codes.Unavailable, "sidecar crash")
				}
				return &model.VRPResultEnvelope{Status: model.SolverStatusFeasible, MatrixSize: 3}, nil
			},
		},
	}

	result, err := worker.solveVRPWithRetry(context.Background(), &model.VRPRequestEnvelope{Meta: model.SolverMetadata{JobID: "job-vrp"}})
	if err != nil {
		t.Fatalf("expected successful retry recovery, got error: %v", err)
	}
	if result == nil || result.Status != model.SolverStatusFeasible {
		t.Fatalf("expected feasible result after retries, got: %+v", result)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestSolveVRPWithRetryExhaustsOnUnavailable(t *testing.T) {
	attempts := 0

	worker := &Worker{
		cfg: config.Config{
			SolverMaxAttempts:   3,
			SolverRetryBaseTime: time.Millisecond,
		},
		solverClient: &fakeSolverClient{
			calculateRouteFn: func(context.Context, *model.VRPRequestEnvelope) (*model.VRPResultEnvelope, error) {
				attempts++
				return nil, status.Error(codes.Unavailable, "transport unavailable")
			},
		},
	}

	_, err := worker.solveVRPWithRetry(context.Background(), &model.VRPRequestEnvelope{Meta: model.SolverMetadata{JobID: "job-vrp-crash"}})
	if err == nil {
		t.Fatal("expected retry exhaustion error")
	}
	if !strings.Contains(err.Error(), "vrp retries exhausted") {
		t.Fatalf("expected exhaustion wrapper, got: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestSolveCPSATWithRetryReturnsInfeasibleWithoutRetry(t *testing.T) {
	attempts := 0

	worker := &Worker{
		cfg: config.Config{
			SolverMaxAttempts:   3,
			SolverRetryBaseTime: time.Millisecond,
		},
		solverClient: &fakeSolverClient{
			resolveConstraintFn: func(context.Context, *model.CPSATRequestEnvelope) (*model.CPSATResultEnvelope, error) {
				attempts++
				return &model.CPSATResultEnvelope{
					Status:                model.SolverStatusInfeasible,
					TimedOut:              false,
					MatrixSize:            2,
					UnassignedManifestIDs: []string{"manifest-2"},
					Warnings:              []string{"infeasible under current capacities"},
				}, nil
			},
		},
	}

	result, err := worker.solveCPSATWithRetry(context.Background(), &model.CPSATRequestEnvelope{Meta: model.SolverMetadata{JobID: "job-cpsat-infeasible"}})
	if err != nil {
		t.Fatalf("expected infeasible response to be returned without error, got: %v", err)
	}
	if result == nil || result.Status != model.SolverStatusInfeasible {
		t.Fatalf("expected infeasible result, got: %+v", result)
	}
	if attempts != 1 {
		t.Fatalf("expected no retry for valid infeasible response, got %d attempts", attempts)
	}
}

func TestSolveCPSATWithRetryExhaustsOnTimeout(t *testing.T) {
	attempts := 0

	worker := &Worker{
		cfg: config.Config{
			SolverMaxAttempts:   2,
			SolverRetryBaseTime: time.Millisecond,
		},
		solverClient: &fakeSolverClient{
			resolveConstraintFn: func(context.Context, *model.CPSATRequestEnvelope) (*model.CPSATResultEnvelope, error) {
				attempts++
				return nil, context.DeadlineExceeded
			},
		},
	}

	_, err := worker.solveCPSATWithRetry(context.Background(), &model.CPSATRequestEnvelope{Meta: model.SolverMetadata{JobID: "job-cpsat-timeout"}})
	if err == nil {
		t.Fatal("expected timeout exhaustion error")
	}
	if !strings.Contains(err.Error(), "cp-sat retries exhausted") {
		t.Fatalf("expected cp-sat exhaustion wrapper, got: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}
