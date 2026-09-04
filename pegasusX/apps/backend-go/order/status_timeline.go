package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
)

// OrderStatusTransition is one immutable audit row for order lifecycle changes.
type OrderStatusTransition struct {
	TransitionID   string         `json:"transition_id"`
	OrderID          string         `json:"order_id"`
	PreviousStatus   string         `json:"previous_status,omitempty"`
	NewStatus        string         `json:"new_status"`
	Reason           string         `json:"reason,omitempty"`
	ActorRole        string         `json:"actor_role,omitempty"`
	ActorID          string         `json:"actor_id,omitempty"`
	EventKind        string         `json:"event_kind,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}



// DefaultTransitionEventKind gives a default event kind for a status.
func DefaultTransitionEventKind(status Status) string {
	switch status {
	case StatusDelayed:
		return "DELAY"
	case StatusCancelled:
		return "CANCEL"
	case StatusPending:
		return "PROMOTE"
	case StatusCompleted:
		return "COMPLETE"
	default:
		return "STATUS_CHANGE"
	}
}

// ListOrderTimeline returns status transitions newest-first.
func (s *Service) ListOrderTimeline(ctx context.Context, orderID string, limit int) ([]OrderStatusTransition, error) {
	if s.spannerClient == nil {
		return nil, errors.New("timeline_unavailable")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	stmt := spanner.Statement{
		SQL: `SELECT TransitionId, OrderId, PreviousStatus, NewStatus, Reason, ActorRole, ActorId, EventKind, MetadataJson, CreatedAt
		      FROM OrderStatusTransitions
		      WHERE OrderId = @oid
		      ORDER BY CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"oid": orderID, "lim": limit},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]OrderStatusTransition, 0, limit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var (
			transitionID, oid, prev, next, reason, actorRole, actorID, kind string
			metaRaw                                                       []byte
			createdAt                                                     time.Time
		)
		if err := row.Columns(&transitionID, &oid, &prev, &next, &reason, &actorRole, &actorID, &kind, &metaRaw, &createdAt); err != nil {
			continue
		}
		entry := OrderStatusTransition{
			TransitionID: transitionID,
			OrderID:      oid,
			PreviousStatus: prev,
			NewStatus:    next,
			Reason:       reason,
			ActorRole:    actorRole,
			ActorID:      actorID,
			EventKind:    kind,
			CreatedAt:    createdAt,
		}
		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &entry.Metadata)
		}
		out = append(out, entry)
	}
	return out, nil
}

// HandleGetOrderTimeline serves GET /v1/order/{orderID}/timeline.
func (s *Service) HandleGetOrderTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	o, found, err := s.loadOrderForRequest(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if err := assertTimelineAccess(claims, o); err != nil {
		if errors.Is(err, ErrOrderForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	items, err := s.ListOrderTimeline(r.Context(), orderID, 100)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "timeline_unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"order_id": orderID,
		"items":    items,
	})
}

func assertTimelineAccess(claims auth.Claims, o Order) error {
	switch claims.Role {
	case auth.RoleAdmin:
		if claims.SupplierID == "" || claims.SupplierID == o.SupplierID {
			return nil
		}
	case auth.RoleRetailer:
		retID := auth.ResolveRetailerOrgID(claims)
		if retID != "" && retID == o.RetailerID {
			return nil
		}
	case auth.RoleWarehouseAdmin, auth.RoleWarehouse:
		wh := claims.HomeNodeID
		if wh != "" && wh == o.WarehouseID {
			return nil
		}
	}
	return ErrOrderForbidden
}


// delayTransitionMetadata builds audit metadata for warehouse delay actions.
func delayTransitionMetadata(proposedDelivery *time.Time, reason string) map[string]any {
	meta := map[string]any{"reason": strings.TrimSpace(reason)}
	if proposedDelivery != nil {
		meta["proposed_delivery_date"] = proposedDelivery.UTC().Format(time.RFC3339Nano)
	}
	return meta
}

func parseDelayMetadataFromRequest(proposedDate string) map[string]any {
	proposedDate = strings.TrimSpace(proposedDate)
	if proposedDate == "" {
		return nil
	}
	t, err := parseOptionalRFC3339(proposedDate)
	if err != nil || t == nil {
		// Accept YYYY-MM-DD calendar input from portals.
		if len(proposedDate) >= 10 {
			meta := map[string]any{"proposed_delivery_date": proposedDate[:10]}
			return meta
		}
		return nil
	}
	return map[string]any{"proposed_delivery_date": t.UTC().Format(time.RFC3339Nano)}
}
