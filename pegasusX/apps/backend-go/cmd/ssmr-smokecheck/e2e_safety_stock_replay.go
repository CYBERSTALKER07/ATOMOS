package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// runSafetyStockReplayE2E seeds ~60d COMPLETED demand and calls ADMIN fill-rate replay.
func runSafetyStockReplayE2E(
	ctx context.Context,
	client *http.Client,
	base, supplierID, retailerID string,
	cfg *bootstrap.Config,
) error {
	if !envTruthy("SAFETY_STOCK_V2_ENABLED") {
		fmt.Println("PX_E2E_SAFETY_STOCK_REPLAY_SKIPPED")
		return nil
	}

	spannerClient, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		fmt.Println("PX_E2E_SAFETY_STOCK_REPLAY_SKIPPED")
		return nil
	}
	defer spannerClient.Close()

	op := smokeOperatingCurrency(ctx, cfg.SeedSupplierCurrency)
	if op == "" {
		fmt.Println("PX_E2E_SAFETY_STOCK_REPLAY_SKIPPED")
		return nil
	}

	warehouseID := strings.TrimSpace(envOr("SSMR_SMOKE_WAREHOUSE_ID", "ssmr-warehouse-1"))
	productID := "SSMR-SKU-SS-REPLAY-" + uuid.NewString()[:8]
	now := time.Now().UTC().Truncate(24 * time.Hour)

	mutations := make([]*spanner.Mutation, 0, 60)
	for i := 59; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		qty := int64(8 + (i % 6))
		if i%7 == 0 {
			qty = 22
		}
		lineItems, _ := json.Marshal([]map[string]any{
			{"sku": productID, "name": "ss replay", "quantity": qty, "unit_price_minor": 100},
		})
		orderID := fmt.Sprintf("ord-ssr-%s-%d", uuid.NewString()[:8], i)
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
	for i := 0; i < len(mutations); i += 30 {
		end := i + 30
		if end > len(mutations) {
			end = len(mutations)
		}
		if _, err := spannerClient.Apply(ctx, mutations[i:end]); err != nil {
			return fmt.Errorf("seed replay orders: %w", err)
		}
	}

	adminToken, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-safety-stock-replay-admin",
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

	url := fmt.Sprintf("%s/v1/admin/planning/safety-stock/replay?days=60&supplier_id=%s", base, supplierID)
	status, body, _, err := clientPost(ctx, client, url, []byte("{}"), adminToken, "ss-replay-"+productID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		if strings.Contains(string(body), "not found") || status == http.StatusNotFound {
			fmt.Println("PX_E2E_SAFETY_STOCK_REPLAY_SKIPPED")
			return nil
		}
		return fmt.Errorf("safety-stock replay status=%d body=%s", status, string(body))
	}

	var resp struct {
		OK       bool `json:"ok"`
		SKUCount int  `json:"sku_count"`
		Legacy   struct {
			UnitFillRate float64 `json:"unit_fill_rate"`
		} `json:"legacy"`
		V2 struct {
			UnitFillRate float64 `json:"unit_fill_rate"`
		} `json:"v2"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode replay: %w", err)
	}
	if !resp.OK || resp.SKUCount < 1 {
		return fmt.Errorf("replay expected sku_count>=1 ok=true got sku=%d ok=%v body=%s", resp.SKUCount, resp.OK, string(body))
	}
	if resp.Legacy.UnitFillRate < 0 || resp.V2.UnitFillRate < 0 {
		return fmt.Errorf("invalid fill rates legacy=%v v2=%v", resp.Legacy.UnitFillRate, resp.V2.UnitFillRate)
	}

	fmt.Println("PX_E2E_SAFETY_STOCK_REPLAY_OK")
	return nil
}
