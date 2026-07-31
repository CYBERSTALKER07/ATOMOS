package order

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// RunShopClosedWorker continuously polls for orders whose grace period has ended
// without a resolution, and applies the configured timeout decision.
func (s *Service) RunShopClosedWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.InfoContext(ctx, "shop closed worker shutting down")
			return
		case <-ticker.C:
			if err := s.processShopClosedTimeouts(ctx); err != nil {
				s.log.ErrorContext(ctx, "shop closed worker error", "err", err)
			}
		}
	}
}

func (s *Service) processShopClosedTimeouts(ctx context.Context) error {
	now := s.now().UTC()

	// Find orders that are still SHOP_CLOSED_PENDING and grace period has passed.
	stmt := spanner.Statement{
		SQL: `
			SELECT OrderId 
			FROM Orders 
			WHERE Status = 'SHOP_CLOSED_PENDING' 
			  AND ShopClosedGraceEndsAt IS NOT NULL 
			  AND ShopClosedGraceEndsAt <= @now
			LIMIT 50
		`,
		Params: map[string]any{"now": now},
	}

	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	var expired []string

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
		var oid string
		if err := row.Columns(&oid); err != nil {
			s.log.ErrorContext(ctx, "failed to parse expired order row", "err", err)
			continue
		}
		expired = append(expired, oid)
	}

	for _, oid := range expired {
		if err := s.resolveOneShopClosedTimeout(ctx, oid); err != nil {
			s.log.ErrorContext(ctx, "failed to resolve shop closed timeout", "order_id", oid, "err", err)
		}
	}

	return nil
}

func (s *Service) resolveOneShopClosedTimeout(ctx context.Context, orderID string) error {
	now := s.now().UTC()

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		order, err := s.loadOrderForUpdate(ctx, txn, orderID)
		if err != nil {
			return err
		}
		if order.Status != StatusShopClosedPending {
			return nil // already resolved by retailer or another worker
		}
		if order.ShopClosedGraceEndsAt == nil || order.ShopClosedGraceEndsAt.After(now) {
			return nil // not yet due
		}

		// fetch credit profile for update
		profile, err := s.getProfileForUpdate(ctx, txn, order.RetailerID, order.SupplierID)
		if err != nil {
			s.log.WarnContext(ctx, "failed to load credit profile for timeout resolution", "err", err, "retailer_id", order.RetailerID)
		}

		// For now hardcode cfg, this could come from Supplier or global config
		cfg := TimeoutConfig{
			MaxAutoCreditMinor:            50000000, // example: 500,000 max
			MaxRiskTierForAutoCredit:      2,        // LOW/MEDIUM
			AllowForceBypass:              false,
			CreditScoreEnforcementEnabled: s.creditScoreEnforcement,
		}

		score, err := getRetailerCreditScore(ctx, txn, order.RetailerID)
		if err != nil {
			s.log.WarnContext(ctx, "failed to load credit score for timeout resolution", "err", err, "retailer_id", order.RetailerID)
		}

		decision := DecideShopClosedTimeout(order, profile, score, cfg, cfg.CreditScoreEnforcementEnabled)

		buf := &spannerTxnBuffer{}
		var mutations []*spanner.Mutation

		// Emit timeout event
		_ = outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventShopClosedTimeout, Timestamp: now.Format(time.RFC3339Nano)},
			OrderID:    order.OrderID,
			SupplierID: order.SupplierID,
			RetailerID: order.RetailerID,
			Resolution: string(decision),
		})

		switch decision {
		case DecisionCreditLeave:
			// same credit-draw logic as driver CreditLeave endpoint
			// mark order as delivering on credit
			mutations = append(mutations, spanner.UpdateMap("Orders", map[string]any{
				"OrderId":              order.OrderID,
				"Status":               string(StatusDeliveredOnCredit),
				"ShopClosedResolution": ShopClosedResolutionCreditLeave,
				"Version":              order.Version + 1,
				"UpdatedAt":            now,
			}))
			// Deduct credit
			if profile != nil {
				newBalance := profile.CurrentBalanceMinor + order.TotalMinor
				mutations = append(mutations, spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
					"RetailerId":           profile.RetailerID,
					"SupplierId":           profile.SupplierID,
					"CurrentBalanceMinor":  newBalance,
					"AvailableCreditMinor": max(0, profile.CreditLimitMinor-newBalance),
					"Version":              profile.Version + 1,
					"UpdatedAt":            spanner.CommitTimestamp,
				}))
			}
			_ = outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
				BaseEvent:  events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: now.Format(time.RFC3339Nano)},
				OrderID:    order.OrderID,
				SupplierID: order.SupplierID,
				Status:     string(StatusDeliveredOnCredit),
			})

		case DecisionReturnToWarehouse:
			// status -> CANCELLED + release reserved inventory in the same txn
			if err := releaseOrderReservationsInTxn(ctx, txn, order); err != nil {
				return err
			}
			mutations = append(mutations, spanner.UpdateMap("Orders", map[string]any{
				"OrderId":              order.OrderID,
				"Status":               string(StatusCancelled),
				"ShopClosedResolution": ShopClosedResolutionReturned,
				"Version":              order.Version + 1,
				"UpdatedAt":            now,
			}))
			_ = outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
				BaseEvent:  events.BaseEvent{Type: events.EventOrderStatusChanged, Timestamp: now.Format(time.RFC3339Nano)},
				OrderID:    order.OrderID,
				SupplierID: order.SupplierID,
				Status:     string(StatusCancelled),
				Reason:     "shop_closed_timeout",
			})

		case DecisionForceBypass:
			mutations = append(mutations, spanner.UpdateMap("Orders", map[string]any{
				"OrderId":              order.OrderID,
				"ShopClosedResolution": ShopClosedResolutionBypass,
				"Version":              order.Version + 1,
				"UpdatedAt":            now,
			}))
		}

		// Log entry
		logPayload, _ := json.Marshal(map[string]any{
			"decision": decision,
		})
		mutations = append(mutations, spanner.InsertMap("OrderShopClosedLog", map[string]any{
			"OrderId":   order.OrderID,
			"EventId":   s.newID(),
			"Actor":     "system",
			"Action":    "TIMEOUT",
			"Payload":   logPayload,
			"CreatedAt": now,
		}))

		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}

		return txn.BufferWrite(mutations)
	})

	if err == nil {
		s.invalidateOrderCache(ctx, orderID)
	}

	return err
}

