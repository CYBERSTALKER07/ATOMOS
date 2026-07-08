package order

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandleReassignHandshake serves POST /v1/fleet/orders/{orderID}/reassign-handshake.
// It is used in the partial reassignment flow where both drivers receive the order,
// and one driver presses "Start" to notify the other driver.
func (s *Service) HandleReassignHandshake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	if s.driverHub != nil {
		siblings, err := s.repo.FindSiblingDriversForOrder(r.Context(), orderID)
		if err == nil && len(siblings) > 1 {
			for _, sib := range siblings {
				if sib != claims.Subject {
					payload := map[string]any{
						"type":     "REASSIGN_HANDSHAKE_COMPLETED",
						"order_id": orderID,
						"message":  "The other driver has started the reassigned order.",
					}
					b, _ := json.Marshal(payload)
					go s.driverHub.Broadcast(context.Background(), "driver:"+sib, b)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
