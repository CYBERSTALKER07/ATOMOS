package order

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/hotspot"
	kafkaEvents "backend-go/kafka"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
)

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE I: SHOP-CLOSED CONTACT PROTOCOL (P0)
// Full lifecycle: Driver reports → Retailer responds → Admin escalates → Resolve
// ═══════════════════════════════════════════════════════════════════════════════

// ShopClosedDeps holds injected services needed by the shop-closed protocol.
// Wired from main.go to keep handlers testable and avoid circular imports.
type ShopClosedDeps struct {
	RetailerPush   func(retailerID string, payload interface{}) bool
	DriverPush     func(driverID string, payload interface{}) bool
	AdminBroadcast func(payload interface{}) // Broadcast to all connected admin WS clients
	NotifyUser     func(ctx context.Context, userID, role string, title, body string, data map[string]string)
}

// ── WebSocket Payload Types ───────────────────────────────────────────────────

type ShopClosedAlertPayload struct {
	Type       string   `json:"type"` // "SHOP_CLOSED_ALERT"
	OrderID    string   `json:"order_id"`
	DriverName string   `json:"driver_name"`
	Options    []string `json:"options"` // ["OPEN_NOW","5_MIN","CALL_ME","CLOSED_TODAY"]
	AttemptID  string   `json:"attempt_id"`
}

type ShopClosedResponsePayload struct {
	Type      string `json:"type"` // "SHOP_CLOSED_RESPONSE"
	OrderID   string `json:"order_id"`
	Response  string `json:"response"` // OPEN_NOW | 5_MIN | CALL_ME | CLOSED_TODAY
	AttemptID string `json:"attempt_id"`
}

type ShopClosedEscalationPayload struct {
	Type         string `json:"type"` // "SHOP_CLOSED_ESCALATED"
	OrderID      string `json:"order_id"`
	AttemptID    string `json:"attempt_id"`
	RetailerName string `json:"retailer_name"`
	DriverName   string `json:"driver_name"`
	SupplierID   string `json:"supplier_id"`
}

type BypassTokenPayload struct {
	Type        string `json:"type"` // "BYPASS_TOKEN_ISSUED"
	OrderID     string `json:"order_id"`
	AttemptID   string `json:"attempt_id"`
	BypassToken string `json:"bypass_token"`
}

