package whatif

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const simQueryCap = 1000

// NewSpannerService constructs a What-If Service backed by Cloud Spanner.
func NewSpannerService(client *spanner.Client, cfg ServiceConfig) *Service {
	if client == nil {
		return NewService(nil, cfg)
	}
	return NewService(NewSpannerRepository(client), cfg)
}

// SpannerRepository implements Repository over Cloud Spanner using
// ReadOnlyTransaction queries and atomic ReadWriteTransaction mutations.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed What-If repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

type tenantStampedOutboxBuf struct {
	inner    outbox.TxnBuffer
	tenantID string
}

func (b *tenantStampedOutboxBuf) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(b.tenantID) != "" {
		e.SupplierID = outbox.ResolveSupplierID(b.tenantID, e.Payload)
	}
	return b.inner.BufferOutbox(ctx, e)
}

func (r *SpannerRepository) LoadSimulationDataset(ctx context.Context, tenantID, scenarioID, entityID string) (SimulationDataset, error) {
	if r == nil || r.client == nil {
		return SimulationDataset{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	sid := strings.TrimSpace(scenarioID)
	eid := strings.TrimSpace(entityID)
	ds := SimulationDataset{TenantID: tid, EntityID: eid}

	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	var err error
	if sid != "" {
		if ds.Scenario, err = loadScenario(ctx, roTxn, tid, sid); err != nil {
			return SimulationDataset{}, err
		}
		if ds.ScenarioPatches, err = loadScenarioPatches(ctx, roTxn, tid, sid); err != nil {
			return SimulationDataset{}, err
		}
	}
	if ds.Forecasts, err = loadForecasts(ctx, roTxn, tid); err != nil {
		return SimulationDataset{}, err
	}
	if ds.Inventory, err = loadInventory(ctx, roTxn, tid); err != nil {
		return SimulationDataset{}, err
	}
	if ds.Products, err = loadProducts(ctx, roTxn, tid); err != nil {
		return SimulationDataset{}, err
	}
	if ds.CostLayers, err = loadCostLayers(ctx, roTxn, tid); err != nil {
		return SimulationDataset{}, err
	}
	if ds.Orders, err = loadOrders(ctx, roTxn, tid); err != nil {
		return SimulationDataset{}, err
	}
	if ds.Routes, err = loadRoutes(ctx, roTxn, tid); err != nil {
		return SimulationDataset{}, err
	}
	if ds.CreditProfiles, err = loadCreditProfiles(ctx, roTxn, tid); err != nil {
		return SimulationDataset{}, err
	}
	if ds.ArInvoices, err = loadOpenInvoices(ctx, roTxn, tid); err != nil {
		return SimulationDataset{}, err
	}
	return ds, nil
}

func loadScenario(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid, sid string) (*ScenarioRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT ScenarioId, Status, COALESCE(Label, ''), HorizonDays, FactoryDowntimeHours
		      FROM PlanningScenarios
		      WHERE SupplierId = @tid AND ScenarioId = @sid
		      LIMIT 1`,
		Params: map[string]any{"tid": tid, "sid": sid},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query PlanningScenarios: %w", err)
	}
	var rec ScenarioRecord
	if err := row.Columns(&rec.ScenarioID, &rec.Status, &rec.Label, &rec.HorizonDays, &rec.FactoryDowntimeHours); err != nil {
		return nil, fmt.Errorf("scan PlanningScenarios: %w", err)
	}
	return &rec, nil
}

func loadScenarioPatches(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid, sid string) ([]ScenarioPatchRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT ScenarioId, ObjectType, ObjectId, Operation, PatchJson
		      FROM ScenarioObjectPatches
		      WHERE SupplierId = @tid AND ScenarioId = @sid
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "sid": sid, "lim": int64(500)},
	})
	defer iter.Stop()
	var out []ScenarioPatchRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query ScenarioObjectPatches: %w", err)
		}
		var rec ScenarioPatchRecord
		if err := row.Columns(&rec.ScenarioID, &rec.ObjectType, &rec.ObjectID, &rec.Operation, &rec.PatchJSON); err != nil {
			return nil, fmt.Errorf("scan ScenarioObjectPatches: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadForecasts(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]ForecastRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT Sku, COALESCE(ProductId, ''), LocationId, IsWinner, P50Milli
		      FROM ForecastValues
		      WHERE (EntityId = @tid OR EntityId IN (SELECT EntityId FROM LegalEntities WHERE TenantId = @tid))
		      ORDER BY ForecastDate DESC
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(simQueryCap)},
	})
	defer iter.Stop()
	var out []ForecastRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query ForecastValues: %w", err)
		}
		var rec ForecastRecord
		if err := row.Columns(&rec.SKU, &rec.ProductID, &rec.LocationID, &rec.IsWinner, &rec.P50Milli); err != nil {
			return nil, fmt.Errorf("scan ForecastValues: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadInventory(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]InventoryRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, ProductId, QuantityOnHand, QuantityReserved, ReorderThreshold
		      FROM SupplierInventoryV2
		      WHERE SupplierId = @tid
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(simQueryCap)},
	})
	defer iter.Stop()
	var out []InventoryRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query SupplierInventoryV2: %w", err)
		}
		var rec InventoryRecord
		if err := row.Columns(&rec.WarehouseID, &rec.ProductID, &rec.QuantityOnHand, &rec.QuantityReserved, &rec.ReorderThreshold); err != nil {
			return nil, fmt.Errorf("scan SupplierInventoryV2: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadProducts(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]ProductPriceRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, Name, PriceMinor, Currency
		      FROM Products
		      WHERE SupplierId = @tid AND IsActive = true
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(simQueryCap)},
	})
	defer iter.Stop()
	var out []ProductPriceRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query Products: %w", err)
		}
		var rec ProductPriceRecord
		if err := row.Columns(&rec.ProductID, &rec.Name, &rec.PriceMinor, &rec.Currency); err != nil {
			return nil, fmt.Errorf("scan Products: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadCostLayers(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]CostLayerRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, UnitCostMinor
		      FROM InventoryCostLayers
		      WHERE (EntityId = @tid OR EntityId IN (SELECT EntityId FROM LegalEntities WHERE TenantId = @tid))
		      ORDER BY ReceivedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(simQueryCap)},
	})
	defer iter.Stop()
	var out []CostLayerRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query InventoryCostLayers: %w", err)
		}
		var rec CostLayerRecord
		if err := row.Columns(&rec.ProductID, &rec.UnitCostMinor); err != nil {
			return nil, fmt.Errorf("scan InventoryCostLayers: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadOrders(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]OrderDispatchRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, COALESCE(WarehouseId, ''), COALESCE(RouteId, ''), COALESCE(ManifestId, ''),
		             Status, TotalMinor, DeliveryFeeMinor, LineItemsJson
		      FROM Orders
		      WHERE SupplierId = @tid
		      ORDER BY CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(500)},
	})
	defer iter.Stop()
	var out []OrderDispatchRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query Orders: %w", err)
		}
		var (
			rec       OrderDispatchRecord
			lineBytes []byte
		)
		if err := row.Columns(
			&rec.OrderID, &rec.WarehouseID, &rec.RouteID, &rec.ManifestID,
			&rec.Status, &rec.TotalMinor, &rec.DeliveryFeeMinor, &lineBytes,
		); err != nil {
			return nil, fmt.Errorf("scan Orders: %w", err)
		}
		rec.LineItemsJSON = string(lineBytes)
		out = append(out, rec)
	}
	return out, nil
}

func loadRoutes(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]RouteRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT RouteId, COALESCE(ReplanCount, 0)
		      FROM RoutePerformanceAnalytics
		      WHERE SupplierId = @tid
		      ORDER BY ComputedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(500)},
	})
	defer iter.Stop()
	var out []RouteRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query RoutePerformanceAnalytics: %w", err)
		}
		var rec RouteRecord
		if err := row.Columns(&rec.RouteID, &rec.ReplanCount); err != nil {
			return nil, fmt.Errorf("scan RoutePerformanceAnalytics: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadCreditProfiles(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]CreditProfileRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT RetailerId, CreditLimitMinor, CurrentBalanceMinor, ReservedMinor,
		             RiskScore, DelinquencyCount, Status
		      FROM RetailerCreditProfiles
		      WHERE SupplierId = @tid
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(simQueryCap)},
	})
	defer iter.Stop()
	var out []CreditProfileRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query RetailerCreditProfiles: %w", err)
		}
		var rec CreditProfileRecord
		if err := row.Columns(
			&rec.RetailerID, &rec.CreditLimitMinor, &rec.CurrentBalanceMinor, &rec.ReservedMinor,
			&rec.RiskScore, &rec.DelinquencyCount, &rec.Status,
		); err != nil {
			return nil, fmt.Errorf("scan RetailerCreditProfiles: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func loadOpenInvoices(ctx context.Context, roTxn *spanner.ReadOnlyTransaction, tid string) ([]ArInvoiceRecord, error) {
	iter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT InvoiceId, RetailerId, BalanceMinor, Status
		      FROM ArInvoices
		      WHERE SupplierId = @tid AND Status IN ('OPEN', 'PARTIAL', 'OVERDUE')
		      LIMIT @lim`,
		Params: map[string]any{"tid": tid, "lim": int64(simQueryCap)},
	})
	defer iter.Stop()
	var out []ArInvoiceRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query ArInvoices: %w", err)
		}
		var rec ArInvoiceRecord
		if err := row.Columns(&rec.InvoiceID, &rec.RetailerID, &rec.BalanceMinor, &rec.Status); err != nil {
			return nil, fmt.Errorf("scan ArInvoices: %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func (r *SpannerRepository) SaveSimulation(
	ctx context.Context,
	sim WhatIfSimulation,
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) (WhatIfSimulation, bool, error) {
	if r == nil || r.client == nil {
		return WhatIfSimulation{}, false, ErrRepoUnavailable
	}
	var saved WhatIfSimulation
	var replayed bool

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		saved = WhatIfSimulation{}
		replayed = false

		existingRow, rErr := txn.ReadRow(txCtx, "WhatIfSimulations", spanner.Key{sim.TenantID, sim.SimulationID}, simulationColumns())
		if rErr == nil {
			existing, sErr := scanSimulationRow(existingRow)
			if sErr != nil {
				return sErr
			}
			saved = existing
			replayed = true
			return nil
		} else if spanner.ErrCode(rErr) != codes.NotFound && !errors.Is(rErr, spanner.ErrRowNotFound) {
			return fmt.Errorf("read existing WhatIfSimulations: %w", rErr)
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("WhatIfSimulations", map[string]any{
				"TenantId":                 sim.TenantID,
				"SimulationId":             sim.SimulationID,
				"ScenarioId":               nullableStr(sim.ScenarioID),
				"Name":                     sim.Name,
				"EntityId":                 nullableStr(sim.EntityID),
				"HorizonDays":              sim.HorizonDays,
				"ShocksJson":               sim.ShocksJSON,
				"ForecastDeltaMinor":       sim.ForecastDeltaMinor,
				"ReplenishmentCostMinor":   sim.ReplenishmentCostMinor,
				"DispatchCostDeltaMinor":   sim.DispatchCostDeltaMinor,
				"CreditExposureDeltaMinor": sim.CreditExposureDeltaMinor,
				"NetPnLDeltaMinor":         sim.NetPnLDeltaMinor,
				"BreakdownJson":            nullableStr(sim.BreakdownJSON),
				"CommittedProposalIdsJson": sim.CommittedProposalIdsJSON,
				"Status":                   sim.Status,
				"CreatedBy":                sim.CreatedBy,
				"CreatedAt":                spanner.CommitTimestamp,
				"UpdatedAt":                spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("buffer WhatIfSimulations insert: %w", err)
		}

		buf := outbox.NewSpannerTxnBuffer(txn)
		stamped := &tenantStampedOutboxBuf{inner: buf, tenantID: sim.TenantID}
		if eErr := outbox.EmitJSON(txCtx, stamped, events.AggregateWhatIfSimulation, sim.SimulationID, events.TopicIntel, events.WhatIfSimulatedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventWhatIfSimulated,
				Version:   1,
				Timestamp: sim.CreatedAt.UTC().Format(time.RFC3339Nano),
			},
			TenantID:                 sim.TenantID,
			SimulationID:             sim.SimulationID,
			ScenarioID:               sim.ScenarioID,
			Name:                     sim.Name,
			EntityID:                 sim.EntityID,
			HorizonDays:              sim.HorizonDays,
			ForecastDeltaMinor:       sim.ForecastDeltaMinor,
			ReplenishmentCostMinor:   sim.ReplenishmentCostMinor,
			DispatchCostDeltaMinor:   sim.DispatchCostDeltaMinor,
			CreditExposureDeltaMinor: sim.CreditExposureDeltaMinor,
			NetPnLDeltaMinor:         sim.NetPnLDeltaMinor,
			Status:                   sim.Status,
			CreatedBy:                sim.CreatedBy,
			OccurredAt:               sim.CreatedAt.UTC().Format(time.RFC3339Nano),
		}); eErr != nil {
			return fmt.Errorf("emit whatif.simulation.completed: %w", eErr)
		}
		if err := buf.Flush(txCtx); err != nil {
			return fmt.Errorf("flush whatif outbox: %w", err)
		}
		if auditRec != nil {
			if err := auditRec.RecordEventInTxn(txCtx, txn, auditIn); err != nil {
				return fmt.Errorf("audit whatif.simulate: %w", err)
			}
		}
		saved = sim
		return nil
	})
	if err != nil {
		return WhatIfSimulation{}, false, err
	}
	return saved, replayed, nil
}

