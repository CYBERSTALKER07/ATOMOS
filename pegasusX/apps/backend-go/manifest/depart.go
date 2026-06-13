package manifest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// DepartedManifest summarizes the orders rolled into transit when a driver departs.
type DepartedManifest struct {
	ManifestID  string
	SupplierID  string
	WarehouseID string
	RouteID     string
	DriverID    string
	OrderIDs    []string
}

// departOrderRow is the read-phase projection of one manifest order.
type departOrderRow struct {
	OrderID    string
	SupplierID string
	RetailerID string
	Status     string
	Version    int64
}

// DepartDriver flips a driver's SEALED manifest to DISPATCHED and rolls every
// LOADED order on it to IN_TRANSIT, emitting MANIFEST_DISPATCHED plus one
// ORDER_STATUS_CHANGED per transitioned order — all in a single transaction so
// the manifest state and the orders can never diverge. Returns ok=false when the
// driver has no SEALED manifest (idempotent no-op for double-tap depart).
func (s *Store) DepartDriver(ctx context.Context, driverID string, now time.Time) (DepartedManifest, bool, error) {
	if s == nil || s.client == nil {
		return DepartedManifest{}, false, fmt.Errorf("manifest store: nil client")
	}
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return DepartedManifest{}, false, fmt.Errorf("manifest store: empty driver id")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var result DepartedManifest
	var found bool
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		result = DepartedManifest{}
		found = false

		manifestRow, ok, err := readSealedManifest(ctx, txn, driverID)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		orders, err := readManifestOrders(ctx, txn, manifestRow.ManifestID)
		if err != nil {
			return err
		}

		buf := &txnBuffer{}
		mutations := []*spanner.Mutation{spanner.InsertOrUpdateMap("SupplierTruckManifests", map[string]any{
			"ManifestId":   manifestRow.ManifestID,
			"State":        "DISPATCHED",
			"DispatchedAt": now.UTC(),
			"UpdatedAt":    spanner.CommitTimestamp,
		})}

		transitioned := make([]string, 0, len(orders))
		for _, o := range orders {
			if !strings.EqualFold(o.Status, "LOADED") {
				continue
			}
			nextVersion := o.Version + 1
			mutations = append(mutations, spanner.UpdateMap("Orders", map[string]any{
				"OrderId":   o.OrderID,
				"Status":    "IN_TRANSIT",
				"Version":   nextVersion,
				"UpdatedAt": spanner.CommitTimestamp,
			}))
			if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, o.OrderID, events.TopicMain, events.OrderEvent{
				BaseEvent: events.BaseEvent{
					Type:      events.EventOrderStatusChanged,
					Version:   nextVersion,
					Timestamp: now.UTC().Format(time.RFC3339Nano),
				},
				OrderID:        o.OrderID,
				SupplierID:     o.SupplierID,
				RetailerID:     o.RetailerID,
				DriverID:       driverID,
				ManifestID:     manifestRow.ManifestID,
				PreviousStatus: "LOADED",
				Status:         "IN_TRANSIT",
				Version:        nextVersion,
			}); err != nil {
				return err
			}
			transitioned = append(transitioned, o.OrderID)
		}

		if err := outbox.EmitJSON(ctx, buf, events.AggregateManifest, manifestRow.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventManifestDispatched, Timestamp: now.UTC().Format(time.RFC3339Nano)},
			ManifestID:  manifestRow.ManifestID,
			SupplierID:  manifestRow.SupplierID,
			WarehouseID: manifestRow.WarehouseID,
			RouteID:     manifestRow.RouteID,
			DriverID:    driverID,
		}); err != nil {
			return err
		}

		mutations = append(mutations, outboxMutations(buf)...)
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		result = DepartedManifest{
			ManifestID:  manifestRow.ManifestID,
			SupplierID:  manifestRow.SupplierID,
			WarehouseID: manifestRow.WarehouseID,
			RouteID:     manifestRow.RouteID,
			DriverID:    driverID,
			OrderIDs:    transitioned,
		}
		found = true
		return nil
	})
	if err != nil {
		return DepartedManifest{}, false, fmt.Errorf("depart driver %s: %w", driverID, err)
	}
	return result, found, nil
}

