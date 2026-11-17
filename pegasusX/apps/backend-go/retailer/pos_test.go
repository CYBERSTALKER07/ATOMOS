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

	// Close session then historical offline sync MUST succeed (D-4), while online sales on closed session remain blocked.
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

	// Online sale on closed session -> 409 session_not_open
	onlineClosedBody, _ := json.Marshal(map[string]any{
		"session_id": sess.SessionID,
		"origin":     "online",
		"lines":      []map[string]any{{"sku": "SKU-BREAD", "qty": 1, "unit_price_minor": 10000}},
		"tenders":    []map[string]any{{"method": "CASH", "amount_minor": 10000}},
	})
	reqOnlineClosed := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(onlineClosedBody))
	reqOnlineClosed = reqOnlineClosed.WithContext(auth.WithClaims(reqOnlineClosed.Context(), cashier))
	rrOnlineClosed := httptest.NewRecorder()
	svc.HandlePosSale(rrOnlineClosed, reqOnlineClosed)
	if rrOnlineClosed.Code != http.StatusConflict {
		t.Fatalf("online closed session status=%d want 409 body=%s", rrOnlineClosed.Code, rrOnlineClosed.Body.String())
	}
	var onlineErr map[string]string
	if err := json.Unmarshal(rrOnlineClosed.Body.Bytes(), &onlineErr); err != nil {
		t.Fatal(err)
	}
	if onlineErr["error"] != "session_not_open" {
		t.Fatalf("error=%v want session_not_open", onlineErr)
	}

	// Offline historical CASH sync on closed session -> 201 Created + stock decremented
	lateBody, _ := json.Marshal(map[string]any{
		"session_id":        sess.SessionID,
		"origin":            "offline",
		"client_sale_id":    "client-late-1",
		"client_created_at": "2026-08-02T12:05:00Z",
		"lines":             []map[string]any{{"sku": "SKU-BREAD", "qty": 1, "unit_price_minor": 10000}},
		"tenders":           []map[string]any{{"method": "CASH", "amount_minor": 10000}},
	})
	reqLate := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(lateBody))
	reqLate = reqLate.WithContext(auth.WithClaims(reqLate.Context(), cashier))
	rrLate := httptest.NewRecorder()
	svc.HandlePosSale(rrLate, reqLate)
	if rrLate.Code != http.StatusCreated {
		t.Fatalf("late offline sync status=%d want 201 body=%s", rrLate.Code, rrLate.Body.String())
	}
	var lateSale PosSaleDTO
	if err := json.Unmarshal(rrLate.Body.Bytes(), &lateSale); err != nil {
		t.Fatal(err)
	}
	onHand, _ = svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-BREAD")
	if onHand != 48 {
		t.Fatalf("stock after closed-session offline sync=%d want 48", onHand)
	}

	// Replaying client_sale_id on closed session stays idempotent (no double decrement)
	reqLateReplay := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(lateBody))
	reqLateReplay = reqLateReplay.WithContext(auth.WithClaims(reqLateReplay.Context(), cashier))
	rrLateReplay := httptest.NewRecorder()
	svc.HandlePosSale(rrLateReplay, reqLateReplay)
	if rrLateReplay.Code != http.StatusCreated {
		t.Fatalf("late offline replay status=%d want 201 body=%s", rrLateReplay.Code, rrLateReplay.Body.String())
	}
	var lateReplay PosSaleDTO
	if err := json.Unmarshal(rrLateReplay.Body.Bytes(), &lateReplay); err != nil {
		t.Fatal(err)
	}
	if lateReplay.SaleID != lateSale.SaleID {
		t.Fatalf("replay sale_id=%s want %s", lateReplay.SaleID, lateSale.SaleID)
	}
	onHand, _ = svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-BREAD")
	if onHand != 48 {
		t.Fatalf("stock after closed-session offline replay=%d want 48", onHand)
	}

	// Cross-tenant offline sync against ret-off's closed session MUST be rejected
	otherCashier := auth.Claims{
		Subject: "other-cashier", Role: auth.RoleRetailer, RetailerOrgID: "ret-other",
		RetailerRole: "CASHIER", RetailerUserID: "other-cashier",
	}
	crossBody, _ := json.Marshal(map[string]any{
		"session_id":     sess.SessionID,
		"origin":         "offline",
		"client_sale_id": "cross-tenant-1",
		"lines":          []map[string]any{{"sku": "SKU-BREAD", "qty": 1, "unit_price_minor": 10000}},
		"tenders":        []map[string]any{{"method": "CASH", "amount_minor": 10000}},
	})
	reqCross := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(crossBody))
	reqCross = reqCross.WithContext(auth.WithClaims(reqCross.Context(), otherCashier))
	rrCross := httptest.NewRecorder()
	svc.HandlePosSale(rrCross, reqCross)
	if rrCross.Code != http.StatusConflict {
		t.Fatalf("cross-tenant offline sync status=%d want 409 body=%s", rrCross.Code, rrCross.Body.String())
	}
	onHand, _ = svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-BREAD")
	if onHand != 48 {
		t.Fatalf("stock after cross-tenant rejection=%d want 48", onHand)
	}
}

