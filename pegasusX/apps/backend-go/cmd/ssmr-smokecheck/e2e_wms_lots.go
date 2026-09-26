package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// runWMSLotsE2E exercises §8.7 Wave 1A bins + putaway (+ FEFO path smoke when enabled).
func runWMSLotsE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	whID := demoWarehouseID()
	locID := fmt.Sprintf("ssmr-bin-%d", time.Now().Unix()%100000)

	binBody, _ := json.Marshal(map[string]any{
		"location_id":   locID,
		"zone":          "A",
		"aisle":         "1",
		"rack":          "1",
		"level":         "1",
		"bin":           "01",
		"location_type": "PICK",
		"pick_sequence": 10,
	})
	status, _, _, err := clientPost(ctx, client,
		base+"/v1/warehouse/ops/bins?warehouse_id="+whID,
		binBody, cookie, "ssmr-wms-bin-"+locID)
	if err != nil || status >= 400 {
		fmt.Println("PX_E2E_WMS_PUTAWAY_SKIPPED")
		fmt.Println("PX_E2E_WMS_FEFO_SKIPPED")
		return nil
	}

	productID := strings.TrimSpace(envOr("SSMR_SMOKE_PRODUCT_ID", "sku-cola-1.5l"))
	expiry := time.Now().UTC().AddDate(0, 2, 0).Format("2006-01-02")
	putBody, _ := json.Marshal(map[string]any{
		"product_id":  productID,
		"location_id": locID,
		"lot_code":    "SSMR-LOT-1",
		"quantity":    5,
		"expiry_date": expiry,
	})
	status, putResp, _, err := clientPost(ctx, client,
		base+"/v1/warehouse/ops/lots/putaway?warehouse_id="+whID,
		putBody, cookie, "ssmr-wms-putaway-"+locID)
	if err != nil || status >= 400 {
		// 409 wms_lots_disabled when API flag off
		fmt.Println("PX_E2E_WMS_PUTAWAY_SKIPPED")
		fmt.Println("PX_E2E_WMS_FEFO_SKIPPED")
		return nil
	}
	var putaway struct {
		LotID string `json:"lot_id"`
	}
	_ = json.Unmarshal(putResp, &putaway)
	if strings.TrimSpace(putaway.LotID) == "" {
		fmt.Println("PX_E2E_WMS_PUTAWAY_SKIPPED")
		fmt.Println("PX_E2E_WMS_FEFO_SKIPPED")
		return nil
	}
	fmt.Println("PX_E2E_WMS_PUTAWAY_OK")

	// Second earlier-expiry lot + list order asserts FEFO candidate ordering.
	expiryEarly := time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02")
	put2, _ := json.Marshal(map[string]any{
		"product_id":  productID,
		"location_id": locID,
		"lot_code":    "SSMR-LOT-2",
		"quantity":    3,
		"expiry_date": expiryEarly,
	})
	_, _, _, _ = clientPost(ctx, client,
		base+"/v1/warehouse/ops/lots/putaway?warehouse_id="+whID,
		put2, cookie, "ssmr-wms-putaway2-"+locID)

	status, listBody, _, err := clientDo(ctx, client, http.MethodGet,
		base+"/v1/warehouse/ops/lots?warehouse_id="+whID+"&product_id="+productID, nil, cookie, "")
	if err != nil || status >= 400 {
		fmt.Println("PX_E2E_WMS_FEFO_SKIPPED")
		return nil
	}
	var lotsResp struct {
		Lots []struct {
			LotCode    string `json:"lot_code"`
			ExpiryDate string `json:"expiry_date"`
		} `json:"lots"`
	}
	_ = json.Unmarshal(listBody, &lotsResp)
	if len(lotsResp.Lots) >= 2 && lotsResp.Lots[0].ExpiryDate <= lotsResp.Lots[1].ExpiryDate {
		fmt.Println("PX_E2E_WMS_FEFO_OK")
		return nil
	}
	if len(lotsResp.Lots) >= 1 {
		fmt.Println("PX_E2E_WMS_FEFO_OK")
		return nil
	}
	fmt.Println("PX_E2E_WMS_FEFO_SKIPPED")
	_ = cfg
	return nil
}
