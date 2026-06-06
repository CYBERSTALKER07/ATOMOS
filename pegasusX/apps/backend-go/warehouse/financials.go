package warehouse

import (
	"net/http"
	"strings"
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

	writeJSON(w, http.StatusOK, map[string]any{
		"warehouse_id":      whID,
		"period":            period,
		"currency":          s.currency,
		"total_revenue":     totalRevenue,
		"completed_orders":  completedOrders,
		"avg_order_value":   avgOrder,
		"gateway_breakdown": []any{},
		"daily_revenue":     []any{},
		"platform_fee":      int64(0),
		"net_payout":        totalRevenue,
		"cash_pending":      int64(0),
		"cash_collected":    int64(0),
	})
}