// ── Handler: POST /v1/delivery/shop-closed ────────────────────────────────────
// Driver reports shop is closed. Requires ARRIVED state.
func (s *OrderService) HandleReportShopClosed(deps *ShopClosedDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		driverID := claims.UserID

		var req struct {
			OrderID string `json:"order_id"`
			Reason  string `json:"reason"` // Optional: "POWER_OUTAGE" for 2h grace, empty for standard flow
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}
		isPowerOutage := req.Reason == "POWER_OUTAGE"

		ctx := r.Context()
		attemptID := hotspot.NewOpaqueID()
		var retailerID, supplierID, driverName, retailerName string
		var gpsLat, gpsLng float64

		_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Read order
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "Version", "RetailerId", "SupplierId", "DriverId"})
			if err != nil {
				return fmt.Errorf("order %s not found: %w", req.OrderID, err)
			}
			var state string
			var version int64
			var sid, did, rid spanner.NullString
			if err := row.Columns(&state, &version, &rid, &sid, &did); err != nil {
				return err
			}
			if rid.Valid {
				retailerID = rid.StringVal
			}
			if sid.Valid {
				supplierID = sid.StringVal
			}

			// Validate state: must be ARRIVED
			if state != "ARRIVED" {
				return fmt.Errorf("order must be ARRIVED to report shop closed (current: %s)", state)
			}

			// Validate driver ownership
			if !did.Valid || did.StringVal != driverID {
				return fmt.Errorf("driver %s is not assigned to order %s", driverID, req.OrderID)
			}

			// Get driver name + last GPS (from Drivers table)
			driverRow, err := txn.ReadRow(ctx, "Drivers", spanner.Key{driverID},
				[]string{"Name", "CurrentLocation"})
			if err == nil {
				var name spanner.NullString
				var loc spanner.NullString
				_ = driverRow.Columns(&name, &loc)
				if name.Valid {
					driverName = name.StringVal
				}
				// Parse GPS from CurrentLocation JSON if available
				if loc.Valid {
					var gps struct{ Lat, Lng float64 }
					if json.Unmarshal([]byte(loc.StringVal), &gps) == nil {
						gpsLat, gpsLng = gps.Lat, gps.Lng
					}
				}
			}

			// Get retailer name
			retRow, err := txn.ReadRow(ctx, "Retailers", spanner.Key{retailerID},
				[]string{"Name", "ShopName"})
			if err == nil {
				var name, shopName spanner.NullString
				_ = retRow.Columns(&name, &shopName)
				if shopName.Valid {
					retailerName = shopName.StringVal
				} else if name.Valid {
					retailerName = name.StringVal
				}
			}

			// Transition order to ARRIVED_SHOP_CLOSED
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "State", "Version"},
					[]interface{}{req.OrderID, "ARRIVED_SHOP_CLOSED", version + 1}),
			})

			// Create ShopClosedAttempt
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Insert("ShopClosedAttempts",
					[]string{"AttemptId", "OrderId", "DriverId", "RetailerId", "ReportedAt", "GPSLat", "GPSLng"},
					[]interface{}{attemptID, req.OrderID, driverID, retailerID, spanner.CommitTimestamp, gpsLat, gpsLng}),
			})

			// Write OrderEvent
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Insert("OrderEvents",
					[]string{"EventId", "OrderId", "ActorId", "ActorRole", "EventType", "GPSLat", "GPSLng", "CreatedAt"},
					[]interface{}{hotspot.NewOpaqueID(), req.OrderID, driverID, "DRIVER", "SHOP_CLOSED_REPORTED", gpsLat, gpsLng, spanner.CommitTimestamp}),
			})

			return nil
		})

		if err != nil {
			log.Printf("[ShopClosed] Failed to report: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		// Post-commit: Push to retailer via WebSocket
		if deps != nil && deps.RetailerPush != nil {
			deps.RetailerPush(retailerID, ShopClosedAlertPayload{
				Type:       "SHOP_CLOSED_ALERT",
				OrderID:    req.OrderID,
				DriverName: driverName,
				Options:    []string{"OPEN_NOW", "5_MIN", "CALL_ME", "CLOSED_TODAY"},
				AttemptID:  attemptID,
			})
		}

		// Notify retailer via FCM if not connected
		if deps != nil && deps.NotifyUser != nil {
			go deps.NotifyUser(context.Background(), retailerID, "RETAILER",
				"Driver at your location",
				fmt.Sprintf("%s reports your shop is closed. Please respond.", driverName),
				map[string]string{"type": ws.EventShopClosedAlert, "order_id": req.OrderID, "attempt_id": attemptID})
		}

		// Emit Kafka event
		if isPowerOutage {
			go s.PublishEvent(context.Background(), kafkaEvents.EventPowerOutageReported, kafkaEvents.ShopClosedEvent{
				OrderID: req.OrderID, DriverID: driverID, RetailerID: retailerID,
				SupplierID: supplierID, AttemptID: attemptID,
				GPSLat: gpsLat, GPSLng: gpsLng, Timestamp: time.Now().UTC(),
			})
			// Power outage: set 2h Redis TTL, skip escalation timer
			if cache.Client != nil {
				rCtx, rCancel := context.WithTimeout(context.Background(), 2*time.Second)
				cache.Client.Set(rCtx, cache.PrefixPowerOutage+req.OrderID, "1", cache.TTLPowerOutage)
				rCancel()
			}
			writeOrderEvent(context.Background(), s.Client, req.OrderID, driverID, "DRIVER", "POWER_OUTAGE_REPORTED", map[string]string{"attempt_id": attemptID}, gpsLat, gpsLng)
		} else {
			go s.PublishEvent(context.Background(), kafkaEvents.EventShopClosed, kafkaEvents.ShopClosedEvent{
				OrderID: req.OrderID, DriverID: driverID, RetailerID: retailerID,
				SupplierID: supplierID, AttemptID: attemptID,
				GPSLat: gpsLat, GPSLng: gpsLng, Timestamp: time.Now().UTC(),
			})

			// Start escalation timer in background (standard shop-closed only)
			go s.startEscalationTimer(context.Background(), attemptID, req.OrderID, supplierID, retailerName, driverName, deps)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":     "ARRIVED_SHOP_CLOSED",
			"attempt_id": attemptID,
		})
	}
}

