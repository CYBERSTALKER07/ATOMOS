package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// e2eTimeout bounds the full multi-role SSMR smoke path (supplier through driver edges).
func runWarehouseDispatchLock(ctx context.Context, client *http.Client, base, cookie, orderID string) error {
	body, _ := json.Marshal(map[string]string{
		"entity_type": "ORDER",
		"entity_id":   orderID,
		"reason":      "ssmr-smoke-lock",
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/warehouse/dispatch-lock", body, cookie, "ssmr-lock-"+orderID)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("dispatch lock acquire status %d body %s", status, string(respBody))
	}
	var lock struct {
		LockID string `json:"lock_id"`
	}
	_ = json.Unmarshal(respBody, &lock)
	if lock.LockID == "" {
		return fmt.Errorf("dispatch lock missing lock_id")
	}
	releaseURL := base + "/v1/warehouse/dispatch-lock?lock_id=" + lock.LockID
	status, respBody, _, err = clientDo(ctx, client, http.MethodDelete, releaseURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("dispatch lock release status %d body %s", status, string(respBody))
	}
	return nil
}

func runWarehouseDispatchSettingsE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	getURL := base + "/v1/warehouse/ops/dispatch/settings?warehouse_id=" + whID
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, getURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("dispatch settings get status %d body %s", status, string(respBody))
	}
	patchBody, _ := json.Marshal(map[string]bool{"auto_dispatch_enabled": true})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, getURL, patchBody, cookie, "ssmr-dispatch-settings")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("dispatch settings patch status %d body %s", status, string(respBody))
	}
	return nil
}

func runWarehouseOpsPolicyE2E(ctx context.Context, client *http.Client, base, cookie, retailerToken string) error {
	whID := demoWarehouseID()
	settingsURL := base + "/v1/warehouse/ops/settings?warehouse_id=" + whID
	patchBody, _ := json.Marshal(map[string]any{
		"preorder_min_lead_days":  int64(5),
		"preorder_max_lead_days":  int64(60),
		"order_line_max_quantity": int64(50),
		"delivery_fee_rules": map[string]any{
			"currency":       "UZS",
			"base_fee_minor": int64(0),
			"tiers": []map[string]any{
				{"max_km": 5.0, "fee_minor": int64(0)},
				{"max_km": nil, "fee_minor": int64(100000)},
			},
		},
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPatch, settingsURL, patchBody, cookie, "ssmr-wh-ops-policy")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse ops policy patch status %d body %s", status, string(respBody))
	}
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, settingsURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse ops policy get status %d body %s", status, string(respBody))
	}
	var settings struct {
		PreorderMinLeadDays  int64  `json:"preorder_min_lead_days"`
		PreorderMaxLeadDays  int64  `json:"preorder_max_lead_days"`
		OrderLineMaxQuantity *int64 `json:"order_line_max_quantity"`
	}
	if err := json.Unmarshal(respBody, &settings); err != nil {
		return fmt.Errorf("decode warehouse ops policy: %w", err)
	}
	if settings.PreorderMinLeadDays != 5 || settings.PreorderMaxLeadDays != 60 {
		return fmt.Errorf("warehouse lead days not persisted: %s", string(respBody))
	}
	if settings.OrderLineMaxQuantity == nil || *settings.OrderLineMaxQuantity != 50 {
		return fmt.Errorf("warehouse line max not persisted: %s", string(respBody))
	}
	fmt.Println("PX_E2E_WAREHOUSE_OPS_POLICY_OK")

	lineLimitBody, _ := json.Marshal(map[string]any{
		"latitude":  41.31,
		"longitude": 69.24,
		"items": []map[string]any{
			{"sku_id": envOr("SSMR_SMOKE_SKU", "SSMR-SKU-1"), "quantity": 100, "unit_price": 1000},
		},
	})
	status, respBody, _, err = clientDoRetry(ctx, client, http.MethodPost, base+"/v1/checkout/preview", lineLimitBody, retailerToken, "ssmr-checkout-line-limit")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("checkout line limit preview status %d body %s", status, string(respBody))
	}
	var blocked struct {
		Blocked    bool              `json:"blocked"`
		Code       string            `json:"code"`
		LineErrors map[string]string `json:"line_errors"`
	}
	if err := json.Unmarshal(respBody, &blocked); err != nil {
		return fmt.Errorf("decode line limit preview: %w", err)
	}
	if !blocked.Blocked || blocked.Code != "line_quantity_out_of_range" || len(blocked.LineErrors) == 0 {
		return fmt.Errorf("expected line_quantity_out_of_range preview: %s", string(respBody))
	}
	fmt.Println("PX_E2E_CHECKOUT_LINE_LIMIT_OK")
	return nil
}

