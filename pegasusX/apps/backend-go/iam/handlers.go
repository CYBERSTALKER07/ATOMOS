package iam

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// AssignRoleRequest is the JSON request body for POST /v1/iam/users/{userId}/roles.
type AssignRoleRequest struct {
	EntityID string `json:"entity_id,omitempty"`
	TenantID string `json:"tenant_id,omitempty"`
	RoleName string `json:"role_name"`
	Action   string `json:"action,omitempty"` // ASSIGN (default) | REPLACE | REVOKE
}

// UpdateSodPolicyRequest is the JSON request body for PUT /v1/iam/sod-policies/{id}.
type UpdateSodPolicyRequest struct {
	EntityID          string  `json:"entity_id,omitempty"`
	TenantID          string  `json:"tenant_id,omitempty"`
	Name              *string `json:"name,omitempty"`
	PermissionA       *string `json:"permission_a,omitempty"`
	PermissionB       *string `json:"permission_b,omitempty"`
	EnforceSameObject *bool   `json:"enforce_same_object,omitempty"`
	Enabled           *bool   `json:"enabled,omitempty"`
}

// HandleListRoles serves GET /v1/iam/roles?entity=.
func (s *Service) HandleListRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeIAMError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeIAMError(w, http.StatusServiceUnavailable, "iam_unavailable", "IAM service unavailable")
		return
	}
	tenantID, ok := resolveIAMTenantScope(w, r, r.URL.Query().Get("entity"), true)
	if !ok {
		return
	}
	roles, userRoles, err := s.ListRoles(r.Context(), tenantID)
	if err != nil {
		writeIAMDomainError(w, err)
		return
	}
	if roles == nil {
		roles = []RoleDefinition{}
	}
	if userRoles == nil {
		userRoles = []UserRoleAssignment{}
	}
	writeIAMJSON(w, http.StatusOK, map[string]any{
		"tenant_id":  tenantID,
		"roles":      roles,
		"user_roles": userRoles,
	})
}

// HandleListPermissions serves GET /v1/iam/permissions.
func (s *Service) HandleListPermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeIAMError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	perms := CatalogPermissions()
	if s != nil {
		perms = s.ListPermissions()
	}
	writeIAMJSON(w, http.StatusOK, map[string]any{
		"permissions": perms,
		"count":       len(perms),
	})
}

// HandleAssignUserRole serves POST /v1/iam/users/{userId}/roles.
func (s *Service) HandleAssignUserRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeIAMError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeIAMError(w, http.StatusServiceUnavailable, "iam_unavailable", "IAM service unavailable")
		return
	}
	userID := strings.TrimSpace(chi.URLParam(r, "userId"))
	if userID == "" {
		writeIAMError(w, http.StatusBadRequest, "user_id_required", "userId is required")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeIAMError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return
	}
	defer r.Body.Close()

	var req AssignRoleRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeIAMError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	requestedEntity := firstNonEmpty(req.EntityID, req.TenantID, r.URL.Query().Get("entity"))
	tenantID, ok := resolveIAMTenantScope(w, r, requestedEntity, false)
	if !ok {
		return
	}

	claims, _ := auth.FromContext(r.Context())
	roles, err := s.AssignRole(r.Context(), AssignRoleInput{
		TenantID:   tenantID,
		UserID:     userID,
		RoleName:   req.RoleName,
		Action:     req.Action,
		AssignedBy: auth.ActorFromContext(r.Context()),
		ActorRole:  auth.EffectiveSupplierRole(claims),
		TraceID:    firstNonEmpty(outbox.TraceIDFromContext(r.Context()), claims.TraceID),
		IP:         r.RemoteAddr,
		DeviceID:   strings.TrimSpace(r.Header.Get("X-Device-Id")),
	})
	if err != nil {
		writeIAMDomainError(w, err)
		return
	}
	writeIAMJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
		"role_name": auth.NormalizeSupplierRoleName(req.RoleName),
		"action":    strings.ToUpper(firstNonEmpty(req.Action, RoleActionAssign)),
		"roles":     roles,
	})
}

// HandleListSodPolicies serves GET /v1/iam/sod-policies?entity=.
func (s *Service) HandleListSodPolicies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeIAMError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeIAMError(w, http.StatusServiceUnavailable, "iam_unavailable", "IAM service unavailable")
		return
	}
	tenantID, ok := resolveIAMTenantScope(w, r, r.URL.Query().Get("entity"), true)
	if !ok {
		return
	}
	policies, err := s.ListSodPolicies(r.Context(), tenantID)
	if err != nil {
		writeIAMDomainError(w, err)
		return
	}
	writeIAMJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"policies":  policies,
		"count":     len(policies),
	})
}

// HandleUpdateSodPolicy serves PUT /v1/iam/sod-policies/{id}.
func (s *Service) HandleUpdateSodPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeIAMError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeIAMError(w, http.StatusServiceUnavailable, "iam_unavailable", "IAM service unavailable")
		return
	}
	policyID := strings.TrimSpace(chi.URLParam(r, "id"))
	if policyID == "" {
		writeIAMError(w, http.StatusBadRequest, "policy_id_required", "policy id is required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeIAMError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return
	}
	defer r.Body.Close()

	var req UpdateSodPolicyRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeIAMError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}
	requestedEntity := firstNonEmpty(req.EntityID, req.TenantID, r.URL.Query().Get("entity"))
	tenantID, ok := resolveIAMTenantScope(w, r, requestedEntity, false)
	if !ok {
		return
	}
	claims, _ := auth.FromContext(r.Context())
	updated, err := s.UpdateSodPolicy(r.Context(), UpdatePolicyInput{
		TenantID:          tenantID,
		PolicyID:          policyID,
		Name:              req.Name,
		PermissionA:       req.PermissionA,
		PermissionB:       req.PermissionB,
		EnforceSameObject: req.EnforceSameObject,
		Enabled:           req.Enabled,
		ActorID:           auth.ActorFromContext(r.Context()),
		ActorRole:         auth.EffectiveSupplierRole(claims),
		TraceID:           firstNonEmpty(outbox.TraceIDFromContext(r.Context()), claims.TraceID),
		IP:                r.RemoteAddr,
		DeviceID:          strings.TrimSpace(r.Header.Get("X-Device-Id")),
	})
	if err != nil {
		writeIAMDomainError(w, err)
		return
	}
	writeIAMJSON(w, http.StatusOK, map[string]any{
		"policy": updated,
	})
}

