package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type warehouseOrderMutationRequest struct {
	Reason string `json:"reason"`
}

// WarehouseMarkDelayed transitions PENDING/LOADED → DELAYED for warehouse ops.
func (s *Service) WarehouseMarkDelayed(ctx context.Context, ops *auth.WarehouseOps, orderID, reason string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return errors.New("order_id required")
	}
	resolvedOps, err := s.resolveWarehouseOps(ctx, ops, orderID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(reason) == "" {
		reason = "WAREHOUSE_DELAY"
	}
	return s.warehouseTransition(ctx, resolvedOps, orderID, reason, func(current *Order) (Status, error) {
		switch current.Status {
		case StatusPending, StatusLoaded:
			return StatusDelayed, nil
		default:
			return "", fmt.Errorf("%w: %s cannot be delayed", ErrInvalidStatusTransition, current.Status)
		}
	}, false)
}

// WarehouseRejectOrder hard-cancels an order scoped to the warehouse (origin rejection).
func (s *Service) WarehouseRejectOrder(ctx context.Context, ops *auth.WarehouseOps, orderID, reason string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return errors.New("order_id required")
	}
	resolvedOps, err := s.resolveWarehouseOps(ctx, ops, orderID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(reason) == "" {
		return errors.New("reason required")
	}
	return s.warehouseTransition(ctx, resolvedOps, orderID, reason, func(current *Order) (Status, error) {
		switch current.Status {
		case StatusPending, StatusLoaded, StatusInTransit, StatusScheduled, StatusAutoAccepted:
			return StatusCancelled, nil
		default:
			return "", fmt.Errorf("%w: %s cannot be rejected", ErrInvalidStatusTransition, current.Status)
		}
	}, true)
}

// WarehousePayloadOverflow returns a loaded order to the dispatch pool (soft recovery).
func (s *Service) WarehousePayloadOverflow(ctx context.Context, ops *auth.WarehouseOps, orderID, reason string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return errors.New("order_id required")
	}
	resolvedOps, err := s.resolveWarehouseOps(ctx, ops, orderID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(reason) == "" {
		reason = "PAYLOAD_OVERFLOW"
	}
	return s.warehouseTransition(ctx, resolvedOps, orderID, reason, func(current *Order) (Status, error) {
		switch current.Status {
		case StatusLoaded, StatusInTransit:
			return StatusPending, nil
		default:
			return "", fmt.Errorf("%w: %s cannot overflow", ErrInvalidStatusTransition, current.Status)
		}
	}, true)
}

func (s *Service) warehouseTransition(
	ctx context.Context,
	ops *auth.WarehouseOps,
	orderID string,
	reason string,
	pickNext func(*Order) (Status, error),
	clearAssignment bool,
) error {
	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !ok {
		return ErrOrderNotFound
	}
	if err := assertWarehouseOrderScope(current, ops); err != nil {
		return err
	}

	nextStatus, err := pickNext(&current)
	if err != nil {
		return err
	}

	prevStatus := current.Status
	current.Status = nextStatus
	current.UpdatedAt = s.now()
	if clearAssignment {
		current.ManifestID = ""
		current.DriverID = ""
		current.VehicleID = ""
		current.RouteID = ""
	}

	actorID := ops.Subject
	if actorID == "" {
		actorID = ops.WarehouseID
	}

	err = s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		if nextStatus == StatusCancelled && current.Source == OrderSourceManualPreorder {
			return emitPreorderEvent(ctx, txn, events.EventPreOrderCancelled, current, string(auth.RoleWarehouse), actorID)
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:             events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: current.UpdatedAt.UTC().Format(time.RFC3339Nano)},
			OrderID:               current.OrderID,
			SupplierID:            current.SupplierID,
			RetailerID:            current.RetailerID,
			WarehouseID:           current.WarehouseID,
			DriverID:              current.DriverID,
			PreviousStatus:        string(prevStatus),
			Status:                string(current.Status),
			Reason:                strings.TrimSpace(reason),
			ActorRole:             string(auth.RoleWarehouse),
			ActorID:               actorID,
			OrderSource:           string(current.Source),
			ConfirmationStatus:    string(current.ConfirmationStatus),
			RequestedDeliveryDate: formatOptionalRFC3339(current.RequestedDeliveryDate),
		})
	})
	if err != nil {
		return fmt.Errorf("warehouse order mutation %s: %w", orderID, err)
	}

	if nextStatus == StatusCancelled && prevStatus != StatusCancelled {
		if err := s.releaseOrderReservations(ctx, &current); err != nil {
			s.log.Warn("release inventory reservation on warehouse cancel failed", "order_id", orderID, "err", err)
		}
	}

	s.afterOrderMutation(ctx, current)
	if s.cache != nil {
		s.cache.Invalidate(ctx, retailerOrdersKey(current.RetailerID), supplierOrdersKey(current.SupplierID))
	}
	return nil
}

