package foundry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PreconditionResult captures the outcome of a single pre-execution rubric check.
type PreconditionResult struct {
	RuleName string `json:"rule_name"`
	Passed   bool   `json:"passed"`
	Reason   string `json:"reason,omitempty"`
}

// PreconditionEvaluator validates whether a Kinetic Action is allowed to execute.
type PreconditionEvaluator func(ctx context.Context, req ActionExecutionRequest) PreconditionResult

// ActionExecutionRequest defines the inputs to a Kinetic Action contract.
type ActionExecutionRequest struct {
	CellID         string         `json:"cell_id"`
	SupplierID     string         `json:"supplier_id"`
	ActionType     string         `json:"action_type"`
	ActorID        string         `json:"actor_id"`
	ActorRole      string         `json:"actor_role"`
	ScenarioID     string         `json:"scenario_id,omitempty"`
	IdempotencyKey string         `json:"idempotency_key"`
	Parameters     map[string]any `json:"parameters"`
}

// ActionExecutionRecord is the immutable audit log entry written for every Kinetic Action.
type ActionExecutionRecord struct {
	ActionExecutionID      string               `json:"action_execution_id" spanner:"ActionExecutionId"`
	CellID                 string               `json:"cell_id" spanner:"CellId"`
	SupplierID             string               `json:"supplier_id" spanner:"SupplierId"`
	ActionType             string               `json:"action_type" spanner:"ActionType"`
	ActorID                string               `json:"actor_id" spanner:"ActorId"`
	ActorRole              string               `json:"actor_role" spanner:"ActorRole"`
	ScenarioID             *string              `json:"scenario_id,omitempty" spanner:"ScenarioId"`
	Status                 string               `json:"status" spanner:"Status"` // COMMITTED | SIMULATED_IN_SCENARIO | REJECTED_PRECONDITION
	PreconditionReport     []PreconditionResult `json:"precondition_report,omitempty" spanner:"-"`
	PreconditionReportJson string               `json:"precondition_report_json,omitempty" spanner:"PreconditionReportJson"`
	DiffPayload            map[string]any       `json:"diff_payload,omitempty" spanner:"-"`
	DiffPayloadJson        string               `json:"diff_payload_json,omitempty" spanner:"DiffPayloadJson"`
	IdempotencyKey         string               `json:"idempotency_key" spanner:"IdempotencyKey"`
	ExecutedAt             time.Time            `json:"executed_at" spanner:"ExecutedAt"`
}

type KineticActionExecution = ActionExecutionRecord
type KineticActionExecutions = ActionExecutionRecord

func (k *ActionExecutionRecord) PrepareForSpanner() error {
	if k.PreconditionReportJson == "" && k.PreconditionReport != nil {
		b, err := json.Marshal(k.PreconditionReport)
		if err != nil {
			return err
		}
		k.PreconditionReportJson = string(b)
	}
	if k.DiffPayloadJson == "" && k.DiffPayload != nil {
		b, err := json.Marshal(k.DiffPayload)
		if err != nil {
			return err
		}
		k.DiffPayloadJson = string(b)
	}
	return nil
}

// RegisterPrecondition attaches a mandatory pre-execution guard to an ActionType.
func (e *Engine) RegisterPrecondition(actionType string, evaluator PreconditionEvaluator) {
	actionType = strings.TrimSpace(actionType)
	if actionType == "" || evaluator == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.preconditions[actionType] = append(e.preconditions[actionType], evaluator)
}

// ExecuteKineticAction evaluates all registered pre-execution rubrics for the ActionType,
// enforces idempotency, executes the state mutation closure, and records an immutable audit log.
func (e *Engine) ExecuteKineticAction(
	ctx context.Context,
	req ActionExecutionRequest,
	mutateFn func(ctx context.Context) (map[string]any, error),
) (*ActionExecutionRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	actionType := strings.TrimSpace(req.ActionType)
	if actionType == "" {
		return nil, fmt.Errorf("%w: action_type is required", ErrPreconditionFailed)
	}
	idemKey := strings.TrimSpace(req.IdempotencyKey)
	if idemKey == "" {
		return nil, fmt.Errorf("%w: idempotency_key is required", ErrPreconditionFailed)
	}

	e.mu.RLock()
	if existing, ok := e.actionsByKey[idemKey]; ok {
		e.mu.RUnlock()
		return existing, nil
	}
	evaluators := append([]PreconditionEvaluator(nil), e.preconditions[actionType]...)
	e.mu.RUnlock()

	var scenPtr *string
	if s := strings.TrimSpace(req.ScenarioID); s != "" {
		scenPtr = &s
	}

	var report []PreconditionResult
	for _, eval := range evaluators {
		res := eval(ctx, req)
		report = append(report, res)
		if !res.Passed {
			rejected := &ActionExecutionRecord{
				ActionExecutionID:  uuid.NewString(),
				CellID:             req.CellID,
				SupplierID:         req.SupplierID,
				ActionType:         actionType,
				ActorID:            req.ActorID,
				ActorRole:          req.ActorRole,
				ScenarioID:         scenPtr,
				Status:             "REJECTED_PRECONDITION",
				PreconditionReport: report,
				DiffPayload:        map[string]any{"rejected_by": res.RuleName, "reason": res.Reason},
				IdempotencyKey:     idemKey,
				ExecutedAt:         e.now(),
			}
			e.mu.Lock()
			e.actionHistory = append(e.actionHistory, rejected)
			if len(e.actionHistory) > 1000 {
				e.actionHistory = e.actionHistory[len(e.actionHistory)-1000:]
			}
			e.mu.Unlock()
			return rejected, fmt.Errorf("%w: [%s] %s", ErrPreconditionFailed, res.RuleName, res.Reason)
		}
	}

	var diff map[string]any
	if mutateFn != nil {
		var err error
		diff, err = mutateFn(ctx)
		if err != nil {
			return nil, err
		}
	}
	if diff == nil {
		diff = make(map[string]any)
	}

	status := "COMMITTED"
	if scenPtr != nil {
		status = "SIMULATED_IN_SCENARIO"
	}

	record := &ActionExecutionRecord{
		ActionExecutionID:  uuid.NewString(),
		CellID:             req.CellID,
		SupplierID:         req.SupplierID,
		ActionType:         actionType,
		ActorID:            req.ActorID,
		ActorRole:          req.ActorRole,
		ScenarioID:         scenPtr,
		Status:             status,
		PreconditionReport: report,
		DiffPayload:        diff,
		IdempotencyKey:     idemKey,
		ExecutedAt:         e.now(),
	}

	repo := e.spannerRepo
	if repo != nil {
		if err := repo.PersistKineticActionExecution(ctx, record); err != nil {
			return nil, err
		}
	}

	e.mu.Lock()
	if existing, ok := e.actionsByKey[idemKey]; ok {
		e.mu.Unlock()
		return existing, nil
	}
	e.actionsByKey[idemKey] = record
	e.actionHistory = append(e.actionHistory, record)
	if len(e.actionHistory) > 1000 {
		e.actionHistory = e.actionHistory[len(e.actionHistory)-1000:]
	}
	e.mu.Unlock()
	return record, nil
}
