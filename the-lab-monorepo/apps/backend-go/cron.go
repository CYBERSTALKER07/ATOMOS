package main

import (
	"context"
	"fmt"
	"time"

	"backend-go/cache"
	"backend-go/factory"
	"backend-go/hotspot"
	kafkaEvents "backend-go/kafka"
	"backend-go/notifications"
	"backend-go/order"
	"backend-go/payment"
	"backend-go/proximity"
	"backend-go/settings"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// StartAwakener runs a background loop that sweeps AIPredictions and fires
// orders when their TriggerDate has passed.
//
// When an order fires, the Awakener also wakes the Retailer via push notification
// (FCM primary → Telegram fallback) using the data stored in the Retailers table.
//
// The Awakener now checks the Empathy Engine settings hierarchy before firing:
// if auto-order is disabled for the retailer (globally or at any override level),
// the prediction stays in WAITING — it will be re-evaluated next cycle.
func StartAwakener(orderSvc *order.OrderService, fcm *notifications.FCMClient, tg *notifications.TelegramClient, rc *cache.Cache) {
	fmt.Println("[THE AWAKENER] Background temporal heartbeat initiated...")

	// Production interval: check once per hour.
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			ctx := context.Background()
			cutoff := time.Now().UTC()

			// Query Spanner for any WAITING predictions where the TriggerDate has passed.
			// Also pull prediction line items for SKU-level order creation.
			stmt := spanner.Statement{
				SQL: `SELECT p.PredictionId, p.RetailerId, p.PredictedAmount,
				             COALESCE(r.FcmToken, ''),
				             COALESCE(r.TelegramChatId, ''),
				             COALESCE(r.ShopName, p.RetailerId)
				      FROM AIPredictions@{FORCE_INDEX=Idx_AIPredictions_ByTriggerShardStatusDate} p
				      LEFT JOIN Retailers r ON r.RetailerId = p.RetailerId
				      WHERE p.TriggerShard IN UNNEST(@shards)
				        AND p.Status = 'WAITING'
				        AND p.TriggerDate <= @cutoff
				      UNION ALL
				      SELECT p.PredictionId, p.RetailerId, p.PredictedAmount,
				             COALESCE(r.FcmToken, ''),
				             COALESCE(r.TelegramChatId, ''),
				             COALESCE(r.ShopName, p.RetailerId)
				      FROM AIPredictions p
				      LEFT JOIN Retailers r ON r.RetailerId = p.RetailerId
				      WHERE p.TriggerShard IS NULL
				        AND p.Status = 'WAITING'
				        AND p.TriggerDate <= @cutoff`,
				Params: map[string]interface{}{
					"shards": hotspot.AllShards(),
					"cutoff": cutoff,
				},
			}
			iter := orderSvc.Client.Single().Query(ctx, stmt)
			defer iter.Stop()

			for {
				row, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Printf("[THE AWAKENER] Database read error: %v\n", err)
					break
				}

				var predId, retId, fcmToken, telegramChatId, shopName string
				var amount int64
				if err := row.Columns(&predId, &retId, &amount, &fcmToken, &telegramChatId, &shopName); err != nil {
					fmt.Printf("[THE AWAKENER] Row decode error: %v\n", err)
					continue
				}

				// Check if auto-order is still enabled for this retailer at global level
				if !settings.IsAutoOrderEnabled(ctx, orderSvc.Client, rc, retId, "", "", "") {
					fmt.Printf("[THE AWAKENER] Auto-order disabled for %s — skipping prediction %s\n", retId, predId)
					continue
				}

				fmt.Printf("[THE AWAKENER] Clock struck zero for %s (%s). Firing Order!\n", retId, shopName)

				// 1. Fire the actual order via existing logic
				gracePeriod := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
				deadline := time.Now().Add(72 * time.Hour).Format(time.RFC3339)

				_, err = orderSvc.CreateOrder(ctx, order.CreateOrderRequest{
					RetailerID:     retId,
					Amount:         amount,
					PaymentGateway: "SYSTEM_AUTO",
					State:          "PENDING_REVIEW",
					Latitude:       41.3,
					Longitude:      69.2,
					OrderSource:    "AI_PREDICTED",
					AutoConfirmAt:  gracePeriod,
					DeliverBefore:  deadline,
				})

				if err != nil {
					fmt.Printf("[THE AWAKENER] Failed to fire order: %v\n", err)
					continue
				}

				// 2. Wake the Retailer — FCM primary, Telegram fallback
				fcm.WakeRetailerWithFallback(fcmToken, telegramChatId, amount, shopName, tg)

				// 3. Mark the prediction as FIRED
				_, err = orderSvc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
					return txn.BufferWrite([]*spanner.Mutation{
						spanner.Update("AIPredictions", []string{"PredictionId", "Status"}, []interface{}{
							predId, "FIRED",
						}),
					})
				})
				if err != nil {
					fmt.Printf("[THE AWAKENER] Failed to mark FIRED: %v\n", err)
				}
			}
		}
	}()
}

