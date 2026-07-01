package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"google.golang.org/api/iterator"

	"backend-go/auth"
)

// DemandBaselineRow mirrors pegasusX DemandForecastBaseline read shape.
type DemandBaselineRow struct {
	ForecastDate    string   `json:"forecast_date"`
	WarehouseID     string   `json:"warehouse_id"`
	ProductID       string   `json:"product_id"`
	ProductName     string   `json:"product_name,omitempty"`
	BaselineQty     int64    `json:"baseline_qty"`
	Confidence      float64  `json:"confidence"`
	Source          string   `json:"source"`
	LowUnits        *int64   `json:"low_units,omitempty"`
	HighUnits       *int64   `json:"high_units,omitempty"`
	ConfidencePct   *int64   `json:"confidence_pct,omitempty"`
	BaselineSource  string   `json:"baseline_source,omitempty"`
	BlockedReason   string   `json:"blocked_reason,omitempty"`
}

// TenantBaselineResponse is the federated baseline read model for one supplier tenant.
type TenantBaselineResponse struct {
	SupplierID           string              `json:"supplier_id"`
	Rows                 []DemandBaselineRow `json:"rows"`
	WorkspaceSnapshotJSON string             `json:"workspace_snapshot_json,omitempty"`
	GeneratedAt          string              `json:"generated_at"`
}

// MEIOWarehouseNode mirrors pegasusX supplier MEIO rollup node shape.
type MEIOWarehouseNode struct {
	WarehouseID  string  `json:"warehouse_id"`
	SKUCount     int     `json:"sku_count"`
	CriticalSKUs int     `json:"critical_skus"`
	WarningSKUs  int     `json:"warning_skus"`
	TotalStock   int64   `json:"total_stock"`
	AvgDaysCover float64 `json:"avg_days_cover"`
}

// TenantMEIOResponse is a read-only MEIO rollup stub (insights + recent planning events).
type TenantMEIOResponse struct {
	SupplierID              string              `json:"supplier_id"`
	WarehousesScanned       int                 `json:"warehouses_scanned"`
	SKUsAnalyzed            int                 `json:"skus_analyzed"`
	InsightsGenerated       int                 `json:"insights_generated"`
	TransferRecommendations int                 `json:"transfer_recommendations"`
	WarehouseBalances       []MEIOWarehouseNode `json:"warehouse_balances"`
	LastEventType           string              `json:"last_event_type,omitempty"`
	LastEventAt             string              `json:"last_event_at,omitempty"`
	StubSource              string              `json:"stub_source"`
	GeneratedAt             string              `json:"generated_at"`
}

// KGNode is an EKG-lite vertex (pegasusX planning knowledge-graph subset).
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

// TenantKnowledgeGraph is the federated EKG read model for one supplier tenant.
type TenantKnowledgeGraph struct {
	SupplierID string   `json:"supplier_id"`
	Nodes      []KGNode `json:"nodes"`
	Edges      []KGEdge `json:"edges"`
}

// ControlTowerOverrideRow is one active zone override row.
type ControlTowerOverrideRow struct {
	OverrideID     string `json:"override_id"`
	SupplierID     string `json:"supplier_id"`
	SupplierName   string `json:"supplier_name,omitempty"`
	WarehouseID    string `json:"warehouse_id,omitempty"`
	Action         string `json:"action"`
	PolygonGeoJSON string `json:"polygon_geojson"`
	TTLExpiresAt   string `json:"ttl_expires_at"`
	IsActive       bool   `json:"is_active"`
}

// ControlTowerRollupResponse aggregates DISPATCH_ZONE_OVERRIDE rows across tenants.
type ControlTowerRollupResponse struct {
	Overrides   []ControlTowerOverrideRow    `json:"overrides"`
	BySupplier  map[string]int               `json:"by_supplier"`
	ByAction    map[string]int               `json:"by_action"`
	GeneratedAt string                       `json:"generated_at"`
}

// HandleGetTenantBaseline serves GET /v1/admin/planning/tenants/{supplier_id}/baseline.
func HandleGetTenantBaseline(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		supplierID, err := tenantIDFromRequest(r)
		if err != nil {
			writePlanningJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()

		resp := TenantBaselineResponse{
			SupplierID:  supplierID,
			Rows:        []DemandBaselineRow{},
			GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano),
		}

		if snap, snapErr := loadWorkspaceSnapshot(ctx, client, supplierID); snapErr == nil {
			resp.WorkspaceSnapshotJSON = snap
		}

		rows, qErr := queryDemandBaseline(ctx, client, supplierID)
		if qErr != nil {
			if isSpannerTableMissing(qErr) {
				writePlanningJSON(w, http.StatusOK, resp)
				return
			}
			writePlanningJSON(w, http.StatusInternalServerError, map[string]string{"error": "baseline_query_failed"})
			return
		}
		resp.Rows = rows
		writePlanningJSON(w, http.StatusOK, resp)
	}
}

