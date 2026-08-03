package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"google.golang.org/api/iterator"
)

// Service implements PX90 planning read models and sandbox APIs.
type Service struct {
	Spanner             *spanner.Client
	Cache               *cache.Cache
	Now                 func() time.Time
	TwinScenarioEnabled bool

	scenarioCache sync.Map // fallback when Redis unavailable
}

func NewService(client *spanner.Client) *Service {
	return &Service{
		Spanner: client,
		Now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// WithCache attaches a Redis cache for scenario and forecast aggregates.
func (s *Service) WithCache(c *cache.Cache) *Service {
	if s != nil {
		s.Cache = c
	}
	return s
}

// ScenarioInput is the what-if sandbox request body.
type ScenarioInput struct {
	FactoryDowntimeHours int     `json:"factory_downtime_hours"`
	DemandDeltaPct       float64 `json:"demand_delta_pct"`
	HorizonDays          int     `json:"horizon_days"`
}

// ScenarioResult is a read-only projection cached for 15 minutes.
type ScenarioResult struct {
	ScenarioID           string   `json:"scenario_id"`
	SupplierID           string   `json:"supplier_id"`
	SLARiskPct           float64  `json:"sla_risk_pct"`
	FleetVolume          int64    `json:"fleet_volume_orders"`
	StockoutSKUs         []string `json:"stockout_skus"`
	CapacityBreach       bool     `json:"capacity_breach"`
	CachedUntil          string   `json:"cached_until"`
	Mode                 string   `json:"mode,omitempty"`
	BaselineSLARiskPct   float64  `json:"baseline_sla_risk_pct,omitempty"`
	RevenueAtRiskMinor   int64    `json:"revenue_at_risk_minor,omitempty"`
}

type cachedScenario struct {
	result    ScenarioResult
	expiresAt time.Time
}

// RunScenario executes a read-only what-if projection.
func (s *Service) RunScenario(ctx context.Context, supplierID string, in ScenarioInput) (ScenarioResult, error) {
	if s == nil || s.Spanner == nil {
		return ScenarioResult{}, errors.New("planning unavailable")
	}
	if in.HorizonDays <= 0 {
		in.HorizonDays = 7
	}
	cacheKey := fmt.Sprintf("%d:%.2f:%d", in.FactoryDowntimeHours, in.DemandDeltaPct, in.HorizonDays)
	redisKey := ScenarioCacheKey(supplierID, cacheKey)
	if s.Cache != nil {
		if raw, found, err := s.Cache.Get(ctx, redisKey); err == nil && found {
			var result ScenarioResult
			if json.Unmarshal(raw, &result) == nil {
				return result, nil
			}
		}
	}
	if raw, ok := s.scenarioCache.Load(cacheKey); ok {
		entry := raw.(cachedScenario)
		if time.Now().Before(entry.expiresAt) {
			return entry.result, nil
		}
	}

	warehouseCount, orderVolume, criticalSKUs, deliveryVolume, err := s.scenarioSignals(ctx, supplierID, in.HorizonDays)
	if err != nil {
		return ScenarioResult{}, err
	}

	if s.TwinScenarioEnabled {
		snap, snapErr := LoadNetworkSnapshot(ctx, s.Spanner, supplierID)
		if snapErr == nil && !snap.TooLarge() {
			projected := ProjectSnapshot(snap, in)
			result := ScenarioResult{
				ScenarioID:         uuid.NewString(),
				SupplierID:         supplierID,
				SLARiskPct:         projected.SLARiskPct,
				BaselineSLARiskPct: projected.BaselineSLARiskPct,
				FleetVolume:        projected.FleetVolume,
				StockoutSKUs:       projected.StockoutSKUs,
				CapacityBreach:     projected.CapacityBreach,
				RevenueAtRiskMinor: projected.RevenueAtRiskMinor,
				Mode:               projected.Mode,
				CachedUntil:        s.Now().Add(15 * time.Minute).Format(time.RFC3339Nano),
			}
			s.storeScenarioCache(cacheKey, redisKey, result)
			return result, nil
		}
	}

	downtimeFactor := 1.0 - math.Min(float64(in.FactoryDowntimeHours)/168.0, 0.9)
	demandFactor := 1.0 + in.DemandDeltaPct/100.0
	slaRisk := math.Min(95, (float64(criticalSKUs)*12+float64(in.FactoryDowntimeHours)*2)*demandFactor)
	fleetVolume := int64(float64(orderVolume+deliveryVolume) * demandFactor)
	capacityBreach := in.FactoryDowntimeHours > 24 && warehouseCount > 0

	result := ScenarioResult{
		ScenarioID:     uuid.NewString(),
		SupplierID:     supplierID,
		SLARiskPct:     slaRisk * downtimeFactor,
		FleetVolume:    fleetVolume,
		StockoutSKUs:   s.projectStockouts(criticalSKUs),
		CapacityBreach: capacityBreach,
		Mode:           "heuristic",
		CachedUntil:    s.Now().Add(15 * time.Minute).Format(time.RFC3339Nano),
	}
	s.storeScenarioCache(cacheKey, redisKey, result)
	return result, nil
}

func (s *Service) storeScenarioCache(cacheKey, redisKey string, result ScenarioResult) {
	ctx := context.Background()
	s.scenarioCache.Store(cacheKey, cachedScenario{result: result, expiresAt: s.Now().Add(15 * time.Minute)})
	if s.Cache != nil {
		if raw, err := json.Marshal(result); err == nil {
			_ = s.Cache.Set(ctx, redisKey, raw, time.Duration(scenarioCacheTTL)*time.Second)
		}
	}
}

func (s *Service) scenarioSignals(ctx context.Context, supplierID string, horizonDays int) (warehouseCount, orderVolume, criticalSKUs, deliveryVolume int, err error) {
	if horizonDays <= 0 {
		horizonDays = 7
	}
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM Warehouses WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return
	}
	_ = row.Columns(&warehouseCount)

	iter2 := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*) FROM Orders WHERE SupplierId = @sid
		      AND Status IN ('PENDING','LOADED','IN_TRANSIT')`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter2.Stop()
	row2, err := iter2.Next()
	if err != nil {
		return
	}
	_ = row2.Columns(&orderVolume)

	iter3 := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*) FROM ReplenishmentInsights
		      WHERE SupplierId = @sid AND UrgencyLevel = 'CRITICAL' AND Status = 'PENDING'`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter3.Stop()
	row3, err := iter3.Next()
	if err != nil {
		return
	}
	_ = row3.Columns(&criticalSKUs)

	iter4 := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*) FROM OrderDeliveryProofs
		      WHERE SupplierId = @sid
		        AND CapturedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL @days DAY)`,
		Params: map[string]any{"sid": supplierID, "days": horizonDays},
	})
	defer iter4.Stop()
	row4, err := iter4.Next()
	if err != nil {
		return
	}
	_ = row4.Columns(&deliveryVolume)
	return
}

func (s *Service) projectStockouts(critical int) []string {
	out := make([]string, 0, critical)
	for i := 0; i < critical && i < 10; i++ {
		out = append(out, fmt.Sprintf("sku-projection-%d", i+1))
	}
	return out
}

// SAndOPSnapshot compares factory capacity vs warehouse throughput (7-day horizon).
type SAndOPSnapshot struct {
	SupplierID           string  `json:"supplier_id"`
	HorizonDays          int     `json:"horizon_days"`
	FactoryCapacityUnits int64   `json:"factory_capacity_units"`
	WarehouseInboundCap  int64   `json:"warehouse_inbound_cap_units"`
	WarehouseOutboundCap int64   `json:"warehouse_outbound_cap_units"`
	UtilizationPct       float64 `json:"utilization_pct"`
	CapacityAlert        bool    `json:"capacity_alert"`
}

// GetSAndOP returns lightweight S&OP capacity comparison.
func (s *Service) GetSAndOP(ctx context.Context, supplierID string) (SAndOPSnapshot, error) {
	out := SAndOPSnapshot{SupplierID: supplierID, HorizonDays: 7}
	if s == nil || s.Spanner == nil {
		return out, errors.New("planning unavailable")
	}
	var factoryCount int64
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM Factories WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	if row, err := iter.Next(); err == nil {
		_ = row.Columns(&factoryCount)
	}
	var whCount int64
	iter2 := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM Warehouses WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter2.Stop()
	if row, err := iter2.Next(); err == nil {
		_ = row.Columns(&whCount)
	}
	out.FactoryCapacityUnits = factoryCount * 700 * 7
	out.WarehouseInboundCap = whCount * 500 * 7
	out.WarehouseOutboundCap = whCount * 450 * 7
	if out.WarehouseInboundCap > 0 {
		out.UtilizationPct = float64(out.FactoryCapacityUnits) / float64(out.WarehouseInboundCap) * 100
	}
	out.CapacityAlert = out.FactoryCapacityUnits > out.WarehouseInboundCap
	return out, nil
}

// KGNode is an EKG-lite vertex.
type KGNode struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// KGEdge is an EKG-lite relationship.
type KGEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

// KnowledgeGraph is the supplier network read model.
type KnowledgeGraph struct {
	SupplierID string   `json:"supplier_id"`
	Nodes      []KGNode `json:"nodes"`
	Edges      []KGEdge `json:"edges"`
}

// GetKnowledgeGraph materializes topology + inventory + active orders as a graph.
func (s *Service) GetKnowledgeGraph(ctx context.Context, supplierID string) (KnowledgeGraph, error) {
	kg := KnowledgeGraph{SupplierID: supplierID}
	if s == nil || s.Spanner == nil {
		return kg, errors.New("planning unavailable")
	}
	kg.Nodes = append(kg.Nodes, KGNode{ID: supplierID, Type: "supplier"})

	addNodes := func(nodeType, table, idCol, nameCol string) error {
		sql := fmt.Sprintf(`SELECT %s, COALESCE(%s, %s) FROM %s WHERE SupplierId = @sid`, idCol, nameCol, idCol, table)
		iter := s.Spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: map[string]any{"sid": supplierID}})
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return err
			}
			var id, name string
			if err := row.Columns(&id, &name); err != nil {
				continue
			}
			kg.Nodes = append(kg.Nodes, KGNode{ID: id, Type: nodeType, Name: name})
			kg.Edges = append(kg.Edges, KGEdge{From: id, To: supplierID, Relation: "owned_by"})
		}
		return nil
	}
	_ = addNodes("factory", "Factories", "FactoryId", "Name")
	_ = addNodes("warehouse", "Warehouses", "WarehouseId", "Name")
	_ = addNodes("driver", "Drivers", "DriverId", "Name")
	_ = addNodes("vehicle", "Vehicles", "VehicleId", "Label")
	_ = s.addDriverVehicleEdges(ctx, supplierID, &kg)
	if err := s.addRetailerNodes(ctx, supplierID, &kg); err != nil {
		return kg, err
	}
	if err := s.addActiveOrderNodes(ctx, supplierID, &kg); err != nil {
		return kg, err
	}

	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT DISTINCT ProductId FROM Products WHERE SupplierId = @sid AND IsActive = true LIMIT 200`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			break
		}
		var pid string
		if err := row.Columns(&pid); err == nil {
			kg.Nodes = append(kg.Nodes, KGNode{ID: pid, Type: "sku"})
			kg.Edges = append(kg.Edges, KGEdge{From: supplierID, To: pid, Relation: "catalogs"})
		}
	}
	return kg, nil
}

