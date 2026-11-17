package order

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"google.golang.org/grpc/codes"
)

// StartPreorderSweeper runs Midnight Guard + scheduled promoter loops.
func StartPreorderSweeper(svc *Service) {
	StartPreorderSweeperContext(context.Background(), svc)
}

// StartPreorderSweeperContext runs Midnight Guard + scheduled promoter loops with context cancellation and optional WaitGroup.
func StartPreorderSweeperContext(ctx context.Context, svc *Service, wg ...*sync.WaitGroup) {
	if svc == nil || svc.spannerClient == nil {
		return
	}
	svc.log.Info("preorder sweeper started")

	var waitGroup *sync.WaitGroup
	if len(wg) > 0 && wg[0] != nil {
		waitGroup = wg[0]
		waitGroup.Add(1)
	}

	go func() {
		if waitGroup != nil {
			defer waitGroup.Done()
		}
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sweepCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
				if err := svc.RunPreorderSweeperOnce(sweepCtx); err != nil {
					svc.log.Warn("preorder sweeper tick failed", "err", err)
				}
				cancel()
			}
		}
	}()
}

// RunPreorderSweeperOnce executes one Midnight Guard + promotion pass.
func (s *Service) RunPreorderSweeperOnce(ctx context.Context) error {
	now := s.now()
	if v := preorderSweeperNowOverride(); v != nil {
		now = v.UTC()
	}
	if err := s.sweepPreorderNotifications(ctx, now); err != nil {
		return err
	}
	if err := s.sweepPreorderAutoAccept(ctx, now); err != nil {
		return err
	}
	return s.sweepPreorderPromote(ctx, now)
}

func preorderSweeperNowOverride() *time.Time {
	raw := strings.TrimSpace(os.Getenv("PREORDER_SWEEPER_NOW"))
	if raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &t
}

func (s *Service) sweepPreorderNotifications(ctx context.Context, now time.Time) error {
	const pageSize = 200
	var afterDate *time.Time
	var afterOrderID string
	hasCursor := false

	for {
		orders, err := s.listScheduledPreordersPage(ctx, pageSize, hasCursor, afterDate, afterOrderID)
		if err != nil {
			return err
		}
		if len(orders) == 0 {
			break
		}
		for _, o := range orders {
			loc, locErr := resolveCalendarLocation(ctx, o.SupplierID, o.Timezone)
			if locErr != nil {
				continue
			}
			today := proximity.TodayStart(now, loc)

			if o.RequestedDeliveryDate == nil {
				continue
			}
			lead := PreorderLeadDays(now, o.RequestedDeliveryDate, loc)
			phase := PreorderGuardPhase(lead)
			if phase == "" {
				continue
			}
			deliveryDay := proximity.TodayStart(*o.RequestedDeliveryDate, loc)
			if phase == "long" {
				if lead >= 7 && (o.PreorderReminderSentAt == nil || now.Sub(*o.PreorderReminderSentAt) >= 48*time.Hour) {
					if err := s.patchPreorderNotify(ctx, o, now, events.EventPreOrderNudge, func(u *Order) {
						t := now
						u.PreorderReminderSentAt = &t
					}); err != nil {
						s.log.Warn("preorder reminder failed", "order_id", o.OrderID, "err", err)
					}
				}
				t5 := today.AddDate(0, 0, 5)
				if deliveryDay.Equal(t5) && o.NudgeNotifiedAt == nil {
					if err := s.patchPreorderNotify(ctx, o, now, events.EventPreOrderNudge, func(u *Order) {
						t := now
						u.NudgeNotifiedAt = &t
					}); err != nil {
						s.log.Warn("preorder t5 nudge failed", "order_id", o.OrderID, "err", err)
					}
				}
				t4 := today.AddDate(0, 0, 4)
				if deliveryDay.Equal(t4) && o.ConfirmationNotifiedAt == nil {
					if err := s.patchPreorderNotify(ctx, o, now, events.EventPreOrderConfirmation, func(u *Order) {
						t := now
						u.ConfirmationNotifiedAt = &t
						u.CancelLockedAt = &t
						u.CancelLockReason = "T4_CANCEL_LOCK"
					}); err != nil {
						s.log.Warn("preorder t4 confirmation failed", "order_id", o.OrderID, "err", err)
					}
				}
			} else {
				// compressed 3–6 day lead
				if o.NudgeNotifiedAt == nil {
					if err := s.patchPreorderNotify(ctx, o, now, events.EventPreOrderNudge, func(u *Order) {
						t := now
						u.NudgeNotifiedAt = &t
					}); err != nil {
						s.log.Warn("preorder compressed nudge failed", "order_id", o.OrderID, "err", err)
					}
				}
				lockDay := deliveryDay.AddDate(0, 0, -PreorderEditLockDays)
				if !today.Before(lockDay) && o.CancelLockedAt == nil {
					if err := s.patchPreorderNotify(ctx, o, now, events.EventPreOrderConfirmation, func(u *Order) {
						t := now
						u.ConfirmationNotifiedAt = &t
						u.CancelLockedAt = &t
						u.CancelLockReason = "T2_CANCEL_LOCK"
					}); err != nil {
						s.log.Warn("preorder compressed lock failed", "order_id", o.OrderID, "err", err)
					}
				}
			}
		}
		if len(orders) < pageSize {
			break
		}
		last := orders[len(orders)-1]
		afterDate = last.RequestedDeliveryDate
		afterOrderID = last.OrderID
		hasCursor = true
	}
	return nil
}

