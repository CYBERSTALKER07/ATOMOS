package demand

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	h3 "github.com/uber/h3-go/v4"
	"google.golang.org/api/iterator"
)

const (
	densityParentRes     = 7
	densityWindowDays    = 7
	densityMinOrders     = 5
	densitySignalHorizon = 3 // days forward the EVENT_DENSITY signal stays active
)

// RunDensityWorker runs H3 order-density aggregation periodically.
func (s *Service) RunDensityWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// First tick immediately so SSMR/local see signals without waiting a full interval.
	if err := s.ComputeDensitySignals(ctx); err != nil {
		slog.Warn("demand density worker tick failed", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.ComputeDensitySignals(ctx); err != nil {
				slog.Warn("demand density worker tick failed", "err", err)
			}
		}
	}
}

type densityRow struct {
	H3Cell     string
	RetailerID string
	SupplierID string
	OrderCount int64
}

// ComputeDensitySignals aggregates recent orders by H3 parent cell and upserts
// EVENT_DENSITY DemandSignals (retailer-scoped) for hot cells. Also derives
// COMPETITOR_PRESSURE from FlywheelDemandFeed peer skew (P2-6).
func (s *Service) ComputeDensitySignals(ctx context.Context) error {
	if s == nil || s.spanner == nil {
		return nil
	}
	now := time.Now().UTC()
	rows, err := s.loadOrderDensity(ctx, now)
	if err != nil {
		return fmt.Errorf("load order density: %w", err)
	}

	var signals []DemandSignal
	for _, row := range rows {
		parent := parentH3(row.H3Cell, densityParentRes)
		if parent == "" {
			parent = row.H3Cell
		}
		mult := densityMultiplier(row.OrderCount)
		meta, _ := json.Marshal(map[string]any{
			"h3_cell":     row.H3Cell,
			"h3_parent":   parent,
			"order_count": row.OrderCount,
			"window_days": densityWindowDays,
			"source":      "orders_h3",
		})
		seed := fmt.Sprintf("event-density:%s:%s:%s", row.RetailerID, parent, now.Format("2006-01-02"))
		id := uuid.NewMD5(uuid.NameSpaceOID, []byte(seed)).String()
		signals = append(signals, DemandSignal{
			SignalId:   id,
			Type:       SignalEventDensity,
			Scope:      fmt.Sprintf("retailer:%s", row.RetailerID),
			SupplierId: ResolveSupplierID(row.SupplierID),
			StartAt:    time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
			EndAt:      now.AddDate(0, 0, densitySignalHorizon),
			Multiplier: mult,
			Meta:       meta,
			CreatedAt:  now,
			CreatedBy:  "system:density-worker",
		})
		// Extreme local density also emits EVENT (ingestion path for EVENT type — P2-6).
		if row.OrderCount >= 15 {
			eventSeed := fmt.Sprintf("event-hotspot:%s:%s:%s", row.RetailerID, parent, now.Format("2006-01-02"))
			eventID := uuid.NewMD5(uuid.NameSpaceOID, []byte(eventSeed)).String()
			signals = append(signals, DemandSignal{
				SignalId:   eventID,
				Type:       SignalEvent,
				Scope:      fmt.Sprintf("retailer:%s", row.RetailerID),
				SupplierId: ResolveSupplierID(row.SupplierID),
				StartAt:    time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
				EndAt:      now.AddDate(0, 0, densitySignalHorizon),
				Multiplier: math.Min(1.35, mult+0.05),
				Meta:       meta,
				CreatedAt:  now,
				CreatedBy:  "system:density-worker",
			})
		}
	}

	comp, err := s.deriveCompetitorPressure(ctx, now)
	if err != nil {
		slog.Warn("competitor pressure derive failed", "err", err)
	} else {
		signals = append(signals, comp...)
	}

	if len(signals) == 0 {
		return nil
	}
	return s.UpsertSignals(ctx, signals)
}

