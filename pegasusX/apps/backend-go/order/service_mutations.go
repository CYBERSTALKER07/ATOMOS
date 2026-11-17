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
	"github.com/pegasusx/pegasusx/apps/backend-go/drivercmd"
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

	if err := s.validateOrderPackMultiples(ctx, lineItems); err != nil {
		return CreateResponse{}, err
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
			if req.AutoConfirmOptIn {
				status = StatusPending
				confirmation = ConfirmationStatusAutoConfirmed
			} else {
				status = StatusScheduled
				confirmation = ConfirmationStatusPending
			}
		}
	}

	type atomicOrderCreator interface {
		CreateAtomicInTxn(ctx context.Context, fn func(context.Context, *spanner.ReadWriteTransaction) error) error
	}
	type backorderTxnWriter interface {
		writeOrderWithBackorderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, o *Order, backorder *Order, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error
	}

	policyOverride, _ := s.resolveCheckoutPolicyOverride(whPolicy, req.CheckoutPolicyToken)
	baseLines := append([]LineItem(nil), lineItems...)
	baseTotal := total
	orderEstimate := baseTotal + deliveryFeeMinor
	paymentMethod := NormalizeCheckoutPaymentMethod(req.PaymentMethod, "")
	explicitClientOrderID := strings.TrimSpace(req.ClientOrderID)
	if explicitClientOrderID == "" {
		explicitClientOrderID = clientOrderIDFromContext(ctx)
	}
	explicitClientOrderID = truncateClientOrderID(explicitClientOrderID)
	usageUserID, usageRole, _ := resolveOrderActorAndRole(ctx, retailerID)
	usageDate := sellerUsageDate(now)
	approvedReplay, isApproved := approvedApprovalFromContext(ctx)
	if isApproved && !approvedReplay.UsageDate.IsZero() {
		usageDate = approvedReplay.UsageDate
	}

	if explicitClientOrderID != "" {
		if existing, found, exErr := s.findExistingOrderByIdempotencyKey(ctx, retailerID, supplierID, explicitClientOrderID); exErr == nil && found {
			if existing.Status == StatusCancelled && (existing.ParentOrderID != "" && s.isParentSagaCompensated(ctx, existing.ParentOrderID)) {
				return CreateResponse{}, fmt.Errorf("order for idempotency key %q was cancelled as part of a compensated saga and cannot be replayed; retry with fresh order id", explicitClientOrderID)
			}
			return CreateResponse{
				OrderID:               existing.OrderID,
				WarehouseID:           existing.WarehouseID,
				Status:                existing.Status,
				Source:                existing.Source,
				ConfirmationStatus:    existing.ConfirmationStatus,
				AutoConfirmOptIn:      existing.AutoConfirmOptIn,
				HoldReason:            existing.HoldReason,
				RequestedDeliveryDate: formatOptionalRFC3339(existing.RequestedDeliveryDate),
				DeliverBefore:         formatOptionalRFC3339(existing.DeliverBefore),
				DeliveryPriority:      string(existing.DeliveryPriority),
				DeliveryMode:          normalizeDeliveryMode(req.DeliveryMode),
				TotalMinor:            existing.TotalMinor,
				Currency:              existing.Currency,
				CreatedAt:             existing.CreatedAt.UTC().Format(time.RFC3339Nano),
				ReceivingWindowOpen:   existing.ReceivingWindowOpen,
				ReceivingWindowClose:  existing.ReceivingWindowClose,
				LineItems:             append([]LineItem(nil), existing.LineItems...),
			}, nil
		}
	}

	if !isMandatePrechecked(ctx) {
		gateClientOrderID := explicitClientOrderID
		if gateClientOrderID == "" {
			gateClientOrderID = newMandateUUID()
		}
		reqCopy := req
		reqCopy.SupplierID = supplierID
		reqCopy.PaymentMethod = paymentMethod
		reqCopy.ClientOrderID = gateClientOrderID
		if err := s.checkSellerMandateGate(ctx, retailerID, usageUserID, usageRole, supplierID, gateClientOrderID, paymentMethod, orderEstimate, ApprovalPayloadEnvelope{
			Kind:          ApprovalPayloadCreate,
			SupplierID:    supplierID,
			Currency:      orderCurrency,
			CreateRequest: &reqCopy,
		}); err != nil {
			return CreateResponse{}, err
		}
	}

	mandateLookup, mandateErr := s.resolveEffectiveMandate(ctx, retailerID, usageUserID, usageRole)
	if mandateErr != nil {
		return CreateResponse{}, mandateErr
	}
	enforceCapInTxn := !isApproved && !mandateLookup.IsOwner && !strings.EqualFold(mandateLookup.Role, "OWNER")

	orderID := s.newID()
	boID := s.newID()
	if boID == orderID {
		boID = s.newID()
	}

	var invPlan InventoryPlan
	var o Order
	var bo *Order
	var allBackordered bool

	var heldCacheKeys []string
	var heldCacheQtys []int64
	releaseInflight := func() {
		if s.cache == nil {
			return
		}
		for i, key := range heldCacheKeys {
			if _, decrErr := s.cache.DecrBy(context.Background(), key, heldCacheQtys[i]); decrErr != nil && s.log != nil {
				s.log.Warn("release inflight cache counter failed", "key", key, "qty", heldCacheQtys[i], "error", decrErr)
			}
		}
		heldCacheKeys = nil
		heldCacheQtys = nil
	}
	defer releaseInflight()

	acquireInflight := func(checkCtx context.Context, fulfillable []LineItem, availableStock map[string]int64) error {
		if s.cache == nil {
			return nil
		}
		for _, li := range fulfillable {
			sku := strings.TrimSpace(li.SKU)
			if sku == "" {
				continue
			}
			key := fmt.Sprintf("inflight_rsv:%s:%s", warehouseID, sku)
			inflight, cerr := s.cache.IncrBy(checkCtx, key, li.Quantity)
			if cerr != nil {
				continue
			}
			heldCacheKeys = append(heldCacheKeys, key)
			heldCacheQtys = append(heldCacheQtys, li.Quantity)

			avail := availableStock[sku]
			if inflight > avail {
				return &InventoryCheckoutError{
					Code:         "PARTIAL_OUT_OF_STOCK_REJECTED",
					Message:      "inventory_exhausted_concurrent",
					OOSItems:     []string{sku},
					RejectedSKUs: []string{sku},
					Shortfall:    map[string]int64{sku: inflight - avail},
				}
			}
		}
		return nil
	}

	buildPrimaryAndBackorder := func(plan InventoryPlan, orderTotal int64) (Order, *Order) {
		primary := Order{
			OrderID:               orderID,
			SupplierID:            supplierID,
			RetailerID:            retailerID,
			ParentOrderID:         strings.TrimSpace(req.ParentOrderID),
			IdempotencyKey:        explicitClientOrderID,
			WarehouseID:           warehouseID,
			Status:                status,
			Source:                source,
			ConfirmationStatus:    confirmation,
			AutoConfirmOptIn:      req.AutoConfirmOptIn,
			LineItems:             plan.Fulfillable,
			TotalMinor:            orderTotal,
			OriginalTotalMinor:    baseTotal + deliveryFeeMinor,
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
		var splitBO *Order
		if len(plan.Backorder) > 0 {
			var boTotal int64
			for _, li := range plan.Backorder {
				boTotal += li.Quantity * li.UnitPrice
			}
			boCurrency := strings.TrimSpace(primary.Currency)
			if boCurrency == "" {
				boCurrency = s.currency
			}
			splitBO = &Order{
				OrderID:            boID,
				SupplierID:         primary.SupplierID,
				RetailerID:         primary.RetailerID,
				WarehouseID:        primary.WarehouseID,
				Status:             StatusBackordered,
				Source:             OrderSourceBackorder,
				ConfirmationStatus: ConfirmationStatusConfirmed,
				LineItems:          plan.Backorder,
				TotalMinor:         boTotal,
				Currency:           boCurrency,
				H3Cell:             primary.H3Cell,
				Lat:                primary.Lat,
				Lng:                primary.Lng,
				DerivedFromOrderID: primary.OrderID,
				Version:            1,
				CreatedAt:          now,
				UpdatedAt:          now,
			}
		}
		return primary, splitBO
	}

	buildMainEmit := func(emitCtx context.Context, primary *Order, splitBO *Order) func(outbox.TxnBuffer) error {
		return func(txn outbox.TxnBuffer) error {
			if err := outbox.EmitJSON(emitCtx, txn, events.AggregateOrder, primary.OrderID, events.TopicMain, events.OrderEvent{
				BaseEvent:             events.BaseEvent{Type: events.EventOrderCreated, Timestamp: primary.CreatedAt.Format(time.RFC3339Nano)},
				OrderID:               primary.OrderID,
				SupplierID:            primary.SupplierID,
				RetailerID:            primary.RetailerID,
				WarehouseID:           primary.WarehouseID,
				Status:                string(primary.Status),
				OrderSource:           string(primary.Source),
				ConfirmationStatus:    string(primary.ConfirmationStatus),
				TotalMinor:            primary.TotalMinor,
				Currency:              primary.Currency,
				H3Cell:                primary.H3Cell,
				Lat:                   primary.Lat,
				Lng:                   primary.Lng,
				RequestedDeliveryDate: formatOptionalRFC3339(primary.RequestedDeliveryDate),
				ReceivingWindowOpen:   primary.ReceivingWindowOpen,
				ReceivingWindowClose:  primary.ReceivingWindowClose,
				LineItems:             primary.LineItems,
			}); err != nil {
				return err
			}
			if splitBO != nil {
				if err := outbox.EmitJSON(emitCtx, txn, events.AggregateOrder, splitBO.OrderID, events.TopicMain, events.OrderEvent{
					BaseEvent:          events.BaseEvent{Type: events.EventOrderCreated, Timestamp: splitBO.CreatedAt.Format(time.RFC3339Nano)},
					OrderID:            splitBO.OrderID,
					SupplierID:         splitBO.SupplierID,
					RetailerID:         splitBO.RetailerID,
					WarehouseID:        splitBO.WarehouseID,
					Status:             string(splitBO.Status),
					OrderSource:        string(splitBO.Source),
					ConfirmationStatus: string(splitBO.ConfirmationStatus),
					TotalMinor:         splitBO.TotalMinor,
					Currency:           splitBO.Currency,
					LineItems:          splitBO.LineItems,
				}); err != nil {
					return err
				}
			}
			if primary.Status == StatusScheduled {
				if err := emitPreorderEvent(emitCtx, txn, events.EventPreOrderNotified, *primary, string(auth.RoleRetailer), retailerID); err != nil {
					return err
				}
			}
			if ab, ok := txn.(outbox.AuditBuffer); ok {
				creditDelta, cashDelta := usageDeltasForPayment(paymentMethod, primary.TotalMinor)
				return outbox.WriteAudit(emitCtx, ab, primary.SupplierID, retailerID, "RETAILER", "ORDER_CREATED", "Order", primary.OrderID, map[string]any{
					"receiving_window_open":  primary.ReceivingWindowOpen,
					"receiving_window_close": primary.ReceivingWindowClose,
					"usage_user_id":          usageUserID,
					"usage_date":             usageDate.String(),
					"payment_method":         paymentMethod,
					"spent_credit_minor":     creditDelta,
					"spent_cash_minor":       cashDelta,
					"order_count":            int64(1),
				})
			}
			return nil
		}
	}

	atomicCreator, hasAtomic := s.repo.(atomicOrderCreator)
	txnWriter, hasTxnWriter := s.repo.(backorderTxnWriter)
	var existingDuplicate bool
	if s.spannerClient != nil && hasAtomic && hasTxnWriter {
		err = atomicCreator.CreateAtomicInTxn(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Ensure Spanner transaction aborts/retries start with clean inflight Redis locks.
			releaseInflight()

			// Mandatory lock order: RetailerSellerDailyUsage -> CreditProfile -> Orders.
			lockRes, lockErr := s.acquireCreateLocksInTxn(txCtx, txn, retailerID, usageUserID, supplierID, usageDate, paymentMethod, orderEstimate, mandateLookup, enforceCapInTxn)
			if lockErr != nil {
				return lockErr
			}

			if explicitClientOrderID != "" {
				if dupOID, dupFound, dupErr := findExistingOrderByIdempotencyInTxn(txCtx, txn, retailerID, supplierID, explicitClientOrderID); dupErr != nil {
					return dupErr
				} else if dupFound && dupOID != "" {
					if loaded, lOK, lErr := s.repo.GetOrderTxn(txCtx, txn, dupOID); lErr == nil && lOK {
						o = loaded
						bo = nil
						existingDuplicate = true
						return nil
					}
				}
			}

			bufferUsageBeforeOrder := func(actualAmountMinor int64) error {
				if enforceCapInTxn && actualAmountMinor > orderEstimate {
					curUsage := SellerDailyUsage{
						RetailerID:       retailerID,
						UserID:           usageUserID,
						UsageDate:        usageDate,
						SpentCreditMinor: lockRes.BaseCreditMinor,
						SpentCashMinor:   lockRes.BaseCashMinor,
						OrderCount:       lockRes.BaseOrderCount,
					}
					if eval := evaluateSellerMandate(mandateLookup.Mandate, mandateLookup.IsOwner, mandateLookup.Role, paymentMethod, actualAmountMinor, curUsage); !eval.Allowed {
						return &OwnerApprovalRequiredError{
							TrippedRule:            eval.TrippedRule,
							AmountMinor:            actualAmountMinor,
							ThresholdMinor:         eval.ThresholdMinor,
							RemainingCapMinor:      eval.RemainingCapMinor,
							FallbackPaymentMethods: eval.Fallbacks,
						}
					}
				}
				uMut, _, _ := buildSellerDailyUsageMutation(retailerID, usageUserID, usageDate, paymentMethod, actualAmountMinor, lockRes)
				if err := txn.BufferWrite([]*spanner.Mutation{uMut}); err != nil {
					return err
				}
				recordLockStage(txCtx, s.lockOrderTracer, LockStageOrders)
				return nil
			}

			evalLines := baseLines
			var shortPlan *OrderShortagePlan
			if source != OrderSourceBackorder && (s.shortageStockReader != nil || hasLineShortageConstraints(baseLines)) {
				workingLines, sc, scErr := s.loadOrderShortageContext(txCtx, txn, supplierID, warehouseID, baseLines)
				if scErr != nil {
					return scErr
				}
				evaluated := EvaluateOrderShortages(workingLines, sc.AvailableBySKU, sc.CandidatesBySKU, OrderShortageOptions{
					SupplierID:       supplierID,
					EnforceAllowlist: sc.EnforceAllowlist,
				})
				shortPlan = &evaluated
				if evaluated.HoldOrder {
					heldOrder, _ := buildPrimaryAndBackorder(InventoryPlan{Fulfillable: evaluated.Lines}, evaluated.UpdatedTotalMinor+deliveryFeeMinor)
					heldOrder.Status = StatusDelayed
					heldOrder.ConfirmationStatus = ConfirmationStatusPendingWarehouse
					heldOrder.HoldReason = evaluated.HoldReason
					heldOrder.WarehouseNotes = evaluated.HoldReason
					heldOrder.Substitutions = evaluated.Substitutions
					heldOrder.CutLines = evaluated.CutLines
					if bErr := bufferUsageBeforeOrder(heldOrder.TotalMinor); bErr != nil {
						return bErr
					}
					if werr := txnWriter.writeOrderWithBackorderInTxn(txCtx, txn, &heldOrder, nil, buildMainEmit(txCtx, &heldOrder, nil), StockReservationOpts{Skip: true}); werr != nil {
						return fmt.Errorf("persist held order: %w", werr)
					}
					if paymentMethod == PaymentMethodCredit {
						if cerr := s.reserveCreditForOrderInTxn(txCtx, txn, retailerID, supplierID, heldOrder.OrderID, heldOrder.TotalMinor); cerr != nil {
							return cerr
						}
					}
					o = heldOrder
					bo = nil
					invPlan = InventoryPlan{Fulfillable: heldOrder.LineItems}
					allBackordered = false
					return nil
				}
				if len(evaluated.Lines) == 0 {
					return ErrInventoryExhausted
				}
				evalLines = evaluated.Lines
			}

			txPlan, perr := PlanInventoryCheckoutInTxn(txCtx, txn, supplierID, warehouseID, evalLines, policyOverride)
			if perr != nil {
				return perr
			}
			if len(txPlan.Fulfillable) == 0 && len(txPlan.Backorder) == 0 {
				return ErrInventoryExhausted
			}

			if len(txPlan.Fulfillable) == 0 && len(txPlan.Backorder) > 0 {
				var boTotal int64
				for _, li := range txPlan.Backorder {
					boTotal += li.Quantity * li.UnitPrice
				}
				boCurrency := strings.TrimSpace(orderCurrency)
				if boCurrency == "" {
					boCurrency = s.currency
				}
				boOnly := Order{
					OrderID:            boID,
					SupplierID:         supplierID,
					RetailerID:         retailerID,
					ParentOrderID:      strings.TrimSpace(req.ParentOrderID),
					IdempotencyKey:     explicitClientOrderID,
					WarehouseID:        warehouseID,
					Status:             StatusBackordered,
					Source:             OrderSourceBackorder,
					ConfirmationStatus: ConfirmationStatusConfirmed,
					AutoConfirmOptIn:   req.AutoConfirmOptIn,
					LineItems:          txPlan.Backorder,
					TotalMinor:         boTotal,
					Currency:           boCurrency,
					H3Cell:             req.H3Cell,
					Lat:                req.Lat,
					Lng:                req.Lng,
					Version:            1,
					CreatedAt:          now,
					UpdatedAt:          now,
				}
				boEmit := func(outTxn outbox.TxnBuffer) error {
					if err := outbox.EmitJSON(txCtx, outTxn, events.AggregateOrder, boOnly.OrderID, events.TopicMain, events.OrderEvent{
						BaseEvent:          events.BaseEvent{Type: events.EventOrderCreated, Timestamp: boOnly.CreatedAt.Format(time.RFC3339Nano)},
						OrderID:            boOnly.OrderID,
						SupplierID:         boOnly.SupplierID,
						RetailerID:         boOnly.RetailerID,
						WarehouseID:        boOnly.WarehouseID,
						Status:             string(boOnly.Status),
						OrderSource:        string(boOnly.Source),
						ConfirmationStatus: string(boOnly.ConfirmationStatus),
						TotalMinor:         boOnly.TotalMinor,
						Currency:           boOnly.Currency,
						LineItems:          boOnly.LineItems,
					}); err != nil {
						return err
					}
					if ab, ok := outTxn.(outbox.AuditBuffer); ok {
						creditDelta, cashDelta := usageDeltasForPayment(paymentMethod, boOnly.TotalMinor)
						return outbox.WriteAudit(txCtx, ab, boOnly.SupplierID, retailerID, "RETAILER", "ORDER_CREATED", "Order", boOnly.OrderID, map[string]any{
							"usage_user_id":      usageUserID,
							"usage_date":         usageDate.String(),
							"payment_method":     paymentMethod,
							"spent_credit_minor": creditDelta,
							"spent_cash_minor":   cashDelta,
							"order_count":        int64(1),
						})
					}
					return nil
				}
				if bErr := bufferUsageBeforeOrder(boOnly.TotalMinor); bErr != nil {
					return bErr
				}
				if werr := txnWriter.writeOrderWithBackorderInTxn(txCtx, txn, &boOnly, nil, boEmit, StockReservationOpts{}); werr != nil {
					return fmt.Errorf("persist backorder: %w", werr)
				}
				o = boOnly
				bo = nil
				invPlan = txPlan
				allBackordered = true
				return nil
			}

			if ierr := acquireInflight(txCtx, txPlan.Fulfillable, txPlan.AvailableStock); ierr != nil {
				return ierr
			}

			var txTotal int64
			for _, li := range txPlan.Fulfillable {
				txTotal += li.Quantity * li.UnitPrice
			}
			txTotal += deliveryFeeMinor

			txOrder, txBO := buildPrimaryAndBackorder(txPlan, txTotal)
			if shortPlan != nil {
				txOrder.Substitutions = shortPlan.Substitutions
				txOrder.CutLines = shortPlan.CutLines
			}
			stockOpts := StockReservationOpts{
				Skip: deferStockReservationAtCreate(s.allocationRequired, txOrder.Status, source, warehouseID, len(txOrder.LineItems)),
			}

			if stockOpts.Skip && txOrder.Status == StatusPending {
				prepAlloc, aerr := s.prepareAllocationInTxn(txCtx, txn, &txOrder)
				if aerr != nil {
					if errors.Is(aerr, ErrOrderHoldRequired) {
						txOrder.Status = StatusDelayed
						txOrder.ConfirmationStatus = ConfirmationStatusPendingWarehouse
						txOrder.HoldReason = aerr.Error()
						txOrder.WarehouseNotes = aerr.Error()
						if bErr := bufferUsageBeforeOrder(txOrder.TotalMinor); bErr != nil {
							return bErr
						}
						if werr := txnWriter.writeOrderWithBackorderInTxn(txCtx, txn, &txOrder, nil, buildMainEmit(txCtx, &txOrder, nil), StockReservationOpts{Skip: true}); werr != nil {
							return fmt.Errorf("persist held order: %w", werr)
						}
						if paymentMethod == PaymentMethodCredit {
							if cerr := s.reserveCreditForOrderInTxn(txCtx, txn, retailerID, supplierID, txOrder.OrderID, txOrder.TotalMinor); cerr != nil {
								return cerr
							}
						}
						o = txOrder
						bo = nil
						invPlan = txPlan
						allBackordered = false
						return nil
					}
					return fmt.Errorf("allocation failed: %w", aerr)
				}
				if prepAlloc != nil && prepAlloc.targetWarehouse != "" {
					txOrder.WarehouseID = prepAlloc.targetWarehouse
				}
				if bErr := bufferUsageBeforeOrder(txOrder.TotalMinor); bErr != nil {
					return bErr
				}
				if werr := txnWriter.writeOrderWithBackorderInTxn(txCtx, txn, &txOrder, txBO, buildMainEmit(txCtx, &txOrder, txBO), StockReservationOpts{Skip: true}); werr != nil {
					return fmt.Errorf("persist order: %w", werr)
				}
				if cerr := s.commitAllocationInTxn(txCtx, txn, &txOrder, prepAlloc, false); cerr != nil {
					return fmt.Errorf("allocation failed: %w", cerr)
				}
			} else {
				if bErr := bufferUsageBeforeOrder(txOrder.TotalMinor); bErr != nil {
					return bErr
				}
				if werr := txnWriter.writeOrderWithBackorderInTxn(txCtx, txn, &txOrder, txBO, buildMainEmit(txCtx, &txOrder, txBO), stockOpts); werr != nil {
					return fmt.Errorf("persist order: %w", werr)
				}
			}

			if paymentMethod == PaymentMethodCredit {
				if cerr := s.reserveCreditForOrderInTxn(txCtx, txn, retailerID, supplierID, txOrder.OrderID, txOrder.TotalMinor); cerr != nil {
					return cerr
				}
			}

			o = txOrder
			bo = txBO
			invPlan = txPlan
			allBackordered = false
			return nil
		})
		if err != nil {
			var approvalErr *OwnerApprovalRequiredError
			if errors.As(err, &approvalErr) && !isMandatePrechecked(ctx) {
				gateID := explicitClientOrderID
				if gateID == "" {
					gateID = newMandateUUID()
				}
				reqCopy := req
				reqCopy.SupplierID = supplierID
				reqCopy.PaymentMethod = paymentMethod
				reqCopy.ClientOrderID = gateID
				if gateErr := s.checkSellerMandateGate(ctx, retailerID, usageUserID, usageRole, supplierID, gateID, paymentMethod, approvalErr.AmountMinor, ApprovalPayloadEnvelope{
					Kind:          ApprovalPayloadCreate,
					SupplierID:    supplierID,
					Currency:      orderCurrency,
					CreateRequest: &reqCopy,
				}); gateErr != nil {
					return CreateResponse{}, gateErr
				}
				return CreateResponse{}, approvalErr
			}
			var ice *InventoryCheckoutError
			if errors.Is(err, ErrInventoryExhausted) ||
				errors.Is(err, ErrCreditLimitBreached) ||
				errors.As(err, &ice) {
				return CreateResponse{}, err
			}
			return CreateResponse{}, err
		}
		if existingDuplicate {
			return CreateResponse{
				OrderID:               o.OrderID,
				WarehouseID:           o.WarehouseID,
				Status:                o.Status,
				Source:                o.Source,
				ConfirmationStatus:    o.ConfirmationStatus,
				AutoConfirmOptIn:      o.AutoConfirmOptIn,
				HoldReason:            o.HoldReason,
				RequestedDeliveryDate: formatOptionalRFC3339(o.RequestedDeliveryDate),
				DeliverBefore:         formatOptionalRFC3339(o.DeliverBefore),
				DeliveryPriority:      string(o.DeliveryPriority),
				DeliveryMode:          normalizeDeliveryMode(req.DeliveryMode),
				TotalMinor:            o.TotalMinor,
				Currency:              o.Currency,
				CreatedAt:             o.CreatedAt.UTC().Format(time.RFC3339Nano),
				ReceivingWindowOpen:   o.ReceivingWindowOpen,
				ReceivingWindowClose:  o.ReceivingWindowClose,
				LineItems:             append([]LineItem(nil), o.LineItems...),
			}, nil
		}
	} else {
		evalLines := baseLines
		evalTotal := baseTotal
		var shortPlan *OrderShortagePlan
		if source != OrderSourceBackorder && s.shortageStockReader != nil {
			workingLines, sc, scErr := s.loadOrderShortageContext(ctx, nil, supplierID, warehouseID, baseLines)
			if scErr != nil {
				return CreateResponse{}, scErr
			}
			evaluated := EvaluateOrderShortages(workingLines, sc.AvailableBySKU, sc.CandidatesBySKU, OrderShortageOptions{
				SupplierID:       supplierID,
				EnforceAllowlist: sc.EnforceAllowlist,
			})
			shortPlan = &evaluated
			if !evaluated.HoldOrder && len(evaluated.Lines) == 0 {
				return CreateResponse{}, ErrInventoryExhausted
			}
			evalLines = evaluated.Lines
			evalTotal = evaluated.UpdatedTotalMinor
		}

		invPlan.Fulfillable = evalLines
		total = evalTotal + deliveryFeeMinor
		if len(invPlan.Fulfillable) == 0 && len(invPlan.Backorder) == 0 {
			return CreateResponse{}, ErrInventoryExhausted
		}
		o, bo = buildPrimaryAndBackorder(invPlan, total)
		if shortPlan != nil {
			o.Substitutions = shortPlan.Substitutions
			o.CutLines = shortPlan.CutLines
			if shortPlan.HoldOrder {
				o.Status = StatusDelayed
				o.ConfirmationStatus = ConfirmationStatusPendingWarehouse
				o.HoldReason = shortPlan.HoldReason
				o.WarehouseNotes = shortPlan.HoldReason
			}
		}
		stockOpts := StockReservationOpts{
			Skip: deferStockReservationAtCreate(s.allocationRequired, o.Status, source, warehouseID, len(o.LineItems)),
		}
		if s.spannerClient == nil {
			recordLockStage(ctx, s.lockOrderTracer, LockStageUsage)
			if paymentMethod == PaymentMethodCredit {
				recordLockStage(ctx, s.lockOrderTracer, LockStageCreditProfile)
			}
			recordLockStage(ctx, s.lockOrderTracer, LockStageOrders)
		}
		createCtx := withPreOrderTxnPlan(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction, _ *Order) error {
			lockRes, lockErr := s.acquireCreateLocksInTxn(txCtx, txn, retailerID, usageUserID, supplierID, usageDate, paymentMethod, o.TotalMinor, mandateLookup, enforceCapInTxn)
			if lockErr != nil {
				return lockErr
			}
			uMut, _, _ := buildSellerDailyUsageMutation(retailerID, usageUserID, usageDate, paymentMethod, o.TotalMinor, lockRes)
			if err := txn.BufferWrite([]*spanner.Mutation{uMut}); err != nil {
				return err
			}
			recordLockStage(txCtx, s.lockOrderTracer, LockStageOrders)
			return nil
		})
		inTxn := func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
			if paymentMethod != PaymentMethodCredit {
				return nil
			}
			return s.reserveCreditForOrderInTxn(txCtx, txn, retailerID, supplierID, o.OrderID, o.TotalMinor)
		}
		if err := s.repo.CreateOrderWithBackorder(createCtx, &o, bo, inTxn, buildMainEmit(ctx, &o, bo), stockOpts); err != nil {
			return CreateResponse{}, fmt.Errorf("persist order: %w", err)
		}
		if s.spannerClient == nil {
			s.recordOrderDailyUsageMemory(o.OrderID, retailerID, usageUserID, usageDate, paymentMethod, o.TotalMinor)
			s.recordOrderIdempotencyMemory(o)
		}
	}

	if allBackordered {
		if s.cache != nil {
			s.cache.Invalidate(ctx, append(
				[]string{retailerOrdersKey(o.RetailerID), supplierOrdersKey(o.SupplierID)},
				dashboardCacheKeys(o.SupplierID, o.WarehouseID)...,
			)...)
		}
		return CreateResponse{
			OrderID:              o.OrderID,
			WarehouseID:          o.WarehouseID,
			Status:               StatusBackordered,
			Source:               OrderSourceBackorder,
			ConfirmationStatus:   ConfirmationStatusConfirmed,
			AutoConfirmOptIn:     o.AutoConfirmOptIn,
			TotalMinor:           o.TotalMinor,
			Currency:             o.Currency,
			CreatedAt:            o.CreatedAt.Format(time.RFC3339Nano),
			BackorderedItemCount: invPlan.BackorderCount,
			StockWarnings:        invPlan.Warnings,
		}, nil
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
		AutoConfirmOptIn:      o.AutoConfirmOptIn,
		HoldReason:            o.HoldReason,
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
		LineItems:             append([]LineItem(nil), o.LineItems...),
		Substitutions:         append([]ShortageEvaluationResult(nil), o.Substitutions...),
		CutLines:              append([]ShortageEvaluationResult(nil), o.CutLines...),
	}, nil
}

