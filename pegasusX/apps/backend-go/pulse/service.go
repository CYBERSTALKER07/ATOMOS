package pulse

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
)

// Event is one merged pulse timeline row.
type Event struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	OccurredAt  string `json:"occurred_at"`
	DeepLink    string `json:"deep_link,omitempty"`
	OrderID     string `json:"order_id,omitempty"`
	ManifestID  string `json:"manifest_id,omitempty"`
}

// Response is GET /v1/*/pulse.
type Response struct {
	Events      []Event `json:"events"`
	FetchedAt   string  `json:"fetched_at"`
	UnreadCount int64   `json:"unread_count,omitempty"`
}

// SupplierActivityLoader loads supplier-scoped order activity for pulse merge.
type SupplierActivityLoader interface {
	ListRecentSupplierOrders(ctx context.Context, supplierID string, limit int) ([]SupplierActivityOrder, error)
}

// SupplierActivityOrder is the minimal order row for activity projection.
type SupplierActivityOrder struct {
	OrderID    string
	ManifestID string
	Status     string
	UpdatedAt  time.Time
}

// Service merges notifications, order transitions, and role-specific activity.
type Service struct {
	notifications *notifications.Service
	spanner       *spanner.Client
	supplierAct   SupplierActivityLoader
	log           *slog.Logger
	now           func() time.Time
}

// Config wires pulse dependencies.
type Config struct {
	Notifications *notifications.Service
	Spanner       *spanner.Client
	SupplierAct   SupplierActivityLoader
	Log           *slog.Logger
}

// NewService creates a pulse aggregator.
func NewService(cfg Config) *Service {
	log := cfg.Log
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		notifications: cfg.Notifications,
		spanner:       cfg.Spanner,
		supplierAct:   cfg.SupplierAct,
		log:           log,
		now:           time.Now,
	}
}

// ListForRecipient merges inbox + transitions + optional supplier activity.
func (s *Service) ListForRecipient(ctx context.Context, recipientID, role, scopeID string, limit int) (Response, error) {
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	now := s.now().UTC()
	out := Response{
		Events:    []Event{},
		FetchedAt: now.Format(time.RFC3339Nano),
	}
	if s.notifications != nil && recipientID != "" {
		notifs, err := s.notifications.ListForRecipient(ctx, recipientID, limit, 0)
		if err != nil {
			return out, fmt.Errorf("pulse notifications: %w", err)
		}
		for _, n := range notifs {
			out.Events = append(out.Events, Event{
				ID:          "notif-" + n.NotificationID,
				Kind:        "notification",
				Title:       n.Title,
				Description: n.Body,
				OccurredAt:  n.CreatedAt.UTC().Format(time.RFC3339Nano),
				DeepLink:    n.DeepLink,
			})
		}
		if count, err := s.notifications.UnreadCount(ctx, recipientID); err == nil {
			out.UnreadCount = count
		}
	}
	if s.spanner != nil {
		transitions, err := s.listRecentTransitions(ctx, scopeID, role, limit)
		if err != nil {
			s.log.WarnContext(ctx, "pulse transitions read failed", "err", err)
		} else {
			out.Events = append(out.Events, transitions...)
		}
	}
	if strings.EqualFold(role, "ADMIN") && s.supplierAct != nil && scopeID != "" {
		orders, err := s.supplierAct.ListRecentSupplierOrders(ctx, scopeID, limit)
		if err != nil {
			s.log.WarnContext(ctx, "pulse supplier activity failed", "err", err)
		} else {
			out.Events = append(out.Events, buildSupplierActivityEvents(orders)...)
		}
	}
	sort.Slice(out.Events, func(i, j int) bool {
		return out.Events[i].OccurredAt > out.Events[j].OccurredAt
	})
	if len(out.Events) > limit {
		out.Events = out.Events[:limit]
	}
	return out, nil
}

