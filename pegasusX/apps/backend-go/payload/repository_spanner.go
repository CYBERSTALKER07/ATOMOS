package payload

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// SpannerRepository implements the payload repository using Spanner ReadWriteTransactions.
type SpannerRepository struct {
	client      *spanner.Client
	supplierID  string
	warehouseID string
}

func NewSpannerRepository(client *spanner.Client, supplierID, warehouseID string) *SpannerRepository {
	return &SpannerRepository{
		client:      client,
		supplierID:  supplierID,
		warehouseID: strings.TrimSpace(warehouseID),
	}
}

// SpannerClient exposes the underlying client for GS1 ship-unit helpers.
func (r *SpannerRepository) SpannerClient() *spanner.Client {
	if r == nil {
		return nil
	}
	return r.client
}

// spannerTxnBuffer buffers outbox events during a Spanner transaction.
type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

// RunTx executes a function within a Spanner ReadWriteTransaction and flushes the outbox buffer.
func (r *SpannerRepository) RunTx(ctx context.Context, fn func(ctx context.Context, tx PayloadTx) error, emit func(outbox.TxnBuffer) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		txImpl := &spannerPayloadTx{
			txn:         txn,
			supplierID:  r.supplierID,
			warehouseID: r.warehouseID,
		}
		if err := fn(ctx, txImpl); err != nil {
			return err
		}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts := make([]*spanner.Mutation, 0, len(buf.events))
			for _, e := range buf.events {
				muts = append(muts, spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(e)))
			}
			if len(muts) > 0 {
				if err := txn.BufferWrite(muts); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return err
}

func (r *SpannerRepository) Hydrate(ctx context.Context, supplierID string, s *Service) error {
	return r.RunTx(ctx, func(ctx context.Context, tx PayloadTx) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		priorOrders := make(map[string][]ManifestOrder, len(s.manifestOrders))
		for manifestID, orders := range s.manifestOrders {
			if len(orders) == 0 {
				continue
			}
			priorOrders[manifestID] = append([]ManifestOrder(nil), orders...)
		}
		var err error
		s.manifests, err = tx.ListManifests(ctx)
		if err != nil {
			return err
		}
		s.manifestOrders = make(map[string][]ManifestOrder)
		for _, m := range s.manifests {
			orders, err := tx.ListManifestOrders(ctx, m.ManifestID)
			if err != nil {
				return err
			}
			if len(orders) > 0 {
				s.manifestOrders[m.ManifestID] = orders
				continue
			}
			if prior, ok := priorOrders[m.ManifestID]; ok {
				s.manifestOrders[m.ManifestID] = prior
			}
		}
		s.exceptions, err = tx.ListExceptions(ctx)
		if err != nil {
			return err
		}
		s.spannerLoaded = true
		return nil
	}, nil)
}

type spannerPayloadTx struct {
	txn         *spanner.ReadWriteTransaction
	supplierID  string
	warehouseID string
}

func (tx *spannerPayloadTx) ListManifests(ctx context.Context) ([]ManifestRow, error) {
	// B7 PL-P0-6: when repo is warehouse-scoped, filter SQL; empty/NULL WarehouseId
	// rows remain visible (historical) — mutate path enforces non-empty mismatch.
	sql := `SELECT ManifestId, State, StopCount, TotalVolumeVU, MaxVolumeVU,
			  DriverId, TruckId, CreatedAt, UpdatedAt, LoadingStartedAt, SealedAt, WarehouseId
			  FROM SupplierTruckManifests WHERE SupplierId = @sid`
	params := map[string]interface{}{"sid": tx.supplierID}
	if tx.warehouseID != "" {
		sql += ` AND (WarehouseId IS NULL OR WarehouseId = '' OR WarehouseId = @wh)`
		params["wh"] = tx.warehouseID
	}
	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := tx.txn.Query(ctx, stmt)
	defer iter.Stop()

	var manifests []ManifestRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var m ManifestRow
		var stopCount int64
		var totalVolume, maxVolume float64
		var driverID, truckID string
		var createdAt, updatedAt time.Time
		var loadingAt, sealedAt spanner.NullTime
		var warehouseID spanner.NullString

		if err := row.Columns(&m.ManifestID, &m.State, &stopCount, &totalVolume, &maxVolume,
			&driverID, &truckID, &createdAt, &updatedAt,
			&loadingAt, &sealedAt, &warehouseID); err != nil {
			return nil, err
		}
		m.StopCount = int(stopCount)
		m.TotalVolumeVU = int64(totalVolume)
		m.MaxVolumeVU = int64(maxVolume)
		m.DriverID = driverID
		m.VehicleID = truckID
		m.CreatedAt = createdAt.Format(time.RFC3339Nano)
		m.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
		if loadingAt.Valid {
			m.LoadingStartedAt = loadingAt.Time.Format(time.RFC3339Nano)
		}
		if sealedAt.Valid {
			m.SealedAt = sealedAt.Time.Format(time.RFC3339Nano)
		}
		if warehouseID.Valid {
			m.WarehouseID = strings.TrimSpace(warehouseID.StringVal)
		}
		// Defense in depth: filter foreign warehouse even if SQL lacked param (runtime scope).
		if tx.warehouseID != "" && m.WarehouseID != "" && m.WarehouseID != tx.warehouseID {
			continue
		}
		manifests = append(manifests, m)
	}
	return manifests, nil
}

