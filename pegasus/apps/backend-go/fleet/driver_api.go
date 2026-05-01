package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── Driver Earnings ─────────────────────────────────────────────────────────
// GET /v1/driver/earnings — Returns delivery statistics and volume for the authenticated driver.

type DailyEarning struct {
	Date          string `json:"date"`
	DeliveryCount int64  `json:"delivery_count"`
	Volume        int64  `json:"volume"`
}

type DriverEarningsResponse struct {
	TotalDeliveries int64          `json:"total_deliveries"`
	TotalVolume     int64          `json:"total_volume"`
	TotalRoutes     int64          `json:"total_routes"`
	Last30Days      []DailyEarning `json:"last_30_days"`
}

func HandleDriverEarnings(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		resp := DriverEarningsResponse{}

		// Aggregate totals
		totalStmt := spanner.Statement{
			SQL: `SELECT
					COUNT(OrderId) as total_deliveries,
					IFNULL(SUM(Amount), 0) as total_volume,
					COUNT(DISTINCT RouteId) as total_routes
				FROM Orders
				WHERE DriverId = @driverId AND State = 'COMPLETED'`,
			Params: map[string]interface{}{"driverId": claims.UserID},
		}
		totalIter := client.Single().Query(ctx, totalStmt)
		defer totalIter.Stop()
		row, err := totalIter.Next()
		if err == nil {
			if err := row.Columns(&resp.TotalDeliveries, &resp.TotalVolume, &resp.TotalRoutes); err != nil {
				log.Printf("[DRIVER EARNINGS] Decode error: %v", err)
			}
		}
		totalIter.Stop()

		// Daily breakdown (last 30 days)
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		dailyStmt := spanner.Statement{
			SQL: `SELECT
					FORMAT_TIMESTAMP('%Y-%m-%d', CreatedAt) as day,
					COUNT(OrderId) as delivery_count,
					IFNULL(SUM(Amount), 0) as volume
				FROM Orders
				WHERE DriverId = @driverId
				  AND State = 'COMPLETED'
				  AND CreatedAt >= @since
				GROUP BY day
				ORDER BY day DESC`,
			Params: map[string]interface{}{
				"driverId": claims.UserID,
				"since":    thirtyDaysAgo,
			},
		}
		dailyIter := client.Single().Query(ctx, dailyStmt)
		defer dailyIter.Stop()
		for {
			row, err := dailyIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[DRIVER EARNINGS] Daily query error: %v", err)
				break
			}
			var d DailyEarning
			if err := row.Columns(&d.Date, &d.DeliveryCount, &d.Volume); err != nil {
				continue
			}
			resp.Last30Days = append(resp.Last30Days, d)
		}

		if resp.Last30Days == nil {
			resp.Last30Days = []DailyEarning{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ── Driver Delivery History ─────────────────────────────────────────────────
// GET /v1/driver/history — Returns completed and cancelled delivery records.

type DeliveryHistoryItem struct {
	OrderID     string `json:"order_id"`
	RetailerID  string `json:"retailer_id"`
	SupplierId  string `json:"supplier_id"`
	State       string `json:"state"`
	Amount      int64  `json:"amount"`
	RouteID     string `json:"route_id"`
	CompletedAt string `json:"completed_at"`
}

type DriverHistoryResponse struct {
	Deliveries []DeliveryHistoryItem `json:"deliveries"`
	Total      int                   `json:"total"`
}

func HandleDriverHistory(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		stmt := spanner.Statement{
			SQL: `SELECT OrderId, RetailerId, SupplierId, State, Amount, RouteId, CreatedAt
				FROM Orders
				WHERE DriverId = @driverId
				  AND State IN ('COMPLETED', 'CANCELLED', 'QUARANTINE')
				ORDER BY CreatedAt DESC
				LIMIT 100`,
			Params: map[string]interface{}{"driverId": claims.UserID},
		}

		resp := DriverHistoryResponse{Deliveries: []DeliveryHistoryItem{}}
		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, `{"error":"query_failed"}`, http.StatusInternalServerError)
				return
			}
			var item DeliveryHistoryItem
			var amountNull spanner.NullInt64
			var routeNull spanner.NullString
			var supplierNull spanner.NullString
			var ts time.Time
			if err := row.Columns(&item.OrderID, &item.RetailerID, &supplierNull, &item.State, &amountNull, &routeNull, &ts); err != nil {
				log.Printf("[DRIVER HISTORY] Decode error: %v", err)
				continue
			}
			item.Amount = amountNull.Int64
			item.RouteID = routeNull.StringVal
			item.SupplierId = supplierNull.StringVal
			item.CompletedAt = ts.Format(time.RFC3339)
			resp.Deliveries = append(resp.Deliveries, item)
		}
		resp.Total = len(resp.Deliveries)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ── Driver Availability / Session Management ────────────────────────────────
