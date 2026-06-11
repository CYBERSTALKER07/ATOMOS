package order

import (
	"encoding/json"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/httppagination"
	"google.golang.org/api/iterator"
)

// ShopClosedAttemptDTO is the supplier-portal projection for an active escalation.
type ShopClosedAttemptDTO struct {
	AttemptID       string     `json:"attempt_id"`
	OrderID         string     `json:"order_id"`
	OriginalRouteID string     `json:"original_route_id,omitempty"`
	DriverID        string     `json:"driver_id"`
	RetailerID      string     `json:"retailer_id"`
	Resolution      string     `json:"resolution"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

// HandleListActiveShopClosedAttempts serves GET /v1/supplier/shop-closed/active.
func (s *Service) HandleListActiveShopClosedAttempts(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "shop_closed_unavailable"})
		return
	}

	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok {
		supplierID = s.supplierID
	}
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}

	ctx := r.Context()
	limit, offset := httppagination.ParseLimitOffset(r, 100, 500)
	stmt := spanner.Statement{
		SQL: `SELECT s.AttemptId, s.OrderId, IFNULL(o.RouteId, ''), s.DriverId, s.RetailerId,
		             IFNULL(s.Resolution, ''), s.ReportedAt, s.ResolvedAt
		      FROM ShopClosedAttempts s
		      JOIN Orders o ON s.OrderId = o.OrderId
		      WHERE o.SupplierId = @supplierId
		        AND s.ResolvedAt IS NULL
		        AND (s.EscalatedAt IS NOT NULL OR IFNULL(s.Resolution, '') IN ('WAITING', 'ESCALATED'))
		      ORDER BY s.ReportedAt DESC
		      LIMIT @limit OFFSET @offset`,
		Params: map[string]any{
			"supplierId": supplierID,
			"limit":      int64(limit),
			"offset":     int64(offset),
		},
	}

	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	active := make([]ShopClosedAttemptDTO, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			s.log.ErrorContext(ctx, "list shop-closed attempts failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_shop_closed_failed"})
			return
		}

		var dto ShopClosedAttemptDTO
		var resolved spanner.NullTime
		if err := row.Columns(
			&dto.AttemptID,
			&dto.OrderID,
			&dto.OriginalRouteID,
			&dto.DriverID,
			&dto.RetailerID,
			&dto.Resolution,
			&dto.CreatedAt,
			&resolved,
		); err != nil {
			s.log.ErrorContext(ctx, "parse shop-closed attempt row failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "parse_shop_closed_failed"})
			return
		}
		if resolved.Valid {
			t := resolved.Time
			dto.UpdatedAt = &t
		}
		active = append(active, dto)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":    active,
		"limit":   limit,
		"offset":  offset,
		"count":   len(active),
		"has_more": len(active) == limit,
	})
}
