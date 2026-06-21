package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

func runReplenishmentSupplyChainE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	whID := demoWarehouseID()
	createURL := fmt.Sprintf("%s/v1/warehouse/supply-requests?warehouse_id=%s&start_date=2026-06-14&days=3", base, whID)
	status, respBody, _, err := clientPost(ctx, client, createURL, nil, cookie, "ssmr-replen-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("warehouse supply create status %d body %s", status, string(respBody))
	}
	var created struct {
		RequestID string `json:"request_id"`
		State     string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return fmt.Errorf("decode warehouse supply create: %w", err)
	}
	if created.RequestID == "" {
		return fmt.Errorf("warehouse supply create missing request_id: %s", string(respBody))
	}
	if state := strings.ToUpper(strings.TrimSpace(created.State)); state != "SUBMITTED" {
		return fmt.Errorf("warehouse supply create expected SUBMITTED got %s", created.State)
	}

	ackBody, _ := json.Marshal(map[string]string{"action": "ACKNOWLEDGE"})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/factory/supply-requests/"+created.RequestID, ackBody, cookie, "ssmr-replen-ack-"+created.RequestID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory supply acknowledge status %d body %s", status, string(respBody))
	}

	fulfillBody, _ := json.Marshal(map[string]string{
		"action":    "FULFILL",
		"driver_id": "drv_factory_1",
	})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/factory/supply-requests/"+created.RequestID, fulfillBody, cookie, "ssmr-replen-fulfill-"+created.RequestID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory supply fulfill status %d body %s", status, string(respBody))
	}
	var fulfillResp struct {
		State            string `json:"state"`
		LinkedTransferID string `json:"linked_transfer_id"`
		TransferMode     string `json:"transfer_mode"`
	}
	if err := json.Unmarshal(respBody, &fulfillResp); err != nil {
		return fmt.Errorf("decode factory supply fulfill: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(fulfillResp.State)) != "FULFILLED" {
		return fmt.Errorf("factory supply fulfill expected FULFILLED got %s", fulfillResp.State)
	}
	if fulfillResp.LinkedTransferID == "" {
		return fmt.Errorf("factory supply fulfill missing linked_transfer_id: %s", string(respBody))
	}
	if mode := strings.ToUpper(strings.TrimSpace(fulfillResp.TransferMode)); mode != "" && mode != "TRUCK" {
		return fmt.Errorf("replenishment supply chain expected TRUCK transfer_mode got %q", fulfillResp.TransferMode)
	}

	factoryDriverToken, err := driverBearerToken(ctx, client, base)
	if err != nil {
		return fmt.Errorf("factory driver login for supply arrive: %w", err)
	}
	arriveBody, _ := json.Marshal(map[string]float64{
		"latitude":  cfg.DeliveryZoneCenterLat,
		"longitude": cfg.DeliveryZoneCenterLng,
	})
	arriveURL := base + "/v1/driver/supply-transfers/" + fulfillResp.LinkedTransferID + "/arrive"
	status, respBody, _, err = clientPost(ctx, client, arriveURL, arriveBody, factoryDriverToken, "ssmr-factory-driver-arrive-"+fulfillResp.LinkedTransferID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("factory driver supply arrive status %d body %s", status, string(respBody))
	}
	var arriveResp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &arriveResp); err != nil {
		return fmt.Errorf("decode factory driver supply arrive: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(arriveResp.State)) != "ARRIVED" {
		return fmt.Errorf("factory driver supply arrive expected ARRIVED got %s body %s", arriveResp.State, string(respBody))
	}

	status, respBody, _, err = clientPost(ctx, client, base+"/v1/warehouse/transfers/"+fulfillResp.LinkedTransferID+"/receive", nil, cookie, "ssmr-replen-receive-"+fulfillResp.LinkedTransferID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("warehouse transfer receive status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_FACTORY_DRIVER_SUPPLY_ARRIVE_OK")
	return nil
}

func runReplenishColocateE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	if err := putSupplierTopologyColocate(ctx, client, base, cookie, cfg); err != nil {
		return err
	}
	whID := demoWarehouseID()
	createURL := fmt.Sprintf("%s/v1/warehouse/supply-requests?warehouse_id=%s&start_date=2026-06-14&days=1", base, whID)
	status, respBody, _, err := clientPost(ctx, client, createURL, nil, cookie, "ssmr-colocate-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("colocate supply create status %d body %s", status, string(respBody))
	}
	var created struct {
		RequestID    string `json:"request_id"`
		TransferMode string `json:"transfer_mode"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return fmt.Errorf("decode colocate supply create: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(created.TransferMode)) != "INTERNAL" {
		return fmt.Errorf("colocate supply expected INTERNAL transfer_mode got %q", created.TransferMode)
	}

	fulfillBody, _ := json.Marshal(map[string]string{"action": "FULFILL"})
	status, respBody, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/factory/supply-requests/"+created.RequestID, fulfillBody, cookie, "ssmr-colocate-fulfill-"+created.RequestID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("colocate fulfill status %d body %s", status, string(respBody))
	}
	var fulfillResp struct {
		State            string `json:"state"`
		LinkedTransferID string `json:"linked_transfer_id"`
	}
	if err := json.Unmarshal(respBody, &fulfillResp); err != nil {
		return fmt.Errorf("decode colocate fulfill: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(fulfillResp.State)) != "FULFILLED" || fulfillResp.LinkedTransferID == "" {
		return fmt.Errorf("colocate fulfill invalid response: %s", string(respBody))
	}

	invURL := base + "/v1/warehouse/ops/inventory?warehouse_id=" + whID
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, invURL, nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("colocate inventory status %d body %s", status, string(respBody))
	}
	if !strings.Contains(string(respBody), "replenishment-bulk-vu") {
		return fmt.Errorf("colocate inventory missing replenishment-bulk-vu: %s", string(respBody))
	}
	return nil
}

func putSupplierTopologyColocate(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	factoryID := demoFactoryID()
	body, _ := json.Marshal(map[string]any{
		"warehouses": []map[string]any{
			{
				"warehouse_id":              demoWarehouseID(),
				"name":                      "SSMR Co-locate WH",
				"lat":                       cfg.DeliveryZoneCenterLat,
				"lng":                       cfg.DeliveryZoneCenterLng,
				"coverage_radius_km":        cfg.DeliveryZoneRadiusKm,
				"transfer_mode":             "INTERNAL",
				"co_locate_with_factory_id": factoryID,
				"is_active":                 true,
				"is_on_shift":               true,
			},
		},
		"factories": []map[string]any{
			{
				"factory_id": factoryID,
				"name":       "SSMR Factory",
				"lat":        cfg.DeliveryZoneCenterLat + 0.01,
				"lng":        cfg.DeliveryZoneCenterLng + 0.01,
				"is_active":  true,
			},
		},
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPut, base+"/v1/supplier/topology", body, cookie, "ssmr-topology-colocate")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("colocate topology status %d body %s", status, string(respBody))
	}
	return nil
}

func putSupplierTopology(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	body, _ := json.Marshal(map[string]any{
		"warehouses": []map[string]any{
			{
				"warehouse_id":                demoWarehouseID(),
				"name":                        "SSMR Central WH",
				"lat":                         cfg.DeliveryZoneCenterLat,
				"lng":                         cfg.DeliveryZoneCenterLng,
				"coverage_radius_km":          cfg.DeliveryZoneRadiusKm,
				"is_active":                   true,
				"is_on_shift":                 true,
				"default_out_of_stock_policy": "REJECT",
				"initial_inventory": []map[string]any{
					{"product_id": "SSMR-SKU-1", "quantity": 5000},
				},
			},
		},
		"factories": []map[string]any{
			{
				"factory_id": demoFactoryID(),
				"name":       "SSMR Factory",
				"lat":        cfg.DeliveryZoneCenterLat + 0.01,
				"lng":        cfg.DeliveryZoneCenterLng + 0.01,
				"is_active":  true,
			},
		},
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPut, base+"/v1/supplier/topology", body, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("topology status %d body %s", status, string(respBody))
	}
	return nil
}

// runSupplierTopologyEditE2E verifies GET → PUT location edit → GET round-trip for warehouses and factories.
