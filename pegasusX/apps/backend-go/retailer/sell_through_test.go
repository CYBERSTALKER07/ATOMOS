package retailer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestSellThroughFromPOSSaleAndVoid(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "st-" + string(rune('A'+n%26))
		},
	})
	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-st")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.applyDelta(t.Context(), "ret-st", primary.LocationID, BinFloor, "SKU-TEA", 100, MoveReceive, "TEST", "o1", "u", "")

	owner := auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-st",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}
	// register + open
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/registers",
		bytes.NewBufferString(`{"location_id":"`+primary.LocationID+`","label":"Till ST"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleRegisters(rr, req)
	var reg RegisterDTO
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)

	reqOpen := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sessions/open",
		bytes.NewBufferString(`{"register_id":"`+reg.RegisterID+`","opening_float_minor":1000,"currency":"UZS"}`))
	reqOpen = reqOpen.WithContext(auth.WithClaims(reqOpen.Context(), owner))
	rrOpen := httptest.NewRecorder()
	svc.HandlePosSessionOpen(rrOpen, reqOpen)
	var sess PosSessionDTO
	_ = json.Unmarshal(rrOpen.Body.Bytes(), &sess)

	saleBody, _ := json.Marshal(map[string]any{
		"session_id": sess.SessionID,
		"lines": []map[string]any{
			{"sku": "SKU-TEA", "name": "Tea", "qty": 5, "unit_price_minor": 1000},
		},
		"tenders": []map[string]any{{"method": "CASH", "amount_minor": 5000}},
	})
	reqSale := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(saleBody))
	reqSale = reqSale.WithContext(auth.WithClaims(reqSale.Context(), owner))
	rrSale := httptest.NewRecorder()
	svc.HandlePosSale(rrSale, reqSale)
	if rrSale.Code != http.StatusCreated {
		t.Fatalf("sale status=%d body=%s", rrSale.Code, rrSale.Body.String())
	}

	day := time.Now().UTC().Format("2006-01-02")
	items, err := svc.ListSellThrough(t.Context(), "ret-st", primary.LocationID, day, day, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].QtySold != 5 || items[0].NetSold != 5 {
		t.Fatalf("sell-through after sale=%+v", items)
	}
	if items[0].Source != "STORE_POS" {
		t.Fatalf("source=%s", items[0].Source)
	}
	factor := svc.GetSellThroughFactor("ret-st", "SKU-TEA", day)
	if factor != 5 {
		t.Fatalf("SELL_THROUGH factor=%v want 5", factor)
	}

	// Insights API
	reqIns := httptest.NewRequest(http.MethodGet, "/v1/retailer/insights/sell-through?location_id="+primary.LocationID, nil)
	reqIns = reqIns.WithContext(auth.WithClaims(reqIns.Context(), owner))
	rrIns := httptest.NewRecorder()
	svc.HandleSellThroughInsights(rrIns, reqIns)
	if rrIns.Code != http.StatusOK {
		t.Fatalf("insights status=%d body=%s", rrIns.Code, rrIns.Body.String())
	}
	var payload struct {
		Items []SellThroughDayDTO `json:"items"`
	}
	_ = json.Unmarshal(rrIns.Body.Bytes(), &payload)
	if len(payload.Items) < 1 {
		t.Fatalf("insights empty: %s", rrIns.Body.String())
	}

	// Void reverses net
	var sale PosSaleDTO
	_ = json.Unmarshal(rrSale.Body.Bytes(), &sale)
	reqVoid := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales/"+sale.SaleID+"/void",
		bytes.NewBufferString(`{"reason":"test"}`))
	rctx := chiRoute(sale.SaleID)
	reqVoid = reqVoid.WithContext(auth.WithClaims(reqVoid.Context(), owner))
	reqVoid = reqVoid.WithContext(contextWithChi(reqVoid, rctx))
	rrVoid := httptest.NewRecorder()
	svc.HandlePosSaleVoid(rrVoid, reqVoid)
	if rrVoid.Code != http.StatusOK {
		t.Fatalf("void status=%d body=%s", rrVoid.Code, rrVoid.Body.String())
	}

	items, _ = svc.ListSellThrough(t.Context(), "ret-st", primary.LocationID, day, day, 50)
	if len(items) != 1 || items[0].QtyVoided != 5 || items[0].NetSold != 0 {
		t.Fatalf("after void=%+v", items)
	}
	factor = svc.GetSellThroughFactor("ret-st", "SKU-TEA", day)
	if factor != 0 {
		t.Fatalf("factor after void=%v want 0", factor)
	}

	// B4: DEMAND_SIGNAL emitted for sale (+5) and void (−5); no local: noise
	sigs := svc.EmittedDemandSignals()
	if len(sigs) < 2 {
		t.Fatalf("want ≥2 DEMAND_SIGNAL emits, got %d", len(sigs))
	}
	var saleSig, voidSig *events.DemandSignalEvent
	for i := range sigs {
		if sigs[i].SKU == "SKU-TEA" && sigs[i].Kind == "sale" && saleSig == nil {
			saleSig = &sigs[i]
		}
		if sigs[i].SKU == "SKU-TEA" && sigs[i].Kind == "void" {
			voidSig = &sigs[i]
		}
	}
	if saleSig == nil || saleSig.QtyDelta != 5 || saleSig.Source != "STORE_POS" || saleSig.NetSold != 5 {
		t.Fatalf("sale signal=%+v", saleSig)
	}
	if voidSig == nil || voidSig.QtyDelta != -5 || voidSig.NetSold != 0 {
		t.Fatalf("void signal=%+v", voidSig)
	}
}

func TestDemandSignalSkipsLocalSKU(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now:   func() time.Time { return time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC) },
		NewID: func() string { return "ds-1" },
	})
	loc, err := svc.EnsurePrimaryLocation(t.Context(), "ret-ds-local")
	if err != nil {
		t.Fatal(err)
	}
	// Seed local catalog sale path through recordSellThroughSale directly
	svc.recordSellThroughSale(t.Context(), "ret-ds-local", loc.LocationID, []PosSaleLine{
		{Sku: "local:house-tea", Name: "House", Qty: 3, UnitPriceMinor: 100},
		{Sku: "SKU-PEG", Name: "Peg", Qty: 2, UnitPriceMinor: 200},
	})
	sigs := svc.EmittedDemandSignals()
	for _, sig := range sigs {
		if IsLocalSKU(sig.SKU) {
			t.Fatalf("local SKU must not emit DEMAND_SIGNAL: %+v", sig)
		}
	}
	if len(sigs) != 1 || sigs[0].SKU != "SKU-PEG" || sigs[0].QtyDelta != 2 {
		t.Fatalf("sigs=%+v", sigs)
	}
}

// chiRoute builds a chi context with saleID param (shared helper name unique to this file).
func chiRoute(saleID string) *chi.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("saleID", saleID)
	return rctx
}
