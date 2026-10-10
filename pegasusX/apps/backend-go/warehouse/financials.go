package warehouse

import (
	"context"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// HandleOpsFinancials serves GET /v1/warehouse/ops/financials.
func (s *Service) HandleOpsFinancials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" {
		period = s.now().UTC().Format("2006-01")
	}

	var totalRevenue, completedOrders int64
	if s.analyticsQuery != nil {
		counts, err := s.analyticsQuery(r.Context(), whID)
		if err == nil {
			totalRevenue = counts.TotalRevenue
			completedOrders = counts.CompletedOrders
		} else {
			s.log.WarnContext(r.Context(), "financials analytics query failed", "err", err)
		}
	}

	avgOrder := int64(0)
	if completedOrders > 0 {
		avgOrder = totalRevenue / completedOrders
	}

	payload := map[string]any{
		"warehouse_id":           whID,
		"period":                 period,
		"currency":               s.currency,
		"total_revenue":          totalRevenue,
		"completed_orders":       completedOrders,
		"avg_order_value":        avgOrder,
		"platform_fee":           int64(0),
		"net_payout":             totalRevenue,
		"cash_pending":           int64(0),
		"cash_collected":         int64(0),
	}

	if rows, ok := s.loadFinancialsGatewayBreakdown(r.Context(), whID, period); ok {
		payload["gateway_breakdown"] = rows
		payload["gateway_breakdown_available"] = true
	} else {
		payload["gateway_breakdown_available"] = false
	}

	if fee, ok := s.loadFinancialsPlatformFee(r.Context(), whID, period, completedOrders, totalRevenue); ok {
		payload["platform_fee"] = fee
		payload["platform_fee_available"] = true
		payload["net_payout"] = totalRevenue - fee
	} else {
		payload["platform_fee_available"] = false
	}

	if daily, ok := s.loadFinancialsDailyRevenue(r.Context(), whID, period); ok {
		payload["daily_revenue"] = daily
		payload["daily_revenue_available"] = true
	} else {
		payload["daily_revenue_available"] = false
	}

	writeJSON(w, http.StatusOK, payload)
}

func (s *Service) loadFinancialsDailyRevenue(ctx context.Context, warehouseID, period string) ([]map[string]any, bool) {
	if s == nil || s.spannerClient == nil || strings.TrimSpace(warehouseID) == "" {
		return nil, false
	}
	startAt := financialsPeriodStart(period, s.now().UTC())
	params := map[string]any{
		"warehouseId": warehouseID,
		"startAt":     startAt,
	}
	if sid := strings.TrimSpace(s.analyticsSupplierID(ctx)); sid != "" {
		params["supplierId"] = sid
	}
	txn := s.spannerClient.Single().WithTimestampBound(spanner.MaxStaleness(15 * time.Second))
	defer txn.Close()
	rows, err := s.loadAnalyticsDailyBreakdown(ctx, txn, params)
	if err != nil {
		if s.log != nil {
			s.log.WarnContext(ctx, "financials daily revenue query failed", "warehouse_id", warehouseID, "err", err)
		}
		return nil, false
	}
	return rows, true
}

func (s *Service) loadFinancialsGatewayBreakdown(ctx context.Context, warehouseID, period string) ([]map[string]any, bool) {
	if s != nil && s.gatewayBreakdownQuery != nil {
		return s.gatewayBreakdownQuery(ctx, warehouseID, period)
	}
	if s == nil || s.spannerClient == nil || strings.TrimSpace(warehouseID) == "" {
		return nil, false
	}
	startAt := financialsPeriodStart(period, s.now().UTC())
	params := map[string]any{
		"warehouseId": warehouseID,
		"startAt":     startAt,
	}
	sql := `SELECT ps.Gateway, COALESCE(SUM(ps.AmountMinor), 0)
		FROM PaymentSessions ps
		INNER JOIN Orders o ON o.OrderId = ps.OrderId
		WHERE o.WarehouseId = @warehouseId
		  AND ps.CreatedAt >= @startAt
		  AND ps.Status IN ('CAPTURED', 'SUCCEEDED', 'COMPLETED', 'PAID')
		GROUP BY ps.Gateway
		ORDER BY ps.Gateway`
	if sid := strings.TrimSpace(s.analyticsSupplierID(ctx)); sid != "" {
		params["supplierId"] = sid
		sql = `SELECT ps.Gateway, COALESCE(SUM(ps.AmountMinor), 0)
			FROM PaymentSessions ps
			INNER JOIN Orders o ON o.OrderId = ps.OrderId
			WHERE o.WarehouseId = @warehouseId
			  AND o.SupplierId = @supplierId
			  AND ps.CreatedAt >= @startAt
			  AND ps.Status IN ('CAPTURED', 'SUCCEEDED', 'COMPLETED', 'PAID')
			GROUP BY ps.Gateway
			ORDER BY ps.Gateway`
	}
	txn := s.spannerClient.Single().WithTimestampBound(spanner.MaxStaleness(15 * time.Second))
	defer txn.Close()
	iter := txn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var rows []map[string]any
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if s.log != nil {
				s.log.WarnContext(ctx, "financials gateway breakdown query failed", "warehouse_id", warehouseID, "err", err)
			}
			return nil, false
		}
		var gateway string
		var amount int64
		if err := row.Columns(&gateway, &amount); err != nil {
			return nil, false
		}
		rows = append(rows, map[string]any{
			"gateway":      gateway,
			"amount_minor": amount,
		})
	}
	if rows == nil {
		rows = []map[string]any{}
	}
	return rows, true
}

func (s *Service) loadFinancialsPlatformFee(ctx context.Context, warehouseID, period string, completedOrders, gmvMinor int64) (int64, bool) {
	if s != nil && s.platformFeeQuery != nil {
		return s.platformFeeQuery(ctx, warehouseID, period)
	}
	if s == nil || s.spannerClient == nil {
		return 0, false
	}
	supplierID := strings.TrimSpace(s.analyticsSupplierID(ctx))
	if supplierID == "" {
		return 0, false
	}
	now := financialsPeriodStart(period, s.now().UTC())
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT FeeScheduleId, PerOrderMinor, GmvBps, MonthlySubscriptionMinor
		      FROM BillingFeeSchedules
		      WHERE (SupplierId = @sid OR SupplierId = '')
		        AND EffectiveFrom <= @now AND (EffectiveTo IS NULL OR EffectiveTo > @now)
		      ORDER BY CASE WHEN SupplierId = @sid THEN 0 ELSE 1 END, EffectiveFrom DESC
		      LIMIT 1`,
		Params: map[string]any{"sid": supplierID, "now": now},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return 0, false
	}
	if err != nil {
		if s.log != nil {
			s.log.WarnContext(ctx, "financials platform fee query failed", "warehouse_id", warehouseID, "err", err)
		}
		return 0, false
	}
	var scheduleID string
	var perOrder, gmvBps, monthly int64
	if err := row.Columns(&scheduleID, &perOrder, &gmvBps, &monthly); err != nil {
		return 0, false
	}
	if strings.TrimSpace(scheduleID) == "" {
		return 0, false
	}
	fee := perOrder*completedOrders + gmvMinor*gmvBps/10000 + monthly
	return fee, true
}

func financialsPeriodStart(period string, now time.Time) time.Time {
	period = strings.TrimSpace(period)
	if t, err := time.Parse("2006-01", period); err == nil {
		return t.UTC()
	}
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}