func runWarehouseStockPolicyE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	settingsURL := base + "/v1/warehouse/ops/settings?warehouse_id=" + whID

	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, settingsURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse settings get status %d body %s", status, string(respBody))
	}
	var settings struct {
		DefaultOutOfStockPolicy string `json:"default_out_of_stock_policy"`
		OpsAlwaysAvailable      bool   `json:"ops_always_available"`
	}
	if err := json.Unmarshal(respBody, &settings); err != nil {
		return fmt.Errorf("decode warehouse settings: %w", err)
	}
	if settings.DefaultOutOfStockPolicy == "" {
		return fmt.Errorf("warehouse settings missing default_out_of_stock_policy: %s", string(respBody))
	}
	if !settings.OpsAlwaysAvailable {
		return fmt.Errorf("warehouse settings expected ops_always_available=true: %s", string(respBody))
	}

	patchBody, _ := json.Marshal(map[string]string{
		"default_out_of_stock_policy": "ACCEPT_BACKORDER",
	})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, settingsURL, patchBody, cookie, "ssmr-wh-stock-policy")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse settings patch status %d body %s", status, string(respBody))
	}

	invURL := base + "/v1/warehouse/ops/inventory?warehouse_id=" + whID
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, invURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse inventory get status %d body %s", status, string(respBody))
	}
	var invResp struct {
		Items []struct {
			ProductID       string `json:"product_id"`
			EffectivePolicy string `json:"effective_policy"`
		} `json:"items"`
	}
	if err := json.Unmarshal(respBody, &invResp); err != nil {
		return fmt.Errorf("decode warehouse inventory: %w", err)
	}
	if len(invResp.Items) == 0 {
		return nil
	}
	productID := invResp.Items[0].ProductID
	policyURL := base + "/v1/warehouse/ops/inventory/" + productID + "/policy?warehouse_id=" + whID
	skuPolicyBody, _ := json.Marshal(map[string]string{
		"out_of_stock_policy": "INHERIT",
	})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, policyURL, skuPolicyBody, cookie, "ssmr-wh-sku-policy-"+productID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse inventory policy patch status %d body %s", status, string(respBody))
	}
	return nil
}

func runWarehouseReplenishmentInsightE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	listURL := base + "/v1/warehouse/replenishment/insights?warehouse_id=" + whID
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, listURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("replenishment insights status %d body %s", status, string(respBody))
	}
	var insightsResp struct {
		Insights []struct {
			ID string `json:"id"`
		} `json:"insights"`
	}
	if err := json.Unmarshal(respBody, &insightsResp); err != nil {
		return fmt.Errorf("decode replenishment insights: %w", err)
	}
	if len(insightsResp.Insights) == 0 {
		return fmt.Errorf("replenishment insights empty: %s", string(respBody))
	}
	insightID := insightsResp.Insights[0].ID
	actionURL := base + "/v1/warehouse/replenishment/insights/" + insightID + "/approve?warehouse_id=" + whID
	status, respBody, _, err = clientPost(ctx, client, actionURL, nil, cookie, "ssmr-wh-insight-approve")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("replenishment insight approve status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_WAREHOUSE_REPLENISHMENT_OK")
	return nil
}

