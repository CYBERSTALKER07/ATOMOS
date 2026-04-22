package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── Retailer Expense Analytics ──

type MonthlyExpense struct {
	Month string `json:"month"`
	Total int64  `json:"total"`
}

type TopSupplier struct {
	SupplierID   string `json:"supplier_id"`
	SupplierName string `json:"supplier_name"`
	Total        int64  `json:"total"`
	OrderCount   int64  `json:"order_count"`
}

type TopProduct struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Total       int64  `json:"total"`
	Quantity    int64  `json:"quantity"`
}

type RetailerAnalyticsResponse struct {
	MonthlyExpenses []MonthlyExpense `json:"monthly_expenses"`
	TopSuppliers    []TopSupplier    `json:"top_suppliers"`
	TopProducts     []TopProduct     `json:"top_products"`
	TotalThisMonth  int64            `json:"total_this_month"`
	TotalLastMonth  int64            `json:"total_last_month"`
}

// HandleGetRetailerExpenses aggregates expense analytics for the authenticated retailer.
func HandleGetRetailerExpenses(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		retailerID := claims.UserID

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		now := time.Now()
		thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		lastMonthStart := thisMonthStart.AddDate(0, -1, 0)
		sixMonthsAgo := thisMonthStart.AddDate(0, -6, 0)

		resp := RetailerAnalyticsResponse{}

		// 1. Monthly expenses (last 6 months)
		monthlyStmt := spanner.Statement{
			SQL: `SELECT 
					FORMAT_TIMESTAMP('%Y-%m', o.CreatedAt) as month,
					CAST(SUM(o.Amount) AS INT64) as total
				FROM Orders o
				WHERE o.RetailerId = @retailerId
				  AND o.State IN ('COMPLETED', 'ARRIVED', 'IN_TRANSIT')
				  AND o.CreatedAt >= @since
				GROUP BY month
				ORDER BY month`,
			Params: map[string]interface{}{
				"retailerId": retailerID,
				"since":      sixMonthsAgo,
			},
		}
		monthlyIter := client.Single().WithTimestampBound(spanner.ExactStaleness(10*time.Second)).Query(ctx, monthlyStmt)
		defer monthlyIter.Stop()
		for {
			row, err := monthlyIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, `{"error":"query_failed"}`, http.StatusInternalServerError)
				return
			}
			var m MonthlyExpense
			if err := row.Columns(&m.Month, &m.Total); err != nil {
				continue
			}
			resp.MonthlyExpenses = append(resp.MonthlyExpenses, m)
		}

		// 2. This month vs last month totals
		totalsStmt := spanner.Statement{
			SQL: `SELECT 
					CAST(COALESCE(SUM(CASE WHEN o.CreatedAt >= @thisMonth THEN o.Amount ELSE 0 END), 0) AS INT64) as this_month,
					CAST(COALESCE(SUM(CASE WHEN o.CreatedAt >= @lastMonth AND o.CreatedAt < @thisMonth THEN o.Amount ELSE 0 END), 0) AS INT64) as last_month
				FROM Orders o
				WHERE o.RetailerId = @retailerId
				  AND o.State IN ('COMPLETED', 'ARRIVED', 'IN_TRANSIT')
				  AND o.CreatedAt >= @lastMonth`,
			Params: map[string]interface{}{
				"retailerId": retailerID,
				"thisMonth":  thisMonthStart,
				"lastMonth":  lastMonthStart,
			},
		}
		totalsIter := client.Single().WithTimestampBound(spanner.ExactStaleness(10*time.Second)).Query(ctx, totalsStmt)
		defer totalsIter.Stop()
		if row, err := totalsIter.Next(); err == nil {
			_ = row.Columns(&resp.TotalThisMonth, &resp.TotalLastMonth)
		}

		// 3. Top suppliers by spend
		topSuppStmt := spanner.Statement{
			SQL: `SELECT 
					p.SupplierId as supplier_id,
					COALESCE(s.Name, p.SupplierId) as supplier_name,
					CAST(SUM(oi.Quantity * p.Price) AS INT64) as total,
					COUNT(DISTINCT o.OrderId) as order_count
				FROM Orders o
				JOIN OrderItems oi ON o.OrderId = oi.OrderId
				JOIN Products p ON oi.ProductId = p.ProductId
				LEFT JOIN Suppliers s ON p.SupplierId = s.SupplierId
				WHERE o.RetailerId = @retailerId
				  AND o.State IN ('COMPLETED', 'ARRIVED', 'IN_TRANSIT')
				  AND o.CreatedAt >= @since
				GROUP BY p.SupplierId, s.Name
				ORDER BY total DESC
				LIMIT 5`,
			Params: map[string]interface{}{
				"retailerId": retailerID,
				"since":      sixMonthsAgo,
			},
		}
		suppIter := client.Single().WithTimestampBound(spanner.ExactStaleness(10*time.Second)).Query(ctx, topSuppStmt)
		defer suppIter.Stop()
		for {
			row, err := suppIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var ts TopSupplier
			if err := row.Columns(&ts.SupplierID, &ts.SupplierName, &ts.Total, &ts.OrderCount); err != nil {
				continue
			}
			resp.TopSuppliers = append(resp.TopSuppliers, ts)
		}

		// 4. Top products by spend
		topProdStmt := spanner.Statement{
			SQL: `SELECT 
					oi.ProductId as product_id,
					p.Name as product_name,
					CAST(SUM(oi.Quantity * p.Price) AS INT64) as total,
					SUM(oi.Quantity) as quantity
				FROM Orders o
				JOIN OrderItems oi ON o.OrderId = oi.OrderId
				JOIN Products p ON oi.ProductId = p.ProductId
				WHERE o.RetailerId = @retailerId
				  AND o.State IN ('COMPLETED', 'ARRIVED', 'IN_TRANSIT')
				  AND o.CreatedAt >= @since
				GROUP BY oi.ProductId, p.Name
				ORDER BY total DESC
				LIMIT 10`,
			Params: map[string]interface{}{
				"retailerId": retailerID,
				"since":      sixMonthsAgo,
			},
		}
		prodIter := client.Single().WithTimestampBound(spanner.ExactStaleness(10*time.Second)).Query(ctx, topProdStmt)
		defer prodIter.Stop()
		for {
			row, err := prodIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var tp TopProduct
			if err := row.Columns(&tp.ProductID, &tp.ProductName, &tp.Total, &tp.Quantity); err != nil {
				continue
			}
			resp.TopProducts = append(resp.TopProducts, tp)
		}

		if resp.MonthlyExpenses == nil {
			resp.MonthlyExpenses = []MonthlyExpense{}
		}
		if resp.TopSuppliers == nil {
			resp.TopSuppliers = []TopSupplier{}
		}
		if resp.TopProducts == nil {
			resp.TopProducts = []TopProduct{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
