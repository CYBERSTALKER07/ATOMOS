package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// StaticRoleConflictError provides structured policy context when role assignment violates SoD.
type StaticRoleConflictError struct {
	PolicyID    string
	PermissionA string
	PermissionB string
	Roles       []string
}

func (e *StaticRoleConflictError) Error() string {
	if e.PolicyID == "" {
		return fmt.Sprintf("%s: conflicting roles %v", ErrSodRoleConflict, e.Roles)
	}
	return fmt.Sprintf("%s: policy %s forbids combining %s and %s across roles %v",
		ErrSodRoleConflict, e.PolicyID, e.PermissionA, e.PermissionB, e.Roles)
}

func (e *StaticRoleConflictError) Unwrap() []error {
	return []error{ErrSodRoleConflict, ErrSodViolation}
}

// FourEyesError represents a blocked runtime SoD / auditor / agent action.
type FourEyesError struct {
	Primary  error
	RuleCode string
	Message  string
}

func (e *FourEyesError) Error() string {
	return fmt.Sprintf("%s (%s): %s", e.Primary, e.RuleCode, e.Message)
}

func (e *FourEyesError) Unwrap() []error {
	if e.Primary == ErrSodViolation {
		return []error{ErrSodViolation}
	}
	return []error{e.Primary, ErrSodViolation}
}

type trackedAuditorJTI struct {
	jti       string
	expiresAt time.Time
}

// Service orchestrates supplier RBAC roles, SoD policies, four-eyes enforcement, and violation logging.
type Service struct {
	repo        Repository
	audit       AuditRecorder
	now         func() time.Time
	mu          sync.RWMutex
	jwtSecret   string
	jwtIssuer   string
	auditorJTIs map[string][]trackedAuditorJTI
}