func (tx *spannerPayloadTx) SaveManifest(ctx context.Context, m ManifestRow) error {
	truckID := strings.TrimSpace(m.VehicleID)
	if truckID == "" {
		truckID = "ssmr-truck-unknown"
	}
	driverID := strings.TrimSpace(m.DriverID)
	if driverID == "" {
		driverID = "ssmr-driver-unknown"
	}
	row := map[string]interface{}{
		"ManifestId":       m.ManifestID,
		"SupplierId":       tx.supplierID,
		"State":            m.State,
		"StopCount":        int64(m.StopCount),
		"TotalVolumeVU":    float64(m.TotalVolumeVU),
		"MaxVolumeVU":      float64(m.MaxVolumeVU),
		"DriverId":         driverID,
		"TruckId":          truckID,
		"RouteId":          "route_" + truckID,
		"CreatedAt":        parseTime(m.CreatedAt),
		"UpdatedAt":        spanner.CommitTimestamp,
		"LoadingStartedAt": parseNullTime(m.LoadingStartedAt),
		"SealedAt":         parseNullTime(m.SealedAt),
	}
	if tx.warehouseID != "" {
		row["WarehouseId"] = tx.warehouseID
	}
	mut := spanner.InsertOrUpdateMap("SupplierTruckManifests", row)
	return tx.txn.BufferWrite([]*spanner.Mutation{mut})
}

func (tx *spannerPayloadTx) ListManifestOrders(ctx context.Context, manifestID string) ([]ManifestOrder, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, State, VolumeVU, RemovedReason, UpdatedAt
			  FROM ManifestOrders WHERE ManifestId = @mid`,
		Params: map[string]interface{}{"mid": manifestID},
	}
	iter := tx.txn.Query(ctx, stmt)
	defer iter.Stop()

	var orders []ManifestOrder
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var o ManifestOrder
		o.ManifestID = manifestID
		var volume float64
		var reason spanner.NullString
		var updatedAt time.Time

		if err := row.Columns(&o.OrderID, &o.State, &volume, &reason, &updatedAt); err != nil {
			return nil, err
		}
		o.VolumeVU = int64(volume)
		o.Reason = reason.StringVal
		o.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
		orders = append(orders, o)
	}
	return orders, nil
}

func (tx *spannerPayloadTx) SaveManifestOrder(ctx context.Context, mo ManifestOrder, seq int64) error {
	mut := spanner.InsertOrUpdateMap("ManifestOrders", map[string]interface{}{
		"ManifestId":    mo.ManifestID,
		"OrderId":       mo.OrderID,
		"SequenceIndex": seq,
		"LoadingOrder":  seq,
		"VolumeVU":      float64(mo.VolumeVU),
		"State":         mo.State,
		"RemovedReason": spanner.NullString{StringVal: mo.Reason, Valid: mo.Reason != ""},
		"UpdatedAt":     spanner.CommitTimestamp,
	})
	return tx.txn.BufferWrite([]*spanner.Mutation{mut})
}

func (tx *spannerPayloadTx) ListExceptions(ctx context.Context) ([]ManifestException, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ExceptionId, ManifestId, OrderId, Reason, Metadata, AttemptCount, EscalatedAt, CreatedAt
			  FROM ManifestExceptions WHERE SupplierId = @sid`,
		Params: map[string]interface{}{"sid": tx.supplierID},
	}
	iter := tx.txn.Query(ctx, stmt)
	defer iter.Stop()

	var exceptions []ManifestException
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var e ManifestException
		var metadata spanner.NullString
		var escalatedAt spanner.NullTime
		var createdAt time.Time

		if err := row.Columns(&e.ExceptionID, &e.ManifestID, &e.OrderID, &e.Reason, &metadata, &e.AttemptCount, &escalatedAt, &createdAt); err != nil {
			return nil, err
		}
		e.Metadata = metadata.StringVal
		e.Escalated = escalatedAt.Valid
		e.CreatedAt = createdAt.Format(time.RFC3339Nano)
		exceptions = append(exceptions, e)
	}
	return exceptions, nil
}