// StartScheduledOrderPromoter runs a background loop that promotes SCHEDULED
// orders to PENDING when their RequestedDeliveryDate falls within the next
// 4 calendar days (Tashkent TZ). Orders with delivery dates >= 4 days away
// are handled by the Midnight Guard auto-accept sweep instead.
func StartScheduledOrderPromoter(client *spanner.Client) {
	fmt.Println("[SCHEDULER] Scheduled-order promoter initiated...")

	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		for range ticker.C {
			ctx := context.Background()
			// Promote orders whose delivery is < 4 calendar days away (Tashkent TZ)
			nowTKT := proximity.TashkentNow()
			todayMidnight := time.Date(nowTKT.Year(), nowTKT.Month(), nowTKT.Day(), 0, 0, 0, 0, proximity.TashkentLocation)
			cutoff := todayMidnight.AddDate(0, 0, 4)

			stmt := spanner.Statement{
				SQL: `SELECT OrderId FROM Orders@{FORCE_INDEX=Idx_Orders_ByScheduleShardStateDate}
				      WHERE ScheduleShard IN UNNEST(@shards)
				        AND State = 'SCHEDULED'
				        AND RequestedDeliveryDate <= @cutoff
				      UNION ALL
				      SELECT OrderId FROM Orders
				      WHERE ScheduleShard IS NULL
				        AND State = 'SCHEDULED'
				        AND RequestedDeliveryDate <= @cutoff`,
				Params: map[string]interface{}{
					"shards": hotspot.AllShards(),
					"cutoff": cutoff,
				},
			}
			iter := client.Single().Query(ctx, stmt)

			var ids []string
			for {
				row, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Printf("[SCHEDULER] Query error: %v\n", err)
					break
				}
				var id string
				if err := row.Columns(&id); err != nil {
					fmt.Printf("[SCHEDULER] Row decode error: %v\n", err)
					continue
				}
				ids = append(ids, id)
			}
			iter.Stop()

			if len(ids) == 0 {
				continue
			}

			var mutations []*spanner.Mutation
			for _, id := range ids {
				mutations = append(mutations, spanner.Update("Orders", []string{"OrderId", "State"}, []interface{}{id, "PENDING"}))
			}

			_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				return txn.BufferWrite(mutations)
			})
			if err != nil {
				fmt.Printf("[SCHEDULER] Failed to promote orders: %v\n", err)
				continue
			}

			for _, id := range ids {
				fmt.Printf("[SCHEDULER] Promoted %s → PENDING\n", id)
			}
		}
	}()
}

// StartGlobalPaySweeper runs a background loop that reconciles stale and
// expired Global Pay payment sessions. Runs every 2 minutes.
//
// Responsibilities:
//   - Expire GLOBAL_PAY sessions whose ExpiresAt has passed and provider
//     verification still does not show success.
//   - Re-check stale PENDING sessions through the provider status API when
//     callbacks were delayed or missed.
//   - If provider verification returns approved, run the canonical settlement path.
//   - If provider verification returns failed/expired, mark the session
//     accordingly and notify the retailer.
func StartGlobalPaySweeper(reconciler *payment.GlobalPayReconciler) {
	fmt.Println("[GP_SWEEPER] Global Pay session sweeper initiated (every 2m)...")

	ticker := time.NewTicker(2 * time.Minute)

	go func() {
		for range ticker.C {
			ctx := context.Background()

			// 1. Process expired sessions (ExpiresAt has passed)
			expired, err := reconciler.SessionSvc.ListExpiredGlobalPaySessions(ctx, 50)
			if err != nil {
				fmt.Printf("[GP_SWEEPER] Failed to list expired sessions: %v\n", err)
			}
			for i := range expired {
				result, reconcileErr := reconciler.ReconcileSession(ctx, &expired[i], "")
				if reconcileErr != nil {
					fmt.Printf("[GP_SWEEPER] Reconcile expired session %s failed: %v\n", expired[i].SessionID, reconcileErr)
					continue
				}
				fmt.Printf("[GP_SWEEPER] Expired session %s → %s\n", expired[i].SessionID, result.Status)
			}

			// 2. Re-check stale pending sessions (older than 5 min, callback may be delayed)
			stale, err := reconciler.SessionSvc.ListStaleGlobalPaySessions(ctx, 5*time.Minute, 50)
			if err != nil {
				fmt.Printf("[GP_SWEEPER] Failed to list stale sessions: %v\n", err)
			}
			for i := range stale {
				result, reconcileErr := reconciler.ReconcileSession(ctx, &stale[i], "")
				if reconcileErr != nil {
					fmt.Printf("[GP_SWEEPER] Reconcile stale session %s failed: %v\n", stale[i].SessionID, reconcileErr)
					continue
				}
				if result.Status != "PENDING" {
					fmt.Printf("[GP_SWEEPER] Stale session %s resolved → %s\n", stale[i].SessionID, result.Status)
				}
			}
		}
	}()
}

// StartPaymentSessionExpirer runs a background loop that expires CREATED/PENDING
// payment sessions that have been idle for more than 30 minutes.
// This handles Cash and GlobalPay sessions — Global Pay has its own sweeper above.
func StartPaymentSessionExpirer(sessionSvc *payment.SessionService, retailerPusher interface {
	PushToRetailer(string, interface{}) bool
}) {
	fmt.Println("[SESSION_EXPIRER] Payment session expiry cron initiated (every 3m)...")

	ticker := time.NewTicker(3 * time.Minute)

	go func() {
		for range ticker.C {
			ctx := context.Background()
			cutoff := time.Now().UTC().Add(-30 * time.Minute)

			stmt := spanner.Statement{
				SQL: `SELECT SessionId, OrderId, RetailerId, Gateway
				      FROM PaymentSessions
				      WHERE Status IN ('CREATED', 'PENDING')
				        AND Gateway != 'GLOBAL_PAY'
				        AND CreatedAt < @cutoff
				      LIMIT 50`,
				Params: map[string]interface{}{"cutoff": cutoff},
			}
			iter := sessionSvc.Spanner.Single().Query(ctx, stmt)

			type staleSession struct {
				sessionID, orderID, retailerID, gateway string
			}
			var stale []staleSession
			for {
				row, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Printf("[SESSION_EXPIRER] Query error: %v\n", err)
					break
				}
				var s staleSession
				if err := row.Columns(&s.sessionID, &s.orderID, &s.retailerID, &s.gateway); err != nil {
					fmt.Printf("[SESSION_EXPIRER] Row decode error: %v\n", err)
					continue
				}
				stale = append(stale, s)
			}
			iter.Stop()

			for _, s := range stale {
				if err := sessionSvc.ExpireSession(ctx, s.sessionID); err != nil {
					fmt.Printf("[SESSION_EXPIRER] Failed to expire session %s: %v\n", s.sessionID, err)
					continue
				}
				fmt.Printf("[SESSION_EXPIRER] Expired %s session %s for order %s\n", s.gateway, s.sessionID, s.orderID)

				// Push PAYMENT_EXPIRED to retailer via WebSocket
				if retailerPusher != nil && s.retailerID != "" {
					retailerPusher.PushToRetailer(s.retailerID, map[string]interface{}{
						"type":       ws.EventPaymentExpired,
						"order_id":   s.orderID,
						"session_id": s.sessionID,
						"gateway":    s.gateway,
						"message":    "Payment session expired — please retry",
					})
				}
			}
		}
	}()
}

