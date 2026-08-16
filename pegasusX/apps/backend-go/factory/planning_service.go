package factory

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// PlanningService is the Spanner-backed factory planning OS (P5). Flags default off.
type PlanningService struct {
	Spanner *spanner.Client
	Log     *slog.Logger
	Now     func() time.Time
}

func NewPlanningService(client *spanner.Client, log *slog.Logger) *PlanningService {
	if log == nil {
		log = slog.Default()
	}
	return &PlanningService{
		Spanner: client,
		Log:     log,
		Now:     func() time.Time { return time.Now().UTC() },
	}
}

func (p *PlanningService) now() time.Time {
	if p == nil || p.Now == nil {
		return time.Now().UTC()
	}
	return p.Now()
}

// GetNetworkMode returns BALANCED when no row exists.
func (p *PlanningService) GetNetworkMode(ctx context.Context, supplierID string) (string, error) {
	supplierID = strings.TrimSpace(supplierID)
	if p == nil || p.Spanner == nil || supplierID == "" {
		return NetworkModeBalanced, nil
	}
	row, err := p.Spanner.Single().ReadRow(ctx, "NetworkOptimizationMode",
		spanner.Key{supplierID}, []string{"Mode"})
	if err != nil {
		return NetworkModeBalanced, nil
	}
	var mode string
	if err := row.Columns(&mode); err != nil {
		return NetworkModeBalanced, nil
	}
	return normalizeNetworkMode(mode), nil
}

// SetNetworkMode upserts mode and emits NETWORK_MODE_CHANGED in the same txn.
func (p *PlanningService) SetNetworkMode(ctx context.Context, supplierID, mode, updatedBy, reason string) (oldMode, newMode string, err error) {
	if p == nil || p.Spanner == nil {
		return "", "", errors.New("planning_unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	newMode = strings.ToUpper(strings.TrimSpace(mode))
	if supplierID == "" || !validNetworkMode(newMode) {
		return "", "", errors.New("invalid_mode")
	}
	oldMode, _ = p.GetNetworkMode(ctx, supplierID)
	if strings.TrimSpace(updatedBy) == "" {
		updatedBy = "system"
	}
	_, err = p.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("NetworkOptimizationMode", map[string]any{
				"SupplierId": supplierID,
				"Mode":       newMode,
				"UpdatedAt":  spanner.CommitTimestamp,
				"UpdatedBy":  updatedBy,
			}),
		}); err != nil {
			return err
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		payload := map[string]any{
			"type":        events.EventNetworkModeChanged,
			"supplier_id": supplierID,
			"old_mode":    oldMode,
			"new_mode":    newMode,
			"changed_by":  updatedBy,
			"reason":      strings.TrimSpace(reason),
			"timestamp":   p.now().UTC().Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateSupplier, supplierID, events.TopicMain, payload); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
	return oldMode, newMode, err
}

// SelectOptimalFactory loads lanes then falls back to Haversine nearest factory.
func (p *PlanningService) SelectOptimalFactory(ctx context.Context, supplierID, warehouseID, skuID, mode string) (FactorySelection, error) {
	mode = normalizeNetworkMode(mode)
	if mode == NetworkModeManualOnly {
		return FactorySelection{}, nil
	}
	if p == nil || p.Spanner == nil {
		return FactorySelection{}, errors.New("planning_unavailable")
	}
	lanes, err := p.listActiveLanes(ctx, supplierID, warehouseID)
	if err != nil {
		return FactorySelection{}, err
	}
	sel := SelectOptimalFactoryFromLanes(mode, lanes)
	if sel.FactoryID != "" {
		return sel, nil
	}
	return p.fallbackFactory(ctx, supplierID, warehouseID)
}

func (p *PlanningService) listActiveLanes(ctx context.Context, supplierID, warehouseID string) ([]SupplyLane, error) {
	stmt := spanner.Statement{
		SQL: `SELECT LaneId, SupplierId, FactoryId, WarehouseId,
		             DampenedTransitHours, FreightCostMinor, CarbonScoreKg, IsActive, Priority
		      FROM SupplyLanes
		      WHERE SupplierId = @sid AND WarehouseId = @wid AND IsActive = TRUE`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID},
	}
	iter := p.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	var lanes []SupplyLane
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var lane SupplyLane
		if err := row.Columns(
			&lane.LaneID, &lane.SupplierID, &lane.FactoryID, &lane.WarehouseID,
			&lane.DampenedTransitHours, &lane.FreightCostMinor, &lane.CarbonScoreKg,
			&lane.IsActive, &lane.Priority,
		); err != nil {
			continue
		}
		lanes = append(lanes, lane)
	}
	return lanes, nil
}

