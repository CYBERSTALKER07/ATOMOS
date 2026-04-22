package analytics

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

// SupplierFinancialsResponse is the response for GET /v1/supplier/financials.
type SupplierFinancialsResponse struct {
	SupplierID       string               `json:"supplier_id"`
	Period           string               `json:"period"`
	TotalRevenue     int64                `json:"total_revenue"`
	PlatformFee      int64                `json:"platform_fee"`
	NetPayout        int64                `json:"net_payout"`
	CompletedOrders  int64                `json:"completed_orders"`
	AvgOrderValue    int64                `json:"avg_order_value"`
	Currency         string               `json:"currency"`
	GatewayBreakdown []GatewayFinBucket   `json:"gateway_breakdown"`
	WarehouseRevenue []WarehouseRevBucket `json:"warehouse_revenue"`
	CashPending      int64                `json:"cash_pending"`
	CashCollected    int64                `json:"cash_collected"`
	SettlementRate   float64              `json:"settlement_rate"` // fraction 0.0–1.0
}

// GatewayFinBucket is a per-gateway revenue aggregate.
type GatewayFinBucket struct {
	Gateway    string `json:"gateway"`
	Revenue    int64  `json:"revenue"`
	OrderCount int64  `json:"order_count"`
}

// WarehouseRevBucket is a per-warehouse revenue aggregate.
type WarehouseRevBucket struct {
	WarehouseID string `json:"warehouse_id"`
	Revenue     int64  `json:"revenue"`
	OrderCount  int64  `json:"order_count"`
}

// HandleSupplierFinancials serves GET /v1/supplier/financials.
func HandleSupplierFinancials(spannerClient *spanner.Client) http.HandlerFunc {
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
		supplierID := claims.ResolveSupplierID()

		period := r.URL.Query().Get("period")
		if period == "" {
			period = time.Now().UTC().Format("2006-01")
		}
		startDate := period + "-01"
		t, err := time.Parse("2006-01", period)
		if err != nil {
			http.Error(w, `{"error":"invalid period format, expected YYYY-MM"}`, http.StatusBadRequest)
			return
		}
		endDate := t.AddDate(0, 1, 0).Format("2006-01-02")

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		resp, err := querySupplierFinancials(ctx, spannerClient, supplierID, startDate, endDate, period)
		if err != nil {
			log.Printf("[FINANCIALS] supplier=%s error: %v", supplierID, err)
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func querySupplierFinancials(ctx context.Context, client *spanner.Client, supplierID, startDate, endDate, period string) (*SupplierFinancialsResponse, error) {
	resp := &SupplierFinancialsResponse{
		SupplierID:       supplierID,
		Period:           period,
		Currency:         "UZS",
		GatewayBreakdown: []GatewayFinBucket{},
		WarehouseRevenue: []WarehouseRevBucket{},
	}

	txn := client.Single().WithTimestampBound(spanner.MaxStaleness(15 * time.Second))

	// Revenue by gateway
	stmt := spanner.Statement{
		SQL: `SELECT IFNULL(SUM(o.Amount), 0), COUNT(*), IFNULL(o.PaymentGateway, 'UNKNOWN')
		      FROM Orders o
		      WHERE o.SupplierId = @sid
		        AND o.State = 'COMPLETED'
		        AND o.CreatedAt >= @start AND o.CreatedAt < @end
		      GROUP BY o.PaymentGateway`,
		Params: map[string]interface{}{
			"sid":   supplierID,
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
		resp.GatewayBreakdown = append(resp.GatewayBreakdown, GatewayFinBucket{
			Gateway: gw, Revenue: rev, OrderCount: cnt,
		})
	}

	if resp.CompletedOrders > 0 {
		resp.AvgOrderValue = resp.TotalRevenue / resp.CompletedOrders
	}

	// Revenue by warehouse
	wStmt := spanner.Statement{
		SQL: `SELECT IFNULL(o.WarehouseId, ''), IFNULL(SUM(o.Amount), 0), COUNT(*)
		      FROM Orders o
		      WHERE o.SupplierId = @sid AND o.State = 'COMPLETED'
		        AND o.CreatedAt >= @start AND o.CreatedAt < @end
		      GROUP BY o.WarehouseId`,
		Params: map[string]interface{}{
			"sid":   supplierID,
			"start": startDate,
			"end":   endDate,
		},
	}
	wIter := txn.Query(ctx, wStmt)
	defer wIter.Stop()

	for {
		row, err := wIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("warehouse revenue: %w", err)
		}
		var wid string
		var rev, cnt int64
		if err := row.Columns(&wid, &rev, &cnt); err != nil {
			return nil, err
		}
		resp.WarehouseRevenue = append(resp.WarehouseRevenue, WarehouseRevBucket{
			WarehouseID: wid, Revenue: rev, OrderCount: cnt,
		})
	}

	// Platform fee from ledger
	fStmt := spanner.Statement{
		SQL: `SELECT IFNULL(SUM(le.Amount), 0)
		      FROM LedgerEntries le
		      JOIN Orders o ON le.OrderId = o.OrderId
		      WHERE o.SupplierId = @sid
		        AND le.AccountId = 'ACC-THE-LAB'
		        AND le.EntryType = 'COMMISSION'
		        AND o.CreatedAt >= @start AND o.CreatedAt < @end`,
		Params: map[string]interface{}{
			"sid":   supplierID,
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

	// Cash collection
	cStmt := spanner.Statement{
		SQL: `SELECT
		        IFNULL(SUM(CASE WHEN mi.CustodyStatus = 'HELD_BY_DRIVER' THEN mi.Total ELSE 0 END), 0),
		        IFNULL(SUM(CASE WHEN mi.CustodyStatus = 'DEPOSITED' THEN mi.Total ELSE 0 END), 0)
		      FROM MasterInvoices mi
		      JOIN Orders o ON mi.OrderId = o.OrderId
		      WHERE o.SupplierId = @sid AND mi.PaymentMode = 'CASH'
		        AND o.CreatedAt >= @start AND o.CreatedAt < @end`,
		Params: map[string]interface{}{
			"sid":   supplierID,
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

	// Settlement rate
	if resp.TotalRevenue > 0 {
		settled := resp.TotalRevenue - resp.CashPending
		resp.SettlementRate = float64(settled) / float64(resp.TotalRevenue)
	}

	return resp, nil
}