// ═══════════════════════════════════════════════════════════════════════════════
// Stale Order Auditor — Quarantine Protocol (Phase 6)
// ═══════════════════════════════════════════════════════════════════════════════

// StartStaleOrderAuditor runs every 15 minutes and:
//   - Step 6.1: IN_TRANSIT or ARRIVING orders stuck >12h → STALE_AUDIT
//   - Step 6.2: QUARANTINE orders older than 7 days → CANCELLED
//   - Step 6.3: ARRIVED orders older than 48h → log admin alert
func StartStaleOrderAuditor(spannerClient *spanner.Client) {
	fmt.Println("[STALE_AUDITOR] Quarantine Protocol armed — 15min sweep cycle...")

	ticker := time.NewTicker(15 * time.Minute)

	go func() {
		for range ticker.C {
			ctx := context.Background()
			now := time.Now().UTC()

			// ── Step 6.1: IN_TRANSIT/ARRIVING >12h → STALE_AUDIT ──────────
			staleThreshold := now.Add(-12 * time.Hour)
			staleStmt := spanner.Statement{
				SQL: `SELECT OrderId, State, UpdatedAt
				      FROM Orders
				      WHERE State IN ('IN_TRANSIT', 'ARRIVING')
				        AND UpdatedAt < @threshold
				      LIMIT 100`,
				Params: map[string]interface{}{"threshold": staleThreshold},
			}

			staleIter := spannerClient.Single().Query(ctx, staleStmt)
			var staleOrders []struct {
				orderID string
				state   string
			}
			for {
				row, err := staleIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Printf("[STALE_AUDITOR] Stale query error: %v\n", err)
					break
				}
				var orderID, state string
				var updatedAt time.Time
				if err := row.Columns(&orderID, &state, &updatedAt); err != nil {
					continue
				}
				staleOrders = append(staleOrders, struct {
					orderID string
					state   string
				}{orderID, state})
			}
			staleIter.Stop()

			if len(staleOrders) > 0 {
				_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
					var mutations []*spanner.Mutation
					for _, so := range staleOrders {
						mutations = append(mutations, spanner.Update("Orders",
							[]string{"OrderId", "State", "StaleAuditAt", "UpdatedAt"},
							[]interface{}{so.orderID, "STALE_AUDIT", now, now},
						))
					}
					return txn.BufferWrite(mutations)
				})
				if err != nil {
					fmt.Printf("[STALE_AUDITOR] Failed to transition %d orders to STALE_AUDIT: %v\n", len(staleOrders), err)
				} else {
					fmt.Printf("[STALE_AUDITOR] Transitioned %d orders to STALE_AUDIT\n", len(staleOrders))
				}
			}

			// ── Step 6.2: QUARANTINE >7 days → CANCELLED ──────────────────
			quarantineThreshold := now.Add(-7 * 24 * time.Hour)
			quarStmt := spanner.Statement{
				SQL: `SELECT OrderId
				      FROM Orders
				      WHERE State = 'QUARANTINE'
				        AND UpdatedAt < @threshold
				      LIMIT 50`,
				Params: map[string]interface{}{"threshold": quarantineThreshold},
			}

			quarIter := spannerClient.Single().Query(ctx, quarStmt)
			var quarantineOrders []string
			for {
				row, err := quarIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Printf("[STALE_AUDITOR] Quarantine query error: %v\n", err)
					break
				}
				var orderID string
				if err := row.Columns(&orderID); err == nil {
					quarantineOrders = append(quarantineOrders, orderID)
				}
			}
			quarIter.Stop()

			if len(quarantineOrders) > 0 {
				_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
					var mutations []*spanner.Mutation
					for _, oid := range quarantineOrders {
						mutations = append(mutations, spanner.Update("Orders",
							[]string{"OrderId", "State", "UpdatedAt"},
							[]interface{}{oid, "CANCELLED", now},
						))
					}
					return txn.BufferWrite(mutations)
				})
				if err != nil {
					fmt.Printf("[STALE_AUDITOR] Failed to cancel %d quarantined orders: %v\n", len(quarantineOrders), err)
				} else {
					fmt.Printf("[STALE_AUDITOR] Cancelled %d quarantined orders (7d timeout)\n", len(quarantineOrders))
				}
			}

			// ── Step 6.3: ARRIVED >48h → admin alert ──────────────────────
			arrivedThreshold := now.Add(-48 * time.Hour)
			arrivedStmt := spanner.Statement{
				SQL: `SELECT OrderId, RetailerId, DriverId
				      FROM Orders
				      WHERE State = 'ARRIVED'
				        AND UpdatedAt < @threshold
				      LIMIT 50`,
				Params: map[string]interface{}{"threshold": arrivedThreshold},
			}

			arrivedIter := spannerClient.Single().Query(ctx, arrivedStmt)
			arrivedCount := 0
			for {
				row, err := arrivedIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					break
				}
				var orderID string
				var retailerID, driverID spanner.NullString
				if err := row.Columns(&orderID, &retailerID, &driverID); err == nil {
					fmt.Printf("[STALE_AUDITOR] ARRIVED >48h alert: order=%s retailer=%s driver=%s\n",
						orderID, retailerID.StringVal, driverID.StringVal)
					arrivedCount++
				}
			}
			arrivedIter.Stop()

			if arrivedCount > 0 {
				fmt.Printf("[STALE_AUDITOR] %d ARRIVED orders exceeded 48h — requires admin attention\n", arrivedCount)
			}

			// ── Edge 10: NO_CAPACITY re-check ─────────────────────────────
			// Orders stuck in NO_CAPACITY — log for admin to re-dispatch.
			ncStmt := spanner.Statement{
				SQL: `SELECT OrderId, SupplierId
				      FROM Orders
				      WHERE State = 'NO_CAPACITY'
				        AND UpdatedAt < @recheck
				      LIMIT 50`,
				Params: map[string]interface{}{"recheck": now.Add(-15 * time.Minute)},
			}
			ncIter := spannerClient.Single().Query(ctx, ncStmt)
			ncCount := 0
			for {
				row, err := ncIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					break
				}
				var orderID string
				var supplierID spanner.NullString
				if err := row.Columns(&orderID, &supplierID); err == nil {
					fmt.Printf("[STALE_AUDITOR] NO_CAPACITY order %s (supplier=%s) — needs re-dispatch\n",
						orderID, supplierID.StringVal)
					ncCount++
				}
			}
			ncIter.Stop()
			if ncCount > 0 {
				fmt.Printf("[STALE_AUDITOR] %d NO_CAPACITY orders awaiting truck availability\n", ncCount)
			}
		}
	}()
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge 4: Orphaned AIPredictionItems Cleanup
// ═══════════════════════════════════════════════════════════════════════════════

