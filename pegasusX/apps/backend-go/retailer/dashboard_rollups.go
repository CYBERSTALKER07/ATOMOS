package retailer

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Same 17-key funnel as warehouse/supplier command boards.
var retailerOrderStatusFunnel = []string{
	"PENDING", "SCHEDULED", "AUTO_ACCEPTED", "BACKORDERED",
	"LOADED", "IN_TRANSIT", "DELAYED",
	"ARRIVED", "ARRIVED_SHOP_CLOSED",
	"AWAITING_PAYMENT", "PENDING_CASH_COLLECTION", "DELIVERED_ON_CREDIT",
	"FISCALIZING", "FISCAL_FAILED", "RECONCILIATION_REQUIRED",
	"COMPLETED", "CANCELLED",
}

// RetailerOrderStatusRow is one GROUP BY (supplier, status) count.
// Count child Orders rows, never ParentOrders rollups.
type RetailerOrderStatusRow struct {
	SupplierID string
	Status     string
	Count      int
}

// RetailerSupplierOrderFacet is the per-supplier command stack.
type RetailerSupplierOrderFacet struct {
	SupplierID     string         `json:"supplier_id"`
	OrdersByStatus map[string]int `json:"orders_by_status"`
}

// RetailerPulseLoyalty is honesty-only on the command pulse.
// Real enrollment stays on GET /v1/retailer/loyalty/tier. Never invent Bronze.
type RetailerPulseLoyalty struct {
	Enrolled bool `json:"enrolled"`
}

// RetailerDashboardOrdersQuery injects Spanner-shaped child-order counts (tests + live).
type RetailerDashboardOrdersQuery func(ctx context.Context, retailerID string) ([]RetailerOrderStatusRow, error)

func emptyRetailerOrderStatusCounts() map[string]int {
	out := make(map[string]int, len(retailerOrderStatusFunnel))
	for _, key := range retailerOrderStatusFunnel {
		out[key] = 0
	}
	return out
}

func canonicalizeRetailerOrderStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "DISPATCHED":
		return "LOADED"
	case "EN_ROUTE":
		return "IN_TRANSIT"
	case "ARRIVING":
		return "ARRIVED"
	case "SHOP_CLOSED_PENDING":
		return "ARRIVED_SHOP_CLOSED"
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}

func applyRetailerOrderRows(rows []RetailerOrderStatusRow) (map[string]int, []RetailerSupplierOrderFacet, int) {
	overall := emptyRetailerOrderStatusCounts()
	bySupplier := map[string]map[string]int{}
	open := 0
	for _, row := range rows {
		status := canonicalizeRetailerOrderStatus(row.Status)
		if _, ok := overall[status]; !ok {
			continue
		}
		n := row.Count
		if n <= 0 {
			continue
		}
		overall[status] += n
		if status != "COMPLETED" && status != "CANCELLED" {
			open += n
		}
		sid := strings.TrimSpace(row.SupplierID)
		if bySupplier[sid] == nil {
			bySupplier[sid] = emptyRetailerOrderStatusCounts()
		}
		bySupplier[sid][status] += n
	}
	ids := make([]string, 0, len(bySupplier))
	for id := range bySupplier {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	facets := make([]RetailerSupplierOrderFacet, 0, len(ids))
	for _, id := range ids {
		facets = append(facets, RetailerSupplierOrderFacet{
			SupplierID:     id,
			OrdersByStatus: bySupplier[id],
		})
	}
	return overall, facets, open
}

func (s *Service) loadRetailerDashboardOrders(ctx context.Context, retailerID string) ([]RetailerOrderStatusRow, string, error) {
	if s != nil && s.dashboardOrders != nil {
		rows, err := s.dashboardOrders(ctx, retailerID)
		if err != nil {
			return nil, "", fmt.Errorf("control tower dashboard orders: %w", err)
		}
		return rows, "spanner", nil
	}
	if s != nil && s.spannerClient != nil {
		rows, err := s.loadRetailerOrderStatusFromSpanner(ctx, retailerID)
		if err != nil {
			return nil, "", fmt.Errorf("control tower dashboard orders: %w", err)
		}
		return rows, "spanner", nil
	}
	return nil, "empty", nil
}

func (s *Service) loadRetailerOrderStatusFromSpanner(ctx context.Context, retailerID string) ([]RetailerOrderStatusRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SupplierId, ''), Status, COUNT(1)
		      FROM Orders@{FORCE_INDEX=Idx_Orders_ByRetailerCreated}
		      WHERE RetailerId = @RetailerId
		      GROUP BY SupplierId, Status`,
		Params: map[string]any{"RetailerId": retailerID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	rows := make([]RetailerOrderStatusRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var supplierID, status string
		var n int64
		if err := row.Columns(&supplierID, &status, &n); err != nil {
			return nil, err
		}
		rows = append(rows, RetailerOrderStatusRow{
			SupplierID: supplierID,
			Status:     status,
			Count:      int(n),
		})
	}
	return rows, nil
}
