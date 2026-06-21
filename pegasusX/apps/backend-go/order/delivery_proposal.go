package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

var (
	ErrDeliveryProposalRequired = errors.New("delivery_proposal_required")
	ErrDeliveryProposalPending  = errors.New("delivery_proposal_pending")
)

func clearDeliveryProposal(o *Order) {
	o.ProposedDeliveryDate = nil
	o.DeliveryProposalAt = nil
	o.DeliveryProposalBy = ""
	o.DeliveryProposalReason = ""
}

func hasFutureDeliveryAnchor(now time.Time, o Order) bool {
	today := proximity.TashkentTodayStart(now)
	if o.RequestedDeliveryDate != nil {
		if calendarDaysBetween(today, proximity.TashkentTodayStart(*o.RequestedDeliveryDate)) >= 0 {
			return true
		}
	}
	if o.DeliverBefore != nil {
		if calendarDaysBetween(today, proximity.TashkentTodayStart(*o.DeliverBefore)) >= 0 {
			return true
		}
	}
	return false
}

func orderEligibleForDeliveryProposal(now time.Time, o Order) error {
	switch o.Status {
	case StatusCompleted, StatusCancelled, StatusLoaded, StatusInTransit:
		return fmt.Errorf("%w: %s cannot receive delivery proposal", ErrInvalidStatusTransition, o.Status)
	}
	if !hasFutureDeliveryAnchor(now, o) {
		return fmt.Errorf("%w: order has no future delivery date", ErrInvalidStatusTransition)
	}
	if IsScheduledPreorder(o) && PreorderEditLocked(now, o) {
		return fmt.Errorf("%w: preorder edit locked", ErrInvalidStatusTransition)
	}
	return nil
}

func applyAcceptedDeliveryDate(o *Order, proposed time.Time) {
	proposed = proposed.UTC()
	if IsScheduledPreorder(*o) || (o.Source == OrderSourceManualPreorder && (o.Status == StatusScheduled || o.Status == StatusAutoAccepted)) {
		o.RequestedDeliveryDate = &proposed
		return
	}
	o.DeliverBefore = &proposed
}

// WarehouseProposeDeliveryDate stores a pending warehouse proposal for retailer review.
func (s *Service) WarehouseProposeDeliveryDate(ctx context.Context, ops *auth.WarehouseOps, orderID string, req ProposeDeliveryDateRequest) (RetailerOrderLifecycleResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return RetailerOrderLifecycleResponse{}, errors.New("order_id required")
	}
	proposed, err := parseOptionalRFC3339(req.ProposedDeliveryDate)
	if err != nil || proposed == nil {
		if err == nil {
			err = errors.New("proposed_delivery_date required")
		}
		return RetailerOrderLifecycleResponse{}, fmt.Errorf("parse proposed_delivery_date: %w", err)
	}
	if strings.TrimSpace(req.Reason) == "" {
		return RetailerOrderLifecycleResponse{}, errors.New("reason required")
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
	if err := orderEligibleForDeliveryProposal(s.now(), current); err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}

	now := s.now()
	current.ProposedDeliveryDate = proposed
	current.DeliveryProposalAt = &now
	actorID := strings.TrimSpace(resolvedOps.Subject)
	if actorID == "" {
		actorID = resolvedOps.WarehouseID
	}
	current.DeliveryProposalBy = actorID
	current.DeliveryProposalReason = strings.TrimSpace(req.Reason)
	current.ConfirmationStatus = ConfirmationStatusPendingWarehouse
	current.UpdatedAt = now

	if err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		return emitPreorderEvent(ctx, txn, events.EventPreOrderDateProposed, current, string(auth.RoleWarehouse), actorID)
	}); err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	s.afterOrderMutation(ctx, current)
	return lifecycleResponse(current, current.Version, false), nil
}

// AcceptDeliveryProposal applies the warehouse-proposed date after retailer approval.
func (s *Service) AcceptDeliveryProposal(ctx context.Context, retailerID string, req AcceptDeliveryProposalRequest) (RetailerOrderLifecycleResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return RetailerOrderLifecycleResponse{}, errors.New("order_id required")
	}
	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	if !ok {
		return RetailerOrderLifecycleResponse{}, ErrOrderNotFound
	}
	if current.RetailerID != strings.TrimSpace(retailerID) {
		return RetailerOrderLifecycleResponse{}, ErrOrderForbidden
	}
	if current.ConfirmationStatus != ConfirmationStatusPendingWarehouse || current.ProposedDeliveryDate == nil {
		return RetailerOrderLifecycleResponse{}, ErrDeliveryProposalRequired
	}
	proposed := *current.ProposedDeliveryDate
	applyAcceptedDeliveryDate(&current, proposed)
	clearDeliveryProposal(&current)
	current.ConfirmationStatus = ConfirmationStatusConfirmed
	decisionAt := s.now()
	current.DecisionAt = &decisionAt
	current.DecisionBy = retailerID
	current.UpdatedAt = decisionAt

	if err := s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		return emitPreorderEvent(ctx, txn, events.EventPreOrderDateAccepted, current, string(auth.RoleRetailer), retailerID)
	}); err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	s.afterOrderMutation(ctx, current)
	return lifecycleResponse(current, current.Version, false), nil
}

// RejectDeliveryProposal cancels the order when the retailer declines a warehouse date proposal.
func (s *Service) RejectDeliveryProposal(ctx context.Context, retailerID string, req RejectDeliveryProposalRequest) (RetailerOrderLifecycleResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		return RetailerOrderLifecycleResponse{}, errors.New("order_id required")
	}
	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return RetailerOrderLifecycleResponse{}, err
	}
	if !ok {
		return RetailerOrderLifecycleResponse{}, ErrOrderNotFound
	}
	if current.RetailerID != strings.TrimSpace(retailerID) {
		return RetailerOrderLifecycleResponse{}, ErrOrderForbidden
	}
	if current.ConfirmationStatus != ConfirmationStatusPendingWarehouse {
		return RetailerOrderLifecycleResponse{}, ErrDeliveryProposalRequired
	}
	return s.cancelOrderWithReason(ctx, &current, retailerID, string(auth.RoleRetailer), strings.TrimSpace(req.Reason), events.EventPreOrderDateRejected)
}

// HandleWarehouseProposeDelivery serves POST /v1/warehouse/ops/orders/{id}/propose-delivery and preorder alias.
func (s *Service) HandleWarehouseProposeDelivery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	ops := auth.GetWarehouseOps(r.Context())
	orderID := strings.TrimSpace(chi.URLParam(r, "id"))
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
	var req ProposeDeliveryDateRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.WarehouseProposeDeliveryDate(r.Context(), ops, orderID, req)
	if err != nil {
		mapWarehouseOrderMutationError(w, r, orderID, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, bodyBytes, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}
