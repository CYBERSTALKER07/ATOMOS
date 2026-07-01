package planning

import "testing"

func TestValidateAgentInvocation(t *testing.T) {
	if err := ValidateAgentInvocation(AgentInvocation{
		Action:         AgentApproveInsight,
		IdempotencyKey: "key-1",
		SupplierID:     "sup-1",
		TargetID:       "insight-1",
	}); err != nil {
		t.Fatalf("expected valid invocation: %v", err)
	}
	if err := ValidateAgentInvocation(AgentInvocation{
		Action:         "run_sql",
		IdempotencyKey: "key-1",
		SupplierID:     "sup-1",
		TargetID:       "x",
	}); err == nil {
		t.Fatal("expected denied action")
	}
	if err := ValidateAgentInvocation(AgentInvocation{
		Action:         AgentOpenSupplyRequest,
		IdempotencyKey: "key-1",
		SupplierID:     "sup-1",
	}); err == nil {
		t.Fatal("expected missing target_id")
	}
}
