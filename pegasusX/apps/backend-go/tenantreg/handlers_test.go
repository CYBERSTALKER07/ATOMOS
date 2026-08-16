package tenantreg

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleRegister_Created(t *testing.T) {
	svc := testService(&fakeRegistry{})
	body, _ := json.Marshal(Request{
		LegalName: "Acme", Phone: "+99890", Password: "secret12", MarketCode: "UZ",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/platform/tenants/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	svc.HandleRegister(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.SupplierID == "seed-1" || resp.SupplierID == "" {
		t.Fatalf("supplier_id=%s", resp.SupplierID)
	}
	if resp.Token == "" {
		t.Fatal("expected JWT")
	}
	got, err := auth.Parse(resp.Token, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if got.MarketCode != "UZ" || got.HomeCell != "cell-uz" || got.SupplierID != resp.SupplierID {
		t.Fatalf("claims=%+v", got)
	}
}

func TestHandleRegister_EmptyMarket400(t *testing.T) {
	svc := testService(&fakeRegistry{})
	body, _ := json.Marshal(Request{
		LegalName: "Acme", Phone: "+1", Password: "x",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/platform/tenants/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	svc.HandleRegister(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("empty market must not mint UZ: status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRegister_Unknown404(t *testing.T) {
	svc := testService(&fakeRegistry{})
	body, _ := json.Marshal(Request{
		LegalName: "Acme", Phone: "+1", Password: "x", MarketCode: "XX",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/platform/tenants/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	svc.HandleRegister(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRegister_Planned404(t *testing.T) {
	svc := testService(&fakeRegistry{})
	body, _ := json.Marshal(Request{
		LegalName: "Acme", Phone: "+1", Password: "x", MarketCode: "EU",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/platform/tenants/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	svc.HandleRegister(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != ErrMarketNotShipped.Error() || resp["code"] != "EU" {
		t.Fatalf("body=%v", resp)
	}
}
