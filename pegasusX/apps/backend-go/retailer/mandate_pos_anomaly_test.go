package retailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func setupMandateTestService(t *testing.T) (*Service, RetailerUser, RetailerUser, http.Handler) {
	t.Helper()
	var seq atomic.Int64
	fixedNow := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		JWTSecret: "retailer-mandate-secret",
		JWTIssuer: "test",
		Now:       func() time.Time { return fixedNow },
		NewID:     func() string { return fmt.Sprintf("id-%d", seq.Add(1)) },
	})
	owner, err := svc.EnsureOwnerUser(t.Context(), Retailer{RetailerID: "ret-1", Phone: "+998901110001", Name: "Main Shop"})
	if err != nil {
		t.Fatal(err)
	}
	cashierReq := httptest.NewRequest(http.MethodPost, "/v1/retailer/org/members", bytes.NewBufferString(`{"name":"Cashier Ali","phone":"+998901110002","password":"pass1234","retailer_role":"CASHIER"}`))
	cashierReq = cashierReq.WithContext(auth.WithClaims(cashierReq.Context(), auth.Claims{
		Subject: owner.UserID, Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerUserID: owner.UserID, RetailerRole: "OWNER",
	}))
	cashierRR := httptest.NewRecorder()
	svc.HandleOrgMembers(cashierRR, cashierReq)
	if cashierRR.Code != http.StatusCreated {
		t.Fatalf("create cashier failed: %d %s", cashierRR.Code, cashierRR.Body.String())
	}
	var membersResp struct {
		Items []OrgMemberDTO `json:"items"`
	}
	if err := json.Unmarshal(cashierRR.Body.Bytes(), &membersResp); err != nil {
		t.Fatal(err)
	}
	var cashierID string
	for _, it := range membersResp.Items {
		if it.RetailerRole == "CASHIER" && it.Phone == "+998901110002" {
			cashierID = it.UserID
			break
		}
	}
	cashier, found, err := svc.findRetailerUserByID(t.Context(), cashierID)
	if err != nil || !found {
		t.Fatalf("load created cashier failed: found=%v err=%v", found, err)
	}

	r := chi.NewRouter()
	r.Get("/v1/retailer/org/members/{userID}/mandate", svc.HandleOrgMemberMandate)
	r.Patch("/v1/retailer/org/members/{userID}/mandate", svc.HandleOrgMemberMandate)
	r.Put("/v1/retailer/org/members/{userID}/mandate", svc.HandleOrgMemberMandate)
	return svc, owner, cashier, r
}

