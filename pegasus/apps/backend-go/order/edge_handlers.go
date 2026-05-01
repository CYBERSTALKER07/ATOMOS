package order

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/hotspot"
	kafkaEvents "backend-go/kafka"
	"backend-go/notifications"
	"backend-go/proximity"
	"backend-go/workers"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Edge 5: AWAITING_PAYMENT Bypass Token
// ═══════════════════════════════════════════════════════════════════════════════

// HandleIssuePaymentBypass lets a supplier admin issue a 6-digit bypass token
// for an order stuck in AWAITING_PAYMENT (e.g. payment gateway is down).
// POST /v1/admin/orders/{id}/payment-bypass
func HandleIssuePaymentBypass(svc *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID string `json:"order_id"`
			Reason  string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Generate 6-digit token
		token, err := generatePaymentBypassToken()
		if err != nil {
			http.Error(w, `{"error":"token generation failed"}`, http.StatusInternalServerError)
			return
		}

		now := time.Now().UTC()
		_, err = svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "Version"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var state string
			var version int64
			if err := row.Columns(&state, &version); err != nil {
				return err
			}
			if state != "AWAITING_PAYMENT" {
				return fmt.Errorf("order must be AWAITING_PAYMENT, got %s", state)
			}
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "PaymentBypassToken", "PaymentBypassIssuedBy", "PaymentBypassIssuedAt", "Version", "UpdatedAt"},
					[]interface{}{req.OrderID, token, claims.UserID, now, version + 1, now}),
			})
		})
		if err != nil {
			log.Printf("[PAYMENT_BYPASS] Issue failed for order %s: %v", req.OrderID, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		// Audit event
		svc.PublishEvent(ctx, kafkaEvents.EventPaymentBypassIssued, map[string]string{
			"order_id":  req.OrderID,
			"issued_by": claims.UserID,
			"reason":    req.Reason,
		})
		writeOrderEvent(ctx, svc.Client, req.OrderID, claims.UserID, claims.Role, kafkaEvents.EventPaymentBypassIssued,
			map[string]string{"reason": req.Reason}, 0, 0)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "bypass_issued",
			"bypass_token": token,
			"order_id":     req.OrderID,
		})
	}
}

// HandleConfirmPaymentBypass lets the driver use a bypass token to mark payment complete.
// POST /v1/delivery/confirm-payment-bypass
func HandleConfirmPaymentBypass(svc *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID     string `json:"order_id"`
			BypassToken string `json:"bypass_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" || req.BypassToken == "" {
			http.Error(w, `{"error":"order_id and bypass_token required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		now := time.Now().UTC()
		_, err := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "PaymentBypassToken", "Version"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var state string
			var storedToken spanner.NullString
			var version int64
			if err := row.Columns(&state, &storedToken, &version); err != nil {
				return err
			}
			if state != "AWAITING_PAYMENT" {
				return fmt.Errorf("order must be AWAITING_PAYMENT, got %s", state)
			}
			if !storedToken.Valid || storedToken.StringVal != req.BypassToken {
				return fmt.Errorf("invalid bypass token")
			}
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "State", "Version", "UpdatedAt"},
					[]interface{}{req.OrderID, "COMPLETED", version + 1, now}),
			})
		})
		if err != nil {
			log.Printf("[PAYMENT_BYPASS] Confirm failed for order %s: %v", req.OrderID, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		svc.PublishEvent(ctx, kafkaEvents.EventPaymentBypassCompleted, map[string]string{
			"order_id":  req.OrderID,
			"driver_id": claims.UserID,
		})
		writeOrderEvent(ctx, svc.Client, req.OrderID, claims.UserID, claims.Role, kafkaEvents.EventPaymentBypassCompleted, nil, 0, 0)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "completed",
			"order_id": req.OrderID,
		})
	}
}

func generatePaymentBypassToken() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge 7: CANCEL_REQUESTED State + Supplier Approval
// ═══════════════════════════════════════════════════════════════════════════════

// HandleRequestCancel lets a retailer request cancellation of a pre-transit order.
// POST /v1/orders/{id}/request-cancel
func HandleRequestCancel(svc *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID string `json:"order_id"`
			Reason  string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		now := time.Now().UTC()
		_, err := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "RetailerId", "Version", "RequestedDeliveryDate", "CancelLockedAt"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var state string
			var retailerID spanner.NullString
			var version int64
			var requestedDD spanner.NullTime
			var cancelLockedAt spanner.NullTime
			if err := row.Columns(&state, &retailerID, &version, &requestedDD, &cancelLockedAt); err != nil {
				return err
			}
			// Only allowed for pre-transit states
			if state != "PENDING" && state != "PENDING_REVIEW" && state != "DISPATCHED" && state != "LOADED" && state != "SCHEDULED" && state != "AUTO_ACCEPTED" {
				return fmt.Errorf("cancel only allowed in PENDING/PENDING_REVIEW/DISPATCHED/LOADED/SCHEDULED/AUTO_ACCEPTED, got %s", state)
			}
			// Preorder date gate: cancel only if delivery is >= 5 calendar days away (Tashkent TZ)
			if state == "SCHEDULED" || state == "AUTO_ACCEPTED" {
				if cancelLockedAt.Valid {
					return &ErrStateConflict{OrderID: req.OrderID, CurrentState: state, AttemptedOp: "cancel (cancel-locked)"}
				}
				if requestedDD.Valid {
					nowTKT := proximity.TashkentNow()
					todayMidnight := time.Date(nowTKT.Year(), nowTKT.Month(), nowTKT.Day(), 0, 0, 0, 0, proximity.TashkentLocation)
					cancelCutoff := todayMidnight.AddDate(0, 0, 5)
					deliveryTKT := requestedDD.Time.In(proximity.TashkentLocation)
					if deliveryTKT.Before(cancelCutoff) {
						return &ErrStateConflict{OrderID: req.OrderID, CurrentState: state, AttemptedOp: "cancel (delivery < 5 days away)"}
					}
				}
			}
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "State", "CancelRequestedBy", "CancelRequestedAt", "CancelReason", "Version", "UpdatedAt"},
					[]interface{}{req.OrderID, "CANCEL_REQUESTED", claims.UserID, now, req.Reason, version + 1, now}),
			})
		})
		if err != nil {
			log.Printf("[CANCEL_REQUEST] Failed for order %s: %v", req.OrderID, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		svc.PublishEvent(ctx, kafkaEvents.EventCancelRequested, map[string]string{
			"order_id":     req.OrderID,
			"requested_by": claims.UserID,
			"reason":       req.Reason,
		})
		writeOrderEvent(ctx, svc.Client, req.OrderID, claims.UserID, claims.Role, kafkaEvents.EventCancelRequested,
			map[string]string{"reason": req.Reason}, 0, 0)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "cancel_requested",
			"order_id": req.OrderID,
		})
	}
}

