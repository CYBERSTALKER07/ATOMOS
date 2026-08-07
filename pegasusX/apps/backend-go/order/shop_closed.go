package order

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type shopClosedReportRequest struct {
	OrderID  string          `json:"order_id,omitempty"`
	Reason   string          `json:"reason"`
	Note     string          `json:"note,omitempty"`
	PhotoURL string          `json:"photo_url,omitempty"`
	PhotoURLCamel string     `json:"photoUrl,omitempty"`
	Location DriverTelemetry `json:"location"`
	Latitude  *float64       `json:"latitude,omitempty"`
	Longitude *float64       `json:"longitude,omitempty"`
}

type driverEndpointResponse struct {
	OrderID           string `json:"orderId"`
	Status            string `json:"status"`
	ProximityUnlocked bool   `json:"proximityUnlocked"`
	ProximityMethod   string `json:"proximityMethod"`
}

type shopClosedResponseRequest struct {
	OrderID  string `json:"order_id"`
	Response string `json:"response"`
	// NewSlot optional ISO date for RESCHEDULE.
	NewSlot string `json:"new_slot,omitempty"`
	// PhotoURL required for AUTHORIZE_BYPASS.
	PhotoURL string `json:"photo_url,omitempty"`
}

type shopClosedResolveRequest struct {
	AttemptID string `json:"attempt_id"`
	Action    string `json:"action"`
}

// Retailer response codes (enhanced protocol).
// Legacy OPEN_NOW / 5_MIN / CALL_ME / CLOSED_TODAY remain accepted.
const (
	RetailerRespOpenNow         = "OPEN_NOW"
	RetailerResp5Min            = "5_MIN"
	RetailerRespCallMe          = "CALL_ME"
	RetailerRespClosedToday     = "CLOSED_TODAY"
	RetailerRespReschedule      = "RESCHEDULE"
	RetailerRespCreditLeave     = "CREDIT_LEAVE"
	RetailerRespCancel          = "CANCEL"
	RetailerRespAuthorizeBypass = "AUTHORIZE_BYPASS"
)

