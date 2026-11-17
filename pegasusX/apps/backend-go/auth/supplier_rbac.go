package auth

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Supplier actor provenance values for Claims.ActorType and AuditEvents.
const (
	ActorTypeHuman     = "human"
	ActorTypeAgent     = "agent"
	ActorTypeConnector = "connector"
	ActorTypeSystem    = "system"
)

// Supplier operational RBAC permission keys.
// Note: PermOrderCancel ("order.cancel") and PermStockCount ("stock.count") are
// declared in claims.go and aliased here so callers can reference either name.
const (
	PermSupplierOrderCancel   = PermOrderCancel // "order.cancel"
	PermSupplierStockCount    = PermStockCount  // "stock.count"
	PermPriceChange           = "price.change"
	PermCreditLimitChange     = "credit.limit.change"
	PermJournalPost           = "journal.post"
	PermJournalReverse        = "journal.reverse"
	PermJournalApprove        = "journal.approve"
	PermPeriodClose           = "period.close"
	PermPeriodReopen          = "period.reopen"
	PermPayoutRelease         = "payout.release"
	PermStockAdjustApprove    = "stock.adjust.approve"
	PermWriteoffApprove       = "writeoff.approve"
	PermAgentProposalApprove  = "agent.proposal.approve"
	PermCreditNotePropose     = "credit_note.propose"
	PermCreditNoteApprove     = "credit_note.approve"
	PermDraftApprove          = "draft.approve"
	PermAuditRead             = "audit.read"
	PermAuditManage           = "audit.manage"
	PermIAMManage             = "iam.manage"
	PermReconManage           = "recon.manage"
	PermConfigManage          = "config.manage"
	PermVendorCreate          = "vendor.create"
	PermOrderOverLimitApprove = "order.over_limit.approve"
)

// Canonical supplier role template identifiers (9 roles).
const (
	SupplierRoleOwner            = "OWNER"
	SupplierRoleFinanceManager   = "FINANCE_MANAGER"
	SupplierRoleAccountant       = "ACCOUNTANT"
	SupplierRoleSalesManager     = "SALES_MANAGER"
	SupplierRoleCreditOfficer    = "CREDIT_OFFICER"
	SupplierRoleWarehouseManager = "WAREHOUSE_MANAGER"
	SupplierRoleDispatcher       = "DISPATCHER"
	SupplierRoleAuditor          = "AUDITOR" // Strictly read-only: audit.read only.
	SupplierRoleAgent            = "AGENT"   // Never holds approvals, releases, reversals, or period controls.
)

var allSupplierPerms = []string{
	PermSupplierOrderCancel,
	PermPriceChange,
	PermCreditLimitChange,
	PermJournalPost,
	PermJournalReverse,
	PermJournalApprove,
	PermPeriodClose,
	PermPeriodReopen,
	PermPayoutRelease,
	PermSupplierStockCount,
	PermStockAdjustApprove,
	PermWriteoffApprove,
	PermAgentProposalApprove,
	PermCreditNotePropose,
	PermCreditNoteApprove,
	PermDraftApprove,
	PermAuditRead,
	PermAuditManage,
	PermIAMManage,
	PermReconManage,
	PermConfigManage,
	PermVendorCreate,
	PermOrderOverLimitApprove,
}

var allSupplierRoles = []string{
	SupplierRoleOwner,
	SupplierRoleFinanceManager,
	SupplierRoleAccountant,
	SupplierRoleSalesManager,
	SupplierRoleCreditOfficer,
	SupplierRoleWarehouseManager,
	SupplierRoleDispatcher,
	SupplierRoleAuditor,
	SupplierRoleAgent,
}

