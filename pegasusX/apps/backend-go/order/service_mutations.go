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
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/loyalty"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func (s *Service) Create(ctx context.Context, retailerID string, req CreateRequest) (CreateResponse, error) {
	if err := req.Validate(); err != nil {
		return CreateResponse{}, err
	}
	if retailerID == "" {
		return CreateResponse{}, errors.New("retailer_id required from session")
	}

	supplierID, err := s.resolveSupplierIDForCreate(ctx, req.SupplierID)
	if err != nil {
		return CreateResponse{}, err
	}

	orderCurrency, err := s.resolveOrderCurrency(ctx, supplierID, req.Currency)
	if err != nil {
		return CreateResponse{}, err
	}

	requestedDeliveryDate, err := parseOptionalRFC3339(req.RequestedDeliveryDate)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("parse requested_delivery_date: %w", err)
	}

	lineItems, total, err := s.normalizeAndQuoteLineItems(ctx, req.LineItems, requestedDeliveryDate)
	if err != nil {
		return CreateResponse{}, err
	}
	deliverBefore, err := parseOptionalRFC3339(req.DeliverBefore)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("parse deliver_before: %w", err)
	}
	now := s.now()
	priority := normalizeDeliveryPriority(req.DeliveryPriority)

	if s.warehouse == nil {
		return CreateResponse{}, fmt.Errorf("%w: warehouse_resolver_unavailable", ErrServiceabilityUnavailable)
	}

	store, err := s.resolveCheckoutStore(ctx, retailerID, req.Lat, req.Lng)
	if err != nil {
		return CreateResponse{}, err
	}
	if pack, packErr := auth.CheckoutPackFromContext(ctx); packErr == nil {
		if packCountry, countryErr := auth.PackCountryCode(pack); countryErr == nil {
			if err := auth.AssertSameMarket(packCountry, store.CountryCode); err != nil {
				return CreateResponse{}, err
			}
		}
	}
	resolvedWarehouseID, err := s.warehouse.ResolveNearestWarehouseID(ctx, supplierID, store)
	if err != nil {
		return CreateResponse{}, mapWarehouseResolveError(err)
	}
	warehouseID := strings.TrimSpace(resolvedWarehouseID)
	if warehouseID == "" {
		return CreateResponse{}, fmt.Errorf("%w: no_eligible_warehouse", ErrZoneMiss)
	}

	var whPolicy WarehouseOpsPolicy
	var deliveryFeeMinor int64
	if s.spannerClient != nil {
		whPolicy, err = LoadWarehouseOpsPolicy(ctx, s.spannerClient, warehouseID)
		if err != nil {
			return CreateResponse{}, fmt.Errorf("load warehouse policy: %w", err)
		}
		if lineErrs := ValidateLineQuantities(lineItems, whPolicy); len(lineErrs) > 0 {
			return CreateResponse{}, fmt.Errorf("%w: %v", ErrLineQuantityOutOfRange, lineErrs)
		}
		if packErrs := validatePackMultiples(ctx, s.spannerClient, lineItems); len(packErrs) > 0 {
			return CreateResponse{}, fmt.Errorf("%w: %v", ErrLineQuantityOutOfRange, packErrs)
		}
		if open, _, _, closedMsg := checkOrderAcceptanceGate(whPolicy, now); !open {
			return CreateResponse{}, fmt.Errorf("%w: %s", ErrOrderAcceptanceClosed, closedMsg)
		}
		deliveryFeeMinor, _ = ComputeOrderDeliveryFee(whPolicy, store.Lat, store.Lng)
	}

	loc, locErr := auth.TimezoneFromContext(ctx, supplierID)
	if locErr != nil {
		return CreateResponse{}, locErr
	}
	if whPolicy.OperatingSchedule.Timezone != "" {
		if l, err := time.LoadLocation(whPolicy.OperatingSchedule.Timezone); err == nil {
			loc = l
		}
	}

	source, status, confirmation, committedDate, latestDelivery, classifyErr := ClassifyDelivery(
		now, loc, req.DeliveryMode, requestedDeliveryDate, deliverBefore,
		whPolicy.PreorderMinLeadDays, whPolicy.PreorderMaxLeadDays,
	)
	if classifyErr != nil {
		return CreateResponse{}, classifyErr
	}
	// Auto-order place (and other explicit sources) override ClassifyDelivery provenance.
	if req.Source != "" {
		source = req.Source
		if source == OrderSourceAutoOrder {
			status = StatusPending
		}
	}

	var invPlan InventoryPlan
	if s.spannerClient != nil {
		policyOverride, _ := s.resolveCheckoutPolicyOverride(whPolicy, req.CheckoutPolicyToken)
		invPlan, err = PlanInventoryCheckout(ctx, s.spannerClient, supplierID, warehouseID, lineItems, policyOverride)
		if err != nil {
			return CreateResponse{}, err
		}
		lineItems = invPlan.Fulfillable

		if s.cache != nil {
			var inflightKeys []string
			var inflightQtys []int64
			defer func() {
				for i, key := range inflightKeys {
					_, _ = s.cache.DecrBy(context.Background(), key, inflightQtys[i])
				}
			}()

			for _, li := range invPlan.Fulfillable {
				sku := strings.TrimSpace(li.SKU)
				if sku == "" {
					continue
				}
				key := fmt.Sprintf("inflight_rsv:%s:%s", warehouseID, sku)
				inflight, cerr := s.cache.IncrBy(ctx, key, li.Quantity)
				if cerr != nil {
					continue
				}
				inflightKeys = append(inflightKeys, key)
				inflightQtys = append(inflightQtys, li.Quantity)

				avail := invPlan.AvailableStock[sku]
				if inflight > avail {
					return CreateResponse{}, &InventoryCheckoutError{
						Code:         "PARTIAL_OUT_OF_STOCK_REJECTED",
						Message:      "inventory_exhausted_concurrent",
						OOSItems:     []string{sku},
						RejectedSKUs: []string{sku},
						Shortfall:    map[string]int64{sku: inflight - avail},
					}
				}
			}
		}

		total = 0
		for _, li := range lineItems {
			total += li.Quantity * li.UnitPrice
		}
		total += deliveryFeeMinor
	} else {
		invPlan.Fulfillable = lineItems
		total += deliveryFeeMinor
	}

	// Credit check is credit-path only. Cash/card/COD orders must not require a profile.
	// Optional reserve at create when CREDIT_RESERVE_AT_CREATE=1 and relationship allows credit.
	if s.credit != nil && total > 0 && creditReserveAtCreateEnabled() {
		check, err := s.credit.CheckCreditPath(ctx, retailerID, supplierID, total)
		if err != nil {
			return CreateResponse{}, fmt.Errorf("credit check failed: %w", err)
		}
		if !check.Allowed {
			// Soft-skip: create still proceeds without reserve (cash/card remain available).
			s.log.Info("credit reserve skipped", "reason", check.Reason, "retailer_id", retailerID, "supplier_id", supplierID)
		}
	}

	if len(lineItems) == 0 && len(invPlan.Backorder) == 0 {
		return CreateResponse{}, ErrInventoryExhausted
	}

	// All lines are backordered — single BACKORDERED order (no empty primary).
	if len(lineItems) == 0 && len(invPlan.Backorder) > 0 {
		boID, err := s.createBackorderOrder(ctx, retailerID, warehouseID, "", invPlan.Backorder, Order{
			H3Cell:   req.H3Cell,
			Lat:      req.Lat,
			Lng:      req.Lng,
			Currency: orderCurrency,
		})
		if err != nil {
			return CreateResponse{}, fmt.Errorf("persist backorder: %w", err)
		}
		var boTotal int64
		for _, li := range invPlan.Backorder {
			boTotal += li.Quantity * li.UnitPrice
		}
		return CreateResponse{
			OrderID:              boID,
			WarehouseID:          warehouseID,
			Status:               StatusBackordered,
			Source:               OrderSourceBackorder,
			ConfirmationStatus:   ConfirmationStatusConfirmed,
			TotalMinor:           boTotal,
			Currency:             orderCurrency,
			CreatedAt:            s.now().Format(time.RFC3339Nano),
			BackorderedItemCount: invPlan.BackorderCount,
			StockWarnings:        invPlan.Warnings,
		}, nil
	}

	o := Order{
		OrderID:               s.newID(),
		SupplierID:            supplierID,
		RetailerID:            retailerID,
		ParentOrderID:         strings.TrimSpace(req.ParentOrderID),
		WarehouseID:           warehouseID,
		Status:                status,
		Source:                source,
		ConfirmationStatus:    confirmation,
		LineItems:             lineItems,
		TotalMinor:            total,
		Currency:              orderCurrency,
		H3Cell:                req.H3Cell,
		Lat:                   req.Lat,
		Lng:                   req.Lng,
		RequestedDeliveryDate: committedDate,
		DeliverBefore:         latestDelivery,
		DeliveryPriority:      priority,
		DeliveryFeeMinor:      deliveryFeeMinor,
		Version:               1,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	stockOpts := StockReservationOpts{
		Skip: deferStockReservationAtCreate(s.allocationRequired, status, source, warehouseID, len(lineItems)),
	}

	var bo *Order
	if len(invPlan.Backorder) > 0 {
		var boTotal int64
		for _, li := range invPlan.Backorder {
			boTotal += li.Quantity * li.UnitPrice
		}
		boCurrency := strings.TrimSpace(o.Currency)
		if boCurrency == "" {
			boCurrency = s.currency
		}
		bo = &Order{
			OrderID:            s.newID(),
			SupplierID:         o.SupplierID,
			RetailerID:         o.RetailerID,
			WarehouseID:        o.WarehouseID,
			Status:             StatusBackordered,
			Source:             OrderSourceBackorder,
			ConfirmationStatus: ConfirmationStatusConfirmed,
			LineItems:          invPlan.Backorder,
			TotalMinor:         boTotal,
			Currency:           boCurrency,
			H3Cell:             o.H3Cell,
			Lat:                o.Lat,
			Lng:                o.Lng,
			DerivedFromOrderID: o.OrderID,
			Version:            1,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
	}

	inTxn := func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if s.credit != nil && total > 0 && creditReserveAtCreateEnabled() {
			check, cerr := s.credit.CheckCreditPath(ctx, retailerID, supplierID, total)
			if cerr != nil {
				return cerr
			}
			if !check.Allowed {
				return ErrCreditLimitBreached
			}
			if rerr := s.credit.ReserveOrderInTxn(ctx, txn, retailerID, supplierID, o.OrderID, total); rerr != nil {
				if errors.Is(rerr, credit.ErrLimitBreached) || errors.Is(rerr, credit.ErrCreditNotEnabled) {
					return ErrCreditLimitBreached
				}
				return fmt.Errorf("credit reserve failed: %w", rerr)
			}
		}
		return nil
	}

	err = s.repo.CreateOrderWithBackorder(ctx, &o, bo, inTxn, func(txn outbox.TxnBuffer) error {
		if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, o.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:             events.BaseEvent{Type: events.EventOrderCreated, Timestamp: o.CreatedAt.Format(time.RFC3339Nano)},
			OrderID:               o.OrderID,
			SupplierID:            o.SupplierID,
			RetailerID:            o.RetailerID,
			WarehouseID:           o.WarehouseID,
			Status:                string(o.Status),
			OrderSource:           string(o.Source),
			ConfirmationStatus:    string(o.ConfirmationStatus),
			TotalMinor:            o.TotalMinor,
			Currency:              o.Currency,
			H3Cell:                o.H3Cell,
			Lat:                   o.Lat,
			Lng:                   o.Lng,
			RequestedDeliveryDate: formatOptionalRFC3339(o.RequestedDeliveryDate),
			ReceivingWindowOpen:   o.ReceivingWindowOpen,
			ReceivingWindowClose:  o.ReceivingWindowClose,
			LineItems:             o.LineItems,
		}); err != nil {
			return err
		}
		if bo != nil {
			if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, bo.OrderID, events.TopicMain, events.OrderEvent{
				BaseEvent:          events.BaseEvent{Type: events.EventOrderCreated, Timestamp: bo.CreatedAt.Format(time.RFC3339Nano)},
				OrderID:            bo.OrderID,
				SupplierID:         bo.SupplierID,
				RetailerID:         bo.RetailerID,
				WarehouseID:        bo.WarehouseID,
				Status:             string(bo.Status),
				OrderSource:        string(bo.Source),
				ConfirmationStatus: string(bo.ConfirmationStatus),
				TotalMinor:         bo.TotalMinor,
				Currency:           bo.Currency,
				LineItems:          bo.LineItems,
			}); err != nil {
				return err
			}
		}
		if o.Status == StatusScheduled {
			if err := emitPreorderEvent(ctx, txn, events.EventPreOrderNotified, o, string(auth.RoleRetailer), retailerID); err != nil {
				return err
			}
		}
		if ab, ok := txn.(outbox.AuditBuffer); ok {
			return outbox.WriteAudit(ctx, ab, o.SupplierID, retailerID, "RETAILER", "ORDER_CREATED", "Order", o.OrderID, map[string]any{
				"receiving_window_open":  o.ReceivingWindowOpen,
				"receiving_window_close": o.ReceivingWindowClose,
			})
		}
		return nil
	}, stockOpts)
	if err != nil {
		return CreateResponse{}, fmt.Errorf("persist order: %w", err)
	}

	if stockOpts.Skip && o.Status == StatusPending {
		if err := s.ConfirmAndAllocate(ctx, o.OrderID); err != nil {
			if _, cancelErr := s.cancelOrderWithReason(ctx, &o, "system", "SYSTEM", "allocation_failed", ""); cancelErr != nil {
				s.log.Warn("failed to cancel order after allocation failure", "order_id", o.OrderID, "err", cancelErr)
			}
			if bo != nil {
				if _, cancelErr := s.cancelOrderWithReason(ctx, bo, "system", "SYSTEM", "parent_allocation_failed", ""); cancelErr != nil {
					s.log.Warn("failed to cancel backorder after allocation failure", "order_id", bo.OrderID, "err", cancelErr)
				}
			}
			return CreateResponse{}, fmt.Errorf("allocation failed: %w", err)
		}
		o, _, _ = s.repo.GetOrder(ctx, o.OrderID)
	}

	var backorderOrderID string
	if bo != nil {
		backorderOrderID = bo.OrderID
	}

	// Post-commit cache invalidation: any retailer-orders or supplier-orders
	// list cache MUST be dropped so the next read sees the new row.
	if s.cache != nil {
		s.cache.Invalidate(ctx, append(
			[]string{
				retailerOrdersKey(o.RetailerID),
				supplierOrdersKey(o.SupplierID),
				"catalog:products:" + o.SupplierID,
			},
			dashboardCacheKeys(o.SupplierID, o.WarehouseID)...,
		)...)
	}

	s.log.Info("order created",
		"order_id", o.OrderID,
		"supplier_id", o.SupplierID,
		"retailer_id", o.RetailerID,
		"warehouse_id", o.WarehouseID,
		"total_minor", o.TotalMinor,
		"currency", o.Currency,
		"h3_cell", o.H3Cell,
		"receiving_window_open", o.ReceivingWindowOpen,
		"receiving_window_close", o.ReceivingWindowClose,
	)
	return CreateResponse{
		OrderID:               o.OrderID,
		WarehouseID:           o.WarehouseID,
		Status:                o.Status,
		Source:                o.Source,
		ConfirmationStatus:    o.ConfirmationStatus,
		RequestedDeliveryDate: formatOptionalRFC3339(o.RequestedDeliveryDate),
		DeliverBefore:         formatOptionalRFC3339(o.DeliverBefore),
		DeliveryPriority:      string(o.DeliveryPriority),
		DeliveryMode:          normalizeDeliveryMode(req.DeliveryMode),
		TotalMinor:            o.TotalMinor,
		Currency:              o.Currency,
		CreatedAt:             o.CreatedAt.Format(time.RFC3339Nano),
		ReceivingWindowOpen:   o.ReceivingWindowOpen,
		ReceivingWindowClose:  o.ReceivingWindowClose,
		BackorderOrderID:      backorderOrderID,
		BackorderedItemCount:  invPlan.BackorderCount,
		StockWarnings:         invPlan.Warnings,
	}, nil
}

