package order

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
)

func validatePackMultiples(ctx context.Context, client *spanner.Client, items []LineItem) map[string]string {
	if client == nil {
		return nil
	}
	errs := make(map[string]string)
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		row, err := client.Single().ReadRow(ctx, "Products", spanner.Key{sku},
			[]string{"SaleUnit", "UnitsPerPack"})
		if err != nil {
			continue
		}
		var saleUnit string
		var unitsPerPack spanner.NullInt64
		if err := row.Columns(&saleUnit, &unitsPerPack); err != nil {
			continue
		}
		saleUnit = strings.ToUpper(strings.TrimSpace(saleUnit))
		if saleUnit != "CASE" && saleUnit != "BLOCK" {
			continue
		}
		if !unitsPerPack.Valid || unitsPerPack.Int64 <= 1 {
			continue
		}
		pack := unitsPerPack.Int64
		if item.Quantity%pack != 0 {
			errs[sku] = fmt.Sprintf("quantity must be a multiple of %d (%s)", pack, saleUnit)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}
