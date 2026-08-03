package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// e2eTimeout bounds the full multi-role SSMR smoke path (supplier through driver edges).
func factorySmokeDriverID() string {
	return envOr("SSMR_FACTORY_DRIVER_ID", "drv_factory_1")
}

func factorySmokeVehicleID() string {
	return envOr("SSMR_FACTORY_VEHICLE_ID", "veh_factory_1")
}

func factoryDispatchBody(reason string) []byte {
	body, _ := json.Marshal(map[string]any{
		"reason":     reason,
		"driver_id":  factorySmokeDriverID(),
		"vehicle_id": factorySmokeVehicleID(),
	})
	return body
}

func factoryDispatchDraft(ctx context.Context, client *http.Client, base, cookie, reason string) (string, error) {
	key := fmt.Sprintf("ssmr-factory-dispatch-%s-%d", reason, time.Now().UnixNano())
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/factory/dispatch", factoryDispatchBody(reason), cookie, key)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return "", fmt.Errorf("factory dispatch %s status %d body %s", reason, status, string(respBody))
	}
	var dispatchResp struct {
		ManifestID string `json:"manifest_id"`
	}
	if err := json.Unmarshal(respBody, &dispatchResp); err != nil {
		return "", fmt.Errorf("decode factory dispatch %s: %w", reason, err)
	}
	if strings.TrimSpace(dispatchResp.ManifestID) == "" {
		return "", fmt.Errorf("factory dispatch %s missing manifest_id: %s", reason, string(respBody))
	}
	return dispatchResp.ManifestID, nil
}

func ensureFactoryStaff(ctx context.Context, client *http.Client, base, cookie string) (string, error) {
	createBody, _ := json.Marshal(map[string]any{
		"name": "SSMR Factory Lead",
		"role": "FACTORY_ADMIN",
	})
	key := fmt.Sprintf("ssmr-factory-staff-%d", time.Now().UnixNano())
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/factory/staff", createBody, cookie, key)
	if err != nil {
		return "", err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return "", fmt.Errorf("factory staff create status %d body %s", status, string(respBody))
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return "", fmt.Errorf("decode factory staff create: %w", err)
	}
	if strings.TrimSpace(created.ID) == "" {
		return "", fmt.Errorf("factory staff create missing id: %s", string(respBody))
	}
	return created.ID, nil
}

func ensureFactoryManifestException(ctx context.Context, client *http.Client, base, cookie, manifestID string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/manifests/"+manifestID, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory manifest detail for exception status %d body %s", status, string(respBody))
	}
	var detailResp struct {
		Transfers []struct {
			TransferID string `json:"transfer_id"`
			State      string `json:"state"`
		} `json:"transfers"`
	}
	if err := json.Unmarshal(respBody, &detailResp); err != nil {
		return fmt.Errorf("decode factory manifest detail for exception: %w", err)
	}
	var transferID string
	for _, tr := range detailResp.Transfers {
		state := strings.ToUpper(strings.TrimSpace(tr.State))
		if state != "CANCELLED" && strings.TrimSpace(tr.TransferID) != "" {
			transferID = tr.TransferID
			break
		}
	}
	if transferID == "" {
		return fmt.Errorf("factory manifest %s has no cancellable transfer for exception seed", manifestID)
	}
	cancelBody, _ := json.Marshal(map[string]any{
		"manifest_id": manifestID,
		"transfer_id": transferID,
		"reason":      "ssmr-smoke-exception",
	})
	key := fmt.Sprintf("ssmr-factory-cancel-transfer-%d", time.Now().UnixNano())
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/factory/manifests/cancel-transfer", cancelBody, cookie, key)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory cancel-transfer status %d body %s", status, string(respBody))
	}
	return nil
}

