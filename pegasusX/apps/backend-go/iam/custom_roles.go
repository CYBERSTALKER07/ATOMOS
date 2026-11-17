package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

var customRoleNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,62}$`)

type createCustomRoleRequest struct {
	EntityID    string   `json:"entity_id,omitempty"`
	TenantID    string   `json:"tenant_id,omitempty"`
	RoleName    string   `json:"role_name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
	ReadOnly    bool     `json:"read_only,omitempty"`
}

func isBuiltinOrReservedRole(roleName string) bool {
	norm := strings.ToUpper(strings.TrimSpace(roleName))
	switch norm {
	case "",
		auth.SupplierRoleOwner,
		auth.SupplierRoleFinanceManager,
		auth.SupplierRoleAccountant,
		auth.SupplierRoleSalesManager,
		auth.SupplierRoleCreditOfficer,
		auth.SupplierRoleWarehouseManager,
		auth.SupplierRoleDispatcher,
		auth.SupplierRoleAuditor,
		auth.SupplierRoleAgent,
		string(auth.RoleAdmin),
		string(auth.RolePlatformAdmin),
		string(auth.RoleWarehouseAdmin),
		string(auth.RoleWarehouse),
		string(auth.RoleFactoryAdmin),
		string(auth.RoleFactory),
		string(auth.RoleRetailer),
		string(auth.RoleDriver),
		string(auth.RolePayload),
		"FINANCE":
		return true
	default:
		return false
	}
}

func canonicalPermSet() map[string]struct{} {
	all := auth.AllSupplierPerms()
	set := make(map[string]struct{}, len(all))
	for _, p := range all {
		set[strings.ToLower(strings.TrimSpace(p))] = struct{}{}
	}
	return set
}

func normalizeAndValidateCustomPerms(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: at least one permission is required", ErrInvalidIAMInput)
	}
	valid := canonicalPermSet()
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		norm := strings.ToLower(strings.TrimSpace(p))
		if norm == "" {
			continue
		}
		if _, ok := valid[norm]; !ok {
			return nil, fmt.Errorf("%w: unknown permission %q", ErrInvalidIAMInput, p)
		}
		if _, dup := seen[norm]; !dup {
			seen[norm] = struct{}{}
			out = append(out, norm)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: at least one valid permission is required", ErrInvalidIAMInput)
	}
	sort.Strings(out)
	return out, nil
}

// ListCustomRoles returns all active custom roles defined for tenantID.
func (s *Service) ListCustomRoles(ctx context.Context, tenantID string) ([]CustomRole, error) {
	if s == nil || s.repo == nil {
		return nil, ErrIAMUnavailable
	}
	return s.repo.ListCustomRoles(ctx, strings.TrimSpace(tenantID))
}

