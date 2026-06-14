package supplier

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

var (
	errOrgMemberNotFound      = errors.New("supplier_org_member_not_found")
	errOrgMemberFounderLocked = errors.New("supplier_org_member_founder_locked")
)

type orgMemberUpdateRequest struct {
	Name                *string `json:"name,omitempty"`
	SupplierRole        *string `json:"supplier_role,omitempty"`
	AssignedWarehouseID *string `json:"assigned_warehouse_id,omitempty"`
	AssignedFactoryID   *string `json:"assigned_factory_id,omitempty"`
	IsActive            *bool   `json:"is_active,omitempty"`
}

// UpdateOrgMemberPatch is the repository input for org member mutations.
type UpdateOrgMemberPatch struct {
	Name                *string
	SupplierRole        *auth.Role
	AssignedWarehouseID *string
	AssignedFactoryID   *string
	IsActive            *bool
}

// HandleOrgMemberByID serves PATCH/PUT/DELETE /v1/supplier/org/members/{userID}.
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

func (s *Service) handleOrgMemberPatch(w http.ResponseWriter, r *http.Request, userID string) {
	body, ok := readMutationBody(w, r, 16*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

	var req orgMemberUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	sid := s.scopedSupplierID(r)
	if err := s.guardOrgMemberMutation(sid, userID, req); err != nil {
		writeOrgMemberMutationError(w, err)
		return
	}

	patch, err := s.buildOrgMemberPatch(r, req, sid)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	if patch.isEmpty() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no_fields_to_update"})
		return
	}

	now := s.now().UTC()
	if err := s.repo.UpdateOrgMember(r.Context(), sid, userID, patch, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSupplier, sid, events.TopicMain, events.SupplierEvent{
			BaseEvent:           events.BaseEvent{Type: events.EventSupplierMemberAdded, Timestamp: now.Format(time.RFC3339Nano)},
			SupplierID:          sid,
			UserID:              userID,
			SupplierRole:        roleString(patch.SupplierRole),
			AssignedWarehouseID: stringPtrValue(patch.AssignedWarehouseID),
			AssignedFactoryID:   stringPtrValue(patch.AssignedFactoryID),
			Action:              "ORG_MEMBER_UPDATED",
		})
	}); err != nil {
		writeOrgMemberMutationError(w, err)
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(sid))
	}
	resp, err := s.loadOrgMembersResponse(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_org_members_failed"})
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func (s *Service) handleOrgMemberDeactivate(w http.ResponseWriter, r *http.Request, userID string) {
	body, ok := readMutationBody(w, r, 1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

	sid := s.scopedSupplierID(r)
	if err := s.guardOrgMemberMutation(sid, userID, orgMemberUpdateRequest{IsActive: boolPtr(false)}); err != nil {
		writeOrgMemberMutationError(w, err)
		return
	}

	now := s.now().UTC()
	isActive := false
	if err := s.repo.UpdateOrgMember(r.Context(), sid, userID, UpdateOrgMemberPatch{IsActive: &isActive}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateSupplier, sid, events.TopicMain, events.SupplierEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventSupplierMemberAdded, Timestamp: now.Format(time.RFC3339Nano)},
			SupplierID: sid,
			UserID:     userID,
			Action:     "ORG_MEMBER_DEACTIVATED",
		})
	}); err != nil {
		writeOrgMemberMutationError(w, err)
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(sid))
	}
	resp, err := s.loadOrgMembersResponse(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_org_members_failed"})
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func (s *Service) guardOrgMemberMutation(supplierID, targetUserID string, req orgMemberUpdateRequest) error {
	if targetUserID == supplierID {
		if req.SupplierRole != nil && strings.ToUpper(strings.TrimSpace(*req.SupplierRole)) != string(auth.RoleAdmin) {
			return errOrgMemberFounderLocked
		}
		if req.IsActive != nil && !*req.IsActive {
			return errOrgMemberFounderLocked
		}
	}
	return nil
}

func (s *Service) buildOrgMemberPatch(r *http.Request, req orgMemberUpdateRequest, supplierID string) (UpdateOrgMemberPatch, error) {
	patch := UpdateOrgMemberPatch{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if len(name) < 2 {
			return UpdateOrgMemberPatch{}, fmt.Errorf("supplier_org_member_name_required")
		}
		patch.Name = &name
	}
	if req.SupplierRole != nil {
		role := auth.Role(strings.ToUpper(strings.TrimSpace(*req.SupplierRole)))
		topology, err := s.repo.GetTopology(r.Context(), supplierID)
		if err != nil {
			return UpdateOrgMemberPatch{}, fmt.Errorf("load_supplier_topology_failed")
		}
		warehouseID := ""
		factoryID := ""
		if req.AssignedWarehouseID != nil {
			warehouseID = strings.TrimSpace(*req.AssignedWarehouseID)
		}
		if req.AssignedFactoryID != nil {
			factoryID = strings.TrimSpace(*req.AssignedFactoryID)
		}
		if err := validateOrgRoleAssignment(role, warehouseID, factoryID, newTopologyLookup(topology)); err != nil {
			return UpdateOrgMemberPatch{}, err
		}
		patch.SupplierRole = &role
		if role == auth.RoleAdmin {
			empty := ""
			patch.AssignedWarehouseID = &empty
			patch.AssignedFactoryID = &empty
		}
		if role == auth.RoleWarehouseAdmin {
			empty := ""
			patch.AssignedFactoryID = &empty
			if warehouseID != "" {
				patch.AssignedWarehouseID = &warehouseID
			}
		}
		if role == auth.RoleFactoryAdmin {
			empty := ""
			patch.AssignedWarehouseID = &empty
			if factoryID != "" {
				patch.AssignedFactoryID = &factoryID
			}
		}
		if role == auth.RolePayload {
			if warehouseID != "" {
				patch.AssignedWarehouseID = &warehouseID
			}
			if factoryID != "" {
				patch.AssignedFactoryID = &factoryID
			}
		}
	} else {
		if req.AssignedWarehouseID != nil {
			warehouseID := strings.TrimSpace(*req.AssignedWarehouseID)
			patch.AssignedWarehouseID = &warehouseID
		}
		if req.AssignedFactoryID != nil {
			factoryID := strings.TrimSpace(*req.AssignedFactoryID)
			patch.AssignedFactoryID = &factoryID
		}
	}
	if req.IsActive != nil {
		patch.IsActive = req.IsActive
	}
	return patch, nil
}

func (p UpdateOrgMemberPatch) isEmpty() bool {
	return p.Name == nil && p.SupplierRole == nil && p.AssignedWarehouseID == nil &&
		p.AssignedFactoryID == nil && p.IsActive == nil
}

func writeOrgMemberMutationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errOrgMemberNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "supplier_org_member_not_found"})
	case errors.Is(err, errOrgMemberFounderLocked):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_org_member_founder_locked"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_supplier_org_member_failed"})
	}
}

func roleString(role *auth.Role) string {
	if role == nil {
		return ""
	}
	return string(*role)
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolPtr(value bool) *bool {
	return &value
}