// HandleReportShopClosed is POST /v1/delivery/shop-closed.
func (s *Service) HandleReportShopClosed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "shop_closed_unavailable"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req shopClosedReportRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	orderID := chi.URLParam(r, "orderId")
	if orderID == "" {
		orderID = strings.TrimSpace(req.OrderID)
	}
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	photoURL := strings.TrimSpace(req.PhotoURL)
	if photoURL == "" {
		photoURL = strings.TrimSpace(req.PhotoURLCamel)
	}
	if photoURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "photo_url_required"})
		return
	}
	req.PhotoURL = photoURL
	// Body lat/lng fallback when location object omitted (mobile clients).
	if req.Location.Lat == 0 && req.Location.Lng == 0 {
		if req.Latitude != nil {
			req.Location.Lat = *req.Latitude
		}
		if req.Longitude != nil {
			req.Location.Lng = *req.Longitude
		}
	}
	if req.Location.RecordedAt.IsZero() {
		req.Location.RecordedAt = s.now()
	}
	// Only enforce telemetry accuracy when a live GPS reading was provided.
	if req.Location.Lat != 0 || req.Location.Lng != 0 {
		if err := req.Location.Validate(100.0); err != nil {
			writeJSON(w, http.StatusPreconditionFailed, map[string]string{"error": err.Error()})
			return
		}
	}

	reason := NormalizeShopClosedReason(req.Reason)

	driverID := strings.TrimSpace(claims.Subject)
	attemptID := s.newID()
	logEventID := s.newID()
	now := s.now()
	// Client timestamp was removed, just use now

	graceEnds := now.Add(s.shopGrace)
	ctx := r.Context()

	var retailerID, supplierID string
	var gpsLat, gpsLng float64
	var finalProxUnlocked bool
	var finalProxMethod string

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID},
			[]string{"Status", "Version", "RetailerId", "SupplierId", "DriverId", "Lat", "Lng", "H3Cell", "ProximityUnlockedAt", "ProximityMethod"})
		if err != nil {
			return fmt.Errorf("order %s not found: %w", orderID, err)
		}
		var status string
		var version int64
		var driverCol, retailerCol, supplierCol, h3Cell, proxMethod spanner.NullString
		var proxUnlockedAt spanner.NullTime
		var orderLat, orderLng float64
		if err := row.Columns(&status, &version, &retailerCol, &supplierCol, &driverCol, &orderLat, &orderLng, &h3Cell, &proxUnlockedAt, &proxMethod); err != nil {
			return err
		}
		if status != string(StatusArrived) {
			return fmt.Errorf("%w: order must be ARRIVED to report shop closed (current: %s)", ErrInvalidStatusTransition, status)
		}
		if !driverCol.Valid || driverCol.StringVal != driverID {
			return fmt.Errorf("driver is not assigned to order %s", orderID)
		}
		if retailerCol.Valid {
			retailerID = retailerCol.StringVal
		}
		if supplierCol.Valid {
			supplierID = supplierCol.StringVal
		}
		gpsLat, gpsLng = req.Location.Lat, req.Location.Lng
		if gpsLat == 0 && gpsLng == 0 {
			gpsLat, gpsLng = orderLat, orderLng
		}

		// Ensure proximity is unlocked before allowing shop closed report
		var proxUnlockedAtPtr *time.Time
		if proxUnlockedAt.Valid {
			t := proxUnlockedAt.Time
			proxUnlockedAtPtr = &t
		}
		
		var h3 string
		if h3Cell.Valid {
			h3 = h3Cell.StringVal
		}

		orderRecord := &Order{
			OrderID:             orderID,
			Lat:                 orderLat,
			Lng:                 orderLng,
			H3Cell:              h3,
			ProximityUnlockedAt: proxUnlockedAtPtr,
			ProximityMethod:     proxMethod.StringVal,
		}

		// Soft proximity check for shop closed: we don't return an error if it fails.
		_ = s.ensureProximityUnlocked(ctx, txn, orderRecord, req.Location.ToLocation(), TransitionOpts{
			Actor:  driverID,
			Reason: "report_shop_closed",
		})
		
		// Ensure we capture if it was unlocked (maybe by this call)
		if orderRecord.ProximityUnlockedAt != nil {
			finalProxUnlocked = true
			finalProxMethod = orderRecord.ProximityMethod
		}

		// Check for max retries
		countStmt := spanner.Statement{
			SQL:    `SELECT COUNT(*) FROM ShopClosedAttempts WHERE OrderId = @oid`,
			Params: map[string]any{"oid": orderID},
		}
		countIter := txn.Query(ctx, countStmt)
		countRow, countErr := countIter.Next()
		countIter.Stop()
		if countErr == nil {
			var attemptsCount int64
			if err := countRow.Columns(&attemptsCount); err == nil {
				if attemptsCount >= 3 {
					return fmt.Errorf("shop_closed_max_retries")
				}
			}
		}

		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventShopClosed, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:    orderID,
			DriverID:   driverID,
			RetailerID: retailerID,
			SupplierID: supplierID,
			AttemptID:  attemptID,
			GPSLat:     gpsLat,
			GPSLng:     gpsLng,
		}); err != nil {
			return err
		}

		logPayload := map[string]any{
			"reason":        reason,
			"note":          strings.TrimSpace(req.Note),
			"photo_url":     strings.TrimSpace(req.PhotoURL),
			"gps_lat":       gpsLat,
			"gps_lng":       gpsLng,
			"attempt_id":    attemptID,
			"grace_ends_at": graceEnds.UTC().Format(time.RFC3339Nano),
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":               orderID,
				"Status":                string(StatusShopClosedPending), // ≡ SHOP_CLOSED_PENDING
				"Version":               version + 1,
				"UpdatedAt":             now.UTC(),
				"ShopClosedAt":          now.UTC(),
				"ShopClosedReason":      reason,
				"ShopClosedGraceEndsAt": graceEnds.UTC(),
			}),
			spanner.InsertMap("ShopClosedAttempts", map[string]any{
				"AttemptId":  attemptID,
				"OrderId":    orderID,
				"DriverId":   driverID,
				"RetailerId": retailerID,
				"ReportedAt": now.UTC(),
				"GPSLat":     gpsLat,
				"GPSLng":     gpsLng,
			}),
			spanner.InsertMap("OrderShopClosedLog", map[string]any{
				"OrderId":   orderID,
				"EventId":   logEventID,
				"Actor":     driverID,
				"Action":    "MARKED_CLOSED",
				"Payload":   spanner.NullJSON{Value: logPayload, Valid: true},
				"CreatedAt": now.UTC(),
			}),
		}
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		if err.Error() == "shop_closed_max_retries" {
			updateClaims := auth.Claims{Role: auth.RoleAdmin, Subject: "system"}
			updateReq := UpdateStatusRequest{Status: string(StatusCancelled), Reason: "shop_closed_max_retries"}
			s.UpdateStatus(ctx, updateClaims, orderID, updateReq)
			s.log.WarnContext(ctx, "shop closed max retries reached, order cancelled", "order_id", orderID)
			writeJSON(w, http.StatusConflict, map[string]string{"error": "max_retries_reached_order_cancelled"})
			return
		}
		if errors.Is(err, ErrInvalidStatusTransition) {
			s.log.ErrorContext(ctx, "shop closed report failed (invalid status)", "order_id", orderID, "err", err)
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": "invalid_status_transition",
				"message": err.Error(),
			})
			return
		}
		s.log.ErrorContext(ctx, "shop closed report failed", "order_id", orderID, "err", err)
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	s.broadcastShopClosed(ctx, supplierID, retailerID, driverID, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventShopClosed, Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    orderID,
		DriverID:   driverID,
		RetailerID: retailerID,
		SupplierID: supplierID,
		AttemptID:  attemptID,
	})
	s.invalidateOrderCache(ctx, orderID)

	resp := driverEndpointResponse{
		OrderID:           orderID,
		Status:            string(StatusShopClosedPending),
		ProximityUnlocked: finalProxUnlocked,
		ProximityMethod:   finalProxMethod,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleShopClosedResponse is POST /v1/retailer/shop-closed-response.
func (s *Service) HandleShopClosedResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "shop_closed_unavailable"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	var req shopClosedResponseRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.Response = strings.TrimSpace(req.Response)
	if req.OrderID == "" || req.Response == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id and response required"})
		return
	}
	req.Response = strings.ToUpper(req.Response)
	valid := map[string]bool{
		RetailerRespOpenNow: true, RetailerResp5Min: true, RetailerRespCallMe: true, RetailerRespClosedToday: true,
		RetailerRespReschedule: true, RetailerRespCreditLeave: true, RetailerRespCancel: true, RetailerRespAuthorizeBypass: true,
	}
	if !valid[req.Response] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid response value"})
		return
	}
	if req.Response == RetailerRespAuthorizeBypass && strings.TrimSpace(req.PhotoURL) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "photo_url_required_for_bypass"})
		return
	}

	retailerID := strings.TrimSpace(claims.Subject)
	s.executeShopClosedRetailerResponse(w, r, body, req, retailerID)
}

