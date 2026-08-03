package retailer

import (
	"bytes"
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

func holdsOn() *bool {
	v := true
	return &v
}

func holdsOff() *bool {
	v := false
	return &v
}

func TestPosHolds_DisabledReturns404(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now: time.Now, NewID: func() string { return "h1" },
		PosHoldsEnabled: holdsOff(),
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/pos/holds", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "u1", Role: auth.RoleRetailer, RetailerOrgID: "org-1", RetailerRole: "OWNER",
	}))
	rr := httptest.NewRecorder()
	svc.HandlePosHolds(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "POS_HOLDS_DISABLED") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestPosHolds_ParkListResumeVoid_NoStockTouch(t *testing.T) {
	t.Parallel()
	ids := []string{"hold-1", "hold-x"}
	i := 0
	svc := NewService(ServiceConfig{
		Now:             func() time.Time { return time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC) },
		NewID:           func() string { id := ids[i%len(ids)]; i++; return id },
		PosHoldsEnabled: holdsOn(),
	})
	// Seed empty stock map so we can assert it stays empty (no OnHand writes).
	if len(svc.stockBalances) != 0 {
		t.Fatal("expected empty stock at start")
	}

	claims := auth.Claims{
		Subject: "cashier-1", Role: auth.RoleRetailer, RetailerOrgID: "org-h",
		RetailerRole: "CASHIER", RetailerUserID: "cashier-1", ActiveLocationID: "loc-1",
	}

	// Park
	parkBody := `{"location_id":"loc-1","register_id":"reg-1","cart":{"lines":[{"sku":"SKU1","qty":2}]},"note":"customer back"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/holds", bytes.NewBufferString(parkBody))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	rr := httptest.NewRecorder()
	svc.HandlePosHolds(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("park status=%d body=%s", rr.Code, rr.Body.String())
	}
	var hold PosHoldDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &hold); err != nil {
		t.Fatal(err)
	}
	if hold.Status != PosHoldHELD || hold.HoldID == "" {
		t.Fatalf("hold=%+v", hold)
	}
	if hold.LocationID != "loc-1" {
		t.Fatalf("location=%s", hold.LocationID)
	}
	// TTL ~24h
	exp, err := time.Parse(time.RFC3339Nano, hold.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	if exp.Sub(time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)) != 24*time.Hour {
		t.Fatalf("ttl=%v", exp)
	}
	if len(svc.stockBalances) != 0 {
		t.Fatal("park must not touch stock")
	}

	// List
	reqList := httptest.NewRequest(http.MethodGet, "/v1/retailer/pos/holds?location_id=loc-1", nil)
	reqList = reqList.WithContext(auth.WithClaims(reqList.Context(), claims))
	rrList := httptest.NewRecorder()
	svc.HandlePosHolds(rrList, reqList)
	if rrList.Code != http.StatusOK {
		t.Fatalf("list status=%d", rrList.Code)
	}
	var list struct {
		Items []PosHoldDTO `json:"items"`
	}
	_ = json.Unmarshal(rrList.Body.Bytes(), &list)
	if len(list.Items) != 1 {
		t.Fatalf("items=%d", len(list.Items))
	}

	// Resume wrong location
	reqBad := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/holds/"+hold.HoldID+"/resume",
		bytes.NewBufferString(`{"location_id":"loc-OTHER"}`))
	reqBad = reqBad.WithContext(auth.WithClaims(reqBad.Context(), claims))
	reqBad = withChiHoldID(reqBad, hold.HoldID)
	rrBad := httptest.NewRecorder()
	svc.HandlePosHoldResume(rrBad, reqBad)
	if rrBad.Code != http.StatusForbidden {
		t.Fatalf("wrong loc status=%d body=%s", rrBad.Code, rrBad.Body.String())
	}
	if !strings.Contains(rrBad.Body.String(), "HOLD_LOCATION_MISMATCH") {
		t.Fatalf("body=%s", rrBad.Body.String())
	}

	// Resume same location
	reqRes := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/holds/"+hold.HoldID+"/resume",
		bytes.NewBufferString(`{"location_id":"loc-1"}`))
	reqRes = reqRes.WithContext(auth.WithClaims(reqRes.Context(), claims))
	reqRes = withChiHoldID(reqRes, hold.HoldID)
	rrRes := httptest.NewRecorder()
	svc.HandlePosHoldResume(rrRes, reqRes)
	if rrRes.Code != http.StatusOK {
		t.Fatalf("resume status=%d body=%s", rrRes.Code, rrRes.Body.String())
	}
	var resumed PosHoldDTO
	_ = json.Unmarshal(rrRes.Body.Bytes(), &resumed)
	if resumed.Status != PosHoldRESUMED {
		t.Fatalf("status=%s", resumed.Status)
	}
	if len(svc.stockBalances) != 0 {
		t.Fatal("resume must not touch stock")
	}

	// Park another for void
	park2 := `{"location_id":"loc-1","cart":{"lines":[]}}`
	req2 := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/holds", bytes.NewBufferString(park2))
	req2 = req2.WithContext(auth.WithClaims(req2.Context(), claims))
	rr2 := httptest.NewRecorder()
	svc.HandlePosHolds(rr2, req2)
	var hold2 PosHoldDTO
	_ = json.Unmarshal(rr2.Body.Bytes(), &hold2)

	reqV := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/holds/"+hold2.HoldID+"/void", nil)
	reqV = reqV.WithContext(auth.WithClaims(reqV.Context(), claims))
	reqV = withChiHoldID(reqV, hold2.HoldID)
	rrV := httptest.NewRecorder()
	svc.HandlePosHoldVoid(rrV, reqV)
	if rrV.Code != http.StatusOK {
		t.Fatalf("void status=%d body=%s", rrV.Code, rrV.Body.String())
	}
	var voided PosHoldDTO
	_ = json.Unmarshal(rrV.Body.Bytes(), &voided)
	if voided.Status != PosHoldVOIDED {
		t.Fatalf("status=%s", voided.Status)
	}
	if len(svc.stockBalances) != 0 {
		t.Fatal("void must not touch stock")
	}
}

func TestPosHolds_ExpireOnResume(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		Now:             func() time.Time { return now },
		NewID:           func() string { return "hold-exp" },
		PosHoldsEnabled: holdsOn(),
	})
	claims := auth.Claims{
		Subject: "u", Role: auth.RoleRetailer, RetailerOrgID: "org-e",
		RetailerRole: "OWNER", ActiveLocationID: "loc-1",
	}
	// Manually insert expired hold
	svc.mu.Lock()
	svc.posHolds["org-e|hold-exp"] = PosHoldDTO{
		HoldID: "hold-exp", RetailerID: "org-e", LocationID: "loc-1", UserID: "u",
		Status: PosHoldHELD, Cart: json.RawMessage(`{}`),
		ExpiresAt: now.Add(-time.Minute).Format(time.RFC3339Nano),
	}
	svc.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/holds/hold-exp/resume",
		bytes.NewBufferString(`{"location_id":"loc-1"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	req = withChiHoldID(req, "hold-exp")
	rr := httptest.NewRecorder()
	svc.HandlePosHoldResume(rr, req)
	if rr.Code != http.StatusGone {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func withChiHoldID(r *http.Request, holdID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("holdID", holdID)
	return r.WithContext(contextWithChi(r, rctx))
}

func TestSweepExpiredPosHolds_Memory(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		Now:             func() time.Time { return now },
		NewID:           func() string { return "sw" },
		PosHoldsEnabled: holdsOn(),
	})
	svc.mu.Lock()
	svc.posHolds["org|past"] = PosHoldDTO{
		HoldID: "past", RetailerID: "org", LocationID: "loc", UserID: "u",
		Status: PosHoldHELD, Cart: json.RawMessage(`{}`),
		ExpiresAt: now.Add(-time.Hour).Format(time.RFC3339Nano),
	}
	svc.posHolds["org|future"] = PosHoldDTO{
		HoldID: "future", RetailerID: "org", LocationID: "loc", UserID: "u",
		Status: PosHoldHELD, Cart: json.RawMessage(`{}`),
		ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano),
	}
	svc.posHolds["org|voided"] = PosHoldDTO{
		HoldID: "voided", RetailerID: "org", LocationID: "loc", UserID: "u",
		Status: PosHoldVOIDED, Cart: json.RawMessage(`{}`),
		ExpiresAt: now.Add(-time.Hour).Format(time.RFC3339Nano),
	}
	svc.mu.Unlock()

	n, err := svc.SweepExpiredPosHolds(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expired count=%d want 1", n)
	}
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	if svc.posHolds["org|past"].Status != PosHoldEXPIRED {
		t.Fatalf("past status=%s", svc.posHolds["org|past"].Status)
	}
	if svc.posHolds["org|future"].Status != PosHoldHELD {
		t.Fatalf("future should stay HELD")
	}
	if svc.posHolds["org|voided"].Status != PosHoldVOIDED {
		t.Fatalf("voided should be untouched")
	}
	if len(svc.stockBalances) != 0 {
		t.Fatal("sweeper must not touch stock")
	}
}

func TestSweepExpiredPosHolds_DisabledNoop(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now: time.Now, NewID: func() string { return "x" },
		PosHoldsEnabled: holdsOff(),
	})
	svc.mu.Lock()
	svc.posHolds["org|past"] = PosHoldDTO{
		HoldID: "past", RetailerID: "org", LocationID: "loc", UserID: "u",
		Status: PosHoldHELD, Cart: json.RawMessage(`{}`),
		ExpiresAt: time.Now().Add(-time.Hour).Format(time.RFC3339Nano),
	}
	svc.mu.Unlock()
	n, err := svc.SweepExpiredPosHolds(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
