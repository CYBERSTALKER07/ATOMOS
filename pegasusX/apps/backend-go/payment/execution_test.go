package payment

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProviderExecutionRouter_DefaultGateway(t *testing.T) {
	t.Parallel()

	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{})
	result, err := router.Execute(context.Background(), ExecutionRequest{
		Action:  ExecutionActionCheckoutInit,
		OrderID: "order-1",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.ResolvedGateway != "GLOBAL_PAY" {
		t.Fatalf("resolved gateway = %s, want GLOBAL_PAY", result.ResolvedGateway)
	}
	if result.Mode != ExecutionModeHostedRedirect {
		t.Fatalf("mode = %s, want %s", result.Mode, ExecutionModeHostedRedirect)
	}
}

func TestProviderExecutionRouter_AirwallexDisabled(t *testing.T) {
	t.Parallel()

	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{AirwallexDirectExecutionEnabled: false})
	_, err := router.Execute(context.Background(), ExecutionRequest{
		Gateway: "AIRWALLEX",
		Action:  ExecutionActionCheckoutInit,
		OrderID: "order-2",
	})
	if err == nil {
		t.Fatal("expected policy error")
	}
	var policyErr *GatewayPolicyError
	if !errors.As(err, &policyErr) {
		t.Fatalf("expected GatewayPolicyError, got %T", err)
	}
	if policyErr.Code != "card_tokenization_gateway_unsupported" {
		t.Fatalf("code = %s, want card_tokenization_gateway_unsupported", policyErr.Code)
	}
}

func TestProviderExecutionRouter_RetryableFailureRetries(t *testing.T) {
	t.Parallel()

	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{
		MaxAttempts: 3,
		BaseBackoff: time.Nanosecond,
		MaxBackoff:  time.Nanosecond,
	})
	exec := &flakyExecutor{failuresBeforeSuccess: 2}
	router.SetExecutor("GLOBAL_PAY", exec)

	result, err := router.Execute(context.Background(), ExecutionRequest{
		Gateway: "GLOBAL_PAY",
		Action:  ExecutionActionCheckoutInit,
		OrderID: "order-3",
	})
	if err != nil {
		t.Fatalf("execute with retry: %v", err)
	}
	if exec.calls != 3 {
		t.Fatalf("executor calls = %d, want 3", exec.calls)
	}
	if result.ResolvedGateway != "GLOBAL_PAY" {
		t.Fatalf("resolved gateway = %s, want GLOBAL_PAY", result.ResolvedGateway)
	}
}

func TestProviderExecutionRouter_UnsupportedGateway(t *testing.T) {
	t.Parallel()

	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{})
	_, err := router.Execute(context.Background(), ExecutionRequest{
		Gateway: "UNKNOWN",
		Action:  ExecutionActionCheckoutInit,
		OrderID: "order-4",
	})
	if err == nil {
		t.Fatal("expected policy error")
	}
	var policyErr *GatewayPolicyError
	if !errors.As(err, &policyErr) {
		t.Fatalf("expected GatewayPolicyError, got %T", err)
	}
	if policyErr.Code != "payment_gateway_policy_violation" {
		t.Fatalf("code = %s, want payment_gateway_policy_violation", policyErr.Code)
	}
}

type flakyExecutor struct {
	failuresBeforeSuccess int
	calls                 int
}

func (f *flakyExecutor) Execute(_ context.Context, req ExecutionRequest) (ExecutionResult, error) {
	f.calls++
	if f.calls <= f.failuresBeforeSuccess {
		return ExecutionResult{}, MarkRetryable(errors.New("temporary provider failure"))
	}
	return ExecutionResult{
		ResolvedGateway: req.Gateway,
		Mode:            ExecutionModeHostedRedirect,
		PolicySource:    "SUPPLIER_DEFAULT",
		RedirectURL:     "/redirect/success",
	}, nil
}