// CreateCustomRole creates an Enterprise Custom Role backed by CustomRoles and synced to RolePermissions.
// Enforces anti-escalation rules:
//   - Cannot shadow or overwrite built-in system roles.
//   - AUDITOR / read_only custom roles cannot hold any mutating permission (only audit.read).
//   - AGENT custom roles cannot hold approval, release, reversal, payout, or period-close permissions.
//   - Caller cannot grant permissions they do not themselves hold.
//   - Custom role permissions cannot violate enabled tenant SoD pairs.
func (s *Service) CreateCustomRole(ctx context.Context, in CreateCustomRoleInput) (CustomRole, error) {
	if s == nil || s.repo == nil {
		return CustomRole{}, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	normRole := strings.ToUpper(strings.TrimSpace(in.RoleName))
	if tid == "" || normRole == "" {
		return CustomRole{}, fmt.Errorf("%w: tenant_id and role_name required", ErrInvalidIAMInput)
	}
	if isBuiltinOrReservedRole(normRole) {
		return CustomRole{}, fmt.Errorf("%w: cannot create or overwrite built-in role %q", ErrCannotModifyBuiltin, normRole)
	}
	if !customRoleNamePattern.MatchString(normRole) {
		return CustomRole{}, fmt.Errorf("%w: role_name must match ^[A-Z][A-Z0-9_]{1,62}$", ErrInvalidIAMInput)
	}

	claims, hasClaims := auth.FromContext(ctx)
	if !hasClaims && in.CallerClaims != nil {
		claims = *in.CallerClaims
		hasClaims = true
	}
	createdBy := strings.TrimSpace(in.CreatedBy)
	if createdBy == "" {
		createdBy = strings.TrimSpace(auth.ActorFromContext(ctx))
	}
	actorRole := strings.TrimSpace(in.ActorRole)
	if actorRole == "" && hasClaims {
		actorRole = auth.EffectiveSupplierRole(claims)
	}
	actorType := strings.TrimSpace(in.ActorType)
	if actorType == "" && hasClaims {
		actorType = auth.EffectiveActorType(claims)
	}

	if hasClaims && auth.IsReadOnlyAuditor(claims) {
		return CustomRole{}, fmt.Errorf("%w: AUDITOR cannot create custom roles", ErrAuditorReadOnly)
	}
	if hasClaims && auth.IsAgentActor(claims) {
		return CustomRole{}, fmt.Errorf("%w: AGENT actor cannot create custom roles", ErrAgentApprovalForbidden)
	}

	perms, err := normalizeAndValidateCustomPerms(in.Permissions)
	if err != nil {
		return CustomRole{}, err
	}

	// Enforce AUDITOR / read-only custom role constraint.
	isAuditorScoped := in.ReadOnly || strings.Contains(normRole, "AUDITOR")
	if isAuditorScoped {
		for _, p := range perms {
			if p != auth.PermAuditRead {
				_ = s.recordViolation(ctx, SodViolation{
					TenantID:    tid,
					ViolationID: uuid.NewString(),
					ActorID:     createdBy,
					Action:      "iam.custom_role.create:" + normRole,
					ObjectType:  "CustomRole",
					ObjectID:    normRole,
					RuleCode:    RuleCodeAuditorReadOnly,
					RoutePath:   "/v1/iam/custom-roles",
					TraceID:     in.TraceID,
					CreatedAt:   s.currentNow(),
				}, actorRole, actorType, in.IP, in.DeviceID)
				return CustomRole{}, fmt.Errorf("%w: read-only/AUDITOR custom role cannot grant mutating permission %q", ErrAuditorReadOnly, p)
			}
		}
	}

	// Enforce AGENT custom role non-approval / non-payout / non-period-close constraint.
	if strings.Contains(normRole, "AGENT") {
		for _, p := range perms {
			if auth.IsAgentForbiddenPerm(p) || p == auth.PermWriteoffApprove || p == auth.PermPayoutRelease {
				_ = s.recordViolation(ctx, SodViolation{
					TenantID:    tid,
					ViolationID: uuid.NewString(),
					ActorID:     createdBy,
					Action:      "iam.custom_role.create:" + normRole,
					ObjectType:  "CustomRole",
					ObjectID:    normRole,
					RuleCode:    RuleCodeAgentApprovalForbidden,
					RoutePath:   "/v1/iam/custom-roles",
					TraceID:     in.TraceID,
					CreatedAt:   s.currentNow(),
				}, actorRole, actorType, in.IP, in.DeviceID)
				return CustomRole{}, fmt.Errorf("%w: AGENT custom role cannot grant restricted permission %q", ErrAgentApprovalForbidden, p)
			}
		}
	}

	// Enforce caller anti-privilege-escalation: caller must hold every permission they grant.
	if hasClaims {
		for _, p := range perms {
			if !auth.HasSupplierPerm(claims, p) {
				return CustomRole{}, fmt.Errorf("%w: caller cannot grant permission %q they do not possess", ErrPrivilegeEscalation, p)
			}
		}
	}

	// Enforce SoD policy pairs on the custom role's own permission bundle.
	policies, err := s.repo.ListSodPolicies(ctx, tid)
	if err != nil {
		return CustomRole{}, err
	}
	permMap := map[string]map[string]struct{}{
		normRole: make(map[string]struct{}, len(perms)),
	}
	for _, p := range perms {
		permMap[normRole][p] = struct{}{}
	}
	if err := checkStaticRoleSoD([]string{normRole}, permMap, policies); err != nil {
		var conflict *StaticRoleConflictError
		if errors.As(err, &conflict) {
			_ = s.recordViolation(ctx, SodViolation{
				TenantID:    tid,
				ViolationID: uuid.NewString(),
				ActorID:     createdBy,
				Action:      "iam.custom_role.create:" + normRole,
				ObjectType:  "CustomRole",
				ObjectID:    normRole,
				RuleCode:    RuleCodeStaticRoleConflict,
				RoutePath:   "/v1/iam/custom-roles",
				TraceID:     in.TraceID,
				CreatedAt:   s.currentNow(),
			}, actorRole, actorType, in.IP, in.DeviceID)
		}
		return CustomRole{}, err
	}

	if existing, found, gerr := s.repo.GetCustomRole(ctx, tid, normRole); gerr != nil {
		return CustomRole{}, gerr
	} else if found && strings.EqualFold(existing.Status, "ACTIVE") {
		return CustomRole{}, fmt.Errorf("%w: custom role %q already exists", ErrCustomRoleExists, normRole)
	}

	now := s.currentNow()
	role := CustomRole{
		TenantID:    tid,
		RoleName:    normRole,
		Description: strings.TrimSpace(in.Description),
		Permissions: perms,
		ReadOnly:    in.ReadOnly,
		CreatedBy:   createdBy,
		Status:      "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	auditFn := func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if s.audit == nil {
			return nil
		}
		afterRaw, _ := json.Marshal(role)
		ev := AuditRecordInput{
			TenantID:   tid,
			ActorID:    createdBy,
			ActorType:  firstNonEmpty(actorType, auth.ActorTypeHuman),
			Role:       actorRole,
			Action:     "iam.custom_role.created",
			ObjectType: "CustomRole",
			ObjectID:   normRole,
			AfterJSON:  string(afterRaw),
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		}
		if txn != nil {
			return s.audit.RecordEventInTxn(txCtx, txn, ev)
		}
		return s.audit.RecordEvent(txCtx, ev)
	}

	return s.repo.UpsertCustomRole(ctx, role, auditFn)
}

