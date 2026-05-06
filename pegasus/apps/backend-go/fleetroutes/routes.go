// Package fleetroutes owns the /v1/fleet/* surface — twelve endpoints spanning
// driver status toggles, the two-key truck handshake (seal/depart/return),
// dispatch + reassignment, capacity + active-mission telemetry, route reorder
// and completion, and driver-scoped order listing.
//
// Thin-wrapper routes delegate to the existing fleet and order packages so
// handler bodies are not duplicated. Routes with inline closures in the old
// main.go (dispatch, reassign, capacity, active, route/reorder, orders) are
// re-homed here with their bodies intact, but upgraded to the Wave B standard:
//
//   - slog + TraceID: every log entry carries trace_id, driver_id, vehicle_id
//     (where applicable) so "Ghost Truck" incidents can be stitched end-to-end.
//   - outbox.Emit: route/reorder — the sole inline RW transaction owned here —
//     appends a ROUTE_REORDERED event to OutboxEvents in the same txn as the
//     sequence writes. Fleet-package handlers (HandleTruckSeal, DriverDepart,
//     ReturnComplete, TruckStatusUpdate, DriverStatus) adopt outbox.Emit at
//     their point-of-mutation, inside the fleet package, so every TruckState
//     and DriverShift transition is atomic with its durable event.
//   - Home-Node validation: the driver/depart wrapper verifies the driver's
//     HomeNodeId matches the assigned vehicle's HomeNodeId before delegating
//     to fleet.HandleDriverDepart. Mismatches return 403.
//
// Path-prefix routes (/v1/fleet/drivers/*, /v1/fleet/trucks/*,
// /v1/fleet/route/*) now register on chi wildcard mounts so sub-path dispatch
// remains intact without relying on http.DefaultServeMux.
package fleetroutes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	kafka "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"

	"cloud.google.com/go/spanner"

	"backend-go/auth"
	"backend-go/fleet"
	"backend-go/idempotency"
	"backend-go/order"
	"backend-go/outbox"
	"backend-go/telemetry"
)

