package factory

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

func TestH3CellFromLatLngFactoryParity(t *testing.T) {
	got := proximity.H3CellFromLatLng(41.3111, 69.2797)
	if got == "" {
		t.Fatal("expected H3 cell for factory depot coordinates")
	}
}

func TestHandleFactoryLocationPatch_IdempotencyReplay(t *testing.T) {
	body := []byte(`{"address":"Depot 1","place_id":"place-1","lat":41.3111,"lng":69.2797}`)
	cached := factoryLocationResponse{
		FactoryID: "fac-1",
		Name:      "Main",
		Address:   "Depot 1",
		PlaceID:   "place-1",
		Lat:       41.3111,
		Lng:       69.2797,
		UpdatedAt: "2026-06-28T12:00:00Z",
	}
	cachedBytes, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("marshal cached: %v", err)
	}

	store := idempotency.NewInMemoryStore()
	key := "factory-ops-location:fac-1:test"
	if err := store.Save(context.Background(), key, idempotency.Record{
		BodyHash:   sha256HexBytes(body),
		StatusCode: http.StatusOK,
		Response:   cachedBytes,
	}, 24*time.Hour); err != nil {
		t.Fatalf("save replay: %v", err)
	}

	svc := &Service{idem: store}
	req := httptest.NewRequest(http.MethodPatch, "/v1/factory/ops/location", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.FactoryScopeKey, &auth.FactoryScope{FactoryID: "fac-1"}))
	req.Header.Set("Idempotency-Key", key)
	rr := httptest.NewRecorder()

	svc.handleFactoryLocationPatch(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var got factoryLocationResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.FactoryID != cached.FactoryID || got.Address != cached.Address {
		t.Fatalf("replay response = %+v want %+v", got, cached)
	}
}

func TestHandleFactoryLocationPatch_InvalidJSON(t *testing.T) {
	svc := &Service{idem: idempotency.NewInMemoryStore()}
	req := httptest.NewRequest(http.MethodPatch, "/v1/factory/ops/location", bytes.NewReader([]byte(`{`)))
	req = req.WithContext(context.WithValue(req.Context(), auth.FactoryScopeKey, &auth.FactoryScope{FactoryID: "fac-1"}))
	rr := httptest.NewRecorder()
	svc.handleFactoryLocationPatch(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d want 400", rr.Code)
	}
}
