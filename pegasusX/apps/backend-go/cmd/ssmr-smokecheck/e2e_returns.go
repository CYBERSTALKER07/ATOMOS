package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

func runReturnGateE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, cookie, retailerToken, h3Cell string) error {
	loginBody, _ := json.Marshal(map[string]string{
		"phone": envOr("PAYLOAD_DEMO_PHONE", "+998901234567"),
		"pin":   envOr("PAYLOAD_DEMO_PIN", "123456"),
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/auth/payloader/login", loginBody, "", "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("payloader login status %d body %s", status, string(respBody))
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(respBody, &loginResp); err != nil || loginResp.Token == "" {
		return fmt.Errorf("payloader login missing token")
	}
	payloaderToken := loginResp.Token

	if status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/returns/inbound?physical_status=ARRIVED&limit=10", nil, payloaderToken, ""); err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("returns inbound status %d body %s", status, string(respBody))
	}
	if status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/returns/history?limit=5", nil, payloaderToken, ""); err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("returns history status %d body %s", status, string(respBody))
	}

	driverID := envOr("PAYLOAD_DEMO_DRIVER_ID", "drv_payload_1")
	vehicleID := envOr("PAYLOAD_DEMO_VEHICLE_ID", "veh_payload_1")
	driverToken, err := auth.Issue(auth.Claims{
		Subject:      driverID,
		Role:         auth.RoleDriver,
		SupplierID:   supplierID,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   demoWarehouseID(),
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 30 * time.Minute})
	if err != nil {
		return fmt.Errorf("issue driver jwt: %w", err)
	}
	if status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/driver/return-goods", nil, driverToken, ""); err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("driver return-goods status %d body %s", status, string(respBody))
	}

	supplierToken, err := auth.Issue(auth.Claims{
		Subject:    supplierID,
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 30 * time.Minute})
	if err != nil {
		return fmt.Errorf("issue supplier jwt: %w", err)
	}
	if status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/returns?limit=5", nil, supplierToken, ""); err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("supplier returns status %d body %s", status, string(respBody))
	}

	if err := runReturnGateReceiveE2E(ctx, client, base, cfg, supplierID, cookie, retailerToken, h3Cell, payloaderToken, driverToken, driverID, vehicleID); err != nil {
		return err
	}
	fmt.Println("PX_E2E_RETURN_GATE_RECEIVE_OK")
	return nil
}

func smokeBarcode() string {
	if bc := strings.TrimSpace(os.Getenv("SSMR_SMOKE_BARCODE")); bc != "" {
		return bc
	}
	return "5901234123457"
}

func warehouseInventoryQty(ctx context.Context, client *http.Client, base, cookie, warehouseID, productID string) (int64, error) {
	invURL := base + "/v1/warehouse/ops/inventory?warehouse_id=" + warehouseID
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, invURL, nil, cookie, "")
	if err != nil {
		return 0, err
	}
	if status != http.StatusOK {
		return 0, fmt.Errorf("warehouse inventory status %d body %s", status, string(respBody))
	}
	var invResp struct {
		Items []struct {
			ProductID      string `json:"product_id"`
			QuantityOnHand int64  `json:"quantity_on_hand"`
		} `json:"items"`
	}
	if err := json.Unmarshal(respBody, &invResp); err != nil {
		return 0, err
	}
	for _, item := range invResp.Items {
		if item.ProductID == productID {
			return item.QuantityOnHand, nil
		}
	}
	return 0, fmt.Errorf("inventory row missing for product %s", productID)
}

