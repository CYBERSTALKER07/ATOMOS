package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// e2eTimeout bounds the full multi-role SSMR smoke path (supplier through driver edges).
func runPayloaderE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID string, dispatch *dispatchManifestHint) error {
	loginBody, _ := json.Marshal(map[string]string{
		"phone": envOr("PAYLOAD_DEMO_PHONE", "+998901110022"),
		"pin":   envOr("PAYLOAD_DEMO_PIN", "33333333"),
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/auth/payloader/login", loginBody, "", "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("payloader login status %d body %s", status, string(respBody))
	}
	var login struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(respBody, &login); err != nil {
		return err
	}
	if login.Token == "" {
		return fmt.Errorf("payloader login missing token")
	}
	if login.RefreshToken == "" {
		return fmt.Errorf("payloader login missing refresh_token")
	}
	token := login.Token

	refreshBody, _ := json.Marshal(map[string]string{"refresh_token": login.RefreshToken})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/auth/payloader/refresh", refreshBody, "", "")
	if err != nil {
		return fmt.Errorf("payloader refresh: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("payloader refresh status %d body %s", status, string(respBody))
	}
	var refreshResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(respBody, &refreshResp); err != nil {
		return err
	}
	if refreshResp.Token == "" {
		return fmt.Errorf("payloader refresh missing token")
	}
	token = refreshResp.Token
	fmt.Println("PX_E2E_PAYLOAD_AUTH_REFRESH_OK")

	if testIDToken := strings.TrimSpace(os.Getenv("PAYLOAD_FIREBASE_TEST_ID_TOKEN")); testIDToken != "" {
		otpBody, _ := json.Marshal(map[string]string{"id_token": testIDToken})
		status, respBody, _, err = clientPost(ctx, client, base+"/v1/auth/payloader/login", otpBody, "", "")
		if err != nil {
			return fmt.Errorf("payloader firebase otp login: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("payloader firebase otp login status %d body %s", status, string(respBody))
		}
		fmt.Println("PX_E2E_PAYLOAD_FIREBASE_OTP_OK")
	}

	if status, _, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/payloader/trucks", nil, token, ""); err != nil {
		return fmt.Errorf("payloader trucks: %w", err)
	} else if status != http.StatusOK {
		return fmt.Errorf("payloader trucks status %d", status)
	}

	var (
		manifestID      string
		driverID        string
		vehicleID       string
		sealOrder       string
		dispatchJourney bool
	)
	if dispatch != nil && strings.TrimSpace(dispatch.ManifestID) != "" {
		dispatchJourney = true
		manifestID = dispatch.ManifestID
		driverID = dispatch.DriverID
		vehicleID = dispatch.VehicleID
		if len(dispatch.OrderIDs) > 0 {
			sealOrder = dispatch.OrderIDs[0]
		}
		fmt.Println("PX_E2E_PAYLOAD_DISPATCH_JOURNEY_OK")
	} else {
		vehicleID = "veh_payload_1"
		sealOrder = "ord_payload_1"
		driverID = envOr("PAYLOAD_DEMO_DRIVER_ID", "drv_payload_1")
		status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/payloader/manifests?state=DRAFT&truck_id="+vehicleID, nil, token, "")
		if err != nil {
			return fmt.Errorf("supplier manifests: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("payloader manifests status %d body %s", status, string(respBody))
		}
		var manifests struct {
			Manifests []struct {
				ManifestID string `json:"manifest_id"`
			} `json:"manifests"`
		}
		if err := json.Unmarshal(respBody, &manifests); err != nil {
			return err
		}
		if len(manifests.Manifests) == 0 {
			return fmt.Errorf("payloader manifests empty")
		}
		manifestID = manifests.Manifests[0].ManifestID
	}

	status, _, _, err = clientPost(ctx, client, base+"/v1/payloader/manifests/"+manifestID+"/start-loading", nil, token, "ssmr-start-"+manifestID)
	if err != nil {
		return fmt.Errorf("start-loading: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("start-loading status %d", status)
	}

	if err := assertDriverManifestGate(ctx, client, base, cfg, supplierID, driverID, manifestID, false); err != nil {
		return fmt.Errorf("driver manifest-gate pre-seal: %w", err)
	}

	if !dispatchJourney {
		if err := runPayloaderReassignE2E(ctx, client, base, token); err != nil {
			return err
		}
		fmt.Println("PX_E2E_PAYLOAD_REASSIGN_OK")
		fmt.Println("PX_E2E_REASSIGN_FLOWS_OK")
	}

	if vehicleID != "" {
		status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/payloader/orders?vehicle_id="+vehicleID+"&state=LOADED", nil, token, "")
		if err != nil {
			return fmt.Errorf("payloader orders: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("payloader orders status %d body %s", status, string(respBody))
		}
	}

	if sealOrder != "" {
		sealBody, _ := json.Marshal(map[string]any{
			"order_id":         sealOrder,
			"terminal_id":      vehicleID,
			"manifest_cleared": true,
		})
		status, _, _, err = clientPost(ctx, client, base+"/v1/payload/seal", sealBody, token, "ssmr-seal-"+sealOrder)
		if err != nil {
			return fmt.Errorf("payload seal order: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("payload seal order status %d", status)
		}
	}

	batchBody, _ := json.Marshal(map[string]any{"manifest_ids": []string{manifestID}})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/payloader/manifests/seal-completed", batchBody, token, "ssmr-seal-batch-"+manifestID)
	if err != nil {
		return fmt.Errorf("manifest seal-completed: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("manifest seal-completed status %d body %s", status, string(respBody))
	}
	var batchResp struct {
		SealedCount int `json:"sealed_count"`
	}
	_ = json.Unmarshal(respBody, &batchResp)
	if batchResp.SealedCount < 1 {
		return fmt.Errorf("manifest seal-completed expected sealed_count >= 1 body %s", string(respBody))
	}
	fmt.Println("PX_E2E_PAYLOAD_SEAL_FLOWS_OK")
	fmt.Println("PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK")

	if err := assertDriverManifestGate(ctx, client, base, cfg, supplierID, driverID, manifestID, true); err != nil {
		return fmt.Errorf("driver manifest-gate post-seal: %w", err)
	}
	fmt.Println("PX_E2E_PAYLOAD_DRIVER_GATE_OK")

	if err := assertDriverDepart(ctx, client, base, cfg, supplierID, driverID); err != nil {
		return fmt.Errorf("driver depart: %w", err)
	}
	fmt.Println("PX_E2E_PAYLOAD_DRIVER_DEPART_OK")

	if err := assertDriverManifestDetail(ctx, client, base, cfg, supplierID, driverID, manifestID, "DISPATCHED"); err != nil {
		return fmt.Errorf("driver manifest detail: %w", err)
	}

	if !dispatchJourney {
		reassignBody, _ := json.Marshal(map[string]any{
			"order_ids":    []string{"ord_payload_1"},
			"new_route_id": "drv_payload_2",
		})
		status, _, _, err = clientPost(ctx, client, base+"/v1/fleet/reassign", reassignBody, token, "ssmr-fleet-reassign")
		if err != nil {
			return fmt.Errorf("fleet reassign: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("fleet reassign status %d", status)
		}
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/user/notifications?limit=10", nil, token, "")
	if err != nil {
		return fmt.Errorf("payloader notifications: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("payloader notifications status %d body %s", status, string(respBody))
	}
	if err := runDeviceTokenE2E(ctx, client, base, token); err != nil {
		return fmt.Errorf("payloader device token: %w", err)
	}
	fmt.Println("PX_E2E_PAYLOAD_DEVICE_TOKEN_OK")
	if err := runPayloadClientPolicyE2E(ctx, client, base); err != nil {
		return fmt.Errorf("payload client policy: %w", err)
	}
	return nil
}

func runPayloaderReassignE2E(ctx context.Context, client *http.Client, base, token string) error {
	recommendBody, _ := json.Marshal(map[string]string{"order_id": "ord_payload_2"})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/payloader/recommend-reassign", recommendBody, token, "ssmr-recommend-reassign")
	if err != nil {
		return fmt.Errorf("recommend-reassign: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("recommend-reassign status %d body %s", status, string(respBody))
	}
	var recommend struct {
		OrderID         string `json:"order_id"`
		Recommendations []struct {
			DriverID  string  `json:"driver_id"`
			VehicleID string  `json:"vehicle_id"`
			Score     float64 `json:"score"`
		} `json:"recommendations"`
	}
	if err := json.Unmarshal(respBody, &recommend); err != nil {
		return err
	}
	if recommend.OrderID != "ord_payload_2" || len(recommend.Recommendations) == 0 {
		return fmt.Errorf("expected recommendations for ord_payload_2, got %#v", recommend)
	}
	targetDriver := recommend.Recommendations[0].DriverID
	if targetDriver == "" {
		targetDriver = "drv_payload_2"
	}

	applyBody, _ := json.Marshal(map[string]any{
		"order_id":       "ord_payload_2",
		"to_manifest_id": "mf_payload_2",
		"to_driver_id":   targetDriver,
		"reason":         "ssmr_balance",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/payloader/reassign-order", applyBody, token, "ssmr-reassign-order")
	if err != nil {
		return fmt.Errorf("reassign-order: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("reassign-order status %d body %s", status, string(respBody))
	}
	var applied struct {
		Status       string `json:"status"`
		ToManifestID string `json:"to_manifest_id"`
	}
	if err := json.Unmarshal(respBody, &applied); err != nil {
		return err
	}
	if applied.Status != "order_reassigned" {
		return fmt.Errorf("unexpected reassign status %q body %s", applied.Status, string(respBody))
	}
	if applied.ToManifestID != "mf_payload_2" {
		return fmt.Errorf("expected to_manifest_id mf_payload_2, got %q", applied.ToManifestID)
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/payloader/manifest-exceptions?limit=5", nil, token, "")
	if err != nil {
		return fmt.Errorf("manifest-exceptions: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("manifest-exceptions status %d body %s", status, string(respBody))
	}
	return nil
}

func runPayloadClientPolicyE2E(ctx context.Context, client *http.Client, base string) error {
	body, err := clientGet(ctx, client, base+"/v1/platform/client-policy?role=PAYLOAD&platform=android&version=1.0.0&channel=production")
	if err != nil {
		return err
	}
	var resp struct {
		Role           string `json:"role"`
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode payload client policy: %w", err)
	}
	if resp.Role != "PAYLOAD" {
		return fmt.Errorf("payload client policy role=%q want PAYLOAD", resp.Role)
	}
	if strings.TrimSpace(resp.MinimumVersion) == "" {
		return fmt.Errorf("payload client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_PAYLOAD_CLIENT_POLICY_OK")
	return nil
}
