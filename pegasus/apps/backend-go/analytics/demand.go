package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/hotspot"
	"backend-go/proximity"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── FUTURE DEMAND ANALYTICS ─────────────────────────────────────────────────
// Surfaces AI-planned orders (Empathy Engine predictions) to suppliers so they
// can adjust manufacturing schedules proactively.

// DemandSummaryItem represents aggregated upcoming demand for a single SKU.
type DemandSummaryItem struct {
	SkuID         string `json:"sku_id"`
	ProductName   string `json:"product_name"`
	TotalQty      int64  `json:"total_qty"`
	RetailerCount int64  `json:"retailer_count"`
}

// DemandSummaryResponse is the payload for the "next 24h" demand card.
type DemandSummaryResponse struct {
	TotalRetailers  int64               `json:"total_retailers"`
	TotalPallets    int64               `json:"total_pallets"`
	TotalValue      int64               `json:"total_value"`
	PredictionCount int64               `json:"prediction_count"`
	Items           []DemandSummaryItem `json:"items"`
	GeneratedAt     string              `json:"generated_at"`
}

// HandleDemandToday returns AI-predicted demand for the next 24 hours
// for the authenticated supplier's products.
//
// Flow: AIPredictions (WAITING, TriggerDate <= now+24h)
//
//	→ joined to Orders created from past AI predictions (OrderSource = 'AI_PREDICTED')
//	  to estimate SKU breakdown via OrderLineItems → SupplierProducts.
//
// Since AIPredictions only stores RetailerId + PredictedAmount (no SKU detail),
// we approximate SKU breakdown from the retailer's historical AI orders for this supplier.
func HandleDemandToday(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierId := claims.ResolveSupplierID()

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		cutoff := time.Now().UTC().Add(24 * time.Hour)
		shards := hotspot.AllShards()

		// 1. Count upcoming predictions (next 24h)
		predStmt := spanner.Statement{
			SQL: `WITH scoped_predictions AS (
			        SELECT p.PredictedAmount
			        FROM AIPredictions@{FORCE_INDEX=Idx_AIPredictions_ByTriggerShardStatusDate} p
			        JOIN SupplierRetailerClients src ON src.RetailerId = p.RetailerId AND src.SupplierId = @sid
			        WHERE p.TriggerShard IN UNNEST(@shards)
			          AND p.Status = 'WAITING'
			          AND p.TriggerDate <= @cutoff
			        UNION ALL
			        SELECT p.PredictedAmount
			        FROM AIPredictions p
			        JOIN SupplierRetailerClients src ON src.RetailerId = p.RetailerId AND src.SupplierId = @sid
			        WHERE p.TriggerShard IS NULL
			          AND p.Status = 'WAITING'
			          AND p.TriggerDate <= @cutoff
			      )
			      SELECT COUNT(*) as cnt, COALESCE(SUM(PredictedAmount), 0)
			      FROM scoped_predictions`,
			Params: map[string]interface{}{
				"shards": shards,
				"cutoff": cutoff,
				"sid":    supplierId,
			},
		}
		readClient := getReadClient(r.Context(), client, readRouter, nil)
		predIter := readClient.Single().Query(ctx, predStmt)
		predRow, err := predIter.Next()
		predIter.Stop()
		var predCount, totalValue int64
		if err == nil {
			predRow.Columns(&predCount, &totalValue)
		}

		// 2. Get SKU-level demand estimation: look at what retailers with upcoming
		//    AI predictions have historically ordered from this supplier's catalog.
		//    This gives the supplier a realistic breakdown of expected SKUs.
		skuStmt := spanner.Statement{
			SQL: `WITH scoped_predictions AS (
			        SELECT p.PredictionId, p.RetailerId
			        FROM AIPredictions@{FORCE_INDEX=Idx_AIPredictions_ByTriggerShardStatusDate} p
			        WHERE p.TriggerShard IN UNNEST(@shards)
			          AND p.Status = 'WAITING'
			          AND p.TriggerDate <= @cutoff
			        UNION ALL
			        SELECT p.PredictionId, p.RetailerId
			        FROM AIPredictions p
			        WHERE p.TriggerShard IS NULL
			          AND p.Status = 'WAITING'
			          AND p.TriggerDate <= @cutoff
			      )
			      SELECT sp.SkuId, sp.Name,
			             SUM(oli.Quantity) as total_qty,
			             COUNT(DISTINCT o.RetailerId) as retailer_count
			      FROM scoped_predictions pred
			      JOIN Orders o ON o.RetailerId = pred.RetailerId
			                    AND o.OrderSource = 'AI_PREDICTED'
			      JOIN OrderLineItems oli ON oli.OrderId = o.OrderId
			      JOIN SupplierProducts sp ON oli.SkuId = sp.SkuId
			                               AND sp.SupplierId = @sid
			      GROUP BY sp.SkuId, sp.Name
			      ORDER BY total_qty DESC`,
			Params: map[string]interface{}{"sid": supplierId, "shards": shards, "cutoff": cutoff},
		}

		skuIter := readClient.Single().Query(ctx, skuStmt)
		defer skuIter.Stop()

		var items []DemandSummaryItem
		var totalPallets int64
		var totalRetailersSet = make(map[int64]bool)
		var maxRetailers int64

		for {
			row, err := skuIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Analytics computation fault", http.StatusInternalServerError)
				return
			}

			var skuId, name spanner.NullString
			var qty, retCount spanner.NullInt64
			if err := row.Columns(&skuId, &name, &qty, &retCount); err != nil {
				continue
			}

			item := DemandSummaryItem{}
			if skuId.Valid {
				item.SkuID = skuId.StringVal
			}
			if name.Valid {
				item.ProductName = name.StringVal
			}
			if qty.Valid {
				item.TotalQty = qty.Int64
				totalPallets += qty.Int64
			}
			if retCount.Valid {
				item.RetailerCount = retCount.Int64
				if retCount.Int64 > maxRetailers {
					maxRetailers = retCount.Int64
				}
				totalRetailersSet[retCount.Int64] = true
			}
			items = append(items, item)
		}

		if items == nil {
			items = []DemandSummaryItem{}
		}

		resp := DemandSummaryResponse{
			TotalRetailers:  maxRetailers,
			TotalPallets:    totalPallets,
			TotalValue:      totalValue,
			PredictionCount: predCount,
			Items:           items,
			GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ── DEMAND HISTORY (TIME-SERIES) ────────────────────────────────────────────

// DemandHistoryPoint is a single data point in the prediction vs actual chart.
type DemandHistoryPoint struct {
	Date         string `json:"date"`
	Predicted    int64  `json:"predicted"`
	Actual       int64  `json:"actual"`
	PredictedQty int64  `json:"predicted_qty"`
	ActualQty    int64  `json:"actual_qty"`
}

// DemandDetailRow is a single upcoming AI order detail (for the data grid).
type DemandDetailRow struct {
	Date         string `json:"date"`
	RetailerName string `json:"retailer_name"`
	SkuID        string `json:"sku_id"`
	ProductName  string `json:"product_name"`
	PredictedQty int64  `json:"predicted_qty"`
}

// DemandHistoryResponse is the full payload for the advanced analytics page.
type DemandHistoryResponse struct {
	TimeSeries []DemandHistoryPoint `json:"time_series"`
	Upcoming   []DemandDetailRow    `json:"upcoming"`
}

// HandleDemandHistory returns a time-series of predicted vs actual demand
// for the supplier, plus a detailed table of upcoming AI orders.
func HandleDemandHistory(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierId := claims.ResolveSupplierID()

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		shards := hotspot.AllShards()

		// 1. Time-series: Predicted volume vs actual ordered volume per day (last 30 days)
		tsStmt := spanner.Statement{
			SQL: `WITH scoped_predictions AS (
			        SELECT p.TriggerDate, p.PredictedAmount
			        FROM AIPredictions@{FORCE_INDEX=Idx_AIPredictions_ByTriggerShardStatusDate} p
			        JOIN SupplierRetailerClients src ON src.RetailerId = p.RetailerId AND src.SupplierId = @sid
			        WHERE p.TriggerShard IN UNNEST(@shards)
			          AND p.TriggerDate >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)
			        UNION ALL
			        SELECT p.TriggerDate, p.PredictedAmount
			        FROM AIPredictions p
			        JOIN SupplierRetailerClients src ON src.RetailerId = p.RetailerId AND src.SupplierId = @sid
			        WHERE p.TriggerShard IS NULL
			          AND p.TriggerDate >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)
			      ),
			      predicted AS (
			        SELECT CAST(pred.TriggerDate AS DATE) as dt,
			               SUM(pred.PredictedAmount) as predicted,
			               COUNT(*) as predicted_qty
			        FROM scoped_predictions pred
			        GROUP BY dt
			      ),
			      actual AS (
			        SELECT CAST(o.CreatedAt AS DATE) as dt,
			               SUM(o.Amount) as actual,
			               COUNT(DISTINCT oli.OrderId) as actual_qty
			        FROM Orders o
			        JOIN OrderLineItems oli ON oli.OrderId = o.OrderId
			        JOIN SupplierProducts sp ON oli.SkuId = sp.SkuId AND sp.SupplierId = @sid
			        WHERE o.CreatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)
			        GROUP BY dt
			      )
			      SELECT COALESCE(p.dt, a.dt) as day,
			             COALESCE(p.predicted, 0),
			             COALESCE(a.actual, 0),
			             COALESCE(p.predicted_qty, 0),
			             COALESCE(a.actual_qty, 0)
			      FROM predicted p
			      FULL OUTER JOIN actual a ON p.dt = a.dt
			      ORDER BY day DESC`,
			Params: map[string]interface{}{"sid": supplierId, "shards": shards},
		}

		readClient := getReadClient(r.Context(), client, readRouter, nil)
		tsIter := readClient.Single().Query(ctx, tsStmt)
		defer tsIter.Stop()

		var timeSeries []DemandHistoryPoint
		for {
			row, err := tsIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var day spanner.NullDate
			var pred, act, predQty, actQty spanner.NullInt64
			if err := row.Columns(&day, &pred, &act, &predQty, &actQty); err != nil {
				continue
			}
			pt := DemandHistoryPoint{}
			if day.Valid {
				pt.Date = day.Date.String()
			}
			if pred.Valid {
				pt.Predicted = pred.Int64
			}
			if act.Valid {
				pt.Actual = act.Int64
			}
			if predQty.Valid {
				pt.PredictedQty = predQty.Int64
			}
			if actQty.Valid {
				pt.ActualQty = actQty.Int64
			}
			timeSeries = append(timeSeries, pt)
		}

		// 2. Upcoming AI orders detail: WAITING predictions with SKU breakdown
		upStmt := spanner.Statement{
			SQL: `WITH scoped_predictions AS (
			        SELECT p.RetailerId, p.TriggerDate
			        FROM AIPredictions@{FORCE_INDEX=Idx_AIPredictions_ByTriggerShardStatusDate} p
			        WHERE p.TriggerShard IN UNNEST(@shards)
			          AND p.Status = 'WAITING'
			        UNION ALL
			        SELECT p.RetailerId, p.TriggerDate
			        FROM AIPredictions p
			        WHERE p.TriggerShard IS NULL
			          AND p.Status = 'WAITING'
			      )
			      SELECT CAST(pred.TriggerDate AS DATE) as dt,
			             COALESCE(ret.Name, pred.RetailerId) as retailer_name,
			             sp.SkuId, sp.Name,
			             SUM(oli.Quantity) as predicted_qty
			      FROM scoped_predictions pred
			      JOIN Orders o ON o.RetailerId = pred.RetailerId
			                    AND o.OrderSource = 'AI_PREDICTED'
			      JOIN OrderLineItems oli ON oli.OrderId = o.OrderId
			      JOIN SupplierProducts sp ON oli.SkuId = sp.SkuId
			                               AND sp.SupplierId = @sid
			      LEFT JOIN Retailers ret ON ret.RetailerId = pred.RetailerId
			      GROUP BY dt, retailer_name, sp.SkuId, sp.Name
			      ORDER BY dt ASC, predicted_qty DESC
			      LIMIT 200`,
			Params: map[string]interface{}{"sid": supplierId, "shards": shards},
		}

		upIter := readClient.Single().Query(ctx, upStmt)
		defer upIter.Stop()

		var upcoming []DemandDetailRow
		for {
			row, err := upIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var day spanner.NullDate
			var retName, skuId, prodName spanner.NullString
			var qty spanner.NullInt64
			if err := row.Columns(&day, &retName, &skuId, &prodName, &qty); err != nil {
				continue
			}
			d := DemandDetailRow{}
			if day.Valid {
				d.Date = day.Date.String()
			}
			if retName.Valid {
				d.RetailerName = retName.StringVal
			}
			if skuId.Valid {
				d.SkuID = skuId.StringVal
			}
			if prodName.Valid {
				d.ProductName = prodName.StringVal
			}
			if qty.Valid {
				d.PredictedQty = qty.Int64
			}
			upcoming = append(upcoming, d)
		}

		if timeSeries == nil {
			timeSeries = []DemandHistoryPoint{}
		}
		if upcoming == nil {
			upcoming = []DemandDetailRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DemandHistoryResponse{
			TimeSeries: timeSeries,
			Upcoming:   upcoming,
		})
	}
}