// HandleApproveCancel lets a supplier approve a cancel request.
// POST /v1/admin/orders/{id}/approve-cancel
func HandleApproveCancel(svc *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		now := time.Now().UTC()
		_, err := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "Version"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var state string
			var version int64
			if err := row.Columns(&state, &version); err != nil {
				return err
			}
			if state != "CANCEL_REQUESTED" {
				return fmt.Errorf("order must be CANCEL_REQUESTED, got %s", state)
			}
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "State", "Version", "UpdatedAt"},
					[]interface{}{req.OrderID, "CANCELLED", version + 1, now}),
			})
		})
		if err != nil {
			log.Printf("[CANCEL_APPROVE] Failed for order %s: %v", req.OrderID, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		svc.PublishEvent(ctx, kafkaEvents.EventCancelApproved, map[string]string{
			"order_id":    req.OrderID,
			"approved_by": claims.UserID,
		})
		writeOrderEvent(ctx, svc.Client, req.OrderID, claims.UserID, claims.Role, kafkaEvents.EventCancelApproved, nil, 0, 0)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "cancelled",
			"order_id": req.OrderID,
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge 23: SMS Quick-Complete for Dead Phone
// ═══════════════════════════════════════════════════════════════════════════════

// HandleSMSComplete processes a delivery completion triggered via SMS gateway webhook.
// POST /v1/delivery/sms-complete
// Body: {"driver_phone": "+998...", "order_id": "...", "message": "DONE abc123"}
func HandleSMSComplete(svc *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			DriverPhone string `json:"driver_phone"`
			OrderID     string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DriverPhone == "" || req.OrderID == "" {
			http.Error(w, `{"error":"driver_phone and order_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Verify driver identity by phone
		driverStmt := spanner.Statement{
			SQL:    `SELECT DriverId FROM Drivers WHERE Phone = @phone LIMIT 1`,
			Params: map[string]interface{}{"phone": req.DriverPhone},
		}
		driverIter := svc.Client.Single().Query(ctx, driverStmt)
		driverRow, err := driverIter.Next()
		if err != nil {
			driverIter.Stop()
			http.Error(w, `{"error":"driver not found"}`, http.StatusUnauthorized)
			return
		}
		var driverID string
		if err := driverRow.Columns(&driverID); err != nil {
			driverIter.Stop()
			http.Error(w, `{"error":"driver lookup failed"}`, http.StatusInternalServerError)
			return
		}
		driverIter.Stop()

		now := time.Now().UTC()
		_, err = svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "DriverId", "Version"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var state string
			var assignedDriver spanner.NullString
			var version int64
			if err := row.Columns(&state, &assignedDriver, &version); err != nil {
				return err
			}
			if state != "ARRIVED" && state != "ARRIVED_SHOP_CLOSED" {
				return fmt.Errorf("order must be ARRIVED or ARRIVED_SHOP_CLOSED, got %s", state)
			}
			if !assignedDriver.Valid || assignedDriver.StringVal != driverID {
				return fmt.Errorf("driver mismatch")
			}
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "State", "Version", "UpdatedAt"},
					[]interface{}{req.OrderID, "COMPLETED", version + 1, now}),
			})
		})
		if err != nil {
			log.Printf("[SMS_COMPLETE] Failed for order %s: %v", req.OrderID, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		svc.PublishEvent(ctx, kafkaEvents.EventSmsQuickComplete, map[string]string{
			"order_id":  req.OrderID,
			"driver_id": driverID,
		})
		writeOrderEvent(ctx, svc.Client, req.OrderID, driverID, "DRIVER", kafkaEvents.EventSmsQuickComplete, nil, 0, 0)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "completed",
			"order_id": req.OrderID,
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// OrderEvents Audit Writer (shared utility)
// ═══════════════════════════════════════════════════════════════════════════════

// writeOrderEvent inserts an audit event into the OrderEvents table.
// Fire-and-forget: errors are logged but do not interrupt the caller.
// Submitted to the global EventPool to prevent unbounded goroutine creation.
func writeOrderEvent(ctx context.Context, client *spanner.Client, orderID, actorID, actorRole, eventType string, metadata map[string]string, gpsLat, gpsLng float64) {
	eventID := hotspot.NewOpaqueID()
	var metaJSON []byte
	if metadata != nil {
		metaJSON, _ = json.Marshal(metadata)
	}

	workers.EventPool.Submit(func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := client.ReadWriteTransaction(bgCtx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			cols := []string{"EventId", "OrderId", "ActorId", "ActorRole", "EventType", "CreatedAt"}
			vals := []interface{}{eventID, orderID, actorID, actorRole, eventType, spanner.CommitTimestamp}

			if len(metaJSON) > 0 {
				cols = append(cols, "Metadata")
				vals = append(vals, string(metaJSON))
			}
			if gpsLat != 0 || gpsLng != 0 {
				cols = append(cols, "GPSLat", "GPSLng")
				vals = append(vals, gpsLat, gpsLng)
			}

			return txn.BufferWrite([]*spanner.Mutation{
				spanner.InsertOrUpdate("OrderEvents", cols, vals),
			})
		})
		if err != nil {
			log.Printf("[ORDER_EVENT] Failed to write %s for order %s: %v", eventType, orderID, err)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge 27: Early Route Complete (Driver Fatigue)
// ═══════════════════════════════════════════════════════════════════════════════

// EarlyCompleteDeps holds push function refs for early-complete notifications.
type EarlyCompleteDeps struct {
	SupplierPush func(supplierID string, payload interface{}) bool
	DriverPush   func(driverID string, payload interface{}) bool
	NotifyUser   func(ctx context.Context, userID, role string, title, body string, data map[string]string)
}

// HandleRequestEarlyComplete lets a driver request early route completion.
// POST /v1/fleet/route/request-early-complete (DRIVER role)
func HandleRequestEarlyComplete(svc *OrderService, deps *EarlyCompleteDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			Reason string `json:"reason"` // FATIGUE | TRAFFIC | VEHICLE_ISSUE | OTHER
			Note   string `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reason == "" {
			http.Error(w, `{"error":"reason required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Find driver's remaining undelivered orders
		stmt := spanner.Statement{
			SQL: `SELECT OrderId, RouteId, SupplierId FROM Orders
			      WHERE DriverId = @did AND State IN ('IN_TRANSIT', 'ARRIVING', 'ARRIVED', 'LOADED', 'DISPATCHED')
			      LIMIT 50`,
			Params: map[string]interface{}{"did": claims.UserID},
		}
		iter := svc.Client.Single().Query(ctx, stmt)
		defer iter.Stop()

		var orderIDs []string
		var routeID, supplierID string
		for {
			row, err := iter.Next()
			if err != nil {
				break
			}
			var oid string
			var rid, sid spanner.NullString
			if err := row.Columns(&oid, &rid, &sid); err == nil {
				orderIDs = append(orderIDs, oid)
				if rid.Valid && routeID == "" {
					routeID = rid.StringVal
				}
				if sid.Valid && supplierID == "" {
					supplierID = sid.StringVal
				}
			}
		}

		if len(orderIDs) == 0 {
			http.Error(w, `{"error":"no pending orders found"}`, http.StatusNotFound)
			return
		}

		// Store in Redis with 2h TTL
		if cache.Client != nil {
			data, _ := json.Marshal(map[string]interface{}{
				"order_ids":    orderIDs,
				"reason":       req.Reason,
				"note":         req.Note,
				"requested_at": time.Now().UTC(),
			})
			rCtx, rCancel := context.WithTimeout(ctx, 2*time.Second)
			cache.Client.Set(rCtx, cache.PrefixEarlyComplete+claims.UserID, string(data), cache.TTLEarlyComplete)
			rCancel()
		}

		// Write OrderEvent for each affected order
		for _, oid := range orderIDs {
			writeOrderEvent(ctx, svc.Client, oid, claims.UserID, "DRIVER", "EARLY_COMPLETE_REQUESTED",
				map[string]string{"reason": req.Reason, "note": req.Note}, 0, 0)
		}

		// Push to supplier
		if deps != nil && deps.SupplierPush != nil {
			deps.SupplierPush(supplierID, map[string]interface{}{
				"type":        ws.EventEarlyCompleteRequested,
				"driver_id":   claims.UserID,
				"route_id":    routeID,
				"order_count": len(orderIDs),
				"reason":      req.Reason,
				"note":        req.Note,
			})
		}
		if deps != nil && deps.NotifyUser != nil {
			go deps.NotifyUser(context.Background(), supplierID, "SUPPLIER",
				"Early Route Complete Request",
				fmt.Sprintf("Driver requests early stop: %s — %d orders affected", req.Reason, len(orderIDs)),
				map[string]string{"type": ws.EventEarlyCompleteRequested, "driver_id": claims.UserID})
		}

		go svc.PublishEvent(context.Background(), kafkaEvents.EventEarlyCompleteRequested, kafkaEvents.EarlyCompleteRequestedEvent{
			DriverID: claims.UserID, SupplierID: supplierID, RouteID: routeID,
			OrderIDs: orderIDs, Reason: req.Reason, Note: req.Note, Timestamp: time.Now().UTC(),
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "REQUESTED",
			"order_count": len(orderIDs),
			"order_ids":   orderIDs,
		})
	}
}

// HandleApproveEarlyComplete lets a supplier approve the early route completion.
// POST /v1/admin/route/approve-early-complete (ADMIN/SUPPLIER role)
func HandleApproveEarlyComplete(svc *OrderService, deps *EarlyCompleteDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			DriverID string `json:"driver_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DriverID == "" {
			http.Error(w, `{"error":"driver_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		// Read from Redis
		var orderIDs []string
		var reason, note string
		if cache.Client != nil {
			rCtx, rCancel := context.WithTimeout(ctx, 2*time.Second)
			data, err := cache.Client.Get(rCtx, cache.PrefixEarlyComplete+req.DriverID).Result()
			rCancel()
			if err != nil {
				http.Error(w, `{"error":"no pending early-complete request found"}`, http.StatusNotFound)
				return
			}
			var parsed struct {
				OrderIDs []string `json:"order_ids"`
				Reason   string   `json:"reason"`
				Note     string   `json:"note"`
			}
			if json.Unmarshal([]byte(data), &parsed) == nil {
				orderIDs = parsed.OrderIDs
				reason = parsed.Reason
				note = parsed.Note
			}
		}

		if len(orderIDs) == 0 {
			http.Error(w, `{"error":"no orders in early-complete request"}`, http.StatusNotFound)
			return
		}

		// Quarantine all remaining orders
		_, txnErr := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			for _, oid := range orderIDs {
				row, err := txn.ReadRow(ctx, "Orders", spanner.Key{oid}, []string{"State", "Version"})
				if err != nil {
					continue
				}
				var state string
				var version int64
				if err := row.Columns(&state, &version); err != nil {
					continue
				}
				// Only quarantine undelivered orders
				switch state {
				case "LOADED", "DISPATCHED", "IN_TRANSIT", "ARRIVING", "ARRIVED":
					txn.BufferWrite([]*spanner.Mutation{
						spanner.Update("Orders",
							[]string{"OrderId", "State", "Version", "EarlyCompleteReason", "EarlyCompleteNote"},
							[]interface{}{oid, "QUARANTINE", version + 1, reason, note}),
					})
				}
			}
			return nil
		})

		if txnErr != nil {
			log.Printf("[EARLY_COMPLETE] Quarantine failed: %v", txnErr)
			http.Error(w, `{"error":"quarantine failed"}`, http.StatusInternalServerError)
			return
		}

		// Delete Redis key
		if cache.Client != nil {
			rCtx, rCancel := context.WithTimeout(ctx, 2*time.Second)
			cache.Client.Del(rCtx, cache.PrefixEarlyComplete+req.DriverID)
			rCancel()
		}

		// Write OrderEvents
		for _, oid := range orderIDs {
			writeOrderEvent(ctx, svc.Client, oid, claims.UserID, claims.Role, "EARLY_COMPLETE_APPROVED",
				map[string]string{"reason": reason, "approved_by": claims.UserID}, 0, 0)
		}

		// Push to driver
		if deps != nil && deps.DriverPush != nil {
			deps.DriverPush(req.DriverID, map[string]interface{}{
				"type":    ws.EventEarlyCompleteApproved,
				"message": "Early route complete approved. Return to depot.",
			})
		}

		go svc.PublishEvent(context.Background(), kafkaEvents.EventEarlyCompleteApproved, kafkaEvents.EarlyCompleteRequestedEvent{
			DriverID: req.DriverID, SupplierID: claims.ResolveSupplierID(),
			OrderIDs: orderIDs, Reason: reason, Note: note, Timestamp: time.Now().UTC(),
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "APPROVED",
			"quarantined": len(orderIDs),
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge 32: Trust-Based Credit Delivery
// ═══════════════════════════════════════════════════════════════════════════════

// HandleCreditDelivery lets a driver mark an order as "Delivered on Credit".
// POST /v1/delivery/credit-delivery (DRIVER role)
func HandleCreditDelivery(svc *OrderService, deps *EarlyCompleteDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID       string `json:"order_id"`
			PhotoProofURL string `json:"photo_proof_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var supplierID, retailerID string

		_, err := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Read order
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "Version", "DriverId", "RetailerId", "SupplierId", "Amount"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var state string
			var version, amount int64
			var did, rid, sid spanner.NullString
			if err := row.Columns(&state, &version, &did, &rid, &sid); err != nil {
				return err
			}
			if state != "ARRIVED" {
				return fmt.Errorf("order must be ARRIVED for credit delivery (current: %s)", state)
			}
			if !did.Valid || did.StringVal != claims.UserID {
				return fmt.Errorf("driver mismatch")
			}
			if rid.Valid {
				retailerID = rid.StringVal
			}
			if sid.Valid {
				supplierID = sid.StringVal
			}

			// Check credit eligibility
			settingsRow, err := txn.ReadRow(ctx, "RetailerSupplierSettings",
				spanner.Key{retailerID, supplierID},
				[]string{"CreditEnabled", "CreditLimit", "CreditBalance"})
			if err != nil {
				return fmt.Errorf("credit not configured for this retailer-supplier pair")
			}
			var creditEnabled bool
			var creditLimit, creditBalance int64
			if err := settingsRow.Columns(&creditEnabled, &creditLimit, &creditBalance); err != nil {
				return err
			}
			if !creditEnabled {
				return fmt.Errorf("credit delivery is not enabled for this retailer")
			}
			if creditLimit > 0 && creditBalance+amount > creditLimit {
				return fmt.Errorf("credit limit exceeded: balance=%d + order=%d > limit=%d", creditBalance, amount, creditLimit)
			}

			// Transition order to DELIVERED_ON_CREDIT
			orderCols := []string{"OrderId", "State", "Version"}
			orderVals := []interface{}{req.OrderID, "DELIVERED_ON_CREDIT", version + 1}
			if req.PhotoProofURL != "" {
				orderCols = append(orderCols, "CreditPhotoProofUrl")
				orderVals = append(orderVals, req.PhotoProofURL)
			}
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders", orderCols, orderVals),
			})

			// Increment credit balance
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("RetailerSupplierSettings",
					[]string{"RetailerId", "SupplierId", "CreditBalance"},
					[]interface{}{retailerID, supplierID, creditBalance + amount}),
			})

			return nil
		})

		if err != nil {
			log.Printf("[CREDIT_DELIVERY] Failed: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		// Push to supplier
		if deps != nil && deps.SupplierPush != nil {
			deps.SupplierPush(supplierID, map[string]interface{}{
				"type":     ws.EventCreditDeliveryMarked,
				"order_id": req.OrderID,
			})
		}
		if deps != nil && deps.NotifyUser != nil {
			go deps.NotifyUser(context.Background(), supplierID, "SUPPLIER",
				"Credit Delivery",
				fmt.Sprintf("Order %s delivered on credit — approve or deny", req.OrderID[:8]),
				map[string]string{"type": ws.EventCreditDeliveryMarked, "order_id": req.OrderID})
		}

		go svc.PublishEvent(context.Background(), kafkaEvents.EventCreditDeliveryMarked, kafkaEvents.CreditDeliveryEvent{
			OrderID: req.OrderID, RetailerID: retailerID, SupplierID: supplierID,
			DriverID: claims.UserID, Timestamp: time.Now().UTC(),
		})
		writeOrderEvent(ctx, svc.Client, req.OrderID, claims.UserID, "DRIVER", "CREDIT_DELIVERY_MARKED",
			map[string]string{"photo_proof_url": req.PhotoProofURL}, 0, 0)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "DELIVERED_ON_CREDIT",
			"order_id": req.OrderID,
		})
	}
}

