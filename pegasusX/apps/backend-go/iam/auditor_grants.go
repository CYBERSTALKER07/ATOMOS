package iam

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type grantAuditorAccessRequest struct {
	EntityID        string  `json:"entity_id,omitempty"`
	TenantID        string  `json:"tenant_id,omitempty"`
	UserID          string  `json:"user_id"`
	TargetActorType string  `json:"target_actor_type,omitempty"`
	ActorType       string  `json:"actor_type,omitempty"`
	ExpiresAt       *string `json:"expires_at,omitempty"`
	TTLHours        int64   `json:"ttl_hours,omitempty"`
	Reason          string  `json:"reason,omitempty"`
}

type revokeAuditorGrantRequest struct {
	EntityID string `json:"entity_id,omitempty"`
	TenantID string `json:"tenant_id,omitempty"`
}

func isAssignmentExpired(a UserRoleAssignment, now time.Time) bool {
	if a.RoleName != auth.SupplierRoleAuditor || a.ExpiresAt == nil || a.ExpiresAt.IsZero() {
		return false
	}
	return !a.ExpiresAt.UTC().After(now.UTC())
}

func filterActiveRoleNames(existing []UserRoleAssignment, now time.Time) []string {
	roles := make([]string, 0, len(existing))
	for _, a := range existing {
		if isAssignmentExpired(a, now) {
			continue
		}
		roles = append(roles, a.RoleName)
	}
	return roles
}

func checkAuditorGrantEligibility(existing []UserRoleAssignment, now time.Time) error {
	var nonAuditor []string
	hasAgent := false
	for _, a := range existing {
		if isAssignmentExpired(a, now) {
			continue
		}
		rn := auth.NormalizeSupplierRoleName(a.RoleName)
		if rn == "" || rn == auth.SupplierRoleAuditor {
			continue
		}
		if rn == auth.SupplierRoleAgent {
			hasAgent = true
		}
		nonAuditor = append(nonAuditor, rn)
	}
	if len(nonAuditor) == 0 {
		return nil
	}
	roles := append([]string{auth.SupplierRoleAuditor}, nonAuditor...)
	if hasAgent {
		conflict := &StaticRoleConflictError{
			PolicyID:    RuleCodeAgentApprovalForbidden,
			PermissionA: auth.SupplierRoleAuditor,
			PermissionB: auth.SupplierRoleAgent,
			Roles:       roles,
		}
		return fmt.Errorf("%w: %w", ErrAgentApprovalForbidden, conflict)
	}
	return &StaticRoleConflictError{
		PolicyID:    RuleCodeAuditorReadOnly,
		PermissionA: auth.PermAuditRead,
		PermissionB: "*",
		Roles:       roles,
	}
}

// HandleGrantAuditorAccess serves POST /v1/iam/auditor-grants using service-configured JWT signing.
func (s *Service) HandleGrantAuditorAccess(w http.ResponseWriter, r *http.Request) {
	s.handleGrantAuditorAccess(w, r, "", "")
}

// HandleGrantAuditorAccessWithJWT serves POST /v1/iam/auditor-grants with explicit fallback JWT secret/issuer.
func (s *Service) HandleGrantAuditorAccessWithJWT(secret, issuer string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.handleGrantAuditorAccess(w, r, secret, issuer)
	}
}