func (s *Service) getProfileForUpdate(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID string) (*credit.Profile, error) {
	row, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID}, []string{
		"CreditLimitMinor", "CurrentBalanceMinor", "AvailableCreditMinor", "Status", "RiskScore", "DelinquencyCount", "Version",
	})
	if err != nil {
		if spanner.ErrCode(err) == 5 { // NotFound
			return nil, nil
		}
		return nil, err
	}
	var p credit.Profile
	var status spanner.NullString
	if err := row.Columns(&p.CreditLimitMinor, &p.CurrentBalanceMinor, &p.AvailableCreditMinor, &status, &p.RiskScore, &p.DelinquencyCount, &p.Version); err != nil {
		return nil, err
	}
	p.RetailerID = retailerID
	p.SupplierID = supplierID
	p.Status = credit.Status(status.StringVal)
	// Re-derive RiskTier
	p.RiskTier = credit.RiskTierLow // simplified for inline; normally deriveRiskTier is called
	if p.DelinquencyCount >= 3 || p.CurrentBalanceMinor > p.CreditLimitMinor {
		p.RiskTier = credit.RiskTierBlock
	} else if p.DelinquencyCount >= 1 || (p.CreditLimitMinor > 0 && p.CurrentBalanceMinor > p.CreditLimitMinor/2) {
		p.RiskTier = credit.RiskTierHigh
	} else if p.CreditLimitMinor > 0 && p.CurrentBalanceMinor > p.CreditLimitMinor/4 {
		p.RiskTier = credit.RiskTierMedium
	}
	return &p, nil
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (s *Service) loadOrderForUpdate(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (*Order, error) {
	row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{
		"OrderId", "Status", "RetailerId", "SupplierId", "WarehouseId", "OrderSource", "LineItemsJson",
		"TotalMinor", "Version", "ShopClosedGraceEndsAt",
	})
	if err != nil {
		return nil, err
	}
	var o Order
	var ts spanner.NullTime
	var warehouseID, orderSource spanner.NullString
	var lineItemsRaw []byte
	if err := row.Columns(
		&o.OrderID, &o.Status, &o.RetailerID, &o.SupplierID, &warehouseID, &orderSource, &lineItemsRaw,
		&o.TotalMinor, &o.Version, &ts,
	); err != nil {
		return nil, err
	}
	if warehouseID.Valid {
		o.WarehouseID = warehouseID.StringVal
	}
	if orderSource.Valid {
		o.Source = OrderSource(orderSource.StringVal)
	}
	if len(lineItemsRaw) > 0 {
		if err := json.Unmarshal(lineItemsRaw, &o.LineItems); err != nil {
			return nil, err
		}
	}
	if ts.Valid {
		o.ShopClosedGraceEndsAt = &ts.Time
	}
	return &o, nil
}
