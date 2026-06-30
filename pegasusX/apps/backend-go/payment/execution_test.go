package payment

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProviderExecutionRouter_DefaultGateway(t *testing.T) {
	t.Parallel()

	// Start a lightweight httptest server that mimics the Global Pay simulator
	// endpoints so the executor can complete CheckoutInit without a real server.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sim/globalpay/v1/merchant/auth":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"test-token"}`))
		case "/sim/globalpay/v1/user-service-tokens":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"tok","userRedirectUrl":"http://localhost/pay"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	exec := newGlobalPayProviderExecutorWithSimulator("dev", "svc", "", "", ts.URL)
	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{})
	router.SetExecutor("GLOBAL_PAY", exec)

	result, err := router.Execute(context.Background(), ExecutionRequest{
		Action:      ExecutionActionCheckoutInit,
		OrderID:     "order-1",
		AmountMinor: 10000,
		Currency:    "UZS",
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

func TestProviderExecutionRouter_Failover(t *testing.T) {
	t.Parallel()

	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{
		MaxAttempts: 1,
		BaseBackoff: time.Nanosecond,
		MaxBackoff:  time.Nanosecond,
	})
	execPrimary := &flakyExecutor{failuresBeforeSuccess: 5} // Always fails since MaxAttempts=1
	execFallback := &staticProviderExecutor{
		gateway:      "ADYEN",
		checkoutMode: ExecutionModeHostedRedirect,
		urlPrefix:    "/v1/payment/redirect/adyen/",
	}
	router.SetExecutor("GLOBAL_PAY", execPrimary)
	router.SetExecutor("ADYEN", execFallback)

	result, err := router.Execute(context.Background(), ExecutionRequest{
		Gateway:         "GLOBAL_PAY",
		FallbackGateway: "ADYEN",
		Action:          ExecutionActionCheckoutInit,
		OrderID:         "order-failover-1",
	})
	if err != nil {
		t.Fatalf("expected failover to succeed, got error: %v", err)
	}
	if result.ResolvedGateway != "ADYEN" {
		t.Fatalf("resolved gateway = %s, want ADYEN", result.ResolvedGateway)
	}
	if result.PolicySource != "ROUTER_FAILOVER" {
		t.Fatalf("policy source = %s, want ROUTER_FAILOVER", result.PolicySource)
	}
}

func TestProviderExecutionRouter_Failover_FallbackAlsoFails(t *testing.T) {
	t.Parallel()

	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{
		MaxAttempts: 1,
		BaseBackoff: time.Nanosecond,
		MaxBackoff:  time.Nanosecond,
	})
	execPrimary := &flakyExecutor{failuresBeforeSuccess: 5}
	execFallback := &flakyExecutor{failuresBeforeSuccess: 5}
	router.SetExecutor("GLOBAL_PAY", execPrimary)
	router.SetExecutor("ADYEN", execFallback)

	_, err := router.Execute(context.Background(), ExecutionRequest{
		Gateway:         "GLOBAL_PAY",
		FallbackGateway: "ADYEN",
		Action:          ExecutionActionCheckoutInit,
		OrderID:         "order-failover-2",
	})
	if err == nil {
		t.Fatal("expected failure after fallback exhausted")
	}
}
