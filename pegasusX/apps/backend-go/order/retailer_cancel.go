package order

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandleRetailerCancel serves POST /v1/order/cancel for retailer-scoped cancellation.
func (s *Service) HandleRetailerCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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

	var req struct {
		OrderID    string `json:"order_id"`
		RetailerID string `json:"retailer_id"`
		Reason     string `json:"reason,omitempty"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	// B3 M-P0-4: cancel ownership is org-scoped; body retailer_id must match org.
	orgID := auth.ResolveRetailerOrgID(claims)
	if orgID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	retailerID := strings.TrimSpace(req.RetailerID)
	if retailerID == "" {
		retailerID = orgID
	}
	if retailerID != orgID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	resp, err := s.UpdateStatus(r.Context(), claims, orderID, UpdateStatusRequest{
		Status: string(StatusCancelled),
		Reason: strings.TrimSpace(req.Reason),
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrOrderNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		case errors.Is(err, ErrOrderForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		case errors.Is(err, ErrOrderCancelLocked):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "cancel_locked"})
		case errors.Is(err, ErrInvalidStatusTransition):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_status_transition"})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}

	out := map[string]any{
		"status":          "cancelled",
		"order_id":        resp.OrderID,
		"previous_status": resp.PreviousStatus,
		"order_status":    resp.Status,
		"version":         resp.Version,
		"updated_at":      resp.UpdatedAt,
		"event_type":      resp.EventType,
	}
	respBytes, _ := json.Marshal(out)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}
