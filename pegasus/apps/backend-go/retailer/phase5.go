package retailer

import (
	"encoding/json"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/hotspot"

	"github.com/go-chi/chi/v5"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type loyaltyTierResponse struct {
	RetailerID    string `json:"retailer_id"`
	TierKey       string `json:"tier_key"`
	TierLabel     string `json:"tier_label"`
	PointsBalance int64  `json:"points_balance"`
	CheckedAt     string `json:"checked_at"`
}

// HandleLoyaltyTier serves GET /v1/retailer/loyalty/tier (MVP stub).
func HandleLoyaltyTier(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		resp := loyaltyTierResponse{
			RetailerID:    claims.UserID,
			TierKey:       "starter",
			TierLabel:     "Starter",
			PointsBalance: 0,
			CheckedAt:     time.Now().UTC().Format(time.RFC3339),
		}

		stmt := spanner.Statement{
			SQL: `SELECT TierKey, TierLabel, PointsBalance
			      FROM RetailerLoyaltyTiers
			      WHERE RetailerId = @id
			      LIMIT 1`,
			Params: map[string]interface{}{"id": claims.UserID},
		}
		row, err := client.Single().Query(r.Context(), stmt).Next()
		if err == nil {
			_ = row.Columns(&resp.TierKey, &resp.TierLabel, &resp.PointsBalance)
		} else if err != iterator.Done {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

type submitRatingRequest struct {
	Stars   int64  `json:"stars"`
	Comment string `json:"comment,omitempty"`
}

// HandleSubmitOrderRating serves POST /v1/retailer/orders/{id}/rating.
func HandleSubmitOrderRating(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		orderID := chi.URLParam(r, "orderID")
		if orderID == "" {
			http.Error(w, `{"error":"order id required"}`, http.StatusBadRequest)
			return
		}

		var req submitRatingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Stars < 1 || req.Stars > 5 {
			http.Error(w, `{"error":"stars must be between 1 and 5"}`, http.StatusBadRequest)
			return
		}

		row, err := client.Single().ReadRow(r.Context(), "Orders", spanner.Key{orderID},
			[]string{"RetailerId", "SupplierId", "DriverId", "State"})
		if err != nil {
			http.Error(w, `{"error":"order not found"}`, http.StatusNotFound)
			return
		}
		var retailerID, supplierID, driverID, state string
		if err := row.Columns(&retailerID, &supplierID, &driverID, &state); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if retailerID != claims.UserID {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		if state != "COMPLETED" {
			http.Error(w, `{"error":"order must be completed before rating"}`, http.StatusConflict)
			return
		}

		ratingID := hotspot.NewOpaqueID()
		_, err = client.Apply(r.Context(), []*spanner.Mutation{
			spanner.InsertOrUpdate("RetailerRatings",
				[]string{"RatingId", "OrderId", "RetailerId", "SupplierId", "DriverId", "Stars", "Comment", "CreatedAt"},
				[]interface{}{ratingID, orderID, retailerID, supplierID, driverID, req.Stars, req.Comment, spanner.CommitTimestamp}),
		})
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"rating_id": ratingID,
			"order_id":  orderID,
			"status":    "RECORDED",
		})
	}
}