func (s *Service) sweepPreorderAutoAccept(ctx context.Context, now time.Time) error {
	const pageSize = 200
	var afterDate *time.Time
	var afterOrderID string
	hasCursor := false

	for {
		orders, err := s.listScheduledPreordersPage(ctx, pageSize, hasCursor, afterDate, afterOrderID)
		if err != nil {
			return err
		}
		if len(orders) == 0 {
			break
		}
		for _, o := range orders {
			if o.Status != StatusScheduled || o.RequestedDeliveryDate == nil {
				continue
			}
			loc, locErr := resolveCalendarLocation(ctx, o.SupplierID, o.Timezone)
			if locErr != nil {
				continue
			}
			today := proximity.TodayStart(now, loc)
			lead := PreorderLeadDays(now, o.RequestedDeliveryDate, loc)
			deliveryDay := proximity.TodayStart(*o.RequestedDeliveryDate, loc)
			shouldAccept := false
			if PreorderGuardPhase(lead) == "long" {
				t4 := today.AddDate(0, 0, 4)
				shouldAccept = o.CancelLockedAt != nil && !deliveryDay.After(t4)
			} else if lead >= PreorderMinScheduledLeadDays {
				lockDay := deliveryDay.AddDate(0, 0, -PreorderEditLockDays)
				shouldAccept = o.CancelLockedAt != nil && !today.Before(lockDay)
			}
			if !shouldAccept {
				continue
			}
			if o.ConfirmationStatus == ConfirmationStatusPendingWarehouse {
				continue
			}
			alreadyConfirmed := o.ConfirmationStatus == ConfirmationStatusConfirmed || o.ConfirmationStatus == ConfirmationStatusAutoConfirmed
			if o.ConfirmationStatus == ConfirmationStatusRejected || (!alreadyConfirmed && !o.AutoConfirmOptIn) {
				if _, expErr := s.expireUnconfirmedProposal(ctx, o, now, "PREORDER_AUTO_CONFIRM_OPT_OUT_EXPIRED", "system:midnight_guard"); expErr != nil {
					s.log.Warn("preorder opt-out expiry failed", "order_id", o.OrderID, "err", expErr)
				}
				continue
			}
			if err := ValidateStatusTransition(string(o.Status), string(StatusAutoAccepted), TransitionOpts{}); err != nil {
				s.log.Warn("preorder auto-accept transition blocked", "order_id", o.OrderID, "err", err)
				continue
			}
			updated := o
			updated.Status = StatusAutoAccepted
			if updated.ConfirmationStatus != ConfirmationStatusConfirmed {
				updated.ConfirmationStatus = ConfirmationStatusAutoConfirmed
				decisionAt := now
				updated.DecisionAt = &decisionAt
				updated.DecisionBy = "system:midnight_guard"
			}
			updated.AutoConfirmAt = nil
			updated.UpdatedAt = now
			if err := s.repo.UpdateOrder(ctx, updated, nil, func(txn outbox.TxnBuffer) error {
				return emitPreorderEvent(ctx, txn, events.EventPreOrderAutoAccepted, updated, "SYSTEM", "system:midnight_guard")
			}); err != nil {
				s.log.Warn("preorder auto-accept failed", "order_id", o.OrderID, "err", err)
				continue
			}
			s.afterOrderMutation(ctx, updated)
		}
		if len(orders) < pageSize {
			break
		}
		last := orders[len(orders)-1]
		afterDate = last.RequestedDeliveryDate
		afterOrderID = last.OrderID
		hasCursor = true
	}
	return nil
}

