package iam

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// MemoryRepository is an in-memory Repository implementation for unit & e2e tests.
type MemoryRepository struct {
	mu            sync.RWMutex
	seeded        map[string]bool
	rolePerms     map[string]map[string]map[string]struct{}  // tenant -> role -> perm -> struct{}
	userRoles     map[string]map[string][]UserRoleAssignment // tenant -> userID -> []UserRoleAssignment
	sodPolicies   map[string]map[string]SodPolicy            // tenant -> policyID -> SodPolicy
	customRoles   map[string]map[string]CustomRole           // tenant -> roleName -> CustomRole
	scimResources map[string]map[string]ScimResourceRecord   // tenant -> type+":"+id -> ScimResourceRecord
	violations    []SodViolation
}

// NewMemoryRepository creates a thread-safe in-memory IAM repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		seeded:        make(map[string]bool),
		rolePerms:     make(map[string]map[string]map[string]struct{}),
		userRoles:     make(map[string]map[string][]UserRoleAssignment),
		sodPolicies:   make(map[string]map[string]SodPolicy),
		customRoles:   make(map[string]map[string]CustomRole),
		scimResources: make(map[string]map[string]ScimResourceRecord),
	}
}

func (r *MemoryRepository) ensureSeedLocked(tenantID string) {
	tid := strings.TrimSpace(tenantID)
	if tid == "" || r.seeded[tid] {
		return
	}
	now := time.Now().UTC()
	if r.rolePerms[tid] == nil {
		r.rolePerms[tid] = make(map[string]map[string]struct{})
	}
	for role, perms := range auth.SupplierRoleTemplates() {
		set := make(map[string]struct{}, len(perms))
		for _, p := range perms {
			set[p] = struct{}{}
		}
		r.rolePerms[tid][role] = set
	}
	if r.sodPolicies[tid] == nil {
		r.sodPolicies[tid] = make(map[string]SodPolicy)
	}
	for _, p := range DefaultSodPolicies(tid, now) {
		r.sodPolicies[tid][p.PolicyID] = p
	}
	r.seeded[tid] = true
}

func (r *MemoryRepository) EnsureTenantSeeded(_ context.Context, tenantID string) error {
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return fmt.Errorf("%w: tenant_id required", ErrInvalidIAMInput)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureSeedLocked(tid)
	return nil
}

