package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"backend-go/auth"
	"backend-go/proximity"
)

type GraphQueryMode string

const (
	GraphQueryModeProductLocationTime GraphQueryMode = "PRODUCT_LOCATION_TIME"
	GraphQueryModeSupplierTier        GraphQueryMode = "SUPPLIER_TIER"
	GraphQueryModeLaneCapacity        GraphQueryMode = "LANE_CAPACITY"

	defaultGraphQueryPageSize = 50
	maxGraphQueryPageSize     = 200
	defaultGraphWindowDays    = 30
)

type GraphQueryRequest struct {
	QueryMode   string `json:"query_mode"`
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
	WarehouseID string `json:"warehouse_id,omitempty"`
	FactoryID   string `json:"factory_id,omitempty"`
	SKUID       string `json:"sku_id,omitempty"`
	PageSize    int    `json:"page_size,omitempty"`
	Offset      int64  `json:"offset,omitempty"`
}

type GraphQueryNode struct {
	NodeID   string            `json:"node_id"`
	NodeType string            `json:"node_type"`
	Label    string            `json:"label"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type GraphQueryEdge struct {
	From     string            `json:"from"`
	To       string            `json:"to"`
	Relation string            `json:"relation"`
	Weight   float64           `json:"weight"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type GraphQueryRow struct {
	RowID       string             `json:"row_id"`
	Dimensions  map[string]string  `json:"dimensions"`
	Metrics     map[string]float64 `json:"metrics"`
	ExplainTags []string           `json:"explain_tags,omitempty"`
}

type GraphQueryPagination struct {
	PageSize   int    `json:"page_size"`
	Offset     int64  `json:"offset"`
	Returned   int    `json:"returned"`
	HasMore    bool   `json:"has_more"`
	NextOffset *int64 `json:"next_offset,omitempty"`
}

type GraphQueryExplainability struct {
	QueryMode        string            `json:"query_mode"`
	ScopeSupplierID  string            `json:"scope_supplier_id"`
	ScopeWarehouseID string            `json:"scope_warehouse_id,omitempty"`
	AppliedFilters   map[string]string `json:"applied_filters"`
	DataSources      []string          `json:"data_sources"`
	GeneratedAt      string            `json:"generated_at"`
}

type GraphQueryResult struct {
	QueryMode      string                   `json:"query_mode"`
	Nodes          []GraphQueryNode         `json:"nodes"`
	Edges          []GraphQueryEdge         `json:"edges"`
	Rows           []GraphQueryRow          `json:"rows"`
	Pagination     GraphQueryPagination     `json:"pagination"`
	Explainability GraphQueryExplainability `json:"explainability"`
}

type normalizedGraphQueryRequest struct {
	QueryMode   GraphQueryMode
	From        time.Time
	To          time.Time
	WarehouseID string
	FactoryID   string
	SKUID       string
	PageSize    int
	Offset      int64
}

type graphQueryExecution struct {
	Nodes       []GraphQueryNode
	Edges       []GraphQueryEdge
	Rows        []GraphQueryRow
	HasMore     bool
	DataSources []string
}

// HandleGraphQuery exposes a supplier-scoped graph-query analytics surface.
//
// POST /v1/supplier/analytics/graph/query
func HandleGraphQuery(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ws := extractScope(r)
		if claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req GraphQueryRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
			return
		}

		normalized, err := normalizeGraphQueryRequest(req)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"invalid_request","message":%q}`, err.Error()), http.StatusBadRequest)
			return
		}

		readClient := getReadClient(r.Context(), client, readRouter, ws)
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()

		execution, err := runGraphQueryMode(ctx, readClient, claims, ws, normalized)
		if err != nil {
			http.Error(w, `{"error":"graph_query_failed"}`, http.StatusInternalServerError)
			return
		}

		nextOffset := (*int64)(nil)
		if execution.HasMore {
			v := normalized.Offset + int64(len(execution.Rows))
			nextOffset = &v
		}

		result := GraphQueryResult{
			QueryMode: string(normalized.QueryMode),
			Nodes:     execution.Nodes,
			Edges:     execution.Edges,
			Rows:      execution.Rows,
			Pagination: GraphQueryPagination{
				PageSize:   normalized.PageSize,
				Offset:     normalized.Offset,
				Returned:   len(execution.Rows),
				HasMore:    execution.HasMore,
				NextOffset: nextOffset,
			},
			Explainability: GraphQueryExplainability{
				QueryMode:        string(normalized.QueryMode),
				ScopeSupplierID:  claims.ResolveSupplierID(),
				ScopeWarehouseID: graphQueryScopeWarehouseID(ws),
				AppliedFilters:   graphQueryFilters(normalized),
				DataSources:      execution.DataSources,
				GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
			},
		}

		writeJSON(w, result)
	}
}

func normalizeGraphQueryRequest(raw GraphQueryRequest) (normalizedGraphQueryRequest, error) {
	normalized := normalizedGraphQueryRequest{
		QueryMode:   GraphQueryMode(strings.ToUpper(strings.TrimSpace(raw.QueryMode))),
		WarehouseID: strings.TrimSpace(raw.WarehouseID),
		FactoryID:   strings.TrimSpace(raw.FactoryID),
		SKUID:       strings.TrimSpace(raw.SKUID),
		PageSize:    raw.PageSize,
		Offset:      raw.Offset,
	}

	if !isValidGraphQueryMode(normalized.QueryMode) {
		return normalizedGraphQueryRequest{}, fmt.Errorf("unsupported query_mode")
	}

	if normalized.PageSize <= 0 {
		normalized.PageSize = defaultGraphQueryPageSize
	}
	if normalized.PageSize > maxGraphQueryPageSize {
		return normalizedGraphQueryRequest{}, fmt.Errorf("page_size exceeds max %d", maxGraphQueryPageSize)
	}
	if normalized.Offset < 0 {
		return normalizedGraphQueryRequest{}, fmt.Errorf("offset must be >= 0")
	}

	now := time.Now().UTC()
	normalized.From = now.AddDate(0, 0, -defaultGraphWindowDays)
	normalized.To = now

	if strings.TrimSpace(raw.From) != "" {
		from, err := parseGraphQueryTime(raw.From, false)
		if err != nil {
			return normalizedGraphQueryRequest{}, fmt.Errorf("invalid from timestamp")
		}
		normalized.From = from
	}

	if strings.TrimSpace(raw.To) != "" {
		to, err := parseGraphQueryTime(raw.To, true)
		if err != nil {
			return normalizedGraphQueryRequest{}, fmt.Errorf("invalid to timestamp")
		}
		normalized.To = to
	}

	if normalized.From.After(normalized.To) {
		normalized.From, normalized.To = normalized.To, normalized.From
	}

	maxWindow := time.Duration(maxRangeDays) * 24 * time.Hour
	if normalized.To.Sub(normalized.From) > maxWindow {
		normalized.From = normalized.To.Add(-maxWindow)
	}

	return normalized, nil
}

func parseGraphQueryTime(raw string, endOfDay bool) (time.Time, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return time.Time{}, fmt.Errorf("empty value")
	}

	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UTC(), nil
	}

	if d, err := time.Parse("2006-01-02", v); err == nil {
		if endOfDay {
			return d.UTC().Add(24*time.Hour - time.Nanosecond), nil
		}
		return d.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("unsupported time format")
}

func isValidGraphQueryMode(mode GraphQueryMode) bool {
	switch mode {
	case GraphQueryModeProductLocationTime, GraphQueryModeSupplierTier, GraphQueryModeLaneCapacity:
		return true
	default:
		return false
	}
}

func runGraphQueryMode(
	ctx context.Context,
	client *spanner.Client,
	claims *auth.PegasusClaims,
	ws *auth.WarehouseScope,
	input normalizedGraphQueryRequest,
) (graphQueryExecution, error) {
	switch input.QueryMode {
	case GraphQueryModeProductLocationTime:
		return queryProductLocationTime(ctx, client, claims, ws, input)
	case GraphQueryModeSupplierTier:
		return querySupplierTier(ctx, client, claims, ws, input)
	case GraphQueryModeLaneCapacity:
		return queryLaneCapacity(ctx, client, claims, ws, input)
	default:
		return graphQueryExecution{}, fmt.Errorf("unsupported query mode")
	}
}

func queryProductLocationTime(
	ctx context.Context,
	client *spanner.Client,
	claims *auth.PegasusClaims,
	ws *auth.WarehouseScope,
	input normalizedGraphQueryRequest,
) (graphQueryExecution, error) {
	scopeClause, params := ApplyScopeFilter(claims, ws, "o.SupplierId", "o.WarehouseId")
	params["_from"] = input.From
	params["_to"] = input.To
	params["_pageLimit"] = int64(input.PageSize + 1)
	params["_offset"] = input.Offset
	params["_warehouseFilter"] = input.WarehouseID
	params["_skuFilter"] = input.SKUID

	sql := fmt.Sprintf(`
		SELECT
			FORMAT_TIMESTAMP('%%Y-%%m-%%d', o.CreatedAt) AS DayBucket,
			o.WarehouseId,
			COALESCE(w.Name, o.WarehouseId) AS WarehouseName,
			oli.SkuId,
			COALESCE(sp.Name, oli.SkuId) AS ProductName,
			CAST(SUM(oli.Quantity) AS INT64) AS TotalQuantity,
			COUNT(DISTINCT o.OrderId) AS OrderCount
		FROM Orders o
		JOIN OrderLineItems oli ON oli.OrderId = o.OrderId
		LEFT JOIN Warehouses w ON w.WarehouseId = o.WarehouseId
		LEFT JOIN SupplierProducts sp ON sp.SupplierId = o.SupplierId AND sp.SkuId = oli.SkuId
		WHERE o.CreatedAt >= @_from AND o.CreatedAt <= @_to
		  AND o.State IN ('LOADED', 'IN_TRANSIT', 'ARRIVED', 'COMPLETED')
		  AND (@_warehouseFilter = '' OR o.WarehouseId = @_warehouseFilter)
		  AND (@_skuFilter = '' OR oli.SkuId = @_skuFilter)
		  %s
		GROUP BY DayBucket, o.WarehouseId, WarehouseName, oli.SkuId, ProductName
		ORDER BY DayBucket DESC, o.WarehouseId, oli.SkuId
		LIMIT @_pageLimit OFFSET @_offset`, scopeClause)

	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	builder := newGraphQueryBuilder()
	hasMore := false
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return graphQueryExecution{}, err
		}

		if len(builder.rows) >= input.PageSize {
			hasMore = true
			break
		}

		var dayBucket, warehouseID, warehouseName, skuID, productName spanner.NullString
		var totalQty, orderCount spanner.NullInt64
		if err := row.Columns(&dayBucket, &warehouseID, &warehouseName, &skuID, &productName, &totalQty, &orderCount); err != nil {
			continue
		}

		productNodeID := "product:" + skuID.StringVal
		warehouseNodeID := "warehouse:" + warehouseID.StringVal
		dayNodeID := "day:" + dayBucket.StringVal

		builder.addNode(GraphQueryNode{
			NodeID:   productNodeID,
			NodeType: "PRODUCT",
			Label:    firstNonEmpty(productName.StringVal, skuID.StringVal),
			Metadata: map[string]string{"sku_id": skuID.StringVal},
		})
		builder.addNode(GraphQueryNode{
			NodeID:   warehouseNodeID,
			NodeType: "WAREHOUSE",
			Label:    firstNonEmpty(warehouseName.StringVal, warehouseID.StringVal),
			Metadata: map[string]string{"warehouse_id": warehouseID.StringVal},
		})
		builder.addNode(GraphQueryNode{
			NodeID:   dayNodeID,
			NodeType: "DAY_BUCKET",
			Label:    dayBucket.StringVal,
		})

		builder.addEdge(productNodeID, warehouseNodeID, "PRODUCT_FLOW", float64(totalQty.Int64), nil)
		builder.addEdge(warehouseNodeID, dayNodeID, "WAREHOUSE_ACTIVITY", float64(orderCount.Int64), nil)

		builder.addRow(GraphQueryRow{
			RowID: dayBucket.StringVal + "|" + warehouseID.StringVal + "|" + skuID.StringVal,
			Dimensions: map[string]string{
				"day":            dayBucket.StringVal,
				"warehouse_id":   warehouseID.StringVal,
				"warehouse_name": warehouseName.StringVal,
				"sku_id":         skuID.StringVal,
				"product_name":   productName.StringVal,
			},
			Metrics: map[string]float64{
				"quantity":    float64(totalQty.Int64),
				"order_count": float64(orderCount.Int64),
			},
			ExplainTags: []string{"order_line_items", "time_bucket"},
		})
	}

	return graphQueryExecution{
		Nodes:       builder.Nodes(),
		Edges:       builder.Edges(),
		Rows:        builder.rows,
		HasMore:     hasMore,
		DataSources: []string{"Orders", "OrderLineItems", "SupplierProducts", "Warehouses"},
	}, nil
}

func querySupplierTier(
	ctx context.Context,
	client *spanner.Client,
	claims *auth.PegasusClaims,
	ws *auth.WarehouseScope,
	input normalizedGraphQueryRequest,
) (graphQueryExecution, error) {
	scopeClause, params := ApplyScopeFilter(claims, ws, "sl.SupplierId", "sl.WarehouseId")
	params["_pageLimit"] = int64(input.PageSize + 1)
	params["_offset"] = input.Offset
	params["_warehouseFilter"] = input.WarehouseID
	params["_factoryFilter"] = input.FactoryID

	sql := fmt.Sprintf(`
		SELECT
			sl.LaneId,
			sl.SupplierId,
			sl.FactoryId,
			COALESCE(f.Name, sl.FactoryId) AS FactoryName,
			sl.WarehouseId,
			COALESCE(w.Name, sl.WarehouseId) AS WarehouseName,
			sl.Priority,
			sl.IsActive,
			sl.TransitTimeHours,
			sl.DampenedTransitHours
		FROM SupplyLanes sl
		LEFT JOIN Factories f ON f.FactoryId = sl.FactoryId AND f.SupplierId = sl.SupplierId
		LEFT JOIN Warehouses w ON w.WarehouseId = sl.WarehouseId AND w.SupplierId = sl.SupplierId
		WHERE (@_warehouseFilter = '' OR sl.WarehouseId = @_warehouseFilter)
		  AND (@_factoryFilter = '' OR sl.FactoryId = @_factoryFilter)
		  %s
		ORDER BY sl.Priority DESC, sl.LaneId
		LIMIT @_pageLimit OFFSET @_offset`, scopeClause)

	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	builder := newGraphQueryBuilder()
	hasMore := false
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return graphQueryExecution{}, err
		}

		if len(builder.rows) >= input.PageSize {
			hasMore = true
			break
		}

		var laneID, supplierID, factoryID, factoryName, warehouseID, warehouseName spanner.NullString
		var priority spanner.NullInt64
		var isActive spanner.NullBool
		var transitHours, dampenedHours spanner.NullFloat64
		if err := row.Columns(&laneID, &supplierID, &factoryID, &factoryName, &warehouseID, &warehouseName, &priority, &isActive, &transitHours, &dampenedHours); err != nil {
			continue
		}

		supplierNodeID := "supplier:" + supplierID.StringVal
		factoryNodeID := "factory:" + factoryID.StringVal
		warehouseNodeID := "warehouse:" + warehouseID.StringVal

		builder.addNode(GraphQueryNode{NodeID: supplierNodeID, NodeType: "SUPPLIER", Label: supplierID.StringVal})
		builder.addNode(GraphQueryNode{
			NodeID:   factoryNodeID,
			NodeType: "FACTORY",
			Label:    firstNonEmpty(factoryName.StringVal, factoryID.StringVal),
			Metadata: map[string]string{"factory_id": factoryID.StringVal},
		})
		builder.addNode(GraphQueryNode{
			NodeID:   warehouseNodeID,
			NodeType: "WAREHOUSE",
			Label:    firstNonEmpty(warehouseName.StringVal, warehouseID.StringVal),
			Metadata: map[string]string{"warehouse_id": warehouseID.StringVal},
		})

		builder.addEdge(supplierNodeID, factoryNodeID, "SUPPLIER_TIER_LINK", 1, nil)
		builder.addEdge(factoryNodeID, warehouseNodeID, "SUPPLY_LANE", float64(priority.Int64), map[string]string{
			"lane_id": laneID.StringVal,
		})

		builder.addRow(GraphQueryRow{
			RowID: laneID.StringVal,
			Dimensions: map[string]string{
				"lane_id":        laneID.StringVal,
				"supplier_id":    supplierID.StringVal,
				"factory_id":     factoryID.StringVal,
				"factory_name":   factoryName.StringVal,
				"warehouse_id":   warehouseID.StringVal,
				"warehouse_name": warehouseName.StringVal,
				"is_active":      strconv.FormatBool(isActive.Bool),
			},
			Metrics: map[string]float64{
				"priority":               float64(priority.Int64),
				"transit_time_hours":     transitHours.Float64,
				"dampened_transit_hours": dampenedHours.Float64,
			},
			ExplainTags: []string{"supply_lanes", "tier_graph"},
		})
	}

	return graphQueryExecution{
		Nodes:       builder.Nodes(),
		Edges:       builder.Edges(),
		Rows:        builder.rows,
		HasMore:     hasMore,
		DataSources: []string{"SupplyLanes", "Factories", "Warehouses"},
	}, nil
}

func queryLaneCapacity(
	ctx context.Context,
	client *spanner.Client,
	claims *auth.PegasusClaims,
	ws *auth.WarehouseScope,
	input normalizedGraphQueryRequest,
) (graphQueryExecution, error) {
	scopeClause, params := ApplyScopeFilter(claims, ws, "sl.SupplierId", "sl.WarehouseId")
	params["_pageLimit"] = int64(input.PageSize + 1)
	params["_offset"] = input.Offset
	params["_warehouseFilter"] = input.WarehouseID
	params["_factoryFilter"] = input.FactoryID

	sql := fmt.Sprintf(`
		SELECT
			sl.LaneId,
			sl.FactoryId,
			COALESCE(f.Name, sl.FactoryId) AS FactoryName,
			sl.WarehouseId,
			COALESCE(w.Name, sl.WarehouseId) AS WarehouseName,
			sl.FreightCostMinor,
			sl.CarbonScoreKg,
			sl.DirectDistanceKm,
			sl.TransitTimeHours,
			sl.DampenedTransitHours,
			f.DailyOutputCapacity,
			f.CurrentLoad
		FROM SupplyLanes sl
		LEFT JOIN Factories f ON f.FactoryId = sl.FactoryId AND f.SupplierId = sl.SupplierId
		LEFT JOIN Warehouses w ON w.WarehouseId = sl.WarehouseId AND w.SupplierId = sl.SupplierId
		WHERE (@_warehouseFilter = '' OR sl.WarehouseId = @_warehouseFilter)
		  AND (@_factoryFilter = '' OR sl.FactoryId = @_factoryFilter)
		  %s
		ORDER BY sl.Priority DESC, sl.LaneId
		LIMIT @_pageLimit OFFSET @_offset`, scopeClause)

	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	builder := newGraphQueryBuilder()
	hasMore := false
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return graphQueryExecution{}, err
		}

		if len(builder.rows) >= input.PageSize {
			hasMore = true
			break
		}

		var laneID, factoryID, factoryName, warehouseID, warehouseName spanner.NullString
		var freightCost, dailyCapacity, currentLoad spanner.NullInt64
		var carbonScore, distanceKm, transitHours, dampenedHours spanner.NullFloat64
		if err := row.Columns(
			&laneID,
			&factoryID,
			&factoryName,
			&warehouseID,
			&warehouseName,
			&freightCost,
			&carbonScore,
			&distanceKm,
			&transitHours,
			&dampenedHours,
			&dailyCapacity,
			&currentLoad,
		); err != nil {
			continue
		}

		capacityUtilization := 0.0
		if dailyCapacity.Int64 > 0 {
			capacityUtilization = float64(currentLoad.Int64) / float64(dailyCapacity.Int64)
		}

		factoryNodeID := "factory:" + factoryID.StringVal
		warehouseNodeID := "warehouse:" + warehouseID.StringVal

		builder.addNode(GraphQueryNode{
			NodeID:   factoryNodeID,
			NodeType: "FACTORY",
			Label:    firstNonEmpty(factoryName.StringVal, factoryID.StringVal),
			Metadata: map[string]string{
				"factory_id": factoryID.StringVal,
			},
		})
		builder.addNode(GraphQueryNode{
			NodeID:   warehouseNodeID,
			NodeType: "WAREHOUSE",
			Label:    firstNonEmpty(warehouseName.StringVal, warehouseID.StringVal),
			Metadata: map[string]string{
				"warehouse_id": warehouseID.StringVal,
			},
		})

		builder.addEdge(factoryNodeID, warehouseNodeID, "LANE_CAPACITY", capacityUtilization, map[string]string{
			"lane_id": laneID.StringVal,
		})

		builder.addRow(GraphQueryRow{
			RowID: laneID.StringVal,
			Dimensions: map[string]string{
				"lane_id":        laneID.StringVal,
				"factory_id":     factoryID.StringVal,
				"factory_name":   factoryName.StringVal,
				"warehouse_id":   warehouseID.StringVal,
				"warehouse_name": warehouseName.StringVal,
			},
			Metrics: map[string]float64{
				"freight_cost_minor":     float64(freightCost.Int64),
				"carbon_score_kg":        carbonScore.Float64,
				"direct_distance_km":     distanceKm.Float64,
				"transit_time_hours":     transitHours.Float64,
				"dampened_transit_hours": dampenedHours.Float64,
				"daily_output_capacity":  float64(dailyCapacity.Int64),
				"current_load":           float64(currentLoad.Int64),
				"capacity_utilization":   capacityUtilization,
			},
			ExplainTags: []string{"supply_lanes", "capacity_projection"},
		})
	}

	return graphQueryExecution{
		Nodes:       builder.Nodes(),
		Edges:       builder.Edges(),
		Rows:        builder.rows,
		HasMore:     hasMore,
		DataSources: []string{"SupplyLanes", "Factories", "Warehouses"},
	}, nil
}

type graphQueryBuilder struct {
	nodes map[string]GraphQueryNode
	edges map[string]GraphQueryEdge
	rows  []GraphQueryRow
}

func newGraphQueryBuilder() *graphQueryBuilder {
	return &graphQueryBuilder{
		nodes: make(map[string]GraphQueryNode),
		edges: make(map[string]GraphQueryEdge),
		rows:  make([]GraphQueryRow, 0),
	}
}

func (b *graphQueryBuilder) addNode(node GraphQueryNode) {
	if node.NodeID == "" {
		return
	}
	if _, exists := b.nodes[node.NodeID]; exists {
		return
	}
	b.nodes[node.NodeID] = node
}

func (b *graphQueryBuilder) addEdge(from, to, relation string, weight float64, metadata map[string]string) {
	if from == "" || to == "" || relation == "" {
		return
	}
	key := from + "|" + to + "|" + relation
	if edge, exists := b.edges[key]; exists {
		edge.Weight += weight
		b.edges[key] = edge
		return
	}
	b.edges[key] = GraphQueryEdge{
		From:     from,
		To:       to,
		Relation: relation,
		Weight:   weight,
		Metadata: metadata,
	}
}

func (b *graphQueryBuilder) addRow(row GraphQueryRow) {
	b.rows = append(b.rows, row)
}

func (b *graphQueryBuilder) Nodes() []GraphQueryNode {
	if len(b.nodes) == 0 {
		return []GraphQueryNode{}
	}
	out := make([]GraphQueryNode, 0, len(b.nodes))
	for _, node := range b.nodes {
		out = append(out, node)
	}
	return out
}

func (b *graphQueryBuilder) Edges() []GraphQueryEdge {
	if len(b.edges) == 0 {
		return []GraphQueryEdge{}
	}
	out := make([]GraphQueryEdge, 0, len(b.edges))
	for _, edge := range b.edges {
		out = append(out, edge)
	}
	return out
}

func graphQueryScopeWarehouseID(ws *auth.WarehouseScope) string {
	if ws == nil {
		return ""
	}
	return ws.WarehouseID
}

func graphQueryFilters(input normalizedGraphQueryRequest) map[string]string {
	filters := map[string]string{
		"from":      input.From.Format(time.RFC3339),
		"to":        input.To.Format(time.RFC3339),
		"page_size": strconv.Itoa(input.PageSize),
		"offset":    strconv.FormatInt(input.Offset, 10),
	}
	if input.WarehouseID != "" {
		filters["warehouse_id"] = input.WarehouseID
	}
	if input.FactoryID != "" {
		filters["factory_id"] = input.FactoryID
	}
	if input.SKUID != "" {
		filters["sku_id"] = input.SKUID
	}
	return filters
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
