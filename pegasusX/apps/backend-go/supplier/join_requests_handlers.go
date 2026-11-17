package supplier

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/innlookup"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type joinRequestDecisionBody struct {
	Reason string `json:"reason"`
}

// HandleListJoinRequests serves GET /v1/supplier/join-requests for supplier admins.
func (s *Service) HandleListJoinRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	innSvc := s.INNLookup()
	if innSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "join_requests_unavailable"})
		return
	}
	sid := strings.TrimSpace(s.scopedSupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id_required"})
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	limit := parseLimitParam(r.URL.Query().Get("limit"), 100)
	items, err := innSvc.ListJoinRequests(r.Context(), innlookup.TenantTypeSupplier, sid, status, limit)
	if err != nil {
		if errors.Is(err, innlookup.ErrRegistryUnavailable) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_join_requests_failed"})
		return
	}
	if items == nil {
		items = []innlookup.JoinRequest{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"supplier_id": sid,
		"requests":    items,
	})
}

// HandleApproveJoinRequest serves POST /v1/supplier/join-requests/{requestId}/approve.
func (s *Service) HandleApproveJoinRequest(w http.ResponseWriter, r *http.Request) {
	s.handleDecideJoinRequest(w, r, innlookup.JoinStatusApproved)
}

// HandleRejectJoinRequest serves POST /v1/supplier/join-requests/{requestId}/reject.
func (s *Service) HandleRejectJoinRequest(w http.ResponseWriter, r *http.Request) {
	s.handleDecideJoinRequest(w, r, innlookup.JoinStatusRejected)
}

func (s *Service) handleDecideJoinRequest(w http.ResponseWriter, r *http.Request, decision string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin || (claims.SupplierRole != "" && claims.SupplierRole != auth.RoleAdmin) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin_role_required"})
		return
	}
	innSvc := s.INNLookup()
	if innSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "join_requests_unavailable"})
		return
	}
	sid := strings.TrimSpace(s.scopedSupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id_required"})
		return
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "requestId"))
	if requestID == "" {
		requestID = strings.TrimSpace(chi.URLParam(r, "requestID"))
	}
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request_id_required"})
		return
	}

	var body joinRequestDecisionBody
	raw, err := io.ReadAll(io.LimitReader(r.Body, 16*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_failed"})
		return
	}
	defer r.Body.Close()
	if len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}

	reason := strings.TrimSpace(body.Reason)
	decidedBy := strings.TrimSpace(claims.Subject)
	if decidedBy == "" {
		decidedBy = string(claims.Role)
	}
	now := s.serviceNow()
	decided, err := innSvc.DecideJoinRequest(
		r.Context(),
		innlookup.TenantTypeSupplier,
		sid,
		requestID,
		decision,
		decidedBy,
		reason,
		func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateSupplier, sid, events.TopicMain, map[string]any{
				"type":               "JOIN_REQUEST_RESOLVED",
				"timestamp":          now.Format(time.RFC3339Nano),
				"supplier_id":        sid,
				"target_tenant_type": innlookup.TenantTypeSupplier,
				"target_tenant_id":   sid,
				"request_id":         requestID,
				"status":             decision,
				"decided_by":         decidedBy,
				"decision_reason":    reason,
			})
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, innlookup.ErrJoinRequestNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		case errors.Is(err, innlookup.ErrJoinRequestDecided):
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		case errors.Is(err, innlookup.ErrRegistryUnavailable):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "decide_join_request_failed"})
		}
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), supplierCacheKey(sid))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"supplier_id": sid,
		"request":     decided,
	})
}