func (s *Service) sweepPreorderPromote(ctx context.Context, now time.Time) error {
	today := proximity.PackTodayStart(now)
	if today.IsZero() {
		return nil
	}
	// Prefetch pre-orders due within the next calendar day (T-1 promotion window).
	cutoff := today.AddDate(0, 0, 1)
	const pageSize = 100
	var afterDate *time.Time
	var afterOrderID string

	for {
		orders, err := s.listPreorderPromotePage(ctx, cutoff, pageSize, afterDate, afterOrderID)
		if err != nil {
			return err
		}
		if len(orders) == 0 {
			break
		}
		for _, o := range orders {
			if o.RequestedDeliveryDate == nil {
				continue
			}
			loc, locErr := resolveCalendarLocation(ctx, o.SupplierID, o.Timezone)
			if locErr != nil {
				continue
			}
			lead := PreorderLeadDays(now, o.RequestedDeliveryDate, loc)
			// Promote at T-1 (day before delivery) or on delivery day if a tick was missed.
			if lead > 1 {
				continue
			}
			if err := s.promotePreorderToPending(ctx, o, now); err != nil {
				s.log.Warn("preorder promote failed", "order_id", o.OrderID, "err", err)
			}
		}
		if len(orders) < pageSize {
			break
		}
		last := orders[len(orders)-1]
		afterDate = last.RequestedDeliveryDate
		afterOrderID = last.OrderID
	}
	return nil
}

func isConfirmedForDispatch(c ConfirmationStatus) bool {
	return c == ConfirmationStatusConfirmed || c == ConfirmationStatusAutoConfirmed
}

func (s *Service) promotePreorderToPending(ctx context.Context, o Order, now time.Time) error {
	if o.Status != StatusScheduled && o.Status != StatusAutoAccepted {
		return nil
	}
	if o.ConfirmationStatus == ConfirmationStatusPendingWarehouse {
		return nil
	}
	if o.ConfirmationStatus == ConfirmationStatusRejected || (!isConfirmedForDispatch(o.ConfirmationStatus) && !o.AutoConfirmOptIn) {
		_, err := s.expireUnconfirmedProposal(ctx, o, now, "UNCONFIRMED_PREORDER_EXPIRED", "system:midnight_guard")
		return err
	}

	emitPromoEvent := func(txn outbox.TxnBuffer, prev Status, updated Order, reason string) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, updated.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:          events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:            updated.OrderID,
			SupplierID:         updated.SupplierID,
			RetailerID:         updated.RetailerID,
			WarehouseID:        updated.WarehouseID,
			PreviousStatus:     string(prev),
			Status:             string(updated.Status),
			Reason:             reason,
			OrderSource:        string(updated.Source),
			ConfirmationStatus: string(updated.ConfirmationStatus),
		})
	}

	if mut, ok := s.repo.(txnOrderMutator); ok {
		changed := false
		updated, err := mut.mutateOrderWithTxn(ctx, o.OrderID, 0, func(txCtx context.Context, txn *spanner.ReadWriteTransaction, curr *Order) (func(outbox.TxnBuffer) error, error) {
			changed = false
			prev, reason, skipAlloc, alreadyReserved, origLines, err := s.applyPreorderPromotionPlan(txCtx, txn, curr, now)
			if err != nil || prev == "" {
				return nil, err
			}
			changed = true

			if !skipAlloc && s.allocationRequired {
				if alreadyReserved && len(origLines) > 0 {
					if err := ReleaseReservationsForOrderInTxn(txCtx, txn, curr.SupplierID, curr.WarehouseID, curr.OrderID, curr.Source, origLines); err != nil {
						return nil, fmt.Errorf("release prior preorder reservation: %w", err)
					}
				}
				prep, err := s.prepareAllocationInTxn(txCtx, txn, curr)
				if errors.Is(err, ErrOrderHoldRequired) {
					if verr := ValidateStatusTransition(string(prev), string(StatusDelayed), TransitionOpts{
						Actor:  "system:midnight_guard",
						Reason: err.Error(),
					}); verr != nil {
						return nil, verr
					}
					curr.Status = StatusDelayed
					curr.ConfirmationStatus = ConfirmationStatusPendingWarehouse
					curr.HoldReason = err.Error()
					curr.WarehouseNotes = err.Error()
					curr.TransitionReason = err.Error()
					curr.TransitionActorRole = "SYSTEM"
					curr.TransitionActorID = "system:midnight_guard"
					curr.TransitionEventKind = "HOLD_ORDER"
					reason = err.Error()
				} else if err != nil {
					return nil, err
				} else {
					curr.WarehouseID = prep.targetWarehouse
					if err := s.commitAllocationInTxn(txCtx, txn, curr, prep, false); err != nil {
						return nil, err
					}
				}
			}

			snapshot := *curr
			return func(buf outbox.TxnBuffer) error {
				return emitPromoEvent(buf, prev, snapshot, reason)
			}, nil
		})
		if err != nil {
			return err
		}
		if changed {
			s.afterOrderMutation(ctx, updated)
		}
		return nil
	}

	prev, reason, _, _, _, err := s.applyPreorderPromotionPlan(ctx, nil, &o, now)
	if err != nil || prev == "" {
		return err
	}
	if err := s.repo.UpdateOrder(ctx, o, nil, func(txn outbox.TxnBuffer) error {
		return emitPromoEvent(txn, prev, o, reason)
	}); err != nil {
		return err
	}
	if o.Status == StatusCancelled && s.credit != nil {
		if cerr := s.credit.ReleaseReserve(ctx, o.OrderID); cerr != nil {
			s.log.Warn("credit reserve release on cut/expired preorder failed", "order_id", o.OrderID, "err", cerr)
		}
	}
	o.Version++
	s.afterOrderMutation(ctx, o)
	return nil
}