func (s *Service) addDriverVehicleEdges(ctx context.Context, supplierID string, kg *KnowledgeGraph) error {
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT DriverId, VehicleId FROM Drivers
		      WHERE SupplierId = @sid AND IsActive = true AND VehicleId IS NOT NULL
		      LIMIT 200`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return nil
		}
		if err != nil {
			return err
		}
		var driverID, vehicleID string
		if err := row.Columns(&driverID, &vehicleID); err != nil {
			continue
		}
		if strings.TrimSpace(vehicleID) == "" {
			continue
		}
		kg.Edges = append(kg.Edges, KGEdge{From: driverID, To: vehicleID, Relation: "operates"})
	}
}

func (s *Service) addRetailerNodes(ctx context.Context, supplierID string, kg *KnowledgeGraph) error {
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT DISTINCT r.RetailerId, COALESCE(r.Name, r.RetailerId)
		      FROM Orders o
		      JOIN Retailers r ON o.RetailerId = r.RetailerId
		      WHERE o.SupplierId = @sid
		      LIMIT 200`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return nil
		}
		if err != nil {
			return err
		}
		var id, name string
		if err := row.Columns(&id, &name); err != nil {
			continue
		}
		kg.Nodes = append(kg.Nodes, KGNode{ID: id, Type: "retailer", Name: name})
		kg.Edges = append(kg.Edges, KGEdge{From: id, To: supplierID, Relation: "orders_from"})
	}
}

