package retailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestPOSRegisterSessionSaleVoid(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "pos-" + string(rune('A'+n%26)) + string(rune('0'+n/26))
		},
	})
	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-1")
	if err != nil {
		t.Fatal(err)
	}
	// Stock FLOOR
	_ = svc.applyDelta(t.Context(), "ret-1", primary.LocationID, BinFloor, "SKU-MILK", 20, MoveReceive, "TEST", "o1", "u", "")

	// Create register
	body := `{"location_id":"` + primary.LocationID + `","label":"Till 1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/registers", bytes.NewBufferString(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerRole: "OWNER",
	}))
	rr := httptest.NewRecorder()
	svc.HandleRegisters(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rr.Code, rr.Body.String())
	}
	var reg RegisterDTO
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)
	if reg.RegisterID == "" {
		t.Fatal("missing register id")
	}

	// Open session
	openBody := `{"register_id":"` + reg.RegisterID + `","opening_float_minor":10000,"currency":"UZS"}`
	reqOpen := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/open", bytes.NewBufferString(openBody))
	reqOpen = reqOpen.WithContext(auth.WithClaims(reqOpen.Context(), auth.Claims{
		Subject: "cashier", Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerRole: "CASHIER", RetailerUserID: "cashier",
	}))
	rrOpen := httptest.NewRecorder()
	svc.HandlePosSessionOpen(rrOpen, reqOpen)
	if rrOpen.Code != http.StatusCreated && rrOpen.Code != http.StatusOK {
		t.Fatalf("open status=%d body=%s", rrOpen.Code, rrOpen.Body.String())
	}
	var sess PosSessionDTO
	_ = json.Unmarshal(rrOpen.Body.Bytes(), &sess)

	// Sale
	saleBody, _ := json.Marshal(map[string]any{
		"session_id": sess.SessionID,
		"stock_bin":  "FLOOR",
		"lines": []map[string]any{
			{"sku": "SKU-MILK", "name": "Milk", "qty": 2, "unit_price_minor": 15000},
		},
		"tenders": []map[string]any{
			{"method": "CASH", "amount_minor": 30000},
		},
	})
	reqSale := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(saleBody))
	reqSale = reqSale.WithContext(auth.WithClaims(reqSale.Context(), auth.Claims{
		Subject: "cashier", Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerRole: "CASHIER", RetailerUserID: "cashier",
	}))
	rrSale := httptest.NewRecorder()
	svc.HandlePosSale(rrSale, reqSale)
	if rrSale.Code != http.StatusCreated {
		t.Fatalf("sale status=%d body=%s", rrSale.Code, rrSale.Body.String())
	}
	var sale PosSaleDTO
	_ = json.Unmarshal(rrSale.Body.Bytes(), &sale)
	if sale.TotalMinor != 30000 || sale.ReceiptNumber == "" {
		t.Fatalf("sale: %+v", sale)
	}
	onHand, _ := svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-MILK")
	if onHand != 18 {
		t.Fatalf("stock after sale=%d want 18", onHand)
	}

	// Void by manager
	reqVoid := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales/"+sale.SaleID+"/void",
		bytes.NewBufferString(`{"reason":"customer return"}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("saleID", sale.SaleID)
	reqVoid = reqVoid.WithContext(auth.WithClaims(reqVoid.Context(), auth.Claims{
		Subject: "mgr", Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerRole: "MANAGER", RetailerUserID: "mgr",
	}))
	reqVoid = reqVoid.WithContext(contextWithChi(reqVoid, rctx))
	rrVoid := httptest.NewRecorder()
	svc.HandlePosSaleVoid(rrVoid, reqVoid)
	if rrVoid.Code != http.StatusOK {
		t.Fatalf("void status=%d body=%s", rrVoid.Code, rrVoid.Body.String())
	}
	onHand, _ = svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-MILK")
	if onHand != 20 {
		t.Fatalf("stock after void=%d want 20", onHand)
	}

	// Close session
	reqClose := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/"+sess.SessionID+"/close",
		bytes.NewBufferString(`{"closing_cash_minor":10000}`))
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("sessionID", sess.SessionID)
	reqClose = reqClose.WithContext(auth.WithClaims(reqClose.Context(), auth.Claims{
		Subject: "cashier", Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerRole: "CASHIER", RetailerUserID: "cashier",
	}))
	reqClose = reqClose.WithContext(contextWithChi(reqClose, rctx2))
	rrClose := httptest.NewRecorder()
	svc.HandlePosSessionClose(rrClose, reqClose)
	if rrClose.Code != http.StatusOK {
		t.Fatalf("close status=%d body=%s", rrClose.Code, rrClose.Body.String())
	}
	var closed PosSessionDTO
	_ = json.Unmarshal(rrClose.Body.Bytes(), &closed)
	if closed.Status != PosSessionClosed {
		t.Fatalf("expected closed, got %s", closed.Status)
	}
	// Cash sales voided → expected = opening float only
	if closed.ExpectedCashMinor == nil || *closed.ExpectedCashMinor != 10000 {
		t.Fatalf("expected cash=%v", closed.ExpectedCashMinor)
	}
}