// HandleResolveCreditDelivery lets a supplier approve or deny a credit delivery.
// POST /v1/admin/orders/resolve-credit (ADMIN/SUPPLIER role)
func HandleResolveCreditDelivery(svc *OrderService, deps *EarlyCompleteDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID string `json:"order_id"`
			Action  string `json:"action"` // APPROVE | DENY
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" || (req.Action != "APPROVE" && req.Action != "DENY") {
			http.Error(w, `{"error":"order_id and action (APPROVE|DENY) required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var driverID string

		_, err := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "Version", "DriverId", "RetailerId", "SupplierId", "Amount"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var state string
			var version, amount int64
			var did, rid, sid spanner.NullString
			if err := row.Columns(&state, &version, &did, &rid, &sid); err != nil {
				return err
			}
			if state != "DELIVERED_ON_CREDIT" {
				return fmt.Errorf("order must be DELIVERED_ON_CREDIT (current: %s)", state)
			}
			if did.Valid {
				driverID = did.StringVal
			}

			if req.Action == "APPROVE" {
				txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("Orders",
						[]string{"OrderId", "State", "Version"},
						[]interface{}{req.OrderID, "COMPLETED", version + 1}),
				})
				// Create ledger entry for credit
				if rid.Valid && sid.Valid {
					txn.BufferWrite([]*spanner.Mutation{
						spanner.Insert("LedgerEntries",
							[]string{"TransactionId", "OrderId", "AccountId", "Amount", "EntryType", "CreatedAt"},
							[]interface{}{hotspot.NewOpaqueID(), req.OrderID, rid.StringVal, amount, "CREDIT_DELIVERY", spanner.CommitTimestamp}),
					})
				}
			} else {
				// DENY → QUARANTINE + decrement credit balance
				txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("Orders",
						[]string{"OrderId", "State", "Version"},
						[]interface{}{req.OrderID, "QUARANTINE", version + 1}),
				})
				if rid.Valid && sid.Valid {
					settingsRow, err := txn.ReadRow(ctx, "RetailerSupplierSettings",
						spanner.Key{rid.StringVal, sid.StringVal},
						[]string{"CreditBalance"})
					if err == nil {
						var balance int64
						if settingsRow.Columns(&balance) == nil {
							newBalance := balance - amount
							if newBalance < 0 {
								newBalance = 0
							}
							txn.BufferWrite([]*spanner.Mutation{
								spanner.Update("RetailerSupplierSettings",
									[]string{"RetailerId", "SupplierId", "CreditBalance"},
									[]interface{}{rid.StringVal, sid.StringVal, newBalance}),
							})
						}
					}
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("[CREDIT_RESOLVE] Failed: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		// Push to driver
		if deps != nil && deps.DriverPush != nil {
			deps.DriverPush(driverID, map[string]interface{}{
				"type":     ws.EventCreditDeliveryResolved,
				"order_id": req.OrderID,
				"action":   req.Action,
			})
		}

		go svc.PublishEvent(context.Background(), kafkaEvents.EventCreditDeliveryResolved, kafkaEvents.CreditDeliveryEvent{
			OrderID: req.OrderID, Action: req.Action, Timestamp: time.Now().UTC(),
		})
		writeOrderEvent(ctx, svc.Client, req.OrderID, claims.UserID, claims.Role, "CREDIT_DELIVERY_RESOLVED",
			map[string]string{"action": req.Action}, 0, 0)

		newState := "COMPLETED"
		if req.Action == "DENY" {
			newState = "QUARANTINE"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   newState,
			"order_id": req.OrderID,
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge 33: Missing Items After Seal
// ═══════════════════════════════════════════════════════════════════════════════

// HandleMissingItems lets a driver report items missing after manifest seal.
// POST /v1/delivery/missing-items (DRIVER role)
func HandleMissingItems(svc *OrderService, deps *EarlyCompleteDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID      string `json:"order_id"`
			MissingItems []struct {
				SkuID      string `json:"sku_id"`
				MissingQty int64  `json:"missing_qty"`
			} `json:"missing_items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" || len(req.MissingItems) == 0 {
			http.Error(w, `{"error":"order_id and missing_items required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		// Build AmendOrder request — reduce quantities by missing amounts
		amendItems := make([]AmendItemReq, len(req.MissingItems))
		for i, item := range req.MissingItems {
			amendItems[i] = AmendItemReq{
				ProductId:   item.SkuID,
				AcceptedQty: 0, // Will be calculated by AmendOrder from original - rejected
				RejectedQty: item.MissingQty,
				Reason:      "MISSING_FROM_SEAL",
			}
		}

		// We need original quantities to properly set AcceptedQty
		for i, item := range req.MissingItems {
			liStmt := spanner.Statement{
				SQL:    `SELECT Quantity FROM OrderLineItems WHERE OrderId = @oid AND SkuId = @skuId LIMIT 1`,
				Params: map[string]interface{}{"oid": req.OrderID, "skuId": item.SkuID},
			}
			iter := svc.Client.Single().Query(ctx, liStmt)
			row, err := iter.Next()
			iter.Stop()
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"line item not found for sku %s"}`, item.SkuID), http.StatusBadRequest)
				return
			}
			var origQty int64
			if err := row.Columns(&origQty); err != nil {
				http.Error(w, `{"error":"failed to read line item"}`, http.StatusInternalServerError)
				return
			}
			acceptedQty := origQty - item.MissingQty
			if acceptedQty < 0 {
				acceptedQty = 0
			}
			amendItems[i].AcceptedQty = acceptedQty
			amendItems[i].RejectedQty = origQty - acceptedQty
		}

		amendReq := AmendOrderRequest{
			OrderID: req.OrderID,
			Items:   amendItems,
		}
		resp, err := svc.AmendOrder(ctx, amendReq)
		if err != nil {
			log.Printf("[MISSING_ITEMS] AmendOrder failed: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		// If adjusted total is 0, all items missing → quarantine
		if resp != nil && resp.AdjustedTotal == 0 {
			svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				return txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("Orders",
						[]string{"OrderId", "State"},
						[]interface{}{req.OrderID, "QUARANTINE"}),
				})
			})
		}

		// Alert supplier
		if deps != nil && deps.SupplierPush != nil && resp != nil {
			deps.SupplierPush(resp.SupplierID, map[string]interface{}{
				"type":          ws.EventMissingItemsReported,
				"order_id":      req.OrderID,
				"driver_id":     claims.UserID,
				"missing_count": len(req.MissingItems),
			})
		}

		go svc.PublishEvent(context.Background(), kafkaEvents.EventMissingItemsReported, kafkaEvents.MissingItemsEvent{
			OrderID: req.OrderID, DriverID: claims.UserID, SupplierID: resp.SupplierID,
			ItemCount: len(req.MissingItems), Timestamp: time.Now().UTC(),
		})
		writeOrderEvent(ctx, svc.Client, req.OrderID, claims.UserID, "DRIVER", "MISSING_ITEMS_REPORTED",
			map[string]string{"item_count": fmt.Sprintf("%d", len(req.MissingItems))}, 0, 0)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "REPORTED",
			"order_id":       req.OrderID,
			"adjusted_total": resp.AdjustedTotal,
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge 35: Split Payment
// ═══════════════════════════════════════════════════════════════════════════════

