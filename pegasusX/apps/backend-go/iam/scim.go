package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

const scimContentType = "application/scim+json; charset=utf-8"

type scimUserInput struct {
	Schemas     []string      `json:"schemas,omitempty"`
	ID          string        `json:"id,omitempty"`
	ExternalID  string        `json:"externalId,omitempty"`
	UserName    string        `json:"userName"`
	DisplayName string        `json:"displayName,omitempty"`
	Emails      []ScimEmail   `json:"emails,omitempty"`
	Active      *bool         `json:"active,omitempty"`
	Roles       []ScimRoleRef `json:"roles,omitempty"`
}

type scimGroupInput struct {
	Schemas     []string        `json:"schemas,omitempty"`
	ID          string          `json:"id,omitempty"`
	ExternalID  string          `json:"externalId,omitempty"`
	DisplayName string          `json:"displayName"`
	Members     []ScimMemberRef `json:"members,omitempty"`
	Roles       []ScimRoleRef   `json:"roles,omitempty"`
}

func (r ScimResourceRecord) toScimUser() ScimUser {
	emails := []ScimEmail{}
	if strings.TrimSpace(r.Email) != "" {
		emails = append(emails, ScimEmail{
			Value:   r.Email,
			Type:    "work",
			Primary: true,
		})
	}
	roles := append([]ScimRoleRef(nil), r.Roles...)
	if roles == nil {
		roles = []ScimRoleRef{}
	}
	meta := r.Meta
	if meta.ResourceType == "" {
		meta.ResourceType = ScimResourceTypeUser
	}
	if meta.Created == "" && !r.CreatedAt.IsZero() {
		meta.Created = r.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	if meta.LastModified == "" && !r.UpdatedAt.IsZero() {
		meta.LastModified = r.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	if meta.Location == "" {
		meta.Location = "/scim/v2/Users/" + r.ResourceID
	}
	return ScimUser{
		Schemas:     []string{ScimSchemaUser},
		ID:          r.ResourceID,
		ExternalID:  r.ExternalID,
		UserName:    r.UserName,
		DisplayName: r.DisplayName,
		Emails:      emails,
		Active:      r.Active,
		Roles:       roles,
		Meta:        meta,
	}
}

func (r ScimResourceRecord) toScimGroup() ScimGroup {
	members := append([]ScimMemberRef(nil), r.Members...)
	if members == nil {
		members = []ScimMemberRef{}
	}
	roles := append([]ScimRoleRef(nil), r.Roles...)
	if roles == nil {
		roles = []ScimRoleRef{}
	}
	meta := r.Meta
	if meta.ResourceType == "" {
		meta.ResourceType = ScimResourceTypeGroup
	}
	if meta.Created == "" && !r.CreatedAt.IsZero() {
		meta.Created = r.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	if meta.LastModified == "" && !r.UpdatedAt.IsZero() {
		meta.LastModified = r.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	if meta.Location == "" {
		meta.Location = "/scim/v2/Groups/" + r.ResourceID
	}
	return ScimGroup{
		Schemas:     []string{ScimSchemaGroup},
		ID:          r.ResourceID,
		ExternalID:  r.ExternalID,
		DisplayName: r.DisplayName,
		Members:     members,
		Roles:       roles,
		Meta:        meta,
	}
}

func primaryScimEmail(emails []ScimEmail) string {
	for _, e := range emails {
		if e.Primary && strings.TrimSpace(e.Value) != "" {
			return strings.TrimSpace(e.Value)
		}
	}
	for _, e := range emails {
		if strings.TrimSpace(e.Value) != "" {
			return strings.TrimSpace(e.Value)
		}
	}
	return ""
}

func normalizeScimRoles(roles []ScimRoleRef) ([]ScimRoleRef, error) {
	if len(roles) == 0 {
		return []ScimRoleRef{}, nil
	}
	seen := make(map[string]struct{}, len(roles))
	out := make([]ScimRoleRef, 0, len(roles))
	for _, r := range roles {
		raw := strings.TrimSpace(r.Value)
		if raw == "" {
			raw = strings.TrimSpace(r.Display)
		}
		if raw == "" {
			continue
		}
		norm := auth.NormalizeSupplierRoleName(raw)
		if norm == string(auth.RolePlatformAdmin) {
			return nil, fmt.Errorf("%w: SCIM cannot provision PLATFORM_ADMIN role", ErrPrivilegeEscalation)
		}
		if _, dup := seen[norm]; dup {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, ScimRoleRef{
			Value:   norm,
			Display: nonEmptySCIM(r.Display, norm),
			Type:    r.Type,
			Primary: r.Primary,
		})
	}
	return out, nil
}

func normalizeScimMembers(members []ScimMemberRef) []ScimMemberRef {
	if len(members) == 0 {
		return []ScimMemberRef{}
	}
	seen := make(map[string]struct{}, len(members))
	out := make([]ScimMemberRef, 0, len(members))
	for _, m := range members {
		val := strings.TrimSpace(m.Value)
		if val == "" {
			continue
		}
		if _, dup := seen[val]; dup {
			continue
		}
		seen[val] = struct{}{}
		ref := m.Ref
		if ref == "" {
			ref = "/scim/v2/Users/" + val
		}
		out = append(out, ScimMemberRef{
			Value:   val,
			Display: strings.TrimSpace(m.Display),
			Type:    nonEmptySCIM(m.Type, ScimResourceTypeUser),
			Ref:     ref,
		})
	}
	return out
}

func nonEmptySCIM(v, fallback string) string {
	if s := strings.TrimSpace(v); s != "" {
		return s
	}
	return fallback
}

func scimActorMeta(ctx context.Context, r *http.Request) (actorID, actorRole, actorType, traceID, ip, deviceID string) {
	actorID = strings.TrimSpace(auth.ActorFromContext(ctx))
	claims, ok := auth.FromContext(ctx)
	if ok {
		actorRole = auth.EffectiveSupplierRole(claims)
		actorType = auth.EffectiveActorType(claims)
		traceID = claims.TraceID
	}
	if actorType == "" {
		actorType = auth.ActorTypeConnector
	}
	if r != nil {
		traceID = firstNonEmpty(outbox.TraceIDFromContext(ctx), traceID, r.Header.Get("X-Trace-Id"))
		ip = r.RemoteAddr
		deviceID = strings.TrimSpace(r.Header.Get("X-Device-Id"))
	}
	return
}

func (s *Service) scimAuditFn(tid, actorID, actorType, actorRole, action, resourceType, resourceID, beforeJSON, afterJSON, traceID, ip, deviceID string) func(context.Context, *spanner.ReadWriteTransaction) error {
	return func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if s.audit == nil {
			return nil
		}
		ev := AuditRecordInput{
			TenantID:   tid,
			ActorID:    actorID,
			ActorType:  actorType,
			Role:       actorRole,
			Action:     action,
			ObjectType: "Scim" + resourceType,
			ObjectID:   resourceID,
			BeforeJSON: beforeJSON,
			AfterJSON:  afterJSON,
			TraceID:    traceID,
			IP:         ip,
			DeviceID:   deviceID,
		}
		if txn != nil {
			return s.audit.RecordEventInTxn(txCtx, txn, ev)
		}
		return s.audit.RecordEvent(txCtx, ev)
	}
}

func (s *Service) guardScimWriteActor(ctx context.Context) error {
	claims, ok := auth.FromContext(ctx)
	if !ok {
		return nil
	}
	if auth.IsReadOnlyAuditor(claims) {
		return fmt.Errorf("%w: AUDITOR cannot mutate SCIM identities", ErrAuditorReadOnly)
	}
	if auth.IsAgentActor(claims) {
		return fmt.Errorf("%w: AGENT actor cannot mutate SCIM identities", ErrAgentApprovalForbidden)
	}
	return nil
}

// HandleListScimUsers serves GET /scim/v2/Users.
func (s *Service) HandleListScimUsers(w http.ResponseWriter, r *http.Request) {
	s.handleListScim(w, r, ScimResourceTypeUser)
}

// HandleListScimGroups serves GET /scim/v2/Groups.
func (s *Service) HandleListScimGroups(w http.ResponseWriter, r *http.Request) {
	s.handleListScim(w, r, ScimResourceTypeGroup)
}

func (s *Service) handleListScim(w http.ResponseWriter, r *http.Request, resourceType string) {
	if r.Method != http.MethodGet {
		writeScimError(w, http.StatusMethodNotAllowed, "invalidMethod", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeScimError(w, http.StatusServiceUnavailable, "unavailable", "IAM service unavailable")
		return
	}
	tenantID, ok := resolveScimTenantScope(w, r)
	if !ok {
		return
	}
	startIndex := 1
	if raw := strings.TrimSpace(r.URL.Query().Get("startIndex")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 {
			startIndex = n
		}
	}
	count := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("count")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 && n <= 500 {
			count = n
		}
	}
	filter := strings.TrimSpace(r.URL.Query().Get("filter"))

	rows, total, err := s.repo.ListScimResources(r.Context(), tenantID, resourceType, filter, startIndex, count)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	resources := make([]any, 0, len(rows))
	for _, rec := range rows {
		if resourceType == ScimResourceTypeUser {
			resources = append(resources, rec.toScimUser())
		} else {
			resources = append(resources, rec.toScimGroup())
		}
	}
	writeScimJSON(w, http.StatusOK, ScimListResponse{
		Schemas:      []string{ScimSchemaListResponse},
		TotalResults: total,
		StartIndex:   startIndex,
		ItemsPerPage: len(resources),
		Resources:    resources,
	})
}

// HandleGetScimUser serves GET /scim/v2/Users/{id}.
func (s *Service) HandleGetScimUser(w http.ResponseWriter, r *http.Request) {
	s.handleGetScim(w, r, ScimResourceTypeUser)
}

// HandleGetScimGroup serves GET /scim/v2/Groups/{id}.
func (s *Service) HandleGetScimGroup(w http.ResponseWriter, r *http.Request) {
	s.handleGetScim(w, r, ScimResourceTypeGroup)
}

func (s *Service) handleGetScim(w http.ResponseWriter, r *http.Request, resourceType string) {
	if r.Method != http.MethodGet {
		writeScimError(w, http.StatusMethodNotAllowed, "invalidMethod", "Method Not Allowed")
		return
	}
	if s == nil || s.repo == nil {
		writeScimError(w, http.StatusServiceUnavailable, "unavailable", "IAM service unavailable")
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeScimError(w, http.StatusBadRequest, "invalidValue", "resource id is required")
		return
	}
	tenantID, ok := resolveScimTenantScope(w, r)
	if !ok {
		return
	}
	rec, found, err := s.repo.GetScimResource(r.Context(), tenantID, resourceType, id)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	if !found {
		writeScimError(w, http.StatusNotFound, "notFound", fmt.Sprintf("SCIM %s %s not found", resourceType, id))
		return
	}
	if resourceType == ScimResourceTypeUser {
		writeScimJSON(w, http.StatusOK, rec.toScimUser())
		return
	}
	writeScimJSON(w, http.StatusOK, rec.toScimGroup())
}

// HandleCreateScimUser serves POST /scim/v2/Users.
func (s *Service) HandleCreateScimUser(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeScimError(w, http.StatusServiceUnavailable, "unavailable", "IAM service unavailable")
		return
	}
	if err := s.guardScimWriteActor(r.Context()); err != nil {
		writeScimDomainError(w, err)
		return
	}
	tenantID, ok := resolveScimTenantScope(w, r)
	if !ok {
		return
	}
	var in scimUserInput
	if !decodeScimBody(w, r, &in) {
		return
	}
	userName := strings.TrimSpace(in.UserName)
	if userName == "" {
		writeScimError(w, http.StatusBadRequest, "invalidValue", "userName is required")
		return
	}
	roles, err := normalizeScimRoles(in.Roles)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = "scim_u_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
	}

	// Reject duplicate userName in tenant.
	if existing, _, lerr := s.repo.ListScimResources(r.Context(), tenantID, ScimResourceTypeUser, "", 1, 500); lerr == nil {
		for _, cur := range existing {
			if strings.EqualFold(cur.UserName, userName) {
				writeScimError(w, http.StatusConflict, "uniqueness", fmt.Sprintf("userName %q already exists", userName))
				return
			}
		}
	}

	now := s.currentNow()
	rec := ScimResourceRecord{
		TenantID:     tenantID,
		ResourceType: ScimResourceTypeUser,
		ResourceID:   id,
		ExternalID:   strings.TrimSpace(in.ExternalID),
		UserName:     userName,
		DisplayName:  nonEmptySCIM(in.DisplayName, userName),
		Email:        primaryScimEmail(in.Emails),
		Active:       active,
		Roles:        roles,
		Meta: ScimMeta{
			ResourceType: ScimResourceTypeUser,
			Created:      now.Format(time.RFC3339Nano),
			LastModified: now.Format(time.RFC3339Nano),
			Location:     "/scim/v2/Users/" + id,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	actorID, actorRole, actorType, traceID, ip, deviceID := scimActorMeta(r.Context(), r)
	afterRaw, _ := json.Marshal(rec.toScimUser())
	saved, _, err := s.repo.UpsertScimResource(
		r.Context(), rec, "CREATE", actorID, traceID,
		s.scimAuditFn(tenantID, actorID, actorType, actorRole, "iam.scim.user_created", ScimResourceTypeUser, id, "", string(afterRaw), traceID, ip, deviceID),
	)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	writeScimJSON(w, http.StatusCreated, saved.toScimUser())
}

// HandlePutScimUser serves PUT /scim/v2/Users/{id}.
func (s *Service) HandlePutScimUser(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeScimError(w, http.StatusServiceUnavailable, "unavailable", "IAM service unavailable")
		return
	}
	if err := s.guardScimWriteActor(r.Context()); err != nil {
		writeScimDomainError(w, err)
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeScimError(w, http.StatusBadRequest, "invalidValue", "user id is required")
		return
	}
	tenantID, ok := resolveScimTenantScope(w, r)
	if !ok {
		return
	}
	cur, found, err := s.repo.GetScimResource(r.Context(), tenantID, ScimResourceTypeUser, id)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	if !found {
		writeScimError(w, http.StatusNotFound, "notFound", "SCIM User not found")
		return
	}
	beforeRaw, _ := json.Marshal(cur.toScimUser())

	var in scimUserInput
	if !decodeScimBody(w, r, &in) {
		return
	}
	userName := strings.TrimSpace(in.UserName)
	if userName == "" {
		writeScimError(w, http.StatusBadRequest, "invalidValue", "userName is required")
		return
	}
	roles, err := normalizeScimRoles(in.Roles)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	now := s.currentNow()
	cur.ExternalID = strings.TrimSpace(in.ExternalID)
	cur.UserName = userName
	cur.DisplayName = nonEmptySCIM(in.DisplayName, userName)
	cur.Email = primaryScimEmail(in.Emails)
	cur.Active = active
	cur.Roles = roles
	cur.UpdatedAt = now
	cur.Meta.LastModified = now.Format(time.RFC3339Nano)

	actorID, actorRole, actorType, traceID, ip, deviceID := scimActorMeta(r.Context(), r)
	afterRaw, _ := json.Marshal(cur.toScimUser())
	saved, _, err := s.repo.UpsertScimResource(
		r.Context(), cur, "UPDATE", actorID, traceID,
		s.scimAuditFn(tenantID, actorID, actorType, actorRole, "iam.scim.user_updated", ScimResourceTypeUser, id, string(beforeRaw), string(afterRaw), traceID, ip, deviceID),
	)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	writeScimJSON(w, http.StatusOK, saved.toScimUser())
}

// HandlePatchScimUser serves PATCH /scim/v2/Users/{id} (RFC 7644 PatchOp).
func (s *Service) HandlePatchScimUser(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeScimError(w, http.StatusServiceUnavailable, "unavailable", "IAM service unavailable")
		return
	}
	if err := s.guardScimWriteActor(r.Context()); err != nil {
		writeScimDomainError(w, err)
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeScimError(w, http.StatusBadRequest, "invalidValue", "user id is required")
		return
	}
	tenantID, ok := resolveScimTenantScope(w, r)
	if !ok {
		return
	}
	cur, found, err := s.repo.GetScimResource(r.Context(), tenantID, ScimResourceTypeUser, id)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	if !found {
		writeScimError(w, http.StatusNotFound, "notFound", "SCIM User not found")
		return
	}
	beforeRaw, _ := json.Marshal(cur.toScimUser())

	var patch ScimPatchRequest
	if !decodeScimBody(w, r, &patch) {
		return
	}
	if len(patch.Operations) == 0 {
		writeScimError(w, http.StatusBadRequest, "invalidSyntax", "Operations must not be empty")
		return
	}

	if err := applyScimUserPatch(&cur, patch.Operations); err != nil {
		writeScimDomainError(w, err)
		return
	}
	roles, err := normalizeScimRoles(cur.Roles)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	cur.Roles = roles
	now := s.currentNow()
	cur.UpdatedAt = now
	cur.Meta.LastModified = now.Format(time.RFC3339Nano)

	actorID, actorRole, actorType, traceID, ip, deviceID := scimActorMeta(r.Context(), r)
	afterRaw, _ := json.Marshal(cur.toScimUser())
	saved, _, err := s.repo.UpsertScimResource(
		r.Context(), cur, "PATCH", actorID, traceID,
		s.scimAuditFn(tenantID, actorID, actorType, actorRole, "iam.scim.user_patched", ScimResourceTypeUser, id, string(beforeRaw), string(afterRaw), traceID, ip, deviceID),
	)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	writeScimJSON(w, http.StatusOK, saved.toScimUser())
}

// HandleDeleteScimUser serves DELETE /scim/v2/Users/{id}.
func (s *Service) HandleDeleteScimUser(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeScimError(w, http.StatusServiceUnavailable, "unavailable", "IAM service unavailable")
		return
	}
	if err := s.guardScimWriteActor(r.Context()); err != nil {
		writeScimDomainError(w, err)
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeScimError(w, http.StatusBadRequest, "invalidValue", "user id is required")
		return
	}
	tenantID, ok := resolveScimTenantScope(w, r)
	if !ok {
		return
	}
	cur, found, err := s.repo.GetScimResource(r.Context(), tenantID, ScimResourceTypeUser, id)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	if !found {
		writeScimError(w, http.StatusNotFound, "notFound", "SCIM User not found")
		return
	}
	beforeRaw, _ := json.Marshal(cur.toScimUser())
	actorID, actorRole, actorType, traceID, ip, deviceID := scimActorMeta(r.Context(), r)
	deleted, _, err := s.repo.DeleteScimResource(
		r.Context(), tenantID, ScimResourceTypeUser, id, actorID, traceID,
		s.scimAuditFn(tenantID, actorID, actorType, actorRole, "iam.scim.user_deprovisioned", ScimResourceTypeUser, id, string(beforeRaw), `{"deleted":true}`, traceID, ip, deviceID),
	)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	if !deleted {
		writeScimError(w, http.StatusNotFound, "notFound", "SCIM User not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleCreateScimGroup serves POST /scim/v2/Groups.
func (s *Service) HandleCreateScimGroup(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeScimError(w, http.StatusServiceUnavailable, "unavailable", "IAM service unavailable")
		return
	}
	if err := s.guardScimWriteActor(r.Context()); err != nil {
		writeScimDomainError(w, err)
		return
	}
	tenantID, ok := resolveScimTenantScope(w, r)
	if !ok {
		return
	}
	var in scimGroupInput
	if !decodeScimBody(w, r, &in) {
		return
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		writeScimError(w, http.StatusBadRequest, "invalidValue", "displayName is required")
		return
	}
	roles, err := normalizeScimRoles(in.Roles)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = "scim_g_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
	}

	now := s.currentNow()
	rec := ScimResourceRecord{
		TenantID:     tenantID,
		ResourceType: ScimResourceTypeGroup,
		ResourceID:   id,
		ExternalID:   strings.TrimSpace(in.ExternalID),
		DisplayName:  displayName,
		Active:       true,
		Members:      normalizeScimMembers(in.Members),
		Roles:        roles,
		Meta: ScimMeta{
			ResourceType: ScimResourceTypeGroup,
			Created:      now.Format(time.RFC3339Nano),
			LastModified: now.Format(time.RFC3339Nano),
			Location:     "/scim/v2/Groups/" + id,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	actorID, actorRole, actorType, traceID, ip, deviceID := scimActorMeta(r.Context(), r)
	afterRaw, _ := json.Marshal(rec.toScimGroup())
	saved, _, err := s.repo.UpsertScimResource(
		r.Context(), rec, "CREATE", actorID, traceID,
		s.scimAuditFn(tenantID, actorID, actorType, actorRole, "iam.scim.group_created", ScimResourceTypeGroup, id, "", string(afterRaw), traceID, ip, deviceID),
	)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	writeScimJSON(w, http.StatusCreated, saved.toScimGroup())
}

// HandlePutScimGroup serves PUT /scim/v2/Groups/{id}.
func (s *Service) HandlePutScimGroup(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeScimError(w, http.StatusServiceUnavailable, "unavailable", "IAM service unavailable")
		return
	}
	if err := s.guardScimWriteActor(r.Context()); err != nil {
		writeScimDomainError(w, err)
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeScimError(w, http.StatusBadRequest, "invalidValue", "group id is required")
		return
	}
	tenantID, ok := resolveScimTenantScope(w, r)
	if !ok {
		return
	}
	cur, found, err := s.repo.GetScimResource(r.Context(), tenantID, ScimResourceTypeGroup, id)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	if !found {
		writeScimError(w, http.StatusNotFound, "notFound", "SCIM Group not found")
		return
	}
	beforeRaw, _ := json.Marshal(cur.toScimGroup())

	var in scimGroupInput
	if !decodeScimBody(w, r, &in) {
		return
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		writeScimError(w, http.StatusBadRequest, "invalidValue", "displayName is required")
		return
	}
	roles, err := normalizeScimRoles(in.Roles)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	now := s.currentNow()
	cur.ExternalID = strings.TrimSpace(in.ExternalID)
	cur.DisplayName = displayName
	cur.Members = normalizeScimMembers(in.Members)
	cur.Roles = roles
	cur.UpdatedAt = now
	cur.Meta.LastModified = now.Format(time.RFC3339Nano)

	actorID, actorRole, actorType, traceID, ip, deviceID := scimActorMeta(r.Context(), r)
	afterRaw, _ := json.Marshal(cur.toScimGroup())
	saved, _, err := s.repo.UpsertScimResource(
		r.Context(), cur, "UPDATE", actorID, traceID,
		s.scimAuditFn(tenantID, actorID, actorType, actorRole, "iam.scim.group_updated", ScimResourceTypeGroup, id, string(beforeRaw), string(afterRaw), traceID, ip, deviceID),
	)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	writeScimJSON(w, http.StatusOK, saved.toScimGroup())
}

// HandlePatchScimGroup serves PATCH /scim/v2/Groups/{id}.
func (s *Service) HandlePatchScimGroup(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeScimError(w, http.StatusServiceUnavailable, "unavailable", "IAM service unavailable")
		return
	}
	if err := s.guardScimWriteActor(r.Context()); err != nil {
		writeScimDomainError(w, err)
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeScimError(w, http.StatusBadRequest, "invalidValue", "group id is required")
		return
	}
	tenantID, ok := resolveScimTenantScope(w, r)
	if !ok {
		return
	}
	cur, found, err := s.repo.GetScimResource(r.Context(), tenantID, ScimResourceTypeGroup, id)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	if !found {
		writeScimError(w, http.StatusNotFound, "notFound", "SCIM Group not found")
		return
	}
	beforeRaw, _ := json.Marshal(cur.toScimGroup())

	var patch ScimPatchRequest
	if !decodeScimBody(w, r, &patch) {
		return
	}
	if len(patch.Operations) == 0 {
		writeScimError(w, http.StatusBadRequest, "invalidSyntax", "Operations must not be empty")
		return
	}
	if err := applyScimGroupPatch(&cur, patch.Operations); err != nil {
		writeScimDomainError(w, err)
		return
	}
	roles, err := normalizeScimRoles(cur.Roles)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	cur.Roles = roles
	cur.Members = normalizeScimMembers(cur.Members)
	now := s.currentNow()
	cur.UpdatedAt = now
	cur.Meta.LastModified = now.Format(time.RFC3339Nano)

	actorID, actorRole, actorType, traceID, ip, deviceID := scimActorMeta(r.Context(), r)
	afterRaw, _ := json.Marshal(cur.toScimGroup())
	saved, _, err := s.repo.UpsertScimResource(
		r.Context(), cur, "PATCH", actorID, traceID,
		s.scimAuditFn(tenantID, actorID, actorType, actorRole, "iam.scim.group_patched", ScimResourceTypeGroup, id, string(beforeRaw), string(afterRaw), traceID, ip, deviceID),
	)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	writeScimJSON(w, http.StatusOK, saved.toScimGroup())
}

// HandleDeleteScimGroup serves DELETE /scim/v2/Groups/{id}.
func (s *Service) HandleDeleteScimGroup(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.repo == nil {
		writeScimError(w, http.StatusServiceUnavailable, "unavailable", "IAM service unavailable")
		return
	}
	if err := s.guardScimWriteActor(r.Context()); err != nil {
		writeScimDomainError(w, err)
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeScimError(w, http.StatusBadRequest, "invalidValue", "group id is required")
		return
	}
	tenantID, ok := resolveScimTenantScope(w, r)
	if !ok {
		return
	}
	cur, found, err := s.repo.GetScimResource(r.Context(), tenantID, ScimResourceTypeGroup, id)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	if !found {
		writeScimError(w, http.StatusNotFound, "notFound", "SCIM Group not found")
		return
	}
	beforeRaw, _ := json.Marshal(cur.toScimGroup())
	actorID, actorRole, actorType, traceID, ip, deviceID := scimActorMeta(r.Context(), r)
	deleted, _, err := s.repo.DeleteScimResource(
		r.Context(), tenantID, ScimResourceTypeGroup, id, actorID, traceID,
		s.scimAuditFn(tenantID, actorID, actorType, actorRole, "iam.scim.group_deleted", ScimResourceTypeGroup, id, string(beforeRaw), `{"deleted":true}`, traceID, ip, deviceID),
	)
	if err != nil {
		writeScimDomainError(w, err)
		return
	}
	if !deleted {
		writeScimError(w, http.StatusNotFound, "notFound", "SCIM Group not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func applyScimUserPatch(rec *ScimResourceRecord, ops []ScimPatchOperation) error {
	for _, op := range ops {
		kind := strings.ToLower(strings.TrimSpace(op.Op))
		if kind != "add" && kind != "replace" && kind != "remove" {
			return fmt.Errorf("%w: unsupported patch op %q", ErrInvalidScimRequest, op.Op)
		}
		path := strings.ToLower(strings.TrimSpace(op.Path))
		if path == "" {
			m, ok := op.Value.(map[string]any)
			if !ok {
				return fmt.Errorf("%w: patch value must be an object when path is omitted", ErrInvalidScimRequest)
			}
			for k, v := range m {
				if err := applyScimUserField(rec, kind, strings.ToLower(strings.TrimSpace(k)), v); err != nil {
					return err
				}
			}
			continue
		}
		if err := applyScimUserField(rec, kind, path, op.Value); err != nil {
			return err
		}
	}
	return nil
}

func applyScimUserField(rec *ScimResourceRecord, op, path string, val any) error {
	switch path {
	case "active":
		if op == "remove" {
			rec.Active = false
			return nil
		}
		b, ok := val.(bool)
		if !ok {
			return fmt.Errorf("%w: active must be boolean", ErrInvalidScimRequest)
		}
		rec.Active = b
	case "username":
		if op == "remove" {
			return fmt.Errorf("%w: cannot remove required userName", ErrInvalidScimRequest)
		}
		s, ok := val.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return fmt.Errorf("%w: userName must be non-empty string", ErrInvalidScimRequest)
		}
		rec.UserName = strings.TrimSpace(s)
	case "displayname":
		if op == "remove" {
			rec.DisplayName = ""
			return nil
		}
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("%w: displayName must be string", ErrInvalidScimRequest)
		}
		rec.DisplayName = strings.TrimSpace(s)
	case "externalid":
		if op == "remove" {
			rec.ExternalID = ""
			return nil
		}
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("%w: externalId must be string", ErrInvalidScimRequest)
		}
		rec.ExternalID = strings.TrimSpace(s)
	case "emails":
		if op == "remove" {
			rec.Email = ""
			return nil
		}
		emails, err := decodeScimEmailsValue(val)
		if err != nil {
			return err
		}
		rec.Email = primaryScimEmail(emails)
	case "roles":
		if op == "remove" {
			rec.Roles = []ScimRoleRef{}
			return nil
		}
		roles, err := decodeScimRolesValue(val)
		if err != nil {
			return err
		}
		if op == "add" {
			rec.Roles = append(rec.Roles, roles...)
		} else {
			rec.Roles = roles
		}
	default:
		if strings.HasPrefix(path, "roles[") && op == "remove" {
			rec.Roles = []ScimRoleRef{}
			return nil
		}
		return fmt.Errorf("%w: unsupported SCIM User patch path %q", ErrInvalidScimRequest, path)
	}
	return nil
}