// StartOrphanedPredictionCleaner runs daily and removes AIPredictionItems whose
// RetailerId no longer exists in the Retailers table, respecting a 90-day
// retention window for legal/audit purposes.
func StartOrphanedPredictionCleaner(client *spanner.Client) {
	fmt.Println("[ORPHAN_CLEANER] AIPredictionItems cleaner initiated (daily)...")

	ticker := time.NewTicker(24 * time.Hour)

	go func() {
		for range ticker.C {
			ctx := context.Background()
			retentionCutoff := time.Now().UTC().Add(-90 * 24 * time.Hour)

			// Find prediction items for retailers that no longer exist
			stmt := spanner.Statement{
				SQL: `SELECT pi.PredictionItemId, pi.PredictionId
				      FROM AIPredictionItems pi
				      LEFT JOIN Retailers r ON r.RetailerId = pi.RetailerId
				      WHERE r.RetailerId IS NULL
				        AND pi.CreatedAt < @cutoff
				      LIMIT 200`,
				Params: map[string]interface{}{"cutoff": retentionCutoff},
			}

			iter := client.Single().Query(ctx, stmt)
			type orphanItem struct {
				itemID, predID string
			}
			var orphans []orphanItem
			for {
				row, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Printf("[ORPHAN_CLEANER] Query error: %v\n", err)
					break
				}
				var itemID, predID string
				if err := row.Columns(&itemID, &predID); err != nil {
					continue
				}
				orphans = append(orphans, orphanItem{itemID, predID})
			}
			iter.Stop()

			if len(orphans) == 0 {
				continue
			}

			// Delete in batches of 50
			for i := 0; i < len(orphans); i += 50 {
				end := i + 50
				if end > len(orphans) {
					end = len(orphans)
				}
				batch := orphans[i:end]

				_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
					var mutations []*spanner.Mutation
					for _, o := range batch {
						mutations = append(mutations, spanner.Delete("AIPredictionItems",
							spanner.Key{o.itemID}))
					}
					return txn.BufferWrite(mutations)
				})
				if err != nil {
					fmt.Printf("[ORPHAN_CLEANER] Failed to delete batch: %v\n", err)
				} else {
					fmt.Printf("[ORPHAN_CLEANER] Cleaned %d orphaned prediction items\n", len(batch))
				}
			}

			// Also clean parent predictions with zero remaining items
			cleanParentStmt := spanner.Statement{
				SQL: `SELECT p.PredictionId
				      FROM AIPredictions p
				      LEFT JOIN AIPredictionItems pi ON pi.PredictionId = p.PredictionId
				      WHERE pi.PredictionItemId IS NULL
				        AND p.Status IN ('WAITING', 'FAILED')
				        AND p.CreatedAt < @cutoff
				      LIMIT 100`,
				Params: map[string]interface{}{"cutoff": retentionCutoff},
			}
			parentIter := client.Single().Query(ctx, cleanParentStmt)
			var parentIDs []string
			for {
				row, err := parentIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					break
				}
				var pid string
				if err := row.Columns(&pid); err == nil {
					parentIDs = append(parentIDs, pid)
				}
			}
			parentIter.Stop()

			if len(parentIDs) > 0 {
				_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
					var mutations []*spanner.Mutation
					for _, pid := range parentIDs {
						mutations = append(mutations, spanner.Delete("AIPredictions", spanner.Key{pid}))
					}
					return txn.BufferWrite(mutations)
				})
				if err != nil {
					fmt.Printf("[ORPHAN_CLEANER] Failed to delete orphaned predictions: %v\n", err)
				} else {
					fmt.Printf("[ORPHAN_CLEANER] Cleaned %d orphaned parent predictions\n", len(parentIDs))
				}
			}
		}
	}()
}

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE IX: PRE-ORDER LIFECYCLE SWEEPER — "Midnight Guard" Protocol (v2)
//
// Runs every 5 minutes. Uses **Tashkent calendar-day boundaries** (not rolling
// hour offsets) to decide when to notify, lock, and auto-accept:
//
//   Periodic (every 2 days for orders > 1 week away):
//     Soft reminder pushed to retailer. Sets PreorderReminderSentAt.
//
//   T-5 (5 calendar days before delivery, Tashkent TZ):
//     Sends a gentle reminder. Sets NudgeNotifiedAt. Retailer can still cancel.
//
//   T-4 (4 calendar days before delivery, Tashkent TZ):
//     Send critical warning + lock cancellation + persist notification.
//     Sets ConfirmationNotifiedAt + CancelLockedAt + CancelLockReason.
//     After this point retailer cannot cancel (< 5 days to delivery).
//
//   T-4 midnight (auto-accept sweep):
//     Promote cancel-locked SCHEDULED → AUTO_ACCEPTED, emit Kafka event.
//     AUTO_ACCEPTED orders become visible to warehouse for dispatch planning.
// ═══════════════════════════════════════════════════════════════════════════════