func readSealedManifest(ctx context.Context, txn *spanner.ReadWriteTransaction, driverID string) (SupplierTruckRow, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, SupplierId, COALESCE(WarehouseId, '') AS WarehouseId, COALESCE(RouteId, '') AS RouteId
			FROM SupplierTruckManifests
			WHERE DriverId = @driverId AND State = 'SEALED'
			ORDER BY SealedAt DESC LIMIT 1`,
		Params: map[string]any{"driverId": driverID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return SupplierTruckRow{}, false, nil
	}
	if err != nil {
		return SupplierTruckRow{}, false, fmt.Errorf("read sealed manifest: %w", err)
	}
	var m SupplierTruckRow
	if err := row.Columns(&m.ManifestID, &m.SupplierID, &m.WarehouseID, &m.RouteID); err != nil {
		return SupplierTruckRow{}, false, fmt.Errorf("scan sealed manifest: %w", err)
	}
	return m, true, nil
}

func readManifestOrders(ctx context.Context, txn *spanner.ReadWriteTransaction, manifestID string) ([]departOrderRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId, o.SupplierId, o.RetailerId, o.Status, o.Version
			FROM ManifestOrders mo
			JOIN Orders o ON mo.OrderId = o.OrderId
			WHERE mo.ManifestId = @manifestId`,
		Params: map[string]any{"manifestId": manifestID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	rows := make([]departOrderRow, 0, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return rows, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read manifest orders: %w", err)
		}
		var o departOrderRow
		if err := row.Columns(&o.OrderID, &o.SupplierID, &o.RetailerID, &o.Status, &o.Version); err != nil {
			return nil, fmt.Errorf("scan manifest order: %w", err)
		}
		rows = append(rows, o)
	}
}

// ReturnedManifest summarizes the manifest closed when a driver returns to depot.
type ReturnedManifest struct {
	ManifestID  string
	SupplierID  string
	WarehouseID string
	RouteID     string
	DriverID    string
	OrderIDs    []string // orders that were still IN_TRANSIT / RETURNING at the time
}

// ReturnDriver flips a driver's DISPATCHED manifest to COMPLETED and emits
// MANIFEST_COMPLETED to the outbox — all in a single Spanner transaction so
// manifest state and downstream consumers can never diverge.
// Returns ok=false when the driver has no DISPATCHED manifest (idempotent).
func (s *Store) ReturnDriver(ctx context.Context, driverID string, now time.Time) (ReturnedManifest, bool, error) {
	if s == nil || s.client == nil {
		return ReturnedManifest{}, false, fmt.Errorf("manifest store: nil client")
	}
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return ReturnedManifest{}, false, fmt.Errorf("manifest store: empty driver id")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var result ReturnedManifest
	var found bool
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		result = ReturnedManifest{}
		found = false

		manifestRow, ok, err := readDispatchedManifest(ctx, txn, driverID)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		orders, err := readManifestOrders(ctx, txn, manifestRow.ManifestID)
		if err != nil {
			return err
		}

		buf := &txnBuffer{}
		mutations := []*spanner.Mutation{spanner.InsertOrUpdateMap("SupplierTruckManifests", map[string]any{
			"ManifestId":  manifestRow.ManifestID,
			"State":       "COMPLETED",
			"CompletedAt": now.UTC(),
			"UpdatedAt":   spanner.CommitTimestamp,
		})}

		// Collect order IDs that were still active (useful for diagnostics).
		activeIDs := make([]string, 0, len(orders))
		for _, o := range orders {
			if o.Status == "COMPLETED" || o.Status == "CANCELLED" {
				continue
			}
			activeIDs = append(activeIDs, o.OrderID)
		}

		if err := outbox.EmitJSON(ctx, buf, events.AggregateManifest, manifestRow.ManifestID, events.TopicMain, events.ManifestEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventManifestCompleted, Timestamp: now.UTC().Format(time.RFC3339Nano)},
			ManifestID:  manifestRow.ManifestID,
			SupplierID:  manifestRow.SupplierID,
			WarehouseID: manifestRow.WarehouseID,
			RouteID:     manifestRow.RouteID,
			DriverID:    driverID,
		}); err != nil {
			return err
		}

		mutations = append(mutations, outboxMutations(buf)...)
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		result = ReturnedManifest{
			ManifestID:  manifestRow.ManifestID,
			SupplierID:  manifestRow.SupplierID,
			WarehouseID: manifestRow.WarehouseID,
			RouteID:     manifestRow.RouteID,
			DriverID:    driverID,
			OrderIDs:    activeIDs,
		}
		found = true
		return nil
	})
	if err != nil {
		return ReturnedManifest{}, false, fmt.Errorf("return driver %s: %w", driverID, err)
	}
	return result, found, nil
}

// readDispatchedManifest reads the most recent DISPATCHED manifest for a driver.
func readDispatchedManifest(ctx context.Context, txn *spanner.ReadWriteTransaction, driverID string) (SupplierTruckRow, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, SupplierId, COALESCE(WarehouseId, '') AS WarehouseId, COALESCE(RouteId, '') AS RouteId
			FROM SupplierTruckManifests
			WHERE DriverId = @driverId AND State = 'DISPATCHED'
			ORDER BY DispatchedAt DESC LIMIT 1`,
		Params: map[string]any{"driverId": driverID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return SupplierTruckRow{}, false, nil
	}
	if err != nil {
		return SupplierTruckRow{}, false, fmt.Errorf("read dispatched manifest: %w", err)
	}
	var m SupplierTruckRow
	if err := row.Columns(&m.ManifestID, &m.SupplierID, &m.WarehouseID, &m.RouteID); err != nil {
		return SupplierTruckRow{}, false, fmt.Errorf("scan dispatched manifest: %w", err)
	}
	return m, true, nil
}
