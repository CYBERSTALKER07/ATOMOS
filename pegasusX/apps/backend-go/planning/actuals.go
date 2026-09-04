package planning

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
)

// ActualKey is the SKU-day grain shared by baselines and accuracy.
type ActualKey struct {
	SupplierID  string
	WarehouseID string
	ProductID   string
	Day         civil.Date
}

// ActualQtyMap maps ActualKey → completed line units.
type ActualQtyMap map[ActualKey]int64

// LoadCompletedActuals sums LineItemsJson quantities from COMPLETED orders in [start,end] (UTC days inclusive).
// ProductId is taken from line.sku (canonical product id on order lines).
func LoadCompletedActuals(ctx context.Context, client *spanner.Client, supplierID string, start, end time.Time) (ActualQtyMap, error) {
	out := make(ActualQtyMap)
	if client == nil {
		return out, nil
	}
	start = start.UTC().Truncate(24 * time.Hour)
	end = end.UTC().Truncate(24 * time.Hour)
	if end.Before(start) {
		return out, nil
	}
	// Inclusive end day → exclusive upper bound next midnight.
	endExclusive := end.Add(24 * time.Hour)

	sql := `SELECT SupplierId, WarehouseId, LineItemsJson, UpdatedAt
		FROM Orders
		WHERE Status = 'COMPLETED'
		  AND UpdatedAt >= @start
		  AND UpdatedAt < @end`
	params := map[string]any{
		"start": start,
		"end":   endExclusive,
	}
	if sid := strings.TrimSpace(supplierID); sid != "" {
		sql += ` AND SupplierId = @sid`
		params["sid"] = sid
	}

	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var sid, whID string
		var raw []byte
		var updatedAt time.Time
		if err := row.Columns(&sid, &whID, &raw, &updatedAt); err != nil {
			return nil, err
		}
		items, err := parseOrderLineItems(raw)
		if err != nil || len(items) == 0 {
			continue
		}
		day := civil.DateOf(updatedAt.UTC())
		for _, item := range items {
			pid := strings.TrimSpace(item.SKU)
			if pid == "" || item.Quantity <= 0 {
				continue
			}
			key := ActualKey{SupplierID: sid, WarehouseID: whID, ProductID: pid, Day: day}
			out[key] += item.Quantity
		}
	}
	return out, nil
}

func parseOrderLineItems(raw []byte) ([]order.LineItem, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var items []order.LineItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// DayActualTotals rolls SKU-day actuals up to (supplier, day) for chart series.
func DayActualTotals(actuals ActualQtyMap, supplierID string) map[string]int64 {
	out := make(map[string]int64)
	sid := strings.TrimSpace(supplierID)
	for k, qty := range actuals {
		if sid != "" && k.SupplierID != sid {
			continue
		}
		out[k.Day.String()] += qty
	}
	return out
}

// LoadBaselineDayTotals sums BaselineQty by ForecastDate for a supplier in [start,end].
func LoadBaselineDayTotals(ctx context.Context, client *spanner.Client, supplierID string, start, end time.Time) (map[string]int64, error) {
	out := make(map[string]int64)
	if client == nil || strings.TrimSpace(supplierID) == "" {
		return out, nil
	}
	sql := `SELECT ForecastDate, COALESCE(SUM(BaselineQty), 0)
		FROM DemandForecastBaseline
		WHERE SupplierId = @sid AND ForecastDate BETWEEN @start AND @end
		GROUP BY ForecastDate`
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: sql,
		Params: map[string]any{
			"sid":   supplierID,
			"start": civil.DateOf(start.UTC()),
			"end":   civil.DateOf(end.UTC()),
		},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var day civil.Date
		var qty int64
		if err := row.Columns(&day, &qty); err != nil {
			return nil, err
		}
		out[day.String()] = qty
	}
	return out, nil
}