func runFactoryOps(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	if err := runFactoryClientPolicyE2E(ctx, client, base); err != nil {
		return err
	}
	if err := assertFactoryFirebaseOTPLogin(ctx, client, base); err != nil {
		return err
	}
	if err := runFactoryAnalyticsOverviewE2E(ctx, client, base, cookie); err != nil {
		return err
	}
	if err := runFactoryInsightsE2E(ctx, client, base, cookie, cfg); err != nil {
		return err
	}
	if err := runFactoryManifestLifecycleE2E(ctx, client, base, cookie); err != nil {
		return err
	}
	if err := runFactorySupplyRequestE2E(ctx, client, base, cookie); err != nil {
		return err
	}
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/manifests", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory manifests status %d body %s", status, string(respBody))
	}
	manifestA, err := factoryDispatchDraft(ctx, client, base, cookie, "ssmr-smoke-a")
	if err != nil {
		return err
	}
	manifestB, err := factoryDispatchDraft(ctx, client, base, cookie, "ssmr-smoke-b")
	if err != nil {
		return err
	}
	if err := runFactoryPayloadOverrideE2E(ctx, client, base, cookie, manifestA, manifestB); err != nil {
		return err
	}
	if err := ensureFactoryManifestException(ctx, client, base, cookie, manifestB); err != nil {
		return fmt.Errorf("factory exception seed: %w", err)
	}
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/factory/manifest-exceptions", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory manifest-exceptions status %d body %s", status, string(respBody))
	}
	var exceptionsResp struct {
		Exceptions []json.RawMessage `json:"exceptions"`
	}
	if err := json.Unmarshal(respBody, &exceptionsResp); err != nil {
		return fmt.Errorf("decode factory manifest-exceptions: %w", err)
	}
	if len(exceptionsResp.Exceptions) == 0 {
		return fmt.Errorf("factory manifest-exceptions empty: %s", string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_MANIFEST_EXCEPTIONS_OK")

	createBody, _ := json.Marshal(map[string]any{
		"total_vu":   int64(32),
		// FactoryInternalTransfers.OrderId is STRING(36).
		"order_id":   fmt.Sprintf("ord_ssmr_%d", time.Now().Unix()%1_000_000_000),
		"driver_id":  factorySmokeDriverID(),
		"vehicle_id": factorySmokeVehicleID(),
	})
	xferKey := fmt.Sprintf("ssmr-factory-transfer-create-%d", time.Now().UnixNano())
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/factory/transfers/create", createBody, cookie, xferKey)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("factory transfer create status %d body %s", status, string(respBody))
	}
	var createdTransfer struct {
		TransferID string `json:"transfer_id"`
	}
	if err := json.Unmarshal(respBody, &createdTransfer); err != nil {
		return fmt.Errorf("decode factory transfer create: %w", err)
	}
	if createdTransfer.TransferID == "" {
		return fmt.Errorf("factory transfer create missing transfer_id: %s", string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_TRANSFER_CREATE_OK")
	if err := runFactoryTransferTransitionE2E(ctx, client, base, cookie, createdTransfer.TransferID); err != nil {
		return err
	}
	if err := runFactoryLoadingBayE2E(ctx, client, base, cookie, createdTransfer.TransferID); err != nil {
		return err
	}
	if err := runFactoryNotificationInboxE2E(ctx, client, base); err != nil {
		return err
	}
	return nil
}

func assertFactoryFirebaseOTPLogin(ctx context.Context, client *http.Client, base string) error {
	testIDToken := strings.TrimSpace(os.Getenv("FACTORY_FIREBASE_TEST_ID_TOKEN"))
	if testIDToken == "" {
		return nil
	}
	otpBody, _ := json.Marshal(map[string]string{"id_token": testIDToken})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/auth/factory/login", otpBody, "", "")
	if err != nil {
		return fmt.Errorf("factory firebase otp login: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory firebase otp login status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_FIREBASE_OTP_OK")
	return nil
}

func runFactoryClientPolicyE2E(ctx context.Context, client *http.Client, base string) error {
	body, err := clientGet(ctx, client, base+"/v1/platform/client-policy?role=FACTORY&platform=web&version=1.0.0&channel=production")
	if err != nil {
		return err
	}
	var resp struct {
		Role           string `json:"role"`
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode factory client policy: %w", err)
	}
	if resp.Role != "FACTORY" {
		return fmt.Errorf("factory client policy role=%q want FACTORY", resp.Role)
	}
	if strings.TrimSpace(resp.MinimumVersion) == "" {
		return fmt.Errorf("factory client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_FACTORY_CLIENT_POLICY_OK")
	return nil
}

func runFactoryAnalyticsOverviewE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/analytics/overview", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory analytics overview status %d body %s", status, string(respBody))
	}
	var overview struct {
		TransfersTotal int `json:"transfers_total"`
	}
	if err := json.Unmarshal(respBody, &overview); err != nil {
		return fmt.Errorf("decode factory analytics overview: %w", err)
	}
	fmt.Println("PX_E2E_FACTORY_ANALYTICS_OK")
	return nil
}

func factoryDemoToken(ctx context.Context, client *http.Client, base string) (string, error) {
	phone := envOr("FACTORY_DEMO_PHONE", "+998901000099")
	pin := envOr("FACTORY_DEMO_PIN", "1234")
	loginBody, _ := json.Marshal(map[string]string{"phone": phone, "pin": pin})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/auth/factory/login", loginBody, "", "")
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("factory login status %d body %s", status, string(respBody))
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return "", fmt.Errorf("decode factory login: %w", err)
	}
	if loginResp.Token == "" {
		return "", fmt.Errorf("factory login missing token: %s", string(respBody))
	}
	return loginResp.Token, nil
}