func (s *Service) createBackorderOrder(
	ctx context.Context,
	retailerID, warehouseID, parentOrderID string,
	lines []LineItem,
	parent Order,
) (string, error) {
	if len(lines) == 0 {
		return "", nil
	}
	var total int64
	for _, li := range lines {
		total += li.Quantity * li.UnitPrice
	}
	boCurrency := strings.TrimSpace(parent.Currency)
	if boCurrency == "" {
		boCurrency = s.currency
	}
	bo := Order{
		OrderID:            s.newID(),
		SupplierID:         parent.SupplierID,
		RetailerID:         retailerID,
		WarehouseID:        warehouseID,
		Status:             StatusBackordered,
		Source:             OrderSourceBackorder,
		ConfirmationStatus: ConfirmationStatusConfirmed,
		LineItems:          lines,
		TotalMinor:         total,
		Currency:           boCurrency,
		H3Cell:             parent.H3Cell,
		Lat:                parent.Lat,
		Lng:                parent.Lng,
		DerivedFromOrderID: parentOrderID,
		Version:            1,
		CreatedAt:          s.now(),
		UpdatedAt:          s.now(),
	}
	err := s.repo.CreateOrder(ctx, &bo, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, bo.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:          events.BaseEvent{Type: events.EventOrderCreated, Timestamp: bo.CreatedAt.Format(time.RFC3339Nano)},
			OrderID:            bo.OrderID,
			SupplierID:         bo.SupplierID,
			RetailerID:         bo.RetailerID,
			WarehouseID:        bo.WarehouseID,
			Status:             string(bo.Status),
			OrderSource:        string(bo.Source),
			ConfirmationStatus: string(bo.ConfirmationStatus),
			TotalMinor:         bo.TotalMinor,
			Currency:           bo.Currency,
			LineItems:          bo.LineItems,
		})
	}, StockReservationOpts{})
	if err != nil {
		return "", err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, append(
			[]string{retailerOrdersKey(bo.RetailerID), supplierOrdersKey(bo.SupplierID)},
			dashboardCacheKeys(bo.SupplierID, bo.WarehouseID)...,
		)...)
	}
	return bo.OrderID, nil
}

