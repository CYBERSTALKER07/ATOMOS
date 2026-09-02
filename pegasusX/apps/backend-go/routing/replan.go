package routing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// ErrDispatchLocked is returned when warehouse freeze-lock blocks replan.
var ErrDispatchLocked = errors.New("warehouse dispatch locked")

// ReplanProblem represents the input to the solver.
type ReplanProblem struct {
	RouteID        string
	RemainingStops []StopContext
	DepotLat       float64
	DepotLng       float64
}

// StopContext holds stop data for the solver.
type StopContext struct {
	OrderID       string
	Lat           float64
	Lng           float64
	SequenceIndex int64
	VolumeVU      float64
}

func sequenceUnchanged(stops []StopContext, newSeq []string) bool {
	if len(stops) != len(newSeq) {
		return false
	}
	for i, s := range stops {
		if s.OrderID != newSeq[i] {
			return false
		}
	}
	return true
}

type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func outboxMutation(e outbox.Event) *spanner.Mutation {
	return spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
		"EventId":       e.EventID,
		"AggregateType": e.AggregateType,
		"AggregateId":   e.AggregateID,
		"TopicName":     e.TopicName,
		"Payload":       e.Payload,
		"CreatedAt":     spanner.CommitTimestamp,
		"SupplierId":    e.SupplierID,
	})
}

// ReplanRoute performs continuous replan of remaining stops using local search.
// Respects warehouse freeze-lock, replan cooldown, and max replans/day.
func (s *Service) ReplanRoute(ctx context.Context, routeID, reason string, actor string) error {
	if s == nil || s.spannerClient == nil {
		return fmt.Errorf("routing service unavailable")
	}

	whID, depotLat, depotLng := s.loadManifestMeta(ctx, routeID)
	if whID != "" {
		if frozen, why := s.isWarehouseFrozen(ctx, whID); frozen {
			return fmt.Errorf("%w: %s", ErrDispatchLocked, why)
		}
	}

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "SupplierTruckManifests", spanner.Key{routeID},
			[]string{"State", "ReplanCount", "LastReplannedAt"})
		if err != nil {
			return err
		}
		var state string
		var replanCount int64
		var lastReplanned spanner.NullTime
		if err := row.Columns(&state, &replanCount, &lastReplanned); err != nil {
			return err
		}

		if state == "COMPLETED" || state == "CANCELLED" {
			return nil
		}
		if replanCount >= maxReplansPerDay() {
			return nil
		}
		if lastReplanned.Valid {
			cooldown := time.Duration(replanCooldownSeconds()) * time.Second
			if time.Since(lastReplanned.Time) < cooldown {
				return nil
			}
		}

		stmt := spanner.Statement{
			SQL: `SELECT mo.OrderId, mo.SequenceIndex, o.Lat, o.Lng
			      FROM ManifestOrders mo
			      JOIN Orders o ON mo.OrderId = o.OrderId
			      WHERE mo.ManifestId = @rid AND mo.State NOT IN ('DELIVERED', 'COMPLETED', 'CANCELLED', 'RETURN_TO_WAREHOUSE')
			      ORDER BY mo.SequenceIndex ASC`,
			Params: map[string]any{"rid": routeID},
		}
		iter := txn.Query(ctx, stmt)
		var stops []StopContext
		var oldSeq []string

		err = iter.Do(func(r *spanner.Row) error {
			var sc StopContext
			if err := r.Columns(&sc.OrderID, &sc.SequenceIndex, &sc.Lat, &sc.Lng); err != nil {
				return err
			}
			stops = append(stops, sc)
			oldSeq = append(oldSeq, sc.OrderID)
			return nil
		})
		if err != nil {
			return err
		}
		if len(stops) <= 1 {
			return nil
		}

		problem := ReplanProblem{
			RouteID:        routeID,
			RemainingStops: stops,
			DepotLat:       depotLat,
			DepotLng:       depotLng,
		}
		solver := s.solver
		if solver == nil {
			solver = &DispatchLocalSearchSolver{DepotLat: depotLat, DepotLng: depotLng}
		} else if _, ok := solver.(*DispatchLocalSearchSolver); ok {
			solver = &DispatchLocalSearchSolver{DepotLat: depotLat, DepotLng: depotLng}
		}

		newSequence, err := solver.Solve(problem)
		if err != nil {
			return err
		}
		if sequenceUnchanged(stops, newSequence) {
			return nil
		}

		now := time.Now().UTC()
		var muts []*spanner.Mutation
		for i, orderID := range newSequence {
			muts = append(muts, spanner.UpdateMap("ManifestOrders", map[string]any{
				"ManifestId":    routeID,
				"OrderId":       orderID,
				"SequenceIndex": int64(i + 1),
				"UpdatedAt":     spanner.CommitTimestamp,
			}))
		}
		muts = append(muts, spanner.UpdateMap("SupplierTruckManifests", map[string]any{
			"ManifestId":      routeID,
			"LastReplannedAt": spanner.CommitTimestamp,
			"ReplanCount":     replanCount + 1,
			"ReplanReason":    reason,
			"UpdatedAt":       spanner.CommitTimestamp,
		}))

		replanID := uuid.New().String()
		oldBytes, _ := json.Marshal(oldSeq)
		newBytes, _ := json.Marshal(newSequence)
		muts = append(muts, spanner.Insert("ManifestReplanLog",
			[]string{"ManifestId", "ReplanId", "Reason", "OldSequenceJson", "NewSequenceJson", "TriggeredBy", "CreatedAt"},
			[]any{routeID, replanID, reason, oldBytes, newBytes, actor, spanner.CommitTimestamp},
		))

		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRoute, routeID, events.TopicMain, events.RouteEvent{
			BaseEvent: events.BaseEvent{Type: "route.replanned", Timestamp: now.Format(time.RFC3339Nano)},
			RouteID:   routeID,
		}); err != nil {
			return err
		}
		for _, e := range buf.events {
			muts = append(muts, outboxMutation(e))
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func (s *Service) loadManifestMeta(ctx context.Context, routeID string) (warehouseID string, depotLat, depotLng float64) {
	stmt := spanner.Statement{
		SQL:    `SELECT WarehouseId FROM SupplierTruckManifests WHERE ManifestId = @rid`,
		Params: map[string]any{"rid": routeID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return "", 0, 0
	}
	_ = row.Columns(&warehouseID)
	if warehouseID == "" {
		return "", 0, 0
	}
	wstmt := spanner.Statement{
		SQL:    `SELECT COALESCE(Lat, 0), COALESCE(Lng, 0) FROM Warehouses WHERE WarehouseId = @wid`,
		Params: map[string]any{"wid": warehouseID},
	}
	witer := s.spannerClient.Single().Query(ctx, wstmt)
	defer witer.Stop()
	wrow, err := witer.Next()
	if err != nil {
		return warehouseID, 0, 0
	}
	_ = wrow.Columns(&depotLat, &depotLng)
	return warehouseID, depotLat, depotLng
}

func (s *Service) isWarehouseFrozen(ctx context.Context, warehouseID string) (bool, string) {
	stmt := spanner.Statement{
		SQL: `SELECT EntityType, EntityId FROM WarehouseDispatchLocks
		      WHERE WarehouseId = @wid`,
		Params: map[string]any{"wid": warehouseID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return false, ""
		}
		if err != nil {
			return false, ""
		}
		var entityType, entityID string
		if err := row.Columns(&entityType, &entityID); err != nil {
			return false, ""
		}
		if entityType == "WAREHOUSE" && (entityID == warehouseID || entityID == "warehouse-scope") {
			return true, "warehouse_dispatch_locked"
		}
	}
}