func runWarehouseBroadcastOpsE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	listURL := base + "/v1/warehouse/ops/broadcast/templates?warehouse_id=" + whID
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, listURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse broadcast templates status %d body %s", status, string(respBody))
	}
	var list struct {
		Templates []struct {
			ID    string `json:"id"`
			Scope string `json:"scope"`
		} `json:"templates"`
	}
	if err := json.Unmarshal(respBody, &list); err != nil {
		return fmt.Errorf("decode warehouse broadcast templates: %w", err)
	}
	if len(list.Templates) == 0 {
		return fmt.Errorf("warehouse broadcast templates empty")
	}

	createBody, _ := json.Marshal(map[string]any{
		"title":        "SSMR depot notice",
		"body":         "Gate delay for smoke test.",
		"default_role": "DRIVER",
		"category":     "custom",
	})
	createURL := base + "/v1/warehouse/ops/broadcast/templates?warehouse_id=" + whID
	status, respBody, _, err = clientPost(ctx, client, createURL, createBody, cookie, "ssmr-wh-broadcast-template")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("warehouse broadcast template create status %d body %s", status, string(respBody))
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return fmt.Errorf("decode warehouse broadcast template create: %w", err)
	}
	if created.ID == "" {
		return fmt.Errorf("warehouse broadcast template create missing id")
	}

	broadcastBody, _ := json.Marshal(map[string]any{
		"title": "SSMR broadcast",
		"body":  "Depot-scoped broadcast smoke.",
		"role":  "DRIVER",
	})
	broadcastURL := base + "/v1/warehouse/ops/broadcast?warehouse_id=" + whID
	status, respBody, _, err = clientPost(ctx, client, broadcastURL, broadcastBody, cookie, "ssmr-wh-broadcast-send")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse broadcast send status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_WAREHOUSE_BROADCAST_OPS_OK")
	return nil
}

func runWarehouseSupplyRequestItemsE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	createBody, _ := json.Marshal(map[string]any{
		"priority": "HIGH",
		"notes":    "ssmr supply items",
		"items": []map[string]any{
			{"product_id": "SSMR-SKU-1", "requested_quantity": 4},
		},
	})
	createURL := base + "/v1/warehouse/supply-requests?warehouse_id=" + whID
	status, respBody, _, err := clientPost(ctx, client, createURL, createBody, cookie, "ssmr-wh-supply-items-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("warehouse supply items create status %d body %s", status, string(respBody))
	}
	var created struct {
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return fmt.Errorf("decode warehouse supply items create: %w", err)
	}
	if created.RequestID == "" {
		return fmt.Errorf("warehouse supply items create missing request_id: %s", string(respBody))
	}

	listURL := base + "/v1/warehouse/supply-requests?warehouse_id=" + whID
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, listURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse supply list status %d body %s", status, string(respBody))
	}
	var listResp struct {
		Requests []struct {
			RequestID     string  `json:"request_id"`
			Priority      string  `json:"priority"`
			Notes         string  `json:"notes"`
			ItemCount     int     `json:"item_count"`
			TotalVolumeVU float64 `json:"total_volume_vu"`
			Items         []struct {
				ProductID         string `json:"product_id"`
				RequestedQuantity int64  `json:"requested_quantity"`
			} `json:"items"`
		} `json:"requests"`
	}
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return fmt.Errorf("decode warehouse supply list: %w", err)
	}
	var matched *struct {
		RequestID     string  `json:"request_id"`
		Priority      string  `json:"priority"`
		Notes         string  `json:"notes"`
		ItemCount     int     `json:"item_count"`
		TotalVolumeVU float64 `json:"total_volume_vu"`
		Items         []struct {
			ProductID         string `json:"product_id"`
			RequestedQuantity int64  `json:"requested_quantity"`
		} `json:"items"`
	}
	for i := range listResp.Requests {
		if listResp.Requests[i].RequestID == created.RequestID {
			matched = &listResp.Requests[i]
			break
		}
	}
	if matched == nil {
		return fmt.Errorf("warehouse supply list missing created request %s: %s", created.RequestID, string(respBody))
	}
	if matched.ItemCount < 1 || len(matched.Items) < 1 {
		return fmt.Errorf("warehouse supply list missing items for %s: %s", created.RequestID, string(respBody))
	}
	if strings.ToUpper(strings.TrimSpace(matched.Priority)) != "HIGH" {
		return fmt.Errorf("warehouse supply list priority want HIGH got %q", matched.Priority)
	}
	if strings.TrimSpace(matched.Notes) != "ssmr supply items" {
		return fmt.Errorf("warehouse supply list notes want ssmr supply items got %q", matched.Notes)
	}
	if matched.Items[0].ProductID != "SSMR-SKU-1" || matched.Items[0].RequestedQuantity != 4 {
		return fmt.Errorf("warehouse supply list item mismatch: %+v", matched.Items[0])
	}

	detailURL := base + "/v1/warehouse/supply-requests/" + created.RequestID + "?warehouse_id=" + whID
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, detailURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse supply detail status %d body %s", status, string(respBody))
	}
	var detail struct {
		RequestID string `json:"request_id"`
		ItemCount int    `json:"item_count"`
		Items     []struct {
			ProductID         string `json:"product_id"`
			RequestedQuantity int64  `json:"requested_quantity"`
		} `json:"items"`
	}
	if err := json.Unmarshal(respBody, &detail); err != nil {
		return fmt.Errorf("decode warehouse supply detail: %w", err)
	}
	if detail.RequestID != created.RequestID || detail.ItemCount < 1 || len(detail.Items) < 1 {
		return fmt.Errorf("warehouse supply detail missing items: %s", string(respBody))
	}
	if detail.Items[0].ProductID != "SSMR-SKU-1" || detail.Items[0].RequestedQuantity != 4 {
		return fmt.Errorf("warehouse supply detail item mismatch: %+v", detail.Items[0])
	}
	fmt.Println("PX_E2E_WAREHOUSE_SUPPLY_REQUEST_ITEMS_OK")
	return nil
}