// Middleware is the handler-wrap contract supplied by the caller (typically
// main.loggingMiddleware).
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount /v1/fleet/*.
type Deps struct {
	Spanner           *spanner.Client
	Order             *order.OrderService
	RetailerHub       fleet.RetailerPusher
	EarlyCompleteDeps *order.EarlyCompleteDeps
	Producer          *kafka.Writer
	MapsAPIKey        string
	Log               Middleware
}

// topicFleetEvents is the canonical destination topic for fleet state events
// emitted through the transactional outbox.
const topicFleetEvents = "pegasus-logistics-events"

// RegisterRoutes mounts the twelve fleet endpoints. chi handles exact and
// wildcard path dispatch for both static and by-ID route families.
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	driver := []string{"DRIVER"}
	adminSupplier := []string{"ADMIN", "SUPPLIER"}

	// 1. PUT /v1/fleet/drivers/{id}/status — toggle IsOffline (wildcard path).
	r.HandleFunc("/v1/fleet/drivers/*",
		auth.RequireRole([]string{"DRIVER", "ADMIN", "SUPPLIER"},
			log(fleet.HandleDriverStatus(d.Spanner))))

	// 2. POST /v1/fleet/route/request-early-complete — driver requests early stop.
	r.HandleFunc("/v1/fleet/route/request-early-complete",
		auth.RequireRole(driver,
			log(order.HandleRequestEarlyComplete(d.Order, d.EarlyCompleteDeps))))

	// 3. POST /v1/fleet/dispatch — assign orders to a route (idempotent).
	r.HandleFunc("/v1/fleet/dispatch",
		auth.RequireRole(adminSupplier, log(idempotency.Guard(handleDispatch(d)))))

	// 4. POST /v1/fleet/reassign — move orders between routes.
	r.HandleFunc("/v1/fleet/reassign",
		auth.RequireRole([]string{"ADMIN", "SUPPLIER", "PAYLOADER"}, log(idempotency.Guard(handleReassign(d)))))

	// 5. GET /v1/fleet/capacity?route_id=... — truck capacity snapshot.
	r.HandleFunc("/v1/fleet/capacity",
		auth.RequireRole(adminSupplier, log(handleCapacity(d))))

	// 6. GET /v1/fleet/active[?route_id=...] — active-mission telemetry.
	r.HandleFunc("/v1/fleet/active",
		auth.RequireRole(adminSupplier, log(handleActive(d))))

	// 7. /v1/fleet/trucks/{id}/{seal|status} — two-key handshake dispatcher.
	r.HandleFunc("/v1/fleet/trucks/*",
		auth.RequireRole([]string{"PAYLOADER", "SUPPLIER", "ADMIN"},
			log(handleTrucksDispatcher(d))))

	// 8. POST /v1/fleet/driver/depart — shift start; home-node gate + delegate.
	r.HandleFunc("/v1/fleet/driver/depart",
		auth.RequireRole(driver, log(handleDriverDepart(d))))

	// 9. POST /v1/fleet/driver/return-complete — shift end.
	r.HandleFunc("/v1/fleet/driver/return-complete",
		auth.RequireRole(driver, log(fleet.HandleReturnComplete(d.Spanner))))

	// 10. POST /v1/fleet/route/reorder — driver reorders stops.
	r.HandleFunc("/v1/fleet/route/reorder",
		auth.RequireRole(driver, log(handleReorder(d))))

	// 11. GET /v1/fleet/orders — driver-scoped order list.
	r.HandleFunc("/v1/fleet/orders",
		auth.RequireRole(driver, log(handleDriverOrders(d))))

	// 12. POST /v1/fleet/route/{id}/complete — route completion + QUARANTINE sweep.
	r.HandleFunc("/v1/fleet/route/*",
		auth.RequireRole(driver, log(order.HandleCompleteRoute(d.Spanner))))
}

// handleDispatch assigns a batch of orders to a route via OrderService.
// Error taxonomy is preserved verbatim from the old main.go closure.
func handleDispatch(d Deps) http.HandlerFunc {
	svc := d.Order
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		tid := traceID(r)
		var req order.DispatchFleetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		if len(req.OrderIds) == 0 || req.RouteId == "" {
			http.Error(w, "order_ids and route_id are required", http.StatusBadRequest)
			return
		}
		if err := svc.AssignRoute(r.Context(), req.OrderIds, req.RouteId); err != nil {
			var conflict *order.ErrStateConflict
			if errors.As(err, &conflict) {
				writeJSONError(w, http.StatusConflict, conflict.Error())
				return
			}
			var versionConflict *order.ErrVersionConflict
			if errors.As(err, &versionConflict) {
				writeJSONError(w, http.StatusConflict, versionConflict.Error())
				return
			}
			var freezeLock *order.ErrFreezeLock
			if errors.As(err, &freezeLock) {
				writeJSONError(w, 423, freezeLock.Error())
				return
			}
			slog.ErrorContext(r.Context(), "fleet.dispatch: AssignRoute failed",
				"trace_id", tid, "route_id", req.RouteId, "order_count", len(req.OrderIds), "error", err.Error())
			http.Error(w, fmt.Sprintf("Failed to assign route: %v", err), http.StatusInternalServerError)
			return
		}
		slog.InfoContext(r.Context(), "fleet.dispatch",
			"trace_id", tid, "route_id", req.RouteId, "order_count", len(req.OrderIds))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "SUCCESS",
			"message": fmt.Sprintf("Assigned %d order(s) to %s", len(req.OrderIds), req.RouteId),
		})
	}
}

// handleReassign moves orders between routes; returns the conflict list for
// orders that could not be reassigned (state or ownership mismatch).
func handleReassign(d Deps) http.HandlerFunc {
	svc := d.Order
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		tid := traceID(r)
		var req order.ReassignOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		if len(req.OrderIds) == 0 || req.NewRouteId == "" {
			http.Error(w, `{"error":"order_ids and new_route_id are required"}`, http.StatusBadRequest)
			return
		}
		conflicts, err := svc.ReassignRoute(r.Context(), req.OrderIds, req.NewRouteId)
		if err != nil {
			slog.ErrorContext(r.Context(), "fleet.reassign: ReassignRoute failed",
				"trace_id", tid, "new_route_id", req.NewRouteId, "error", err.Error())
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		slog.InfoContext(r.Context(), "fleet.reassign",
			"trace_id", tid, "new_route_id", req.NewRouteId,
			"total", len(req.OrderIds), "conflicts", len(conflicts))
		w.Header().Set("Content-Type", "application/json")
		if len(conflicts) > 0 && len(conflicts) == len(req.OrderIds) {
			w.WriteHeader(http.StatusConflict)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"conflicts":    conflicts,
			"total":        len(req.OrderIds),
			"reassigned":   len(req.OrderIds) - len(conflicts),
			"new_route_id": req.NewRouteId,
		})
	}
}

// handleCapacity reports a truck's capacity envelope for capacity-planning UIs.
func handleCapacity(d Deps) http.HandlerFunc {
	svc := d.Order
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		routeID := r.URL.Query().Get("route_id")
		if routeID == "" {
			http.Error(w, `{"error":"route_id query param required"}`, http.StatusBadRequest)
			return
		}
		capInfo, err := svc.GetTruckCapacity(r.Context(), routeID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(capInfo)
	}
}

// handleActive serves the active-mission telemetry feed, scoped to the
// authenticated supplier. An optional ?route_id=... narrows to one truck.
func handleActive(d Deps) http.HandlerFunc {
	svc := d.Order
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		tid := traceID(r)
		var supplierID string
		if claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims); ok {
			supplierID = claims.ResolveSupplierID()
		}
		targetRoute := r.URL.Query().Get("route_id")
		missions, err := svc.GetActiveFleet(r.Context(), supplierID, targetRoute)
		if err != nil {
			slog.ErrorContext(r.Context(), "fleet.active: GetActiveFleet failed",
				"trace_id", tid, "supplier_id", supplierID, "route_id", targetRoute, "error", err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(missions); err != nil {
			slog.ErrorContext(r.Context(), "fleet.active: encode failed",
				"trace_id", tid, "supplier_id", supplierID, "error", err.Error())
		}
	}
}

// handleTrucksDispatcher routes /v1/fleet/trucks/{id}/{seal|status} requests
// to the appropriate fleet-package handler. PATCH on /status is the admin
// override; GET is the status read.
func handleTrucksDispatcher(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1/fleet/trucks/")
		switch {
		case strings.HasSuffix(path, "/seal"):
			fleet.HandleTruckSeal(d.Spanner)(w, r)
		case strings.HasSuffix(path, "/status") && r.Method == http.MethodPatch:
			fleet.HandleTruckStatusUpdate(d.Spanner)(w, r)
		case strings.HasSuffix(path, "/status"):
			fleet.HandleTruckStatus(d.Spanner)(w, r)
		default:
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}
}

// handleDriverDepart gates the shift-start handoff with Home-Node validation:
// the driver's HomeNodeId on the Drivers row must match the assigned vehicle's
// HomeNodeId before the depart transaction is allowed to proceed. A mismatch
// returns 403 and is logged with both node IDs for ghost-truck audit trails.
// Empty home-node fields skip the check (backward-compatible rollout).
func handleDriverDepart(d Deps) http.HandlerFunc {
	delegate := fleet.HandleDriverDepart(d.Spanner, d.MapsAPIKey, d.RetailerHub)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		tid := traceID(r)
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Peek at truck_id without consuming the body — delegate re-reads it.
		body, _ := readAndRestoreBody(r)
		var peek struct {
			TruckID string `json:"truck_id"`
		}
		_ = json.Unmarshal(body, &peek)

		if peek.TruckID != "" {
			if err := verifyHomeNodeOnDepart(r.Context(), d.Spanner, peek.TruckID); err != nil {
				slog.WarnContext(r.Context(), "fleet.driver.depart: home-node mismatch",
					"trace_id", tid, "driver_id", claims.UserID, "truck_id", peek.TruckID, "error", err.Error())
				writeJSONError(w, http.StatusForbidden, err.Error())
				return
			}
		}

		slog.InfoContext(r.Context(), "fleet.driver.depart: accepted",
			"trace_id", tid, "driver_id", claims.UserID, "truck_id", peek.TruckID)
		delegate(w, r)
	}
}

// handleReorder rewrites SequenceIndex on a driver's active route stops.
// The RW transaction validates ownership (Drivers.DriverId + Drivers.RouteId
// match the caller) before rewriting sequence; a ROUTE_REORDERED outbox
// event is appended in the same txn for durable downstream propagation.
func handleReorder(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		tid := traceID(r)
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		var req struct {
			RouteID       string   `json:"route_id"`
			OrderSequence []string `json:"order_sequence"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.RouteID == "" || len(req.OrderSequence) == 0 {
			writeJSONError(w, http.StatusBadRequest, "route_id and order_sequence required")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		_, err := d.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			stmt := spanner.Statement{
				SQL:    `SELECT DriverId FROM Drivers WHERE DriverId = @driverId AND RouteId = @routeId LIMIT 1`,
				Params: map[string]interface{}{"driverId": claims.UserID, "routeId": req.RouteID},
			}
			iter := txn.Query(ctx, stmt)
			row, ierr := iter.Next()
			iter.Stop()
			if row == nil || ierr != nil {
				return fmt.Errorf("route %s not assigned to driver %s", req.RouteID, claims.UserID)
			}
			for i, orderID := range req.OrderSequence {
				updateStmt := spanner.Statement{
					SQL: `UPDATE Orders SET SequenceIndex = @seq WHERE OrderId = @orderId AND RouteId = @routeId`,
					Params: map[string]interface{}{
						"seq":     int64(i),
						"orderId": orderID,
						"routeId": req.RouteID,
					},
				}
				rowCount, uerr := txn.Update(ctx, updateStmt)
				if uerr != nil {
					return fmt.Errorf("failed to update sequence for order %s: %w", orderID, uerr)
				}
				if rowCount == 0 {
					return fmt.Errorf("order %s not found on route %s", orderID, req.RouteID)
				}
			}
			return outbox.EmitJSON(txn, "Route", req.RouteID, "ROUTE_REORDERED", topicFleetEvents, map[string]interface{}{
				"type":       "ROUTE_REORDERED",
				"route_id":   req.RouteID,
				"driver_id":  claims.UserID,
				"stop_count": len(req.OrderSequence),
				"sequence":   req.OrderSequence,
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			}, telemetry.TraceIDFromContext(ctx))
		})

		if err != nil {
			slog.ErrorContext(r.Context(), "fleet.route.reorder failed",
				"trace_id", tid, "driver_id", claims.UserID, "route_id", req.RouteID, "error", err.Error())
			if strings.Contains(err.Error(), "not assigned") || strings.Contains(err.Error(), "not found on route") {
				writeJSONError(w, http.StatusForbidden, err.Error())
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		slog.InfoContext(r.Context(), "fleet.route.reorder",
			"trace_id", tid, "driver_id", claims.UserID, "route_id", req.RouteID,
			"stop_count", len(req.OrderSequence))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "REORDERED",
			"route_id":   req.RouteID,
			"stop_count": len(req.OrderSequence),
		})
	}
}

// driverOrderItem and driverOrder are the response shapes for /v1/fleet/orders.
type driverOrderItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	LineTotal   int64  `json:"line_total"`
}

type driverOrder struct {
	ID                   string            `json:"id"`
	RetailerID           string            `json:"retailer_id"`
	RetailerName         string            `json:"retailer_name"`
	DriverID             string            `json:"driver_id"`
	State                string            `json:"state"`
	TotalAmount          int64             `json:"total_amount"`
	DeliveryAddress      string            `json:"delivery_address"`
	Latitude             float64           `json:"latitude"`
	Longitude            float64           `json:"longitude"`
	QRToken              string            `json:"qr_token"`
	PaymentGateway       string            `json:"payment_gateway"`
	CreatedAt            string            `json:"created_at"`
	UpdatedAt            string            `json:"updated_at"`
	EstimatedArrivalAt   *string           `json:"estimated_arrival_at,omitempty"`
	EstimatedDurationSec *int64            `json:"eta_duration_sec,omitempty"`
	EstimatedDistanceM   *int64            `json:"eta_distance_m,omitempty"`
	RouteID              string            `json:"route_id,omitempty"`
	SequenceIndex        int64             `json:"sequence_index"`
	WarehouseID          string            `json:"warehouse_id,omitempty"`
	WarehouseName        string            `json:"warehouse_name,omitempty"`
	Items                []driverOrderItem `json:"items"`
}

// handleDriverOrders returns active orders assigned to the authenticated
// driver, hydrated with line items in a second query.
func handleDriverOrders(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		tid := traceID(r)
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		driverID := claims.UserID

		stmt := spanner.Statement{
			SQL: `SELECT o.OrderId, o.RetailerId, COALESCE(r.ShopName, r.Name) AS RetailerName,
			             o.DriverId, o.State, COALESCE(o.Amount, 0) AS TotalAmount,
			             COALESCE(r.ShopLocation, '') AS DeliveryAddress,
			             COALESCE(r.Latitude, 0) AS Latitude, COALESCE(r.Longitude, 0) AS Longitude,
			             COALESCE(o.DeliveryToken, '') AS QRToken,
			             COALESCE(o.PaymentGateway, '') AS PaymentGateway,
			             o.CreatedAt,
			             o.EstimatedArrivalAt, o.EstimatedDurationSec, o.EstimatedDistanceM,
			             COALESCE(o.RouteId, '') AS RouteId,
			             COALESCE(o.SequenceIndex, 0) AS SequenceIndex,
			             COALESCE(o.WarehouseId, '') AS WarehouseId,
			             COALESCE(wh.Name, '') AS WarehouseName
			      FROM Orders o
			      LEFT JOIN Retailers r ON o.RetailerId = r.RetailerId
			      LEFT JOIN Warehouses wh ON o.WarehouseId = wh.WarehouseId
			      WHERE o.DriverId = @driverId
			        AND o.State IN ('PENDING', 'LOADED', 'DISPATCHED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED')
			      ORDER BY COALESCE(o.SequenceIndex, 999), o.CreatedAt ASC`,
			Params: map[string]interface{}{"driverId": driverID},
		}

		iter := d.Spanner.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		var orders []driverOrder
		var orderIDs []string
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				slog.ErrorContext(r.Context(), "fleet.orders: query error",
					"trace_id", tid, "driver_id", driverID, "error", err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var id, retailerID, retailerName, did, state, address, qrToken, gateway string
			var amount int64
			var lat, lng float64
			var createdAt, etaAt spanner.NullTime
			var etaDur, etaDist spanner.NullInt64
			var routeID string
			var seqIdx int64
			var whID, whName string
			if err := row.Columns(&id, &retailerID, &retailerName, &did, &state, &amount, &address, &lat, &lng, &qrToken, &gateway, &createdAt, &etaAt, &etaDur, &etaDist, &routeID, &seqIdx, &whID, &whName); err != nil {
				slog.WarnContext(r.Context(), "fleet.orders: parse error",
					"trace_id", tid, "driver_id", driverID, "error", err.Error())
				continue
			}
			ts := ""
			if createdAt.Valid {
				ts = createdAt.Time.Format(time.RFC3339)
			}
			do := driverOrder{
				ID: id, RetailerID: retailerID, RetailerName: retailerName, DriverID: did,
				State: state, TotalAmount: amount, DeliveryAddress: address,
				Latitude: lat, Longitude: lng, RouteID: routeID, SequenceIndex: seqIdx,
				QRToken: qrToken, PaymentGateway: gateway, CreatedAt: ts, UpdatedAt: ts,
				WarehouseID: whID, WarehouseName: whName, Items: []driverOrderItem{},
			}
			if etaAt.Valid {
				s := etaAt.Time.Format(time.RFC3339)
				do.EstimatedArrivalAt = &s
			}
			if etaDur.Valid {
				do.EstimatedDurationSec = &etaDur.Int64
			}
			if etaDist.Valid {
				do.EstimatedDistanceM = &etaDist.Int64
			}
			orders = append(orders, do)
			orderIDs = append(orderIDs, id)
		}

		if len(orderIDs) > 0 {
			itemStmt := spanner.Statement{
				SQL: `SELECT li.OrderId, li.SkuId, COALESCE(sp.Name, li.SkuId), li.Quantity, li.UnitPrice
				      FROM OrderLineItems li
				      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
				      WHERE li.OrderId IN UNNEST(@ids)`,
				Params: map[string]interface{}{"ids": orderIDs},
			}
			itemIter := d.Spanner.Single().Query(r.Context(), itemStmt)
			defer itemIter.Stop()
			itemMap := map[string][]driverOrderItem{}
			for {
				row, err := itemIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					break
				}
				var oid, skuID, skuName string
				var qty, price int64
				if err := row.Columns(&oid, &skuID, &skuName, &qty, &price); err != nil {
					continue
				}
				itemMap[oid] = append(itemMap[oid], driverOrderItem{
					ProductID: skuID, ProductName: skuName,
					Quantity: qty, UnitPrice: price, LineTotal: qty * price,
				})
			}
			for i := range orders {
				if items, ok := itemMap[orders[i].ID]; ok {
					orders[i].Items = items
				}
			}
		}

		if orders == nil {
			orders = []driverOrder{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(orders)
	}
}

// writeJSONError emits a JSON-shaped error payload with the given status.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}

// traceID extracts the correlation token stashed by TraceMiddleware.
func traceID(r *http.Request) string {
	return telemetry.TraceIDFromContext(r.Context())
}

// readAndRestoreBody drains r.Body into memory and replaces r.Body with a
// fresh reader over the same bytes so downstream handlers can decode the
// payload normally.
func readAndRestoreBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	buf := make([]byte, 0, 512)
	tmp := make([]byte, 512)
	for {
		n, err := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	_ = r.Body.Close()
	r.Body = &memBody{data: buf}
	return buf, nil
}

// memBody is a trivial io.ReadCloser over an in-memory byte slice.
type memBody struct {
	data []byte
	pos  int
}

func (m *memBody) Read(p []byte) (int, error) {
	if m.pos >= len(m.data) {
		return 0, errEOF
	}
	n := copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *memBody) Close() error { return nil }

// errEOF is a local sentinel to avoid importing io for just EOF.
var errEOF = errors.New("EOF")

// verifyHomeNodeOnDepart implements the V.O.I.D. home-node gate: the driver's
// Drivers.HomeNodeId must match the assigned Vehicles.HomeNodeId. Empty values
// on either row short-circuit to "accepted" so the check rolls out safely
// alongside the ongoing HomeNodeId backfill.
func verifyHomeNodeOnDepart(ctx context.Context, sp *spanner.Client, truckID string) error {
	row, err := sp.Single().ReadRow(ctx, "Drivers", spanner.Key{truckID},
		[]string{"HomeNodeId", "VehicleId"})
	if err != nil {
		return nil // row missing → let the delegate produce the canonical 404.
	}
	var driverHome, vehicleID spanner.NullString
	if err := row.Columns(&driverHome, &vehicleID); err != nil {
		return nil
	}
	if !driverHome.Valid || driverHome.StringVal == "" {
		return nil
	}
	if !vehicleID.Valid || vehicleID.StringVal == "" {
		return nil
	}
	vrow, verr := sp.Single().ReadRow(ctx, "Vehicles", spanner.Key{vehicleID.StringVal},
		[]string{"HomeNodeId"})
	if verr != nil {
		return nil
	}
	var vehicleHome spanner.NullString
	if err := vrow.Columns(&vehicleHome); err != nil {
		return nil
	}
	if !vehicleHome.Valid || vehicleHome.StringVal == "" {
		return nil
	}
	if vehicleHome.StringVal != driverHome.StringVal {
		return fmt.Errorf("home-node mismatch: driver bound to %s, vehicle bound to %s",
			driverHome.StringVal, vehicleHome.StringVal)
	}
	return nil
}