func contextWithChi(r *http.Request, rctx *chi.Context) context.Context {
	return context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
}

func TestPOSOfflineCashAndIdempotency(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return fmt.Sprintf("off-%d", n)
		},
	})
	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-off")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.applyDelta(t.Context(), "ret-off", primary.LocationID, BinFloor, "SKU-BREAD", 50, MoveReceive, "TEST", "o1", "u", "")

	owner := auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-off",
		RetailerRole: "OWNER", RetailerUserID: "owner",
	}
	cashier := auth.Claims{
		Subject: "cashier", Role: auth.RoleRetailer, RetailerOrgID: "ret-off",
		RetailerRole: "CASHIER", RetailerUserID: "cashier",
	}
	// register + open
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/registers",
		bytes.NewBufferString(`{"location_id":"`+primary.LocationID+`","label":"Till Offline"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleRegisters(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rr.Code, rr.Body.String())
	}
	var reg RegisterDTO
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)
	if reg.RegisterID == "" {
		t.Fatal("missing register")
	}

	reqOpen := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/open",
		bytes.NewBufferString(`{"register_id":"`+reg.RegisterID+`","opening_float_minor":5000,"currency":"UZS"}`))
	reqOpen = reqOpen.WithContext(auth.WithClaims(reqOpen.Context(), cashier))
	rrOpen := httptest.NewRecorder()
	svc.HandlePosSessionOpen(rrOpen, reqOpen)
	if rrOpen.Code != http.StatusCreated && rrOpen.Code != http.StatusOK {
		t.Fatalf("open status=%d body=%s", rrOpen.Code, rrOpen.Body.String())
	}
	var sess PosSessionDTO
	_ = json.Unmarshal(rrOpen.Body.Bytes(), &sess)
	if sess.SessionID == "" {
		t.Fatalf("missing session body=%s", rrOpen.Body.String())
	}

	clientID := "client-sale-uuid-1"
	saleBody, _ := json.Marshal(map[string]any{
		"session_id":        sess.SessionID,
		"stock_bin":         "FLOOR",
		"client_sale_id":    clientID,
		"client_created_at": "2026-08-02T12:00:00Z",
		"origin":            "offline",
		"lines": []map[string]any{
			{"sku": "SKU-BREAD", "name": "Bread", "qty": 1, "unit_price_minor": 10000},
		},
		"tenders": []map[string]any{
			{"method": "CASH", "amount_minor": 10000},
		},
	})

	// First offline sale
	reqSale := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(saleBody))
	reqSale.Header.Set("Idempotency-Key", "pos-sale:"+clientID)
	reqSale = reqSale.WithContext(auth.WithClaims(reqSale.Context(), cashier))
	rrSale := httptest.NewRecorder()
	svc.HandlePosSale(rrSale, reqSale)
	if rrSale.Code != http.StatusCreated {
		t.Fatalf("offline sale status=%d body=%s saleBody=%s", rrSale.Code, rrSale.Body.String(), string(saleBody))
	}
	var sale1 PosSaleDTO
	_ = json.Unmarshal(rrSale.Body.Bytes(), &sale1)
	if sale1.Origin != "offline" || sale1.ClientSaleID != clientID {
		t.Fatalf("sale1=%+v", sale1)
	}
	onHand, _ := svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-BREAD")
	if onHand != 49 {
		t.Fatalf("stock=%d want 49", onHand)
	}

	// Replay same client_sale_id without Idempotency-Key store (new request) — must not double-stock
	reqSale2 := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(saleBody))
	reqSale2.Header.Set("Idempotency-Key", "pos-sale:"+clientID+"-replay")
	reqSale2 = reqSale2.WithContext(auth.WithClaims(reqSale2.Context(), cashier))
	rrSale2 := httptest.NewRecorder()
	svc.HandlePosSale(rrSale2, reqSale2)
	if rrSale2.Code != http.StatusCreated {
		t.Fatalf("replay status=%d body=%s", rrSale2.Code, rrSale2.Body.String())
	}
	var sale2 PosSaleDTO
	_ = json.Unmarshal(rrSale2.Body.Bytes(), &sale2)
	if sale2.SaleID != sale1.SaleID {
		t.Fatalf("want same sale_id, got %s vs %s", sale2.SaleID, sale1.SaleID)
	}
	onHand, _ = svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-BREAD")
	if onHand != 49 {
		t.Fatalf("stock after replay=%d want 49", onHand)
	}

	// Offline + CARD forbidden
	cardBody, _ := json.Marshal(map[string]any{
		"session_id":     sess.SessionID,
		"origin":         "offline",
		"client_sale_id": "client-card-1",
		"lines":          []map[string]any{{"sku": "SKU-BREAD", "qty": 1, "unit_price_minor": 10000}},
		"tenders":        []map[string]any{{"method": "CARD", "amount_minor": 10000}},
	})
	reqCard := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(cardBody))
	reqCard = reqCard.WithContext(auth.WithClaims(reqCard.Context(), cashier))
	rrCard := httptest.NewRecorder()
	svc.HandlePosSale(rrCard, reqCard)
	if rrCard.Code != http.StatusUnprocessableEntity {
		t.Fatalf("card offline status=%d want 422 body=%s", rrCard.Code, rrCard.Body.String())
	}

	// Close session then offline sync → session_closed_for_offline_sync
	reqClose := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/"+sess.SessionID+"/close",
		bytes.NewBufferString(`{"closing_cash_minor":15000}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("sessionID", sess.SessionID)
	reqClose = reqClose.WithContext(auth.WithClaims(reqClose.Context(), cashier))
	reqClose = reqClose.WithContext(contextWithChi(reqClose, rctx))
	rrClose := httptest.NewRecorder()
	svc.HandlePosSessionClose(rrClose, reqClose)
	if rrClose.Code != http.StatusOK {
		t.Fatalf("close status=%d body=%s", rrClose.Code, rrClose.Body.String())
	}

	lateBody, _ := json.Marshal(map[string]any{
		"session_id":     sess.SessionID,
		"origin":         "offline",
		"client_sale_id": "client-late-1",
		"lines":          []map[string]any{{"sku": "SKU-BREAD", "qty": 1, "unit_price_minor": 10000}},
		"tenders":        []map[string]any{{"method": "CASH", "amount_minor": 10000}},
	})
	reqLate := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(lateBody))
	reqLate = reqLate.WithContext(auth.WithClaims(reqLate.Context(), cashier))
	rrLate := httptest.NewRecorder()
	svc.HandlePosSale(rrLate, reqLate)
	if rrLate.Code != http.StatusConflict {
		t.Fatalf("late sync status=%d want 409 body=%s", rrLate.Code, rrLate.Body.String())
	}
	var lateErr map[string]string
	_ = json.Unmarshal(rrLate.Body.Bytes(), &lateErr)
	if lateErr["error"] != "session_closed_for_offline_sync" {
		t.Fatalf("error=%v", lateErr)
	}
}