func applyScimGroupPatch(rec *ScimResourceRecord, ops []ScimPatchOperation) error {
	for _, op := range ops {
		kind := strings.ToLower(strings.TrimSpace(op.Op))
		if kind != "add" && kind != "replace" && kind != "remove" {
			return fmt.Errorf("%w: unsupported patch op %q", ErrInvalidScimRequest, op.Op)
		}
		path := strings.ToLower(strings.TrimSpace(op.Path))
		if path == "" {
			m, ok := op.Value.(map[string]any)
			if !ok {
				return fmt.Errorf("%w: patch value must be an object when path is omitted", ErrInvalidScimRequest)
			}
			for k, v := range m {
				if err := applyScimGroupField(rec, kind, strings.ToLower(strings.TrimSpace(k)), v); err != nil {
					return err
				}
			}
			continue
		}
		if err := applyScimGroupField(rec, kind, path, op.Value); err != nil {
			return err
		}
	}
	return nil
}

func applyScimGroupField(rec *ScimResourceRecord, op, path string, val any) error {
	switch path {
	case "displayname":
		if op == "remove" {
			return fmt.Errorf("%w: cannot remove required displayName", ErrInvalidScimRequest)
		}
		s, ok := val.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return fmt.Errorf("%w: displayName must be non-empty string", ErrInvalidScimRequest)
		}
		rec.DisplayName = strings.TrimSpace(s)
	case "externalid":
		if op == "remove" {
			rec.ExternalID = ""
			return nil
		}
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("%w: externalId must be string", ErrInvalidScimRequest)
		}
		rec.ExternalID = strings.TrimSpace(s)
	case "members":
		if op == "remove" {
			if val == nil {
				rec.Members = []ScimMemberRef{}
				return nil
			}
			toRemove, err := decodeScimMembersValue(val)
			if err != nil {
				return err
			}
			rmSet := make(map[string]struct{}, len(toRemove))
			for _, m := range toRemove {
				rmSet[strings.TrimSpace(m.Value)] = struct{}{}
			}
			kept := make([]ScimMemberRef, 0, len(rec.Members))
			for _, m := range rec.Members {
				if _, rm := rmSet[strings.TrimSpace(m.Value)]; !rm {
					kept = append(kept, m)
				}
			}
			rec.Members = kept
			return nil
		}
		members, err := decodeScimMembersValue(val)
		if err != nil {
			return err
		}
		if op == "add" {
			rec.Members = append(rec.Members, members...)
		} else {
			rec.Members = members
		}
	case "roles":
		if op == "remove" {
			rec.Roles = []ScimRoleRef{}
			return nil
		}
		roles, err := decodeScimRolesValue(val)
		if err != nil {
			return err
		}
		if op == "add" {
			rec.Roles = append(rec.Roles, roles...)
		} else {
			rec.Roles = roles
		}
	default:
		if strings.HasPrefix(path, "members[") && op == "remove" {
			rec.Members = []ScimMemberRef{}
			return nil
		}
		return fmt.Errorf("%w: unsupported SCIM Group patch path %q", ErrInvalidScimRequest, path)
	}
	return nil
}