func (s *Service) applyPreorderPromotionPlan(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	target *Order,
	now time.Time,
) (prev Status, reason string, skipAllocation bool, alreadyReserved bool, origLines []LineItem, err error) {
	if target.Status != StatusScheduled && target.Status != StatusAutoAccepted {
		return "", "", true, false, nil, nil
	}
	if target.ConfirmationStatus == ConfirmationStatusPendingWarehouse {
		return "", "", true, false, nil, nil
	}
	prev = target.Status
	origLines = append([]LineItem(nil), target.LineItems...)

	if target.ConfirmationStatus == ConfirmationStatusRejected || (!isConfirmedForDispatch(target.ConfirmationStatus) && !target.AutoConfirmOptIn) {
		if verr := ValidateStatusTransition(string(prev), string(StatusCancelled), TransitionOpts{
			Actor:  "system:midnight_guard",
			Reason: "UNCONFIRMED_PREORDER_EXPIRED",
		}); verr != nil {
			return "", "", true, false, nil, verr
		}
		decAt := now
		target.Status = StatusCancelled
		target.ConfirmationStatus = ConfirmationStatusRejected
		target.AutoConfirmAt = nil
		target.DecisionAt = &decAt
		target.DecisionBy = "system:midnight_guard"
		target.UpdatedAt = now
		target.TransitionReason = "UNCONFIRMED_PREORDER_EXPIRED"
		target.TransitionActorRole = "SYSTEM"
		target.TransitionActorID = "system:midnight_guard"
		target.TransitionEventKind = "EXPIRE"
		if txn != nil && s.credit != nil {
			if cerr := s.credit.AdjustReserveInTxn(ctx, txn, target.OrderID, 0); cerr != nil {
				return "", "", true, false, nil, fmt.Errorf("release credit reserve on unconfirmed preorder expiry: %w", cerr)
			}
		}
		return prev, "UNCONFIRMED_PREORDER_EXPIRED", true, false, origLines, nil
	}

	if !isConfirmedForDispatch(target.ConfirmationStatus) && target.AutoConfirmOptIn {
		decAt := now
		target.ConfirmationStatus = ConfirmationStatusAutoConfirmed
		target.DecisionAt = &decAt
		target.DecisionBy = "system:midnight_guard"
		target.AutoConfirmAt = nil
	}

	if txn != nil && s.spannerClient != nil {
		if _, rerr := txn.ReadRow(ctx, "OrderStockReservationMarkers", spanner.Key{target.OrderID}, []string{"OrderId"}); rerr == nil {
			alreadyReserved = true
		} else if spanner.ErrCode(rerr) != codes.NotFound {
			return "", "", true, false, nil, fmt.Errorf("check reservation marker %s: %w", target.OrderID, rerr)
		}
	}

	if s.shortageStockReader != nil || (txn != nil && s.spannerClient != nil) {
		workingLines, sc, scErr := s.loadOrderShortageContext(ctx, txn, target.SupplierID, target.WarehouseID, target.LineItems)
		if scErr != nil {
			return "", "", true, false, nil, scErr
		}
		avail := make(map[string]int64, len(sc.AvailableBySKU)+len(target.LineItems))
		for sku, qty := range sc.AvailableBySKU {
			avail[sku] = qty
		}
		if alreadyReserved {
			for sku, qty := range reservationCreditForLines(target.LineItems) {
				avail[sku] += qty
			}
		}
		plan := EvaluateOrderShortages(workingLines, avail, sc.CandidatesBySKU, OrderShortageOptions{
			SupplierID:       target.SupplierID,
			EnforceAllowlist: sc.EnforceAllowlist,
		})
		if plan.HoldOrder {
			if verr := ValidateStatusTransition(string(prev), string(StatusDelayed), TransitionOpts{
				Actor:  "system:midnight_guard",
				Reason: plan.HoldReason,
			}); verr != nil {
				return "", "", true, false, nil, verr
			}
			target.Status = StatusDelayed
			target.ConfirmationStatus = ConfirmationStatusPendingWarehouse
			target.HoldReason = plan.HoldReason
			target.WarehouseNotes = plan.HoldReason
			target.UpdatedAt = now
			target.TransitionReason = plan.HoldReason
			target.TransitionActorRole = "SYSTEM"
			target.TransitionActorID = "system:midnight_guard"
			target.TransitionEventKind = "HOLD_ORDER"
			return prev, plan.HoldReason, true, alreadyReserved, origLines, nil
		}
		if plan.HasModifications {
			target.Substitutions = plan.Substitutions
			target.CutLines = plan.CutLines
			if len(plan.Lines) == 0 {
				if verr := ValidateStatusTransition(string(prev), string(StatusCancelled), TransitionOpts{
					Actor:  "system:midnight_guard",
					Reason: "ALL_LINES_CUT_OUT_OF_STOCK",
				}); verr != nil {
					return "", "", true, false, nil, verr
				}
				decAt := now
				target.LineItems = nil
				target.TotalMinor = 0
				target.Status = StatusCancelled
				target.ConfirmationStatus = ConfirmationStatusRejected
				target.AutoConfirmAt = nil
				target.DecisionAt = &decAt
				target.DecisionBy = "system:midnight_guard"
				target.UpdatedAt = now
				target.TransitionReason = "ALL_LINES_CUT_OUT_OF_STOCK"
				target.TransitionActorRole = "SYSTEM"
				target.TransitionActorID = "system:midnight_guard"
				target.TransitionEventKind = "CANCEL_CUT_ALL"
				if txn != nil && s.credit != nil {
					if cerr := s.credit.AdjustReserveInTxn(ctx, txn, target.OrderID, 0); cerr != nil {
						return "", "", true, false, nil, fmt.Errorf("release credit reserve on cut-all preorder: %w", cerr)
					}
				}
				return prev, "ALL_LINES_CUT_OUT_OF_STOCK", true, alreadyReserved, origLines, nil
			}
			target.LineItems = plan.Lines
			newTotal := plan.UpdatedTotalMinor
			if newTotal > 0 && target.DeliveryFeeMinor > 0 {
				newTotal += target.DeliveryFeeMinor
			}
			target.TotalMinor = newTotal
			if txn != nil && s.credit != nil {
				if cerr := s.credit.AdjustReserveInTxn(ctx, txn, target.OrderID, newTotal); cerr != nil {
					return "", "", true, false, nil, fmt.Errorf("adjust credit reserve on preorder shortage: %w", cerr)
				}
			}
		}
	}

	if verr := ValidateStatusTransition(string(prev), string(StatusPending), TransitionOpts{}); verr != nil {
		return "", "", true, false, nil, verr
	}
	target.Status = StatusPending
	target.UpdatedAt = now
	target.TransitionReason = "PREORDER_PROMOTED"
	target.TransitionActorRole = "SYSTEM"
	target.TransitionActorID = "system:midnight_guard"
	target.TransitionEventKind = "PROMOTE"
	return prev, "PREORDER_PROMOTED", false, alreadyReserved, origLines, nil
}