// PATCH /v1/driver/availability — Toggle driver online/offline with reason codes.
// When going offline, driver MUST provide a reason:
//   SHIFT_COMPLETE — normal end-of-day
//   TRUCK_DAMAGED — vehicle issue (auto-sets TruckStatus=MAINTENANCE)
//   PERSONAL      — personal break / errand
//   OTHER         — free-text note required
// When going online, reason fields are cleared.
// Emits DRIVER_AVAILABILITY_CHANGED to Kafka for real-time admin visibility.

// AvailabilityChangedEmitter is called after a successful availability toggle.
// Wired in main.go to emit Kafka events + WebSocket push without circular imports.
type AvailabilityChangedEmitter func(driverID, supplierID string, available bool, reason, note, truckID string)

// AvailabilityEmitter is set from main.go during init.
var AvailabilityEmitter AvailabilityChangedEmitter

func HandleDriverAvailability(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch && r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			Available bool   `json:"available"`
			Reason    string `json:"reason,omitempty"` // SHIFT_COMPLETE | TRUCK_DAMAGED | PERSONAL | OTHER
			Note      string `json:"note,omitempty"`   // free-text for OTHER
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}

		// Validate reason when going offline
		validReasons := map[string]bool{
			"SHIFT_COMPLETE": true,
			"TRUCK_DAMAGED":  true,
			"PERSONAL":       true,
			"OTHER":          true,
		}
		if !req.Available {
			if req.Reason == "" {
				http.Error(w, `{"error":"reason is required when going offline (SHIFT_COMPLETE, TRUCK_DAMAGED, PERSONAL, OTHER)"}`, http.StatusBadRequest)
				return
			}
			if !validReasons[req.Reason] {
				http.Error(w, `{"error":"invalid reason — must be SHIFT_COMPLETE, TRUCK_DAMAGED, PERSONAL, or OTHER"}`, http.StatusBadRequest)
				return
			}
			if req.Reason == "OTHER" && strings.TrimSpace(req.Note) == "" {
				http.Error(w, `{"error":"note is required when reason is OTHER"}`, http.StatusBadRequest)
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var supplierID, vehicleID string

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Drivers", spanner.Key{claims.UserID}, []string{"TruckStatus", "SupplierId", "VehicleId"})
			if err != nil {
				return fmt.Errorf("driver not found: %w", err)
			}

			var currentStatus, sid, vid spanner.NullString
			if err := row.Columns(&currentStatus, &sid, &vid); err != nil {
				return err
			}
			if sid.Valid {
				supplierID = sid.StringVal
			}
			if vid.Valid {
				vehicleID = vid.StringVal
			}

			// Cannot go offline while IN_TRANSIT
			if !req.Available && currentStatus.StringVal == StatusInTransit {
				return fmt.Errorf("cannot go offline while IN_TRANSIT")
			}

			if req.Available {
				// Going ONLINE — clear offline fields, set active
				mut := spanner.Update("Drivers",
					[]string{"DriverId", "IsActive", "OfflineReason", "OfflineReasonNote", "OfflineAt"},
					[]interface{}{claims.UserID, true, nil, nil, nil},
				)
				return txn.BufferWrite([]*spanner.Mutation{mut})
			}

			// Going OFFLINE — set reason fields
			truckStatus := StatusAvailable
			if req.Reason == "TRUCK_DAMAGED" {
				truckStatus = StatusMaintenance
			}

			mut := spanner.Update("Drivers",
				[]string{"DriverId", "IsActive", "TruckStatus", "OfflineReason", "OfflineReasonNote", "OfflineAt"},
				[]interface{}{claims.UserID, false, truckStatus, req.Reason, spanner.NullString{StringVal: req.Note, Valid: req.Note != ""}, time.Now().UTC()},
			)
			return txn.BufferWrite([]*spanner.Mutation{mut})
		})
		if err != nil {
			log.Printf("[AVAILABILITY] Failed: %v", err)
			if strings.Contains(err.Error(), "cannot go offline") {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
				return
			}
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Emit DRIVER_AVAILABILITY_CHANGED (Kafka + WebSocket) — non-blocking
		if AvailabilityEmitter != nil && supplierID != "" {
			go AvailabilityEmitter(claims.UserID, supplierID, req.Available, req.Reason, req.Note, vehicleID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"available": req.Available,
			"reason":    req.Reason,
			"status":    "OK",
		})
	}
}

