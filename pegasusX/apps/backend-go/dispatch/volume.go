package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
)

const defaultUnitVolumeVU = 1.0

type volumeEnrichRow struct {
	Order        *DispatchableOrder
	LineItemsRaw []byte
}

// LookupProductUnitVolumes bulk-reads UnitVolumeVU for product IDs (SKU = ProductId at checkout).
func LookupProductUnitVolumes(ctx context.Context, client *spanner.Client, productIDs []string) (map[string]float64, error) {
	out := make(map[string]float64, len(productIDs))
	if client == nil || len(productIDs) == 0 {
		return out, nil
	}
	unique := make([]string, 0, len(productIDs))
	seen := make(map[string]struct{}, len(productIDs))
	for _, id := range productIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return out, nil
	}

	stmt := spanner.Statement{
		SQL:    `SELECT ProductId, UnitVolumeVU FROM Products WHERE ProductId IN UNNEST(@ids)`,
		Params: map[string]any{"ids": unique},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("lookup product unit volumes: %w", err)
		}
		var productID string
		var unitVU float64
		if err := row.Columns(&productID, &unitVU); err != nil {
			return nil, fmt.Errorf("scan product unit volume: %w", err)
		}
		if unitVU <= 0 {
			unitVU = defaultUnitVolumeVU
		}
		out[productID] = unitVU
	}
}

func productIDsFromLineItemsRaw(lineItemsRaw []byte) []string {
	if len(lineItemsRaw) == 0 {
		return nil
	}
	var items []order.LineItem
	if err := json.Unmarshal(lineItemsRaw, &items); err != nil {
		return nil
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if id := strings.TrimSpace(item.SKU); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// OrderVolumeVU sums qty × unit VU from line-item snapshots with product lookup fallback.
func OrderVolumeVU(lineItemsRaw []byte, lookup map[string]float64) float64 {
	if len(lineItemsRaw) == 0 {
		return defaultUnitVolumeVU
	}
	var items []order.LineItem
	if err := json.Unmarshal(lineItemsRaw, &items); err != nil || len(items) == 0 {
		return defaultUnitVolumeVU
	}
	var total float64
	for _, item := range items {
		qty := float64(item.Quantity)
		if qty <= 0 {
			qty = 1
		}
		unitVU := item.UnitVolumeVU
		if unitVU <= 0 {
			if lookup != nil {
				unitVU = lookup[strings.TrimSpace(item.SKU)]
			}
		}
		if unitVU <= 0 {
			unitVU = defaultUnitVolumeVU
		}
		total += qty * unitVU
	}
	if total <= 0 {
		return defaultUnitVolumeVU
	}
	return total
}

func enrichDispatchableVolumes(ctx context.Context, client *spanner.Client, rows []volumeEnrichRow) error {
	if len(rows) == 0 {
		return nil
	}
	collect := make([]string, 0, len(rows)*2)
	for _, row := range rows {
		collect = append(collect, productIDsFromLineItemsRaw(row.LineItemsRaw)...)
	}
	lookup, err := LookupProductUnitVolumes(ctx, client, collect)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.Order == nil {
			continue
		}
		row.Order.VolumeVU = OrderVolumeVU(row.LineItemsRaw, lookup)
	}
	return nil
}
