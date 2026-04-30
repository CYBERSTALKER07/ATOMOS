package analytics

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"backend-go/auth"
	"backend-go/proximity"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── Supplier Earnings ───────────────────────────────────────────────────────
// GET /v1/supplier/earnings — Revenue breakdown from LedgerEntries + Orders.

type MonthlyRevenue struct {
	Month           string `json:"month"`
	GrossVolume     int64  `json:"gross_volume"`
	PlatformFee     int64  `json:"platform_fee"`
	NetPayout       int64  `json:"net_payout"`
	CompletedOrders int64  `json:"completed_orders"`
}

type SupplierEarningsResponse struct {
	TotalGross       int64            `json:"total_gross"`
	TotalNet         int64            `json:"total_net"`
	TotalFee         int64            `json:"total_fee"`
	TotalOrders      int64            `json:"total_orders"`
	MonthlyBreakdown []MonthlyRevenue `json:"monthly_breakdown"`
}

func HandleSupplierEarnings(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
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
		supplierID := claims.ResolveSupplierID()

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		resp := SupplierEarningsResponse{}

		// 1. Total earnings from LedgerEntries
		// The Treasurer writes AccountId = raw supplierID (not prefixed).
		// Only sum SETTLED entries — PENDING_GATEWAY hasn't been captured yet.
		totalStmt := spanner.Statement{
			SQL: `SELECT IFNULL(SUM(Amount), 0) as total
				FROM LedgerEntries
				WHERE AccountId = @supplierId
				  AND EntryType = 'CREDIT'
				  AND Status = 'SETTLED'`,
			Params: map[string]interface{}{"supplierId": supplierID},
		}
		readClient := getReadClient(r.Context(), client, readRouter, nil)
		totalIter := readClient.Single().Query(ctx, totalStmt)
		defer totalIter.Stop()
		row, err := totalIter.Next()
		if err == nil {
			row.Columns(&resp.TotalNet)
		}
		totalIter.Stop()

		// 2. Platform fee total (ACC-THE-LAB entries for this supplier's orders)
		feeStmt := spanner.Statement{
			SQL: `SELECT IFNULL(SUM(le.Amount), 0) as fee
				FROM LedgerEntries le
				JOIN Orders o ON le.OrderId = o.OrderId
				WHERE le.AccountId = 'ACC-THE-LAB'
				  AND o.SupplierId = @supplierId
				  AND le.EntryType = 'CREDIT'
				  AND le.Status = 'SETTLED'`,
			Params: map[string]interface{}{"supplierId": supplierID},
		}
		feeIter := readClient.Single().Query(ctx, feeStmt)
		defer feeIter.Stop()
		feeRow, err := feeIter.Next()
		if err == nil {
			feeRow.Columns(&resp.TotalFee)
		}
		feeIter.Stop()

		resp.TotalGross = resp.TotalNet + resp.TotalFee

		// Read platform fee percent from SystemConfig (default 0% — zero-fee era)
		feePercent := int64(0)
		if cfgRow, cfgErr := readClient.Single().ReadRow(ctx, "SystemConfig",
			spanner.Key{"platform_fee_percent"}, []string{"ConfigValue"}); cfgErr == nil {
			var cfgVal string
			if cfgRow.Columns(&cfgVal) == nil {
				if n, parseErr := strconv.ParseInt(cfgVal, 10, 64); parseErr == nil && n >= 0 {
					feePercent = n
				}
			}
		}

		// 3. Monthly breakdown (last 12 months from Orders)
		twelveMonthsAgo := time.Now().AddDate(-1, 0, 0)
		monthlyStmt := spanner.Statement{
			SQL: `SELECT
					FORMAT_TIMESTAMP('%Y-%m', o.CreatedAt) as month,
					IFNULL(SUM(o.Amount), 0) as gross_volume,
					COUNT(o.OrderId) as completed_orders
				FROM Orders o
				WHERE o.SupplierId = @supplierId
				  AND o.State = 'COMPLETED'
				  AND o.CreatedAt >= @since
				GROUP BY month
				ORDER BY month DESC`,
			Params: map[string]interface{}{
				"supplierId": supplierID,
				"since":      twelveMonthsAgo,
			},
		}
		monthlyIter := readClient.Single().Query(ctx, monthlyStmt)
		defer monthlyIter.Stop()
		for {
			row, err := monthlyIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[SUPPLIER EARNINGS] Monthly query error: %v", err)
				break
			}
			var m MonthlyRevenue
			if err := row.Columns(&m.Month, &m.GrossVolume, &m.CompletedOrders); err != nil {
				continue
			}
			// Apply dynamic platform fee from SystemConfig
			m.PlatformFee = m.GrossVolume * feePercent / 100
			m.NetPayout = m.GrossVolume - m.PlatformFee
			resp.TotalOrders += m.CompletedOrders
			resp.MonthlyBreakdown = append(resp.MonthlyBreakdown, m)
		}

		if resp.MonthlyBreakdown == nil {
			resp.MonthlyBreakdown = []MonthlyRevenue{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