func (p *PlanningService) fallbackFactory(ctx context.Context, supplierID, warehouseID string) (FactorySelection, error) {
	var lat, lng float64
	var primary, secondary string
	whIter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT IFNULL(Lat, 0), IFNULL(Lng, 0),
		             COALESCE(PrimaryFactoryId, ''), COALESCE(SecondaryFactoryId, '')
		      FROM Warehouses
		      WHERE WarehouseId = @wid AND SupplierId = @sid AND IsActive = TRUE
		      LIMIT 1`,
		Params: map[string]any{"wid": warehouseID, "sid": supplierID},
	})
	defer whIter.Stop()
	whRow, err := whIter.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		return FactorySelection{}, err
	}
	if err == nil {
		_ = whRow.Columns(&lat, &lng, &primary, &secondary)
	}
	facIter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT FactoryId, IFNULL(Lat, 0), IFNULL(Lng, 0), IsActive
		      FROM Factories WHERE SupplierId = @sid AND IsActive = TRUE
		      ORDER BY FactoryId`,
		Params: map[string]any{"sid": supplierID},
	})
	defer facIter.Stop()
	var cands []FactoryCandidate
	for {
		row, err := facIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return FactorySelection{}, err
		}
		var c FactoryCandidate
		if err := row.Columns(&c.FactoryID, &c.Lat, &c.Lng, &c.IsActive); err != nil {
			continue
		}
		cands = append(cands, c)
	}
	return SelectFallbackFactory(lat, lng, primary, secondary, cands), nil
}

func (p *PlanningService) acquireLock(ctx context.Context, supplierID, warehouseID, skuID, factoryID string) bool {
	if p == nil || p.Spanner == nil {
		return true
	}
	lockKey := replenishmentLockKey(skuID, factoryID)
	now := p.now().UTC()
	velocity := p.salesVelocity(ctx, supplierID, warehouseID, skuID)
	var acquired bool
	_, err := p.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		held := lockSnapshot{}
		row, err := txn.ReadRow(ctx, "ReplenishmentLocks", spanner.Key{lockKey},
			[]string{"AcquiredBy", "Priority", "ExpiresAt"})
		if err == nil {
			held.Present = true
			if scanErr := row.Columns(&held.AcquiredBy, &held.Priority, &held.ExpiresAt); scanErr != nil {
				return scanErr
			}
		}
		decision := DecideLockAcquire(now, warehouseID, velocity, held)
		acquired = decision.Acquired
		if !decision.Acquired {
			return nil
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("ReplenishmentLocks", map[string]any{
				"LockKey":    lockKey,
				"AcquiredBy": warehouseID,
				"SupplierId": supplierID,
				"Priority":   velocity,
				"AcquiredAt": spanner.CommitTimestamp,
				"ExpiresAt":  now.Add(replenishmentLockTTL),
			}),
		})
	})
	if err != nil {
		p.Log.Warn("replenishment lock failed", "err", err, "lock_key", lockKey)
		return false
	}
	return acquired
}

func (p *PlanningService) salesVelocity(ctx context.Context, supplierID, warehouseID, skuID string) float64 {
	stmt := spanner.Statement{
		SQL: `SELECT LineItemsJson FROM Orders
		      WHERE SupplierId = @sid AND WarehouseId = @wid AND Status = 'COMPLETED'
		        AND UpdatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)
		      LIMIT 500`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID},
	}
	iter := p.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	var total int64
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0
		}
		var raw []byte
		if err := row.Columns(&raw); err != nil {
			continue
		}
		for _, item := range parsePlanningLineItems(raw) {
			if item.skuID() == skuID {
				total += item.Quantity
			}
		}
	}
	return float64(total)
}