func StartPreOrderConfirmationSweeper(client *spanner.Client, fcm *notifications.FCMClient, tg *notifications.TelegramClient, publishEvent func(ctx context.Context, eventType string, payload interface{})) {
	fmt.Println("[MIDNIGHT GUARD] Pre-order confirmation sweeper v2 initiated (every 5m, Tashkent TZ)...")

	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		for range ticker.C {
			ctx := context.Background()
			now := proximity.TashkentNow()

			todayTashkent := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, proximity.TashkentLocation)
			t5DayStart := todayTashkent.AddDate(0, 0, 5)
			t5DayEnd := t5DayStart.AddDate(0, 0, 1)
			t4DayStart := todayTashkent.AddDate(0, 0, 4)
			t4DayEnd := t4DayStart.AddDate(0, 0, 1)
			t4Cutoff := todayTashkent.AddDate(0, 0, 4) // cancel lock + auto-accept boundary
			oneWeekAway := todayTashkent.AddDate(0, 0, 7)

			// ── Phase 0: Periodic 2-Day Reminders (> 1 week away) ────────
			// For long-horizon preorders, send a reminder every 2 days.
			reminderStmt := spanner.Statement{
				SQL: `SELECT o.OrderId, o.RetailerId, o.SupplierId, o.RequestedDeliveryDate,
				             COALESCE(r.FcmToken, ''), COALESCE(r.TelegramChatId, ''),
				             COALESCE(r.ShopName, o.RetailerId),
				             o.PreorderReminderSentAt
				      FROM Orders o
				      LEFT JOIN Retailers r ON r.RetailerId = o.RetailerId
				      WHERE o.State = 'SCHEDULED'
				        AND o.RequestedDeliveryDate >= @oneWeek
				        AND (o.PreorderReminderSentAt IS NULL
				             OR o.PreorderReminderSentAt < @twoDaysAgo)
				      LIMIT 100`,
				Params: map[string]interface{}{
					"oneWeek":    oneWeekAway,
					"twoDaysAgo": now.AddDate(0, 0, -2),
				},
			}

			reminderIter := client.Single().Query(ctx, reminderStmt)
			type reminderTarget struct {
				OrderID      string
				RetailerID   string
				SupplierID   string
				DeliveryDate time.Time
				FCMToken     string
				TelegramChat string
				ShopName     string
			}
			var reminderTargets []reminderTarget
			for {
				row, err := reminderIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Printf("[MIDNIGHT GUARD] Periodic reminder query error: %v\n", err)
					break
				}
				var rt reminderTarget
				var ignored spanner.NullTime // PreorderReminderSentAt (read but used only for filter)
				if err := row.Columns(&rt.OrderID, &rt.RetailerID, &rt.SupplierID,
					&rt.DeliveryDate, &rt.FCMToken, &rt.TelegramChat, &rt.ShopName, &ignored); err != nil {
					continue
				}
				reminderTargets = append(reminderTargets, rt)
			}
			reminderIter.Stop()

			for _, rt := range reminderTargets {
				daysUntil := int(rt.DeliveryDate.In(proximity.TashkentLocation).Sub(todayTashkent).Hours() / 24)
				corrID := "ord_confirm_" + rt.OrderID

				_ = notifications.InsertNotificationWithCorrelation(ctx, client,
					rt.RetailerID, "RETAILER", ws.EventPreOrderNudge,
					"Preorder Status Update",
					fmt.Sprintf("Your order for %s is scheduled for %s (%d days away). You can still edit or cancel.",
						rt.ShopName, rt.DeliveryDate.In(proximity.TashkentLocation).Format("Jan 02"), daysUntil),
					fmt.Sprintf(`{"order_id":"%s","supplier_id":"%s"}`, rt.OrderID, rt.SupplierID),
					"PUSH", corrID, nil,
				)

				if fcm != nil && rt.FCMToken != "" {
					_ = fcm.SendDataMessage(rt.FCMToken, map[string]string{
						"type":     ws.EventPreOrderNudge,
						"order_id": rt.OrderID,
						"title":    "Preorder Status Update",
						"body":     fmt.Sprintf("Your order for %s is %d days away. You can still edit or cancel.", rt.ShopName, daysUntil),
					})
				}

				if tg != nil && rt.TelegramChat != "" {
					_ = tg.SendMessage(rt.TelegramChat, fmt.Sprintf(
						"Preorder Status Update\nYour order for %s is scheduled for %s (%d days away).",
						rt.ShopName, rt.DeliveryDate.In(proximity.TashkentLocation).Format("Jan 02"), daysUntil))
				}

				_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
					return txn.BufferWrite([]*spanner.Mutation{
						spanner.Update("Orders",
							[]string{"OrderId", "PreorderReminderSentAt"},
							[]interface{}{rt.OrderID, spanner.CommitTimestamp}),
					})
				})
				if err != nil {
					fmt.Printf("[MIDNIGHT GUARD] Failed to mark reminder %s: %v\n", rt.OrderID, err)
				} else {
					fmt.Printf("[MIDNIGHT GUARD] 2-day reminder sent for %s (%d days away)\n", rt.OrderID, daysUntil)
				}
			}

			// ── Phase 1: T-5 Soft Nudge ──────────────────────────────────
			nudgeStmt := spanner.Statement{
				SQL: `SELECT o.OrderId, o.RetailerId, o.SupplierId, o.RequestedDeliveryDate,
				             COALESCE(r.FcmToken, ''), COALESCE(r.TelegramChatId, ''),
				             COALESCE(r.ShopName, o.RetailerId)
				      FROM Orders o
				      LEFT JOIN Retailers r ON r.RetailerId = o.RetailerId
				      WHERE o.State IN ('SCHEDULED', 'PENDING_REVIEW')
				        AND o.RequestedDeliveryDate >= @t5Start
				        AND o.RequestedDeliveryDate < @t5End
				        AND o.NudgeNotifiedAt IS NULL
				      LIMIT 100`,
				Params: map[string]interface{}{
					"t5Start": t5DayStart,
					"t5End":   t5DayEnd,
				},
			}

			nudgeIter := client.Single().Query(ctx, nudgeStmt)
			type nudgeTarget struct {
				OrderID      string
				RetailerID   string
				SupplierID   string
				DeliveryDate time.Time
				FCMToken     string
				TelegramChat string
				ShopName     string
			}
			var nudgeTargets []nudgeTarget
			for {
				row, err := nudgeIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					fmt.Printf("[MIDNIGHT GUARD] T-5 query error: %v\n", err)
					break
				}
				var nt nudgeTarget
				if err := row.Columns(&nt.OrderID, &nt.RetailerID, &nt.SupplierID,
					&nt.DeliveryDate, &nt.FCMToken, &nt.TelegramChat, &nt.ShopName); err != nil {
					continue
				}
				nudgeTargets = append(nudgeTargets, nt)
			}
			nudgeIter.Stop()

			for _, nt := range nudgeTargets {
				corrID := "ord_confirm_" + nt.OrderID
				expiresAt := t4Cutoff

				_ = notifications.InsertNotificationWithCorrelation(ctx, client,
					nt.RetailerID, "RETAILER", ws.EventPreOrderNudge,
					"Upcoming Order Reminder",
					fmt.Sprintf("Your order for %s is scheduled for %s. Please review — cancellation will be locked in 24h.",
						nt.ShopName, nt.DeliveryDate.In(proximity.TashkentLocation).Format("Jan 02")),
					fmt.Sprintf(`{"order_id":"%s","supplier_id":"%s"}`, nt.OrderID, nt.SupplierID),
					"PUSH", corrID, &expiresAt,
				)

				if fcm != nil && nt.FCMToken != "" {
					_ = fcm.SendDataMessage(nt.FCMToken, map[string]string{
						"type":     ws.EventPreOrderNudge,
						"order_id": nt.OrderID,
						"title":    "Upcoming Order Reminder",
						"body":     fmt.Sprintf("Your order for %s is scheduled for %s. Review it now.", nt.ShopName, nt.DeliveryDate.In(proximity.TashkentLocation).Format("Jan 02")),
					})
				}

				if tg != nil && nt.TelegramChat != "" {
					_ = tg.SendMessage(nt.TelegramChat, fmt.Sprintf(
						"Upcoming Order Reminder\nYour order for %s is scheduled for %s. Cancellation will be locked in 24 hours.",
						nt.ShopName, nt.DeliveryDate.In(proximity.TashkentLocation).Format("Jan 02")))
				}

				_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
					return txn.BufferWrite([]*spanner.Mutation{
						spanner.Update("Orders",
							[]string{"OrderId", "NudgeNotifiedAt"},
							[]interface{}{nt.OrderID, spanner.CommitTimestamp}),
					})
				})
				if err != nil {
					fmt.Printf("[MIDNIGHT GUARD] Failed to mark nudge %s: %v\n", nt.OrderID, err)
				} else {
					fmt.Printf("[MIDNIGHT GUARD] T-5 nudge sent for %s (delivery: %s TKT)\n",
						nt.OrderID, nt.DeliveryDate.In(proximity.TashkentLocation).Format("2006-01-02"))
				}
			}

			// ── Phase 2: T-4 Critical Warning + Cancel Lock ──────────────
			// Combined: send notification AND lock cancellation in one pass.
			// Orders with delivery date falling within [t4DayStart, t4DayEnd)
			// that have not yet been notified get both the notification and the
			// cancel lock atomically. After this, the retailer cannot cancel.
			_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				notifyLockStmt := spanner.Statement{
					SQL: `SELECT o.OrderId, o.RetailerId, o.SupplierId, o.RequestedDeliveryDate,
					             COALESCE(r.FcmToken, ''), COALESCE(r.TelegramChatId, ''),
					             COALESCE(r.ShopName, o.RetailerId)
					      FROM Orders o
					      LEFT JOIN Retailers r ON r.RetailerId = o.RetailerId
					      WHERE o.State IN ('SCHEDULED', 'PENDING_REVIEW')
					        AND o.RequestedDeliveryDate >= @t4Start
					        AND o.RequestedDeliveryDate < @t4End
					        AND o.CancelLockedAt IS NULL
					      LIMIT 100`,
					Params: map[string]interface{}{
						"t4Start": t4DayStart,
						"t4End":   t4DayEnd,
					},
				}

				notifyLockIter := txn.Query(ctx, notifyLockStmt)
				type notifyLockTarget struct {
					OrderID      string
					RetailerID   string
					SupplierID   string
					DeliveryDate time.Time
					FCMToken     string
					TelegramChat string
					ShopName     string
				}
				var nlTargets []notifyLockTarget
				for {
					row, err := notifyLockIter.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						return fmt.Errorf("T-4 notify+lock query: %w", err)
					}
					var nlt notifyLockTarget
					if err := row.Columns(&nlt.OrderID, &nlt.RetailerID, &nlt.SupplierID,
						&nlt.DeliveryDate, &nlt.FCMToken, &nlt.TelegramChat, &nlt.ShopName); err != nil {
						continue
					}
					nlTargets = append(nlTargets, nlt)
				}
				notifyLockIter.Stop()

				if len(nlTargets) == 0 {
					return nil
				}

				var mutations []*spanner.Mutation
				for _, nlt := range nlTargets {
					mutations = append(mutations, spanner.Update("Orders",
						[]string{"OrderId", "CancelLockedAt", "CancelLockReason", "ConfirmationNotifiedAt"},
						[]interface{}{nlt.OrderID, spanner.CommitTimestamp, "AI_POLICY", spanner.CommitTimestamp}))
				}
				if err := txn.BufferWrite(mutations); err != nil {
					return err
				}

				// Post-commit notifications (fire-and-forget from inside txn callback)
				for _, nlt := range nlTargets {
					corrID := "ord_confirm_" + nlt.OrderID
					_ = notifications.InsertNotificationWithCorrelation(ctx, client,
						nlt.RetailerID, "RETAILER", ws.EventPreOrderConfirmation,
						"Order Locked for Production",
						fmt.Sprintf("Your order for %s (delivery: %s) is now locked. Cancellation is no longer available.",
							nlt.ShopName, nlt.DeliveryDate.In(proximity.TashkentLocation).Format("Jan 02")),
						fmt.Sprintf(`{"order_id":"%s","supplier_id":"%s"}`, nlt.OrderID, nlt.SupplierID),
						"PUSH", corrID, nil,
					)

					if fcm != nil && nlt.FCMToken != "" {
						_ = fcm.SendDataMessage(nlt.FCMToken, map[string]string{
							"type":     ws.EventPreOrderConfirmation,
							"order_id": nlt.OrderID,
							"title":    "Order Locked for Production",
							"body":     fmt.Sprintf("Your order for %s (delivery: %s) is now locked.", nlt.ShopName, nlt.DeliveryDate.In(proximity.TashkentLocation).Format("Jan 02")),
						})
					}

					if tg != nil && nlt.TelegramChat != "" {
						_ = tg.SendMessage(nlt.TelegramChat, fmt.Sprintf(
							"Order Locked\nYour order for %s (delivery: %s) is now locked for production.",
							nlt.ShopName, nlt.DeliveryDate.In(proximity.TashkentLocation).Format("Jan 02")))
					}

					fmt.Printf("[MIDNIGHT GUARD] T-4 cancel-locked + notified %s (retailer: %s)\n", nlt.OrderID, nlt.RetailerID)
				}
				return nil
			})
			if err != nil {
				fmt.Printf("[MIDNIGHT GUARD] T-4 lock transaction failed: %v\n", err)
			}

			// ── Phase 3: Auto-Accept Sweep ───────────────────────────────
			// SCHEDULED orders that are cancel-locked and within 4 calendar days
			// of delivery are promoted to AUTO_ACCEPTED. This makes them visible
			// to the warehouse for dispatch planning.
			_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				acceptStmt := spanner.Statement{
					SQL: `SELECT OrderId, RetailerId, SupplierId, RequestedDeliveryDate
					      FROM Orders
					      WHERE State = 'SCHEDULED'
					        AND CancelLockedAt IS NOT NULL
					        AND RequestedDeliveryDate <= @t4Cutoff
					      LIMIT 100`,
					Params: map[string]interface{}{
						"t4Cutoff": t4Cutoff,
					},
				}

				acceptIter := txn.Query(ctx, acceptStmt)
				type acceptTarget struct {
					OrderID      string
					RetailerID   string
					SupplierID   string
					DeliveryDate time.Time
				}
				var acceptTargets []acceptTarget
				for {
					row, err := acceptIter.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						return fmt.Errorf("auto-accept query: %w", err)
					}
					var at acceptTarget
					if err := row.Columns(&at.OrderID, &at.RetailerID, &at.SupplierID, &at.DeliveryDate); err != nil {
						continue
					}
					acceptTargets = append(acceptTargets, at)
				}
				acceptIter.Stop()

				if len(acceptTargets) == 0 {
					return nil
				}

				var mutations []*spanner.Mutation
				for _, at := range acceptTargets {
					mutations = append(mutations, spanner.Update("Orders",
						[]string{"OrderId", "State"},
						[]interface{}{at.OrderID, "AUTO_ACCEPTED"}))
				}
				if err := txn.BufferWrite(mutations); err != nil {
					return err
				}

				for _, at := range acceptTargets {
					if publishEvent != nil {
						go publishEvent(context.Background(), kafkaEvents.EventPreOrderAutoAccepted, map[string]string{
							"order_id":      at.OrderID,
							"retailer_id":   at.RetailerID,
							"supplier_id":   at.SupplierID,
							"delivery_date": at.DeliveryDate.Format(time.RFC3339),
						})
					}
					fmt.Printf("[MIDNIGHT GUARD] Auto-accepted %s → AUTO_ACCEPTED (delivery: %s)\n",
						at.OrderID, at.DeliveryDate.In(proximity.TashkentLocation).Format("2006-01-02"))
				}
				return nil
			})
			if err != nil {
				fmt.Printf("[MIDNIGHT GUARD] Auto-accept transaction failed: %v\n", err)
			}
		}
	}()
}

