package retailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHQSalesDaily_SaleAndVoid_Memory(t *testing.T) {
	t.Parallel()
	n := 0
	fixed := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		Now: func() time.Time { return fixed },
		NewID: func() string {
			n++
			return fmt.Sprintf("hq-%d", n)
		},
	})
	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-hq")
	if err != nil {
		t.Fatal(err)
	}
	// Stock for sale
	_ = svc.applyDelta(t.Context(), "ret-hq", primary.LocationID, BinFloor, "SKU-HQ", 100, MoveReceive, "TEST", "o1", "owner", "")
	_ = svc.applyDelta(t.Context(), "ret-hq", primary.LocationID, BinFloor, "local:BREAD", 50, MoveReceive, "REF", "o1", "owner", "")

	// Open register + session
	regID := "reg-hq-1"
	_ = svc.saveRegister(t.Context(), RegisterDTO{
		RegisterID: regID, RetailerID: "ret-hq", LocationID: primary.LocationID,
		Label: "HQ Till", Status: RegisterStatusActive,
	})
	sessID := "sess-hq-1"
	_ = svc.savePosSession(t.Context(), PosSessionDTO{
		SessionID: sessID, RegisterID: regID, LocationID: primary.LocationID,
		RetailerID: "ret-hq", OpenedByUserID: "owner", Status: PosSessionOpen,
		OpeningFloatMinor: 0, Currency: "UZS",
	})

	claims := auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-hq",
		RetailerRole: "OWNER", RetailerUserID: "owner",
	}

	// Complete sale: 3× SKU-HQ @ 1000 + 2× local:BREAD @ 500
	saleBody := map[string]any{
		"session_id": sessID,
		"stock_bin":  BinFloor,
		"currency":   "UZS",
		"lines": []map[string]any{
			{"sku": "SKU-HQ", "name": "Tea", "qty": 3, "unit_price_minor": 1000},
			{"sku": "local:BREAD", "name": "Bread", "qty": 2, "unit_price_minor": 500},
		},
		"tenders": []map[string]any{{"method": "CASH", "amount_minor": 4000}},
	}
	raw, _ := json.Marshal(saleBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales", bytes.NewReader(raw))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	rr := httptest.NewRecorder()
	svc.HandlePosSale(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("sale status=%d body=%s", rr.Code, rr.Body.String())
	}
	var sale PosSaleDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &sale); err != nil {
		t.Fatal(err)
	}

	day := "2026-08-12"
	tea, ok := svc.GetHqSalesDay("ret-hq", primary.LocationID, day, "SKU-HQ")
	if !ok || tea.QtySold != 3 || tea.GrossMinor != 3000 || tea.NetMinor != 3000 {
		t.Fatalf("tea hq=%+v ok=%v", tea, ok)
	}
	// local: must be included in HQ sales
	bread, ok := svc.GetHqSalesDay("ret-hq", primary.LocationID, day, "local:BREAD")
	if !ok || bread.QtySold != 2 || bread.NetMinor != 1000 {
		t.Fatalf("local bread hq=%+v ok=%v", bread, ok)
	}

	// Void reverses net + qty voided; gross stays
	reqV := httptest.NewRequest(http.MethodPost, "/v1/retailer/pos/sales/"+sale.SaleID+"/void",
		bytes.NewBufferString(`{"reason":"test"}`))
	reqV = reqV.WithContext(auth.WithClaims(reqV.Context(), claims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("saleID", sale.SaleID)
	reqV = reqV.WithContext(contextWithChi(reqV, rctx))
	rrV := httptest.NewRecorder()
	svc.HandlePosSaleVoid(rrV, reqV)
	if rrV.Code != http.StatusOK {
		t.Fatalf("void status=%d body=%s", rrV.Code, rrV.Body.String())
	}

	tea2, _ := svc.GetHqSalesDay("ret-hq", primary.LocationID, day, "SKU-HQ")
	if tea2.QtySold != 3 || tea2.QtyVoided != 3 || tea2.GrossMinor != 3000 || tea2.NetMinor != 0 {
		t.Fatalf("after void tea=%+v", tea2)
	}
	bread2, _ := svc.GetHqSalesDay("ret-hq", primary.LocationID, day, "local:BREAD")
	if bread2.QtyVoided != 2 || bread2.NetMinor != 0 {
		t.Fatalf("after void bread=%+v", bread2)
	}

	// Property: sum locations = org net
	byLoc, err := svc.ListHqSalesByLocation(t.Context(), "ret-hq", day)
	if err != nil {
		t.Fatal(err)
	}
	var sum int64
	for _, v := range byLoc {
		sum += v
	}
	if sum != 0 {
		t.Fatalf("sum locations net=%d want 0 after full void", sum)
	}
}

func TestHQSalesDaily_MultiLocation_SumEqualsOrg(t *testing.T) {
	t.Parallel()
	fixed := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		Now:   func() time.Time { return fixed },
		NewID: func() string { return "x" },
	})
	// Two locations, direct memory HQ deltas (simulate sales)
	svc.mu.Lock()
	svc.applyHqSalesDeltaMemory(hqSalesKey{
		RetailerID: "ret-ml", LocationID: "loc-a", Day: "2026-08-12", SkuID: "S1",
	}, 1, 0, 100, 100, "UZS")
	svc.applyHqSalesDeltaMemory(hqSalesKey{
		RetailerID: "ret-ml", LocationID: "loc-b", Day: "2026-08-12", SkuID: "S1",
	}, 2, 0, 200, 200, "UZS")
	svc.mu.Unlock()

	byLoc, err := svc.ListHqSalesByLocation(t.Context(), "ret-ml", "2026-08-12")
	if err != nil {
		t.Fatal(err)
	}
	if byLoc["loc-a"] != 100 || byLoc["loc-b"] != 200 {
		t.Fatalf("byLoc=%v", byLoc)
	}
	var sum int64
	for _, v := range byLoc {
		sum += v
	}
	if sum != 300 {
		t.Fatalf("sum=%d", sum)
	}
}