func TestOrgMemberMandate_GetPatchRBACAndReceiveEnforcement(t *testing.T) {
	t.Parallel()
	svc, owner, cashier, r := setupMandateTestService(t)

	ownerClaims := auth.Claims{
		Subject: owner.UserID, Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerUserID: owner.UserID, RetailerRole: "OWNER",
	}
	cashierClaims := auth.Claims{
		Subject: cashier.UserID, Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerUserID: cashier.UserID, RetailerRole: "CASHIER",
	}

	// Default CASHIER mandate allows cash only, disables credit and dock receive.
	reqGet := httptest.NewRequest(http.MethodGet, "/v1/retailer/org/members/"+cashier.UserID+"/mandate", nil)
	reqGet = reqGet.WithContext(auth.WithClaims(reqGet.Context(), cashierClaims))
	rrGet := httptest.NewRecorder()
	r.ServeHTTP(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Fatalf("cashier self GET status = %d, want 200 (%s)", rrGet.Code, rrGet.Body.String())
	}
	var getResp OrgMemberMandateResponse
	if err := json.Unmarshal(rrGet.Body.Bytes(), &getResp); err != nil {
		t.Fatal(err)
	}
	if getResp.Mandate.CanUseCredit || !getResp.Mandate.CanPayCash || getResp.Mandate.CanReceiveGoods {
		t.Fatalf("unexpected default CASHIER mandate: %+v", getResp.Mandate)
	}

	// Cashier cannot PATCH their own mandate -> 403.
	reqSelfPatch := httptest.NewRequest(http.MethodPatch, "/v1/retailer/org/members/"+cashier.UserID+"/mandate", bytes.NewBufferString(`{"can_use_credit":true}`))
	reqSelfPatch = reqSelfPatch.WithContext(auth.WithClaims(reqSelfPatch.Context(), cashierClaims))
	rrSelfPatch := httptest.NewRecorder()
	r.ServeHTTP(rrSelfPatch, reqSelfPatch)
	if rrSelfPatch.Code != http.StatusForbidden {
		t.Fatalf("cashier self PATCH status = %d, want 403", rrSelfPatch.Code)
	}

	// Negative cap returns 400.
	reqNeg := httptest.NewRequest(http.MethodPatch, "/v1/retailer/org/members/"+cashier.UserID+"/mandate", bytes.NewBufferString(`{"daily_cash_cap_minor":-10}`))
	reqNeg = reqNeg.WithContext(auth.WithClaims(reqNeg.Context(), ownerClaims))
	rrNeg := httptest.NewRecorder()
	r.ServeHTTP(rrNeg, reqNeg)
	if rrNeg.Code != http.StatusBadRequest {
		t.Fatalf("negative cap status = %d, want 400", rrNeg.Code)
	}

	// Owner target cannot be restricted -> 409 retailer_org_member_owner_locked.
	reqOwnerLock := httptest.NewRequest(http.MethodPatch, "/v1/retailer/org/members/"+owner.UserID+"/mandate", bytes.NewBufferString(`{"can_use_credit":false}`))
	reqOwnerLock = reqOwnerLock.WithContext(auth.WithClaims(reqOwnerLock.Context(), ownerClaims))
	rrOwnerLock := httptest.NewRecorder()
	r.ServeHTTP(rrOwnerLock, reqOwnerLock)
	if rrOwnerLock.Code != http.StatusConflict {
		t.Fatalf("owner PATCH status = %d, want 409 (%s)", rrOwnerLock.Code, rrOwnerLock.Body.String())
	}

	// Owner updates cashier mandate partially; ResolveSellerMandate reflects update.
	reqPatch := httptest.NewRequest(http.MethodPatch, "/v1/retailer/org/members/"+cashier.UserID+"/mandate", bytes.NewBufferString(`{"can_use_credit":true,"daily_credit_cap_minor":75000,"max_single_order_minor":25000}`))
	reqPatch = reqPatch.WithContext(auth.WithClaims(reqPatch.Context(), ownerClaims))
	rrPatch := httptest.NewRecorder()
	r.ServeHTTP(rrPatch, reqPatch)
	if rrPatch.Code != http.StatusOK {
		t.Fatalf("owner PATCH status = %d, want 200 (%s)", rrPatch.Code, rrPatch.Body.String())
	}
	var patchResp OrgMemberMandateResponse
	if err := json.Unmarshal(rrPatch.Body.Bytes(), &patchResp); err != nil {
		t.Fatal(err)
	}
	if !patchResp.Configured || patchResp.RetailerRole != "CASHIER" || !patchResp.Mandate.CanUseCredit || !patchResp.Mandate.CanPayCash || patchResp.Mandate.CanReceiveGoods || patchResp.Mandate.DailyCreditCapMinor != 75000 || patchResp.Mandate.MaxSingleOrderMinor != 25000 {
		t.Fatalf("unexpected patched mandate: %+v", patchResp)
	}

	// Wrapped {"mandate":{...}} PATCH also works cleanly.
	reqWrapped := httptest.NewRequest(http.MethodPatch, "/v1/retailer/org/members/"+cashier.UserID+"/mandate", bytes.NewBufferString(`{"mandate":{"can_use_credit":true,"daily_cash_cap_minor":42000}}`))
	reqWrapped = reqWrapped.WithContext(auth.WithClaims(reqWrapped.Context(), ownerClaims))
	rrWrapped := httptest.NewRecorder()
	r.ServeHTTP(rrWrapped, reqWrapped)
	if rrWrapped.Code != http.StatusOK {
		t.Fatalf("wrapped PATCH status = %d, want 200 (%s)", rrWrapped.Code, rrWrapped.Body.String())
	}
	var wrappedResp OrgMemberMandateResponse
	if err := json.Unmarshal(rrWrapped.Body.Bytes(), &wrappedResp); err != nil {
		t.Fatal(err)
	}
	if wrappedResp.Mandate.DailyCashCapMinor != 42000 || wrappedResp.Mandate.DailyCreditCapMinor != 75000 {
		t.Fatalf("wrapped PATCH lost existing fields or failed update: %+v", wrappedResp)
	}

	resolved, err := svc.ResolveSellerMandate(t.Context(), "ret-1", cashier.UserID, "CASHIER")
	if err != nil {
		t.Fatalf("ResolveSellerMandate err: %v", err)
	}
	if !resolved.Found || resolved.IsOwner || !resolved.Mandate.CanUseCredit || resolved.Mandate.DailyCreditCapMinor != 75000 {
		t.Fatalf("unexpected resolved mandate: %+v", resolved)
	}

	// Enforce CanReceiveGoods=false on HandleStockReceiveSession when stock_Clerk mandate disables receiving.
	if !svc.enforceMandateReceive(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil), "ret-1", ownerClaims) {
		t.Fatal("owner should always be allowed to receive goods")
	}
	clerkClaims := auth.Claims{
		Subject: cashier.UserID, Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerUserID: cashier.UserID, RetailerRole: "STOCK_CLERK",
	}
	reqRecv := httptest.NewRequest(http.MethodPost, "/v1/retailer/stock/receive-sessions", bytes.NewBufferString(`{"order_id":"ord-1"}`))
	reqRecv = reqRecv.WithContext(auth.WithClaims(reqRecv.Context(), clerkClaims))
	rrRecv := httptest.NewRecorder()
	svc.HandleStockReceiveSession(rrRecv, reqRecv)
	if rrRecv.Code != http.StatusForbidden {
		t.Fatalf("HandleStockReceiveSession with CanReceiveGoods=false status = %d, want 403 (%s)", rrRecv.Code, rrRecv.Body.String())
	}
}