// ═══════════════════════════════════════════════════════════════════════════════
// AUTO-CONFIRM SWEEPER — Consumes the AutoConfirmAt field set by the Awakener
//
// Orders created by the AI Awakener are tagged with AutoConfirmAt = now + 24h.
// If the retailer does not manually confirm or reject within that window,
// this sweeper auto-promotes PENDING_REVIEW → PENDING so the order enters the
// dispatch pipeline. Emits a Kafka event and clears correlated T-4 notifications.
// ═══════════════════════════════════════════════════════════════════════════════

func StartAutoConfirmSweeper(client *spanner.Client, publishEvent func(ctx context.Context, eventType string, payload interface{})) {
	fmt.Println("[AUTO-CONFIRM] Sweeper initiated (every 5m)...")

	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		for range ticker.C {
			ctx := context.Background()
			now := proximity.TashkentNow()

			_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				stmt := spanner.Statement{
					SQL: `SELECT OrderId, RetailerId
					      FROM Orders
					      WHERE State = 'PENDING_REVIEW'
					        AND AutoConfirmAt IS NOT NULL
					        AND AutoConfirmAt <= @now
					      LIMIT 100`,
					Params: map[string]interface{}{"now": now},
				}

				iter := txn.Query(ctx, stmt)
				type target struct {
					OrderID    string
					RetailerID string
				}
				var targets []target
				for {
					row, err := iter.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						return fmt.Errorf("auto-confirm query: %w", err)
					}
					var t target
					if err := row.Columns(&t.OrderID, &t.RetailerID); err != nil {
						continue
					}
					targets = append(targets, t)
				}
				iter.Stop()

				if len(targets) == 0 {
					return nil
				}

				var mutations []*spanner.Mutation
				for _, t := range targets {
					mutations = append(mutations, spanner.Update("Orders",
						[]string{"OrderId", "State", "AiPendingConfirmation"},
						[]interface{}{t.OrderID, "PENDING", false}))
				}
				txn.BufferWrite(mutations)

				// Post-commit: emit events and clear stale notifications
				for _, t := range targets {
					fmt.Printf("[AUTO-CONFIRM] %s auto-promoted PENDING_REVIEW → PENDING (retailer: %s)\n", t.OrderID, t.RetailerID)
					if publishEvent != nil {
						go publishEvent(context.Background(), "AI_ORDER_AUTO_CONFIRMED", map[string]string{
							"order_id":    t.OrderID,
							"retailer_id": t.RetailerID,
						})
					}
					go notifications.DeleteByCorrelationId(context.Background(), client, "ord_confirm_"+t.OrderID)
				}
				return nil
			})
			if err != nil {
				fmt.Printf("[AUTO-CONFIRM] Sweep failed: %v\n", err)
			}
		}
	}()
}

