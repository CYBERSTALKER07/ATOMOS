package iam

import (
	"context"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Repository defines persistence operations for RolePermissions, SupplierUserRoles,
// SodPolicies, and SodViolations.
type Repository interface {
	EnsureTenantSeeded(ctx context.Context, tenantID string) error
	ListRolePermissions(ctx context.Context, tenantID string) (map[string][]string, error)
	ListUserRoles(ctx context.Context, tenantID, userID string) ([]UserRoleAssignment, error)
	ApplyUserRoleChange(ctx context.Context, tenantID, userID, roleName, action, assignedBy, actorRole, traceID string, auditFn func(context.Context, *spanner.ReadWriteTransaction) error) ([]UserRoleAssignment, error)
	ListSodPolicies(ctx context.Context, tenantID string) ([]SodPolicy, error)
	UpdateSodPolicy(ctx context.Context, in UpdatePolicyInput, auditFn func(context.Context, *spanner.ReadWriteTransaction) error) (SodPolicy, error)
	RecordSodViolation(ctx context.Context, v SodViolation, auditFn func(context.Context, *spanner.ReadWriteTransaction) error) error
	ListSodViolations(ctx context.Context, tenantID, actorID string, limit int) ([]SodViolation, error)
	GrantAuditorRole(ctx context.Context, tenantID, userID, assignedBy, actorRole, reason, traceID string, expiresAt time.Time, auditFn func(context.Context, *spanner.ReadWriteTransaction) error) (AuditorGrant, error)
	ListAuditorGrants(ctx context.Context, tenantID string) ([]AuditorGrant, error)
	RevokeAuditorRole(ctx context.Context, tenantID, userID, revokedBy, actorRole, traceID string, auditFn func(context.Context, *spanner.ReadWriteTransaction) error) (bool, error)
	ListCustomRoles(ctx context.Context, tenantID string) ([]CustomRole, error)
	GetCustomRole(ctx context.Context, tenantID, roleName string) (CustomRole, bool, error)
	UpsertCustomRole(ctx context.Context, role CustomRole, auditFn func(context.Context, *spanner.ReadWriteTransaction) error) (CustomRole, error)
	DeleteCustomRole(ctx context.Context, tenantID, roleName string, auditFn func(context.Context, *spanner.ReadWriteTransaction) error) (bool, error)
	ListScimResources(ctx context.Context, tenantID, resourceType, filter string, startIndex, count int) ([]ScimResourceRecord, int, error)
	GetScimResource(ctx context.Context, tenantID, resourceType, resourceID string) (ScimResourceRecord, bool, error)
	UpsertScimResource(ctx context.Context, rec ScimResourceRecord, action, actorID, traceID string, auditFn func(context.Context, *spanner.ReadWriteTransaction) error) (ScimResourceRecord, []string, error)
	DeleteScimResource(ctx context.Context, tenantID, resourceType, resourceID, actorID, traceID string, auditFn func(context.Context, *spanner.ReadWriteTransaction) error) (bool, []string, error)
}

// DefaultSodPolicies returns the 5 canonical SoD pairs seeded for every tenant.
func DefaultSodPolicies(tenantID string, now time.Time) []SodPolicy {
	tid := strings.TrimSpace(tenantID)
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return []SodPolicy{
		{
			TenantID:          tid,
			PolicyID:          "sod_vendor_payout",
			Name:              "Vendor Creation vs Payout Release",
			PermissionA:       auth.PermVendorCreate,
			PermissionB:       auth.PermPayoutRelease,
			EnforceSameObject: true,
			Enabled:           true,
			CreatedAt:         now,
		},
		{
			TenantID:          tid,
			PolicyID:          "sod_credit_note",
			Name:              "Credit Note Proposal vs Approval",
			PermissionA:       auth.PermCreditNotePropose,
			PermissionB:       auth.PermCreditNoteApprove,
			EnforceSameObject: true,
			Enabled:           true,
			CreatedAt:         now,
		},
		{
			TenantID:          tid,
			PolicyID:          "sod_journal_post_approve",
			Name:              "Journal Posting vs Journal Approval",
			PermissionA:       auth.PermJournalPost,
			PermissionB:       auth.PermJournalApprove,
			EnforceSameObject: true,
			Enabled:           true,
			CreatedAt:         now,
		},
		{
			TenantID:          tid,
			PolicyID:          "sod_stock_count_adjust",
			Name:              "Cycle Count Submission vs Adjustment Approval",
			PermissionA:       auth.PermSupplierStockCount,
			PermissionB:       auth.PermStockAdjustApprove,
			EnforceSameObject: true,
			Enabled:           true,
			CreatedAt:         now,
		},
		{
			TenantID:          tid,
			PolicyID:          "sod_credit_limit_over_limit",
			Name:              "Credit Limit Change vs Over-Limit Order Approval",
			PermissionA:       auth.PermCreditLimitChange,
			PermissionB:       auth.PermOrderOverLimitApprove,
			EnforceSameObject: true,
			Enabled:           true,
			CreatedAt:         now,
		},
	}
}

// CatalogPermissions returns the 23 canonical supplier permissions with descriptions.
func CatalogPermissions() []PermissionDefinition {
	catalog := []PermissionDefinition{
		{Key: auth.PermSupplierOrderCancel, Category: "orders", Description: "Cancel confirmed supplier orders"},
		{Key: auth.PermPriceChange, Category: "pricing", Description: "Modify catalog or retailer override prices"},
		{Key: auth.PermCreditLimitChange, Category: "credit", Description: "Modify retailer credit limits and terms"},
		{Key: auth.PermOrderOverLimitApprove, Category: "credit", Description: "Approve orders exceeding retailer credit limits"},
		{Key: auth.PermJournalPost, Category: "finance", Description: "Post manual General Ledger journal entries"},
		{Key: auth.PermJournalReverse, Category: "finance", Description: "Reverse posted General Ledger entries"},
		{Key: auth.PermJournalApprove, Category: "finance", Description: "Approve sensitive General Ledger adjustments"},
		{Key: auth.PermPeriodClose, Category: "finance", Description: "Close an accounting period"},
		{Key: auth.PermPeriodReopen, Category: "finance", Description: "Reopen a closed accounting period"},
		{Key: auth.PermPayoutRelease, Category: "payout", Description: "Export, dispatch, or release supplier payout batches"},
		{Key: auth.PermVendorCreate, Category: "payout", Description: "Create or onboard beneficiary vendors"},
		{Key: auth.PermSupplierStockCount, Category: "warehouse", Description: "Submit warehouse cycle count quantities"},
		{Key: auth.PermStockAdjustApprove, Category: "warehouse", Description: "Approve or reject inventory count adjustments"},
		{Key: auth.PermWriteoffApprove, Category: "credit", Description: "Approve AR invoice or cash variance write-offs"},
		{Key: auth.PermAgentProposalApprove, Category: "foundry", Description: "Approve AI agent or Control Tower playbook proposals"},
		{Key: auth.PermCreditNotePropose, Category: "finance", Description: "Propose financial adjustment or credit-note drafts"},
		{Key: auth.PermCreditNoteApprove, Category: "finance", Description: "Approve or reject credit-note drafts"},
		{Key: auth.PermDraftApprove, Category: "finance", Description: "Approve or reject financial adjustment drafts"},
		{Key: auth.PermAuditRead, Category: "governance", Description: "Read immutable audit logs, chain verifications, and holds"},
		{Key: auth.PermAuditManage, Category: "governance", Description: "Create audit chain anchors and manage legal holds"},
		{Key: auth.PermIAMManage, Category: "governance", Description: "Manage supplier RBAC role assignments and SoD policies"},
		{Key: auth.PermReconManage, Category: "finance", Description: "Run four-way reconciliations and resolve breaks"},
		{Key: auth.PermConfigManage, Category: "governance", Description: "Manage tenant payout, fiscal, and operational configuration"},
	}
	sort.Slice(catalog, func(i, j int) bool { return catalog[i].Key < catalog[j].Key })
	return catalog
}

func roleDescription(role string) string {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case auth.SupplierRoleOwner:
		return "Full supplier tenant owner with governance and break-glass authority"
	case auth.SupplierRoleFinanceManager:
		return "Approves payouts, credit notes, financial drafts, write-offs, reversals, and period closes"
	case auth.SupplierRoleAccountant:
		return "Posts journals, proposes credit notes, runs reconciliation, and creates vendors"
	case auth.SupplierRoleSalesManager:
		return "Manages pricing, order cancellations, over-limit approvals, and credit-note proposals"
	case auth.SupplierRoleCreditOfficer:
		return "Manages retailer credit limits, write-offs, and credit-note proposals"
	case auth.SupplierRoleWarehouseManager:
		return "Approves inventory adjustments and warehouse agent proposals"
	case auth.SupplierRoleDispatcher:
		return "Executes cycle counts and dispatch-level order operations"
	case auth.SupplierRoleAuditor:
		return "Strictly read-only compliance auditor (audit.read only)"
	case auth.SupplierRoleAgent:
		return "Autonomous AI agent actor (never holds approvals, releases, reversals, or period controls)"
	default:
		return "Custom supplier role"
	}
}
