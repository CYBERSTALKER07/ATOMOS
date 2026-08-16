package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/seed"
)

func ensureSupplierSession(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config) (string, string, error) {
	phone := envOr("SSMR_SMOKE_SUPPLIER_PHONE", "+998901000001")
	password := envOr("SSMR_SMOKE_SUPPLIER_PASSWORD", "SmokeTest!234")

	loginBody, _ := json.Marshal(map[string]string{
		"phone":    phone,
		"password": password,
	})
	status, respBody, hdrs, err := clientPostRetry(ctx, client, base+"/v1/auth/supplier/login", loginBody, "", "")
	if err != nil {
		return "", "", err
	}
	if status == http.StatusOK {
		return supplierSessionFromResponse(respBody, hdrs, cfg)
	}

	registerBody, _ := json.Marshal(map[string]any{
		"phone": phone,
		"account": map[string]any{
			"legalName":   envOr("SSMR_SMOKE_SUPPLIER_NAME", "SSMR Smoke Supplier"),
			"contactName": "Smoke Admin",
			"email":       "smoke-supplier@pegasusx.local",
			"password":    password,
			"country":     cfg.SeedSupplierCountry,
		},
		"location": map[string]any{
			"warehouse": map[string]any{
				"name":    "SSMR Warehouse",
				"address": "Tashkent SSMR",
				"lat":     cfg.DeliveryZoneCenterLat,
				"lng":     cfg.DeliveryZoneCenterLng,
			},
			"sameAsWarehouse": true,
		},
		"business": map[string]any{
			"taxId":             "SSMR-TAX",
			"companyRegNumber":  "SSMR-REG",
			"fleetVehicleCount": 2,
			"fleetMaxVU":        100,
			"factoryCount":      1,
		},
		"categories": []string{"GENERAL"},
	})
	status, respBody, hdrs, err = clientPostRetry(ctx, client, base+"/v1/auth/supplier/register", registerBody, "", "ssmr-supplier-register")
	if err != nil {
		return "", "", err
	}
	if status == http.StatusConflict || status == http.StatusTooManyRequests {
		status, respBody, hdrs, err = clientPostRetry(ctx, client, base+"/v1/auth/supplier/login", loginBody, "", "")
		if err != nil {
			return "", "", err
		}
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return "", "", fmt.Errorf("supplier session status %d body %s", status, string(respBody))
	}
	return supplierSessionFromResponse(respBody, hdrs, cfg)
}

func supplierSessionFromResponse(respBody []byte, hdrs http.Header, cfg *bootstrap.Config) (string, string, error) {
	cookie := sessionCookie(hdrs)
	if cookie == "" {
		return "", "", fmt.Errorf("supplier session missing cookie")
	}
	var resp struct {
		SupplierID string `json:"supplier_id"`
	}
	_ = json.Unmarshal(respBody, &resp)
	sid := strings.TrimSpace(resp.SupplierID)
	if sid == "" {
		sid = supplierIDFromJWT(cookie, cfg.JWTSecret)
	}
	if sid == "" {
		return "", "", fmt.Errorf("supplier session missing supplier_id")
	}
	return sid, cookie, nil
}

func demoWarehouseID() string {
	if id := strings.TrimSpace(os.Getenv("WAREHOUSE_DEMO_ID")); id != "" {
		return id
	}
	if id := strings.TrimSpace(os.Getenv("SSMR_SMOKE_WAREHOUSE_ID")); id != "" {
		return id
	}
	return "ssmr-warehouse-1"
}

func demoFactoryID() string {
	if id := strings.TrimSpace(os.Getenv("FACTORY_DEMO_ID")); id != "" {
		return id
	}
	return "factory-demo-1"
}

func issueRoleJWT(cfg *bootstrap.Config, claims auth.Claims) (string, error) {
	return auth.Issue(claims, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
}

func wsURL(base string) string {
	root := strings.TrimRight(base, "/")
	root = strings.Replace(root, "https://", "wss://", 1)
	root = strings.Replace(root, "http://", "ws://", 1)
	return root + "/v1/ws"
}

func dialWS(ctx context.Context, base, token string) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	return websocket.DefaultDialer.DialContext(ctx, wsURL(base), header)
}

func registerRetailer(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config) (string, string, error) {
	return registerRetailerWithPhone(ctx, client, base, cfg, envOr("SSMR_SMOKE_RETAILER_PHONE", "+998901000099"))
}

func registerRetailerWithPhone(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, phone string) (string, string, error) {
	body, _ := json.Marshal(map[string]any{
		"phone":       phone,
		"name":        "SSMR Retailer",
		"supplier_id": envOr("SSMR_SMOKE_SUPPLIER_ID", seed.DefaultSupplierID),
		"lat":         cfg.DeliveryZoneCenterLat,
		"lng":         cfg.DeliveryZoneCenterLng,
	})
	status, respBody, _, err := clientPostRetry(ctx, client, base+"/v1/auth/retailer/register", body, "", "")
	if err != nil {
		return "", "", err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return "", "", fmt.Errorf("retailer register status %d body %s", status, string(respBody))
	}
	var resp struct {
		RetailerID string `json:"retailer_id"`
		H3Cell     string `json:"h3_cell"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", "", err
	}
	if resp.RetailerID == "" || len(resp.H3Cell) != 15 {
		return "", "", fmt.Errorf("retailer register invalid response: %s", string(respBody))
	}
	return resp.RetailerID, resp.H3Cell, nil
}

// grantRetailerCredit provisions a supplier-granted credit line for a
// registered retailer, mirroring the real operator engagement: supplier vets
// the retailer and grants credit before the retailer's first pay-later order.
// Order create fail-closes on missing credit profiles, so every smoke retailer
// that places orders must pass through this seam.
func grantRetailerCredit(ctx context.Context, client *http.Client, base, cookie, retailerID string, limitMinor int64) error {
	body, _ := json.Marshal(map[string]any{
		"retailer_id":        retailerID,
		"credit_limit_minor": limitMinor,
		"reason":             "ssmr-smoke-grant",
	})
	status, respBody, _, err := clientDoRetry(ctx, client, http.MethodPatch, base+"/v1/supplier/retailer-credit-profile", body, cookie, "")
	if err != nil {
		return fmt.Errorf("grant retailer credit: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("grant retailer credit status %d body %s", status, string(respBody))
	}
	return nil
}
