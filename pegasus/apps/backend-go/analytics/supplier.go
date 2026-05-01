package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"backend-go/auth"
	"backend-go/proximity"
)

// SkuVelocity represents the aggregated sales data for a specific product
type SkuVelocity struct {
	SkuId        string `json:"sku_id"`
	TotalPallets int64  `json:"total_pallets"`
	GrossVolume  int64  `json:"gross_volume"`
}

// HandleGetVelocity returns the real-time sales velocity for the authenticated supplier
func HandleGetVelocity(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract SupplierId from JWT
		// Assuming we retrieve it from our custom auth claims
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		var supplierId string
		if ok && claims != nil {
			supplierId = claims.ResolveSupplierID()
		} else {
			// Fallback check if it was set simply as "user_id"
			val := r.Context().Value("user_id")
			if val != nil {
				supplierId = val.(string)
			} else {
				http.Error(w, "Unauthorized: Context missing", http.StatusUnauthorized)
				return
			}
		}

		// Strictly indexed Spanner query — join through SupplierProducts for supplier ownership.
		stmt := spanner.Statement{
			SQL: `SELECT 
					oli.SkuId, 
					SUM(oli.Quantity) as TotalPallets, 
					SUM(oli.UnitPrice) as GrossVolume
				  FROM OrderLineItems oli
				  JOIN SupplierProducts sp ON oli.SkuId = sp.SkuId
				  WHERE sp.SupplierId = @supplierId 
				  AND oli.Status = 'DELIVERED'
				  GROUP BY oli.SkuId`,
			Params: map[string]interface{}{
				"supplierId": supplierId,
			},
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		iter := getReadClient(r.Context(), client, readRouter, nil).Single().WithTimestampBound(spanner.ExactStaleness(10*time.Second)).Query(ctx, stmt)
		defer iter.Stop()

		var velocities []SkuVelocity
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Analytics computation fault", http.StatusInternalServerError)
				return
			}

			var v SkuVelocity
			var skuId spanner.NullString
			var totalPallets spanner.NullInt64
			var grossVolume spanner.NullInt64

			if err := row.Columns(&skuId, &totalPallets, &grossVolume); err != nil {
				http.Error(w, "Data extraction fault", http.StatusInternalServerError)
				return
			}

			if skuId.Valid {
				v.SkuId = skuId.StringVal
			}
			if totalPallets.Valid {
				v.TotalPallets = totalPallets.Int64
			}
			if grossVolume.Valid {
				v.GrossVolume = grossVolume.Int64
			}

			velocities = append(velocities, v)
		}

		if velocities == nil {
			velocities = []SkuVelocity{} // Ensure we don't return null
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"data":      velocities,
		})
	}
}