// WarehouseEditPreorder allows warehouse staff to edit a scheduled pre-order until T-2.
func (s *Service) WarehouseEditPreorder(ctx context.Context, ops *auth.WarehouseOps, req EditPreorderRequest, reason string) (RetailerOrderLifecycleResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return RetailerOrderLifecycleResponse{}, errors.New("order_id required")
	}
	resolvedOps, err := s.resolveWarehouseOps(ctx, ops, orderID)
	if err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	if !ok {
		return RetailerOrderLifecycleResponse{}, ErrOrderNotFound
	}
	if err := assertWarehouseOrderScope(current, resolvedOps); err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	if current.Source != OrderSourceManualPreorder || current.Status != StatusScheduled {
		return RetailerOrderLifecycleResponse{}, ErrInvalidStatusTransition
	}
	if PreorderEditLocked(s.now(), current) {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("%w: warehouse preorder edit locked", ErrInvalidStatusTransition)
	}
	requestedDeliveryDate, err := parseOptionalRFC3339(req.RequestedDeliveryDate)
	if err != nil || requestedDeliveryDate == nil {
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("requested_delivery_date required")
	}
	if current.RequestedDeliveryDate == nil || !current.RequestedDeliveryDate.Equal(requestedDeliveryDate.UTC()) {
		return s.WarehouseProposeDeliveryDate(ctx, resolvedOps, orderID, ProposeDeliveryDateRequest{
			ProposedDeliveryDate: req.RequestedDeliveryDate,
			Reason:               strings.TrimSpace(reason),
		})
	}
	lineItems, total, err := s.normalizeAndQuoteLineItems(ctx, req.LineItems)
	if err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	current.LineItems = lineItems
	current.TotalMinor = total
	current.WarehouseNotes = strings.TrimSpace(reason)
	current.UpdatedAt = s.now()
	actorID := strings.TrimSpace(resolvedOps.Subject)
	if actorID == "" {
		actorID = resolvedOps.WarehouseID
	}
	if err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		return emitPreorderEvent(ctx, txn, events.EventPreOrderEdited, current, "WAREHOUSE_ADMIN", actorID)
	}); err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	s.afterOrderMutation(ctx, current)
	return lifecycleResponse(current, current.Version+1, false), nil
}

// ListOrdersForStockCommitment implements warehouse.OrderStockReader.
func (s *Service) ListOrdersForStockCommitment(ctx context.Context, warehouseID string, limit int) ([]Order, error) {
	return s.repo.ListOrdersForStockCommitment(ctx, warehouseID, limit)
}

// ListWarehousePreordersForOps returns scheduled/auto-accepted pre-orders for the warehouse home node.
func (s *Service) ListWarehousePreordersForOps(ctx context.Context, ops *auth.WarehouseOps, limit, offset int) ([]RetailerOrderLifecycleResponse, error) {
	if ops == nil || strings.TrimSpace(ops.WarehouseID) == "" {
		return nil, ErrOrderForbidden
	}
	orders, err := s.repo.ListWarehousePreorders(ctx, ops.WarehouseID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]RetailerOrderLifecycleResponse, 0, len(orders))
	for _, o := range orders {
		out = append(out, lifecycleResponse(o, o.Version, false))
	}
	return out, nil
}

type warehousePreorderEditBody struct {
	LineItems             []LineItem `json:"line_items"`
	RequestedDeliveryDate string     `json:"requested_delivery_date"`
	Reason                string     `json:"reason"`
}

