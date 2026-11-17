// Package order owns the order aggregate: creation, state transitions,
// reassignment, and lifecycle event emission. Every mutating method runs
// inside a Repository.* method that wraps a ReadWriteTransaction so the row
// mutation and the OutboxEvents row commit atomically.
//
// Gate 5: request paths resolve supplier via TenantContext
// (resolveSupplierScope); bootstrap SeedSupplierID is fixtures/fallback only.
package order

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/packages/handoff"
)

func (s *Service) validateOfflineQR(manifestID, driverID, orderID, token string) error {
	h := sha256.New()
	h.Write([]byte(manifestID))
	h.Write([]byte(driverID))
	h.Write([]byte(s.jwtSecret))
	offlineNonce := hex.EncodeToString(h.Sum(nil))

	expected := sha256.New()
	expected.Write([]byte(offlineNonce))
	expected.Write([]byte(orderID))
	expectedHash := hex.EncodeToString(expected.Sum(nil))

	hashMatch := handoff.ConstantTimeEqual(token, expectedHash)
	nonceMatch := handoff.ConstantTimeEqual(token, offlineNonce)
	if !hashMatch && !nonceMatch {
		return errors.New("invalid offline qr token")
	}
	return nil
}

func buildDeliveryProofArtifacts(proofID string, orderRecord Order, driverID string, proofType DeliveryProofType, qrToken string, scannedToken string, latitude *float64, longitude *float64, distanceM *float64) []DeliveryProofArtifact {
	resolvedDriverID := strings.TrimSpace(driverID)
	if resolvedDriverID == "" {
		resolvedDriverID = strings.TrimSpace(orderRecord.DriverID)
	}
	qrTokenHash := hashDeliveryProofToken(qrToken)
	scannedTokenHash := hashDeliveryProofToken(scannedToken)
	if strings.TrimSpace(proofID) == "" || strings.TrimSpace(orderRecord.OrderID) == "" || resolvedDriverID == "" {
		return nil
	}
	if qrTokenHash == "" && scannedTokenHash == "" && (latitude == nil || longitude == nil) {
		return nil
	}
	capturedAt := orderRecord.UpdatedAt.UTC()
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
	}
	return []DeliveryProofArtifact{{
		ProofID:          strings.TrimSpace(proofID),
		OrderID:          orderRecord.OrderID,
		SupplierID:       orderRecord.SupplierID,
		RetailerID:       orderRecord.RetailerID,
		DriverID:         resolvedDriverID,
		ProofType:        proofType,
		QRTokenHash:      qrTokenHash,
		ScannedTokenHash: scannedTokenHash,
		Latitude:         latitude,
		Longitude:        longitude,
		DistanceM:        distanceM,
		CapturedAt:       capturedAt,
	}}
}

func hashDeliveryProofToken(token string) string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func deliveryProofCoordinatesFromValues(latitude, longitude float64) (*float64, *float64) {
	if latitude == 0 && longitude == 0 {
		return nil, nil
	}
	lat := latitude
	lng := longitude
	return &lat, &lng
}

func deliveryProofCoordinatesFromPointers(latitude, longitude *float64) (*float64, *float64) {
	if latitude == nil || longitude == nil {
		return nil, nil
	}
	lat := *latitude
	lng := *longitude
	return &lat, &lng
}

func deliveryProofDistance(distanceM float64, latitude, longitude *float64) *float64 {
	if latitude == nil || longitude == nil {
		return nil
	}
	value := distanceM
	return &value
}

func validatePointerGeofence(ctx context.Context, latitude, longitude *float64, orderRecord Order) (float64, error) {
	if latitude == nil || longitude == nil {
		return 0, errors.New("latitude and longitude required")
	}
	return validateRequiredGeofence(ctx, *latitude, *longitude, orderRecord)
}

func validateOptionalGeofence(ctx context.Context, latitude, longitude float64, orderRecord Order) (float64, error) {
	return validateRequiredGeofence(ctx, latitude, longitude, orderRecord)
}

