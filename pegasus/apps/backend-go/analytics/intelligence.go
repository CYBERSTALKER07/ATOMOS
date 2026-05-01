package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"backend-go/auth"
	"backend-go/proximity"
)

// ═══════════════════════════════════════════════════════════════════════════════
// INTELLIGENCE VECTOR — ANALYTICS HANDLERS
//
// Five RBAC-scoped endpoints feeding the Bento Grid intelligence page.
// All handlers follow the same contract:
//   - Extract PegasusClaims + WarehouseScope from context
//   - Build Spanner query with ApplyScopeFilter
//   - Return { timestamp, data } JSON envelope
// ═══════════════════════════════════════════════════════════════════════════════

// ── Response Types ──────────────────────────────────────────────────────────

type TransitPoint struct {
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Count int64   `json:"count"`
	State string  `json:"state"`
}

type ThroughputBucket struct {
	Date           string `json:"date"`
	OrderCount     int64  `json:"order_count"`
	CompletedCount int64  `json:"completed_count"`
	CancelledCount int64  `json:"cancelled_count"`
}

type LoadBucket struct {
	VehicleClass string  `json:"vehicle_class"`
	VehicleCount int64   `json:"vehicle_count"`
	AvgLoadPct   float64 `json:"avg_load_pct"`
	MaxLoadPct   float64 `json:"max_load_pct"`
}

type NodeMetric struct {
	WarehouseID   string  `json:"warehouse_id"`
	WarehouseName string  `json:"warehouse_name"`
	OrderCount    int64   `json:"order_count"`
	AvgCycleMin   float64 `json:"avg_cycle_min"`
	OnTimeRate    float64 `json:"on_time_rate"`
}

type SLAEntry struct {
	Date        string `json:"date"`
	OnTime      int64  `json:"on_time"`
	Late        int64  `json:"late"`
	Breached    int64  `json:"breached"`
	TotalOrders int64  `json:"total_orders"`
}

// ── Helper: extract claims + scope ─────────────────────────────────────────

func extractScope(r *http.Request) (*auth.PegasusClaims, *auth.WarehouseScope) {
	claims, _ := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	ws := auth.GetWarehouseScope(r.Context())
	return claims, ws
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"data":      data,
	})
}

func getReadClient(ctx context.Context, primary *spanner.Client, readRouter proximity.ReadRouter, ws *auth.WarehouseScope) *spanner.Client {
	if ws != nil && ws.WarehouseID != "" {
		return proximity.WarehouseReadClient(ctx, primary, readRouter, ws.WarehouseID)
	}
	if readRouter != nil {
		return readRouter.Primary()
	}
	return primary
}

// ═══════════════════════════════════════════════════════════════════════════════
// 1. TRANSIT HEATMAP — Geo-density of active orders by state
// ═══════════════════════════════════════════════════════════════════════════════

func HandleTransitHeatmap(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ws := extractScope(r)
		scopeClause, scopeParams := ApplyScopeFilter(claims, ws, "o.SupplierId", "o.WarehouseId")

		// Optional date-range; default to all active orders (no time filter)
		dateClause := ""
		if r.URL.Query().Get("from") != "" || r.URL.Query().Get("to") != "" {
			dr := ParseDateRange(r, 30)
			scopeParams["_from"] = dr.From
			scopeParams["_to"] = dr.To
			dateClause = " AND o.CreatedAt >= @_from AND o.CreatedAt <= @_to"
		}

		sql := fmt.Sprintf(`
			SELECT r.Latitude, r.Longitude, o.State, COUNT(*) as Cnt
			FROM Orders o
			JOIN Retailers r ON o.RetailerId = r.RetailerId
			WHERE o.State IN ('PENDING', 'LOADED', 'IN_TRANSIT', 'ARRIVED')
			%s%s
			GROUP BY r.Latitude, r.Longitude, o.State`, dateClause, scopeClause)

		stmt := spanner.Statement{SQL: sql, Params: scopeParams}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		iter := getReadClient(r.Context(), client, readRouter, ws).Single().Query(ctx, stmt)
		defer iter.Stop()

		var points []TransitPoint
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Transit heatmap query fault", http.StatusInternalServerError)
				return
			}
			var lat, lng spanner.NullFloat64
			var state spanner.NullString
			var cnt spanner.NullInt64
			if err := row.Columns(&lat, &lng, &state, &cnt); err != nil {
				http.Error(w, "Data extraction fault", http.StatusInternalServerError)
				return
			}
			points = append(points, TransitPoint{
				Lat:   lat.Float64,
				Lng:   lng.Float64,
				State: state.StringVal,
				Count: cnt.Int64,
			})
		}
		if points == nil {
			points = []TransitPoint{}
		}
		writeJSON(w, points)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// 2. THROUGHPUT — Orders created/completed/cancelled per day (last 30 days)
