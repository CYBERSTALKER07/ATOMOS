package order

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
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// ParentOrderChild is one child order in a parent rollup response.
type ParentOrderChild struct {
	OrderID    string `json:"order_id"`
	SupplierID string `json:"supplier_id"`
	Status     string `json:"status"`
	TotalMinor int64  `json:"total_minor"`
	Currency   string `json:"currency"`
}

// ParentOrderView is the retailer-facing parent rollup (on-read aggregation).
type ParentOrderView struct {
	ParentOrderID string             `json:"parent_order_id"`
	RetailerID    string             `json:"retailer_id"`
	Status        string             `json:"status"`
	Currency      string             `json:"currency"`
	TotalMinor    int64              `json:"total_minor"`
	ChildCount    int                `json:"child_count"`
	SagaState     string             `json:"saga_state,omitempty"`
	Children      []ParentOrderChild `json:"children"`
	CreatedAt     string             `json:"created_at,omitempty"`
	UpdatedAt     string             `json:"updated_at,omitempty"`
}

// GetParentOrder loads ParentOrders + children and rolls status on-read.
func (s *Service) GetParentOrder(ctx context.Context, retailerID, parentOrderID string) (ParentOrderView, error) {
	retailerID = strings.TrimSpace(retailerID)
	parentOrderID = strings.TrimSpace(parentOrderID)
	if retailerID == "" || parentOrderID == "" {
		return ParentOrderView{}, errors.New("parent_order_id and retailer_id required")
	}
	if s.spannerClient == nil {
		return ParentOrderView{}, fmt.Errorf("parent orders require spanner")
	}

	row, err := s.spannerClient.Single().ReadRow(ctx, "ParentOrders", spanner.Key{parentOrderID},
		[]string{"ParentOrderId", "RetailerId", "Status", "Currency", "TotalMinor", "ChildCount", "CreatedAt", "UpdatedAt"})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return ParentOrderView{}, ErrOrderNotFound
		}
		return ParentOrderView{}, fmt.Errorf("read parent order: %w", err)
	}
	var (
		view      ParentOrderView
		createdAt time.Time
		updatedAt time.Time
		childN    int64
	)
	if err := row.Columns(
		&view.ParentOrderID,
		&view.RetailerID,
		&view.Status,
		&view.Currency,
		&view.TotalMinor,
		&childN,
		&createdAt,
		&updatedAt,
	); err != nil {
		return ParentOrderView{}, fmt.Errorf("scan parent order: %w", err)
	}
	if view.RetailerID != retailerID {
		return ParentOrderView{}, ErrOrderForbidden
	}
	view.ChildCount = int(childN)
	view.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	view.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)

	children, err := s.listChildrenByParent(ctx, parentOrderID)
	if err != nil {
		return ParentOrderView{}, err
	}
	view.Children = children
	view.Status = rollupParentStatus(children)
	view.ChildCount = len(children)
	var total int64
	for _, c := range children {
		total += c.TotalMinor
		if c.Currency != "" {
			view.Currency = c.Currency
		}
	}
	view.TotalMinor = total

	// Best-effort persist rolled status for list surfaces.
	_ = s.updateParentOrderTotals(ctx, parentOrderID, view.Status, view.Currency, view.TotalMinor, view.ChildCount)
	return view, nil
}

func (s *Service) listChildrenByParent(ctx context.Context, parentOrderID string) ([]ParentOrderChild, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, SupplierId, Status, TotalMinor, Currency
		      FROM Orders@{FORCE_INDEX=Idx_Orders_ByParentOrder}
		      WHERE ParentOrderId = @pid
		      ORDER BY CreatedAt ASC`,
		Params: map[string]any{"pid": parentOrderID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := make([]ParentOrderChild, 0, 4)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Index may be absent until migration; retry without FORCE_INDEX.
			if strings.Contains(err.Error(), "Idx_Orders_ByParentOrder") || strings.Contains(err.Error(), "ParentOrderId") {
				return s.listChildrenByParentFallback(ctx, parentOrderID)
			}
			return nil, fmt.Errorf("list parent children: %w", err)
		}
		var c ParentOrderChild
		if err := row.Columns(&c.OrderID, &c.SupplierID, &c.Status, &c.TotalMinor, &c.Currency); err != nil {
			return nil, fmt.Errorf("scan parent child: %w", err)
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *Service) listChildrenByParentFallback(ctx context.Context, parentOrderID string) ([]ParentOrderChild, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, SupplierId, Status, TotalMinor, Currency
		      FROM Orders WHERE ParentOrderId = @pid ORDER BY CreatedAt ASC`,
		Params: map[string]any{"pid": parentOrderID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]ParentOrderChild, 0, 4)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list parent children fallback: %w", err)
		}
		var c ParentOrderChild
		if err := row.Columns(&c.OrderID, &c.SupplierID, &c.Status, &c.TotalMinor, &c.Currency); err != nil {
			return nil, fmt.Errorf("scan parent child: %w", err)
		}
		out = append(out, c)
	}
	return out, nil
}

func rollupParentStatus(children []ParentOrderChild) string {
	if len(children) == 0 {
		return parentStatusPending
	}
	cancelled := 0
	complete := 0
	for _, c := range children {
		switch Status(strings.TrimSpace(c.Status)) {
		case StatusCancelled:
			cancelled++
		case StatusCompleted:
			complete++
		}
	}
	if cancelled == len(children) {
		return parentStatusCancelled
	}
	if complete == len(children) {
		return parentStatusComplete
	}
	if cancelled > 0 || complete > 0 {
		return parentStatusPartial
	}
	return parentStatusPending
}

// HandleGetParentOrder serves GET /v1/retailer/parent-orders/{parentOrderID}.
func (s *Service) HandleGetParentOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleRetailer && claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	// B3 M-P0-4: ParentOrders.RetailerId is org id.
	retailerID := auth.ResolveRetailerOrgID(claims)
	if retailerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	parentID := strings.TrimSpace(chi.URLParam(r, "parentOrderID"))
	if parentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parent_order_id_required"})
		return
	}
	view, err := s.GetParentOrder(r.Context(), retailerID, parentID)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrderNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		case errors.Is(err, ErrOrderForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		default:
			s.log.Warn("get parent order failed", "parent_order_id", parentID, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		}
		return
	}
	body, _ := json.Marshal(view)
	writeJSONBytes(w, http.StatusOK, body)
}
