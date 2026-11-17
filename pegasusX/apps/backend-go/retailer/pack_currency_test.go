package retailer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandlePosSessionOpen_EmptyCurrencyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{})
	regID := mustCreateRegister(t, svc, "ret-cur")
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/open",
		bytes.NewBufferString(`{"register_id":"`+regID+`","opening_float_minor":1000}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "c1", Role: auth.RoleRetailer, RetailerOrgID: "ret-cur", RetailerRole: "CASHIER",
		RetailerUserID: "c1", MarketCode: "UZ",
	}))
	rr := httptest.NewRecorder()
	svc.HandlePosSessionOpen(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var sess PosSessionDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	if sess.Currency != "UZS" {
		t.Fatalf("currency=%q want pack UZS", sess.Currency)
	}
}

func TestHandlePosSessionOpen_USDOnUZRejected(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{})
	regID := mustCreateRegister(t, svc, "ret-usd")
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/open",
		bytes.NewBufferString(`{"register_id":"`+regID+`","opening_float_minor":1000,"currency":"USD"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "c1", Role: auth.RoleRetailer, RetailerOrgID: "ret-usd", RetailerRole: "CASHIER",
		RetailerUserID: "c1", MarketCode: "UZ",
	}))
	rr := httptest.NewRecorder()
	svc.HandlePosSessionOpen(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["error"] != "pack_currency_mismatch" {
		t.Fatalf("error=%v", body)
	}
}

func TestHandlePosSessionOpen_PlannedFailsClosed(t *testing.T) {
	svc := NewService(ServiceConfig{})
	regID := mustCreateRegister(t, svc, "ret-ca")
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/open",
		bytes.NewBufferString(`{"register_id":"`+regID+`","opening_float_minor":1000}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "c1", Role: auth.RoleRetailer, RetailerOrgID: "ret-ca", RetailerRole: "CASHIER",
		RetailerUserID: "c1", MarketCode: "CA",
	}))
	rr := httptest.NewRecorder()
	svc.HandlePosSessionOpen(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleShiftOpen_EmptyCurrencyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/shifts",
		bytes.NewBufferString(`{"opening_float_minor":500}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "c1", Role: auth.RoleRetailer, RetailerOrgID: "ret-sh-cur", RetailerRole: "CASHIER",
		RetailerUserID: "c1", MarketCode: "UZ",
	}))
	rr := httptest.NewRecorder()
	svc.HandleShifts(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var shift ShiftDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &shift); err != nil {
		t.Fatal(err)
	}
	if shift.Currency != "UZS" {
		t.Fatalf("currency=%q want pack UZS", shift.Currency)
	}
}

func mustCreateRegister(t *testing.T, svc *Service, orgID string) string {
	t.Helper()
	if _, err := svc.EnsurePrimaryLocation(t.Context(), orgID); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/registers", bytes.NewBufferString(`{"label":"Till"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: orgID, RetailerRole: "OWNER",
	}))
	rr := httptest.NewRecorder()
	svc.HandleRegisters(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rr.Code, rr.Body.String())
	}
	var reg RegisterDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &reg); err != nil || reg.RegisterID == "" {
		t.Fatalf("register parse: %v id=%q", err, reg.RegisterID)
	}
	return reg.RegisterID
}