func (s *Service) addActiveOrderNodes(ctx context.Context, supplierID string, kg *KnowledgeGraph) error {
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, RetailerId, COALESCE(WarehouseId, ''), COALESCE(DriverId, ''), COALESCE(VehicleId, ''), Status
		      FROM Orders
		      WHERE SupplierId = @sid
		        AND Status IN ('PENDING','LOADED','IN_TRANSIT','ARRIVED')
		      ORDER BY CreatedAt DESC
		      LIMIT 100`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return nil
		}
		if err != nil {
			return err
		}
		var orderID, retailerID, warehouseID, driverID, vehicleID, status string
		if err := row.Columns(&orderID, &retailerID, &warehouseID, &driverID, &vehicleID, &status); err != nil {
			continue
		}
		kg.Nodes = append(kg.Nodes, KGNode{ID: orderID, Type: "order", Name: status})
		if retailerID != "" {
			kg.Edges = append(kg.Edges, KGEdge{From: orderID, To: retailerID, Relation: "delivers_to"})
		}
		if warehouseID != "" {
			kg.Edges = append(kg.Edges, KGEdge{From: warehouseID, To: orderID, Relation: "fulfills"})
		}
		if driverID != "" {
			kg.Edges = append(kg.Edges, KGEdge{From: driverID, To: orderID, Relation: "assigned"})
		}
		if vehicleID != "" {
			kg.Edges = append(kg.Edges, KGEdge{From: vehicleID, To: orderID, Relation: "carries"})
		}
		kg.Edges = append(kg.Edges, KGEdge{From: supplierID, To: orderID, Relation: "owns"})
	}
}

// WriteDemandBaseline upserts one-number forecast rows for a supplier/day.
func (s *Service) WriteDemandBaseline(ctx context.Context, supplierID, warehouseID, productID string, qty int64, confidence float64, source string) error {
	if s == nil || s.Spanner == nil {
		return errors.New("planning unavailable")
	}
	err := WriteBaselineWithOutbox(ctx, s.Spanner, s.Now(), BaselineWriteInput{
		SupplierID:  supplierID,
		WarehouseID: warehouseID,
		ProductID:   productID,
		BaselineQty: qty,
		Confidence:  confidence,
		Source:      source,
	})
	if err == nil {
		InvalidateForecastAggCache(ctx, s.Cache, supplierID)
	}
	return err
}

func (s *Service) invalidateForecastAgg(ctx context.Context, supplierID string) {
	InvalidateForecastAggCache(ctx, s.Cache, supplierID)
}

// ReadDemandBaseline returns baseline rows for warehouse forecast (one-number path).
func (s *Service) ReadDemandBaseline(ctx context.Context, supplierID, warehouseID string, forecastDays int) (map[string]int64, error) {
	out := map[string]int64{}
	if s == nil || s.Spanner == nil {
		return out, nil
	}
	if forecastDays <= 0 {
		forecastDays = 7
	}
	end := s.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -forecastDays+1)
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, SUM(BaselineQty) FROM DemandForecastBaseline
		      WHERE SupplierId = @sid AND WarehouseId = @wh
		        AND ForecastDate BETWEEN @start AND @end
		      GROUP BY ProductId`,
		Params: map[string]any{"sid": supplierID, "wh": warehouseID, "start": start, "end": end},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var pid string
		var qty int64
		if err := row.Columns(&pid, &qty); err == nil {
			out[pid] = qty
		}
	}
	return out, nil
}