// HandleGetTenantMEIO serves GET /v1/admin/planning/tenants/{supplier_id}/meio.
func HandleGetTenantMEIO(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		supplierID, err := tenantIDFromRequest(r)
		if err != nil {
			writePlanningJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()

		resp, buildErr := buildMEIOStub(ctx, client, supplierID)
		if buildErr != nil && !isSpannerTableMissing(buildErr) {
			writePlanningJSON(w, http.StatusInternalServerError, map[string]string{"error": "meio_stub_failed"})
			return
		}
		writePlanningJSON(w, http.StatusOK, resp)
	}
}

// HandleGetTenantKnowledgeGraph serves GET /v1/admin/planning/tenants/{supplier_id}/knowledge-graph.
func HandleGetTenantKnowledgeGraph(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		supplierID, err := tenantIDFromRequest(r)
		if err != nil {
			writePlanningJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		kg, kgErr := materializeKnowledgeGraph(ctx, client, supplierID)
		if kgErr != nil {
			writePlanningJSON(w, http.StatusInternalServerError, map[string]string{"error": "knowledge_graph_failed"})
			return
		}
		writePlanningJSON(w, http.StatusOK, kg)
	}
}

// HandleGetControlTowerRollup serves GET /v1/admin/planning/control-tower/rollup (GLOBAL_ADMIN only).
func HandleGetControlTowerRollup(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if err := auth.RequireGlobalAdmin(w, claims); err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		rollup, err := queryControlTowerRollup(ctx, client)
		if err != nil && !isSpannerTableMissing(err) {
			writePlanningJSON(w, http.StatusInternalServerError, map[string]string{"error": "control_tower_rollup_failed"})
			return
		}
		if rollup == nil {
			rollup = &ControlTowerRollupResponse{
				Overrides:  []ControlTowerOverrideRow{},
				BySupplier: map[string]int{},
				ByAction:   map[string]int{},
				GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano),
			}
		}
		writePlanningJSON(w, http.StatusOK, rollup)
	}
}

func tenantIDFromRequest(r *http.Request) (string, error) {
	supplierID := strings.TrimSpace(chi.URLParam(r, "supplier_id"))
	if supplierID == "" {
		return "", fmt.Errorf("supplier_id_required")
	}
	if claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims); ok && claims != nil {
		if claims.Role == "SUPPLIER" {
			own := strings.TrimSpace(claims.ResolveSupplierID())
			if own != "" && own != supplierID {
				return "", fmt.Errorf("supplier_scope_mismatch")
			}
		}
	}
	return supplierID, nil
}

func loadWorkspaceSnapshot(ctx context.Context, client *spanner.Client, supplierID string) (string, error) {
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COALESCE(BaselineSnapshotJson, '')
		      FROM PlanningTenantWorkspace
		      WHERE SupplierId = @sid LIMIT 1`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var snap string
	if err := row.Columns(&snap); err != nil {
		return "", err
	}
	return strings.TrimSpace(snap), nil
}

func queryDemandBaseline(ctx context.Context, client *spanner.Client, supplierID string) ([]DemandBaselineRow, error) {
	day := time.Now().UTC().Truncate(24 * time.Hour)
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT b.ForecastDate, b.WarehouseId, b.ProductId, COALESCE(p.Name, b.ProductId),
		             b.BaselineQty, b.Confidence, b.Source,
		             b.LowUnits, b.HighUnits, b.ConfidencePct,
		             COALESCE(b.BaselineSource, ''), COALESCE(b.BlockedReason, '')
		      FROM DemandForecastBaseline b
		      LEFT JOIN SupplierProducts p ON b.ProductId = p.SkuId AND p.SupplierId = b.SupplierId
		      WHERE b.SupplierId = @sid AND b.ForecastDate >= @start
		      ORDER BY b.ForecastDate DESC, b.BaselineQty DESC
		      LIMIT 500`,
		Params: map[string]any{
			"sid":   supplierID,
			"start": day.AddDate(0, 0, -7),
		},
	})
	defer iter.Stop()

	var rows []DemandBaselineRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var rec DemandBaselineRow
		var forecastDate time.Time
		var low, high, confPct spanner.NullInt64
		if err := row.Columns(
			&forecastDate, &rec.WarehouseID, &rec.ProductID, &rec.ProductName,
			&rec.BaselineQty, &rec.Confidence, &rec.Source,
			&low, &high, &confPct, &rec.BaselineSource, &rec.BlockedReason,
		); err != nil {
			continue
		}
		rec.ForecastDate = forecastDate.Format("2006-01-02")
		if low.Valid {
			v := low.Int64
			rec.LowUnits = &v
		}
		if high.Valid {
			v := high.Int64
			rec.HighUnits = &v
		}
		if confPct.Valid {
			v := confPct.Int64
			rec.ConfidencePct = &v
		}
		rows = append(rows, rec)
	}
	if rows == nil {
		rows = []DemandBaselineRow{}
	}
	return rows, nil
}