// UpdateStatus transitions one order across the canonical lifecycle. Mutations
// and outbox emission happen atomically in repository UpdateOrder.
func (s *Service) UpdateStatus(ctx context.Context, claims auth.Claims, orderID string, req UpdateStatusRequest) (UpdateStatusResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return UpdateStatusResponse{}, errors.New("order_id required")
	}

	nextStatus, err := req.Validate()
	if err != nil {
		return UpdateStatusResponse{}, err
	}

	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return UpdateStatusResponse{}, fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !ok {
		return UpdateStatusResponse{}, ErrOrderNotFound
	}

	if claims.Role != auth.RoleAdmin && claims.Role != auth.RoleRetailer {
		return UpdateStatusResponse{}, ErrOrderForbidden
	}
	if claims.Role == auth.RoleRetailer {
		// B3 M-P0-4: order.RetailerID is org id; staff Subject must not be used for ownership.
		orgID := auth.ResolveRetailerOrgID(claims)
		if orgID == "" || orgID != current.RetailerID || nextStatus != StatusCancelled {
			return UpdateStatusResponse{}, ErrOrderForbidden
		}
		if PreorderCancelLocked(s.now(), current) {
			return UpdateStatusResponse{}, ErrOrderCancelLocked
		}
	}

	if err := ValidateStatusTransition(string(current.Status), string(nextStatus), TransitionOpts{}); err != nil {
		return UpdateStatusResponse{}, err
	}

	// ADR-009 hard-gate: no COMPLETED without reconstructible fiscal SUCCESS or audited FORCE.
	// Force-complete and fiscal worker set FiscalStatus in the same mutation as COMPLETED;
	// generic status patches (incl. reconciliation resolve) must not soft-complete.
	if nextStatus == StatusCompleted {
		fs := strings.TrimSpace(current.FiscalStatus)
		if fs != FiscalStatusSuccess && fs != FiscalStatusForceSkipped {
			return UpdateStatusResponse{}, fmt.Errorf(
				"%w: COMPLETED requires fiscal SUCCESS or FORCE_SKIPPED (use force-complete with reason_code); fiscal_status=%s",
				ErrInvalidStatusTransition, fs,
			)
		}
	}

	if current.Status == nextStatus {
		if strings.TrimSpace(current.QRToken) == "" {
			prevStatus := current.Status
			previousDriverID := strings.TrimSpace(current.DriverID)
			s.applyHandoffLifecycle(&current, prevStatus, previousDriverID)
			if strings.TrimSpace(current.QRToken) != "" {
				current.UpdatedAt = s.now()
				_ = s.repo.UpdateOrder(ctx, current, nil, nil)
			}
		}
		return UpdateStatusResponse{
			OrderID:        current.OrderID,
			PreviousStatus: current.Status,
			Status:         current.Status,
			Version:        current.Version,
			UpdatedAt:      current.UpdatedAt.Format(time.RFC3339Nano),
			EventType:      events.EventOrderStatusChanged,
		}, nil
	}

	prevStatus := current.Status
	previousDriverID := strings.TrimSpace(current.DriverID)
	current.Status = nextStatus
	current.UpdatedAt = s.now()
	s.applyHandoffLifecycle(&current, prevStatus, previousDriverID)
	if nextStatus == StatusCompleted {
		_ = s.ApplyClaimWindowSnapshot(ctx, &current, current.UpdatedAt)
	}
	actorID := claims.Subject
	if actorID == "" {
		actorID = claims.SupplierID
	}

	current.TransitionReason = req.Reason
	current.TransitionActorRole = string(claims.Role)
	current.TransitionActorID = actorID

	err = s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: current.UpdatedAt.Format(time.RFC3339Nano)},
			OrderID:        current.OrderID,
			SupplierID:     current.SupplierID,
			RetailerID:     current.RetailerID,
			DriverID:       current.DriverID,
			PreviousStatus: string(prevStatus),
			Status:         string(current.Status),
			Reason:         strings.TrimSpace(req.Reason),
			ActorRole:      string(claims.Role),
			ActorID:        actorID,
		}); err != nil {
			return err
		}
		if ab, ok := txn.(outbox.AuditBuffer); ok {
			return outbox.WriteAudit(ctx, ab, current.SupplierID, actorID, string(claims.Role), "ORDER_STATUS_CHANGED", "Order", current.OrderID, map[string]string{"from": string(prevStatus), "to": string(current.Status)})
		}
		return nil
	})
	if err != nil {
		return UpdateStatusResponse{}, fmt.Errorf("update order status %s: %w", orderID, err)
	}

	if s.cache != nil {
		s.cache.Invalidate(ctx, append(
			[]string{
				retailerOrdersKey(current.RetailerID),
				supplierOrdersKey(current.SupplierID),
			},
			dashboardCacheKeys(current.SupplierID, current.WarehouseID)...,
		)...)
	}

	s.log.Info("order status updated",
		"order_id", current.OrderID,
		"supplier_id", current.SupplierID,
		"retailer_id", current.RetailerID,
		"prev_status", prevStatus,
		"status", current.Status,
		"actor_role", claims.Role,
		"actor_id", actorID,
	)

	return UpdateStatusResponse{
		OrderID:        current.OrderID,
		PreviousStatus: prevStatus,
		Status:         current.Status,
		Version:        current.Version + 1,
		UpdatedAt:      current.UpdatedAt.Format(time.RFC3339Nano),
		EventType:      events.EventOrderStatusChanged,
	}, nil
}

