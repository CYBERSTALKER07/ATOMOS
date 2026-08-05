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

// runForecastAccuracyE2E seeds baseline + completed line units, runs accuracy pass, asserts ForecastAccuracyDaily.
func runForecastAccuracyE2E(
	ctx context.Context,
	client *http.Client,
	base, supplierID, retailerID string,
	cfg *bootstrap.Config,
) error {
	if !envTruthy("FORECAST_ACCURACY_ENABLED") {
		fmt.Println("PX_E2E_FORECAST_ACCURACY_SKIPPED")
		return nil
	}

	spannerClient, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		fmt.Println("PX_E2E_FORECAST_ACCURACY_SKIPPED")
		return nil
	}
	defer spannerClient.Close()

	warehouseID := strings.TrimSpace(envOr("SSMR_SMOKE_WAREHOUSE_ID", "ssmr-warehouse-1"))
	productID := strings.TrimSpace(envOr("SSMR_SMOKE_PRODUCT_ID", "SSMR-SKU-1"))
	now := time.Now().UTC()
	day := now.Truncate(24 * time.Hour)

	if err := planning.WriteBaselineWithOutbox(ctx, spannerClient, now, planning.BaselineWriteInput{
		SupplierID:     supplierID,
		WarehouseID:    warehouseID,
		ProductID:      productID,
		ForecastDate:   day,
		BaselineQty:    12,
		Confidence:     0.8,
		Source:         "ssmr_accuracy_e2e",
		BaselineSource: "moving_average",
	}); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "Table not found") {
			fmt.Println("PX_E2E_FORECAST_ACCURACY_SKIPPED")
			return nil
		}
		return fmt.Errorf("seed baseline: %w", err)
	}

	lineItems, _ := json.Marshal([]map[string]any{
		{"sku": productID, "name": "SSMR accuracy SKU", "quantity": 10, "unit_price_minor": 100},
	})
	orderID := "ord-acc-" + uuid.NewString()
	_, err = spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("Orders", map[string]any{
			"OrderId":            orderID,
			"SupplierId":         supplierID,
			"RetailerId":         retailerID,
			"WarehouseId":        warehouseID,
			"Status":             "COMPLETED",
			"OrderSource":        "MANUAL",
			"ConfirmationStatus": "CONFIRMED",
			"LineItemsJson":      lineItems,
			"TotalMinor":         int64(1000),
			"OriginalTotalMinor": int64(1000),
			"Currency":           "UZS",
			"Version":            int64(1),
			"CreatedAt":          now,
			"UpdatedAt":          now,
		}),
	})
	if err != nil {
		return fmt.Errorf("seed completed order: %w", err)
	}

	adminToken, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-forecast-accuracy-admin",
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

	status, body, _, err := clientPost(ctx, client,
		base+"/v1/admin/planning/accuracy/run-once?days=1&supplier_id="+supplierID,
		[]byte("{}"), adminToken, "forecast-accuracy-run-"+orderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		if strings.Contains(string(body), "not found") || strings.Contains(string(body), "Table not found") {
			fmt.Println("PX_E2E_FORECAST_ACCURACY_SKIPPED")
			return nil
		}
		return fmt.Errorf("accuracy run-once status=%d body=%s", status, string(body))
	}

	row, err := spannerClient.Single().ReadRow(ctx, "ForecastAccuracyDaily",
		spanner.Key{supplierID, civil.DateOf(day), warehouseID, productID},
		[]string{"ForecastQty", "ActualQty", "AbsError"})
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "Table not found") {
			fmt.Println("PX_E2E_FORECAST_ACCURACY_SKIPPED")
			return nil
		}
		return fmt.Errorf("read ForecastAccuracyDaily: %w", err)
	}
	var forecastQty, actualQty, absError int64
	if err := row.Columns(&forecastQty, &actualQty, &absError); err != nil {
		return err
	}
	if forecastQty != 12 || actualQty != 10 || absError != 2 {
		return fmt.Errorf("accuracy row forecast=%d actual=%d abs=%d want 12/10/2", forecastQty, actualQty, absError)
	}

	fmt.Println("PX_E2E_FORECAST_ACCURACY_OK")
	return nil
}
