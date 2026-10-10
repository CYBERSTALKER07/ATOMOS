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
	meta, err := LookupProductDispatchMeta(ctx, client, productIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]float64, len(meta))
	for id, m := range meta {
		if m.UnitVolumeVU > 0 {
			out[id] = m.UnitVolumeVU
		}
	}
	return out, nil
}

// productDispatchMeta carries catalog fields used at dispatch hydrate time.
type productDispatchMeta struct {
	UnitVolumeVU      float64
	HandlingClass     string
	RequiresColdChain bool
	IsHazardous       bool
}

// LookupProductDispatchMeta bulk-reads volume + handling flags for product IDs.
func LookupProductDispatchMeta(ctx context.Context, client *spanner.Client, productIDs []string) (map[string]productDispatchMeta, error) {
	out := make(map[string]productDispatchMeta, len(productIDs))
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
		SQL:    `SELECT ProductId, UnitVolumeVU, COALESCE(HandlingClass, 'GENERAL'), RequiresColdChain, IsHazardous FROM Products WHERE ProductId IN UNNEST(@ids)`,
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
			return nil, fmt.Errorf("lookup product dispatch meta: %w", err)
		}
		var productID string
		var meta productDispatchMeta
		if err := row.Columns(&productID, &meta.UnitVolumeVU, &meta.HandlingClass, &meta.RequiresColdChain, &meta.IsHazardous); err != nil {
			return nil, fmt.Errorf("scan product dispatch meta: %w", err)
		}
		if meta.UnitVolumeVU <= 0 {
			meta.UnitVolumeVU = 0 // leave absent in volume map
		}
		out[productID] = meta
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
	meta, err := LookupProductDispatchMeta(ctx, client, collect)
	if err != nil {
		return err
	}
	lookup := make(map[string]float64, len(meta))
	for id, m := range meta {
		if m.UnitVolumeVU > 0 {
			lookup[id] = m.UnitVolumeVU
		}
	}
	for _, row := range rows {
		if row.Order == nil {
			continue
		}
		vu, src := OrderVolumeVUWithSource(row.LineItemsRaw, lookup)
		row.Order.VolumeVU = vu
		row.Order.VolumeSource = src
		applyHandlingFlags(row.Order, row.LineItemsRaw, meta)
	}
	return nil
}

// applyHandlingFlags ORs cold/hazmat across line-item snapshots and product meta.
func applyHandlingFlags(ord *DispatchableOrder, lineItemsRaw []byte, meta map[string]productDispatchMeta) {
	if ord == nil {
		return
	}
	var items []order.LineItem
	_ = json.Unmarshal(lineItemsRaw, &items)
	handling := ""
	for _, item := range items {
		if item.RequiresColdChain {
			ord.RequiresColdChain = true
		}
		if item.IsHazardous {
			ord.IsHazardous = true
		}
		if hc := strings.TrimSpace(item.HandlingClass); hc != "" && handling == "" {
			handling = hc
		}
		sku := strings.TrimSpace(item.SKU)
		if m, ok := meta[sku]; ok {
			if m.RequiresColdChain {
				ord.RequiresColdChain = true
			}
			if m.IsHazardous {
				ord.IsHazardous = true
			}
			if handling == "" && strings.TrimSpace(m.HandlingClass) != "" {
				handling = m.HandlingClass
			}
		}
	}
	if ord.RequiresColdChain && handling == "" {
		handling = "COLD_CHAIN"
	} else if ord.IsHazardous && handling == "" {
		handling = "HAZARDOUS"
	}
	ord.HandlingClass = handling
}