// ── Driver Offline Toggle ───────────────────────────────────────────────────
// PUT /v1/fleet/drivers/{id}/status
// Allows the driver app to toggle IsOffline — a transient connectivity state
// separate from IsActive (account-level). IsOffline drivers are excluded from
// dispatch queries but retain their account and assignment.

func HandleDriverStatus(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse driver ID from path: /v1/fleet/drivers/{id}/status
		path := strings.TrimPrefix(r.URL.Path, "/v1/fleet/drivers/")
		driverID := strings.TrimSuffix(path, "/status")
		if driverID == "" || strings.Contains(driverID, "/") {
			http.Error(w, `{"error":"driver id required in path"}`, http.StatusBadRequest)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// Only the driver themselves or a supplier admin can toggle offline status
		if claims.UserID != driverID && claims.Role != "ADMIN" {
			http.Error(w, `{"error":"forbidden — only the driver or their supplier can toggle offline status"}`, http.StatusForbidden)
			return
		}

		var req struct {
			IsOffline bool `json:"is_offline"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON — expected {\"is_offline\": true|false}"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Drivers", spanner.Key{driverID}, []string{"TruckStatus", "IsActive"})
			if err != nil {
				return fmt.Errorf("driver %s not found: %w", driverID, err)
			}

			var currentStatus spanner.NullString
			var isActive spanner.NullBool
			if err := row.Columns(&currentStatus, &isActive); err != nil {
				return err
			}

			// Cannot go offline while IN_TRANSIT
			status := StatusAvailable
			if currentStatus.Valid {
				status = currentStatus.StringVal
			}
			if req.IsOffline && status == StatusInTransit {
				return fmt.Errorf("cannot go offline while IN_TRANSIT — complete or return the route first")
			}

			mut := spanner.Update("Drivers",
				[]string{"DriverId", "IsOffline"},
				[]interface{}{driverID, req.IsOffline},
			)
			return txn.BufferWrite([]*spanner.Mutation{mut})
		})

		if err != nil {
			log.Printf("[FLEET] status toggle error for driver %s: %v", driverID, err)
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
				return
			}
			if strings.Contains(err.Error(), "cannot go offline") {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
				return
			}
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("[FLEET] driver %s IsOffline=%v (toggled by %s)", driverID, req.IsOffline, claims.UserID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"driver_id":  driverID,
			"is_offline": req.IsOffline,
			"status":     "OK",
		})
	}
}