// DeleteCustomRole marks a custom role DELETED and purges its RolePermissions & SupplierUserRoles bindings.
func (s *Service) DeleteCustomRole(ctx context.Context, in DeleteCustomRoleInput) error {
	if s == nil || s.repo == nil {
		return ErrIAMUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	normRole := strings.ToUpper(strings.TrimSpace(in.RoleName))
	if tid == "" || normRole == "" {
		return fmt.Errorf("%w: tenant_id and role_name required", ErrInvalidIAMInput)
	}
	if isBuiltinOrReservedRole(normRole) {
		return fmt.Errorf("%w: cannot delete built-in role %q", ErrCannotModifyBuiltin, normRole)
	}

	claims, hasClaims := auth.FromContext(ctx)
	if hasClaims && auth.IsReadOnlyAuditor(claims) {
		return fmt.Errorf("%w: AUDITOR cannot delete custom roles", ErrAuditorReadOnly)
	}
	if hasClaims && auth.IsAgentActor(claims) {
		return fmt.Errorf("%w: AGENT actor cannot delete custom roles", ErrAgentApprovalForbidden)
	}

	deletedBy := strings.TrimSpace(in.DeletedBy)
	if deletedBy == "" {
		deletedBy = strings.TrimSpace(auth.ActorFromContext(ctx))
	}
	actorRole := strings.TrimSpace(in.ActorRole)
	if actorRole == "" && hasClaims {
		actorRole = auth.EffectiveSupplierRole(claims)
	}
	actorType := strings.TrimSpace(in.ActorType)
	if actorType == "" && hasClaims {
		actorType = auth.EffectiveActorType(claims)
	}

	before, found, err := s.repo.GetCustomRole(ctx, tid, normRole)
	if err != nil {
		return err
	}
	if !found || !strings.EqualFold(before.Status, "ACTIVE") {
		return fmt.Errorf("%w: %s", ErrCustomRoleNotFound, normRole)
	}
	beforeRaw, _ := json.Marshal(before)

	auditFn := func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if s.audit == nil {
			return nil
		}
		afterRaw, _ := json.Marshal(map[string]any{
			"role_name":  normRole,
			"status":     "DELETED",
			"deleted_by": deletedBy,
		})
		ev := AuditRecordInput{
			TenantID:   tid,
			ActorID:    deletedBy,
			ActorType:  firstNonEmpty(actorType, auth.ActorTypeHuman),
			Role:       actorRole,
			Action:     "iam.custom_role.deleted",
			ObjectType: "CustomRole",
			ObjectID:   normRole,
			BeforeJSON: string(beforeRaw),
			AfterJSON:  string(afterRaw),
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		}
		if txn != nil {
			return s.audit.RecordEventInTxn(txCtx, txn, ev)
		}
		return s.audit.RecordEvent(txCtx, ev)
	}

	ok, err := s.repo.DeleteCustomRole(ctx, tid, normRole, auditFn)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: %s", ErrCustomRoleNotFound, normRole)
	}
	return nil
}