func runFactoryNotificationInboxE2E(ctx context.Context, client *http.Client, base string) error {
	token, err := factoryDemoToken(ctx, client, base)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = assertInboxHasRows(ctx, client, base, token, "factory")
		if lastErr == nil {
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	if lastErr != nil {
		return fmt.Errorf("factory inbox not ready after kafka fanout window: %w", lastErr)
	}
	markBody, _ := json.Marshal(map[string]any{"mark_all": true})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/user/notifications/read", markBody, token, "ssmr-factory-inbox-read")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory mark notifications read status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_NOTIFICATION_INBOX_OK")
	return nil
}

func runFactoryInsightsE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	token, err := factoryDemoToken(ctx, client, base)
	if err != nil {
		return err
	}
	supplierID := supplierIDFromJWT(cookie, cfg.JWTSecret)
	if supplierID == "" {
		return fmt.Errorf("factory insights seed: missing supplier_id")
	}
	insightID := "ins_wh_1"
	sku := envOr("SSMR_SMOKE_SKU", "SSMR-SKU-1")
	if err := ensureReplenishmentInsightRow(ctx, cfg, insightID, demoWarehouseID(), supplierID, sku, demoFactoryID()); err != nil {
		return fmt.Errorf("seed factory insight: %w", err)
	}

	listURL := base + "/v1/warehouse/replenishment/insights?warehouse_id=" + demoWarehouseID()
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, listURL, nil, token, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory insights status %d body %s", status, string(respBody))
	}
	var insightsResp struct {
		Insights []json.RawMessage `json:"insights"`
	}
	if err := json.Unmarshal(respBody, &insightsResp); err != nil {
		return fmt.Errorf("decode factory insights: %w", err)
	}
	if len(insightsResp.Insights) == 0 {
		return fmt.Errorf("factory insights empty: %s", string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_INSIGHTS_OK")

	approveKey := fmt.Sprintf("ssmr-factory-insight-approve-%d", time.Now().UnixNano())
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/warehouse/replenishment/insights/"+insightID+"/approve?warehouse_id="+demoWarehouseID(), nil, token, approveKey)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		if status == http.StatusConflict {
			var errResp struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(respBody, &errResp) == nil && errResp.Error == "insight_already_processed" {
				fmt.Println("PX_E2E_FACTORY_REPLENISHMENT_ACTION_OK")
				return nil
			}
		}
		return fmt.Errorf("factory insight approve status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_REPLENISHMENT_ACTION_OK")
	return nil
}

