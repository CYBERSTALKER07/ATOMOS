package retailer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestOfflineCountVersionAndConflict(t *testing.T) {
	t.Parallel()
	enabled := true
	svc := NewService(ServiceConfig{
		Now:                 time.Now,
		NewID:               func() string { return "cnt-test-id" },
		OfflineCountEnabled: &enabled,
	})
	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-offline")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.injectMemoryReceive("ret-offline", primary.LocationID, "ord-1", []ReceiveLine{
		{Sku: "SKU-X", ProductName: "X", OrderedQty: 10, AcceptedQty: 10},
	}); err != nil {
		t.Fatal(err)
	}
	svc.setStockLocationVersionForTest(primary.LocationID, BinBackroom, 3)

	claims := auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-offline", RetailerRole: "OWNER",
		ActiveLocationID: primary.LocationID,
	}

	reqV := httptest.NewRequest(http.MethodGet, "/v1/retailer/stock/counts/version?location_id="+primary.LocationID+"&stock_bin="+BinBackroom, nil)
	reqV = reqV.WithContext(auth.WithClaims(reqV.Context(), claims))
	rrV := httptest.NewRecorder()
	svc.HandleStockCountVersion(rrV, reqV)
	if rrV.Code != http.StatusOK {
		t.Fatalf("version status=%d body=%s", rrV.Code, rrV.Body.String())
	}
	var verBody struct {
		Version int64 `json:"version"`
	}
	_ = json.Unmarshal(rrV.Body.Bytes(), &verBody)
	if verBody.Version != 3 {
		t.Fatalf("version=%d want 3", verBody.Version)
	}

	// Stale base_version → 409
	commitBody, _ := json.Marshal(map[string]any{
		"location_id":  primary.LocationID,
		"stock_bin":    BinBackroom,
		"base_version": 1,
		"force":        false,
		"lines":        []map[string]any{{"sku_id": "SKU-X", "counted_qty": 8}},
	})
	reqC := httptest.NewRequest(http.MethodPost, "/v1/retailer/stock/counts/commit", bytes.NewReader(commitBody))
	reqC = reqC.WithContext(auth.WithClaims(reqC.Context(), claims))
	rrC := httptest.NewRecorder()
	svc.HandleStockCountCommit(rrC, reqC)
	if rrC.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", rrC.Code, rrC.Body.String())
	}
	if !bytes.Contains(rrC.Body.Bytes(), []byte("COUNT_VERSION_CONFLICT")) {
		t.Fatalf("expected conflict code body=%s", rrC.Body.String())
	}

	// Matching version commits
	commitBody2, _ := json.Marshal(map[string]any{
		"location_id":  primary.LocationID,
		"stock_bin":    BinBackroom,
		"base_version": 3,
		"force":        false,
		"lines":        []map[string]any{{"sku_id": "SKU-X", "counted_qty": 8}},
	})
	reqC2 := httptest.NewRequest(http.MethodPost, "/v1/retailer/stock/counts/commit", bytes.NewReader(commitBody2))
	reqC2 = reqC2.WithContext(auth.WithClaims(reqC2.Context(), claims))
	rrC2 := httptest.NewRecorder()
	svc.HandleStockCountCommit(rrC2, reqC2)
	if rrC2.Code != http.StatusOK {
		t.Fatalf("commit status=%d body=%s", rrC2.Code, rrC2.Body.String())
	}
	onHand, _ := svc.getOnHand(t.Context(), primary.LocationID, BinBackroom, "SKU-X")
	if onHand != 8 {
		t.Fatalf("on_hand=%d want 8", onHand)
	}
	newVer, _ := svc.getStockLocationVersion(t.Context(), primary.LocationID, BinBackroom)
	if newVer <= 3 {
		t.Fatalf("version should bump after commit, got %d", newVer)
	}
}

func TestOfflineCountDisabledReturns404(t *testing.T) {
	t.Parallel()
	disabled := false
	svc := NewService(ServiceConfig{
		Now:                 time.Now,
		NewID:               func() string { return "id" },
		OfflineCountEnabled: &disabled,
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/stock/counts/version", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "u", Role: auth.RoleRetailer, RetailerOrgID: "r", RetailerRole: "OWNER",
	}))
	rr := httptest.NewRecorder()
	svc.HandleStockCountVersion(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rr.Code)
	}
}

func TestOfflineCountForceRequiresManager(t *testing.T) {
	t.Parallel()
	enabled := true
	svc := NewService(ServiceConfig{
		Now:                 time.Now,
		NewID:               func() string { return "id" },
		OfflineCountEnabled: &enabled,
	})
	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-force")
	if err != nil {
		t.Fatal(err)
	}
	svc.setStockLocationVersionForTest(primary.LocationID, BinFloor, 5)
	commitBody, _ := json.Marshal(map[string]any{
		"location_id":  primary.LocationID,
		"stock_bin":    BinFloor,
		"base_version": 0,
		"force":        true,
		"lines":        []map[string]any{{"sku_id": "SKU-Z", "counted_qty": 1}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/stock/counts/commit", bytes.NewReader(commitBody))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "cashier", Role: auth.RoleRetailer, RetailerOrgID: "ret-force", RetailerRole: "CASHIER",
		ActiveLocationID: primary.LocationID,
	}))
	rr := httptest.NewRecorder()
	svc.HandleStockCountCommit(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rr.Code, rr.Body.String())
	}
}