func (r *MemoryRepository) ListRolePermissions(_ context.Context, tenantID string) (map[string][]string, error) {
	tid := strings.TrimSpace(tenantID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if tid != "" {
		r.ensureSeedLocked(tid)
	}
	src := r.rolePerms[tid]
	if src == nil {
		return auth.SupplierRoleTemplates(), nil
	}
	out := make(map[string][]string, len(src))
	for role, set := range src {
		list := make([]string, 0, len(set))
		for p := range set {
			list = append(list, p)
		}
		sort.Strings(list)
		out[role] = list
	}
	return out, nil
}

func (r *MemoryRepository) ListUserRoles(_ context.Context, tenantID, userID string) ([]UserRoleAssignment, error) {
	tid := strings.TrimSpace(tenantID)
	uid := strings.TrimSpace(userID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	byUser := r.userRoles[tid]
	if byUser == nil {
		return []UserRoleAssignment{}, nil
	}
	if uid != "" {
		list := byUser[uid]
		out := make([]UserRoleAssignment, len(list))
		copy(out, list)
		sort.Slice(out, func(i, j int) bool { return out[i].RoleName < out[j].RoleName })
		return out, nil
	}
	var out []UserRoleAssignment
	for _, list := range byUser {
		out = append(out, list...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UserID != out[j].UserID {
			return out[i].UserID < out[j].UserID
		}
		return out[i].RoleName < out[j].RoleName
	})
	return out, nil
}

func (r *MemoryRepository) ApplyUserRoleChange(
	ctx context.Context,
	tenantID, userID, roleName, action, assignedBy, _ string, _ string,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) ([]UserRoleAssignment, error) {
	tid := strings.TrimSpace(tenantID)
	uid := strings.TrimSpace(userID)
	role := auth.NormalizeSupplierRoleName(roleName)
	act := strings.ToUpper(strings.TrimSpace(action))
	if act == "" {
		act = RoleActionAssign
	}
	if tid == "" || uid == "" || role == "" {
		return nil, fmt.Errorf("%w: tenant_id, user_id, and role_name required", ErrInvalidIAMInput)
	}

	r.mu.Lock()
	r.ensureSeedLocked(tid)
	if r.rolePerms[tid][role] == nil {
		r.mu.Unlock()
		return nil, fmt.Errorf("%w: unknown role %q", ErrInvalidIAMInput, roleName)
	}
	if r.userRoles[tid] == nil {
		r.userRoles[tid] = make(map[string][]UserRoleAssignment)
	}

	existing := r.userRoles[tid][uid]
	now := time.Now().UTC()

	var target []UserRoleAssignment
	switch act {
	case RoleActionRevoke:
		for _, a := range existing {
			if a.RoleName != role {
				target = append(target, a)
			}
		}
	case RoleActionReplace:
		target = []UserRoleAssignment{{
			TenantID: tid, UserID: uid, RoleName: role, AssignedBy: assignedBy, CreatedAt: now, UpdatedAt: now,
		}}
	case RoleActionAssign:
		found := false
		for _, a := range existing {
			if a.RoleName == role {
				a.AssignedBy = assignedBy
				a.UpdatedAt = now
				found = true
			}
			target = append(target, a)
		}
		if !found {
			target = append(target, UserRoleAssignment{
				TenantID: tid, UserID: uid, RoleName: role, AssignedBy: assignedBy, CreatedAt: now, UpdatedAt: now,
			})
		}
	default:
		r.mu.Unlock()
		return nil, fmt.Errorf("%w: unsupported role action %q", ErrInvalidIAMInput, action)
	}

	if act != RoleActionRevoke {
		policies := make([]SodPolicy, 0, len(r.sodPolicies[tid]))
		for _, p := range r.sodPolicies[tid] {
			policies = append(policies, p)
		}
		roSet := make(map[string]bool)
		for name, cr := range r.customRoles[tid] {
			if strings.EqualFold(cr.Status, "ACTIVE") && cr.ReadOnly {
				roSet[name] = true
			}
		}
		roleNames := filterActiveRoleNames(target, now)
		if err := checkStaticRoleSoDWithMeta(roleNames, r.rolePerms[tid], policies, roSet); err != nil {
			r.mu.Unlock()
			return nil, err
		}
	}

	r.userRoles[tid][uid] = target
	out := make([]UserRoleAssignment, len(target))
	copy(out, target)
	sort.Slice(out, func(i, j int) bool { return out[i].RoleName < out[j].RoleName })
	r.mu.Unlock()

	if auditFn != nil {
		if err := auditFn(ctx, nil); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *MemoryRepository) ListSodPolicies(_ context.Context, tenantID string) ([]SodPolicy, error) {
	tid := strings.TrimSpace(tenantID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if tid != "" {
		r.ensureSeedLocked(tid)
		out := make([]SodPolicy, 0, len(r.sodPolicies[tid]))
		for _, p := range r.sodPolicies[tid] {
			out = append(out, p)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].PolicyID < out[j].PolicyID })
		return out, nil
	}
	var out []SodPolicy
	for _, byPol := range r.sodPolicies {
		for _, p := range byPol {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return DefaultSodPolicies("default", time.Now().UTC()), nil
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TenantID != out[j].TenantID {
			return out[i].TenantID < out[j].TenantID
		}
		return out[i].PolicyID < out[j].PolicyID
	})
	return out, nil
}

func (r *MemoryRepository) UpdateSodPolicy(
	ctx context.Context,
	in UpdatePolicyInput,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (SodPolicy, error) {
	tid := strings.TrimSpace(in.TenantID)
	pid := strings.TrimSpace(in.PolicyID)
	if tid == "" || pid == "" {
		return SodPolicy{}, fmt.Errorf("%w: tenant_id and policy_id required", ErrInvalidIAMInput)
	}
	r.mu.Lock()
	r.ensureSeedLocked(tid)
	cur, ok := r.sodPolicies[tid][pid]
	if !ok {
		r.mu.Unlock()
		return SodPolicy{}, ErrPolicyNotFound
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
		cur.Name = strings.TrimSpace(*in.Name)
	}
	if in.PermissionA != nil && strings.TrimSpace(*in.PermissionA) != "" {
		cur.PermissionA = strings.ToLower(strings.TrimSpace(*in.PermissionA))
	}
	if in.PermissionB != nil && strings.TrimSpace(*in.PermissionB) != "" {
		cur.PermissionB = strings.ToLower(strings.TrimSpace(*in.PermissionB))
	}
	if in.EnforceSameObject != nil {
		cur.EnforceSameObject = *in.EnforceSameObject
	}
	if in.Enabled != nil {
		cur.Enabled = *in.Enabled
	}
	if cur.PermissionA == cur.PermissionB {
		r.mu.Unlock()
		return SodPolicy{}, fmt.Errorf("%w: permission_a and permission_b must differ", ErrInvalidIAMInput)
	}
	r.sodPolicies[tid][pid] = cur
	r.mu.Unlock()

	if auditFn != nil {
		if err := auditFn(ctx, nil); err != nil {
			return SodPolicy{}, err
		}
	}
	return cur, nil
}

func (r *MemoryRepository) RecordSodViolation(
	ctx context.Context,
	v SodViolation,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) error {
	r.mu.Lock()
	r.violations = append(r.violations, v)
	r.mu.Unlock()
	if auditFn != nil {
		return auditFn(ctx, nil)
	}
	return nil
}

func (r *MemoryRepository) ListSodViolations(_ context.Context, tenantID, actorID string, limit int) ([]SodViolation, error) {
	tid := strings.TrimSpace(tenantID)
	actor := strings.TrimSpace(actorID)
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]SodViolation, 0)
	for i := len(r.violations) - 1; i >= 0; i-- {
		v := r.violations[i]
		if tid != "" && v.TenantID != tid {
			continue
		}
		if actor != "" && v.ActorID != actor {
			continue
		}
		out = append(out, v)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *MemoryRepository) GrantAuditorRole(
	ctx context.Context,
	tenantID, userID, assignedBy, _ string, _ string, _ string,
	expiresAt time.Time,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (AuditorGrant, error) {
	tid := strings.TrimSpace(tenantID)
	uid := strings.TrimSpace(userID)
	if tid == "" || uid == "" || expiresAt.IsZero() {
		return AuditorGrant{}, fmt.Errorf("%w: tenant_id, user_id, and expires_at required", ErrInvalidIAMInput)
	}
	now := time.Now().UTC()
	expUTC := expiresAt.UTC()

	r.mu.Lock()
	r.ensureSeedLocked(tid)
	if r.userRoles[tid] == nil {
		r.userRoles[tid] = make(map[string][]UserRoleAssignment)
	}
	existing := r.userRoles[tid][uid]
	if err := checkAuditorGrantEligibility(existing, now); err != nil {
		r.mu.Unlock()
		return AuditorGrant{}, err
	}

	createdAt := now
	var next []UserRoleAssignment
	for _, a := range existing {
		if a.RoleName == auth.SupplierRoleAuditor {
			if !a.CreatedAt.IsZero() {
				createdAt = a.CreatedAt
			}
			continue
		}
		next = append(next, a)
	}
	next = append(next, UserRoleAssignment{
		TenantID:   tid,
		UserID:     uid,
		RoleName:   auth.SupplierRoleAuditor,
		AssignedBy: strings.TrimSpace(assignedBy),
		ExpiresAt:  &expUTC,
		CreatedAt:  createdAt,
		UpdatedAt:  now,
	})
	r.userRoles[tid][uid] = next
	r.mu.Unlock()

	if auditFn != nil {
		if err := auditFn(ctx, nil); err != nil {
			return AuditorGrant{}, err
		}
	}

	status := "ACTIVE"
	if !expUTC.After(now) {
		status = "EXPIRED"
	}
	return AuditorGrant{
		TenantID:   tid,
		UserID:     uid,
		RoleName:   auth.SupplierRoleAuditor,
		AssignedBy: strings.TrimSpace(assignedBy),
		GrantedBy:  strings.TrimSpace(assignedBy),
		Status:     status,
		ExpiresAt:  expUTC,
		CreatedAt:  createdAt,
		UpdatedAt:  now,
	}, nil
}

func (r *MemoryRepository) ListAuditorGrants(_ context.Context, tenantID string) ([]AuditorGrant, error) {
	tid := strings.TrimSpace(tenantID)
	now := time.Now().UTC()
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []AuditorGrant
	appendFromTenant := func(byUser map[string][]UserRoleAssignment) {
		for _, list := range byUser {
			for _, a := range list {
				if a.RoleName != auth.SupplierRoleAuditor {
					continue
				}
				var exp time.Time
				status := "ACTIVE"
				if a.ExpiresAt != nil {
					exp = a.ExpiresAt.UTC()
					if !exp.After(now) {
						status = "EXPIRED"
					}
				}
				out = append(out, AuditorGrant{
					TenantID:   a.TenantID,
					UserID:     a.UserID,
					RoleName:   a.RoleName,
					AssignedBy: a.AssignedBy,
					GrantedBy:  a.AssignedBy,
					Status:     status,
					ExpiresAt:  exp,
					CreatedAt:  a.CreatedAt,
					UpdatedAt:  a.UpdatedAt,
				})
			}
		}
	}
	if tid != "" {
		appendFromTenant(r.userRoles[tid])
	} else {
		for _, byUser := range r.userRoles {
			appendFromTenant(byUser)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].UpdatedAt.After(out[j].UpdatedAt)
		}
		return out[i].UserID < out[j].UserID
	})
	if out == nil {
		out = []AuditorGrant{}
	}
	return out, nil
}

func (r *MemoryRepository) RevokeAuditorRole(
	ctx context.Context,
	tenantID, userID, _, _, _ string,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (bool, error) {
	tid := strings.TrimSpace(tenantID)
	uid := strings.TrimSpace(userID)
	if tid == "" || uid == "" {
		return false, fmt.Errorf("%w: tenant_id and user_id required", ErrInvalidIAMInput)
	}
	r.mu.Lock()
	byUser := r.userRoles[tid]
	if byUser == nil {
		r.mu.Unlock()
		return false, nil
	}
	existing := byUser[uid]
	var next []UserRoleAssignment
	found := false
	for _, a := range existing {
		if a.RoleName == auth.SupplierRoleAuditor {
			found = true
			continue
		}
		next = append(next, a)
	}
	if !found {
		r.mu.Unlock()
		return false, nil
	}
	byUser[uid] = next
	r.mu.Unlock()

	if auditFn != nil {
		if err := auditFn(ctx, nil); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *MemoryRepository) ListCustomRoles(_ context.Context, tenantID string) ([]CustomRole, error) {
	tid := strings.TrimSpace(tenantID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	byName := r.customRoles[tid]
	if byName == nil {
		return []CustomRole{}, nil
	}
	out := make([]CustomRole, 0, len(byName))
	for _, cr := range byName {
		if !strings.EqualFold(cr.Status, "ACTIVE") {
			continue
		}
		cp := cr
		cp.Permissions = append([]string(nil), cr.Permissions...)
		out = append(out, cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RoleName < out[j].RoleName })
	return out, nil
}

func (r *MemoryRepository) GetCustomRole(_ context.Context, tenantID, roleName string) (CustomRole, bool, error) {
	tid := strings.TrimSpace(tenantID)
	rn := strings.ToUpper(strings.TrimSpace(roleName))
	r.mu.RLock()
	defer r.mu.RUnlock()
	cr, ok := r.customRoles[tid][rn]
	if !ok {
		return CustomRole{}, false, nil
	}
	cp := cr
	cp.Permissions = append([]string(nil), cr.Permissions...)
	return cp, true, nil
}

func (r *MemoryRepository) UpsertCustomRole(
	ctx context.Context,
	role CustomRole,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (CustomRole, error) {
	tid := strings.TrimSpace(role.TenantID)
	rn := strings.ToUpper(strings.TrimSpace(role.RoleName))
	if tid == "" || rn == "" {
		return CustomRole{}, fmt.Errorf("%w: tenant_id and role_name required", ErrInvalidIAMInput)
	}
	r.mu.Lock()
	r.ensureSeedLocked(tid)
	if r.customRoles[tid] == nil {
		r.customRoles[tid] = make(map[string]CustomRole)
	}
	cp := role
	cp.TenantID = tid
	cp.RoleName = rn
	cp.Permissions = append([]string(nil), role.Permissions...)
	if prev, exists := r.customRoles[tid][rn]; exists && !prev.CreatedAt.IsZero() {
		cp.CreatedAt = prev.CreatedAt
	}
	r.customRoles[tid][rn] = cp

	permSet := make(map[string]struct{}, len(cp.Permissions))
	for _, p := range cp.Permissions {
		permSet[p] = struct{}{}
	}
	r.rolePerms[tid][rn] = permSet
	r.mu.Unlock()

	if auditFn != nil {
		if err := auditFn(ctx, nil); err != nil {
			return CustomRole{}, err
		}
	}
	out := cp
	out.Permissions = append([]string(nil), cp.Permissions...)
	return out, nil
}

func (r *MemoryRepository) DeleteCustomRole(
	ctx context.Context,
	tenantID, roleName string,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (bool, error) {
	tid := strings.TrimSpace(tenantID)
	rn := strings.ToUpper(strings.TrimSpace(roleName))
	r.mu.Lock()
	byName := r.customRoles[tid]
	if byName == nil {
		r.mu.Unlock()
		return false, nil
	}
	cr, ok := byName[rn]
	if !ok || !strings.EqualFold(cr.Status, "ACTIVE") {
		r.mu.Unlock()
		return false, nil
	}
	cr.Status = "DELETED"
	cr.UpdatedAt = time.Now().UTC()
	byName[rn] = cr
	if r.rolePerms[tid] != nil {
		delete(r.rolePerms[tid], rn)
	}
	if r.userRoles[tid] != nil {
		for uid, list := range r.userRoles[tid] {
			var filtered []UserRoleAssignment
			for _, a := range list {
				if a.RoleName != rn {
					filtered = append(filtered, a)
				}
			}
			r.userRoles[tid][uid] = filtered
		}
	}
	r.mu.Unlock()

	if auditFn != nil {
		if err := auditFn(ctx, nil); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *MemoryRepository) ListScimResources(_ context.Context, tenantID, resourceType, filter string, startIndex, count int) ([]ScimResourceRecord, int, error) {
	tid := strings.TrimSpace(tenantID)
	rt := strings.TrimSpace(resourceType)
	r.mu.RLock()
	defer r.mu.RUnlock()
	var matched []ScimResourceRecord
	for _, rec := range r.scimResources[tid] {
		if rt != "" && rec.ResourceType != rt {
			continue
		}
		if !matchesScimFilter(rec, filter) {
			continue
		}
		matched = append(matched, cloneScimRecord(rec))
	}
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].ResourceType != matched[j].ResourceType {
			return matched[i].ResourceType < matched[j].ResourceType
		}
		return matched[i].ResourceID < matched[j].ResourceID
	})
	total := len(matched)
	if startIndex < 1 {
		startIndex = 1
	}
	startOffset := startIndex - 1
	if startOffset >= total || count == 0 {
		return []ScimResourceRecord{}, total, nil
	}
	endOffset := total
	if count > 0 && startOffset+count < endOffset {
		endOffset = startOffset + count
	}
	return matched[startOffset:endOffset], total, nil
}

func (r *MemoryRepository) GetScimResource(_ context.Context, tenantID, resourceType, resourceID string) (ScimResourceRecord, bool, error) {
	tid := strings.TrimSpace(tenantID)
	key := strings.TrimSpace(resourceType) + ":" + strings.TrimSpace(resourceID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.scimResources[tid][key]
	if !ok {
		return ScimResourceRecord{}, false, nil
	}
	return cloneScimRecord(rec), true, nil
}

func (r *MemoryRepository) UpsertScimResource(
	ctx context.Context,
	rec ScimResourceRecord,
	action, _, _ string,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (ScimResourceRecord, []string, error) {
	tid := strings.TrimSpace(rec.TenantID)
	rt := strings.TrimSpace(rec.ResourceType)
	rid := strings.TrimSpace(rec.ResourceID)
	if tid == "" || rt == "" || rid == "" {
		return ScimResourceRecord{}, nil, fmt.Errorf("%w: tenant_id, resource_type, and resource_id required", ErrInvalidScimRequest)
	}
	now := time.Now().UTC()

	r.mu.Lock()
	r.ensureSeedLocked(tid)
	if r.scimResources[tid] == nil {
		r.scimResources[tid] = make(map[string]ScimResourceRecord)
	}
	if r.userRoles[tid] == nil {
		r.userRoles[tid] = make(map[string][]UserRoleAssignment)
	}
	policies := make([]SodPolicy, 0, len(r.sodPolicies[tid]))
	for _, p := range r.sodPolicies[tid] {
		policies = append(policies, p)
	}
	roSet := make(map[string]bool)
	for name, cr := range r.customRoles[tid] {
		if strings.EqualFold(cr.Status, "ACTIVE") && cr.ReadOnly {
			roSet[name] = true
		}
	}

	key := rt + ":" + rid
	prev, hadPrev := r.scimResources[tid][key]
	if hadPrev && !prev.CreatedAt.IsZero() {
		rec.CreatedAt = prev.CreatedAt
		rec.Meta.Created = prev.CreatedAt.UTC().Format(time.RFC3339Nano)
	}

	var rolesSynced []string
	switch rt {
	case ScimResourceTypeUser:
		if strings.EqualFold(action, "CREATE") {
			for k, existing := range r.scimResources[tid] {
				if k != key && existing.ResourceType == ScimResourceTypeUser && strings.EqualFold(existing.UserName, rec.UserName) {
					r.mu.Unlock()
					return ScimResourceRecord{}, nil, fmt.Errorf("%w: userName %q already exists", ErrScimConflict, rec.UserName)
				}
			}
		}
		for _, rr := range rec.Roles {
			rn := auth.NormalizeSupplierRoleName(rr.Value)
			if r.rolePerms[tid][rn] == nil {
				r.mu.Unlock()
				return ScimResourceRecord{}, nil, fmt.Errorf("%w: unknown role %q", ErrInvalidScimRequest, rr.Value)
			}
			rolesSynced = append(rolesSynced, rn)
		}
		if !rec.Active {
			delete(r.userRoles[tid], rid)
			rolesSynced = nil
		} else {
			var next []UserRoleAssignment
			for _, a := range r.userRoles[tid][rid] {
				if a.AssignedBy != "scim:sync" {
					next = append(next, a)
				}
			}
			for _, rn := range rolesSynced {
				next = append(next, UserRoleAssignment{
					TenantID: tid, UserID: rid, RoleName: rn, AssignedBy: "scim:sync", CreatedAt: now, UpdatedAt: now,
				})
			}
			roleNames := filterActiveRoleNames(next, now)
			if err := checkStaticRoleSoDWithMeta(roleNames, r.rolePerms[tid], policies, roSet); err != nil {
				r.mu.Unlock()
				return ScimResourceRecord{}, nil, err
			}
			r.userRoles[tid][rid] = next
		}
	case ScimResourceTypeGroup:
		groupRoles, err := resolveGroupSyncRoles(rec, r.rolePerms[tid])
		if err != nil {
			r.mu.Unlock()
			return ScimResourceRecord{}, nil, err
		}
		rolesSynced = groupRoles
		groupTag := "scim:group:" + rid
		newMembers := make(map[string]struct{}, len(rec.Members))
		for _, m := range rec.Members {
			newMembers[m.Value] = struct{}{}
		}
		if hadPrev {
			for _, pm := range prev.Members {
				if _, kept := newMembers[pm.Value]; !kept {
					var filtered []UserRoleAssignment
					for _, a := range r.userRoles[tid][pm.Value] {
						if a.AssignedBy != groupTag {
							filtered = append(filtered, a)
						}
					}
					r.userRoles[tid][pm.Value] = filtered
				}
			}
		}
		if len(groupRoles) > 0 {
			for _, m := range rec.Members {
				var next []UserRoleAssignment
				for _, a := range r.userRoles[tid][m.Value] {
					if a.AssignedBy != groupTag {
						next = append(next, a)
					}
				}
				for _, rn := range groupRoles {
					next = append(next, UserRoleAssignment{
						TenantID: tid, UserID: m.Value, RoleName: rn, AssignedBy: groupTag, CreatedAt: now, UpdatedAt: now,
					})
				}
				roleNames := filterActiveRoleNames(next, now)
				if err := checkStaticRoleSoDWithMeta(roleNames, r.rolePerms[tid], policies, roSet); err != nil {
					r.mu.Unlock()
					return ScimResourceRecord{}, nil, err
				}
				r.userRoles[tid][m.Value] = next
			}
		}
	}

	cp := cloneScimRecord(rec)
	r.scimResources[tid][key] = cp
	r.mu.Unlock()

	if auditFn != nil {
		if err := auditFn(ctx, nil); err != nil {
			return ScimResourceRecord{}, nil, err
		}
	}
	return cloneScimRecord(cp), rolesSynced, nil
}

func (r *MemoryRepository) DeleteScimResource(
	ctx context.Context,
	tenantID, resourceType, resourceID, _, _ string,
	auditFn func(context.Context, *spanner.ReadWriteTransaction) error,
) (bool, []string, error) {
	tid := strings.TrimSpace(tenantID)
	rt := strings.TrimSpace(resourceType)
	rid := strings.TrimSpace(resourceID)
	key := rt + ":" + rid

	r.mu.Lock()
	byKey := r.scimResources[tid]
	if byKey == nil {
		r.mu.Unlock()
		return false, nil, nil
	}
	cur, ok := byKey[key]
	if !ok {
		r.mu.Unlock()
		return false, nil, nil
	}
	delete(byKey, key)

	var rolesSynced []string
	if r.userRoles[tid] != nil {
		switch rt {
		case ScimResourceTypeUser:
			for _, a := range r.userRoles[tid][rid] {
				rolesSynced = append(rolesSynced, a.RoleName)
			}
			delete(r.userRoles[tid], rid)
		case ScimResourceTypeGroup:
			groupTag := "scim:group:" + rid
			for _, m := range cur.Members {
				var kept []UserRoleAssignment
				for _, a := range r.userRoles[tid][m.Value] {
					if a.AssignedBy == groupTag {
						rolesSynced = append(rolesSynced, a.RoleName)
						continue
					}
					kept = append(kept, a)
				}
				r.userRoles[tid][m.Value] = kept
			}
		}
	}
	r.mu.Unlock()

	if auditFn != nil {
		if err := auditFn(ctx, nil); err != nil {
			return false, nil, err
		}
	}
	return true, rolesSynced, nil
}
