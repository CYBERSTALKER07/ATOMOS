package payment

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProviderExecutionRouter_StripeIsNotTheatre(t *testing.T) {
	t.Parallel()
	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{})
	result, err := router.Execute(context.Background(), ExecutionRequest{
		Gateway: "STRIPE",
		Action:  ExecutionActionCheckoutInit,
		OrderID: "ord-stripe",
	})
	if result.RedirectURL != "" {
		t.Fatalf("redirect=%q", result.RedirectURL)
	}
	var policy *GatewayPolicyError
	if !errors.As(err, &policy) || policy.Code != "adapter_planned" {
		t.Fatalf("err=%v", err)
	}
}

func TestProviderExecutionRouter_AdyenIsNotTheatre(t *testing.T) {
	t.Parallel()
	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{})
	result, err := router.Execute(context.Background(), ExecutionRequest{
		Gateway: "ADYEN",
		Action:  ExecutionActionCheckoutInit,
		OrderID: "ord-adyen",
	})
	if result.RedirectURL != "" {
		t.Fatalf("redirect=%q", result.RedirectURL)
	}
	var policy *GatewayPolicyError
	if !errors.As(err, &policy) || policy.Code != "adapter_planned" {
		t.Fatalf("err=%v", err)
	}
}

func TestProviderExecutionRouter_GlobalPayUnkeyed(t *testing.T) {
	t.Setenv("GLOBAL_PAY_STUB_MODE", "")
	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{})
	result, err := router.Execute(context.Background(), ExecutionRequest{
		Gateway: "GLOBAL_PAY",
		Action:  ExecutionActionCheckoutInit,
		OrderID: "ord-gp-unkeyed",
	})
	if result.RedirectURL != "" {
		t.Fatalf("redirect=%q", result.RedirectURL)
	}
	var policy *GatewayPolicyError
	if !errors.As(err, &policy) || policy.Code != "no_live_keys" {
		t.Fatalf("err=%v", err)
	}
	if !errors.Is(err, errGlobalPayCredentialsMissing) {
		t.Fatalf("unwrap sentinel: %v", err)
	}
	svc := &Service{}
	rr := httptest.NewRecorder()
	svc.writeExecutionError(rr, "/v1/order/card-checkout", err)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestProviderExecutionRouter_PaymeUnkeyed(t *testing.T) {
	t.Parallel()
	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{})
	_, err := router.Execute(context.Background(), ExecutionRequest{
		Gateway: "PAYME",
		Action:  ExecutionActionCheckoutInit,
		OrderID: "ord-payme",
	})
	var policy *GatewayPolicyError
	if !errors.As(err, &policy) || policy.Code != "no_live_keys" {
		t.Fatalf("err=%v", err)
	}
	if _, ok := router.executors["PAYME"].(*catalogHonestyExecutor); !ok {
		t.Fatalf("PAYME must stay catalogHonestyExecutor, got %T", router.executors["PAYME"])
	}
	if _, ok := router.executors["PAYME"].(*paymeProviderExecutor); ok {
		t.Fatal("PAYME executor must not be wired")
	}
}

func TestProviderExecutionRouter_ClickUnkeyed(t *testing.T) {
	t.Parallel()
	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{})
	_, err := router.Execute(context.Background(), ExecutionRequest{
		Gateway: "CLICK",
		Action:  ExecutionActionCheckoutInit,
		OrderID: "ord-click",
	})
	var policy *GatewayPolicyError
	if !errors.As(err, &policy) || policy.Code != "no_live_keys" {
		t.Fatalf("err=%v", err)
	}
	if _, ok := router.executors["CLICK"].(*catalogHonestyExecutor); !ok {
		t.Fatalf("CLICK must stay catalogHonestyExecutor, got %T", router.executors["CLICK"])
	}
	if _, ok := router.executors["CLICK"].(*clickProviderExecutor); ok {
		t.Fatal("CLICK executor must not be wired")
	}
}

func TestProviderExecutionRouter_EmptyGatewayUsesPackNotLiteral(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	got, err := resolveDefaultCardGateway(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != "GLOBAL_PAY" {
		t.Fatalf("default=%q", got)
	}
}

func TestWriteExecutionError_NoLiveKeysIs501(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	rr := httptest.NewRecorder()
	svc.writeExecutionError(rr, "/v1/payment/checkout", &GatewayPolicyError{Code: "no_live_keys", Message: "no_live_keys"})
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestDefaultRouter_NoStripeUrlPrefix(t *testing.T) {
	t.Parallel()
	router := NewProviderExecutionRouter(ProviderExecutionRouterConfig{})
	if _, ok := router.executors["STRIPE"].(*staticProviderExecutor); ok {
		t.Fatal("STRIPE must not be a redirect theatre executor")
	}
	if _, ok := router.executors["ADYEN"].(*staticProviderExecutor); ok {
		t.Fatal("ADYEN must not be a redirect theatre executor")
	}
}