func decodeScimEmailsValue(val any) ([]ScimEmail, error) {
	raw, err := json.Marshal(val)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid emails value", ErrInvalidScimRequest)
	}
	var out []ScimEmail
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%w: emails must be an array", ErrInvalidScimRequest)
	}
	return out, nil
}

func decodeScimRolesValue(val any) ([]ScimRoleRef, error) {
	raw, err := json.Marshal(val)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid roles value", ErrInvalidScimRequest)
	}
	var out []ScimRoleRef
	if err := json.Unmarshal(raw, &out); err == nil {
		return out, nil
	}
	var strList []string
	if err := json.Unmarshal(raw, &strList); err == nil {
		res := make([]ScimRoleRef, 0, len(strList))
		for _, s := range strList {
			res = append(res, ScimRoleRef{Value: s})
		}
		return res, nil
	}
	return nil, fmt.Errorf("%w: roles must be an array", ErrInvalidScimRequest)
}

func decodeScimMembersValue(val any) ([]ScimMemberRef, error) {
	raw, err := json.Marshal(val)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid members value", ErrInvalidScimRequest)
	}
	var out []ScimMemberRef
	if err := json.Unmarshal(raw, &out); err == nil {
		return out, nil
	}
	var single ScimMemberRef
	if err := json.Unmarshal(raw, &single); err == nil && strings.TrimSpace(single.Value) != "" {
		return []ScimMemberRef{single}, nil
	}
	return nil, fmt.Errorf("%w: members must be an array", ErrInvalidScimRequest)
}

