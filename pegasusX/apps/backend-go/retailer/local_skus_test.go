package retailer

import (
	"context"
	"encoding/json"
	"errors"
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
		RetailerRole: "OWNER", RetailerUserID: "o1", MarketCode: "UZ",
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
	if created.Currency != "UZS" {
		t.Fatalf("empty create currency=%q want pack UZS", created.Currency)
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
	if list.Items[0].Currency != "UZS" {
		t.Fatalf("list currency=%q want pack UZS", list.Items[0].Currency)
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

func TestHandleLocalSKUs_ListCoalescesEmptyCurrency(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{})
	owner := auth.Claims{
		Subject: "o1", Role: auth.RoleRetailer, RetailerOrgID: "ret-empty-sku",
		RetailerRole: "OWNER", RetailerUserID: "o1", MarketCode: "UZ",
	}
	ctx := auth.WithClaims(context.Background(), owner)
	if err := svc.saveLocalSKU(ctx, "ret-empty-sku", LocalSKU{
		LocalSkuID:        "local:old",
		Name:              "Old tea",
		DefaultPriceMinor: 100,
		IsActive:          true,
	}); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/local-skus", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleLocalSKUs(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var list struct {
		Items []LocalSKU `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Currency != "UZS" {
		t.Fatalf("list=%+v want coalesced UZS", list)
	}
}

func TestHandleLocalSKUs_PatchUSDOnUZRejected(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{})
	owner := auth.Claims{
		Subject: "o1", Role: auth.RoleRetailer, RetailerOrgID: "ret-patch-usd",
		RetailerRole: "OWNER", RetailerUserID: "o1", MarketCode: "UZ",
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/local-skus",
		strings.NewReader(`{"name":"Tea","default_price_minor":100,"local_sku_id":"tea"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleLocalSKUs(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("localSkuID", "local:tea")
	req = httptest.NewRequest(http.MethodPatch, "/v1/retailer/local-skus/local:tea",
		strings.NewReader(`{"currency":"USD"}`))
	req = req.WithContext(auth.WithClaims(context.WithValue(req.Context(), chi.RouteCtxKey, rctx), owner))
	rr = httptest.NewRecorder()
	svc.HandleLocalSKUByID(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["error"] != "pack_currency_mismatch" {
		t.Fatalf("error=%v", body)
	}
}

func TestHandleLocalSKUs_USDOnUZRejected(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/local-skus",
		strings.NewReader(`{"name":"Tea","default_price_minor":100,"currency":"USD"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o1", Role: auth.RoleRetailer, RetailerOrgID: "ret-usd-sku",
		RetailerRole: "OWNER", RetailerUserID: "o1", MarketCode: "UZ",
	}))
	rr := httptest.NewRecorder()
	svc.HandleLocalSKUs(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["error"] != "pack_currency_mismatch" {
		t.Fatalf("error=%v", body)
	}
}

func TestHandleLocalSKUs_PlannedFailsClosed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/local-skus",
		strings.NewReader(`{"name":"Tea","default_price_minor":100}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o1", Role: auth.RoleRetailer, RetailerOrgID: "ret-ca-sku",
		RetailerRole: "OWNER", RetailerUserID: "o1", MarketCode: "CA",
	}))
	rr := httptest.NewRecorder()
	svc.HandleLocalSKUs(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleLocalSKUsListErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.localSKUsQuery = func(context.Context, string, string, bool) ([]LocalSKU, error) {
		return nil, errors.New("spanner_unavailable")
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/local-skus", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-lsku",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}))
	rr := httptest.NewRecorder()
	svc.HandleLocalSKUs(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "local_skus_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["items"]; ok {
		t.Fatal("failed local-skus GET must not return items[]")
	}
}

func TestHandlePOSCatalogSearchLocalSKUsErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.localSKUsQuery = func(context.Context, string, string, bool) ([]LocalSKU, error) {
		return nil, errors.New("spanner_unavailable")
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/pos/catalog?q=tea", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-cat-lsku",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}))
	rr := httptest.NewRecorder()
	svc.HandlePOSCatalogSearch(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "pos_catalog_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["items"]; ok {
		t.Fatal("failed POS catalog must not return items[]")
	}
}

func TestHandlePOSScanLocalSKUsErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.localSKUsQuery = func(context.Context, string, string, bool) ([]LocalSKU, error) {
		return nil, errors.New("spanner_unavailable")
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/scan", strings.NewReader(`{"barcode":"123"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "c", Role: auth.RoleRetailer, RetailerOrgID: "ret-scan-lsku",
		RetailerRole: "CASHIER", RetailerUserID: "c",
	}))
	rr := httptest.NewRecorder()
	svc.HandlePOSScan(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "local_skus_failed" {
		t.Fatalf("payload=%v", payload)
	}
}

func TestValidatePosSaleSKUBarcodeLookupFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{})
	svc.localSKUsQuery = func(context.Context, string, string, bool) ([]LocalSKU, error) {
		return nil, errors.New("spanner_unavailable")
	}
	sku, name, errMsg := svc.validatePosSaleSKU(context.Background(), "ret-lsku", "123", "", 5000)
	if errMsg != "local_skus_failed" || sku != "" || name != "" {
		t.Fatalf("sku=%q name=%q err=%q", sku, name, errMsg)
	}
}

func TestHandlePosSaleLocalSKUsErrorFailed(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "lsku-" + string(rune('A'+n%26))
		},
	})
	svc.localSKUsQuery = func(context.Context, string, string, bool) ([]LocalSKU, error) {
		return nil, errors.New("spanner_unavailable")
	}
	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-sale-lsku")
	if err != nil {
		t.Fatal(err)
	}
	body := `{"location_id":"` + primary.LocationID + `","label":"Till"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/registers", strings.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-sale-lsku", RetailerRole: "OWNER",
	}))
	rr := httptest.NewRecorder()
	svc.HandleRegisters(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rr.Code, rr.Body.String())
	}
	var reg RegisterDTO
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)
	openBody := `{"register_id":"` + reg.RegisterID + `","opening_float_minor":10000,"currency":"UZS"}`
	reqOpen := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/open", strings.NewReader(openBody))
	reqOpen = reqOpen.WithContext(auth.WithClaims(reqOpen.Context(), auth.Claims{
		Subject: "cashier", Role: auth.RoleRetailer, RetailerOrgID: "ret-sale-lsku",
		RetailerRole: "CASHIER", RetailerUserID: "cashier",
	}))
	rrOpen := httptest.NewRecorder()
	svc.HandlePosSessionOpen(rrOpen, reqOpen)
	if rrOpen.Code != http.StatusCreated && rrOpen.Code != http.StatusOK {
		t.Fatalf("open status=%d body=%s", rrOpen.Code, rrOpen.Body.String())
	}
	var sess PosSessionDTO
	_ = json.Unmarshal(rrOpen.Body.Bytes(), &sess)
	saleBody := `{"session_id":"` + sess.SessionID + `","stock_bin":"FLOOR","lines":[{"sku":"123","name":"Tea","qty":1,"unit_price_minor":5000}],"tenders":[{"method":"CASH","amount_minor":5000}]}`
	reqSale := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", strings.NewReader(saleBody))
	reqSale = reqSale.WithContext(auth.WithClaims(reqSale.Context(), auth.Claims{
		Subject: "cashier", Role: auth.RoleRetailer, RetailerOrgID: "ret-sale-lsku",
		RetailerRole: "CASHIER", RetailerUserID: "cashier",
	}))
	rrSale := httptest.NewRecorder()
	svc.HandlePosSale(rrSale, reqSale)
	if rrSale.Code != http.StatusInternalServerError {
		t.Fatalf("sale status=%d body=%s", rrSale.Code, rrSale.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rrSale.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "local_skus_failed" {
		t.Fatalf("payload=%v", payload)
	}
}

func TestHandlePOSCatalogSearchStockErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.stockBalancesQuery = func(context.Context, string, string) ([]StockBalanceDTO, error) {
		return nil, errors.New("spanner_unavailable")
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/pos/catalog?q=tea", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-cat",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}))
	rr := httptest.NewRecorder()
	svc.HandlePOSCatalogSearch(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "pos_catalog_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["items"]; ok {
		t.Fatal("failed POS catalog must not return items[]")
	}
}