func validateRequiredGeofence(ctx context.Context, latitude, longitude float64, orderRecord Order) (float64, error) {
	if latitude == 0 && longitude == 0 {
		return 0, errors.New("latitude and longitude required")
	}
	if orderRecord.Lat == 0 && orderRecord.Lng == 0 {
		return 0, fmt.Errorf("%w: order coordinates unavailable", ErrServiceabilityUnavailable)
	}
	radius, err := auth.BreachRadiusFromContext(ctx, orderRecord.SupplierID)
	if err != nil {
		return 0, err
	}
	distanceM := distanceMeters(latitude, longitude, orderRecord.Lat, orderRecord.Lng)
	if distanceM > radius {
		return distanceM, fmt.Errorf("%w: %.0fm from delivery point", ErrGeofenceViolation, distanceM)
	}
	return distanceM, nil
}

func distanceMeters(latA, lngA, latB, lngB float64) float64 {
	const earthRadiusMeters = 6371000.0
	latARadians := latA * math.Pi / 180
	latBRadians := latB * math.Pi / 180
	deltaLat := (latB - latA) * math.Pi / 180
	deltaLng := (lngB - lngA) * math.Pi / 180

	angle := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(latARadians)*math.Cos(latBRadians)*math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	circle := 2 * math.Atan2(math.Sqrt(angle), math.Sqrt(1-angle))
	return earthRadiusMeters * circle
}

// AmendItemRequest is one line adjustment from the driver offload review surface.
type AmendItemRequest struct {
	ProductID    string `json:"product_id"`
	AcceptedQty  int64  `json:"accepted_qty"`
	RejectedQty  int64  `json:"rejected_qty"`
	Reason       string `json:"reason"`
	CustomReason string `json:"custom_reason"`
}

// AmendOrderRequest is POST /v1/order/amend.
type AmendOrderRequest struct {
	OrderID     string             `json:"order_id"`
	Items       []AmendItemRequest `json:"items"`
	DriverNotes string             `json:"driver_notes"`
}

// AmendOrderResponse matches native driver amend contracts.
type AmendOrderResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	AdjustedTotal int64  `json:"adjusted_total"`
}

// AmendOrder applies driver-reported rejections and recomputes order totals.
func (s *Service) AmendOrder(ctx context.Context, claims auth.Claims, req AmendOrderRequest) (AmendOrderResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return AmendOrderResponse{}, errors.New("order_id required")
	}
	if len(req.Items) == 0 {
		return AmendOrderResponse{}, errors.New("items required")
	}

	current, err := s.loadDriverTransitionOrder(ctx, claims, orderID)
	if err != nil {
		return AmendOrderResponse{}, err
	}

	return s.applyOrderAmendments(ctx, current, req, claims.Subject)
}

