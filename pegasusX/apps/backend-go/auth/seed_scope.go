package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/spanner"
)

// EnsureDemoScopeLinks upserts SSMR demo warehouse/factory rows with factory linkage
// so FACTORY_ADMIN warehouse scope resolution succeeds in the isolated sandbox.
func EnsureDemoScopeLinks(ctx context.Context, client *spanner.Client, supplierID string) error {
	if client == nil || strings.TrimSpace(supplierID) == "" {
		return nil
	}

	warehouseID := strings.TrimSpace(os.Getenv("WAREHOUSE_DEMO_ID"))
	if warehouseID == "" {
		warehouseID = strings.TrimSpace(os.Getenv("SSMR_SMOKE_WAREHOUSE_ID"))
	}
	if warehouseID == "" {
		warehouseID = "wh-demo-1"
	}
	factoryID := strings.TrimSpace(os.Getenv("FACTORY_DEMO_ID"))
	if factoryID == "" {
		factoryID = "factory-demo-1"
	}

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]any{
				"TransferId":    "ssmr-wh-transfer-receive",
				"FactoryId":     factoryID,
				"SupplierId":    supplierID,
				"State":         "IN_TRANSIT",
				"TotalVolumeVU": 12.0,
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
			spanner.InsertOrUpdateMap("Factories", map[string]any{
				"FactoryId":  factoryID,
				"SupplierId": supplierID,
				"Name":       "SSMR Demo Factory",
				"IsActive":   true,
				"CreatedAt":  spanner.CommitTimestamp,
				"UpdatedAt":  spanner.CommitTimestamp,
			}),
			spanner.InsertOrUpdateMap("Warehouses", map[string]any{
				"WarehouseId":        warehouseID,
				"SupplierId":         supplierID,
				"Name":               "SSMR Demo Warehouse",
				"CoverageRadiusKm":   10.0,
				"PrimaryFactoryId":   factoryID,
				"IsActive":           true,
				"IsOnShift":          true,
				"CreatedAt":          spanner.CommitTimestamp,
				"UpdatedAt":          spanner.CommitTimestamp,
			}),
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("ensure demo scope links: %w", err)
	}
	return nil
}
