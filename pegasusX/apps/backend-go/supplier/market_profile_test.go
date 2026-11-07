package supplier

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleProfile_EmptyMarketNotUZ(t *testing.T) {
	repo := &countingSupplierRepo{
		profiles: map[string]Profile{
			"sup-1": {SupplierID: "sup-1", LegalName: "Acme", Country: "UZ"},
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/profile", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role: auth.RoleAdmin, SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleProfile(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["market_code"] != "" {
		t.Fatalf("empty row must not advertise UZ, got %v", body["market_code"])
	}
	if body["home_cell"] != "" {
		t.Fatalf("empty row must not advertise cell-uz, got %v", body["home_cell"])
	}
}

func TestHandleProfile_PersistsMarketPack(t *testing.T) {
	repo := &countingSupplierRepo{
		profiles: map[string]Profile{
			"sup-1": {SupplierID: "sup-1", LegalName: "Acme"},
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})
	body, _ := json.Marshal(map[string]string{"market_code": "eu"})
	req := httptest.NewRequest(http.MethodPut, "/v1/supplier/profile", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role: auth.RoleAdmin, SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleProfile(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	got := repo.profiles["sup-1"]
	if got.MarketCode != "EU" || got.HomeCell != "cell-eu" {
		t.Fatalf("persisted=%+v", got)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["market_code"] != "EU" || resp["home_cell"] != "cell-eu" {
		t.Fatalf("resp=%v", resp)
	}
}

func TestHandleProfile_UnknownMarket404(t *testing.T) {
	repo := &countingSupplierRepo{
		profiles: map[string]Profile{
			"sup-1": {SupplierID: "sup-1"},
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1"})
	body, _ := json.Marshal(map[string]string{"market_code": "XX"})
	req := httptest.NewRequest(http.MethodPut, "/v1/supplier/profile", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role: auth.RoleAdmin, SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleProfile(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestWireMarketProfileLookup_EmptyIsNotChosen(t *testing.T) {
	t.Cleanup(func() { auth.SetMarketProfileLookup(nil) })
	repo := &countingSupplierRepo{
		profiles: map[string]Profile{
			"sup-1": {SupplierID: "sup-1"},
		},
	}
	WireMarketProfileLookup(repo)
	asg := auth.ResolveMarketAssignment(auth.Claims{
		Subject: "u", Role: auth.RoleAdmin, SupplierID: "sup-1",
		MarketCode: "UZ",
	})
	if asg.Source != auth.MarketSourceEnv {
		t.Fatalf("empty row must be env, got %+v", asg)
	}

	repo.profiles["sup-1"] = Profile{SupplierID: "sup-1", MarketCode: "EU", HomeCell: "cell-eu"}
	asg = auth.ResolveMarketAssignment(auth.Claims{
		Subject: "u", Role: auth.RoleAdmin, SupplierID: "sup-1",
		MarketCode: "UZ",
	})
	if asg.Source != auth.MarketSourceProfile || asg.MarketCode != "EU" {
		t.Fatalf("persisted pack must be profile, got %+v", asg)
	}
}
