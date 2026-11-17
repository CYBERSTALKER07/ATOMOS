package iam

import (
	"context"
	"errors"
	"sync"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type spyAuditRecorder struct {
	mu     sync.Mutex
	events []AuditRecordInput
}

func (s *spyAuditRecorder) RecordEvent(_ context.Context, ev AuditRecordInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
	return nil
}

func (s *spyAuditRecorder) RecordEventInTxn(_ context.Context, _ *spanner.ReadWriteTransaction, ev AuditRecordInput) error {
	return s.RecordEvent(context.Background(), ev)
}

func TestService_DefaultSeedAndStaticRoleSoD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := NewMemoryRepository()
	auditSpy := &spyAuditRecorder{}
	svc := NewServiceWithRepository(repo)
	svc.SetAuditRecorder(auditSpy)

	// 1. Default seeded 5 SoD pairs + 9 roles + 23 permissions.
	pols, err := svc.ListSodPolicies(ctx, "sup-1")
	if err != nil || len(pols) != 5 {
		t.Fatalf("ListSodPolicies = (%d, %v), want 5", len(pols), err)
	}
	roles, _, err := svc.ListRoles(ctx, "sup-1")
	if err != nil || len(roles) != 9 {
		t.Fatalf("ListRoles = (%d, %v), want 9", len(roles), err)
	}
	if perms := svc.ListPermissions(); len(perms) != 23 {
		t.Fatalf("ListPermissions = %d, want 23", len(perms))
	}

	// 2. Assign ACCOUNTANT to user-1 -> succeeds and records audit.
	assigned, err := svc.AssignRole(ctx, AssignRoleInput{
		TenantID:   "sup-1",
		UserID:     "user-1",
		RoleName:   auth.SupplierRoleAccountant,
		AssignedBy: "owner-1",
	})
	if err != nil || len(assigned) != 1 || assigned[0].RoleName != auth.SupplierRoleAccountant {
		t.Fatalf("AssignRole ACCOUNTANT = (%+v, %v)", assigned, err)
	}

	// 3. Also assigning FINANCE_MANAGER to user-1 violates static SoD (journal.post <-> journal.approve, etc.).
	_, err = svc.AssignRole(ctx, AssignRoleInput{
		TenantID:   "sup-1",
		UserID:     "user-1",
		RoleName:   auth.SupplierRoleFinanceManager,
		AssignedBy: "owner-1",
	})
	if !errors.Is(err, ErrSodRoleConflict) || !errors.Is(err, ErrSodViolation) {
		t.Fatalf("expected ErrSodRoleConflict+ErrSodViolation, got %v", err)
	}

	// 4. Credit Officer + Sales Manager violates credit.limit.change <-> order.over_limit.approve.
	if _, err := svc.AssignRole(ctx, AssignRoleInput{TenantID: "sup-1", UserID: "user-2", RoleName: auth.SupplierRoleCreditOfficer, AssignedBy: "owner-1"}); err != nil {
		t.Fatalf("AssignRole CREDIT_OFFICER: %v", err)
	}
	if _, err := svc.AssignRole(ctx, AssignRoleInput{TenantID: "sup-1", UserID: "user-2", RoleName: auth.SupplierRoleSalesManager, AssignedBy: "owner-1"}); !errors.Is(err, ErrSodRoleConflict) {
		t.Fatalf("expected conflict on CREDIT_OFFICER+SALES_MANAGER, got %v", err)
	}

	// 5. Dispatcher + Warehouse Manager violates stock.count <-> stock.adjust.approve.
	if _, err := svc.AssignRole(ctx, AssignRoleInput{TenantID: "sup-1", UserID: "user-3", RoleName: auth.SupplierRoleDispatcher, AssignedBy: "owner-1"}); err != nil {
		t.Fatalf("AssignRole DISPATCHER: %v", err)
	}
	if _, err := svc.AssignRole(ctx, AssignRoleInput{TenantID: "sup-1", UserID: "user-3", RoleName: auth.SupplierRoleWarehouseManager, AssignedBy: "owner-1"}); !errors.Is(err, ErrSodRoleConflict) {
		t.Fatalf("expected conflict on DISPATCHER+WAREHOUSE_MANAGER, got %v", err)
	}

	// 6. Violations recorded + audit events recorded.
	violations, err := svc.ListSodViolations(ctx, "sup-1", "", 20)
	if err != nil || len(violations) != 3 {
		t.Fatalf("ListSodViolations = (%d, %v), want 3", len(violations), err)
	}
	if len(auditSpy.events) < 6 {
		t.Fatalf("expected >= 6 audit events, got %d", len(auditSpy.events))
	}
}