func runWarehouseAnalyticsE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	supplierID := supplierIDFromJWT(cookie, cfg.JWTSecret)
	if count, err := countWarehouseImportAnomalyRows(ctx, cfg, supplierID, demoWarehouseID()); err != nil {
		return fmt.Errorf("warehouse import anomaly projection: %w", err)
	} else if count < 1 {
		return fmt.Errorf("warehouse import anomaly projection want >=1 got %d supplier=%s warehouse=%s", count, supplierID, demoWarehouseID())
	}
	for _, period := range []string{"7d", "30d"} {
		url := base + "/v1/warehouse/ops/analytics?period=" + period + "&warehouse_id=" + demoWarehouseID()
		status, respBody, _, err := clientDo(ctx, client, http.MethodGet, url, nil, cookie, "")
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("warehouse analytics %s status %d body %s", period, status, string(respBody))
		}
		var overview struct {
			Period         string `json:"period"`
			TotalOrders    int64  `json:"total_orders"`
			DailyBreakdown []struct {
				Date string `json:"date"`
			} `json:"daily_breakdown"`
			ImportFreshness struct {
				AppliedRows30d   int64  `json:"applied_rows_30d"`
				AppliedSkus30d   int64  `json:"applied_skus_30d"`
				QuantityDelta30d int64  `json:"quantity_delta_30d"`
				LastSessionID    string `json:"last_session_id"`
				LastAppliedAt    string `json:"last_applied_at"`
			} `json:"import_freshness"`
			ImportAnomalyQueue struct {
				OpenRows30d         int64  `json:"open_rows_30d"`
				AffectedSessions30d int64  `json:"affected_sessions_30d"`
				LastSessionID       string `json:"last_session_id"`
				LastDetectedAt      string `json:"last_detected_at"`
				LastDetail          string `json:"last_detail"`
			} `json:"import_anomaly_queue"`
		}
		if err := json.Unmarshal(respBody, &overview); err != nil {
			return fmt.Errorf("decode warehouse analytics %s: %w", period, err)
		}
		if overview.Period != period {
			return fmt.Errorf("warehouse analytics period mismatch want %s got %s", period, overview.Period)
		}
	}
	fmt.Println("PX_E2E_WAREHOUSE_ANALYTICS_OK")
	return nil
}

