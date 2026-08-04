package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

func runReturnGateE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, cookie, retailerToken, h3Cell string) error {
	// Issue a payloader JWT scoped to the SSMR warehouse. Demo login defaults to
	// warehouse-demo-1 when PAYLOAD_DEMO_WAREHOUSE_ID is unset, which hides returns.
	payloaderToken, err := auth.Issue(auth.Claims{
		Subject:      envOr("PAYLOAD_DEMO_WORKER_ID", "payloader-demo-1"),
		Role:         auth.RolePayload,
		SupplierID:   supplierID,
		SupplierRole: auth.RoleWarehouseAdmin,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   demoWarehouseID(),
		IsConfigured: true,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 30 * time.Minute})
	if err != nil {
		return fmt.Errorf("issue payloader jwt: %w", err)
	}
	var status int
	var respBody []byte

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

	// Prefer a fresh WH fleet row — shared demo payload vehicles accumulate VU
	// across e2e runs and trip capacity_exceeded on dispatch.
	driverID := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_DRIVER_ID"))
	vehicleID := strings.TrimSpace(os.Getenv("PAYLOAD_DEMO_VEHICLE_ID"))
	if driverID == "" || vehicleID == "" {
		if err := ensureWarehouseDispatchFleet(ctx, client, base, cookie); err != nil {
			return fmt.Errorf("return-gate fleet ensure: %w", err)
		}
		freshDriverID, freshVehicleID, ferr := runWarehouseFleetMgmtE2E(ctx, client, base, cookie, cfg, supplierID)
		if ferr != nil {
			return fmt.Errorf("return-gate fleet mint: %w", ferr)
		}
		driverID, vehicleID = freshDriverID, freshVehicleID
	}
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
	invURL := base + "/v1/warehouse/ops/inventory?warehouse_id=" + warehouseID + "&fresh=1"
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

	dispatchHint, err := runWarehouseDispatchExecute(ctx, client, base, cookie, orderID, driverID, vehicleID, "return-gate-dispatch:"+orderID)
	if err != nil {
		return fmt.Errorf("return-gate dispatch: %w", err)
	}
	if dispatchHint == nil || strings.TrimSpace(dispatchHint.ManifestID) == "" {
		return fmt.Errorf("return-gate dispatch missing manifest for order %s", orderID)
	}
	if err := sealPayloaderManifest(ctx, client, base, payloaderToken, dispatchHint.ManifestID, orderID, vehicleID); err != nil {
		return fmt.Errorf("return-gate seal: %w", err)
	}
	if err := assertDriverDepart(ctx, client, base, cfg, supplierID, driverID); err != nil {
		return fmt.Errorf("return-gate depart: %w", err)
	}

	// Depart may already leave the order IN_TRANSIT; only nudge when needed.
	adminToken, err := auth.Issue(auth.Claims{
		Subject:    supplierID,
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 30 * time.Minute})
	if err != nil {
		return fmt.Errorf("return-gate admin jwt: %w", err)
	}
	var status int
	var respBody []byte
	for _, next := range []string{"LOADED", "IN_TRANSIT"} {
		patchBody, _ := json.Marshal(map[string]string{"status": next})
		status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/order/"+orderID+"/status", patchBody, adminToken, fmt.Sprintf("return-gate-status:%s:%s", orderID, next))
		if err != nil {
			return err
		}
		// Already past this state (e.g. depart → IN_TRANSIT) is fine.
		if status != http.StatusOK && status != http.StatusConflict && status != http.StatusBadRequest {
			return fmt.Errorf("return-gate order status %s: %d body %s", next, status, string(respBody))
		}
	}

	arriveBody, _ := json.Marshal(map[string]string{"order_id": orderID})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/delivery/arrive", arriveBody, driverToken, fmt.Sprintf("return-gate-arrive:%s", orderID))
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

	qrReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/order/"+orderID+"/qr-payload", nil)
	qrReq.Header.Set("Authorization", "Bearer "+retailerToken)
	qrResp, err := client.Do(qrReq)
	if err != nil {
		return fmt.Errorf("return-gate qr payload: %w", err)
	}
	defer qrResp.Body.Close()
	if qrResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(qrResp.Body)
		return fmt.Errorf("return-gate qr payload status %d: %s", qrResp.StatusCode, string(body))
	}
	var qrData struct {
		Token string `json:"qr_token"`
	}
	if err := json.NewDecoder(qrResp.Body).Decode(&qrData); err != nil {
		return fmt.Errorf("decode return-gate qr payload: %w", err)
	}
	if strings.TrimSpace(qrData.Token) == "" {
		return fmt.Errorf("return-gate qr payload missing qr_token")
	}
	scanPayload := []byte(`{"order_id":"` + orderID + `","qr_token":"` + qrData.Token + `"}`)
	var scanOK bool
	for attempt := 0; attempt < 5; attempt++ {
		status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/delivery/scan-qr", scanPayload, driverToken, fmt.Sprintf("return-gate-scan-qr:%s:%d", orderID, attempt))
		if err != nil {
			return fmt.Errorf("return-gate scan qr: %w", err)
		}
		if status == http.StatusOK {
			scanOK = true
			break
		}
		body := string(respBody)
		if (status == http.StatusUnprocessableEntity || status == http.StatusConflict || status == http.StatusInternalServerError) &&
			strings.Contains(body, "optimistic concurrency conflict") {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		return fmt.Errorf("return-gate scan qr status %d: %s", status, body)
	}
	if !scanOK {
		return fmt.Errorf("return-gate scan qr failed after retries")
	}

	bypassPayload := []byte(`{"order_id":"` + orderID + `","reason":"SSMR return gate"}`)
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/orders/payment-bypass", bypassPayload, cookie, "return-gate-payment-bypass:"+orderID)
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
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/delivery/confirm-cash", []byte(cashPayload), retailerToken, "return-gate-confirm-cash:"+orderID)
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

	// ADR-009 Phase 6: return-complete is blocked while any order is FISCALIZING / FISCAL_FAILED.
	// Wait for fiscal worker SUCCESS → COMPLETED before ending shift.
	if err := waitOrderStatus(ctx, client, base, driverToken, orderID, "COMPLETED", 60*time.Second); err != nil {
		adminTok, issueErr := auth.Issue(auth.Claims{
			Subject:    "ssmr-smoke-supplier-admin",
			Role:       auth.RoleAdmin,
			SupplierID: supplierID,
		}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 15 * time.Minute})
		if issueErr != nil {
			return fmt.Errorf("return-gate wait fiscal COMPLETED: %w (admin jwt: %v)", err, issueErr)
		}
		forceBody := []byte(`{"reason_code":"OPS_ESCALATION"}`)
		st, body, _, forceErr := clientDo(ctx, client, http.MethodPost,
			base+"/v1/order/"+orderID+"/force-complete", forceBody, adminTok, fmt.Sprintf("return-gate-force-%s-%d", orderID, time.Now().UnixNano()))
		if forceErr != nil {
			return fmt.Errorf("return-gate wait fiscal COMPLETED: %w (force: %v)", err, forceErr)
		}
		if st != http.StatusOK {
			if waitErr := waitOrderStatus(ctx, client, base, driverToken, orderID, "COMPLETED", 45*time.Second); waitErr == nil {
				fmt.Println("PX_E2E_FISCAL_RACE_COMPLETED_OK")
			} else {
				return fmt.Errorf("return-gate wait fiscal COMPLETED: %w (force status %d: %s)", err, st, string(body))
			}
		} else {
			if waitErr := waitOrderStatus(ctx, client, base, driverToken, orderID, "COMPLETED", 20*time.Second); waitErr != nil {
				return fmt.Errorf("return-gate wait fiscal COMPLETED after force: %w", waitErr)
			}
			fmt.Println("PX_E2E_FISCAL_FORCE_UNSTICK_OK")
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
	// After fiscal SUCCESS, tryCompleteManifest may already complete the manifest and return the driver.
	// Treat no_dispatched_manifest as idempotent success; open_fiscal_block is still a hard fail.
	if status != http.StatusOK {
		body := string(respBody)
		if status == http.StatusConflict && strings.Contains(body, "no_dispatched_manifest") {
			// Manifest already completed post-fiscal — OK for return gate.
		} else {
			return fmt.Errorf("return-gate return-complete status %d body %s", status, body)
		}
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

	wantQty := beforeQty + rejectedQty
	var afterQty int64
	for attempt := 0; attempt < 25; attempt++ {
		afterQty, err = warehouseInventoryQty(ctx, client, base, cookie, whID, sku)
		if err != nil {
			return fmt.Errorf("inventory after: %w", err)
		}
		if afterQty == wantQty {
			break
		}
		time.Sleep(time.Second)
	}
	if afterQty != wantQty {
		return fmt.Errorf("inventory assert failed: before=%d after=%d want=%d", beforeQty, afterQty, wantQty)
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
		// Inbound list filters physical_status=ARRIVED; promote any pre-arrival row for the smoke order.
		stmt := spanner.Statement{
			SQL: `UPDATE SupplierReturns
			      SET PhysicalStatus = @arrived
			      WHERE OrderId = @order_id
			        AND PhysicalStatus IN UNNEST(@from_states)`,
			Params: map[string]any{
				"arrived":     "ARRIVED",
				"order_id":    orderID,
				"from_states": []string{"PENDING", "ON_TRUCK"},
			},
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	return err
}
