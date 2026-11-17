package planning

import "strings"

// GovernedAgentAction is an allowlisted mutation an agent may invoke.
type GovernedAgentAction string

const (
	AgentApproveInsight    GovernedAgentAction = "approve_insight"
	AgentOpenSupplyRequest GovernedAgentAction = "open_supply_request"
	AgentBroadcastTemplate GovernedAgentAction = "broadcast_template"
)

var governedAllowlist = map[GovernedAgentAction]bool{
	AgentApproveInsight:    true,
	AgentOpenSupplyRequest: true,
	AgentBroadcastTemplate: true,
}

// AgentInvocation is a governed hook request (no arbitrary SQL/LLM tools).
type AgentInvocation struct {
	Action         GovernedAgentAction `json:"action"`
	IdempotencyKey string              `json:"idempotency_key"`
	SupplierID     string              `json:"supplier_id"`
	TargetID       string              `json:"target_id"`
	Note           string              `json:"note,omitempty"`
}

// ValidateAgentInvocation rejects non-allowlisted agent actions.
func ValidateAgentInvocation(inv AgentInvocation) error {
	action := GovernedAgentAction(strings.TrimSpace(string(inv.Action)))
	if !governedAllowlist[action] {
		return ErrAgentActionDenied
	}
	if strings.TrimSpace(inv.IdempotencyKey) == "" || strings.TrimSpace(inv.SupplierID) == "" {
		return ErrAgentInvocationInvalid
	}
	if strings.TrimSpace(inv.TargetID) == "" && inv.Action != AgentBroadcastTemplate {
		return ErrAgentInvocationInvalid
	}
	return nil
}

// ValidateAgentGovernance enforces that autonomous agents may only propose actions
// and never approve or self-execute governed proposals.
func ValidateAgentGovernance(actorType string, isApprovalOrExecute bool) error {
	if isApprovalOrExecute && strings.EqualFold(strings.TrimSpace(actorType), "agent") {
		return ErrAgentSelfApproveDenied
	}
	return nil
}

// ErrAgentActionDenied is returned when an agent requests a non-allowlisted mutation.
var ErrAgentActionDenied = errDenied("agent action not allowlisted")

// ErrAgentInvocationInvalid is returned when required agent fields are missing.
var ErrAgentInvocationInvalid = errDenied("agent invocation invalid")

// ErrAgentSelfApproveDenied is returned when an agent attempts to approve or execute a proposal.
var ErrAgentSelfApproveDenied = errDenied("agent actors may only propose, never approve or execute")

type errDenied string

func (e errDenied) Error() string { return string(e) }