func resolveScimTenantScope(w http.ResponseWriter, r *http.Request) (string, bool) {
	reqTenant := firstNonEmpty(
		r.Header.Get("X-Tenant-Id"),
		r.Header.Get("X-Supplier-Id"),
		r.URL.Query().Get("entity"),
		r.URL.Query().Get("tenant_id"),
	)
	tenantID, ok := resolveIAMTenantScope(w, r, reqTenant, false)
	if !ok {
		return "", false
	}
	return tenantID, true
}

func decodeScimBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil {
		writeScimError(w, http.StatusBadRequest, "invalidSyntax", "request body is required")
		return false
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	if err != nil {
		writeScimError(w, http.StatusBadRequest, "invalidSyntax", "unable to read request body")
		return false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		writeScimError(w, http.StatusBadRequest, "invalidSyntax", "request body is required")
		return false
	}
	if err := json.Unmarshal(body, dst); err != nil {
		writeScimError(w, http.StatusBadRequest, "invalidSyntax", "invalid JSON payload")
		return false
	}
	return true
}

func writeScimJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", scimContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeScimError(w http.ResponseWriter, status int, scimType, detail string) {
	writeScimJSON(w, status, map[string]any{
		"schemas":  []string{ScimSchemaError},
		"status":   strconv.Itoa(status),
		"scimType": scimType,
		"detail":   detail,
		"error":    detail,
	})
}