func (s *Service) loadOrderDensity(ctx context.Context, now time.Time) ([]densityRow, error) {
	start := now.AddDate(0, 0, -densityWindowDays)
	stmt := spanner.Statement{
		SQL: `
			SELECT H3Cell, RetailerId, SupplierId, COUNT(*) AS OrderCount
			FROM Orders
			WHERE CreatedAt >= @Start
			  AND H3Cell IS NOT NULL AND H3Cell != ''
			  AND Status IN UNNEST(@Statuses)
			GROUP BY H3Cell, RetailerId, SupplierId
			HAVING COUNT(*) >= @MinOrders
		`,
		Params: map[string]any{
			"Start":     start,
			"MinOrders": int64(densityMinOrders),
			"Statuses": []string{
				"DELIVERED", "COMPLETED", "DISPATCHED", "OUT_FOR_DELIVERY", "LOADED",
			},
		},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var out []densityRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var r densityRow
		if err := row.Columns(&r.H3Cell, &r.RetailerID, &r.SupplierID, &r.OrderCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

type flywheelPeerRow struct {
	SupplierID string
	RetailerID string
	SKU        string
	NetSold    int64
}

func (s *Service) deriveCompetitorPressure(ctx context.Context, now time.Time) ([]DemandSignal, error) {
	start := now.AddDate(0, 0, -densityWindowDays)
	stmt := spanner.Statement{
		SQL: `
			SELECT IFNULL(SupplierId, '') AS SupplierId, RetailerId, SkuId, SUM(NetSold) AS NetSold
			FROM FlywheelDemandFeed
			WHERE Day >= @Start AND NetSold > 0
			GROUP BY SupplierId, RetailerId, SkuId
		`,
		Params: map[string]any{"Start": start},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var rows []flywheelPeerRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Table may be absent pre-migration — soft no-op.
			return nil, nil
		}
		var r flywheelPeerRow
		if err := row.Columns(&r.SupplierID, &r.RetailerID, &r.SKU, &r.NetSold); err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}

	type key struct{ sid, sku string }
	grouped := map[key][]flywheelPeerRow{}
	for _, r := range rows {
		k := key{sid: r.SupplierID, sku: r.SKU}
		grouped[k] = append(grouped[k], r)
	}

	var signals []DemandSignal
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for k, peers := range grouped {
		if len(peers) < 3 {
			continue
		}
		var sum int64
		for _, p := range peers {
			sum += p.NetSold
		}
		if sum <= 0 {
			continue
		}
		avg := float64(sum) / float64(len(peers))
		sku := k.sku
		for _, p := range peers {
			if float64(p.NetSold) >= 0.5*avg {
				continue
			}
			// Peer sell-through is much stronger → competitive pressure on this retailer.
			meta, _ := json.Marshal(map[string]any{
				"sku":           p.SKU,
				"net_sold":      p.NetSold,
				"peer_avg":      avg,
				"peer_count":    len(peers),
				"window_days":   densityWindowDays,
				"source":        "flywheel_peer_skew",
			})
			seed := fmt.Sprintf("competitor-pressure:%s:%s:%s:%s", p.SupplierID, p.RetailerID, p.SKU, now.Format("2006-01-02"))
			id := uuid.NewMD5(uuid.NameSpaceOID, []byte(seed)).String()
			signals = append(signals, DemandSignal{
				SignalId:   id,
				Type:       SignalCompetitorPressure,
				Scope:      fmt.Sprintf("retailer:%s", p.RetailerID),
				Sku:        &sku,
				SupplierId: ResolveSupplierID(p.SupplierID),
				StartAt:    dayStart,
				EndAt:      now.AddDate(0, 0, densitySignalHorizon),
				Multiplier: 0.85,
				Meta:       meta,
				CreatedAt:  now,
				CreatedBy:  "system:density-worker",
			})
		}
	}
	return signals, nil
}

func densityMultiplier(orderCount int64) float64 {
	// 5 → ~1.05, 20 → ~1.20, clamp [1.05, 1.40]
	m := 1.0 + float64(orderCount-densityMinOrders)*0.01
	return math.Max(1.05, math.Min(1.40, m))
}

func parentH3(cellHex string, res int) string {
	cell := h3.Cell(h3.IndexFromString(cellHex))
	if !cell.IsValid() {
		return ""
	}
	parent, err := cell.Parent(res)
	if err != nil || !parent.IsValid() {
		return ""
	}
	return parent.String()
}
