package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// FinancialsResponse is the response for /v1/warehouse/ops/financials.
type FinancialsResponse struct {
	WarehouseID      string           `json:"warehouse_id"`
	Period           string           `json:"period"` // e.g. "2025-06"
	TotalRevenue     int64            `json:"total_revenue"`
	CompletedOrders  int64            `json:"completed_orders"`
	AvgOrderValue    int64            `json:"avg_order_value"`
	Currency         string           `json:"currency"`
	GatewayBreakdown []GatewayBucket  `json:"gateway_breakdown"`
	DailyRevenue     []DailyRevBucket `json:"daily_revenue"`
	PlatformFee      int64            `json:"platform_fee"`
	NetPayout        int64            `json:"net_payout"`
	CashPending      int64            `json:"cash_pending"`
	CashCollected    int64            `json:"cash_collected"`
}

// GatewayBucket aggregates revenue by payment gateway.
type GatewayBucket struct {
	Gateway    string `json:"gateway"`
	Revenue    int64  `json:"revenue"`
	OrderCount int64  `json:"order_count"`
}

// DailyRevBucket is a single day's revenue sum.
type DailyRevBucket struct {
	Date    string `json:"date"`
	Revenue int64  `json:"revenue"`
	Orders  int64  `json:"orders"`
}

// HandleWarehouseFinancials serves GET /v1/warehouse/ops/financials.
// Scoped to the authenticated warehouse via RequireWarehouseScope.
func HandleWarehouseFinancials(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		warehouseID := auth.EffectiveWarehouseID(r.Context())
		if warehouseID == "" {
			http.Error(w, `{"error":"missing warehouse scope"}`, http.StatusForbidden)
			return
		}

		period := r.URL.Query().Get("period")
		if period == "" {
			period = time.Now().UTC().Format("2006-01")
		}
		startDate := period + "-01"
		endDate := nextMonth(period)

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		resp, err := queryWarehouseFinancials(ctx, spannerClient, warehouseID, startDate, endDate, period)
		if err != nil {
			log.Printf("[FINANCIALS] warehouse=%s error: %v", warehouseID, err)
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func queryWarehouseFinancials(ctx context.Context, client *spanner.Client, warehouseID, startDate, endDate, period string) (*FinancialsResponse, error) {
	resp := &FinancialsResponse{
		WarehouseID:      warehouseID,
		Period:           period,
		Currency:         "UZS",
		GatewayBreakdown: []GatewayBucket{},
		DailyRevenue:     []DailyRevBucket{},
	}

	txn := client.Single().WithTimestampBound(spanner.MaxStaleness(15 * time.Second))

	// Aggregate revenue + order count for completed orders
	stmt := spanner.Statement{
		SQL: `SELECT IFNULL(SUM(o.Amount), 0), COUNT(*), IFNULL(o.PaymentGateway, 'UNKNOWN')
		      FROM Orders o
		      WHERE o.WarehouseId = @wid
		        AND o.State = 'COMPLETED'
		        AND o.CreatedAt >= @start AND o.CreatedAt < @end
		      GROUP BY o.PaymentGateway`,
		Params: map[string]interface{}{
			"wid":   warehouseID,
			"start": startDate,
			"end":   endDate,
		},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gateway breakdown: %w", err)
		}
		var rev, cnt int64
		var gw string
		if err := row.Columns(&rev, &cnt, &gw); err != nil {
			return nil, err
		}
		resp.TotalRevenue += rev
		resp.CompletedOrders += cnt
		resp.GatewayBreakdown = append(resp.GatewayBreakdown, GatewayBucket{
			Gateway: gw, Revenue: rev, OrderCount: cnt,
		})
	}

	if resp.CompletedOrders > 0 {
		resp.AvgOrderValue = resp.TotalRevenue / resp.CompletedOrders
	}

	// Daily revenue breakdown
	dStmt := spanner.Statement{
		SQL: `SELECT CAST(o.CreatedAt AS DATE) AS d, IFNULL(SUM(o.Amount), 0), COUNT(*)
		      FROM Orders o
		      WHERE o.WarehouseId = @wid AND o.State = 'COMPLETED'
		        AND o.CreatedAt >= @start AND o.CreatedAt < @end
		      GROUP BY d ORDER BY d`,
		Params: map[string]interface{}{
			"wid":   warehouseID,
			"start": startDate,
			"end":   endDate,
		},
	}
	dIter := txn.Query(ctx, dStmt)
	defer dIter.Stop()

	for {
		row, err := dIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("daily revenue: %w", err)
		}
		var dt string
		var rev, cnt int64
		if err := row.Columns(&dt, &rev, &cnt); err != nil {
			return nil, err
		}
		resp.DailyRevenue = append(resp.DailyRevenue, DailyRevBucket{
			Date: dt, Revenue: rev, Orders: cnt,
		})
	}

	// Platform fee from LedgerEntries
	fStmt := spanner.Statement{
		SQL: `SELECT IFNULL(SUM(le.Amount), 0)
		      FROM LedgerEntries le
		      JOIN Orders o ON le.OrderId = o.OrderId
		      WHERE o.WarehouseId = @wid
		        AND le.AccountId = 'ACC-THE-LAB'
		        AND le.EntryType = 'COMMISSION'
		        AND o.CreatedAt >= @start AND o.CreatedAt < @end`,
		Params: map[string]interface{}{
			"wid":   warehouseID,
			"start": startDate,
			"end":   endDate,
		},
	}
	fRow, err := txn.Query(ctx, fStmt).Next()
	if err == nil {
		var fee int64
		if err := fRow.Columns(&fee); err == nil {
			resp.PlatformFee = fee
		}
	}
	resp.NetPayout = resp.TotalRevenue - resp.PlatformFee

	// Cash collection status
	cStmt := spanner.Statement{
		SQL: `SELECT
		        IFNULL(SUM(CASE WHEN mi.CustodyStatus = 'HELD_BY_DRIVER' THEN mi.Total ELSE 0 END), 0),
		        IFNULL(SUM(CASE WHEN mi.CustodyStatus = 'DEPOSITED' THEN mi.Total ELSE 0 END), 0)
		      FROM MasterInvoices mi
		      JOIN Orders o ON mi.OrderId = o.OrderId
		      WHERE o.WarehouseId = @wid AND mi.PaymentMode = 'CASH'
		        AND o.CreatedAt >= @start AND o.CreatedAt < @end`,
		Params: map[string]interface{}{
			"wid":   warehouseID,
			"start": startDate,
			"end":   endDate,
		},
	}
	cRow, err := txn.Query(ctx, cStmt).Next()
	if err == nil {
		var pending, collected int64
		if err := cRow.Columns(&pending, &collected); err == nil {
			resp.CashPending = pending
			resp.CashCollected = collected
		}
	}

	return resp, nil
}

// nextMonth returns the first day of the month after the given "YYYY-MM" period.
func nextMonth(period string) string {
	t, err := time.Parse("2006-01", period)
	if err != nil {
		return time.Now().UTC().AddDate(0, 1, 0).Format("2006-01") + "-01"
	}
	return t.AddDate(0, 1, 0).Format("2006-01-02")
}
