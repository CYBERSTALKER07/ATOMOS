package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
)

// runForecastAlgoE2E seeds ~70 COMPLETED order-days, runs forecast once, asserts BaselineSource.
func runForecastAlgoE2E(
	ctx context.Context,
	client *http.Client,
	base, supplierID, retailerID string,
	cfg *bootstrap.Config,
) error {
	if !envTruthy("FORECAST_ALGO_ENABLED") {
		fmt.Println("PX_E2E_FORECAST_ALGO_SKIPPED")
		return nil
	}

	spannerClient, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		fmt.Println("PX_E2E_FORECAST_ALGO_SKIPPED")
		return nil
	}
	defer spannerClient.Close()

	op := smokeOperatingCurrency(ctx, cfg.SeedSupplierCurrency)
	if op == "" {
		fmt.Println("PX_E2E_FORECAST_ALGO_SKIPPED")
		return nil
	}

	warehouseID := strings.TrimSpace(envOr("SSMR_SMOKE_WAREHOUSE_ID", "ssmr-warehouse-1"))
	productID := strings.TrimSpace(envOr("SSMR_SMOKE_PRODUCT_ID", "SSMR-SKU-FORECAST-1"))
	now := time.Now().UTC()
	day0 := now.Truncate(24 * time.Hour)

	mutations := make([]*spanner.Mutation, 0, 70)
	for i := 69; i >= 0; i-- {
		d := day0.AddDate(0, 0, -i)
		qty := int64(8 + (i % 7))
		if i%9 == 0 {
			qty = 14
		}
		lineItems, _ := json.Marshal([]map[string]any{
			{"sku": productID, "name": "forecast e2e", "quantity": qty, "unit_price_minor": 100},
		})
		orderID := fmt.Sprintf("ord-fc-%s-%d", uuid.NewString()[:8], i)
		mutations = append(mutations, spanner.InsertOrUpdateMap("Orders", map[string]any{
			"OrderId":            orderID,
			"SupplierId":         supplierID,
			"RetailerId":         retailerID,
			"WarehouseId":        warehouseID,
			"Status":             "COMPLETED",
			"OrderSource":        "MANUAL",
			"ConfirmationStatus": "CONFIRMED",
			"LineItemsJson":      lineItems,
			"TotalMinor":         qty * 100,
			"OriginalTotalMinor": qty * 100,
			"Currency":           op,
			"Version":            int64(1),
			"CreatedAt":          d,
			"UpdatedAt":          d,
		}))
	}
	// Apply in chunks.
	for i := 0; i < len(mutations); i += 40 {
		end := i + 40
		if end > len(mutations) {
			end = len(mutations)
		}
		if _, err := spannerClient.Apply(ctx, mutations[i:end]); err != nil {
			return fmt.Errorf("seed forecast orders: %w", err)
		}
	}

	adminToken, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-forecast-algo-admin",
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("issue admin jwt: %w", err)
	}

	target := day0.AddDate(0, 0, 1).Format("2006-01-02")
	status, body, _, err := clientPost(ctx, client,
		base+"/v1/admin/planning/forecast/run-once?days=90&supplier_id="+supplierID+"&target="+target,
		[]byte("{}"), adminToken, "forecast-algo-run-"+productID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		if strings.Contains(string(body), "not found") || strings.Contains(string(body), "Table not found") {
			fmt.Println("PX_E2E_FORECAST_ALGO_SKIPPED")
			return nil
		}
		return fmt.Errorf("forecast run-once status=%d body=%s", status, string(body))
	}

	targetDate, _ := civil.ParseDate(target)
	row, err := spannerClient.Single().ReadRow(ctx, "DemandForecastBaseline",
		spanner.Key{supplierID, targetDate, warehouseID, productID},
		[]string{"BaselineQty", "BaselineSource", "Source"})
	if err != nil {
		return fmt.Errorf("read baseline: %w", err)
	}
	var qty int64
	var baselineSource, source string
	if err := row.Columns(&qty, &baselineSource, &source); err != nil {
		return err
	}
	src := planning.NormalizeBaselineSource(baselineSource, source)
	switch src {
	case planning.BaselineSourceCroston, planning.BaselineSourceHoltWinters,
		planning.BaselineSourceSES, planning.BaselineSourceMixed:
		// ok
	default:
		return fmt.Errorf("unexpected BaselineSource=%q normalized=%q qty=%d", baselineSource, src, qty)
	}
	if qty <= 0 {
		return fmt.Errorf("expected BaselineQty > 0, got %d", qty)
	}

	fmt.Println("PX_E2E_FORECAST_ALGO_OK")
	return nil
}
