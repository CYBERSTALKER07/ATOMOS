package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

func runManualPreorderE2E(ctx context.Context, client *http.Client, base, retailerToken, cookie string, cfg *bootstrap.Config, h3Cell string) error {
	delivery := time.Now().UTC().AddDate(0, 0, 10)
	createBody, _ := json.Marshal(map[string]any{
		"line_items": []map[string]any{
			{"sku": "SSMR-SKU-1", "quantity": 2, "unit_price_minor": 50000},
		},
		"h3_cell":                 h3Cell,
		"lat":                     cfg.DeliveryZoneCenterLat,
		"lng":                     cfg.DeliveryZoneCenterLng,
		"delivery_mode":           "SCHEDULED",
		"requested_delivery_date": delivery.Format(time.RFC3339),
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/order/create", createBody, retailerToken, "ssmr-manual-preorder-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("manual preorder create status %d body %s", status, string(respBody))
	}
	var created struct {
		OrderID string `json:"order_id"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return err
	}
	if created.OrderID == "" {
		return fmt.Errorf("manual preorder missing order_id")
	}
	confirmBody, _ := json.Marshal(map[string]string{"order_id": created.OrderID})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/orders/confirm-preorder", confirmBody, retailerToken, "ssmr-manual-preorder-confirm-"+created.OrderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("confirm preorder status %d body %s", status, string(respBody))
	}
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/warehouse/ops/preorders", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK || !strings.Contains(string(respBody), created.OrderID) {
		return fmt.Errorf("warehouse preorders status %d body %s", status, string(respBody))
	}
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/warehouse/ops/stock-commitments", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK || !strings.Contains(string(respBody), "SSMR-SKU-1") {
		return fmt.Errorf("stock commitments status %d body %s", status, string(respBody))
	}
	rejectCreate, _ := json.Marshal(map[string]any{
		"line_items": []map[string]any{
			{"sku": "SSMR-SKU-1", "quantity": 1, "unit_price_minor": 50000},
		},
		"h3_cell":                 h3Cell,
		"lat":                     cfg.DeliveryZoneCenterLat,
		"lng":                     cfg.DeliveryZoneCenterLng,
		"delivery_mode":           "SCHEDULED",
		"requested_delivery_date": delivery.Format(time.RFC3339),
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/order/create", rejectCreate, retailerToken, "ssmr-manual-preorder-reject-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("manual preorder reject create status %d body %s", status, string(respBody))
	}
	var rejectOrder struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(respBody, &rejectOrder); err != nil || rejectOrder.OrderID == "" {
		return fmt.Errorf("manual preorder reject create missing order_id")
	}
	rejectBody, _ := json.Marshal(map[string]string{"reason": "SSMR_WAREHOUSE_REJECT"})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/warehouse/ops/preorders/"+rejectOrder.OrderID+"/reject", rejectBody, cookie, "ssmr-wh-preorder-reject-"+rejectOrder.OrderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse preorder reject status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_MANUAL_PREORDER_OK")
	fmt.Println("PX_E2E_WAREHOUSE_PREORDER_REJECT_OK")
	return nil
}

func runDeliveryProposalE2E(ctx context.Context, client *http.Client, base, retailerToken, cookie string, cfg *bootstrap.Config, h3Cell string) error {
	delivery := time.Now().UTC().AddDate(0, 0, 14).Truncate(24 * time.Hour).Add(12 * time.Hour)
	proposed := delivery.AddDate(0, 0, 5)
	createBody, _ := json.Marshal(map[string]any{
		"line_items": []map[string]any{
			{"sku": "SSMR-SKU-1", "quantity": 2, "unit_price_minor": 50000},
		},
		"h3_cell":                 h3Cell,
		"lat":                     cfg.DeliveryZoneCenterLat,
		"lng":                     cfg.DeliveryZoneCenterLng,
		"delivery_mode":           "SCHEDULED",
		"requested_delivery_date": delivery.Format(time.RFC3339),
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/order/create", createBody, retailerToken, "ssmr-delivery-proposal-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("delivery proposal create status %d body %s", status, string(respBody))
	}
	var created struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil || created.OrderID == "" {
		return fmt.Errorf("delivery proposal create missing order_id")
	}
	confirmBody, _ := json.Marshal(map[string]string{"order_id": created.OrderID})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/orders/confirm-preorder", confirmBody, retailerToken, "ssmr-delivery-proposal-confirm-"+created.OrderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("delivery proposal confirm status %d body %s", status, string(respBody))
	}
	proposeBody, _ := json.Marshal(map[string]string{
		"proposed_delivery_date": proposed.Format(time.RFC3339),
		"reason":                 "SSMR_PROPOSE_DATE",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/warehouse/ops/preorders/"+created.OrderID+"/propose-delivery", proposeBody, cookie, "ssmr-delivery-propose-"+created.OrderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse propose delivery status %d body %s", status, string(respBody))
	}
	if !strings.Contains(string(respBody), "PENDING_WAREHOUSE") {
		return fmt.Errorf("propose response missing PENDING_WAREHOUSE: %s", string(respBody))
	}
	acceptBody, _ := json.Marshal(map[string]string{"order_id": created.OrderID})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/orders/accept-delivery-proposal", acceptBody, retailerToken, "ssmr-delivery-accept-"+created.OrderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("accept delivery proposal status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_DELIVERY_PROPOSAL_OK")

	rejectCreate, _ := json.Marshal(map[string]any{
		"line_items": []map[string]any{
			{"sku": "SSMR-SKU-1", "quantity": 1, "unit_price_minor": 50000},
		},
		"h3_cell":                 h3Cell,
		"lat":                     cfg.DeliveryZoneCenterLat,
		"lng":                     cfg.DeliveryZoneCenterLng,
		"delivery_mode":           "SCHEDULED",
		"requested_delivery_date": delivery.Format(time.RFC3339),
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/order/create", rejectCreate, retailerToken, "ssmr-delivery-proposal-reject-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("delivery proposal reject create status %d body %s", status, string(respBody))
	}
	var rejectOrder struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(respBody, &rejectOrder); err != nil || rejectOrder.OrderID == "" {
		return fmt.Errorf("delivery proposal reject create missing order_id")
	}
	confirmBody2, _ := json.Marshal(map[string]string{"order_id": rejectOrder.OrderID})
	status, _, _, err = clientPost(ctx, client, base+"/v1/orders/confirm-preorder", confirmBody2, retailerToken, "ssmr-delivery-proposal-reject-confirm-"+rejectOrder.OrderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("delivery proposal reject confirm status %d", status)
	}
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/warehouse/ops/preorders/"+rejectOrder.OrderID+"/propose-delivery", proposeBody, cookie, "ssmr-delivery-propose-reject-"+rejectOrder.OrderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse propose for reject path status %d body %s", status, string(respBody))
	}
	rejectBody, _ := json.Marshal(map[string]string{"order_id": rejectOrder.OrderID, "reason": "SSMR_REJECT_PROPOSAL"})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/orders/reject-delivery-proposal", rejectBody, retailerToken, "ssmr-delivery-reject-"+rejectOrder.OrderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("reject delivery proposal status %d body %s", status, string(respBody))
	}
	if !strings.Contains(string(respBody), "CANCELLED") {
		return fmt.Errorf("reject proposal response missing CANCELLED: %s", string(respBody))
	}
	fmt.Println("PX_E2E_DELIVERY_PROPOSAL_REJECT_CANCEL_OK")
	return nil
}

func assertRetailerTracking(ctx context.Context, client *http.Client, base, retailerToken, orderID string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/retailer/tracking", nil, retailerToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("tracking status %d body %s", status, string(respBody))
	}
	if !strings.Contains(string(respBody), orderID) {
		return fmt.Errorf("tracking response missing order_id %s", orderID)
	}
	return nil
}

func runDeliveryEdgeCasesE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, retailerToken, supplierID, cookie, h3Cell string) error {
	orderID, err := createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	sessionID, err := runUnifiedCheckout(ctx, client, base, retailerToken, orderID, cfg)
	if err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	if err := replayGlobalPayWebhook(ctx, client, base, cfg, sessionID, orderID); err != nil {
		return fmt.Errorf("webhook: %w", err)
	}

	driverID := envOr("SSMR_SMOKE_DRIVER_ID", "ssmr-driver-1")
	adminToken, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-supplier-admin",
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
	assignBody, _ := json.Marshal(map[string]any{
		"driver_id": driverID,
		"route_id":  "route-ssmr-delivery-edge",
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/orders/"+orderID+"/assign", assignBody, adminToken, "admin-order-assign:"+orderID+":"+driverID)
	if err != nil {
		return fmt.Errorf("assign: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("assign order status %d body %s", status, string(respBody))
	}
	for _, next := range []string{"LOADED", "IN_TRANSIT"} {
		patchBody, _ := json.Marshal(map[string]string{"status": next})
		status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/order/"+orderID+"/status", patchBody, adminToken, "admin-order-status:"+orderID+":"+next)
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("order status %s: %d body %s", next, status, string(respBody))
		}
	}

	driverToken, err := auth.Issue(auth.Claims{
		Subject:      driverID,
		Role:         auth.RoleDriver,
		SupplierID:   supplierID,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   envOr("SSMR_SMOKE_WAREHOUSE_ID", "ssmr-warehouse-1"),
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("issue driver jwt: %w", err)
	}
	arriveBody, _ := json.Marshal(map[string]string{"order_id": orderID})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/delivery/arrive", arriveBody, driverToken, "")
	if err != nil {
		return fmt.Errorf("delivery arrive: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("delivery arrive status %d body %s", status, string(respBody))
	}

	// QR flow: Retailer gets payload
	qrReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/order/"+orderID+"/qr-payload", nil)
	qrReq.Header.Set("Authorization", "Bearer "+retailerToken)
	qrResp, err := client.Do(qrReq)
	if err != nil {
		return fmt.Errorf("qr payload request: %w", err)
	}
	defer qrResp.Body.Close()
	if qrResp.StatusCode != http.StatusOK {
		return fmt.Errorf("qr payload status %d", qrResp.StatusCode)
	}
	var qrData struct {
		Token string `json:"qr_token"`
	}
	if err := json.NewDecoder(qrResp.Body).Decode(&qrData); err != nil {
		return fmt.Errorf("decode qr payload: %w", err)
	}
	if strings.TrimSpace(qrData.Token) == "" {
		return fmt.Errorf("qr payload missing qr_token")
	}

	// QR flow: Driver validates token (no state transition)
	validatePayload := `{"order_id":"` + orderID + `","scanned_token":"` + qrData.Token + `"}`
	validateReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/order/validate-qr", strings.NewReader(validatePayload))
	validateReq.Header.Set("Authorization", "Bearer "+driverToken)
	validateReq.Header.Set("Content-Type", "application/json")
	validateResp, err := client.Do(validateReq)
	if err != nil {
		return fmt.Errorf("validate qr request: %w", err)
	}
	defer validateResp.Body.Close()
	if validateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(validateResp.Body)
		return fmt.Errorf("validate qr status %d: %s", validateResp.StatusCode, string(body))
	}

	// Damaged goods report (must run before scan-qr transitions to AWAITING_PAYMENT)
	dmgPayload := `{"order_id":"` + orderID + `","damaged_items":[{"sku":"SSMR-SKU-1","quantity":1}],"reason":"Broken bottle"}`
	dmgReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/delivery/report-damage", strings.NewReader(dmgPayload))
	dmgReq.Header.Set("Authorization", "Bearer "+driverToken)
	dmgReq.Header.Set("Content-Type", "application/json")
	dmgResp, err := client.Do(dmgReq)
	if err != nil {
		return fmt.Errorf("report damage request: %w", err)
	}
	defer dmgResp.Body.Close()
	if dmgResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(dmgResp.Body)
		return fmt.Errorf("report damage status %d: %s", dmgResp.StatusCode, string(body))
	}

	// QR flow: Driver scans QR
	scanPayload := `{"order_id":"` + orderID + `","qr_token":"` + qrData.Token + `"}`
	scanReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/delivery/scan-qr", strings.NewReader(scanPayload))
	scanReq.Header.Set("Authorization", "Bearer "+driverToken)
	scanReq.Header.Set("Content-Type", "application/json")
	scanResp, err := client.Do(scanReq)
	if err != nil {
		return fmt.Errorf("scan qr request: %w", err)
	}
	defer scanResp.Body.Close()
	if scanResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(scanResp.Body)
		return fmt.Errorf("scan qr status %d: %s", scanResp.StatusCode, string(body))
	}

	// Payment bypass while order is AWAITING_PAYMENT (after scan-qr, before cash selection).
	bypassPayload := []byte(`{"order_id":"` + orderID + `","reason":"SSMR smoke"}`)
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/orders/payment-bypass", bypassPayload, cookie, "application/json")
	if err != nil {
		return fmt.Errorf("payment bypass: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("payment bypass status %d body %s", status, string(respBody))
	}
	var bypassResp struct {
		BypassToken string `json:"bypass_token"`
		OrderID     string `json:"order_id"`
	}
	if err := json.Unmarshal(respBody, &bypassResp); err != nil {
		return fmt.Errorf("decode payment bypass: %w", err)
	}
	if strings.TrimSpace(bypassResp.BypassToken) == "" {
		return fmt.Errorf("payment bypass missing bypass_token body %s", string(respBody))
	}
	if bypassResp.OrderID != orderID {
		return fmt.Errorf("payment bypass order_id mismatch got %s want %s", bypassResp.OrderID, orderID)
	}
	fmt.Println("PX_E2E_SUPPLIER_PAYMENT_BYPASS_OK")

	// Retailer selects cash (driver-only completion path)
	cashPayload := `{"order_id":"` + orderID + `"}`
	cashReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/delivery/confirm-cash", strings.NewReader(cashPayload))
	cashReq.Header.Set("Authorization", "Bearer "+retailerToken)
	cashReq.Header.Set("Content-Type", "application/json")
	cashResp, err := client.Do(cashReq)
	if err != nil {
		return fmt.Errorf("confirm cash request: %w", err)
	}
	defer cashResp.Body.Close()
	if cashResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(cashResp.Body)
		return fmt.Errorf("confirm cash status %d: %s", cashResp.StatusCode, string(body))
	}
	var cashSelect struct {
		State string `json:"state"`
	}
	_ = json.NewDecoder(cashResp.Body).Decode(&cashSelect)
	if cashSelect.State != "" && cashSelect.State != "PENDING_CASH_COLLECTION" {
		return fmt.Errorf("confirm cash expected PENDING_CASH_COLLECTION, got %s", cashSelect.State)
	}

	// Driver collects cash and completes order (coords must match order delivery point)
	collectBody, _ := json.Marshal(map[string]any{
		"order_id":  orderID,
		"latitude":  cfg.DeliveryZoneCenterLat,
		"longitude": cfg.DeliveryZoneCenterLng,
	})
	collectReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/order/collect-cash", bytes.NewReader(collectBody))
	collectReq.Header.Set("Authorization", "Bearer "+driverToken)
	collectReq.Header.Set("Content-Type", "application/json")
	collectReq.Header.Set("Idempotency-Key", "ssmr-collect-cash-"+orderID)
	collectResp, err := client.Do(collectReq)
	if err != nil {
		return fmt.Errorf("collect cash request: %w", err)
	}
	defer collectResp.Body.Close()
	if collectResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(collectResp.Body)
		return fmt.Errorf("collect cash status %d: %s", collectResp.StatusCode, string(body))
	}

	return nil
}

// runPaymentSmokeCheck exercises unified checkout and global-pay webhook replay.