var supplierRolePerms = map[string]map[string]struct{}{
	SupplierRoleOwner: permSet(allSupplierPerms...),
	SupplierRoleFinanceManager: permSet(
		PermJournalReverse,
		PermJournalApprove,
		PermPeriodClose,
		PermPeriodReopen,
		PermPayoutRelease,
		PermWriteoffApprove,
		PermCreditNoteApprove,
		PermDraftApprove,
		PermAgentProposalApprove,
		PermReconManage,
		PermAuditRead,
		PermOrderOverLimitApprove,
	),
	SupplierRoleAccountant: permSet(
		PermJournalPost,
		PermCreditNotePropose,
		PermReconManage,
		PermAuditRead,
		PermVendorCreate,
	),
	SupplierRoleSalesManager: permSet(
		PermSupplierOrderCancel,
		PermPriceChange,
		PermCreditNotePropose,
		PermOrderOverLimitApprove,
		PermAgentProposalApprove,
	),
	SupplierRoleCreditOfficer: permSet(
		PermCreditLimitChange,
		PermCreditNotePropose,
		PermWriteoffApprove,
		PermAuditRead,
	),
	SupplierRoleWarehouseManager: permSet(
		PermStockAdjustApprove,
		PermSupplierOrderCancel,
		PermAgentProposalApprove,
	),
	SupplierRoleDispatcher: permSet(
		PermSupplierStockCount,
		PermSupplierOrderCancel,
	),
	SupplierRoleAuditor: permSet(
		PermAuditRead,
	),
	SupplierRoleAgent: permSet(
		PermAuditRead,
	),
}

func permSet(perms ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(perms))
	for _, p := range perms {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			m[p] = struct{}{}
		}
	}
	return m
}

// AllSupplierPerms returns a sorted copy of all 23 canonical supplier permission keys.
func AllSupplierPerms() []string {
	out := append([]string(nil), allSupplierPerms...)
	sort.Strings(out)
	return out
}

// AllSupplierRoles returns a copy of all 9 canonical supplier role template names.
func AllSupplierRoles() []string {
	return append([]string(nil), allSupplierRoles...)
}

// SupplierRoleTemplates returns a deep copy of the 9 default role -> sorted permissions matrix.
func SupplierRoleTemplates() map[string][]string {
	out := make(map[string][]string, len(supplierRolePerms))
	for role, set := range supplierRolePerms {
		perms := make([]string, 0, len(set))
		for p := range set {
			perms = append(perms, p)
		}
		sort.Strings(perms)
		out[role] = perms
	}
	return out
}

// TemplatePermsForRole returns the sorted permission list for a canonical supplier role template.
func TemplatePermsForRole(role string) []string {
	norm := NormalizeSupplierRoleName(role)
	set, ok := supplierRolePerms[norm]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// NormalizeSupplierRoleName maps legacy or lowercase role labels to canonical supplier role names.
func NormalizeSupplierRoleName(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "ADMIN", SupplierRoleOwner:
		return SupplierRoleOwner
	case "FINANCE", SupplierRoleFinanceManager:
		return SupplierRoleFinanceManager
	case SupplierRoleAccountant:
		return SupplierRoleAccountant
	case SupplierRoleSalesManager:
		return SupplierRoleSalesManager
	case SupplierRoleCreditOfficer:
		return SupplierRoleCreditOfficer
	case string(RoleWarehouseAdmin), SupplierRoleWarehouseManager:
		return SupplierRoleWarehouseManager
	case string(RoleWarehouse), SupplierRoleDispatcher:
		return SupplierRoleDispatcher
	case SupplierRoleAuditor:
		return SupplierRoleAuditor
	case SupplierRoleAgent:
		return SupplierRoleAgent
	default:
		return strings.ToUpper(strings.TrimSpace(raw))
	}
}

// EffectiveSupplierRole resolves the effective supplier role for c while keeping
// single-tenant legacy ADMIN tokens (with empty SupplierRole) backward compatible as OWNER.
func EffectiveSupplierRole(c Claims) string {
	actorType := strings.ToLower(strings.TrimSpace(c.ActorType))
	rawSupRole := strings.ToUpper(strings.TrimSpace(string(c.SupplierRole)))
	rawRole := strings.ToUpper(strings.TrimSpace(string(c.Role)))

	if rawSupRole == SupplierRoleAuditor || rawRole == SupplierRoleAuditor {
		return SupplierRoleAuditor
	}
	if actorType == ActorTypeAgent && (rawSupRole == "" || rawSupRole == SupplierRoleAgent || rawSupRole == string(RoleAdmin)) {
		return SupplierRoleAgent
	}
	if rawSupRole == SupplierRoleAgent || rawRole == SupplierRoleAgent {
		return SupplierRoleAgent
	}
	if rawRole == string(RolePlatformAdmin) && rawSupRole == "" {
		return string(RolePlatformAdmin)
	}
	if rawSupRole != "" {
		return NormalizeSupplierRoleName(rawSupRole)
	}
	switch Role(rawRole) {
	case RoleAdmin:
		return SupplierRoleOwner
	case RoleWarehouseAdmin:
		return SupplierRoleWarehouseManager
	case RoleWarehouse:
		return SupplierRoleDispatcher
	default:
		return rawRole
	}
}