func runWarehouseClientPolicyE2E(ctx context.Context, client *http.Client, base string) error {
	body, err := clientGet(ctx, client, base+"/v1/platform/client-policy?role=WAREHOUSE&platform=web&version=1.0.0&channel=production")
	if err != nil {
		return err
	}
	var resp struct {
		Role           string `json:"role"`
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode warehouse client policy: %w", err)
	}
	if resp.Role != "WAREHOUSE" {
		return fmt.Errorf("warehouse client policy role=%q want WAREHOUSE", resp.Role)
	}
	if strings.TrimSpace(resp.MinimumVersion) == "" {
		return fmt.Errorf("warehouse client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_WAREHOUSE_CLIENT_POLICY_OK")
	return nil
}

func runWarehouseFleetLiveMapE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	url := base + "/v1/warehouse/ops/fleet/live-map?warehouse_id=" + whID
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, url, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse fleet live map status %d body %s", status, string(respBody))
	}
	var liveMap struct {
		Routes      []json.RawMessage `json:"routes"`
		WarehouseID string            `json:"warehouse_id"`
		FetchedAt   string            `json:"fetched_at"`
	}
	if err := json.Unmarshal(respBody, &liveMap); err != nil {
		return fmt.Errorf("decode warehouse fleet live map: %w", err)
	}
	if strings.TrimSpace(liveMap.WarehouseID) != whID {
		return fmt.Errorf("warehouse fleet live map warehouse_id=%q want %q", liveMap.WarehouseID, whID)
	}
	if strings.TrimSpace(liveMap.FetchedAt) == "" {
		return fmt.Errorf("warehouse fleet live map missing fetched_at")
	}
	fmt.Println("PX_E2E_WAREHOUSE_FLEET_LIVE_MAP_OK")
	return nil
}