// HandleListSodViolations serves GET /v1/iam/sod-violations?entity=&actor=&limit=.
func (s *Service) HandleListSodViolations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeIAMError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeIAMError(w, http.StatusServiceUnavailable, "iam_unavailable", "IAM service unavailable")
		return
	}
	tenantID, ok := resolveIAMTenantScope(w, r, r.URL.Query().Get("entity"), true)
	if !ok {
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 500 {
			writeIAMError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 500")
			return
		}
		limit = n
	}
	actor := strings.TrimSpace(r.URL.Query().Get("actor"))
	rows, err := s.ListSodViolations(r.Context(), tenantID, actor, limit)
	if err != nil {
		writeIAMDomainError(w, err)
		return
	}
	writeIAMJSON(w, http.StatusOK, map[string]any{
		"tenant_id":  tenantID,
		"violations": rows,
		"count":      len(rows),
	})
}

func resolveIAMTenantScope(w http.ResponseWriter, r *http.Request, requestedEntity string, allowCrossTenantPlatformRead bool) (string, bool) {
	requested := strings.TrimSpace(requestedEntity)
	ctx := r.Context()

	var callerEntity string
	if t, ok := auth.TenantFromContext(ctx); ok && strings.TrimSpace(t.SupplierID) != "" {
		callerEntity = strings.TrimSpace(t.SupplierID)
	} else if !auth.TenantContextEnforced() {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			callerEntity = strings.TrimSpace(sid)
		}
	}

	if auth.IsPlatformAdmin(ctx) {
		target := firstNonEmpty(requested, callerEntity)
		if target == "" {
			if allowCrossTenantPlatformRead {
				return "", true
			}
			writeIAMError(w, http.StatusBadRequest, "entity_required", "entity query/body parameter is required for platform admin")
			return "", false
		}
		return target, true
	}

	if callerEntity == "" {
		writeIAMError(w, http.StatusUnauthorized, "tenant_required", "authenticated supplier tenant context is required")
		return "", false
	}
	if requested != "" && requested != callerEntity {
		writeIAMError(w, http.StatusForbidden, "entity_scope_mismatch", "requested entity does not match authenticated supplier tenant")
		return "", false
	}
	return callerEntity, true
}

func writeIAMDomainError(w http.ResponseWriter, err error) {
	var conflict *StaticRoleConflictError
	var fourEyes *FourEyesError
	switch {
	case errors.As(err, &conflict):
		writeIAMJSON(w, http.StatusForbidden, map[string]any{
			"error":        "sod_violation",
			"code":         "sod_role_conflict",
			"policy_id":    conflict.PolicyID,
			"permission_a": conflict.PermissionA,
			"permission_b": conflict.PermissionB,
			"message":      err.Error(),
		})
	case errors.As(err, &fourEyes):
		writeIAMJSON(w, http.StatusForbidden, map[string]any{
			"error":     "sod_violation",
			"code":      fourEyes.RuleCode,
			"rule_code": fourEyes.RuleCode,
			"message":   fourEyes.Message,
		})
	case errors.Is(err, ErrCannotModifyBuiltin):
		writeIAMError(w, http.StatusForbidden, "cannot_modify_builtin_role", err.Error())
	case errors.Is(err, ErrPrivilegeEscalation):
		writeIAMError(w, http.StatusForbidden, "privilege_escalation_forbidden", err.Error())
	case errors.Is(err, ErrSodViolation), errors.Is(err, ErrSodRoleConflict), errors.Is(err, ErrAuditorReadOnly), errors.Is(err, ErrAgentApprovalForbidden):
		writeIAMError(w, http.StatusForbidden, "sod_violation", err.Error())
	case errors.Is(err, ErrCustomRoleExists):
		writeIAMError(w, http.StatusConflict, "custom_role_exists", err.Error())
	case errors.Is(err, ErrScimConflict):
		writeIAMError(w, http.StatusConflict, "scim_conflict", err.Error())
	case errors.Is(err, ErrPolicyNotFound):
		writeIAMError(w, http.StatusNotFound, "sod_policy_not_found", err.Error())
	case errors.Is(err, ErrCustomRoleNotFound):
		writeIAMError(w, http.StatusNotFound, "custom_role_not_found", err.Error())
	case errors.Is(err, ErrScimResourceNotFound):
		writeIAMError(w, http.StatusNotFound, "scim_resource_not_found", err.Error())
	case errors.Is(err, ErrInvalidIAMInput), errors.Is(err, ErrInvalidScimRequest):
		writeIAMError(w, http.StatusBadRequest, "invalid_iam_input", err.Error())
	case errors.Is(err, ErrIAMUnavailable):
		writeIAMError(w, http.StatusServiceUnavailable, "iam_unavailable", err.Error())
	default:
		writeIAMError(w, http.StatusInternalServerError, "iam_internal_error", err.Error())
	}
}

func writeIAMJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeIAMError(w http.ResponseWriter, status int, code, msg string) {
	writeIAMJSON(w, status, map[string]any{
		"error":   code,
		"code":    code,
		"message": msg,
	})
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