func buildMEIOStub(ctx context.Context, client *spanner.Client, supplierID string) (TenantMEIOResponse, error) {
	resp := TenantMEIOResponse{
		SupplierID:        supplierID,
		WarehouseBalances: []MEIOWarehouseNode{},
		StubSource:        "replenishment_insights",
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339Nano),
	}

	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId,
		             COUNT(*) AS sku_count,
		             SUM(CASE WHEN UrgencyLevel = 'CRITICAL' THEN 1 ELSE 0 END) AS critical_skus,
		             SUM(CASE WHEN UrgencyLevel = 'WARNING' THEN 1 ELSE 0 END) AS warning_skus,
		             SUM(CurrentStock) AS total_stock,
		             AVG(TimeToEmptyDays) AS avg_days_cover
		      FROM ReplenishmentInsights
		      WHERE SupplierId = @sid AND Status = 'PENDING'
		      GROUP BY WarehouseId`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return resp, err
		}
		var node MEIOWarehouseNode
		var skuCount, critical, warning int64
		var totalStock int64
		var avgDays spanner.NullFloat64
		if err := row.Columns(&node.WarehouseID, &skuCount, &critical, &warning, &totalStock, &avgDays); err != nil {
			continue
		}
		node.SKUCount = int(skuCount)
		node.CriticalSKUs = int(critical)
		node.WarningSKUs = int(warning)
		node.TotalStock = totalStock
		if avgDays.Valid {
			node.AvgDaysCover = avgDays.Float64
		}
		resp.WarehouseBalances = append(resp.WarehouseBalances, node)
		resp.SKUsAnalyzed += node.SKUCount
		resp.InsightsGenerated += node.SKUCount
		if node.CriticalSKUs > 0 {
			resp.TransferRecommendations++
		}
	}
	resp.WarehousesScanned = len(resp.WarehouseBalances)

	eventType, eventAt, eventErr := latestPlanningEvent(ctx, client, supplierID)
	if eventErr == nil && eventType != "" {
		resp.LastEventType = eventType
		resp.LastEventAt = eventAt
		resp.StubSource = "replenishment_insights+outbox_events"
	}
	return resp, nil
}

func latestPlanningEvent(ctx context.Context, client *spanner.Client, supplierID string) (string, string, error) {
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT EventType, CreatedAt
		      FROM OutboxEvents
		      WHERE AggregateId = @sid
		        AND (EventType LIKE '%meio%' OR EventType LIKE '%DISPATCH_ZONE_OVERRIDE%'
		             OR EventType LIKE '%DEMAND_BASELINE%')
		      ORDER BY CreatedAt DESC
		      LIMIT 1`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	var eventType string
	var createdAt time.Time
	if err := row.Columns(&eventType, &createdAt); err != nil {
		return "", "", err
	}
	return eventType, createdAt.UTC().Format(time.RFC3339Nano), nil
}