func runWarehouseOrderMutationE2E(ctx context.Context, client *http.Client, base, cookie, orderID string) error {
	delayBody, _ := json.Marshal(map[string]string{"reason": "ssmr-warehouse-delay"})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/warehouse/ops/orders/"+orderID+"/delay", delayBody, cookie, "ssmr-wh-order-delay")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse order delay status %d body %s", status, string(respBody))
	}
	var delayResp struct {
		OrderID string `json:"order_id"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &delayResp); err != nil {
		return fmt.Errorf("decode warehouse order delay: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(delayResp.Status)) != "DELAYED" {
		return fmt.Errorf("warehouse order delay expected DELAYED got %s body %s", delayResp.Status, string(respBody))
	}
	fmt.Println("PX_E2E_WAREHOUSE_ORDER_MUTATION_OK")
	return nil
}

func runWarehouseTransferActionsE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	emergencyBody, _ := json.Marshal(map[string]any{
		"total_volume_vu": 18.0,
		"notes":           "ssmr-emergency-transfer",
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/warehouse/transfers/emergency", emergencyBody, cookie, "ssmr-wh-emergency-transfer")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("warehouse emergency transfer status %d body %s", status, string(respBody))
	}
	var emergencyResp struct {
		TransferID string `json:"transfer_id"`
		State      string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &emergencyResp); err != nil {
		return fmt.Errorf("decode warehouse emergency transfer: %w", err)
	}
	if emergencyResp.TransferID == "" || strings.ToUpper(emergencyResp.State) != "APPROVED" {
		return fmt.Errorf("warehouse emergency transfer invalid: %s", string(respBody))
	}

	forceBody, _ := json.Marshal(map[string]any{
		"total_volume_vu": 22.0,
		"notes":           "ssmr-force-receive",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/warehouse/transfers/force-receive", forceBody, cookie, "ssmr-wh-force-receive")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("warehouse force receive status %d body %s", status, string(respBody))
	}
	var forceResp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &forceResp); err != nil {
		return fmt.Errorf("decode warehouse force receive: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(forceResp.State)) != "RECEIVED" {
		return fmt.Errorf("warehouse force receive expected RECEIVED got %s body %s", forceResp.State, string(respBody))
	}

	const receiveID = "ssmr-wh-transfer-receive"
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/warehouse/transfers/"+receiveID+"/receive", nil, cookie, "ssmr-wh-receive-transfer")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse receive transfer status %d body %s", status, string(respBody))
	}
	var receiveResp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &receiveResp); err != nil {
		return fmt.Errorf("decode warehouse receive transfer: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(receiveResp.State)) != "RECEIVED" {
		return fmt.Errorf("warehouse receive transfer expected RECEIVED got %s body %s", receiveResp.State, string(respBody))
	}
	fmt.Println("PX_E2E_WAREHOUSE_TRANSFER_ACTIONS_OK")
	return nil
}

func runWarehouseDispatchPreview(ctx context.Context, client *http.Client, base, supplierCookie string) error {
	whID := demoWarehouseID()
	previewURL := base + "/v1/warehouse/ops/dispatch/preview?warehouse_id=" + whID
	status, respBody, _, err := clientPost(ctx, client, previewURL, []byte(`{}`), supplierCookie, "ssmr-dispatch-preview")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("dispatch preview status %d body %s", status, string(respBody))
	}
	if !strings.Contains(string(respBody), "preview_ready") {
		return fmt.Errorf("dispatch preview unexpected body %s", string(respBody))
	}
	return nil
}

// runWarehouseOptimizerSourceE2E asserts the OR-Tools sidecar attribution when fleet
// and at least one dispatchable order are present for preview.
func runWarehouseOptimizerSourceE2E(ctx context.Context, client *http.Client, base, supplierCookie, orderID string) error {
	whID := demoWarehouseID()
	previewURL := base + "/v1/warehouse/ops/dispatch/preview?warehouse_id=" + whID
	reqBody, _ := json.Marshal(map[string]any{
		"order_ids": []string{strings.TrimSpace(orderID)},
	})
	var preview struct {
		UndispatchedOrders []any  `json:"undispatched_orders"`
		AvailableDrivers   []any  `json:"available_drivers"`
		OptimizerSource    string `json:"optimizer_source"`
	}
	var respBody []byte
	ready := false
	for attempt := 0; attempt < 30; attempt++ {
		status, body, _, err := clientPost(ctx, client, previewURL, reqBody, supplierCookie, fmt.Sprintf("ssmr-dispatch-optimizer-preview:%d", attempt))
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("optimizer dispatch preview status %d body %s", status, string(body))
		}
		respBody = body
		if err := json.Unmarshal(respBody, &preview); err != nil {
			return fmt.Errorf("decode optimizer dispatch preview: %w", err)
		}
		if len(preview.AvailableDrivers) > 0 && len(preview.UndispatchedOrders) > 0 && preview.OptimizerSource == "optimizer" {
			ready = true
			break
		}
		time.Sleep(time.Second)
	}
	if !ready {
		if len(preview.AvailableDrivers) == 0 {
			return fmt.Errorf("optimizer dispatch preview missing available drivers body %s", string(respBody))
		}
		if len(preview.UndispatchedOrders) == 0 {
			return fmt.Errorf("optimizer dispatch preview missing undispatched orders for %s body %s", orderID, string(respBody))
		}
		return fmt.Errorf("dispatch preview expected optimizer_source=optimizer got %q body %s", preview.OptimizerSource, string(respBody))
	}
	fmt.Println("PX_E2E_OPTIMIZER_SOURCE_OK")
	return nil
}

func runWarehouseDispatchExecute(ctx context.Context, client *http.Client, base, supplierCookie, orderID, driverID, vehicleID, idempotencyKey string) (*dispatchManifestHint, error) {
	whID := demoWarehouseID()
	url := base + "/v1/warehouse/ops/dispatch/execute?warehouse_id=" + whID
	var reqBody []byte
	if strings.TrimSpace(orderID) != "" && strings.TrimSpace(driverID) != "" {
		route := map[string]any{
			"driver_id":  driverID,
			"order_ids":  []string{orderID},
		}
		if strings.TrimSpace(vehicleID) != "" {
			route["vehicle_id"] = vehicleID
		}
		reqBody, _ = json.Marshal(map[string]any{
			"mode":   "MANUAL",
			"routes": []map[string]any{route},
		})
	} else {
		reqBody = []byte(`{}`)
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		idempotencyKey = "ssmr-dispatch-execute"
	}
	status, respBody, _, err := clientPost(ctx, client, url, reqBody, supplierCookie, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("dispatch execute status %d body %s", status, string(respBody))
	}
	var result struct {
		Status           string `json:"status"`
		ManifestsCreated int    `json:"manifests_created"`
		Manifests        []struct {
			ManifestID string   `json:"manifest_id"`
			DriverID   string   `json:"driver_id"`
			VehicleID  string   `json:"vehicle_id"`
			OrderIDs   []string `json:"order_ids"`
		} `json:"manifests"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode dispatch execute: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(result.Status)) {
	case "no_op":
		if strings.TrimSpace(orderID) != "" && strings.TrimSpace(driverID) != "" {
			return nil, fmt.Errorf("dispatch execute no_op for order %s body %s", orderID, string(respBody))
		}
		fmt.Println("PX_E2E_WAREHOUSE_DISPATCH_EXECUTE_OK")
		return nil, nil
	case "dispatched":
	default:
		return nil, fmt.Errorf("dispatch execute unexpected status %q body %s", result.Status, string(respBody))
	}
	if result.ManifestsCreated <= 0 || len(result.Manifests) == 0 {
		return nil, fmt.Errorf("dispatch execute missing manifests body %s", string(respBody))
	}
	var picked *dispatchManifestHint
	for i := range result.Manifests {
		m := result.Manifests[i]
		if strings.TrimSpace(m.ManifestID) == "" {
			return nil, fmt.Errorf("dispatch execute manifest missing id body %s", string(respBody))
		}
		if orderID != "" && sliceContains(m.OrderIDs, orderID) {
			picked = &dispatchManifestHint{
				ManifestID: m.ManifestID,
				DriverID:   m.DriverID,
				VehicleID:  m.VehicleID,
				OrderIDs:   append([]string(nil), m.OrderIDs...),
			}
			break
		}
	}
	if picked == nil {
		m := result.Manifests[0]
		picked = &dispatchManifestHint{
			ManifestID: m.ManifestID,
			DriverID:   m.DriverID,
			VehicleID:  m.VehicleID,
			OrderIDs:   append([]string(nil), m.OrderIDs...),
		}
	}
	fmt.Println("PX_E2E_WAREHOUSE_DISPATCH_EXECUTE_OK")
	return picked, nil
}

