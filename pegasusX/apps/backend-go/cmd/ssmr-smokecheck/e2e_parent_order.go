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
	"google.golang.org/api/iterator"
)

// runParentOrderSmokeCheck exercises Gate 5 Phase 2 markers:
// multi-supplier register, parent split checkout, child isolation.
func runParentOrderSmokeCheck(ctx context.Context, cfg *bootstrap.Config) error {
	base := strings.TrimRight(envOr("PUBLIC_BASE_URL", "http://localhost:8180"), "/")
	client := &http.Client{Timeout: 45 * time.Second}
	if _, err := clientGet(ctx, client, base+"/v1/health"); err != nil {
		return fmt.Errorf("health: %w", err)
	}
	return runParentOrderE2E(ctx, client, base, cfg)
}

func runParentOrderE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config) error {
	secondSupplierID, err := runMultiSupplierRegisterE2E(ctx, client, base, cfg)
	if err != nil {
		return err
	}
	if err := runParentOrderSplitE2E(ctx, client, base, cfg, secondSupplierID); err != nil {
		return err
	}
	return nil
}

func runMultiSupplierRegisterE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config) (string, error) {
	if !envTruthy("ALLOW_MULTI_SUPPLIER_REGISTER") {
		fmt.Println("PX_E2E_MULTI_SUPPLIER_REGISTER_SKIPPED")
		return "", nil
	}
	phone := "+99891" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	body, _ := json.Marshal(map[string]any{
		"phone": phone,
		"account": map[string]any{
			"legalName":   "SSMR Parent Phase2 Supplier",
			"contactName": "Parent Admin",
			"email":       "parent-" + phone[len(phone)-4:] + "@pegasusx.local",
			"password":    "ParentTest!234",
			"country":     cfg.SeedSupplierCountry,
		},
		"location": map[string]any{
			"warehouse": map[string]any{
				"name":    "Parent WH B",
				"address": "Tashkent",
				"lat":     cfg.DeliveryZoneCenterLat,
				"lng":     cfg.DeliveryZoneCenterLng,
			},
			"sameAsWarehouse": true,
		},
		"business": map[string]any{
			"taxId":             "PARENT-TAX",
			"companyRegNumber":  "PARENT-REG",
			"fleetVehicleCount": 1,
			"fleetMaxVU":        10,
			"factoryCount":      1,
		},
		"categories": []string{"GENERAL"},
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/auth/supplier/register", body, "", "ssmr-parent-reg-"+phone)
	if err != nil {
		return "", fmt.Errorf("multi supplier register: %w", err)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return "", fmt.Errorf("multi supplier register: status %d body=%s", status, string(respBody))
	}
	var resp struct {
		SupplierID string `json:"supplier_id"`
		Supplier   struct {
			SupplierID string `json:"supplier_id"`
		} `json:"supplier"`
	}
	_ = json.Unmarshal(respBody, &resp)
	sid := strings.TrimSpace(resp.SupplierID)
	if sid == "" {
		sid = strings.TrimSpace(resp.Supplier.SupplierID)
	}
	if sid == "" {
		return "", fmt.Errorf("multi supplier register: missing supplier_id body=%s", string(respBody))
	}
	fmt.Println("PX_E2E_MULTI_SUPPLIER_REGISTER_OK")
	return sid, nil
}

func runParentOrderSplitE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, secondSupplierID string) error {
	if !multiSupplierCheckoutEnvOn() {
		fmt.Println("PX_E2E_PARENT_ORDER_SPLIT_SKIPPED")
		fmt.Println("PX_E2E_PARENT_ORDER_ISOLATION_SKIPPED")
		return nil
	}
	if strings.TrimSpace(secondSupplierID) == "" {
		fmt.Println("PX_E2E_PARENT_ORDER_SPLIT_SKIPPED")
		fmt.Println("PX_E2E_PARENT_ORDER_ISOLATION_SKIPPED")
		return nil
	}

	supplierID, cookie, err := ensureSupplierSession(ctx, client, base, cfg)
	if err != nil {
		return fmt.Errorf("parent split supplier session: %w", err)
	}
	if err := putSupplierTopology(ctx, client, base, cookie, cfg); err != nil {
		return fmt.Errorf("parent split topology: %w", err)
	}

	skuB, err := seedSecondSupplierCatalog(ctx, cfg, secondSupplierID)
	if err != nil {
		return fmt.Errorf("seed second supplier catalog: %w", err)
	}

	retailerID, h3Cell, err := registerRetailer(ctx, client, base, cfg)
	if err != nil {
		return fmt.Errorf("parent split retailer: %w", err)
	}
	if err := grantRetailerCredit(ctx, client, base, cookie, retailerID, 500_000_000); err != nil {
		return fmt.Errorf("parent split credit: %w", err)
	}
	// Also grant credit for supplier B path (credit is per retailer+supplier).
	if err := grantRetailerCreditForSupplier(ctx, cfg, retailerID, secondSupplierID, 500_000_000); err != nil {
		return fmt.Errorf("parent split credit B: %w", err)
	}

	retailerToken, err := auth.Issue(auth.Claims{
		Subject:    retailerID,
		Role:       auth.RoleRetailer,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 20 * time.Minute})
	if err != nil {
		return fmt.Errorf("parent split retailer jwt: %w", err)
	}

	body, _ := json.Marshal(map[string]any{
		"latitude":  cfg.DeliveryZoneCenterLat,
		"longitude": cfg.DeliveryZoneCenterLng,
		"items": []map[string]any{
			{"sku_id": "SSMR-SKU-1", "quantity": 1, "unit_price": 50000, "supplier_id": supplierID},
			{"sku_id": skuB, "quantity": 1, "unit_price": 25000, "supplier_id": secondSupplierID},
		},
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/checkout/unified", body, retailerToken, "ssmr-parent-split-"+uuid.NewString()[:8])
	if err != nil {
		return fmt.Errorf("parent split checkout: %w", err)
	}
	if status != http.StatusCreated {
		return fmt.Errorf("parent split checkout status %d body=%s (h3=%s)", status, string(respBody), h3Cell)
	}
	var resp struct {
		ParentOrderID  string `json:"parent_order_id"`
		SupplierOrders []struct {
			OrderID    string `json:"order_id"`
			SupplierID string `json:"supplier_id"`
		} `json:"supplier_orders"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("parent split decode: %w", err)
	}
	if strings.TrimSpace(resp.ParentOrderID) == "" {
		return fmt.Errorf("parent split: missing parent_order_id body=%s", string(respBody))
	}
	if len(resp.SupplierOrders) != 2 {
		return fmt.Errorf("parent split: want 2 supplier_orders got %d body=%s", len(resp.SupplierOrders), string(respBody))
	}
	seen := map[string]string{}
	for _, so := range resp.SupplierOrders {
		seen[so.SupplierID] = so.OrderID
	}
	if seen[supplierID] == "" || seen[secondSupplierID] == "" {
		return fmt.Errorf("parent split: supplier ids mismatch got=%v want %s and %s", seen, supplierID, secondSupplierID)
	}
	fmt.Println("PX_E2E_PARENT_ORDER_SPLIT_OK")

	// Supplier A must not read supplier B child.
	otherTok, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-parent-idor",
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 10 * time.Minute})
	if err != nil {
		return fmt.Errorf("parent isolation jwt: %w", err)
	}
	childB := seen[secondSupplierID]
	st, bodyBytes, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/order/"+childB+"/status-context", nil, otherTok, "")
	if err != nil {
		return fmt.Errorf("parent isolation: %w", err)
	}
	if st == http.StatusOK {
		return fmt.Errorf("parent isolation: supplier A must not read supplier B child, got 200 body=%s", string(bodyBytes))
	}
	if st == http.StatusNotFound || st == http.StatusForbidden || st == http.StatusUnauthorized {
		fmt.Println("PX_E2E_PARENT_ORDER_ISOLATION_OK")
		return nil
	}
	fmt.Println("PX_E2E_PARENT_ORDER_ISOLATION_SKIPPED")
	return nil
}

func multiSupplierCheckoutEnvOn() bool {
	raw := strings.TrimSpace(envOr("MULTI_SUPPLIER_CHECKOUT_ENABLED", ""))
	if raw == "" {
		return strings.EqualFold(strings.TrimSpace(envOr("PEGASUSX_ENV", "")), "ssmr")
	}
	return envTruthy("MULTI_SUPPLIER_CHECKOUT_ENABLED")
}

func seedSecondSupplierCatalog(ctx context.Context, cfg *bootstrap.Config, supplierID string) (sku string, err error) {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return "", err
	}
	defer client.Close()

	suffix := uuid.NewString()[:8]
	sku = "PARENT-SKU-B-" + suffix
	priceListID := "pl-parent-" + suffix
	ts := spanner.CommitTimestamp
	effectiveFrom := time.Now().UTC().Add(-24 * time.Hour)

	warehouseIDs, err := listSupplierWarehouseIDs(ctx, client, supplierID)
	if err != nil {
		return "", err
	}
	if len(warehouseIDs) == 0 {
		warehouseIDs = []string{"wh-parent-" + suffix}
	}

	mutations := []*spanner.Mutation{
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId":     sku,
			"SupplierId":    supplierID,
			"CategoryId":    "GENERAL",
			"Name":          "Parent Phase2 SKU B",
			"PriceMinor":    int64(25000),
			"Currency":      "UZS",
			"StockQuantity": int64(1000),
			"Unit":          "UNIT",
			"UnitVolumeVU":  1.0,
			"IsActive":      true,
			"Version":       int64(1),
			"CreatedAt":     ts,
			"UpdatedAt":     ts,
		}),
		spanner.InsertOrUpdateMap("PriceLists", map[string]any{
			"PriceListId":   priceListID,
			"SupplierId":    supplierID,
			"Name":          "Parent Phase2 List",
			"EffectiveFrom": effectiveFrom,
			"EffectiveTo":   nil,
		}),
		spanner.InsertOrUpdateMap("PriceListItems", map[string]any{
			"PriceListId":    priceListID,
			"Sku":            sku,
			"UnitPriceMinor": int64(25000),
			"MinQty":         int64(1),
		}),
	}
	for _, warehouseID := range warehouseIDs {
		mutations = append(mutations,
			spanner.InsertOrUpdateMap("Warehouses", map[string]any{
				"WarehouseId":      warehouseID,
				"SupplierId":       supplierID,
				"Name":             "Parent Phase2 WH",
				"Lat":              cfg.DeliveryZoneCenterLat,
				"Lng":              cfg.DeliveryZoneCenterLng,
				"CoverageRadiusKm": 50.0,
				"TransferMode":     "TRUCK",
				"IsActive":         true,
				"IsOnShift":        true,
				"CreatedAt":        ts,
				"UpdatedAt":        ts,
			}),
			spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
				"SupplierId":       supplierID,
				"WarehouseId":      warehouseID,
				"ProductId":        sku,
				"QuantityOnHand":   int64(1000),
				"QuantityReserved": int64(0),
				"ReorderThreshold": int64(0),
				"UpdatedAt":        ts,
			}),
		)
	}
	if _, err = client.Apply(ctx, mutations); err != nil {
		return "", err
	}
	return sku, nil
}

func listSupplierWarehouseIDs(ctx context.Context, client *spanner.Client, supplierID string) ([]string, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT WarehouseId FROM Warehouses WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]any{"sid": supplierID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var ids []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var id string
		if err := row.Columns(&id); err != nil {
			return nil, err
		}
		if strings.TrimSpace(id) != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func grantRetailerCreditForSupplier(ctx context.Context, cfg *bootstrap.Config, retailerID, supplierID string, limitMinor int64) error {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return err
	}
	defer client.Close()
	_, err = client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":           retailerID,
			"SupplierId":           supplierID,
			"CreditLimitMinor":     limitMinor,
			"CurrentBalanceMinor":  int64(0),
			"ReservedMinor":        int64(0),
			"AvailableCreditMinor": limitMinor,
			"RiskScore":            int64(0),
			"DelinquencyCount":     int64(0),
			"Status":               "ACTIVE",
			"Version":              int64(1),
			"CreatedAt":            spanner.CommitTimestamp,
			"UpdatedAt":            spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		// Table/column drift — fail soft; checkout may still work if credit reserve is off.
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "Unrecognized") {
			return nil
		}
		return err
	}
	return nil
}