// ═══════════════════════════════════════════════════════════════════════════════

func HandleThroughput(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ws := extractScope(r)
		scopeClause, scopeParams := ApplyScopeFilter(claims, ws, "o.SupplierId", "o.WarehouseId")
		dr := ParseDateRange(r, 30)
		scopeParams["_from"] = dr.From
		scopeParams["_to"] = dr.To

		sql := fmt.Sprintf(`
			SELECT
				FORMAT_TIMESTAMP('%%Y-%%m-%%d', o.CreatedAt) as Day,
				COUNT(*) as Total,
				COUNTIF(o.State = 'COMPLETED') as Completed,
				COUNTIF(o.State IN ('CANCELLED', 'CANCELLED_BY_ADMIN')) as Cancelled
			FROM Orders o
			WHERE o.CreatedAt >= @_from AND o.CreatedAt <= @_to
			%s
			GROUP BY Day
			ORDER BY Day`, scopeClause)

		stmt := spanner.Statement{SQL: sql, Params: scopeParams}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		iter := getReadClient(r.Context(), client, readRouter, ws).Single().Query(ctx, stmt)
		defer iter.Stop()

		var buckets []ThroughputBucket
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Throughput query fault", http.StatusInternalServerError)
				return
			}
			var day spanner.NullString
			var total, completed, cancelled spanner.NullInt64
			if err := row.Columns(&day, &total, &completed, &cancelled); err != nil {
				http.Error(w, "Data extraction fault", http.StatusInternalServerError)
				return
			}
			buckets = append(buckets, ThroughputBucket{
				Date:           day.StringVal,
				OrderCount:     total.Int64,
				CompletedCount: completed.Int64,
				CancelledCount: cancelled.Int64,
			})
		}
		if buckets == nil {
			buckets = []ThroughputBucket{}
		}
		writeJSON(w, buckets)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// 3. LOAD DISTRIBUTION — Fleet capacity utilization by vehicle class
// ═══════════════════════════════════════════════════════════════════════════════

func HandleLoadDistribution(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ws := extractScope(r)
		scopeClause, scopeParams := ApplyScopeFilter(claims, ws, "m.SupplierId", "m.WarehouseId")
		dr := ParseDateRange(r, 7)
		scopeParams["_from"] = dr.From
		scopeParams["_to"] = dr.To

		sql := fmt.Sprintf(`
			SELECT
				v.VehicleClass,
				COUNT(DISTINCT v.VehicleId) as VehicleCount,
				AVG(SAFE_DIVIDE(m.TotalVolumeVU, m.MaxVolumeVU) * 100) as AvgLoadPct,
				MAX(SAFE_DIVIDE(m.TotalVolumeVU, m.MaxVolumeVU) * 100) as MaxLoadPct
			FROM SupplierTruckManifests m
			JOIN Vehicles v ON m.TruckId = v.VehicleId
			WHERE m.State IN ('DISPATCHED', 'COMPLETED')
			AND m.CreatedAt >= @_from AND m.CreatedAt <= @_to
			%s
			GROUP BY v.VehicleClass`, scopeClause)

		stmt := spanner.Statement{SQL: sql, Params: scopeParams}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		iter := getReadClient(r.Context(), client, readRouter, ws).Single().Query(ctx, stmt)
		defer iter.Stop()

		var buckets []LoadBucket
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Load distribution query fault", http.StatusInternalServerError)
				return
			}
			var vc spanner.NullString
			var count spanner.NullInt64
			var avgPct, maxPct spanner.NullFloat64
			if err := row.Columns(&vc, &count, &avgPct, &maxPct); err != nil {
				http.Error(w, "Data extraction fault", http.StatusInternalServerError)
				return
			}
			buckets = append(buckets, LoadBucket{
				VehicleClass: vc.StringVal,
				VehicleCount: count.Int64,
				AvgLoadPct:   avgPct.Float64,
				MaxLoadPct:   maxPct.Float64,
			})
		}
		if buckets == nil {
			buckets = []LoadBucket{}
		}
		writeJSON(w, buckets)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// 4. NODE EFFICIENCY — Per-warehouse operational metrics
// ═══════════════════════════════════════════════════════════════════════════════