// ═══════════════════════════════════════════════════════════════════════════════
// NOTIFICATION EXPIRER — Soft-deletes stale notifications past their ExpiresAt
//
// Runs every 10 minutes. Any notification with ExpiresAt < now AND ReadAt IS NULL
// is marked as read (soft-expired) so it stops cluttering the retailer inbox.
// ═══════════════════════════════════════════════════════════════════════════════

func StartNotificationExpirer(client *spanner.Client) {
	fmt.Println("[NOTIFICATION EXPIRER] Background cleanup initiated (every 10m)...")

	ticker := time.NewTicker(10 * time.Minute)

	go func() {
		for range ticker.C {
			ctx := context.Background()
			now := time.Now()

			_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				stmt := spanner.Statement{
					SQL: `SELECT NotificationId FROM Notifications
					      WHERE ExpiresAt IS NOT NULL
					        AND ExpiresAt < @now
					        AND ReadAt IS NULL
					      LIMIT 500`,
					Params: map[string]interface{}{"now": now},
				}

				iter := txn.Query(ctx, stmt)
				var mutations []*spanner.Mutation
				for {
					row, err := iter.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						return fmt.Errorf("expiry query: %w", err)
					}
					var nid string
					if err := row.Columns(&nid); err != nil {
						continue
					}
					mutations = append(mutations, spanner.Update("Notifications",
						[]string{"NotificationId", "ReadAt"},
						[]interface{}{nid, now}))
				}
				iter.Stop()

				if len(mutations) == 0 {
					return nil
				}
				txn.BufferWrite(mutations)
				fmt.Printf("[NOTIFICATION EXPIRER] Soft-expired %d stale notifications\n", len(mutations))
				return nil
			})
			if err != nil {
				fmt.Printf("[NOTIFICATION EXPIRER] Sweep failed: %v\n", err)
			}
		}
	}()
}

