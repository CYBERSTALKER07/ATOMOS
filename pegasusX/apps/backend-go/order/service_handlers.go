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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
)

func (s *Service) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	// B3 M-P0-4: multi-user JWT — org is tenant; Subject is staff person.
	retailerID := auth.ResolveRetailerOrgID(claims)
	if retailerID == "" {
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

	var req CreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	reqCtx := WithClientOrderID(r.Context(), idempotencyKeyFromRequest(r, body))
	resp, err := s.Create(reqCtx, retailerID, req)
	if err != nil {
		s.log.Warn("order create failed",
			"retailer_id", retailerID, "err", err)
		var approvalErr *OwnerApprovalRequiredError
		if errors.As(err, &approvalErr) {
			writeOwnerApprovalConflict(w, approvalErr)
			return
		}
		if writePackValidationOrMetadataError(w, err) {
			return
		}
		switch {
		case errors.Is(err, ErrZoneMiss):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": ErrZoneMiss.Error()})
		case errors.Is(err, ErrOrderAcceptanceClosed):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error(), "code": ErrOrderAcceptanceClosed.Error()})
		case errors.Is(err, ErrServiceabilityUnavailable):
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": ErrServiceabilityUnavailable.Error()})
		case errors.Is(err, ErrLineQuantityOutOfRange):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": ErrLineQuantityOutOfRange.Error(), "code": ErrLineQuantityOutOfRange.Error(), "message": err.Error()})
		case errors.Is(err, ErrCurrencyNotAllowed):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": ErrCurrencyNotAllowed.Error(), "code": ErrCurrencyNotAllowed.Error()})
		case errors.Is(err, ErrCreditLimitBreached), errors.Is(err, credit.ErrLimitBreached):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"error":           "credit_limit_breached",
				"code":            "CREDIT_LIMIT_BREACHED",
				"action_guidance": "Wire bank payment or settle outstanding overdue invoices to restore credit headroom.",
			})
		case errors.Is(err, auth.ErrMarketPackUnknown), errors.Is(err, auth.ErrMarketPackNotShipped),
			errors.Is(err, auth.ErrPackCurrencyMismatch), errors.Is(err, auth.ErrGeographyIncomplete),
			errors.Is(err, auth.ErrCrossMarketDeferred):
			st, code := auth.CheckoutPackHTTPStatus(err)
			writeJSON(w, st, map[string]string{"error": code})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusCreated, respBytes)
}

