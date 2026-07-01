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

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// e2eTimeout bounds the full multi-role SSMR smoke path (supplier through driver edges).
func runCatalogCategorySuppliersE2E(ctx context.Context, client *http.Client, base, retailerToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/catalog/categories/ssmr-category/suppliers", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("category suppliers status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var rows []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return fmt.Errorf("decode category suppliers: %w", err)
	}
	return nil
}

func runDeviceTokenE2E(ctx context.Context, client *http.Client, base, actorToken string) error {
	payload := map[string]string{"token": "ssmr-fcm-test-token", "platform": "android"}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/user/device-token", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+actorToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("device token status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func runRetailerReceivingWindowE2E(ctx context.Context, client *http.Client, base, retailerToken string) error {
	updateBody := []byte(`{"receiving_window_open":"10:30","receiving_window_close":"19:45"}`)
	status, body, _, err := clientDo(ctx, client, http.MethodPut, base+"/v1/retailer/profile", updateBody, retailerToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("PUT retailer/profile status %d body %s", status, string(body))
	}

	status, body, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/retailer/profile", nil, retailerToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("GET retailer/profile status %d body %s", status, string(body))
	}
	var profile map[string]any
	if err := json.Unmarshal(body, &profile); err != nil {
		return fmt.Errorf("decode retailer profile: %w", err)
	}
	if profile["receiving_window_open"] != "10:30" {
		return fmt.Errorf("receiving_window_open=%v want 10:30", profile["receiving_window_open"])
	}
	if profile["receiving_window_close"] != "19:45" {
		return fmt.Errorf("receiving_window_close=%v want 19:45", profile["receiving_window_close"])
	}

	fmt.Println("PX_E2E_RETAILER_RECEIVING_WINDOW_OK")
	return nil
}

func runRetailerCatalogProductsE2E(ctx context.Context, client *http.Client, base, retailerToken string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/catalog/products", nil, retailerToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("GET catalog/products status %d body %s", status, string(body))
	}
	var products []map[string]any
	if err := json.Unmarshal(body, &products); err != nil {
		return fmt.Errorf("decode catalog/products: %w", err)
	}
	for _, product := range products {
		if _, ok := product["available_stock"]; ok {
			fmt.Println("PX_E2E_RETAILER_CATALOG_STOCK_OK")
			break
		}
	}
	fmt.Println("PX_E2E_RETAILER_CATALOG_PRODUCTS_OK")
	return nil
}

func runRetailerCancelE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, retailerToken, orderID, retailerID string) error {
	const orderQty int64 = 2
	sku := envOr("SSMR_SMOKE_SKU", "SSMR-SKU-1")
	whID := demoWarehouseID()

	reservedAfterCreate, err := supplierInventoryReserved(ctx, cfg, supplierID, whID, sku)
	if err != nil {
		return fmt.Errorf("reserved after create: %w", err)
	}

	cancelBody, _ := json.Marshal(map[string]string{
		"order_id":    orderID,
		"retailer_id": retailerID,
	})
	status, body, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/order/cancel", cancelBody, retailerToken, "retailer-cancel-smoke:"+orderID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("POST order/cancel status %d body %s", status, string(body))
	}

	reservedAfterCancel, err := supplierInventoryReserved(ctx, cfg, supplierID, whID, sku)
	if err != nil {
		return fmt.Errorf("reserved after cancel: %w", err)
	}
	if reservedAfterCancel+orderQty != reservedAfterCreate {
		return fmt.Errorf("retailer cancel inventory release: reserved_after_create=%d reserved_after_cancel=%d want=%d",
			reservedAfterCreate, reservedAfterCancel, reservedAfterCreate-orderQty)
	}

	fmt.Println("PX_E2E_RETAILER_CANCEL_OK")
	fmt.Println("PX_E2E_INVENTORY_RELEASE_RETAILER_CANCEL_OK")
	return nil
}

func runRetailerCardInitiateE2E(ctx context.Context, client *http.Client, base, retailerToken string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/retailer/card/initiate", []byte("{}"), retailerToken, "retailer-card-initiate-smoke")
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusServiceUnavailable {
		return fmt.Errorf("POST retailer/card/initiate status %d body %s", status, string(body))
	}
	fmt.Println("PX_E2E_RETAILER_CARD_INITIATE_OK")
	return nil
}