func HandleNodeEfficiency(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ws := extractScope(r)
		scopeClause, scopeParams := ApplyScopeFilter(claims, ws, "o.SupplierId", "o.WarehouseId")
		dr := ParseDateRange(r, 7)
		scopeParams["_from"] = dr.From
		scopeParams["_to"] = dr.To

		sql := fmt.Sprintf(`
			SELECT
				w.WarehouseId,
				w.Name as WarehouseName,
				COUNT(o.OrderId) as OrderCount,
				AVG(TIMESTAMP_DIFF(o.UpdatedAt, o.CreatedAt, MINUTE)) as AvgCycleMin,
				SAFE_DIVIDE(
					COUNTIF(o.State = 'COMPLETED'),
					NULLIF(COUNTIF(o.State IN ('COMPLETED', 'CANCELLED', 'CANCELLED_BY_ADMIN')), 0)
				) as OnTimeRate
			FROM Orders o
			JOIN Warehouses w ON o.WarehouseId = w.WarehouseId
			WHERE o.CreatedAt >= @_from AND o.CreatedAt <= @_to
			%s
			GROUP BY w.WarehouseId, w.Name`, scopeClause)

		stmt := spanner.Statement{SQL: sql, Params: scopeParams}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		iter := getReadClient(r.Context(), client, readRouter, ws).Single().Query(ctx, stmt)
		defer iter.Stop()

		var metrics []NodeMetric
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Node efficiency query fault", http.StatusInternalServerError)
				return
			}
			var whID, whName spanner.NullString
			var orderCount spanner.NullInt64
			var avgCycle, onTime spanner.NullFloat64
			if err := row.Columns(&whID, &whName, &orderCount, &avgCycle, &onTime); err != nil {
				http.Error(w, "Data extraction fault", http.StatusInternalServerError)
				return
			}
			metrics = append(metrics, NodeMetric{
				WarehouseID:   whID.StringVal,
				WarehouseName: whName.StringVal,
				OrderCount:    orderCount.Int64,
				AvgCycleMin:   avgCycle.Float64,
				OnTimeRate:    onTime.Float64,
			})
		}
		if metrics == nil {
			metrics = []NodeMetric{}
		}
		writeJSON(w, metrics)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// 5. SLA HEALTH — Delivery compliance by day (last 30 days)
// ═══════════════════════════════════════════════════════════════════════════════

func HandleSLAHealth(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ws := extractScope(r)
		scopeClause, scopeParams := ApplyScopeFilter(claims, ws, "o.SupplierId", "o.WarehouseId")
		dr := ParseDateRange(r, 30)
		scopeParams["_from"] = dr.From
		scopeParams["_to"] = dr.To

		// SLA definition: order completed within 24h of creation = on-time,
		// 24-48h = late, >48h = breached.
		sql := fmt.Sprintf(`
			SELECT
				FORMAT_TIMESTAMP('%%Y-%%m-%%d', o.CreatedAt) as Day,
				COUNTIF(o.State = 'COMPLETED'
					AND TIMESTAMP_DIFF(o.UpdatedAt, o.CreatedAt, HOUR) <= 24) as OnTime,
				COUNTIF(o.State = 'COMPLETED'
					AND TIMESTAMP_DIFF(o.UpdatedAt, o.CreatedAt, HOUR) > 24
					AND TIMESTAMP_DIFF(o.UpdatedAt, o.CreatedAt, HOUR) <= 48) as Late,
				COUNTIF(o.State = 'COMPLETED'
					AND TIMESTAMP_DIFF(o.UpdatedAt, o.CreatedAt, HOUR) > 48) as Breached,
				COUNT(*) as TotalOrders
			FROM Orders o
			WHERE o.CreatedAt >= @_from AND o.CreatedAt <= @_to
			AND o.State IN ('COMPLETED', 'CANCELLED', 'CANCELLED_BY_ADMIN')
			%s
			GROUP BY Day
			ORDER BY Day`, scopeClause)

		stmt := spanner.Statement{SQL: sql, Params: scopeParams}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		iter := getReadClient(r.Context(), client, readRouter, ws).Single().Query(ctx, stmt)
		defer iter.Stop()

		var entries []SLAEntry
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "SLA health query fault", http.StatusInternalServerError)
				return
			}
			var day spanner.NullString
			var onTime, late, breached, total spanner.NullInt64
			if err := row.Columns(&day, &onTime, &late, &breached, &total); err != nil {
				http.Error(w, "Data extraction fault", http.StatusInternalServerError)
				return
			}
			entries = append(entries, SLAEntry{
				Date:        day.StringVal,
				OnTime:      onTime.Int64,
				Late:        late.Int64,
				Breached:    breached.Int64,
				TotalOrders: total.Int64,
			})
		}
		if entries == nil {
			entries = []SLAEntry{}
		}
		writeJSON(w, entries)
	}
}