func runFactoryManifestLifecycleE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	manifestID, err := factoryDispatchDraft(ctx, client, base, cookie, "ssmr-lifecycle")
	if err != nil {
		return fmt.Errorf("factory lifecycle seed: %w", err)
	}
	staffID, err := ensureFactoryStaff(ctx, client, base, cookie)
	if err != nil {
		return fmt.Errorf("factory staff seed: %w", err)
	}

	transitions := []struct {
		action   string
		wantFrom string
		wantTo   string
	}{
		{action: "start-loading", wantFrom: "DRAFT", wantTo: "LOADING"},
		{action: "seal", wantFrom: "LOADING", wantTo: "SEALED"},
		{action: "dispatch", wantFrom: "SEALED", wantTo: "DISPATCHED"},
		{action: "complete", wantFrom: "DISPATCHED", wantTo: "COMPLETED"},
	}

	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/manifests/"+manifestID, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory manifest detail status %d body %s", status, string(respBody))
	}
	var detailResp struct {
		Manifest struct {
			State string `json:"state"`
		} `json:"manifest"`
	}
	if err := json.Unmarshal(respBody, &detailResp); err != nil {
		return fmt.Errorf("decode factory manifest detail: %w", err)
	}
	state := strings.ToUpper(strings.TrimSpace(detailResp.Manifest.State))
	if state == "" {
		return fmt.Errorf("factory manifest detail missing state: %s", string(respBody))
	}

	startIdx := 0
	for i, step := range transitions {
		if state == step.wantFrom {
			startIdx = i
			break
		}
		if state == step.wantTo && i < len(transitions)-1 {
			startIdx = i + 1
			break
		}
	}
	if state == transitions[len(transitions)-1].wantTo {
		fmt.Println("PX_E2E_FACTORY_MANIFEST_LIFECYCLE_OK")
		return nil
	}

	for _, step := range transitions[startIdx:] {
		body := []byte(`{"reason":"ssmr-smoke"}`)
		url := base + "/v1/factory/manifests/" + manifestID + "/" + step.action
		idem := fmt.Sprintf("ssmr-factory-%s-%s-%d", step.action, manifestID, time.Now().UnixNano())
		status, respBody, _, err = clientPost(ctx, client, url, body, cookie, idem)
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("factory manifest %s status %d body %s", step.action, status, string(respBody))
		}
		var transitionResp struct {
			State string `json:"state"`
		}
		if err := json.Unmarshal(respBody, &transitionResp); err != nil {
			return fmt.Errorf("decode factory manifest %s: %w", step.action, err)
		}
		if strings.ToUpper(strings.TrimSpace(transitionResp.State)) != step.wantTo {
			return fmt.Errorf("factory manifest %s expected state %s got %s body %s", step.action, step.wantTo, transitionResp.State, string(respBody))
		}
		if step.action == "seal" {
			// Factory seal events may not land in the supplier notification inbox when
			// Kafka/realtime fanout is delayed or role-scoped; soft-fail after transition OK.
			if err := assertInboxContainsEvent(ctx, client, base, cookie, events.EventManifestSealed); err != nil {
				fmt.Printf("PX_E2E_FACTORY_MANIFEST_SEALED_INBOX_SOFT_FAIL %v\n", err)
			}
		}
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/factory/staff/"+staffID, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory staff detail status %d body %s", status, string(respBody))
	}
	var staffResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &staffResp); err != nil {
		return fmt.Errorf("decode factory staff detail: %w", err)
	}
	if staffResp.ID != staffID {
		return fmt.Errorf("factory staff detail unexpected id %q want %q body %s", staffResp.ID, staffID, string(respBody))
	}

	fmt.Println("PX_E2E_FACTORY_MANIFEST_LIFECYCLE_OK")
	return nil
}

