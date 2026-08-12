package demand

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

type emitData struct {
	RetailerId string
	Sku        string
	ComputedAt time.Time
}

// RunDemandSensingWorker runs the demand-sensing logic periodically.
func (s *Service) RunDemandSensingWorker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.RunDemandSensing(ctx); err != nil {
				// log error
			}
		}
	}
}

// RunDemandSensing represents the demand-sensing-worker logic.
func (s *Service) RunDemandSensing(ctx context.Context) error {
	now := time.Now().UTC()
	fourteenDaysFromNow := now.Add(14 * 24 * time.Hour)

	// 1. Load active signals for the next 14 days.
	activeSignals, err := s.loadActiveSignals(ctx, now, fourteenDaysFromNow)
	if err != nil {
		return fmt.Errorf("load active signals: %w", err)
	}

	// 2. We need to iterate over retailers and their active SKUs.
	// For Phase 1, we can find retailers who had order lines in the last 28 days.
	retailerSkuVelocities, err := s.computeBaseVelocities(ctx, now)
	if err != nil {
		return fmt.Errorf("compute base velocities: %w", err)
	}

	// Group signals for fast lookup
	// For simplicity, we just iterate, but grouping by scope/sku is better for large datasets.

	var mutations []*spanner.Mutation
	var outboxEvents []emitData

	for _, item := range retailerSkuVelocities {
		// Calculate adjustments for the next 14 days
		for dayOffset := 0; dayOffset < 14; dayOffset++ {
			today := now.AddDate(0, 0, dayOffset)

			adj := DemandAdjustment{
				RetailerId:   item.RetailerId,
				Sku:          item.Sku,
				SupplierId:   item.SupplierId,
				Date:         time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC),
				BaseVelocity: item.BaseVelocity,
				Factors:      make(map[string]float64),
				ComputedAt:   now,
			}

			// Base multiplier
			adj.Adjustment = 1.0

			for _, sig := range activeSignals {
				// Check if signal overlaps with targetDate
				if today.Before(sig.StartAt) || today.After(sig.EndAt) {
					continue
				}

				// Check scope applicability
				// Scope formats: legacy = "country:UZ" | "city:Tashkent" | "retailer:uuid"
				//                 new    = "GLOBAL" | "REGION" | "CITY" | "RETAILER" | "RETAILER_SKU"
				// GLOBAL and country-level scopes apply to all retailers.
				// RETAILER scope requires the signal's Meta or scope to encode the retailer id.
				scopeMatch := false
				switch sig.Scope {
				case "GLOBAL", "country:UZ":
					scopeMatch = true
				case "REGION", "CITY", "city:Tashkent":
					scopeMatch = true // Phase 1: all retailers assumed same region
				case "RETAILER", "RETAILER_SKU":
					scopeMatch = sig.Scope == fmt.Sprintf("retailer:%s", adj.RetailerId)
				default:
					scopeMatch = sig.Scope == fmt.Sprintf("retailer:%s", adj.RetailerId)
				}
				if !scopeMatch {
					continue
				}

				// Check SKU applicability
				if sig.Sku != nil && *sig.Sku != adj.Sku {
					continue
				}

				// Overlap logic: for promos, take strongest, etc.
				// Here we just multiply for simplicity, as per "multiply all applicable factors" edge rule.
				adj.Adjustment *= sig.Multiplier
				adj.Factors[string(sig.Type)] = sig.Multiplier
			}

			// Day-of-week / payday: prefer DemandSignals when present (P2-6);
			// otherwise keep static UZ retail calendar as fallback.
			if _, ok := adj.Factors[string(SignalPayday)]; !ok {
				if _, hasDOW := adj.Factors["DAY_OF_WEEK"]; !hasDOW {
					dowFactor := dayOfWeekFactor(today.Weekday())
					adj.Adjustment *= dowFactor
					adj.Factors["DAY_OF_WEEK"] = dowFactor
				}
				pdFactor := paydayFactor(today.Day())
				if pdFactor != 1.0 {
					adj.Adjustment *= pdFactor
					adj.Factors["PAYDAY"] = pdFactor
				}
			} else if _, hasDOW := adj.Factors["DAY_OF_WEEK"]; !hasDOW {
				dowFactor := dayOfWeekFactor(today.Weekday())
				adj.Adjustment *= dowFactor
				adj.Factors["DAY_OF_WEEK"] = dowFactor
			}

			// Clamp final adjustment to [0.6, 1.8]
			adj.Adjustment = math.Max(0.6, math.Min(1.8, adj.Adjustment))
			adj.AdjustedDemand = adj.BaseVelocity * adj.Adjustment

			factorsJson, _ := json.Marshal(adj.Factors)

			mutations = append(mutations, spanner.InsertOrUpdateMap("DemandAdjustments", map[string]any{
				"RetailerId":     adj.RetailerId,
				"Sku":            adj.Sku,
				"Date":           spanner.NullDate{Valid: true, Date: civil.DateOf(today)},
				"SupplierId":     adj.SupplierId,
				"BaseVelocity":   adj.BaseVelocity,
				"Adjustment":     adj.Adjustment,
				"AdjustedDemand": adj.AdjustedDemand,
				"FactorsJson":    string(factorsJson),
				"ComputedAt":     adj.ComputedAt,
			}))
		}

		// 3. Emit outbox event `demand.adjustment.updated`
		outboxEvents = append(outboxEvents, emitData{
			RetailerId: item.RetailerId,
			Sku:        item.Sku,
			ComputedAt: now,
		})

		// Batch write every 500 mutations to avoid Spanner transaction limits
		if len(mutations) > 500 {
			if err := s.flushMutations(ctx, mutations, outboxEvents); err != nil {
				return err
			}
			mutations = nil
			outboxEvents = nil
		}
	}

	if len(mutations) > 0 {
		if err := s.flushMutations(ctx, mutations, outboxEvents); err != nil {
			return err
		}
	}

	if s.afterSensingHook != nil {
		if err := s.afterSensingHook(ctx); err != nil {
			return fmt.Errorf("after sensing hook: %w", err)
		}
	}

	return nil
}