// AssignOrder durably binds an order to the driver/route authority used by
// driver execution and retailer tracking.
func (s *Service) AssignOrder(ctx context.Context, claims auth.Claims, orderID string, req AssignOrderRequest) (AssignOrderResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return AssignOrderResponse{}, errors.New("order_id required")
	}
	if !canAssignOrders(claims.Role) {
		return AssignOrderResponse{}, ErrOrderForbidden
	}

	normalized, err := req.Validate()
	if err != nil {
		return AssignOrderResponse{}, err
	}

	current, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return AssignOrderResponse{}, fmt.Errorf("get order %s: %w", orderID, err)
	}
	if !ok {
		return AssignOrderResponse{}, ErrOrderNotFound
	}
	if strings.TrimSpace(claims.SupplierID) != "" && claims.SupplierID != current.SupplierID {
		return AssignOrderResponse{}, ErrOrderForbidden
	}

	previousDriverID := strings.TrimSpace(current.DriverID)
	previousRouteID := strings.TrimSpace(current.RouteID)
	previousManifestID := strings.TrimSpace(current.ManifestID)
	if current.DriverID == normalized.DriverID && current.VehicleID == normalized.VehicleID && current.RouteID == normalized.RouteID && current.ManifestID == normalized.ManifestID {
		return assignmentResponse(current, assignmentEventType(previousDriverID), current.Version, true), nil
	}

	previousStatus := current.Status
	current.DriverID = normalized.DriverID
	current.VehicleID = normalized.VehicleID
	current.RouteID = normalized.RouteID
	current.ManifestID = normalized.ManifestID
	current.UpdatedAt = s.now()
	s.applyHandoffLifecycle(&current, previousStatus, previousDriverID)
	eventType := assignmentEventType(previousDriverID)

	err = s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {
		if eventType == events.EventOrderReassigned {
			return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, orderReassignedData(current, previousDriverID, previousRouteID))
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, current.OrderID, events.TopicMain, orderAssignedData(current))
	})
	if err != nil {
		return AssignOrderResponse{}, fmt.Errorf("assign order %s: %w", orderID, err)
	}

	s.afterOrderMutation(ctx, current)

	if previousManifestID != "" && previousManifestID != current.ManifestID {
		s.replanAsync(previousManifestID, "order_removed", claims.Subject)
	}
	if current.ManifestID != "" {
		s.replanAsync(current.ManifestID, "order_assigned", claims.Subject)
	}
	s.log.Info("order assignment updated",
		"order_id", current.OrderID,
		"supplier_id", current.SupplierID,
		"retailer_id", current.RetailerID,
		"driver_id", current.DriverID,
		"route_id", current.RouteID,
		"event_type", eventType,
	)

	return assignmentResponse(current, eventType, current.Version+1, false), nil
}

// MarkArrived moves a driver's active order from IN_TRANSIT to ARRIVED.
func (s *Service) MarkArrived(ctx context.Context, claims auth.Claims, orderID string) (UpdateStatusResponse, error) {
	result, err := s.transitionDriverOrder(ctx, claims, driverTransitionRequest{
		OrderID:    orderID,
		NextStatus: StatusArrived,
		Reason:     "driver_arrived",
	})
	if err != nil {
		return UpdateStatusResponse{}, err
	}
	return UpdateStatusResponse{
		OrderID:        result.Order.OrderID,
		PreviousStatus: result.PreviousStatus,
		Status:         result.Order.Status,
		Version:        result.Version,
		UpdatedAt:      result.UpdatedAt.Format(time.RFC3339Nano),
		EventType:      events.EventOrderStatusChanged,
	}, nil
}