func runRetailerClientPolicyE2E(ctx context.Context, client *http.Client, base string) error {
	body, err := clientGet(ctx, client, base+"/v1/platform/client-policy?role=RETAILER&platform=web&version=1.0.0&channel=production")
	if err != nil {
		return err
	}
	var resp struct {
		Role           string `json:"role"`
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode retailer client policy: %w", err)
	}
	if resp.Role != "RETAILER" {
		return fmt.Errorf("retailer client policy role=%q want RETAILER", resp.Role)
	}
	if strings.TrimSpace(resp.MinimumVersion) == "" {
		return fmt.Errorf("retailer client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_RETAILER_CLIENT_POLICY_OK")
	return nil
}

func runRetailerNotificationInboxE2E(ctx context.Context, client *http.Client, base, retailerToken string) error {
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = assertInboxHasRows(ctx, client, base, retailerToken, "retailer")
		if lastErr == nil {
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	if lastErr != nil {
		return fmt.Errorf("retailer inbox not ready after kafka fanout window: %w", lastErr)
	}
	markBody, _ := json.Marshal(map[string]any{"mark_all": true})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/user/notifications/read", markBody, retailerToken, "ssmr-retailer-inbox-read")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("retailer mark notifications read status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_RETAILER_NOTIFICATION_INBOX_OK")
	return nil
}

func runRetailerPricingOverrideE2E(
	ctx context.Context,
	client *http.Client,
	base, cookie, supplierID, retailerID, retailerToken string,
) error {
	const overridePrice = int64(42000)
	productID := envOr("SSMR_SMOKE_SKU", "SSMR-SKU-1")

	createBody, _ := json.Marshal(map[string]any{
		"retailer_id": retailerID,
		"product_id":  productID,
		"price":       overridePrice,
		"notes":       "SSMR override",
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/pricing/retailer-overrides", createBody, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("create retailer override status %d body %s", status, string(respBody))
	}
	var created struct {
		OverrideID string `json:"override_id"`
		Price      int64  `json:"price"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return err
	}
	if created.OverrideID == "" || created.Price != overridePrice {
		return fmt.Errorf("create retailer override invalid response: %s", string(respBody))
	}

	listURL := base + "/v1/supplier/pricing/retailer-overrides?retailer_id=" + retailerID + "&product_id=" + productID
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, listURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("list retailer overrides status %d body %s", status, string(respBody))
	}
	var listed struct {
		Overrides []struct {
			OverrideID string `json:"override_id"`
			Price      int64  `json:"price"`
		} `json:"overrides"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(respBody, &listed); err != nil {
		return err
	}
	if listed.Total < 1 || len(listed.Overrides) == 0 || listed.Overrides[0].Price != overridePrice {
		return fmt.Errorf("list retailer overrides missing active row: %s", string(respBody))
	}

	quoteBody, _ := json.Marshal(map[string]any{
		"supplier_id": supplierID,
		"lines": []map[string]any{
			{"product_id": productID, "quantity": 1, "unit_price_minor": 50000, "currency": "UZS"},
		},
	})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/retailer/checkout/quote", quoteBody, retailerToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("checkout quote status %d body %s", status, string(respBody))
	}
	var quote struct {
		Lines []struct {
			UnitPrice int64 `json:"unit_price_minor"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(respBody, &quote); err != nil {
		return err
	}
	if len(quote.Lines) == 0 || quote.Lines[0].UnitPrice != overridePrice {
		return fmt.Errorf("checkout quote did not apply override: %s", string(respBody))
	}

	if err := assertInboxContainsEvent(ctx, client, base, retailerToken, events.EventRetailerPriceOverride); err != nil {
		return fmt.Errorf("retailer inbox after override: %w", err)
	}

	deleteURL := base + "/v1/supplier/pricing/retailer-overrides/" + created.OverrideID
	status, respBody, _, err = clientDo(ctx, client, http.MethodDelete, deleteURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("delete retailer override status %d body %s", status, string(respBody))
	}

	fmt.Println("PX_E2E_RETAILER_PRICING_OVERRIDE_OK")
	return nil
}

func runCheckoutPreviewE2E(ctx context.Context, client *http.Client, base, retailerToken string) error {
	sku := envOr("SSMR_SMOKE_SKU", "SSMR-SKU-1")
	body, _ := json.Marshal(map[string]any{
		"latitude":  41.31,
		"longitude": 69.24,
		"items": []map[string]any{
			{"sku_id": sku, "quantity": 1, "unit_price": 1000},
		},
	})
	status, respBody, _, err := clientDoRetry(ctx, client, http.MethodPost, base+"/v1/checkout/preview", body, retailerToken, "ssmr-checkout-preview")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("checkout preview status %d body %s", status, string(respBody))
	}
	var preview struct {
		OK               bool  `json:"ok"`
		DeliveryFeeMinor int64 `json:"delivery_fee_minor"`
	}
	if err := json.Unmarshal(respBody, &preview); err != nil {
		return fmt.Errorf("decode checkout preview: %w", err)
	}
	if !preview.OK {
		return fmt.Errorf("checkout preview not ok: %s", string(respBody))
	}
	fmt.Println("PX_E2E_CHECKOUT_PREVIEW_OK")
	fmt.Println("PX_E2E_DELIVERY_FEE_PREVIEW_OK")
	return nil
}

func runUnifiedCheckout(ctx context.Context, client *http.Client, base, retailerToken, orderID string, cfg *bootstrap.Config, supplierID string) (string, error) {
	return runPayAtDeliveryCheckout(ctx, client, base, retailerToken, orderID, cfg, supplierID)
}

func runCheckoutPolicyGraceE2E(
	ctx context.Context,
	client *http.Client,
	base string,
	cfg *bootstrap.Config,
	supplierID, retailerToken, cookie, h3Cell string,
) error {
	whID := demoWarehouseID()
	sku := envOr("SSMR_SMOKE_SKU", "SSMR-SKU-1")
	settingsURL := base + "/v1/warehouse/ops/settings?warehouse_id=" + whID

	patchBackorder, _ := json.Marshal(map[string]string{"default_out_of_stock_policy": "ACCEPT_BACKORDER"})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPatch, settingsURL, patchBackorder, cookie, "ssmr-policy-grace-backorder")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("policy grace backorder patch status %d body %s", status, string(respBody))
	}

	if err := setSupplierInventoryLevels(ctx, cfg, supplierID, whID, sku, 5, 0); err != nil {
		return fmt.Errorf("policy grace seed inventory: %w", err)
	}

	previewBody, _ := json.Marshal(map[string]any{
		"latitude":  cfg.DeliveryZoneCenterLat,
		"longitude": cfg.DeliveryZoneCenterLng,
		"items": []map[string]any{
			{"sku_id": sku, "quantity": 10, "unit_price": 50000},
		},
	})
	status, respBody, _, err = clientDoRetry(ctx, client, http.MethodPost, base+"/v1/checkout/preview", previewBody, retailerToken, "ssmr-policy-grace-preview")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("policy grace preview status %d body %s", status, string(respBody))
	}
	var preview struct {
		CheckoutPolicyToken string `json:"checkout_policy_token"`
	}
	if err := json.Unmarshal(respBody, &preview); err != nil {
		return err
	}
	if preview.CheckoutPolicyToken == "" {
		return fmt.Errorf("policy grace preview missing checkout_policy_token")
	}

	patchReject, _ := json.Marshal(map[string]string{"default_out_of_stock_policy": "REJECT"})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, settingsURL, patchReject, cookie, "ssmr-policy-grace-reject")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("policy grace reject patch status %d body %s", status, string(respBody))
	}

	createBody, _ := json.Marshal(map[string]any{
		"line_items": []map[string]any{
			{"sku": sku, "quantity": 10, "unit_price_minor": 50000},
		},
		"h3_cell":               h3Cell,
		"lat":                   cfg.DeliveryZoneCenterLat,
		"lng":                   cfg.DeliveryZoneCenterLng,
		"checkout_policy_token": preview.CheckoutPolicyToken,
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/order/create", createBody, retailerToken, "ssmr-policy-grace-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("policy grace create status %d body %s", status, string(respBody))
	}
	var created struct {
		BackorderedItemCount int `json:"backordered_item_count"`
	}
	_ = json.Unmarshal(respBody, &created)
	if created.BackorderedItemCount < 1 {
		return fmt.Errorf("policy grace expected backorder child, got %s", string(respBody))
	}

	fmt.Println("PX_E2E_CHECKOUT_POLICY_GRACE_OK")
	return nil
}
