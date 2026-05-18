package optimizergrpc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"backend-go/cache"
	contract "optimizercontract"

	"google.golang.org/grpc"
)

type failingStub struct {
	err error
}

func (f failingStub) Solve(ctx context.Context, req *contract.SolveRequest, opts ...grpc.CallOption) (*contract.SolveResponse, error) {
	return nil, f.err
}

type successStub struct{}

func (successStub) Solve(ctx context.Context, req *contract.SolveRequest, opts ...grpc.CallOption) (*contract.SolveResponse, error) {
	return &contract.SolveResponse{}, nil
}

func TestSolveTripsCircuitBreakerOnRepeatedFailure(t *testing.T) {
	breaker := cache.NewCircuitBreaker("optimizer_grpc_test")
	breaker.FailureThreshold = 1
	breaker.OpenDuration = time.Hour

	client := &GRPCClient{
		stub:   failingStub{err: errors.New("downstream unavailable")},
		apiKey: "test-key",
		cb:     breaker,
	}

	req := &contract.SolveRequest{TraceID: "trace-1"}

	firstErr := mustFailSolve(t, client, req)
	if !strings.Contains(firstErr.Error(), "downstream unavailable") {
		t.Fatalf("expected downstream failure, got: %v", firstErr)
	}

	secondErr := mustFailSolve(t, client, req)
	if !strings.Contains(secondErr.Error(), "circuit breaker") {
		t.Fatalf("expected circuit breaker fast-fail, got: %v", secondErr)
	}
	if !strings.Contains(secondErr.Error(), "OPEN") {
		t.Fatalf("expected OPEN state in breaker error, got: %v", secondErr)
	}
}

func TestSolveSucceedsWhenStubSucceeds(t *testing.T) {
	breaker := cache.NewCircuitBreaker("optimizer_grpc_success_test")
	breaker.FailureThreshold = 1
	breaker.OpenDuration = time.Hour

	client := &GRPCClient{
		stub:   successStub{},
		apiKey: "test-key",
		cb:     breaker,
	}

	resp, err := client.Solve(context.Background(), &contract.SolveRequest{TraceID: "trace-ok"})
	if err != nil {
		t.Fatalf("expected solve success, got error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if breaker.State() != cache.CircuitClosed {
		t.Fatalf("expected circuit breaker CLOSED, got: %s", breaker.State())
	}
}

func mustFailSolve(t *testing.T, client *GRPCClient, req *contract.SolveRequest) error {
	t.Helper()
	_, err := client.Solve(context.Background(), req)
	if err == nil {
		t.Fatal("expected solve failure")
	}
	return err
}