func (r *SpannerRepository) ListSimulations(ctx context.Context, f ListFilter) ([]WhatIfSimulation, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	clauses := []string{"TenantId = @tid"}
	params := map[string]any{
		"tid": strings.TrimSpace(f.TenantID),
		"lim": int64(clampLimit(f.Limit)),
	}
	if st := strings.ToUpper(strings.TrimSpace(f.Status)); st != "" {
		clauses = append(clauses, "Status = @status")
		params["status"] = st
	}
	sql := fmt.Sprintf(`SELECT TenantId, SimulationId, COALESCE(ScenarioId, ''), Name, COALESCE(EntityId, ''),
	                           HorizonDays, ShocksJson, ForecastDeltaMinor, ReplenishmentCostMinor,
	                           DispatchCostDeltaMinor, CreditExposureDeltaMinor, NetPnLDeltaMinor,
	                           COALESCE(BreakdownJson, ''), COALESCE(CommittedProposalIdsJson, '[]'),
	                           Status, CreatedBy, CreatedAt, UpdatedAt
	                    FROM WhatIfSimulations
	                    WHERE %s
	                    ORDER BY CreatedAt DESC, SimulationId ASC
	                    LIMIT @lim`, strings.Join(clauses, " AND "))

	iter := roTxn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []WhatIfSimulation
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query WhatIfSimulations: %w", err)
		}
		sim, err := scanSimulationRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, sim)
	}
	return out, nil
}