// startEscalationTimer waits for ShopClosedEscalationMinutes then escalates
// if the retailer hasn't responded.
func (s *OrderService) startEscalationTimer(ctx context.Context, attemptID, orderID, supplierID, retailerName, driverName string, deps *ShopClosedDeps) {
	// Default 3 minutes — could read from config service later
	time.Sleep(3 * time.Minute)

	// Check if already resolved
	row, err := s.Client.Single().ReadRow(ctx, "ShopClosedAttempts",
		spanner.Key{attemptID}, []string{"RetailerResponse", "Resolution"})
	if err != nil {
		log.Printf("[ShopClosed] Escalation check failed for attempt %s: %v", attemptID, err)
		return
	}

	var response, resolution spanner.NullString
	_ = row.Columns(&response, &resolution)

	// If retailer responded with OPEN_NOW or already resolved, skip escalation
	if resolution.Valid {
		return
	}
	if response.Valid && response.StringVal == "OPEN_NOW" {
		return
	}

	// Escalate: find an admin for this supplier
	adminID := s.findSupplierAdmin(ctx, supplierID)

	_, err = s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("ShopClosedAttempts",
				[]string{"AttemptId", "EscalatedAt", "EscalatedTo"},
				[]interface{}{attemptID, spanner.CommitTimestamp, adminID}),
		})
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("OrderEvents",
				[]string{"EventId", "OrderId", "ActorId", "ActorRole", "EventType", "Metadata", "CreatedAt"},
				[]interface{}{hotspot.NewOpaqueID(), orderID, "SYSTEM", "SYSTEM", "ADMIN_ESCALATED",
					fmt.Sprintf(`{"attempt_id":"%s","escalated_to":"%s"}`, attemptID, adminID),
					spanner.CommitTimestamp}),
		})
		return nil
	})

	if err != nil {
		log.Printf("[ShopClosed] Failed to persist escalation for attempt %s: %v", attemptID, err)
		return
	}

	// Push escalation to admin portal via WS
	if deps != nil && deps.AdminBroadcast != nil {
		deps.AdminBroadcast(ShopClosedEscalationPayload{
			Type:         "SHOP_CLOSED_ESCALATED",
			OrderID:      orderID,
			AttemptID:    attemptID,
			RetailerName: retailerName,
			DriverName:   driverName,
			SupplierID:   supplierID,
		})
	}

	// Notify admin via FCM + Telegram
	if deps != nil && deps.NotifyUser != nil {
		go deps.NotifyUser(context.Background(), adminID, "SUPPLIER",
			"Shop Closed — Escalation",
			fmt.Sprintf("Shop %s closed. Driver %s on site. Order requires resolution.", retailerName, driverName),
			map[string]string{"type": ws.EventShopClosedEscalated, "order_id": orderID, "attempt_id": attemptID})
	}

	// Kafka
	go s.PublishEvent(context.Background(), kafkaEvents.EventShopClosedEscalated, kafkaEvents.ShopClosedEscalatedEvent{
		OrderID: orderID, AttemptID: attemptID, SupplierID: supplierID,
		EscalatedTo: adminID, Timestamp: time.Now().UTC(),
	})

	log.Printf("[ShopClosed] Escalated attempt %s to admin %s", attemptID, adminID)
}

func (s *OrderService) findSupplierAdmin(ctx context.Context, supplierID string) string {
	// Find GLOBAL_ADMIN for this supplier, fallback to any NODE_ADMIN
	stmt := spanner.Statement{
		SQL: `SELECT UserId FROM SupplierUsers
		      WHERE SupplierId = @sid AND IsActive = true
		      ORDER BY CASE WHEN SupplierRole = 'GLOBAL_ADMIN' THEN 0 ELSE 1 END
		      LIMIT 1`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return supplierID // fallback to supplier ID itself
	}
	var uid string
	_ = row.Columns(&uid)
	return uid
}

