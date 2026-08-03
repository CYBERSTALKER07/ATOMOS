// Package schemadrift asserts Spanner objects required by live product paths.
package schemadrift

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ShopClosedOrdersColumns must exist on Orders for grace/proximity/partial paths.
var ShopClosedOrdersColumns = []string{
	"ShopClosedAt",
	"ShopClosedReason",
	"ShopClosedGraceEndsAt",
	"ShopClosedResolution",
	"PartialDelivery",
	"ProximityUnlockedAt",
	"ProximityMethod",
}

const shopClosedLogTable = "OrderShopClosedLog"

// AssertShopClosedSchema fails closed when shop-closed / proximity columns or
// OrderShopClosedLog are missing (migration 20260729_shop_closed_proximity_partial).
func AssertShopClosedSchema(ctx context.Context, client *spanner.Client) error {
	if client == nil {
		return fmt.Errorf("schemadrift: nil spanner client")
	}

	missingCols, err := missingColumns(ctx, client, "Orders", ShopClosedOrdersColumns)
	if err != nil {
		return err
	}
	hasLog, err := tableExists(ctx, client, shopClosedLogTable)
	if err != nil {
		return err
	}

	var missing []string
	for _, c := range missingCols {
		missing = append(missing, "Orders."+c)
	}
	if !hasLog {
		missing = append(missing, shopClosedLogTable)
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf(
		"schema drift: shop-closed objects missing (apply schema/migrations/20260729_shop_closed_proximity_partial.ddl): %s",
		strings.Join(missing, ", "),
	)
}

func missingColumns(ctx context.Context, client *spanner.Client, table string, required []string) ([]string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COLUMN_NAME
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_NAME = @table_name
  AND COLUMN_NAME IN UNNEST(@required_columns)`,
		Params: map[string]any{
			"table_name":       table,
			"required_columns": required,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	found := make(map[string]bool, len(required))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("schemadrift: query columns for %s: %w", table, err)
		}
		var name string
		if err := row.Columns(&name); err != nil {
			return nil, fmt.Errorf("schemadrift: decode column: %w", err)
		}
		found[name] = true
	}

	var missing []string
	for _, col := range required {
		if !found[col] {
			missing = append(missing, col)
		}
	}
	return missing, nil
}

func tableExists(ctx context.Context, client *spanner.Client, table string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT TABLE_NAME
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_NAME = @table_name
LIMIT 1`,
		Params: map[string]any{"table_name": table},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("schemadrift: query tables for %s: %w", table, err)
	}
	return true, nil
}