func (s *Service) validateOrderPackMultiples(ctx context.Context, items []LineItem) error {
	return validatePackMultiplesWithReader(ctx, s.spannerClient, s.packRuleReader, items)
}

func hasLineShortageConstraints(items []LineItem) bool {
	for _, it := range items {
		if it.MustHave || it.AllowSubstitutes || len(it.AllowedSubstituteSKUs) > 0 {
			return true
		}
	}
	return false
}

func isOrderEligibleForAssignment(o Order) bool {
	switch o.Status {
	case StatusDelayed, StatusScheduled, StatusAutoAccepted, StatusCancelled, StatusBackordered, StatusCompleted:
		return false
	}
	switch o.ConfirmationStatus {
	case ConfirmationStatusDraft, ConfirmationStatusPending, ConfirmationStatusPendingWarehouse, ConfirmationStatusRejected:
		return false
	}
	return true
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

func (s *Service) reserveCreditForOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, totalMinor int64) error {
	if s.credit == nil || totalMinor <= 0 || !creditReserveAtCreateEnabled() {
		return nil
	}
	check, cerr := s.credit.CheckCreditPath(ctx, retailerID, supplierID, totalMinor)
	if cerr != nil {
		return fmt.Errorf("credit check failed: %w", cerr)
	}
	if !check.Allowed {
		if isOptionalCreditSkipReason(check.Reason) {
			if s.log != nil {
				s.log.Info("credit reserve skipped", "reason", check.Reason, "retailer_id", retailerID, "supplier_id", supplierID)
			}
			return nil
		}
		return ErrCreditLimitBreached
	}
	if txn == nil {
		return nil
	}
	if rerr := s.credit.ReserveOrderInTxn(ctx, txn, retailerID, supplierID, orderID, totalMinor); rerr != nil {
		if errors.Is(rerr, credit.ErrProfileNotFound) {
			if s.log != nil {
				s.log.Info("credit reserve skipped", "reason", "profile_not_found", "retailer_id", retailerID, "supplier_id", supplierID)
			}
			return nil
		}
		if errors.Is(rerr, credit.ErrLimitBreached) || errors.Is(rerr, credit.ErrCreditNotEnabled) {
			return ErrCreditLimitBreached
		}
		return fmt.Errorf("credit reserve failed: %w", rerr)
	}
	return nil
}

