package order

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

// StartPreorderSweeper runs Midnight Guard + scheduled promoter loops.
func StartPreorderSweeper(svc *Service) {
	if svc == nil || svc.spannerClient == nil {
		return
	}
	svc.log.Info("preorder sweeper started")
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			ctx := context.Background()
			if err := svc.RunPreorderSweeperOnce(ctx); err != nil {
				svc.log.Warn("preorder sweeper tick failed", "err", err)
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
	today := proximity.TashkentTodayStart(now)
	orders, err := s.listScheduledPreorders(ctx, 200)
	if err != nil {
		return err
	}
	for _, o := range orders {
		if o.RequestedDeliveryDate == nil {
			continue
		}
		lead := PreorderLeadDays(now, o.RequestedDeliveryDate)
		phase := PreorderGuardPhase(lead)
		if phase == "" {
			continue
		}
		deliveryDay := proximity.TashkentTodayStart(*o.RequestedDeliveryDate)
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
	return nil
}

func (s *Service) sweepPreorderAutoAccept(ctx context.Context, now time.Time) error {
	today := proximity.TashkentTodayStart(now)
	orders, err := s.listScheduledPreorders(ctx, 200)
	if err != nil {
		return err
	}
	for _, o := range orders {
		if o.Status != StatusScheduled || o.RequestedDeliveryDate == nil {
			continue
		}
		lead := PreorderLeadDays(now, o.RequestedDeliveryDate)
		deliveryDay := proximity.TashkentTodayStart(*o.RequestedDeliveryDate)
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
		updated := o
		updated.Status = StatusAutoAccepted
		updated.ConfirmationStatus = ConfirmationStatusAutoConfirmed
		decisionAt := now
		updated.DecisionAt = &decisionAt
		updated.DecisionBy = "system:midnight_guard"
		updated.UpdatedAt = now
		if err := s.repo.UpdateOrder(ctx, updated, nil, func(txn outbox.TxnBuffer) error {
			return emitPreorderEvent(ctx, txn, events.EventPreOrderAutoAccepted, updated, "SYSTEM", "system:midnight_guard")
		}); err != nil {
			s.log.Warn("preorder auto-accept failed", "order_id", o.OrderID, "err", err)
			continue
		}
		s.afterOrderMutation(ctx, updated)
	}
	return nil
}

func (s *Service) sweepPreorderPromote(ctx context.Context, now time.Time) error {
	today := proximity.TashkentTodayStart(now)
	cutoff := today.AddDate(0, 0, 4)
	stmt := spanner.Statement{
		SQL: `SELECT ` + orderSelectColumns + ` FROM Orders
		      WHERE Status IN ('SCHEDULED', 'AUTO_ACCEPTED')
		        AND OrderSource = @src
		        AND RequestedDeliveryDate <= @cutoff
		      LIMIT 100`,
		Params: map[string]any{
			"src":    string(OrderSourceManualPreorder),
			"cutoff": cutoff,
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		o, err := scanOrderRowRow(row)
		if err != nil {
			continue
		}
		if err := s.promotePreorderToPending(ctx, o, now); err != nil {
			s.log.Warn("preorder promote failed", "order_id", o.OrderID, "err", err)
		}
	}
	return nil
}

func (s *Service) promotePreorderToPending(ctx context.Context, o Order, now time.Time) error {
	if o.Status != StatusScheduled && o.Status != StatusAutoAccepted {
		return nil
	}
	prev := o.Status
	o.Status = StatusPending
	o.UpdatedAt = now
	return s.repo.UpdateOrder(ctx, o, nil, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateOrder, o.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:          events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:            o.OrderID,
			SupplierID:         o.SupplierID,
			RetailerID:         o.RetailerID,
			WarehouseID:        o.WarehouseID,
			PreviousStatus:     string(prev),
			Status:             string(o.Status),
			Reason:             "PREORDER_PROMOTED",
			OrderSource:        string(o.Source),
			ConfirmationStatus: string(o.ConfirmationStatus),
		})
	})
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
	if s.spannerClient == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	stmt := spanner.Statement{
		SQL: `SELECT ` + orderSelectColumns + ` FROM Orders
		      WHERE Status = 'SCHEDULED' AND OrderSource = @src
		      ORDER BY RequestedDeliveryDate ASC LIMIT @lim`,
		Params: map[string]any{"src": string(OrderSourceManualPreorder), "lim": limit},
	}
	repo := &SpannerRepository{client: s.spannerClient}
	return repo.queryOrders(ctx, stmt)
}