func TestService_CheckFourEyes_BlocksAuditorAgentSelfApproval(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := NewMemoryRepository()
	auditSpy := &spyAuditRecorder{}
	svc := NewServiceWithRepository(repo)
	svc.SetAuditRecorder(auditSpy)

	// 1. AUDITOR write blocked.
	err := svc.CheckFourEyes(ctx, CheckFourEyesInput{
		TenantID:   "sup-1",
		ActorID:    "aud-1",
		Role:       auth.SupplierRoleAuditor,
		Action:     auth.PermJournalPost,
		ObjectType: "JournalEntry",
		ObjectID:   "je-1",
	})
	if !errors.Is(err, ErrAuditorReadOnly) || !errors.Is(err, ErrSodViolation) {
		t.Fatalf("AUDITOR check got %v", err)
	}

	// 2. AGENT approval blocked.
	err = svc.CheckFourEyes(ctx, CheckFourEyesInput{
		TenantID:   "sup-1",
		ActorID:    "agt-1",
		ActorType:  auth.ActorTypeAgent,
		Role:       auth.SupplierRoleAgent,
		Action:     auth.PermDraftApprove,
		ObjectType: "FinancialDraft",
		ObjectID:   "dr-1",
		IsApproval: true,
	})
	if !errors.Is(err, ErrAgentApprovalForbidden) || !errors.Is(err, ErrSodViolation) {
		t.Fatalf("AGENT check got %v", err)
	}

	// 3. Maker == Actor self-approval blocked.
	err = svc.CheckFourEyes(ctx, CheckFourEyesInput{
		TenantID:    "sup-1",
		ActorID:     "fin-1",
		Role:        auth.SupplierRoleFinanceManager,
		Action:      auth.PermCreditNoteApprove,
		MakerAction: auth.PermCreditNotePropose,
		ObjectType:  "DraftCreditNote",
		ObjectID:    "cn-1",
		MakerID:     "fin-1",
		IsApproval:  true,
	})
	if !errors.Is(err, ErrSodViolation) {
		t.Fatalf("self-approval check got %v, want ErrSodViolation", err)
	}

	// 4. Distinct Maker != Checker succeeds cleanly.
	err = svc.CheckFourEyes(ctx, CheckFourEyesInput{
		TenantID:    "sup-1",
		ActorID:     "fin-2",
		Role:        auth.SupplierRoleFinanceManager,
		Action:      auth.PermCreditNoteApprove,
		MakerAction: auth.PermCreditNotePropose,
		ObjectType:  "DraftCreditNote",
		ObjectID:    "cn-1",
		MakerID:     "fin-1",
		IsApproval:  true,
	})
	if err != nil {
		t.Fatalf("distinct maker/checker failed: %v", err)
	}

	// Verify 3 violations recorded.
	violations, err := svc.ListSodViolations(ctx, "sup-1", "", 20)
	if err != nil || len(violations) != 3 {
		t.Fatalf("ListSodViolations = (%d, %v), want 3", len(violations), err)
	}

	// 5. Disabling sod_credit_note policy allows policy toggle via UpdateSodPolicy.
	disabled := false
	pol, err := svc.UpdateSodPolicy(ctx, UpdatePolicyInput{
		TenantID: "sup-1",
		PolicyID: "sod_credit_note",
		Enabled:  &disabled,
		ActorID:  "owner-1",
	})
	if err != nil || pol.Enabled {
		t.Fatalf("UpdateSodPolicy got (%+v, %v)", pol, err)
	}
}
