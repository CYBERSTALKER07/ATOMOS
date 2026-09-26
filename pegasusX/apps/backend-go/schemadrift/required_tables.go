package schemadrift

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/spanner"
)

// AssertRequiredTables fails closed when any RequiredProductTables row is missing.
func AssertRequiredTables(ctx context.Context, client *spanner.Client) error {
	if client == nil {
		return fmt.Errorf("schemadrift: nil spanner client")
	}
	var missing []string
	for _, table := range RequiredProductTables {
		ok, err := tableExists(ctx, client, table)
		if err != nil {
			return err
		}
		if !ok {
			missing = append(missing, table)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf(
		"schema drift: required product tables missing (apply schema/spanner.ddl): %s",
		strings.Join(missing, ", "),
	)
}

// AssertLiveSchema runs shop-closed column checks + required product tables.
func AssertLiveSchema(ctx context.Context, client *spanner.Client) error {
	if err := AssertShopClosedSchema(ctx, client); err != nil {
		return err
	}
	return AssertRequiredTables(ctx, client)
}