func (s *Service) applyOrderAmendments(ctx context.Context, current Order, req AmendOrderRequest, actorID string) (AmendOrderResponse, error) {
	orderID := strings.TrimSpace(current.OrderID)

	amendByProduct := make(map[string]AmendItemRequest, len(req.Items))
	reasons := make([]string, 0, len(req.Items))
	for _, item := range req.Items {
		key := strings.TrimSpace(item.ProductID)
		if key == "" {
			continue
		}
		amendByProduct[key] = item
		if r := normalizeAmendReason(item.Reason); r != "" {
			reasons = append(reasons, r)
		}
	}
	if len(amendByProduct) == 0 {
		return AmendOrderResponse{}, errors.New("items required")
	}

	now := s.now()
	if !orderAmendable(current.Status) {
		// Post-delivery concealed damage / shortage: COMPLETED within claim window.
		completedAt := current.UpdatedAt
		if !orderPostDeliveryAmendable(current.Status, completedAt, now, reasons) {
			return AmendOrderResponse{}, fmt.Errorf("order %s cannot be amended from state %s", orderID, current.Status)
		}
	}

	origQtyBySKU := make(map[string]int64, len(current.LineItems))
	for _, line := range current.LineItems {
		origQtyBySKU[strings.TrimSpace(line.SKU)] += line.Quantity
	}

	remainingAccepted := make(map[string]int64, len(amendByProduct))
	pendingReturns := make([]SupplierReturn, 0, len(req.Items))
	for sku, amend := range amendByProduct {
		origQty, exists := origQtyBySKU[sku]
		if !exists {
			return AmendOrderResponse{}, fmt.Errorf("sku %s not found in order line items", sku)
		}
		acceptedQty, rejectedQty, err := resolveAmendQuantities(origQty, amend.AcceptedQty, amend.RejectedQty)
		if err != nil {
			return AmendOrderResponse{}, fmt.Errorf("%s for sku %s", err.Error(), sku)
		}
		remainingAccepted[sku] = acceptedQty
		if rejectedQty > 0 {
			reason := normalizeAmendReason(amend.Reason)
			if reason == "" {
				return AmendOrderResponse{}, fmt.Errorf("reason required for rejected quantity on sku %s", sku)
			}
			if err := validateAmendReason(reason, amend.CustomReason); err != nil {
				return AmendOrderResponse{}, fmt.Errorf("sku %s: %w", sku, err)
			}
			pendingReturns = append(pendingReturns, SupplierReturn{
				ReturnID:    s.newID(),
				SKU:         sku,
				RejectedQty: rejectedQty,
				Reason:      reason,
				DriverNotes: supplierReturnNotes(reason, amend.CustomReason, req.DriverNotes),
				ManifestID:  strings.TrimSpace(current.ManifestID),
				DriverID:    strings.TrimSpace(current.DriverID),
				WarehouseID: strings.TrimSpace(current.WarehouseID),
			})
		}
	}

	updatedItems := make([]LineItem, 0, len(current.LineItems))
	var adjustedTotal int64
	for _, line := range current.LineItems {
		key := strings.TrimSpace(line.SKU)
		if _, ok := amendByProduct[key]; ok {
			rem := remainingAccepted[key]
			var lineAlloc int64
			if rem >= line.Quantity {
				lineAlloc = line.Quantity
				remainingAccepted[key] -= line.Quantity
			} else if rem > 0 {
				lineAlloc = rem
				remainingAccepted[key] = 0
			} else {
				lineAlloc = 0
			}
			line.Quantity = lineAlloc
		}
		adjustedTotal += line.UnitPrice * line.Quantity
		updatedItems = append(updatedItems, line)
	}

	previousTotal := current.TotalMinor
	if current.OriginalTotalMinor == 0 {
		current.OriginalTotalMinor = previousTotal
	}
	current.LineItems = updatedItems
	if adjustedTotal > 0 && current.DeliveryFeeMinor > 0 {
		current.TotalMinor = adjustedTotal + current.DeliveryFeeMinor
	} else {
		current.TotalMinor = adjustedTotal
	}
	current.PendingSupplierReturns = pendingReturns
	current.UpdatedAt = s.now()

	if err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		payload := map[string]any{
			"type":            events.EventOrderAmended,
			"order_id":        current.OrderID,
			"driver_id":       strings.TrimSpace(actorID),
			"previous_total":  previousTotal,
			"adjusted_total":  adjustedTotal,
			"original_amount": current.OriginalTotalMinor,
			"currency":        current.Currency,
			"driver_notes":    strings.TrimSpace(req.DriverNotes),
			"amended_items":   req.Items,
			"timestamp":       current.UpdatedAt.Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, payload); err != nil {
			return err
		}
		if len(pendingReturns) > 0 {
			for _, pr := range pendingReturns {
				returnPayload := map[string]any{
					"type":            events.EventSupplierReturnCreated,
					"return_id":       pr.ReturnID,
					"order_id":        current.OrderID,
					"sku_id":          pr.SKU,
					"quantity":        pr.RejectedQty,
					"reason":          pr.Reason,
					"manifest_id":     strings.TrimSpace(current.ManifestID),
					"driver_id":       strings.TrimSpace(current.DriverID),
					"warehouse_id":    strings.TrimSpace(current.WarehouseID),
					"supplier_id":     strings.TrimSpace(current.SupplierID),
					"physical_status": "PENDING",
					"timestamp":       current.UpdatedAt.Format(time.RFC3339Nano),
				}
				if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, returnPayload); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return AmendOrderResponse{}, fmt.Errorf("amend order %s: %w", orderID, err)
	}

	current.PendingSupplierReturns = nil
	s.afterOrderMutation(ctx, current)

	return AmendOrderResponse{
		Success:       true,
		Message:       "amended",
		AdjustedTotal: adjustedTotal,
	}, nil
}