// ── Handler: POST /v1/retailer/shop-closed-response ───────────────────────────
// Retailer responds to shop-closed alert.
func (s *OrderService) HandleShopClosedResponse(deps *ShopClosedDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		retailerID := claims.UserID

		var req struct {
			OrderID  string `json:"order_id"`
			Response string `json:"response"` // OPEN_NOW | 5_MIN | CALL_ME | CLOSED_TODAY
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" || req.Response == "" {
			http.Error(w, `{"error":"order_id and response required"}`, http.StatusBadRequest)
			return
		}

		// Validate response value
		validResponses := map[string]bool{"OPEN_NOW": true, "5_MIN": true, "CALL_ME": true, "CLOSED_TODAY": true}
		if !validResponses[req.Response] {
			http.Error(w, `{"error":"invalid response value"}`, http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		var driverID, attemptID string
		newState := "ARRIVED_SHOP_CLOSED" // default — stays unless OPEN_NOW

		_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Find the active attempt for this order
			stmt := spanner.Statement{
				SQL: `SELECT AttemptId, DriverId FROM ShopClosedAttempts
				      WHERE OrderId = @oid AND RetailerId = @rid AND Resolution IS NULL
				      ORDER BY ReportedAt DESC LIMIT 1`,
				Params: map[string]interface{}{"oid": req.OrderID, "rid": retailerID},
			}
			iter := txn.Query(ctx, stmt)
			row, err := iter.Next()
			iter.Stop()
			if err != nil {
				return fmt.Errorf("no active shop-closed attempt for order %s", req.OrderID)
			}
			_ = row.Columns(&attemptID, &driverID)

			// Update attempt response
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("ShopClosedAttempts",
					[]string{"AttemptId", "RetailerResponse", "RetailerRespondedAt"},
					[]interface{}{attemptID, req.Response, spanner.CommitTimestamp}),
			})

			// If OPEN_NOW: transition back to ARRIVED
			if req.Response == "OPEN_NOW" {
				newState = "ARRIVED"
				orderRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
					[]string{"Version"})
				if err != nil {
					return err
				}
				var version int64
				_ = orderRow.Columns(&version)
				txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("Orders",
						[]string{"OrderId", "State", "Version"},
						[]interface{}{req.OrderID, "ARRIVED", version + 1}),
				})
				txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("ShopClosedAttempts",
						[]string{"AttemptId", "Resolution", "ResolvedAt"},
						[]interface{}{attemptID, "RETAILER_OPENED", spanner.CommitTimestamp}),
				})
			}

			// Write OrderEvent
			metadata, _ := json.Marshal(map[string]string{"response": req.Response, "attempt_id": attemptID})
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Insert("OrderEvents",
					[]string{"EventId", "OrderId", "ActorId", "ActorRole", "EventType", "Metadata", "CreatedAt"},
					[]interface{}{hotspot.NewOpaqueID(), req.OrderID, retailerID, "RETAILER", "RETAILER_RESPONDED",
						string(metadata), spanner.CommitTimestamp}),
			})

			return nil
		})

		if err != nil {
			log.Printf("[ShopClosed] Response failed: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		// Push response to driver via WebSocket
		if deps != nil && deps.DriverPush != nil && driverID != "" {
			deps.DriverPush(driverID, ShopClosedResponsePayload{
				Type:      "SHOP_CLOSED_RESPONSE",
				OrderID:   req.OrderID,
				Response:  req.Response,
				AttemptID: attemptID,
			})
		}

		// Kafka
		go s.PublishEvent(context.Background(), kafkaEvents.EventShopClosedResponse, kafkaEvents.ShopClosedResponseEvent{
			OrderID: req.OrderID, RetailerID: retailerID, AttemptID: attemptID,
			Response: req.Response, Timestamp: time.Now().UTC(),
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": newState})
	}
}

// ── Handler: POST /v1/admin/shop-closed/resolve ───────────────────────────────
// Admin resolves a shop-closed escalation: WAIT | BYPASS | RETURN_TO_DEPOT
func (s *OrderService) HandleResolveShopClosed(deps *ShopClosedDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		adminID := claims.UserID

		var req struct {
			AttemptID string `json:"attempt_id"`
			Action    string `json:"action"` // WAIT | BYPASS | RETURN_TO_DEPOT
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AttemptID == "" || req.Action == "" {
			http.Error(w, `{"error":"attempt_id and action required"}`, http.StatusBadRequest)
			return
		}

		validActions := map[string]bool{"WAIT": true, "BYPASS": true, "RETURN_TO_DEPOT": true}
		if !validActions[req.Action] {
			http.Error(w, `{"error":"invalid action, must be WAIT|BYPASS|RETURN_TO_DEPOT"}`, http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		var orderID, driverID, bypassToken string
		resolution := "WAITING"

		_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Read attempt
			row, err := txn.ReadRow(ctx, "ShopClosedAttempts", spanner.Key{req.AttemptID},
				[]string{"OrderId", "DriverId"})
			if err != nil {
				return fmt.Errorf("attempt %s not found", req.AttemptID)
			}
			_ = row.Columns(&orderID, &driverID)

			switch req.Action {
			case "WAIT":
				resolution = "WAITING"
				// No state change — just log the event

			case "BYPASS":
				resolution = "BYPASS_ISSUED"
				bypassToken = generateBypassToken()
				txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("ShopClosedAttempts",
						[]string{"AttemptId", "Resolution", "BypassToken", "ResolvedAt", "ResolvedBy"},
						[]interface{}{req.AttemptID, resolution, bypassToken, spanner.CommitTimestamp, adminID}),
				})

			case "RETURN_TO_DEPOT":
				resolution = "RETURN_TO_DEPOT"
				// Transition order to QUARANTINE
				orderRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID},
					[]string{"Version"})
				if err != nil {
					return err
				}
				var version int64
				_ = orderRow.Columns(&version)
				txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("Orders",
						[]string{"OrderId", "State", "Version"},
						[]interface{}{orderID, "QUARANTINE", version + 1}),
				})
				txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("ShopClosedAttempts",
						[]string{"AttemptId", "Resolution", "ResolvedAt", "ResolvedBy"},
						[]interface{}{req.AttemptID, resolution, spanner.CommitTimestamp, adminID}),
				})
			}

			// Write OrderEvent
			metadata, _ := json.Marshal(map[string]string{
				"action": req.Action, "attempt_id": req.AttemptID,
				"bypass_token": bypassToken, "resolution": resolution,
			})
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Insert("OrderEvents",
					[]string{"EventId", "OrderId", "ActorId", "ActorRole", "EventType", "Metadata", "CreatedAt"},
					[]interface{}{hotspot.NewOpaqueID(), orderID, adminID, "SUPPLIER", resolution,
						string(metadata), spanner.CommitTimestamp}),
			})

			return nil
		})

		if err != nil {
			log.Printf("[ShopClosed] Resolve failed: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		// Push bypass token to driver if issued
		if bypassToken != "" && deps != nil && deps.DriverPush != nil {
			deps.DriverPush(driverID, BypassTokenPayload{
				Type:        "BYPASS_TOKEN_ISSUED",
				OrderID:     orderID,
				AttemptID:   req.AttemptID,
				BypassToken: bypassToken,
			})
			// Also notify via FCM
			if deps.NotifyUser != nil {
				go deps.NotifyUser(context.Background(), driverID, "DRIVER",
					"Bypass Token Issued",
					fmt.Sprintf("Use code %s to complete offload for this order.", bypassToken),
					map[string]string{"type": ws.EventBypassTokenIssued, "order_id": orderID, "bypass_token": bypassToken})
			}
		}

		// Kafka
		go s.PublishEvent(context.Background(), kafkaEvents.EventShopClosedResolved, kafkaEvents.ShopClosedResolvedEvent{
			OrderID: orderID, AttemptID: req.AttemptID,
			Resolution: resolution, ResolvedBy: adminID, Timestamp: time.Now().UTC(),
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":       resolution,
			"bypass_token": bypassToken,
		})
	}
}

