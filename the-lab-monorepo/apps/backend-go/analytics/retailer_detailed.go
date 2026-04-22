package analytics

import (
	"context"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"backend-go/auth"
)

// ═══════════════════════════════════════════════════════════════════════════════
// RETAILER DETAILED ANALYTICS — Rich data for native retailer mobile apps
//
// GET /v1/retailer/analytics/detailed?from=&to=
// Auth: RETAILER
// ═══════════════════════════════════════════════════════════════════════════════

type RetailerDayExpense struct {
	Date  string `json:"date"`
	Total int64  `json:"total"`
	Count int64  `json:"count"`
}

type OrderStateCount struct {
	State string `json:"state"`
	Count int64  `json:"count"`
}

type CategorySpend struct {
	Category string `json:"category"`
	Total    int64  `json:"total"`
	Count    int64  `json:"count"`
}

type DayOfWeekPattern struct {
	Weekday string `json:"weekday"`
	Avg     int64  `json:"avg"`
	Count   int64  `json:"count"`
}

type RetailerDetailedResponse struct {
	DailySpending     []RetailerDayExpense `json:"daily_spending"`
	OrdersByState     []OrderStateCount    `json:"orders_by_state"`
	CategoryBreakdown []CategorySpend      `json:"category_breakdown"`
	WeekdayPattern    []DayOfWeekPattern   `json:"weekday_pattern"`
	TotalSpent        int64                `json:"total_spent"`
	TotalOrders       int64                `json:"total_orders"`
	AvgOrderValue     int64                `json:"avg_order_value"`
}

func HandleRetailerDetailedAnalytics(client *spanner.Client) http.HandlerFunc {
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

		retailerID := claims.UserID
		dr := ParseDateRange(r, 30)
		params := map[string]interface{}{
			"retailerId": retailerID,
			"_from":      dr.From,
			"_to":        dr.To,
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		resp := RetailerDetailedResponse{}
		staleness := spanner.ExactStaleness(10 * time.Second)

		// 1. Daily spending time-series
		dailySql := `
			SELECT
				FORMAT_TIMESTAMP('%Y-%m-%d', o.CreatedAt) as Day,
				CAST(SUM(o.Amount) AS INT64) as Total,
				COUNT(*) as Cnt
			FROM Orders o
			WHERE o.RetailerId = @retailerId
			AND o.State IN ('COMPLETED', 'ARRIVED', 'IN_TRANSIT')
			AND o.CreatedAt >= @_from AND o.CreatedAt <= @_to
			GROUP BY Day
			ORDER BY Day`

		dailyIter := client.Single().WithTimestampBound(staleness).Query(ctx, spanner.Statement{SQL: dailySql, Params: params})
		defer dailyIter.Stop()

		for {
			row, err := dailyIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Daily spending query fault", http.StatusInternalServerError)
				return
			}
			var day spanner.NullString
			var total, count spanner.NullInt64
			if err := row.Columns(&day, &total, &count); err != nil {
				continue
			}
			resp.DailySpending = append(resp.DailySpending, RetailerDayExpense{
				Date:  day.StringVal,
				Total: total.Int64,
				Count: count.Int64,
			})
			resp.TotalSpent += total.Int64
			resp.TotalOrders += count.Int64
		}

		if resp.TotalOrders > 0 {
			resp.AvgOrderValue = resp.TotalSpent / resp.TotalOrders
		}

		// 2. Orders by state (pie chart)
		stateSql := `
			SELECT o.State, COUNT(*) as Cnt
			FROM Orders o
			WHERE o.RetailerId = @retailerId
			AND o.CreatedAt >= @_from AND o.CreatedAt <= @_to
			GROUP BY o.State
			ORDER BY Cnt DESC`

		stateIter := client.Single().WithTimestampBound(staleness).Query(ctx, spanner.Statement{SQL: stateSql, Params: params})
		defer stateIter.Stop()

		for {
			row, err := stateIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var state spanner.NullString
			var count spanner.NullInt64
			if err := row.Columns(&state, &count); err != nil {
				continue
			}
			resp.OrdersByState = append(resp.OrdersByState, OrderStateCount{
				State: state.StringVal,
				Count: count.Int64,
			})
		}

		// 3. Category breakdown
		catSql := `
			SELECT
				COALESCE(p.Category, 'Uncategorized') as Category,
				CAST(SUM(oi.LineTotal) AS INT64) as Total,
				COUNT(DISTINCT o.OrderId) as Cnt
			FROM Orders o
			JOIN OrderItems oi ON o.OrderId = oi.OrderId
			JOIN Products p ON oi.ProductId = p.ProductId
			WHERE o.RetailerId = @retailerId
			AND o.State IN ('COMPLETED', 'ARRIVED', 'IN_TRANSIT')
			AND o.CreatedAt >= @_from AND o.CreatedAt <= @_to
			GROUP BY Category
			ORDER BY Total DESC
			LIMIT 10`

		catIter := client.Single().WithTimestampBound(staleness).Query(ctx, spanner.Statement{SQL: catSql, Params: params})
		defer catIter.Stop()

		for {
			row, err := catIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var cat spanner.NullString
			var total, count spanner.NullInt64
			if err := row.Columns(&cat, &total, &count); err != nil {
				continue
			}
			resp.CategoryBreakdown = append(resp.CategoryBreakdown, CategorySpend{
				Category: cat.StringVal,
				Total:    total.Int64,
				Count:    count.Int64,
			})
		}

		// 4. Weekday ordering pattern
		wdSql := `
			SELECT
				FORMAT_TIMESTAMP('%A', o.CreatedAt) as Weekday,
				CAST(AVG(o.Amount) AS INT64) as Avg,
				COUNT(*) as Cnt
			FROM Orders o
			WHERE o.RetailerId = @retailerId
			AND o.State IN ('COMPLETED', 'ARRIVED', 'IN_TRANSIT')
			AND o.CreatedAt >= @_from AND o.CreatedAt <= @_to
			GROUP BY Weekday
			ORDER BY Cnt DESC`

		wdIter := client.Single().WithTimestampBound(staleness).Query(ctx, spanner.Statement{SQL: wdSql, Params: params})
		defer wdIter.Stop()

		for {
			row, err := wdIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var wd spanner.NullString
			var avg, count spanner.NullInt64
			if err := row.Columns(&wd, &avg, &count); err != nil {
				continue
			}
			resp.WeekdayPattern = append(resp.WeekdayPattern, DayOfWeekPattern{
				Weekday: wd.StringVal,
				Avg:     avg.Int64,
				Count:   count.Int64,
			})
		}

		// Nil slice guards
		if resp.DailySpending == nil {
			resp.DailySpending = []RetailerDayExpense{}
		}
		if resp.OrdersByState == nil {
			resp.OrdersByState = []OrderStateCount{}
		}
		if resp.CategoryBreakdown == nil {
			resp.CategoryBreakdown = []CategorySpend{}
		}
		if resp.WeekdayPattern == nil {
			resp.WeekdayPattern = []DayOfWeekPattern{}
		}

		writeJSON(w, resp)
	}
}
