package factory

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"backend-go/kafka"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── SLA Monitor — Factory Promise Enforcement ─────────────────────────────────
// Runs as a 30-minute cron. Scans InternalTransferOrders with State=APPROVED/LOADING
// that have exceeded their expected delivery window.
//
// Three-tier escalation:
//   - WARNING (>1x expected transit time): Log + FactorySLAEvents row
//   - CRITICAL (>1.5x expected transit time): Kafka event + FactorySLAEvents row
//   - AUTO_REROUTE (>2x expected transit time): Find alternate factory, create replacement
//     transfer, cancel the stalled one, write FactorySLAEvents with ReplacementTransferId

// SLAMonitorService scans for stalled transfers and escalates.
type SLAMonitorService struct {
	Spanner   *spanner.Client
	Producer  *kafkago.Writer
	Optimizer *NetworkOptimizerService
}

type stalledTransfer struct {
	TransferId   string
	FactoryId    string
	WarehouseId  string
	SupplierId   string
	State        string
	CreatedAt    time.Time
	ExpectedHrs  float64 // from SupplyLanes.DampenedTransitHours
	OverdueHrs   float64 // actual elapsed - expected
	OverdueRatio float64 // elapsed / expected
}

// RunSLACheck is the cron entry point — scans all transfers and escalates as needed.
func (s *SLAMonitorService) RunSLACheck(ctx context.Context) error {
	stalled, err := s.findStalledTransfers(ctx)
	if err != nil {
		return err
	}

	if len(stalled) == 0 {
		return nil
	}

	log.Printf("[SLA_MONITOR] Found %d stalled transfers", len(stalled))

	for _, t := range stalled {
		if t.OverdueRatio >= 2.0 {
			s.escalateAutoReroute(ctx, t)
		} else if t.OverdueRatio >= 1.5 {
			s.escalateCritical(ctx, t)
		} else if t.OverdueRatio >= 1.0 {
			s.escalateWarning(ctx, t)
		}
	}

	return nil
}

func (s *SLAMonitorService) findStalledTransfers(ctx context.Context) ([]stalledTransfer, error) {
	// Join InternalTransferOrders with SupplyLanes to get expected transit time
	stmt := spanner.Statement{
		SQL: `SELECT t.TransferId, t.FactoryId, t.WarehouseId, t.SupplierId, t.State, t.CreatedAt,
		             COALESCE(sl.DampenedTransitHours, 48.0)
		      FROM InternalTransferOrders t
		      LEFT JOIN SupplyLanes sl ON sl.FactoryId = t.FactoryId
		                               AND sl.WarehouseId = t.WarehouseId
		                               AND sl.SupplierId = t.SupplierId
		                               AND sl.IsActive = TRUE
		      WHERE t.State IN ('APPROVED', 'LOADING', 'DISPATCHED', 'IN_TRANSIT')`,
	}

	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []stalledTransfer
	now := time.Now().UTC()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var t stalledTransfer
		if err := row.Columns(&t.TransferId, &t.FactoryId, &t.WarehouseId,
			&t.SupplierId, &t.State, &t.CreatedAt, &t.ExpectedHrs); err != nil {
			continue
		}

		elapsed := now.Sub(t.CreatedAt).Hours()
		if t.ExpectedHrs <= 0 {
			t.ExpectedHrs = 48
		}
		t.OverdueRatio = elapsed / t.ExpectedHrs
		t.OverdueHrs = elapsed - t.ExpectedHrs

		// Only consider if past expected time
		if t.OverdueRatio >= 1.0 {
			results = append(results, t)
		}
	}

	return results, nil
}

func (s *SLAMonitorService) escalateWarning(ctx context.Context, t stalledTransfer) {
	eventID := uuid.New().String()
	breachMinutes := int64(t.OverdueHrs * 60)

	_, _ = s.Spanner.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("FactorySLAEvents",
			[]string{"EventId", "TransferId", "SupplierId", "FactoryId", "WarehouseId",
				"EscalationLevel", "SLABreachMinutes", "CreatedAt"},
			[]interface{}{eventID, t.TransferId, t.SupplierId, t.FactoryId, t.WarehouseId,
				"WARNING", breachMinutes, spanner.CommitTimestamp},
		),
	})

	log.Printf("[SLA_MONITOR] WARNING: transfer %s overdue by %.1fh (ratio=%.2f)",
		t.TransferId[:8], t.OverdueHrs, t.OverdueRatio)
}

func (s *SLAMonitorService) escalateCritical(ctx context.Context, t stalledTransfer) {
	eventID := uuid.New().String()
	breachMinutes := int64(t.OverdueHrs * 60)

	_, _ = s.Spanner.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("FactorySLAEvents",
			[]string{"EventId", "TransferId", "SupplierId", "FactoryId", "WarehouseId",
				"EscalationLevel", "SLABreachMinutes", "CreatedAt"},
			[]interface{}{eventID, t.TransferId, t.SupplierId, t.FactoryId, t.WarehouseId,
				"CRITICAL", breachMinutes, spanner.CommitTimestamp},
		),
	})

	if s.Producer != nil {
		evt := kafka.FactorySLABreachEvent{
			TransferId:      t.TransferId,
			SupplierId:      t.SupplierId,
			FactoryId:       t.FactoryId,
			WarehouseId:     t.WarehouseId,
			EscalationLevel: "CRITICAL",
			SLABreachMin:    breachMinutes,
			Timestamp:       time.Now().UTC(),
		}
		payload, _ := json.Marshal(evt)
		_ = s.Producer.WriteMessages(ctx, kafkago.Message{
			Key:   []byte(kafka.EventFactorySLABreach),
			Value: payload,
		})
	}

	log.Printf("[SLA_MONITOR] CRITICAL: transfer %s overdue by %.1fh (ratio=%.2f)",
		t.TransferId[:8], t.OverdueHrs, t.OverdueRatio)
}

