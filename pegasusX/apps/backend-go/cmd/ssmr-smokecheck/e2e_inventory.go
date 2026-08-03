package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

func runInventoryReleaseBypassCancelE2E(
	ctx context.Context,
	client *http.Client,
	base string,
	cfg *bootstrap.Config,
	supplierID, cookie, retailerToken, h3Cell string,
) error {
	const orderQty int64 = 1
	sku := envOr("SSMR_SMOKE_SKU", "SSMR-SKU-1")
	whID := demoWarehouseID()

	// Supplier vet REJECT should release reservations created at order create.
	baseline, err := supplierInventoryReserved(ctx, cfg, supplierID, whID, sku)
	if err != nil {
		return fmt.Errorf("vet reject baseline reserved: %w", err)
	}
	vetOrderID, err := createOrderWithQuantity(ctx, client, base, retailerToken, cfg, h3Cell, orderQty)
	if err != nil {
		return fmt.Errorf("vet reject order create: %w", err)
	}
	reservedAfterCreate, err := supplierInventoryReserved(ctx, cfg, supplierID, whID, sku)
	if err != nil {
		return fmt.Errorf("vet reject reserved after create: %w", err)
	}
	if reservedAfterCreate < baseline+orderQty {
		return fmt.Errorf("vet reject reservation missing: baseline=%d after_create=%d want>=%d", baseline, reservedAfterCreate, baseline+orderQty)
	}
	if err := setOrderConfirmationStatus(ctx, cfg, vetOrderID, "PENDING"); err != nil {
		return fmt.Errorf("vet reject await vet: %w", err)
	}
	vetBody, _ := json.Marshal(map[string]string{
		"order_id": vetOrderID,
		"decision": "REJECTED",
		"note":     "SSMR vet reject inventory release",
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/orders/vet", vetBody, cookie, "ssmr-vet-reject-inventory:"+vetOrderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("supplier vet reject status %d body %s", status, string(respBody))
	}
	reservedAfterVet, err := supplierInventoryReserved(ctx, cfg, supplierID, whID, sku)
	if err != nil {
		return fmt.Errorf("vet reject reserved after reject: %w", err)
	}
	if reservedAfterVet != baseline {
		return fmt.Errorf("vet reject inventory release: baseline=%d after_reject=%d", baseline, reservedAfterVet)
	}
	fmt.Println("PX_E2E_INVENTORY_RELEASE_VET_REJECT_OK")

	// Warehouse reject should release reservations on a live pending order.
	baseline, err = supplierInventoryReserved(ctx, cfg, supplierID, whID, sku)
	if err != nil {
		return fmt.Errorf("warehouse reject baseline reserved: %w", err)
	}
	whRejectOrderID, err := createOrderWithQuantity(ctx, client, base, retailerToken, cfg, h3Cell, orderQty)
	if err != nil {
		return fmt.Errorf("warehouse reject order create: %w", err)
	}
	reservedAfterCreate, err = supplierInventoryReserved(ctx, cfg, supplierID, whID, sku)
	if err != nil {
		return fmt.Errorf("warehouse reject reserved after create: %w", err)
	}
	if reservedAfterCreate < baseline+orderQty {
		return fmt.Errorf("warehouse reject reservation missing: baseline=%d after_create=%d want>=%d", baseline, reservedAfterCreate, baseline+orderQty)
	}
	rejectBody, _ := json.Marshal(map[string]string{"reason": "SSMR_WAREHOUSE_REJECT_INVENTORY"})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/warehouse/ops/orders/"+whRejectOrderID+"/reject", rejectBody, cookie, "ssmr-wh-reject-inventory:"+whRejectOrderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse reject status %d body %s", status, string(respBody))
	}
	reservedAfterReject, err := supplierInventoryReserved(ctx, cfg, supplierID, whID, sku)
	if err != nil {
		return fmt.Errorf("warehouse reject reserved after reject: %w", err)
	}
	if reservedAfterReject != baseline {
		return fmt.Errorf("warehouse reject inventory release: baseline=%d after_reject=%d", baseline, reservedAfterReject)
	}
	fmt.Println("PX_E2E_INVENTORY_RELEASE_WAREHOUSE_REJECT_OK")
	return nil
}

