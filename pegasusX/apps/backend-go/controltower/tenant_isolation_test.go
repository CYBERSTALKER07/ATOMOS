package controltower

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUpdatePlaybook_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{
		playbooks: []Playbook{
			{PlaybookID: "pb-own", SupplierID: "sup-1", Name: "Own Playbook", IsActive: true},
			{PlaybookID: "pb-other", SupplierID: "sup-2", Name: "Competitor Playbook", IsActive: true},
			{PlaybookID: "pb-platform", SupplierID: "", Name: "Platform Global Playbook", IsActive: true},
		},
	}
	svc := NewService(repo, nil, Config{Enabled: true})

	newName := "Hacked Name"
	in := UpdatePlaybookInput{Name: &newName}

	// 1. Cross-tenant update must return ErrForbidden
	err := svc.UpdatePlaybook(ctx, "sup-1", "pb-other", in)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-tenant update: got err %v, want %v", err, ErrForbidden)
	}

	// 2. Platform playbook update by tenant admin must return ErrForbidden
	err = svc.UpdatePlaybook(ctx, "sup-1", "pb-platform", in)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("platform playbook update: got err %v, want %v", err, ErrForbidden)
	}

	// 3. Non-existent playbook returns ErrPlaybookNotFound
	err = svc.UpdatePlaybook(ctx, "sup-1", "pb-nonexistent", in)
	if !errors.Is(err, ErrPlaybookNotFound) {
		t.Fatalf("non-existent playbook: got err %v, want %v", err, ErrPlaybookNotFound)
	}

	// 4. Legitimate own playbook update succeeds
	legitName := "Updated Legitimate Name"
	err = svc.UpdatePlaybook(ctx, "sup-1", "pb-own", UpdatePlaybookInput{Name: &legitName})
	if err != nil {
		t.Fatalf("legitimate own playbook update failed: %v", err)
	}
	pb, _ := repo.GetPlaybook(ctx, "pb-own")
	if pb.Name != legitName {
		t.Fatalf("expected name %q, got %q", legitName, pb.Name)
	}
}

func TestDeactivatePlaybook_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{
		playbooks: []Playbook{
			{PlaybookID: "pb-own", SupplierID: "sup-1", Name: "Own Playbook", IsActive: true},
			{PlaybookID: "pb-other", SupplierID: "sup-2", Name: "Competitor Playbook", IsActive: true},
			{PlaybookID: "pb-platform", SupplierID: "", Name: "Platform Global Playbook", IsActive: true},
		},
	}
	svc := NewService(repo, nil, Config{Enabled: true})

	// 1. Cross-tenant deactivation must return ErrForbidden
	err := svc.DeactivatePlaybook(ctx, "sup-1", "pb-other")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-tenant deactivate: got err %v, want %v", err, ErrForbidden)
	}

	// 2. Platform playbook deactivation by tenant admin must return ErrForbidden
	err = svc.DeactivatePlaybook(ctx, "sup-1", "pb-platform")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("platform playbook deactivate: got err %v, want %v", err, ErrForbidden)
	}

	// 3. Legitimate own playbook deactivation succeeds
	err = svc.DeactivatePlaybook(ctx, "sup-1", "pb-own")
	if err != nil {
		t.Fatalf("legitimate own playbook deactivation failed: %v", err)
	}
	pb, _ := repo.GetPlaybook(ctx, "pb-own")
	if pb.IsActive {
		t.Fatal("expected pb-own IsActive to be false")
	}
}

