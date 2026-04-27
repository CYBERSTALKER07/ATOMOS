package order

import (
	"encoding/json"
	"net/http"
	"time"

	"backend-go/auth"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type ShopClosedAttemptDTO struct {
	AttemptID       string     `json:"attempt_id"`
	OrderID         string     `json:"order_id"`
	OriginalRouteID string     `json:"original_route_id"`
	DriverID        string     `json:"driver_id"`
	RetailerID      string     `json:"retailer_id"`
	Resolution      string     `json:"resolution"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

func (s *OrderService) HandleListActiveShopClosedAttempts(deps *ShopClosedDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		supplierID := claims.ResolveSupplierID()

		ctx := r.Context()
		stmt := spanner.Statement{
			SQL: `SELECT AttemptId, OrderId, OriginalRouteId, DriverId, RetailerId, Resolution, CreatedAt, UpdatedAt 
			      FROM ShopClosedAttempts 
			      WHERE Resolution IN ('ESCALATED', 'WAITING')`,
		}

		// In a real robust model we might join Orders to filter by SupplierId
		// but since we only have 60 lines max let's fetch all and filter or assume multi-tenant query.
		// Wait, ShopClosedAttempts lacks SupplierId but Order has it.
		// Let's join on Orders to filter by supplier.
		stmt = spanner.Statement{
			SQL: `SELECT s.AttemptId, s.OrderId, s.OriginalRouteId, s.DriverId, s.RetailerId, s.Resolution, s.CreatedAt, s.UpdatedAt
				  FROM ShopClosedAttempts s
				  JOIN Orders o ON s.OrderId = o.OrderId
				  WHERE s.Resolution IN ('ESCALATED', 'WAITING') AND o.SupplierId = @supplierId
				  ORDER BY s.CreatedAt DESC LIMIT 100`,
			Params: map[string]interface{}{"supplierId": supplierID},
		}

		iter := s.Client.Single().Query(ctx, stmt)
		defer iter.Stop()

		var active []ShopClosedAttemptDTO
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, `{"error":"failed to fetch attempts"}`, http.StatusInternalServerError)
				return
			}
			var dto ShopClosedAttemptDTO
			var u time.Time
			var uPtr *time.Time
			if err := row.Columns(&dto.AttemptID, &dto.OrderID, &dto.OriginalRouteID, &dto.DriverID, &dto.RetailerID, &dto.Resolution, &dto.CreatedAt, &u); err == nil {
				// updatedat might be null so handling that properly requires spanner.NullTime
			}
			// Let's fetch safely
			var attemptId, orderId, routeId, driverId, retailerId, res string
			var created time.Time
			var updated spanner.NullTime

			if err := row.Columns(&attemptId, &orderId, &routeId, &driverId, &retailerId, &res, &created, &updated); err != nil {
				http.Error(w, `{"error":"failed to parse attempt"}`, http.StatusInternalServerError)
				return
			}
			if updated.Valid {
				uPtr = &updated.Time
			}
			active = append(active, ShopClosedAttemptDTO{
				AttemptID:       attemptId,
				OrderID:         orderId,
				OriginalRouteID: routeId,
				DriverID:        driverId,
				RetailerID:      retailerId,
				Resolution:      res,
				CreatedAt:       created,
				UpdatedAt:       uPtr,
			})
		}
		if active == nil {
			active = []ShopClosedAttemptDTO{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": active})
	}
}