func supplierInventoryReserved(ctx context.Context, cfg *bootstrap.Config, supplierID, warehouseID, productID string) (int64, error) {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	row, err := client.Single().ReadRow(ctx, "SupplierInventoryV2",
		spanner.Key{supplierID, warehouseID, productID},
		[]string{"QuantityReserved"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return 0, nil
		}
		return 0, err
	}
	var reserved int64
	if err := row.Columns(&reserved); err != nil {
		return 0, err
	}
	return reserved, nil
}

func setSupplierInventoryLevels(ctx context.Context, cfg *bootstrap.Config, supplierID, warehouseID, productID string, qoh, qr int64) error {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.Apply(ctx, []*spanner.Mutation{
		spanner.UpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      warehouseID,
			"ProductId":        productID,
			"QuantityOnHand":   qoh,
			"QuantityReserved": qr,
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
	})
	return err
}

func runConcurrentStockRejectE2E(
	ctx context.Context,
	client *http.Client,
	base string,
	cfg *bootstrap.Config,
	supplierID, retailerToken, cookie, h3Cell string,
) error {
	const (
		stockQty int64 = 80
		orderQty int64 = 50
	)
	whID := demoWarehouseID()
	sku := envOr("SSMR_SMOKE_SKU", "SSMR-SKU-1")

	settingsURL := base + "/v1/warehouse/ops/settings?warehouse_id=" + whID
	patchBody, _ := json.Marshal(map[string]string{"default_out_of_stock_policy": "REJECT"})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPatch, settingsURL, patchBody, cookie, "ssmr-concurrent-stock-reject-policy")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("concurrent stock reject policy patch status %d body %s", status, string(respBody))
	}

	baselineReserved, err := supplierInventoryReserved(ctx, cfg, supplierID, whID, sku)
	if err != nil {
		return fmt.Errorf("concurrent stock reject baseline reserved: %w", err)
	}
	if err := setSupplierInventoryLevels(ctx, cfg, supplierID, whID, sku, stockQty, baselineReserved); err != nil {
		return fmt.Errorf("concurrent stock reject seed inventory: %w", err)
	}

	retailer2ID, h3Cell2, err := registerRetailerWithPhone(ctx, client, base, cfg, "+998901000199")
	if err != nil {
		return fmt.Errorf("concurrent stock reject register retailer2: %w", err)
	}
	if err := grantRetailerCredit(ctx, client, base, cookie, retailer2ID, 500_000_000); err != nil {
		return fmt.Errorf("concurrent stock reject credit grant retailer2: %w", err)
	}
	retailer2Token, err := auth.Issue(auth.Claims{
		Subject:    retailer2ID,
		Role:       auth.RoleRetailer,
		SupplierID: supplierID,
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("concurrent stock reject issue retailer2 jwt: %w", err)
	}

	type createOutcome struct {
		status int
		body   []byte
		err    error
	}
	results := make(chan createOutcome, 2)
	launchCreate := func(token, idemKey, cell string) {
		body, _ := json.Marshal(map[string]any{
			"line_items": []map[string]any{
				{"sku": sku, "quantity": orderQty, "unit_price_minor": 50000},
			},
			"h3_cell": cell,
			"lat":     cfg.DeliveryZoneCenterLat,
			"lng":     cfg.DeliveryZoneCenterLng,
		})
		go func() {
			status, respBody, _, err := clientPost(ctx, client, base+"/v1/order/create", body, token, idemKey)
			results <- createOutcome{status: status, body: respBody, err: err}
		}()
	}
	launchCreate(retailerToken, "ssmr-concurrent-stock-a", h3Cell)
	launchCreate(retailer2Token, "ssmr-concurrent-stock-b", h3Cell2)

	var created, rejected int
	for i := 0; i < 2; i++ {
		out := <-results
		if out.err != nil {
			return out.err
		}
		switch out.status {
		case http.StatusCreated:
			created++
		case http.StatusUnprocessableEntity:
			if !strings.Contains(string(out.body), "inventory_exhausted") {
				return fmt.Errorf("expected inventory_exhausted rejection, got %s", string(out.body))
			}
			rejected++
		default:
			return fmt.Errorf("unexpected concurrent create status %d body %s", out.status, string(out.body))
		}
	}
	if created != 1 || rejected != 1 {
		return fmt.Errorf("concurrent stock reject want 1 create + 1 reject, got created=%d rejected=%d", created, rejected)
	}

	reservedAfter, err := supplierInventoryReserved(ctx, cfg, supplierID, whID, sku)
	if err != nil {
		return fmt.Errorf("concurrent stock reject reserved after: %w", err)
	}
	if reservedAfter != baselineReserved+orderQty {
		return fmt.Errorf("concurrent stock reject reserved=%d want=%d", reservedAfter, baselineReserved+orderQty)
	}

	fmt.Println("PX_E2E_CONCURRENT_STOCK_REJECT_OK")
	return nil
}