func materializeKnowledgeGraph(ctx context.Context, client *spanner.Client, supplierID string) (TenantKnowledgeGraph, error) {
	kg := TenantKnowledgeGraph{
		SupplierID: supplierID,
		Nodes:      []KGNode{{ID: supplierID, Type: "supplier"}},
		Edges:      []KGEdge{},
	}

	addNodes := func(nodeType, table, idCol, nameCol string) error {
		sql := fmt.Sprintf(`SELECT %s, COALESCE(%s, %s) FROM %s WHERE SupplierId = @sid LIMIT 200`, idCol, nameCol, idCol, table)
		iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: map[string]any{"sid": supplierID}})
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				return nil
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
	}

	if err := addNodes("factory", "Factories", "FactoryId", "Name"); err != nil {
		return kg, err
	}
	if err := addNodes("warehouse", "Warehouses", "WarehouseId", "Name"); err != nil {
		return kg, err
	}
	if err := addNodes("driver", "Drivers", "DriverId", "Name"); err != nil {
		return kg, err
	}
	if err := addNodes("vehicle", "Vehicles", "VehicleId", "Label"); err != nil {
		return kg, err
	}
	if err := addDriverVehicleEdges(ctx, client, supplierID, &kg); err != nil {
		return kg, err
	}
	if err := addRetailerNodes(ctx, client, supplierID, &kg); err != nil {
		return kg, err
	}
	if err := addActiveOrderNodes(ctx, client, supplierID, &kg); err != nil {
		return kg, err
	}

	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT DISTINCT SkuId FROM SupplierProducts WHERE SupplierId = @sid AND IsActive = true LIMIT 200`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
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

func addDriverVehicleEdges(ctx context.Context, client *spanner.Client, supplierID string, kg *TenantKnowledgeGraph) error {
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT DriverId, VehicleId FROM Drivers
		      WHERE SupplierId = @sid AND IsActive = true AND VehicleId IS NOT NULL
		      LIMIT 200`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
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

func addRetailerNodes(ctx context.Context, client *spanner.Client, supplierID string, kg *TenantKnowledgeGraph) error {
	iter := client.Single().Query(ctx, spanner.Statement{
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
		if err == iterator.Done {
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

func addActiveOrderNodes(ctx context.Context, client *spanner.Client, supplierID string, kg *TenantKnowledgeGraph) error {
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, RetailerId, COALESCE(WarehouseId, ''), COALESCE(DriverId, ''), State
		      FROM Orders
		      WHERE SupplierId = @sid
		        AND State IN ('PENDING','LOADED','IN_TRANSIT','ARRIVED','DISPATCHED','ARRIVING')
		      ORDER BY CreatedAt DESC
		      LIMIT 100`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var orderID, retailerID, warehouseID, driverID, state string
		if err := row.Columns(&orderID, &retailerID, &warehouseID, &driverID, &state); err != nil {
			continue
		}
		kg.Nodes = append(kg.Nodes, KGNode{ID: orderID, Type: "order", Name: state})
		if retailerID != "" {
			kg.Edges = append(kg.Edges, KGEdge{From: orderID, To: retailerID, Relation: "delivers_to"})
		}
		if warehouseID != "" {
			kg.Edges = append(kg.Edges, KGEdge{From: warehouseID, To: orderID, Relation: "fulfills"})
		}
		if driverID != "" {
			kg.Edges = append(kg.Edges, KGEdge{From: driverID, To: orderID, Relation: "assigned"})
		}
		kg.Edges = append(kg.Edges, KGEdge{From: supplierID, To: orderID, Relation: "owns"})
	}
}

func queryControlTowerRollup(ctx context.Context, client *spanner.Client) (*ControlTowerRollupResponse, error) {
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT o.OverrideId, o.SupplierId, COALESCE(s.Name, o.SupplierId),
		             COALESCE(o.WarehouseId, ''), o.Action, o.PolygonGeoJSON, o.TtlExpiresAt, o.IsActive
		      FROM ControlTowerZoneOverrides o
		      LEFT JOIN Suppliers s ON o.SupplierId = s.SupplierId
		      WHERE o.IsActive = true AND o.TtlExpiresAt > CURRENT_TIMESTAMP()
		      ORDER BY o.CreatedAt DESC
		      LIMIT 500`,
	})
	defer iter.Stop()

	rollup := &ControlTowerRollupResponse{
		Overrides:   []ControlTowerOverrideRow{},
		BySupplier:  map[string]int{},
		ByAction:    map[string]int{},
		GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var rec ControlTowerOverrideRow
		var expires time.Time
		if err := row.Columns(
			&rec.OverrideID, &rec.SupplierID, &rec.SupplierName,
			&rec.WarehouseID, &rec.Action, &rec.PolygonGeoJSON, &expires, &rec.IsActive,
		); err != nil {
			continue
		}
		rec.TTLExpiresAt = expires.UTC().Format(time.RFC3339Nano)
		rec.Action = strings.ToUpper(strings.TrimSpace(rec.Action))
		if rec.Action == "" {
			rec.Action = "DISPATCH_ZONE_OVERRIDE"
		}
		rollup.Overrides = append(rollup.Overrides, rec)
		rollup.BySupplier[rec.SupplierID]++
		rollup.ByAction[rec.Action]++
	}
	return rollup, nil
}

func writePlanningJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func isSpannerTableMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "unknown table")
}
