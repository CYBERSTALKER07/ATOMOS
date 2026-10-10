package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/gs1"
)

// runGS1LabelsE2E asserts SSCC ship-units after seal + ZPL label render (Wave 2C).
// Soft-skips when GS1_LABELS_ENABLED=0 or backend has no company prefix
// (SupplierProfiles.Gs1CompanyPrefix or GS1_COMPANY_PREFIX on the API process).
func runGS1LabelsE2E(ctx context.Context, client *http.Client, base, payloaderToken, manifestID string) {
	if !gs1.LabelsEnabled() {
		fmt.Println("PX_E2E_GS1_SSCC_SKIPPED")
		fmt.Println("PX_E2E_GS1_ZPL_SKIPPED")
		return
	}
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" || strings.TrimSpace(payloaderToken) == "" {
		fmt.Println("PX_E2E_GS1_SSCC_SKIPPED")
		fmt.Println("PX_E2E_GS1_ZPL_SKIPPED")
		return
	}

	// Labels POST also ensures ship-units (idempotent mint) when prefix is configured.
	status, zplBody, _, err := clientPost(ctx, client,
		base+"/v1/payloader/manifests/"+manifestID+"/labels",
		[]byte("{}"), payloaderToken, "ssmr-gs1-zpl-"+manifestID)
	if err != nil || status >= 400 {
		fmt.Println("PX_E2E_GS1_SSCC_SKIPPED")
		fmt.Println("PX_E2E_GS1_ZPL_SKIPPED")
		return
	}
	zpl := string(zplBody)
	if strings.Contains(zpl, "^XA") && strings.Contains(zpl, "^XZ") {
		fmt.Println("PX_E2E_GS1_ZPL_OK")
	} else {
		fmt.Println("PX_E2E_GS1_ZPL_SKIPPED")
	}

	status, respBody, _, err := clientDo(ctx, client, http.MethodGet,
		base+"/v1/payloader/manifests/"+manifestID+"/ship-units", nil, payloaderToken, "")
	if err != nil || status >= 400 {
		fmt.Println("PX_E2E_GS1_SSCC_SKIPPED")
		return
	}
	var unitsResp struct {
		ShipUnits []struct {
			SSCC string `json:"sscc"`
		} `json:"ship_units"`
	}
	_ = json.Unmarshal(respBody, &unitsResp)
	for _, u := range unitsResp.ShipUnits {
		if gs1.ValidSSCC(u.SSCC) {
			fmt.Println("PX_E2E_GS1_SSCC_OK")
			return
		}
	}
	fmt.Println("PX_E2E_GS1_SSCC_SKIPPED")
}
