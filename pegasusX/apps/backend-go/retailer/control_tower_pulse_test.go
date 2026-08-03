package retailer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestControlTowerPulseEmptyAndLive(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "ct-" + string(rune('A'+n%26))
		},
	})
	owner := auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-ct",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/control-tower/pulse", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleControlTowerPulse(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var pulse ControlTowerPulse
	_ = json.Unmarshal(rr.Body.Bytes(), &pulse)
	if !pulse.Empty {
		t.Fatalf("expected empty pulse, got %+v", pulse)
	}
	if pulse.RetailerID != "ret-ct" {
		t.Fatalf("retailer_id=%s", pulse.RetailerID)
	}
	if len(pulse.Capabilities) == 0 {
		t.Fatal("capabilities should include CORE")
	}

	// Seed open POS session + sale → non-empty sales and sessions
	primary, _ := svc.EnsurePrimaryLocation(t.Context(), "ret-ct")
	regID := svc.newID()
	sessID := svc.newID()
	_ = svc.saveRegister(t.Context(), RegisterDTO{
		RegisterID: regID, RetailerID: "ret-ct", LocationID: primary.LocationID,
		Label: "R", Status: RegisterStatusActive,
	})
	_ = svc.savePosSession(t.Context(), PosSessionDTO{
		SessionID: sessID, RegisterID: regID, LocationID: primary.LocationID,
		RetailerID: "ret-ct", OpenedByUserID: "o", Status: PosSessionOpen,
		OpeningFloatMinor: 0, Currency: "UZS",
		OpenedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	_ = svc.savePosSale(t.Context(), PosSaleDTO{
		SaleID: svc.newID(), SessionID: sessID, RegisterID: regID,
		LocationID: primary.LocationID, RetailerID: "ret-ct", CashierUserID: "o",
		Status: "COMPLETED", TotalMinor: 1200, Currency: "UZS", ReceiptNumber: "RCT1",
		Lines: []PosSaleLine{{Sku: "X", Qty: 1, UnitPriceMinor: 1200, LineTotalMinor: 1200}},
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	_ = svc.applyDelta(t.Context(), "ret-ct", primary.LocationID, BinFloor, "X", 3, MoveReceive, "TEST", "o1", "o", "")

	req2 := httptest.NewRequest(http.MethodGet, "/v1/retailer/control-tower/pulse", nil)
	req2 = req2.WithContext(auth.WithClaims(req2.Context(), owner))
	rr2 := httptest.NewRecorder()
	svc.HandleControlTowerPulse(rr2, req2)
	_ = json.Unmarshal(rr2.Body.Bytes(), &pulse)
	if pulse.Empty {
		t.Fatalf("expected non-empty after seed, got %+v", pulse)
	}
	if pulse.PosOpenSessions < 1 {
		t.Fatalf("pos_open_sessions=%d", pulse.PosOpenSessions)
	}
	if pulse.SalesMinor7d < 1200 {
		t.Fatalf("sales_minor_7d=%d", pulse.SalesMinor7d)
	}
}