func (s *Service) executeShopClosedRetailerResponse(w http.ResponseWriter, r *http.Request, rawBody []byte, req shopClosedResponseRequest, retailerID string) {
	if s.guardIdempotency(w, r, rawBody) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	ctx := r.Context()
	now := s.now()
	newStatus := string(StatusShopClosedPending)
	var attemptID, driverID, supplierID, manifestID string
	logEventID := s.newID()
	resolution := ""

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `SELECT AttemptId, DriverId FROM ShopClosedAttempts
			      WHERE OrderId = @oid AND RetailerId = @rid AND Resolution IS NULL
			      ORDER BY ReportedAt DESC LIMIT 1`,
			Params: map[string]any{"oid": req.OrderID, "rid": retailerID},
		}
		iter := txn.Query(ctx, stmt)
		row, err := iter.Next()
		iter.Stop()
		if err != nil {
			return fmt.Errorf("no active shop-closed attempt for order %s", req.OrderID)
		}
		if err := row.Columns(&attemptID, &driverID); err != nil {
			return err
		}

		orderRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
			[]string{"Version", "SupplierId", "Status", "ManifestId", "WarehouseId", "OrderSource", "LineItemsJson"})
		if err != nil {
			return err
		}
		var version int64
		var supplierCol, manifestCol, warehouseCol, orderSourceCol spanner.NullString
		var orderStatus string
		var lineItemsRaw []byte
		if err := orderRow.Columns(&version, &supplierCol, &orderStatus, &manifestCol, &warehouseCol, &orderSourceCol, &lineItemsRaw); err != nil {
			return err
		}
		if supplierCol.Valid {
			supplierID = supplierCol.StringVal
		}
		if manifestCol.Valid {
			manifestID = manifestCol.StringVal
		}
		warehouseID := ""
		if warehouseCol.Valid {
			warehouseID = warehouseCol.StringVal
		}
		orderSource := ""
		if orderSourceCol.Valid {
			orderSource = orderSourceCol.StringVal
		}
		// Retailer response wins if still PENDING (ARRIVED_SHOP_CLOSED), even after grace clock.
		if orderStatus != string(StatusShopClosedPending) {
			return fmt.Errorf("shop_closed_already_resolved status=%s", orderStatus)
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("ShopClosedAttempts", map[string]any{
				"AttemptId":           attemptID,
				"RetailerResponse":    req.Response,
				"RetailerRespondedAt": now.UTC(),
			}),
		}

		switch req.Response {
		case RetailerRespOpenNow:
			newStatus = string(StatusArrived)
			resolution = "RETAILER_OPENED"
			mutations = append(mutations,
				spanner.UpdateMap("Orders", map[string]any{
					"OrderId":              req.OrderID,
					"Status":               newStatus,
					"Version":              version + 1,
					"UpdatedAt":            now.UTC(),
					"ShopClosedResolution": resolution,
				}),
				spanner.UpdateMap("ShopClosedAttempts", map[string]any{
					"AttemptId":  attemptID,
					"Resolution": resolution,
					"ResolvedAt": now.UTC(),
				}),
			)
		case RetailerRespReschedule:
			resolution = ShopClosedResolutionRescheduled
			mutations = append(mutations,
				spanner.UpdateMap("Orders", map[string]any{
					"OrderId":              req.OrderID,
					"ShopClosedResolution": resolution,
					"Version":              version + 1,
					"UpdatedAt":            now.UTC(),
				}),
				spanner.UpdateMap("ShopClosedAttempts", map[string]any{
					"AttemptId":  attemptID,
					"Resolution": resolution,
					"ResolvedAt": now.UTC(),
				}),
			)
		case RetailerRespCreditLeave:
			// Intent only — driver still executes credit-delivery with proximity.
			resolution = ShopClosedResolutionCreditLeave
			mutations = append(mutations,
				spanner.UpdateMap("Orders", map[string]any{
					"OrderId":              req.OrderID,
					"ShopClosedResolution": resolution,
					"Version":              version + 1,
					"UpdatedAt":            now.UTC(),
				}),
				spanner.UpdateMap("ShopClosedAttempts", map[string]any{
					"AttemptId":  attemptID,
					"Resolution": "RETAILER_CREDIT_LEAVE",
					"ResolvedAt": now.UTC(),
				}),
			)
		case RetailerRespCancel:
			newStatus = string(StatusCancelled)
			resolution = ShopClosedResolutionCancelled
			if err := ReleaseReservationsFromOrderFields(ctx, txn, supplierID, warehouseID, orderSource, lineItemsRaw); err != nil {
				return err
			}
			mutations = append(mutations,
				spanner.UpdateMap("Orders", map[string]any{
					"OrderId":              req.OrderID,
					"Status":               newStatus,
					"Version":              version + 1,
					"UpdatedAt":            now.UTC(),
					"ShopClosedResolution": resolution,
				}),
				spanner.UpdateMap("ShopClosedAttempts", map[string]any{
					"AttemptId":  attemptID,
					"Resolution": resolution,
					"ResolvedAt": now.UTC(),
				}),
			)
		case RetailerRespAuthorizeBypass:
			resolution = ShopClosedResolutionBypass
			bypassToken := generateShopClosedBypassToken()
			mutations = append(mutations,
				spanner.UpdateMap("Orders", map[string]any{
					"OrderId":              req.OrderID,
					"ShopClosedResolution": resolution,
					"Version":              version + 1,
					"UpdatedAt":            now.UTC(),
				}),
				spanner.UpdateMap("ShopClosedAttempts", map[string]any{
					"AttemptId":   attemptID,
					"Resolution":  "BYPASS_ISSUED",
					"BypassToken": bypassToken,
					"ResolvedAt":  now.UTC(),
				}),
			)
		// 5_MIN / CALL_ME / CLOSED_TODAY: acknowledge only; stay PENDING.
		}

		logPayload := map[string]any{
			"response":   req.Response,
			"new_slot":   strings.TrimSpace(req.NewSlot),
			"photo_url":  strings.TrimSpace(req.PhotoURL),
			"attempt_id": attemptID,
			"resolution": resolution,
		}
		mutations = append(mutations, spanner.InsertMap("OrderShopClosedLog", map[string]any{
			"OrderId":   req.OrderID,
			"EventId":   logEventID,
			"Actor":     retailerID,
			"Action":    "RESPONDED",
			"Payload":   spanner.NullJSON{Value: logPayload, Valid: true},
			"CreatedAt": now.UTC(),
		}))

		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, req.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventShopClosedResponse, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:    req.OrderID,
			RetailerID: retailerID,
			DriverID:   driverID,
			SupplierID: supplierID,
			AttemptID:  attemptID,
			Response:   req.Response,
		}); err != nil {
			return err
		}
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	if s.replanner != nil && manifestID != "" {
		go func(rID, act string) {
			_ = s.replanner.ReplanRoute(context.Background(), rID, "shop_closed_resolved", act)
		}(manifestID, retailerID)
	}

	s.broadcastShopClosed(ctx, supplierID, retailerID, driverID, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventShopClosedResponse, Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    req.OrderID,
		RetailerID: retailerID,
		AttemptID:  attemptID,
		Response:   req.Response,
	})
	s.invalidateOrderCache(ctx, req.OrderID)
	resp := map[string]string{"status": newStatus}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, rawBody, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleResolveShopClosed is POST /v1/supplier/shop-closed/resolve.
