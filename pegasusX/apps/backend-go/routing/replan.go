package routing

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// ReplanProblem represents the input to the solver.
type ReplanProblem struct {
	RouteID        string
	RemainingStops []StopContext
}

// StopContext holds stop data for the solver.
type StopContext struct {
	OrderID       string
	Lat           float64
	Lng           float64
	SequenceIndex int64
}

// sequenceUnchanged compares the old and new sequences.
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

// spannerTxnBuffer implements outbox.TxnBuffer.
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
		"Topic":         e.TopicName,
		"PayloadJson":   string(e.Payload),
		"CreatedAt":     spanner.CommitTimestamp,
	})
}

// ReplanRoute performs continuous replan/dynamic resequencing of remaining stops.
func (s *Service) ReplanRoute(ctx context.Context, routeID, reason string, actor string) error {
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Load Route and remaining stops (using SupplierTruckManifests as Route table)
		// For now we'll just read the route to ensure it exists.
		row, err := txn.ReadRow(ctx, "SupplierTruckManifests", spanner.Key{routeID}, []string{"State", "ReplanCount"})
		if err != nil {
			return err
		}
		var state string
		var replanCount int64
		if err := row.Columns(&state, &replanCount); err != nil {
			return err
		}

		if state == "COMPLETED" || state == "CANCELLED" {
			return nil
		}

		// Read ManifestOrders (Remaining Stops)
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
			return nil // nothing to replan
		}

		// Build problem for OR-Tools / heuristic
		problem := ReplanProblem{
			RouteID:        routeID,
			RemainingStops: stops,
		}

		newSequence, err := s.solver.Solve(problem)
		if err != nil {
			return err
		}

		if sequenceUnchanged(stops, newSequence) {
			return nil
		}

		now := time.Now().UTC()
		var muts []*spanner.Mutation

		// Apply new sequence numbers
		for i, orderID := range newSequence {
			muts = append(muts, spanner.UpdateMap("ManifestOrders", map[string]any{
				"ManifestId":    routeID,
				"OrderId":       orderID,
				"SequenceIndex": int64(i + 1), // 1-based index or just sequential
				"UpdatedAt":     spanner.CommitTimestamp,
			}))
		}

		// Update Route.LastReplannedAt, ReplanCount, ReplanReason
		muts = append(muts, spanner.UpdateMap("SupplierTruckManifests", map[string]any{
			"ManifestId":      routeID,
			"LastReplannedAt": spanner.CommitTimestamp,
			"ReplanCount":     replanCount + 1,
			"ReplanReason":    reason,
			"UpdatedAt":       spanner.CommitTimestamp,
		}))

		// Write RouteReplanLog
		replanID := uuid.New().String()
		oldBytes, _ := json.Marshal(oldSeq)
		newBytes, _ := json.Marshal(newSequence)
		
		muts = append(muts, spanner.Insert("ManifestReplanLog",
			[]string{"ManifestId", "ReplanId", "Reason", "OldSequenceJson", "NewSequenceJson", "TriggeredBy", "CreatedAt"},
			[]any{routeID, replanID, reason, oldBytes, newBytes, actor, spanner.CommitTimestamp},
		))

		// Emit outbox: route.replanned
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
	
	// Trigger ETA recalculation for remaining stops
	// After every successful replan -> call existing ETA recalculation.
	// This would typically be an async job or outbox processor, but we can do it here if synchronous.
	return err
}