// HandleSplitPayment lets a driver create a split payment (pay now + pay later).
// POST /v1/delivery/split-payment (DRIVER role)
func HandleSplitPayment(svc *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID      string `json:"order_id"`
			FirstAmount  int64  `json:"first_amount"`
			SecondAmount int64  `json:"second_amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Verify the order
		row, err := svc.Client.Single().ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
			[]string{"State", "Amount", "DriverId", "RetailerId", "SupplierId"})
		if err != nil {
			http.Error(w, `{"error":"order not found"}`, http.StatusNotFound)
			return
		}
		var state string
		var orderAmount int64
		var did, rid, sid spanner.NullString
		if err := row.Columns(&state, &orderAmount, &did, &rid, &sid); err != nil {
			http.Error(w, `{"error":"failed to read order"}`, http.StatusInternalServerError)
			return
		}
		if state != "AWAITING_PAYMENT" && state != "ARRIVED" {
			http.Error(w, fmt.Sprintf(`{"error":"order must be AWAITING_PAYMENT or ARRIVED (current: %s)"}`, state), http.StatusConflict)
			return
		}
		if !did.Valid || did.StringVal != claims.UserID {
			http.Error(w, `{"error":"driver mismatch"}`, http.StatusForbidden)
			return
		}
		if req.FirstAmount+req.SecondAmount != orderAmount {
			http.Error(w, fmt.Sprintf(`{"error":"amounts must sum to %d"}`, orderAmount), http.StatusBadRequest)
			return
		}
		if req.FirstAmount <= 0 || req.SecondAmount <= 0 {
			http.Error(w, `{"error":"both amounts must be positive"}`, http.StatusBadRequest)
			return
		}

		retailerID := ""
		supplierID := ""
		if rid.Valid {
			retailerID = rid.StringVal
		}
		if sid.Valid {
			supplierID = sid.StringVal
		}

		// Create two payment sessions via SessionService
		if svc.SessionSvc != nil {
			// Cancel any existing active sessions for this order
			cancelStmt := spanner.Statement{
				SQL: `UPDATE PaymentSessions SET Status = 'CANCELLED', UpdatedAt = PENDING_COMMIT_TIMESTAMP()
				      WHERE OrderId = @oid AND Status IN ('CREATED', 'PENDING')`,
				Params: map[string]interface{}{"oid": req.OrderID},
			}
			svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				_, err := txn.Update(ctx, cancelStmt)
				return err
			})

			// Create session 1 (immediate cash)
			session1ID := hotspot.NewOpaqueID()
			session2ID := hotspot.NewOpaqueID()
			now := time.Now().UTC()
			_, txnErr := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				txn.BufferWrite([]*spanner.Mutation{
					spanner.Insert("PaymentSessions",
						[]string{"SessionId", "OrderId", "RetailerId", "SupplierId", "Gateway", "LockedAmount", "Currency", "Status", "CreatedAt"},
						[]interface{}{session1ID, req.OrderID, retailerID, supplierID, "CASH", req.FirstAmount, "UZS", "CREATED", spanner.CommitTimestamp}),
					spanner.Insert("PaymentSessions",
						[]string{"SessionId", "OrderId", "RetailerId", "SupplierId", "Gateway", "LockedAmount", "Currency", "Status", "ExpiresAt", "CreatedAt"},
						[]interface{}{session2ID, req.OrderID, retailerID, supplierID, "CASH", req.SecondAmount, "UZS", "CREATED", now.Add(72 * time.Hour), spanner.CommitTimestamp}),
				})
				return nil
			})
			if txnErr != nil {
				log.Printf("[SPLIT_PAYMENT] Failed to create sessions: %v", txnErr)
				http.Error(w, `{"error":"failed to create payment sessions"}`, http.StatusInternalServerError)
				return
			}

			// Settle session 1 immediately (cash collected now)
			if svc.SessionSvc != nil {
				if settleErr := svc.SessionSvc.PartialSettleSession(ctx, session1ID, req.FirstAmount, "CASH_SPLIT"); settleErr != nil {
					log.Printf("[SPLIT_PAYMENT] Settle session 1 failed: %v", settleErr)
				}
			}
		}

		go svc.PublishEvent(context.Background(), kafkaEvents.EventSplitPaymentCreated, kafkaEvents.SplitPaymentEvent{
			OrderID: req.OrderID, DriverID: claims.UserID,
			FirstAmount: req.FirstAmount, SecondAmount: req.SecondAmount,
			Timestamp: time.Now().UTC(),
		})
		writeOrderEvent(ctx, svc.Client, req.OrderID, claims.UserID, "DRIVER", "SPLIT_PAYMENT_CREATED",
			map[string]string{
				"first_amount":  fmt.Sprintf("%d", req.FirstAmount),
				"second_amount": fmt.Sprintf("%d", req.SecondAmount),
			}, 0, 0)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":        "SPLIT_CREATED",
			"order_id":      req.OrderID,
			"first_amount":  req.FirstAmount,
			"second_amount": req.SecondAmount,
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge 34: AI Order Confirmation Gate
// ═══════════════════════════════════════════════════════════════════════════════

// HandleConfirmAiOrder lets a retailer confirm an AI-suggested order.
// POST /v1/retailer/orders/confirm-ai (RETAILER role)
func HandleConfirmAiOrder(svc *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		_, err := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "AiPendingConfirmation", "RetailerId", "Version"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var state string
			var pending spanner.NullBool
			var rid spanner.NullString
			var version int64
			if err := row.Columns(&state, &pending, &rid, &version); err != nil {
				return err
			}
			if !pending.Valid || !pending.Bool {
				return fmt.Errorf("order is not pending AI confirmation")
			}
			if state != "PENDING" {
				return fmt.Errorf("order must be PENDING (current: %s)", state)
			}
			if rid.Valid && rid.StringVal != claims.UserID {
				return fmt.Errorf("retailer mismatch")
			}

			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "AiPendingConfirmation", "Version"},
					[]interface{}{req.OrderID, false, version + 1}),
			})
			return nil
		})

		if err != nil {
			log.Printf("[AI_CONFIRM] Failed: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		go svc.PublishEvent(context.Background(), kafkaEvents.EventAiOrderConfirmed, kafkaEvents.AiOrderEvent{
			OrderID: req.OrderID, RetailerID: claims.UserID, Action: "CONFIRMED", Timestamp: time.Now().UTC(),
		})
		writeOrderEvent(ctx, svc.Client, req.OrderID, claims.UserID, "RETAILER", "AI_ORDER_CONFIRMED", nil, 0, 0)

		// Clear stale "Confirm your order" notifications — the retailer has already acted.
		go notifications.DeleteByCorrelationId(context.Background(), svc.Client, "ord_confirm_"+req.OrderID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "CONFIRMED",
			"order_id": req.OrderID,
		})
	}
}

// HandleRejectAiOrder lets a retailer reject an AI-suggested order.
// POST /v1/retailer/orders/reject-ai (RETAILER role)
func HandleRejectAiOrder(svc *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID string `json:"order_id"`
			Reason  string `json:"reason"` // DONT_NEED | WRONG_ITEMS | TOO_EXPENSIVE | MISTAKE | OTHER
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" || req.Reason == "" {
			http.Error(w, `{"error":"order_id and reason required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		_, err := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "AiPendingConfirmation", "RetailerId", "Version"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var state string
			var pending spanner.NullBool
			var rid spanner.NullString
			var version int64
			if err := row.Columns(&state, &pending, &rid, &version); err != nil {
				return err
			}
			if !pending.Valid || !pending.Bool {
				return fmt.Errorf("order is not pending AI confirmation")
			}
			if rid.Valid && rid.StringVal != claims.UserID {
				return fmt.Errorf("retailer mismatch")
			}

			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "State", "AiPendingConfirmation", "Version", "CancelReason"},
					[]interface{}{req.OrderID, "CANCELLED", false, version + 1, "AI_REJECTED:" + req.Reason}),
			})
			return nil
		})

		if err != nil {
			log.Printf("[AI_REJECT] Failed: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		go svc.PublishEvent(context.Background(), kafkaEvents.EventAiOrderRejected, kafkaEvents.AiOrderEvent{
			OrderID: req.OrderID, RetailerID: claims.UserID, Action: "REJECTED",
			Reason: req.Reason, Timestamp: time.Now().UTC(),
		})
		writeOrderEvent(ctx, svc.Client, req.OrderID, claims.UserID, "RETAILER", "AI_ORDER_REJECTED",
			map[string]string{"reason": req.Reason}, 0, 0)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "REJECTED",
			"order_id": req.OrderID,
		})
	}
}