type planningLineItem struct {
	SKU       string `json:"sku"`
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

func (p planningLineItem) skuID() string {
	if s := strings.TrimSpace(p.SKU); s != "" {
		return s
	}
	return strings.TrimSpace(p.ProductID)
}

func parsePlanningLineItems(raw []byte) []planningLineItem {
	if len(raw) == 0 {
		return nil
	}
	var items []planningLineItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	return items
}

func (p *PlanningService) hasOpenSystemTransfer(ctx context.Context, supplierID, warehouseID, factoryID string) bool {
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT TransferId FROM FactoryInternalTransfers
		      WHERE SupplierId = @sid AND WarehouseId = @wid AND FactoryId = @fid
		        AND Source IN UNNEST(@sources)
		        AND State IN UNNEST(@states)
		      LIMIT 1`,
		Params: map[string]any{
			"sid":     supplierID,
			"wid":     warehouseID,
			"fid":     factoryID,
			"sources": []string{TransferSourceThreshold, TransferSourcePredicted},
			"states":  []string{TransferStateCreated, "DRAFT", TransferStateApproved, TransferStateLoading},
		},
	})
	defer iter.Stop()
	_, err := iter.Next()
	return err == nil
}

func (p *PlanningService) insertPlannedTransfers(ctx context.Context, rows []plannedTransfer) (int, error) {
	written := 0
	for _, row := range rows {
		if p.hasOpenSystemTransfer(ctx, row.SupplierID, row.WarehouseID, row.FactoryID) {
			continue
		}
		transferID := uuid.NewString()
		_, err := p.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			muts := []*spanner.Mutation{
				spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]any{
					"TransferId":    transferID,
					"FactoryId":     row.FactoryID,
					"SupplierId":    row.SupplierID,
					"WarehouseId":   row.WarehouseID,
					"State":         row.State,
					"TotalVolumeVU": row.TotalVU,
					"Source":        row.Source,
					"CreatedAt":     spanner.CommitTimestamp,
					"UpdatedAt":     spanner.CommitTimestamp,
				}),
			}
			buf := outbox.NewSpannerTxnBuffer(txn)
			payload := map[string]any{
				"type":         events.EventFactoryTransferCreated,
				"transfer_id":  transferID,
				"factory_id":   row.FactoryID,
				"warehouse_id": row.WarehouseID,
				"supplier_id":  row.SupplierID,
				"source":       row.Source,
				"timestamp":    p.now().UTC().Format(time.RFC3339Nano),
			}
			if err := outbox.EmitJSON(ctx, buf, events.AggregateFactory, transferID, events.TopicMain, payload); err != nil {
				return err
			}
			if err := buf.Flush(ctx); err != nil {
				return err
			}
			return txn.BufferWrite(muts)
		})
		if err != nil {
			p.Log.Warn("planned transfer insert failed", "err", err, "factory_id", row.FactoryID)
			continue
		}
		written++
	}
	return written, nil
}

func (p *PlanningService) recordPullMatrixRun(ctx context.Context, supplierID, source string, transfers, skus, durationMs int64) {
	if p == nil || p.Spanner == nil || supplierID == "" {
		return
	}
	runID := uuid.NewString()
	_, _ = p.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("PullMatrixRuns", map[string]any{
				"RunId":              runID,
				"SupplierId":         supplierID,
				"RunAt":              spanner.CommitTimestamp,
				"TransfersGenerated": transfers,
				"SKUsProcessed":      skus,
				"DurationMs":         durationMs,
				"Source":             source,
			}),
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		payload := map[string]any{
			"type":                events.EventPullMatrixCompleted,
			"run_id":              runID,
			"supplier_id":         supplierID,
			"transfers_generated": transfers,
			"skus_processed":      skus,
			"duration_ms":         durationMs,
			"source":              source,
			"timestamp":           p.now().UTC().Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregatePlanning, runID, events.TopicMain, payload); err != nil {
			return err
		}
		if err := buf.Flush(ctx); err != nil {
			return err
		}
		return txn.BufferWrite(muts)
	})
}

func (p *PlanningService) emitLookAheadCompleted(ctx context.Context, supplierID, source string, durationMs int64) {
	runID := uuid.NewString()
	_, _ = p.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := outbox.NewSpannerTxnBuffer(txn)
		payload := map[string]any{
			"type":         events.EventLookAheadCompleted,
			"run_id":       runID,
			"supplier_id":  supplierID,
			"source":       source,
			"duration_ms":  durationMs,
			"horizon_days": LookAheadWindowDays,
			"timestamp":    p.now().UTC().Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregatePlanning, runID, events.TopicMain, payload); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
}

func (p *PlanningService) SeedDefaultLanes(ctx context.Context, supplierID string) error {
	if p == nil || p.Spanner == nil || strings.TrimSpace(supplierID) == "" {
		return nil
	}
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT LaneId FROM SupplyLanes WHERE SupplierId = @sid LIMIT 1`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	if _, err := iter.Next(); err == nil {
		return nil
	} else if !errors.Is(err, iterator.Done) {
		return err
	}
	whIter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, PrimaryFactoryId FROM Warehouses
		      WHERE SupplierId = @sid AND IsActive = TRUE
		        AND PrimaryFactoryId IS NOT NULL AND PrimaryFactoryId != ''`,
		Params: map[string]any{"sid": supplierID},
	})
	defer whIter.Stop()
	var muts []*spanner.Mutation
	for {
		row, err := whIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var wid, fid string
		if err := row.Columns(&wid, &fid); err != nil {
			continue
		}
		muts = append(muts, spanner.InsertOrUpdateMap("SupplyLanes", map[string]any{
			"LaneId":               uuid.NewString(),
			"SupplierId":           supplierID,
			"FactoryId":            fid,
			"WarehouseId":          wid,
			"TransitTimeHours":     24.0,
			"DampenedTransitHours": 24.0,
			"FreightCostMinor":     int64(0),
			"CarbonScoreKg":        0.0,
			"IsActive":             true,
			"Priority":             int64(0),
			"CreatedAt":            spanner.CommitTimestamp,
			"UpdatedAt":            spanner.CommitTimestamp,
		}))
	}
	if len(muts) == 0 {
		return nil
	}
	_, err := p.Spanner.Apply(ctx, muts)
	return err
}

func (p *PlanningService) findBreachedSKUs(ctx context.Context, supplierID string) ([]breachedSKU, error) {
	sql := `SELECT SupplierId, WarehouseId, ProductId,
	               QuantityOnHand, QuantityReserved, ReorderThreshold
	        FROM SupplierInventoryV2
	        WHERE ReorderThreshold > 0`
	params := map[string]any{}
	if strings.TrimSpace(supplierID) != "" {
		sql += ` AND SupplierId = @sid`
		params["sid"] = supplierID
	}
	sql += ` LIMIT 1000`
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	vu, _ := p.unitVolumes(ctx, supplierID)
	var out []breachedSKU
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var b breachedSKU
		var reserved int64
		if err := row.Columns(&b.SupplierID, &b.WarehouseID, &b.ProductID, &b.CurrentQty, &reserved, &b.SafetyLevel); err != nil {
			continue
		}
		available := b.CurrentQty - reserved
		if available < 0 {
			available = 0
		}
		b.CurrentQty = available
		b.Deficit = safetyDeficit(available, b.SafetyLevel)
		if b.Deficit <= 0 {
			continue
		}
		b.UnitVU = vu[b.ProductID]
		out = append(out, b)
	}
	return out, nil
}

func (p *PlanningService) unitVolumes(ctx context.Context, supplierID string) (map[string]float64, error) {
	result := map[string]float64{}
	if strings.TrimSpace(supplierID) == "" {
		return result, nil
	}
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT ProductId, UnitVolumeVU FROM Products WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return result, err
		}
		var pid string
		var vu float64
		if err := row.Columns(&pid, &vu); err != nil {
			continue
		}
		result[pid] = vu
	}
	return result, nil
}

func (p *PlanningService) picker(ctx context.Context) factoryPicker {
	return func(supplierID, warehouseID, productID, mode string) (string, error) {
		sel, err := p.SelectOptimalFactory(ctx, supplierID, warehouseID, productID, mode)
		if err != nil {
			return "", err
		}
		return sel.FactoryID, nil
	}
}

func (p *PlanningService) locker(ctx context.Context) lockFn {
	return func(supplierID, warehouseID, productID, factoryID string) bool {
		return p.acquireLock(ctx, supplierID, warehouseID, productID, factoryID)
	}
}

// RunPullMatrixForSupplier scans safety breaches and writes SYSTEM_THRESHOLD transfers.
func (p *PlanningService) RunPullMatrixForSupplier(ctx context.Context, supplierID, source string) (int, int, error) {
	if p == nil || p.Spanner == nil {
		return 0, 0, errors.New("planning_unavailable")
	}
	if !PlanningEnabled() {
		return 0, 0, errors.New("factory_planning_disabled")
	}
	start := p.now()
	supplierID = strings.TrimSpace(supplierID)
	_ = p.SeedDefaultLanes(ctx, supplierID)
	mode, _ := p.GetNetworkMode(ctx, supplierID)
	if mode == NetworkModeManualOnly {
		p.recordPullMatrixRun(ctx, supplierID, source, 0, 0, time.Since(start).Milliseconds())
		return 0, 0, nil
	}
	breached, err := p.findBreachedSKUs(ctx, supplierID)
	if err != nil {
		return 0, 0, err
	}
	planned := PlanPullTransfers(mode, source, breached, p.picker(ctx), p.locker(ctx))
	n, err := p.insertPlannedTransfers(ctx, planned)
	p.recordPullMatrixRun(ctx, supplierID, source, int64(n), int64(len(breached)), time.Since(start).Milliseconds())
	return n, len(breached), err
}

func (p *PlanningService) listSupplierIDs(ctx context.Context) ([]string, error) {
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT DISTINCT SupplierId FROM Warehouses WHERE IsActive = TRUE LIMIT 500`,
	})
	defer iter.Stop()
	var ids []string
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var id string
		if err := row.Columns(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// RunPullMatrixAll is the 4h cron (all suppliers).
func (p *PlanningService) RunPullMatrixAll(ctx context.Context) error {
	if !PlanningEnabled() {
		return nil
	}
	ids, err := p.listSupplierIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, _, runErr := p.RunPullMatrixForSupplier(ctx, id, "CRON"); runErr != nil {
			p.Log.Warn("pull matrix supplier failed", "supplier_id", id, "err", runErr)
		}
		if _, _, predErr := p.RunPredictivePushForSupplier(ctx, id, "CRON"); predErr != nil {
			p.Log.Warn("predictive push supplier failed", "supplier_id", id, "err", predErr)
		}
		if lookErr := p.RunLookAheadForSupplier(ctx, id, "CRON"); lookErr != nil {
			p.Log.Warn("look-ahead supplier failed", "supplier_id", id, "err", lookErr)
		}
	}
	return nil
}