func runWarehouseFleetMgmtE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config, supplierID string) (string, string, error) {
	whID := demoWarehouseID()
	plate := fmt.Sprintf("WH%04d", time.Now().Unix()%10000)
	vehicleBody, _ := json.Marshal(map[string]any{
		"label":         "SSMR WH Ops Truck",
		"license_plate": plate,
		"vehicle_class": "CLASS_A",
		"max_volume_vu": 8.0,
	})
	vehicleURL := base + "/v1/warehouse/ops/vehicles?warehouse_id=" + whID
	status, respBody, _, err := clientPost(ctx, client, vehicleURL, vehicleBody, cookie, "ssmr-wh-ops-vehicle")
	if err != nil {
		return "", "", err
	}
	if status != http.StatusCreated {
		return "", "", fmt.Errorf("warehouse ops vehicle status %d body %s", status, string(respBody))
	}
	var vehicleResp struct {
		VehicleID string `json:"vehicle_id"`
	}
	if err := json.Unmarshal(respBody, &vehicleResp); err != nil {
		return "", "", fmt.Errorf("decode warehouse vehicle: %w", err)
	}
	if vehicleResp.VehicleID == "" {
		return "", "", fmt.Errorf("warehouse vehicle missing id body %s", string(respBody))
	}

	driverBody, _ := json.Marshal(map[string]any{
		"name":  "SSMR WH Ops Driver",
		"phone": fmt.Sprintf("+99890100%04d", time.Now().Unix()%10000),
	})
	driverURL := base + "/v1/warehouse/ops/drivers?warehouse_id=" + whID
	status, respBody, _, err = clientPost(ctx, client, driverURL, driverBody, cookie, "ssmr-wh-ops-driver")
	if err != nil {
		return "", "", err
	}
	if status != http.StatusCreated {
		return "", "", fmt.Errorf("warehouse ops driver status %d body %s", status, string(respBody))
	}
	var driverResp struct {
		DriverID string `json:"driver_id"`
	}
	if err := json.Unmarshal(respBody, &driverResp); err != nil {
		return "", "", fmt.Errorf("decode warehouse driver: %w", err)
	}
	if driverResp.DriverID == "" {
		return "", "", fmt.Errorf("warehouse driver missing id body %s", string(respBody))
	}

	assignBody, _ := json.Marshal(map[string]string{"vehicle_id": vehicleResp.VehicleID})
	assignURL := base + "/v1/warehouse/ops/drivers/" + driverResp.DriverID + "/assign-vehicle?warehouse_id=" + whID
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, assignURL, assignBody, cookie, "ssmr-wh-assign-vehicle")
	if err != nil {
		return "", "", err
	}
	if status != http.StatusOK {
		return "", "", fmt.Errorf("warehouse assign vehicle status %d body %s", status, string(respBody))
	}

	driverToken, err := auth.Issue(auth.Claims{
		Subject:      driverResp.DriverID,
		Role:         auth.RoleDriver,
		SupplierID:   supplierID,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   whID,
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return "", "", fmt.Errorf("issue driver jwt: %w", err)
	}
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/driver/profile", nil, driverToken, "")
	if err != nil {
		return "", "", err
	}
	if status != http.StatusOK {
		return "", "", fmt.Errorf("driver profile status %d body %s", status, string(respBody))
	}
	var profile struct {
		VehicleID string `json:"vehicle_id"`
	}
	if err := json.Unmarshal(respBody, &profile); err != nil {
		return "", "", fmt.Errorf("decode driver profile: %w", err)
	}
	if strings.TrimSpace(profile.VehicleID) != vehicleResp.VehicleID {
		return "", "", fmt.Errorf("driver profile missing vehicle_id want %s body %s", vehicleResp.VehicleID, string(respBody))
	}

	fmt.Println("PX_E2E_WAREHOUSE_FLEET_MGMT_OK")
	fmt.Println("PX_E2E_DRIVER_ASSIGN_DETECTION_OK")
	return driverResp.DriverID, vehicleResp.VehicleID, nil
}

