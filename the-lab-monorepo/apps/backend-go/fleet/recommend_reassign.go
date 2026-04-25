package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/proximity"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ═══════════════════════════════════════════════════════════════════════════════
// RECOMMEND REASSIGN — GPS-AWARE TRUCK RECOMMENDATION FOR PAYLOADER RE-DISPATCHING
// Used by Payload Terminal and Admin Portal to find the best truck to move an
// order to, based on proximity, capacity, and route compatibility.
// ═══════════════════════════════════════════════════════════════════════════════

type RecommendReassignRequest struct {
	OrderID string `json:"order_id"`
}

type TruckRecommendation struct {
	DriverID       string  `json:"driver_id"`
	DriverName     string  `json:"driver_name"`
	VehicleID      string  `json:"vehicle_id"`
	VehicleClass   string  `json:"vehicle_class"`
	LicensePlate   string  `json:"license_plate"`
	MaxVolumeVU    float64 `json:"max_volume_vu"`
	UsedVolumeVU   float64 `json:"used_volume_vu"`
	FreeVolumeVU   float64 `json:"free_volume_vu"`
	DistanceKm     float64 `json:"distance_km"`
	OrderCount     int     `json:"order_count"`
	TruckStatus    string  `json:"truck_status"`
	Score          float64 `json:"score"`
	Recommendation string  `json:"recommendation"`
}

type RecommendReassignResponse struct {
	OrderID         string                `json:"order_id"`
	RetailerName    string                `json:"retailer_name"`
	OrderVolumeVU   float64               `json:"order_volume_vu"`
	CurrentDriver   string                `json:"current_driver,omitempty"`
	Recommendations []TruckRecommendation `json:"recommendations"`
}