func (p *PlanningService) warehouseInventory(ctx context.Context, supplierID, warehouseID string) ([]shadowDemandEntry, error) {
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, QuantityOnHand, QuantityReserved FROM SupplierInventoryV2
		      WHERE SupplierId = @sid AND WarehouseId = @wid`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID},
	})
	defer iter.Stop()
	vu, _ := p.unitVolumes(ctx, supplierID)
	var out []shadowDemandEntry
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var e shadowDemandEntry
		var reserved int64
		if err := row.Columns(&e.ProductID, &e.CurrentStock, &reserved); err != nil {
			continue
		}
		e.CurrentStock -= reserved
		if e.CurrentStock < 0 {
			e.CurrentStock = 0
		}
		e.SupplierID = supplierID
		e.WarehouseID = warehouseID
		e.UnitVU = vu[e.ProductID]
		out = append(out, e)
	}
	return out, nil
}

func (p *PlanningService) futureDemandBySKU(ctx context.Context, supplierID, warehouseID string) (map[string]int64, map[string][]string, error) {
	loc, locErr := auth.TimezoneFromContext(ctx, supplierID)
	if locErr != nil {
		return nil, nil, locErr
	}
	now := time.Now().In(loc)
	horizon := now.Add(LookAheadWindowDays * 24 * time.Hour)
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, ConfirmationStatus, Status, LineItemsJson
		      FROM Orders
		      WHERE SupplierId = @sid AND WarehouseId = @wid
		        AND RequestedDeliveryDate IS NOT NULL
		        AND RequestedDeliveryDate >= @now
		        AND RequestedDeliveryDate <= @horizon`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID, "now": now, "horizon": horizon},
	})
	defer iter.Stop()
	qty := map[string]int64{}
	orders := map[string][]string{}
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		var orderID, conf, status string
		var raw []byte
		if err := row.Columns(&orderID, &conf, &status, &raw); err != nil {
			continue
		}
		if !lookAheadConfirmed(conf) || !lookAheadOpenStatus(status) {
			continue
		}
		for _, item := range parsePlanningLineItems(raw) {
			sku := item.skuID()
			if sku == "" || item.Quantity <= 0 {
				continue
			}
			qty[sku] += item.Quantity
			orders[sku] = append(orders[sku], orderID)
		}
	}
	return qty, orders, nil
}

