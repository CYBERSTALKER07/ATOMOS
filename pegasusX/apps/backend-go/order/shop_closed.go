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

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type shopClosedReportRequest struct {
	OrderID   string  `json:"order_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	// Reason: NO_ANSWER | CLOSED | REFUSED | OTHER
	Reason string `json:"reason,omitempty"`
	// PhotoURL optional evidence of closed shop.
	PhotoURL string `json:"photo_url,omitempty"`
	// ClientTimestamp for offline mark-closed (original capture time).
	ClientTimestamp string `json:"client_timestamp,omitempty"`
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
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id required"})
		return
	}
	reason := NormalizeShopClosedReason(req.Reason)

	driverID := strings.TrimSpace(claims.Subject)
	attemptID := s.newID()
	logEventID := s.newID()
	now := s.now()
	if ts := strings.TrimSpace(req.ClientTimestamp); ts != "" {
		if t, perr := time.Parse(time.RFC3339Nano, ts); perr == nil {
			if t.UTC().Before(now) {
				now = t.UTC()
			}
		} else if t, perr := time.Parse(time.RFC3339, ts); perr == nil {
			if t.UTC().Before(now) {
				now = t.UTC()
			}
		}
	}
	graceEnds := now.Add(s.shopGrace)
	ctx := r.Context()

	var retailerID, supplierID string
	var gpsLat, gpsLng float64

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
			[]string{"Status", "Version", "RetailerId", "SupplierId", "DriverId", "Lat", "Lng"})
		if err != nil {
			return fmt.Errorf("order %s not found: %w", req.OrderID, err)
		}
		var status string
		var version int64
		var driverCol, retailerCol, supplierCol spanner.NullString
		var orderLat, orderLng float64
		if err := row.Columns(&status, &version, &retailerCol, &supplierCol, &driverCol, &orderLat, &orderLng); err != nil {
			return err
		}
		if status != string(StatusArrived) {
			return fmt.Errorf("order must be ARRIVED to report shop closed (current: %s)", status)
		}
		if !driverCol.Valid || driverCol.StringVal != driverID {
			return fmt.Errorf("driver is not assigned to order %s", req.OrderID)
		}
		if retailerCol.Valid {
			retailerID = retailerCol.StringVal
		}
		if supplierCol.Valid {
			supplierID = supplierCol.StringVal
		}
		gpsLat, gpsLng = req.Latitude, req.Longitude
		if gpsLat == 0 && gpsLng == 0 {
			gpsLat, gpsLng = orderLat, orderLng
		}

		// Check for max retries
		countStmt := spanner.Statement{
			SQL:    `SELECT COUNT(*) FROM ShopClosedAttempts WHERE OrderId = @oid`,
			Params: map[string]any{"oid": req.OrderID},
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
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, req.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventShopClosed, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:    req.OrderID,
			DriverID:   driverID,
			RetailerID: retailerID,
			SupplierID: supplierID,
			AttemptID:  attemptID,
			GPSLat:     gpsLat,
			GPSLng:     gpsLng,
		}); err != nil {
			return err
		}

		logPayload, _ := json.Marshal(map[string]any{
			"reason":        reason,
			"photo_url":     strings.TrimSpace(req.PhotoURL),
			"gps_lat":       gpsLat,
			"gps_lng":       gpsLng,
			"attempt_id":    attemptID,
			"grace_ends_at": graceEnds.UTC().Format(time.RFC3339Nano),
		})

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":               req.OrderID,
				"Status":                string(StatusArrivedShopClosed), // ≡ SHOP_CLOSED_PENDING
				"Version":               version + 1,
				"UpdatedAt":             now.UTC(),
				"ShopClosedAt":          now.UTC(),
				"ShopClosedReason":      reason,
				"ShopClosedGraceEndsAt": graceEnds.UTC(),
			}),
			spanner.InsertMap("ShopClosedAttempts", map[string]any{
				"AttemptId":  attemptID,
				"OrderId":    req.OrderID,
				"DriverId":   driverID,
				"RetailerId": retailerID,
				"ReportedAt": now.UTC(),
				"GPSLat":     gpsLat,
				"GPSLng":     gpsLng,
			}),
			spanner.InsertMap("OrderShopClosedLog", map[string]any{
				"OrderId":   req.OrderID,
				"EventId":   logEventID,
				"Actor":     driverID,
				"Action":    "MARKED_CLOSED",
				"Payload":   logPayload,
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
			s.UpdateStatus(ctx, updateClaims, req.OrderID, updateReq)
			s.log.WarnContext(ctx, "shop closed max retries reached, order cancelled", "order_id", req.OrderID)
			writeJSON(w, http.StatusConflict, map[string]string{"error": "max_retries_reached_order_cancelled"})
			return
		}
		s.log.ErrorContext(ctx, "shop closed report failed", "order_id", req.OrderID, "err", err)
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	s.broadcastShopClosed(ctx, supplierID, retailerID, driverID, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventShopClosed, Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    req.OrderID,
		DriverID:   driverID,
		RetailerID: retailerID,
		SupplierID: supplierID,
		AttemptID:  attemptID,
	})
	s.invalidateOrderCache(ctx, req.OrderID)
	go s.scheduleShopClosedEscalation(context.WithoutCancel(ctx), attemptID, req.OrderID, retailerID, supplierID, driverID)

	resp := map[string]any{
		"status":        string(StatusArrivedShopClosed), // design: SHOP_CLOSED_PENDING
		"attempt_id":    attemptID,
		"reason":        reason,
		"grace_ends_at": graceEnds.UTC().Format(time.RFC3339Nano),
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
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

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
	ctx := r.Context()
	now := s.now()
	newStatus := string(StatusArrivedShopClosed)
	var attemptID, driverID, supplierID string
	logEventID := s.newID()
	resolution := ""

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
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
			[]string{"Version", "SupplierId", "Status"})
		if err != nil {
			return err
		}
		var version int64
		var supplierCol spanner.NullString
		var orderStatus string
		if err := orderRow.Columns(&version, &supplierCol, &orderStatus); err != nil {
			return err
		}
		if supplierCol.Valid {
			supplierID = supplierCol.StringVal
		}
		// Retailer response wins if still PENDING (ARRIVED_SHOP_CLOSED), even after grace clock.
		if orderStatus != string(StatusArrivedShopClosed) {
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
			resolution = ShopClosedResRescheduled
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
			resolution = ShopClosedResCreditLeave
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
			resolution = ShopClosedResCancelled
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
			resolution = ShopClosedResBypass
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

		logPayload, _ := json.Marshal(map[string]any{
			"response":   req.Response,
			"new_slot":   strings.TrimSpace(req.NewSlot),
			"photo_url":  strings.TrimSpace(req.PhotoURL),
			"attempt_id": attemptID,
			"resolution": resolution,
		})
		mutations = append(mutations, spanner.InsertMap("OrderShopClosedLog", map[string]any{
			"OrderId":   req.OrderID,
			"EventId":   logEventID,
			"Actor":     retailerID,
			"Action":    "RESPONDED",
			"Payload":   logPayload,
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
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
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
	var orderID, driverID, retailerID, supplierID, bypassToken, resolution string
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
			[]string{"Version", "SupplierId"})
		if err != nil {
			return err
		}
		var version int64
		var supplierCol spanner.NullString
		if err := orderRow.Columns(&version, &supplierCol); err != nil {
			return err
		}
		if supplierCol.Valid {
			supplierID = supplierCol.StringVal
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
			mutations = append(mutations,
				spanner.UpdateMap("Orders", map[string]any{
					"OrderId":   orderID,
					"Status":    string(StatusCancelled),
					"Version":   version + 1,
					"UpdatedAt": now.UTC(),
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

func (s *Service) scheduleShopClosedEscalation(ctx context.Context, attemptID, orderID, retailerID, supplierID, driverID string) {
	timer := time.NewTimer(s.shopGrace)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	if s.spannerClient == nil {
		return
	}
	now := s.now()
	escalatedTo := supplierID
	if escalatedTo == "" {
		escalatedTo = s.supplierID
	}

	// Precompute timeout decision outside the tight CAS window (credit read is best-effort).
	decision := s.evaluateShopClosedTimeout(ctx, orderID, retailerID, supplierID)
	logEventID := s.newID()

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    `SELECT Resolution, RetailerRespondedAt FROM ShopClosedAttempts WHERE AttemptId = @aid`,
			Params: map[string]any{"aid": attemptID},
		}
		iter := txn.Query(ctx, stmt)
		row, err := iter.Next()
		iter.Stop()
		if err != nil {
			return err
		}
		var resolution spanner.NullString
		var responded spanner.NullTime
		if err := row.Columns(&resolution, &responded); err != nil {
			return err
		}
		// Retailer response wins if still PENDING when worker fires.
		if resolution.Valid || responded.Valid {
			return nil
		}

		orderRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"Status", "Version"})
		if err != nil {
			return err
		}
		var orderStatus string
		var version int64
		if err := orderRow.Columns(&orderStatus, &version); err != nil {
			return err
		}
		if orderStatus != string(StatusArrivedShopClosed) {
			return nil
		}

		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.OrderEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventShopClosedEscalated, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:     orderID,
			AttemptID:   attemptID,
			SupplierID:  supplierID,
			EscalatedTo: escalatedTo,
			Resolution:  string(decision.Resolution),
		}); err != nil {
			return err
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventShopClosedTimeout, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:    orderID,
			AttemptID:  attemptID,
			SupplierID: supplierID,
			Resolution: string(decision.Resolution),
			Response:   decision.Reason,
		}); err != nil {
			return err
		}

		logPayload, _ := json.Marshal(map[string]any{
			"resolution": decision.Resolution,
			"reason":     decision.Reason,
			"attempt_id": attemptID,
		})

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("ShopClosedAttempts", map[string]any{
				"AttemptId":   attemptID,
				"EscalatedAt": now.UTC(),
				"EscalatedTo": escalatedTo,
			}),
			spanner.InsertMap("OrderShopClosedLog", map[string]any{
				"OrderId":   orderID,
				"EventId":   logEventID,
				"Actor":     "system",
				"Action":    "TIMEOUT",
				"Payload":   logPayload,
				"CreatedAt": now.UTC(),
			}),
		}

		// Apply auto-resolution matrix (design §4.2).
		switch decision.Resolution {
		case TimeoutCreditLeave:
			// Mark intent; driver/credit-leave path still requires proximity or supervised bypass.
			mutations = append(mutations,
				spanner.UpdateMap("Orders", map[string]any{
					"OrderId":              orderID,
					"ShopClosedResolution": ShopClosedResCreditLeave,
					"Version":              version + 1,
					"UpdatedAt":            now.UTC(),
				}),
				spanner.UpdateMap("ShopClosedAttempts", map[string]any{
					"AttemptId":  attemptID,
					"Resolution": "TIMEOUT_CREDIT_LEAVE",
					"ResolvedAt": now.UTC(),
					"ResolvedBy": "system",
				}),
			)
		case TimeoutReturnToWarehouse:
			mutations = append(mutations,
				spanner.UpdateMap("Orders", map[string]any{
					"OrderId":              orderID,
					"Status":               string(StatusCancelled),
					"ShopClosedResolution": ShopClosedResReturned,
					"Version":              version + 1,
					"UpdatedAt":            now.UTC(),
				}),
				spanner.UpdateMap("ShopClosedAttempts", map[string]any{
					"AttemptId":  attemptID,
					"Resolution": "TIMEOUT_RETURN_TO_WAREHOUSE",
					"ResolvedAt": now.UTC(),
					"ResolvedBy": "system",
				}),
			)
		case TimeoutForceBypass:
			tok := generateShopClosedBypassToken()
			mutations = append(mutations,
				spanner.UpdateMap("Orders", map[string]any{
					"OrderId":              orderID,
					"ShopClosedResolution": ShopClosedResBypass,
					"Version":              version + 1,
					"UpdatedAt":            now.UTC(),
				}),
				spanner.UpdateMap("ShopClosedAttempts", map[string]any{
					"AttemptId":   attemptID,
					"Resolution":  "TIMEOUT_FORCE_BYPASS",
					"BypassToken": tok,
					"ResolvedAt":  now.UTC(),
					"ResolvedBy":  "system",
				}),
			)
		case TimeoutReschedule:
			mutations = append(mutations,
				spanner.UpdateMap("Orders", map[string]any{
					"OrderId":              orderID,
					"ShopClosedResolution": ShopClosedResRescheduled,
					"Version":              version + 1,
					"UpdatedAt":            now.UTC(),
				}),
				spanner.UpdateMap("ShopClosedAttempts", map[string]any{
					"AttemptId":  attemptID,
					"Resolution": "TIMEOUT_RESCHEDULE",
					"ResolvedAt": now.UTC(),
					"ResolvedBy": "system",
				}),
			)
		default:
			// Escalate only (supplier queue) — preserve prior behavior for unknown.
		}

		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		s.log.WarnContext(ctx, "shop closed escalation skipped", "attempt_id", attemptID, "err", err)
		return
	}

	s.broadcastShopClosed(ctx, supplierID, retailerID, driverID, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventShopClosedEscalated, Timestamp: now.Format(time.RFC3339Nano)},
		OrderID:    orderID,
		AttemptID:  attemptID,
		SupplierID: supplierID,
		Resolution: string(decision.Resolution),
	})
}

// evaluateShopClosedTimeout loads credit signals and runs DecideShopClosedTimeout.
func (s *Service) evaluateShopClosedTimeout(ctx context.Context, orderID, retailerID, supplierID string) ShopClosedTimeoutDecision {
	in := ShopClosedTimeoutInput{
		RiskTier:      ShopClosedRiskMedium,
		ProfileStatus:  "",
		CreditAllowed:  false,
		OrderTotalMinor: 0,
	}
	if s.repo != nil {
		if o, found, err := s.repo.GetOrder(ctx, orderID); err == nil && found {
			in.OrderTotalMinor = o.TotalMinor
			if supplierID == "" {
				supplierID = o.SupplierID
			}
			if retailerID == "" {
				retailerID = o.RetailerID
			}
		}
	}
	if s.credit != nil && retailerID != "" && supplierID != "" {
		check, err := s.credit.CheckOrder(ctx, retailerID, supplierID, in.OrderTotalMinor)
		if err == nil {
			in.CreditAllowed = check.Allowed
			in.AvailableCreditMinor = check.CreditLimitMinor - check.CurrentBalance
			if in.AvailableCreditMinor < 0 {
				in.AvailableCreditMinor = 0
			}
			if check.Allowed {
				in.ProfileStatus = "ACTIVE"
			} else {
				switch check.Reason {
				case "profile_frozen":
					in.ProfileStatus = "FROZEN"
				case "profile_blacklisted":
					in.ProfileStatus = "BLACKLISTED"
				case "risk_tier_block":
					in.RiskTier = ShopClosedRiskBlock
					in.ProfileStatus = "ACTIVE"
				case "no_credit_profile", "no_credit_limit":
					in.ProfileStatus = ""
				default:
					in.ProfileStatus = "ACTIVE"
				}
			}
		}
	}
	// FORCE_BYPASS only via supplier resolve for now (safer default).
	in.ForceBypassEnabled = false
	return DecideShopClosedTimeout(in)
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
	createdAt := e.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	row := map[string]any{
		"EventId":       e.EventID,
		"AggregateType": e.AggregateType,
		"AggregateId":   e.AggregateID,
		"TopicName":     e.TopicName,
		"Payload":       e.Payload,
		"CreatedAt":     createdAt,
		"PublishedAt":   nil,
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