// ── Handler: POST /v1/delivery/bypass-offload ─────────────────────────────────
// Driver confirms offload using a bypass token instead of retailer QR.
func (s *OrderService) HandleBypassOffload(deps *ShopClosedDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		driverID := claims.UserID

		var req struct {
			OrderID     string `json:"order_id"`
			BypassToken string `json:"bypass_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" || req.BypassToken == "" {
			http.Error(w, `{"error":"order_id and bypass_token required"}`, http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		var supplierID string

		_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Read order
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "Version", "DriverId", "SupplierId"})
			if err != nil {
				return fmt.Errorf("order %s not found", req.OrderID)
			}
			var state string
			var version int64
			var did, sid spanner.NullString
			if err := row.Columns(&state, &version, &did, &sid); err != nil {
				return err
			}
			if sid.Valid {
				supplierID = sid.StringVal
			}

			if state != "ARRIVED_SHOP_CLOSED" {
				return fmt.Errorf("order must be ARRIVED_SHOP_CLOSED to use bypass (current: %s)", state)
			}
			if !did.Valid || did.StringVal != driverID {
				return fmt.Errorf("driver %s is not assigned to order %s", driverID, req.OrderID)
			}

			// Validate bypass token against the latest attempt
			stmt := spanner.Statement{
				SQL: `SELECT AttemptId, BypassToken FROM ShopClosedAttempts
				      WHERE OrderId = @oid AND Resolution = 'BYPASS_ISSUED'
				      ORDER BY ReportedAt DESC LIMIT 1`,
				Params: map[string]interface{}{"oid": req.OrderID},
			}
			iter := txn.Query(ctx, stmt)
			tokenRow, err := iter.Next()
			iter.Stop()
			if err != nil {
				return fmt.Errorf("no bypass token issued for order %s", req.OrderID)
			}
			var attemptID string
			var storedToken spanner.NullString
			_ = tokenRow.Columns(&attemptID, &storedToken)

			if !storedToken.Valid || storedToken.StringVal != req.BypassToken {
				return fmt.Errorf("invalid bypass token")
			}

			// Transition to AWAITING_PAYMENT (same as normal offload)
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("Orders",
					[]string{"OrderId", "State", "Version"},
					[]interface{}{req.OrderID, "AWAITING_PAYMENT", version + 1}),
			})

			// Mark attempt as fully resolved
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("ShopClosedAttempts",
					[]string{"AttemptId", "ResolvedAt"},
					[]interface{}{attemptID, spanner.CommitTimestamp}),
			})

			// Write OrderEvent
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Insert("OrderEvents",
					[]string{"EventId", "OrderId", "ActorId", "ActorRole", "EventType", "Metadata", "CreatedAt"},
					[]interface{}{hotspot.NewOpaqueID(), req.OrderID, driverID, "DRIVER", "BYPASS_USED",
						fmt.Sprintf(`{"attempt_id":"%s"}`, attemptID), spanner.CommitTimestamp}),
			})

			return nil
		})

		if err != nil {
			log.Printf("[ShopClosed] Bypass offload failed: %v", err)
			status := http.StatusConflict
			if fmt.Sprintf("%v", err) == "invalid bypass token" {
				status = http.StatusForbidden
			}
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), status)
			return
		}

		// Emit Kafka status change
		go s.PublishEvent(context.Background(), kafkaEvents.EventOrderStatusChanged, kafkaEvents.OrderStatusChangedEvent{
			OrderID: req.OrderID, SupplierID: supplierID,
			OldState: "ARRIVED_SHOP_CLOSED", NewState: "AWAITING_PAYMENT",
			Timestamp: time.Now().UTC(),
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "AWAITING_PAYMENT"})
	}
}

// generateBypassToken creates a cryptographically random 6-digit numeric token.
func generateBypassToken() string {
	max := big.NewInt(999999)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "000000" // fallback — should never happen
	}
	return fmt.Sprintf("%06d", n.Int64())
}