func (p *PlanningService) latestInsightID(ctx context.Context, warehouseID, productID string) string {
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT InsightId FROM ReplenishmentInsights
		      WHERE WarehouseId = @wid AND ProductId = @pid
		      ORDER BY CreatedAt DESC LIMIT 1`,
		Params: map[string]any{"wid": warehouseID, "pid": productID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return ""
	}
	var id string
	_ = row.Columns(&id)
	return id
}

func (p *PlanningService) insertLookAheadTransfers(ctx context.Context, rows []plannedTransfer, insightBySKU map[string]string) (int, error) {
	written := 0
	for _, row := range rows {
		if p.hasOpenSystemTransfer(ctx, row.SupplierID, row.WarehouseID, row.FactoryID) {
			continue
		}
		transferID := uuid.NewString()
		insightID := ""
		if len(row.ProductIDs) > 0 {
			insightID = insightBySKU[row.ProductIDs[0]]
		}
		_, err := p.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			m := map[string]any{
				"TransferId":    transferID,
				"FactoryId":     row.FactoryID,
				"SupplierId":    row.SupplierID,
				"WarehouseId":   row.WarehouseID,
				"State":         row.State,
				"TotalVolumeVU": row.TotalVU,
				"Source":        row.Source,
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}
			if insightID != "" {
				m["SourceInsightId"] = insightID
			}
			buf := outbox.NewSpannerTxnBuffer(txn)
			payload := map[string]any{
				"type":         events.EventFactoryTransferCreated,
				"transfer_id":  transferID,
				"factory_id":   row.FactoryID,
				"warehouse_id": row.WarehouseID,
				"supplier_id":  row.SupplierID,
				"source":       row.Source,
				"timestamp":    p.now().UTC().Format(time.RFC3339Nano),
			}
			if err := outbox.EmitJSON(ctx, buf, events.AggregateFactory, transferID, events.TopicMain, payload); err != nil {
				return err
			}
			if err := buf.Flush(ctx); err != nil {
				return err
			}
			return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("FactoryInternalTransfers", m)})
		})
		if err != nil {
			p.Log.Warn("look-ahead transfer insert failed", "err", err)
			continue
		}
		written++
	}
	return written, nil
}

// RunLookAheadForSupplier writes transfers when 7d confirmed demand +15% exceeds stock.
func (p *PlanningService) RunLookAheadForSupplier(ctx context.Context, supplierID, source string) error {
	if p == nil || p.Spanner == nil {
		return errors.New("planning_unavailable")
	}
	if !PlanningEnabled() {
		return errors.New("factory_planning_disabled")
	}
	start := p.now()
	mode, _ := p.GetNetworkMode(ctx, supplierID)
	if mode == NetworkModeManualOnly {
		p.emitLookAheadCompleted(ctx, supplierID, source, time.Since(start).Milliseconds())
		return nil
	}
	whIter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT WarehouseId FROM Warehouses WHERE SupplierId = @sid AND IsActive = TRUE`,
		Params: map[string]any{"sid": supplierID},
	})
	defer whIter.Stop()
	var planned []plannedTransfer
	insightBySKU := map[string]string{}
	for {
		row, err := whIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var warehouseID string
		if err := row.Columns(&warehouseID); err != nil {
			continue
		}
		inv, err := p.warehouseInventory(ctx, supplierID, warehouseID)
		if err != nil {
			continue
		}
		demand, _, err := p.futureDemandBySKU(ctx, supplierID, warehouseID)
		if err != nil {
			continue
		}
		var entries []shadowDemandEntry
		for i := range inv {
			inv[i].FutureDemand = demand[inv[i].ProductID]
			inv[i].ShadowDeficit = ShadowDeficit(inv[i].FutureDemand, inv[i].CurrentStock)
			if inv[i].ShadowDeficit <= 0 {
				continue
			}
			if id := p.latestInsightID(ctx, warehouseID, inv[i].ProductID); id != "" {
				insightBySKU[inv[i].ProductID] = id
			}
			entries = append(entries, inv[i])
		}
		planned = append(planned, PlanLookAheadTransfers(mode, entries, p.picker(ctx), p.locker(ctx))...)
	}
	_, err := p.insertLookAheadTransfers(ctx, planned, insightBySKU)
	p.emitLookAheadCompleted(ctx, supplierID, source, time.Since(start).Milliseconds())
	return err
}