// HandleEditPreorder lets a retailer edit a SCHEDULED preorder's delivery date
// or line items, as long as the delivery date is >= 5 calendar days away (Tashkent TZ).
// POST /v1/orders/{id}/edit-preorder
func HandleEditPreorder(svc *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID               string `json:"order_id"`
			RequestedDeliveryDate string `json:"requested_delivery_date,omitempty"`
			Items                 []struct {
				LineItemID string `json:"line_item_id"`
				Quantity   int64  `json:"quantity"`
			} `json:"items,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		now := time.Now().UTC()
		_, err := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "RetailerId", "Version", "RequestedDeliveryDate", "CancelLockedAt"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var state string
			var retailerID spanner.NullString
			var version int64
			var requestedDD spanner.NullTime
			var cancelLockedAt spanner.NullTime
			if err := row.Columns(&state, &retailerID, &version, &requestedDD, &cancelLockedAt); err != nil {
				return err
			}

			if state != "SCHEDULED" {
				return &ErrStateConflict{OrderID: req.OrderID, CurrentState: state, AttemptedOp: "edit-preorder (only SCHEDULED)"}
			}
			if cancelLockedAt.Valid {
				return &ErrStateConflict{OrderID: req.OrderID, CurrentState: state, AttemptedOp: "edit-preorder (cancel-locked)"}
			}

			// Edit gate: delivery date must be >= 5 calendar days away
			if requestedDD.Valid {
				nowTKT := proximity.TashkentNow()
				todayMidnight := time.Date(nowTKT.Year(), nowTKT.Month(), nowTKT.Day(), 0, 0, 0, 0, proximity.TashkentLocation)
				editCutoff := todayMidnight.AddDate(0, 0, 5)
				deliveryTKT := requestedDD.Time.In(proximity.TashkentLocation)
				if deliveryTKT.Before(editCutoff) {
					return &ErrStateConflict{OrderID: req.OrderID, CurrentState: state, AttemptedOp: "edit-preorder (delivery < 5 days away)"}
				}
			}

			cols := []string{"OrderId", "Version", "UpdatedAt"}
			vals := []interface{}{req.OrderID, version + 1, now}

			if req.RequestedDeliveryDate != "" {
				parsedTime, parseErr := time.Parse(time.RFC3339, req.RequestedDeliveryDate)
				if parseErr != nil {
					return fmt.Errorf("invalid requested_delivery_date: %w", parseErr)
				}
				// New date must also be >= 4 calendar days away to remain SCHEDULED
				nowTKT := proximity.TashkentNow()
				todayMidnight := time.Date(nowTKT.Year(), nowTKT.Month(), nowTKT.Day(), 0, 0, 0, 0, proximity.TashkentLocation)
				preorderCutoff := todayMidnight.AddDate(0, 0, 4)
				newDeliveryTKT := parsedTime.In(proximity.TashkentLocation)
				if newDeliveryTKT.Before(preorderCutoff) {
					return fmt.Errorf("new delivery date must be at least 4 calendar days away")
				}
				cols = append(cols, "RequestedDeliveryDate")
				vals = append(vals, spanner.NullTime{Time: parsedTime, Valid: true})
				// Reset notification timestamps since date changed
				cols = append(cols, "NudgeNotifiedAt", "ConfirmationNotifiedAt", "PreorderReminderSentAt")
				vals = append(vals, nil, nil, nil)
			}

			return txn.BufferWrite([]*spanner.Mutation{spanner.Update("Orders", cols, vals)})
		})
		if err != nil {
			log.Printf("[EDIT_PREORDER] Failed for order %s: %v", req.OrderID, err)
			var sc *ErrStateConflict
			if errors.As(err, &sc) {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, sc.Error()), http.StatusConflict)
			} else {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			}
			return
		}

		svc.PublishEvent(ctx, kafkaEvents.EventPreOrderEdited, map[string]string{
			"order_id":  req.OrderID,
			"edited_by": claims.UserID,
			"new_date":  req.RequestedDeliveryDate,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "edited",
			"order_id": req.OrderID,
		})
	}
}

// HandleConfirmPreorder lets a retailer explicitly confirm a SCHEDULED preorder.
// This removes the preorder from the pending-confirmation pipeline and keeps it
// scheduled until the auto-accept sweep promotes it.
// POST /v1/orders/{id}/confirm-preorder
func HandleConfirmPreorder(svc *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		_, err := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "RetailerId", "Version"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var state string
			var retailerID spanner.NullString
			var version int64
			if err := row.Columns(&state, &retailerID, &version); err != nil {
				return err
			}

			if state != "SCHEDULED" && state != "PENDING_REVIEW" {
				return &ErrStateConflict{OrderID: req.OrderID, CurrentState: state, AttemptedOp: "confirm-preorder"}
			}

			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "Version", "UpdatedAt"},
					[]interface{}{req.OrderID, version + 1, time.Now().UTC()}),
			})
		})
		if err != nil {
			log.Printf("[CONFIRM_PREORDER] Failed for order %s: %v", req.OrderID, err)
			var sc *ErrStateConflict
			if errors.As(err, &sc) {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, sc.Error()), http.StatusConflict)
			} else {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			}
			return
		}

		// Clear confirmation notifications on explicit confirm
		go notifications.DeleteByCorrelationId(context.Background(), svc.Client, "ord_confirm_"+req.OrderID)

		svc.PublishEvent(ctx, kafkaEvents.EventPreOrderConfirmed, map[string]string{
			"order_id":     req.OrderID,
			"confirmed_by": claims.UserID,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "confirmed",
			"order_id": req.OrderID,
		})
	}
}
