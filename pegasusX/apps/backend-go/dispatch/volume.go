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

const (
	VolumeSourceCatalog   = "catalog"
	VolumeSourceDefault10 = "default_1_0"
)

type volumeEnrichRow struct {
	Order        *DispatchableOrder
	LineItemsRaw []byte
}

// VolumeSourceCounts reports how order volume was derived (honesty for dispatch preview).
type VolumeSourceCounts struct {
	Catalog    int `json:"catalog"`
	Default1_0 int `json:"default_1_0"`
}

// CountVolumeSources tallies VolumeSource on hydrated orders.
func CountVolumeSources(orders []DispatchableOrder) VolumeSourceCounts {
	var c VolumeSourceCounts
	for _, o := range orders {
		switch o.VolumeSource {
		case VolumeSourceCatalog:
			c.Catalog++
		default:
			c.Default1_0++
		}
	}
	return c
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
			continue // leave absent so OrderVolumeVU marks default
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
	total, _ := OrderVolumeVUWithSource(lineItemsRaw, lookup)
	return total
}

// OrderVolumeVUWithSource returns volume and whether any line used catalog VU (not default 1.0).
func OrderVolumeVUWithSource(lineItemsRaw []byte, lookup map[string]float64) (float64, string) {
	if len(lineItemsRaw) == 0 {
		return defaultUnitVolumeVU, VolumeSourceDefault10
	}
	var items []order.LineItem
	if err := json.Unmarshal(lineItemsRaw, &items); err != nil || len(items) == 0 {
		return defaultUnitVolumeVU, VolumeSourceDefault10
	}
	var total float64
	usedCatalog := false
	usedDefault := false
	for _, item := range items {
		qty := float64(item.Quantity)
		if qty <= 0 {
			qty = 1
		}
		unitVU := item.UnitVolumeVU
		fromCatalog := unitVU > 0
		if unitVU <= 0 {
			if lookup != nil {
				if v, ok := lookup[strings.TrimSpace(item.SKU)]; ok && v > 0 {
					unitVU = v
					fromCatalog = true
				}
			}
		}
		if unitVU <= 0 {
			unitVU = defaultUnitVolumeVU
			fromCatalog = false
		}
		if fromCatalog {
			usedCatalog = true
		} else {
			usedDefault = true
		}
		total += qty * unitVU
	}
	if total <= 0 {
		return defaultUnitVolumeVU, VolumeSourceDefault10
	}
	if usedCatalog && !usedDefault {
		return total, VolumeSourceCatalog
	}
	if usedCatalog {
		// Mixed: still report catalog present for honesty (partial).
		return total, VolumeSourceCatalog
	}
	return total, VolumeSourceDefault10
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
		vu, src := OrderVolumeVUWithSource(row.LineItemsRaw, lookup)
		row.Order.VolumeVU = vu
		row.Order.VolumeSource = src
	}
	return nil
}
