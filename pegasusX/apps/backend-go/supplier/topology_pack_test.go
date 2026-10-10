package supplier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func TestHandleTopologyPut_CrossMarketDeferred(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := &Service{supplierID: "sup-1"}
	body := `{"warehouses":[{"name":"WH","lat":41.3111,"lng":69.2797,"country_code":"US"}]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/supplier/topology", strings.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "u1", Role: auth.RoleAdmin, SupplierID: "sup-1", MarketCode: "UZ",
	}))
	rr := httptest.NewRecorder()
	svc.handleTopologyPut(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != auth.ErrCrossMarketDeferred.Error() {
		t.Fatalf("error=%q", payload["error"])
	}
}

func TestHandleTopologyPut_InheritsPackCountry(t *testing.T) {
	t.Parallel()
	pack, ok := auth.ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	geo, err := proximity.StampNodeGeography(pack, 41.3111, 69.2797, "")
	if err != nil {
		t.Fatal(err)
	}
	if geo.CountryCode != "UZ" || geo.H3Cell == "" {
		t.Fatalf("geo=%+v", geo)
	}
}
