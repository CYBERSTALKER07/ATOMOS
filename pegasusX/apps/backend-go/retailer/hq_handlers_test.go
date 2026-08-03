package retailer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func hqOn() *bool {
	v := true
	return &v
}

func hqOff() *bool {
	v := false
	return &v
}

func TestHqAPI_Disabled404(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now: time.Now, NewID: func() string { return "x" },
		HqAnalyticsEnabled: hqOff(),
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/hq/summary", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "org", RetailerRole: "OWNER",
	}))
	rr := httptest.NewRecorder()
	svc.HandleHqSummary(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "HQ_ANALYTICS_DISABLED") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestHqAPI_CashierForbidden(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now: time.Now, NewID: func() string { return "x" },
		HqAnalyticsEnabled: hqOn(),
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/hq/summary", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "c", Role: auth.RoleRetailer, RetailerOrgID: "org", RetailerRole: "CASHIER",
	}))
	rr := httptest.NewRecorder()
	svc.HandleHqSummary(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHqAPI_SalesByLocationBalanced(t *testing.T) {
	t.Parallel()
	fixed := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		Now:                func() time.Time { return fixed },
		NewID:              func() string { return "x" },
		HqAnalyticsEnabled: hqOn(),
	})
	svc.mu.Lock()
	svc.applyHqSalesDeltaMemory(hqSalesKey{
		RetailerID: "ret-hq2", LocationID: "loc-a", Day: "2026-08-12", SkuID: "S1",
	}, 1, 0, 1000, 1000, "UZS")
	svc.applyHqSalesDeltaMemory(hqSalesKey{
		RetailerID: "ret-hq2", LocationID: "loc-b", Day: "2026-08-12", SkuID: "S2",
	}, 2, 0, 2000, 2000, "UZS")
	// local: SKU included
	svc.applyHqSalesDeltaMemory(hqSalesKey{
		RetailerID: "ret-hq2", LocationID: "loc-a", Day: "2026-08-12", SkuID: "local:X",
	}, 1, 0, 100, 100, "UZS")
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/hq/sales-by-location?day=2026-08-12", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-hq2", RetailerRole: "OWNER",
	}))
	rr := httptest.NewRecorder()
	svc.HandleHqSalesByLocation(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		OrgNetMinor int64 `json:"org_net_minor"`
		SumLocations int64 `json:"sum_locations"`
		Balanced    bool  `json:"balanced"`
		Items       []struct {
			LocationID string `json:"location_id"`
			NetMinor   int64  `json:"net_minor"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Balanced || resp.OrgNetMinor != resp.SumLocations {
		t.Fatalf("not balanced: %+v", resp)
	}
	if resp.OrgNetMinor != 3100 {
		t.Fatalf("org_net=%d want 3100", resp.OrgNetMinor)
	}

	// Summary
	reqS := httptest.NewRequest(http.MethodGet, "/v1/retailer/hq/summary?day=2026-08-12", nil)
	reqS = reqS.WithContext(auth.WithClaims(reqS.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-hq2", RetailerRole: "OWNER",
	}))
	rrS := httptest.NewRecorder()
	svc.HandleHqSummary(rrS, reqS)
	if rrS.Code != http.StatusOK {
		t.Fatalf("summary status=%d", rrS.Code)
	}
	var sum map[string]any
	_ = json.Unmarshal(rrS.Body.Bytes(), &sum)
	if sum["net_minor"].(float64) != 3100 {
		t.Fatalf("summary net=%v", sum["net_minor"])
	}

	// By SKU includes local
	reqK := httptest.NewRequest(http.MethodGet, "/v1/retailer/hq/sales-by-sku?day=2026-08-12", nil)
	reqK = reqK.WithContext(auth.WithClaims(reqK.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-hq2", RetailerRole: "OWNER",
	}))
	rrK := httptest.NewRecorder()
	svc.HandleHqSalesBySku(rrK, reqK)
	var skuResp struct {
		Items []struct {
			SkuID   string `json:"sku_id"`
			IsLocal bool   `json:"is_local"`
		} `json:"items"`
	}
	_ = json.Unmarshal(rrK.Body.Bytes(), &skuResp)
	foundLocal := false
	for _, it := range skuResp.Items {
		if it.SkuID == "local:X" && it.IsLocal {
			foundLocal = true
		}
	}
	if !foundLocal {
		t.Fatalf("expected local sku in by-sku: %+v", skuResp.Items)
	}
}