// RunKillSwitch cancels system drafts and forces MANUAL_ONLY.
func (p *PlanningService) RunKillSwitch(ctx context.Context, supplierID, actor, reason string) (cancelled int, err error) {
	if p == nil || p.Spanner == nil {
		return 0, errors.New("planning_unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT TransferId, COALESCE(Source, 'MANUAL_EMERGENCY'), State
		      FROM FactoryInternalTransfers WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	var rows []killSwitchRow
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		var r killSwitchRow
		if err := row.Columns(&r.TransferID, &r.Source, &r.State); err != nil {
			continue
		}
		rows = append(rows, r)
	}
	cancelIDs, _ := ClassifyKillSwitch(rows)
	_, _, err = p.SetNetworkMode(ctx, supplierID, NetworkModeManualOnly, actor, reason)
	if err != nil {
		return 0, err
	}
	if len(cancelIDs) == 0 {
		return 0, nil
	}
	_, err = p.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var muts []*spanner.Mutation
		for _, id := range cancelIDs {
			muts = append(muts, spanner.UpdateMap("FactoryInternalTransfers", map[string]any{
				"TransferId": id,
				"State":      TransferStateCancelled,
				"UpdatedAt":  spanner.CommitTimestamp,
			}))
		}
		return txn.BufferWrite(muts)
	})
	return len(cancelIDs), err
}

func (p *PlanningService) laneTransitHours(ctx context.Context, supplierID, factoryID, warehouseID string) float64 {
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT DampenedTransitHours FROM SupplyLanes
		      WHERE SupplierId = @sid AND FactoryId = @fid AND WarehouseId = @wid AND IsActive = TRUE
		      LIMIT 1`,
		Params: map[string]any{"sid": supplierID, "fid": factoryID, "wid": warehouseID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 24
	}
	var hours float64
	if err := row.Columns(&hours); err != nil || hours <= 0 {
		return 24
	}
	return hours
}

func (p *PlanningService) slaAlreadyRaised(ctx context.Context, transferID, level string) bool {
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT EventId FROM FactorySLAEvents
		      WHERE TransferId = @tid AND EscalationLevel = @lvl LIMIT 1`,
		Params: map[string]any{"tid": transferID, "lvl": level},
	})
	defer iter.Stop()
	_, err := iter.Next()
	return err == nil
}

