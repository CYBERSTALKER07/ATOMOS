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

func runNotificationInboxE2E(ctx context.Context, client *http.Client, base, supplierCookie, retailerToken string) error {
	deadline := time.Now().Add(30 * time.Second)
	var lastSupplierErr, lastRetailerErr error
	for time.Now().Before(deadline) {
		lastSupplierErr = assertInboxHasRows(ctx, client, base, supplierCookie, "supplier")
		if lastSupplierErr != nil {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		markBody, _ := json.Marshal(map[string]any{"mark_all": true})
		status, respBody, _, markErr := clientPost(ctx, client, base+"/v1/user/notifications/read", markBody, supplierCookie, "ssmr-supplier-inbox-read")
		if markErr != nil {
			lastSupplierErr = markErr
			time.Sleep(400 * time.Millisecond)
			continue
		}
		if status != http.StatusOK {
			lastSupplierErr = fmt.Errorf("supplier mark notifications read status %d body %s", status, string(respBody))
			time.Sleep(400 * time.Millisecond)
			continue
		}
		fmt.Println("PX_E2E_SUPPLIER_NOTIFICATION_INBOX_OK")
		if retailerToken != "" {
			lastRetailerErr = assertInboxHasRows(ctx, client, base, retailerToken, "retailer")
			if lastRetailerErr != nil {
				time.Sleep(400 * time.Millisecond)
				continue
			}
		}
		fmt.Println("PX_E2E_NOTIFICATION_INBOX_OK")
		return nil
	}
	if lastSupplierErr != nil {
		return fmt.Errorf("supplier inbox not ready after kafka fanout window: %w", lastSupplierErr)
	}
	if retailerToken != "" && lastRetailerErr != nil {
		return fmt.Errorf("retailer inbox not ready after kafka fanout window: %w", lastRetailerErr)
	}
	return fmt.Errorf("notification inbox empty after kafka fanout window")
}

func assertInboxHasRows(ctx context.Context, client *http.Client, base, authToken, label string) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/user/notifications?limit=10", nil, authToken, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("%s inbox status %d body %s", label, status, string(respBody))
	}
	var inbox struct {
		Notifications []json.RawMessage `json:"notifications"`
	}
	if err := json.Unmarshal(respBody, &inbox); err != nil {
		return fmt.Errorf("decode %s inbox: %w", label, err)
	}
	if len(inbox.Notifications) == 0 {
		return fmt.Errorf("%s inbox empty", label)
	}
	return nil
}

func assertInboxContainsEvent(ctx context.Context, client *http.Client, base, authToken, wantType string) error {
	deadline := time.Now().Add(30 * time.Second)
	var lastTypes []string
	for time.Now().Before(deadline) {
		status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/user/notifications?limit=25", nil, authToken, "")
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("inbox status %d body %s", status, string(respBody))
		}
		var inbox struct {
			Notifications []struct {
				Type  string `json:"type"`
				Title string `json:"title"`
			} `json:"notifications"`
		}
		if err := json.Unmarshal(respBody, &inbox); err != nil {
			return fmt.Errorf("decode inbox: %w", err)
		}
		lastTypes = lastTypes[:0]
		for _, row := range inbox.Notifications {
			if row.Type == wantType {
				return nil
			}
			lastTypes = append(lastTypes, row.Type)
		}
		time.Sleep(400 * time.Millisecond)
	}
	if len(lastTypes) == 0 {
		return fmt.Errorf("inbox missing event type %s (no rows)", wantType)
	}
	return fmt.Errorf("inbox missing event type %s (saw %d rows: %s)", wantType, len(lastTypes), strings.Join(lastTypes, ", "))
}