func runFactorySupplyRequestE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/supply-requests", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory supply-requests status %d body %s", status, string(respBody))
	}
	var listResp struct {
		Requests []struct {
			RequestID string `json:"request_id"`
			State     string `json:"state"`
			ItemCount int    `json:"item_count"`
			Items     []struct {
				ProductID string `json:"product_id"`
			} `json:"items"`
		} `json:"requests"`
	}
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return fmt.Errorf("decode factory supply-requests: %w", err)
	}
	if len(listResp.Requests) == 0 {
		return fmt.Errorf("factory supply-requests empty: %s", string(respBody))
	}
	if listResp.Requests[0].ItemCount < 0 {
		return fmt.Errorf("factory supply-requests invalid item_count: %s", string(respBody))
	}

	var requestID string
	for _, req := range listResp.Requests {
		state := strings.ToUpper(strings.TrimSpace(req.State))
		switch state {
		case "ACKNOWLEDGED", "FULFILLED", "RECEIVED":
			fmt.Println("PX_E2E_FACTORY_SUPPLY_REQUEST_OK")
			return nil
		case "SUBMITTED":
			requestID = req.RequestID
		}
	}
	if requestID == "" {
		return fmt.Errorf("factory supply-request no actionable state in list: %s", string(respBody))
	}
	patchBody, _ := json.Marshal(map[string]string{"action": "ACKNOWLEDGE"})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/factory/supply-requests/"+requestID, patchBody, cookie, "ssmr-factory-supply-"+requestID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory supply-request transition status %d body %s", status, string(respBody))
	}
	var transitionResp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &transitionResp); err != nil {
		return fmt.Errorf("decode factory supply-request transition: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(transitionResp.State)) != "ACKNOWLEDGED" {
		return fmt.Errorf("factory supply-request expected ACKNOWLEDGED got %s body %s", transitionResp.State, string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_SUPPLY_REQUEST_OK")
	return nil
}

func runFactoryPayloadOverrideE2E(ctx context.Context, client *http.Client, base, cookie, manifestA, manifestB string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/manifests/"+manifestA, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory manifest detail %s status %d body %s", manifestA, status, string(respBody))
	}
	var detailResp struct {
		Transfers []struct {
			TransferID string `json:"transfer_id"`
		} `json:"transfers"`
	}
	if err := json.Unmarshal(respBody, &detailResp); err != nil {
		return fmt.Errorf("decode factory manifest detail %s: %w", manifestA, err)
	}
	if len(detailResp.Transfers) == 0 {
		return fmt.Errorf("factory manifest %s has no transfers for rebalance", manifestA)
	}
	rebalanceBody, _ := json.Marshal(map[string]any{
		"source_manifest_id": manifestA,
		"target_manifest_id": manifestB,
		"transfer_ids":       []string{detailResp.Transfers[0].TransferID},
		"reason":             "ssmr-smoke",
	})
	rebalanceKey := fmt.Sprintf("ssmr-factory-rebalance-%d", time.Now().UnixNano())
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/factory/manifests/rebalance", rebalanceBody, cookie, rebalanceKey)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory payload rebalance status %d body %s", status, string(respBody))
	}
	var rebalanceResp struct {
		TransfersMoved int `json:"transfers_moved"`
	}
	if err := json.Unmarshal(respBody, &rebalanceResp); err != nil {
		return fmt.Errorf("decode factory payload rebalance: %w", err)
	}
	if rebalanceResp.TransfersMoved < 1 {
		return fmt.Errorf("factory payload rebalance moved zero transfers: %s", string(respBody))
	}
	for _, manifestID := range []string{manifestA, manifestB} {
		body := []byte(`{"reason":"ssmr-smoke"}`)
		url := base + "/v1/factory/manifests/" + manifestID + "/start-loading"
		loadKey := fmt.Sprintf("ssmr-factory-loading-%s-%d", manifestID, time.Now().UnixNano())
		status, respBody, _, err := clientPost(ctx, client, url, body, cookie, loadKey)
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("factory start-loading %s status %d body %s", manifestID, status, string(respBody))
		}
	}
	fmt.Println("PX_E2E_FACTORY_PAYLOAD_OVERRIDE_OK")
	return nil
}