func writeScimDomainError(w http.ResponseWriter, err error) {
	var conflict *StaticRoleConflictError
	switch {
	case errors.As(err, &conflict):
		writeScimError(w, http.StatusForbidden, "sodViolation", err.Error())
	case errors.Is(err, ErrAuditorReadOnly), errors.Is(err, ErrAgentApprovalForbidden), errors.Is(err, ErrPrivilegeEscalation), errors.Is(err, ErrSodViolation), errors.Is(err, ErrSodRoleConflict):
		writeScimError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, ErrScimConflict):
		writeScimError(w, http.StatusConflict, "uniqueness", err.Error())
	case errors.Is(err, ErrScimResourceNotFound):
		writeScimError(w, http.StatusNotFound, "notFound", err.Error())
	case errors.Is(err, ErrInvalidScimRequest), errors.Is(err, ErrInvalidIAMInput):
		writeScimError(w, http.StatusBadRequest, "invalidValue", err.Error())
	case errors.Is(err, ErrIAMUnavailable):
		writeScimError(w, http.StatusServiceUnavailable, "unavailable", err.Error())
	default:
		writeScimError(w, http.StatusInternalServerError, "internalError", err.Error())
	}
}

func parseScimEqFilter(raw string) (field, val string, ok bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", false
	}
	parts := strings.SplitN(trimmed, " eq ", 2)
	if len(parts) != 2 {
		parts = strings.SplitN(trimmed, " EQ ", 2)
	}
	if len(parts) != 2 {
		return "", "", false
	}
	f := strings.ToLower(strings.TrimSpace(parts[0]))
	v := strings.TrimSpace(parts[1])
	v = strings.Trim(v, `"'`)
	if f == "" || v == "" {
		return "", "", false
	}
	return f, v, true
}

