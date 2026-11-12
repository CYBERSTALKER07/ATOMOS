package order

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SnapMode determines how arbitrary quantities snap to case pack multiples.
type SnapMode string

const (
	SnapModeNearest SnapMode = "NEAREST"
	SnapModeCeil    SnapMode = "CEIL"
	SnapModeFloor   SnapMode = "FLOOR"
)

type productPackRule struct {
	SaleUnit     string
	UnitsPerPack int64
}

// SnapToPackMultiple snaps a raw order quantity to the nearest or directed multiple of pack size.
// If pack <= 1 or qty <= 0, returns qty (or 0 if negative).
// E.g. with pack=6, qty=7, mode="NEAREST" -> 6; qty=10 -> 12; mode="CEIL" -> 12.
func SnapToPackMultiple(qty int64, pack int64, mode SnapMode) int64 {
	if qty <= 0 {
		return 0
	}
	if pack <= 1 {
		return qty
	}

	remainder := qty % pack
	if remainder == 0 {
		return qty
	}

	lower := qty - remainder
	upper := lower + pack

	switch mode {
	case SnapModeFloor:
		return lower
	case SnapModeCeil:
		return upper
	case SnapModeNearest:
		fallthrough
	default:
		if remainder*2 >= pack {
			return upper
		}
		if lower == 0 {
			// Do not drop to 0 if retailer requested at least one item under nearest
			return upper
		}
		return lower
	}
}

func validatePackMultiples(ctx context.Context, client *spanner.Client, items []LineItem) map[string]string {
	if client == nil || len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	keySets := make([]spanner.KeySet, 0, len(items))
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		if _, exists := seen[sku]; !exists {
			seen[sku] = struct{}{}
			keySets = append(keySets, spanner.Key{sku})
		}
	}
	if len(keySets) == 0 {
		return nil
	}

	rules := make(map[string]productPackRule, len(keySets))
	iter := client.Single().Read(ctx, "Products", spanner.KeySets(keySets...), []string{"ProductId", "SaleUnit", "UnitsPerPack"})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Fail-soft on read errors to match legacy single-row behavior when product metadata is unavailable.
			break
		}
		var sku, saleUnit string
		var unitsPerPack spanner.NullInt64
		if err := row.Columns(&sku, &saleUnit, &unitsPerPack); err != nil {
			continue
		}
		if !unitsPerPack.Valid {
			continue
		}
		rules[strings.TrimSpace(sku)] = productPackRule{
			SaleUnit:     saleUnit,
			UnitsPerPack: unitsPerPack.Int64,
		}
	}
	return evaluatePackMultipleRules(items, rules)
}

func evaluatePackMultipleRules(items []LineItem, rules map[string]productPackRule) map[string]string {
	if len(items) == 0 || len(rules) == 0 {
		return nil
	}
	errs := make(map[string]string)
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		rule, ok := rules[sku]
		if !ok {
			continue
		}
		saleUnit := strings.ToUpper(strings.TrimSpace(rule.SaleUnit))
		if saleUnit != "CASE" && saleUnit != "BLOCK" {
			continue
		}
		if rule.UnitsPerPack <= 1 {
			continue
		}
		if item.Quantity%rule.UnitsPerPack != 0 {
			errs[sku] = fmt.Sprintf("quantity must be a multiple of %d (%s)", rule.UnitsPerPack, saleUnit)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}
