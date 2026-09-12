package controltower

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type mockRepo struct {
	playbooks  []Playbook
	exceptions []Exception
	runs       map[string]PlaybookRun
	blocking   map[string]bool
}

func (m *mockRepo) ListActivePlaybooks(ctx context.Context, supplierID string) ([]Playbook, error) {
	return m.playbooks, nil
}
func (m *mockRepo) GetPlaybook(ctx context.Context, playbookID string) (Playbook, error) {
	for _, p := range m.playbooks {
		if p.PlaybookID == playbookID {
			return p, nil
		}
	}
	return Playbook{}, context.Canceled
}
func (m *mockRepo) CreatePlaybook(ctx context.Context, pb Playbook) error { return nil }
func (m *mockRepo) UpdatePlaybook(ctx context.Context, playbookID string, fields map[string]any) error {
	return nil
}
func (m *mockRepo) DeactivatePlaybook(ctx context.Context, playbookID string) error { return nil }
func (m *mockRepo) ListOpenExceptions(ctx context.Context, supplierID string) ([]Exception, error) {
	return m.exceptions, nil
}
func (m *mockRepo) ListSupplierIDsWithOpenExceptions(ctx context.Context) ([]string, error) {
	return []string{"sup-1"}, nil
}
func (m *mockRepo) HasBlockingRun(ctx context.Context, exceptionID string) (bool, error) {
	return m.blocking[exceptionID], nil
}
func (m *mockRepo) CreateRun(ctx context.Context, run PlaybookRun) error {
	if m.runs == nil {
		m.runs = map[string]PlaybookRun{}
	}
	m.runs[run.RunID] = run
	return nil
}
func (m *mockRepo) GetRun(ctx context.Context, runID string) (PlaybookRun, error) {
	return m.runs[runID], nil
}
func (m *mockRepo) UpdateRun(ctx context.Context, run PlaybookRun) error {
	m.runs[run.RunID] = run
	return nil
}
func (m *mockRepo) ListRuns(ctx context.Context, supplierID, status string, limit int) ([]PlaybookRun, error) {
	var out []PlaybookRun
	for _, r := range m.runs {
		if r.SupplierID == supplierID && (status == "" || r.Status == status) {
			out = append(out, r)
		}
	}
	return out, nil
}
func (m *mockRepo) ListRunsForException(ctx context.Context, exceptionID string) ([]PlaybookRun, error) {
	return nil, nil
}
func (m *mockRepo) UpdateExceptionStatus(ctx context.Context, exceptionID, newStatus string) error {
	return nil
}
func (m *mockRepo) UpdateExceptionAssignee(ctx context.Context, exceptionID, role string) error {
	return nil
}

type mockExecutor struct {
	calls int
}

func (m *mockExecutor) ExecuteAction(ctx context.Context, runID string, idx int, spec ActionSpec, ex Exception, actor string) ActionResult {
	m.calls++
	return ActionResult{Index: idx, Type: spec.Type, Status: "ok"}
}

func TestEngine_EvaluateCreatesSuggestedRun(t *testing.T) {
	matchRaw, _ := json.Marshal(MatchRules{Types: []string{"BUYER_REJECTED"}})
	actionsRaw, _ := json.Marshal([]ActionSpec{{Type: "ACKNOWLEDGE_EXCEPTION"}})
	repo := &mockRepo{
		playbooks: []Playbook{{
			PlaybookID: "pb-1", Priority: 10, IsActive: true,
			MatchRules:    MatchRules{Types: []string{"BUYER_REJECTED"}},
			MatchRulesRaw: matchRaw,
			Actions:       []ActionSpec{{Type: "ACKNOWLEDGE_EXCEPTION"}},
			ActionsRaw:    actionsRaw,
		}},
		exceptions: []Exception{{
			ExceptionID: "exc-1", Type: "BUYER_EHF_REJECTION", SupplierID: "sup-1", OrderID: "ord-1",
			CreatedAt: time.Now(),
		}},
		runs: map[string]PlaybookRun{},
	}
	exec := NewActionExecutor(repo, ExecutorDeps{})
	engine := NewEngine(repo, exec, nil, Config{Enabled: true}, nil)
	if err := engine.Evaluate(context.Background(), "sup-1", nil); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(repo.runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(repo.runs))
	}
	for _, run := range repo.runs {
		if run.Status != RunStatusSuggested {
			t.Fatalf("status=%s want SUGGESTED", run.Status)
		}
	}
}

