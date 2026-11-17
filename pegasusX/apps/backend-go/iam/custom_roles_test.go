package iam

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestCustomRoles_Lifecycle_AntiEscalation_And_SoD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := NewMemoryRepository()
	auditSpy := &spyAuditRecorder{}
	svc := NewServiceWithRepository(repo)
	svc.SetAuditRecorder(auditSpy)

	const tid = "sup-custom-1"
	ownerClaims := auth.Claims{
		Subject:      "owner-1",
		Role:         auth.RoleAdmin,
		SupplierID:   tid,
		SupplierRole: auth.SupplierRoleOwner,
	}
	ownerCtx := auth.WithClaims(ctx, ownerClaims)

	// 1. Built-in role cannot be created or overwritten.
	if _, err := svc.CreateCustomRole(ownerCtx, CreateCustomRoleInput{
		TenantID:    tid,
		RoleName:    "AUDITOR",
		Permissions: []string{auth.PermAuditRead},
	}); !errors.Is(err, ErrCannotModifyBuiltin) {
		t.Fatalf("expected ErrCannotModifyBuiltin for AUDITOR, got %v", err)
	}

	// 2. AUDITOR / read_only custom role MUST reject mutating permissions.
	if _, err := svc.CreateCustomRole(ownerCtx, CreateCustomRoleInput{
		TenantID:    tid,
		RoleName:    "EXTERNAL_AUDITOR_PLUS",
		Permissions: []string{auth.PermAuditRead, auth.PermJournalPost},
		ReadOnly:    true,
	}); !errors.Is(err, ErrAuditorReadOnly) {
		t.Fatalf("expected ErrAuditorReadOnly for read_only mutating role, got %v", err)
	}

	// 3. AGENT custom role MUST reject approval/reversal/payout/period-close permissions.
	for _, badPerm := range []string{
		auth.PermAgentProposalApprove,
		auth.PermJournalReverse,
		auth.PermPayoutRelease,
		auth.PermPeriodClose,
		auth.PermWriteoffApprove,
	} {
		if _, err := svc.CreateCustomRole(ownerCtx, CreateCustomRoleInput{
			TenantID:    tid,
			RoleName:    "AUTO_AGENT_PLUS",
			Permissions: []string{auth.PermAuditRead, badPerm},
		}); !errors.Is(err, ErrAgentApprovalForbidden) {
			t.Fatalf("expected ErrAgentApprovalForbidden for perm %s, got %v", badPerm, err)
		}
	}

	// 4. Custom role bundling an enabled SoD pair (journal.post + journal.approve) is rejected.
	if _, err := svc.CreateCustomRole(ownerCtx, CreateCustomRoleInput{
		TenantID:    tid,
		RoleName:    "SUPER_CONTROLLER",
		Permissions: []string{auth.PermJournalPost, auth.PermJournalApprove},
	}); !errors.Is(err, ErrSodRoleConflict) {
		t.Fatalf("expected ErrSodRoleConflict for SoD bundle, got %v", err)
	}

	// 5. Privilege escalation: caller without payout.release cannot create role granting payout.release.
	iamMgrClaims := auth.Claims{
		Subject:       "iam-mgr-1",
		Role:          auth.RoleAdmin,
		SupplierID:    tid,
		SupplierRole:  "IAM_ADMIN",
		SupplierPerms: []string{auth.PermIAMManage, auth.PermAuditRead},
	}
	if _, err := svc.CreateCustomRole(auth.WithClaims(ctx, iamMgrClaims), CreateCustomRoleInput{
		TenantID:     tid,
		RoleName:     "SHADOW_TREASURER",
		Permissions:  []string{auth.PermPayoutRelease},
		CallerClaims: &iamMgrClaims,
	}); !errors.Is(err, ErrPrivilegeEscalation) {
		t.Fatalf("expected ErrPrivilegeEscalation when caller lacks perm, got %v", err)
	}

	// 6. Valid custom role creates cleanly, syncs RolePermissions, appears in ListRoles, and runs SoD on AssignRole.
	created, err := svc.CreateCustomRole(ownerCtx, CreateCustomRoleInput{
		TenantID:    tid,
		RoleName:    "REGIONAL_AUDITOR_READ",
		Description: "Read-only audit viewer",
		Permissions: []string{auth.PermAuditRead},
		ReadOnly:    true,
	})
	if err != nil {
		t.Fatalf("CreateCustomRole read-only: %v", err)
	}
	if created.RoleName != "REGIONAL_AUDITOR_READ" || !created.ReadOnly {
		t.Fatalf("unexpected created role: %+v", created)
	}

	opRole, err := svc.CreateCustomRole(ownerCtx, CreateCustomRoleInput{
		TenantID:    tid,
		RoleName:    "GL_POSTER_ONLY",
		Description: "Posts manual GL journals only",
		Permissions: []string{auth.PermJournalPost, auth.PermAuditRead},
	})
	if err != nil {
		t.Fatalf("CreateCustomRole GL_POSTER_ONLY: %v", err)
	}
	if opRole.Status != "ACTIVE" {
		t.Fatalf("expected ACTIVE custom role, got %+v", opRole)
	}

	list, err := svc.ListCustomRoles(ctx, tid)
	if err != nil || len(list) != 2 {
		t.Fatalf("ListCustomRoles = (%d, %v), want 2", len(list), err)
	}

	// Assign GL_POSTER_ONLY to usr-custom -> succeeds; adding FINANCE_MANAGER -> 403 SoD conflict!
	assigned, err := svc.AssignRole(ownerCtx, AssignRoleInput{
		TenantID: tid, UserID: "usr-custom", RoleName: "GL_POSTER_ONLY", AssignedBy: "owner-1",
	})
	if err != nil || len(assigned) != 1 {
		t.Fatalf("AssignRole custom role = (%+v, %v)", assigned, err)
	}
	if _, err := svc.AssignRole(ownerCtx, AssignRoleInput{
		TenantID: tid, UserID: "usr-custom", RoleName: auth.SupplierRoleFinanceManager, AssignedBy: "owner-1",
	}); !errors.Is(err, ErrSodRoleConflict) {
		t.Fatalf("expected ErrSodRoleConflict combining custom GL_POSTER_ONLY + FINANCE_MANAGER, got %v", err)
	}

	// Assigning REGIONAL_AUDITOR_READ alongside GL_POSTER_ONLY is also blocked (read-only cannot mix).
	if _, err := svc.AssignRole(ownerCtx, AssignRoleInput{
		TenantID: tid, UserID: "usr-custom", RoleName: "REGIONAL_AUDITOR_READ", AssignedBy: "owner-1",
	}); !errors.Is(err, ErrSodRoleConflict) {
		t.Fatalf("expected ErrSodRoleConflict mixing read-only custom role, got %v", err)
	}

	// Delete GL_POSTER_ONLY -> purges RolePermissions & SupplierUserRoles and marks DELETED.
	if err := svc.DeleteCustomRole(ownerCtx, DeleteCustomRoleInput{
		TenantID: tid, RoleName: "GL_POSTER_ONLY", DeletedBy: "owner-1",
	}); err != nil {
		t.Fatalf("DeleteCustomRole: %v", err)
	}
	remaining, err := svc.ListCustomRoles(ctx, tid)
	if err != nil || len(remaining) != 1 || remaining[0].RoleName != "REGIONAL_AUDITOR_READ" {
		t.Fatalf("expected 1 remaining custom role, got %+v err=%v", remaining, err)
	}
	if ur, err := repo.ListUserRoles(ctx, tid, "usr-custom"); err != nil || len(ur) != 0 {
		t.Fatalf("expected deleted custom role purged from user assignments, got %+v err=%v", ur, err)
	}
}