// SubmitDelivery records QR/offline-token handoff and opens payment settlement (ADR-009).
// Does not complete the order — money + fiscal hard-gate still required.
func (s *Service) SubmitDelivery(ctx context.Context, claims auth.Claims, req DeliverySubmitRequest) (DeliverySubmitResponse, error) {
	if strings.TrimSpace(req.OrderID) == "" {
		return DeliverySubmitResponse{}, errors.New("order_id required")
	}
	if strings.TrimSpace(req.token()) == "" {
		return DeliverySubmitResponse{}, errors.New("qr_token required")
	}

	var distanceM float64
	result, err := s.transitionDriverOrder(ctx, claims, driverTransitionRequest{
		OrderID:         req.OrderID,
		NextStatus:      StatusAwaitingPayment,
		Reason:          "driver_delivery_submit",
		ClientTimestamp: req.ClientTimestamp,
		TransformNextStatus: func(orderRecord Order, next Status) Status {
			if orderRecord.Status == StatusCancelled {
				return StatusReconciliationRequired
			}
			// Already past handoff — leave payment/fiscal states alone.
			switch orderRecord.Status {
			case StatusAwaitingPayment, StatusPendingCashCollection, StatusFiscalizing, StatusFiscalFailed, StatusCompleted, StatusDeliveredOnCredit:
				return orderRecord.Status
			default:
				return next
			}
		},
		Precheck: func(orderRecord Order) error {
			if token := req.token(); token != "" {
				if err := s.validateDeliveryToken(orderRecord, token); err != nil {
					if s.jwtSecret == "" || orderRecord.ManifestID == "" ||
						s.validateOfflineQR(orderRecord.ManifestID, claims.Subject, orderRecord.OrderID, token) != nil {
						return err
					}
				}
			}

			if !req.BypassGeofence {
				if req.Latitude == 0 && req.Longitude == 0 {
					return errors.New("gps coordinates required for delivery submission")
				}
				computedDistance, err := validateOptionalGeofence(ctx, req.Latitude, req.Longitude, orderRecord)
				if err == nil {
					distanceM = computedDistance
				}
				return err
			}
			return nil
		},
		BuildProofs: func(orderRecord Order) []DeliveryProofArtifact {
			latitude, longitude := deliveryProofCoordinatesFromValues(req.Latitude, req.Longitude)
			return buildDeliveryProofArtifacts(
				s.newID(),
				orderRecord,
				claims.Subject,
				DeliveryProofTypeQRHandoff,
				strings.TrimSpace(req.QRToken),
				strings.TrimSpace(req.ScannedToken),
				latitude,
				longitude,
				deliveryProofDistance(distanceM, latitude, longitude),
			)
		},
		EmitExtra: func(txn outbox.TxnBuffer, orderRecord Order, previousStatus Status) error {
			if orderRecord.Status != StatusAwaitingPayment || previousStatus == StatusAwaitingPayment {
				return nil
			}
			if err := s.emitSettlementRequired(ctx, txn, orderRecord); err != nil {
				return err
			}
			return s.emitPaymentRequired(ctx, txn, orderRecord)
		},
	})
	if err != nil {
		return DeliverySubmitResponse{}, err
	}

	return DeliverySubmitResponse{
		Success:  true,
		Message:  "Handoff recorded. Awaiting payment and fiscal receipt.",
		NewState: result.Order.Status,
	}, nil
}

// ConfirmOffload records the driver handoff and opens retailer payment settlement.
func (s *Service) ConfirmOffload(ctx context.Context, claims auth.Claims, req ConfirmOffloadRequest) (ConfirmOffloadResponse, error) {
	result, err := s.transitionDriverOrder(ctx, claims, driverTransitionRequest{
		OrderID:         req.OrderID,
		NextStatus:      StatusAwaitingPayment,
		Reason:          "confirm_offload",
		ClientTimestamp: req.ClientTimestamp,
		EmitExtra: func(txn outbox.TxnBuffer, orderRecord Order, _ Status) error {
			if err := s.emitSettlementRequired(ctx, txn, orderRecord); err != nil {
				return err
			}
			return s.emitPaymentRequired(ctx, txn, orderRecord)
		},
	})
	if err != nil {
		return ConfirmOffloadResponse{}, err
	}

	paymentMethod := resolveOrderPaymentMethod(ctx, s.spannerClient, result.Order.OrderID)
	return ConfirmOffloadResponse{
		OrderID:       result.Order.OrderID,
		State:         result.Order.Status,
		PaymentMethod: paymentMethod,
		Amount:        result.Order.TotalMinor,
		Currency:      result.Order.Currency,
		RetailerID:    result.Order.RetailerID,
		Message:       fmt.Sprintf("Collect %d %s", result.Order.TotalMinor, result.Order.Currency),
	}, nil
}

// CompleteOrder captures non-cash settlement intent and enters fiscal hard-gate (ADR-009).
// Order becomes FISCALIZING; COMPLETED only after fiscal SUCCESS (or audited force-complete).
func (s *Service) CompleteOrder(ctx context.Context, claims auth.Claims, req CompleteOrderRequest) (DriverOrderResponse, error) {
	var distanceM float64
	var attemptID string
	result, err := s.transitionDriverOrder(ctx, claims, driverTransitionRequest{
		OrderID:         req.OrderID,
		NextStatus:      StatusFiscalizing,
		Reason:          "complete_order",
		ClientTimestamp: req.ClientTimestamp,
		Precheck: func(orderRecord Order) error {
			switch orderRecord.Status {
			case StatusAwaitingPayment, StatusDeliveredOnCredit, StatusFiscalizing:
			default:
				return fmt.Errorf("%w: complete requires AWAITING_PAYMENT or credit settlement (current %s)", ErrInvalidStatusTransition, orderRecord.Status)
			}
			if orderRecord.Status == StatusFiscalizing {
				return nil
			}
			if err := s.requireFiscalPack(ctx, orderRecord.SupplierID); err != nil {
				return err
			}
			computedDistance, err := validatePointerGeofence(ctx, req.Latitude, req.Longitude, orderRecord)
			if err == nil {
				distanceM = computedDistance
			}
			return err
		},
		PrepareOrder: func(o *Order, _ Status) {
			row := s.newFiscalPendingRow(ctx, *o, "CARD", s.newID(), o.TotalMinor)
			attemptID = row.AttemptID
			o.PendingFiscalReceipts = []FiscalReceiptRow{row}
			o.FiscalStatus = FiscalStatusPending
			o.LatestFiscalAttemptID = row.AttemptID
		},
		BuildProofs: func(orderRecord Order) []DeliveryProofArtifact {
			latitude, longitude := deliveryProofCoordinatesFromPointers(req.Latitude, req.Longitude)
			return buildDeliveryProofArtifacts(
				s.newID(),
				orderRecord,
				claims.Subject,
				DeliveryProofTypeFinalizationGeo,
				"",
				"",
				latitude,
				longitude,
				deliveryProofDistance(distanceM, latitude, longitude),
			)
		},
		EmitExtra: func(txn outbox.TxnBuffer, orderRecord Order, previousStatus Status) error {
			if previousStatus == StatusDeliveredOnCredit {
				if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, events.CreditDeliveryEvent{
					BaseEvent:  events.BaseEvent{Type: events.EventCreditDeliveryResolved, Timestamp: s.now().UTC().Format(time.RFC3339Nano)},
					OrderID:    orderRecord.OrderID,
					DriverID:   claims.Subject,
					SupplierID: orderRecord.SupplierID,
					RetailerID: orderRecord.RetailerID,
					Status:     string(StatusFiscalizing),
				}); err != nil {
					return err
				}
			}
			// PAYMENT_CLEARED and FISCAL_RECEIPT_REQUESTED are emitted by the
			// post-commit settlement step: for card tenders only after provider
			// capture confirms, otherwise immediately (see settleOutstandingCardPayment).
			return nil
		},
		InTxn: func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
			orderRow, found, err := s.repo.GetOrderTxn(txnCtx, txn, req.OrderID)
			if err != nil {
				return fmt.Errorf("failed to get order: %w", err)
			}
			if !found {
				return fmt.Errorf("order not found: %s", req.OrderID)
			}
			delivered := deliveredGrossFromOrder(orderRow)
			paid, err := s.getCapturedPaymentMinorTxn(txnCtx, txn, req.OrderID)
			if err != nil {
				return err
			}
			exceptions, err := s.getExceptionsTotalMinorTxn(txnCtx, txn, req.OrderID)
			if err != nil {
				return err
			}
			due := delivered - paid - exceptions
			if due < 0 {
				due = 0
			}
			if err := s.AssertMoneyCoversDeliveryTxn(txnCtx, txn, req.OrderID, due, 0); err != nil {
				return err
			}
			if due == 0 {
				// Fully covered by credit/cash legs and exceptions; nothing to capture.
				return nil
			}
			// Record the card leg PENDING. It becomes CAPTURED only after the
			// provider confirms the capture (settleOutstandingCardPayment); the
			// stable idempotency key makes a duplicate card leg impossible once
			// the unique index is in place.
			leg := PaymentLeg{
				OrderID:        req.OrderID,
				LegID:          s.newID(),
				Method:         MethodCard,
				AmountMinor:    due,
				Status:         PaymentStatusPending,
				IdempotencyKey: fmt.Sprintf("card-capture-%s", req.OrderID),
				CreatedAt:      s.now(),
			}
			return s.RecordPaymentLeg(txnCtx, txn, leg)
		},
	})
	if err != nil {
		return DriverOrderResponse{}, err
	}
	// G1-A2: ClearBalance moved into finalizeCardSettlement (same txn as CAPTURED).
	// Do not clear credit books before provider capture confirms.
	// Synchronous provider-confirmed settlement. Runs on both the fresh
	// transition and idempotent retries (NoChange) so a capture interrupted
	// mid-flight is driven to CAPTURED or FAILED, never left optimistic.
	if err := s.settleOutstandingCardPayment(ctx, result.Order); err != nil {
		return DriverOrderResponse{}, err
	}

	msg := "Payment captured. Waiting for fiscal receipt."
	if attemptID != "" {
		_ = attemptID
	}
	return driverOrderResponse(result.Order, msg), nil
}

