package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
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
	Label                string  `json:"label,omitempty"`
}

const (
	ScenarioStatusDraft      = "DRAFT"
	ScenarioStatusPublished  = "PUBLISHED"
	ScenarioStatusSuperseded = "SUPERSEDED"
	ScenarioStatusRejected   = "REJECTED"
)

// ScenarioResult is a what-if projection persisted as DRAFT (also cached 15m).
type ScenarioResult struct {
	ScenarioID           string   `json:"scenario_id"`
	SupplierID           string   `json:"supplier_id"`
	Version              int64    `json:"version,omitempty"`
	Status               string   `json:"status,omitempty"`
	ParentScenarioID     string   `json:"parent_scenario_id,omitempty"`
	Label                string   `json:"label,omitempty"`
	HorizonDays          int      `json:"horizon_days,omitempty"`
	SLARiskPct           float64  `json:"sla_risk_pct"`
	FleetVolume          int64    `json:"fleet_volume_orders"`
	StockoutSKUs         []string `json:"stockout_skus"`
	CapacityBreach       bool     `json:"capacity_breach"`
	CachedUntil          string   `json:"cached_until,omitempty"`
	Mode                 string   `json:"mode,omitempty"`
	BaselineSLARiskPct   float64  `json:"baseline_sla_risk_pct,omitempty"`
	RevenueAtRiskMinor   int64    `json:"revenue_at_risk_minor,omitempty"`
	UnitValueSource      string   `json:"unit_value_source,omitempty"`
	FactoryDowntimeHours int      `json:"factory_downtime_hours,omitempty"`
	DemandDeltaPct       float64  `json:"demand_delta_pct,omitempty"`
	CreatedBy            string   `json:"created_by,omitempty"`
	PublishedBy          string   `json:"published_by,omitempty"`
	PublishedAt          string   `json:"published_at,omitempty"`
	UpdatedAt            string   `json:"updated_at,omitempty"`
}

type cachedScenario struct {
	result    ScenarioResult
	expiresAt time.Time
}

// RunScenario executes a what-if projection and persists a DRAFT row.
func (s *Service) RunScenario(ctx context.Context, supplierID, createdBy string, in ScenarioInput) (ScenarioResult, error) {
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
			if json.Unmarshal(raw, &result) == nil && result.ScenarioID != "" {
				return result, nil
			}
		}
	}
	if raw, ok := s.scenarioCache.Load(cacheKey); ok {
		entry := raw.(cachedScenario)
		if time.Now().Before(entry.expiresAt) && entry.result.ScenarioID != "" {
			return entry.result, nil
		}
	}

	result, err := s.computeScenario(ctx, supplierID, in)
	if err != nil {
		return ScenarioResult{}, err
	}
	result.ScenarioID = uuid.NewString()
	result.SupplierID = supplierID
	result.Version = 1
	result.Status = ScenarioStatusDraft
	result.Label = strings.TrimSpace(in.Label)
	result.HorizonDays = in.HorizonDays
	result.FactoryDowntimeHours = in.FactoryDowntimeHours
	result.DemandDeltaPct = in.DemandDeltaPct
	result.CreatedBy = strings.TrimSpace(createdBy)
	result.CachedUntil = s.Now().Add(15 * time.Minute).Format(time.RFC3339Nano)
	result.UpdatedAt = s.Now().Format(time.RFC3339Nano)

	if err := s.persistScenarioDraft(ctx, result, nil); err != nil {
		return ScenarioResult{}, fmt.Errorf("persist scenario: %w", err)
	}
	s.storeScenarioCache(cacheKey, redisKey, result)
	return result, nil
}