func (r *SpannerRepository) GetSimulation(ctx context.Context, tenantID, simID string) (WhatIfSimulation, bool, error) {
	if r == nil || r.client == nil {
		return WhatIfSimulation{}, false, ErrRepoUnavailable
	}
	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	row, err := roTxn.ReadRow(ctx, "WhatIfSimulations", spanner.Key{strings.TrimSpace(tenantID), strings.TrimSpace(simID)}, simulationColumns())
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound || errors.Is(err, spanner.ErrRowNotFound) {
			return WhatIfSimulation{}, false, nil
		}
		return WhatIfSimulation{}, false, fmt.Errorf("read WhatIfSimulations: %w", err)
	}
	sim, err := scanSimulationRow(row)
	if err != nil {
		return WhatIfSimulation{}, false, err
	}
	return sim, true, nil
}

func (r *SpannerRepository) CommitSimulationProposals(
	ctx context.Context,
	tenantID, simID string,
	mutate func(cur WhatIfSimulation) (WhatIfSimulation, error),
	auditRec AuditRecorder,
	auditIn func(updated WhatIfSimulation) AuditRecordInput,
	_ time.Time,
) (WhatIfSimulation, error) {
	if r == nil || r.client == nil {
		return WhatIfSimulation{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	sid := strings.TrimSpace(simID)
	var updated WhatIfSimulation

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, rErr := txn.ReadRow(txCtx, "WhatIfSimulations", spanner.Key{tid, sid}, simulationColumns())
		if rErr != nil {
			if spanner.ErrCode(rErr) == codes.NotFound || errors.Is(rErr, spanner.ErrRowNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("read WhatIfSimulations for commit: %w", rErr)
		}
		cur, sErr := scanSimulationRow(row)
		if sErr != nil {
			return sErr
		}
		next, mErr := mutate(cur)
		if mErr != nil {
			return mErr
		}
		updated = next

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("WhatIfSimulations", map[string]any{
				"TenantId":                 tid,
				"SimulationId":             sid,
				"Status":                   next.Status,
				"CommittedProposalIdsJson": next.CommittedProposalIdsJSON,
				"BreakdownJson":            nullableStr(next.BreakdownJSON),
				"UpdatedAt":                spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("buffer WhatIfSimulations commit update: %w", err)
		}
		if auditRec != nil && auditIn != nil {
			if err := auditRec.RecordEventInTxn(txCtx, txn, auditIn(next)); err != nil {
				return fmt.Errorf("audit whatif.commit_proposals: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return WhatIfSimulation{}, err
	}
	return updated, nil
}

func simulationColumns() []string {
	return []string{
		"TenantId", "SimulationId", "ScenarioId", "Name", "EntityId", "HorizonDays",
		"ShocksJson", "ForecastDeltaMinor", "ReplenishmentCostMinor", "DispatchCostDeltaMinor",
		"CreditExposureDeltaMinor", "NetPnLDeltaMinor", "BreakdownJson", "CommittedProposalIdsJson",
		"Status", "CreatedBy", "CreatedAt", "UpdatedAt",
	}
}

func scanSimulationRow(row *spanner.Row) (WhatIfSimulation, error) {
	var (
		sim                WhatIfSimulation
		scenID, entID      spanner.NullString
		breakdown, propIDs spanner.NullString
	)
	if err := row.Columns(
		&sim.TenantID, &sim.SimulationID, &scenID, &sim.Name, &entID, &sim.HorizonDays,
		&sim.ShocksJSON, &sim.ForecastDeltaMinor, &sim.ReplenishmentCostMinor, &sim.DispatchCostDeltaMinor,
		&sim.CreditExposureDeltaMinor, &sim.NetPnLDeltaMinor, &breakdown, &propIDs,
		&sim.Status, &sim.CreatedBy, &sim.CreatedAt, &sim.UpdatedAt,
	); err != nil {
		return WhatIfSimulation{}, fmt.Errorf("scan WhatIfSimulations row: %w", err)
	}
	sim.ScenarioID = scenID.StringVal
	sim.EntityID = entID.StringVal
	sim.BreakdownJSON = breakdown.StringVal
	sim.CommittedProposalIdsJSON = propIDs.StringVal
	if sim.CommittedProposalIdsJSON == "" {
		sim.CommittedProposalIdsJSON = "[]"
	}
	_ = json.Unmarshal([]byte(sim.ShocksJSON), &sim.Shocks)
	if sim.BreakdownJSON != "" {
		var b WhatIfBreakdown
		dec := json.NewDecoder(strings.NewReader(sim.BreakdownJSON))
		dec.UseNumber()
		if err := dec.Decode(&b); err == nil {
			sim.Breakdown = &b
		}
	}
	if err := json.Unmarshal([]byte(sim.CommittedProposalIdsJSON), &sim.CommittedProposalIDs); err != nil || sim.CommittedProposalIDs == nil {
		sim.CommittedProposalIDs = []string{}
	}
	sim.CreatedAt = sim.CreatedAt.UTC()
	sim.UpdatedAt = sim.UpdatedAt.UTC()
	return sim, nil
}

func nullableStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
