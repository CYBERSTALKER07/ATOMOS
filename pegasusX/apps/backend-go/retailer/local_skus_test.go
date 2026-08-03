package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestNormalizeLocalSKUID(t *testing.T) {
	t.Parallel()
	if got := NormalizeLocalSKUID("tea"); got != "local:tea" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeLocalSKUID("local:tea"); got != "local:tea" {
		t.Fatalf("got %q", got)
	}
	if !IsLocalSKU("local:x") || IsLocalSKU("SKU-1") {
		t.Fatal("IsLocalSKU")
	}
}

func TestLocalSKUCRUDAndPOSCatalog(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now:   func() time.Time { return time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC) },
		NewID: func() string { n++; return "ls" + string(rune('0'+n%10)) },
	})
	owner := auth.Claims{
		Subject: "o1", Role: auth.RoleRetailer, RetailerOrgID: "ret-local",
		RetailerRole: "OWNER", RetailerUserID: "o1",
	}
	// Create
	body := `{"name":"House tea","barcode":"123","default_price_minor":5000,"local_sku_id":"tea"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/local-skus", strings.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleLocalSKUs(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}
	var created LocalSKU
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.LocalSkuID != "local:tea" || created.Name != "House tea" {
		t.Fatalf("created=%+v", created)
	}

	// List
	req = httptest.NewRequest(http.MethodGet, "/v1/retailer/local-skus?q=tea", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr = httptest.NewRecorder()
	svc.HandleLocalSKUs(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d", rr.Code)
	}
	var list struct {
		Items []LocalSKU `json:"items"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	if len(list.Items) != 1 {
		t.Fatalf("list=%+v", list)
	}

	// POS catalog
	req = httptest.NewRequest(http.MethodGet, "/v1/retailer/pos/catalog?q=tea", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr = httptest.NewRecorder()
	svc.HandlePOSCatalogSearch(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", rr.Code, rr.Body.String())
	}

	// Validate POS line via barcode
	sku, name, errMsg := svc.validatePosSaleSKU(context.Background(), "ret-local", "123", "", 5000)
	if errMsg != "" || sku != "local:tea" || name != "House tea" {
		t.Fatalf("validate barcode sku=%q name=%q err=%q", sku, name, errMsg)
	}
	sku, _, errMsg = svc.validatePosSaleSKU(context.Background(), "ret-local", "PEG-1", "Peg", 100)
	if errMsg != "" || sku != "PEG-1" {
		t.Fatalf("validate pegasus sku=%q err=%q", sku, errMsg)
	}

	// Patch deactivate
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("localSkuID", "local:tea")
	req = httptest.NewRequest(http.MethodPatch, "/v1/retailer/local-skus/local:tea", strings.NewReader(`{"is_active":false}`))
	req = req.WithContext(auth.WithClaims(context.WithValue(req.Context(), chi.RouteCtxKey, rctx), owner))
	rr = httptest.NewRecorder()
	svc.HandleLocalSKUByID(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rr.Code, rr.Body.String())
	}
}
