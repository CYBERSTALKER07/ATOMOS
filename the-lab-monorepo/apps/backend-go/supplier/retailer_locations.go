package supplier

import (
	"encoding/json"
	"log"
	"net/http"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Retailer Locations (Supplier Map Surface) ────────────────────────────────
//
// Returns lat/lng of all active retailers linked to the calling supplier,
// consumed by the admin portal's factory-warehouse network map.

type RetailerLocation struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ShopName string  `json:"shop_name"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

// HandleRetailerLocations returns GPS coordinates of all active retailers
// connected to the authenticated supplier.
// GET /v1/supplier/retailers/locations
func HandleRetailerLocations(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		stmt := spanner.Statement{
			SQL: `SELECT r.RetailerId, r.Name, COALESCE(r.ShopName, ''),
			             IFNULL(r.Latitude, 0), IFNULL(r.Longitude, 0)
			      FROM Retailers r
			      JOIN RetailerSuppliers rs ON r.RetailerId = rs.RetailerId
			      WHERE rs.SupplierId = @sid
			        AND r.Status = 'ACTIVE'
			        AND (r.Latitude IS NOT NULL OR r.Longitude IS NOT NULL)
			      ORDER BY r.Name ASC`,
			Params: map[string]interface{}{"sid": claims.ResolveSupplierID()},
		}

		iter := client.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		retailers := []RetailerLocation{}
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[RETAILER-LOCATIONS] query error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var rl RetailerLocation
			if err := row.Columns(&rl.ID, &rl.Name, &rl.ShopName, &rl.Lat, &rl.Lng); err != nil {
				log.Printf("[RETAILER-LOCATIONS] parse error: %v", err)
				continue
			}
			// Skip zero-coordinate entries
			if rl.Lat == 0 && rl.Lng == 0 {
				continue
			}
			retailers = append(retailers, rl)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"retailers": retailers,
			"total":     len(retailers),
		})
	}
}
