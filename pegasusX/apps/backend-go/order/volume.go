package order

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// productSnapshot captures catalog handling flags needed at order time.
type productSnapshot struct {
	handlingClass string
	isHazardous   bool
	isPerishable  bool
}

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
	lookup, err := lookupProductSnapshots(ctx, s.spannerClient, ids)
	if err != nil {
		return nil, err
	}
	for i := range items {
		snap, ok := lookup[strings.TrimSpace(items[i].SKU)]
		if !ok {
			continue
		}
		items[i].HandlingClass = snap.handlingClass
		items[i].IsHazardous = snap.isHazardous
		items[i].IsPerishable = snap.isPerishable
	}
	return items, nil
}

func lookupProductSnapshots(ctx context.Context, client *spanner.Client, productIDs []string) (map[string]productSnapshot, error) {
	out := make(map[string]productSnapshot, len(productIDs))
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
		SQL:    `SELECT ProductId, HandlingClass, IsHazardous, IsPerishable FROM Products WHERE ProductId IN UNNEST(@ids)`,
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
			return nil, fmt.Errorf("lookup product snapshots: %w", err)
		}
		var productID string
		var snap productSnapshot
		var handlingClass spanner.NullString
		if err := row.Columns(&productID, &handlingClass, &snap.isHazardous, &snap.isPerishable); err != nil {
			return nil, fmt.Errorf("scan product snapshot: %w", err)
		}
		snap.handlingClass = handlingClass.StringVal
		out[productID] = snap
	}
}