func (tx *spannerPayloadTx) SaveException(ctx context.Context, e ManifestException) error {
	row := map[string]interface{}{
		"ExceptionId":  e.ExceptionID,
		"ManifestId":   e.ManifestID,
		"OrderId":      e.OrderID,
		"SupplierId":   tx.supplierID,
		"Reason":       e.Reason,
		"Metadata":     spanner.NullString{StringVal: e.Metadata, Valid: e.Metadata != ""},
		"AttemptCount": e.AttemptCount,
		"CreatedAt":    parseTime(e.CreatedAt),
	}
	if e.Escalated {
		row["EscalatedAt"] = parseTime(e.CreatedAt) // Assuming escalated at creation for simplicity
	} else {
		row["EscalatedAt"] = spanner.NullTime{}
	}
	mut := spanner.InsertOrUpdateMap("ManifestExceptions", row)
	return tx.txn.BufferWrite([]*spanner.Mutation{mut})
}

func (tx *spannerPayloadTx) UpdateOrderAssignment(ctx context.Context, orderID, routeID, driverID string) error {
	orderID = strings.TrimSpace(orderID)
	routeID = strings.TrimSpace(routeID)
	driverID = strings.TrimSpace(driverID)
	if orderID == "" {
		return fmt.Errorf("order_id_required")
	}
	if tx.txn == nil {
		return fmt.Errorf("order assignment: missing transaction")
	}
	row, err := tx.txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"SupplierId"})
	if err != nil {
		return fmt.Errorf("order assignment read: %w", err)
	}
	var supplierID string
	if err := row.Columns(&supplierID); err != nil {
		return fmt.Errorf("order assignment scan: %w", err)
	}
	if tx.supplierID != "" && supplierID != tx.supplierID {
		return fmt.Errorf("order_supplier_mismatch")
	}
	cols := map[string]interface{}{
		"OrderId":   orderID,
		"RouteId":   spanner.NullString{StringVal: routeID, Valid: routeID != ""},
		"UpdatedAt": spanner.CommitTimestamp,
	}
	if driverID != "" {
		cols["DriverId"] = driverID
	}
	return tx.txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("Orders", cols)})
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t
}