// EffectiveActorType resolves the normalized actor provenance ("human", "agent", "connector", "system").
func EffectiveActorType(c Claims) string {
	switch strings.ToLower(strings.TrimSpace(c.ActorType)) {
	case ActorTypeAgent:
		return ActorTypeAgent
	case ActorTypeConnector:
		return ActorTypeConnector
	case ActorTypeSystem:
		return ActorTypeSystem
	case ActorTypeHuman:
		return ActorTypeHuman
	}
	if strings.EqualFold(strings.TrimSpace(string(c.SupplierRole)), SupplierRoleAgent) ||
		strings.EqualFold(strings.TrimSpace(string(c.Role)), SupplierRoleAgent) {
		return ActorTypeAgent
	}
	return ActorTypeHuman
}

// IsReadOnlyAuditor returns true when the caller resolves to the strictly read-only AUDITOR role.
func IsReadOnlyAuditor(c Claims) bool {
	return EffectiveSupplierRole(c) == SupplierRoleAuditor
}

// IsAgentActor returns true when the caller is an autonomous AGENT role or actor_type="agent".
func IsAgentActor(c Claims) bool {
	return strings.EqualFold(strings.TrimSpace(c.ActorType), ActorTypeAgent) ||
		EffectiveSupplierRole(c) == SupplierRoleAgent
}

// IsAgentForbiddenPerm reports whether a permission is unconditionally prohibited for AGENT actors.
func IsAgentForbiddenPerm(perm string) bool {
	p := strings.ToLower(strings.TrimSpace(perm))
	if p == "" {
		return true
	}
	if strings.HasSuffix(p, ".approve") ||
		strings.HasSuffix(p, ".release") ||
		strings.HasSuffix(p, ".reverse") ||
		strings.HasPrefix(p, "period.") {
		return true
	}
	switch p {
	case PermIAMManage, PermAuditManage, PermConfigManage, PermCreditLimitChange:
		return true
	}
	return false
}

// IsFinanceOrAdminRole reports whether a role string is authorized to view unmasked PII in audit logs.
func IsFinanceOrAdminRole(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case string(RoleAdmin), string(RolePlatformAdmin), SupplierRoleOwner, SupplierRoleFinanceManager, SupplierRoleAccountant, "FINANCE":
		return true
	default:
		return false
	}
}

// HasSupplierPerm evaluates whether c grants perm, enforcing AUDITOR read-only and AGENT non-approval invariants.
func HasSupplierPerm(c Claims, perm string) bool {
	p := strings.ToLower(strings.TrimSpace(perm))
	if p == "" {
		return false
	}
	if IsPendingOrgSelect(c) || IsWSTicket(c) {
		return false
	}

	// Strict invariant 1: AUDITOR is strictly read-only (audit.read only).
	if IsReadOnlyAuditor(c) {
		return p == PermAuditRead
	}

	// Strict invariant 2: AGENT actors can never hold approvals, releases, reversals, or period controls.
	if IsAgentActor(c) && IsAgentForbiddenPerm(p) {
		return false
	}

	// Platform admin (non-agent, non-auditor) has full supplier governance permissions.
	if c.Role == RolePlatformAdmin {
		if len(c.SupplierPerms) > 0 {
			return containsNormPerm(c.SupplierPerms, p)
		}
		return true
	}

	if !isSupplierPortalCapableRole(c.Role) {
		return false
	}

	// When explicit SupplierPerms are embedded in the token, evaluate against them directly.
	if len(c.SupplierPerms) > 0 {
		return containsNormPerm(c.SupplierPerms, p)
	}

	effRole := EffectiveSupplierRole(c)
	set, ok := supplierRolePerms[effRole]
	if !ok {
		return false
	}
	_, granted := set[p]
	return granted
}

// HasAnySupplierPerm reports whether c holds at least one permission in perms.
func HasAnySupplierPerm(c Claims, perms ...string) bool {
	for _, p := range perms {
		if HasSupplierPerm(c, p) {
			return true
		}
	}
	return false
}