type txBuf struct {
	mutations *[]*spanner.Mutation
}

func (b *txBuf) BufferOutbox(_ context.Context, e outbox.Event) error {
	createdAt := e.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	row := map[string]any{
		"EventId":       e.EventID,
		"AggregateType": e.AggregateType,
		"AggregateId":   e.AggregateID,
		"TopicName":     e.TopicName,
		"Payload":       e.Payload,
		"CreatedAt":     createdAt,
		"PublishedAt":   nil,
	}
	if e.PublishedAt != nil {
		row["PublishedAt"] = e.PublishedAt.UTC()
	}

	*b.mutations = append(*b.mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
	return nil
}

func (s *Service) flushMutations(ctx context.Context, mutations []*spanner.Mutation, events []emitData) error {
	_, err := s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &txBuf{mutations: &mutations}
		for _, ev := range events {
			_ = outbox.EmitJSON(ctx, buf, "DemandAdjustment", ev.RetailerId+":"+ev.Sku, "demand.adjustment.updated", ev)
		}
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}
		return nil
	})
	return err
}

func (s *Service) loadActiveSignals(ctx context.Context, start time.Time, end time.Time) ([]DemandSignal, error) {
	stmt := spanner.Statement{
		SQL: `
			SELECT SignalId, Type, Scope, Sku, StartAt, EndAt, Multiplier, Meta, CreatedAt, CreatedBy, SupplierId
			FROM DemandSignals
			WHERE StartAt <= @End AND EndAt >= @Start
		`,
		Params: map[string]interface{}{
			"Start": start,
			"End":   end,
		},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var signals []DemandSignal
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var sig DemandSignal
		var sku spanner.NullString
		var meta spanner.NullJSON
		var supplier spanner.NullString
		if err := row.Columns(&sig.SignalId, &sig.Type, &sig.Scope, &sku, &sig.StartAt, &sig.EndAt, &sig.Multiplier, &meta, &sig.CreatedAt, &sig.CreatedBy, &supplier); err != nil {
			return nil, err
		}
		if sku.Valid {
			sig.Sku = &sku.StringVal
		}
		if meta.Valid {
			sig.Meta = []byte(meta.Value.(string))
		}
		if supplier.Valid {
			sig.SupplierId = supplier.StringVal
		}
		signals = append(signals, sig)
	}
	return signals, nil
}