func TestCustomRoles_HTTPHandlers(t *testing.T) {
	t.Parallel()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			role := req.Header.Get("X-Test-Role")
			if role != "" {
				c := auth.Claims{
					Subject:      "actor-" + strings.ToLower(role),
					Role:         auth.RoleAdmin,
					SupplierID:   "sup-h-1",
					SupplierRole: auth.Role(role),
				}
				if role == auth.SupplierRoleAgent {
					c.ActorType = auth.ActorTypeAgent
				}
				req = req.WithContext(auth.WithTenant(auth.WithClaims(req.Context(), c), auth.TenantContext{SupplierID: "sup-h-1"}))
			}
			next.ServeHTTP(w, req)
		})
	})
	r.Get("/v1/iam/custom-roles", svc.HandleListCustomRoles)
	r.Post("/v1/iam/custom-roles", svc.HandleCreateCustomRole)
	r.Delete("/v1/iam/custom-roles/{roleName}", svc.HandleDeleteCustomRole)

	// AUDITOR cannot POST or DELETE -> 403
	reqAud := httptest.NewRequest(http.MethodPost, "/v1/iam/custom-roles", strings.NewReader(`{"role_name":"AUD_TEST","permissions":["audit.read"]}`))
	reqAud.Header.Set("X-Test-Role", auth.SupplierRoleAuditor)
	rrAud := httptest.NewRecorder()
	r.ServeHTTP(rrAud, reqAud)
	if rrAud.Code != http.StatusForbidden {
		t.Fatalf("AUDITOR POST custom-roles = %d body=%s", rrAud.Code, rrAud.Body.String())
	}

	// AGENT cannot POST -> 403
	reqAgt := httptest.NewRequest(http.MethodPost, "/v1/iam/custom-roles", strings.NewReader(`{"role_name":"AGT_TEST","permissions":["audit.read"]}`))
	reqAgt.Header.Set("X-Test-Role", auth.SupplierRoleAgent)
	rrAgt := httptest.NewRecorder()
	r.ServeHTTP(rrAgt, reqAgt)
	if rrAgt.Code != http.StatusForbidden {
		t.Fatalf("AGENT POST custom-roles = %d body=%s", rrAgt.Code, rrAgt.Body.String())
	}

	// OWNER POST -> 201, GET -> 200, DELETE -> 200
	reqCreate := httptest.NewRequest(http.MethodPost, "/v1/iam/custom-roles", strings.NewReader(`{"role_name":"PRICING_ANALYST","description":"Pricing","permissions":["price.change","audit.read"]}`))
	reqCreate.Header.Set("X-Test-Role", auth.SupplierRoleOwner)
	rrCreate := httptest.NewRecorder()
	r.ServeHTTP(rrCreate, reqCreate)
	if rrCreate.Code != http.StatusCreated {
		t.Fatalf("OWNER POST custom-roles = %d body=%s", rrCreate.Code, rrCreate.Body.String())
	}

	reqList := httptest.NewRequest(http.MethodGet, "/v1/iam/custom-roles", nil)
	reqList.Header.Set("X-Test-Role", auth.SupplierRoleAuditor)
	rrList := httptest.NewRecorder()
	r.ServeHTTP(rrList, reqList)
	if rrList.Code != http.StatusOK || !strings.Contains(rrList.Body.String(), "PRICING_ANALYST") {
		t.Fatalf("AUDITOR GET custom-roles = %d body=%s", rrList.Code, rrList.Body.String())
	}

	reqDel := httptest.NewRequest(http.MethodDelete, "/v1/iam/custom-roles/PRICING_ANALYST", nil)
	reqDel.Header.Set("X-Test-Role", auth.SupplierRoleOwner)
	rrDel := httptest.NewRecorder()
	r.ServeHTTP(rrDel, reqDel)
	if rrDel.Code != http.StatusOK {
		t.Fatalf("OWNER DELETE custom-roles = %d body=%s", rrDel.Code, rrDel.Body.String())
	}
}
