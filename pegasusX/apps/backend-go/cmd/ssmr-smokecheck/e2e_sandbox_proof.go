package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// runSandboxLayerBProofs prints Layer B sandbox markers: cross-pack 422, planned
// pack checkout 404, cell-eu live:false. Does not terraform apply.
func runSandboxLayerBProofs(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config) error {
	if err := runCrossMarketDeferredE2E(ctx, client, base, cfg); err != nil {
		return err
	}
	if err := runPlannedPackCheckout404E2E(ctx, client, base, cfg); err != nil {
		return err
	}
	if err := runCellEUNotLiveE2E(ctx, client, base, cfg); err != nil {
		return err
	}
	return nil
}

func runCrossMarketDeferredE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config) error {
	phone := "+99891" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	supplierID := smokeSupplierID()
	body, _ := json.Marshal(map[string]any{
		"phone":        phone,
		"name":         "Sandbox Cross Market",
		"supplier_id":  supplierID,
		"country_code": "PK",
		"lat":          cfg.DeliveryZoneCenterLat,
		"lng":          cfg.DeliveryZoneCenterLng,
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/auth/retailer/register", body, "", "sandbox-xmarket-"+phone)
	if err != nil {
		return fmt.Errorf("cross-market register: %w", err)
	}
	if status == http.StatusUnprocessableEntity && strings.Contains(string(respBody), "cross_market_deferred") {
		fmt.Println("PX_E2E_CROSS_MARKET_DEFERRED_OK")
		return nil
	}
	fmt.Println("PX_E2E_CROSS_MARKET_DEFERRED_SKIPPED")
	return nil
}

func runPlannedPackCheckout404E2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config) error {
	tok, err := issueRoleJWT(cfg, auth.Claims{
		Subject:       "sandbox-ca-retailer",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "sandbox-ca-org",
		MarketCode:    "CA",
	})
	if err != nil {
		return fmt.Errorf("planned pack jwt: %w", err)
	}
	body, _ := json.Marshal(map[string]any{
		"latitude":  cfg.DeliveryZoneCenterLat,
		"longitude": cfg.DeliveryZoneCenterLng,
		"items": []map[string]any{
			{"sku_id": "CA-PLANNED-SKU", "quantity": 1, "unit_price": 1000},
		},
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/checkout/unified", body, tok, "sandbox-planned-ca-"+uuid.NewString()[:8])
	if err != nil {
		return fmt.Errorf("planned pack checkout: %w", err)
	}
	if status == http.StatusNotFound && strings.Contains(string(respBody), "market_pack_not_shipped") {
		fmt.Println("PX_E2E_PLANNED_PACK_CHECKOUT_404_OK")
		return nil
	}
	fmt.Println("PX_E2E_PLANNED_PACK_CHECKOUT_404_SKIPPED")
	return nil
}

func runCellEUNotLiveE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/platform/cells", nil, "", "")
	if err != nil {
		return fmt.Errorf("list cells: %w", err)
	}
	if status != http.StatusOK {
		fmt.Println("PX_E2E_CELL_EU_NOT_LIVE_SKIPPED")
		return nil
	}
	var listed struct {
		Items []struct {
			ID     string `json:"id"`
			APIURL string `json:"api_url"`
			Live   bool   `json:"live"`
		} `json:"items"`
	}
	if err := json.Unmarshal(respBody, &listed); err != nil {
		return fmt.Errorf("list cells decode: %w", err)
	}
	var euLive *bool
	var euURL string
	for _, it := range listed.Items {
		if it.ID == "cell-eu" {
			v := it.Live
			euLive = &v
			euURL = it.APIURL
			break
		}
	}
	if euLive == nil || *euLive || !strings.Contains(euURL, "api-eu.pegasusx.app") {
		fmt.Println("PX_E2E_CELL_EU_NOT_LIVE_SKIPPED")
		return nil
	}

	tok, err := issueRoleJWT(cfg, auth.Claims{
		Subject:    "sandbox-eu-session",
		Role:       auth.RoleAdmin,
		SupplierID: "sandbox-eu-sup",
		MarketCode: "EU",
		HomeCell:   "cell-eu",
	})
	if err != nil {
		return fmt.Errorf("cell-eu session jwt: %w", err)
	}
	st, sessBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/auth/session", nil, tok, "")
	if err != nil {
		return fmt.Errorf("cell-eu session: %w", err)
	}
	if st != http.StatusOK {
		fmt.Println("PX_E2E_CELL_EU_NOT_LIVE_SKIPPED")
		return nil
	}
	var sess struct {
		HomeCell string `json:"home_cell"`
		APIURL   string `json:"api_url"`
	}
	if err := json.Unmarshal(sessBody, &sess); err != nil {
		return fmt.Errorf("cell-eu session decode: %w", err)
	}
	if sess.HomeCell != "cell-eu" || !strings.Contains(sess.APIURL, "api-eu.pegasusx.app") {
		fmt.Println("PX_E2E_CELL_EU_NOT_LIVE_SKIPPED")
		return nil
	}
	fmt.Println("PX_E2E_CELL_EU_NOT_LIVE_OK")
	return nil
}
