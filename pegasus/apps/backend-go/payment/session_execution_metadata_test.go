package payment

import "testing"

func TestNormalizeExecutionAction_Default(t *testing.T) {
	if got := normalizeExecutionAction(""); got != AttemptExecutionActionCheckoutInit {
		t.Fatalf("normalizeExecutionAction(\"\") = %q, want %q", got, AttemptExecutionActionCheckoutInit)
	}
}

func TestNormalizeExecutionAction_UpperTrimmed(t *testing.T) {
	if got := normalizeExecutionAction("  hosted_checkout_init  "); got != AttemptExecutionActionHostedCheckoutInit {
		t.Fatalf("normalizeExecutionAction(hosted_checkout_init) = %q, want %q", got, AttemptExecutionActionHostedCheckoutInit)
	}
}

func TestNormalizeExecutionMode_Default(t *testing.T) {
	if got := normalizeExecutionMode(""); got != AttemptExecutionModeAuto {
		t.Fatalf("normalizeExecutionMode(\"\") = %q, want %q", got, AttemptExecutionModeAuto)
	}
}

func TestNormalizeExecutionMode_DirectStoredMethod(t *testing.T) {
	if got := normalizeExecutionMode(" direct_stored_method "); got != AttemptExecutionModeDirectStoredMethod {
		t.Fatalf("normalizeExecutionMode(direct_stored_method) = %q, want %q", got, AttemptExecutionModeDirectStoredMethod)
	}
}

func TestNormalizeExecutionMode_Direct3DSRedirect(t *testing.T) {
	if got := normalizeExecutionMode(" direct_3ds_redirect "); got != AttemptExecutionModeDirect3DSRedirect {
		t.Fatalf("normalizeExecutionMode(direct_3ds_redirect) = %q, want %q", got, AttemptExecutionModeDirect3DSRedirect)
	}
}