func (s *Service) handleGrantAuditorAccess(w http.ResponseWriter, r *http.Request, secret, issuer string) {
	if r.Method != http.MethodPost {
		writeIAMError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeIAMError(w, http.StatusServiceUnavailable, "iam_unavailable", "IAM service unavailable")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeIAMError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
		return
	}
	defer r.Body.Close()

	var req grantAuditorAccessRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeIAMError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
		return
	}

	requestedEntity := firstNonEmpty(req.EntityID, req.TenantID, r.URL.Query().Get("entity"))
	tenantID, ok := resolveIAMTenantScope(w, r, requestedEntity, false)
	if !ok {
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		raw := strings.TrimSpace(*req.ExpiresAt)
		if raw != "" {
			parsed, perr := parseTimestampRFC3339(raw)
			if perr != nil {
				writeIAMError(w, http.StatusBadRequest, "invalid_expires_at", "expires_at must be an RFC3339 timestamp")
				return
			}
			expiresAt = &parsed
		}
	}

	claims, _ := auth.FromContext(r.Context())
	res, err := s.GrantAuditorAccess(r.Context(), GrantAuditorAccessInput{
		TenantID:        tenantID,
		UserID:          req.UserID,
		TargetActorType: firstNonEmpty(req.TargetActorType, req.ActorType),
		ExpiresAt:       expiresAt,
		TTLHours:        req.TTLHours,
		Reason:          req.Reason,
		GrantedBy:       auth.ActorFromContext(r.Context()),
		ActorRole:       auth.EffectiveSupplierRole(claims),
		ActorType:       claims.ActorType,
		TraceID:         firstNonEmpty(outbox.TraceIDFromContext(r.Context()), claims.TraceID),
		IP:              r.RemoteAddr,
		DeviceID:        strings.TrimSpace(r.Header.Get("X-Device-Id")),
		JWTSecret:       secret,
		JWTIssuer:       issuer,
	})
	if err != nil {
		if errors.Is(err, ErrAuditorGrantNotFound) {
			writeIAMError(w, http.StatusNotFound, "auditor_grant_not_found", err.Error())
			return
		}
		writeIAMDomainError(w, err)
		return
	}
	writeIAMJSON(w, http.StatusCreated, res)
}

// HandleListAuditorGrants serves GET /v1/iam/auditor-grants.
func (s *Service) HandleListAuditorGrants(w http.ResponseWriter, r *http.Request) {
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
	grants, err := s.ListAuditorGrants(r.Context(), tenantID)
	if err != nil {
		writeIAMDomainError(w, err)
		return
	}
	if grants == nil {
		grants = []AuditorGrant{}
	}
	writeIAMJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"grants":    grants,
		"count":     len(grants),
	})
}

// HandleRevokeAuditorGrant serves POST /v1/iam/auditor-grants/{userID}/revoke.
func (s *Service) HandleRevokeAuditorGrant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeIAMError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeIAMError(w, http.StatusServiceUnavailable, "iam_unavailable", "IAM service unavailable")
		return
	}

	userID := strings.TrimSpace(firstNonEmpty(chi.URLParam(r, "userID"), chi.URLParam(r, "userId")))
	if userID == "" {
		writeIAMError(w, http.StatusBadRequest, "user_id_required", "userID is required")
		return
	}

	var req revokeAuditorGrantRequest
	if r.Body != nil {
		body, err := io.ReadAll(io.LimitReader(r.Body, 16*1024))
		if err != nil {
			writeIAMError(w, http.StatusBadRequest, "read_body_failed", "unable to read request body")
			return
		}
		_ = r.Body.Close()
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				writeIAMError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload")
				return
			}
		}
	}

	requestedEntity := firstNonEmpty(req.EntityID, req.TenantID, r.URL.Query().Get("entity"))
	tenantID, ok := resolveIAMTenantScope(w, r, requestedEntity, false)
	if !ok {
		return
	}

	claims, _ := auth.FromContext(r.Context())
	if err := s.RevokeAuditorGrant(r.Context(), RevokeAuditorGrantInput{
		TenantID:  tenantID,
		UserID:    userID,
		RevokedBy: auth.ActorFromContext(r.Context()),
		ActorRole: auth.EffectiveSupplierRole(claims),
		TraceID:   firstNonEmpty(outbox.TraceIDFromContext(r.Context()), claims.TraceID),
		IP:        r.RemoteAddr,
		DeviceID:  strings.TrimSpace(r.Header.Get("X-Device-Id")),
	}); err != nil {
		if errors.Is(err, ErrAuditorGrantNotFound) {
			writeIAMError(w, http.StatusNotFound, "auditor_grant_not_found", err.Error())
			return
		}
		writeIAMDomainError(w, err)
		return
	}
	writeIAMJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
		"revoked":   true,
	})
}

func parseTimestampRFC3339(raw string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}