func TestApproveAndSkipRun_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{
		playbooks: []Playbook{
			{PlaybookID: "pb-1", SupplierID: "sup-1", Priority: 10, IsActive: true, Actions: []ActionSpec{{Type: "ACKNOWLEDGE_EXCEPTION"}}},
		},
		exceptions: []Exception{
			{ExceptionID: "exc-1", SupplierID: "sup-1", OrderID: "ord-1", CreatedAt: time.Now()},
		},
		runs: map[string]PlaybookRun{
			"run-own": {
				RunID: "run-own", PlaybookID: "pb-1", ExceptionID: "exc-1",
				SupplierID: "sup-1", Status: RunStatusSuggested, Mode: RunModeSuggested,
			},
			"run-other": {
				RunID: "run-other", PlaybookID: "pb-1", ExceptionID: "exc-1",
				SupplierID: "sup-2", Status: RunStatusSuggested, Mode: RunModeSuggested,
			},
			"run-platform": {
				RunID: "run-platform", PlaybookID: "pb-1", ExceptionID: "exc-1",
				SupplierID: "", Status: RunStatusSuggested, Mode: RunModeSuggested,
			},
		},
	}
	engine := NewEngine(repo, NewActionExecutor(repo, ExecutorDeps{}), nil, Config{Enabled: true}, nil)
	svc := NewService(repo, engine, Config{Enabled: true})

	// 1. Cross-tenant approve rejected
	err := svc.ApproveRun(ctx, "sup-1", "run-other", "attacker")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-tenant approve: got err %v, want %v", err, ErrForbidden)
	}

	// 2. Platform run approve by tenant admin rejected
	err = svc.ApproveRun(ctx, "sup-1", "run-platform", "attacker")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("platform run approve: got err %v, want %v", err, ErrForbidden)
	}

	// 3. Cross-tenant skip rejected
	err = svc.SkipRun(ctx, "sup-1", "run-other", "attacker")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-tenant skip: got err %v, want %v", err, ErrForbidden)
	}

	// 4. Platform run skip rejected
	err = svc.SkipRun(ctx, "sup-1", "run-platform", "attacker")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("platform run skip: got err %v, want %v", err, ErrForbidden)
	}

	// 5. Non-existent run returns ErrRunNotFound
	err = svc.ApproveRun(ctx, "sup-1", "run-404", "ops")
	if !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("nonexistent run: got err %v, want %v", err, ErrRunNotFound)
	}

	// 6. Legitimate own run skip succeeds
	err = svc.SkipRun(ctx, "sup-1", "run-own", "ops-1")
	if err != nil {
		t.Fatalf("legitimate skip failed: %v", err)
	}
	run, _ := repo.GetRun(ctx, "run-own")
	if run.Status != RunStatusSkipped {
		t.Fatalf("expected RunStatusSkipped, got %s", run.Status)
	}

	// 7. Legitimate own run approve succeeds (reset status to suggested first)
	run.Status = RunStatusSuggested
	_ = repo.UpdateRun(ctx, run)
	err = svc.ApproveRun(ctx, "sup-1", "run-own", "ops-1")
	if err != nil {
		t.Fatalf("legitimate approve failed: %v", err)
	}
	run, _ = repo.GetRun(ctx, "run-own")
	if run.Status != RunStatusExecuted {
		t.Fatalf("expected RunStatusExecuted, got %s", run.Status)
	}
}

func TestListRunsForException_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{
		runs: map[string]PlaybookRun{
			"r1": {RunID: "r1", ExceptionID: "exc-100", SupplierID: "sup-1", Status: RunStatusSuggested},
			"r2": {RunID: "r2", ExceptionID: "exc-100", SupplierID: "sup-2", Status: RunStatusExecuted},
			"r3": {RunID: "r3", ExceptionID: "exc-200", SupplierID: "sup-1", Status: RunStatusExecuted},
		},
	}
	svc := NewService(repo, nil, Config{Enabled: true})

	// Supplier 1 query for exc-100 should only return r1
	runs1, err := svc.ListRunsForException(ctx, "sup-1", "exc-100")
	if err != nil {
		t.Fatalf("ListRunsForException failed: %v", err)
	}
	if len(runs1) != 1 || runs1[0].RunID != "r1" {
		t.Fatalf("expected only r1, got %v", runs1)
	}

	// Supplier 2 query for exc-100 should only return r2
	runs2, err := svc.ListRunsForException(ctx, "sup-2", "exc-100")
	if err != nil {
		t.Fatalf("ListRunsForException failed: %v", err)
	}
	if len(runs2) != 1 || runs2[0].RunID != "r2" {
		t.Fatalf("expected only r2, got %v", runs2)
	}

	// Empty supplier ID returns ErrForbidden
	_, err = svc.ListRunsForException(ctx, "", "exc-100")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for empty supplier, got %v", err)
	}
}
