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
	Reason    string  `json:"reason,omitempty"`
}

type shopClosedResponseRequest struct {
	OrderID  string `json:"order_id"`
	Response string `json:"response"`
}

type shopClosedResolveRequest struct {
	AttemptID string `json:"attempt_id"`
	Action    string `json:"action"`
}



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

	var req shopClosedReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id required"})
		return
	}

	driverID := strings.TrimSpace(claims.Subject)
	attemptID := s.newID()
	now := s.now()
	ctx := r.Context()

	var retailerID, supplierID string
	var gpsLat, gpsLng float64

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
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

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId": req.OrderID,
				"Status":  string(StatusArrivedShopClosed),
				"Version": version + 1,
				"UpdatedAt": now.UTC(),
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
		}
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
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

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     string(StatusArrivedShopClosed),
		"attempt_id": attemptID,
	})
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

	var req shopClosedResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.Response = strings.TrimSpace(req.Response)
	if req.OrderID == "" || req.Response == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id and response required"})
		return
	}
	valid := map[string]bool{"OPEN_NOW": true, "5_MIN": true, "CALL_ME": true, "CLOSED_TODAY": true}
	if !valid[req.Response] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid response value"})
		return
	}

	retailerID := strings.TrimSpace(claims.Subject)
	ctx := r.Context()
	now := s.now()
	newStatus := string(StatusArrivedShopClosed)
	var attemptID, driverID, supplierID string

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

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("ShopClosedAttempts", map[string]any{
				"AttemptId":           attemptID,
				"RetailerResponse":    req.Response,
				"RetailerRespondedAt": now.UTC(),
			}),
		}
		if req.Response == "OPEN_NOW" {
			newStatus = string(StatusArrived)
			mutations = append(mutations,
				spanner.UpdateMap("Orders", map[string]any{
					"OrderId":   req.OrderID,
					"Status":    newStatus,
					"Version":   version + 1,
					"UpdatedAt": now.UTC(),
				}),
				spanner.UpdateMap("ShopClosedAttempts", map[string]any{
					"AttemptId":  attemptID,
					"Resolution": "RETAILER_OPENED",
					"ResolvedAt": now.UTC(),
				}),
			)
		}

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
	writeJSON(w, http.StatusOK, map[string]string{"status": newStatus})
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

	var req shopClosedResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()
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

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
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
				"AttemptId":  req.AttemptID,
				"Resolution": resolution,
				"BypassToken": bypassToken,
				"ResolvedAt": now.UTC(),
				"ResolvedBy": adminID,
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
	writeJSON(w, http.StatusOK, resp)
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

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `SELECT Resolution, RetailerRespondedAt FROM ShopClosedAttempts WHERE AttemptId = @aid`,
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
		if resolution.Valid || responded.Valid {
			return nil
		}

		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.OrderEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventShopClosedEscalated, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:     orderID,
			AttemptID:   attemptID,
			SupplierID:  supplierID,
			EscalatedTo: escalatedTo,
		}); err != nil {
			return err
		}
		mutations := []*spanner.Mutation{
			spanner.UpdateMap("ShopClosedAttempts", map[string]any{
				"AttemptId":   attemptID,
				"EscalatedAt": now.UTC(),
				"EscalatedTo": escalatedTo,
			}),
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
	})
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
