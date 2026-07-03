package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"google.golang.org/api/iterator"
)

// runPlanningBaselineSeed upserts at least one DemandForecastBaseline row for PX-PROD-3 local export.
func runPlanningBaselineSeed(ctx context.Context, cfg *bootstrap.Config) error {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return fmt.Errorf("spanner client: %w", err)
	}
	defer client.Close()

	supplierID, err := seedSupplierID(ctx, client, cfg)
	if err != nil {
		return err
	}

	warehouseID := strings.TrimSpace(envOr("SSMR_SMOKE_WAREHOUSE_ID", "ssmr-warehouse-1"))
	productID := strings.TrimSpace(envOr("SSMR_SMOKE_PRODUCT_ID", "SSMR-SKU-1"))
	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)

	if count, err := demandBaselineRowCount(ctx, client, supplierID, today); err != nil {
		return err
	} else if count > 0 {
		fmt.Println("PX_PLANNING_BASELINE_SEED_OK")
		return nil
	}

	if err := planning.WriteBaselineWithOutbox(ctx, client, now, planning.BaselineWriteInput{
		SupplierID:     supplierID,
		WarehouseID:    warehouseID,
		ProductID:      productID,
		ForecastDate:   today,
		BaselineQty:    42,
		Confidence:     0.75,
		Source:         "ssmr_seed",
		BaselineSource: "moving_average",
	}); err != nil {
		return fmt.Errorf("write baseline: %w", err)
	}

	fmt.Println("PX_PLANNING_BASELINE_SEED_OK")
	return nil
}

func seedSupplierID(ctx context.Context, client *spanner.Client, cfg *bootstrap.Config) (string, error) {
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT SupplierId FROM Suppliers WHERE Name = @name LIMIT 1`,
		Params: map[string]any{"name": cfg.SeedSupplierName},
	})
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return "", fmt.Errorf("seed supplier %q missing", cfg.SeedSupplierName)
		}
		return "", fmt.Errorf("query supplier: %w", err)
	}
	var supplierID string
	if err := row.Columns(&supplierID); err != nil {
		return "", fmt.Errorf("decode supplier: %w", err)
	}
	if strings.TrimSpace(supplierID) == "" {
		return "", fmt.Errorf("empty supplier id")
	}
	return supplierID, nil
}

func demandBaselineRowCount(ctx context.Context, client *spanner.Client, supplierID string, day time.Time) (int64, error) {
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*) FROM DemandForecastBaseline
		      WHERE SupplierId = @sid AND ForecastDate = @day`,
		Params: map[string]any{"sid": supplierID, "day": civil.DateOf(day)},
	})
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, err
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, err
	}
	return count, nil
}