func matchesScimFilter(rec ScimResourceRecord, filter string) bool {
	if strings.TrimSpace(filter) == "" {
		return true
	}
	field, val, ok := parseScimEqFilter(filter)
	if !ok {
		return true
	}
	switch field {
	case "username":
		return strings.EqualFold(rec.UserName, val)
	case "displayname":
		return strings.EqualFold(rec.DisplayName, val)
	case "externalid":
		return strings.EqualFold(rec.ExternalID, val)
	case "email", "emails.value":
		return strings.EqualFold(rec.Email, val)
	case "id":
		return rec.ResourceID == val
	default:
		return true
	}
}

func resolveGroupSyncRoles(rec ScimResourceRecord, rolePerms map[string]map[string]struct{}) ([]string, error) {
	if len(rec.Roles) > 0 {
		out := make([]string, 0, len(rec.Roles))
		for _, r := range rec.Roles {
			rn := auth.NormalizeSupplierRoleName(r.Value)
			if rn == string(auth.RolePlatformAdmin) {
				return nil, fmt.Errorf("%w: SCIM cannot provision PLATFORM_ADMIN role", ErrPrivilegeEscalation)
			}
			if rolePerms[rn] == nil {
				return nil, fmt.Errorf("%w: unknown SCIM role %q", ErrInvalidScimRequest, r.Value)
			}
			out = append(out, rn)
		}
		return out, nil
	}
	candidate := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(rec.DisplayName), " ", "_"))
	if candidate != "" && candidate != string(auth.RolePlatformAdmin) && rolePerms[candidate] != nil {
		return []string{candidate}, nil
	}
	return nil, nil
}

func cloneScimRecord(rec ScimResourceRecord) ScimResourceRecord {
	cp := rec
	cp.Roles = append([]ScimRoleRef(nil), rec.Roles...)
	cp.Members = append([]ScimMemberRef(nil), rec.Members...)
	return cp
}
