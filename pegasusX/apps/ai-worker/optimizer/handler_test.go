package optimizer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	contract "github.com/pegasusx/pegasusx/packages/optimizer-contract"
)

func TestHandler_FailClosedOnEmptyOrWrongAPIKey(t *testing.T) {
	emptyKeyHandler := Handler("", nil, time.Second)
	req := httptest.NewRequest(http.MethodPost, contract.SolvePath, strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	emptyKeyHandler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("empty configured key must fail-closed with 401, got %d", rec.Code)
	}

	h := Handler("secret-internal-key", nil, time.Second)
	reqWrong := httptest.NewRequest(http.MethodPost, contract.SolvePath, strings.NewReader(`{}`))
	reqWrong.Header.Set(contract.AuthHeader, "wrong-key")
	recWrong := httptest.NewRecorder()
	h(recWrong, reqWrong)
	if recWrong.Code != http.StatusUnauthorized {
		t.Fatalf("wrong key must return 401, got %d", recWrong.Code)
	}
}

func TestHandler_RejectsOversizedBodyWith413(t *testing.T) {
	h := Handler("secret-internal-key", nil, time.Second)
	oversized := `{"v":"v1","trace_id":"t1","pad":"` + strings.Repeat("a", (4<<20)+1024) + `"}`
	req := httptest.NewRequest(http.MethodPost, contract.SolvePath, strings.NewReader(oversized))
	req.Header.Set(contract.AuthHeader, "secret-internal-key")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status=%d want 413", rec.Code)
	}
}

func TestHandler_ValidSolveRequestSucceeds(t *testing.T) {
	h := Handler("secret-internal-key", nil, 2*time.Second)
	payload := contract.SolveRequest{
		V:          contract.V,
		TraceID:    "trace-123",
		SupplierID: "sup-1",
		HomeNodeID: "wh-1",
		Stops: []contract.Stop{
			{OrderID: "ord-1", RetailerID: "ret-1", Lat: 41.3111, Lng: 69.2797, VolumeVU: 10},
		},
		Vehicles: []contract.Vehicle{
			{VehicleID: "veh-1", DriverID: "drv-1", StartLat: 41.3000, StartLng: 69.2700, MaxVolumeVU: 100},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, contract.SolvePath, bytes.NewReader(raw))
	req.Header.Set(contract.AuthHeader, "secret-internal-key")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("valid solve status=%d want 200 (body=%s)", rec.Code, rec.Body.String())
	}
}