// ListSupplierPerms returns sorted effective supplier permissions for c.
func ListSupplierPerms(c Claims) []string {
	if IsPendingOrgSelect(c) || IsWSTicket(c) {
		return nil
	}
	if IsReadOnlyAuditor(c) {
		return []string{PermAuditRead}
	}
	if !isSupplierPortalCapableRole(c.Role) && c.Role != RolePlatformAdmin {
		return nil
	}

	seen := make(map[string]struct{})
	if len(c.SupplierPerms) > 0 {
		for _, raw := range c.SupplierPerms {
			p := strings.ToLower(strings.TrimSpace(raw))
			if p == "" {
				continue
			}
			if IsAgentActor(c) && IsAgentForbiddenPerm(p) {
				continue
			}
			seen[p] = struct{}{}
		}
	} else if c.Role == RolePlatformAdmin {
		for _, p := range allSupplierPerms {
			if IsAgentActor(c) && IsAgentForbiddenPerm(p) {
				continue
			}
			seen[p] = struct{}{}
		}
	} else if set, ok := supplierRolePerms[EffectiveSupplierRole(c)]; ok {
		for p := range set {
			if IsAgentActor(c) && IsAgentForbiddenPerm(p) {
				continue
			}
			seen[p] = struct{}{}
		}
	}

	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// RequireSupplierPerm returns middleware rejecting unauthenticated requests (401)
// and callers lacking perm (403 JSON with detail "missing_permission").
func RequireSupplierPerm(perm string) func(http.Handler) http.Handler {
	return RequireAnySupplierPerm(perm)
}

// RequireAnySupplierPerm returns middleware requiring at least one permission in perms.
func RequireAnySupplierPerm(perms ...string) func(http.Handler) http.Handler {
	clean := make([]string, 0, len(perms))
	for _, p := range perms {
		if s := strings.TrimSpace(p); s != "" {
			clean = append(clean, s)
		}
	}
	missingLabel := ""
	if len(clean) > 0 {
		missingLabel = strings.Join(clean, "|")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, ok := FromContext(r.Context())
			if !ok {
				if IsAuditorExpiredInContext(r.Context()) {
					writeAuditorExpiredJSON(w)
					return
				}
				writeRBACJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			if IsAuditorExpired(c, time.Now().UTC()) {
				writeAuditorExpiredJSON(w)
				return
			}
			if IsPendingOrgSelect(c) {
				writeRBACJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "code": "ORG_SELECT_REQUIRED"})
				return
			}
			if IsWSTicket(c) {
				writeRBACJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "code": "ws_ticket_not_allowed"})
				return
			}
			if !HasAnySupplierPerm(c, clean...) {
				writeRBACJSON(w, http.StatusForbidden, map[string]string{
					"error":      "forbidden",
					"code":       "missing_permission",
					"detail":     "missing_permission",
					"permission": missingLabel,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuditorActive enforces that time-boxed AUDITOR sessions have not expired,
// returning 401 auditor_access_expired when AuditorExpiresAt has elapsed.
func RequireAuditorActive(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, ok := FromContext(r.Context()); ok {
			if IsAuditorExpired(c, time.Now().UTC()) {
				writeAuditorExpiredJSON(w)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if IsAuditorExpiredInContext(r.Context()) {
			writeAuditorExpiredJSON(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RejectAuditorOrAgentMutation rejects read-only AUDITOR writes and optionally AGENT approvals.
func RejectAuditorOrAgentMutation(isApproval bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, ok := FromContext(r.Context())
			if !ok {
				if IsAuditorExpiredInContext(r.Context()) {
					writeAuditorExpiredJSON(w)
					return
				}
				writeRBACJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			if IsAuditorExpired(c, time.Now().UTC()) {
				writeAuditorExpiredJSON(w)
				return
			}
			if IsReadOnlyAuditor(c) {
				writeRBACJSON(w, http.StatusForbidden, map[string]string{
					"error":  "forbidden",
					"code":   "auditor_read_only",
					"detail": "missing_permission",
				})
				return
			}
			if isApproval && IsAgentActor(c) {
				writeRBACJSON(w, http.StatusForbidden, map[string]string{
					"error":  "forbidden",
					"code":   "agent_approval_forbidden",
					"detail": "missing_permission",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isSupplierPortalCapableRole(r Role) bool {
	switch r {
	case RoleAdmin, RoleWarehouseAdmin, RoleWarehouse, RoleFactoryAdmin, RoleFactory, RolePayload:
		return true
	default:
		return false
	}
}

func containsNormPerm(perms []string, target string) bool {
	for _, raw := range perms {
		if strings.ToLower(strings.TrimSpace(raw)) == target {
			return true
		}
	}
	return false
}

func writeRBACJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