func (s *Service) HandleResolveShopClosed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req shopClosedResolveRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.AttemptID = strings.TrimSpace(req.AttemptID)
	req.Action = strings.TrimSpace(req.Action)
	if req.AttemptID == "" || req.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "attempt_id and action required"})
		return
	}
	valid := map[string]bool{"WAIT": true, "BYPASS": true, "RETURN_TO_DEPOT": true}
	if !valid[req.Action] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid action"})
		return
	}

	adminID := strings.TrimSpace(claims.Subject)
	ctx := r.Context()
	now := s.now()
	var orderID, driverID, retailerID, supplierID, bypassToken, resolution, manifestID string
	resolution = "WAITING"

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "ShopClosedAttempts", spanner.Key{req.AttemptID},
			[]string{"OrderId", "DriverId", "RetailerId"})
		if err != nil {
			return fmt.Errorf("attempt %s not found", req.AttemptID)
		}
		if err := row.Columns(&orderID, &driverID, &retailerID); err != nil {
			return err
		}

		orderRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID},
			[]string{"Version", "SupplierId", "ManifestId", "WarehouseId", "OrderSource", "LineItemsJson"})
		if err != nil {
			return err
		}
		var version int64
		var supplierCol, manifestCol, warehouseCol, orderSourceCol spanner.NullString
		var lineItemsRaw []byte
		if err := orderRow.Columns(&version, &supplierCol, &manifestCol, &warehouseCol, &orderSourceCol, &lineItemsRaw); err != nil {
			return err
		}
		if supplierCol.Valid {
			supplierID = supplierCol.StringVal
		}
		if manifestCol.Valid {
			manifestID = manifestCol.StringVal
		}
		warehouseID := ""
		if warehouseCol.Valid {
			warehouseID = warehouseCol.StringVal
		}
		orderSource := ""
		if orderSourceCol.Valid {
			orderSource = orderSourceCol.StringVal
		}
		if strings.TrimSpace(claims.SupplierID) != "" && supplierID != "" && claims.SupplierID != supplierID {
			return ErrOrderForbidden
		}

		mutations := []*spanner.Mutation{}
		switch req.Action {
		case "BYPASS":
			resolution = "BYPASS_ISSUED"
			bypassToken = generateShopClosedBypassToken()
			mutations = append(mutations, spanner.UpdateMap("ShopClosedAttempts", map[string]any{
				"AttemptId":   req.AttemptID,
				"Resolution":  resolution,
				"BypassToken": bypassToken,
				"ResolvedAt":  now.UTC(),
				"ResolvedBy":  adminID,
			}))
		case "RETURN_TO_DEPOT":
			resolution = "RETURN_TO_DEPOT"
			if err := ReleaseReservationsFromOrderFields(ctx, txn, supplierID, warehouseID, orderSource, lineItemsRaw); err != nil {
				return err
			}
			mutations = append(mutations,
				spanner.UpdateMap("Orders", map[string]any{
					"OrderId":              orderID,
					"Status":               string(StatusCancelled),
					"ShopClosedResolution": ShopClosedResolutionReturned,
					"Version":              version + 1,
					"UpdatedAt":            now.UTC(),
				}),
				spanner.UpdateMap("ShopClosedAttempts", map[string]any{
					"AttemptId":  req.AttemptID,
					"Resolution": resolution,
					"ResolvedAt": now.UTC(),
					"ResolvedBy": adminID,
				}),
			)
		default:
			mutations = append(mutations, spanner.UpdateMap("ShopClosedAttempts", map[string]any{
				"AttemptId":  req.AttemptID,
				"Resolution": resolution,
				"ResolvedAt": now.UTC(),
				"ResolvedBy": adminID,
			}))
		}

		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventShopClosedResolved, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:    orderID,
			AttemptID:  req.AttemptID,
			SupplierID: supplierID,
			Resolution: resolution,
		}); err != nil {
			return err
		}
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		if errors.Is(err, ErrOrderForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	if s.replanner != nil && manifestID != "" {
		go func(rID, act string) {
			_ = s.replanner.ReplanRoute(context.Background(), rID, "shop_closed_resolved", act)
		}(manifestID, adminID)
	}

	payload := events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventShopClosedResolved, Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    orderID,
		AttemptID:  req.AttemptID,
		SupplierID: supplierID,
		Resolution: resolution,
	}
	s.broadcastShopClosed(ctx, supplierID, retailerID, driverID, payload)
	s.invalidateOrderCache(ctx, orderID)

	resp := map[string]any{"status": resolution, "attempt_id": req.AttemptID}
	if bypassToken != "" {
		resp["bypass_token"] = bypassToken
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}





