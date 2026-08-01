package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

var (
	errRetailerMemberNotFound      = errors.New("retailer_org_member_not_found")
	errRetailerMemberPhoneExists   = errors.New("retailer_org_member_phone_exists")
	errRetailerMemberOwnerLocked   = errors.New("retailer_org_member_owner_locked")
	errRetailerMemberInvalidRole   = errors.New("retailer_org_member_invalid_role")
	errRetailerTeamPackRequired    = errors.New("retailer_team_pack_required")
)

// OrgMemberDTO is the wire shape for team roster rows.
type OrgMemberDTO struct {
	UserID       string   `json:"user_id"`
	RetailerID   string   `json:"retailer_id"`
	Name         string   `json:"name"`
	Phone        string   `json:"phone"`
	RetailerRole string   `json:"retailer_role"`
	IsOwner      bool     `json:"is_owner"`
	IsActive     bool     `json:"is_active"`
	LocationIDs  []string `json:"location_ids,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

type orgMembersResponse struct {
	RetailerID string         `json:"retailer_id"`
	Items      []OrgMemberDTO `json:"items"`
	UpdatedAt  string         `json:"updated_at"`
}

type orgMemberCreateRequest struct {
	Name         string   `json:"name"`
	Phone        string   `json:"phone"`
	Password     string   `json:"password"`
	RetailerRole string   `json:"retailer_role"`
	IsActive     *bool    `json:"is_active,omitempty"`
	LocationIDs  []string `json:"location_ids,omitempty"`
}

type orgMemberUpdateRequest struct {
	Name         *string `json:"name,omitempty"`
	RetailerRole *string `json:"retailer_role,omitempty"`
	Password     *string `json:"password,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}

// allowedAssignableRoles excludes OWNER (bootstrap-only).
var allowedAssignableRoles = map[string]bool{
	"ADMIN": true, "MANAGER": true, "BUYER": true, "RECEIVER": true,
	"CASHIER": true, "STOCK_CLERK": true, "SECTION_LEAD": true, "VIEWER": true,
}

// HandleOrgMembers serves GET/POST /v1/retailer/org/members.
func (s *Service) HandleOrgMembers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleOrgMembersGet(w, r)
	case http.MethodPost:
		s.handleOrgMembersPost(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleOrgMemberByID serves PATCH/PUT/DELETE /v1/retailer/org/members/{userID}.
func (s *Service) HandleOrgMemberByID(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(chi.URLParam(r, "userID"))
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id_required"})
		return
	}
	switch r.Method {
	case http.MethodPatch, http.MethodPut:
		s.handleOrgMemberPatch(w, r, userID)
	case http.MethodDelete:
		s.handleOrgMemberDeactivate(w, r, userID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleOrgMembersGet(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStaffManage) {
		// VIEWER/BUYER may still need to know own row — allow list for any authenticated retailer
		// but redact if no staff.manage: return only self.
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	items, err := s.listOrgMembers(r.Context(), orgID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_retailer_org_members_failed"})
		return
	}
	if ok && !auth.HasRetailerPerm(claims, auth.PermStaffManage) {
		uid := auth.ResolveRetailerUserID(claims)
		filtered := items[:0]
		for _, m := range items {
			if m.UserID == uid {
				filtered = append(filtered, m)
			}
		}
		items = filtered
	}
	writeJSON(w, http.StatusOK, orgMembersResponse{
		RetailerID: orgID,
		Items:      items,
		UpdatedAt:  s.now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *Service) handleOrgMembersPost(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStaffManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermStaffManage})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, okBody := readLimitedBody(w, r, 32*1024)
	if !okBody {
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req orgMemberCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	name := strings.TrimSpace(req.Name)
	phone := strings.TrimSpace(req.Phone)
	password := strings.TrimSpace(req.Password)
	role := strings.ToUpper(strings.TrimSpace(req.RetailerRole))
	if len(name) < 2 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "retailer_org_member_name_required"})
		return
	}
	if phone == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "retailer_org_member_phone_required"})
		return
	}
	if len(password) < 4 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "retailer_org_member_password_too_short"})
		return
	}
	if !allowedAssignableRoles[role] {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "retailer_org_member_invalid_role"})
		return
	}

	// TEAM pack: auto-enable if missing so owner can invite without extra step.
	enabled, _ := s.LoadEnabledPacks(r.Context(), orgID)
	if !enabled.Has(PackTEAM) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackTEAM, auth.ResolveRetailerUserID(claims), true, map[string]any{})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hash_password_failed"})
		return
	}
	user := RetailerUser{
		UserID:       s.newID(),
		RetailerID:   orgID,
		Phone:        phone,
		Name:         name,
		PasswordHash: string(hash),
		RetailerRole: role,
		IsOwner:      false,
		IsActive:     true,
		CreatedAt:    s.now(),
		UpdatedAt:    s.now(),
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if err := s.createOrgMember(r.Context(), user); err != nil {
		if errors.Is(err, errRetailerMemberPhoneExists) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "retailer_org_member_phone_exists"})
			return
		}
		s.log.Error("create retailer org member failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_retailer_org_member_failed"})
		return
	}
	// Phase 2: optional location scope for staff.
	if len(req.LocationIDs) > 0 {
		_ = s.replaceUserLocations(r.Context(), orgID, user.UserID, req.LocationIDs)
	}

	resp, err := s.loadOrgMembersResponse(r.Context(), orgID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_retailer_org_members_failed"})
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

func (s *Service) handleOrgMemberPatch(w http.ResponseWriter, r *http.Request, userID string) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStaffManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermStaffManage})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, okBody := readLimitedBody(w, r, 16*1024)
	if !okBody {
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req orgMemberUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	target, found, err := s.findRetailerUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_member_failed"})
		return
	}
	if !found || target.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "retailer_org_member_not_found"})
		return
	}
	if target.IsOwner {
		if req.RetailerRole != nil && strings.ToUpper(strings.TrimSpace(*req.RetailerRole)) != "OWNER" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "retailer_org_member_owner_locked"})
			return
		}
		if req.IsActive != nil && !*req.IsActive {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "retailer_org_member_owner_locked"})
			return
		}
	}
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if len(n) < 2 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "retailer_org_member_name_required"})
			return
		}
		target.Name = n
	}
	if req.RetailerRole != nil && !target.IsOwner {
		role := strings.ToUpper(strings.TrimSpace(*req.RetailerRole))
		if !allowedAssignableRoles[role] {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "retailer_org_member_invalid_role"})
			return
		}
		target.RetailerRole = role
	}
	if req.IsActive != nil && !target.IsOwner {
		target.IsActive = *req.IsActive
	}
	if req.Password != nil {
		pwd := strings.TrimSpace(*req.Password)
		if len(pwd) < 4 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "retailer_org_member_password_too_short"})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hash_password_failed"})
			return
		}
		target.PasswordHash = string(hash)
	}
	target.UpdatedAt = s.now()
	if err := s.updateOrgMember(r.Context(), target); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_member_failed"})
		return
	}
	resp, err := s.loadOrgMembersResponse(r.Context(), orgID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_retailer_org_members_failed"})
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func (s *Service) handleOrgMemberDeactivate(w http.ResponseWriter, r *http.Request, userID string) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStaffManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermStaffManage})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, okBody := readLimitedBody(w, r, 1024)
	if !okBody {
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	target, found, err := s.findRetailerUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_member_failed"})
		return
	}
	if !found || target.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "retailer_org_member_not_found"})
		return
	}
	if target.IsOwner {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "retailer_org_member_owner_locked"})
		return
	}
	// Prevent self-deactivate last admin? allow for now.
	target.IsActive = false
	target.UpdatedAt = s.now()
	if err := s.updateOrgMember(r.Context(), target); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "deactivate_failed"})
		return
	}
	resp, err := s.loadOrgMembersResponse(r.Context(), orgID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_retailer_org_members_failed"})
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func (s *Service) loadOrgMembersResponse(ctx context.Context, orgID string) (orgMembersResponse, error) {
	items, err := s.listOrgMembers(ctx, orgID)
	if err != nil {
		return orgMembersResponse{}, err
	}
	return orgMembersResponse{
		RetailerID: orgID,
		Items:      items,
		UpdatedAt:  s.now().UTC().Format(time.RFC3339Nano),
	}, nil
}

func (s *Service) listOrgMembers(ctx context.Context, retailerID string) ([]OrgMemberDTO, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		var users []RetailerUser
		if u, ok := s.ownerByRetailer[retailerID]; ok {
			users = append(users, u)
		}
		users = append(users, s.staffByRetailer[retailerID]...)
		s.mu.RUnlock()
		out := make([]OrgMemberDTO, 0, len(users))
		for _, u := range users {
			out = append(out, s.dtoFromUserWithLocations(ctx, u))
		}
		return out, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT UserId, RetailerId, Phone, Name, IFNULL(PasswordHash, ''), IFNULL(FirebaseUid, ''),
			RetailerRole, IsOwner, IsActive, CreatedAt, UpdatedAt
			FROM RetailerUsers@{FORCE_INDEX=Idx_RetailerUsers_ByRetailer}
			WHERE RetailerId = @rid
			ORDER BY IsOwner DESC, UpdatedAt DESC`,
		Params: map[string]any{"rid": retailerID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []OrgMemberDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		u, err := decodeRetailerUserRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, s.dtoFromUserWithLocations(ctx, u))
	}
	return out, nil
}

// ListActiveUserIDs returns active staff user ids for push fanout (includes owner).
func (s *Service) ListActiveUserIDs(ctx context.Context, retailerID string) ([]string, error) {
	items, err := s.listOrgMembers(ctx, retailerID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, m := range items {
		if m.IsActive {
			ids = append(ids, m.UserID)
		}
	}
	if len(ids) == 0 {
		// Legacy: no users table rows yet — push to org id.
		ids = append(ids, retailerID)
	}
	return ids, nil
}

func (s *Service) createOrgMember(ctx context.Context, u RetailerUser) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.staffByRetailer == nil {
			s.staffByRetailer = map[string][]RetailerUser{}
		}
		for _, existing := range s.staffByRetailer[u.RetailerID] {
			if existing.Phone == u.Phone && existing.IsActive {
				return errRetailerMemberPhoneExists
			}
		}
		if owner, ok := s.ownerByRetailer[u.RetailerID]; ok && owner.Phone == u.Phone {
			return errRetailerMemberPhoneExists
		}
		s.staffByRetailer[u.RetailerID] = append(s.staffByRetailer[u.RetailerID], u)
		return nil
	}
	// Phone uniqueness under retailer
	if existing, ok, err := s.findRetailerUserByRetailerPhone(ctx, u.RetailerID, u.Phone); err != nil {
		return err
	} else if ok && existing.IsActive {
		return errRetailerMemberPhoneExists
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row := map[string]any{
			"UserId":       u.UserID,
			"RetailerId":   u.RetailerID,
			"Phone":        u.Phone,
			"Name":         u.Name,
			"PasswordHash": u.PasswordHash,
			"RetailerRole": u.RetailerRole,
			"IsOwner":      false,
			"IsActive":     u.IsActive,
			"CreatedAt":    spanner.CommitTimestamp,
			"UpdatedAt":    spanner.CommitTimestamp,
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("RetailerUsers", row)}); err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, u.RetailerID, events.TopicMain, events.RetailerEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventRetailerStaffCreated, Timestamp: s.now().Format(time.RFC3339Nano)},
			RetailerID: u.RetailerID,
			Phone:      u.Phone,
			Name:       u.Name,
			SupplierID: s.supplierID,
			UserID:     u.UserID,
		}); err != nil {
			return err
		}
		return buf.Flush(txn)
	})
	if err != nil {
		if strings.Contains(err.Error(), "AlreadyExists") || strings.Contains(err.Error(), "Unique") {
			return errRetailerMemberPhoneExists
		}
		return fmt.Errorf("create org member: %w", err)
	}
	return nil
}

func (s *Service) updateOrgMember(ctx context.Context, u RetailerUser) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if u.IsOwner {
			s.ownerByRetailer[u.RetailerID] = u
			return nil
		}
		list := s.staffByRetailer[u.RetailerID]
		for i := range list {
			if list[i].UserID == u.UserID {
				list[i] = u
				s.staffByRetailer[u.RetailerID] = list
				return nil
			}
		}
		return errRetailerMemberNotFound
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Partial update via DML so we do not wipe CreatedAt / PasswordHash unintentionally.
		stmt := spanner.Statement{
			SQL: `UPDATE RetailerUsers SET
				Name = @name,
				RetailerRole = @role,
				IsActive = @active,
				UpdatedAt = PENDING_COMMIT_TIMESTAMP()`,
			Params: map[string]any{
				"name":   u.Name,
				"role":   u.RetailerRole,
				"active": u.IsActive,
			},
		}
		sql := stmt.SQL + ` WHERE UserId = @uid`
		params := map[string]any{
			"name":   u.Name,
			"role":   u.RetailerRole,
			"active": u.IsActive,
			"uid":    u.UserID,
		}
		if strings.TrimSpace(u.PasswordHash) != "" {
			sql = `UPDATE RetailerUsers SET
				Name = @name,
				RetailerRole = @role,
				IsActive = @active,
				PasswordHash = @ph,
				UpdatedAt = PENDING_COMMIT_TIMESTAMP()
				WHERE UserId = @uid`
			params["ph"] = u.PasswordHash
		}
		_, err := txn.Update(ctx, spanner.Statement{SQL: sql, Params: params})
		return err
	})
	return err
}

func dtoFromUser(u RetailerUser) OrgMemberDTO {
	return OrgMemberDTO{
		UserID:       u.UserID,
		RetailerID:   u.RetailerID,
		Name:         u.Name,
		Phone:        u.Phone,
		RetailerRole: u.RetailerRole,
		IsOwner:      u.IsOwner,
		IsActive:     u.IsActive,
		CreatedAt:    formatTimeOpt(u.CreatedAt),
		UpdatedAt:    formatTimeOpt(u.UpdatedAt),
	}
}

func (s *Service) dtoFromUserWithLocations(ctx context.Context, u RetailerUser) OrgMemberDTO {
	d := dtoFromUser(u)
	if ids, err := s.listUserLocationIDs(ctx, u.UserID); err == nil {
		d.LocationIDs = ids
	}
	return d
}

func formatTimeOpt(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// findRetailerUserByPhoneAny finds first active user by phone (v1: one org per phone).
func (s *Service) findRetailerUserByPhoneAny(ctx context.Context, phone string) (RetailerUser, bool, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return RetailerUser{}, false, nil
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, u := range s.ownerByRetailer {
			if u.Phone == phone && u.IsActive {
				return u, true, nil
			}
		}
		for _, list := range s.staffByRetailer {
			for _, u := range list {
				if u.Phone == phone && u.IsActive {
					return u, true, nil
				}
			}
		}
		return RetailerUser{}, false, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT UserId, RetailerId, Phone, Name, IFNULL(PasswordHash, ''), IFNULL(FirebaseUid, ''),
			RetailerRole, IsOwner, IsActive, CreatedAt, UpdatedAt
			FROM RetailerUsers@{FORCE_INDEX=Idx_RetailerUsers_ByPhone}
			WHERE Phone = @phone AND IsActive = TRUE
			LIMIT 1`,
		Params: map[string]any{"phone": phone},
	}
	return s.scanOneRetailerUser(ctx, stmt)
}
