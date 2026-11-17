package order

import (
	"context"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
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

	if s.repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "repository_unavailable"})
		return
	}

	siblings, err := s.repo.FindSiblingDriversForOrder(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "find_siblings_failed"})
		return
	}
	if len(siblings) <= 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no_sibling_drivers"})
		return
	}

	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "spanner_unavailable"})
		return
	}

	_, err = s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := outbox.NewSpannerTxnBuffer(txn)
		for _, sib := range siblings {
			if sib != claims.Subject {
				outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.OrderEvent{
					BaseEvent: events.BaseEvent{Type: events.EventReassignHandshakeCompleted},
					OrderID:   orderID,
					DriverID:  sib,
					Message:   "The other driver has started the reassigned order.",
				})
			}
		}
		return buf.Flush(ctx)
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "handshake_failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