func TestEngine_DisabledNoOp(t *testing.T) {
	repo := &mockRepo{
		playbooks: []Playbook{{PlaybookID: "pb-1", MatchRules: MatchRules{Types: []string{"BUYER_REJECTED"}}}},
		exceptions: []Exception{{ExceptionID: "exc-1", Type: "BUYER_EHF_REJECTION", SupplierID: "sup-1", CreatedAt: time.Now()}},
		runs: map[string]PlaybookRun{},
	}
	engine := NewEngine(repo, NewActionExecutor(repo, ExecutorDeps{}), nil, Config{Enabled: false}, nil)
	if err := engine.Evaluate(context.Background(), "sup-1", nil); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(repo.runs) != 0 {
		t.Fatal("expected no runs when disabled")
	}
}

func TestApproveRun_NoIntermediateApprovedWrite(t *testing.T) {
	// ApproveRun must not leave an APPROVED-only update; only terminal EXECUTED/FAILED.
	actionsRaw, _ := json.Marshal([]ActionSpec{{Type: "ACKNOWLEDGE_EXCEPTION"}})
	runID := "run-approve-1"
	repo := &mockRepo{
		playbooks: []Playbook{{
			PlaybookID: "pb-1", Priority: 10, IsActive: true,
			Actions:    []ActionSpec{{Type: "ACKNOWLEDGE_EXCEPTION"}},
			ActionsRaw: actionsRaw,
		}},
		exceptions: []Exception{{
			ExceptionID: "exc-1", Type: "BUYER_EHF_REJECTION", SupplierID: "sup-1", OrderID: "ord-1",
			CreatedAt: time.Now(),
		}},
		runs: map[string]PlaybookRun{
			runID: {
				RunID: runID, PlaybookID: "pb-1", ExceptionID: "exc-1",
				SupplierID: "sup-1", Status: RunStatusSuggested, Mode: RunModeSuggested,
			},
		},
	}
	engine := NewEngine(repo, NewActionExecutor(repo, ExecutorDeps{}), nil, Config{Enabled: true}, nil)
	if err := engine.ApproveRun(context.Background(), runID, "ops-1"); err != nil {
		t.Fatalf("ApproveRun: %v", err)
	}
	got := repo.runs[runID]
	if got.Status != RunStatusExecuted {
		t.Fatalf("status=%s want EXECUTED (no sticky APPROVED intermediate)", got.Status)
	}
}

func TestEngine_AutoExecuteStillSuggestedWhenGlobalAutoOff(t *testing.T) {
	matchRaw, _ := json.Marshal(MatchRules{Types: []string{"CASH_SHORT"}})
	actionsRaw, _ := json.Marshal([]ActionSpec{{Type: "NOTIFY"}})
	repo := &mockRepo{
		playbooks: []Playbook{{
			PlaybookID: "pb-auto", Priority: 10, AutoExecute: true,
			MatchRules:    MatchRules{Types: []string{"CASH_SHORT"}},
			MatchRulesRaw: matchRaw,
			Actions:       []ActionSpec{{Type: "NOTIFY"}},
			ActionsRaw:    actionsRaw,
		}},
		exceptions: []Exception{{ExceptionID: "exc-2", Type: "CASH_SHORTFALL", SupplierID: "sup-1", CreatedAt: time.Now()}},
		runs: map[string]PlaybookRun{},
	}
	engine := NewEngine(repo, NewActionExecutor(repo, ExecutorDeps{}), nil, Config{Enabled: true, AutoExecute: false}, nil)
	if err := engine.Evaluate(context.Background(), "sup-1", nil); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	for _, run := range repo.runs {
		if run.Mode != RunModeSuggested {
			t.Fatalf("mode=%s want SUGGESTED when global auto off", run.Mode)
		}
	}
}