// HandleListCustomRoles serves GET /v1/iam/custom-roles.
func (s *Service) HandleListCustomRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeIAMError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeIAMError(w, http.StatusServiceUnavailable, "iam_unavailable", "IAM service unavailable")
		return
	}
	tenantID, ok := resolveIAMTenantScope(w, r, r.URL.Query().Get("entity"), false)
	if !ok {
		return
	}
	roles, err := s.ListCustomRoles(r.Context(), tenantID)
	if err != nil {
		writeIAMDomainError(w, err)
		return
	}
	if roles == nil {
		roles = []CustomRole{}
	}
	writeIAMJSON(w, http.StatusOK, map[string]any{
		"tenant_id":    tenantID,
		"custom_roles": roles,
		"count":        len(roles),
	})
}

// HandleCreateCustomRole serves POST /v1/iam/custom-roles.
func (s *Service) HandleCreateCustomRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeIAMError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeIAMError(w, http.StatusServiceUnavailable, "iam_unavailable", "IAM service unavailable")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 128*1024))
	if err != nil {
		writeIAMError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return
	}
	defer r.Body.Close()

	var req createCustomRoleRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeIAMError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}
	requestedEntity := firstNonEmpty(req.EntityID, req.TenantID, r.URL.Query().Get("entity"))
	tenantID, ok := resolveIAMTenantScope(w, r, requestedEntity, false)
	if !ok {
		return
	}

	claims, hasClaims := auth.FromContext(r.Context())
	var claimsCopy *auth.Claims
	if hasClaims {
		cp := claims
		claimsCopy = &cp
	}
	created, err := s.CreateCustomRole(r.Context(), CreateCustomRoleInput{
		TenantID:     tenantID,
		RoleName:     req.RoleName,
		Description:  req.Description,
		Permissions:  req.Permissions,
		ReadOnly:     req.ReadOnly,
		CreatedBy:    auth.ActorFromContext(r.Context()),
		ActorRole:    auth.EffectiveSupplierRole(claims),
		ActorType:    auth.EffectiveActorType(claims),
		CallerClaims: claimsCopy,
		TraceID:      firstNonEmpty(outbox.TraceIDFromContext(r.Context()), claims.TraceID),
		IP:           r.RemoteAddr,
		DeviceID:     strings.TrimSpace(r.Header.Get("X-Device-Id")),
	})
	if err != nil {
		writeIAMDomainError(w, err)
		return
	}
	writeIAMJSON(w, http.StatusCreated, map[string]any{
		"custom_role": created,
	})
}

// HandleDeleteCustomRole serves DELETE /v1/iam/custom-roles/{roleName}.
func (s *Service) HandleDeleteCustomRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeIAMError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeIAMError(w, http.StatusServiceUnavailable, "iam_unavailable", "IAM service unavailable")
		return
	}
	roleName := strings.TrimSpace(firstNonEmpty(chi.URLParam(r, "roleName"), chi.URLParam(r, "role")))
	if roleName == "" {
		writeIAMError(w, http.StatusBadRequest, "role_name_required", "roleName is required")
		return
	}
	tenantID, ok := resolveIAMTenantScope(w, r, r.URL.Query().Get("entity"), false)
	if !ok {
		return
	}
	claims, _ := auth.FromContext(r.Context())
	if err := s.DeleteCustomRole(r.Context(), DeleteCustomRoleInput{
		TenantID:  tenantID,
		RoleName:  roleName,
		DeletedBy: auth.ActorFromContext(r.Context()),
		ActorRole: auth.EffectiveSupplierRole(claims),
		ActorType: auth.EffectiveActorType(claims),
		TraceID:   firstNonEmpty(outbox.TraceIDFromContext(r.Context()), claims.TraceID),
		IP:        r.RemoteAddr,
		DeviceID:  strings.TrimSpace(r.Header.Get("X-Device-Id")),
	}); err != nil {
		writeIAMDomainError(w, err)
		return
	}
	writeIAMJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"role_name": strings.ToUpper(roleName),
		"deleted":   true,
		"status":    "DELETED",
	})
}