// HandleAmendOrder is POST /v1/order/amend.
func (s *Service) HandleAmendOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req AmendOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	resp, err := s.AmendOrder(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "amend order failed", req.OrderID, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleReportDamage serves POST /v1/delivery/report-damage.
func (s *Service) HandleReportDamage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req struct {
		OrderID      string `json:"order_id"`
		DamagedItems []struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
		} `json:"damaged_items"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	if len(req.DamagedItems) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "damaged_items_required"})
		return
	}

	current, found, err := s.repo.GetOrder(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if current.DriverID != claims.Subject {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	if current.Status != StatusInTransit && current.Status != StatusArrived {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_status_for_damage_report"})
		return
	}

	origQtyBySKU := make(map[string]int64, len(current.LineItems))
	for _, line := range current.LineItems {
		origQtyBySKU[strings.TrimSpace(line.SKU)] += line.Quantity
	}

	damagedBySKU := make(map[string]int64, len(req.DamagedItems))
	for _, item := range req.DamagedItems {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_damaged_item"})
			return
		}
		damagedBySKU[sku] += item.Quantity
	}

	for sku, dmgQty := range damagedBySKU {
		origQty, ok := origQtyBySKU[sku]
		if !ok || dmgQty > origQty {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("damage quantity exceeds item quantity for sku %s", sku)})
			return
		}
	}

	amendItems := make([]AmendItemRequest, 0, len(damagedBySKU))
	for sku, dmgQty := range damagedBySKU {
		origQty := origQtyBySKU[sku]
		amendItems = append(amendItems, AmendItemRequest{
			ProductID:   sku,
			AcceptedQty: origQty - dmgQty,
			RejectedQty: dmgQty,
			Reason:      "DAMAGED",
		})
	}

	conditionReports := make([]ConditionReport, 0, len(req.DamagedItems))
	for _, item := range req.DamagedItems {
		sku := strings.TrimSpace(item.SKU)
		conditionReports = append(conditionReports, ConditionReport{
			ReportID:         s.newID(),
			OrderID:          current.OrderID,
			SupplierID:       current.SupplierID,
			RetailerID:       current.RetailerID,
			SKU:              sku,
			ConditionType:    ConditionTypeDamaged,
			Severity:         SeverityHigh,
			Description:      strings.TrimSpace(req.Reason),
			ReportedBy:       claims.Subject,
			ReportedByRole:   string(claims.Role),
			ResolutionStatus: ResolutionStatusOpen,
			CreatedAt:        s.now(),
		})
	}
	current.ConditionReports = conditionReports

	amendResp, err := s.applyOrderAmendments(r.Context(), current, AmendOrderRequest{
		OrderID:     orderID,
		Items:       amendItems,
		DriverNotes: req.Reason,
	}, claims.Subject)
	if err != nil {
		s.writeOrderMutationError(w, "report damage amend failed", orderID, err)
		return
	}

	idemCommitted = true
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]any{
		"success":        true,
		"message":        "damage_reported",
		"adjusted_total": amendResp.AdjustedTotal,
	})
}

// SelectCashAtDelivery transitions ARRIVED|AWAITING_PAYMENT → PENDING_CASH_COLLECTION
// with same-txn outbox (B1 M-P0-5). Implements payment.OrderCashSelector.
func (s *Service) SelectCashAtDelivery(ctx context.Context, orderID, retailerID, actorID string) (status string, amountMinor int64, currency string, err error) {
	orderID = strings.TrimSpace(orderID)
	retailerID = strings.TrimSpace(retailerID)
	if orderID == "" || retailerID == "" {
		return "", 0, "", errors.New("order_id and retailer_id required")
	}
	current, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return "", 0, "", err
	}
	if !found {
		return "", 0, "", ErrOrderNotFound
	}
	if current.RetailerID != retailerID {
		return "", 0, "", ErrOrderForbidden
	}
	// Idempotent replay: already selected cash.
	if current.Status == StatusPendingCashCollection {
		return string(current.Status), current.TotalMinor, current.Currency, nil
	}
	switch current.Status {
	case StatusArrived, StatusAwaitingPayment:
		// ok
	default:
		return "", 0, "", fmt.Errorf("%w: cannot select cash from %s", ErrInvalidStatusTransition, current.Status)
	}
	if err := ValidateStatusTransition(string(current.Status), string(StatusPendingCashCollection), TransitionOpts{}); err != nil {
		return "", 0, "", err
	}

	previousStatus := current.Status
	current.Status = StatusPendingCashCollection
	current.UpdatedAt = s.now()
	claims := auth.Claims{Subject: actorID, Role: auth.RoleRetailer}

	if err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Claims:         claims,
			Order:          current,
			PreviousStatus: previousStatus,
			Reason:         "retailer_selected_cash",
			ActorID:        actorID,
		}); err != nil {
			return err
		}
		cashPayment := s.paymentRequiredData(ctx, current)
		cashPayment["payment_method"] = "CASH"
		cashPayment["status"] = string(StatusPendingCashCollection)
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, cashPayment)
	}); err != nil {
		return "", 0, "", err
	}
	s.afterOrderMutation(ctx, current)
	return string(current.Status), current.TotalMinor, current.Currency, nil
}

// HandleRetailerConfirmCash serves POST /v1/delivery/confirm-cash for Retailers.
func (s *Service) HandleRetailerConfirmCash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	retailerID := auth.ResolveRetailerOrgID(claims)
	if retailerID == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "retailer_scope_missing"})
		return
	}
	actorID := auth.ResolveRetailerUserID(claims)
	if actorID == "" {
		actorID = claims.Subject
	}
	status, amount, currency, err := s.SelectCashAtDelivery(r.Context(), orderID, retailerID, actorID)
	if err != nil {
		statusCode := http.StatusConflict
		if errors.Is(err, ErrOrderNotFound) {
			statusCode = http.StatusNotFound
		} else if errors.Is(err, ErrOrderForbidden) {
			statusCode = http.StatusForbidden
		} else if errors.Is(err, ErrInvalidStatusTransition) {
			statusCode = http.StatusConflict
		} else {
			statusCode = http.StatusInternalServerError
		}
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}

	resp := map[string]any{
		"success":      true,
		"order_id":     orderID,
		"state":        status,
		"amount_minor": amount,
		"currency":     currency,
		"message":      "awaiting_driver_cash_collection",
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// ── helpers ────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func retailerOrdersKey(retailerID string) string {
	return "orders:retailer:" + retailerID
}

func supplierOrdersKey(supplierID string) string {
	return "orders:supplier:" + supplierID
}

func dashboardCacheKeys(supplierID, warehouseID string) []string {
	keys := make([]string, 0, 2)
	if sid := strings.TrimSpace(supplierID); sid != "" {
		keys = append(keys, cache.DashboardKey("supplier", sid))
	}
	if wh := strings.TrimSpace(warehouseID); wh != "" {
		keys = append(keys, cache.DashboardKey("warehouse", wh))
	}
	return keys
}

func defaultOrderID() string {
	return fmt.Sprintf("ord_%s", strings.ReplaceAll(uuid.NewString(), "-", ""))
}

func resolveOrderPaymentMethod(ctx context.Context, client *spanner.Client, orderID string) string {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" || client == nil {
		return "CASH"
	}
	stmt := spanner.Statement{
		SQL: `SELECT Gateway FROM PaymentLedgerEntries
		      WHERE OrderId = @oid
		      ORDER BY OccurredAt DESC
		      LIMIT 1`,
		Params: map[string]any{"oid": orderID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return "CASH"
	}
	var gateway spanner.NullString
	if err := row.Columns(&gateway); err != nil || !gateway.Valid {
		return "CASH"
	}
	switch strings.ToUpper(strings.TrimSpace(gateway.StringVal)) {
	case "CASH", "":
		return "CASH"
	default:
		return "CARD"
	}
}

func resolveRetailerDisplayName(ctx context.Context, client *spanner.Client, retailerID string) string {
	retailerID = strings.TrimSpace(retailerID)
	if retailerID == "" {
		return ""
	}
	if client == nil {
		return retailerID
	}
	row, err := client.Single().ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{"Name"})
	if err != nil {
		return retailerID
	}
	var name spanner.NullString
	if err := row.Column(0, &name); err != nil || !name.Valid || strings.TrimSpace(name.StringVal) == "" {
		return retailerID
	}
	return strings.TrimSpace(name.StringVal)
}

// ConfirmAndAllocate runs multi-warehouse allocation and reserves stock inside one Spanner RW txn.
func (s *Service) ConfirmAndAllocate(ctx context.Context, orderID string) error {
	if !s.allocationRequired {
		return nil
	}
	if s.spannerClient == nil {
		return fmt.Errorf("spanner client unavailable for allocation")
	}
	orderID = strings.TrimSpace(orderID)
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		current, found, err := s.repo.GetOrderTxn(ctx, txn, orderID)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("order %s not found", orderID)
		}
		if current.Status != StatusPending {
			return fmt.Errorf("order %s in status %s cannot be allocated", orderID, current.Status)
		}
		return s.allocateAndReserveInTxn(ctx, txn, &current)
	})
	return err
}