// ZoneOverrideInput is the control-tower polygon mutation body.
type ZoneOverrideInput struct {
	WarehouseID      string          `json:"warehouse_id"`
	Action           string          `json:"action"`
	PolygonGeoJSON   json.RawMessage `json:"polygon_geojson"`
	TTLSeconds       int64           `json:"ttl_seconds"`
}

// ZoneOverrideRow is a persisted override.
type ZoneOverrideRow struct {
	OverrideID     string `json:"override_id"`
	SupplierID     string `json:"supplier_id"`
	WarehouseID    string `json:"warehouse_id,omitempty"`
	Action         string `json:"action"`
	PolygonGeoJSON string `json:"polygon_geojson"`
	TTLExpiresAt   string `json:"ttl_expires_at"`
	IsActive       bool   `json:"is_active"`
}

// CreateZoneOverride persists a control-tower polygon action.
func (s *Service) CreateZoneOverride(ctx context.Context, supplierID, createdBy string, in ZoneOverrideInput) (ZoneOverrideRow, error) {
	if s == nil || s.Spanner == nil {
		return ZoneOverrideRow{}, errors.New("planning unavailable")
	}
	action := strings.ToUpper(strings.TrimSpace(in.Action))
	if action == "" {
		action = "REROUTE"
	}
	ttl := in.TTLSeconds
	if ttl <= 0 {
		ttl = 3600
	}
	expires := s.Now().Add(time.Duration(ttl) * time.Second)
	overrideID := uuid.NewString()
	polygon := string(in.PolygonGeoJSON)
	row := ZoneOverrideRow{
		OverrideID:     overrideID,
		SupplierID:     supplierID,
		WarehouseID:    strings.TrimSpace(in.WarehouseID),
		Action:         action,
		PolygonGeoJSON: polygon,
		TTLExpiresAt:   expires.Format(time.RFC3339Nano),
		IsActive:       true,
	}
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("ControlTowerZoneOverrides", map[string]any{
				"OverrideId":     overrideID,
				"SupplierId":     supplierID,
				"WarehouseId":    row.WarehouseID,
				"Action":         action,
				"PolygonGeoJSON": polygon,
				"TtlExpiresAt":   expires,
				"CreatedBy":      createdBy,
				"IsActive":       true,
				"CreatedAt":      spanner.CommitTimestamp,
			}),
		}
		payload := events.PlanningEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventDispatchZoneOverride,
				Timestamp: s.Now().Format(time.RFC3339Nano),
			},
			SupplierID:  supplierID,
			WarehouseID: row.WarehouseID,
			OverrideID:  overrideID,
			Action:      action,
			Polygon:     polygon,
			TTLSeconds:  ttl,
		}
		buf := &planningTxnBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregatePlanning, supplierID, events.TopicMain, payload); emitErr != nil {
			return emitErr
		}
		mutations = append(mutations, planningOutboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	return row, err
}

