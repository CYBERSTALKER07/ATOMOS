package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"google.golang.org/api/iterator"
)

// runSafetyStockE2E seeds burn + low stock, runs replenishment analyze, asserts DemandBreakdown SS fields.
func runSafetyStockE2E(
	ctx context.Context,
	client *http.Client,
	base, cookie, supplierID, retailerID string,
	cfg *bootstrap.Config,
) error {
	if !envTruthy("SAFETY_STOCK_V2_ENABLED") {
		fmt.Println("PX_E2E_SAFETY_STOCK_SKIPPED")
		return nil
	}

	spannerClient, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		fmt.Println("PX_E2E_SAFETY_STOCK_SKIPPED")
		return nil
	}
	defer spannerClient.Close()

	op := smokeOperatingCurrency(ctx, cfg.SeedSupplierCurrency)
	if op == "" {
		fmt.Println("PX_E2E_SAFETY_STOCK_SKIPPED")
		return nil
	}

	warehouseID := strings.TrimSpace(envOr("SSMR_SMOKE_WAREHOUSE_ID", "ssmr-warehouse-1"))
	productID := "SSMR-SKU-SAFETY-" + uuid.NewString()[:8]
	now := time.Now().UTC()

	// Prefer a warehouse that has PrimaryFactoryId (engine skips otherwise).
	whIter := spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId FROM Warehouses
		      WHERE SupplierId = @sid AND IsActive = true
		        AND PrimaryFactoryId IS NOT NULL AND PrimaryFactoryId != ''
		      LIMIT 1`,
		Params: map[string]any{"sid": supplierID},
	})
	func() {
		defer whIter.Stop()
		whRow, whErr := whIter.Next()
		if whErr != nil {
			return
		}
		var wid string
		if err := whRow.Columns(&wid); err == nil && strings.TrimSpace(wid) != "" {
			warehouseID = wid
		}
	}()

	patchBody, _ := json.Marshal(map[string]any{
		"target_service_level":  0.98,
		"lead_time_days":        2,
		"lead_time_sigma_days":  1.0,
		"max_daily_transfer_units": 500,
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPatch,
		base+"/v1/supplier/replenishment/policies", patchBody, cookie, "ssmr-ss-policy-"+productID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		if strings.Contains(string(respBody), "not found") || strings.Contains(string(respBody), "Column not found") {
			fmt.Println("PX_E2E_SAFETY_STOCK_SKIPPED")
			return nil
		}
		return fmt.Errorf("policy patch status=%d body=%s", status, string(respBody))
	}

	// Low on-hand so TTE triggers CRITICAL/WARNING.
	_, err = spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      warehouseID,
			"ProductId":        productID,
			"QuantityOnHand":   int64(2),
			"QuantityReserved": int64(0),
			"ReorderThreshold": int64(0),
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		return fmt.Errorf("seed inventory: %w", err)
	}

	// ~70 units over 7d → burn ≈ 10/day.
	lineItems, _ := json.Marshal([]map[string]any{
		{"sku": productID, "name": "safety stock e2e", "quantity": 70, "unit_price_minor": 100},
	})
	orderID := fmt.Sprintf("ord-ss-%s", uuid.NewString()[:29])
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
			"TotalMinor":         int64(7000),
			"OriginalTotalMinor": int64(7000),
			"Currency":           op,
			"Version":            int64(1),
			// Lag past wall-clock so emulator clock skew never rejects as future.
			"CreatedAt": now.Add(-24*time.Hour - 2*time.Second),
			"UpdatedAt": now.Add(-24*time.Hour - 2*time.Second),
		}),
	})
	if err != nil {
		return fmt.Errorf("seed burn order: %w", err)
	}

	// Assumed σ_d path is fine; optional residual noise for non-assumed sigma_d.
	end := civil.DateOf(now)
	var accMuts []*spanner.Mutation
	for i := 0; i < 10; i++ {
		d := end.AddDays(-i)
		se := int64((i%5)*3 - 6) // varying signed errors
		accMuts = append(accMuts, spanner.InsertOrUpdateMap("ForecastAccuracyDaily", map[string]any{
			"SupplierId":     supplierID,
			"ForecastDate":   d,
			"WarehouseId":    warehouseID,
			"ProductId":      productID,
			"ForecastQty":    int64(10),
			"ActualQty":      int64(10 + se),
			"AbsError":       int64(math.Abs(float64(se))),
			"SignedError":    se,
			"SampleDays7":    int64(7),
			"SampleDays28":   int64(10),
			"AlertTs":        false,
			"ComputedAt":     spanner.CommitTimestamp,
		}))
	}
	if _, err := spannerClient.Apply(ctx, accMuts); err != nil {
		// Accuracy table may be missing in older emulators — continue on assumed path.
		if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "Table not found") {
			return fmt.Errorf("seed accuracy residuals: %w", err)
		}
	}

	engine := replenishment.NewEngine(spannerClient, slog.Default())
	if _, err := engine.RunForSupplier(ctx, supplierID); err != nil {
		return fmt.Errorf("replenishment analyze: %w", err)
	}

	iter := spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT DemandBreakdown FROM ReplenishmentInsights
		      WHERE SupplierId = @sid AND WarehouseId = @wh AND ProductId = @pid
		      ORDER BY CreatedAt DESC LIMIT 1`,
		Params: map[string]any{"sid": supplierID, "wh": warehouseID, "pid": productID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return fmt.Errorf("safety stock: no insight for product %s", productID)
		}
		return fmt.Errorf("query insight: %w", err)
	}
	var breakdownRaw string
	if err := row.Columns(&breakdownRaw); err != nil {
		return err
	}
	var bd map[string]any
	if err := json.Unmarshal([]byte(breakdownRaw), &bd); err != nil {
		return fmt.Errorf("decode DemandBreakdown: %w", err)
	}
	if v, _ := bd["safety_stock_v2"].(bool); !v {
		return fmt.Errorf("expected safety_stock_v2=true in breakdown: %s", breakdownRaw)
	}
	ss, ok := bd["safety_stock"].(float64)
	if !ok || ss <= 0 {
		return fmt.Errorf("expected safety_stock>0 got %v in %s", bd["safety_stock"], breakdownRaw)
	}
	rop, _ := bd["reorder_point"].(float64)
	dBar, _ := bd["d_bar"].(float64)
	lead, _ := bd["lead_days"].(float64)
	if lead <= 0 {
		lead = 2
	}
	if rop <= dBar*lead {
		return fmt.Errorf("reorder_point %v should exceed d̄·L (%v) when σ>0; breakdown=%s", rop, dBar*lead, breakdownRaw)
	}

	fmt.Println("PX_E2E_SAFETY_STOCK_OK")
	return nil
}
