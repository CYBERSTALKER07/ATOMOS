package payment

import (
	"context"
	"errors"
	"testing"
)

// Phase-0 money-path gate (c): empty Global Pay credentials are hard errors.
// Fabricated stub success is only possible in non-production with explicit
// opt-in (GLOBAL_PAY_STUB_MODE or test-only allowStub); production never stubs.
func TestMoneyPathGate_GlobalPayEmptyCredentialsHardError(t *testing.T) {
	t.Setenv("GLOBAL_PAY_STUB_MODE", "")

	cases := []struct {
		name string
		env  string
	}{
		{name: "production", env: "production"},
		{name: "staging", env: "staging"},
		{name: "dev", env: "dev"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exec := newGlobalPayProviderExecutor(tc.env, "", "", "")
			_, err := exec.Execute(context.Background(), ExecutionRequest{
				Action:      ExecutionActionCheckoutInit,
				OrderID:     "order-gate-c",
				AmountMinor: 1000,
				Currency:    "UZS",
			})
			if !errors.Is(err, errGlobalPayCredentialsMissing) {
				t.Fatalf("env=%s err=%v, want errGlobalPayCredentialsMissing", tc.env, err)
			}
		})
	}
}

// Production must never stub, even if someone exports GLOBAL_PAY_STUB_MODE=true
// into a prod pod by mistake.
func TestMoneyPathGate_GlobalPayProductionNeverStubs(t *testing.T) {
	t.Setenv("GLOBAL_PAY_STUB_MODE", "true")

	exec := newGlobalPayProviderExecutor("production", "", "", "")
	_, err := exec.Execute(context.Background(), ExecutionRequest{
		Action:      ExecutionActionCheckoutInit,
		OrderID:     "order-gate-c-prod",
		AmountMinor: 1000,
		Currency:    "UZS",
	})
	if !errors.Is(err, errGlobalPayCredentialsMissing) {
		t.Fatalf("production with GLOBAL_PAY_STUB_MODE=true still stubs: err=%v", err)
	}
}
