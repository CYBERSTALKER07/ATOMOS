package foundry_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
)

func TestExecuteKineticAction_AuditHook_CommittedAndIdempotent(t *testing.T) {
	ctx := context.Background()
	eng := foundry.NewEngine()

	var mu sync.Mutex
	var recorded []*foundry.ActionExecutionRecord
	eng.SetAuditHook(func(_ context.Context, rec *foundry.ActionExecutionRecord) error {
		mu.Lock()
		defer mu.Unlock()
		recorded = append(recorded, rec)
		return nil
	})

	req := foundry.ActionExecutionRequest{
		CellID:         "cell-uz",
		SupplierID:     "sup_kinetic_1",
		ActionType:     "REORDER_APPROVE_TEST",
		ActorID:        "agent_planner_1",
		ActorRole:      "AGENT",
		IdempotencyKey: "idem_kinetic_hook_1",
		Parameters:     map[string]any{"sku": "SKU-1", "qty": 12},
	}

	first, err := eng.ExecuteKineticAction(ctx, req, func(context.Context) (map[string]any, error) {
		return map[string]any{"order_id": "ord_901"}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteKineticAction failed: %v", err)
	}
	second, err := eng.ExecuteKineticAction(ctx, req, func(context.Context) (map[string]any, error) {
		return map[string]any{"order_id": "ord_should_not_run"}, nil
	})
	if err != nil {
		t.Fatalf("idempotent replay failed: %v", err)
	}
	if first.ActionExecutionID != second.ActionExecutionID {
		t.Fatalf("expected same ActionExecutionID on replay, got %s vs %s", first.ActionExecutionID, second.ActionExecutionID)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(recorded) != 1 {
		t.Fatalf("expected audit hook invoked once on idempotent calls, got %d", len(recorded))
	}
	if recorded[0].Status != "COMMITTED" || recorded[0].ActorID != "agent_planner_1" {
		t.Fatalf("unexpected audit record: %+v", recorded[0])
	}
}

func TestExecuteKineticAction_AuditHook_RejectedAndErrorPropagation(t *testing.T) {
	ctx := context.Background()
	eng := foundry.NewEngine()
	eng.RegisterPrecondition("GUARDED_ACTION", func(context.Context, foundry.ActionExecutionRequest) foundry.PreconditionResult {
		return foundry.PreconditionResult{RuleName: "DenyTest", Passed: false, Reason: "blocked"}
	})

	var rejectedCount int
	eng.WithAuditHook(func(_ context.Context, rec *foundry.ActionExecutionRecord) error {
		if rec.Status == "REJECTED_PRECONDITION" {
			rejectedCount++
			return nil
		}
		return errors.New("audit unavailable")
	})

	_, err := eng.ExecuteKineticAction(ctx, foundry.ActionExecutionRequest{
		CellID:         "cell-uz",
		SupplierID:     "sup_kinetic_2",
		ActionType:     "GUARDED_ACTION",
		ActorID:        "system_worker",
		ActorRole:      "SYSTEM",
		IdempotencyKey: "idem_rejected_1",
	}, nil)
	if !errors.Is(err, foundry.ErrPreconditionFailed) {
		t.Fatalf("expected ErrPreconditionFailed, got %v", err)
	}
	if rejectedCount != 1 {
		t.Fatalf("expected rejected attempt audited once, got %d", rejectedCount)
	}

	_, err = eng.ExecuteKineticAction(ctx, foundry.ActionExecutionRequest{
		CellID:         "cell-uz",
		SupplierID:     "sup_kinetic_2",
		ActionType:     "UNGUARDED_ACTION",
		ActorID:        "system_worker",
		ActorRole:      "SYSTEM",
		IdempotencyKey: "idem_hook_fail_1",
	}, nil)
	if err == nil || err.Error() != "audit unavailable" {
		t.Fatalf("expected audit hook error propagation, got %v", err)
	}
}