func isOptionalCreditSkipReason(reason string) bool {
	return reason == "no_credit_profile" ||
		strings.HasPrefix(reason, "credit_program_") ||
		strings.HasPrefix(reason, "credit_relationship_")
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
		if orgID == "" || orgID != current.RetailerID || (nextStatus != StatusCancelled && nextStatus != StatusCancelRequested) {
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

	emitStatusUpdate := func(txn outbox.TxnBuffer) error {
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
	}
	if nextStatus == StatusCompleted {
		preFee, feeErr := s.resolvePerOrderFeeBeforeTxn(ctx, current.SupplierID)
		if feeErr != nil {
			return UpdateStatusResponse{}, fmt.Errorf("resolve per-order fee %s: %w", orderID, feeErr)
		}
		err = s.repo.UpdateOrderWithTxn(ctx, current, nil, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
			return s.recordOrderDeliveredGLInTxn(txnCtx, txn, nil, &current, 0, 0, deliveredGrossFromOrder(current), preFee, "", current.UpdatedAt)
		}, emitStatusUpdate)
	} else {
		err = s.repo.UpdateOrder(ctx, current, nil, emitStatusUpdate)
	}
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
	if !isOrderEligibleForAssignment(current) {
		return AssignOrderResponse{}, fmt.Errorf("%w: order status=%s confirmation_status=%s not eligible for assignment",
			ErrInvalidStatusTransition, current.Status, current.ConfirmationStatus)
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
	var wasDeliveredOnCredit bool
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
			if orderRecord.Status == StatusDeliveredOnCredit {
				wasDeliveredOnCredit = true
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
			skipDoorstepEscrowShortfall := wasDeliveredOnCredit || orderRecord.Status == StatusDeliveredOnCredit
			if !skipDoorstepEscrowShortfall {
				deliveredGL, gErr := hasOrderDeliveredGLInTxn(txnCtx, txn, orderRecord.SupplierID, req.OrderID)
				if gErr != nil {
					return gErr
				}
				skipDoorstepEscrowShortfall = deliveredGL
			}
			if !skipDoorstepEscrowShortfall {
				if err := s.recordDoorstepShortfallInTxn(txnCtx, txn, req.OrderID, orderRecord.SupplierID, claims.Subject, orderRecord.Currency, orderRecord.TotalMinor, receivedMinor); err != nil {
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
			if s.ar != nil && receivedMinor > 0 {
				if err := s.ar.RecordPaymentForOrderWithModeInTxn(txnCtx, txn, req.OrderID, receivedMinor, "CASH",
					"ar-cash-collect-"+req.OrderID, orderRecord.Currency); err != nil {
					return err
				}
			} else if orderRecord.Status == StatusDeliveredOnCredit && ar.InvoicesEnabled() && orderRecord.TotalMinor > 0 {
				return fmt.Errorf("ar service required for credit cash collection")
			}
			// G1-A2: clear credit headroom when cash settles a credit-left order (same txn).
			// No-op when reservation was never CONVERTED for this order.
			if s.credit != nil && receivedMinor > 0 {
				clearAmt := receivedMinor
				if orderRecord.TotalMinor > 0 && clearAmt > orderRecord.TotalMinor {
					clearAmt = orderRecord.TotalMinor
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
					b, merr := json.Marshal(payload)
					if merr != nil {
						s.log.Warn("sibling broadcast marshal failed", "order_id", current.OrderID, "err", merr)
						continue
					}

					go func(sibID string, msg []byte) {
						defer func() {
							if r := recover(); r != nil {
								s.log.Error("sibling driver broadcast panic", "driver_id", sibID, "panic", r)
							}
						}()
						bctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
						defer cancel()
						msg = drivercmd.PrepareAndRecordJSON(bctx, s.spannerClient, sibID, msg)
						s.driverHub.Broadcast(bctx, "driver:"+sibID, msg)
					}(sib, b)
				}
			}
		}
	}
}

func (s *Service) recordDoorstepShortfallInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	orderID, supplierID, driverID, currency string,
	expectedMinor, receivedMinor int64,
) error {
	if expectedMinor <= receivedMinor {
		return nil
	}
	if s.doorstepShortfall == nil {
		if txn != nil {
			return fmt.Errorf("collect cash: doorstep shortfall journal hook required for order %s", orderID)
		}
		return nil
	}
	return s.doorstepShortfall(ctx, txn, orderID, supplierID, driverID, currency, expectedMinor, receivedMinor, s.now())
}

// HandleCreate is POST /v1/order/create. Wired by orderroutes.RegisterRoutes
// behind auth.RequireRole(RETAILER).
