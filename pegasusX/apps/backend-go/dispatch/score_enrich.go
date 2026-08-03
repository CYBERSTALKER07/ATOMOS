package dispatch

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// EnrichScoreSignals fills optional PriorityScore / DriverScore / ShopClosedRisk from Spanner.
// Pure BinPack stays free of I/O; call this in warehouse/supplier orchestration before Solve.
func EnrichScoreSignals(ctx context.Context, client *spanner.Client, orders []DispatchableOrder, fleet []AvailableDriver) ([]DispatchableOrder, []AvailableDriver, error) {
	if client == nil {
		return orders, fleet, nil
	}
	outOrders := append([]DispatchableOrder(nil), orders...)
	outFleet := append([]AvailableDriver(nil), fleet...)

	retailerIDs := uniqueNonEmpty(func() []string {
		ids := make([]string, 0, len(outOrders))
		for _, o := range outOrders {
			ids = append(ids, o.RetailerID)
		}
		return ids
	}())
	driverIDs := uniqueNonEmpty(func() []string {
		ids := make([]string, 0, len(outFleet))
		for _, d := range outFleet {
			ids = append(ids, d.DriverID)
		}
		return ids
	}())

	scores, _ := lookupDriverScores(ctx, client, driverIDs)
	for i := range outFleet {
		if s, ok := scores[outFleet[i].DriverID]; ok {
			outFleet[i].DriverScore = s
		}
	}

	// Shop-closed risk: fraction of recent shop-closed outcomes per retailer (best-effort).
	risk, _ := lookupShopClosedRisk(ctx, client, retailerIDs)
	// Priority: prefer higher TotalMinor as a stable proxy when segment PriorityScore unavailable.
	for i := range outOrders {
		if outOrders[i].PriorityScore <= 0 {
			// Map amount into 40–160 band.
			p := int64(40 + outOrders[i].TotalMinor/1_000_000)
			if p > 160 {
				p = 160
			}
			outOrders[i].PriorityScore = p
		}
		if r, ok := risk[outOrders[i].RetailerID]; ok {
			outOrders[i].ShopClosedRisk = r
		}
	}
	return outOrders, outFleet, nil
}

func uniqueNonEmpty(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func lookupDriverScores(ctx context.Context, client *spanner.Client, driverIDs []string) (map[string]int64, error) {
	out := map[string]int64{}
	if len(driverIDs) == 0 {
		return out, nil
	}
	stmt := spanner.Statement{
		SQL:    `SELECT DriverId, Score FROM DriverScores WHERE DriverId IN UNNEST(@ids)`,
		Params: map[string]any{"ids": driverIDs},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			// Table may be absent in some envs — soft-fail.
			if strings.Contains(err.Error(), "DriverScores") {
				return out, nil
			}
			return nil, fmt.Errorf("driver scores: %w", err)
		}
		var id string
		var score int64
		if err := row.Columns(&id, &score); err != nil {
			return nil, err
		}
		out[id] = score
	}
}

func lookupShopClosedRisk(ctx context.Context, client *spanner.Client, retailerIDs []string) (map[string]float64, error) {
	out := map[string]float64{}
	if len(retailerIDs) == 0 {
		return out, nil
	}
	// Best-effort: count shop-closed resolutions vs delivered in recent orders.
	stmt := spanner.Statement{
		SQL: `SELECT RetailerId,
		        COUNTIF(ShopClosedResolution IS NOT NULL AND ShopClosedResolution != '') AS closed_n,
		        COUNT(*) AS total_n
		      FROM Orders
		      WHERE RetailerId IN UNNEST(@ids)
		        AND Status IN ('DELIVERED', 'DELIVERED_ON_CREDIT', 'FISCALIZED', 'ARRIVED_SHOP_CLOSED')
		      GROUP BY RetailerId`,
		Params: map[string]any{"ids": retailerIDs},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			if strings.Contains(err.Error(), "ShopClosedResolution") {
				return out, nil
			}
			return nil, fmt.Errorf("shop-closed risk: %w", err)
		}
		var rid string
		var closedN, totalN int64
		if err := row.Columns(&rid, &closedN, &totalN); err != nil {
			return nil, err
		}
		if totalN > 0 {
			out[rid] = float64(closedN) / float64(totalN)
		}
	}
}
