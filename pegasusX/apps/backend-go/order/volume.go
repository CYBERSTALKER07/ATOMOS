package order

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const defaultUnitVolumeVU = 1.0

func (s *Service) enrichLineItemVolumes(ctx context.Context, items []LineItem) ([]LineItem, error) {
	if s == nil || s.spannerClient == nil || len(items) == 0 {
		return items, nil
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if id := strings.TrimSpace(item.SKU); id != "" {
			ids = append(ids, id)
		}
	}
	lookup, err := lookupProductUnitVolumes(ctx, s.spannerClient, ids)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].UnitVolumeVU > 0 {
			continue
		}
		if vu, ok := lookup[strings.TrimSpace(items[i].SKU)]; ok {
			items[i].UnitVolumeVU = vu
		}
	}
	return items, nil
}

func lookupProductUnitVolumes(ctx context.Context, client *spanner.Client, productIDs []string) (map[string]float64, error) {
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