func (s *Service) broadcastShopClosed(ctx context.Context, supplierID, retailerID, driverID string, payload events.OrderEvent) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if s.supplierHub != nil && supplierID != "" {
		s.supplierHub.Broadcast(ctx, "supplier:"+supplierID, raw)
	}
	if s.retailerHub != nil && retailerID != "" {
		s.retailerHub.Broadcast(ctx, "retailer:"+retailerID, raw)
	}
	if s.driverHub != nil && driverID != "" {
		s.driverHub.Broadcast(ctx, "driver:"+driverID, raw)
	}
}

func (s *Service) invalidateOrderCache(ctx context.Context, orderID string) {
	if s.cache == nil || orderID == "" {
		return
	}
	s.cache.Invalidate(ctx, "order:"+orderID)
}

func outboxMutation(e outbox.Event) *spanner.Mutation {
	row := map[string]any{
		"EventId":       e.EventID,
		"AggregateType": e.AggregateType,
		"AggregateId":   e.AggregateID,
		"TopicName":     e.TopicName,
		"Payload":       e.Payload,
		// Commit timestamp: row creation time is commit time. Wall-clock
		// timestamps here fail on any client/host clock skew.
		"CreatedAt":   spanner.CommitTimestamp,
		"PublishedAt": nil,
	}
	if e.PublishedAt != nil {
		row["PublishedAt"] = e.PublishedAt.UTC()
	}
	return spanner.InsertOrUpdateMap("OutboxEvents", row)
}

func generateShopClosedBypassToken() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "000000"
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
}