// CollectCash geofence-confirms cash collection, captures payment, and starts fiscal hard-gate (ADR-009).
// Fiscal amount = amount_received_minor when provided (P0 T3); shortfall/overage emitted as audit events.
// Proximity settlement: requires ProximityUnlockedAt (or existing geofence within settlement radius).
func (s *Service) CollectCash(ctx context.Context, claims auth.Claims, req CollectCashRequest) (CollectCashResponse, error) {
	var distanceM float64
	var attemptID string
	var receivedMinor, shortfallMinor, overageMinor int64
	// Settlement proximity gate (design §4.1) — unlock or live 100 m / H3.
	if err := s.requireProximityUnlocked(ctx, req.OrderID, ""); err != nil {
		// Fallback: accept if request coords satisfy settlement proximity against order.
		// (Unlock endpoint is preferred; this keeps single-shot cash collect working offline-sync.)
		// Full check deferred to Precheck where order coords are available.
	}
	result, err := s.transitionDriverOrder(ctx, claims, driverTransitionRequest{
		OrderID:         req.OrderID,
		NextStatus:      StatusFiscalizing,
		Reason:          "collect_cash",
		ClientTimestamp: req.ClientTimestamp,
		Precheck: func(orderRecord Order) error {
			switch orderRecord.Status {
			case StatusPendingCashCollection, StatusAwaitingPayment, StatusDeliveredOnCredit, StatusFiscalizing:
			default:
				return fmt.Errorf("%w: collect cash requires payment capture state (current %s)", ErrInvalidStatusTransition, orderRecord.Status)
			}
			if orderRecord.Status == StatusFiscalizing {
				return nil
			}
			if err := s.requireFiscalPack(ctx, orderRecord.SupplierID); err != nil {
				return err
			}
			// Proximity: prior unlock OR live settlement radius / H3 (anti-spoof + doorstep).
			if err := s.requireProximityUnlocked(ctx, orderRecord.OrderID, ""); err != nil {
				method, dist, ok := EvaluateSettlementProximity(req.Latitude, req.Longitude, orderRecord.Lat, orderRecord.Lng, orderRecord.H3Cell)
				if !ok {
					return fmt.Errorf("%w: unlock required or within %.0fm (dist=%.0f)", ErrProximityLocked, SettlementProximityRadiusM, dist)
				}
				_ = method
				distanceM = dist
			}
			if req.AmountReceivedMinor != nil {
				if *req.AmountReceivedMinor < 0 {
					return ErrCashAmountNegative
				}
				receivedMinor = *req.AmountReceivedMinor
			} else {
				// Compat: default to expected total when driver omits received amount.
				receivedMinor = orderRecord.TotalMinor
			}
			if orderRecord.TotalMinor > receivedMinor {
				shortfallMinor = orderRecord.TotalMinor - receivedMinor
			} else if receivedMinor > orderRecord.TotalMinor {
				overageMinor = receivedMinor - orderRecord.TotalMinor
			}
			// Keep approach geofence as outer bound (pack breach_radius_meters) when settlement already satisfied.
			if distanceM == 0 {
				computedDistance, err := validateRequiredGeofence(ctx, req.Latitude, req.Longitude, orderRecord)
				if err != nil {
					return err
				}
				distanceM = computedDistance
			}
			return nil
		},
		PrepareOrder: func(o *Order, _ Status) {
			if receivedMinor == 0 && req.AmountReceivedMinor == nil {
				receivedMinor = o.TotalMinor
			}
			row := s.newFiscalPendingRow(ctx, *o, "CASH", s.newID(), receivedMinor)
			attemptID = row.AttemptID
			o.PendingFiscalReceipts = []FiscalReceiptRow{row}
			o.FiscalStatus = FiscalStatusPending
			o.LatestFiscalAttemptID = row.AttemptID
		},
		BuildProofs: func(orderRecord Order) []DeliveryProofArtifact {
			latitude, longitude := deliveryProofCoordinatesFromValues(req.Latitude, req.Longitude)
			return buildDeliveryProofArtifacts(
				s.newID(),
				orderRecord,
				claims.Subject,
				DeliveryProofTypeCashCollectionGeo,
				"",
				"",
				latitude,
				longitude,
				deliveryProofDistance(distanceM, latitude, longitude),
			)
		},
		EmitExtra: func(txn outbox.TxnBuffer, orderRecord Order, previousStatus Status) error {
			if previousStatus == StatusDeliveredOnCredit {
				if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, events.CreditDeliveryEvent{
					BaseEvent:  events.BaseEvent{Type: events.EventCreditDeliveryResolved, Timestamp: s.now().UTC().Format(time.RFC3339Nano)},
					OrderID:    orderRecord.OrderID,
					DriverID:   claims.Subject,
					SupplierID: orderRecord.SupplierID,
					RetailerID: orderRecord.RetailerID,
					Status:     string(StatusFiscalizing),
				}); err != nil {
					return err
				}
			}
			if err := emitCashVariance(ctx, txn, orderRecord, claims.Subject, orderRecord.TotalMinor, receivedMinor, req.Note); err != nil {
				return err
			}
			row := orderRecord.PendingFiscalReceipts[0]
			return emitPaymentCaptureFiscal(ctx, txn, orderRecord, row, "CASH")
		},
		InTxn: func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
			orderRecord, ok, err := s.repo.GetOrderTxn(txnCtx, txn, req.OrderID)
			if err != nil {
				return fmt.Errorf("failed to get order: %w", err)
			}
			if !ok {
				return fmt.Errorf("order not found: %s", req.OrderID)
			}
			delivered := deliveredGrossFromOrder(orderRecord)
			captured, err := s.getCapturedPaymentMinorTxn(txnCtx, txn, req.OrderID)
			if err != nil {
				return err
			}
			exceptions, err := s.getExceptionsTotalMinorTxn(txnCtx, txn, req.OrderID)
			if err != nil {
				return err
			}
			totalPaid := captured + exceptions + receivedMinor

			var shortfall int64
			if totalPaid < delivered {
				shortfall = delivered - totalPaid
				ex := SettlementException{
					OrderID:     req.OrderID,
					ExceptionID: s.newID(),
					Type:        "CASH_SHORTFALL",
					AmountMinor: shortfall,
					Status:      "OPEN",
					CreatedBy:   claims.Subject,
					CreatedAt:   s.now(),
				}
				if writeErr := s.RecordSettlementException(txnCtx, txn, ex); writeErr != nil {
					return writeErr
				}
			}

			if err := s.AssertMoneyCoversDeliveryTxn(txnCtx, txn, req.OrderID, receivedMinor, shortfall); err != nil {
				return err
			}
			if s.doorstepShortfall != nil && orderRecord.TotalMinor > receivedMinor {
				if err := s.doorstepShortfall(txnCtx, txn, req.OrderID, orderRecord.SupplierID, claims.Subject, orderRecord.Currency, orderRecord.TotalMinor, receivedMinor, s.now()); err != nil {
					return err
				}
			}
			leg := PaymentLeg{
				OrderID:     req.OrderID,
				LegID:       s.newID(),
				Method:      MethodCash,
				AmountMinor: receivedMinor,
				Status:      PaymentStatusCaptured,
				// Stable key: retries must hit the unique index, not mint a second cash leg.
				IdempotencyKey: "cash-" + req.OrderID,
				CreatedAt:      s.now(),
				CapturedAt:     spanner.NullTime{Time: s.now(), Valid: true},
			}
			if err := s.RecordPaymentLeg(txnCtx, txn, leg); err != nil {
				return err
			}
			// G1.A: AR pay-down on the same Spanner RW txn as cash capture.
			// Fail-closed — abort cash commit if AR apply fails when an invoice exists.
			if s.ar != nil {
				if err := s.ar.RecordPaymentForOrderInTxn(txnCtx, txn, req.OrderID, receivedMinor,
					"ar-cash-collect-"+req.OrderID, orderRecord.Currency); err != nil {
					return err
				}
			} else if orderRecord.Status == StatusDeliveredOnCredit && ar.InvoicesEnabled() && orderRecord.TotalMinor > 0 {
				return fmt.Errorf("ar service required for credit cash collection")
			}
			// G1-A2: clear credit headroom when cash settles a credit-left order (same txn).
			// No-op when reservation was never CONVERTED for this order.
			if s.credit != nil && receivedMinor > 0 {
				clearAmt := orderRecord.TotalMinor
				if clearAmt <= 0 {
					clearAmt = receivedMinor
				}
				if err := s.credit.ClearBalanceInTxn(txnCtx, txn,
					orderRecord.RetailerID, orderRecord.SupplierID, req.OrderID, clearAmt); err != nil {
					return err
				}
			}
			if err := loyalty.EarnInTxn(txnCtx, txn, nil, orderRecord.SupplierID, orderRecord.RetailerID, req.OrderID, receivedMinor); err != nil {
				return err
			}
			return nil
		},
	})
	if err != nil {
		return CollectCashResponse{}, err
	}
	if result.NoChange {
		distanceM = 0
		return CollectCashResponse{
			OrderID:             result.Order.OrderID,
			State:               result.Order.Status,
			Amount:              result.Order.TotalMinor,
			AmountReceivedMinor: receivedMinor,
			Currency:            result.Order.Currency,
			DistanceM:           distanceM,
			Message:             "Cash collection already fiscalizing.",
			AttemptID:           result.Order.LatestFiscalAttemptID,
			FiscalStatus:        result.Order.FiscalStatus,
		}, nil
	}

	msg := "Cash collected. Waiting for fiscal receipt."
	if shortfallMinor > 0 {
		msg = fmt.Sprintf("Cash collected with shortfall %d. Fiscalizing received amount.", shortfallMinor)
	} else if overageMinor > 0 {
		msg = fmt.Sprintf("Cash collected with overage %d. Fiscalizing received amount.", overageMinor)
	}

	return CollectCashResponse{
		OrderID:             result.Order.OrderID,
		State:               result.Order.Status,
		Amount:              result.Order.TotalMinor,
		AmountReceivedMinor: receivedMinor,
		ShortfallMinor:      shortfallMinor,
		OverageMinor:        overageMinor,
		Currency:            result.Order.Currency,
		DistanceM:           distanceM,
		Message:             msg,
		AttemptID:           attemptID,
		FiscalStatus:        FiscalStatusPending,
	}, nil
}