func runWarehouseDispatchExecuteWithWS(ctx context.Context, client *http.Client, base, supplierCookie, orderID string, cfg *bootstrap.Config, supplierID, driverID, vehicleID string) (*dispatchManifestHint, error) {
	whID := demoWarehouseID()
	whToken, err := issueRoleJWT(cfg, auth.Claims{
		Subject:      "ssmr-wh-admin-ws",
		Role:         auth.RoleWarehouseAdmin,
		SupplierID:   supplierID,
		SupplierRole: auth.RoleWarehouseAdmin,
		HomeNodeID:   whID,
	})
	if err != nil {
		return nil, fmt.Errorf("issue warehouse admin jwt: %w", err)
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL(base, whToken), nil)
	if err != nil {
		return nil, fmt.Errorf("warehouse ws dial: %w", err)
	}
	defer conn.Close()

	wsErrCh := make(chan error, 1)
	go func() {
		wsErrCh <- waitForWSMessage(ctx, conn, "DISPATCH_COMMITTED")
	}()

	hint, err := runWarehouseDispatchExecute(ctx, client, base, supplierCookie, orderID, driverID, vehicleID, "ssmr-dispatch-execute")
	if err != nil {
		return nil, err
	}
	if hint == nil {
		// no_op path — no realtime dispatch frame expected.
		return hint, nil
	}
	select {
	case wsErr := <-wsErrCh:
		if wsErr != nil {
			return nil, wsErr
		}
		fmt.Println("PX_E2E_CROSS_ROLE_DISPATCH_WS_OK")
		return hint, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(25 * time.Second):
		return nil, fmt.Errorf("warehouse dispatch ws: timed out waiting for DISPATCH_COMMITTED")
	}
}