func setOrderConfirmationStatus(ctx context.Context, cfg *bootstrap.Config, orderID, confirmationStatus string) error {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":            orderID,
				"ConfirmationStatus": confirmationStatus,
				"UpdatedAt":          spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

// ensureOrderDispatchable forces Status=PENDING so warehouse dispatch preview/execute
// includes the order in undispatched_orders (long e2e runs can leave earlier creates out of pool).
func ensureOrderDispatchable(ctx context.Context, cfg *bootstrap.Config, orderID string) error {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return err
	}
	defer client.Close()
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":   orderID,
				"Status":    "PENDING",
				"UpdatedAt": spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

func runOrderAcceptanceClosedE2E(
	ctx context.Context,
	client *http.Client,
	base, cookie, retailerToken string,
	cfg *bootstrap.Config,
	h3Cell string,
) error {
	whID := demoWarehouseID()
	settingsURL := base + "/v1/warehouse/ops/settings?warehouse_id=" + whID
	weekdayKeys := []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	otherDay := weekdayKeys[(int(time.Now().UTC().Weekday())+3)%7]
	closedSchedule, _ := json.Marshal(map[string]any{
		"operating_schedule": map[string]any{
			"enforce_order_acceptance": true,
			"is_24h":                   false,
			"timezone":                 "UTC",
			"schedules": map[string]any{
				otherDay: map[string]string{"open": "09:00", "close": "17:00"},
			},
		},
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPatch, settingsURL, closedSchedule, cookie, "ssmr-order-acceptance-closed")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("order acceptance closed patch status %d body %s", status, string(respBody))
	}

	sku := envOr("SSMR_SMOKE_SKU", "SSMR-SKU-1")
	createBody, _ := json.Marshal(map[string]any{
		"line_items": []map[string]any{
			{"sku": sku, "quantity": 1, "unit_price_minor": 50000},
		},
		"h3_cell": h3Cell,
		"lat":     cfg.DeliveryZoneCenterLat,
		"lng":     cfg.DeliveryZoneCenterLng,
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/order/create", createBody, retailerToken, "ssmr-order-acceptance-closed-create")
	if err != nil {
		return err
	}
	if status != http.StatusUnprocessableEntity {
		return fmt.Errorf("order acceptance closed want 422 got %d body %s", status, string(respBody))
	}
	if !strings.Contains(string(respBody), "order_acceptance_closed") {
		return fmt.Errorf("order acceptance closed missing code: %s", string(respBody))
	}

	// Restore open schedule for downstream tests.
	openSchedule, _ := json.Marshal(map[string]any{"operating_schedule": map[string]any{"is_24h": true, "enforce_order_acceptance": false}})
	_, _, _, _ = clientDo(ctx, client, http.MethodPatch, settingsURL, openSchedule, cookie, "ssmr-order-acceptance-restore")

	fmt.Println("PX_E2E_ORDER_ACCEPTANCE_CLOSED_OK")
	return nil
}
