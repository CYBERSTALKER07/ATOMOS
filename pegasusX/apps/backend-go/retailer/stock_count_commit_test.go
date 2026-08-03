package retailer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func offlineOn() *bool {
	v := true
	return &v
}

func offlineOff() *bool {
	v := false
	return &v
}

func TestStockCountCommit_Disabled404(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now: time.Now, NewID: func() string { return "c1" },
		OfflineCountEnabled: offlineOff(),
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/stock/counts/commit",
		bytes.NewBufferString(`{"location_id":"L","base_version":0,"lines":[{"sku":"A","counted_qty":1}]}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "r", RetailerRole: "OWNER",
	}))
	rr := httptest.NewRecorder()
	svc.HandleStockCountCommit(rr, req)
	if rr.Code != http.StatusNotFound || !strings.Contains(rr.Body.String(), "OFFLINE_COUNT_DISABLED") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestStockCountCommit_VersionConflictAndForce(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "cnt-" + string(rune('a'+n))
		},
		OfflineCountEnabled: offlineOn(),
	})
	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-cnt")
	if err != nil {
		t.Fatal(err)
	}
	// Seed stock → version becomes 1
	if err := svc.applyDelta(t.Context(), "ret-cnt", primary.LocationID, BinFloor, "SKU-A", 10, MoveReceive, "REF", "o1", "u", ""); err != nil {
		t.Fatal(err)
	}
	ver, _ := svc.GetStockLocationVersion(t.Context(), "ret-cnt", primary.LocationID, BinFloor)
	if ver != 1 {
		t.Fatalf("version after receive=%d want 1", ver)
	}

	// Another mutation → version 2
	_ = svc.applyDelta(t.Context(), "ret-cnt", primary.LocationID, BinFloor, "SKU-A", -1, MoveSale, "SALE", "s1", "u", "")
	ver, _ = svc.GetStockLocationVersion(t.Context(), "ret-cnt", primary.LocationID, BinFloor)
	if ver != 2 {
		t.Fatalf("version=%d want 2", ver)
	}

	// Commit with stale base_version=1 → 409 with server lines
	body := map[string]any{
		"location_id":  primary.LocationID,
		"stock_bin":    BinFloor,
		"base_version": 1,
		"force":        false,
		"lines":        []map[string]any{{"sku_id": "SKU-A", "counted_qty": 5}},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/stock/counts/commit", bytes.NewReader(raw))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "clerk", Role: auth.RoleRetailer, RetailerOrgID: "ret-cnt",
		RetailerRole: "STOCK_CLERK", RetailerUserID: "clerk",
	}))
	rr := httptest.NewRecorder()
	svc.HandleStockCountCommit(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var conf map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &conf)
	if conf["error"] != "COUNT_VERSION_CONFLICT" {
		t.Fatalf("error=%v", conf["error"])
	}
	if int64(conf["server_version"].(float64)) != 2 {
		t.Fatalf("server_version=%v", conf["server_version"])
	}
	lines, ok := conf["server_lines"].([]any)
	if !ok || len(lines) == 0 {
		t.Fatalf("server_lines missing: %v", conf)
	}
	// on_hand should be 9
	sl := lines[0].(map[string]any)
	if int64(sl["on_hand"].(float64)) != 9 {
		t.Fatalf("on_hand=%v want 9", sl["on_hand"])
	}

	// Cashier force denied
	body["force"] = true
	raw, _ = json.Marshal(body)
	reqF := httptest.NewRequest(http.MethodPost, "/v1/retailer/stock/counts/commit", bytes.NewReader(raw))
	reqF = reqF.WithContext(auth.WithClaims(reqF.Context(), auth.Claims{
		Subject: "cashier", Role: auth.RoleRetailer, RetailerOrgID: "ret-cnt",
		RetailerRole: "CASHIER", RetailerUserID: "cashier",
	}))
	// CASHIER may lack stock.count — use STOCK_CLERK for force denial role test
	reqF = reqF.WithContext(auth.WithClaims(reqF.Context(), auth.Claims{
		Subject: "clerk2", Role: auth.RoleRetailer, RetailerOrgID: "ret-cnt",
		RetailerRole: "STOCK_CLERK", RetailerUserID: "clerk2",
	}))
	rrF := httptest.NewRecorder()
	svc.HandleStockCountCommit(rrF, reqF)
	if rrF.Code != http.StatusForbidden {
		t.Fatalf("force clerk status=%d body=%s", rrF.Code, rrF.Body.String())
	}

	// Manager force succeeds + audit
	body["force"] = true
	body["force_reason"] = "manager override after recount"
	raw, _ = json.Marshal(body)
	reqM := httptest.NewRequest(http.MethodPost, "/v1/retailer/stock/counts/commit", bytes.NewReader(raw))
	reqM = reqM.WithContext(auth.WithClaims(reqM.Context(), auth.Claims{
		Subject: "mgr", Role: auth.RoleRetailer, RetailerOrgID: "ret-cnt",
		RetailerRole: "MANAGER", RetailerUserID: "mgr",
	}))
	rrM := httptest.NewRecorder()
	svc.HandleStockCountCommit(rrM, reqM)
	if rrM.Code != http.StatusCreated {
		t.Fatalf("force manager status=%d body=%s", rrM.Code, rrM.Body.String())
	}
	var okResp map[string]any
	_ = json.Unmarshal(rrM.Body.Bytes(), &okResp)
	if okResp["forced"] != true {
		t.Fatalf("forced=%v", okResp["forced"])
	}
	onHand, _ := svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-A")
	if onHand != 5 {
		t.Fatalf("on_hand after force count=%d want 5", onHand)
	}
	svc.mu.RLock()
	audits := len(svc.countForceAudits)
	svc.mu.RUnlock()
	if audits != 1 {
		t.Fatalf("force audits=%d want 1", audits)
	}

	// Matching base_version commits without force
	ver, _ = svc.GetStockLocationVersion(t.Context(), "ret-cnt", primary.LocationID, BinFloor)
	body2 := map[string]any{
		"location_id":  primary.LocationID,
		"base_version": ver,
		"lines":        []map[string]any{{"sku": "SKU-A", "counted_qty": 6}},
	}
	raw2, _ := json.Marshal(body2)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/retailer/stock/counts/commit", bytes.NewReader(raw2))
	req2 = req2.WithContext(auth.WithClaims(req2.Context(), auth.Claims{
		Subject: "clerk", Role: auth.RoleRetailer, RetailerOrgID: "ret-cnt",
		RetailerRole: "STOCK_CLERK", RetailerUserID: "clerk",
	}))
	rr2 := httptest.NewRecorder()
	svc.HandleStockCountCommit(rr2, req2)
	if rr2.Code != http.StatusCreated {
		t.Fatalf("fresh commit status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	onHand, _ = svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-A")
	if onHand != 6 {
		t.Fatalf("on_hand=%d want 6", onHand)
	}
}

func TestStockCountVersion_GET(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now, NewID: func() string { return "v" }})
	primary, _ := svc.EnsurePrimaryLocation(t.Context(), "ret-v")
	_ = svc.applyDelta(t.Context(), "ret-v", primary.LocationID, BinFloor, "X", 3, MoveReceive, "R", "1", "u", "")
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/stock/counts/version?location_id="+primary.LocationID, nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-v", RetailerRole: "OWNER",
	}))
	rr := httptest.NewRecorder()
	svc.HandleStockCountVersion(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if int64(resp["version"].(float64)) != 1 {
		t.Fatalf("version=%v", resp["version"])
	}
}