// HandleUpdateStatus is PATCH /v1/order/{orderID}/status.
func (s *Service) HandleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := chi.URLParam(r, "orderID")
	if strings.TrimSpace(orderID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
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

	var req UpdateStatusRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.UpdateStatus(r.Context(), claims, orderID, req)
	if err != nil {
		s.log.Warn("order status update failed", "order_id", orderID, "err", err)
		switch {
		case errors.Is(err, ErrOrderNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		case errors.Is(err, ErrOrderForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		case errors.Is(err, ErrInvalidStatusTransition):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_status_transition"})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// ValidateQR checks a scanned delivery token against the order without advancing lifecycle.
func (s *Service) ValidateQR(ctx context.Context, claims auth.Claims, req ValidateQRRequest) (ValidateQRResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	token := strings.TrimSpace(req.ScannedToken)
	if orderID == "" || token == "" {
		return ValidateQRResponse{}, errors.New("order_id and scanned_token required")
	}

	orderRecord, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return ValidateQRResponse{}, fmt.Errorf("load order %s: %w", orderID, err)
	}
	if !found {
		return ValidateQRResponse{}, ErrOrderNotFound
	}
	if claims.Role == auth.RoleDriver && strings.TrimSpace(orderRecord.DriverID) != strings.TrimSpace(claims.Subject) {
		return ValidateQRResponse{}, ErrOrderForbidden
	}
	if err := s.validateDeliveryToken(orderRecord, token); err != nil {
		return ValidateQRResponse{}, err
	}

	resp := driverOrderResponse(orderRecord, "")
	retailerName := resolveRetailerDisplayName(ctx, s.spannerClient, orderRecord.RetailerID)
	return ValidateQRResponse{
		OrderID:      orderRecord.OrderID,
		RetailerName: retailerName,
		TotalAmount:  orderRecord.TotalMinor,
		State:        orderRecord.Status,
		Items:        resp.Items,
	}, nil
}

// HandleValidateQR is POST /v1/order/validate-qr.
func (s *Service) HandleValidateQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req ValidateQRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	resp, err := s.ValidateQR(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "validate qr failed", req.OrderID, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleGetQRPayload serves GET /v1/order/{orderID}/qr-payload.
func (s *Service) HandleGetQRPayload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := chi.URLParam(r, "orderID")
	if strings.TrimSpace(orderID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	current, found, err := s.loadOrderForRequest(r.Context(), orderID)
	if err != nil {
		s.log.ErrorContext(r.Context(), "get order for qr payload failed", "err", err, "order_id", orderID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if claims.Role == auth.RoleRetailer && current.RetailerID != auth.ResolveRetailerOrgID(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	qrToken := s.publicDeliveryToken(current)

	writeJSON(w, http.StatusOK, map[string]string{
		"order_id": orderID,
		"qr_token": qrToken,
	})
}

// HandleDeliveryScanQR serves POST /v1/delivery/scan-qr.
func (s *Service) HandleDeliveryScanQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		OrderID         string     `json:"order_id"`
		QRToken         string     `json:"qr_token"`
		Latitude        *float64   `json:"latitude"`
		Longitude       *float64   `json:"longitude"`
		ClientTimestamp *time.Time `json:"client_timestamp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()

	if strings.TrimSpace(req.OrderID) == "" || strings.TrimSpace(req.QRToken) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id and qr_token required"})
		return
	}

	result, err := s.transitionDriverOrder(r.Context(), claims, driverTransitionRequest{
		OrderID:         req.OrderID,
		NextStatus:      StatusAwaitingPayment,
		Reason:          "qr_scanned",
		ClientTimestamp: req.ClientTimestamp,
		Precheck: func(orderRecord Order) error {
			return s.validateDeliveryToken(orderRecord, req.QRToken)
		},
		EmitExtra: func(txn outbox.TxnBuffer, orderRecord Order, _ Status) error {
			if err := s.emitSettlementRequired(r.Context(), txn, orderRecord); err != nil {
				return err
			}
			return s.emitPaymentRequired(r.Context(), txn, orderRecord)
		},
	})

	if err != nil {
		s.log.Warn("delivery scan qr failed", "order_id", req.OrderID, "err", err)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"valid":    true,
		"order_id": result.Order.OrderID,
		"state":    result.Order.Status,
	})
}

// HandleAssignOrder is POST /v1/orders/{orderID}/assign.
func (s *Service) HandleAssignOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := chi.URLParam(r, "orderID")
	if strings.TrimSpace(orderID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
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

	var req AssignOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.AssignOrder(r.Context(), claims, orderID, req)
	if err != nil {
		s.log.Warn("order assignment failed", "order_id", orderID, "err", err)
		switch {
		case errors.Is(err, ErrOrderNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		case errors.Is(err, ErrOrderForbidden):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}

	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleMarkArrived is POST /v1/delivery/arrive.
func (s *Service) HandleMarkArrived(w http.ResponseWriter, r *http.Request) {
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

	resp, err := s.MarkArrived(r.Context(), claims, req.OrderID)
	if err != nil {
		s.writeOrderMutationError(w, "driver mark arrived failed", req.OrderID, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleSubmitDelivery is POST /v1/order/deliver.
func (s *Service) HandleSubmitDelivery(w http.ResponseWriter, r *http.Request) {
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

	var req DeliverySubmitRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.SubmitDelivery(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "driver delivery submit failed", req.OrderID, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleConfirmOffload is POST /v1/order/confirm-offload.
func (s *Service) HandleConfirmOffload(w http.ResponseWriter, r *http.Request) {
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

	var req ConfirmOffloadRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.ConfirmOffload(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "driver confirm offload failed", req.OrderID, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleCompleteOrder is POST /v1/order/complete.
func (s *Service) HandleCompleteOrder(w http.ResponseWriter, r *http.Request) {
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

	var req CompleteOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.CompleteOrder(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "driver complete order failed", req.OrderID, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleCollectCash is POST /v1/order/collect-cash.
func (s *Service) HandleCollectCash(w http.ResponseWriter, r *http.Request) {
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

	var req CollectCashRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.CollectCash(r.Context(), claims, req)
	if err != nil {
		s.writeOrderMutationError(w, "driver collect cash failed", req.OrderID, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func (s *Service) writeOrderMutationError(w http.ResponseWriter, operation string, orderID string, err error) {
	s.log.Warn(operation, "order_id", orderID, "err", err)
	if writePackValidationOrMetadataError(w, err) {
		return
	}
	switch {
	case errors.Is(err, ErrOrderNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
	case errors.Is(err, ErrOrderForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, ErrInvalidStatusTransition):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":   "invalid_status_transition",
			"message": err.Error(),
		})
	case errors.Is(err, ErrGeofenceViolation):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "geofence_violation"})
	case errors.Is(err, ErrAssignmentRequired):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "assignment_required"})
	case errors.Is(err, ErrServiceabilityUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "delivery_perimeter_unavailable"})
	case errors.Is(err, ErrFiscalNotFailed):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "fiscal_not_failed"})
	case errors.Is(err, ErrForceCompleteForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "force_complete_forbidden"})
	case errors.Is(err, ErrForceReasonRequired):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "force_reason_required"})
	case errors.Is(err, ErrForceReasonInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "force_reason_invalid"})
	case errors.Is(err, ErrFiscalAlreadySucceeded):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "fiscal_already_succeeded"})
	case errors.Is(err, ErrCashAmountNegative):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cash_amount_negative"})
	case errors.Is(err, auth.ErrMarketPackUnknown), errors.Is(err, auth.ErrMarketPackNotShipped),
		errors.Is(err, auth.ErrFiscalAdapterUnimplemented), errors.Is(err, auth.ErrFakeFiscalForbidden),
		errors.Is(err, auth.ErrBreachRadiusInvalid), errors.Is(err, auth.ErrTimezoneInvalid),
		errors.Is(err, auth.ErrShopClosedGraceInvalid):
		st, code := auth.TimezonePackHTTPStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
	default:
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	}
}

func writePackValidationOrMetadataError(w http.ResponseWriter, err error) bool {
	var packErr *PackValidationError
	if errors.As(err, &packErr) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":       ErrPackMultipleRequired.Error(),
			"code":        ErrPackMultipleRequired.Error(),
			"message":     packErr.Message,
			"line_errors": packErr.LineErrors,
			"suggestions": packErr.Suggestions,
		})
		return true
	}
	if errors.Is(err, ErrPackMetadataUnavailable) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": ErrPackMetadataUnavailable.Error(),
			"code":  "PACK_METADATA_UNAVAILABLE",
		})
		return true
	}
	return false
}

// HandleFiscalRetry is POST /v1/order/{orderID}/fiscal/retry
func (s *Service) HandleFiscalRetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := strings.TrimSpace(chiURLParam(r, "orderID"))
	if orderID == "" {
		// fallback for body-based clients
		var body struct {
			OrderID string `json:"order_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		orderID = strings.TrimSpace(body.OrderID)
	}
	resp, err := s.RetryFiscal(r.Context(), claims, orderID)
	if err != nil {
		s.writeOrderMutationError(w, "fiscal retry failed", orderID, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleForceComplete is POST /v1/order/{orderID}/force-complete
func (s *Service) HandleForceComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := strings.TrimSpace(chiURLParam(r, "orderID"))
	var body struct {
		OrderID    string `json:"order_id"`
		ReasonCode string `json:"reason_code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if orderID == "" {
		orderID = strings.TrimSpace(body.OrderID)
	}
	resp, err := s.ForceCompleteOrder(r.Context(), claims, orderID, body.ReasonCode)
	if err != nil {
		s.writeOrderMutationError(w, "force complete failed", orderID, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func chiURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// ── wire shapes ────────────────────────────────────────────────────────────

func orderAssignedData(orderRecord Order) map[string]any {
	data := map[string]any{
		"type":         events.EventOrderAssigned,
		"order_id":     orderRecord.OrderID,
		"supplier_id":  orderRecord.SupplierID,
		"retailer_id":  orderRecord.RetailerID,
		"warehouse_id": orderRecord.WarehouseID,
		"driver_id":    orderRecord.DriverID,
		"route_id":     orderRecord.RouteID,
		"timestamp":    orderRecord.UpdatedAt.Format(time.RFC3339Nano),
	}
	addOptionalAssignmentFields(data, orderRecord)
	return data
}

func orderReassignedData(orderRecord Order, previousDriverID string, previousRouteID string) map[string]any {
	data := map[string]any{
		"type":           events.EventOrderReassigned,
		"order_id":       orderRecord.OrderID,
		"supplier_id":    orderRecord.SupplierID,
		"retailer_id":    orderRecord.RetailerID,
		"warehouse_id":   orderRecord.WarehouseID,
		"from_driver_id": previousDriverID,
		"to_driver_id":   orderRecord.DriverID,
		"from_route_id":  previousRouteID,
		"to_route_id":    orderRecord.RouteID,
		"timestamp":      orderRecord.UpdatedAt.Format(time.RFC3339Nano),
	}
	addOptionalAssignmentFields(data, orderRecord)
	return data
}

func addOptionalAssignmentFields(data map[string]any, orderRecord Order) {
	if strings.TrimSpace(orderRecord.VehicleID) != "" {
		data["vehicle_id"] = orderRecord.VehicleID
	}
	if strings.TrimSpace(orderRecord.ManifestID) != "" {
		data["manifest_id"] = orderRecord.ManifestID
	}
}

func assignmentEventType(previousDriverID string) string {
	if strings.TrimSpace(previousDriverID) != "" {
		return events.EventOrderReassigned
	}
	return events.EventOrderAssigned
}

func assignmentResponse(orderRecord Order, eventType string, version int64, noChange bool) AssignOrderResponse {
	return AssignOrderResponse{
		OrderID:    orderRecord.OrderID,
		SupplierID: orderRecord.SupplierID,
		RetailerID: orderRecord.RetailerID,
		DriverID:   orderRecord.DriverID,
		VehicleID:  orderRecord.VehicleID,
		RouteID:    orderRecord.RouteID,
		ManifestID: orderRecord.ManifestID,
		EventType:  eventType,
		Version:    version,
		UpdatedAt:  orderRecord.UpdatedAt.Format(time.RFC3339Nano),
		NoChange:   noChange,
	}
}

func (s *Service) afterOrderMutation(ctx context.Context, orderRecord Order) {
	if s.cache != nil {
		keys := []string{
			retailerOrdersKey(orderRecord.RetailerID),
			supplierOrdersKey(orderRecord.SupplierID),
		}
		if orderRecord.Status == StatusCancelled {
			keys = append(keys, "catalog:products:"+orderRecord.SupplierID)
		}
		keys = append(keys, dashboardCacheKeys(orderRecord.SupplierID, orderRecord.WarehouseID)...)
		s.cache.Invalidate(ctx, keys...)
	}
	if s.dispatchPlanWarm != nil {
		if wh := strings.TrimSpace(orderRecord.WarehouseID); wh != "" {
			s.dispatchPlanWarm(ctx, wh)
		}
	}
}

func (s *Service) replanAsync(routeID, reason, actor string) {
	if s == nil || s.replanner == nil || routeID == "" {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil && s.log != nil {
				s.log.Error("async route replanner panic recovered", "panic", r, "route_id", routeID, "reason", reason)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.replanner.ReplanRoute(ctx, routeID, reason, actor); err != nil && s.log != nil {
			s.log.Warn("async route replan failed", "err", err, "route_id", routeID, "reason", reason)
		}
	}()
}

// SetDispatchPlanWarm schedules background smart-dispatch preview recompute for a warehouse.
func (s *Service) SetDispatchPlanWarm(fn func(ctx context.Context, warehouseID string)) {
	s.dispatchPlanWarm = fn
}

func (s *Service) emitSettlementRequired(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, s.settlementRequiredData(ctx, orderRecord))
}

func (s *Service) emitPaymentRequired(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, s.paymentRequiredData(ctx, orderRecord))
}

func emitPaymentCleared(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order, method string) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, paymentClearedData(orderRecord, method))
}

func emitOrderFinalized(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, orderFinalizedData(orderRecord))
}

func (s *Service) paymentRequiredData(ctx context.Context, orderRecord Order) map[string]any {
	data := map[string]any{
		"type":           events.EventPaymentRequired,
		"order_id":       orderRecord.OrderID,
		"supplier_id":    orderRecord.SupplierID,
		"retailer_id":    orderRecord.RetailerID,
		"amount":         moneyData(orderRecord),
		"amount_minor":   orderRecord.TotalMinor,
		"currency":       orderRecord.Currency,
		"payment_method": "CASH",
		"gateway":        "CASH",
		"status":         string(orderRecord.Status),
		"timestamp":      orderRecord.UpdatedAt.Format(time.RFC3339Nano),
	}
	if driverID := strings.TrimSpace(orderRecord.DriverID); driverID != "" {
		data["driver_id"] = driverID
	}
	originalAmount := orderOriginalAmountMinor(orderRecord)
	if originalAmount > orderRecord.TotalMinor {
		data["original_amount"] = originalAmount
	}
	if s != nil && s.gatewayPolicy != nil {
		gateways, acceptor, err := s.gatewayPolicy.AllowedGateways(ctx, orderRecord.SupplierID, orderRecord.WarehouseID)
		if err == nil {
			data["payment_acceptor"] = acceptor
			data["available_card_gateways"] = cardGatewaysOnly(ctx, gateways)
		}
	}
	return data
}

func cardGatewaysOnly(ctx context.Context, gateways []string) []string {
	out := make([]string, 0, len(gateways))
	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return out
	}
	for _, gateway := range gateways {
		canon := auth.CanonicalPSP(gateway)
		if canon == "" || canon == "CASH" {
			continue
		}
		if !auth.PackAllowsPSP(pack, canon) {
			continue
		}
		out = append(out, canon)
	}
	return out
}

func (s *Service) settlementRequiredData(ctx context.Context, orderRecord Order) map[string]any {
	data := s.paymentRequiredData(ctx, orderRecord)
	data["type"] = events.EventSettlementRequired
	return data
}

func paymentClearedData(orderRecord Order, method string) map[string]any {
	data := map[string]any{
		"type":           events.EventPaymentCleared,
		"order_id":       orderRecord.OrderID,
		"supplier_id":    orderRecord.SupplierID,
		"retailer_id":    orderRecord.RetailerID,
		"amount":         moneyData(orderRecord),
		"amount_minor":   orderRecord.TotalMinor,
		"currency":       orderRecord.Currency,
		"payment_method": method,
		"gateway":        method,
		"status":         string(orderRecord.Status),
		"timestamp":      orderRecord.UpdatedAt.Format(time.RFC3339Nano),
	}
	if driverID := strings.TrimSpace(orderRecord.DriverID); driverID != "" {
		data["driver_id"] = driverID
	}
	return data
}

func orderFinalizedData(orderRecord Order) map[string]any {
	data := map[string]any{
		"type":         events.EventOrderFinalized,
		"order_id":     orderRecord.OrderID,
		"supplier_id":  orderRecord.SupplierID,
		"retailer_id":  orderRecord.RetailerID,
		"total":        moneyData(orderRecord),
		"amount_minor": orderRecord.TotalMinor,
		"currency":     orderRecord.Currency,
		"status":       string(orderRecord.Status),
		"timestamp":    orderRecord.UpdatedAt.Format(time.RFC3339Nano),
	}
	if driverID := strings.TrimSpace(orderRecord.DriverID); driverID != "" {
		data["driver_id"] = driverID
	}
	return data
}

func moneyData(orderRecord Order) map[string]any {
	return map[string]any{
		"amount":   orderRecord.TotalMinor,
		"currency": orderRecord.Currency,
	}
}

func (s *Service) emitCreditLimitBreached(ctx context.Context, retailerID string, requestedAmount int64, check credit.CheckResult) error {
	if s.spannerClient == nil {
		return nil
	}
	return spannerutils.RunReadWriteTransaction(ctx, s.spannerClient, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateCreditProfile, retailerID, events.TopicMain, events.CreditLimitEvent{
			BaseEvent:        events.BaseEvent{Type: events.EventRetailerCreditLimitBreached, Timestamp: s.now().Format(time.RFC3339Nano)},
			OrderID:          "", // unknown at this pre-order stage; consumers key on retailer_id
			RetailerID:       retailerID,
			SupplierID:       s.resolveSupplierScope(ctx),
			RequestedAmount:  requestedAmount,
			CreditLimitMinor: check.CreditLimitMinor,
			CurrentBalance:   check.CurrentBalance,
		}); err != nil {
			return err
		}
		var mutations []*spanner.Mutation
		for _, e := range buf.events {
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(e)))
		}
		return txn.BufferWrite(mutations)
	})
}

func driverOrderResponse(orderRecord Order, message string) DriverOrderResponse {
	items := make([]DriverOrderLineItem, 0, len(orderRecord.LineItems))
	for _, lineItem := range orderRecord.LineItems {
		items = append(items, DriverOrderLineItem{
			ProductID:   lineItem.SKU,
			ProductName: lineItem.Name,
			Quantity:    lineItem.Quantity,
			UnitPrice:   lineItem.UnitPrice,
			LineTotal:   lineItem.UnitPrice * lineItem.Quantity,
		})
	}

	stage := ToCustomerStage(orderRecord.Status)
	return DriverOrderResponse{
		ID:                      orderRecord.OrderID,
		OrderID:                 orderRecord.OrderID,
		RetailerID:              orderRecord.RetailerID,
		RetailerName:            orderRecord.RetailerID,
		State:                   orderRecord.Status,
		Status:                  string(orderRecord.Status),
		CustomerStage:           stage,
		CustomerProgressPercent: CustomerStageProgressPercent(stage),
		TotalAmount:             orderRecord.TotalMinor,
		Currency:                orderRecord.Currency,
		DeliveryAddress:         "",
		Latitude:                orderRecord.Lat,
		Longitude:               orderRecord.Lng,
		CreatedAt:               orderRecord.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:               orderRecord.UpdatedAt.Format(time.RFC3339Nano),
		Items:                   items,
		Message:                 message,
	}
}

func (r DeliverySubmitRequest) token() string {
	if strings.TrimSpace(r.QRToken) != "" {
		return r.QRToken
	}
	return r.ScannedToken
}