type retailerSkuVelocity struct {
	RetailerId   string
	SupplierId   string
	Sku          string
	BaseVelocity float64
}

// computeBaseVelocities computes BaseVelocity from the last 28 days of order lines,
// blended with STORE_POS FlywheelDemandFeed daily averages when present (P2-6).
func (s *Service) computeBaseVelocities(ctx context.Context, now time.Time) ([]retailerSkuVelocity, error) {
	twentyEightDaysAgo := now.Add(-28 * 24 * time.Hour)

	stmt := spanner.Statement{
		SQL: `
			SELECT o.RetailerId, o.SupplierId, l.Sku, SUM(l.DeliveredQty) / 28.0 as BaseVelocity
			FROM Orders o
			JOIN OrderLines l ON o.OrderId = l.OrderId
			WHERE o.Status = 'DELIVERED' AND o.CreatedAt >= @Start
			GROUP BY o.RetailerId, o.SupplierId, l.Sku
			HAVING SUM(l.DeliveredQty) > 0
		`,
		Params: map[string]interface{}{
			"Start": twentyEightDaysAgo,
		},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	byKey := map[velocityKey]retailerSkuVelocity{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var v retailerSkuVelocity
		if err := row.Columns(&v.RetailerId, &v.SupplierId, &v.Sku, &v.BaseVelocity); err != nil {
			return nil, err
		}
		byKey[velocityKey{v.RetailerId, v.SupplierId, v.Sku}] = v
	}

	flywheel, err := s.loadFlywheelDailyAvg(ctx, now)
	if err != nil {
		slog.Warn("flywheel velocity blend skipped", "err", err)
	} else {
		for k, fw := range flywheel {
			if existing, ok := byKey[k]; ok {
				existing.BaseVelocity = 0.65*existing.BaseVelocity + 0.35*fw
				byKey[k] = existing
				continue
			}
			// POS-only SKUs still enter planning.
			byKey[k] = retailerSkuVelocity{
				RetailerId:   k.retailer,
				SupplierId:   k.supplier,
				Sku:          k.sku,
				BaseVelocity: fw,
			}
		}
	}

	out := make([]retailerSkuVelocity, 0, len(byKey))
	for _, v := range byKey {
		out = append(out, v)
	}
	return out, nil
}

type velocityKey struct{ retailer, supplier, sku string }

func (s *Service) loadFlywheelDailyAvg(ctx context.Context, now time.Time) (map[velocityKey]float64, error) {
	if s == nil || s.spanner == nil {
		return nil, nil
	}
	start := now.AddDate(0, 0, -7)
	stmt := spanner.Statement{
		SQL: `
			SELECT RetailerId, IFNULL(SupplierId, '') AS SupplierId, SkuId, SUM(NetSold) / 7.0 AS DailyAvg
			FROM FlywheelDemandFeed
			WHERE Day >= @Start AND NetSold > 0
			GROUP BY RetailerId, SupplierId, SkuId
		`,
		Params: map[string]any{"Start": start},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := map[velocityKey]float64{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var retailer, supplier, sku string
		var avg float64
		if err := row.Columns(&retailer, &supplier, &sku, &avg); err != nil {
			return nil, err
		}
		out[velocityKey{retailer, supplier, sku}] = avg
	}
	return out, nil
}

// dayOfWeekFactor returns a static demand multiplier for the given weekday.
// Based on observed UZ retail patterns: Sunday lowest, midweek peaks.
func dayOfWeekFactor(w time.Weekday) float64 {
	switch w {
	case time.Sunday:
		return 0.75
	case time.Monday:
		return 0.95
	case time.Tuesday:
		return 1.05
	case time.Wednesday:
		return 1.10
	case time.Thursday:
		return 1.05
	case time.Friday:
		return 1.00
	case time.Saturday:
		return 0.85
	default:
		return 1.0
	}
}

// paydayFactor returns a demand boost around common payday windows (1st–2nd, 15th–16th).
func paydayFactor(day int) float64 {
	if day == 1 || day == 2 || day == 15 || day == 16 {
		return 1.15
	}
	return 1.0
}