func (s *SLAMonitorService) escalateAutoReroute(ctx context.Context, t stalledTransfer) {
	breachMinutes := int64(t.OverdueHrs * 60)

	// 1. Find alternate factory via SupplyLanes (exclude the stalled one)
	altFactory, err := s.findAlternateFactory(ctx, t.SupplierId, t.WarehouseId, t.FactoryId)
	if err != nil || altFactory == "" {
		// No alternate — stay at CRITICAL, don't auto-reroute
		s.escalateCritical(ctx, t)
		log.Printf("[SLA_MONITOR] AUTO_REROUTE: No alternate factory for %s — staying at CRITICAL", t.TransferId[:8])
		return
	}

	// 2. Create replacement transfer
	replacementID := uuid.New().String()
	_, err = s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Read items from original transfer
		iter := txn.Read(ctx, "InternalTransferItems",
			spanner.KeySets(spanner.Key{t.TransferId}.AsPrefix()),
			[]string{"ProductId", "Quantity", "VolumeVU"})
		defer iter.Stop()

		type item struct {
			ProductId string
			Quantity  int64
			VolumeVU  float64
		}
		var items []item
		var totalVolumeVU float64
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var it item
			if err := row.Columns(&it.ProductId, &it.Quantity, &it.VolumeVU); err != nil {
				continue
			}
			items = append(items, it)
			totalVolumeVU += it.VolumeVU
		}

		mutations := []*spanner.Mutation{}

		// Cancel stalled transfer
		mutations = append(mutations, spanner.Update("InternalTransferOrders",
			[]string{"TransferId", "State", "UpdatedAt"},
			[]interface{}{t.TransferId, "CANCELLED", spanner.CommitTimestamp},
		))

		// Create replacement
		mutations = append(mutations, spanner.Insert("InternalTransferOrders",
			[]string{"TransferId", "FactoryId", "WarehouseId", "SupplierId", "State",
				"TotalVolumeVU", "Source", "CreatedAt"},
			[]interface{}{replacementID, altFactory, t.WarehouseId, t.SupplierId, "DRAFT",
				totalVolumeVU, "SYSTEM_REROUTE", spanner.CommitTimestamp},
		))

		for _, it := range items {
			itemID := uuid.New().String()
			mutations = append(mutations, spanner.Insert("InternalTransferItems",
				[]string{"TransferId", "ItemId", "ProductId", "Quantity", "VolumeVU"},
				[]interface{}{replacementID, itemID, it.ProductId, it.Quantity, it.VolumeVU},
			))
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		log.Printf("[SLA_MONITOR] AUTO_REROUTE: Failed to create replacement for %s: %v", t.TransferId[:8], err)
		s.escalateCritical(ctx, t)
		return
	}

	// 3. SLA event with replacement link
	eventID := uuid.New().String()
	_, _ = s.Spanner.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("FactorySLAEvents",
			[]string{"EventId", "TransferId", "SupplierId", "FactoryId", "WarehouseId",
				"EscalationLevel", "SLABreachMinutes", "ReplacementTransferId", "CreatedAt"},
			[]interface{}{eventID, t.TransferId, t.SupplierId, t.FactoryId, t.WarehouseId,
				"AUTO_REROUTE", breachMinutes, replacementID, spanner.CommitTimestamp},
		),
	})

	// 4. Emit event
	if s.Producer != nil {
		evt := kafka.FactorySLABreachEvent{
			TransferId:      t.TransferId,
			SupplierId:      t.SupplierId,
			FactoryId:       t.FactoryId,
			WarehouseId:     t.WarehouseId,
			EscalationLevel: "AUTO_REROUTE",
			SLABreachMin:    breachMinutes,
			ReplacementId:   replacementID,
			Timestamp:       time.Now().UTC(),
		}
		payload, _ := json.Marshal(evt)
		_ = s.Producer.WriteMessages(ctx, kafkago.Message{
			Key:   []byte(kafka.EventFactorySLABreach),
			Value: payload,
		})
	}

	log.Printf("[SLA_MONITOR] AUTO_REROUTE: transfer %s cancelled, replacement %s → factory %s",
		t.TransferId[:8], replacementID[:8], altFactory[:8])
}

func (s *SLAMonitorService) findAlternateFactory(ctx context.Context, supplierID, warehouseID, excludeFactory string) (string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT FactoryId FROM SupplyLanes
		      WHERE SupplierId = @supplierID AND WarehouseId = @warehouseID
		        AND IsActive = TRUE AND FactoryId != @excludeFactory
		      ORDER BY DampenedTransitHours ASC
		      LIMIT 1`,
		Params: map[string]interface{}{
			"supplierID":     supplierID,
			"warehouseID":    warehouseID,
			"excludeFactory": excludeFactory,
		},
	}

	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return "", err
	}
	var factoryID string
	if err := row.Columns(&factoryID); err != nil {
		return "", err
	}
	return factoryID, nil
}