// NewService creates a Spanner-backed IAM service.
func NewService(client *spanner.Client) *Service {
	if client == nil {
		return &Service{}
	}
	return &Service{
		repo: NewSpannerRepository(client),
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// NewServiceWithRepository creates an IAM service backed by a custom Repository.
func NewServiceWithRepository(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// Repo returns the underlying repository.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// SetAuditRecorder attaches a hash-chained AuditEvents recorder.
func (s *Service) SetAuditRecorder(rec AuditRecorder) {
	if s != nil {
		s.audit = rec
	}
}

// SetJWTSecret configures the HS256 secret and issuer used for signing time-boxed AUDITOR JWTs.
func (s *Service) SetJWTSecret(secret, issuer string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if sec := strings.TrimSpace(secret); sec != "" {
		s.jwtSecret = sec
	}
	if iss := strings.TrimSpace(issuer); iss != "" {
		s.jwtIssuer = iss
	}
}

func (s *Service) currentNow() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func checkStaticRoleSoD(roles []string, rolePerms map[string]map[string]struct{}, policies []SodPolicy) error {
	return checkStaticRoleSoDWithMeta(roles, rolePerms, policies, nil)
}

func checkStaticRoleSoDWithMeta(roles []string, rolePerms map[string]map[string]struct{}, policies []SodPolicy, readOnlyRoles map[string]bool) error {
	if len(roles) == 0 {
		return nil
	}
	// Deduplicate normalized roles.
	seen := make(map[string]struct{}, len(roles))
	uniqueRoles := make([]string, 0, len(roles))
	for _, r := range roles {
		norm := auth.NormalizeSupplierRoleName(r)
		if _, ok := seen[norm]; !ok {
			seen[norm] = struct{}{}
			uniqueRoles = append(uniqueRoles, norm)
		}
	}
	sort.Strings(uniqueRoles)

	// Single OWNER break-glass role is valid on its own; pairing OWNER with scoped roles is blocked below.
	if len(uniqueRoles) == 1 && uniqueRoles[0] == auth.SupplierRoleOwner {
		return nil
	}

	// AUDITOR (or read-only custom role) cannot be combined with any write/operational role.
	hasReadOnly := false
	if _, ok := seen[auth.SupplierRoleAuditor]; ok {
		hasReadOnly = true
	} else {
		for _, r := range uniqueRoles {
			if readOnlyRoles[r] || strings.Contains(r, "AUDITOR") {
				hasReadOnly = true
				break
			}
		}
	}
	if hasReadOnly && len(uniqueRoles) > 1 {
		return &StaticRoleConflictError{
			PolicyID:    RuleCodeAuditorReadOnly,
			PermissionA: auth.PermAuditRead,
			PermissionB: "*",
			Roles:       uniqueRoles,
		}
	}

	combinedPerms := make(map[string]struct{})
	for _, role := range uniqueRoles {
		for perm := range rolePerms[role] {
			combinedPerms[strings.ToLower(strings.TrimSpace(perm))] = struct{}{}
		}
	}

	// AGENT (or AGENT-scoped custom role) cannot hold approval/release/reversal/period/payout permissions.
	hasAgent := false
	if _, ok := seen[auth.SupplierRoleAgent]; ok {
		hasAgent = true
	} else {
		for _, r := range uniqueRoles {
			if strings.Contains(r, "AGENT") {
				hasAgent = true
				break
			}
		}
	}
	if hasAgent {
		for perm := range combinedPerms {
			if auth.IsAgentForbiddenPerm(perm) || perm == auth.PermWriteoffApprove || perm == auth.PermPayoutRelease {
				return &StaticRoleConflictError{
					PolicyID:    RuleCodeAgentApprovalForbidden,
					PermissionA: auth.SupplierRoleAgent,
					PermissionB: perm,
					Roles:       uniqueRoles,
				}
			}
		}
	}

	for _, pol := range policies {
		if !pol.Enabled {
			continue
		}
		pa := strings.ToLower(strings.TrimSpace(pol.PermissionA))
		pb := strings.ToLower(strings.TrimSpace(pol.PermissionB))
		if pa == "" || pb == "" {
			continue
		}
		_, hasA := combinedPerms[pa]
		_, hasB := combinedPerms[pb]
		if hasA && hasB {
			return &StaticRoleConflictError{
				PolicyID:    pol.PolicyID,
				PermissionA: pol.PermissionA,
				PermissionB: pol.PermissionB,
				Roles:       uniqueRoles,
			}
		}
	}
	return nil
}

// AssignRole grants, replaces, or revokes a supplier role for a user after enforcing
// static role SoD conflicts, writing AuditEvents and emitting events.EventIAMRoleChanged.
func (s *Service) AssignRole(ctx context.Context, in AssignRoleInput) ([]UserRoleAssignment, error) {
	if s == nil || s.repo == nil {
		return nil, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	uid := strings.TrimSpace(in.UserID)
	role := auth.NormalizeSupplierRoleName(in.RoleName)
	act := strings.ToUpper(strings.TrimSpace(in.Action))
	if act == "" {
		act = RoleActionAssign
	}
	assignedBy := strings.TrimSpace(in.AssignedBy)
	if assignedBy == "" {
		assignedBy = auth.ActorFromContext(ctx)
	}
	actorRole := strings.TrimSpace(in.ActorRole)
	if actorRole == "" {
		if c, ok := auth.FromContext(ctx); ok {
			actorRole = auth.EffectiveSupplierRole(c)
		}
	}
	if tid == "" || uid == "" || strings.TrimSpace(in.RoleName) == "" {
		return nil, fmt.Errorf("%w: tenant_id, user_id, and role_name required", ErrInvalidIAMInput)
	}

	beforeRoles, _ := s.repo.ListUserRoles(ctx, tid, uid)
	beforeBytes, _ := json.Marshal(beforeRoles)

	auditFn := func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if s.audit == nil {
			return nil
		}
		afterPayload, _ := json.Marshal(map[string]any{
			"user_id":     uid,
			"role_name":   role,
			"action":      act,
			"assigned_by": assignedBy,
		})
		ev := AuditRecordInput{
			TenantID:   tid,
			ActorID:    assignedBy,
			ActorType:  auth.ActorTypeHuman,
			Role:       actorRole,
			Action:     "iam.role.changed",
			ObjectType: "SupplierUserRole",
			ObjectID:   uid + ":" + role,
			BeforeJSON: string(beforeBytes),
			AfterJSON:  string(afterPayload),
			ReasonCode: act,
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		}
		if txn != nil {
			return s.audit.RecordEventInTxn(txCtx, txn, ev)
		}
		return s.audit.RecordEvent(txCtx, ev)
	}

	roles, err := s.repo.ApplyUserRoleChange(ctx, tid, uid, role, act, assignedBy, actorRole, in.TraceID, auditFn)
	if err != nil {
		var conflict *StaticRoleConflictError
		if errors.As(err, &conflict) {
			_ = s.recordViolation(ctx, SodViolation{
				TenantID:    tid,
				ViolationID: uuid.NewString(),
				ActorID:     assignedBy,
				Action:      "iam.role.assign:" + role,
				ObjectType:  "SupplierUserRole",
				ObjectID:    uid,
				MakerID:     uid,
				RuleCode:    RuleCodeStaticRoleConflict,
				RoutePath:   "/v1/iam/users/" + uid + "/roles",
				TraceID:     in.TraceID,
				CreatedAt:   s.currentNow(),
			}, actorRole, auth.ActorTypeHuman, in.IP, in.DeviceID)
		}
		return nil, err
	}
	return roles, nil
}

// CheckFourEyes enforces read-only AUDITOR bans, non-approval AGENT bans, and
// maker != actor four-eyes Segregation of Duties, logging any violation durably
// to SodViolations, AuditEvents, and OutboxEvents (sod.violation.detected).
func (s *Service) CheckFourEyes(ctx context.Context, in CheckFourEyesInput) error {
	var claims auth.Claims
	if in.Claims != nil {
		claims = *in.Claims
	} else {
		claims = auth.Claims{
			Subject:      in.ActorID,
			Role:         auth.RoleAdmin,
			SupplierID:   in.TenantID,
			SupplierRole: auth.Role(in.Role),
			ActorType:    in.ActorType,
		}
	}

	actorID := strings.TrimSpace(in.ActorID)
	if actorID == "" {
		actorID = strings.TrimSpace(auth.ActorLabel(claims))
	}
	actorType := strings.TrimSpace(in.ActorType)
	if actorType == "" {
		actorType = auth.EffectiveActorType(claims)
	}
	effRole := strings.TrimSpace(in.Role)
	if effRole == "" {
		effRole = auth.EffectiveSupplierRole(claims)
	}
	tid := strings.TrimSpace(in.TenantID)
	if tid == "" {
		tid = strings.TrimSpace(claims.SupplierID)
	}
	if tid == "" {
		tid = "default"
	}

	// 1. Strictly read-only AUDITOR cannot perform any write or approval.
	if auth.IsReadOnlyAuditor(claims) {
		v := SodViolation{
			TenantID:    tid,
			ViolationID: uuid.NewString(),
			ActorID:     actorID,
			Action:      in.Action,
			ObjectType:  in.ObjectType,
			ObjectID:    in.ObjectID,
			MakerID:     in.MakerID,
			RuleCode:    RuleCodeAuditorReadOnly,
			RoutePath:   in.RoutePath,
			TraceID:     in.TraceID,
			CreatedAt:   s.currentNow(),
		}
		_ = s.recordViolation(ctx, v, effRole, actorType, in.IP, in.DeviceID)
		return &FourEyesError{
			Primary:  ErrAuditorReadOnly,
			RuleCode: RuleCodeAuditorReadOnly,
			Message:  "AUDITOR role is strictly read-only",
		}
	}

	// 2. AGENT actors cannot approve/release/reverse/close periods.
	isApproval := in.IsApproval || auth.IsAgentForbiddenPerm(in.Action)
	if isApproval && auth.IsAgentActor(claims) {
		v := SodViolation{
			TenantID:    tid,
			ViolationID: uuid.NewString(),
			ActorID:     actorID,
			Action:      in.Action,
			ObjectType:  in.ObjectType,
			ObjectID:    in.ObjectID,
			MakerID:     in.MakerID,
			RuleCode:    RuleCodeAgentApprovalForbidden,
			RoutePath:   in.RoutePath,
			TraceID:     in.TraceID,
			CreatedAt:   s.currentNow(),
		}
		_ = s.recordViolation(ctx, v, effRole, actorType, in.IP, in.DeviceID)
		return &FourEyesError{
			Primary:  ErrAgentApprovalForbidden,
			RuleCode: RuleCodeAgentApprovalForbidden,
			Message:  "AGENT actor cannot execute approval or settlement actions",
		}
	}

	// 3. Four-eyes maker != checker enforcement.
	maker := strings.TrimSpace(in.MakerID)
	if maker != "" && actorID != "" && maker == actorID {
		if !s.isPairDisabled(ctx, tid, in.MakerAction, in.Action) {
			rc := strings.TrimSpace(in.RuleCode)
			if rc == "" {
				rc = RuleCodeFourEyesSelfApproval
			}
			v := SodViolation{
				TenantID:    tid,
				ViolationID: uuid.NewString(),
				ActorID:     actorID,
				Action:      in.Action,
				ObjectType:  in.ObjectType,
				ObjectID:    in.ObjectID,
				MakerID:     maker,
				RuleCode:    rc,
				RoutePath:   in.RoutePath,
				TraceID:     in.TraceID,
				CreatedAt:   s.currentNow(),
			}
			_ = s.recordViolation(ctx, v, effRole, actorType, in.IP, in.DeviceID)
			return &FourEyesError{
				Primary:  ErrSodViolation,
				RuleCode: rc,
				Message:  fmt.Sprintf("maker %q cannot approve own %s action", maker, in.ObjectType),
			}
		}
	}

	return nil
}

func (s *Service) isPairDisabled(ctx context.Context, tenantID, makerAction, checkerAction string) bool {
	if s == nil || s.repo == nil {
		return false
	}
	ma := strings.ToLower(strings.TrimSpace(makerAction))
	ca := strings.ToLower(strings.TrimSpace(checkerAction))
	if ma == "" || ca == "" {
		return false
	}
	pols, err := s.repo.ListSodPolicies(ctx, tenantID)
	if err != nil {
		return false
	}
	for _, p := range pols {
		pa := strings.ToLower(strings.TrimSpace(p.PermissionA))
		pb := strings.ToLower(strings.TrimSpace(p.PermissionB))
		if (pa == ma && pb == ca) || (pa == ca && pb == ma) {
			return !p.Enabled
		}
	}
	return false
}

func (s *Service) recordViolation(ctx context.Context, v SodViolation, role, actorType, ip, deviceID string) error {
	if s == nil || s.repo == nil {
		return nil
	}
	auditFn := func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if s.audit == nil {
			return nil
		}
		payload, _ := json.Marshal(v)
		ev := AuditRecordInput{
			TenantID:   v.TenantID,
			ActorID:    v.ActorID,
			ActorType:  actorType,
			Role:       role,
			Action:     "sod.violation.detected",
			ObjectType: v.ObjectType,
			ObjectID:   v.ObjectID,
			AfterJSON:  string(payload),
			ReasonCode: v.RuleCode,
			TraceID:    v.TraceID,
			IP:         ip,
			DeviceID:   deviceID,
		}
		if txn != nil {
			return s.audit.RecordEventInTxn(txCtx, txn, ev)
		}
		return s.audit.RecordEvent(txCtx, ev)
	}
	return s.repo.RecordSodViolation(ctx, v, auditFn)
}

// ListRoles returns the 9 supplier roles with effective permissions plus tenant user assignments.
func (s *Service) ListRoles(ctx context.Context, tenantID string) ([]RoleDefinition, []UserRoleAssignment, error) {
	if s == nil || s.repo == nil {
		return nil, nil, ErrIAMUnavailable
	}
	permMap, err := s.repo.ListRolePermissions(ctx, tenantID)
	if err != nil {
		return nil, nil, err
	}
	assignments, err := s.repo.ListUserRoles(ctx, tenantID, "")
	if err != nil {
		return nil, nil, err
	}
	customList, err := s.repo.ListCustomRoles(ctx, tenantID)
	if err != nil {
		return nil, nil, err
	}
	customByName := make(map[string]CustomRole, len(customList))
	for _, cr := range customList {
		customByName[cr.RoleName] = cr
	}

	roleSet := make(map[string]struct{})
	orderedRoles := make([]string, 0, len(auth.AllSupplierRoles())+len(permMap))
	for _, r := range auth.AllSupplierRoles() {
		roleSet[r] = struct{}{}
		orderedRoles = append(orderedRoles, r)
	}
	var customRoles []string
	for r := range permMap {
		if _, exists := roleSet[r]; !exists {
			customRoles = append(customRoles, r)
		}
	}
	sort.Strings(customRoles)
	orderedRoles = append(orderedRoles, customRoles...)

	defs := make([]RoleDefinition, 0, len(orderedRoles))
	for _, r := range orderedRoles {
		perms := permMap[r]
		if len(perms) == 0 {
			perms = auth.TemplatePermsForRole(r)
		}
		sort.Strings(perms)
		desc := roleDescription(r)
		ro := r == auth.SupplierRoleAuditor
		if cr, ok := customByName[r]; ok {
			if strings.TrimSpace(cr.Description) != "" {
				desc = cr.Description
			}
			ro = cr.ReadOnly
		}
		defs = append(defs, RoleDefinition{
			RoleName:    r,
			Description: desc,
			ReadOnly:    ro,
			AgentRole:   r == auth.SupplierRoleAgent || strings.Contains(r, "AGENT"),
			Permissions: perms,
		})
	}
	return defs, assignments, nil
}

// ListPermissions returns the canonical 23 supplier permission definitions.
func (s *Service) ListPermissions() []PermissionDefinition {
	return CatalogPermissions()
}

// ListSodPolicies lists SoD policies for tenantID (or cross-tenant when tenantID == "").
func (s *Service) ListSodPolicies(ctx context.Context, tenantID string) ([]SodPolicy, error) {
	if s == nil || s.repo == nil {
		return nil, ErrIAMUnavailable
	}
	return s.repo.ListSodPolicies(ctx, tenantID)
}

// UpdateSodPolicy updates an SoD policy and records an audit event.
func (s *Service) UpdateSodPolicy(ctx context.Context, in UpdatePolicyInput) (SodPolicy, error) {
	if s == nil || s.repo == nil {
		return SodPolicy{}, ErrIAMUnavailable
	}
	actor := strings.TrimSpace(in.ActorID)
	if actor == "" {
		actor = auth.ActorFromContext(ctx)
	}
	auditFn := func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if s.audit == nil {
			return nil
		}
		payload, _ := json.Marshal(in)
		ev := AuditRecordInput{
			TenantID:   in.TenantID,
			ActorID:    actor,
			ActorType:  auth.ActorTypeHuman,
			Role:       in.ActorRole,
			Action:     "iam.sod_policy.updated",
			ObjectType: "SodPolicy",
			ObjectID:   in.PolicyID,
			AfterJSON:  string(payload),
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		}
		if txn != nil {
			return s.audit.RecordEventInTxn(txCtx, txn, ev)
		}
		return s.audit.RecordEvent(txCtx, ev)
	}
	return s.repo.UpdateSodPolicy(ctx, in, auditFn)
}

// ListSodViolations lists SoD violations for tenantID (or cross-tenant when tenantID == "").
func (s *Service) ListSodViolations(ctx context.Context, tenantID, actorID string, limit int) ([]SodViolation, error) {
	if s == nil || s.repo == nil {
		return nil, ErrIAMUnavailable
	}
	return s.repo.ListSodViolations(ctx, tenantID, actorID, limit)
}

// GrantAuditorAccess assigns time-boxed read-only AUDITOR portal access (default 72h, must be future),
// enforces AGENT/self/SoD blocks, logs AuditEvents + OutboxEvents, and issues a short-lived AUDITOR JWT.
func (s *Service) GrantAuditorAccess(ctx context.Context, in GrantAuditorAccessInput) (AuditorGrantResult, error) {
	if s == nil || s.repo == nil {
		return AuditorGrantResult{}, ErrIAMUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	uid := strings.TrimSpace(in.UserID)
	if tid == "" || uid == "" {
		return AuditorGrantResult{}, fmt.Errorf("%w: tenant_id and user_id required", ErrInvalidIAMInput)
	}

	grantedBy := strings.TrimSpace(in.GrantedBy)
	if grantedBy == "" {
		grantedBy = strings.TrimSpace(auth.ActorFromContext(ctx))
	}
	claims, hasClaims := auth.FromContext(ctx)
	actorRole := strings.TrimSpace(in.ActorRole)
	if actorRole == "" && hasClaims {
		actorRole = auth.EffectiveSupplierRole(claims)
	}
	actorType := strings.TrimSpace(in.ActorType)
	if actorType == "" && hasClaims {
		actorType = strings.TrimSpace(claims.ActorType)
	}

	now := s.currentNow()

	// Block AGENT callers or targets (non-human AI agents must never hold/grant AUDITOR delegation).
	if strings.EqualFold(actorType, auth.ActorTypeAgent) || actorRole == auth.SupplierRoleAgent || (hasClaims && auth.IsAgentActor(claims)) {
		s.recordAuditorDenial(ctx, tid, uid, grantedBy, actorRole, auth.ActorTypeAgent, RuleCodeAgentApprovalForbidden, "iam.auditor_access.agent_caller_denied", in.TraceID, in.IP, in.DeviceID, now)
		return AuditorGrantResult{}, fmt.Errorf("%w: agent actor cannot grant auditor access", ErrAgentApprovalForbidden)
	}
	if strings.EqualFold(strings.TrimSpace(in.TargetActorType), auth.ActorTypeAgent) {
		s.recordAuditorDenial(ctx, tid, uid, grantedBy, actorRole, actorType, RuleCodeAgentApprovalForbidden, "iam.auditor_access.agent_target_denied", in.TraceID, in.IP, in.DeviceID, now)
		return AuditorGrantResult{}, fmt.Errorf("%w: cannot grant auditor role to AGENT actor", ErrAgentApprovalForbidden)
	}
	// Block self-grants (maker cannot grant own read-only AUDITOR portal token).
	if grantedBy != "" && strings.EqualFold(uid, grantedBy) {
		s.recordAuditorDenial(ctx, tid, uid, grantedBy, actorRole, actorType, RuleCodeFourEyesSelfApproval, "iam.auditor_access.self_grant_denied", in.TraceID, in.IP, in.DeviceID, now)
		return AuditorGrantResult{}, fmt.Errorf("%w: cannot grant auditor access to self", ErrSodViolation)
	}

	var expiresAt time.Time
	switch {
	case in.ExpiresAt != nil:
		if in.ExpiresAt.IsZero() || !in.ExpiresAt.UTC().After(now) {
			return AuditorGrantResult{}, fmt.Errorf("%w: expires_at must be a future timestamp", ErrInvalidIAMInput)
		}
		expiresAt = in.ExpiresAt.UTC()
	case in.TTLHours < 0:
		return AuditorGrantResult{}, fmt.Errorf("%w: ttl_hours must be positive", ErrInvalidIAMInput)
	case in.TTLHours > 0:
		expiresAt = now.Add(time.Duration(in.TTLHours) * time.Hour)
	default:
		expiresAt = now.Add(72 * time.Hour)
	}

	secret := strings.TrimSpace(in.JWTSecret)
	issuer := strings.TrimSpace(in.JWTIssuer)
	s.mu.RLock()
	if secret == "" {
		secret = s.jwtSecret
	}
	if issuer == "" {
		issuer = s.jwtIssuer
	}
	s.mu.RUnlock()
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	}
	if secret == "" {
		return AuditorGrantResult{}, fmt.Errorf("%w: jwt secret not configured for auditor token issuance", ErrIAMUnavailable)
	}
	if issuer == "" {
		issuer = "atomos-auth"
	}

	reason := strings.TrimSpace(in.Reason)
	beforeRoles, _ := s.repo.ListUserRoles(ctx, tid, uid)
	beforeBytes, _ := json.Marshal(beforeRoles)

	auditFn := func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if s.audit == nil {
			return nil
		}
		afterPayload, _ := json.Marshal(map[string]any{
			"user_id":    uid,
			"role_name":  auth.SupplierRoleAuditor,
			"granted_by": grantedBy,
			"expires_at": expiresAt.Format(time.RFC3339Nano),
			"reason":     reason,
		})
		ev := AuditRecordInput{
			TenantID:   tid,
			ActorID:    grantedBy,
			ActorType:  auth.ActorTypeHuman,
			Role:       actorRole,
			Action:     "iam.auditor_access.granted",
			ObjectType: "SupplierUserRole",
			ObjectID:   uid + ":" + auth.SupplierRoleAuditor,
			BeforeJSON: string(beforeBytes),
			AfterJSON:  string(afterPayload),
			ReasonCode: reason,
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		}
		if txn != nil {
			return s.audit.RecordEventInTxn(txCtx, txn, ev)
		}
		return s.audit.RecordEvent(txCtx, ev)
	}

	grant, err := s.repo.GrantAuditorRole(ctx, tid, uid, grantedBy, actorRole, reason, in.TraceID, expiresAt, auditFn)
	if err != nil {
		var conflict *StaticRoleConflictError
		if errors.As(err, &conflict) || errors.Is(err, ErrSodViolation) || errors.Is(err, ErrAgentApprovalForbidden) {
			rule := RuleCodeStaticRoleConflict
			if conflict != nil && conflict.PolicyID != "" {
				rule = conflict.PolicyID
			}
			s.recordAuditorDenial(ctx, tid, uid, grantedBy, actorRole, auth.ActorTypeHuman, rule, "iam.auditor_access.conflict_denied", in.TraceID, in.IP, in.DeviceID, now)
		}
		return AuditorGrantResult{}, err
	}

	jti := uuid.NewString()
	ttl := expiresAt.Sub(now)
	token, err := auth.Issue(auth.Claims{
		Subject:          uid,
		Role:             auth.RoleAdmin,
		SupplierID:       tid,
		SupplierRole:     auth.SupplierRoleAuditor,
		ActorType:        auth.ActorTypeHuman,
		SupplierPerms:    []string{auth.PermAuditRead},
		JTI:              jti,
		IsRegistered:     true,
		IsConfigured:     true,
		TokenUse:         auth.TokenUseFull,
		TraceID:          in.TraceID,
		AuditorExpiresAt: &expiresAt,
	}, auth.IssueOptions{
		Secret: secret,
		Issuer: issuer,
		TTL:    ttl,
		Now:    func() time.Time { return now },
	})
	if err != nil {
		return AuditorGrantResult{}, fmt.Errorf("issue auditor jwt: %w", err)
	}

	s.trackAuditorJTI(tid, uid, jti, expiresAt, now)

	return AuditorGrantResult{
		Grant:     grant,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// ListAuditorGrants returns AUDITOR grants (active and expired) for tenantID.
func (s *Service) ListAuditorGrants(ctx context.Context, tenantID string) ([]AuditorGrant, error) {
	if s == nil || s.repo == nil {
		return nil, ErrIAMUnavailable
	}
	return s.repo.ListAuditorGrants(ctx, tenantID)
}

// RevokeAuditorGrant revokes the time-boxed AUDITOR role for userID and invalidates active JTIs.
func (s *Service) RevokeAuditorGrant(ctx context.Context, in RevokeAuditorGrantInput) error {
	if s == nil || s.repo == nil {
		return ErrIAMUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	uid := strings.TrimSpace(in.UserID)
	if tid == "" || uid == "" {
		return fmt.Errorf("%w: tenant_id and user_id required", ErrInvalidIAMInput)
	}
	revokedBy := strings.TrimSpace(in.RevokedBy)
	if revokedBy == "" {
		revokedBy = strings.TrimSpace(auth.ActorFromContext(ctx))
	}
	claims, hasClaims := auth.FromContext(ctx)
	actorRole := strings.TrimSpace(in.ActorRole)
	if actorRole == "" && hasClaims {
		actorRole = auth.EffectiveSupplierRole(claims)
	}
	if hasClaims && auth.IsAgentActor(claims) {
		return fmt.Errorf("%w: agent actor cannot revoke auditor access", ErrAgentApprovalForbidden)
	}
	if hasClaims && auth.IsReadOnlyAuditor(claims) {
		return fmt.Errorf("%w: auditor cannot revoke grants", ErrAuditorReadOnly)
	}

	beforeRoles, _ := s.repo.ListUserRoles(ctx, tid, uid)
	beforeBytes, _ := json.Marshal(beforeRoles)

	auditFn := func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if s.audit == nil {
			return nil
		}
		afterPayload, _ := json.Marshal(map[string]any{
			"user_id":    uid,
			"role_name":  auth.SupplierRoleAuditor,
			"revoked_by": revokedBy,
		})
		ev := AuditRecordInput{
			TenantID:   tid,
			ActorID:    revokedBy,
			ActorType:  auth.ActorTypeHuman,
			Role:       actorRole,
			Action:     "iam.auditor_access.revoked",
			ObjectType: "SupplierUserRole",
			ObjectID:   uid + ":" + auth.SupplierRoleAuditor,
			BeforeJSON: string(beforeBytes),
			AfterJSON:  string(afterPayload),
			ReasonCode: RoleActionRevoke,
			TraceID:    in.TraceID,
			IP:         in.IP,
			DeviceID:   in.DeviceID,
		}
		if txn != nil {
			return s.audit.RecordEventInTxn(txCtx, txn, ev)
		}
		return s.audit.RecordEvent(txCtx, ev)
	}

	revoked, err := s.repo.RevokeAuditorRole(ctx, tid, uid, revokedBy, actorRole, in.TraceID, auditFn)
	if err != nil {
		return err
	}
	if !revoked {
		return fmt.Errorf("%w: %s", ErrAuditorGrantNotFound, uid)
	}
	s.revokeTrackedAuditorJTIs(ctx, tid, uid)
	return nil
}

func (s *Service) recordAuditorDenial(ctx context.Context, tid, targetUID, grantedBy, actorRole, actorType, ruleCode, action, traceID, ip, deviceID string, now time.Time) {
	if actorType == "" {
		actorType = auth.ActorTypeHuman
	}
	_ = s.recordViolation(ctx, SodViolation{
		TenantID:    tid,
		ViolationID: uuid.NewString(),
		ActorID:     firstNonEmpty(grantedBy, targetUID),
		Action:      action,
		ObjectType:  "SupplierUserRole",
		ObjectID:    targetUID,
		MakerID:     targetUID,
		RuleCode:    ruleCode,
		RoutePath:   "/v1/iam/auditor-grants",
		TraceID:     traceID,
		CreatedAt:   now,
	}, actorRole, actorType, ip, deviceID)
}

func (s *Service) trackAuditorJTI(tid, uid, jti string, expiresAt, now time.Time) {
	if s == nil || jti == "" {
		return
	}
	key := tid + "|" + uid
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.auditorJTIs == nil {
		s.auditorJTIs = make(map[string][]trackedAuditorJTI)
	}
	var kept []trackedAuditorJTI
	for _, entry := range s.auditorJTIs[key] {
		if entry.expiresAt.After(now) {
			kept = append(kept, entry)
		}
	}
	kept = append(kept, trackedAuditorJTI{jti: jti, expiresAt: expiresAt})
	s.auditorJTIs[key] = kept
}

func (s *Service) revokeTrackedAuditorJTIs(ctx context.Context, tid, uid string) {
	if s == nil {
		return
	}
	key := tid + "|" + uid
	now := s.currentNow()
	s.mu.Lock()
	entries := s.auditorJTIs[key]
	delete(s.auditorJTIs, key)
	s.mu.Unlock()

	store := auth.GetRevocationStore()
	if store == nil {
		return
	}
	for _, entry := range entries {
		ttl := entry.expiresAt.Sub(now)
		if ttl < time.Minute {
			ttl = time.Minute
		}
		_ = store.Revoke(ctx, entry.jti, ttl)
	}
}
