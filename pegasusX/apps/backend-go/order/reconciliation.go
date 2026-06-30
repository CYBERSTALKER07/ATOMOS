package order

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type reconcileResolveRequest struct {
	OrderID string `json:"order_id"`
	Action  string `json:"action"` // "COMPLETE" or "CANCEL"
}

// HandleListReconciliationOrders is GET /v1/supplier/reconciliation.
func (s *Service) HandleListReconciliationOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reconciliation_unavailable"})
		return
	}

	supplierID := strings.TrimSpace(claims.SupplierID)
	// If supplier is effectively the single tenant, supplierID is likely used.

	orders, err := s.repo.ListOrdersByStatus(r.Context(), supplierID, string(StatusReconciliationRequired), 50)
	if err != nil {
		s.log.ErrorContext(r.Context(), "failed to list reconciliation orders", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

// HandleResolveReconciliation is POST /v1/supplier/reconciliation/resolve.
func (s *Service) HandleResolveReconciliation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reconciliation_unavailable"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
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

	var req reconcileResolveRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.Action = strings.TrimSpace(req.Action)
	if req.OrderID == "" || req.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id and action required"})
		return
	}

	var nextStatus Status
	switch req.Action {
	case "COMPLETE":
		nextStatus = StatusCompleted
	case "CANCEL":
		nextStatus = StatusCancelled
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid action"})
		return
	}

	actorID := claims.Subject
	if actorID == "" {
		actorID = "system"
	}

	ctx := r.Context()
	orderRecord, ok, err := s.repo.GetOrder(ctx, req.OrderID)
	if err != nil || !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	
	if claims.Role == auth.RoleAdmin && orderRecord.SupplierID != claims.SupplierID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	if orderRecord.Status != StatusReconciliationRequired {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "order_not_in_reconciliation"})
		return
	}

	updateClaims := auth.Claims{Role: auth.RoleAdmin, Subject: actorID}
	updateReq := UpdateStatusRequest{Status: string(nextStatus), Reason: "reconciliation_resolved"}
	_, err = s.UpdateStatus(ctx, updateClaims, req.OrderID, updateReq)
	if err != nil {
		s.log.ErrorContext(ctx, "reconciliation resolve failed", "order_id", req.OrderID, "err", err)
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	s.invalidateOrderCache(ctx, req.OrderID)
	respBytes, _ := json.Marshal(map[string]string{"status": string(nextStatus)})
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}