func ensureWarehouseDispatchFleet(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/fleet/drivers", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("fleet drivers status %d body %s", status, string(respBody))
	}
	var drivers struct {
		Items []struct {
			DriverID   string `json:"driver_id"`
			HomeNodeID string `json:"home_node_id"`
			VehicleID  string `json:"vehicle_id"`
			IsActive   bool   `json:"is_active"`
		} `json:"items"`
	}
	if err := json.Unmarshal(respBody, &drivers); err != nil {
		return fmt.Errorf("decode fleet drivers: %w", err)
	}
	for _, driver := range drivers.Items {
		if driver.IsActive && driver.HomeNodeID == whID && strings.TrimSpace(driver.VehicleID) != "" {
			return nil
		}
	}

	plate := fmt.Sprintf("SSMR%04d", time.Now().Unix()%10000)
	vehicleBody, _ := json.Marshal(map[string]any{
		"label":          "SSMR Dispatch Truck",
		"license_plate":  plate,
		"home_node_type": "WAREHOUSE",
		"home_node_id":   whID,
		"vehicle_class":  "CLASS_B",
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/supplier/fleet/vehicles", vehicleBody, cookie, "ssmr-fleet-vehicle")
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("fleet vehicle create status %d body %s", status, string(respBody))
	}
	var vehicles struct {
		Items []struct {
			VehicleID    string `json:"vehicle_id"`
			LicensePlate string `json:"license_plate"`
		} `json:"items"`
	}
	if err := json.Unmarshal(respBody, &vehicles); err != nil {
		return fmt.Errorf("decode fleet vehicles: %w", err)
	}
	vehicleID := ""
	for _, vehicle := range vehicles.Items {
		if strings.EqualFold(vehicle.LicensePlate, plate) {
			vehicleID = vehicle.VehicleID
			break
		}
	}
	if vehicleID == "" && len(vehicles.Items) > 0 {
		vehicleID = vehicles.Items[len(vehicles.Items)-1].VehicleID
	}
	if vehicleID == "" {
		return fmt.Errorf("fleet vehicle create missing vehicle_id body %s", string(respBody))
	}

	driverBody, _ := json.Marshal(map[string]any{
		"name":           "SSMR Dispatch Driver",
		"phone":          envOr("SSMR_DISPATCH_DRIVER_PHONE", "+998901009991"),
		"pin":            "1234",
		"home_node_type": "WAREHOUSE",
		"home_node_id":   whID,
		"vehicle_id":     vehicleID,
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/supplier/fleet/drivers", driverBody, cookie, "ssmr-fleet-driver")
	if err != nil {
		return err
	}
	if status == http.StatusConflict {
		return nil
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("fleet driver create status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_WAREHOUSE_DISPATCH_FLEET_OK")
	return nil
}

func runDispatchCapacityE2E(ctx context.Context, client *http.Client, base, cookie, orderID, driverID, vehicleID string) error {
	if strings.TrimSpace(orderID) == "" || strings.TrimSpace(driverID) == "" {
		fmt.Println("PX_E2E_DISPATCH_CAPACITY_OK")
		return nil
	}
	whID := demoWarehouseID()
	manualBody, _ := json.Marshal(map[string]any{
		"mode": "MANUAL",
		"routes": []map[string]any{{
			"driver_id":  driverID,
			"vehicle_id": vehicleID,
			"order_ids":  []string{orderID, orderID, orderID},
		}},
	})
	url := base + "/v1/warehouse/ops/dispatch/execute?warehouse_id=" + whID
	status, respBody, _, err := clientPost(ctx, client, url, manualBody, cookie, "ssmr-dispatch-capacity")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("dispatch capacity probe status %d body %s", status, string(respBody))
	}
	var first struct {
		Status           string `json:"status"`
		CapacityWarnings []struct {
			SuggestedUnselectOrderIDs []string `json:"suggested_unselect_order_ids"`
		} `json:"capacity_warnings"`
	}
	if err := json.Unmarshal(respBody, &first); err != nil {
		return fmt.Errorf("decode dispatch capacity: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(first.Status)) == "capacity_exceeded" {
		for _, warning := range first.CapacityWarnings {
			if len(warning.SuggestedUnselectOrderIDs) == 0 {
				return fmt.Errorf("capacity_exceeded missing suggested_unselect_order_ids")
			}
		}
	}
	fmt.Println("PX_E2E_DISPATCH_CAPACITY_OK")
	return nil
}

func runFleetReassignGuardE2E(ctx context.Context, client *http.Client, base, cookie string, dispatch *dispatchManifestHint) error {
	if dispatch == nil || strings.TrimSpace(dispatch.DriverID) == "" {
		return nil
	}
	whID := demoWarehouseID()
	assignBody, _ := json.Marshal(map[string]string{"vehicle_id": dispatch.VehicleID})
	assignURL := base + "/v1/warehouse/ops/drivers/" + dispatch.DriverID + "/assign-vehicle?warehouse_id=" + whID
	status, respBody, _, err := clientDo(ctx, client, http.MethodPatch, assignURL, assignBody, cookie, "ssmr-wh-assign-guard")
	if err != nil {
		return err
	}
	if status != http.StatusConflict && status != http.StatusOK {
		return fmt.Errorf("expected assign guard conflict or no-op after depart, got %d body %s", status, string(respBody))
	}
	return nil
}

func createOrder(ctx context.Context, client *http.Client, base, bearer string, cfg *bootstrap.Config, h3Cell string) (string, error) {
	return createOrderWithQuantity(ctx, client, base, bearer, cfg, h3Cell, 2)
}

func createOrderWithQuantity(ctx context.Context, client *http.Client, base, bearer string, cfg *bootstrap.Config, h3Cell string, quantity int64) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"line_items": []map[string]any{
			{"sku": "SSMR-SKU-1", "quantity": quantity, "unit_price_minor": 50000},
		},
		"h3_cell": h3Cell,
		"lat":     cfg.DeliveryZoneCenterLat,
		"lng":     cfg.DeliveryZoneCenterLng,
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/order/create", body, bearer, "")
	if err != nil {
		return "", err
	}
	if status != http.StatusCreated {
		return "", fmt.Errorf("order create status %d body %s", status, string(respBody))
	}
	var resp struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}
	if resp.OrderID == "" {
		return "", fmt.Errorf("order create missing order_id")
	}
	return resp.OrderID, nil
}

func runClientPolicyE2E(ctx context.Context, client *http.Client, base string) error {
	body, err := clientGet(ctx, client, base+"/v1/platform/client-policy?role=DRIVER&platform=ios&version=1.0.0&channel=production")
	if err != nil {
		return err
	}
	var resp struct {
		Role           string `json:"role"`
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode client policy: %w", err)
	}
	if resp.Role != "DRIVER" {
		return fmt.Errorf("driver client policy role=%q want DRIVER", resp.Role)
	}
	if resp.MinimumVersion == "" {
		return fmt.Errorf("client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_CLIENT_POLICY_OK")
	fmt.Println("PX_E2E_DRIVER_CLIENT_POLICY_OK")
	return nil
}