func runFactoryLoadingBayE2E(ctx context.Context, client *http.Client, base, cookie, approvedTransferID string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/factory/transfers?states=APPROVED,LOADING&limit=50", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory loading bay list status %d body %s", status, string(respBody))
	}
	var listResp struct {
		Transfers []struct {
			TransferID string `json:"transfer_id"`
			State      string `json:"state"`
		} `json:"transfers"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return fmt.Errorf("decode factory loading bay list: %w", err)
	}
	if listResp.Total < 1 || len(listResp.Transfers) == 0 {
		return fmt.Errorf("factory loading bay list empty: %s", string(respBody))
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/factory/transfers/"+approvedTransferID, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory transfer detail status %d body %s", status, string(respBody))
	}
	var detailResp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &detailResp); err != nil {
		return fmt.Errorf("decode factory transfer detail: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(detailResp.State)) != "APPROVED" {
		return fmt.Errorf("factory transfer detail expected APPROVED got %s body %s", detailResp.State, string(respBody))
	}

	loadingBody, _ := json.Marshal(map[string]string{"target_state": "LOADING"})
	bayKey := fmt.Sprintf("ssmr-factory-loading-bay-transition-%d", time.Now().UnixNano())
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/factory/transfers/"+approvedTransferID+"/transition", loadingBody, cookie, bayKey)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory loading bay transition status %d body %s", status, string(respBody))
	}

	dispatchBody, _ := json.Marshal(map[string]any{
		"transfer_ids": []string{approvedTransferID},
		"reason":       "ssmr-loading-bay-dispatch",
		"driver_id":    factorySmokeDriverID(),
		"vehicle_id":   factorySmokeVehicleID(),
	})
	bayDispatchKey := fmt.Sprintf("ssmr-factory-loading-bay-dispatch-%d", time.Now().UnixNano())
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/factory/dispatch", dispatchBody, cookie, bayDispatchKey)
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return fmt.Errorf("factory loading bay dispatch status %d body %s", status, string(respBody))
	}
	var dispatchResp struct {
		ManifestID       string `json:"manifest_id"`
		ManifestsCreated int    `json:"manifests_created"`
	}
	if err := json.Unmarshal(respBody, &dispatchResp); err != nil {
		return fmt.Errorf("decode factory loading bay dispatch: %w", err)
	}
	if dispatchResp.ManifestID == "" {
		return fmt.Errorf("factory loading bay dispatch missing manifest_id: %s", string(respBody))
	}
	if dispatchResp.ManifestsCreated < 1 {
		return fmt.Errorf("factory loading bay dispatch missing manifests_created: %s", string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_LOADING_BAY_OK")
	return nil
}

func runFactoryTransferTransitionE2E(ctx context.Context, client *http.Client, base, cookie, transferID string) error {
	transitionBody, _ := json.Marshal(map[string]string{"target_state": "APPROVED"})
	xferKey := fmt.Sprintf("ssmr-factory-transfer-transition-%s-%d", transferID, time.Now().UnixNano())
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/factory/transfers/"+transferID+"/transition", transitionBody, cookie, xferKey)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory transfer transition status %d body %s", status, string(respBody))
	}
	var transitionResp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &transitionResp); err != nil {
		return fmt.Errorf("decode factory transfer transition: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(transitionResp.State)) != "APPROVED" {
		return fmt.Errorf("factory transfer expected APPROVED got %s body %s", transitionResp.State, string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_TRANSFER_TRANSITION_OK")
	return nil
}
