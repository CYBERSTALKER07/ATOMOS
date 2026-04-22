package analytics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ═══════════════════════════════════════════════════════════════════════════════
// REVENUE ANALYTICS — Time-series revenue by payment gateway
//
// GET /v1/supplier/analytics/revenue?from=&to=&warehouse_id=
// Auth: SUPPLIER | ADMIN + RequireWarehouseScope
// ═══════════════════════════════════════════════════════════════════════════════

type RevenueDayBucket struct {
	Date  string `json:"date"`
	Total int64  `json:"total"`
	Payme int64  `json:"payme"`
	Click int64  `json:"click"`
	Card  int64  `json:"card"`
	Cash  int64  `json:"cash"`
}

type GatewayBreakdown struct {
	Gateway    string `json:"gateway"`
	Total      int64  `json:"total"`
	OrderCount int64  `json:"order_count"`
}

type RevenueResponse struct {
	TimeSeries       []RevenueDayBucket `json:"time_series"`
	GatewayBreakdown []GatewayBreakdown `json:"gateway_breakdown"`
}

func HandleRevenue(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ws := extractScope(r)
		scopeClause, scopeParams := ApplyScopeFilter(claims, ws, "o.SupplierId", "o.WarehouseId")
		dr := ParseDateRange(r, 30)
		scopeParams["_from"] = dr.From
		scopeParams["_to"] = dr.To

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		resp := RevenueResponse{}

		// 1. Daily revenue by gateway
		tsSql := fmt.Sprintf(`
			SELECT
				FORMAT_TIMESTAMP('%%Y-%%m-%%d', o.CreatedAt) as Day,
				CAST(SUM(o.Amount) AS INT64) as Total,
				CAST(SUM(CASE WHEN o.PaymentGateway = 'PAYME' THEN o.Amount ELSE 0 END) AS INT64) as Payme,
				CAST(SUM(CASE WHEN o.PaymentGateway = 'CLICK' THEN o.Amount ELSE 0 END) AS INT64) as Click,
				CAST(SUM(CASE WHEN o.PaymentGateway = 'CARD' THEN o.Amount ELSE 0 END) AS INT64) as Card,
				CAST(SUM(CASE WHEN o.PaymentGateway IN ('CASH', 'CASH_ON_DELIVERY') THEN o.Amount ELSE 0 END) AS INT64) as Cash
			FROM Orders o
			WHERE o.State IN ('COMPLETED', 'ARRIVED')
			AND o.CreatedAt >= @_from AND o.CreatedAt <= @_to
			%s
			GROUP BY Day
			ORDER BY Day`, scopeClause)

		tsIter := client.Single().Query(ctx, spanner.Statement{SQL: tsSql, Params: scopeParams})
		defer tsIter.Stop()

		for {
			row, err := tsIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Revenue time-series query fault", http.StatusInternalServerError)
				return
			}
			var b RevenueDayBucket
			var day spanner.NullString
			var total, payme, click, card, cash spanner.NullInt64
			if err := row.Columns(&day, &total, &payme, &click, &card, &cash); err != nil {
				continue
			}
			b.Date = day.StringVal
			b.Total = total.Int64
			b.Payme = payme.Int64
			b.Click = click.Int64
			b.Card = card.Int64
			b.Cash = cash.Int64
			resp.TimeSeries = append(resp.TimeSeries, b)
		}

		// 2. Gateway breakdown totals
		gbSql := fmt.Sprintf(`
			SELECT
				COALESCE(o.PaymentGateway, 'UNKNOWN') as Gateway,
				CAST(SUM(o.Amount) AS INT64) as Total,
				COUNT(*) as OrderCount
			FROM Orders o
			WHERE o.State IN ('COMPLETED', 'ARRIVED')
			AND o.CreatedAt >= @_from AND o.CreatedAt <= @_to
			%s
			GROUP BY Gateway
			ORDER BY Total DESC`, scopeClause)

		gbIter := client.Single().Query(ctx, spanner.Statement{SQL: gbSql, Params: scopeParams})
		defer gbIter.Stop()

		for {
			row, err := gbIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var gw spanner.NullString
			var total, count spanner.NullInt64
			if err := row.Columns(&gw, &total, &count); err != nil {
				continue
			}
			resp.GatewayBreakdown = append(resp.GatewayBreakdown, GatewayBreakdown{
				Gateway:    gw.StringVal,
				Total:      total.Int64,
				OrderCount: count.Int64,
			})
		}

		if resp.TimeSeries == nil {
			resp.TimeSeries = []RevenueDayBucket{}
		}
		if resp.GatewayBreakdown == nil {
			resp.GatewayBreakdown = []GatewayBreakdown{}
		}

		writeJSON(w, resp)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// TOP RETAILERS — Revenue ranking for supplier CRM