func TestPOSMultiLineInsufficientStockAtomicRollback(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-atomic")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.applyDelta(t.Context(), "ret-atomic", primary.LocationID, BinFloor, "SKU-A", 10, MoveReceive, "TEST", "o1", "u", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.applyDelta(t.Context(), "ret-atomic", primary.LocationID, BinFloor, "SKU-B", 2, MoveReceive, "TEST", "o2", "u", ""); err != nil {
		t.Fatal(err)
	}

	owner := auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-atomic",
		RetailerRole: "OWNER", RetailerUserID: "owner",
	}
	reqReg := httptest.NewRequest(http.MethodPost, "/v1/retailer/registers",
		bytes.NewBufferString(`{"location_id":"`+primary.LocationID+`","label":"Atomic Till"}`))
	reqReg = reqReg.WithContext(auth.WithClaims(reqReg.Context(), owner))
	rrReg := httptest.NewRecorder()
	svc.HandleRegisters(rrReg, reqReg)
	if rrReg.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rrReg.Code, rrReg.Body.String())
	}
	var reg RegisterDTO
	if err := json.Unmarshal(rrReg.Body.Bytes(), &reg); err != nil {
		t.Fatal(err)
	}

	reqOpen := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/open",
		bytes.NewBufferString(`{"register_id":"`+reg.RegisterID+`","opening_float_minor":0,"currency":"UZS"}`))
	reqOpen = reqOpen.WithContext(auth.WithClaims(reqOpen.Context(), owner))
	rrOpen := httptest.NewRecorder()
	svc.HandlePosSessionOpen(rrOpen, reqOpen)
	if rrOpen.Code != http.StatusCreated && rrOpen.Code != http.StatusOK {
		t.Fatalf("open status=%d body=%s", rrOpen.Code, rrOpen.Body.String())
	}
	var sess PosSessionDTO
	if err := json.Unmarshal(rrOpen.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}

	// Line 1 (SKU-A qty 4) has enough stock, line 2 (SKU-B qty 5) fails — SKU-A must NOT leak.
	badBody, _ := json.Marshal(map[string]any{
		"session_id": sess.SessionID,
		"stock_bin":  "FLOOR",
		"lines": []map[string]any{
			{"sku": "SKU-A", "name": "Item A", "qty": 4, "unit_price_minor": 1000},
			{"sku": "SKU-B", "name": "Item B", "qty": 5, "unit_price_minor": 2000},
		},
		"tenders": []map[string]any{{"method": "CASH", "amount_minor": 14000}},
	})
	reqFail := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(badBody))
	reqFail = reqFail.WithContext(auth.WithClaims(reqFail.Context(), owner))
	rrFail := httptest.NewRecorder()
	svc.HandlePosSale(rrFail, reqFail)
	if rrFail.Code != http.StatusConflict {
		t.Fatalf("multi-line failure status=%d want 409 body=%s", rrFail.Code, rrFail.Body.String())
	}
	gotA, _ := svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-A")
	gotB, _ := svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-B")
	if gotA != 10 || gotB != 2 {
		t.Fatalf("partial stock mutation leaked on rollback: SKU-A=%d (want 10) SKU-B=%d (want 2)", gotA, gotB)
	}

	// Duplicate SKU lines totaling > onHand must also roll back atomically without partial decrement.
	dupBody, _ := json.Marshal(map[string]any{
		"session_id": sess.SessionID,
		"stock_bin":  "FLOOR",
		"lines": []map[string]any{
			{"sku": "SKU-A", "name": "Item A", "qty": 6, "unit_price_minor": 1000},
			{"sku": "SKU-A", "name": "Item A", "qty": 6, "unit_price_minor": 1000},
		},
		"tenders": []map[string]any{{"method": "CASH", "amount_minor": 12000}},
	})
	reqDup := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(dupBody))
	reqDup = reqDup.WithContext(auth.WithClaims(reqDup.Context(), owner))
	rrDup := httptest.NewRecorder()
	svc.HandlePosSale(rrDup, reqDup)
	if rrDup.Code != http.StatusConflict {
		t.Fatalf("duplicate-sku oversell status=%d want 409 body=%s", rrDup.Code, rrDup.Body.String())
	}
	gotA, _ = svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-A")
	if gotA != 10 {
		t.Fatalf("duplicate-sku oversell mutated stock: SKU-A=%d want 10", gotA)
	}

	// Valid multi-line sale + idempotent double-void must not double-restock.
	okBody, _ := json.Marshal(map[string]any{
		"session_id": sess.SessionID,
		"stock_bin":  "FLOOR",
		"lines": []map[string]any{
			{"sku": "SKU-A", "name": "Item A", "qty": 3, "unit_price_minor": 1000},
			{"sku": "SKU-B", "name": "Item B", "qty": 2, "unit_price_minor": 2000},
		},
		"tenders": []map[string]any{{"method": "CASH", "amount_minor": 7000}},
	})
	reqOK := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(okBody))
	reqOK = reqOK.WithContext(auth.WithClaims(reqOK.Context(), owner))
	rrOK := httptest.NewRecorder()
	svc.HandlePosSale(rrOK, reqOK)
	if rrOK.Code != http.StatusCreated {
		t.Fatalf("valid sale status=%d want 201 body=%s", rrOK.Code, rrOK.Body.String())
	}
	var committed PosSaleDTO
	if err := json.Unmarshal(rrOK.Body.Bytes(), &committed); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		reqVoid := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales/"+committed.SaleID+"/void",
			bytes.NewBufferString(`{"reason":"duplicate check"}`))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("saleID", committed.SaleID)
		reqVoid = reqVoid.WithContext(auth.WithClaims(reqVoid.Context(), owner))
		reqVoid = reqVoid.WithContext(contextWithChi(reqVoid, rctx))
		rrVoid := httptest.NewRecorder()
		svc.HandlePosSaleVoid(rrVoid, reqVoid)
		if rrVoid.Code != http.StatusOK {
			t.Fatalf("void attempt %d status=%d body=%s", attempt, rrVoid.Code, rrVoid.Body.String())
		}
	}
	gotA, _ = svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-A")
	gotB, _ = svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-B")
	if gotA != 10 || gotB != 2 {
		t.Fatalf("double void corrupted stock: SKU-A=%d (want 10) SKU-B=%d (want 2)", gotA, gotB)
	}
}