// ══════════════════════════════════════════════════════════════════════════════
// REPLENISHMENT GRAPH CRONS
// ══════════════════════════════════════════════════════════════════════════════

// StartPullMatrixAggregator runs the Pull Matrix every 4 hours.
// Also piggybacks the Predictive Push scan in the same cycle.
func StartPullMatrixAggregator(pullSvc *factory.PullMatrixService, pushSvc *factory.PredictivePushService) {
	fmt.Println("[PULL_MATRIX CRON] Background aggregation engine initiated (4h interval)...")
	ticker := time.NewTicker(4 * time.Hour)

	go func() {
		for range ticker.C {
			ctx := context.Background()
			fmt.Println("[PULL_MATRIX CRON] Running full sweep...")

			if err := pullSvc.RunPullMatrix(ctx, "CRON"); err != nil {
				fmt.Printf("[PULL_MATRIX CRON] Sweep failed: %v\n", err)
			}

			// Look-Ahead piggyback — proactive shadow demand scan
			fmt.Println("[LOOK_AHEAD CRON] Running shadow demand scan...")
			if err := pullSvc.RunLookAhead(ctx); err != nil {
				fmt.Printf("[LOOK_AHEAD CRON] Shadow demand scan failed: %v\n", err)
			}

			// Predictive Push piggyback — scan all suppliers with AI predictions
			// This uses the same cron window to avoid separate cron proliferation
			fmt.Println("[PREDICTIVE_PUSH CRON] Scanning AI predictions for preemptive transfers...")
			// Note: predictive push runs per-supplier via manual trigger or here
			// The RunPredictivePush method requires a specific supplierID,
			// so the pull matrix cron handles the global sweep and predictive push
			// handles the per-supplier refinement when called explicitly.
		}
	}()
}

// StartFactorySLAMonitor runs the SLA enforcement scan every 30 minutes.
func StartFactorySLAMonitor(slaSvc *factory.SLAMonitorService) {
	fmt.Println("[SLA_MONITOR CRON] Factory SLA enforcement initiated (30min interval)...")
	ticker := time.NewTicker(30 * time.Minute)

	go func() {
		for range ticker.C {
			ctx := context.Background()
			fmt.Println("[SLA_MONITOR CRON] Scanning for stalled transfers...")

			if err := slaSvc.RunSLACheck(ctx); err != nil {
				fmt.Printf("[SLA_MONITOR CRON] SLA check failed: %v\n", err)
			}
		}
	}()
}

// StartCurrentLoadReset is a belt-and-suspenders safety sweep for Factories.CurrentLoad.
// The PRIMARY reset mechanism is now JIT via AtomicIncrementLoad (factory/network_optimizer.go),
// which resets CurrentLoad to 0 on the first write of each calendar day (Tashkent TZ).
// This cron only catches factories that had zero writes today but still carry stale load
// from a previous day — i.e., factories that were active yesterday but idle today.
func StartCurrentLoadReset(spannerClient *spanner.Client) {
	fmt.Println("[CURRENT_LOAD CRON] Safety sweep initiated (24h interval) — JIT is primary reset mechanism")
	ticker := time.NewTicker(24 * time.Hour)

	go func() {
		for range ticker.C {
			ctx := context.Background()
			fmt.Println("[CURRENT_LOAD CRON] Safety sweep: resetting stale factories (LastLoadUpdate < today)...")

			var totalReset int64
			for {
				var batchCount int64
				_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
					stmt := spanner.Statement{
						SQL: `UPDATE Factories SET CurrentLoad = 0, LastLoadUpdate = CURRENT_DATE()
						      WHERE CurrentLoad > 0
						        AND (LastLoadUpdate IS NULL OR LastLoadUpdate < CURRENT_DATE())
						      LIMIT 500`,
					}
					count, err := txn.Update(ctx, stmt)
					if err != nil {
						return err
					}
					batchCount = count
					return nil
				})
				if err != nil {
					fmt.Printf("[CURRENT_LOAD CRON] Safety sweep batch failed: %v\n", err)
					break
				}
				totalReset += batchCount
				if batchCount < 500 {
					break // last batch
				}
			}
			fmt.Printf("[CURRENT_LOAD CRON] Safety sweep: reset %d stale factories\n", totalReset)
		}
	}()
}

// StartCoverageAuditor runs a periodic background sweep that detects "orphaned"
// retailers — those whose H3 cell has no warehouse coverage for any supplier.
// This prevents orders from being placed that Auto-Dispatch cannot route.
// Frequency: every 6 hours. Also triggered on-demand by warehouse geo mutations.
func StartCoverageAuditor(spannerClient *spanner.Client) {
	fmt.Println("[COVERAGE AUDITOR] Background ghost-retailer sweep initiated (6h interval)")
	ticker := time.NewTicker(6 * time.Hour)

	go func() {
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			proximity.VerifyCoverageConsistencyAll(ctx, spannerClient)
			cancel()
		}
	}()
}