func (p *PlanningService) emitTransferSLA(ctx context.Context, transferID, supplierID, factoryID, warehouseID, level string, minutes int64, replacementID string) error {
	eventID := uuid.NewString()
	_, err := p.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row := map[string]any{
			"EventId":          eventID,
			"TransferId":       transferID,
			"SupplierId":       supplierID,
			"FactoryId":        factoryID,
			"WarehouseId":      warehouseID,
			"EscalationLevel":  level,
			"SLABreachMinutes": minutes,
			"CreatedAt":        spanner.CommitTimestamp,
		}
		if strings.TrimSpace(replacementID) != "" {
			row["ReplacementTransferId"] = replacementID
		}
		muts := []*spanner.Mutation{spanner.InsertMap("FactorySLAEvents", row)}
		buf := outbox.NewSpannerTxnBuffer(txn)
		payload := map[string]any{
			"type":               events.EventFactorySLABreach,
			"kind":               "transfer_transit",
			"transfer_id":        transferID,
			"supplier_id":        supplierID,
			"factory_id":         factoryID,
			"warehouse_id":       warehouseID,
			"escalation_level":   level,
			"sla_breach_minutes": minutes,
			"timestamp":          p.now().UTC().Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateFactory, transferID, events.TopicMain, payload); err != nil {
			return err
		}
		if err := buf.Flush(ctx); err != nil {
			return err
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func nullableString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// RunFactoryTransferSLAWorker is the P5-E entry (kind=transfer_transit). G7 request SLA stays on factory.Service.
func (p *PlanningService) RunFactoryTransferSLAWorker(ctx context.Context) error {
	return p.ScanTransferTransitSLA(ctx)
}

// ScanTransferTransitSLA is P5-E: 1x warning, 1.5x critical, 2x reroute via optimizer.
func (p *PlanningService) ScanTransferTransitSLA(ctx context.Context) error {
	if p == nil || p.Spanner == nil || !PlanningEnabled() {
		return nil
	}
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT TransferId, SupplierId, FactoryId, COALESCE(WarehouseId, ''), State, CreatedAt
		      FROM FactoryInternalTransfers
		      WHERE State IN UNNEST(@states)
		      LIMIT 500`,
		Params: map[string]any{"states": []string{TransferStateApproved, TransferStateLoading, TransferStateCreated}},
	})
	defer iter.Stop()
	now := p.now().UTC()
	type row struct {
		id, sid, fid, wid, state string
		created                  time.Time
	}
	var cands []row
	for {
		r, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		var c row
		if err := r.Columns(&c.id, &c.sid, &c.fid, &c.wid, &c.state, &c.created); err != nil {
			continue
		}
		cands = append(cands, c)
	}
	for _, c := range cands {
		promised := p.laneTransitHours(ctx, c.sid, c.fid, c.wid)
		elapsed := now.Sub(c.created.UTC()).Hours()
		level := transferSLALevel(elapsed, promised)
		if level == "" || p.slaAlreadyRaised(ctx, c.id, level) {
			continue
		}
		minutes := int64(elapsed * 60)
		replacement := ""
		if level == "AUTO_REROUTE" && c.wid != "" {
			mode, _ := p.GetNetworkMode(ctx, c.sid)
			if mode != NetworkModeManualOnly {
				sel, err := p.SelectOptimalFactory(ctx, c.sid, c.wid, "", mode)
				if err == nil && sel.FactoryID != "" && sel.FactoryID != c.fid {
					replacement = uuid.NewString()
					_, _ = p.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
						return txn.BufferWrite([]*spanner.Mutation{
							spanner.UpdateMap("FactoryInternalTransfers", map[string]any{
								"TransferId": c.id,
								"State":      TransferStateCancelled,
								"UpdatedAt":  spanner.CommitTimestamp,
							}),
							spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]any{
								"TransferId":    replacement,
								"FactoryId":     sel.FactoryID,
								"SupplierId":    c.sid,
								"WarehouseId":   c.wid,
								"State":         TransferStateCreated,
								"TotalVolumeVU": float64(1),
								"Source":        TransferSourceThreshold,
								"CreatedAt":     spanner.CommitTimestamp,
								"UpdatedAt":     spanner.CommitTimestamp,
							}),
						})
					})
				}
			}
		}
		_ = p.emitTransferSLA(ctx, c.id, c.sid, c.fid, c.wid, level, minutes, replacement)
	}
	return nil
}

type BatchDispatchResult struct {
	CreatedManifestCount int      `json:"created_manifest_count"`
	ManifestIDs          []string `json:"manifest_ids"`
	Unassigned           []string `json:"unassigned"`
	DispatchAlgo         string   `json:"dispatch_algo"`
	OptimizerClass       string   `json:"optimizer_class"`
}

// RunBatchDispatch packs CREATED transfers with no manifest. Does not invent transfers.
func (p *PlanningService) RunBatchDispatch(ctx context.Context, factoryID, supplierID string) (BatchDispatchResult, error) {
	result := BatchDispatchResult{
		DispatchAlgo:   DispatchAlgoBatcher,
		OptimizerClass: OptimizerHeuristic,
		ManifestIDs:    []string{},
		Unassigned:     []string{},
	}
	if p == nil || p.Spanner == nil {
		return result, errors.New("planning_unavailable")
	}
	factoryID = strings.TrimSpace(factoryID)
	transfers, err := p.fetchDispatchTransfers(ctx, factoryID)
	if err != nil {
		return result, err
	}
	if len(transfers) == 0 {
		return result, nil
	}
	lat, lng, err := p.factoryCoords(ctx, factoryID)
	if err != nil {
		return result, err
	}
	vehicles, err := p.fetchFactoryVehicles(ctx, factoryID, supplierID)
	if err != nil {
		return result, err
	}
	packed, unassigned := PackFFDNNLIFO(lat, lng, transfers, vehicles)
	result.Unassigned = unassigned
	if len(packed) == 0 {
		return result, nil
	}
	var manifestIDs []string
	_, err = p.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var muts []*spanner.Mutation
		buf := outbox.NewSpannerTxnBuffer(txn)
		for _, m := range packed {
			manifestID := uuid.NewString()
			manifestIDs = append(manifestIDs, manifestID)
			muts = append(muts, spanner.InsertOrUpdateMap("FactoryTruckManifests", map[string]any{
				"ManifestId":    manifestID,
				"FactoryId":     factoryID,
				"SupplierId":    supplierID,
				"DriverId":      nullableString(m.Vehicle.DriverID),
				"VehicleId":     nullableString(m.Vehicle.VehicleID),
				"State":         manifestStateDraft,
				"TotalVolumeVU": m.UsedVU,
				"MaxVolumeVU":   m.Vehicle.MaxVolumeVU,
				"StopCount":     int64(len(m.Transfers)),
				"TransferCount": int64(len(m.Transfers)),
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}))
			for _, tr := range m.Transfers {
				muts = append(muts, spanner.UpdateMap("FactoryInternalTransfers", map[string]any{
					"TransferId": tr.TransferID,
					"ManifestId": manifestID,
					"State":      "ASSIGNED",
					"DriverId":   nullableString(m.Vehicle.DriverID),
					"VehicleId":  nullableString(m.Vehicle.VehicleID),
					"UpdatedAt":  spanner.CommitTimestamp,
				}))
			}
			payload := events.ManifestEvent{
				BaseEvent:      events.BaseEvent{Type: events.EventManifestDraftCreated},
				ManifestID:     manifestID,
				ManifestDomain: events.ManifestDomainFactory,
				SupplierID:     supplierID,
				FactoryID:      factoryID,
				TransferCount:  len(m.Transfers),
				TotalVolumeVU:  int64(m.UsedVU),
				DriverID:       m.Vehicle.DriverID,
				VehicleID:      m.Vehicle.VehicleID,
			}
			if err := outbox.EmitJSON(ctx, buf, events.AggregateManifest, manifestID, events.TopicMain, payload); err != nil {
				return err
			}
		}
		if err := buf.Flush(ctx); err != nil {
			return err
		}
		return txn.BufferWrite(muts)
	})
	if err != nil {
		return result, err
	}
	result.ManifestIDs = manifestIDs
	result.CreatedManifestCount = len(manifestIDs)
	return result, nil
}

func (p *PlanningService) fetchDispatchTransfers(ctx context.Context, factoryID string) ([]batchableTransfer, error) {
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT t.TransferId, COALESCE(t.WarehouseId, ''), t.TotalVolumeVU,
		             IFNULL(w.Lat, 0), IFNULL(w.Lng, 0)
		      FROM FactoryInternalTransfers t
		      LEFT JOIN Warehouses w ON t.WarehouseId = w.WarehouseId
		      WHERE t.FactoryId = @fid
		        AND t.State IN UNNEST(@states)
		        AND (t.ManifestId IS NULL OR t.ManifestId = '')
		      ORDER BY t.TotalVolumeVU DESC`,
		Params: map[string]any{
			"fid":    factoryID,
			"states": []string{TransferStateCreated, TransferStateApproved},
		},
	})
	defer iter.Stop()
	var out []batchableTransfer
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var t batchableTransfer
		if err := row.Columns(&t.TransferID, &t.WarehouseID, &t.VolumeVU, &t.WhLat, &t.WhLng); err != nil {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (p *PlanningService) factoryCoords(ctx context.Context, factoryID string) (float64, float64, error) {
	row, err := p.Spanner.Single().ReadRow(ctx, "Factories", spanner.Key{factoryID}, []string{"Lat", "Lng"})
	if err != nil {
		return 0, 0, nil
	}
	var lat, lng spanner.NullFloat64
	if err := row.Columns(&lat, &lng); err != nil {
		return 0, 0, err
	}
	return lat.Float64, lng.Float64, nil
}

func (p *PlanningService) fetchFactoryVehicles(ctx context.Context, factoryID, supplierID string) ([]factoryVehicle, error) {
	iter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT VehicleId, MaxVolumeVU FROM Vehicles
		      WHERE SupplierId = @sid AND IsActive = TRUE
		        AND HomeNodeType = 'FACTORY' AND HomeNodeId = @fid`,
		Params: map[string]any{"sid": supplierID, "fid": factoryID},
	})
	defer iter.Stop()
	var vehicles []factoryVehicle
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var v factoryVehicle
		if err := row.Columns(&v.VehicleID, &v.MaxVolumeVU); err != nil {
			continue
		}
		vehicles = append(vehicles, v)
	}
	drvIter := p.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT DriverId, COALESCE(VehicleId, '') FROM Drivers
		      WHERE SupplierId = @sid AND IsActive = TRUE AND OnShift = TRUE
		        AND HomeNodeType = 'FACTORY' AND HomeNodeId = @fid`,
		Params: map[string]any{"sid": supplierID, "fid": factoryID},
	})
	defer drvIter.Stop()
	byVehicle := map[string]string{}
	var extras []string
	for {
		row, err := drvIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			break
		}
		var did, vid string
		if err := row.Columns(&did, &vid); err != nil {
			continue
		}
		if vid != "" {
			byVehicle[vid] = did
		} else {
			extras = append(extras, did)
		}
	}
	ei := 0
	for i := range vehicles {
		if d := byVehicle[vehicles[i].VehicleID]; d != "" {
			vehicles[i].DriverID = d
			continue
		}
		if ei < len(extras) {
			vehicles[i].DriverID = extras[ei]
			ei++
		}
	}
	return vehicles, nil
}

// StartPlanningCron runs pull-matrix + look-ahead every 4h when the planning flag is on.
func (p *PlanningService) StartPlanningCron(ctx context.Context) {
	if p == nil || p.Spanner == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(4 * time.Hour)
		defer ticker.Stop()
		slaTicker := time.NewTicker(30 * time.Minute)
		defer slaTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := p.RunPullMatrixAll(ctx); err != nil {
					p.Log.Warn("factory planning cron failed", "err", err)
				}
			case <-slaTicker.C:
				if err := p.RunFactoryTransferSLAWorker(ctx); err != nil {
					p.Log.Warn("factory transfer sla scan failed", "err", err)
				}
			}
		}
	}()
}
