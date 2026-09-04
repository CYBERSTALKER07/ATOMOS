package warehouse

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func TestH3CellFromLatLngWarehouseParity(t *testing.T) {
	got := proximity.H3CellFromLatLng(41.3111, 69.2797)
	if got == "" {
		t.Fatal("expected H3 cell for depot coordinates")
	}
}

func TestHandleOpsLocationPatch_IdempotencyReplay(t *testing.T) {
	body := []byte(`{"address":"Depot 1","place_id":"place-1","lat":41.3111,"lng":69.2797}`)
	cached := warehouseLocationResponse{
		WarehouseID: "wh-1",
		Name:        "Main",
		Address:     "Depot 1",
		PlaceID:     "place-1",
		Lat:         41.3111,
		Lng:         69.2797,
		UpdatedAt:   "2026-06-28T12:00:00Z",
	}
	cachedBytes, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached: %v", err)
	}

	store := idempotency.NewInMemoryStore()
	key := "warehouse-ops-location:wh-1:test"
	if err := store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256Hex(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour); err != nil {
		t.Fatalf("save replay: %v", err)
	}

	svc := &Service{idem: store}
	req := httptest.NewRequest(http.MethodPatch, "/v1/warehouse/ops/location", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:    "ops-1",
		HomeNodeID: "wh-1",
	}))
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()

	svc.handleOpsLocationPatch(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var got warehouseLocationResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.WarehouseID != cached.WarehouseID || got.Address != cached.Address {
		t.Fatalf("replay response = %+v want %+v", got, cached)
	}
}

func TestDecorateLocationWithPack_LocksCountry(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/location", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ops-1", Role: auth.RoleWarehouseAdmin, HomeNodeID: "wh-1", SupplierID: "sup-1", MarketCode: "UZ",
	}))
	got := decorateLocationWithPack(req.Context(), warehouseLocationResponse{WarehouseID: "wh-1"})
	if got.CountryCode != "UZ" || got.PackCountryCode != "UZ" || got.CurrencyCode != "UZS" {
		t.Fatalf("got=%+v", got)
	}
}

func TestHandleOpsLocationPatch_InvalidJSON(t *testing.T) {
	svc := &Service{idem: idempotency.NewInMemoryStore()}
	req := httptest.NewRequest(http.MethodPatch, "/v1/warehouse/ops/location", bytes.NewReader([]byte(`{`)))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{HomeNodeID: "wh-1"}))
	rr := httptest.NewRecorder()
	svc.handleOpsLocationPatch(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400", rr.Code)
	}
}