func parseNullTime(s string) spanner.NullTime {
	if s == "" {
		return spanner.NullTime{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil || t.IsZero() {
		return spanner.NullTime{}
	}
	return spanner.NullTime{Time: t, Valid: true}
}

func (tx *spannerPayloadTx) DeleteManifestOrder(ctx context.Context, manifestID, orderID string) error {
	if tx.txn == nil {
		return fmt.Errorf("delete manifest order: missing transaction")
	}
	m := spanner.Delete("ManifestOrders", spanner.Key{manifestID, orderID})
	return tx.txn.BufferWrite([]*spanner.Mutation{m})
}

func (tx *spannerPayloadTx) SaveShipUnits(ctx context.Context, units []ShipUnit) error {
	if tx.txn == nil {
		return fmt.Errorf("missing transaction")
	}
	var muts []*spanner.Mutation
	for _, u := range units {
		muts = append(muts, spanner.InsertMap("ManifestShipUnits", map[string]any{
			"ManifestId": u.ManifestID,
			"ShipUnitId": u.ShipUnitID,
			"Sscc":       u.SSCC,
			"OrderId":    u.OrderID,
			"Sequence":   u.Sequence,
			"Gtin":       nullableStr(u.GTIN),
			"CreatedAt":  spanner.CommitTimestamp,
		}))
	}
	if len(muts) > 0 {
		return tx.txn.BufferWrite(muts)
	}
	return nil
}

// LookupVehicleRoute queries Spanner SupplierTruckManifests/ManifestOrders (with RouteTwins/StopTwins fallback)
// using an explicit ReadOnlyTransaction so stop sequences reflect live committed dispatch state.
func (r *SpannerRepository) LookupVehicleRoute(ctx context.Context, supplierID, truckOrRouteID string) (LIFORouteLookup, error) {
	if r == nil || r.client == nil {
		return LIFORouteLookup{}, ErrLIFORepoUnavailable
	}
	target := strings.TrimSpace(truckOrRouteID)
	if target == "" {
		return LIFORouteLookup{}, ErrRouteNotFound
	}
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		sid = strings.TrimSpace(r.supplierID)
	}

	var out LIFORouteLookup
	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
		manifestSQL := `SELECT ManifestId, COALESCE(TruckId, ''), COALESCE(RouteId, '')
			FROM SupplierTruckManifests
			WHERE (TruckId = @id OR RouteId = @id OR ManifestId = @id)
			  AND State != 'CANCELLED'`
		params := map[string]any{"id": target}
		if sid != "" {
			manifestSQL += ` AND SupplierId = @sid`
			params["sid"] = sid
		}
		manifestSQL += ` ORDER BY UpdatedAt DESC LIMIT 1`

		mIter := txn.Query(ctx, spanner.Statement{SQL: manifestSQL, Params: params})
		var manifestID, truckID, routeID string
		var foundManifest bool
		func() {
			defer mIter.Stop()
			row, err := mIter.Next()
			if err == nil {
				if scanErr := row.Columns(&manifestID, &truckID, &routeID); scanErr == nil {
					foundManifest = true
				}
			}
		}()

		if foundManifest && manifestID != "" {
			orderIter := txn.Query(ctx, spanner.Statement{
				SQL: `SELECT OrderId
					FROM ManifestOrders
					WHERE ManifestId = @mid
					  AND State NOT IN ('CANCELLED', 'REMOVED', 'REMOVED_REASSIGNED')
					ORDER BY SequenceIndex ASC, LoadingOrder ASC, OrderId ASC`,
				Params: map[string]any{"mid": manifestID},
			})
			var stops []string
			err := func() error {
				defer orderIter.Stop()
				for {
					row, nextErr := orderIter.Next()
					if nextErr == iterator.Done {
						return nil
					}
					if nextErr != nil {
						return nextErr
					}
					var orderID string
					if colErr := row.Columns(&orderID); colErr != nil {
						return colErr
					}
					if strings.TrimSpace(orderID) != "" {
						stops = append(stops, orderID)
					}
				}
			}()
			if err != nil {
				return fmt.Errorf("lookup manifest orders: %w", err)
			}
			if len(stops) > 0 {
				vid := strings.TrimSpace(truckID)
				if vid == "" {
					vid = target
				}
				rid := strings.TrimSpace(routeID)
				if rid == "" {
					rid = "route_" + vid
				}
				out = LIFORouteLookup{
					VehicleID:  vid,
					RouteID:    rid,
					ManifestID: manifestID,
					StopIDs:    stops,
				}
				return nil
			}
		}

		// Fallback to RouteTwins + StopTwins when stops are tracked on the route twin.
		rtLookupID := target
		if strings.TrimSpace(routeID) != "" {
			rtLookupID = strings.TrimSpace(routeID)
		}
		rtSQL := `SELECT RouteId
			FROM RouteTwins
			WHERE (RouteId = @id OR RouteId = @rtid OR DriverId = @id)`
		rtParams := map[string]any{"id": target, "rtid": rtLookupID}
		if sid != "" {
			rtSQL += ` AND SupplierId = @sid`
			rtParams["sid"] = sid
		}
		rtSQL += ` ORDER BY UpdatedAt DESC LIMIT 1`

		rtIter := txn.Query(ctx, spanner.Statement{SQL: rtSQL, Params: rtParams})
		var resolvedRouteID string
		var foundRoute bool
		func() {
			defer rtIter.Stop()
			row, err := rtIter.Next()
			if err == nil {
				if scanErr := row.Columns(&resolvedRouteID); scanErr == nil {
					foundRoute = true
				}
			}
		}()
		if !foundRoute || strings.TrimSpace(resolvedRouteID) == "" {
			return ErrRouteNotFound
		}

		stIter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT StopId
				FROM StopTwins
				WHERE RouteId = @rid
				ORDER BY Sequence ASC, StopId ASC`,
			Params: map[string]any{"rid": resolvedRouteID},
		})
		defer stIter.Stop()
		var stops []string
		for {
			row, nextErr := stIter.Next()
			if nextErr == iterator.Done {
				break
			}
			if nextErr != nil {
				return fmt.Errorf("lookup stop twins: %w", nextErr)
			}
			var stopID string
			if colErr := row.Columns(&stopID); colErr != nil {
				return colErr
			}
			if strings.TrimSpace(stopID) != "" {
				stops = append(stops, stopID)
			}
		}
		if len(stops) == 0 {
			return ErrRouteNotFound
		}
		out = LIFORouteLookup{
			VehicleID:  target,
			RouteID:    resolvedRouteID,
			ManifestID: manifestID,
			StopIDs:    stops,
		}
		return nil
	})
	if err != nil {
		return LIFORouteLookup{}, err
	}
	return out, nil
}