// HandleWarehouseListPreorders serves GET /v1/warehouse/ops/preorders.
func (s *Service) HandleWarehouseListPreorders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	ops := auth.GetWarehouseOps(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 50
	}
	orders, err := s.ListWarehousePreordersForOps(r.Context(), ops, limit, offset)
	if err != nil {
		mapWarehouseOrderMutationError(w, r, "", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preorders": orders, "items": orders})
}

// HandleWarehouseEditPreorder serves POST /v1/warehouse/ops/preorders/{id}/edit.
func (s *Service) HandleWarehouseEditPreorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	ops := auth.GetWarehouseOps(r.Context())
	orderID := strings.TrimSpace(chi.URLParam(r, "id"))
	bodyBytes, err := readLimitedBody(r, 256*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, bodyBytes) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
	var body warehousePreorderEditBody
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.WarehouseEditPreorder(r.Context(), ops, EditPreorderRequest{
		OrderID:               orderID,
		LineItems:             body.LineItems,
		RequestedDeliveryDate: body.RequestedDeliveryDate,
	}, body.Reason)
	if err != nil {
		mapWarehouseOrderMutationError(w, r, orderID, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, bodyBytes, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleWarehouseRejectPreorder serves POST /v1/warehouse/ops/preorders/{id}/reject.
func (s *Service) HandleWarehouseRejectPreorder(w http.ResponseWriter, r *http.Request) {
	s.HandleWarehouseRejectOrder(w, r)
}

func assertWarehouseOrderScope(orderRecord Order, ops *auth.WarehouseOps) error {
	if ops.WarehouseID != "" && orderRecord.WarehouseID != ops.WarehouseID {
		return ErrOrderForbidden
	}
	if ops.SupplierID != "" && orderRecord.SupplierID != ops.SupplierID {
		return ErrOrderForbidden
	}
	return nil
}

// resolveWarehouseOps pins warehouse staff to JWT home node. Global supplier ADMIN
// (supplier portal cookie) may call compat routes without WarehouseOps in context;
// scope is derived from the target order's warehouse_id + supplier_id.
func (s *Service) resolveWarehouseOps(ctx context.Context, ops *auth.WarehouseOps, orderID string) (*auth.WarehouseOps, error) {
	if ops != nil && strings.TrimSpace(ops.WarehouseID) != "" {
		return ops, nil
	}

	claims, ok := auth.FromContext(ctx)
	if !ok || claims.Role != auth.RoleAdmin {
		return nil, ErrOrderForbidden
	}

	current, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !found {
		return nil, ErrOrderNotFound
	}
	if strings.TrimSpace(current.WarehouseID) == "" {
		return nil, ErrOrderForbidden
	}
	if claims.SupplierID != "" && current.SupplierID != "" && claims.SupplierID != current.SupplierID {
		return nil, ErrOrderForbidden
	}

	return &auth.WarehouseOps{
		WarehouseID: current.WarehouseID,
		SupplierID:  current.SupplierID,
		Subject:     claims.Subject,
	}, nil
}

// HandleWarehouseMarkDelayed serves POST /v1/warehouse/ops/orders/{id}/delay.
func (s *Service) HandleWarehouseMarkDelayed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	ops := auth.GetWarehouseOps(r.Context())
	orderID := strings.TrimSpace(chi.URLParam(r, "id"))
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id required"})
		return
	}
	bodyBytes, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, bodyBytes) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
	var body warehouseOrderMutationRequest
	_ = json.Unmarshal(bodyBytes, &body)

	if err := s.WarehouseMarkDelayed(r.Context(), ops, orderID, body.Reason); err != nil {
		mapWarehouseOrderMutationError(w, r, orderID, err)
		return
	}
	resp := map[string]string{"order_id": orderID, "status": string(StatusDelayed)}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, bodyBytes, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleWarehouseRejectOrder serves POST /v1/warehouse/ops/orders/{id}/reject.
func (s *Service) HandleWarehouseRejectOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	ops := auth.GetWarehouseOps(r.Context())
	orderID := strings.TrimSpace(chi.URLParam(r, "id"))
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id required"})
		return
	}
	bodyBytes, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, bodyBytes) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
	var body warehouseOrderMutationRequest
	if err := json.Unmarshal(bodyBytes, &body); err != nil || strings.TrimSpace(body.Reason) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reason required"})
		return
	}

	if err := s.WarehouseRejectOrder(r.Context(), ops, orderID, body.Reason); err != nil {
		mapWarehouseOrderMutationError(w, r, orderID, err)
		return
	}
	resp := map[string]string{"order_id": orderID, "status": "cancelled_by_origin"}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, bodyBytes, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleWarehousePayloadOverflow serves POST /v1/warehouse/ops/orders/{id}/overflow.
func (s *Service) HandleWarehousePayloadOverflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	ops := auth.GetWarehouseOps(r.Context())
	orderID := strings.TrimSpace(chi.URLParam(r, "id"))
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id required"})
		return
	}
	bodyBytes, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, bodyBytes) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
	var body warehouseOrderMutationRequest
	_ = json.Unmarshal(bodyBytes, &body)

	if err := s.WarehousePayloadOverflow(r.Context(), ops, orderID, body.Reason); err != nil {
		mapWarehouseOrderMutationError(w, r, orderID, err)
		return
	}
	resp := map[string]string{"order_id": orderID, "status": string(StatusPending)}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, bodyBytes, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func mapWarehouseOrderMutationError(w http.ResponseWriter, r *http.Request, orderID string, err error) {
	switch {
	case errors.Is(err, ErrOrderNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found", "order_id": orderID})
	case errors.Is(err, ErrOrderForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, ErrInvalidStatusTransition):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	default:
		slog.ErrorContext(r.Context(), "warehouse order mutation failed", "order_id", orderID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
	}
}