// HandleRecommendReassign handles POST /v1/payloader/recommend-reassign
// Returns ranked truck recommendations for reassigning an order.
func HandleRecommendReassign(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req RecommendReassignRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// 1. Fetch the order details (location, volume, current assignment)
		orderRow, err := client.Single().ReadRow(ctx, "Orders",
			spanner.Key{req.OrderID},
			[]string{"RetailerId", "DriverId", "Amount", "State"})
		if err != nil {
			http.Error(w, `{"error":"order not found"}`, http.StatusNotFound)
			return
		}

		var retailerID, currentDriverID spanner.NullString
		var amount spanner.NullInt64
		var state string
		if err := orderRow.Columns(&retailerID, &currentDriverID, &amount, &state); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Only allow reassignment of PENDING/LOADED/DISPATCHED orders
		if state != "PENDING" && state != "LOADED" && state != "DISPATCHED" {
			http.Error(w, `{"error":"order state does not allow reassignment: `+state+`"}`, http.StatusConflict)
			return
		}

		// Fetch retailer location + name
		var retailerLat, retailerLng float64
		var retailerName string
		if retailerID.Valid {
			retRow, err := client.Single().ReadRow(ctx, "Retailers",
				spanner.Key{retailerID.StringVal},
				[]string{"Name", "ShopLocation"})
			if err == nil {
				var name spanner.NullString
				var loc spanner.NullString
				if err := retRow.Columns(&name, &loc); err == nil {
					retailerName = name.StringVal
					retailerLat, retailerLng = parseWKT(loc.StringVal)
				}
			}
		}

		readClient := proximity.ReadClientForRetailer(client, readRouter, retailerLat, retailerLng)

		// Fetch order volume (sum of line items × pallet footprint)
		volStmt := spanner.Statement{
			SQL: `SELECT COALESCE(SUM(li.Quantity * COALESCE(sp.VolumetricUnit, sp.PalletFootprint, 1.0)), 0)
			      FROM OrderLineItems li
			      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
			      WHERE li.OrderId = @orderId`,
			Params: map[string]interface{}{"orderId": req.OrderID},
		}
		volIter := readClient.Single().Query(ctx, volStmt)
		defer volIter.Stop()
		var orderVolumeVU float64
		if volRow, err := volIter.Next(); err == nil {
			var vol spanner.NullFloat64
			if volRow.Columns(&vol) == nil {
				orderVolumeVU = vol.Float64
			}
		}

		// Resolve which supplier owns this order (via the claims or the retailer's supplier)
		supplierID := claims.ResolveSupplierID()
		// For PAYLOADER role, resolve supplier from the order's current driver assignment
		if claims.Role == "PAYLOADER" && currentDriverID.Valid {
			driverRow, err := readClient.Single().ReadRow(ctx, "Drivers",
				spanner.Key{currentDriverID.StringVal}, []string{"SupplierId"})
			if err == nil {
				var sid spanner.NullString
				if driverRow.Columns(&sid) == nil && sid.Valid {
					supplierID = sid.StringVal
				}
			}
		}

		// 2. Fetch all available trucks for this supplier with their current load
		truckStmt := spanner.Statement{
			SQL: `SELECT d.DriverId, d.Name, d.VehicleId, COALESCE(v.VehicleClass, ''),
			             COALESCE(d.LicensePlate, ''), v.MaxVolumeVU,
			             COALESCE(d.TruckStatus, 'AVAILABLE'),
			             COALESCE(d.CurrentLocation, '')
			      FROM Drivers d
			      JOIN Vehicles v ON d.VehicleId = v.VehicleId
			      WHERE d.SupplierId = @sid
			        AND d.IsActive = true
			        AND d.VehicleId IS NOT NULL
			        AND v.IsActive = true
			        AND NOT EXISTS (
			            SELECT 1 FROM Orders o2
			            WHERE o2.DriverId = d.DriverId AND o2.State = 'IN_TRANSIT'
			        )`,
			Params: map[string]interface{}{"sid": supplierID},
		}

		truckIter := readClient.Single().Query(ctx, truckStmt)
		defer truckIter.Stop()

		var recs []TruckRecommendation
		for {
			row, err := truckIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[RECOMMEND-REASSIGN] truck query error: %v", err)
				break
			}

			var rec TruckRecommendation
			var currentLoc string
			if err := row.Columns(&rec.DriverID, &rec.DriverName, &rec.VehicleID, &rec.VehicleClass,
				&rec.LicensePlate, &rec.MaxVolumeVU, &rec.TruckStatus, &currentLoc); err != nil {
				continue
			}

			// Calculate current load for this driver
			loadStmt := spanner.Statement{
				SQL: `SELECT COALESCE(SUM(li.Quantity * COALESCE(sp.VolumetricUnit, sp.PalletFootprint, 1.0)), 0)
				      FROM Orders o
				      JOIN OrderLineItems li ON o.OrderId = li.OrderId
				      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
				      WHERE o.DriverId = @driverId AND o.State IN ('LOADED', 'DISPATCHED', 'IN_TRANSIT')`,
				Params: map[string]interface{}{"driverId": rec.DriverID},
			}
			loadIter := readClient.Single().Query(ctx, loadStmt)
			if loadRow, err := loadIter.Next(); err == nil {
				var usedVU spanner.NullFloat64
				if loadRow.Columns(&usedVU) == nil {
					rec.UsedVolumeVU = usedVU.Float64
				}
			}
			loadIter.Stop()

			rec.FreeVolumeVU = rec.MaxVolumeVU - rec.UsedVolumeVU

			// Count assigned orders
			countStmt := spanner.Statement{
				SQL:    `SELECT COUNT(*) FROM Orders WHERE DriverId = @driverId AND State IN ('LOADED', 'DISPATCHED', 'IN_TRANSIT')`,
				Params: map[string]interface{}{"driverId": rec.DriverID},
			}
			countIter := readClient.Single().Query(ctx, countStmt)
			if countRow, err := countIter.Next(); err == nil {
				var cnt int64
				if countRow.Columns(&cnt) == nil {
					rec.OrderCount = int(cnt)
				}
			}
			countIter.Stop()

			// Distance from truck's last known location to the retailer
			driverLat, driverLng := parseWKT(currentLoc)
			if driverLat != 0 && driverLng != 0 && retailerLat != 0 && retailerLng != 0 {
				rec.DistanceKm = haversine(driverLat, driverLng, retailerLat, retailerLng)
			} else {
				rec.DistanceKm = -1 // Unknown — no GPS data
			}

			// Skip the current driver — we're reassigning AWAY from them
			if currentDriverID.Valid && rec.DriverID == currentDriverID.StringVal {
				continue
			}

			// Score: lower is better. Combines capacity fitness + distance + status penalty
			capacityScore := 0.0
			if rec.FreeVolumeVU >= orderVolumeVU {
				capacityScore = 0 // fits perfectly
			} else {
				capacityScore = 50 // doesn't fit — heavy penalty
			}

			distanceScore := 0.0
			if rec.DistanceKm >= 0 {
				distanceScore = rec.DistanceKm * 2 // 2 points per km
			} else {
				distanceScore = 20 // unknown location penalty
			}

			statusPenalty := 0.0
			if rec.TruckStatus == "MAINTENANCE" {
				statusPenalty = 100
			} else if rec.TruckStatus == "RETURNING" {
				statusPenalty = 30
			} else if rec.TruckStatus == "IN_TRANSIT" {
				statusPenalty = 10
			}

			rec.Score = capacityScore + distanceScore + statusPenalty

			// Human-readable recommendation
			switch {
			case rec.FreeVolumeVU >= orderVolumeVU && rec.DistanceKm >= 0 && rec.DistanceKm < 5:
				rec.Recommendation = "Best match — close and has capacity"
			case rec.FreeVolumeVU >= orderVolumeVU && rec.DistanceKm >= 0 && rec.DistanceKm < 15:
				rec.Recommendation = "Good match — has capacity, moderate distance"
			case rec.FreeVolumeVU >= orderVolumeVU:
				rec.Recommendation = "Has capacity but far away"
			case rec.TruckStatus == "MAINTENANCE":
				rec.Recommendation = "In maintenance — unavailable"
			default:
				rec.Recommendation = "Low capacity — may not fit"
			}

			recs = append(recs, rec)
		}

		// Sort by score ascending (best first)
		sort.Slice(recs, func(i, j int) bool {
			return recs[i].Score < recs[j].Score
		})

		resp := RecommendReassignResponse{
			OrderID:         req.OrderID,
			RetailerName:    retailerName,
			OrderVolumeVU:   orderVolumeVU,
			Recommendations: recs,
		}
		if currentDriverID.Valid {
			resp.CurrentDriver = currentDriverID.StringVal
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ── Geo helpers ──────────────────────────────────────────────────────────────

func parseWKT(wkt string) (lat, lng float64) {
	wkt = strings.TrimSpace(wkt)
	if wkt == "" {
		return 0, 0
	}
	wkt = strings.TrimPrefix(wkt, "POINT(")
	wkt = strings.TrimSuffix(wkt, ")")
	parts := strings.Fields(wkt)
	if len(parts) != 2 {
		return 0, 0
	}
	fmt.Sscanf(parts[0], "%f", &lng)
	fmt.Sscanf(parts[1], "%f", &lat)
	return lat, lng
}

func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