func (s *Service) computeScenario(ctx context.Context, supplierID string, in ScenarioInput) (ScenarioResult, error) {
	warehouseCount, orderVolume, criticalSKUs, deliveryVolume, err := s.scenarioSignals(ctx, supplierID, in.HorizonDays)
	if err != nil {
		return ScenarioResult{}, err
	}

	if s.TwinScenarioEnabled {
		snap, snapErr := LoadNetworkSnapshot(ctx, s.Spanner, supplierID)
		if snapErr == nil && !snap.TooLarge() {
			projected := ProjectSnapshot(snap, in)
			return ScenarioResult{
				SLARiskPct:         projected.SLARiskPct,
				BaselineSLARiskPct: projected.BaselineSLARiskPct,
				FleetVolume:        projected.FleetVolume,
				StockoutSKUs:       projected.StockoutSKUs,
				CapacityBreach:     projected.CapacityBreach,
				RevenueAtRiskMinor: projected.RevenueAtRiskMinor,
				UnitValueSource:    projected.UnitValueSource,
				Mode:               projected.Mode,
			}, nil
		}
	}

	downtimeFactor := 1.0 - math.Min(float64(in.FactoryDowntimeHours)/168.0, 0.9)
	demandFactor := 1.0 + in.DemandDeltaPct/100.0
	slaRisk := math.Min(95, (float64(criticalSKUs)*12+float64(in.FactoryDowntimeHours)*2)*demandFactor)
	fleetVolume := int64(float64(orderVolume+deliveryVolume) * demandFactor)
	capacityBreach := in.FactoryDowntimeHours > 24 && warehouseCount > 0
	stockouts, shortfalls := s.projectStockoutsWithQty(ctx, supplierID, criticalSKUs)
	skuSet := make(map[string]struct{}, len(stockouts))
	for _, sku := range stockouts {
		skuSet[sku] = struct{}{}
	}
	unitValues, _ := loadProductUnitValues(ctx, s.Spanner, supplierID, skuSet)
	rar, src := heuristicRevenueAtRisk(unitValues, stockouts, shortfalls)

	return ScenarioResult{
		SLARiskPct:         slaRisk * downtimeFactor,
		FleetVolume:        fleetVolume,
		StockoutSKUs:       stockouts,
		CapacityBreach:     capacityBreach,
		RevenueAtRiskMinor: rar,
		UnitValueSource:    src,
		Mode:               "heuristic",
	}, nil
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

func (s *Service) projectStockouts(ctx context.Context, supplierID string, critical int) []string {
	skus, _ := s.projectStockoutsWithQty(ctx, supplierID, critical)
	return skus
}

func (s *Service) projectStockoutsWithQty(ctx context.Context, supplierID string, critical int) ([]string, map[string]int64) {
	out := make([]string, 0, critical)
	qty := make(map[string]int64)
	if s == nil || s.Spanner == nil || critical <= 0 {
		return out, qty
	}
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, COALESCE(MAX(SuggestedQuantity), 1)
		      FROM ReplenishmentInsights
		      WHERE SupplierId = @sid AND UrgencyLevel = 'CRITICAL' AND Status = 'PENDING'
		      GROUP BY ProductId
		      LIMIT @lim`,
		Params: map[string]any{"sid": supplierID, "lim": int64(critical)},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var pid string
		var suggested int64
		if err := row.Columns(&pid, &suggested); err != nil || strings.TrimSpace(pid) == "" {
			continue
		}
		out = append(out, pid)
		if suggested < 1 {
			suggested = 1
		}
		qty[pid] = suggested
	}
	return out, qty
}

// SAndOPSnapshot compares factory capacity vs warehouse throughput over a
// configurable horizon (SOP_HORIZON_DAYS; default 7, typically 7/14/28).
type SAndOPSnapshot struct {
	SupplierID           string  `json:"supplier_id"`
	HorizonDays          int     `json:"horizon_days"`
	ProductionLineCount  int64   `json:"production_line_count"`
	FactoryCapacityUnits int64   `json:"factory_capacity_units"`
	ProjectedDemandUnits int64   `json:"projected_demand_units"`
	WarehouseInboundCap  int64   `json:"warehouse_inbound_cap_units"`
	WarehouseOutboundCap int64   `json:"warehouse_outbound_cap_units"`
	UtilizationPct       float64 `json:"utilization_pct"`
	CapacityAlert        bool    `json:"capacity_alert"`
	CapacityModel        string  `json:"capacity_model"`
	CapacitySource       string  `json:"capacity_source"`
}

// GetSAndOP returns S&OP capacity comparison. Factory capacity comes from a
// production-lines model (factories × SOP_LINES_PER_FACTORY × daily × horizon).
// Open warehouse supply-request ProjectedUnits populate ProjectedDemandUnits
// only — they never overwrite FactoryCapacityUnits.
func (s *Service) GetSAndOP(ctx context.Context, supplierID string) (SAndOPSnapshot, error) {
	out := SAndOPSnapshot{
		SupplierID:     supplierID,
		HorizonDays:    sopHorizonDays(),
		CapacityModel:  "production_lines",
		CapacitySource: "env_default",
	}
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

	// Calibrated daily throughput defaults (VU/day) — overridable via env for pilots.
	factoryDaily := envInt64("SOP_FACTORY_DAILY_UNITS", 700)
	whInDaily := envInt64("SOP_WAREHOUSE_INBOUND_DAILY_UNITS", 500)
	whOutDaily := envInt64("SOP_WAREHOUSE_OUTBOUND_DAILY_UNITS", 450)
	linesPerFactory := envInt64("SOP_LINES_PER_FACTORY", 1)
	out.ProductionLineCount = factoryCount * linesPerFactory
	var columnSum int64
	iterCap := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COALESCE(SUM(DailyOutputCapacity), 0) FROM Factories WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]any{"sid": supplierID},
	})
	if row, err := iterCap.Next(); err == nil {
		_ = row.Columns(&columnSum)
	}
	iterCap.Stop()
	out.FactoryCapacityUnits, out.CapacitySource = sopFactoryCapacity(columnSum, out.ProductionLineCount, factoryDaily, int64(out.HorizonDays))
	out.WarehouseInboundCap = whCount * whInDaily * int64(out.HorizonDays)
	out.WarehouseOutboundCap = whCount * whOutDaily * int64(out.HorizonDays)

	var projected int64
	iter3 := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COALESCE(SUM(ProjectedUnits), 0) FROM WarehouseSupplyRequests
		      WHERE SupplierId = @sid AND State IN ('OPEN','SUBMITTED','IN_PROGRESS','PENDING')`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter3.Stop()
	if row, err := iter3.Next(); err == nil {
		_ = row.Columns(&projected)
	}
	out.ProjectedDemandUnits = projected

	out.UtilizationPct, out.CapacityAlert = sandopUtilization(out.ProjectedDemandUnits, out.FactoryCapacityUnits, out.WarehouseInboundCap)
	return out, nil
}

// sandopUtilization returns demand/capacity utilization when projected demand is
// present; otherwise factory capacity vs warehouse inbound. Alert when demand
// exceeds factory capacity or warehouse inbound (or factory exceeds inbound when
// no demand signal).
func sopFactoryCapacity(columnSum, lineCount, envDaily, horizon int64) (units int64, source string) {
	if columnSum > 0 {
		return columnSum * horizon, "factories_column"
	}
	return lineCount * envDaily * horizon, "env_default"
}

func sandopUtilization(projected, factoryCap, whInbound int64) (pct float64, alert bool) {
	if projected > 0 {
		if factoryCap > 0 {
			pct = float64(projected) / float64(factoryCap) * 100
		}
		alert = projected > factoryCap || (whInbound > 0 && projected > whInbound)
		return pct, alert
	}
	if whInbound > 0 {
		pct = float64(factoryCap) / float64(whInbound) * 100
	}
	alert = factoryCap > whInbound
	return pct, alert
}

func sopHorizonDays() int {
	n := envInt64("SOP_HORIZON_DAYS", 7)
	switch n {
	case 7, 14, 28:
		return int(n)
	default:
		if n > 0 && n <= 90 {
			return int(n)
		}
		return 7
	}
}

func envInt64(key string, def int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
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
		SQL:    `SELECT DISTINCT ProductId FROM Products WHERE SupplierId = @sid AND IsActive = true LIMIT 200`,
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
	WarehouseID    string          `json:"warehouse_id"`
	Action         string          `json:"action"`
	PolygonGeoJSON json.RawMessage `json:"polygon_geojson"`
	TTLSeconds     int64           `json:"ttl_seconds"`
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