func (s *Service) patchPreorderNotify(ctx context.Context, o Order, now time.Time, eventType string, patch func(*Order)) error {
	updated := o
	updated.UpdatedAt = now
	if patch != nil {
		patch(&updated)
	}
	return s.repo.UpdateOrder(ctx, updated, nil, func(txn outbox.TxnBuffer) error {
		return emitPreorderEvent(ctx, txn, eventType, updated, "SYSTEM", "system:midnight_guard")
	})
}

func (s *Service) listScheduledPreorders(ctx context.Context, limit int) ([]Order, error) {
	return s.listScheduledPreordersPage(ctx, limit, false, nil, "")
}

func (s *Service) listScheduledPreordersPage(ctx context.Context, limit int, hasCursor bool, afterDate *time.Time, afterOrderID string) ([]Order, error) {
	if s.spannerClient == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	stmt := buildScheduledPreordersPageStatement(limit, hasCursor, afterDate, afterOrderID)
	repo := &SpannerRepository{client: s.spannerClient}
	return repo.queryOrders(ctx, stmt)
}

func (s *Service) listPreorderPromotePage(ctx context.Context, cutoff time.Time, limit int, afterDate *time.Time, afterOrderID string) ([]Order, error) {
	if s.spannerClient == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	stmt := buildPreorderPromotePageStatement(cutoff, limit, afterDate, afterOrderID)
	repo := &SpannerRepository{client: s.spannerClient}
	return repo.queryOrders(ctx, stmt)
}