func (s *Service) transitionDriverOrder(ctx context.Context, claims auth.Claims, req driverTransitionRequest) (driverTransitionResult, error) {
	current, err := s.loadDriverTransitionOrder(ctx, claims, req.OrderID)
	if err != nil {
		return driverTransitionResult{}, err
	}

	if req.TransformNextStatus != nil {
		req.NextStatus = req.TransformNextStatus(current, req.NextStatus)
	}

	if current.Status == req.NextStatus {
		return driverTransitionResult{
			Order:          current,
			PreviousStatus: current.Status,
			Version:        current.Version,
			UpdatedAt:      current.UpdatedAt,
			NoChange:       true,
		}, nil
	}
	if err := ValidateStatusTransition(string(current.Status), string(req.NextStatus), TransitionOpts{}); err != nil {
		return driverTransitionResult{}, err
	}
	if req.Precheck != nil {
		if err := req.Precheck(current); err != nil {
			return driverTransitionResult{}, err
		}
	}

	previousStatus := current.Status
	previousDriverID := strings.TrimSpace(current.DriverID)
	current.Status = req.NextStatus
	if req.ClientTimestamp != nil {
		current.UpdatedAt = *req.ClientTimestamp
	} else {
		current.UpdatedAt = s.now()
	}
	s.applyHandoffLifecycle(&current, previousStatus, previousDriverID)
	if req.PrepareOrder != nil {
		req.PrepareOrder(&current, previousStatus)
	}
	if err := s.persistDriverTransition(ctx, claims, req, current, previousStatus); err != nil {
		return driverTransitionResult{}, err
	}

	s.recordDriverTransitionSuccess(ctx, claims, req, current, previousStatus)

	return driverTransitionResult{
		Order:          current,
		PreviousStatus: previousStatus,
		Version:        current.Version + 1,
		UpdatedAt:      current.UpdatedAt,
	}, nil
}

