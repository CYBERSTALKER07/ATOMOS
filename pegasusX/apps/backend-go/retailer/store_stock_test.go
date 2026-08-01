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

func TestStoreStockReceiveTransferAdjustCount(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now:   time.Now,
		NewID: func() string { return "id-" + time.Now().Format("150405.000000") },
	})
	// unique newID per call
	n := 0
	svc.newID = func() string {
		n++
		return "id-" + string(rune('A'+n%26)) + string(rune('0'+n/26))
	}

	primary, err := svc.EnsurePrimaryLocation(t.Context(), "ret-1")
	if err != nil {
		t.Fatal(err)
	}

	// Receive via inject
	lines := []ReceiveLine{
		{Sku: "SKU-A", ProductName: "Milk", OrderedQty: 10, AcceptedQty: 10},
		{Sku: "SKU-B", ProductName: "Bread", OrderedQty: 5, AcceptedQty: 4},
	}
	if err := svc.injectMemoryReceive("ret-1", primary.LocationID, "order-1", lines); err != nil {
		t.Fatal(err)
	}

	bal, err := svc.listStockBalances(t.Context(), "ret-1", primary.LocationID)
	if err != nil {
		t.Fatal(err)
	}
	bySKU := map[string]int64{}
	for _, b := range bal {
		if b.StockBin == BinBackroom {
			bySKU[b.Sku] = b.OnHand
		}
	}
	if bySKU["SKU-A"] != 10 || bySKU["SKU-B"] != 4 {
		t.Fatalf("balances after receive: %+v", bySKU)
	}

	// Transfer A: 3 backroom -> floor
	if err := svc.applyTransfer(t.Context(), "ret-1", primary.LocationID, primary.LocationID, BinBackroom, BinFloor, "SKU-A", 3, "user-1", "putaway"); err != nil {
		t.Fatal(err)
	}
	onBack, _ := svc.getOnHand(t.Context(), primary.LocationID, BinBackroom, "SKU-A")
	onFloor, _ := svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-A")
	if onBack != 7 || onFloor != 3 {
		t.Fatalf("after transfer back=%d floor=%d", onBack, onFloor)
	}

	// Adjust -1 floor
	if err := svc.applyAdjust(t.Context(), "ret-1", primary.LocationID, BinFloor, "SKU-A", -1, "user-1", "damage"); err != nil {
		t.Fatal(err)
	}
	onFloor, _ = svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-A")
	if onFloor != 2 {
		t.Fatalf("after adjust floor=%d", onFloor)
	}

	// Insufficient stock
	if err := svc.applyDelta(t.Context(), "ret-1", primary.LocationID, BinFloor, "SKU-A", -100, MoveSale, "SALE", "x", "u", ""); err == nil {
		t.Fatal("expected insufficient_stock")
	}

	// HTTP list
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/stock?location_id="+primary.LocationID, nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerRole: "OWNER",
		ActiveLocationID: primary.LocationID,
	}))
	rr := httptest.NewRecorder()
	svc.HandleStockList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rr.Code, rr.Body.String())
	}

	// HTTP adjust
	body := `{"location_id":"` + primary.LocationID + `","sku":"SKU-B","qty_delta":2,"stock_bin":"BACKROOM","note":"found"}`
	reqAdj := httptest.NewRequest(http.MethodPost, "/v1/retailer/stock/adjust", bytes.NewBufferString(body))
	reqAdj = reqAdj.WithContext(auth.WithClaims(reqAdj.Context(), auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerRole: "OWNER",
	}))
	rrAdj := httptest.NewRecorder()
	svc.HandleStockAdjust(rrAdj, reqAdj)
	if rrAdj.Code != http.StatusOK {
		t.Fatalf("adjust status=%d body=%s", rrAdj.Code, rrAdj.Body.String())
	}
	onB, _ := svc.getOnHand(t.Context(), primary.LocationID, BinBackroom, "SKU-B")
	if onB != 6 {
		t.Fatalf("SKU-B after adjust=%d", onB)
	}

	// Count commit
	countBody, _ := json.Marshal(map[string]any{
		"location_id": primary.LocationID,
		"stock_bin":   BinFloor,
		"commit":      true,
		"lines":       []map[string]any{{"sku": "SKU-A", "counted_qty": 5}},
	})
	reqC := httptest.NewRequest(http.MethodPost, "/v1/retailer/stock/counts", bytes.NewReader(countBody))
	reqC = reqC.WithContext(auth.WithClaims(reqC.Context(), auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerRole: "OWNER",
	}))
	rrC := httptest.NewRecorder()
	svc.HandleStockCount(rrC, reqC)
	if rrC.Code != http.StatusCreated {
		t.Fatalf("count status=%d body=%s", rrC.Code, rrC.Body.String())
	}
	onFloor, _ = svc.getOnHand(t.Context(), primary.LocationID, BinFloor, "SKU-A")
	if onFloor != 5 {
		t.Fatalf("after count floor SKU-A=%d want 5", onFloor)
	}

	total, _ := svc.SumOnHandForSKU(t.Context(), "ret-1", "SKU-A")
	if total != 5+7 { // floor 5 + backroom 7
		// floor was adjusted to 2 then counted to 5; backroom still 7
		if total != 12 {
			t.Fatalf("sum on hand SKU-A=%d want 12", total)
		}
	}
}
