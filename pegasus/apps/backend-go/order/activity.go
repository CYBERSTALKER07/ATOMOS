package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	apperrors "backend-go/errors"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// OrderActivityEntry is one timeline row for GET /v1/orders/{id}/activity.
type OrderActivityEntry struct {
	ActivityID string `json:"activity_id"`
	OrderID    string `json:"order_id"`
	ActorID    string `json:"actor_id"`
	ActorRole  string `json:"actor_role"`
	EventType  string `json:"event_type"`
	Summary    string `json:"summary,omitempty"`
	Metadata   string `json:"metadata,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// HandleOrderActivity serves GET /v1/orders/{orderID}/activity.
func (s *OrderService) HandleOrderActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apperrors.MethodNotAllowed(w, r)
		return
	}

	claims, _ := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if claims == nil {
		apperrors.Unauthorized(w, r, "authentication required")
		return
	}

	orderID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/orders/"), "/activity")
	orderID = strings.Trim(orderID, "/")
	if orderID == "" {
		apperrors.NotFound(w, r, "order activity endpoint not found")
		return
	}

	scope, err := legacyOrderScopeForClaims(claims)
	if err != nil {
		apperrors.Forbidden(w, r, "order is outside caller scope")
		return
	}
	if _, _, err := s.fetchLegacyOrderHeader(r.Context(), orderID, scope); err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			apperrors.NotFound(w, r, fmt.Sprintf("order %s not found", orderID))
			return
		}
		slog.ErrorContext(r.Context(), "order_activity.scope_failed",
			"trace_id", telemetry.TraceIDFromContext(r.Context()),
			"order_id", orderID,
			"err", err)
		apperrors.InternalError(w, r, "failed to load order activity")
		return
	}

	entries, err := s.fetchOrderActivity(r.Context(), orderID)
	if err != nil {
		slog.ErrorContext(r.Context(), "order_activity.fetch_failed",
			"trace_id", telemetry.TraceIDFromContext(r.Context()),
			"order_id", orderID,
			"err", err)
		apperrors.InternalError(w, r, "failed to load order activity")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string][]OrderActivityEntry{"activity": entries})
}

func (s *OrderService) fetchOrderActivity(ctx context.Context, orderID string) ([]OrderActivityEntry, error) {
	entries, err := s.queryOrderActivityTable(ctx, orderID, "OrderActivityEvents")
	if err != nil {
		return nil, err
	}
	if len(entries) > 0 {
		return entries, nil
	}
	return s.queryOrderActivityTable(ctx, orderID, "OrderEvents")
}

func (s *OrderService) queryOrderActivityTable(ctx context.Context, orderID, table string) ([]OrderActivityEntry, error) {
	var sql string
	switch table {
	case "OrderActivityEvents":
		sql = `SELECT ActivityId, OrderId, ActorId, ActorRole, EventType,
			COALESCE(Summary, ''), COALESCE(Metadata, ''), CreatedAt
			FROM OrderActivityEvents
			WHERE OrderId = @orderID
			ORDER BY CreatedAt ASC`
	default:
		sql = `SELECT EventId, OrderId, ActorId, ActorRole, EventType,
			'' AS Summary, COALESCE(Metadata, ''), CreatedAt
			FROM OrderEvents@{FORCE_INDEX=Idx_OrderEvents_ByOrder}
			WHERE OrderId = @orderID
			ORDER BY CreatedAt ASC`
	}
	stmt := spanner.Statement{
		SQL:    sql,
		Params: map[string]interface{}{"orderID": orderID},
	}
	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var entries []OrderActivityEntry
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var (
			eventID, oid, actorID, actorRole, eventType, summary, metadata string
			createdAt                                                      time.Time
		)
		if err := row.Columns(&eventID, &oid, &actorID, &actorRole, &eventType, &summary, &metadata, &createdAt); err != nil {
			return nil, err
		}
		entries = append(entries, OrderActivityEntry{
			ActivityID: eventID,
			OrderID:    oid,
			ActorID:    actorID,
			ActorRole:  actorRole,
			EventType:  eventType,
			Summary:    summary,
			Metadata:   metadata,
			CreatedAt:  createdAt.Format(time.RFC3339),
		})
	}
	if entries == nil {
		entries = []OrderActivityEntry{}
	}
	return entries, nil
}
