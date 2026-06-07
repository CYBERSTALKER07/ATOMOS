package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
)

// Repository loads dispatchable orders with receiving windows: order-create
// snapshot wins; legacy rows fall back to live Retailers join.
type Repository struct {
	client *spanner.Client
}

// NewRepository returns a Spanner-backed dispatch reader.
func NewRepository(client *spanner.Client) *Repository {
	return &Repository{client: client}
}

// FetchDispatchable returns unassigned orders scoped to supplier and optional warehouse.
func (r *Repository) FetchDispatchable(ctx context.Context, params FetchParams) ([]DispatchableOrder, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("dispatch repository: nil client")
	}
	supplierID := strings.TrimSpace(params.SupplierID)
	if supplierID == "" {
		return []DispatchableOrder{}, nil
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 300
	}
	if limit > 500 {
		limit = 500
	}

	queryParams := map[string]any{
		"supplierId": supplierID,
		"limit":      int64(limit),
	}
	sql := `SELECT o.OrderId, o.RetailerId, COALESCE(r.Name, '') AS RetailerName,
	               COALESCE(o.WarehouseId, '') AS WarehouseId,
	               o.Status, o.TotalMinor, o.Currency,
	               COALESCE(o.H3Cell, '') AS H3Cell,
	               COALESCE(o.Lat, r.Lat, 0) AS Lat,
	               COALESCE(o.Lng, r.Lng, 0) AS Lng,
	               o.LineItemsJson,
	               COALESCE(NULLIF(o.ReceivingWindowOpen, ''), r.ReceivingWindowOpen, '') AS ReceivingWindowOpen,
	               COALESCE(NULLIF(o.ReceivingWindowClose, ''), r.ReceivingWindowClose, '') AS ReceivingWindowClose
	        FROM Orders o
	        JOIN Retailers r ON o.RetailerId = r.RetailerId
	        WHERE o.SupplierId = @supplierId
	          AND o.Status IN ('PENDING', 'LOADED')
	          AND (o.RouteId IS NULL OR o.RouteId = '')
	          AND (o.ManifestId IS NULL OR o.ManifestId = '')`

	if warehouseID := strings.TrimSpace(params.WarehouseID); warehouseID != "" {
		sql += " AND o.WarehouseId = @warehouseId"
		queryParams["warehouseId"] = warehouseID
	}
	if len(params.FilterIDs) > 0 {
		sql += " AND o.OrderId IN UNNEST(@orderIds)"
		queryParams["orderIds"] = params.FilterIDs
	}
	sql += ` ORDER BY o.UpdatedAt DESC
	         LIMIT @limit`

	stmt := spanner.Statement{SQL: sql, Params: queryParams}
	var iter *spanner.RowIterator
	if params.StrongRead {
		iter = r.client.Single().Query(ctx, stmt)
	} else {
		iter = r.client.Single().
			WithTimestampBound(spanner.ExactStaleness(15*time.Second)).
			Query(ctx, stmt)
	}
	defer iter.Stop()

	results := make([]DispatchableOrder, 0, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return results, nil
		}
		if err != nil {
			return nil, fmt.Errorf("query dispatchable orders: %w", err)
		}

		var current DispatchableOrder
		var lineItemsRaw []byte
		if err := row.Columns(
			&current.OrderID,
			&current.RetailerID,
			&current.RetailerName,
			&current.WarehouseID,
			&current.Status,
			&current.TotalMinor,
			&current.Currency,
			&current.H3Cell,
			&current.Lat,
			&current.Lng,
			&lineItemsRaw,
			&current.ReceivingWindowOpen,
			&current.ReceivingWindowClose,
		); err != nil {
			return nil, fmt.Errorf("scan dispatchable order: %w", err)
		}
		current.VolumeVU = estimateVolumeVU(lineItemsRaw)
		results = append(results, current)
	}
}

func estimateVolumeVU(lineItemsRaw []byte) float64 {
	if len(lineItemsRaw) == 0 {
		return 1
	}
	var items []order.LineItem
	if err := json.Unmarshal(lineItemsRaw, &items); err != nil || len(items) == 0 {
		return 1
	}
	var total float64
	for _, item := range items {
		qty := float64(item.Quantity)
		if qty <= 0 {
			qty = 1
		}
		total += qty
	}
	if total <= 0 {
		return 1
	}
	return total
}