// ListActiveZoneOverrides returns non-expired overrides for a supplier.
func (s *Service) ListActiveZoneOverrides(ctx context.Context, supplierID string) ([]ZoneOverrideRow, error) {
	if s == nil || s.Spanner == nil {
		return nil, errors.New("planning unavailable")
	}
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OverrideId, SupplierId, COALESCE(WarehouseId,''), Action, PolygonGeoJSON, TtlExpiresAt, IsActive
		      FROM ControlTowerZoneOverrides
		      WHERE SupplierId = @sid AND IsActive = true AND TtlExpiresAt > CURRENT_TIMESTAMP()
		      ORDER BY CreatedAt DESC LIMIT 50`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	var rows []ZoneOverrideRow
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var r ZoneOverrideRow
		var expires time.Time
		if err := row.Columns(&r.OverrideID, &r.SupplierID, &r.WarehouseID, &r.Action, &r.PolygonGeoJSON, &expires, &r.IsActive); err != nil {
			continue
		}
		r.TTLExpiresAt = expires.UTC().Format(time.RFC3339Nano)
		rows = append(rows, r)
	}
	return rows, nil
}

type planningTxnBuffer struct {
	events []outbox.Event
}

func (b *planningTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func planningOutboxMutations(eventsList []outbox.Event) []*spanner.Mutation {
	mutations := make([]*spanner.Mutation, 0, len(eventsList))
	for _, event := range eventsList {
		createdAt := event.CreatedAt.UTC()
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		row := map[string]any{
			"EventId":       event.EventID,
			"AggregateType": event.AggregateType,
			"AggregateId":   event.AggregateID,
			"TopicName":     event.TopicName,
			"Payload":       event.Payload,
			"CreatedAt":     createdAt,
			"PublishedAt":   nil,
		}
		if event.PublishedAt != nil {
			row["PublishedAt"] = event.PublishedAt.UTC()
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
	}
	return mutations
}
