package order

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const defaultUnitVolumeVU = 1.0

// productSnapshot captures catalog fields needed at order time.
type productSnapshot struct {
	unitVolumeVU      float64
	handlingClass     string
	requiresColdChain bool
	isHazardous       bool
	isPerishable      bool
	storageTempMinC   *float64
	storageTempMaxC   *float64
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
		if items[i].UnitVolumeVU <= 0 {
			items[i].UnitVolumeVU = snap.unitVolumeVU
		}
		items[i].HandlingClass = snap.handlingClass
		items[i].RequiresColdChain = snap.requiresColdChain
		items[i].IsHazardous = snap.isHazardous
		items[i].IsPerishable = snap.isPerishable
		items[i].StorageTempMinC = snap.storageTempMinC
		items[i].StorageTempMaxC = snap.storageTempMaxC
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
		SQL:    `SELECT ProductId, UnitVolumeVU, HandlingClass, RequiresColdChain, IsHazardous, IsPerishable, StorageTempMinC, StorageTempMaxC FROM Products WHERE ProductId IN UNNEST(@ids)`,
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
		var unitVU spanner.NullFloat64
		var handlingClass spanner.NullString
		var storageTempMinC, storageTempMaxC spanner.NullFloat64
		if err := row.Columns(&productID, &unitVU, &handlingClass, &snap.requiresColdChain, &snap.isHazardous, &snap.isPerishable, &storageTempMinC, &storageTempMaxC); err != nil {
			return nil, fmt.Errorf("scan product snapshot: %w", err)
		}
		if unitVU.Valid {
			snap.unitVolumeVU = unitVU.Float64
		} else {
			snap.unitVolumeVU = defaultUnitVolumeVU
		}
		snap.handlingClass = handlingClass.StringVal
		if storageTempMinC.Valid {
			v := storageTempMinC.Float64
			snap.storageTempMinC = &v
		}
		if storageTempMaxC.Valid {
			v := storageTempMaxC.Float64
			snap.storageTempMaxC = &v
		}
		if snap.unitVolumeVU <= 0 {
			snap.unitVolumeVU = defaultUnitVolumeVU
		}
		out[productID] = snap
	}
}