func buildScheduledPreordersPageStatement(limit int, hasCursor bool, afterDate *time.Time, afterOrderID string) spanner.Statement {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	params := map[string]any{"src": string(OrderSourceManualPreorder), "lim": limit}
	cursorClause := ""
	if hasCursor && afterOrderID != "" {
		if afterDate == nil {
			cursorClause = " AND ((RequestedDeliveryDate IS NULL AND OrderId > @afterOrderId) OR RequestedDeliveryDate IS NOT NULL)"
		} else {
			cursorClause = " AND ((RequestedDeliveryDate = @afterDate AND OrderId > @afterOrderId) OR RequestedDeliveryDate > @afterDate)"
			params["afterDate"] = *afterDate
		}
		params["afterOrderId"] = afterOrderID
	}
	return spanner.Statement{
		SQL: `SELECT ` + orderSelectColumns + ` FROM Orders
		      WHERE SupplierId IS NOT NULL
		        AND Status = 'SCHEDULED' AND OrderSource = @src` + cursorClause + `
		      ORDER BY RequestedDeliveryDate ASC, OrderId ASC
		      LIMIT @lim`,
		Params: params,
	}
}

func buildPreorderPromotePageStatement(cutoff time.Time, limit int, afterDate *time.Time, afterOrderID string) spanner.Statement {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	params := map[string]any{
		"src":    string(OrderSourceManualPreorder),
		"cutoff": cutoff,
		"lim":    limit,
	}
	cursorClause := ""
	if afterDate != nil && afterOrderID != "" {
		cursorClause = " AND (RequestedDeliveryDate > @afterDate OR (RequestedDeliveryDate = @afterDate AND OrderId > @afterOrderId))"
		params["afterDate"] = *afterDate
		params["afterOrderId"] = afterOrderID
	}
	return spanner.Statement{
		SQL: `SELECT ` + orderSelectColumns + ` FROM Orders
		      WHERE SupplierId IS NOT NULL
		        AND Status IN ('SCHEDULED', 'AUTO_ACCEPTED')
		        AND OrderSource = @src
		        AND RequestedDeliveryDate <= @cutoff` + cursorClause + `
		      ORDER BY RequestedDeliveryDate ASC, OrderId ASC
		      LIMIT @lim`,
		Params: params,
	}
}