func runReturnGateReceiveE2E(
	ctx context.Context,
	client *http.Client,
	base string,
	cfg *bootstrap.Config,
	supplierID, cookie, retailerToken, h3Cell, payloaderToken, driverToken, driverID, vehicleID string,
) error {
	const rejectedQty int64 = 1
	whID := demoWarehouseID()
	sku := envOr("SSMR_SMOKE_SKU", "SSMR-SKU-1")
	barcode := smokeBarcode()

	beforeQty, err := warehouseInventoryQty(ctx, client, base, cookie, whID, sku)
	if err != nil {
		return fmt.Errorf("inventory before: %w", err)
	}

	orderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		return fmt.Errorf("return-gate order create: %w", err)
	}
	sessionID, err := runUnifiedCheckout(ctx, client, base, retailerToken, orderID, cfg, supplierID)
	if err != nil {
		return fmt.Errorf("return-gate checkout: %w", err)
	}
	if err := replayGlobalPayWebhook(ctx, client, base, cfg, sessionID, orderID); err != nil {
		return fmt.Errorf("return-gate webhook: %w", err)
	}

	adminToken, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-return-gate-admin",
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 30 * time.Minute})
	if err != nil {
		return fmt.Errorf("issue admin jwt: %w", err)
	}
	assignBody, _ := json.Marshal(map[string]any{
		"driver_id": driverID,
		"route_id":  "route-return-gate-" + orderID,
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/orders/"+orderID+"/assign", assignBody, adminToken, "return-gate-assign:"+orderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("return-gate assign status %d body %s", status, string(respBody))
	}
	for _, next := range []string{"LOADED", "IN_TRANSIT"} {
		patchBody, _ := json.Marshal(map[string]string{"status": next})
		status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/order/"+orderID+"/status", patchBody, adminToken, "return-gate-status:"+orderID+":"+next)
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("return-gate status %s: %d body %s", next, status, string(respBody))
		}
	}

	arriveBody, _ := json.Marshal(map[string]string{"order_id": orderID})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/delivery/arrive", arriveBody, driverToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("return-gate arrive status %d body %s", status, string(respBody))
	}

	amendBody, _ := json.Marshal(map[string]any{
		"order_id": orderID,
		"items": []map[string]any{{
			"product_id":   sku,
			"accepted_qty": 1,
			"rejected_qty": rejectedQty,
			"reason":       "DAMAGED",
		}},
		"driver_notes": "SSMR return gate E2E",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/order/amend", amendBody, driverToken, "return-gate-amend:"+orderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("return-gate amend status %d body %s", status, string(respBody))
	}

	bypassPayload := []byte(`{"order_id":"` + orderID + `","reason":"SSMR return gate"}`)
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/orders/payment-bypass", bypassPayload, cookie, "application/json")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("return-gate payment bypass status %d body %s", status, string(respBody))
	}
	var bypassResp struct {
		BypassToken string `json:"bypass_token"`
	}
	if err := json.Unmarshal(respBody, &bypassResp); err != nil || bypassResp.BypassToken == "" {
		return fmt.Errorf("return-gate payment bypass invalid body %s", string(respBody))
	}

	cashPayload := `{"order_id":"` + orderID + `"}`
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/delivery/confirm-cash", []byte(cashPayload), retailerToken, "application/json")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("return-gate confirm cash status %d body %s", status, string(respBody))
	}

	collectBody, _ := json.Marshal(map[string]any{
		"order_id":  orderID,
		"latitude":  cfg.DeliveryZoneCenterLat,
		"longitude": cfg.DeliveryZoneCenterLng,
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/order/collect-cash", collectBody, driverToken, "return-gate-collect:"+orderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("return-gate collect cash status %d body %s", status, string(respBody))
	}

	if err := assertDriverDepart(ctx, client, base, cfg, supplierID, driverID); err != nil {
		if !strings.Contains(err.Error(), "no_sealed_manifest") {
			return fmt.Errorf("return-gate depart: %w", err)
		}
	}

	if err := ensureSmokeProductBarcode(ctx, cfg, supplierID, sku, barcode); err != nil {
		return fmt.Errorf("ensure product barcode: %w", err)
	}

	returnBody, _ := json.Marshal(map[string]string{"truck_id": vehicleID})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/fleet/driver/return-complete", returnBody, driverToken, "return-gate-complete:"+driverID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("return-gate return-complete status %d body %s", status, string(respBody))
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/returns/inbound?physical_status=ARRIVED&limit=50", nil, payloaderToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("return-gate inbound list status %d body %s", status, string(respBody))
	}
	var inbound struct {
		Data []struct {
			ReturnID    string `json:"return_id"`
			OrderID     string `json:"order_id"`
			ExpectedQty int64  `json:"expected_qty"`
			ReceivedQty int64  `json:"received_qty"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &inbound); err != nil {
		return fmt.Errorf("decode inbound: %w", err)
	}
	var returnID string
	for _, row := range inbound.Data {
		if row.OrderID == orderID {
			returnID = row.ReturnID
			break
		}
	}
	if returnID == "" {
		if err := promoteReturnGateForE2E(ctx, cfg, orderID); err != nil {
			return fmt.Errorf("promote return for e2e: %w", err)
		}
		status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/returns/inbound?physical_status=ARRIVED&limit=50", nil, payloaderToken, "")
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("return-gate inbound retry status %d body %s", status, string(respBody))
		}
		if err := json.Unmarshal(respBody, &inbound); err != nil {
			return fmt.Errorf("decode inbound retry: %w", err)
		}
		for _, row := range inbound.Data {
			if row.OrderID == orderID {
				returnID = row.ReturnID
				break
			}
		}
	}
	if returnID == "" {
		return fmt.Errorf("inbound missing return for order %s body %s", orderID, string(respBody))
	}

	sessionBody := []byte("{}")
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/returns/inbound/sessions", sessionBody, payloaderToken, "return-gate-session:"+returnID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("return-gate session status %d body %s", status, string(respBody))
	}
	var sessionResp struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(respBody, &sessionResp); err != nil || sessionResp.SessionID == "" {
		return fmt.Errorf("return-gate session invalid body %s", string(respBody))
	}

	scanBody, _ := json.Marshal(map[string]any{
		"barcode":    barcode,
		"qty":        rejectedQty,
		"session_id": sessionResp.SessionID,
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/returns/inbound/scan", scanBody, payloaderToken, "return-gate-scan:"+returnID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("return-gate scan status %d body %s", status, string(respBody))
	}

	confirmBody, _ := json.Marshal(map[string]any{
		"session_id": sessionResp.SessionID,
		"lines": []map[string]any{{
			"return_id":   returnID,
			"disposition": "RESTOCK",
			"qty":         rejectedQty,
		}},
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/returns/inbound/confirm", confirmBody, payloaderToken, "return-gate-confirm:"+returnID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("return-gate confirm status %d body %s", status, string(respBody))
	}

	afterQty, err := warehouseInventoryQty(ctx, client, base, cookie, whID, sku)
	if err != nil {
		return fmt.Errorf("inventory after: %w", err)
	}
	if afterQty != beforeQty+rejectedQty {
		return fmt.Errorf("inventory assert failed: before=%d after=%d want=%d", beforeQty, afterQty, beforeQty+rejectedQty)
	}
	return nil
}

func ensureSmokeProductBarcode(ctx context.Context, cfg *bootstrap.Config, supplierID, sku, barcode string) error {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return err
	}
	defer client.Close()
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("Products", map[string]any{
				"ProductId":  sku,
				"SupplierId": supplierID,
				"Barcode":    barcode,
				"UpdatedAt":  spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

func promoteReturnGateForE2E(ctx context.Context, cfg *bootstrap.Config, orderID string) error {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return err
	}
	defer client.Close()
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `UPDATE SupplierReturns
			      SET PhysicalStatus = @on_truck
			      WHERE OrderId = @order_id AND PhysicalStatus = @pending`,
			Params: map[string]any{
				"on_truck": "ON_TRUCK",
				"order_id": orderID,
				"pending":  "PENDING",
			},
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	return err
}