func (s *Service) listRecentTransitions(ctx context.Context, scopeID, role string, limit int) ([]Event, error) {
	if s.spanner == nil {
		return nil, nil
	}
	since := s.now().UTC().Add(-7 * 24 * time.Hour)
	var stmt spanner.Statement
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "RETAILER":
		if scopeID == "" {
			return nil, nil
		}
		stmt = spanner.Statement{
			SQL: `SELECT t.TransitionId, t.OrderId, t.NewStatus, t.Reason, t.EventKind, t.CreatedAt
			      FROM OrderStatusTransitions t
			      JOIN Orders o ON o.OrderId = t.OrderId
			      WHERE o.RetailerId = @scope
			        AND t.CreatedAt >= @since
			      ORDER BY t.CreatedAt DESC
			      LIMIT @lim`,
			Params: map[string]any{"scope": scopeID, "since": since, "lim": int64(limit)},
		}
	case "WAREHOUSE_ADMIN", "WAREHOUSE":
		if scopeID == "" {
			return nil, nil
		}
		stmt = spanner.Statement{
			SQL: `SELECT t.TransitionId, t.OrderId, t.NewStatus, t.Reason, t.EventKind, t.CreatedAt
			      FROM OrderStatusTransitions t
			      JOIN Orders o ON o.OrderId = t.OrderId
			      WHERE o.WarehouseId = @scope
			        AND t.CreatedAt >= @since
			      ORDER BY t.CreatedAt DESC
			      LIMIT @lim`,
			Params: map[string]any{"scope": scopeID, "since": since, "lim": int64(limit)},
		}
	default:
		stmt = spanner.Statement{
			SQL: `SELECT TransitionId, OrderId, NewStatus, Reason, EventKind, CreatedAt
			      FROM OrderStatusTransitions
			      WHERE CreatedAt >= @since
			      ORDER BY CreatedAt DESC
			      LIMIT @lim`,
			Params: map[string]any{"since": since, "lim": int64(limit)},
		}
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]Event, 0, limit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var transitionID, orderID, status, reason, kind string
		var createdAt time.Time
		if err := row.Columns(&transitionID, &orderID, &status, &reason, &kind, &createdAt); err != nil {
			continue
		}
		desc := strings.TrimSpace(reason)
		if desc == "" {
			desc = fmt.Sprintf("Order %s → %s", orderID, status)
		}
		out = append(out, Event{
			ID:          "transition-" + transitionID,
			Kind:        "order_transition",
			Title:       humanizeTransitionKind(kind, status),
			Description: desc,
			OccurredAt:  createdAt.UTC().Format(time.RFC3339Nano),
			DeepLink:    "/orders/" + orderID,
			OrderID:     orderID,
		})
	}
	return out, nil
}

func humanizeTransitionKind(kind, status string) string {
	kind = strings.TrimSpace(kind)
	if kind != "" {
		return strings.ReplaceAll(kind, "_", " ")
	}
	return "Status: " + strings.TrimSpace(status)
}

func buildSupplierActivityEvents(orders []SupplierActivityOrder) []Event {
	out := make([]Event, 0, len(orders))
	for _, o := range orders {
		status := strings.TrimSpace(o.Status)
		if status == "" {
			status = "UPDATED"
		}
		out = append(out, Event{
			ID:          "order-" + o.OrderID,
			Kind:        "supplier_activity",
			Title:       "Order " + status,
			Description: fmt.Sprintf("Order %s · %s", o.OrderID, status),
			OccurredAt:  o.UpdatedAt.UTC().Format(time.RFC3339Nano),
			DeepLink:    "/orders/" + o.OrderID,
			OrderID:     o.OrderID,
			ManifestID:  o.ManifestID,
		})
	}
	return out
}

// MapOrderTransition converts an order audit row to a pulse event.
func MapOrderTransition(t order.OrderStatusTransition) Event {
	desc := strings.TrimSpace(t.Reason)
	if desc == "" {
		desc = fmt.Sprintf("Order %s → %s", t.OrderID, t.NewStatus)
	}
	return Event{
		ID:          "transition-" + t.TransitionID,
		Kind:        "order_transition",
		Title:       humanizeTransitionKind(t.EventKind, t.NewStatus),
		Description: desc,
		OccurredAt:  t.CreatedAt.UTC().Format(time.RFC3339Nano),
		DeepLink:    "/orders/" + t.OrderID,
		OrderID:     t.OrderID,
	}
}