//
// GET /v1/supplier/analytics/top-retailers?from=&to=&sort_by=revenue|count
// Auth: SUPPLIER | ADMIN + RequireWarehouseScope
// ═══════════════════════════════════════════════════════════════════════════════

type TopRetailer struct {
	RetailerID    string `json:"retailer_id"`
	ShopName      string `json:"shop_name"`
	OrderCount    int64  `json:"order_count"`
	TotalRevenue  int64  `json:"total_revenue"`
	AvgOrderValue int64  `json:"avg_order_value"`
	LastOrderAt   string `json:"last_order_at"`
}

func HandleTopRetailers(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ws := extractScope(r)
		scopeClause, scopeParams := ApplyScopeFilter(claims, ws, "o.SupplierId", "o.WarehouseId")
		dr := ParseDateRange(r, 30)
		scopeParams["_from"] = dr.From
		scopeParams["_to"] = dr.To

		// Validate sort_by
		orderBy := "TotalRevenue DESC"
		if sortBy := r.URL.Query().Get("sort_by"); sortBy == "count" {
			orderBy = "OrderCount DESC"
		} else if sortBy == "recency" {
			orderBy = "LastOrderAt DESC"
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		sql := fmt.Sprintf(`
			SELECT
				r.RetailerId,
				COALESCE(r.ShopName, r.RetailerId) as ShopName,
				COUNT(*) as OrderCount,
				CAST(SUM(o.Amount) AS INT64) as TotalRevenue,
				CAST(AVG(o.Amount) AS INT64) as AvgOrderValue,
				FORMAT_TIMESTAMP('%%Y-%%m-%%dT%%H:%%M:%%SZ', MAX(o.CreatedAt)) as LastOrderAt
			FROM Orders o
			JOIN Retailers r ON o.RetailerId = r.RetailerId
			WHERE o.State IN ('COMPLETED', 'ARRIVED', 'IN_TRANSIT')
			AND o.CreatedAt >= @_from AND o.CreatedAt <= @_to
			%s
			GROUP BY r.RetailerId, r.ShopName
			ORDER BY %s
			LIMIT 20`, scopeClause, orderBy)

		iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: scopeParams})
		defer iter.Stop()

		var retailers []TopRetailer
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Top retailers query fault", http.StatusInternalServerError)
				return
			}
			var rid, name, lastOrder spanner.NullString
			var count, total, avg spanner.NullInt64
			if err := row.Columns(&rid, &name, &count, &total, &avg, &lastOrder); err != nil {
				continue
			}
			retailers = append(retailers, TopRetailer{
				RetailerID:    rid.StringVal,
				ShopName:      name.StringVal,
				OrderCount:    count.Int64,
				TotalRevenue:  total.Int64,
				AvgOrderValue: avg.Int64,
				LastOrderAt:   lastOrder.StringVal,
			})
		}
		if retailers == nil {
			retailers = []TopRetailer{}
		}

		writeJSON(w, retailers)
	}
}