func (s *Service) loadDriverTransitionOrder(ctx context.Context, claims auth.Claims, orderID string) (Order, error) {
	trimmedOrderID := strings.TrimSpace(orderID)
	if trimmedOrderID == "" {
		return Order{}, errors.New("order_id required")
	}
	if claims.Role != auth.RoleDriver {
		return Order{}, ErrOrderForbidden
	}

	current, ok, err := s.repo.GetOrder(ctx, trimmedOrderID)
	if err != nil {
		return Order{}, fmt.Errorf("get order %s: %w", trimmedOrderID, err)
	}
	if !ok {
		return Order{}, ErrOrderNotFound
	}
	if strings.TrimSpace(current.DriverID) == "" {
		return Order{}, ErrAssignmentRequired
	}
	if strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(current.DriverID) != strings.TrimSpace(claims.Subject) {
		return Order{}, ErrOrderForbidden
	}

	return current, nil
}

func canAssignOrders(role auth.Role) bool {
	switch role {
	case auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleFactoryAdmin:
		return true
	default:
		return false
	}
}

func (s *Service) persistDriverTransition(ctx context.Context, claims auth.Claims, req driverTransitionRequest, current Order, previousStatus Status) error {
	actorID := claims.Subject
	if strings.TrimSpace(current.TransitionReason) == "" {
		current.TransitionReason = req.Reason
	}
	if strings.TrimSpace(current.TransitionActorRole) == "" {
		current.TransitionActorRole = string(claims.Role)
	}
	if strings.TrimSpace(current.TransitionActorID) == "" {
		current.TransitionActorID = actorID
	}
	var proofs []DeliveryProofArtifact
	if req.BuildProofs != nil {
		proofs = req.BuildProofs(current)
	}
	var err error
	if req.InTxn != nil {
		err = s.repo.UpdateOrderWithTxn(ctx, current, proofs, req.InTxn, func(txn outbox.TxnBuffer) error {
			params := orderStatusEmitParams{
				Claims:         claims,
				Order:          current,
				PreviousStatus: previousStatus,
				Reason:         req.Reason,
				ActorID:        actorID,
			}
			if err := emitOrderStatusChanged(ctx, txn, params); err != nil {
				return err
			}
			if req.EmitExtra != nil {
				if err := req.EmitExtra(txn, current, previousStatus); err != nil {
					return err
				}
			}
			if ab, ok := txn.(outbox.AuditBuffer); ok {
				return outbox.WriteAudit(ctx, ab, current.SupplierID, actorID, string(claims.Role), "ORDER_STATUS_CHANGED", "Order", current.OrderID, map[string]string{"from": string(previousStatus), "to": string(current.Status)})
			}
			return nil
		})
	} else {
		err = s.repo.UpdateOrder(ctx, current, proofs, func(txn outbox.TxnBuffer) error {
			params := orderStatusEmitParams{
				Claims:         claims,
				Order:          current,
				PreviousStatus: previousStatus,
				Reason:         req.Reason,
				ActorID:        actorID,
			}
			if err := emitOrderStatusChanged(ctx, txn, params); err != nil {
				return err
			}
			if req.EmitExtra != nil {
				if err := req.EmitExtra(txn, current, previousStatus); err != nil {
					return err
				}
			}
			if ab, ok := txn.(outbox.AuditBuffer); ok {
				return outbox.WriteAudit(ctx, ab, current.SupplierID, actorID, string(claims.Role), "ORDER_STATUS_CHANGED", "Order", current.OrderID, map[string]string{"from": string(previousStatus), "to": string(current.Status)})
			}
			return nil
		})
	}
	if err != nil {
		return fmt.Errorf("transition driver order %s: %w", current.OrderID, err)
	}

	return nil
}

func emitOrderStatusChanged(ctx context.Context, txn outbox.TxnBuffer, params orderStatusEmitParams) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, params.Order.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:             events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: params.Order.UpdatedAt.Format(time.RFC3339Nano)},
		OrderID:               params.Order.OrderID,
		SupplierID:            params.Order.SupplierID,
		RetailerID:            params.Order.RetailerID,
		WarehouseID:           params.Order.WarehouseID,
		DriverID:              params.Order.DriverID,
		PreviousStatus:        string(params.PreviousStatus),
		Status:                string(params.Order.Status),
		Reason:                params.Reason,
		ActorRole:             string(params.Claims.Role),
		ActorID:               params.ActorID,
		OrderSource:           string(params.Order.Source),
		ConfirmationStatus:    string(params.Order.ConfirmationStatus),
		RequestedDeliveryDate: formatOptionalRFC3339(params.Order.RequestedDeliveryDate),
		Version:               params.Order.Version,
	})
}

func (s *Service) recordDriverTransitionSuccess(ctx context.Context, claims auth.Claims, req driverTransitionRequest, current Order, previousStatus Status) {
	s.afterOrderMutation(ctx, current)
	s.log.Info("driver order status updated",
		"order_id", current.OrderID,
		"supplier_id", current.SupplierID,
		"retailer_id", current.RetailerID,
		"prev_status", previousStatus,
		"status", current.Status,
		"actor_id", claims.Subject,
	)

	if (current.Status == StatusInTransit || current.Status == StatusCompleted) && s.driverHub != nil {
		siblings, err := s.repo.FindSiblingDriversForOrder(ctx, current.OrderID)
		if err == nil && len(siblings) > 1 {
			for _, sib := range siblings {
				if sib != claims.Subject {
					eventType := "OTHER_TRUCK_ON_WAY"
					message := "Another truck is on the way to this route."
					if current.Status == StatusCompleted {
						eventType = "PAYMENT_COMPLETED"
						message = "Payment has been collected by another driver."
					}

					payload := map[string]any{
						"type":     eventType,
						"order_id": current.OrderID,
						"message":  message,
					}
					b, _ := json.Marshal(payload)

					go s.driverHub.Broadcast(context.Background(), "driver:"+sib, b)
				}
			}
		}
	}
}

// HandleCreate is POST /v1/order/create. Wired by orderroutes.RegisterRoutes
// behind auth.RequireRole(RETAILER).