func TestPosAnomalyReview_NonAutoRejectAndControlTowerPulse(t *testing.T) {
	t.Parallel()
	var seq atomic.Int64
	fixedNow := time.Date(2026, 11, 13, 13, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		Now:   func() time.Time { return fixedNow },
		NewID: func() string { return fmt.Sprintf("pos-id-%d", seq.Add(1)) },
	})

	// Empty org keeps pulse.Empty == true and PosAnomalyReview == false.
	emptyPulse, err := svc.buildControlTowerPulse(t.Context(), "ret-empty")
	if err != nil {
		t.Fatalf("empty pulse err: %v", err)
	}
	if !emptyPulse.Empty || emptyPulse.PosAnomalyReview || emptyPulse.PosAnomalyCount != 0 {
		t.Fatalf("empty pulse unexpected: %+v", emptyPulse)
	}

	orgID := "ret-pos"
	ownerClaims := auth.Claims{
		Subject: "owner-pos", Role: auth.RoleRetailer, RetailerOrgID: orgID, RetailerUserID: "owner-pos", RetailerRole: "OWNER",
	}

	svc.mu.Lock()
	svc.posRegisters["reg-1"] = RegisterDTO{
		RegisterID: "reg-1", RetailerID: orgID, LocationID: "loc-1", Label: "Front", Status: "ACTIVE",
	}
	svc.posSessions["sess-1"] = PosSessionDTO{
		SessionID: "sess-1", RegisterID: "reg-1", LocationID: "loc-1", RetailerID: orgID,
		OpenedByUserID: "owner-pos", Status: PosSessionOpen, OpeningFloatMinor: 50000, Currency: "UZS",
		OpenedAt: fixedNow.Add(-2 * time.Hour).Format(time.RFC3339Nano),
	}
	for i := 1; i <= 3; i++ {
		saleID := fmt.Sprintf("sale-%d", i)
		svc.posSales[saleID] = PosSaleDTO{
			SaleID: saleID, SessionID: "sess-1", RegisterID: "reg-1", LocationID: "loc-1", RetailerID: orgID,
			CashierUserID: "owner-pos", Status: PosSaleCompleted, TotalMinor: 5000, Currency: "UZS",
			ReceiptNumber: fmt.Sprintf("R-%d", i),
			Tenders:       []PosTender{{Method: "CASH", AmountMinor: 5000}},
			CreatedAt:     fixedNow.Add(-time.Hour).Format(time.RFC3339Nano),
		}
	}
	svc.mu.Unlock()

	r := chi.NewRouter()
	r.Post("/v1/retailer/pos/sales/{saleID}/void", svc.HandlePosSaleVoid)
	r.Post("/v1/retailer/pos/sessions/{sessionID}/close", svc.HandlePosSessionClose)
	r.Get("/v1/retailer/control-tower/pulse", svc.HandleControlTowerPulse)

	// Void all 3 sales -> MUST return 200 OK every time (never auto-reject), flagging 3rd void with POS_ANOMALY_REVIEW.
	var thirdVoid PosSaleDTO
	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/retailer/pos/sales/sale-%d/void", i), bytes.NewBufferString(`{"reason":"customer changed mind"}`))
		req = req.WithContext(auth.WithClaims(req.Context(), ownerClaims))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("void %d status = %d, want 200 OK (non-blocking) body=%s", i, rr.Code, rr.Body.String())
		}
		if i == 3 {
			if err := json.Unmarshal(rr.Body.Bytes(), &thirdVoid); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(thirdVoid.AnomalyFlags) != 1 || thirdVoid.AnomalyFlags[0] != PosAnomalyReviewFlag || thirdVoid.AnomalyReview == nil || thirdVoid.AnomalyReview.AutoRejected {
		t.Fatalf("expected POS_ANOMALY_REVIEW non-blocking on 3rd void, got flags=%v review=%+v", thirdVoid.AnomalyFlags, thirdVoid.AnomalyReview)
	}
	if thirdVoid.AnomalyReview.VoidCount != 3 || thirdVoid.AnomalyReview.VoidRatioBps != 10000 {
		t.Fatalf("unexpected void stats: %+v", thirdVoid.AnomalyReview)
	}

	// Close session with -25,000 cash variance -> MUST return 200 OK (non-blocking), flagging CASH_MISMATCH.
	reqClose := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/sess-1/close", bytes.NewBufferString(`{"closing_cash_minor":25000}`))
	reqClose = reqClose.WithContext(auth.WithClaims(reqClose.Context(), ownerClaims))
	rrClose := httptest.NewRecorder()
	r.ServeHTTP(rrClose, reqClose)
	if rrClose.Code != http.StatusOK {
		t.Fatalf("close session status = %d, want 200 OK (non-blocking) body=%s", rrClose.Code, rrClose.Body.String())
	}
	var closedSess PosSessionDTO
	if err := json.Unmarshal(rrClose.Body.Bytes(), &closedSess); err != nil {
		t.Fatal(err)
	}
	if len(closedSess.AnomalyFlags) == 0 || closedSess.AnomalyFlags[0] != PosAnomalyReviewFlag || closedSess.AnomalyReview == nil || closedSess.AnomalyReview.AutoRejected {
		t.Fatalf("expected POS_ANOMALY_REVIEW on close with variance, got %+v", closedSess)
	}

	// Control Tower Pulse surfaces POS_ANOMALY_REVIEW cleanly.
	reqPulse := httptest.NewRequest(http.MethodGet, "/v1/retailer/control-tower/pulse", nil)
	reqPulse = reqPulse.WithContext(auth.WithClaims(reqPulse.Context(), ownerClaims))
	rrPulse := httptest.NewRecorder()
	r.ServeHTTP(rrPulse, reqPulse)
	if rrPulse.Code != http.StatusOK {
		t.Fatalf("pulse status = %d, want 200 (%s)", rrPulse.Code, rrPulse.Body.String())
	}
	var pulse ControlTowerPulse
	if err := json.Unmarshal(rrPulse.Body.Bytes(), &pulse); err != nil {
		t.Fatal(err)
	}
	if !pulse.PosAnomalyReview || pulse.PosAnomalyCount < 1 || len(pulse.PosAnomalies) == 0 || pulse.PosAnomalies[0].Flag != PosAnomalyReviewFlag {
		t.Fatalf("pulse did not surface POS_ANOMALY_REVIEW: %+v", pulse)
	}
	if pulse.PosAnomalies[0].AutoRejected {
		t.Fatalf("POS_ANOMALY_REVIEW must never auto-reject: %+v", pulse.PosAnomalies[0])
	}
}
